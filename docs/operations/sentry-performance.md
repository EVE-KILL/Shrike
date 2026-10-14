# Sentry performance investigation

## 2026-10-14

Sentry's production samples identified expensive coalition membership counts,
affiliation refresh selections, entity breakdowns and campaign processing. SQL
plans below were collected read-only with a 10-second statement timeout. These
are database execution times, not page latency or a promise of production p95.

### Entity breakdowns

SHRIKE-C reports a repeated pattern of pool acquisition and database queries on
corporation pages. There are four fixed category queries, executed concurrently;
the count does not grow with the number of rows. Keep their selective index
ranges rather than replacing them with a wider scan solely to silence the alert.

Rank and limit the aggregate to ten rows before joining ship/corporation/alliance
names. Alternating old/new warm-cache runs produced these median times:

| Entity | Category | Before | After | Before/after shared pages |
| --- | --- | ---: | ---: | ---: |
| Corporation 98169165 | Ships used | 1.630 ms | 1.055 ms | 648 / 117 |
| Corporation 98169165 | Ships lost | 8.316 ms | 0.844 ms | 831 / 118 |
| Corporation 98169165 | Opposing corporations | 9.055 ms | 0.795 ms | 1432 / 118 |
| Corporation 98169165 | Opposing alliances | 6.083 ms | 0.986 ms | 1100 / 119 |
| Alliance 99005338 | Ships used | 1.528 ms | 1.000 ms | 588 / 120 |
| Alliance 99005338 | Ships lost | 1.042 ms | 0.904 ms | 549 / 115 |
| Alliance 99005338 | Opposing corporations | 3.372 ms | 0.764 ms | 1142 / 116 |
| Alliance 99005338 | Opposing alliances | 2.301 ms | 0.993 ms | 944 / 114 |

Regression fixtures cover all four categories for corporations and alliances,
both accumulated and 30-day windows, ties, multiple aggregate periods and
missing names. The four-query pattern can still trigger Sentry's heuristic;
reducing its actual work is the objective.

### Character indexes

The coalition overview executed in 5641.806 ms, touching 212,310 database pages.
It counts live characters per alliance, so an alliance-key partial index covering
`deleted IS NOT TRUE` can avoid character heap reads. About 97% of characters
had no alliance when measured. Fixtures verify exact totals, including null
`deleted` values, and an index-only plan.

The active affiliation selection executed in 444.023 ms warm, touching 223,069
pages; its sampled p95 was 4.9 s. About 86% of characters had never been active.
The retained partial index covers `last_active`, `updated_at` and `character_id`.
It can reject recently checked characters without fetching their rows.
Keep the existing index for dormant refreshes. Fixtures verify candidate
priority, limits, cutoff boundaries, null timestamps and deleted characters.

Migration 29 builds both indexes concurrently and verifies that each is valid
before advancing the migration ledger. Both production indexes are valid.
The membership index occupies 4.5 MB; the active affiliation index occupies 107 MB.

The first index builds exposed a maintenance problem. Membership scans still
required 111,300 heap visibility checks, and PostgreSQL avoided the active index.
A regular `VACUUM (ANALYZE, TRUNCATE FALSE) characters` restored visibility and
refreshed the plans. Measured execution after vacuum was:

| Query | Before | After | Before/after database pages |
| --- | ---: | ---: | ---: |
| Coalition overview | 5641.806 ms | 25.589–36.959 ms | 212,310 / 1,175 |
| Active affiliation selection | 444.023 ms warm | 12.792–23.162 ms | 223,069 / 1,581–1,690 |

The coalition baseline included disk reads. The initial affiliation run after
vacuum included disk reads; later runs were warm. These are spot measurements;
observe production percentiles as the table receives updates.

