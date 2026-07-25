# Shrike

The Go backend for [EVE-KILL](https://eve-kill.com). One binary: API, queue workers,
cron runner, WebSocket relay, feed service.

## Quick start

```sh
cp .env.example .env
docker compose up -d          # postgres 18.3, two valkey, memgraph
make build                    # also symlinks bin/ek
./bin/shrike doctor
./bin/shrike db:migrate --apply
```

## Commands

Both spellings work: `shrike db:migrate` and `shrike db migrate`.

```
doctor          check postgres, both valkey, memgraph
serve           run the HTTP server (Ctrl+C drains and exits)
config:show     resolved config, with the origin of each value
db:status       migration state
db:migrate      apply pending migrations
db:baseline     stamp the baseline as applied without executing it
queue:list      declared queues with concurrency and retry policy
queue:status    live queue depths
queue:verify    diff the registry against live Redis
cron:list       scheduled jobs (--by-frequency to sort by load)
completion      bash/zsh/fish
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

## Local vs production

`.env` is the local stack and the default. `.env.prod` must be asked for:

```sh
shrike doctor                        # local
shrike doctor --config .env.prod     # production
```

The asymmetry is deliberate — reaching production is always opt-in.

## Development

```sh
make check    # fmt, vet, test
make dist     # linux/amd64 + linux/arm64
```

Tests needing Postgres create throwaway databases on the local stack and skip when
it is unreachable, so `go test ./...` passes without docker. Override with
`TEST_DATABASE_URL`.
