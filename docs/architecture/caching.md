# Cache architecture

Shrike uses one shared Valkey and process-local caches. River stores all queue
state in Postgres.

The process-local cache is level 1 (L1). It uses a least recently used (LRU)
eviction policy. The shared Valkey cache is level 2 (L2).

## Contract

Application programming interface (API) reads use this order:

1. Read the process-local LRU.
2. Read the shared Valkey cache.
3. Read the source.
4. Write a source response to both cache tiers.

An L2 hit enters L1 for the remaining Valkey time to live. An L1 entry cannot
outlive its matching L2 entry.

Explicit invalidation removes matching entries from both tiers.

EVE Swagger Interface (ESI) reads use the same L1 and L2 order. Expired ESI
entries keep their entity tag for conditional requests.

Image responses use a separate process-local LRU. Backblaze B2 is the shared
image store.

## Response directives

Cloudflare is the tier in front of the origin. It reads the `Cache-Control`
header on each response.

A cached route derives its directives from the time to live that stores the
entry:

- `s-maxage` equals the time to live.
- `stale-while-revalidate` equals the time to live.
- `max-age` is one quarter of the time to live, and never below 30 seconds.

The origin and Cloudflare then expire together. A shared cache that outlives
the stored entry serves a body the origin has already replaced.

Three route groups set their own directives instead. Killmail details, battle
reports, and the static data export (SDE) do not change after the origin writes
them, so they use a longer shared lifetime than their storage lifetime.

`sharedCacheControl` in `internal/api/cache.go` builds the derived value.
`routeJSONCache` in `internal/api/route_cache.go` takes the value per route.

## Defaults

`API_CACHE_BYTES` sets the L1 API response limit. The default is 256 MiB.

`IMAGE_CACHE_BYTES` sets the image response limit. The default is 1 GiB.

Each limit applies to one `serve` process. The process does not reserve this
memory at startup.

Set either value to `0` to disable that local cache.

The shared Valkey uses the `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, and
`REDIS_DB` settings.

Use an `allkeys-lru` eviction policy when Valkey has a memory limit. Recently
used coordination keys then compete with response entries by recency.

## Key roles

Valkey stores these types of data:

- API and ESI response cache entries.
- ESI rate-limit and pause state.
- Short-lived locks and request claims.
- EVE single sign-on flow state.
- WebSocket and ticker publish/subscribe messages.
- Current status and ephemeral announcements.

Valkey does not store River jobs, retries, schedules, or queue history.

## Failure modes

An L1 miss reads Valkey.

A Valkey cache miss reads the source.

A Valkey outage does not make a cached API response invalid. The current
process can still serve an unexpired L1 entry.

A Valkey write failure does not fail a completed request. Another instance can
read the source again.

An application restart clears L1. Valkey can repopulate the new process.

A Valkey restart clears shared cache and coordination state. The application
rebuilds this data from the source or its next scheduled check.
