# Backend parity audit log

Last updated: 2026-07-26 14:39 Europe/Copenhagen

This is the handoff point for the TypeScript backend to Go audit. The working
comparison is:

- TypeScript: `../backend`
- Go: this repository

The TypeScript implementation is the behavioral baseline, not an immutable
specification. The Go implementation should preserve the application schema,
persisted meaning, and emitted-data contracts. It may use more appropriate Go
or River mechanics and may fix clear defects where the intended result is
known.

## Agreed scope and rules

- Goose replaces Drizzle migrations. The final application schema must match;
  the migration ledger and migration implementation do not need to.
- River replaces BullMQ for queues and scheduled jobs.
- Queue implementation details may differ, but priority tiers, retry behavior,
  deduplication lifecycle, and the resulting persisted/emitted data must remain
  compatible.
- Campaign processing has no killmail-count cap. Its bounded workload is the
  campaign time range, currently limited to one year.
- Stable sorting is retained for payload collections whose order has no domain
  meaning. This makes equivalent payloads deterministic without changing their
  semantics.
- HTTP/API/Caddy/serve parity is out of scope for this pass.
- `discord_events` remains consumed by the Discord service rather than Shrike.
- Do not change the application schema merely to make the Go implementation
  easier.

### Comparison methodology

Production has accumulated more application data than the local test database,
so ordinary row counts are not treated as parity evidence. Killmail, entity,
stats, campaign, and other derived-data comparisons use one of:

- the same bounded input corpus against both implementations;
- both implementations reading the same database;
- algorithm-agreement tests over the same stored rows; or
- primary-key/value comparison on rows shared by both databases.

Whole-table counts are only compared for versioned authoritative snapshots,
such as the same EVE SDE build. Production-only application rows are expected
and are not a mismatch.

Commands explicitly excluded from the port:

- `fetch:entity-history`
- `backfill:recent-entity-histories`
- `enrich:fittings`
- `recompute:capital-prices`
- `db:migrate:repair`
- `debug:pop`
- `validate:archetypes`
- all `ekimport:*` commands
- all `storage:*` commands
- `legacy:export-killmails`
- `generate:map-image`
- `hello`

## Audit status

### Completed

#### Schema and migrations

- The Goose migration chain was compared against the TypeScript/Drizzle
  application schema.
- Goose-specific metadata was treated as implementation detail.
- Missing application objects and defaults were aligned.
- A schema-parity test exists:
  `TestGooseSchemaMatchesTypeScriptMigrations`.

#### Killmail processing

- ESI and R2Z2 input handling, parsing, item flattening, attacker storage,
  valuation, points, classification, delayed processing, and effect-ledger
  behavior were compared.
- Price selection and killmail relay data were aligned.
- Archive imports do not incorrectly seed the live effects ledger.
- Authenticated killmail fetches now record the same request data as the
  TypeScript path.
- Deterministic ordering was retained where ordering is semantically
  irrelevant.

#### Killmail effects and downstream data

- Character and corporation killmail effects were completed.
- Stats, achievements, fitting extraction, graph ingestion, war interactions,
  campaign processing, feed/relay publication, routing keys, and ticker
  announcements were compared.
- Historical war-effect repair was added.
- Fitting persistence, graph timestamps, battle team totals, campaign query
  behavior, campaign stats, campaign prize settlement, corporation-wallet
  processing, and wallet references were aligned.

#### Workers and queues

- All 19 queues owned by the Go backend have registered River workers.
- All 20 TypeScript queue declarations were accounted for; `discord_events` is
  intentionally external.
- All queue declarations were compared for concurrency, attempts, backoff, and
  relative priority.
- River uniqueness is limited to active job states so a completed entity job
  does not suppress future refreshes.
- Follow-up jobs retain the semantic priority of their parent.
- `work:queue`/`work:queues` no longer prints a “ported N/M” count at startup.

Current registration check:

```text
19 Go-owned workers
0 unimplemented Go-owned queues
```

#### Scheduled jobs

- All 32 TypeScript cron declarations have Go implementations.
- Schedules, run-on-start behavior, TQ requirements, and job ownership were
  compared.
- River periodic jobs provide leader-elected, cluster-safe scheduling.
- Scheduled SDE imports, entity maintenance, status relays, announcements, and
  missing-war repairs were corrected during the audit.

Current registration check:

```text
32 declared crons
32 implemented crons
```

#### Explicitly requested CLI ports

The requested commands exist and their resulting data paths were compared:

- `rebuild:war-interactions`
  - atomic replacement from killmails and attackers
  - effect-ledger repair included
