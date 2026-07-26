# Backend parity audit log

Last updated: 2026-07-26 06:15 Europe/Copenhagen

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
- The 14 straightforward SDE table row counts exactly match production:
  72,017 total rows.
- The following additional SDE table counts also match production:
  blueprints and all four activity tables, celestials, solar-system jumps, and
  inventory flags.

## Paused investigation: SDE nested-row discrepancy

This is the exact point where the audit stopped.

Both local and production databases report SDE build `3444265`, but three
nested tables have different counts:

| Table | Go/local | TypeScript/production | Difference |
| --- | ---: | ---: | ---: |
| `type_dogma_attributes` | 645,701 | 645,752 | -51 |
| `type_dogma_effects` | 54,070 | 54,109 | -39 |
| `type_materials` | 47,051 | 47,059 | -8 |

Neither side has null values in the relevant payload columns.

The current archive contains exactly 47,051 material entries, so the eight
extra production material rows are probably stale rows retained from an older
SDE build. Both importers upsert current rows but do not delete keys removed
from a later archive. The equivalent archive count check for the two dogma
tables was started but the `jq` expression was malformed and must be rerun.

Next steps for this discrepancy:

1. Count current archive dogma attributes and effects correctly.
2. Diff the primary-key sets between local and production.
3. Check whether every production-only key is absent from build `3444265`.
4. Decide whether snapshot-owned SDE tables should prune stale keys.
   - Pruning is likely the correct fix for fresh/current data.
   - It is a deliberate behavior change from TypeScript and should be tested
     before changing production data.
5. Add nested tables, celestials, jumps, and flags to `sde:verify`; its current
   output checks only the 14 straightforward table declarations.

## Remaining audit work

- Finish the SDE nested-row investigation above.
- Complete the command-by-command notes for the remaining non-excluded command
  surfaces. Most result paths have already been covered through worker/cron
  auditing, but the final matrix has not yet been written.
- Run an authoritative local-versus-production primary-key/value comparison for
  the imported SDE tables, not only row counts.
- Run the populated-database kill-type classifier agreement test.
- Run the Goose schema comparison explicitly and record whether it passed or
  was skipped due to environment requirements.
- Re-run the status integration test at the final commit.
- Perform the final clean validation:

  ```text
  go test ./... -count=1
  go vet ./...
  go build ./cmd/shrike
  ```

- Produce the final parity matrix with one of:
  - matched
  - intentionally different
  - fixed TypeScript defect
  - explicitly skipped
  - residual risk

Residual risks to call out in the final report:

- A full SDE archive import has run locally and its ordinary counts match
  production, but a full value-by-value comparison is not complete.
- Production-scale throughput has not been benchmarked.
- Memgraph being unavailable is treated as degraded operation because graph
  data is rebuildable; this should remain visible operationally.
- HTTP/API surfaces contain intentional placeholders and remain outside this
  audit.

## Resume point

Start with the three SQL primary-key diffs for
`type_dogma_attributes`, `type_dogma_effects`, and `type_materials`.
The worktree was clean before this log was added, and no source-code change was
left half-finished.
