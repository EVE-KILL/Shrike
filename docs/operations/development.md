# Run the development stack

`make dev` serves the Go site at `http://localhost:4000` and rebuilds it when source files change.

## Prerequisites

- Install Go at the version specified in `go.mod`.
- Install Air: `go install github.com/air-verse/air@latest`.
- Install Docker with Compose support.
- Install Rust/Cargo to build the native fitting worker (`make build-dogma`).
- Install Node 24+ and browser dependencies: `npm --prefix frontend ci --ignore-scripts`.

## Procedure

1. Create `.env` from `.env.example` if the file does not exist.
2. Configure the local services in `.env`:

   ```dotenv
   DATABASE_URL=postgresql://evekill:evekill@127.0.0.1:5432/evekill?sslmode=disable
   DATABASE_READ_URL=postgresql://evekill:evekill@127.0.0.1:5432/evekill?sslmode=disable
   REDIS_HOST=127.0.0.1
   REDIS_PORT=6379
   REDIS_PASSWORD=
   REDIS_DB=0
   MEMGRAPH_URL=bolt://127.0.0.1:7687
   NODE_ENV=development
   PORT=4000
   ```

3. Start the dependencies: `docker compose up -d postgres valkey memgraph`.
4. Build the application: `make build`.
5. Check connectivity: `./bin/shrike doctor`.
6. Inspect the schema: `./bin/shrike db:migrations`.
7. Apply pending application migrations: `./bin/shrike db:migrate --apply`.
8. Apply queue migrations: `./bin/shrike queue:migrate`.
9. Start the site: `make dev`.
10. Open `http://localhost:4000`.

`make dev DEV_PORT=4001` selects another port. Stop the server with Ctrl+C.
Local PostgreSQL data persists under `.data/postgres`; restarting Compose preserves it.

## Verification

```sh
curl --fail http://localhost:4000/health
curl --fail http://localhost:4000/
```

All migrated pages are served by Go/templ; unknown routes return HTTP 404.

All HTTP surfaces run in Go: pages, API, authentication, images, and WebSockets.
The backend starts native Rust fitting workers on demand. Their calculation
code and the browser WASM remain pinned to the same upstream engine and SDE.

Templates and frontend assets live in `internal/frontend`. Air regenerates templates and rebuilds Go after source changes.
Open pages share one development reload stream and reload automatically after each Air rebuild. Browsers without a working SharedWorker use short version checks instead, so multiple tabs cannot reserve all HTTP connections. Generated Go files are checked in; the templ tool version is pinned in `go.mod`.
Browser sources live in `frontend/src/browser`; JavaScript output is generated in
`internal/frontend/static`. `make dev` starts a source watcher alongside Air, so
TypeScript, styles and relevant page-source changes rebuild and reload
automatically. Development output includes readable code and linked source maps.
Production builds are minified and remove owned development maps.
All authored browser controllers are strict TypeScript. Stimulus owns component
actions and lifecycle; pure helpers and transport remain ordinary typed modules.
Edit sources under `frontend/`, rather than the generated JavaScript.

```sh
npm --prefix frontend ci --ignore-scripts
make build-browser-assets
make test-frontend
```

`make build` compiles browser sources before Go. A direct `go build` can still use
checked-in production bundles. Asset-group Make targets remain available; JS
changes rebuild the complete shared graph. Rebuild Rust changes with
`make build-dogma` and restart Go to replace its existing worker pool. See
[the browser source guide](../architecture/browser-assets.md).

`DOGMA_WORKER_PATH` overrides the native executable (default `bin/shrike-dogma`).
`DOGMA_SDE_DIR` overrides its FlatBuffer directory; `DOGMA_WORKER_PROCESSES`
sets the pool size (default 4, range 1–32). Both worker and data paths are
explicitly configured in the production image.

Custom-domain requests retain their original Host header. Test an existing board with:

```sh
curl --fail -H 'Host: void.localhost:4000' http://localhost:4000/api/site
```

Custom boards use their configured widgets, themes, navigation and entity scope.

## Recovery

If startup reports a database connection failure, run `docker compose ps` and `./bin/shrike doctor`.
Confirm both database URLs point to the local database. Process environment variables override `.env`.

If migrations report existing tables without a ledger, inspect the database before using `db:baseline`.
Do not baseline an empty database: it needs the initial schema migration.

If port 4000 is occupied, stop its existing server or choose another `DEV_PORT`.

`dev` uses HTTP. To test HTTPS, stop it and run `./bin/shrike serve --port 4000` instead.
Outside production, `serve` uses Caddy's local certificate authority. On macOS, `make dev-trust` installs its certificate.

Images use the local Go image service. The previous Nuxt remote-image proxy is removed.
Configure `IMAGE_STORAGE_PATH` or the image object-store settings when image testing is needed.

Shared Rust adapter changes require `make build-dogma-wasm` as well as the native build. Install the Rust WASM target and wasm-bindgen-cli 0.2.128 for that optional task; checked-in WASM remains sufficient for ordinary development.
