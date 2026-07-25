# Shrike

The Go backend for [EVE-KILL](https://eve-kill.com) — one binary covering the API,
queue workers, cron runner, WebSocket relay, and feed service.

Shrike replaces a set of Bun/TypeScript services (`backend/`, `api/`, `websocket/`,
`relay/`) with a single deployable. It is built and validated against a local stack
first, then cut over; there is no dual-running bridge between the two
implementations.

## Quick start

```sh
cp .env.example .env          # fill in, or point at your own stack
docker compose up -d          # postgres 18.3, two valkey, memgraph
make build
./bin/shrike doctor           # verify every dependency is reachable
./bin/shrike db:migrate --apply
```

`make build` also drops an `ek` symlink next to the binary, so `ek queue:status`
works as a short form.

## Commands

Every command accepts both spellings — `shrike db:migrate` and `shrike db migrate`.
The colon form matches the naming of the Bun CLI it replaces; the space form is the
Cobra tree underneath, and is what shell completion completes.

```
doctor          verify connectivity to postgres, both valkey, memgraph
serve           run the HTTP server (foreground; Ctrl+C drains and exits)
config:show     resolved config with the origin of every value
db:status       migration state and what would be applied
db:migrate      apply pending migrations
db:baseline     stamp the baseline as applied without executing it
queue:list      declared queues with concurrency and retry policy
queue:status    live queue depths
queue:verify    diff the registry against what is live in Redis
cron:list       scheduled jobs and intervals (--by-frequency to sort by load)
completion      shell completion for bash/zsh/fish
```

`--json` on any command gives machine-readable output with all decoration
suppressed. `--config <file>` selects an env file.

## Migrations

Migration 1 is a `pg_dump` of the production schema. **Production must never run
it** — those 102 tables already exist there, created by drizzle and tracked in a
separate ledger. Executing migration 1 would try to `CREATE TABLE` over live data.

For any database that already has the schema:

```sh
shrike db:baseline --apply    # writes the ledger only; runs zero statements
```

`db:migrate` independently refuses when it finds tables but no goose ledger, so the
mistake is blocked from both directions. Both commands require `--apply`; without it
they print a plan and change nothing.

## Local vs production

`.env` is the local docker stack and is the default. `.env.prod` points at
production and must be selected explicitly:

```sh
shrike doctor                        # local
shrike doctor --config .env.prod     # production
```

That asymmetry is deliberate. A stray worker or cron run against production is a
real hazard, so reaching it is always opt-in.

## Layout

```
cmd/shrike/          entrypoint
internal/cli/        cobra commands, help renderer, colon-name bridge
internal/config/     env + .env loading with source tracking
internal/db/         pgx pool (tuned for pgbouncer)
internal/jobs/       canonical queue + cron registry
internal/migrate/    goose wrapper with the baseline guards
internal/ui/         all human-facing output; nothing else touches lipgloss
migrations/          embedded SQL
```

## Development

```sh
make check    # fmt, vet, test
make dist     # linux/amd64 + linux/arm64
```

Tests that need Postgres create throwaway databases on the local stack and skip
when it is unreachable, so `go test ./...` passes without docker running. Point
`TEST_DATABASE_URL` elsewhere to override.
