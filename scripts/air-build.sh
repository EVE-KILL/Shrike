#!/usr/bin/env bash
# Build the Shrike binary for Air, against a bounded repository-local cache.
#
# Air rebuilds on every save. Each rebuild writes new entries into the Go build
# cache, and it does that faster than Go's age-based trimming removes them, so
# a long development session grows the cache without limit. Pointing GOCACHE at
# the repository keeps that churn out of the shared cache, and makes the size
# something this script can measure and reset.
#
# GOMODCACHE is deliberately left alone. It holds downloaded modules, it does
# not grow when sources change, and `go clean -cache` does not touch it — a
# private copy would only re-download the dependency tree for no benefit.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOCACHE="${SHRIKE_AIR_GOCACHE:-$repo_root/.data/go-build}"
output="${SHRIKE_AIR_OUTPUT:-$repo_root/.data/air/shrike}"

mkdir -p "$GOCACHE" "$(dirname "$output")"

# Build beside the target, then rename. Air stops the old process only after
# the build succeeds, so the binary is still executing while this runs. Writing
# into the live file risks a code-signature kill of the running process on
# macOS; rename() swaps the directory entry and leaves it its own inode.
staging="$output.next.$$"
trap 'rm -f "$staging"' EXIT
go build -o "$staging" "$repo_root/cmd/shrike"
mv -f "$staging" "$output"

# Cache entries are disposable, so the cheapest bound is to drop all of them.
# The next build is a full rebuild; that costs one slow restart, which is a
# better trade than an unbounded cache. Override the limit in KiB.
max_kib="${SHRIKE_AIR_CACHE_MAX_KIB:-8388608}"
used_kib="$(du -sk "$GOCACHE" | awk '{print $1}')"
if [ "$used_kib" -gt "$max_kib" ]; then
    echo "air: Go build cache reached $((used_kib / 1024)) MiB (limit $((max_kib / 1024)) MiB); clearing"
    go clean -cache
fi
