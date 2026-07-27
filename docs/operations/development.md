# Run the development stack

`make dev` serves the site at `https://localhost:4001` with Go and the frontend
reloading independently.

## Prerequisites

- Install `air`: `go install github.com/air-verse/air@latest`.
- Install `bun`.
- Start Postgres, Valkey, and Memgraph: `docker compose up -d`.
- Create `.env` from `.env.example`.

## Procedure

1. Run `make dev`.
2. Open `https://localhost:4001`.

Accept the local certificate on the first run. Caddy issues it from its
internal certificate authority.

## What runs

`make dev` starts two processes as siblings:

| Process | Port | Role |
| --- | --- | --- |
| `air` → `shrike dev` | 4001 | Caddy, the API, and the private API on 4002 |
| `nuxt dev` | 3000 | The frontend, with hot module replacement |

Caddy answers `/health`, `/api`, `/auth`, `/images`, and `/ws`. It proxies
every other path to `nuxt dev`.

`air` rebuilds and restarts the Go process alone. `nuxt dev` keeps running
across each Go restart, so the frontend keeps its state and its hot module
replacement connections. `.air.toml` excludes `web/` for this reason.

## Differences from production

Two transports differ. Everything else is wired the same.

Production gives the renderer a Unix socket. `nuxt dev` cannot accept one: it
listens through `listhen`, whose options carry a port and a hostname only. dev
proxies to `127.0.0.1:3000` instead.

Production sends server-side rendering to Go over a second Unix socket. `nuxt
dev` runs under Node, and the Node `fetch` ignores the `unix` request option
that Bun implements. dev serves the same handler on `127.0.0.1:4002` and sets
`NUXT_API_ORIGIN` to it.

`web/nuxt.config.ts` applies the Bun socket entry under `$production`. A
top-level `entry` also replaces the entry that `nuxt dev` needs, which stops
the development server from reporting its address to its parent.

## Change the ports

Set `DEV_PORT`, `DEV_NUXT_PORT`, or `DEV_API_PORT`:

```sh
make dev DEV_PORT=4443 DEV_NUXT_PORT=3100 DEV_API_PORT=4102
```

`shrike dev` reads `--renderer` and `--api-addr`, then
`SHRIKE_DEV_RENDERER` and `SHRIKE_DEV_API_ADDR`. `make dev` sets the
environment variables.

## Verification

- `curl -k https://localhost:4001/api/health` returns `"ok":true`.
- `https://localhost:4001` returns the rendered home page.
- An edit under `web/app/` updates the browser without a reload.
- An edit under `internal/` restarts Go and leaves the frontend running.

## Recovery

A `502` on page requests means Caddy cannot reach `nuxt dev`. Confirm the
process listens on IPv4:

```sh
lsof -nP -iTCP:3000 -sTCP:LISTEN
```

`nuxt dev` binds `[::1]` by default, which the IPv4 upstream cannot reach.
`make dev` passes `--host 127.0.0.1` to prevent this.

A `503` with `Unable to load site configuration` means server-side rendering
cannot reach Go. Confirm the private API is up:

```sh
curl http://127.0.0.1:4002/health
```

`Dev server is unavailable` from port 3000 means the Nitro entry was
overridden. Confirm that `entry` in `web/nuxt.config.ts` stays under
`$production`.

To release a held port, stop every process and retry:

```sh
pkill -f 'air$'; pkill -f 'nuxt dev'
```
