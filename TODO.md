# Go frontend pages

Check a route off only after its Go page matches Nuxt visually and behaviorally, its links point to the right routes, its controls work, and relevant tests pass. Track mobile and custom-theme checks in the final cross-route regression below. Braces list the tab routes to verify separately.
Redirects are checked after their status and destination are tested; destination pages have their own entries.

**Maintenance rule:** Preserve the migrated page markup, SVG geometry, assets and behavior. The original Nuxt source is retained in Git history and the private local archive `.data/source-archives/nuxt-20261006.tar.gz`; reusable browser helpers now live in `frontend/`. Read the existing implementation before changing a feature instead of inventing an approximate replacement. Keep native Rust and browser WASM engine/data versions synchronized, and validate optimizations against existing tests and browser behavior.

**Completed migration:** All 136 checklist entries are implemented and validated against the recorded scope. All 66 current Nuxt route files map to these page families, tabs or redirects. The final review covers desktop/phone/tablet layouts, themes, navigation, live/error flows, metadata/cache policies and measured query optimizations. The user physically confirmed fitting slot drops, slot moves, Cmd+Z and ship-image auto-fit. Nuxt/Go share the tested drag-payload helper (`make build-fit-drag-frontend`). Detailed evidence and validation limits are in the final review linked below. Keep the user's dev server on port 4000 separate from Nuxt comparison (4100) and Go preview (4200).

**Roam Report:** Imported the upstream Go API, response schemas/model, migration 28, and the later Nuxt dashboard/editing/composition/menu fixes. The configured database already has migration 28. Go pages use a shared primary reader, the existing killlist query, scoped compiled Nuxt utilities, the shared combat map, and a port of the UTC/local date picker. Browser comparisons covered populated and empty reports, all composition views/sorts/filters, map scopes and selections, mobile layout, creation/edit/cancel/share, errors, history, and killlist role/size/paging. All four API operations succeeded through `/docs`; the isolated local PostgreSQL integration test covers rebuilding and shared-reader parity. Existing reports were not edited during validation; the browser flows used newly created test reports. Preserve these APIs and reusable components for later ports.

**Advanced search:** Ported the source filter controls, autocomplete, tags/chips, UTC/local picker, cursor killlist, fitting groups/cards/drilldown, and saved-search document conversion. Search URLs render results in Go; subsequent changes replace only the results fragment. Side-by-side checks matched Nuxt rows, both cursor pages, page sizes, ISK shorthand/value ordering, combined tech filters, fitting family/exact groups, drilldown, and 100 appended individual fits. Mobile autocomplete, item slot/side, location/security, collapsed and empty states, and the saved-search popover were checked at 393px. Browser saves, reload/duplicate errors, account import/rename/replace/load/conflicts/refresh, delete cancellation, and custom-date Back history were tested with an isolated local account fixture; real PostgreSQL tests cover account isolation, revisions/deletion, filter exclusions, ordering/cursors, and fitted module/drone loading. Editor links were decoded in tests and opened successfully in the Go editor. The shared API now limits matching IDs before expensive display joins, removing a 90-day attacker-query timeout; one cold 50-row SSR check took 1.23s server time with two SQL queries and 3ms of template work. Keep the shared reader, FitPreview, saved-search utility, and original API for later pages.

**Campaign creator and owner editor:** Ported the source form, tracking modes, live campaign-card preview, participant/location pickers, visibility and area grants, UTC/local time controls, prize metrics/splits, wallet availability, and funding/retry contract. The owner form loads as an authenticated Go fragment and retains funded-rule locks, participant moves, save/cancel, and sign-in recovery. Browser comparisons covered desktop and 393px mobile, all three modes, multi-side changes and duplicates, area scope/grants, fixed/ongoing dates, prize split/reset/over-balance errors, funded creation and metadata editing, unfunded participant editing, failed save/retry, and expired authentication. All campaign and wallet writes used an isolated local PostgreSQL fixture; integration tests also cover authorization, private visibility, transaction rollback, funded idempotency, locked rules, and expired SSR sessions. The browser historical-window policy now matches the existing API's 365-day private limit. Reuse the shared source header/login gate, search picker, date picker, campaign card, and authoritative session reader for later ports. Campaign detail statistics, prize contribution/claim, intel/comments, and mobile hero parity are now validated separately below.

**Campaign detail progress:** Populated area and three-side dashboards match Nuxt's totals, side statistics, valuable kills, and charts. Compared 100-row and 10-row killfeeds and the second cursor page; opening a killmail retains its previous/next context. Corrected typography, responsive side columns, balance labels, victim-side valuable borders, semantic theme colors, and campaign row tints (character before corporation before alliance, applied without extra queries). The original Nuxt formatters are now bundled with campaign policy/palettes. Prize dialogs were compared with a disposable local database: exact use-max amount, invalid decimals, insufficient balance, Enter submission and receipt, contribution count/pool refresh, two winning-character claims, finalizing/projection, and intel limits/damage/more counts. SQL integration tests cover contribution idempotency/decimal precision, private claim permissions, pre-finalization rejection, and repeated claim rejection. The shared source comment composer and thread now support formatting, exact server preview, media/GIFs, replies/edits/reports/deletion, collapse, and live updates; desktop and 393px controls were compared against Nuxt. Remaining campaign checks now pass: all fifty rendered IDs match live Nuxt for each of the three BLOC sides; victim tint assignments match; one-sided and three-sided tooltip values match; prize references copy identically; a stopped owned API produces a clear error and Retry restores the selected fifty rows; Back/Forward restores intel/comments content; saved theme cookies apply; 393px hero/cards and intel match without horizontal overflow. Ported the hover column highlight and per-side active filter accents, preserved Nuxt page-size reset on side changes, and fixed mobile intel sizing/spacing. `scripts/frontend-parity` provides reusable isolated accounts/APIs and a render-only mode so assets can restart without losing test data.

**Shared comments:** Ported the existing `CommentInput.vue`, `KlipyPicker.vue`, recursive thread, and report dialog. Nuxt and Go share embed rules/thumbnails, selection insertion, and the existing `ek_klipy_favs` format. Server preview uses the actual Markdown/sanitization/media pipeline. Browser checks covered selection formatting, preview, YouTube and direct MP4 playback, GIF search/paging/favorites/recovery, length and keyboard shortcuts, posting/replies/edits/cancel, reports, deep-thread collapse/expand, deleted replies, expired-session retry, and drafts retained during live updates. The GIF provider and relay were deterministic local fixtures; auth, comments, Markdown, and reports used real APIs and disposable PostgreSQL. Fixed a real PostgreSQL report-recount parameter type error; SQL tests cover duplicate and concurrent reports/flagging. Production relay events omit Markdown, so editing fetches the current detail. Mobile composer/GIF layouts match at 393px without horizontal overflow. Rebuild shared utilities with `make build-comment-frontend`; keep the remaining page-level parity checks open.

**Faction warfare:** `/wars` renders without waiting for faction aggregates; browser fault injection verified usable filters during loading, unchanged card height, failure isolation, and Retry. Both faction-war details now use the authoritative cached readers, reusable members/intel/killlist components, and retained fragment tabs. Ported the existing hero, control bar, UTC flip groups, all leaderboard periods, side filters, cursor paging and page sizes, systems sorts, and warzone SVG. Nuxt and Go share the original geometry/color helpers; the two maps have identical bounds and 90/70 systems. Browser comparisons covered matching 100-row killfeeds, all 500 sorted members using the same API snapshot, populated intel, desktop/393px layouts, Back navigation, pan/zoom/reset, and system clicks. Fixed stale side rows when returning from non-combat tabs in both faction and declared wars. Read-only SQL comparisons confirmed identical results after replacing full attacker-table scans with per-killmail lookups; members/intel fragments subsequently returned in 3.24s/2.42s. Rebuild shared map helpers with `make build-faction-war-frontend`; the global theme/mobile regression is recorded in the completed final review below.

**Battle generator:** Ported all three wizard stages, responsive side lanes, system search/chips, UTC/local picker, login gate, preview, and save/correction flow. Nuxt and Go share participant grouping, moves, reload preservation, and edit seeding. Preview HTML reuses the existing battle summary and the API's authoritative calculation directly. Browser comparisons covered matching totals/breakdowns, side names/moves/filtering, multi-system reloads retaining assignments, empty windows, five-system/time limits, invalid assignments and retry, creation and correction at the same URL, expired-session draft retention, and 393px lanes/preview without horizontal overflow. Every write used disposable local PostgreSQL; integration tests cover shared-reader/API equality, creator retention, invalid-edit rollback, and expired sessions. Rebuild helpers with `make build-battle-generator-frontend`.

**Statistics:** Ported all 34 source cards and ten sections, with warm cards in SSR and cold cards loading as independent HTML fragments. The shared reader retains global/custom-board queries, number/date formatting, and all fourteen periods. Cache checks use one pipelined Valkey read; a cold document measured 2.5ms server time with zero SQL queries. Browser checks covered section filters, every period, populated desktop/393px layouts, URL history, fast changes, and an isolated failed card whose Retry restored data while filters and other tables stayed usable. Added stable ID tie-breakers for security scores and birth/founding timestamps so Go and Nuxt receive consistent rankings. PostgreSQL tests cover API/reader equality, domain scope, warm-cache reuse, and scope changes; cache tests cover cloning, stale entries, promotion, and mutation invalidation. The shared header supports rich description content and hides empty action slots on narrow layouts.

**Scan tools:** Local Scan and D-Scan now use the original page/result markup, shared header and scoped utilities, direct EVE entity images with fixed sizes, and source-stable group/pilot sorting. Browser comparisons covered clipboard and keyboard paste, threats/unresolved names, known and unknown scan types, nested collapse, authenticated save/share at the same URLs as Nuxt, public reloads, invalid/empty-input recovery, expired-session result retention, and desktop/393px layouts. PostgreSQL integration tests cover public analysis, save authorization, API/reader equality, and refreshed results after saving the same input again. Mutable shared results bypass the old 24-hour response cache; these are single indexed reads. All writes used disposable local accounts/database.

**Battle directory:** Compared populated cards and all five map scopes directly with Nuxt on desktop and at 393px. Counts/labels, region and system selection, ordered report IDs, mobile autocomplete, keyboard tabs, Back, and pan/zoom/reset are validated; mobile panning produced identical marker offsets in both renderers. Restored the shared header and source typography/icons, fixed the missing entity chip label, and used the cached site bootstrap for the custom-board badge. Geography and markers load independently with isolated errors and Retry; local fault tests covered delayed markers, both failure paths, recovery, and subsequent scope changes. Added opt-in loopback fault controls and SVG sprite integrity checks for future parity work.

**Settings progress:** All nine source sections are now ported: overview, wallet, sessions, description, ESI, appearance, preferences, Domains, and My Comments. Tabs retain DOM/drafts and editor URLs through navigation and Back/Forward. Desktop and 393px comparisons covered all 24 themes, preferences/time zones, metrics, devices/revocation, bio editors and pending submissions, wallet holds/ledger pages, ESI charts/scopes/filters/pagination/detail/live/retry, and private comment pagination/links/deletion/empty states. Domains now has all five editor tabs, subdomain/entity search, campaign policy and private audiences, theme/banner/opacity controls, nested navigation, and widgets. Real local APIs validated creation, saves and reloads, failed/expired saves and recovery, edits during saving, account isolation, asset approval/rejection/removal, and native browser selection/uploads for banner/logo/background. The native macOS picker works without changing Chrome extension access. Save patches and widget definitions are shared with Nuxt; newer edits stay dirty. Fixed stale pending assets, oversized widget pickers, mobile tab overflow, and the navigation limit in both renderers. A shared scrollbar correction restores identical mobile content widths. Asset versioning now rewrites only root static URLs, preserving API paths containing `/assets/`, with an embedded-controller regression test and a new transformation revision. Browser mutations used disposable PostgreSQL and memory-only image storage. Rebuild shared data with `make build-settings-frontend`, ESI helpers with `make build-esi-frontend`, and utilities with `make build-css-frontend`. Final account regression now covers reader bio permissions, domain validation and active/inactive lists, failed deletion retaining the board, recovery to matching empty lists, and individual/bulk session revocation spinners, failures and Retry. Both settings entries are validated.

**Account navigation:** Ported the original account card, all menu links, available-wallet badge, scope controls and seven login-delay steps. Generic public SSR remains independent of identity; one authoritative browser session request supplies the account and default-tab preferences. Desktop/393px comparisons cover owner/reader visibility, login-cookie persistence, failed login feedback, wallet refresh/failure/recovery, and logout. The notification bell merges comment replies and opted-in tracker events, preserves focused links during live inserts, and reconnects using the source relay transport. Real PostgreSQL and browser checks cover all reply destinations, read-cursor persistence, account isolation, mixed rows, mark-all-read and dropped-socket recovery. Reply targets now include slugs in the API/client contract; both renderers use account-scoped cursor caches and share formatting/topic helpers. Corrected the mobile menu structure, slider width and popover text colors. Error extraction supports Huma problem details and legacy messages. Rebuild with `make build-account-frontend`; the broader check passed 140 Bun tests, Nuxt typecheck, Go frontend/API/fixture tests, vet and builds. Tracker pages are validated below.

**Watch management:** `/trackers` now uses the original Nuxt layout, shared search picker and account state, source icon/image/time/topic helpers, and the existing server-rendered killlist. The homepage live-row renderer is extracted into a shared module; private feeds reuse it with bounded cursor pages, entity loss coloring, live insertion, queued updates on later pages, reconnection and teardown. Browser comparisons cover all six target types, duplicate prevention, creating watches with alerts off, pause/resume without backfill, independent alert toggles, matching cursor IDs/page sizes, losses, contextual kill navigation/Back, native confirmation cancellation, failed/successful deletion, deleting the last watch, expired-session recovery, isolated accounts, fragment Retry and 393px controls without overflow. OS permission branches are tested with a deterministic mock; no browser notification permission was granted. PostgreSQL tests exercise the real deferred trigger, recording versus alert opt-in, pause/resume, pagination, ownership and cascading deletion. Private fragments perform one existing owned-feed API read instead of loading every tracker again. The reusable private-feed controller also powers `/tracker` below. Checks passed: 142 Bun tests, Nuxt typecheck, Go frontend/API/fixture tests, vet and builds.