The retained active index replaced an unsuccessful due-order variant before
the migration was released. No existing application index was removed.
Migration 29 also lowers this table's vacuum scale factor from 0.02 to 0.005.
At 21 million rows, this triggers maintenance near 105,000 changes instead of
420,000. Analyze thresholds and vacuum cost limits remain unchanged.
This keeps the [visibility map](https://www.postgresql.org/docs/current/indexes-index-only-scans.html)
current enough for index-only scans. The down migration restores the old factor.

### Rejected campaign experiment

Campaign attacker attribution had a sampled p95 of 9.4 s. A read-only comparison
used the latest 20,000 killmails since October 1 and the participant entities in
campaign `kxbrHaZMpxCMJl`. Sharing a materialized attacker input across the three
entity joins timed out twice at 10 seconds. The original query finished in
8.2 seconds initially and 1.5 seconds warm. Its selective entity-first access
is preferable for this sample; no campaign attribution change was made.

Follow-ups remain for campaign candidate/materialization/statistics queries,
faction-warfare valuable kills and solo hull matchups. Preserve campaign
isolation, side masks, specificity and distinct-kill accounting in any change.

### Campaign valuable kills and faction date ranges

Campaign valuable kills previously enriched every candidate before selecting
ten. Rank by value and killmail ID inside a materialized CTE before enrichment.
On the bounded 20,000-killmail sample, warm execution fell from 400.522 ms to
181.885–183.474 ms. Regression fixtures cover campaign isolation, stable value
ties, missing names and types, null sides, empty campaigns and timestamp format.

The Caldari faction-war valuable list intersected a date bitmap with 1.3 million
historical faction winners. It touched about 8,900 pages to return eight cards.
Execution was 920 ms initially and 279 ms on a later run that still needed reads.
Removing the materialization barrier timed out twice at ten seconds; retain it.

Migration 30 adds a partial faction/date/ID index for the recent range.
Keep the existing faction/ID index for cursor paging. A realistic fixture verifies
the selected compound range, faction/date scope, stable ties and valid-type
selection before the result limit. Validate the production build before claiming
the index's performance benefit.

The production build is valid and occupies 205 MB. After maintenance and before
the build, warm runs took 219–232 ms and touched 8,881 pages. After the build,
the unchanged query used the compound index and finished in 10–19 ms, touching
1,997 pages. The first run needed ten disk reads; later runs were warm.
These are database spot measurements. Observe production percentiles after rollout.

The solo hull sample already uses the matchup indexes, including an attacker
index-only existence probe. It finished in 105 ms with 5,625 disk reads.
Killmails had not been automatically vacuumed since August 31. A throttled regular
vacuum restored an index-only killmail plan with zero heap fetches. The first run
took 63 ms with 234 reads; warm runs took 9–12 ms. These timings include different
cache states; they do not prove a corresponding reduction in production p95.
No additional matchup index was needed.

Migration 30 lowers the killmail vacuum factor from 0.02 to 0.001. This triggers
maintenance near 96,000 dead tuples instead of 1.9 million at the measured size.
Other table cost settings and the analyze threshold remain unchanged.

Campaign candidate, materialization, side totals, intel and top-killer queries
already scope their inputs to one campaign. Top killers rank before name joins.
The sampled daily character-intel query runs in the rollup worker with one-day
bounds; it does not recompute those aggregates during page rendering.
The shared campaign-participant totals experiment was inconclusive: both variants
hit the ten-second limit. No speculative attribution or totals rewrite was kept.

### Cache shutdown and the repair-job alert

SHRIKE-E captured `closed pool` from a cache loader while a v0.2.7 serve pod
was shutting down. Detached refreshes had no service lifetime. Singleflight
callbacks could also outlive an HTTP waiter after its request was cancelled.

The response cache now tracks actual loader callbacks. Its close method prevents
new callbacks, cancels background refreshes and waits for registered callbacks.
Serve stops ingress, drains the API cache, then closes shared storage and database
pools. Client cancellation still leaves detached refreshes running during normal
service operation. Shutdown cancellation receives a separate metric outcome.

Deterministic concurrency tests verify refresh cancellation, cleanup ordering,
abandoned foreground loaders and refusal to load after closure. The full uncached
race suite also passes. Genuine closed-pool errors remain visible to Sentry.

SHRIKE-D flags the repair cron's fixed fourteen daily history checks. The trace
contains fourteen HTTP requests, fourteen array-based database checks and fourteen
pool acquisitions. They took 1516 ms, 772 ms and 0.39 ms respectively within
a 2720 ms job. This is one indexed check per day, not one query per killmail.
Retain the day bounds and batched IDs. The alert is an investigated design
tradeoff; no larger historical scan or speculative index was introduced.

### Scoped war pagination

The next production sample exposed scoped war lists that walked historical wars
before finding participant matches. Each requested participant can match an
aggressor, defender or ally. Select the prefix through the requested page in
each matching range, combine IDs with `UNION`, then paginate before name lookup.

Every nonparticipant filter applies before each range's limit. If both a custom
domain and an explicit participant restrict the request, keep their intersection.
Use the explicit participant for the candidate ranges. A globally selected row
must occur in the corresponding prefix of at least one matching range.
This preserves deep pages and removes overlapping IDs without widening the scan.
Upcoming wars now use war ID as a deterministic tie-breaker for equal start times.

Alternating read-only production runs for corporation 98169165 improved from
107–117 ms to 3.1–3.6 ms. Database-page accesses fell from 26,925 to 515.
Two no-match samples improved from 150–230 ms to 2–5 ms, touching 171 pages
instead of 42,850. Two frequently involved corporations retained comparable
performance: about 10 ms and 21–24 ms. No new indexes were added.

Exact result objects match the original SQL inside repeatable-read snapshots
for all five sampled corporations. A 336-case HTTP/database fixture compares
the full-set reference query with the implementation across tenant scopes,
participants, allies, overlap, filters, sorting, null start times and deep pages.
Missing entity names stay visible. These are SQL measurements; verify released
request latency and Sentry percentiles separately.
