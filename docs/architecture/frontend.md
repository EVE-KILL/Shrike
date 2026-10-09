# Go frontend

The frontend renders HTML inside Shrike. Shared rendering code handles metadata, response policy, and assets so each page can focus on its content.

The Nuxt application has been retired. Browser source and build tools live in `frontend/`; native server fitting calculations live in `engine/dogma/`. Go/templ is the only page renderer. References below to the original Vue implementation describe the migration source retained in Git history, not runtime dependencies.

## Files

| File | Responsibility |
| --- | --- |
| `internal/frontend/frontend.go` | Routes, static page registration, deployment options |
| `render.go` | Page loaders, buffered rendering, status codes, cache policy |
| `metadata.go`, `metadata.templ` | Titles, descriptions, canonical URLs, social cards, structured data |
| `layout.templ` | Shared document and page layout |
| `navigation.templ`, `footer.templ` | Site navigation, dialogs, footer |
| `pages.templ` | Current page bodies and common page header |
| `home_components.templ`, `kills_page.templ` | Reusable killboard widgets and category page layout |
| `home_data.go`, `kills_page.go` | Typed page loading and category titles |
| `advanced_search.go`, `advanced_search.templ`, `static/advanced-search*` | URL-backed search controls and result fragments |
| `fit_preview.go`, `fit_preview.templ` | Compact observed-fit cards and interoperable editor links |
| `saved_searches.templ`, `static/saved-searches.js` | Browser/account saved-search controls |
| `utility_page.templ`, `search_picker.templ`, `static/search-picker.js` | Source page headers, login gates, and reusable entity/location pickers |
| `campaign_creator.go`, `campaign_creator.templ`, `static/campaign-creator*` | Campaign draft, preview, validation, and creation |
| `campaign_editor.go`, `campaign_editor.templ`, `static/campaign-owner.js` | Authenticated owner editor fragment and campaign actions |
| `session.go`, `internal/api/session_page_read.go` | Read-only authoritative browser session lookup for SSR |
| `faction_war.go`, `faction_war*.templ`, `static/faction-war*` | Faction matchups, retained fragment tabs, cursor killfeeds, shared members/intel, and the original warzone SVG |
| `assets.go` | Embedded assets, revision URLs, conditional responses |
| `crawlers.go` | Robots rules and static sitemaps |
| `diagnostics.templ`, `static/diagnostics.*` | Development performance panel |
| `static/navigation-progress.js`, `static/loading-state.js` | Shared navigation and fragment loading feedback |
| `static/fragment.js` | Focus-preserving HTML fragment replacement |
| `internal/requeststats`, `internal/db/tracer.go` | Request counters and pgx instrumentation |

## Add a page

1. Create a templ component for the page body. Do not include another document or call `layout`.
2. For a static page, register its path, metadata, and component in `site.pages`.
3. For a dynamic page, register a Go route through `s.pageHandler`.
4. Load data before rendering. Pass finished view models to the component.
5. Return `ErrNotFound` when the requested record does not exist.
6. Run `make fmt-frontend` and `make build-frontend`.
7. Check the page in the browser and run the relevant tests.

Example static registration:

```go
{
    Path: "/about",
    Metadata: Metadata{
        Title:         "About",
        Description:   "About the EVE-KILL community killboard.",
        CanonicalPath: "/about",
        Indexable:     true,
    },
    Content: about(),
}
```

A dynamic loader receives `*http.Request` and returns `(Page, error)`. Use `r.PathValue` for route parameters and `r.Context()` for data access.

Call shared Go services directly. Do not make loopback HTTP requests or invoke an API handler to obtain page data.

Some migrated pages still dispatch an API handler in-process and decode its
JSON response. There are eleven such dispatch sites across ten frontend files,
covering fitting catalogues, maps, battles, campaigns, accounts, comment feeds,
blog previews and sitemaps. These preserve existing validation, authorization,
domain scoping and cache behavior, but remain follow-up cleanup work. Move each
page family onto a typed service shared with its public API; verify those policies
before removing the adapter. API endpoints and compatibility aliases require a
separate consumer audit before retirement.

When replacing a fragment that contains controls, use `replaceFragment` from
`static/fragment.js`. Add a stable `data-focus-key` to controls whose label or
value changes; IDs, accessible labels and form names are also supported. The
helper restores only the currently focused control when its replacement is
visible and enabled, preserves the matching instance of repeated controls,
and leaves the page's scroll position alone. Keep request-generation and
connected-node guards around asynchronous imports and replacements. Entity
members, corporation sorting, conflict sorting/paging, top periods and killlist
controls use this convention.
Extract shared business logic from API handlers when a page needs it. Keep API endpoints until their remaining consumers have been checked.

`/wars` sends the directory and a fixed-size faction-war placeholder in its document. Its independent `/_fragments/wars/faction` request uses the existing cached aggregate reader; failures offer Retry without replacing the directory. Custom boards omit global faction data. Keep unrelated slow widgets outside the document loader when their content can arrive later.

