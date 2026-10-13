# Pinned fitting data

This directory contains the EVEShipFit SDE **12.3579973.0** FlatBuffer paired
with dogma-engine **13.6.0**. Version and source pins are recorded in
`../VERSION.json`; `specs/eve.fbs` is the matching schema.

The browser and native worker both read `dist/sde.dat`. There are no Protobuf
files or JavaScript data lookup callbacks in the current engine.

Shrike's `sde:build-dogma` command regenerates the data using the official CCP
archive and the pinned Go converter/patches under
`internal/dogmadata/upstream`. It also writes optional `names.dat` and
`texts.dat`; those are not embedded in the normal fitting editor download.
Using `--output` for an alternate directory leaves the application's version
manifest unchanged.

Upgrade the schema, converter, dataset, native engine and browser WASM together.
See `docs/architecture/dogma-engine.md` and the converter's `UPSTREAM.md` for
provenance and rollout instructions. Both `LICENSE` and `LICENSE.EVE` are retained.
