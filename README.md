# Shrike

The self-contained [EVE-KILL](https://eve-kill.com) application: Go API, queue
workers, cron runner, WebSocket relay and image service, plus the Nuxt renderer
and its vendored dogma engine.

## Quick start

```sh
cp .env.example .env
docker compose up -d          # postgres 18.3, two valkey, memgraph
make build                    # also symlinks bin/ek
./bin/shrike doctor
./bin/shrike db:migrate --apply
```

## Container

One image is built for every workload:

```sh
docker build -t evekill .
docker run evekill serve
docker run evekill work:queues
docker run evekill work:cron
docker run evekill work:zkb
```

`serve` is the supervisor for the whole site. It starts the Bun/Nuxt renderer,
starts a private Go HTTP listener for SSR, and then starts Shrike's embedded
Caddy front door. Caddy-to-Nuxt and Nuxt-to-Go use separate Unix sockets.
Browser API calls remain relative and same-origin. Every other command runs
only the Go binary, so worker and cron pods do not start Bun or require a second
application image. The image is multi-architecture: Docker's target platform
controls both the Go binary and Nuxt's native image dependencies.

In a source checkout, build the renderer once before running the Go command
directly:

```sh
make build-frontend
go run ./cmd/shrike serve
```

The renderer process is supervised by Go: an early crash fails startup, and a
later crash shuts down the service instead of leaving Caddy returning 502s.

## Documentation

Read the [documentation index](docs/README.md) for operational guides.

All new documents follow the [documentation style](docs/STYLE.md). The style
uses a practical subset of ASD-STE100 and George Orwell's writing rules.

## Commands

Both spellings work: `shrike db:migrate` and `shrike db migrate`.

```
doctor              check postgres, both valkey, memgraph
serve               run the HTTP server (Ctrl+C drains and exits)
config:show         resolved config, with the origin of each value
db:status           migration state
db:migrate          apply pending migrations
db:baseline         stamp the baseline as applied without executing it
queue:list          declared queues with concurrency and retry policy
queue:status        live BullMQ queue depths (the Bun workers' queues)
queue:jobs          live River queue depths (the Go queues)
queue:verify        diff the registry against live Redis
queue:ported        which queues have a Go worker
queue:migrate       apply River's own schema migrations
cron:list           scheduled jobs (--by-frequency to sort by load)
cron:status         which crons have a Go implementation
cron:run            run one scheduled job now, in the foreground
work:zkb            follow the zKillboard R2Z2 feed
work:queues         consume the job queues
work:cron           schedule and run the recurring jobs
sde:status          loaded SDE build vs the published one
sde:import          import the static data export (--only to scope it)
sde:verify          row counts per SDE table
prices:status       what market history is loaded
everef:insurance    replace insurance payouts with the current snapshot
everef:prices       daily Jita market history (--days, --date, --from/--to, --backfill)
everef:sovereignty  the sovereignty map and its history (--latest, --from/--to)
everef:wars         wars and the killmails fought under them (--current, --from/--to)
everef:killmails    daily killmail archives (--backfill, --from/--to, --skip-existing)
killmail:process    fetch from ESI, parse, store (--dry-run, --from-file)
killmail:show       print a stored killmail
killmail:compare    diff a stored killmail against another database
killmail:cache      what the runtime lookup cache holds
completion          bash/zsh/fish
```

`--json` for machine-readable output, `--config <file>` to pick an env file.

## Migrations

Migration 1 is a `pg_dump` of the production schema, so **production must never run
it** — those tables already exist there and it would `CREATE TABLE` over live data.
Any database that already has the schema gets stamped instead:

```sh
shrike db:baseline --apply    # writes the ledger only, runs zero statements
```

`db:migrate` refuses when it finds tables but no goose ledger, so the mistake is
blocked from both directions. Both commands need `--apply`; otherwise they print a
plan and change nothing.

## Workers

Three processes, separate on purpose:

```sh
shrike queue:migrate          # once — River owns its own schema
shrike work:zkb               # follows the live feed, enqueues what arrives
shrike work:queues            # consumes the queues
shrike work:cron              # schedules and runs the recurring jobs
```

The feed reader must never be blocked by a slow job, and the cron process holds a
leader election that exactly one replica wins — so queues can be scaled out while
scheduling stays singular. That last part is the main thing the Bun cron runner
could not do: a second instance of it meant every job ran twice.

Scheduled jobs run on their own queue, on the cron process, so a twenty-minute
nightly rebuild cannot occupy a worker slot that killmails need. A cron already
queued or running is never queued again, so one that overruns its interval does
not stack copies of itself.

Jobs live in Postgres rather than Redis. The backlog is a table, so "what is stuck
and why" is a query anyone can run, a job and the rows it is about commit together,
and a Redis flush no longer loses work.

Two things are deliberately visible rather than hidden while the port is in
progress. `queue:ported` and `cron:status` print what has a Go implementation and
what does not, and a worker consumes only the queues it can actually handle —
consuming an unported queue would fetch its jobs, find no worker, and drain the
backlog into the failure table instead of leaving it to wait.

ESI-dependent queues pause automatically while Tranquility is down. The error
limit is global and shared, so continuing to ask during a downtime exhausts a
budget that takes longer to recover than the downtime itself.

### The two killmail ingest paths

R2Z2 embeds the full ESI killmail in every feed entry, so a live kill costs one
request to zKillboard and nothing from the ESI budget — no hash lookup, no
`/killmails/` round trip, no error-limit exposure. The ESI fetcher is for backfill
and repair, where only an id and a hash are known.

The feed is ephemeral: entries expire after hours. A listener that falls further
behind than that cannot catch up by following the sequence, and the
`missed_killmails` cron repairs the gap from zKillboard's daily history index
instead.

## Local vs production

`.env` is the local stack and the default. `.env.prod` must be asked for:

```sh
shrike doctor                        # local
shrike doctor --config .env.prod     # production
```

The asymmetry is deliberate — reaching production is always opt-in.

## Verifying a port

Each ported component is checked against the one it replaces rather than against
its own tests. For killmails that means processing a mail production already holds
and diffing every column of every row:

```sh
shrike killmail:process 137258027 1d9365aaed385213867e40390d29cd4c7596e0e3
shrike killmail:compare 137258027 --against .env.prod
```

Three caveats when reading a diff:

- Stored ISK values reflect the prices available when production processed the
  mail, so a mail parsed today can legitimately differ. Compare against the same
  formula run over current prices instead.
- ESI does not guarantee a stable attacker order between requests, and EVE Ref's
  archives can hold a different order again. Use `--from-file` to replay one saved
  response so both sides see identical input.
- Snapshot datasets — insurance, and anything CCP recalculates — drift with the
  time of fetch. The key set is what has to match; the values move.

## Development

```sh
make check    # fmt, vet, test
make dist     # linux/amd64 + linux/arm64
```

Tests needing Postgres create throwaway databases on the local stack and skip when
it is unreachable, so `go test ./...` passes without docker. Override with
`TEST_DATABASE_URL` and `TEST_REDIS_ADDR`.

Three tiers, and most of the suite is in the first:

- **No dependencies.** Parsers, schedules, rate limiters, priority mapping, the
  feed listener's failure policies. Real bytes from `testdata/`, fake clocks, and
  `httptest` servers — no network, no database, no sleeping.
- **Redis.** The ESI rate limiter and coordination locks. Each package uses its own
  Redis database, because `go test ./...` runs packages in parallel and two of them
  clearing one keyspace produced a phantom lock failure once already.
- **Postgres.** Anything asserting what the database actually did: deduplication
  collapsing a repeated job, a declared retry budget reaching the stored row, the
  Tranquility gate pausing real queue rows, a killmail arriving twice being stored
  once.

Tests are written to fail with a sentence explaining what breaks, not just which
assertion tripped, and each suite is checked by breaking the code it covers and
confirming the failure names the defect.
