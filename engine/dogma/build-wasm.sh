#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cargo_tool="${CARGO:-cargo}"
bindgen_tool="${WASM_BINDGEN:-wasm-bindgen}"
if [ "$("$bindgen_tool" --version)" != "wasm-bindgen 0.2.128" ]; then
    echo 'Install wasm-bindgen-cli 0.2.128: cargo install wasm-bindgen-cli --version 0.2.128 --locked' >&2
    exit 1
fi
"$cargo_tool" build --locked --lib --features browser --profile wasm --target wasm32-unknown-unknown --manifest-path "$repo_root/engine/dogma/Cargo.toml"
staging="$(mktemp -d)"
trap 'rm -rf "$staging"' EXIT
"$bindgen_tool" --target bundler --out-dir "$staging" --out-name esf_dogma_engine "$repo_root/engine/dogma/target/wasm32-unknown-unknown/wasm/shrike_dogma.wasm"
# Bindgen shares memory-view caches between instances. Clear them whenever our
# facade selects an instance, so two editor factories cannot read each other.
node --input-type=module - "$staging/esf_dogma_engine_bg.js" <<'JS'
import {readFile,writeFile} from 'node:fs/promises'
const path=process.argv[2],source=await readFile(path,'utf8')
const marker='    wasm = val;'
if(!source.includes(marker))throw new Error('wasm-bindgen setter changed; inspect instance cache reset')
await writeFile(path,source.replace(marker,marker+'\n    cachedDataViewMemory0 = null;\n    cachedUint8ArrayMemory0 = null;'))
JS
out="$repo_root/frontend/dogma/dist/upstream/dogma-engine"
cp "$staging"/* "$out/"
node "$repo_root/frontend/scripts/dogma-source.mjs"