**Private dashboard:** `/tracker` reuses the source configurator and a single shared widget catalog, ordering helpers and relay filter. Summary/configuration render in Go; aggregate widgets load independently as private HTML fragments. Comparisons cover first visit, all leaderboards and valuable tabs, tracker creation with alerts off, retained failed saves, Cancel, notes/placement/order/enable/ratios, all four feed modes, matching cursor IDs, live filtering, isolated accounts and 393px containment. Delayed widgets survive editor cancellation; failed widgets recover with Retry. PostgreSQL tests verify deduplicated overlapping watches, filters and private configurations; frontend tests cover escaping, no-store/noindex, bounded fragments and concurrency. Checks passed: 144 Bun tests, Nuxt typecheck, Go frontend/API/fixture tests, frontend race tests, vet and builds. Rebuild source data with `make build-account-frontend`.

**Blog:** Reused the original listing, article typography, author card and shared comments; populated desktop and 393px comparisons use real local blog APIs via `scripts/frontend-parity -blog-fixtures`. Covers, missing excerpts, tags, all 44 published rows, empty filters, Back/clear, delayed failure/retry, archived URLs and all three preview states match Nuxt. Drafts are public 404s; readers/anonymous visitors cannot preview them, and admin previews are private/no-store/noindex. A real disposable comment saved under blog target 9 and appeared in both renderers. Fixed timestamp-tie pagination with a shared publication-time/ID cursor preserving database precision; legacy timestamps remain accepted, and the generated API contract now describes the cursor as a string. Matched the remaining shared comment empty/sign-in typography and theme color, and made both preview banners wrap on mobile. Checks passed: real PostgreSQL API integration tests, Go frontend/API/fixture tests, frontend/fixture race tests, vet/builds, 144 Bun tests and Nuxt typecheck.

**Admin progress:** All eleven admin sections and the standalone user/domain details are implemented and validated in Go using the original sidebar, templates and authenticated APIs. Overview counters/attention badges and the shared ESI volume chart load independently; tab fragments retain visited DOM, filters and Back/Forward. User comparisons cover 56 users across both pages, sorting/search/empty states, token/scopes without secrets, UTC dates, exact configuration JSON/escaping, Discord save/clear/failed-draft recovery, role controls, self-revocation protection and the actual ESI deep link. ESI comparisons cover all endpoint/source/status/new-data/entity/search filters, both cursor pages, requested-by portraits, safe error text, stored/pending discovered killmails, actual kill links, filtered live inserts, pause/resume and hidden-tab recovery. The shared settings log also receives matching account-scoped inserts. Fixed Nuxt polling to replace its shallow response ref instead of mutating invisible nested rows; both renderers share immutable, bounded batch deduplication. Entity selection uses supported 32px EVE images rendered at 16px. Announcement comparisons cover all five status filters, all three tiers, UTC/local schedules, ticker durations, optional fields, Markdown text, create/edit/cancel, two-click archive, failed saves/archives/list retry, expired-session draft recovery and retained Back filters. The original form/status/payload helpers are shared with Nuxt; reusable modal and toast components support later ports. Desktop and 393px layouts match, including the corrected mobile modal footer and ESI detail width. Real PostgreSQL tests cover authorization, private reads/writes, filtering, paging, live cursors, stored killmail projection, schedules, failed-edit rollback, archival and Discord/role validation. Reproduce data with `scripts/frontend-parity -admin-fixtures`; rebuild helpers with `make build-admin-frontend`, `make build-esi-frontend` and `make build-css-frontend`. Checks passed: full local PostgreSQL API suite, Go frontend/API/fixture tests, frontend/fixture race tests, vet/builds, 148 Bun tests and Nuxt typecheck. All admin sections are now validated; see the individual notes below. Shared public announcement delivery is now validated separately below. Fresh browser storage confirms both notification bells show the same seven mixed reply/tracker rows; the earlier 2-vs-7 difference came from reusing fixture IDs with retained browser cursors.


**Admin Domains:** Ported the original local directory filters and standalone detail, with shared counts/search/status/size helpers and the existing authenticated APIs. Desktop and 393px comparisons cover all/pending/inactive filters, combined owner/site/subdomain/hostname search, retained section filters through Back, refreshed pending counts, owner links, entity chips, UTC metadata, Approvals/Files tabs, logo/banner/background previews, all review states and empty/missing records. Real disposable reviews and activation changes persist; approved logos update the managed theme URL. Failed reviews preserve drafts, duplicate submissions are disabled, successful writes followed by failed reads retain the last view and offer Retry, and expired-session drafts recover without leaking private data. Reused previews remain loaded across Go fragment refreshes; hidden file images load on first tab selection. Corrected clipped mobile review controls and contained file-table scrolling in both renderers. Nuxt also retains the last successful detail after a failed refresh and now exposes directory/review errors. PostgreSQL tests cover admin-only reads/writes, scoped private image bytes, review metadata, already-reviewed conflicts, theme updates and colliding activation rollback; frontend tests check auth before data, escaping, no-store/noindex and invalid/missing IDs. Reproduce four domains and six PNG assets with `-admin-fixtures`. Validation passed: full local PostgreSQL API suite, Go frontend/API/fixture tests, frontend/fixture race tests, vet/builds, Bun tests and Nuxt typecheck. The final fitting/map/full-review checks remain open.

**Admin Campaigns:** Ported the source cards, processing metadata and controls, sharing all six state options, status priority and runtime/processed-count helpers with Nuxt. Real local comparisons cover identical 56 IDs across 40/16-row pages, all filters, creator/name/ID/empty searches, slow-search supersession, retained section filters through Back, actual campaign navigation, pause/resume/reprocess/archive, disabled row controls, clear failures and expired-session recovery. Successful writes followed by failed reads retain the list and offer Retry; failed pagination preserves the visible page number and can retry the same Next request. Delete confirmation expires after five seconds, funded archive/delete guards appear correctly, and PostgreSQL tests prove actual deletion in an isolated database. Added a stable campaign-ID tie-breaker to the shared API so equal creation timestamps cannot duplicate rows across pages. Desktop and 393px cards/controls match without horizontal overflow; refined shared select/admin icon sizing to use the source dimensions. Nuxt recovery keeps the original SSR rows; both documents contain the first 40 campaign links. Seed repeatable metadata with `-admin-fixtures`; rebuild helpers with `make build-admin-frontend`. Checks passed: full local PostgreSQL API suite, Go frontend/API/fixture tests, frontend/fixture race tests, vet/builds, 155 Bun tests and Nuxt typecheck. The grouped admin entry is now validated.

**Admin Moderation and Comments:** Both source queues now render in Go with reusable author/context/preview markup and one scoped fragment controller. Comparisons cover all twelve moderation kind/status combinations, all three comment filters and the bounded 100-row view, matching statuses, AI rounding, UTC review metadata, rendered/raw bios, image/video previews and all context types. Legacy blog slugs are resolved in a batch; the actual article link works in both renderers. Real local browser writes cover comment/bio approve and reject, already-reviewed protection, cross-domain hide/restore and soft deletion with the correct reviewer. Failed writes retain rows; failed reads after successful writes preserve the list with Retry. Initial failure/retry, slow-filter cancellation, expired-session recovery, five-second delete confirmation and retained section filters through Back are validated. Loaded media nodes survive result refreshes. Desktop and 393px layouts have matching card heights, responsive media and no horizontal overflow. Stored HTML is sanitized with the existing comment/bio policies; safe combined EVE color/font declarations survive while active content is removed. Disposable PostgreSQL integration tests cover private authorization, publication, filtering, domain isolation and soft-delete restoration. Checks passed: full local API suite, Go frontend/API/fixture tests and race tests, vet/builds, 157 Bun tests and Nuxt typecheck. Seed with `-admin-fixtures`; rebuild shared helpers with `make build-admin-frontend` and preview styles with `make build-css-frontend`. All admin sections are now validated.

**Admin River Workers:** Ported source queue cards, worker/cron status, native selectors, job table and inspector, sharing state/date/JSON helpers with Nuxt. Queue and job reads run in parallel during SSR, and later refresh independently through private fragments. Desktop and 393px comparisons cover all seven state filters, four populated queues, empty results, matching 100/37-row pages, long nested inputs, durable output, literal errors, worker IDs, Escape, mobile filters and contained horizontal/vertical scrolling. Inspector dimensions and scroll areas match the source. Browser checks cover actual local cancel/rerun and pause/resume, disabled controls, independent failed reads after writes, both initial failure/Retry paths, delayed-filter supersession, retained filters through Back, expired-session recovery and independent five-second confirmations. The shared API's exclusive cursor now returns every job without skipping the lookahead row; both SSR documents contain 100 distinct rows. The Go DTO reuses `queue.Depth`. Isolated PostgreSQL tests prove privacy, all pages, inspection, controls, running-delete protection and scoped finalized-history clearing; no fixture workers run. Validation passed: full local API suite, frontend/fixture race tests, vet/builds, 159 Bun tests and Nuxt typecheck. Seed 137 jobs with `-admin-fixtures`; rebuild shared helpers with `make build-admin-frontend`. The grouped admin entry is now validated.

**Admin Wallet:** Ported the source authorization metadata, balances, journal, references and settlement cards with the existing APIs, public-wallet DTO/money helpers and shared Nuxt display/grouping helpers. Desktop and 393px comparisons cover all 115 journal IDs across 50/50/15-row pages, all seven divisions, missing balance divisions, large/nullable/negative amounts, all reference statuses, mixed claim/payment states, literal error text and contained table scrolling. Both SSR documents contain 50 rows without ESI tokens. Real local browser writes cover deduplicated sync, its disabled controls/failures, payment-note Cancel, failed prize records retaining the winner, successful notes/reviewer persistence, expired sessions and recovery. Initial failure/Retry, failed pagination with repeatable Next, slow-filter supersession, empty contract, disabled/missing authorization, retained filters/Back and callback text/clearing are validated. Authorization navigation uses a read-only loopback response; isolated API tests separately verify the real start endpoint's required scopes and bound state. Fixed Nuxt's deferred route keeping Wallet visible under another tab URL; the retained shell and callback banner now follow the router's current route. Shared settlement queries have a campaign-ID tie-breaker, and non-finite money values cannot crash SSR. Validation passed: full local PostgreSQL API suite, frontend/fixture race tests, vet/builds, 161 Bun tests and Nuxt typecheck. Seed balances, journals and prize states with `-admin-fixtures`; no workers run and fake ESI tokens never leave the fixture. The grouped admin entry is now validated.

**Admin Blog:** Ported the original list, all four status filters and 768px editor using the existing private CRUD/preview APIs, public post DTO, shared source slug/tag/form/payload helpers and EVE date picker. Desktop and 393px comparisons cover identical ordering/text/card dimensions, draft/published/archive states, optional fields, tag previews, stored Markdown, safe literal metadata, editor prefill, UTC publication dates, Cancel/Escape and retained filters through Back. Both initial documents contain all 49 current comparison posts with no-store/noindex headers. Real local browser creates/updates persist titles, tags, Markdown and the chosen UTC date. Failed saves preserve every field and disable duplicate requests; expired-session errors retain the draft through account restoration. Failed filters retain the previous list with Retry; initial failure/Retry, true empty responses and slow-filter supersession are validated. Delete arming expires after five seconds; blocked local deletes show errors and keep rows, while isolated API tests prove actual deletion, authorization, preview privacy, conflicts and invalid-edit rollback. A long-word stress test exposed horizontal overflow in both implementations; titles, excerpts and tag chips now wrap, leaving the document at 393px. Shared API ordering includes the ID for tied creation dates. Validation passed: full local PostgreSQL API suite, frontend/fixture race tests, vet/builds, 163 Bun tests and Nuxt typecheck. Seed the existing posts with `-blog-fixtures`; use disposable editor drafts for stress cases. Rebuild shared helpers with `make build-admin-frontend`. The complete admin overview/sidebar was rechecked against Nuxt, so both grouped admin entries are checked off. Remaining work: AIID mobile/audio, fitting drag/hull cases and the full review across all routes.

**Development reload:** Fixed a real multi-tab stall during fitting validation: per-document EventSource streams reserved every HTTP/1 connection, leaving WASM/price requests pending. Development pages now share one stream through a SharedWorker, with bounded version reads for bootstrap/BFCache and polling fallback when workers are missing or blocked. Broker tests cover seven documents, stale-version avoidance and last-client cleanup; fallback tests cover pagehide/restoration and changed-version reload. A fresh Rifter calculates correctly with nine Go documents open, and seven admin documents plus the fitting editor recover across a process restart while preserving the fit URL. This remains development-only. The native automated drag currently makes no change in either renderer; the fitting entry stays open pending a real gesture.

**Further fitting checks:** Rifter hull/resource/stat calculations and Tengu subsystem search rankings match Nuxt. Four real subsystem selections produce identical module rows and dynamic slot groups; full subsystem slots refuse another add, and removing/replacing the offensive subsystem matches the source. Browser validation exposed the old Go selection panel intercepting lower table rows. It is removed; table selection is separate from catalog selection and uses the source mobile Remove/Unload/Fill highs strip. The strip stays below fixed navigation, the sidebar remains 320px on mobile table views, and named columns keep minimum widths with scrolling contained in the table. Real desktop double-click removal/replacement and mobile Remove are verified. Vargur/Bastion fitting rows and state menus match; Online/Active switching produces the same 43,651/62,358 EHP and 137.5/0 m/s. Rifter autocannon/EMP S tables match, and double-click unloads the charge while keeping the weapon. Seven fitting helper tests and frontend/fixture race checks pass. Native automated drag has made no change in either renderer and remains unverified; the user has been offered a manual check. Fixed the reference Nuxt phone table too: the toolbar wraps, the sidebar stacks, resource cards use two columns, statistics use the available width, and fullscreen tables retain usable height. Both versions share checked-in column preference/minimum-width helpers; 393px browser checks cover contained horizontal scrolling, sticky actions, Fill highs, Remove and Undo. The fitting checklist remains open for a real HTML drag gesture; cross-route mobile/theme regression remains in the final full review.