The faction-war detail page renders its combat summary, leaderboards and first killfeed on the server. Members, intel and systems are loaded as HTML fragments on demand and retained when switching tabs. Faction filters use killmail cursors, not a lifetime count. `frontend/src/utils/fwWarzone.ts` contains the exact layout nudges, Delaunay cells, mini systems and colors shared by the Vue map and Go's SVG renderer; rebuild its checked-in browser module with `make build-faction-war-frontend`.

Faction members and intel use per-killmail attacker lookups. The small `LATERAL … OFFSET 0` boundary prevents PostgreSQL from scanning the whole attacker table for a recent faction window; live read-only comparisons confirmed the same members and attacker-ship totals.

## Metadata and indexing

Every page gets title, description, Open Graph, and Twitter card defaults. Override these values in `Metadata` when the page has better information.

Set `CanonicalPath` explicitly. It must be a root-relative URL and always resolves against `https://eve-kill.com`.
Include query parameters only when they identify meaningful content, such as a distinct pagination page.
Choose entity-tab canonicals in the route that understands those tabs. The renderer does not guess or copy request tracking parameters.

Indexing requires all of these conditions:

- `Metadata.Indexable` is true.
- The application runs in production.
- The actual request host is the canonical apex.
- The page returns HTTP 200 and has a canonical URL.

Development, tenant hosts, error pages, and unported pages receive `noindex, nofollow` in both HTML and HTTP headers.
Forwarded host headers cannot change this decision. Host values never become canonical URLs.

Supply `StructuredData` as Go data, such as a struct or map. The renderer JSON-encodes it with HTML escaping.
The content security policy allows only the exact resulting JSON-LD script. Do not use raw HTML or handwritten JSON script tags.
Structured data must describe the visible page content. The apex homepage now declares WebSite and CollectionPage data.

`robots.txt` lets crawlers read page indexing directives. It excludes the API and authentication namespaces.
It advertises a sitemap only when an indexable static page exists on the production apex.
`/sitemap.xml` contains opted-in static pages and the validated `/kills/{type}` routes; `/sitemap_index.xml` lists that sitemap.
Both endpoints return 404 outside the production apex. Dynamic sitemap sources will be added with their corresponding data pages.

## Rendering and response policy

The renderer buffers the complete document before sending headers. A loader or template error cannot leave a partially rendered HTTP 200 response.
Unexpected errors get a generic HTTP 500 page; internal error details stay in logs.
Missing records get the shared HTTP 404 page. Neither error response is cached or indexed.
HEAD requests return the same headers as GET, without a body.

HTML defaults to `private, no-store`. Set `Page.SharedCacheTTL` only for pages whose output contains no personal data.
Shared caching is limited to successful production apex responses. Cookies, Authorization, or Set-Cookie headers disable it.
Eligible responses also vary on Cookie and Authorization so shared caches cannot reuse the anonymous variant for personalized requests. Existing Vary headers are preserved.
Indexability and cacheability are separate choices.

## Loading feedback

The document includes one themed, fixed progress bar and a polite screen-reader
status. Native Navigation API events observe full page changes, including
JavaScript `location.assign`/`href` and row handlers. They do not intercept
navigation or change browser history/scroll behavior. Older browsers have
link/form and the existing admin-row fallbacks. New programmatic navigation
in a browser without the Navigation API needs a link or an explicit fallback.

Fragment controllers set `aria-busy="true"` on their content before starting
work and remove it in their guarded `finally` block. The shared observer tracks
those transitions and releases detached or hidden content. Static lazy
placeholders are not treated as running requests. Overlapping requests keep
one indicator; cancelling an older request cannot complete its replacement.
Use `data-loading-silent` on periodic background updates that should not show
page loading feedback, as the hourly rankings refresh does.

Full page navigation starts feedback synchronously before the browser leaves
the document. Fragment work waits 150ms to avoid flashing on fast responses.
The clicked link, button or killmail row receives a subtle outline pulse.
Reduced-motion settings use a static bar and outline. Downloads, hash-only
links, modified/new-tab clicks and cached fragment tabs retain their usual
behavior. Page restoration clears the departing navigation state.

Do not put authentication state or request-specific data in package globals or reusable components.
Keep authorization checks in the service layer when adding authenticated pages and mutation routes.

## Assets

Put public frontend files under `internal/frontend/static`. Use `assetURL("file.css")` in templates.
Source CSS and JavaScript can use `/assets/...` references; the asset builder versions them with the same asset tree.

Asset URLs include a content-derived revision. These responses use immutable caching for one year.
Changing an asset changes the revision, including references inside CSS and JavaScript.
Unversioned URLs remain available but must revalidate. ETags support HTTP 304 responses.
Directory listings and unknown revisions are not served. Files are embedded in the binary; production needs no frontend directory.

