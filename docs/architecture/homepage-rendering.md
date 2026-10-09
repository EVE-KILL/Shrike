# Homepage rendering

This decision describes how the Go frontend loads and renders the apex homepage.

Status: Accepted

## Context

The apex homepage shows eight recent valuable kills, 50 latest kills, and seven leaderboards for the last seven days.

| Section | Initial read | Rows |
| --- | --- | ---: |
| Most Valuable | `/api/stats?dataType=most_valuable_kills&limit=8&days=7` | 8 |
| Latest Kills | `/api/killlist?type=latest&limit=50` | 50 |
| Top Characters, Corporations, Alliances, Ships, Systems, Regions | `/api/stats?dataType={type}&limit=10&days=7` | 10 each |
| Top Factions | `/api/stats?dataType=factions&limit=6&days=7` | 6 |

The original Nuxt implementation fetched these nine data sets during server rendering. `DeferredTopBox` delays browser hydration, but its data request still runs before the HTML response. The valuable-kill tabs for ships and structures load when selected. The kill list adds live WebSocket updates after the initial render.

The API caches `/api/stats` for five minutes and `/api/killlist` for one minute. The Go homepage uses the same data freshness windows and a 30-second shared HTML lifetime in production.

The Go page loader separates data access from templ components. Its renderer buffers the complete document before sending headers.

## Decision

Render the first page of all three sections into one HTML response. This gives visitors the visible content and links without another browser request. Keep images lazy below the first viewport. Load the other valuable-kill tabs only when selected.

Use direct EVE image URLs for character portraits and corporation and alliance logos. Keep local image routes for inventory types and map images.

Leaderboard accent strips use the same OKLCH palette selection and lightness clamp as the Nuxt component, so dark in-game logo colors remain visible on the site's background.

Load independent data sets concurrently with a bound below the database pool limit. Pass the request context to every read. Measure pool waits and total load time before changing the concurrency limit.

`HomepageReader` calls the existing API SQL and enrichment functions. It gives templ components typed data. It does not call the site's own HTTP API. Public API routes remain available.

The reader uses the API service's L1 and Valkey L2 cache with separate keys for each query. Its statistics and latest-kill windows match the API. A stale value can be served while one refresh runs. Tenant data must use a separate cache scope when the custom-domain page is ported.

The latest-kill list renders as HTML first. A browser controller handles pagination, quick filters, tab changes, and WebSocket reconnection. On page one, newer kill events prepend a row, respect active quick filters, and trim the list to the selected limit. Later pages pause inserts and show a new-kill badge; selecting it loads a fresh first page rather than a cached one-minute snapshot. Explicit refreshes share a single load per key and reuse a result fetched within the last ten seconds.

Use the same templ component for full-page content and any later HTML fragment. Do not stream the page during the first implementation. Buffering keeps page errors and response status coherent, and full HTML avoids fragment requests on the normal path.

## Consequences

A cold request still waits for the slowest required data set. If a required read fails without cached data, the page returns an error. It does not claim that an unavailable list is empty.

If measurements show that cold leaderboard reads delay the first byte, defer only leaderboards below the first viewport. Serve their HTML from the same templ component, reserve their layout space, and load them near the viewport. Keep valuable kills, the latest-kill list, and the first visible leaderboard in the initial response.

Compare warm and cold requests with the development panel against the intended database path. Check first-byte time, data-load time, database query count, pool wait, HTML size, and image loading. Test mobile and desktop layouts. Do not infer production latency from the local database.

The custom-domain homepage uses a configurable widget order and scoped data. It is not ported yet. Production custom hosts still receive the migration placeholder. Do not reuse apex cache entries for a custom domain.

On 2026-09-24, a cold local request over Tailscale took about 0.35–0.55 seconds for 14 database calls. A warm cached request took about 1–3 milliseconds with zero database calls. These measurements do not include a production edge cache or representative user network.

## Source

- `internal/frontend/home_page.templ`: apex layout and custom-domain widget order.
- `internal/frontend/home_components.templ`: server data fetch and visible hydration.
- `internal/frontend/home_components.templ`: initial tab and later tab loads.
- `internal/frontend/killlist.templ`: first page, filters, pagination, and live updates.
- `internal/api/global_stats.go`: global leaderboards and valuable kills.
- `internal/api/killboard.go`: latest-kill list and its one-minute cache.
- `internal/api/cache.go`: five-minute statistics response cache.
- `internal/api/domain_killboard.go`: domain-scoped cache and query paths.
- `internal/frontend/render.go`: Go page loader and buffered rendering.