- `catchup:stats`
  - daily rows are deleted and authoritatively recomputed
  - monthly/yearly rollups remain separate
- `backfill:points`
  - only unscored rows are touched
  - the same scorer as live ingest is used
- `scan:character-trailing`
  - cumulative miss cap and entity-history/corporation cascade retained
  - Go additionally treats ESI 422 as a miss, which avoids aborting at the
    edge of the allocated ID range
- `scan:character-holes`
  - gap sampling, dry-run geometry, miss cap, and resume cursor retained
- `import:zkb_history`
  - newest-to-oldest traversal, daily existence check, two-second pacing,
    dormant priority, and per-day cursor retained
- `export:entities`
  - live output from both implementations was canonicalized and compared
  - all three exports matched: characters, corporations, and alliances

#### Command-by-command matrix

Every TypeScript command has been classified below. “Matched” means the
persisted result or operational effect is the same; the Go command may enqueue
River work rather than doing the work inline.

| TypeScript command | Status | Go result |
| --- | --- | --- |
| `backfill:achievements` | TypeScript defect fixed | Rebuilds from maintained killmail/stats sources, removes stale zero-count rows, and resyncs zero-point characters. The TS query references the removed `character_ship_stats_daily` table. |
| `backfill:battles` | Matched | Same day range, hotspot threshold, detection path, persistence, dry-run behavior, and resume cursor. |
| `backfill:fittings` | Matched via River | Same 90-day default and killmail-ID cursor; fit extraction is performed by the normal River worker. |
| `backfill:graph` | Matched via River | Same recent window and optional clear; normal graph-ingest jobs produce the graph data. |
| `backfill:kills-daily-count` | TypeScript defect fixed | Transactionally replaces each selected `(month, type)` slice, so source deletions cannot leave stale nonzero rows. |
| `backfill:last-active` | Matched | Same maximum activity over attacker and victim appearances; one range query replaces the TS month loop. |
| `backfill:missing-wars` | Matched | Finds every referenced missing war and queues metadata repair after the same explicit confirmation gate. |
| `backfill:points` | Matched | Scores only unscored kills with the live scorer over the selected time range. |
| `backfill:stats` | Matched with correctness fix | Same old-month/recent-day strategy, top-N rollups, table filters, reset, and rollup controls; latest killmail IDs remain paired with their timestamps. |
| `backfill:recent-entity-histories` | Explicitly skipped | Not requested for the Go port. |
| `campaign:process` | Matched with correctness fixes | Full idempotent recompute, lifecycle selection, stats, prize settlement, and wallet references. No killmail-count ceiling; the one-year campaign range is the bound. |
| `catchup:stats` | Matched with correctness fix | Same authoritative daily recompute; each day is now atomically deleted and replaced. |
| `cronjobs` | Intentional River implementation | Same declared cron work and run-one behavior; River periodic jobs replace process-local cron. |
| `db:migrate` | Intentional Goose implementation | Goose replaces Drizzle while producing the same final application schema. |
| `db:migrate:repair` | Explicitly skipped | Drizzle-ledger repair is not relevant to Goose. |
| `db:status` | Matched | Same database health, activity, progress, and table reporting intent. |
| `db:vacuum` | Matched | Same table/all-table vacuum behavior, optional full vacuum, and reindex pass. |
| `debug:killmail` | Matched | Same fetch/parse/insert and optional downstream debug stages. |
| `debug:pop` | Explicitly skipped | Not requested for the Go port. |
| `ekimport:alliances` | Explicitly skipped | Legacy import family excluded. |
| `ekimport:characters` | Explicitly skipped | Legacy import family excluded. |
| `ekimport:corporations` | Explicitly skipped | Legacy import family excluded. |
| `ekimport:killmails` | Explicitly skipped | Legacy import family excluded. |
| `ekimport:prices` | Explicitly skipped | Legacy import family excluded. |
| `enrich:fittings` | Explicitly skipped | Not requested for the Go port. |
| `everef:insurance` | Matched with safety fix | Same full snapshot replacement, now atomic and protected from an empty snapshot. |
| `everef:killmails` | Matched with correctness fixes | Same archive discovery, parser, valuation, and inserts; isolated parse failures no longer strand a day, and successful bookmarks resume at the next day. |
| `everef:prices` | Matched | Same inclusive date selection, The Forge filter, first-write-wins history, and per-day failure continuation. |
| `everef:sovereignty` | TypeScript defects fixed | Current state actually advances; historical replay uses historical state, is idempotent, and cannot rewind a newer live row. |
| `everef:wars` | Matched with correctness fixes | Same metadata, allies, killmails, war-ID repair, yearly/daily cursors, and current snapshot; isolated killmail failures do not discard the archive. |
| `export:entities` | Matched | Character, corporation, and alliance JSON were canonicalized from the same database and matched exactly. |
| `fetch:entity-history` | Explicitly skipped | Not requested for the Go port. |
| `generate:map-image` | Explicitly skipped | Not requested for the Go port. |
| `hello` | Explicitly skipped | Example command excluded. |
| `import:zkb_history` | Matched via River | Same newest-first archive walk, date checks, pacing, low priority, missing-only dispatch, and cursor. |
| `legacy:export-killmails` | Explicitly skipped | Legacy export excluded. |
| `queue:stale-entities` | Matched via River | Same stale-selection rules and entity refresh effects; reported counts reflect actual River inserts. |
| `queues` | Intentional River implementation | Lists queues or runs the selected River worker set; it no longer prints a port-progress banner. |
| `rebuild:war-interactions` | Matched with correctness fixes | Same atomic full/single-war replacement and effect-ledger repair; latest killmail ID and time remain one coherent pair. |
| `recompute:capital-prices` | Explicitly skipped | Not requested for the Go port. |
| `reset:entity-history-queues` | Intentional River implementation | Drains the River queues, clears queue markers, and resumes processing with the same confirmation guard. |
| `scan:character-holes` | Matched | Same gap geometry, probes, skip stride, miss cap, dry run, and resume cursor. |
| `scan:character-trailing` | Matched with ESI edge fix | Same forward scan, cumulative miss cap, continuation, and history cascade; ESI 422 is correctly treated as an unallocated ID. |
| `serve` | Deferred | HTTP/API/Caddy surfaces are explicitly outside this audit pass. |
| `storage:delete` | Explicitly skipped | Storage command family excluded. |
| `storage:get` | Explicitly skipped | Storage command family excluded. |
| `storage:list` | Explicitly skipped | Storage command family excluded. |
| `storage:put` | Explicitly skipped | Storage command family excluded. |
| `update:sde` | Matched with snapshot fix | Goose-era importer produces the same application tables and prunes rows removed from fully authoritative archive snapshots. |
| `validate:archetypes` | Explicitly skipped | Not requested for the Go port. |
| `work:queues` | Intentional River implementation | Runs all or selected Go-owned River workers with graceful draining. |
| `work:zkb` | Matched | Same R2Z2 cursor, sequencing, repost handling, rate behavior, and killmail dispatch. |