To preserve Nuxt markup during a port, wrap the page in `.go-utility-page` and load `utilities.css`. `make build-css-frontend` compiles `frontend/styles/theme.css`, custom utilities, typography, and utility classes used by Go templates and widget scripts. Native CSS `@scope` contains Preflight and utility styles within that page, keeping the shared shell independent. Put utilities on descendants of the scope wrapper: root-level positioning/spacing classes do not apply to the wrapper itself. Global overlays such as announcements and toasts use an outer scope wrapper around their actual positioned nodes. Keep global resets at zero specificity so scoped spacing utilities can override them. Add new widget scripts to the scanner when they generate utility classes. The generated stylesheet is checked in; Air rebuilds it when the optional npm tooling is installed.

`make build-icons-frontend` reads the installed Iconify/Lucide collection in `frontend/node_modules` and preserves aliases/transforms using Iconify's own helpers. The asset manifest creates independently cached SVG symbols for all names while the small common sprite remains unchanged. `iconURL` in templates and `iconHref` in configurable browser controls select the right asset. The source catalog participates in the revision hash but is not exposed as a public asset. Ordinary Go builds require no icon runtime or Bun process.

The `frontend` directory has a small Go module for dogma asset embedding. The root module uses a local replacement, and the Docker build copies its module manifest before downloading dependencies. This boundary keeps Bun's installed package copies out of Go's recursive package discovery without duplicating the engine data.

## Browser code

The current scripts own shell interactions. Page rendering remains on the server.

Shared announcement hosts add no feed or identity read to public SSR. Their browser controller reuses the canonical public feed, shared session request, account dismissal endpoint and relay core. Nuxt and Go share the pure presentation/lifecycle helpers; both merge live changes received during an HTTP refresh and catch up after reconnecting. The ticker runs as a CSS animation with width-change resynchronization. Its measured height sets `--announcement-footer-height` so floating controls remain reachable. `-admin-fixtures` routes the feed to disposable PostgreSQL and exposes a local announcement relay for lifecycle/reconnect checks; normal production transport is unchanged.

The admin Domains directory shares Nuxt's pure counts/filter helpers and filters the existing single bounded response locally. Domain detail uses the original approvals/file markup and authenticated review/toggle APIs. A private HTML fragment refreshes metadata while keeping the selected tab and loaded image nodes; file previews load only on first selection. Failed writes retain drafts, and failed reads after a successful write preserve the last view with Retry. Both renderers contain mobile table scrolling and stack review controls below 640px. `-admin-fixtures` seeds deterministic domain metadata and actual PNG bytes into the disposable database and memory storage; no test image reaches production storage. Rebuild shared helpers with `make build-admin-frontend`.

Admin Campaigns retains the source card/status/runtime markup and the existing bounded primary-reader API. Filters and paging replace only their private results fragment; stale requests are canceled, failed reads retain the last view, and processing actions keep filters while disabling only their affected rows. The shared API orders equal timestamps by campaign ID to stabilize pagination. Pure state/formatting helpers are bundled with `make build-admin-frontend`; `-admin-fixtures` seeds fifty additional campaigns across all statuses and processing metadata, with equal timestamps to exercise both 40-row pages. All comparison mutations and delete tests use disposable local databases.

Admin Moderation and Comments share `admin-review.js` for bounded private queue fragments, canceled stale reads, row actions, failure/retry and five-second deletion confirmation. Refreshes preserve matching image/video/frame nodes and the last successful result on failure. The source and Go reuse canonical comment targets; missing legacy blog slugs are resolved in a batch. Queue APIs sanitize stored HTML with the existing comment policy and `SafeAccountBioHTML`, preserving allowlisted EVE color/font declarations. Preview styles are extracted from the source CSS by `make build-css-frontend`; pure labels and routing helpers are bundled by `make build-admin-frontend`. The `-admin-fixtures` seed covers all target kinds, review statuses, legacy contexts, rich media and more than 100 active comments. Review, visibility and soft-delete tests use disposable PostgreSQL only.

Admin River Workers reuses the existing operations APIs and `queue.Depth`. Its initial queue/job reads run concurrently; private `/_fragments/admin/river/{queues,jobs}` responses update each part independently. Filters cancel superseded reads, retain successful results after failures and keep their state through admin navigation. The browser inspector uses the source markup and shared date/JSON/state helpers inside a native dialog for focus, inertness and Escape. The API cursor is the last returned job ID, preserving the extra row for the following exclusive-cursor page. `-admin-fixtures` migrates River in the disposable database and seeds 137 jobs without starting workers. All control tests remain local; finalized-history deletion and running-job guards are verified in isolated API tests. Rebuild shared helpers with `make build-admin-frontend`.

