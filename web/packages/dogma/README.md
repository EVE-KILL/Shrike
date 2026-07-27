# @evekill/dogma

Ship fitting statistics calculation, powered by the [EVEShipFit dogma engine](https://github.com/EVEShipFit/dogma-engine).

## What this is

A thin wrapper around EVEShipFit's Rust→WASM dogma engine and protobuf-flattened SDE data. Exposes a single `calculateFit(fit, skills?)` entry point that computes DPS, EHP, cap stability, speed, tank, and all the other ship statistics you'd see on eveship.fit.

## How the vendoring works

EVEShipFit publishes `@eveshipfit/data` and `@eveshipfit/dogma-engine` to GitHub Packages, which requires authentication even for public package reads. To avoid making every developer and CI job juggle a PAT, we pull those packages **once** via `scripts/fetch-upstream.sh` and commit the extracted files into `dist/upstream/`.

After cloning this repo, you don't need to do anything. The vendored artifacts are already there.

## Updating the engine or SDE data

When EVEShipFit ships a new version:

```
cd packages/dogma
bun run fetch-upstream
```

Requirements:
- `gh` CLI authenticated with `read:packages` scope (check with `gh auth status`)
- `bun` installed
- `jq` installed

The script will pull the latest versions, replace `dist/upstream/`, and update `VERSION.json`. Commit the changes.

## Usage

```ts
import { calculateFit, allFive } from "@evekill/dogma";

const fit = {
  ship_type_id: 587, // Rifter
  modules: [
    // ... EsfModule entries
  ],
  drones: [],
};

const stats = await calculateFit(fit, allFive());
console.log(stats.dps, stats.ehp);
```

## License

This package wraps MIT-licensed upstream EVEShipFit projects. The SDE-derived protobuf blobs in `dist/upstream/data/` are subject to CCP's EVE Online Third-Party Developer License (see the `LICENSE.EVE` file shipped inside that folder by upstream).
