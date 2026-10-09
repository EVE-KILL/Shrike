# Browser assets

Go/templ renders every page in `internal/frontend`. This directory contains
browser helper sources, their tests, the generated API contract, and the pinned
Rust WASM/SDE used for interactive fitting calculations. It has no page server,
Vue application or server-rendering runtime.
Stimulus supplies lifecycle, targets and event bindings for the Go-rendered DOM.
Browser sources share one build graph, with `site.js` plus page-specific modules.

Use Node 24+:

```sh
npm ci --ignore-scripts
npm test
npm run typecheck
npm run build
```

The build writes browser bundles to `internal/frontend/static` and settings
metadata to `internal/frontend/data`. Generated assets are committed and embedded
in Go. `npm run build -- map css` rebuilds the shared JS graph and utility CSS; the root Makefile
also exposes `build-<group>-frontend` targets. Asset writers leave unchanged files
alone, so rebuilding utility CSS under Air cannot create a reload loop.

- `src/browser`: browser controllers, shared Stimulus application and module entry points.
- `src/utils`: shared browser behavior and data transformations.
- `src/models`: framework-independent types, settings and navigation metadata.
- `src/fit`: fitting document/EFT interchange helpers.
- `shared/api`: types generated from the Go/Huma OpenAPI document.
- `styles`: theme, utilities, typography and admin preview styles.
- `dogma`: browser WASM adapter, vendored data and calculation regressions.
- `scripts`: browser entry points, data generation and bundle builders.
- `test`: interaction/controller regressions and generated-asset integrity.

The small Go module boundary exposes only embedded dogma assets and prevents
Go tooling from discovering installed package copies under `node_modules`.
Server fitting calculations use `engine/dogma`, not the filesystem JS test adapter.
The native and browser targets share the Rust compatibility adapter at version
13.6.0, using the matching 12.3579973.0 FlatBuffer. Upgrade native Rust, browser
WASM, schema and SDE together. See [the engine audit](../docs/architecture/dogma-engine.md)
for retained extensions, changed results and the deployment backfill step.

`src/fit/document.ts` exports the editor's load/save helpers. Its generated
`fit-document.js` bundle uses `toServer.ts` as the only save serializer; rebuild
it with `make build-fit-document-frontend` after changing either source.

All JavaScript assets are now generated from this source project. The shared
module graph, development watcher, source maps, ownership manifest and incremental
TypeScript policy are documented in [browser-assets.md](../docs/architecture/browser-assets.md).
Use `make dev` for Go plus automatic browser builds, or `npm run dev` for the
asset watcher alone. Existing JavaScript controllers remain supported alongside
checked TypeScript primitives.