#### Shared infrastructure

- Config-store key behavior was compared.
- ESI endpoint grouping, cache, rate limiting, error-budget handling,
  single-flight behavior, sequential endpoint locks, header feedback, and
  retry behavior were compared.
- R2Z2 sequence/cursor handling, repost detection, counters, rate limiting, and
  full killmail-body delivery were compared.
- Feed, relay, routing, and ticker payloads were compared.
- Runtime SDE cache, market paths, custom prices, and market-history lookup
  order were compared.
- Entity refresh and history synchronization were hardened.
- Unrotated ESI refresh tokens are now preserved.

## Deliberate fixes and justified divergences

These should not be “fixed back” to TypeScript behavior without revisiting the
reason:

- Campaign processing has no 250,000-killmail candidate cap.
- Feed insertion is not retried merely because the post-insert Redis
  notification failed; retrying the whole operation can create duplicate feed
  rows.
- A stats row with nullable legacy `attacker_count` falls back to the stored
  attacker rows, while an explicit zero remains zero.
- Organization `SHIP_FLOWN` counts one distinct hull per organization per
  killmail. This matches the authoritative TypeScript stats backfill and avoids
  live rows changing after a rebuild.
- The latest stats killmail ID stays paired with the latest killmail timestamp
  instead of independently taking the maximum of both fields.
- Hourly missing-war repair performs the full war killmail-list walk.
  Metadata-only repair remains available for explicit bulk backfills.
- Runtime dogma lookup uses the correct attribute filter.
- River scheduling is leader elected and does not stack another copy while the
  previous scheduled job is active.
- Retry delays have bounded jitter to avoid synchronized retry storms.
- SDE import rejects corrupt JSONL instead of silently ignoring malformed
  lines.
- Empty SDE text is stored as `NULL` instead of an empty string where no reader
  distinguishes the two.
- Go fills a few SDE fields that are present in the archive but the TypeScript
  importer omitted, such as region faction/nebula/wormhole-class IDs.

Known TypeScript-problem areas were treated as intent-based ports rather than
line-for-line ports:

- sovereignty
- `fw_update`
- `fw_stats`
- FW system history

