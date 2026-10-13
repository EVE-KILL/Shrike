# Cloudflare image delivery

Shrike serves inventory types, map images, old portraits, UI assets, and
killmail social cards. Browsers load current character portraits and
corporation and alliance logos from `https://images.evetech.net` directly.
Cloudflare caches Shrike's remaining image responses according to their
`Cache-Control` headers.

## Origin storage

Production image storage is a local filesystem selected with:

```text
IMAGE_STORAGE_PATH=/data/images
```

## Capacity eviction

The filesystem store supports a coarse LRU eviction pass based on filesystem
access time. The production NFS mount uses `relatime`, which records whether an
image was used recently without writing on every request. Do not mount the
image volume with `noatime`.

Run the pass periodically from exactly one process:

```bash
shrike images prune --high-watermark 90 --low-watermark 80 --min-age 24h
```

Both byte and inode utilization are checked. Crossing either high watermark
starts eviction; legacy character, corporation, and alliance objects and their
`.shrike-meta.json` sidecars are removed until both are at the low watermark.
Static types, old-character archives, maps, UI assets, and every other tree are
never eligible. Recent dynamic images are protected by `--min-age`. Use
`--dry-run` to inspect the selected amount without deleting files.

On Caparison this directory is backed by the `shrike-images` PVC using the
`local-nvme` StorageClass. The data is deliberately treated as rebuildable:
the volume is fast node-local storage, not a replicated backup.

Shrike stores source and derived objects beneath the same keys previously used
by its S3-compatible backend:

```text
static/regions/
static/systems/
static/constellations/
static/ui/
types/dust514/
types/overlays/
types/
oldcharacters/
entities/                 # legacy cache; no new entity images are stored
social/
```

Filesystem writes use a temporary file and atomic rename. Metadata such as
content type, cache policy, and source digests is stored beside each object.

## Rebuilding an empty volume

Run these commands in a Shrike pod that mounts the image PVC:

```sh
shrike images generate-maps --type all
shrike images import-static
shrike images sync-types
shrike images import-old-characters
```

`generate-maps` produces 1024px systems, constellations, and regions directly
from the imported SDE tables, one image at a time by default. Each map is
rendered once, then 512px, 256px, 128px, 64px, and 32px variants are derived
from that original. Use `--limit 1` to validate one image per selected map type
before rebuilding the full corpus. `import-static`
installs the UI, Dust 514, and overlay
assets bundled into Shrike itself. TurtleTools
and legacy portrait imports download their upstream archives themselves.

The old dynamic `entities/` tree is not needed to serve the site. Once every
running Shrike version uses direct EVE image URLs, remove those old files to
reclaim storage. Keep `oldcharacters/`, `types/`, `static/`, and `social/`.

Inspect the obsolete cache before deletion:

```sh
shrike images purge-entity-cache
```

After all serving pods run this version, delete only those three old trees:

```sh
shrike images purge-entity-cache --apply
```

The command requires `IMAGE_STORAGE_PATH`. It does not touch the old portrait
archive or static image trees. If an old version must be restored, its entity
cache will be empty and it will fetch dynamic images again.

## Caching

Static and content-addressed objects are immutable. Cloudflare should respect
origin cache headers and must not cache `404`, authorization failures, or `5xx`
responses. EVE's image service controls caching for portraits and logos.

## Rollback

Unset `IMAGE_STORAGE_PATH` to return to the configured S3-compatible image
bucket. This fallback is retained temporarily for deployment rollback and can
be removed after the local origin has been verified in production.
