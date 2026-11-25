# Kill category pages

The Go frontend serves every type in `internal/killtype.Types` at `/kills/{type}`. Unknown types return 404. One page template renders the title, eight most valuable kills from the last seven days, three category top lists, and the first 50 killmail rows. It reuses the homepage's valuable cards, top boxes, kill rows, pagination, and WebSocket controller.

The page loader calls the same SQL helpers as `/api/killlist`, `/api/kills/most-valuable`, and `/api/kills/top`. It does not make loopback HTTP requests. The killlist uses the existing daily-count rollup for numbered pages. A one-minute cached list and five-minute cached statistics retain the API's freshness windows.

The `kill_category_top` cron queues missing snapshots and snapshots that are at
least an hour old. Each River job carries its category, so a retry refreshes
the same boards. The worker publishes the three seven-day lists together in
Postgres and keeps the last successful snapshot available after a failure.
Valkey serves the same snapshot from cache, with a primary-key read to refill
it after a miss. The initial page, fragments, and standard seven-day top-10
API reads use the saved result. Cron logs and metrics report missing snapshots
and the oldest snapshot age.

If a prepared board is unavailable, the page can still use fresh fragment results. Missing lists render same-size placeholders and the browser requests their Go-rendered fragments. On mobile, those requests wait until the Top Lists tab is selected. An error leaves a retry control in the affected box.

Mobile Kills, Top Lists, and Valuable tabs use a URL fragment and `pushState`. A back navigation restores the selected tab. Typed killlists subscribe to their own WebSocket topic and keep the homepage's page-one insertion and later-page badge behavior. Pagination requests preserve the selected type.

Category pages are indexable on the production apex and listed in its sitemap. Until tenant pages are ported, custom production hosts receive the existing placeholder rather than global category data. Top-list fragments and caches use category-specific keys, so results from different types cannot mix.

On 2026-09-24, a cold `/kills/big` request that waited for every top query took about 11.7 seconds over the configured Tailscale database; a warm request took about 5 milliseconds. With cold top lists deferred, `/kills/lowsec` delivered its main HTML in about 1.65 seconds. These are local observations, not production latency targets. Slow top-list SQL still needs a separate optimization if cold fragment completion matters.