**AIID mobile and alarms:** Both maps use the same shared Web Audio core and retain independent near/outer throttles. Fixed suppression of alerts during the first 750ms after loading. Browser comparisons use real Jita data (381 systems/16 regions), real AudioContexts and an opt-in local passthrough analyser: matching three-pulse near and two-pulse outer output has a nonzero destination waveform. Four-, five- and ten-jump boundaries, duplicate/stale/future/out-of-area events and disabled alarms behave identically. Local relay disconnects recover and accept the next kill; watch changes retain the connection, streamed rows and freshness history instead of discarding them. Search/add/remove/empty/eight-anchor coverage and URL checks, native pan/zoom/reset, and 320/393px controls without document overflow pass. All test events use the disposable relay. Rebuild audio/geometry with `make build-map-frontend`.

**Shared announcements:** Ported the source banners, mandatory acknowledgment modal/queue and compositor-driven news ticker on every Go document, with no additional SSR identity/feed query. Nuxt and Go share category caps, ordering, palettes, expiry/fading and immutable lifecycle/snapshot reconciliation. Desktop and 393px comparisons cover rich/sanitized content, all three tiers, links/Back, queued acknowledgments, anonymous reload persistence, ticker collapse/keyboard expansion and matching 49px/19px heights, live inserts/updates, timer-only expiry, last-item removal, failed initial fetch/recovery and dropped-socket catch-up. Real owned-account dismissals persist in PostgreSQL and are honored across the two renderer origins; tests cover account isolation and unchanged public data. Fixed missing event creation timestamps, stale cached public feeds after admin mutations and replica-local announcement caching. Reconnects revalidate the feed, and queued live changes survive an older HTTP snapshot. Floating controls/diagnostics move above the ticker in both renderers. Configurable Lucide icons reuse the original Iconify collection: common icons keep the small sprite, while all 2,083 names/aliases are available as individual immutable SVG assets (the tested rainbow downloads 297 bytes). The raw source collection is not served. Rebuild helpers with `make build-announcement-frontend`, icons with `make build-icons-frontend`, and utilities with `make build-css-frontend`. `-admin-fixtures` now keeps the public announcement API local and supplies an opt-in lifecycle/reconnect transport; the Source comparison also forwards its icon endpoint to Nuxt. Validation passed: 151 Bun tests, Nuxt typecheck, full local PostgreSQL API suite, Go frontend/API/fixture tests, frontend/fixture race tests, vet and builds. The final cross-route mobile/custom-theme review remains open.


- [x] `/`
- [x] `/kills/{type}` (all supported kill categories)
- [x] `/character/2121270074`
- [x] `/character/2121270074/combined`
- [x] `/character/2121270074/kills`
- [x] `/character/2121270074/losses`
- [x] `/character/2121270074/top`
- [x] `/character/2121270074/battles`
- [x] `/character/2121270074/wars`
- [x] `/character/2121270074/campaigns`
- [x] `/character/2121270074/history`
- [x] `/character/2121270074/achievements`
- [x] `/corporation/98598862`
- [x] `/corporation/98598862/combined`
- [x] `/corporation/98598862/kills`
- [x] `/corporation/98598862/losses`
- [x] `/corporation/98598862/top`
- [x] `/corporation/98598862/battles`
- [x] `/corporation/98598862/wars`
- [x] `/corporation/98598862/campaigns`
- [x] `/corporation/98598862/members`
- [x] `/corporation/98598862/history`
- [x] `/alliance/99003581`
- [x] `/alliance/99003581/combined`
- [x] `/alliance/99003581/kills`
- [x] `/alliance/99003581/losses`
- [x] `/alliance/99003581/top`
- [x] `/alliance/99003581/battles`
- [x] `/alliance/99003581/wars`
- [x] `/alliance/99003581/campaigns`
- [x] `/alliance/99003581/corporations`
- [x] `/alliance/99003581/members`
- [x] `/faction/500003`
- [x] `/faction/500003/combined`
- [x] `/faction/500003/kills`
- [x] `/faction/500003/losses`
- [x] `/item/11186`
- [x] `/item/11186/kills`
- [x] `/item/11186/fittings`
- [x] `/item/11186/market`
- [x] `/item/34/kills`
- [x] `/item/34/market`
- [x] `/item/{item_id}/{kills,market}` where available
- [x] `/group/{id}` and `/group/{id}/kills`
- [x] `/system/30002813`
- [x] `/system/30002813/kills`
- [x] `/system/30002813/battles`
- [x] `/system/30002813/map` (direct tab and Map navigation)
- [x] `/constellation/20000396`
- [x] `/constellation/20000396/kills`
- [x] `/constellation/20000396/battles`
- [x] `/constellation/20000396/map` (direct tab and Map navigation)
- [x] `/region/10000042`
- [x] `/region/10000042/kills`
- [x] `/region/10000042/battles`
- [x] `/region/10000042/map` (direct tab and Map navigation)
- [x] `/categories`
- [x] `/map` (Nuxt/Go desktop geography comparison, URL-backed controls, region metrics and hover highlighting, region/system search, empty results, region clicks, and return navigation tested; AIID and mobile regression tracked separately)
- [x] `/map?base=security` (data-backed Nuxt/Go desktop comparison, search and region navigation tested; mobile regression tracked below)
- [x] `/map?base=sovereignty` (data-backed Nuxt/Go desktop comparison, ranked alliance sidebar, direct EVE logos, search, territory focus and hover card, recent-change rings, and alliance navigation tested; mobile regression tracked below)
- [x] `/map?base=live` (data-backed Nuxt/Go desktop comparison, kill feed, relay updates, pulses, row focus and links tested; mobile regression tracked below)
- [x] `/map?base=aiid` (populated desktop and 320/393px comparisons; watch search/add/remove/empty/eight-anchor limit, matching coverage and URLs, pan/zoom/reset, fresh live feed and reconnect tested; real Web Audio output matches near/outer pulses at 4/5/10-jump boundaries; duplicate/stale/future/out-of-area and disabled alarms stay silent; watch changes retain the socket, live rows and freshness history)
- [x] `/map?activity={activity,danger,ship-kills,pod-kills,npc-kills,jumps}` (all 24 metric/window combinations match Nuxt peaks; heatmaps, legends, hourly-data status, and region hover metrics compared on desktop; mobile regression tracked below)
- [x] `/map?scope={zarzakh,wormhole,abyssal,proving}` (all four scopes match Nuxt geometry and system/region counts; scope switching, search, and region navigation preserving filters tested; mobile regression tracked below)
- [x] `/map/region/{id}` (Nuxt geometry, mini systems, gates, hover details, search, pan, zoom, reset, filters, desktop/mobile visuals tested)
- [x] `/map/region/{id}?base=security` (security colors and controls compared with Nuxt on desktop and mobile)
- [x] `/market`
- [x] `/market/{category/...}` (nested groups, breadcrumbs, item links)
- [x] `/market/item/{id}`
- [x] `/advancedsearch` (filters/search, cursor pagination, grouped fits/drilldown, browser/account saved searches)
- [x] `/campaigncreator` (create/edit flow)
- [x] `/tools` (Roam Report card and mobile menu navigation validated)
- [x] `/tools/localscan` (keyboard/clipboard analysis, authenticated save/share, public reload, expired-session retention, desktop/mobile comparison)
- [x] `/tools/localscan/{hash}`
- [x] `/tools/dscan` (clipboard analysis, known/unknown types, group collapse, authenticated save/share, invalid-input recovery, desktop/mobile comparison)
- [x] `/tools/dscan/{hash}`
- [x] `/tools/roam-report` (menu/tools links, fleet paste and deduplication, UTC date pickers, pilot/time limits, creation, errors, and shareable report navigation)
- [x] `/tools/roam-report/{id}` (Information dashboard, hero/metrics, warnings, copy link, inline edit/cancel/rebuild at the same URL, unavailable report handling, and tab navigation)
- [x] `/tools/roam-report/{id}/information` (direct Information-tab alias)
- [x] `/tools/roam-report/{id}/kills` (shared killlist, kills/losses/all filters, friendly losses, pagination, and refreshed rows after editing)
- [x] `/tools/roam-report/{id}/systems` (shared combat map, scope/region/system selection, ordered systems, engagement trail, filters, and links)
- [x] `/tools/roam-report/{id}/intel` (specialist hulls, fleet contributors, opposing corporations/ships, and loss threats)
- [x] `/tools/roam-report/{id}/composition` (all roster pilots including unobserved hulls, pilot/ship/group views, search, class/loss filters, sorting, and empty states)
- [x] `/legacy`
- [x] `/legacy/kill/{id}`
- [x] `/coalitions`
- [x] `/coalitions/{slug}`
- [x] `/rankings`
- [x] `/battlegenerator` (source wizard, location/time controls, side assignment/filter/reload, preview, creation and correction, authentication errors, desktop/mobile comparison)
- [x] `/comments`
- [x] `/wars` (faction warfare cards load as an independent HTML fragment; the directory does not wait for cold aggregates, loading space is reserved, and failures can be retried)
- [x] `/war/{id}`
- [x] `/battles` (source header/cards/filters, desktop and 393px comparison, all five map scopes, matching markers and reports, autocomplete, keyboard tabs, Back, pan/zoom/reset, independent map/marker loading and failure/retry)
- [x] `/battle/{id}` (Nuxt header, themed team cards, expandable alliance/corporation and unsided breakdowns, loss highlights, fixed image dimensions, SEO, strict routes, and generated-killmail failure/redirect handling; desktop and 393px comparison validated)
- [x] `/battle/{id}/{kills,timeline,replay,composition,intel,comments}` where available (fragment tabs and Back/Forward; matching kill IDs, side filters, paging and 100 rows; timeline totals and ISK bars; composition search/views/sorting; matching populated intel rows and mobile layout; shared comments; replay feed/totals, shared projection utilities, playback, scrub, focus, celestials, pan/reset, and failure/retry)
- [x] `/campaigns`
- [x] `/campaign/{id}` (populated area, one-sided and three-sided dashboards; prize funding/claim; all side filters, tints, paging, chart hover, clipboard, failure/retry, saved-theme cookie, and 393px hero/card layout validated)
- [x] `/campaign/{id}/{intel,comments}` (populated intel limits, damage and mobile rows; shared comment controls; fragment navigation and Back/Forward content validated)
- [x] `/faction-war/{matchup}` (Caldari–Gallente and Amarr–Minmatar; hero/control/flips, all seven retained tabs, leaderboard periods, cursor killlists, member side/sort controls, intel, systems table, and original warzone map)
- [x] `/stats` (all source cards/sections/periods, independent cold fragments and Retry, scoped boards, deterministic ranking ties, history, desktop/mobile comparison)
- [x] `/graph` (all nine source tools; independent API loading, original formatters, entity/character autocomplete, type controls, coalition expansion, retained tab results, and URL/keyboard navigation. Live Nuxt comparisons covered populated paths, both intel/hunting/spy entity types, census totals/rows, migration, hot zones, coalitions, and both 50-row rivalry lists. Desktop geometry and 393px containment validated; injected failure/Retry and delayed-response cancellation passed. Rebuild shared formatters with `make build-formatters-frontend`; global theme regression is recorded in the completed final review below.)
- [x] `/fits` (six cached source feeds, ranking/doctrine modes, desktop/mobile cards, working links into search/detail/editor, and isolated feed failure/fragment retry validated)
- [x] `/fits/search` (role and specific-module predicates, count/operator controls, statistics ranges and sort, NPC profiles, capacitor stability, histograms, autocomplete, 24-row pagination, deep links and Back/Forward; populated desktop/393px comparisons, editor payload, and scoped failure/retry validated)
- [x] `/fits/create` (Archon wheel/stats, fullscreen scaling, observed fits, state/undo, unsaved URL reload, remembered workspace and keyboard commands compared with Nuxt. Vexor drones/cargo/tree previews, four-subsystem Tengu builds/replacement, Vargur/Bastion states and Rifter/EMP S charge tables match. EFT paste/export matches byte for byte; private/public save/update used isolated PostgreSQL and Save/Rename/New dialogs were compared. Both phone tables now contain horizontal scrolling and expose sticky actions; Fill highs/Remove/Undo and usable fullscreen tables are verified at 393px. Real slot drops, slot moves, Cmd+Z and ship-image auto-fit are manually confirmed.)
- [x] `/fit/{fit_id}` (shared source wheel/editor; owner updates, rename/undo/New confirmation, editable viewer preview/Fork, exact EFT export, preserved visibility, ratings/clear/error, lazy slug comments/teardown, 403/404/error cards, private/no-store/noindex, desktop/393px comparison, and source strategic-cruiser capacity/validity fallback; all writes validated in disposable PostgreSQL, plus a live public fit comparison)
- [x] `/search`
- [x] `/post` (shared queued-killmail watcher/notifications)
- [x] `/blog` (source cards/header, 44 posts across three cursor pages, tag/empty/clear/Back behavior, failed load-more and recovery, desktop/393px parity)
- [x] `/blog/{slug}` (source article geometry, Markdown, covers, author/affiliation links, archives and real blog-target comments; desktop/393px parity)
- [x] `/blog/preview/{slug}` (draft/published/archived statuses, unpublished note, admin-only access, private no-store/noindex, mobile banner wrapping; Nuxt parity)
- [x] `/faq`
- [x] `/docs` (OpenAPI loading, endpoint navigation, try-it requests)
- [x] `/mcp`
- [x] `/status`
- [x] `/wallet`
- [x] `/about`
- [x] `/donate`
- [x] `/settings` (all nine sections and authenticated navigation validated)
- [x] `/settings/{wallet,sessions,description,esi,appearance,preferences,domains,comments}` (all nine sections; desktop/mobile, real local account APIs, errors, recovery and retained drafts validated)
- [x] `/tracker` (source configurator, all ten widget types, private summary/stats, independent fragments, notes/order/placement/ratios, filters, cursor feeds and live updates; real local API, account isolation, errors/recovery, desktop and 393px parity validated)
- [x] `/trackers` (six watch types; create, select, pause/resume, opt-in alerts, cursor feeds, live updates, delete confirmation, errors/recovery, account isolation and desktop/393px parity validated)
- [x] `/admin`
- [x] `/admin/{users,domains,campaigns,moderation,comments,esi,river,wallet,announcements,blog}`
- [x] `/admin/users/{id}` (private detail, roles/Discord, token/config, ESI deep link and desktop/mobile validation)
- [x] `/admin/domains/{id}` (private approvals/files, real reviews and activation, preview scope, failed refresh/retry, expired-session drafts, desktop and 393px parity validated)
- [x] `/kill/{id}`
- [x] `/ship/{id}[/{tab}]` → `/item/{id}[/{tab}]`
- [x] `/shipgroup/{id}[/{tab}]` → `/market/ships`
- [x] `/wars/{id}` → `/war/{id}`
- [x] `/killmail/{id}` → `/kill/{id}`

