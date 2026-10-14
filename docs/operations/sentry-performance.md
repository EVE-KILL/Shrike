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
An ordered covering partial index excludes those rows from the active refresh
walk. Keep the existing index for dormant refreshes. Fixtures verify candidate
priority, limits, cutoff boundaries, null timestamps and deleted characters.

Migration 29 builds both indexes concurrently and verifies that each is valid
before advancing the migration ledger. Production after-build measurements
must be recorded before claiming either improvement.

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
