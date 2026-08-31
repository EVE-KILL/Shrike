# Run the development stack

`make dev` serves the site at `https://localhost:4001` with Go and the frontend
reloading independently.

## Prerequisites

- Install `air`: `go install github.com/air-verse/air@latest`.
- Install `bun`.
- Start Postgres, Valkey, and Memgraph: `docker compose up -d`.
- Create `.env` from `.env.example`.

## Procedure

1. Run `make dev`. The first run generates the local certificate authority.
2. In another terminal, run `make dev-trust` once on macOS. Enter the password
   in this foreground command; Caddy deliberately does not invoke `sudo` from
   inside Air.
3. Open `https://localhost:4001`.

Caddy issues the certificate from its internal certificate authority. If
`DATA_DIR` is overridden, pass the same value to `make dev-trust`.

## What runs

`make dev` starts two processes as siblings:

| Process | Port | Role |
| --- | --- | --- |
| `air` → `shrike dev` | 4001 | Caddy, the API, and the private API on 4002 |
| `nuxt dev` | 3000–3009 | The frontend, with hot module replacement |

Caddy answers `/health`, `/api`, `/auth`, `/images`, and `/ws`. It proxies
every other path to `nuxt dev`.

`air` rebuilds and restarts the Go process alone. `nuxt dev` keeps running
across each Go restart, so the frontend keeps its state and its hot module
replacement connections. `.air.toml` excludes `web/` for this reason.

## Open a custom killboard

A custom killboard answers on its own hostname. Development maps each board to
`https://<subdomain>.localhost:4001`, where `<subdomain>` is the `subdomain`
column of the `custom_domains` row.

### Procedure

1. Confirm the board row is active in `custom_domains`.
2. Open `https://<subdomain>.localhost:4001`.

Caddy issues a certificate for the hostname at the first request. The local
certificate authority signs it, so the browser needs no new approval.

macOS and Linux resolve every `*.localhost` name to the loopback address. Add
the name to `/etc/hosts` on a system that does not.

### Verification

Run this command for a board named `void`:

```sh
curl --fail --show-error https://void.localhost:4001/api/site
```

The response must contain `"subdomain":"void"`. The rendered page must show
the board name in its title.

### Recovery

A `no alternative certificate subject name` error means an old wildcard
certificate is still stored. Delete it and restart:

```sh
rm -rf ./data/caddy/certificates/local/wildcard_.localhost
```

A page that renders the main site on a board hostname means the renderer host
did not reach Go. Confirm the private API reads the forwarded host:

```sh
curl -H 'X-Forwarded-Host: void.localhost:4001' http://127.0.0.1:4002/api/site
```

## Differences from production

Three transports differ. Everything else is wired the same.

Production gives the renderer a Unix socket. `nuxt dev` cannot accept one: it
listens through `listhen`, whose options carry a port and a hostname only. dev
proxies to `127.0.0.1:3000` instead.

Production sends server-side rendering to Go over a second Unix socket. `nuxt
dev` runs under Node, and the Node `fetch` ignores the `unix` request option
that Bun implements. dev serves the same handler on `127.0.0.1:4002` and sets
`NUXT_API_ORIGIN` to it.

Production carries the browser host to Go in the `Host` header. The Node
`fetch` drops that header, because scripts cannot set it. dev sends
`X-Forwarded-Host` as well, and the private API adopts it as the request host.
The public listener still ignores the header, so no client can claim another
board.

`make dev` selects the first free Nuxt port from 3000 through 3009 and passes
that exact port to both Nuxt and Shrike. This prevents Nuxt's automatic port
fallback from leaving Shrike pointed at another process. Set `DEV_NUXT_PORT`
explicitly to override the selection.

`web/nuxt.config.ts` applies the Bun socket entry under `$production`. A
top-level `entry` also replaces the entry that `nuxt dev` needs, which stops
the development server from reporting its address to its parent.

## Build cache

`air` builds through `scripts/air-build.sh`. The script sets `GOCACHE` to
`.data/go-build`, so rebuilds do not enter the shared Go cache.

A cold build writes about 935 MiB. Each rebuild after that adds about 25 MiB.

The script measures the cache after each build. Above 8 GiB it runs `go clean
-cache`, and the next build is a full rebuild. A cold cache reaches that limit
after about 290 rebuilds.

Set `SHRIKE_AIR_CACHE_MAX_KIB` to change the limit. Set `SHRIKE_AIR_GOCACHE` to
move the cache.

```sh
make dev SHRIKE_AIR_CACHE_MAX_KIB=16777216
```

`.data/` is not tracked, so the cache never enters a commit. Delete
`.data/go-build` to reclaim the space at any time.

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