Supporting public routes to retain when Nuxt is removed:

- [x] `/feed.xml`
- [x] `/icon.png` (RSS image)
- [x] `/llms.txt`
- [x] `/mcp.txt`
- [x] `/_sitemap-data/{kind}` (with complete dynamic XML sitemaps)
- [x] Final full review across all ported routes: mobile layouts, custom themes, navigation, live updates, errors, relevant tests and performance; fix and verify any issues found before completing the goal.

Full review is complete for the recorded scope; evidence, comparison ports,
measured query improvements and validation limits are in [docs/operations/frontend-review.md](docs/operations/frontend-review.md).

Site-wide loading feedback is implemented: a themed top bar, clicked-control/
killmail-row pulse, native page-navigation events and shared fragment busy
tracking. Delayed/error/cancelled/cached fragment checks and native lifecycle
tests pass; native Chrome capture confirms the pending full-document bar and
row highlight during a five-second local-fixture delay. The full Go preview
is on 4200; 4451 is an account test fixture.

**Runtime cleanup, 2026-10-06:** Retired the Nuxt tree and Bun tooling. Reusable TypeScript/browser helpers, the generated API contract and browser/WASM tests moved to `frontend/` with npm/esbuild/Vitest. Server fitting calculations use the original pinned Rust engine through a persistent JSON-line worker in `engine/dogma/`; browser WASM and its SDE remain at the same version. Public APIs and the isolated Go/API browser fixture are retained. Historical validation notes above describe the completed Nuxt comparisons.

**Dogma upgrade, 2026-10-06:** Upgraded the native and browser calculations to Rust 13.6.0 with matching SDE 12.3579973.0. A shared Rust adapter retains remote totals and the editor/API contract. Retired ancillary-effect injection and ammo overlays covered by upstream; pinned the Go FlatBuffer converter. Upgrade analysis and changed reference results are in `docs/architecture/dogma-engine.md`. Production fitting-stat backfill/distribution rebuilding remains a deployment step, not a development database mutation.

**Browser consolidation, 2026-10-07:** Browser sources now compile through a shared TypeScript/esbuild pipeline. `site.js` contains the common shell; page bundles and shared chunks retain feature-specific loading. `make dev` automatically builds/watches browser sources with Air, and `make build`/Docker compile production assets. Stimulus now owns select menus, category filtering, legacy attackers and coalition-activity lifecycles.

The original page migration is complete; the separate full TypeScript conversion remains open:

- [x] Shared browser source ownership, dev watcher and production builds.
- [x] Common site entry plus page bundles/shared chunks; standalone pre-paint appearance.
- [x] Introduce typed Stimulus controllers and fragment/lifecycle regressions.
- [x] Convert the initial 87 JavaScript controllers recorded in the former `frontend/browser-legacy.json` to checked TypeScript by page family, preserving existing DOM and behavior.
- [x] Replace manual component initialization/disposal with Stimulus where it removes real boilerplate; keep pure helpers and transport as ordinary typed modules.
- [x] Repeat the affected family's browser/mobile/error/live checks after each conversion.

See [browser source/build architecture](docs/architecture/browser-assets.md) for ownership, measured bundle sizes and reload limitations.

**Active browser migration goal, 2026-10-07:** Finish all 87 remaining controller conversions to strict TypeScript, remove `allowJs` and the legacy exceptions, consolidate reusable behavior, and use Stimulus for DOM lifecycle/events where it reduces boilerplate. Preserve Go/templ rendering, existing DOM/CSS, themes, public APIs, shared/page bundles and dev/production builds. Validate each family and finish a full desktop/mobile, navigation/focus/loading/error, WebSocket, fitting/map and build/test pass. No deployment or production data mutation is included. Pure helpers, transport and calculations remain ordinary typed modules.

Browser migration checkpoint: 87 of the initial 87 modules are converted and browser validated (`shell`, `account`, `notifications`, `tracker-state`, `reload`, `reload-channel`, `reload-worker`, `navigation-progress`, `diagnostics`, `announcements`, `search-picker`, `kill-row`, `private-killfeed`, `trackers`, `tracker-dashboard`, `entity-search-select`, `esi`, `wallet`, `settings`, `settings-domains`, `admin`, `admin-review`, `date-time-picker`, `admin-announcements`, `admin-blog`, `admin-campaigns`, `admin-domains`, `admin-river`, `admin-wallet`, `advanced-search-state`, `saved-searches`, `advanced-search`, `comment-stream`, `klipy-picker`, `comment-input`, `comments-feed`, `comment-thread`, `campaign-creator-state`, `campaigns`, `campaign-activity`, `campaign-prize`, `campaign-owner`, `campaign-creator`, `campaign-detail`, `battle-composition`, `battle-replay`, `battle-detail`, `battle-generator`, `battles`, `blog`, `search`, `kill-submit`, `post`, `coalitions`, `entity`, `home`, `kill-detail`, `map-page`, `map-live`, `fit-capacity`, `fit-resources`, `fit-stats`, `fit-variants`, `fit-wheel`, `fit-search-state`, `fits-catalog`, `fit-detail`, `fits-search`, `fit-editor`, `wars`, `war-detail`, `faction-war`, `faction-war-map`, `roam-composition`, `roam-report`, `scans`, `docs`, `mcp`, `statistics`, `rankings`, `graph`, `graph-render`, `status`, `legacy`, `market`, `universe`, `system`); all 87 sources are strict TypeScript and have family browser acceptance; the full final audit is complete. The accepted battle checkpoint passed strict checks and 405 browser/WASM tests. All admin components now use Stimulus and shared request ownership; no admin legacy factories remain. Editors retain newer drafts during saves, confirmations clear on tab exit, and canceled work reconciles with reads. Desktop/390px checks cover forms, calendars, filtering, Back, loading/errors/retry and lazy domain previews. Calendars open upwards inside scrolling modals when needed. Fresh final desktop/phone balance refresh and local failure/recovery checks pass; the earlier stall's cause was not proven. Advanced search and saved searches now use scoped actions, typed filter/document models, shared request ownership and deduplicated suggestions. Desktop/phone checks cover account and browser saves, rename/import/load, filter links, Back, fitting mode and failures. Comment components now use scoped actions, typed API/media models, shared socket transport, dialog cleanup and retained drafts/selection. Desktop/two-tab/phone tests cover live create/edit/replies, preview/formatting, media paging/favorites/errors, reports and feed filtering/paging/retry. Campaign components now use typed drafts/API models, scoped actions, request ownership and polling cleanup. Desktop/390px checks cover creation, owner editing, newer drafts during delayed saves, list search/paging/Back, tabs/retry, charts and funded-rule locks. Phone calendars clear the fixed navbar; chart labels stay readable and prize rows contain payment statuses. Funding/claim transport is tested with mocks; no real financial action ran. Battle components now use typed models and scoped lifecycles. Desktop/390px checks cover composition, replay playback/panning/focus/layers, tab fragments, real loss paging, generator previews and retained drafts during a local delayed save, and list/map filtering/Back/errors/retry. All original family conversions and browser acceptance are complete; final cross-site/runtime validation remains open. The goal remains active.


**Paused checkpoint, 2026-10-08:** User requested a pause for the day. The last fully accepted checkpoint remains 44/87 modules and 380 passing tests. Battle composition and replay are now TypeScript work in progress, with typed composition/replay boundaries and temporary compatibility bridges for the still-legacy battle-detail parent. Strict typecheck and their existing composition/replay tests pass; DOM/browser validation and the rest of the battle family remain outstanding. `browser-legacy.json` now contains 41 JavaScript files: two additional TypeScript conversions have not yet been accepted. Generated assets and running preview remain at the validated campaign checkpoint. Resume with battle-detail, generator and directory; replace the temporary bridges with Go-declared Stimulus scopes, then test desktop/mobile composition, replay playback/pan/focus/layers, filters, navigation, generator and map behavior before updating accepted counts.


**Resumed battle checkpoint, 2026-10-08:** All five battle modules are now accepted, superseding the paused WIP note above. Temporary parent bridges are removed; Go templates declare Stimulus scopes/actions. 49/87 converted, 38 JavaScript modules remain; strict TypeScript and 405 browser/WASM tests in 85 files pass. Production browser assets and Go preview/facade binaries build successfully. Public data remained read-only; battle #1 was created only in the retained disposable account fixture. Full-site/remaining-family validation and allowJs/legacy removal remain outstanding. Resume with war/faction-war modules.

**War TypeScript WIP, 2026-10-08:** Four additional modules are typed and tested,
with 427 tests passing; browser-legacy.json contains 34 JS files. The accepted
count remains 49/87 while Chrome connector/browser validation is pending. See
`docs/operations/frontend-review.md` for exact browser/runtime checkpoint. Resume
war desktop/phone filters, eligible sorting, hash tabs, losses/members/intel,
faction periods/systems/map, hover/pan/zoom/reset/navigation and failure/retry.

**Roam composition WIP, 2026-10-08:** One further module is typed, with a temporary
declaration-only bridge for the legacy roam-report parent. Strict checks and five
helper tests pass; 33 JS files remain. Accepted count stays 49/87. Continue with
roam-report and remove the bridge, then finish family DOM/browser validation.

**Roam family TypeScript WIP, 2026-10-08:** Both Roam sources and the scoped form
controller are now strict TypeScript; the temporary composition bridge is removed.
447 tests in 87 files, strict checks, relevant Go checks and production builds pass.
Real create/update checks used only the retained disposable local fixture.
32 JavaScript modules remain; six converted War/Roam modules await desktop/phone
validation, so the accepted checkpoint stays 49/87. Chrome still times out opening
review tabs. Continue independent conversions while that browser gate is pending.

**Scans/archive/status TypeScript WIP, 2026-10-08:** D-Scan/Local Scan, the legacy
archive and live status dashboard are now typed/scoped. Strict checks, 490 tests
in 90 files, relevant Go checks, production build and artifact checks pass.
29 JavaScript modules remain; nine typed modules await live desktop/phone checks,
so accepted progress remains 49/87. Real scan analysis/save/reload checks used
only the disposable local fixture. The latest fixture facade is on 4451;
production preview 4205 is down because the Tailscale dependencies are unreachable.
Chrome focus emulation also still times out. Neither blocker prevents further
source conversion and local tests; the goal stays active.

**Docs/MCP TypeScript WIP, 2026-10-08:** Both documentation modules now use
scoped controllers. The API explorer uses one checked shared OpenAPI adapter,
lazy endpoint fields, owned body/stream reads, entity lookup and URL/curl helpers.
MCP retains registered tools, filtering/copy controls and cached lazy schemas.
The full suite passed 516 tests in 92 files; final family regressions, strict
checks, relevant Go checks, production builds and artifact checks pass.
27 JavaScript modules remain; eleven typed modules await live desktop/phone
checks, so accepted progress stays 49/87. Original docs classes/parameter labels
were checked against the retained pre-conversion renderer. Latest local facade
is on 4451; production preview 4205 still awaits Tailscale recovery. Goal active.

**Graph/rankings/statistics TypeScript WIP, 2026-10-08:** Four more original
modules are strict TypeScript with scoped actions, typed graph/search boundaries,
owned requests, retained caches and polling/concurrency cleanup. 554 tests in 95
files, strict checks, relevant Go checks, production builds and artifact checks
pass. Graph OpenAPI descriptions now match the existing nested affiliations,
cooperation scores and character totals; serializer/contract tests pass and the
TypeScript client is regenerated. 23 JS modules remain; fifteen typed modules
await browser/phone acceptance, so accepted progress stays 49/87. Goal active.


**Market TypeScript WIP, 2026-10-08:** The original market module is now strict
TypeScript with scoped browser-tree, item-market and search controllers. Reusable
Go fragments declare their own scopes; entity tab navigation retains ownership of
item Back history. Pending filters combine, stale bodies cannot replace a newer
selection, sorting survives refresh, and failures retain data with retry. Trees
cache exact nodes and cancel hidden reads; ship/item search is debounced, abortable
and deduplicated by type ID. Sixteen new DOM/boundary regressions use captured Go
HTML. Strict TS, all 570 tests in 96 files, relevant Go checks, asset checks and
production assets/binaries pass. 22 JS modules remain; sixteen typed modules await
browser/phone acceptance. Accepted checkpoint remains 49/87. Next: universe/item
navigation, fitting-filter fragments and kill paging. Goal active.


**Universe/item TypeScript WIP, 2026-10-08:** The original universe module now
uses scoped item/group navigation and item-fitting controllers. Existing hero,
tab, killlist, fitting-family/filter/histogram markup is retained. Tab metadata,
Back, preferred defaults and paging cursors remain supported. Child fitting reads
clean up on tab exit; canceled responses cannot overwrite a newer route or drafts.
Supporting panels retain exact nodes during refresh. Image dialogs close on scope
exit. Shorthand parsing/repair sorting now share helpers with fitting search.
Twenty-four new DOM/lifecycle tests use captured Go HTML, plus a market Back-route
regression. Strict TS, all 595 tests in 97 files, relevant Go checks, production
assets/binaries and source/artifact checks pass. 66/87 original modules are now
typed; 21 remain JS and seventeen typed modules await live browser/phone acceptance.
Accepted checkpoint remains 49/87. Next: system/location interactions. Goal active.