Admin Wallet reuses the public wallet's journal contract and server money formatter. `admin-wallet.js` owns the original sync, authorization URL and manual prize-record APIs; private fragments replace its results while keeping messages, filters, focus and the last successful view after a failure. Source formatting, missing divisions, settlement grouping and callback messages are bundled from `adminWallet.ts`. Amounts remain decimal strings outside presentation. The fixture seeds fake ESI tokens and accounting records without starting workers. Optional read-response rules (`response` in `/__fixture/faults`) exercise external navigation and empty contracts entirely on loopback; they reject write mocks and are absent without `-faults`. Actual OAuth scopes/state and private wallet controls are verified separately through real isolated API tests. The Nuxt retained admin shell follows `router.currentRoute`, avoiding deferred-route stalls after callback query changes. Rebuild helpers with `make build-admin-frontend`.

Admin Blog reuses the public post DTO and private CRUD/preview APIs. `adminBlog.ts` is extracted from the original Vue component and supplies both editors with the same slug, tag, draft and payload rules, including UTC publication dates. The Go editor uses the reusable native modal frame at 768px, the existing date picker and text-only tag previews. Private fragments retain the previous list on errors, cancel stale filters and preserve cached section state; failed or expired-session saves leave drafts intact. Permanent-delete confirmation is tested in the browser only with a blocking local fault; actual deletion is proved through isolated API tests. The list's SQL has a creation-time/ID order, and long title/summary/tag stress cases wrap within 393px layouts in both renderers. Rebuild helpers with `make build-admin-frontend` and utilities with `make build-css-frontend`.


Local Scan and D-Scan use the original Nuxt page/result markup and scoped utilities. Public analysis still calls the existing APIs, then renders the result through `/_fragments/scans/{kind}`; saved pages use `ScanReader.SavedScan` directly. Native disclosures preserve the source's group defaults, and stable sorting follows JavaScript's numeric entity-key order and incoming pilot order for ties. Browser requests cancel superseded analysis, retain results when a save session expires, and refresh authentication on focus. Save/share tests use the disposable PostgreSQL fixture, never production writes.

The fitting catalogue at `/fits` delegates six parallel reads to the existing cached fitting APIs in-process, preserving their SQL, validation and cache invalidation. Its source cards, shared rating display, background hero and statistics row are reusable; `utilityHeader` adds the source background/metadata slots while `utilityPageHeader` preserves existing callers. Flavor/doctrine controls update only their HTML section through `/_fragments/fits/catalog/{part}`, cancel superseded requests, and restore selection through history. The landing page loads no dogma/WASM. Browser validation covers its linked search and detail routes, editor payloads, and isolated feed failure/retry.

Fitting search ports `web/app/pages/fits/search.vue`. Its initial HTML contains the source hull picker, popular hulls, filter controls and URL state using the two existing cached metadata APIs. Heavy results and observed-fit histograms load independently through `/_fragments/fits/search/{results,distributions}`; availability still uses the original `/api/fits/search/availability` contract. The Go templates reuse `fitFamilyName`, `fitContextParts` and `observedFitEditorURL`, so result names, badges and v3 editor payloads remain shared with the item-fitting and catalogue pages. The browser controller keeps module counts, operators, statistics, NPC profiles, sorting and capacitor stability in a replace-state URL; aborts stale requests; and appends one 24-fit page at a time. Native popovers handle menus on desktop and mobile. The page loads no dogma/WASM. Local public-GET fault rules also reach the in-process fitting API reader, allowing each section and retry to be exercised without affecting production data.
Both `/fits` and `/fits/search` enter the production sitemap only when the fitting API is registered. `/fits/create` is deliberately absent because the draft editor is not indexable.

Saved fits at `/fit/{fit_id}` call the existing session-aware fitting API in-process and reuse `fitEditorContent`, the original wheel geometry, and the shared Rust dogma engine. The HTML contains authorized fit rows in inert JSON; `fit-document.js` ports Nuxt's slot ordinals, module states, charges, grouped drones and cargo conversion. Its save payload is tested against the source `fit/toServer.ts`. Owners update the existing row; other viewers fork the current preview into a fresh `/fits/create` draft. Update preloads the saved visibility instead of defaulting a private fit to public. New retains the source discard confirmation, then opens the blank creation route. Share links open the current preview as a creation draft so they include unsaved edits.

`fit-capacity.js` ports `useHullCapacity` and `useFitValidity`, including strategic cruisers' subsystem-granted slot/hardpoint fallback. The editor, compatible-hardware browser, rack filling, suggestions and validity badge share those counts. Validity also retains the source resource, missing subsystem, drone bay/bandwidth, hardpoint and calibration checks. Capacity reads the complete dogma hull attributes directly rather than relying on the subset requested for the statistics display.

