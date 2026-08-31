# Cloudflare image delivery

Shrike owns the canonical image routes at `https://eve-kill.com/images/...` and
the compatibility image hostnames. Cloudflare caches successful responses at
the edge according to Shrike's `Cache-Control` headers.

## Origin storage

Production image storage is a local filesystem selected with:

```text
IMAGE_STORAGE_PATH=/data/images
```

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
entities/
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
from the imported SDE tables, one image at a time by default. The image service
derives the supported smaller variants from those originals. `import-static`
installs the UI, Dust 514, and overlay
assets bundled into Shrike itself. TurtleTools
and legacy portrait imports download their upstream archives themselves.

Character, corporation, and alliance originals are fetched lazily from CCP and
refreshed by the image queue. A freshly provisioned volume can therefore serve
some missing images while the corpus warms.

## Caching

Static and content-addressed objects are immutable. Entity responses use the
long shared-cache policy defined by the Go image service. Cloudflare should
respect origin cache headers and must not cache `404`, authorization failures,
or `5xx` responses.

## Rollback

Unset `IMAGE_STORAGE_PATH` to return to the configured S3-compatible image
bucket. This fallback is retained temporarily for deployment rollback and can
be removed after the local origin has been verified in production.