**Location/blog TypeScript checkpoint, 2026-10-08:** System/region/constellation
interactions and blog pagination/date localization are now strict TypeScript.
Location uses the shared fragment loader with root replacement; item/group and
location kill paging now share cursor history/validation. Existing HTML/CSS and
API payloads remain intact. Location race/identity/retry/Back/paging/hover tests
pass; returning to an already displayed tab/category cancels its competing read.
Blog scopes own reads and localize newly inserted cards automatically. All 632
tests in 99 files, strict TS, relevant Go checks, source/artifact checks and
production assets/binaries pass. 68/87 original sources are typed; 19 remain JS.
Chrome reconnected as browser 2. Blog desktop/390px comparisons against the retained
JavaScript renderer pass: identical 45-card and 22-tagged-card lists/dimensions,
localized short/long dates, terminal paging, failure/retry, rich post layout, Back
and no phone overflow. Blog is accepted, bringing accepted progress to 50/87;
eighteen typed modules await browser acceptance. Goal active. Continue browser
backlog where local fixtures permit, and source conversion with search.js next.


**General search TypeScript checkpoint, 2026-10-08:** The original search source
now uses scoped actions, owned debounce/reads and generated-API-backed projections.
Card DOM/classes, routes/images, affiliations, suggestions and all eleven filters
are retained. Filter changes reuse card/image nodes. Failed searches retain usable
results with Retry; Clear preserves unrelated URL/history state. Market and picker
search now share the validated EVE ID parser. Fifteen new regressions and the real
Go serializer contract pass; all 647 tests in 100 files, strict TS, relevant Go,
source/artifact checks and production assets/binaries pass. Chrome desktop/390px
checks cover all types/filters, matching data/routes/images/dimensions, loading,
error/retry, clear/focus, empty results and Back/Forward, without phone overflow or
console errors. Search is accepted: 51/87 accepted, 69/87 sources typed, 18 JS remain,
eighteen typed modules await browser acceptance. A reusable anonymous read-only
reference proxy serves the retained renderer/assets on 4452 with shared local GET
fixtures, avoiding a restart of the retained 4450 backend. Goal active. Continue
with shared kill submission/post code and the outstanding browser backlog.


**Kill submission TypeScript checkpoint, 2026-10-08:** Shared submission and /post
now use scoped Stimulus controllers, generated API projections and owned requests,
timers and reconnecting relay transport. Pending and ambiguous POST IDs persist
before dispatch and reconcile with reads after navigation, without replaying writes.
Ready notices retain DOM/image nodes and keyboard focus until expiry. Original
markup, counters, links, clipboard/global paste and toast content remain intact;
API failures use the shared descriptive error parser. Twenty-one new regressions,
all 668 tests in 101 files, strict TS, Go frontend/facade checks, source/artifact
checks and production assets/binary pass. Chrome desktop/390px comparisons cover
forms/results, delayed controls, rejected drafts, errors, WS arrivals/reconnect,
poll fallback, full-page navigation and Back. Phone dimensions match, with no
horizontal overflow or console errors. In-memory submission fixtures intercept
POST before both proxies; no database jobs are queued. 53/87 accepted, 71/87 sources
typed, 16 JS remain; eighteen previously typed modules still await browser checks.
Goal active. Next: coalition/entity/home/kill/fitting/map sources and browser backlog.


**Coalition TypeScript checkpoint, 2026-10-08:** Directory, alliance picker and
create/edit forms now use a Go-declared Stimulus scope, generated API projections,
shared search ID guards and owned requests/timers. Editors retain newer drafts and
loaded images during delayed saves/HTML reads; interrupted edits reconcile with GET
before another write. Reload validates coalition identity/revision. Back reapplies
browser-restored filters, fixing an issue also present in the original JS. Nineteen
new DOM regressions and all 687 tests in 102 files pass, along with strict TS, Go,
source/artifact checks and production assets/binary. Desktop/390px checks cover
populated cards/statistics/members/history, filter/empty states, activity loading/
failure/retry, creator search/selection/creation, delayed edit/newer drafts,
membership reset and Back. CSS is byte-identical to the source archive; no mobile
overflow or JS console errors were observed. Only a named disposable coalition and
its expired fixture owner session changed locally; production was untouched.
54/87 accepted, 72/87 sources typed, 15 JS remain; eighteen typed browser checks
remain pending. Next: entity/home/kill detail, fitting/map and browser backlog.
Goal active; full audit and allowJs/legacy removal remain required.


**Entity TypeScript WIP, 2026-10-08:** The original entity module is fully ported
into persistent navigation/widget scopes and a replaceable tab-content scope.
Shared fragments, request ownership, relay transport, paging, search IDs, number
formatting and default-tab preferences are reused. Existing tabs, portraits,
valuable categories, ship classes, dashboards/intel, faction/militia, top periods,
achievements, members/corporation/conflict controls and live kills are retained.
Rapid member filters combine pending choices; interrupted tab work cancels/resumes
within its scope. Kill arrivals during a read retain their refresh counter. Empty
militia success follows the real Go contract. 27 new Go-HTML-backed DOM/model tests
pass; the full suite passed 714 tests in 103 files. Strict TS, Go frontend/facade,
source/artifact checks and production assets pass. 73/87 sources typed, 14 JS remain.
Accepted progress stays 54/87; nineteen typed modules await browser acceptance,
including entity. Next: entity desktop/phone checks using actual populated fixtures,
then home/kill/fitting/map sources and the remaining browser/final-audit backlog.


**Entity browser acceptance, 2026-10-08:** Entity now passes desktop/390px source
comparisons for characters, corporations, alliances and factions. The read-only
local preview uses the existing Go HomepageReader and original archived JS/helpers.
Populated cards, kill rows, intel, achievements, membership/corporation filters,
top periods, battle/war/campaign tabs, portraits, loading/errors/retry, limits,
WS/reconnect/pending arrival counts and Back match or improve the source behavior.
Phone widths and row sizes match with no overflow; CSS is byte-identical. The seed
lacks NPC names, so only two clearly named faction identities are fixture metadata;
warzone/militia/combat still use real read-only SQL. Pagination with large opaque
cursors and legacy portrait switching remain covered by Go-HTML-backed DOM tests;
the live local dataset has too few rows for a second page. 55/87 accepted, 73/87
typed, 14 JS remain and eighteen typed families await browser acceptance. Strict
TS, focused 30 checks and Go preview/reference guards pass; prior full suite 714
passed. No database rows, production services or user listeners changed. Goal
active. Next: home/kill detail, fitting/map and outstanding browser/final audit.


**Public kill-feed/detail TypeScript checkpoint, 2026-10-08:** The original home
and kill-detail modules now use Go-declared Stimulus scopes, owned reads/timers,
shared relay transport, generated API projections and the existing kill-row
renderer. Quick filters, picker drafts, HTML paging/cursors, valuable switches,
phone tabs and independent lazy top lists are preserved. Kill context, attacker
batches, container-aware item sorting, wheel tooltips, image fallback, matchup and
compressed fitting handoff retain their source behavior. Empty feeds accept their
first live row; arrivals during pending reads retain their refresh counter.
Thirty-five focused regressions and all 749 tests in 106 files pass, with strict
TS, relevant Go checks and production assets. Desktop/390px comparisons cover
populated filtered feeds/valuable cards, search IDs, limits, lazy tabs, fitting
handoff, loading/error/retry, result context and live arrivals/reconnect. Phone
rows match at 345x68; kill wheels match at 363px, without overflow. Source CSS is
unchanged. The disposable homepage rollups are empty; populated top cards, large
attacker/item/matchup cases and large opaque cursor paging use actual Go-HTML DOM
fixtures, rather than being claimed as live database/browser scenarios. Browser
Back currently resets homepage quick filters in both source and new code; include
retaining this state in the final navigation audit. 57/87 accepted, 75/87 typed,
12 JS remain; eighteen other typed families await browser acceptance. No database
rows/jobs or user listeners changed. Goal active. Next: ten fitting and two map
sources, browser backlog, final mobile/theme/navigation/live/performance/build
audit, then allowJs/legacy removal at zero handwritten JS.


**Fitting TypeScript WIP, 2026-10-08:** Eight more original sources are typed:
fit-capacity, fit-resources, fit-stats, fit-variants, fit-wheel, fit-search-state,
fits-catalog and fit-detail. SDE metadata maps now have concrete field/attribute/
effect types instead of any. Pure renderers preserve source output, with original
markup captures tested against real pinned dogma 13.6.0 results. Catalogue sections
reuse shared fragment ownership and retain cards on errors. Saved ratings/comments
use Go-declared scopes; ambiguous/interrupted rating writes reconcile through GET,
and comments initialize only on demand. Thirty-two focused checks and all 777
tests in 108 files pass, plus strict TS and relevant Go checks. Browser acceptance
for this fitting family waits for the editor/search port: 83/87 sources typed, four
JS remain, 57/87 accepted and twenty-six typed modules pending acceptance. Next:
fits-search/editor, both map modules, browser backlog and full final audit. Goal
active; allowJs/legacy inventory still required until zero handwritten JS.


**Fitting search TypeScript WIP, 2026-10-08:** fits-search is now a Go-declared
Stimulus controller with independent owned results/profile/availability/item
reads, shared search/filter models and validated generated API projections.
Requirement and statistic controls retain nodes/images/focus during changes.
Offset paging appends to existing results; failures retain usable cards and Retry
repeats the failed offset. Popovers, keyboard choices, role/item/hull selection,
statistic/NPC/capacitor fields, URL/Back and cached-return cancellation are retained.
Seventeen actual-Go-HTML-backed regressions pass, along with strict TS and all
794 tests in 109 files. 84/87 sources typed; three JS remain (fit-editor, map-live,
map-page). Acceptance stays 57/87; twenty-seven typed modules await browser checks
including fitting search. Finish the editor and both map sources, validate fitting
and remaining browser families, then complete the full audit and remove allowJs/
legacy inventory at zero. Goal active; no database rows/jobs or production writes.

**Editor boundary WIP, 2026-10-08:** The live editor now reuses checked TypeScript
helpers for its slot constants, engine input, v3 codec, workspace preferences,
saved API rows, observed families and prices. Duplicate encode/decode/input logic
was removed from the original module. Share URLs with fit after other query
parameters now import correctly; decompression is bounded. Valid empty Go slices
normalize to empty lists, retaining empty fitting/catalogue behavior. Thirteen
new contract regressions pass; full suite 807 tests in 110 files and strict TS/Go
checks pass. The editor's interaction/controller conversion remains unfinished:
84/87 sources typed, still three original JS modules. Do not count the editor as
fully converted or browser accepted. Next: port the existing remaining editor
rendering/interactions into owned component lifecycle, retaining fullscreen scale,
wheel/table, drag/drop, history, clipboard and save behavior; then both maps and
the browser/final audit backlog. Goal active.

**Editor and live-map source checkpoint, 2026-10-08:** fit-editor and map-live
are now checked TypeScript. The editor retains its original rendering/interactions
under a Go-declared Stimulus owner, including cold clipboard import, fullscreen
workspace retention, wheel/table, slot drops, history and observed-fit projection.
Requests, observers, clocks and late WASM/gzip/clipboard results are scoped to the
current page. Saves are single flight, retain newer drafts and never auto-replay
interrupted writes. Twelve actual-Go-HTML/pinned-engine editor regressions pass.
Map live/AIID uses the shared relay; eight real-DOM tests cover retained scope
updates, both alarm bands, freshness, visibility/backoff, row bounds, bootstrap
races, image fallback, clocks and teardown/recreation. All 826 tests in 111 files,
strict TS and relevant Go checks pass. Source count: **86/87**, only map-page.js
remains. Browser acceptance stays **57/87**; twenty-nine typed modules still need
full acceptance. Partial editor comparisons at fixed 1440x900 and 390x844 match
source wheel/statistics/resource markup after asset URL normalization; mobile
wheel is square at 374px and the page does not overflow. Table delete/Cmd+Z and
save-dialog opening work in Chrome. Observed-fit data is unavailable in the
isolated preview, so populated observed fits and the rest of the fitting family
remain open. Next: map-page conversion, fitting/browser backlog, full audit and
allowJs/legacy removal at zero. Goal active; no production/database writes.

**Paused checkpoint, 2026-10-09:** All **87/87 original modules are strict
TypeScript**. map-page now uses Go-declared Stimulus actions, shared checked map
state/API projections, owned reads/debounce/renderer/feed cleanup and guarded
navigation/hover callbacks. Twelve new Go-DOM controller/boundary tests pass.
`allowJs`, `checkJs`, the JS include and `browser-legacy.json` are removed. The
source audit checks every authored src TypeScript file for explicit any types and
suppression comments, rejects JS, and verifies strict compiler settings. Removing
allowJs exposed the Rust background module's missing declaration; a specific
application ABI contract matches the pinned compatibility adapter's structs,
without broad JS-module declarations or any casts. All **838 tests in 112 files**,
strict TS, rebuilt frontend/facade Go checks and the complete production Docker
image build pass. Docker runtime smoke also verifies the Go/Rust binaries and
absence of Node/Bun. Browser acceptance remains **57/87**; thirty typed modules
still need completed acceptance and the final audit remains open.

Map validation is partial: at 1440x900 populated geography has identical rendered
DOM/text and geometry to archived source; base/activity/routes/systems/labels and
search match. Top-level canvas click reaches The Forge region with retained
query controls; Back restores the map state. The iframe's blocked handoff was its
intentional frame policy, not a product failure. Populated sovereignty summaries
and two claimed systems match; finish sovereignty interactions, live/AIID relay
and mobile checks tomorrow before accepting either map module. Use named local
fixture responses, not cold/production performance claims. Screenshot evidence:
/tmp/shrike-map-review/geography-current.png and geography-source.png.

Owned preview: /tmp/shrike-ts-maps-browser-facade on 4451; archived source on 4452
with -reference-maps and -reference-fittings; retained backend 4450 untouched.
User ports 4000/4200 remain unchanged. Local overrides are cleared, comparison
tabs closed. The local Docker image shrike:typescript-audit-20261008 is built;
no deployment or production writes occurred. Resume maps, then populated fitting
and the existing browser backlog, full site/mobile/theme/navigation/loading/live/
performance audit, bundle reproducibility and remaining relevant Go checks.
Validation paused explicitly at the user's bedtime request. Do not mark complete.