Saved-fit HTML is always private/no-store, and restricted fits are noindex. Source 403/404/error cards render before any fitting data or WASM is included. Ratings keep the original PUT/DELETE contract and display failed writes without changing the aggregate. The comment thread is imported and mounted only when expanded, using target type 8, ID 0 and the fit slug; collapse releases the shared thread's requests and WebSocket subscription. Desktop comparison retains the 320px sidebar, 600px wheel and 300px statistics panel. Smaller screens scale the wheel and stack the existing browser, resource/bay and statistics components without horizontal overflow. All write validation uses disposable local accounts/databases.

Scan hashes identify input text, so saving the same input can refresh its analysis at the same URL. Saved HTML and both API aliases use `private, no-store` and a single indexed database read to avoid serving an older result from a browser, proxy, or replica cache. The PostgreSQL integration test covers this read-after-write behavior, API/reader equality, anonymous/cross-origin rejection, and session expiry.

`static/preferences.js` reads the existing `ek_theme` cookie before first paint and applies validated color values as CSS custom properties. New components should use the tokens in `static/site.css` for brand, semantic, and content colors. Use `color-mix()` with those tokens for translucent hover and selected states instead of fixed blue hex values. The killlist uses the brand token for entity-link hover, the ISK token for values, and security tokens for system status.

The fitting editor at `/fits/create` uses the existing Rust dogma WASM and SDE. `frontend/dogma/assets.go` embeds the binary data once; `static/dogma.js` is a checked-in browser bundle generated by `make build-dogma-frontend` whenever the shared dogma code changes. This build step needs Bun, but the running Go frontend and ordinary Go builds do not. Its WASM permission is scoped to the editor page's CSP. The editor's local state owns its DOM and keeps its interoperable `v3:` fit in the URL, so page reloads preserve unsaved changes.

The shared universe map renderer lives in `frontend/src/utils/map/goFrontendCanvas.ts`. It uses the same Delaunay projection as Nuxt and imports its color, activity, and battle aggregation helpers. The dedicated region renderer in `frontend/src/utils/map/goRegionPage.ts` uses Nuxt's 2D region layout. `make build-map-frontend` bundles both into checked-in `internal/frontend/static/map-canvas.js` and `region-map.js`, plus the shared AIID audio core from `frontend/src/utils/map/aiidAudioCore.ts`. Go serves the bundles directly; Bun is a build-time tool only. Pages provide map data and handle navigation, while the renderers own camera state, drawing, and hover. Watch-area changes update the live adapter without reopening its socket or losing streamed rows/freshness history. Old/newly covered history does not replay alarms. AIID retains its background relay; drawing pauses in hidden documents.
The wheel is a direct port of `web/app/components/fit/ring/`: `static/fit-wheel.css` keeps the source CSS, and `static/fit-wheel.js` renders the same SVG geometry and slot rotations. The small fitting UI textures live under `static/fit-ui/`. The expanded editor loads its observed-fit cards from `/_fragments/fits/variants/{id}`, using the same Go reader and fit classifier as the item fittings tab.

In development, each page carries a process identity. A same-origin SharedWorker multiplexes `/__dev/reload` across open documents, avoiding HTTP/1 connection exhaustion when several tabs are open. A single bounded `/__dev/version` read bootstraps each document; browsers without working shared workers poll that endpoint every 1.5 seconds instead. Pagehide releases its port, and BFCache restoration reconnects and rechecks the current version. The worker does not replay a stale process ID into a newly loaded document. Air watches code/assets and restarts Go; the shared EventSource reconnects and notifies every document to reload when its identity changes. The stream sends a heartbeat every 20 seconds and has no database access. Reload routes and scripts are absent from production HTML. Browser regression created nine Go documents without starving a fresh dogma calculation; seven admin documents changed identities after a restart while the fitting URL and calculated stats survived.
Keep immediate interactions local, and give each complex widget ownership of its own DOM subtree.
The fitting table retains its row nodes during selection and tracks table selection independently of catalog/ring selection. Desktop selection has no floating panel that could intercept lower rows. On small screens, a sticky selected-row strip exposes the source Remove/Unload/Fill highs actions below fixed navigation; named columns have a minimum width and scroll inside the table. Column defaults, validation and readable minimum widths live in `frontend/src/utils/fitTable.ts`; rebuild the checked-in browser helper with `make build-fit-table-frontend`. Both phone layouts wrap the toolbar and resource cards, stack the sidebar, and retain usable fullscreen table height. Four-subsystem Tengu builds, dynamic rows and remove/replacement interactions were compared against Nuxt; 393px checks cover contained horizontal keyboard scrolling, sticky actions, Fill highs/Remove/Undo and fullscreen.

Do not let a future HTML update layer replace a fitting editor or map while it owns local state.

Roam Report ports the original `web/app/components/roam/` dashboards and transformations. Its shared API/page reader reads saved reports from primary and enriches legacy observations in one place. The kills tab passes that loaded report into the existing conflict killlist query, avoiding another report lookup. Reads are never cached because anyone with the share link can rebuild a report. Creation and updates retain the existing API endpoints.

