# Shared fitting adapter

The calculation code is EVEShipFit dogma-engine **v13.6.0**, commit
`08ae0ac9ebdd4d69390770181d4e291cf92a0cce`:
https://github.com/EVEShipFit/dogma-engine/tree/v13.6.0

`vendor/esf-data` and `vendor/esf-dogma-engine` contain the unmodified upstream
crates. Their MIT licenses are retained. The root Cargo workspace supplies the
same inherited dependency definitions. The matching SDE is **12.3579973.0**;
its source/schema/patches are pinned in `internal/dogmadata/upstream`.

`src/lib.rs` is Shrike's adapter, compiled once for the native worker and again
for browser WASM. It translates existing fit documents, module states, rack
indices and computed attributes into the application contract. Legacy passive
rigs mean fitted rigs, so they map to upstream's online state. Subsystems retain
upstream's forced-online behavior. `compat.json` is the single source for the
six outgoing totals and the effective target-count alias.

Upstream calculates individual repair and capacitor transfer rates, including
skills, ship bonuses, ancillary charges and spool. The adapter only totals those
rates, keeping local repairs, remote repairs, NOS, neutralizers and capacitor
transmitters distinct. The previous ancillary-effect injection and ammunition
bonus overlay are removed. Ammo modifiers are handled by upstream.

Go starts native workers on demand, each loading the FlatBuffer once. JSON lines
on stdin/stdout retain all 49 hull fields. Cancellation replaces a busy worker.
The browser loads the same SDE directly into WASM once, with no JS callbacks
for individual data lookups. The same bytes also provide catalogue metadata.

Build native: `make build-dogma`.
Build WASM and its browser bundle: `make build-dogma-wasm` (Rust target
`wasm32-unknown-unknown`, wasm-bindgen-cli **0.2.128**, Node 24+).
WASM is committed; ordinary Go development needs no WASM build tools.
`frontend/scripts/dogma-source.mjs --check` verifies its source and artifact
checksums. The builder clears bindgen memory-view caches when switching instances,
so independent editor factories cannot read each other's WASM memory.

`testdata/expected-7.1.0.jsonl` preserves the previous results. Current snapshots
and native/browser parity checks cover the same 16 fits. Focused regressions
cover ammo optimal/falloff/tracking, EWAR scripts, ancillary repairs, remote totals,
rigs, subsystems, and independent WASM factories. See the upgrade analysis in
`docs/architecture/dogma-engine.md` for the retained additions and changed results.

Upgrade native crates, browser WASM, FlatBuffer schema/data and the converter
pin together. Bump the stored-stat version when calculation semantics change;
rebuild fitting statistics and their distributions during the deployment rollout.