**Resumed map validation, 2026-10-09:** Map-page and map-live are accepted against
archived original scripts with identical Go markup/CSS and named in-memory data.
Desktop 1440x900 and phone 390x844 geography, sovereignty, live and AIID layouts,
search, layers, watch add/remove, empty-area guidance, controls and query state
match. Live events insert once, survive reconnect and retain rows when AIID
anchors change; leaving live mode releases both comparison sockets. Delayed API
failures show an error while retaining the previous canvas/watch; a new request
recovers. Keyboard search results now remain open when focus moves into the menu;
three actual-Go-DOM regressions cover general, alliance and watch search. This
is an intentional usability improvement over the original blur timeout.

All **841 tests / 112 files**, strict TypeScript and `go test ./...` pass (the broad
Go run unsets TEST_DATABASE_URL, so database integration tests are skipped).
Acceptance is **59/87**, with 28 typed modules pending. Final full-site/mobile/
theme/navigation/live/performance/build audits remain open. Map screenshots are
in /tmp/shrike-map-review, including aiid-phone-current.png and desktop/live views.
Canvas gesture coverage remains in existing tests and the final interaction audit;
no automated native touch gesture was claimed here. No production performance
claims are derived from these small local fixtures.

Continue populated fitting comparison. The opt-in reference fitting mode now
supplies the real Go variants renderer with read-only item metadata and a clearly
named in-memory Rifter family when the disposable database has no families. No
rows are inserted. Goal active; retained backend 4450 and user ports untouched.


**Populated fitting validation, 2026-10-09:** The opt-in comparison facade now
routes actual Go observed-fit fragments through a guarded read-only UniverseReader.
The tiny seed's missing Rifter ship category and one named family are supplied
in memory only; existing families and read failures remain untouched. This caught
and fixed a genuine TypeScript boundary regression: Go emits charge_type_id=0 for
modules without charges. Zero/null/missing now normalize to null; invalid IDs still
fail. A real Go variants capture plus pinned-engine controller regression covers
preview, modification/reset and restoring the original custom fit. Desktop
observed panels, suggestions, diff values, wheel/statistics and custom restoration
match archived source. Phone wheel/table/statistics remain faithful; fullscreen
variants are hidden below 1200px by shared original CSS, so no phone observed-fit
interaction is claimed. Revisit that discoverability in the final mobile audit.

Catalogue desktop/phone checks cover populated flavor/doctrine/community/rated
cards, flavor modes, corporation selection and error/retry. Refresh errors retain
old cards as an intentional improvement. Search partial checks cover hull lookup,
Rifter results/editor links, 50k shorthand, capacitor flag, sort, role addition,
count/operator changes and 390px layouts with matching URL state and no overflow.
Complete search item requirements, distributions, paging, failures and Back, then
saved-fit/editor remaining interactions before accepting the fitting family.
Formal acceptance remains **59/87**; do not count partial checks as completion.

Latest full suite **843 tests / 112 files**, strict TS, relevant frontend/facade
Go checks and production browser build pass. Broad Go suite passed before the
local fitting-fixture additions; database integration was disabled for that run.
Assets: 118 entries / 104 chunks, 1,151,071 JS bytes; utility CSS 119,988 bytes.
Current owned preview: /tmp/shrike-ts-fitting-family-browser-facade, 4451/current,
4452/archived comparison, retained 4450 unchanged. The source archive now also
supplies fitting catalogue/search and their named original helpers. Current
in-memory override is /api/fits/search/availability; clear it when finished.
No production/database writes or user listener changes. Goal active.

Final audit follow-ups discovered while comparing: catalogue quick statistics
can display -1 because pg_class.reltuples is unknown in an unanalyzed fixture;
represent unknown estimates appropriately instead of a negative public count.
Review the original phone fullscreen hiding of statistics/observed fits. Neither
is claimed fixed or part of completed fitting acceptance yet.


**Current verification checkpoint, 2026-10-09:** The latest broad Go suite passes
after the fitting-fixture changes (`env -u TEST_DATABASE_URL go test ./...`;
database integration is skipped). A second production build reproduces the full
artifact manifest/output hashes, utility CSS and icon data. Full browser suite is
843/112, strict typecheck passes, whitespace checks pass. Phone search populated
cards and delayed results failure/retry now also match the original; old cards
remain during failure. /tmp/shrike-fit-search-response.json is the explicit local
presentation response for subsequent paging/profile checks. All override rules
are cleared, comparison tabs closed; owned preview remains on 4451/4452.
Screenshot of the corrected populated preview:
/tmp/shrike-fitting-review/observed-preview-charge-fixed.png. Formal acceptance
59/87; full fitting acceptance and the 18 older modules/final audit remain open.
Goal remains active. Next: fitting search items/distributions/paging/Back,
saved detail and editor remaining live interactions, then browser backlog.


**Fitting family accepted, 2026-10-09:** All ten original fitting modules now
have meaningful strict/unit/Go-DOM/pinned-engine coverage and live browser
acceptance, advancing progress to **69/87**. Search item requirements, histogram
range clicks, 24-row first pages, retained append failures, retry to the terminal
25th row, editor handoff/Back and catalogue filter Back match archived source.
Desktop and 390px search/catalogue/details have matching markup/text and no
page overflow. Explicit local presentation responses are not SQL-filter proofs;
the existing API tests cover filtering. Archive reference now also runs original
saved-detail/editor/comment helpers on the same Go markup without duplicate typed
scopes or leaking cookies to the source renderer.

A second real conversion regression was caught and fixed: saved-fit pages omit
shortcut help, but the new required-element helper rejected it. This one optional
node now uses a nullable query and guarded actions. The Go saved-page/pinned-engine
regression verifies startup, module/charge, three active/two stored drones, cargo
quantity and stats. Initialization errors now log the cause before the existing
visible failure message. Saved detail and lazy comment/reply content match both
implementations on desktop/phone. Real local rating PUT and DELETE show/clear the
rating; the archived page sees the updated aggregate. A new private fit was
created, updated and reloaded through the real disposable API; anonymous access
is denied. The uniquely named new row was removed afterward. The seeded public
fit's rating returns to unrated. No production data changed.

**User workflow constraint:** The user works on another monitor. Native app mouse
and keyboard input caused interference; do not use native desktop automation.
Use the browser connector's scoped DOM APIs. Native drag did not produce a drop in
either implementation; the user personally confirmed catalogue→slot,
slot→different slot and catalogue→ship image work in the converted 4451 editor.
Their current fitting preview tab is retained as a deliverable. Other comparison
and save-test tabs are closed, and the fixture account is anonymous again.

All **844 tests / 112 files**, strict TypeScript, production assets and broad Go
checks pass (TEST_DATABASE_URL unset for broad Go; DB integration skipped).
Current assets: 118 entries / 104 chunks, 1,151,143 JS bytes; CSS 119,988 bytes.
Owned preview: /tmp/shrike-ts-fitting-saved-fix-browser-facade, PID 30361/session
77793, 4451/current and 4452/reference. Retained 4450 and user listeners remain
unchanged. All local overrides are clear. User port 4200 now answers /health and
renders populated live public data; use it as a read-only upstream when preparing
the remaining public page comparisons instead of unavailable 4205. Do not restart
or reconfigure the user's process. Source screenshots include
/tmp/shrike-fitting-review/saved-fit-phone-fixed.png and private-save-reload-current.png.

Remaining 18 original modules: wars, war-detail, faction-war, faction-war-map,
roam-composition, roam-report, scans, legacy, status, docs, mcp, graph, graph-render,
rankings, statistics, market, universe and system. Finish their browser acceptance,
then the full site/mobile/theme/navigation/focus/loading/live/performance audit,
current Docker rebuild/runtime check, integration/dev reload and reproducibility
checks as applicable. Goal active; no completion or pause is claimed.


**War family browser acceptance, 2026-10-09:** Four original modules are now
accepted, bringing acceptance to **73/87**; every source remains strict TypeScript.
The owned public facade on 4461/current and 4462/archive renders identical current
Go markup with anonymous GET data from the user's healthy 4200 server. It strips
credentials and rejects mutations before forwarding; the private 4450/4451/4452
fixtures and user listeners remain unchanged. The helper now accepts a separate
backend port and serves its viewport wrapper locally on the reference port.

Directory checks cover active/finished/upcoming, all filters, Clear, sorting,
finished page two, and 500-row eligible corporation/alliance sorting. Source name
escaping renders apostrophes as undefined; current shared escaping fixes that.
An eight-second faction-only delay proves the main directory/filter controls remain
usable while cards load. Failed refreshes retain current rows and Retry recovers.
Declared war #763266 has matching stats, both sides, page two, member side/entity/
sort controls, populated intel and Back. Member refresh failure retains 28 rows,
clears busy state, and recovers. Both faction matchups have matching heroes,
leaderboard periods, side filters, 100-row cursor pages, 500 members, intel, system
sorting and maps (90/70 nodes with identical SVG bounds). A faction member failure
leaves the map intact and reopening recovers. Keyboard focus shows/hides cards;
Enter and original click hand off to the same system URL. System destination page
rendering remains in the separate system audit: this facade has no Universe reader.

390px directory/member/map checks match source and fit the 379px document width.
Browser map wheel/pan/reset input was deliberately avoided under the user's desktop
input constraint; all 22 existing war controller tests pass, including those paths,
cleanup, stale responses, failure/retry and empty maps. Frontend/facade Go tests pass.
All fault controls are clear. Screenshots: /tmp/shrike-war-review/
war-members-phone-current.png and faction-map-phone-current.png.

Remaining fourteen: roam-composition, roam-report, scans, legacy, status, docs,
mcp, graph, graph-render, rankings, statistics, market, universe and system.
Full final audit and current Docker/dev/integration/reproducibility remain open.
Goal stays active; no deployment or production mutation ran.


**Roam Report browser acceptance, 2026-10-09:** Both original report/composition
modules are accepted: **75/87**, all sources remain strict TypeScript. Report
a462a5e9c726c1e7e210d9b8452bc3a9 was compared without changing it. Overview,
intel, composition pilot/ship/group aggregates, sorting, search/class/loss filters,
empty results, kills/losses/all and a settled 10-row page size match archive.
Current kill-role/page-size query parameters also survive Back; archived code loses
that state because it never stores it in the URL. The report has three killmails;
longer cursor pagination remains covered by controller/API tests rather than a
claim of browser paging beyond those three.

Combat maps now read identical public anonymous geography on both facade ports;
the initial 5484-vs-2-system difference was a comparison harness routing mismatch,
not a migration defect. Both show matching scopes, region drill-down, keyboard
system-marker activation and engagement filters. Tab failures preserve the map
and it stays interactive; Retry loads intel and removes the old canvases. Full
map drag/wheel validation uses already accepted map coverage and regression tests;
no native desktop input is used. 390px composition/systems/edit screens fit 379px;
edit validation/Cancel match and the calendar stays within the phone width.

The creator's actual calendar, duplicate pilots and 72-hour limit were exercised.
A new disposable report was created and updated via the real local API. Five-second
PUT delay proves newer textarea edits survive and get a visible saved/newer-draft
notice. A normal save refreshes the page and hides the editor; reload and archived
rendering agree with stored data. The exact new ID, dates, input count, creation date
and unique unresolved-name marker guarded cleanup of that one row. The original
report is JSON-identical afterward. All local faults are clear; no production
report or user listener changed. The temporary facade was 4465/current, 4466/source,
/tmp/shrike-ts-roam-map-equal-browser-facade, session 93571. Its archive wrapper
preserves row templates, removes duplicate typed scopes, and isolates old imports.

All **844 tests / 112 files**, strict TypeScript and broad Go checks pass after
these harness changes. Broad Go unsets TEST_DATABASE_URL and skips DB integration.
The two roam test files add 23 targeted cases; war tests have 22. Artifacts:
/tmp/shrike-roam-review/composition-phone-current-cropped.png,
editor-calendar-phone-current.png and created-saved-report-current.png.

Remaining twelve original modules: scans, legacy, status, docs, mcp, graph,
graph-render, rankings, statistics, market, universe and system. Next resume those
browser comparisons, followed by full site/mobile/theme/navigation/focus/loading/
live/performance and current Docker/dev/integration/reproducibility checks. Goal
stays active. No deploy, release or commit is part of this validation.

Cleanup: both owned war/roam facade processes have terminated and comparison
tabs are closed. The user-tested fitting tab is retained; its private 4451/4452
facade and the retained 4450 API are unchanged. Do not expect 4461/4462/4465/4466
to be listening at the next checkpoint; relaunch the named archived comparison
mode on unused ports as needed. No test report from this turn remains.


**Scans/docs/MCP acceptance, 2026-10-09:** Three original modules are accepted,
bringing browser acceptance to **78/87**; all 87 remain strict TypeScript. The
separate private tools facade runs current Go markup against retained 4450 on
4465/current and archived controllers on 4466/source. Public reads remain
anonymous. The source allows only scan analysis/save POSTs to the private fixture
and pure JSON-to-HTML scan fragments; other writes are blocked. Public-origin
aliases on the same port cannot enable this opt-in mode. Tests cover scope removal,
credential stripping and that narrow exception. The source scan fragment POST
initially returned 405 from the comparison wrapper; it is now routed to the local
pure renderer. Do not call that initial comparison failure a product defect.

D-Scan and Local Scan populated results match, including unknown objects, threat
counts, unresolved pilots and expandable groups. Keyboard paste is browser-scoped.
Current analysis failures retain the previous result while the source clears it;
pasting again recovers. Both tools saved and loaded through the real disposable
owner API. Phone result rendering matches within the 379px document at 390px.
The source deliberately remains anonymous: compare result components independently
of current owner's Save controls. Unique inputs produced two new scan hashes;
cleanup checked that neither existed before, then deleted only those exact hashes,
owner 4242, types, creation date and unique input markers. The scan hash inventory
is identical before/after. No production scan or other row changed.

API docs match the archived 528-operation listing, filtering, empty results, lazy
forms, required fields, invalid JSON, real GET killmail and safe POST scan analysis.
Name lookup resolves the same fixture entity IDs. OpenAPI load failure hides the
console and Retry restores all operations. Phone requests and response JSON match
and fit 379px. Two improvements discovered during validation are implemented:
Try it uses cache:no-store so a repeated request actually reaches the API (Loading
and a delayed 503, then recovery to 200, proved in the browser). The model merges
path-level parameters, honors operation overrides and supplies missing required
URL-placeholder inputs for compatibility routes with incomplete metadata. The
/character/:id form now resolves names and returns a real fixture profile using
70131. New coverage also checks every URL placeholder in the full generated spec.
This enhances the console without altering API contracts or rewriting the archive.