Report tab links work as ordinary server-rendered routes and progressively load HTML fragments after initialization. Only tab content changes, so the hero images retain their dimensions and the browser history remains usable. Rebuilding refreshes the full report at the same URL. Composition filters and sorting use a normalized row payload and clone the same row template used by the initial server render; they do not rebuild the dashboard from a duplicate full report payload. Combat maps use the shared renderer with optional marker labels and overview controls.

`dateTimePicker` and `static/date-time-picker.js` port the existing Vue picker. A hidden form value always stores EVE/UTC time; the shared `ek_tz` preference controls display only. Initialization supports fragment replacement, and closing/cancelling restores the original form values. Use this component for later time-window tools instead of introducing another picker.

Advanced search uses the same Go query parser and loaders as `/api/killlist/advanced`, through `HomepageReader.AdvancedSearch`. Initial results render with the document; later filter, page, grouping, and drilldown changes fetch `/_fragments/advancedsearch`. Only the result subtree is replaced, preserving focus, filters, and open controls. The cursor stack remains in the browser; filter changes reset the page while retaining its selected size. Search links retain display names, while the query serializer removes display-only fields and converts ISK shorthand. Query documents and fragments remain private/no-store and noindex. The shared query selects a bounded set of matching killmail IDs before hydrating display joins, so sparse attacker searches do not first hydrate the whole time window.

`fitPreviewCard` ports the original `FitPreview.vue`. It reserves image dimensions, groups modules and drones, links to the killmail, and encodes an ESF v3 fitting once per card. Its popularity badge supplies the family/exact drilldown hash. Keep this component and the shared slot/drone contracts for later fitting pages.

`savedSearches`, `savedSearchPanel`, and `initializeSavedSearches` own saved-search UI and persistence. The panel uses a native manual popover for the top layer while remaining inside the page's CSS scope. Browser storage keeps the existing `evekill-saved-searches` format; account operations use the existing authenticated, revision-checked `/api/me/saved-searches` endpoints. The shared Nuxt document conversion is bundled unchanged into `saved-search-data.js` with `make build-search-frontend`. Rebuild it when `frontend/src/utils/savedSearches.ts` changes. Programmatically restoring picker values updates their display without applying a new search, preserving Back history when loading saved custom dates.

Campaign creator ports `web/app/pages/campaigncreator.vue` and reuses the existing campaign card and UTC/local picker. Its browser module owns the draft, live preview, and prize split controls; creation still uses `/api/campaign/create`. `make build-campaign-frontend` bundles the original campaign policy, side palette, and number/date formatters into `campaign-data.js`. Rebuild it when those shared Nuxt utilities change. API wallet amounts stay exact decimal strings; convert them to numbers only when passing them into the source display formatter. The private historical window is 365 days in both browser implementations, matching the authoritative Go API policy.

`searchPicker` and `initializeSearchPickers` port the source entity/location autocomplete. Parents own selection caps and duplicate rules; the picker debounces searches, cancels stale requests, and emits a typed `search-pick` event. Use `updateSearchPicker` to change selection or disabled state and `closeSearchPickers` before removing its subtree. Suggestions may extend beyond participant cards, so avoid clipping them with `overflow: hidden`.

Authenticated SSR resolves the `evelogin` cookie through `SessionPageReader.BrowserSession`. It does not create or touch a session and never treats the display-only `ek_auth` cookie as authorization. Campaign creation HTML and `/_fragments/campaign/{id}/edit` remain private/no-store. The editor fragment checks the real session and campaign ownership before rendering. Owner changes retain the existing API validation, wallet transactions, funded-rule locks, and idempotency tokens. Background recomputation refreshes defer while an editor or prize dialog is open to preserve its draft, including when it opens during an in-flight poll. Campaign owner/prize widgets share one `/auth/me` request through `readBrowserSession`.

Campaign killfeed fragments keep victim character/corporation/alliance IDs on the shared row markup. `killlist-sides.js` ports Nuxt's precedence and palette/fallback tint rules; the campaign supplies the small membership payload once. New cursor or side-filter rows receive their colors locally, without reloading campaign data from the database.

`go run ./scripts/frontend-parity` creates a disposable database on the explicitly supplied loopback `TEST_DATABASE_URL`, applies migrations, and serves real local account/campaign/wallet/comment and battle-generator APIs on 4450, without starting a separate renderer. Use `frontend-parity.localhost` to isolate cookies from normal development. Public upstreams accept only loopback origins, proxy only GET/HEAD, and discard authentication headers. A second `go run ./scripts/frontend-parity -backend http://127.0.0.1:4450` serves current Go templates/assets on 4451 without recreating the fixture database. Stop the API fixture with SIGINT to remove its database. Its remote session adapter exists only for browser fixtures; production SSR uses the direct reader above.

