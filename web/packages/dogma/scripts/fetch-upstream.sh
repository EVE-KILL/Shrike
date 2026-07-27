#!/usr/bin/env bash
#
# fetch-upstream.sh
#
# Pulls @eveshipfit/data and @eveshipfit/dogma-engine from GitHub Packages and
# vendors them into dist/upstream/. Run this once per upstream release bump.
#
# Requirements:
#   - gh CLI, authenticated with read:packages scope  (check: gh auth status)
#   - bun
#   - jq
#
# After running, commit the resulting changes under dist/upstream/.

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PACKAGE_DIR="$( cd "$SCRIPT_DIR/.." && pwd )"
UPSTREAM_DIR="$PACKAGE_DIR/dist/upstream"

for cmd in gh bun jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command '$cmd' not found in PATH" >&2
    exit 1
  fi
done

GH_TOKEN="$(gh auth token 2>/dev/null || true)"
if [ -z "$GH_TOKEN" ]; then
  echo "error: could not read token from 'gh auth token' — run 'gh auth login' first" >&2
  exit 1
fi

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

echo "==> Fetching upstream packages into $WORK_DIR"

# .npmrc scopes the eveshipfit registry and supplies the PAT. Lives only in the
# temp dir, and is wiped on exit.
cat > "$WORK_DIR/.npmrc" <<EOF
@eveshipfit:registry=https://npm.pkg.github.com/
//npm.pkg.github.com/:_authToken=${GH_TOKEN}
EOF

cat > "$WORK_DIR/package.json" <<'EOF'
{
  "name": "dogma-fetch-temp",
  "private": true,
  "dependencies": {
    "@eveshipfit/data": "latest",
    "@eveshipfit/dogma-engine": "latest"
  }
}
EOF

(
  cd "$WORK_DIR"
  bun install
)

DATA_DIR="$WORK_DIR/node_modules/@eveshipfit/data"
ENGINE_DIR="$WORK_DIR/node_modules/@eveshipfit/dogma-engine"

if [ ! -d "$DATA_DIR" ]; then
  echo "error: @eveshipfit/data not installed — check GitHub Packages access" >&2
  exit 1
fi
if [ ! -d "$ENGINE_DIR" ]; then
  echo "error: @eveshipfit/dogma-engine not installed — check GitHub Packages access" >&2
  exit 1
fi

DATA_VERSION="$(jq -r .version "$DATA_DIR/package.json")"
ENGINE_VERSION="$(jq -r .version "$ENGINE_DIR/package.json")"

echo "==> @eveshipfit/data@${DATA_VERSION}"
echo "==> @eveshipfit/dogma-engine@${ENGINE_VERSION}"

# Replace dist/upstream/ atomically-ish. Stage into a sibling dir, then swap.
STAGING_DIR="${UPSTREAM_DIR}.new"
rm -rf "$STAGING_DIR"
mkdir -p "$STAGING_DIR"

cp -R "$DATA_DIR"   "$STAGING_DIR/data"
cp -R "$ENGINE_DIR" "$STAGING_DIR/dogma-engine"

# Strip node_modules that npm may have pulled in as transitive deps of the
# vendored packages — we only want the published package contents.
rm -rf "$STAGING_DIR/data/node_modules" "$STAGING_DIR/dogma-engine/node_modules"

# Strip the big directories we don't need for pure fit calculation:
#   sde_json/  — 66MB JSON mirror of the protobuf (redundant)
#   icons/     —  4MB module icons (we have our own image CDN)
#   ui/        —  2MB React UI assets (not using the component lib)
# Takes dist/upstream from ~82MB to ~10MB.
rm -rf "$STAGING_DIR/data/dist/sde_json" \
       "$STAGING_DIR/data/dist/icons" \
       "$STAGING_DIR/data/dist/ui"

cat > "$STAGING_DIR/VERSION.json" <<EOF
{
  "data": "${DATA_VERSION}",
  "dogma_engine": "${ENGINE_VERSION}",
  "fetched_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

rm -rf "$UPSTREAM_DIR"
mv "$STAGING_DIR" "$UPSTREAM_DIR"

echo ""
echo "==> Vendored to $UPSTREAM_DIR"
echo ""
du -sh "$UPSTREAM_DIR"/* 2>/dev/null || true

# Regenerate the static-module protobuf classes from the freshly fetched
# .proto schema. This file (src/generated/esf.js) is what core.ts imports —
# it bakes decode/toObject as plain JS functions so we don't need runtime
# codegen (which would require `unsafe-eval` in our CSP). See core.ts for
# the rationale, and src/generated/esf.d.ts for the hand-written type shim.
echo ""
echo "==> Regenerating static-module protobuf classes (src/generated/esf.js)"
(
  cd "$PACKAGE_DIR"
  bun run gen:proto
)

echo ""
echo "Next: git add packages/dogma/dist/upstream/ packages/dogma/src/generated/esf.js && git commit"