MCP's 45 tools, filters/empty state, lazy output schemas and phone layout match.
A failed schema load clears busy state, preserves other loaded schemas and Retry
recovers; loaded output remains when filters change. Clipboard feedback and
cancellation paths remain covered by existing controller tests. No external MCP
connection or real financial action ran. Console logs are clean after settled
browser checks. Screenshot proof: /tmp/shrike-tools-review/
docs-request-phone-current.png, compatibility-id-request-current.png,
mcp-schema-phone-current.png, dscan-phone-current.png and localscan-phone-current.png.

All **845 browser/WASM tests in 112 files**, strict TS, frontend/facade and broad
Go checks pass. Broad Go unsets TEST_DATABASE_URL, so DB integration is skipped.
Production assets: 118 entries, 104 chunks, 1,151,455 JS bytes; CSS 119,988.
Latest owned comparison binary: /tmp/shrike-ts-tools-docs-parameters-browser-facade,
session 28361; retained 4450 and private fitting 4451/4452/user 4200 untouched.
Remaining nine: legacy, status, graph, graph-render, rankings, statistics, market,
universe and system. Full site/mobile/theme/navigation/focus/loading/live,
Docker/dev/integration and final build/runtime audits still remain. Goal active.

Final checkpoint cleanup: production manifest, utility CSS and icon data reproduce
exactly, and every physical output matches its manifest byte count/SHA256. The
owned tools facade terminated; 4465/4466 are no longer serving. Faults are clear,
the fixture account is anonymous, comparison tabs are closed, and the user-tested
fitting preview remains open. The scan inventory is exactly restored. Resume the
nine remaining families; no pause or completion is claimed.


**Statistics/rankings/graph browser acceptance, 2026-10-09:** Four original
modules are accepted: **82/87**; all 87 sources remain strict TypeScript. The
anonymous read-only analysis facade uses real public GET data from user 4200,
current Go markup on 4465 and archived JS on 4466. A reusable remote rankings
adapter uses the existing stats/rankings contract. Archived statistics/rankings/
graph/render/formatter imports run against identical current Go markup; typed
parent/search scopes are removed to avoid duplicate handlers. No user listener
or production row changed.

All statistics sections match populated source: activity, ISK, PvP, locations,
ships, largest entities, security, growth, newest and achievements. Seven/fourteen-
day and hourly results, period URLs, independent population periods, section Back,
phone activity/largest rows and phone controls pass. Some real security/growth
reads outlasted the browser locator observation deadline, then completed on the
same live requests; no process was restarted for those waits. One injected card
failure clears its busy flag and does not block adjacent tables. A dedicated
ninety-day Retry check completes all three cards with the retry button removed.
This is functional evidence, not a warm-cache claim about cold production latency.

Rankings match all six entity categories and all eight windows. The hourly view
actually polled while left open: a query-specific local 503 appeared without any
control click, with the existing table retained and busy cleared. Retry recovers;
source discards its results on error and reload returns the same table. Populated
corporation and interactive character/weekly views match at 390px, with 379px
client/scroll width.

All nine graph tools have real public/Memgraph results and source comparisons:
hot zones (30 systems), allies/enemies for alliance and corporation, hunting
grounds, military census, spy checks, migration, detected coalitions (five, including
expanded members), alliance/character rivalries (50 rows), and Path Finder.
Unrelated pilots produce matching six-hop empty results. Two attackers from
observed killmail 139015012, anthrobeau and Olan Draselac, produce a matching
one-hop/159-flight chain. Real searches choose exact names/tickers; menus are
reopened when focus hides them instead of clicking hidden suggestions. Graph
loading/busy/503/retry recover 30 rows; keyboard ArrowRight selects/focuses census.
Phone hot zones, census, chain, expanded coalitions and rivalries match and fit
379px. Images/code layouts are retained from the existing Go/CSS source. No native
desktop input ran. Memgraph ingestion, query performance and the usefulness of
inferred intelligence are a separate future review, explicitly deferred by the
user; they are not being redesigned as part of this migration.

All 41 targeted tests in four statistics/rankings/graph files pass; broad Go passes
with TEST_DATABASE_URL unset (integration skipped). The latest full frontend suite
remains 845/112 and strict TypeScript/production reproducibility passed at the
preceding checkpoint; authored browser source did not change in this turn.
Artifacts: /tmp/shrike-analysis-review/statistics-phone-current.png,
rankings-phone-current.png, graph-hotzones-phone-current.png and
graph-path-phone-current.png. Owned helper: /tmp/shrike-ts-analysis-browser-facade,
session 92090, 4465/4466. Fault controls are cleared. No deploy, release or commit.

Remaining five original modules: legacy, status, market, universe and system.
Complete those browser checks, then the full site/mobile/theme/navigation/focus/
loading/live/performance and relevant Docker/dev/integration/final artifact/runtime
passes. Goal remains active; full completion is not claimed.

Cleanup: the owned analysis facade terminated, fault rules are clear, and all
comparison tabs are closed. User/retained listeners remain unchanged. The former
fitting preview is no longer present in agent or user tab inventories; do not
claim it is still open or recreate its unsaved state without a request. Browser
checks remain scoped to the connector. Goal remains active with five modules and
the full final audit outstanding.


**Status browser acceptance, 2026-10-09:** Acceptance is **83/87**; all 87
original modules remain strict TypeScript. The real read-only 4200 `/ws/status`
stream delivered a populated dashboard with 24 ingest hours, five database
history polylines and no console errors. A loopback-only, in-memory status relay
then replayed the captured actual Go DTO to current and archived status scripts
on identical Go markup. Populated panel text and graph points match; initial
loading, subsequent frames, click/keyboard tooltips, disconnect with retained
panels, automatic reconnect and navigation away/Back pass. The relay's accepted
connection count increased from four to eight across the controlled outage,
with four live connections restored. No upstream service was disconnected.

At 390px both pages fit 379px. The original tooltip overflowed the right chart
edge; the typed controller now clamps its measured width at both edges, and CSS
caps its width on narrow panels. Browser bounds verify left/right alignment,
with a meaningful regression for both edges and a 150px panel. Fourteen targeted
status tests, **846 frontend tests in 112 files**, strict TypeScript and broad
`env -u TEST_DATABASE_URL go test ./...` pass (database integrations skipped).
Production assets build: 118 entries, 104 chunks, 1,151,510 JS bytes and 119,988
utility CSS bytes. Artifacts: /tmp/shrike-status-review/status-phone-current.png,
status-phone-source.png and status-phone-tooltip-fixed.png. The user-owned 4200
listener was not restarted; the tooltip fix was verified in a freshly built
owned comparison facade. Remaining original modules: legacy, market, universe
and system, followed by the full final audit and Docker/dev/integration checks.
Memgraph ingestion/query/intelligence review remains deferred. Goal is active.

Cleanup: owned status comparison facade terminated; all comparison tabs/sockets
are closed, and user/retained listeners remain unchanged. No test rows, deployment,
release or commit were created. Continue with the four remaining browser modules.


**Legacy archive browser acceptance, 2026-10-09:** Acceptance is **84/87**;
all original browser sources remain strict TypeScript. A new anonymous public
GET adapter lets current and archived legacy controllers use identical Go markup
and real historical API data without a database/session mutation. The archive
contains 436,063 killmails; initial 50 rows and five top boxes match. Ten-row
cursor Next/Previous, all six sorts, capitals/battleships/pods/Jita presets,
2003/2007 date presets and all five yearly top lists plus All match. The earliest
year genuinely has three filtered killmails and four nonempty top boxes; empty
boxes are not invented. Every autocomplete field (victim, attacker, corporation,
alliance, system) has matching real suggestions, selection and populated filters.
A nonexistent ship produces matching empty results and Clear recovers.

Local kill/top 503s retain their previous results, clear busy flags and leave the
other panel usable. Kill Retry restores real Jita rows; top Retry restores the
selected 2004 lists. Both recoveries remove errors. At 390px both pages fit 379px,
with matching autocomplete, fifty filtered rows, mobile Top/Kills visibility,
2003 lists and Back retaining the Jita filter/results. Visible ship-link clicking
and keyboard Enter on the overlay both open real legacy kill 6006759. Clicking
the overlay's geometrical center lands on a nested character link, so that action
was not counted as kill navigation; the visible ship link and keyboard target
provide direct evidence. No native desktop input ran.

Thirteen legacy DOM tests and the full comparison-harness suite pass. The last
full suite remains 846/112 with strict TS and broad Go passing at the preceding
status checkpoint; authored browser source/assets did not change here. Evidence:
/tmp/shrike-legacy-review/legacy-phone-top-current.png,
legacy-phone-top-source.png and legacy-phone-kills-current.png. Owned facade:
/tmp/shrike-ts-legacy-browser-facade, 4465/4466. Remaining original modules:
market, universe and system, followed by the full final audit. No deployment,
release, commit or production mutation. Goal remains active.

Cleanup: legacy facade is terminal, local faults are clear, all comparison tabs
are closed and user/retained listeners are unchanged. Resume market/universe/system
checks; the goal remains active.


**Market browser acceptance, 2026-10-09:** Acceptance is **85/87**; all original
sources remain strict TypeScript. The anonymous universe comparison adapter shares
existing public DTO/query semantics for item/group/location/market/fitting reads
and real location battle queries. The source facade runs archived market,
universe and system scripts together on identical Go markup, removing every
corresponding typed scope to avoid duplicate listeners. Both use user 4200 only
for anonymous public GETs. No database is created or modified.

Market root/category navigation, keyboard lazy expansion, Frigates/Minmatar's six
priced item cards and real Rifter/Probe search results match. Keyboard selection
opens Rifter's standalone market. Its initial 100 sellers/76 buyers match, as do
all four order sorts in both directions in both books. The Forge (64 sellers,
24 buyers), Heimatar (37 sellers), Pochven (two sellers) and All Regions match.
Thirty/ninety-day history text and populated SVG geometry match. Every security
toggle, last-security protection and filter URL update match. A delayed local
market 503 exposes busy state while retaining 100 orders; Retry recovers real
ninety-day data and removes the error. Category failures leave orders usable;
close/reopen recovers the same Drones tree. Search failure retains Probe input,
clears busy and retries into real suggestions. No production writes ran.

At 390px both phone versions fit 379px, including sidebar toggles, real search
suggestions, region/period controls and filtered books. Browser measurements
found a pre-existing CSS grid minimum-content bug: each panel child was 764px
wide and the period controls began at x692, clipped inside a 361px panel. The
shared market body now uses a zero-minimum grid track and wrapping title row;
children are 333px wide, both period buttons are visible, and overflow remains
inside chart/table scroll containers. Actual phone clicks change periods. At
309px the panel, caption, periods and all security controls are contained.
Screenshots: /tmp/shrike-universe-review/market-phone-current.png and
market-phone-source.png preserve the old clipped state; market-phone-fixed-current.png
shows the fix. The same market component matches inside item/587's fragment tab,
including real region filtering and unchanged hero image/height.

Market/universe targeted tests pass (41 total), location tests pass (28), the
comparison harness passes and broad Go passes with TEST_DATABASE_URL unset
(integration skipped). Last full frontend suite remains 846/112; browser authored
TS/assets did not change in this turn. Market CSS is rebuilt into the owned
/tmp/shrike-ts-universe-market-mobile-browser-facade on 4465/4466; the user-owned
4200 preview is not restarted. Universe and system browser checks remain partial;
acceptance is not claimed for those two. Continue their item/group/fitting and
location interactions, then the full final audit. Goal remains active.


**Universe/item browser acceptance, 2026-10-09:** Acceptance is **86/87**; all
original sources remain strict TypeScript. Using identical Go markup and anonymous
public reads, Rifter dashboard/market/kills/fitting tabs match the original script.
The hero image URL and height stay unchanged across tab fragments; canonical
metadata updates. Fifty-row item kills, ten-row Next/Previous, group 25's 85
inventory types, fifty group losses and ten-row group paging match. Tritanium's
non-ship dashboard and its two actual involvement kills match, with no Fittings
tab invented. Image expansion requests the same 512px render and Escape removes
the dialog in both versions.

Real Rifter fitting data has twenty families, fifteen module groups and four
populated observed profiles across more than 9,000 losses. All eleven sorts,
expanded slot/charge/alliance details, Damage Control group selection, combined
5k–20k EHP ranges, Clear retaining sort, Guristas/Angel NPC profiles and exact
histogram bounds match. The combined range and first histogram bucket genuinely
produce matching empty results; Clear recovers twenty families. Some cold
filtered supporting reads outlasted the three-second locator observation window,
then completed on the same request. These are functional comparisons, not claims
about cold production speed.

At 390px populated/expanded families, group selection/removal, fitting inputs,
item kills and group dashboard/kills fit 379px. A phone group text comparison
crossed a relative-age minute boundary (25m vs 26m); all fifty ordered kill IDs
match and inspection isolated the difference to the live clock. This was not
counted as a data mismatch or hidden by masking arbitrary content. The first
phone minimum-range request was interrupted by intentional navigation and was
not accepted as a settled result; settled module selection/removal supplies the
phone filter evidence. Screenshots: /tmp/shrike-universe-review/
item-fittings-phone-current.png and item-fittings-phone-source.png.

Local tab failures keep twenty families and the existing URL; Retry loads fifty
kills and Back restores fittings. A failed fitting sort also retains twenty
families, clears busy and retries to Recently Seen with the error removed.
Independent module-usage failure leaves families/profiles usable. It revealed a
contradictory initial loading caption beside Retry; ItemFitsController now hides
that placeholder on failure and restores it during retry. The existing malformed
supporting-response regression now checks initial loading, failure, retry and
removal after success. A freshly built browser facade proves failure has only
the error, a five-second Retry shows actual loading/busy, and recovery produces
fifteen groups with error/placeholder removed. No controller invocation errors.