For read-only comparisons against the Go preview, use `-backend http://127.0.0.1:4200`. Add `-faults` to enable local delay/failure tests. POST a JSON rule to `/__fixture/faults`, for example `{"path":"/api/conflicts/battles","query":{"map":"true"},"status":503}` or the same rule with `"delayMs":8000` and no status. POST `{}` to clear it. Rules affect only matching public GETs on the loopback comparison servers; they neither reach nor modify the upstream. Delays stop when the browser cancels its request, and injected failures cannot enter caches. The control is disabled by default.

The fixed-size `/__fixture/viewport?path=...&width=393&height=852` comparison frame allows mobile media-query checks when browser viewport control is unavailable. Add `audio=1` only for alarm validation: a fixture script passes the app's own Web Audio through an analyser to its real destination and displays context state, oscillator starts and peak waveform. It never records microphone/system audio and is injected only into an explicitly marked local frame. The render-only facade forwards `/api/map/` to the read-only public preview while keeping account writes and test WebSockets on the disposable API fixture.

The battle directory retains SSR cards, source filters and rich header, while subsequent filters replace only cards/chips. Its domain badge follows the cached site configuration, including the null fallback for an unconfigured host. Geography and battle markers load independently; either can report a failure with Retry, while the other remains usable. Scope changes and filters cancel superseded requests, and view changes reset the local map scope like Nuxt. Search clears stale suggestions immediately and names each candidate for accessibility. The shared Canvas2D renderer retains the original projection, Delaunay cells, palettes, marker totals and camera math; desktop and 393px checks cover all five scopes, matching battle IDs, autocomplete, keyboard tabs, Back, and pan/zoom/reset. Rebuild shared geometry with `make build-map-frontend` and header utilities with `make build-css-frontend`.

Battle Generator ports the three source wizard stages and responsive entity lanes. `frontend/src/utils/battleGenerator.ts` supplies shared helpers for grouping participants, moving entities, preserving assignments after a window reload, and seeding an existing report. Rebuild its checked-in browser module with `make build-battle-generator-frontend`. The source search picker, UTC/local picker, login gate, and reusable wizard steps handle the surrounding controls. Prefilled systems render in the initial HTML and their nodes remain in place when JavaScript initializes.

Entity lookup and save retain `/api/battle/generator/{entities,save}`. Preview requests use `/_fragments/battlegenerator/preview`, which invokes the API's authoritative calculation directly and renders the existing battle summary; its inert JSON payload retains the original save contract and timestamp serialization. Aborted or stale reads cannot replace a changed window or side assignment. The page and preview use private no-store responses; the API continues to enforce sessions, CSRF, conflict validation, and transactional creation/correction. Tests and browser comparisons use deterministic killmails in disposable PostgreSQL, including invalid edits, expired sessions, and creator retention when another user corrects a report.

Statistics ports the source period and population cards, section filters, jump links, and formatting. `StatisticsSite` uses the existing host-aware site configuration; custom boards expose only the sixteen period cards and preserve `/custom/stats` scope/period rules. The shared reader calls the original statistics and ranking queries and retains the API's timestamp serialization. Equal security scores and equal birth/founding timestamps have deterministic ID tie-breakers in both renderers.

The document checks all card caches through `StatisticsBoardsCached`; `ResponseCache.LoadMany` checks L1 before pipelining the remaining shared reads. It performs no SQL or background refresh. Fresh cards render immediately; missing cards use `/_fragments/stats/{card}`. The browser loads at most four cards concurrently and requests only visible sections, retains population cards when changing periods, guards stale responses, and offers per-card Retry. Filter/period URLs and Back/Forward reuse retained nodes when fresh. Rebuild utility classes with `make build-css-frontend`; the runtime remains Go-only. Cache tests cover immutable payload copies, freshness, promotion, and mutable-entry invalidation; disposable PostgreSQL tests verify global/scoped API parity and changed entity scopes.

Add HTMX fragments, lifecycle hooks, or shared formatting helpers when the first page needs them.
Do not add a second router, global client store, or generic data-fetching framework preemptively.

## Checks

```sh
make fmt-frontend
make build-frontend
go test ./internal/frontend ./internal/ingress ./internal/cli
```

CI checks template formatting and generated Go output. Generated files stay in version control.
Frontend tests cover metadata escaping, indexing policy, canonical URLs, partial-render failures, asset references, conditional requests, and sitemap filtering.

## Development diagnostics

The bottom-right DEV control, above the E badge, opens request diagnostics. It appears automatically outside production.

The document displays server, data-loading, template, database, and pool-acquisition times, plus SQL query counts.
It also breaks down navigation start to first byte into pre-request time and request-to-first-byte time. Body transfer and DOM readiness are separate browser measurements. These include work outside the Go renderer.

Recent API requests appear separately in the panel. Open search and enter a term to see a request with database activity.
The data-backed homepage performs database reads on a cold cache. A server-side cache hit can produce a real zero-query response.

All values come from the `Server-Timing` response header and the browser Performance APIs.
The header is also available in browser DevTools. No diagnostics endpoint, polling connection, or additional query is required.