## Commits produced during the audit

Audit and parity work:

```text
1ff5cc2 Port backend processing and workers to Go
d7b4231 Keep Goose schema aligned with backend
8986de4 Align killmail valuation and relay output
82b19a7 Keep archive imports out of effects ledger
8c1115e Match announcement relay payloads
fa6d409 Record authenticated killmail fetches
b4d1a7a Harden entity history synchronization
a91874c Repair historical war killmail effects
7d4b7fa Align fitting persistence with backend
ccbe1a9 Align graph timestamp persistence
bd6ec9b Align battle detection and team totals
877c865 Align campaign query and stats output
4a7e724 Complete corporation wallet processing
4c6d387 Preserve unrotated ESI refresh tokens
37c7e17 Automate scheduled SDE imports
6b5c81a Align scheduled entity maintenance
365b60f Restore status relay payload parity
fcec129 Align scheduled announcement state
06878d6 Preserve legacy attacker counts in stats
855082f Keep organization ship stats rebuildable
ae15278 Complete scheduled missing war repairs
b87dabe Match entity export timestamp shapes
c998b96 Prune stale SDE snapshot rows
d4158c2 Harden historical Everef imports
2edd7b5 Make derived backfills authoritative
50d0220 Keep aggregate killmail markers coherent
```

Concurrent serve/ingress and repository-support commits were preserved:

```text
78db062 feat(serve): embed Caddy and route the HTTP surfaces through Huma
74d67f4 feat(ingress): let each surface answer to several hostnames
feae75a fix(secrets): stop the guard blocking every .env.example edit
```

## Validation already performed

- Every audit commit passed the configured pre-commit checks:
  - gofmt
  - secret scan
  - `go vet`
  - `go test ./...`
  - build
- All 19 Go-owned queues and all 32 crons report implemented.
- Entity exports from TypeScript and Go were compared against the same local
  database and were exactly equal after canonical JSON key sorting.
- Local SDE build is current at build `3444265`.
- The SDE archive itself and all Go-imported SDE tables were compared by
  primary key and value. Shared-key values match; documented differences are
  intentional mappings such as `NULL` for absent text and fields present in
  the archive that TS omitted.
- The populated classifier-agreement test passed on all 15,739 local
  killmails across all 27 kill subsets.
- `TestGooseSchemaMatchesTypeScriptMigrations` passed against a scratch
  database.
- The authoritative achievement-removal regression passed against Postgres.
- `TestCollectorAgainstServices` passed against local Postgres and Redis.
- The matrix mechanically accounts for all 52 TypeScript commands.
- Final uncached `go test ./...`, `go vet ./...`, and the Shrike build all
  passed.

## Resolved investigation: SDE nested-row discrepancy

Both local and production databases report SDE build `3444265`, but three
nested tables have different counts:

| Table | Go/local | TypeScript/production | Difference |
| --- | ---: | ---: | ---: |
| `type_dogma_attributes` | 645,701 | 645,752 | -51 |
| `type_dogma_effects` | 54,070 | 54,109 | -39 |
| `type_materials` | 47,051 | 47,059 | -8 |

Neither side has null values in the relevant payload columns.

The build `3444265` archive contains exactly the Go/local counts shown above:

```text
645,701 dogma attributes
 54,070 dogma effects
 47,051 material rows
```

A primary-key diff found that every mismatch was production-only; Go had no
extra keys. The parent inventory types still exist, but CCP removed those
specific child attributes/effects/materials from the current build. The
TypeScript importer's upsert-only behavior retained them from an older build.

The Go importer now treats fully archive-owned tables as authoritative
snapshots. Merge and prune happen in one transaction, while the four inventory
tables augmented with bundled Dust 514 data remain merge-only. Celestials and
solar-system jumps are also pruned as complete derived snapshots.

The behavior was integration-tested by inserting one synthetic stale row into
each affected nested table and running a partial cached import. Each import
reported one pruned row, and all three synthetic rows were gone afterward.

`sde:verify` now includes nested tables, celestials, jumps, and inventory flags,
not only the 14 straightforward declarations.

## Audit conclusion

No known in-scope command, queue, cron, worker, schema, killmail-processing, or
emitted-data parity gap remains. The known differences are either explicit
scope exclusions, River/Goose implementation substitutions, or documented
correctness fixes.

Residual risks to call out in the final report:

- Production-scale throughput has not been benchmarked.
- Memgraph being unavailable is treated as degraded operation because graph
  data is rebuildable; this should remain visible operationally.
- HTTP/API surfaces contain intentional placeholders and remain outside this
  audit.