Twenty-four universe tests, all **846 frontend tests in 112 files**, strict TS,
production asset build and broad Go pass (TEST_DATABASE_URL unset, integration
skipped). Latest assets: 118 entries/104 chunks, 1,151,589 JS bytes and 119,988
utility CSS bytes. Current owned facade is /tmp/shrike-ts-universe-loading-browser-facade,
4465/4466; user 4200 was not restarted. Remaining original module: system
(system/region/constellation), then the full site/mobile/theme/navigation/loading/
focus/live/performance, Docker/dev/integration and final runtime/artifact audit.
No deployment, release, commit or production mutation. Goal stays active.

Cleanup: the final universe facade is terminal, all fault rules are clear and
all comparison tabs are closed. Fixed initial-error screenshot:
/tmp/shrike-universe-review/item-fittings-metadata-error-fixed.png. User/retained
listeners are unchanged; the goal remains active with system/location validation
and the full final audit still outstanding. Relaunch the named facade for system
comparisons rather than attempting its closed session or stale tab handles.


**System/location browser acceptance, 2026-10-09:** All **87/87** original
modules now have family browser acceptance and remain strict TypeScript. The
anonymous read-only comparison facade uses current Go markup and archived system
script against actual public Jita, The Forge and Kimotoro data. Dashboard text,
Jita's 24-hour/four-metric chart (96 hour slots), selected-hour exact figures,
sovereignty/station/connection/celestial data and all valuable categories match.
Jita/Kimotoro genuinely have no valuable structures; The Forge has eight. No
empty category data was fabricated. Hero image URLs/height remain unchanged
across tabs and canonical metadata updates correctly.

All three location kinds have matching fifty-row kills and fifty-card battles.
System and constellation ten-row cursor paging match; system/region battle page
two match, with fifty cards. Back restores system Kills and its source content.
At 390px all three dashboards, system kills and region/constellation battles fit
379px and match; the phone activity readout also matches exact hour metrics.
The map link's target is the existing region map route; map rendering/interaction
has its previously accepted dedicated family evidence.

Local battle delay/503 exposes busy while retaining fifty cards; Retry recovers
page two and removes the error. Failed tab loading keeps fifty battle cards and
the old URL while valuable categories remain usable; Retry reaches fifty kills.
Valuable failure retains eight ship cards while kills remain usable; Retry
correctly returns the empty structures category. Failed kill-limit changes keep
fifty rows, clear busy, and Retry reaches ten rows. No controller invocation
errors. Twenty-eight location tests pass. Full frontend 846/112, strict TS and
broad Go passed at the immediately preceding universe checkpoint; authored
browser source/assets did not change here. Artifacts:
/tmp/shrike-location-review/system-phone-current.png, system-phone-source.png,
and constellation-battles-phone-current.png. All comparison tabs are closed,
faults are clear, and the owned facade is terminal; user/retained listeners are
unchanged. No production writes, deployment, release or commit. Goal remains active.

**Final audit gates (family acceptance is not overall completion):**
- [x] Recheck strict source ownership, forbidden suppression/any, reusable lifecycle
  ownership and dev/production entry coverage against the current source graph.
- [x] Fix and verify catalogue unknown row estimates and review phone fullscreen
  fitting statistics/observed-fit access; recheck the earlier balance observation.
- [x] Complete the final desktop/mobile/theme/focus/navigation/loading/error/live,
  fitting/map and relevant performance pass with current artifacts and real data.
- [x] Reproduce production asset manifests and every owned output hash/size.
- [x] Verify dev watcher/rebuild/reload and latest production Docker build/runtime.
- [x] Run relevant local database integrations and final Go/frontend/type checks.
- [x] Update architecture/review/TODO docs and audit the objective requirement by
  requirement before marking the goal complete. No deployment is authorized.


**Final fitting follow-up audit, 2026-10-09 (goal still active):** Strict source
and artifact/entry tests pass. The catalogue's pg_class unknown estimate (-1)
now renders an em dash with Count unavailable; real zero counts remain zero.
The numeric public API and inexpensive estimate query are unchanged. A Go
rendering regression covers two unknown values, zero and a positive count;
browser verification uses the retained fixture's real -1/-1/1/0 response.

Fullscreen fitting previously hid statistics at <=900px and observed fits at
<=1199px. The existing responsive grid now puts observed fits beneath tablet
workspace/statistics and stacks both panels under the workspace on phones.
At 390px statistics/variants display and root width/scroll width are both 390px;
keyboard Navigation collapse/reopen works. Clicking a real 200mm AutoCannon II
suggestion fits High 1 and clears status. At 1024px statistics remain a 300px
side panel, observed fits occupy the next grid row, and width/scroll are 1024px.
The desktop three-column rule remains unchanged. Screenshots:
/tmp/shrike-final-fitting-review/fullscreen-phone-statistics.png and
fullscreen-phone-observed-fits.png. No native desktop input was used.

The audit also reproduced cold fitting search loss: typing Rifter while SDE
loaded retained input text but initialized an unfiltered catalogue. Runtime now
reads the current input when it starts. A regression gates the real pinned SDE
before initialization; the browser on a fresh 4467 origin proves Loading fitting
data with Rifter input, then exactly one Rifter result without a second input,
and successful hull selection. Fifteen editor tests and strict TypeScript pass.
Production build is 118 entries/104 chunks, 1,151,631 JS bytes and 119,988 utility
CSS bytes. All 847 frontend tests in 112 files and broad Go pass (TEST_DATABASE_URL unset;
database integrations skipped). The earlier balance observation, global
interaction/theme/mobile/live/performance, reproduction, Docker/dev and local
integration gates remain open. No production mutation or user listener restart.

Final artifact follow-up: a second production build reproduces all 228 tracked
manifest/output/CSS/icon/data files byte-for-byte; all 222 physical JS files match
manifest hashes/sizes and entry coverage. Both full test sessions are terminal
(847/112 frontend and broad Go passed, database integration skipped). All owned
audit tabs are closed and the final fitting facade is terminal; faults were unused
on that facade and user/retained listeners remain unchanged. Goal is active.

Latest local Docker verification started: docker build --progress=plain -t
shrike:typescript-audit-20261009 . (exec session 86840); inspect the live handle before treating
it as finished. No image is deployed. Previous Docker image predates these fixes.


**Final build/runtime and shell audit, 2026-10-09 (goal active):** All 87 original
modules remain strict TypeScript with family desktop/mobile acceptance. Current
source and artifact guards reject authored JS, explicit any, suppression comments,
unowned outputs and missing/invalid imports. Shared Stimulus registration and
RequestComponent/FragmentSections/editor lifecycle ownership were reviewed against
the existing detach/reconnect, request-body cancellation, shared busy-count,
observer, timer, stale-response and fullscreen regressions. Pure helpers, transport
and calculations intentionally remain typed modules rather than controllers.

The isolated dev workflow ran real make dev/Air and the source watcher, skipping
only the already-verified native-worker build prerequisite. TypeScript, stylesheet
and template edits each triggered rebuild and automatic browser reload on the
same tab. An intentional TS syntax error kept the last good instance/page; fixing
it retried and reloaded successfully. Restoring all copied source files also
reloaded and removed every temporary marker. The copied workspace, its process
tree, dedicated Valkey and fresh migrated database are removed. No original .env,
user server, retained fixture or shared dependency was changed. Evidence:
/tmp/shrike-dev-audit-location.json and
/tmp/shrike-final-dev-review/automatic-recovery.png.

Local database/API/ingress integrations applied all 28 migrations and passed
385 tests, with one explicit imported-market-data skip and no failures. The fresh
integration database was removed. The skipped market importer test requires market
orders not supplied by the fresh fixture; populated market UI has separate browser
acceptance. Evidence: /tmp/shrike-browser-audit-go-tests.jsonl.

Fresh desktop/390px account checks repeatedly show the actual fixture's 96k ISK
balance. A delayed local 503 retains the previous value while pending, then shows
an unavailable value; clearing the fault and reopening recovers 96k. Reload retains
authentication. The old Chrome stall did not recur; its original cause remains
unproven. Phone theme changes preview locally, survive Overview/Appearance fragment
navigation, and restore to the original saved state without a database save.
Phone page width/scroll width are 390px in this wrapper.

The phone audit found that both closing the mobile menu and Escape dropped focus
onto body. ShellController now restores focus to its visible summary, taking
precedence over a nested dropdown. The existing shell regression covers both exit
paths. A newly rebuilt production-bundle browser check confirms both paths end on
the Menu summary with all menus closed. Screenshots:
/tmp/shrike-final-site-review/account-phone-menu.png and
account-phone-focus-restored.png. No native desktop input or financial action ran.

After the focus fix, all 847 frontend tests in 112 files, strict TS and relevant
Go frontend/parity/ingress tests pass. Latest production assets have 118 entries,
104 chunks, 1,151,723 JS bytes and 119,988 utility CSS bytes. A second build
reproduces all 228 tracked files; every one of the 222 physical JS outputs matches
its manifest hash/size. Evidence: /tmp/shrike-final-focus-repro.json.

Latest Docker image shrike:typescript-audit-20261009-focus builds and passes actual
production HTTP checks for health/home/about/docs/MCP/fits, plus 86 referenced
assets. Both development reload/version routes are 404; development diagnostics
and instance metadata are absent. Runtime is UID1000 Go/native Rust with pinned SDE
and no Node/Bun. The in-image native worker matches all 16 reference fitting
snapshots. Docker readiness briefly reset its connection; the test harness now
retries that startup condition rather than treating it as a product failure. All
owned runtime containers and the separate migrated database were cleaned afterward.
Evidence: /tmp/shrike-final-focus-runtime.json.

The final cross-site gate stays open. A fresh geography map loads 5,484 systems /
69 regions and its region search navigates correctly; crowded battle-map marker
hit targets still need the recorded follow-up. Remaining work is the final
cross-site/mobile/live/error/navigation/performance review and the requirement
completion/documentation audit. Deeper Memgraph ingestion/query/intelligence work
is explicitly deferred by the user and is outside this migration goal. No
production writes, deployment, release or commit occurred.


**Browser migration complete, 2026-10-09:** All 87 original modules are strict TS
and desktop/phone accepted; all final gates above pass. Latest checks: 855 frontend
tests/113 files, strict TS, broad Go tests/vet, 385 local DB/API/ingress tests (one
explicit imported-market-data skip), 230 reproducible production files and 223
verified shipped JS outputs. Latest Docker image passes production HTTP/assets,
no dev surfaces, Go/Rust UID1000 runtime with no Node/Bun, and all 16 fitting
snapshots. Real make dev watcher/reload/error-recovery is verified.

The final pass fixes overlapping map-marker pointer hits with an explicit native
location choice (keyboard/direct hits unchanged), shares viewport positioning
with fitting-search popovers, and keeps both inside 309/379px content. Live
insertion/dedup/reconnect/focus, loading/failure/Retry, row keyboard navigation,
canonical nav handling and Back are checked. Three stale generated Go templates
are regenerated; all 93 are now idempotent, with current catalogue/map/editor
browser checks. All owned audit services/tabs are cleaned; user/retained services,
original .env and shared dependencies are preserved. No deployment or production
mutation. Deeper Memgraph data/query/intelligence work remains deferred.

The full requirement-by-requirement audit and limits are in
[frontend-review.md](docs/operations/frontend-review.md#browser-migration-completion-audit-2026-10-09).
Proof: /tmp/shrike-final-objective-audit.json, shrike-final-marker-runtime.json,
shrike-final-marker-repro.json and /tmp/shrike-final-map-review/crowded-marker-choice.png.

## Sentry performance leads, 2026-10-14

- [x] Fix SHRIKE-B: distinguish HTTP request timeouts from listener shutdown; v0.2.7 deployed, feed delivery verified.
- [x] SHRIKE-C: v0.2.8 ranks entity breakdowns before enriching names. Warm production benchmarks reduce one corporation query from 9.1 ms to 0.8 ms; regression fixtures preserve ordering, ties, date windows and missing names.
- [x] Coalition directory: v0.2.8 publishes a live-member partial index and maintenance policy. Regular vacuum reduces the 5.6 s / 212k-page query to 26–37 ms / 1,175 pages; exact totals verified in one snapshot.
- [x] Affiliation refresh: v0.2.8 publishes the covering date index and maintenance threshold. Regular vacuum reduces warm selection from 444 ms / 223k pages to 13–23 ms / about 1.6k pages. Dormant index and selection rules retained.
- [x] Benchmark shared campaign attacker input: rejected. On a bounded 20k-killmail sample, the existing entity-first joins finished in 1.5 s warm; sharing all attackers hit the 10 s limit twice. Preserve the selective joins; investigate the remaining campaign work separately.
- [x] Campaign valuable kills: v0.2.9 ranks ten before joining names; measured warm sample improves from 401 ms to 182–183 ms. Scope/tie/missing-name regressions pass.
- [x] Faction-war valuable kills: v0.2.9 publishes migration 30. The valid compound index reduces warm execution from 219–232 ms / 8,881 pages to 10–19 ms / 1,997 pages.
- [x] Solo hull matchups: v0.2.9 publishes the lower killmail vacuum threshold. Regular vacuum restores an index-only killmail scan with zero heap fetches; measured 9–12 ms warm.
- [x] SHRIKE-E: v0.2.9 cancels and drains cache loaders before dependency shutdown. Deterministic/full race tests and a blocked-query local server check pass; all production services active and release error report empty.
- [x] SHRIKE-D: investigate the fixed fourteen-day repair loop. It performs one array-based indexed check per day; keep day bounds and genuine performance signals visible.
- [ ] Follow measured slow campaign candidate/materialization/stat queries, faction-warfare valuable kills, and solo ship matchups; record before/after plans and regression coverage before claiming improvements.
- [ ] Scoped war lists: limit each participant range through the requested page before deduplicating and enriching. Warm sample improves from 107–117 ms to 3.1–3.6 ms; no-match samples fall from 150–230 ms to 2–5 ms. Publish after independent filter/paging/domain regressions and dense-participant benchmarks.

Use production read-only plans for investigation and isolated databases for test
fixtures. Keep real failures visible in Sentry; verify performance findings rather
than optimizing solely to remove a warning. Releases follow docs/deployment/releases.md.