Database instrumentation attaches to pgx in `db.New` and `db.NewRead` outside production:

- Query, QueryRow, and Exec calls increment the query counter.
- Batch queries increment it when pgx reports their results; batch duration is counted once.
- COPY operations have a separate count and contribute to database time.
- Pool acquisition is measured separately from database client time.
- SQL text, argument values, and returned data are never retained.

Pass the request context through every service call. Concurrent queries share a synchronized collector; separate requests have independent collectors.
Calls without a collector, including ordinary background jobs, do not allocate timing spans.

Database time includes client-side driver work and row consumption, not only PostgreSQL execution time.
Parallel query durations can exceed elapsed server time. Close query rows and batch results promptly so pgx can finish their traces.
The counter measures application SQL calls; it does not count protocol messages or automatic statement preparation.
For server execution plans, use PostgreSQL's own diagnostics separately.

Server timing ends before response transfer. API timing ends when response headers are committed.
Streaming feed and WebSocket requests are excluded. Development API responses use `private, no-store` so the browser does not reuse old timing headers.

Production does not attach the pgx tracer, emit timing headers, or render/load the diagnostics panel.


## Shared comment controls

The Go DOM composer uses `frontend/shared/utils/commentInput.ts`: provider validation, thumbnails, text insertion, and the existing GIF favorites schema/key. `make build-comment-frontend` checks in its browser bundle; ordinary Go startup requires no Bun process. `Metadata.CommentEditor` loads the scoped Nuxt utilities and composer CSS and emits inert control templates once per document. Root, reply, edit, media, GIF, and report controls clone those templates locally. Comment media keeps the existing server-validated HTTP(S) image/video support while script and connection policies remain unchanged.

`comment-thread.js` owns API reads/writes, the recursive hierarchy, moderation/report controls, and editor ownership. `comment-input.js` owns formatting, selection, length/loading/error state, keyboard shortcuts, and server preview through `/api/comments/preview`. `klipy-picker.js` retains Nuxt's search, paging, favorites storage, and direct MP4 insertion. Open editors survive live card updates; removing a tab fragment releases its editors, dialogs, and subscriptions. `comment-stream.js` shares one same-origin `/ws/comments` connection per document with source-compatible scoping, visibility pause, and reconnect backoff. Production relay frames omit raw Markdown; edit actions read `/api/comments/{id}` before opening to avoid stale or empty text.

For deterministic media/live UI tests, start the disposable API fixture with `-media-fixtures`. Only the external GIF provider and relay are replaced; authentication, Markdown, reports, and comment persistence stay real. Without that flag, an optional `KLIPY_API_KEY` environment variable supplies the actual provider. The render-only fixture now serves Go on 4451 and Nuxt on 4461 against the existing API, so both can restart without losing data. `/__fixture/viewport?path=/campaign/PrizeWait00001/comments&width=393&height=852` renders a fixed-size frame for actual mobile media queries when browser emulation is unavailable. Only marked local fixture documents permit same-origin framing; production headers are unchanged.


Campaign detail validation now includes populated one-sided/area/multi-sided dashboards, all three side filters against live Nuxt (fifty identical rows per side), source victim tint assignments, chart hover values/highlighting, clipboard references, failure/retry, and actual tab content after Back/Forward. Mobile columns use zero minimum tracks, matching Tailwind's grid sizing; intel name/ship lines retain the source row height and spacing while wrapping safely. Side changes reset the killlist page size to fifty, matching Nuxt's keyed component. Stale failed requests cannot overwrite a newer filter or navigate to an older tab. A failed killfeed offers a retry that retains its requested side/page/limit.

## Native fitting calculations

`internal/dogma` starts a bounded pool of native `shrike-dogma` processes, each
loading its SDE once. Go exchanges JSON lines with a worker through stdin/stdout.
Cancellation kills the busy worker and releases its pool slot; failures restart
it on the next request. The wrapper preserves existing API fields and module
default states, while `engine/dogma/vendor` retains the original Rust math.

`make build` and `make dev` build the worker when its source changes. Docker
builds it with Rust for the target architecture; the runtime image contains only
Go, the native worker and data. Browser WASM is served directly by Go. Node/npm
are only used to build checked-in assets and run browser/WASM regression tests.
See `engine/dogma/UPSTREAM.md` for the pin and coordinated upgrade policy. The current engine is 13.6.0 with SDE 12.3579973.0; `docs/architecture/dogma-engine.md` records the retained additions and upgrade differences.


Browser controllers now live in `frontend/src/browser` and are compiled into
`internal/frontend/static`. See [browser-assets.md](browser-assets.md) for source
ownership, shared chunks, automatic development builds and the TypeScript policy.
The coalition directory renders membership/name data first; combat totals arrive
through `/_fragments/coalitions/activity` without replacing cards or filter state.
