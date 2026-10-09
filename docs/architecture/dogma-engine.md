# Dogma engine

Shrike uses upstream Rust v13.6.0 for server and browser fitting calculations.
The server executes persistent native workers; the browser executes the same
adapter compiled to WASM. Go's HTTP/API behavior and existing fit document URLs
remain unchanged. Node is build/test tooling only.

## Upgrade analysis: 7.1.0 to 13.6.0

The previous additions were in the platform adapter and SDE layer, rather than
changes to the Rust calculation source. The audit covered the old browser core,
protobuf callbacks, native projection, data converter and regression snapshots.
The old implementation is archived at
`.data/source-archives/dogma-7.1.0-before-upgrade.tar.gz`.

| Previous addition | Current treatment |
| --- | --- |
| Six hull totals for remote shield/armor/hull repair, NOS, neutralizers and remote capacitor transfer | Retained in shared Rust. Sum upstream's computed per-item rates; keep remote/local and NOS/transmitter categories distinct. |
| Inject rate effects into ancillary shield boosters and armor repairers | Removed. The matching upstream SDE selectors already include fueled modules and calculate paste/charge behavior. |
| Overlay turret ammunition range/falloff/tracking multipliers | Removed. Upstream applies the actual `weaponRangeMultiplier`, `fallofMultiplier` and `trackingSpeedMultiplier`. Tests ensure one application and unchanged EWAR activation range when scripting. |
| Default all-level-five skills, canonical server module states and one active propulsion module | Retained. Browser state controls remain explicit. |
| Legacy module/slot/result shapes used by editor, EFT and saved links | Retained at the adapter boundary. Passive rigs map to fitted online rigs; old passive subsystem documents benefit from upstream's forced-online fix. |
| JS/protobuf lookups called from WASM | Replaced with direct FlatBuffer access inside Rust. Browser catalogue metadata comes from the same data file. |
| Go CCP-to-protobuf converter | Replaced with the pinned upstream Go FlatBuffer loader/patcher/writer. Existing archive caching and staged replacement are retained. |

The engine and SDE are a matched pair. v13.6.0 was released on 2026-10-06 after
v13.5.0, and requires SDE major version 12. The pin is recorded in
`frontend/dogma/dist/upstream/VERSION.json` and the converter provenance.
Sources: https://github.com/EVEShipFit/dogma-engine/releases/tag/v13.6.0 and
https://github.com/EVEShipFit/sde-patched/releases/tag/v12.3579973.0.

## Changed reference results

All 49 fields were compared for the original 16 reference fits. Remote totals,
weapons, resistance fields and most hull values remain equivalent. Retained
7.1 snapshots make the differences explicit instead of preserving obsolete math:

- Target counts now use the pilot-limited `maxTargets`. A character with no
  trained skills can have zero effective targets; the all-V Archon is limited
  to 10 rather than the previous raw hull limit of 14.
- Vargur mass is 120,000,000 kg rather than 150,000,000 kg in the newer dataset.
  The difference also exists on the bare hull, without Bastion. Align time
  changes slightly; the adapter preserves the upstream result.
- The Armageddon with NOS gains the expected 12 GJ/s in capacitor balance.
  Its all-V depletion time changes from 120 to 156 seconds; with no skills,
  60 to 72 seconds. Neutralizer/remote totals remain separate.

`engine/dogma/testdata/expected-7.1.0.jsonl` and `expected.jsonl` retain the full
comparison. Native and WASM must match within relative 1e-9 / absolute 1e-7
on the current dataset; focused tests check the individual fixes independently.

## Development and rollout

`make build` builds Go and the native worker. `make build-dogma-wasm` rebuilds
both the shared WASM adapter and its checked-in JS bundle. CI checks the WASM
source checksum, native protocol/cancellation/restart, reference fits, browser
regressions and native/browser equality. No browser calculation requires Bun,
Node, or a server request.

`sde:build-dogma` uses the pinned upstream Go converter and patches to generate
`sde.dat`, `names.dat` and `texts.dat` from the cached official CCP archive.
The app embeds `sde.dat`; the optional name/text datasets remain available for
future multilingual imports and descriptions without enlarging normal downloads.

Stored catalogue statistics are tagged **13.6.0+ek.1 / 3579973**. Existing rows
from 7.1.0+ek.3 are not relabeled. Metric filters and histograms intentionally
require the current engine/data pair, so deploying this upgrade includes a
fitting-stat backfill and distribution rebuild. This development task does not
modify production statistics or run production jobs.
