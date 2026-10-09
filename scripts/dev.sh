#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
command -v node >/dev/null || { echo 'Node 24+ is required for browser source builds.' >&2; exit 1; }
test -d frontend/node_modules || { echo 'Install browser tooling: npm --prefix frontend ci --ignore-scripts' >&2; exit 1; }
npm --prefix frontend run build -- --dev
browser_pid=''
air_pid=''
cleanup() {
    if [ -n "$browser_pid" ]; then kill "$browser_pid" 2>/dev/null || true; fi
    if [ -n "$air_pid" ]; then kill "$air_pid" 2>/dev/null || true; fi
    wait 2>/dev/null || true
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
(cd frontend && exec node --import tsx scripts/watch-assets.ts --skip-initial) &
browser_pid=$!
air -c "${SHRIKE_AIR_CONFIG:-.air.toml}" &
air_pid=$!
wait "$air_pid"
