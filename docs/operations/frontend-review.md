# Final frontend review

The migration goal remains active. HTTP checks do not sign off visual or
behavioral parity; each remaining browser pass must be recorded separately.

## Comparison setup

- Go preview: 4200; user development on 4000 stays separate.
- Public Nuxt reference: 4101, with `NUXT_API_ORIGIN=http://127.0.0.1:4200`,
  `NUXT_API_SOCKET=''` and `SHRIKE_NUXT_BUILD_DIR=/tmp/shrike-nuxt-public-review`.
  Its build directory is separate from the account reference on 4100.
- Public facades: `go run ./scripts/frontend-parity -public-review -public
  http://127.0.0.1:4200 -nuxt http://127.0.0.1:4101 -faults` serves Go on 4201
  and Nuxt on 4102. Both are loopback-only and read-only. Account cookies and
  Authorization are stripped; theme/default-tab/background cookies are retained.
  Nuxt receives the actual Go API, image and WebSocket ingress namespaces.
- Use `frontend-review.localhost` for separate review cookies. The account/API
  fixture and its render-only facades remain on 4450, 4451 and 4461. Writes stay
  in that disposable database.
- Browser viewport control now works. Set it after creating comparison tabs,
  verify the actual document width, and reset it before leaving a handoff.

## Evidence from 2026-10-05

`python3 scripts/frontend-review.py` expands TODO routes, kill categories and
map metric/window combinations, discovers real detail IDs from directories,
and checks HTTP transport plus referenced shared assets. It records unresolved
examples instead of treating missing coverage as success. Area campaigns do
not expose intel; the audit selects a participant campaign with an actual intel
link for that case. Production-host indexing and private routes have separate
gates.

The initial pass covered 314 routes and 96 shared assets. No asset was missing.
The corrected participant-intel case returns 200; all 314 audited HTTP cases
now return the expected successful document/redirect destination. Dynamic
examples listed as unresolved are still pending separate verification.
Current reports are `/tmp/shrike-frontend-http-review.json`,
`/tmp/shrike-frontend-optimized-http.json` and
`/tmp/shrike-review-mobile-entities.json`.

At an actual 393 × 852 viewport, populated public comparisons covered the
homepage, latest kills, character/corporation/alliance/faction, item,
system/constellation/region, categories, market, advanced search, campaign
creator, local/directional scan forms, roam creator and battle generator.
Both documents fit the viewport; these records cover geometry/data states,
with remaining visual/control passes tracked below. The anonymous battle
generator was also compared visually and matches the reference.
The three optimized kill-category pages were also checked at 393px. All fifty
drone IDs and their order match after hydration, with no document overflow.

Cold HTTP checks found costly display joins in killboard queries. Page
selection now precedes final-blow/name/location lookups. Valuable rankings
limit display joins to the selected rows and use early value-index stopping
only for value bands; sparse categories materialize recent matches first.
This avoids a planner regression that would scan historical values looking
for scarce victims. A read-only live SQL comparison preserved all eight
selected value-band IDs and reduced its query from 22,373ms to 98ms. A sampled
drone-day query fell from 922ms to 87ms.

| Page | Initial server time | After correction |
| --- | ---: | ---: |
| `/kills/5b-10b` | 21,144ms | 437ms |
| `/kills/category-drone` | 19,662ms | 3,572ms |
| `/kills/category-sovereignty` | 6,757ms | 473ms |

These are individual local-preview observations against live data, rather
than a latency guarantee. Drone paging still reads 25 sparse day/enrichment
queries; examine further reductions while preserving rollup page semantics.
PostgreSQL tests cover value boundaries, missing types, deterministic ties,
category rankings and day ordering/offsets. All production database checks
used read-only sessions; no production schema changes were made.

The public reference also exposed omitted-Boolean-prop hydration loss in
`KillList.vue`. `cachePayload` now explicitly defaults to true. Fifty drone
rows remain through an eight-second failing client-API test and match Go in
identity/order. Compare IDs independently of per-document navigation tokens.
Tiny reference entity badges use a supported 64px CDN source while retaining
their original CSS dimensions.

### Custom themes and the 320px pass

The real Appearance UI on the disposable account saved Amarr Gold and then
custom primary, ISK, success, warning, danger and selection colors. Nuxt's
Appearance page loaded the same saved values. Clearing only the browser's
fixture login cookies retained the theme on public pages. Entity and location
fragment navigation and browser Back retain colors and focus on the tab link.
War badges, side accents and both selection colors were compared directly with
Nuxt after full document reloads. Writes remained in the local fixture database.

Handwritten Go styles now use the same existing brand/semantic palette variables
as Nuxt's Tailwind overrides, rather than fixed blue/red/green/yellow/emerald
palette values. Root defaults and the shared map renderer remain intact.
Text selection now uses its own saved background and text variables. Mobile
fitting validity/price badges remain visible and wrap instead of being hidden.

At an actual 320 × 740 viewport, comparisons covered entity and location bases,
item/group/category/market pages, battle/war/campaign/coalition details, kill and
legacy details, public directories, fitting discovery/search/detail, stats,
graph, FAQ/docs/MCP/status, wallet/about/donate/search/post, tools/scanners and
the advanced-search/campaign/roam/battle creator forms. Homepage, latest kills,
geography and sovereignty maps were also inspected. These are geometry/data
checks, not a claim that every control or tab is signed off.

This pass fixed war-party grid expansion, inline statistics heading overflow,
implicit minimum grid widths in API docs/MCP examples, and populated Status
tables escaping the page. Wide Status tables now scroll inside keyboard-focusable
named regions. Entity/item/location/group icon tabs have accessible names at
mobile widths. Inner content sections no longer create nested main landmarks.
The API docs' unique-operation counter was also fixed in both renderers; one
operation can appear under several tags. Go now shows 528 of 528 rather than
619 of 528, and a real mobile Try it GET for killmail 138647320 returned 200
(559ms observed), with the response contained inside the viewport.

The corrected war header was compared visually at 320px; screenshot:
`/tmp/shrike-review-themed-war-320.png`. A 1px homepage overflow was traced to
the day heading's intrinsic grid width and fixed by keeping it inside its track.
The reloaded homepage now has equal 320px client/scroll widths.
Reference-only overflow was observed on
character, fit detail/discovery and populated Status at 320px; preserve Go's
containment while comparing controls and content. Recent battle 1162820 vanished
as live reports rebuilt; the directory supplied report 1162832 for the visual
comparison. The populated coalition example is `/coalitions/the-imperium`.

Go frontend/fixture race tests, 172 Bun tests (876 assertions), Nuxt typecheck
and docs/status browser-module builds passed. The full review remains open.

At 320px, map Security/Danger/6h controls produce the same URL and settled
legend in both renderers (74.5 per 100 jumps in this sample). Jita search and
navigation/Back restore the populated map and filters. Wait for loading and
Nuxt route transitions before comparing data. The reference's clipped toolbar
buttons were activated with their real Enter-key action.
A broad narrow-width rule had hidden Go's general map status/reset controls;
the compact panel now exposes counts, hourly window and Reset. Search uses the
available width, and reset remains visible in sovereignty/live/AIID modes.
Screenshot: `/tmp/shrike-review-map-controls-320.png`; follow-up evidence:
`/tmp/shrike-review-responsive-followup.json`.

### Tablet and saved detail examples

At 768 × 1024, the published `fixture-settings-post` article and its real
comment thread match Nuxt visually. Collapsing the single reply hides it in both
renderers. Saved local/directional scans were created through the actual local
API, then loaded anonymously in both renderers. Their summaries match: four
resolved pilots/one threat, and five directional objects/four types.

The same local API created and updated a roam report to cover the fixture's
existing kill times. It contains seven kills, one system, one engagement and
three resolved roster pilots. The render-only comparison harness now supplies
the existing RoamReportReader interface from that API without rebuilding or
replacing its database. All five report views fit 768px. Composition table text
matches exactly, including the two pilots without observed ships; filtering
for Support retains the matching unobserved pilot in both renderers.

Saved route IDs are in `/tmp/shrike-review-dynamic-examples.json`; they belong
to the disposable 4450 API and are rendered through 4451/4461. The report ID is
`5056188cd9f186eec5ba1cc9cd071c11`. These populated examples close the discovery
gap, while additional width/control/metadata checks remain part of the final
review. No production writes were used.

Saved scans and every roam view, including the direct information alias, were
also compared at 320 × 740. Go remains contained with no visible alerts/broken
images. Nuxt's populated Intel/Systems views overflowed to 504/348px; Go's
bounded containers are retained. Native drag retries on both an overridden
393px tab and a fresh normal 1703px tab still did not register a drop. The
normal-size Rifter handoff has the module search prepared and High 1 empty;
do not count this as a successful drag test.

## Private layouts and keyboard follow-up

The disposable owner account's nine Settings sections, Tracker/Trackers and
all eleven admin families were compared at desktop, 768 × 1024 and 320 × 740.
The tall pages' actual client widths are 757px and 309px because of the
scrollbar; both renderers have the same gutter. Record settled content rather
than Nuxt's initial client-loading view. Tracker's two tracked entities and
the complete widget configurator were explicitly rechecked after loading.
Standalone user 44001 and domain 9101 details were also checked at tablet and
phone widths. Domain Approvals and Files expose the same three pending assets
and five file rows, with the real local preview images loaded. Wide tables
remain contained; no admin action was submitted during this follow-up.

Preferences' timezone buttons had expanded both tablet documents from 757px
to 904px. Their flex children now permit shrinking, including Go's generated
timezone button state. Default-page selectors use two columns on tablet,
three on larger screens and four on wide screens, leaving Dashboard readable
beside the Settings sidebar. Local/EVE switching and a matching source reload
retain containment. Screenshot: `/tmp/shrike-review-preferences-tablet.png`.

At phone width, moderation status pills and the Comments filter toolbar now
wrap in both implementations. Their former 312px/350px scroll widths are both
309px after correction. Approved and Reported filters were exercised against
the actual local records, including the safely escaped review note. Source
Tracker alone still overflows to 323px at a 309px client width; Go's contained
layout is retained. Existing write/error tests remain the separate behavior
evidence; this sweep does not imply that every control was exercised again.

News ticker animation copies are now hidden from assistive technology and
excluded from the tab sequence in both renderers. A real Tab sequence reaches
the primary Dismiss button once, pauses the animation while focused, and skips
the clone. Collapse/Expand work with Enter. The source collapse/dismiss buttons
also have explicit names. Screenshot: `/tmp/shrike-review-ticker-accessibility.png`.
The accumulated records are `/tmp/shrike-review-private-regression.json`.

The shared production indexing/cache code and existing foundation tests were
reviewed: indexing requires an opted-in successful page on the canonical apex;
private pages, errors, previews and tenant/development hosts remain noindex.
Shared HTML caching is opt-in, production/apex-only and bypassed for Cookie,
Authorization or Set-Cookie. Versioned assets are immutable with conditional
304 support; unversioned aliases revalidate. Sitemaps advertise only published
routes and use the canonical origin. These are code/test checks, not a live
production deployment. Remaining per-route metadata cases are still below.

After these changes, frontend/fixture race tests, 172 Bun tests (876 assertions),
Nuxt typecheck, browser-module builds and `git diff --check` passed. Test logs:
`/tmp/shrike-keyboard-review-bun-tests.log` and
`/tmp/shrike-keyboard-review-typecheck.log`.

## Public tablet follow-up

The 768 × 1024 sweep recorded 114 comparisons covering 111 distinct URLs:
every inventoried character/corporation/alliance/faction tab, item/group/market
and location tabs, frontpage/kill categories, coalitions/rankings/conflicts,
fitting discovery/detail, forms, docs/information pages and every distinct
map mode/scope. Go had no document overflow in these records. Populated Status
cards/tables and the region SVG's 158 systems were explicitly rechecked after
loading. Native tab clicks exercised character Kills/Losses/Top and browser
Back; loss identities match and the saved theme persists. Per-feature controls
remain supported by their earlier evidence rather than inferred from geometry.

Tablet Go entity headers had hidden rankings entirely below 1024px. They now
keep the ranking and affiliation logos alongside the dates in a second row,
leaving names readable. The phone layout retains the source's compact header.
Screenshot: `/tmp/shrike-review-entity-tablet.png`; accumulated records:
`/tmp/shrike-review-tablet-public.json`. Reference-only overflows occurred on
war 763223 and the current battle 1162917 (including its composition/timeline/
replay views); retain Go's containment.

The shared HTML cache policy now includes `Vary: Cookie, Authorization` on
eligible responses, preserving existing variation. Tests cover anonymous,
preferences, authorization, response cookies, tenant and preview cases. The
earlier per-request bypass alone did not tell a shared cache how to distinguish
its already-stored anonymous response.

## Loading feedback, 2026-10-06

One shared progress bar now observes native page navigation and existing
component `aria-busy` transitions. It uses the saved primary theme color,
highlights the clicked control/killmail row, announces loading politely, and
respects reduced motion. Full navigation starts synchronously; fragments wait
150ms. No fetch monkeypatch, polling or per-row network request is added.
Static lazy placeholders and explicitly silent periodic updates do not keep
the bar running. Missing busy markers were added to campaign, wallet, rankings,
blog and legacy controllers; stale completions in older handlers now retain
the replacement request's loading state.

Actual disposable admin fragment tests cover a 2.2/5-second delay, visible
themed bar and clicked-tab highlight, successful completion, a delayed 503,
Retry recovery, cancellation by another section and instant cached Overview.
No bar, pulse or busy marker remained afterward. Screenshot:
`/tmp/shrike-review-navigation-fragment-loading.png`. Native JS user-row
navigation reached the correct detail under a four-second server delay.
The pending full-document paint was subsequently captured through Chrome's
native window during a five-second fixture delay; evidence is recorded below.
Native event lifecycle tests separately cover immediate JS-row feedback, aborts,
replacement requests, errors, downloads (including an empty filename), hashes
and same-document history. No synthetic gesture is claimed as browser proof.

Checks passed: 184 Bun tests (917 assertions), Nuxt typecheck, frontend/fixture
race tests, both Go builds and browser-module builds. Logs:
`/tmp/shrike-loading-review-bun-tests.log` and
`/tmp/shrike-loading-review-typecheck.log`. The complete site is on 4200;
4451 is the disposable account renderer and its homepage is a placeholder.

## Featured attacker item images, 2026-10-06

Killmail 138885892 includes a corporation-owned Standup Arcing Vorton
Projector I in the featured attackers. Its type is a module, although ESI
places it in `ship_type_id`. The old character/corporation heuristic requested
a ship overlay render, which returned 502 for that module.

The existing attacker query now includes the type's category through its
group join, without adding a query. Both Go and the Nuxt reference use item
icons for known non-hull types and ship renders for ships/structures. Featured
images also fall back once to an item icon if a render fails, including older
cached payloads without the category. A missing character does not produce a
portrait request. The generated API contract includes the optional category.

The actual killmail was checked in both renderers: the module icon loads,
the Aeon retains its render and character portrait, and the module card has
no portrait. At a requested 320px viewport, Go's client and scroll widths
both remain 309px, and both featured images load when scrolled into view.
Screenshot: `/tmp/shrike-review-kill-attacker-icon.png`.

Checks passed: 186 Bun tests (925 assertions), Nuxt typecheck, frontend/API/
fixture race tests, both Go builds, and the complete API suite against the
owned local PostgreSQL test databases. New regression coverage exercises
corporation-owned module versus ship categories in the actual SQL reader
and checks the featured card markup and matching image selection rules.
Logs: `/tmp/shrike-attacker-image-bun-tests.log`,
`/tmp/shrike-attacker-image-typecheck.log` and
`/tmp/shrike-attacker-image-api-tests.log`.

## Phone entity and sparse-day follow-up, 2026-10-06

The phone character Losses and Combined tabs retain focus, theme and matching
kill IDs without full-page reloads. The settled Top tab matches the source's
activity/location rankings and remains contained at a 309px client width.
The initial quick navigation through the other character tabs is not a
populated-content sign-off; finish those checks on resumption.

Go had hidden history kill/loss counts and duration below 640px, while the
source crowded them beside the name. Both now use the same compact two-row
grid, preserving counts, dates and duration. The actual character history's
three cards and 48px images were checked in both renderers at 320px without
document overflow. Screenshot: `/tmp/shrike-review-phone-entity-history.png`.
The corporation history uses the same responsive treatment; its final browser
check remains pending.

Numbered killlists now batch sparse day reads through bounded lateral queries,
with display joins applied only after the overall page limit. The single-day
path remains unchanged. Actual local PostgreSQL regression coverage proves
50 sparse days require two reads before optional market-path enrichment,
and checks ties, offsets, empty/stale overcounts, undercounts, and last pages.
Stale overcounts continue into older days to fill the actual page.

A live cold drone document retained all fifty kill IDs and dropped from 26
queries / 2,212ms server time to four queries / 991ms. The latter used a separate
build/cache namespace on temporary port 4203, which has been stopped. Warm
cache timings are excluded from that comparison. The page's arbitrary query
parameters do not alter its initial SSR page; both samples are the first fifty
rows. Records: `/tmp/shrike-sparse-killlist-before.json` and
`/tmp/shrike-sparse-killlist-cold-after.json`. No production DDL/cache flush
was used.

Checks passed: complete local PostgreSQL API suite, frontend/API/fixture race
tests, 186 Bun tests (925 assertions), Nuxt typecheck and browser-module build.
Logs are `/tmp/shrike-sparse-review-api-tests.log`,
`/tmp/shrike-sparse-review-go-race.log`,
`/tmp/shrike-sparse-review-bun-tests.log` and
`/tmp/shrike-sparse-review-typecheck.log`.

The owned 4200/4451 renderers now run the latest binaries. The disposable 4450
API and user port 4000 remain untouched. The resumed browser checks follow.

## Resumed entity review, 2026-10-06

The requested 320px pass now covers all 34 inventoried character, corporation,
alliance and faction base/tab routes. All Go documents remain contained.
Actual client widths vary between 309px and 320px with the browser gutter;
record the measured width. Source-only overflow remains on the character
dashboard and alliance corporation list, where Go's containment is retained.
The saved custom primary and entity palette accents persist. Direct History
loads and 768px/320px resizes reveal the selected tab in both implementations
without moving the page vertically. Records:
`/tmp/shrike-review-entity-tabs-320.json` and
`/tmp/shrike-review-active-tab-resize.json`.

Every combat tab's fifty kill IDs match. Populated battle, war, campaign, top
and history data and links were compared after loading; the empty corporation
war and character campaign cases also match. Corporation/alliance membership
lists match their first hundred identities, with matching activity, sorting,
paging and corporation-scoped results. Live activity cutoffs/cache ages can
change a total by one between requests. Source refreshes retain old content,
so absence of its initial spinner alone does not establish completion; the
late leaderboard and dashboard reads were rechecked. Source Enter did not
activate the security-sort assignment in the connector; its actual click
provides the comparison. This is not claimed as a successful source Enter test.

Restored Go's top member pager on phones. Source member pager/clear buttons
now have accessible names. The corporation picker focuses search, resets it
when closed and supports outside dismissal; Go Escape closes it and returns
focus to the summary. Source's search field also handles Escape explicitly,
though its connector key/dismissal result remains inconclusive.

Keyboard testing found that replacing member fragments lost Go's focus.
`fragment.js` now preserves the matching control through member activity,
sorts and both pagers, corporation sorting, top periods, war sorting and
killlist limit/paging. Actual Go Enter actions retain focus and produce the
same ordered results as source clicks. The helper respects repeated controls,
visibility, disabled controls and focus outside the replaced subtree; request
guards still reject stale replacements. Screenshot:
`/tmp/shrike-review-phone-members-pager.png`.

A desktop loss-row link comparison exposed a corporation-owned NPC portrait
being incorrectly chosen from its display-name entity. Portrait selection is
now separate: NPC type icons link to their item while the corporation name
and badges remain intact. Actual killmail 138941743 loads the type-9965 icon,
and all fifty loss IDs and row links match Nuxt. Mobile victim summaries also
retain the source's character/corporation text. Screenshot:
`/tmp/shrike-review-npc-killlist-portrait.png`. Regression tests cover player,
corporation-owned NPC, missing NPC type and victim portrait behavior.

Faction flips now use the source's Lucide arrows and UTC date format, including
the offset/day-boundary case in tests. Source militia-corporation ties now use
the same ID tie-breaker as Go, so the tenth-ranked corporation agrees. The
populated faction dashboard links and warzone totals match.

Latest frontend/fixture race tests, Nuxt typecheck, Go builds, browser-module
builds and `git diff --check` pass. Logs:
`/tmp/shrike-entity-review-complete-go-race.log` and
`/tmp/shrike-entity-review-complete-typecheck.log`. No API/SQL behavior changed
in this resumed slice, so the prior full local PostgreSQL suite remains its
separate backend evidence.

A fresh Rifter named Drag verification is prepared on port 4200 with Hardware
searched for 200mm AutoCannon I and High 1 empty. The user has been asked for a
physical mouse check; no drag success is claimed yet. Keep browser tab
351724747 as a handoff while awaiting that answer. Its URL is saved in
`/tmp/shrike-fitting-drag-review-url.txt`.

Metadata follow-up checked twenty live/owned-fixture cases, including saved
scans, every roam tab/Information alias, the published blog example, anonymous
preview denial, current entity tabs, kill, market, fitting creator, docs and
MCP. Titles and deliberate canonical paths are correct; preview hosts remain
noindex/no-store. Status intentionally has no canonical and is noindex in both
the source and Go; battle/campaign subviews deliberately canonicalize to their
report base. The inventory parser now distinguishes the document head's title
from SVG tooltip titles. Report: `/tmp/shrike-review-metadata-followup.json`.

The next phone pass has started on item 11186. Its twenty observed-fit cards,
histograms/module usage, fifty kill rows and populated market books remain
contained at 320px. Kill IDs match the source. Both market versions show 100
sellers/70 buyers globally and 51 sellers/16 buyers after selecting The Forge;
the region query is retained in the URL. These checks are recorded in
`/tmp/shrike-review-item-location-mobile.json`. The fitting and representative
location controls now have the additional evidence below; remaining item/group
examples and the wider desktop/tablet/theme review still need completing.

## Item and location controls, 2026-10-06

The item 11186 fitting review covers shorthand/invalid input and correction,
unfinished form entries across sorting, filtered family ordering, expanded
family retention, module requirements, removing one of two named module chips,
Guristas/Angel Cartel NPC profiles, histogram range filters, ten-row kill lists,
next-page results and browser Back. All tested fitting and kill-list orders
match the settled Nuxt results; NPC labels now use the source's faction names.
The earlier raw NPC-label mismatch is retained in the evidence with its passing
follow-up. Go restores the filtered fitting URL on Back; Nuxt currently drops
the manually replaced filter query on that transition. Go retains this useful
behavior. Report: `/tmp/shrike-review-item-fitting-controls.json`.

Loaded module-use/distribution panels remain visible during filter updates;
their query markers distinguish old visible data from settled replacements.
Sorting reuses panels whose data query is unchanged. Form drafts and surviving
expanded families are preserved. Shared fragment focus restoration covers the
main fitting/market/list updates and subsequent supporting-panel replacements,
with request identity, abort and connected-node checks after asynchronous
imports. A selected module group retains keyboard focus through both updates.

Market period/security controls, numeric book sorting, region changes and empty
books are tested at an actual 320px width. A real browser failure found that
the All Regions option sent `region=0` while Go rejected it. Both embedded and
standalone market fragments now accept zero, retain the current book sort and
preserve selector focus; negative regions remain rejected. Regression cases
exercise both fragment handlers. With low-security orders excluded, both
renderers show the same 100 sellers/65 buyers globally. The final security
category cannot be deselected, and sorting an empty book preserves its empty
message. Report: `/tmp/shrike-review-market-mobile-controls.json`.

The location pass reproduced a cancelled Metropolis dashboard link: a handler
matched the content section's `data-system-tab`, prevented normal navigation,
and left the URL on the region. It now intercepts only actual tab anchors and
allows ordinary content links. Clicking Hadozeko opens its system page in both
renderers. Location tab updates also refresh the document description/canonical
and preserve the persistent header. Ten-row kills and next pages match for
Hadozeko, Metropolis and Agiesseson; Hadozeko's eight valuable ships and two
50-record battle pages, Metropolis's 50 battles and Agiesseson's 13 battles
also match. Kill-limit, pager, valuable and battle-pager controls retain focus
after keyboard activation. All tested states remain contained at 320px.
Agiesseson's Map button opens Sinq Laison in both renderers; the loaded SVGs
have the same 1,212 circles and contained phone layout. Report:
`/tmp/shrike-review-location-phone-controls.json`.

Screenshots: `/tmp/shrike-review-phone-item-filters.png` and
`/tmp/shrike-review-phone-constellation.png`. Latest frontend/fixture race tests,
both Go builds, changed browser-controller builds and `git diff --check` pass;
test log: `/tmp/shrike-item-location-review-go-race.log`. The wider final review
and real fitting drag handoff remain open.

## Remaining item/market review and native loading paint, 2026-10-06

At 320px, Tritanium retains a 64px icon, keyboard-closeable full-size image,
fractional volume/prices, matching two-kill results and disabled next paging.
Its global and The Forge order books match all rendered quantities/prices/
locations/minimum quantities; the regional case has 77 sellers/75 buyers.
The mineral disclosure exposes the same unpublished type in both renderers.
Go's unpublished card now ports the actual compact source component: a 32px
icon and muted single name. It also avoids unavailable unpublished ship renders.
Published mineral/interceptor cards have matching ordered names/links, and
50/100-row interceptor kills plus the second 100-row cursor page match Nuxt.
Chalcopyrite opens correctly after waiting for the source transition.

Standalone market search matches all nine Rifter hits and opens its matching
order book. Delayed search exposes the source's inline spinner plus the shared
progress bar/input highlight. Clearing aborts pending Go requests and leaves
no spinner/bar/results; late responses cannot reopen the popup. Lazy tree
requests also expose busy state and a visible failure message. A delayed 503
clears the bar; close/reopen recovers all sixteen ammunition categories in
the same order as Nuxt. Disclosure summaries receive the loading highlight.
All tested states remain contained at 320px. Report:
`/tmp/shrike-review-remaining-item-group-controls.json`; screenshots:
`/tmp/shrike-review-phone-unpublished-type.png` and
`/tmp/shrike-review-market-search-loading.png`.

The full-document visual checkpoint is complete. With only the disposable
4450 API delayed for five seconds, Chrome's native Return action on the roam
killmail link painted the saved purple bar and row highlight while the old
document remained visible. Ordinary browser screenshot/DOM APIs wait for the
navigating frame, so the native Chrome window was captured instead. The
destination is actual local killmail 2000000207 with retained previous/next
context. After arrival the bar is hidden, status empty and no pending class
remains. Screenshot: `/tmp/shrike-review-native-window-loading.png`.
Fixture fault rules were cleared afterward; production services were unchanged.

Native mouse drags were retried with the Go fitting tab visibly selected,
the real draggable type 486 button and measured high-slot target. Neither
attempt changed High 1 or the URL. Keep the real-drop check open; this is not
a passing gesture or proof of a renderer defect. The Rifter handoff remains.

`go test ./...`, `go vet ./...`, frontend/fixture race tests, both Go builds,
browser module builds, 186 Bun tests (925 assertions), isolated Nuxt typecheck
and `git diff --check` pass. Logs:
`/tmp/shrike-review-all-go-tests.log`, `/tmp/shrike-review-all-go-vet.log`,
`/tmp/shrike-market-feedback-go-race.log`,
`/tmp/shrike-market-feedback-bun-tests.log` and
`/tmp/shrike-market-feedback-typecheck.log`. The full Go suite does not replace
the separately recorded disposable PostgreSQL integration suite.

### Current evidence audit

Existing report files were inspected rather than treating TODO checkboxes
alone as proof. The 114 tablet/public comparisons cover 111 URLs; every paired
primary palette matches and no Go document overflows. The private report's
latest state per route/width covers 70 states: older Preferences/Moderation/
Comments overflows have passing follow-ups, and all latest states are contained
with matching palettes. Nineteen comparable responsive follow-ups also have
no Go overflow or palette mismatch. Entity-phone controls, item/location
controls and twenty metadata cases supply separate populated evidence.
These prove their recorded states and controls, not every possible interaction.

The shared home/kill-directory and related killfeed focus checks are completed
below. Continue with the remaining fragment controls, phone map modes/scopes
and semantic-color audit. Fitting drag remains a handoff.

## Shared killfeed focus, 2026-10-06

A real homepage Enter action reproduced focus moving to BODY after a page-size
change, while Nuxt retained its button. The shared fragment helper now supports
both root replacement and replacing a persistent container's children. It
captures focus before optional local-state preparation and retains the matching
enabled control afterward. Home/directory, legacy, war, battle and campaign
killfeeds use it, with abort/request/connected-node checks after imports.
Campaign page-one refreshes keep existing rows visible while updating.

The homepage's open quick-filter panel is retained as a DOM node, preserving
unfinished queries and pending picker targets through row updates. Selecting a
hit hides its suggestions; blur/re-entry matches the source picker. Closing the
panel or Clear all clears drafts/cancels pickers. Detached or superseded picker
responses cannot update a replacement panel.

At 320px, homepage checks cover page sizes, a typed ship query retained across
Solo filtering and 25 rows, Rifter selection, the filtered second cursor page,
Clear all and valuable Ships. All ordered results match Nuxt. Drone-directory
ten-row and next-page IDs also match; its mobile Top Lists view settles without
overflow. Legacy IDs match on both ten-row cursor pages after confirming the
reference's selected size; its reference links use `/kill` while Go uses the
direct `/legacy/kill` route. Battle 1165666 matches both ten-row pages. Its last
page disables Next, so disabled-control focus is not claimed. Current war
763223 has no rows in either renderer; the ten-row control retains Go focus.
Populated public campaign `IyDbzyLSYsbB4x` matches both ten-row pages and stays
contained in both renderers. The older isolated FourSides fixture is now empty
and the reference alone overflows to 351px; it is not used as populated proof.

Delaying a 25-row facade request for 2.5 seconds with a 503 then choosing ten
rows leaves the newer selection/focus intact with no error or busy/bar state.
A genuine failure retains ten previous rows and focus; clearing the fault and
retrying loads 25 rows and removes the notice. Fault rules were cleared.
No production mutation was used. Report:
`/tmp/shrike-review-shared-controls.json`; screenshot:
`/tmp/shrike-review-shared-list-focus.png`.

Initial reference control actions that did not actually change its selected
size are retained with passing confirmed follow-ups. Source navigation timeouts
were re-polled using the same tabs and settled normally; no process was restarted
because of those observation timeouts. Frontend/fixture race tests, 186 Bun
tests (925 assertions), both Go builds, changed browser-module builds and
`git diff --check` pass. Logs: `/tmp/shrike-shared-focus-go-race.log` and
`/tmp/shrike-shared-focus-bun-tests.log`.

## Phone map modes and semantic palette, 2026-10-06

Real 320px scope controls switch both renderers to Zarzakh (1 system/1 region),
Wormhole (2,604/33), Abyssal (200/5) and Proving (201/6), with matching settled
counts and 302px canvas widths. Proving's Security/Danger/6h controls produce
identical URLs and zero-activity legends. The proving-name search correctly
returns no matches. Returning to New Eden restores 5,484 systems/69 regions
while preserving the chosen layer/window. Jita search opens the same system
detail; browser Back restores the populated map and filters. Source activity
buttons can be clipped inside its horizontally scrolling toolbar; actual
keyboard activation was confirmed by the resulting URL before comparison.

Live mode connects to the relay and has the same sixty ordered killmail links
in both renderers. Labels/Routes toggles update identical URLs and keep the
document contained. The visible phone Reset view control is the shared canvas
button; the separate desktop live-status control is intentionally hidden at
this width. Sovereignty loads 73 ranked holders/2,712 claimed systems, with all
ordered holder labels matching. Alliance search opens The Initiative's same
entity page; Back restores sovereignty/layers/window/display toggles, and
Recent changes updates both URLs to `changes=1`. These are navigation links,
not a claim that clicking the search result only changes camera focus.
Report: `/tmp/shrike-review-phone-map-controls.json`; screenshots:
`/tmp/shrike-review-phone-live-map.png` and
`/tmp/shrike-review-phone-sovereignty-map.png`.

The semantic-color audit reads the actual source `@theme` mapping: blue
400/500, red 400/500, green 400/500, emerald 400/500 and yellow 400/500 follow
their saved variables. Amber rarity/security/rating decorations and map-data
gradients remain fixed in the source. Accordingly the group Unpublished badge
retains its source amber color. Diagnostic primary-card tint/border now use
the saved primary and failed-query text uses the saved error color. Browser
computed styles confirm the purple primary at 7% background/22% border; the
panel opens/closes and stays contained at 320px. Screenshot:
`/tmp/shrike-review-theme-diagnostics.png`. Existing fixed render-stage and
geographic palettes were preserved. Frontend/fixture race tests, both Go builds
and `git diff --check` pass; log: `/tmp/shrike-map-theme-review-go-race.log`.

## Final audit and value-band query correction, 2026-10-06

The final HTTP sweep passes 314 routes and 97 referenced assets. Ten separate
requests cover the published fixture blog article, The Imperium coalition,
saved local/directional scans and all six roam URLs, including Information.
All ten return their expected populated documents. These are transport checks;
their visual/control evidence is in the earlier review sections. Reports:
`/tmp/shrike-final-audit-http.json` and
`/tmp/shrike-final-audit-dynamic-http.json`.

Remaining fragment controls were exercised with actual keyboard input. Loki
fit search appends 24, 48 and 72 results; all 72 decoded fitting documents
match Nuxt in order. Compressed URL bytes differ between renderers, so compare
decoded documents. Both versions disable Show more and release focus during
loading. A local failed append retains 48 rows and exposes Retry; clearing the
fault recovers the matching 72. At 320px, an isolated failed Statistics table
leaves the other two Activity tables and 14d selection usable. Retry restores
all thirty ordered entity links matching Nuxt, clears the notice/bar, and
both documents remain contained. A removed Retry button has no enabled
counterpart, so retained focus is not claimed. Fault rules were cleared.
Report: `/tmp/shrike-final-audit-controls.json`; screenshot:
`/tmp/shrike-final-audit-statistics-phone.png`.

The sweep caught another PostgreSQL plan regression: `/kills/5b-10b` took
13.5s, with an 8.8s subsequent server request. Parameterized lateral day reads
chose the historical value index and discarded about 245,000 matches per day.
The two middle value bands now materialize narrow date candidates first,
apply each rollup offset/cap, select the overall page, then retrieve full rows
for existing display joins. Single-day reads and other categories retain
their existing plans. Read-only sampled query plans changed from 1,728ms to
272ms for 5b–10b and 5,759ms to 294ms for 10b–100b. A repeatable-read database
comparison preserves the same fifty IDs and order. Updated Go page requests
measured 862ms and 972ms respectively, each with three SQL queries. These are
individual observations rather than latency guarantees; cold faction-war
documents still measured about 3.6s and the drone directory about 2.8s in the
sweep. No production schema or cache mutation was used.

Browser checks at 320px match all fifty initial 5b–10b rows, both hundred-row
pages, and the adjacent 10b–100b pages. Saved purple palettes apply and all
documents stay contained. The disposable PostgreSQL regression covers both
bands' inclusive lower/exclusive upper bounds, tied timestamp ordering,
display data, pagination and stale day overcounts/undercounts. Each band owns
a separate test database so its boundary data cannot contaminate the other.
Screenshot: `/tmp/shrike-final-audit-value-band-phone.png`.

After the last query change, the full Go test suite, vet, API/frontend/fixture
race tests, disposable PostgreSQL API integration suite and Go build pass.
The unchanged browser sources also pass 186 Bun tests/925 assertions and an
isolated Nuxt typecheck; only known duplicate auto-import warnings remain.
`git diff --check` passes. Logs: `/tmp/shrike-final-value-go-tests.log`,
`/tmp/shrike-final-value-vet.log`, `/tmp/shrike-final-value-race.log`,
`/tmp/shrike-final-value-api-integration.log`,
`/tmp/shrike-final-audit-bun-tests.log` and
`/tmp/shrike-final-audit-typecheck.log`.

## Fitting hull-drop follow-up, 2026-10-06

The final drag review found a concrete source omission: Go rendered the
invisible ship-image drop target without accepting its drops. It now accepts
fresh sidebar modules/drones and auto-fits them through the existing fitting
manager. Fitted-slot moves and charges remain no-ops on the hull, matching
`HullDraggable.vue`. Its payload parsing/routing contract is extracted into
`web/app/utils/fitDrag.ts`, shared by Nuxt and the generated Go browser module.
Sidebar drags also retain the source copy effect and 64px item-image preview.
Rebuild the checked-in module with `make build-fit-drag-frontend`; rendering
still needs no Bun process.

Three new behavioral utility tests cover all accepted slot classes, drones,
fitted indexes including zero, charges/cargo and missing/external payloads.
All 189 Bun tests (943 assertions), frontend/fixture Go tests, isolated Nuxt
typecheck, the Go build and `git diff --check` pass. Logs:
`/tmp/shrike-fitting-hull-drag-bun.log`,
`/tmp/shrike-fitting-hull-drag-go.log` and
`/tmp/shrike-fitting-hull-typecheck.log`.

Native verification used a freshly selected owned fitting tab and the actual
module button/central hull coordinates. Temporary event tracing captured
`dragstart BUTTON` but no `dragover` or `drop`, with no JavaScript errors and
High 1 still empty. The central point is confirmed to hit the visible hull
drop target. An earlier attempt hit the separately selected Home tab and is
not fitting evidence. Native Escape leaves no pending gesture. Report:
`/tmp/shrike-fitting-native-drag-events.json`. Temporary tracing was removed;
it is not a product feature. This proves an inconclusive automated gesture,
not a working real drop or a fitting defect. The subsequent physical checks
are recorded below.

## Completed review

The broad review notes have been reconciled against the recorded evidence:

| Review area | Evidence |
| --- | --- |
| Public desktop/phone/tablet layouts and palettes | Earlier per-feature comparisons, 114 tablet records/111 URLs, 46 phone entity-tab records, location/item controls, all map modes and four extra scopes |
| Private settings, trackers and admin pages | Seventy latest route/width states, plus the real isolated-account save/upload/authorization/error suites recorded in TODO and above |
| Saved scans, roam, blog and coalitions | Populated 320px/768px comparisons, all report tabs and composition filters, twenty metadata checks and the ten fresh HTTP checks |
| Navigation, focus and request feedback | Entity/location Back/Forward, retained fragments/drafts, shared killfeed checks, delayed/failed/cancelled requests and native full-document loading paint |
| Live/reconnect, indexing/cache and performance | Earlier per-feature WebSocket/reconnect suites, production-host metadata/cache tests, sparse-day batching and the two measured value-band corrections |
| Final build/tests | Passing checks and exact logs recorded immediately above |

These cover the recorded routes, states and relevant controls; they do not
claim exhaustive testing of every possible data/input combination.

The user physically confirmed sidebar-to-slot drops, moving a fitted module
between slots, and Cmd+Z undo. The subsequent browser observation shows
**High 1: 200mm AutoCannon I**, with no visible error or horizontal overflow.
This is a real manual check; the inconclusive automated drag is not its proof.

The user also physically confirmed that dropping a fresh sidebar item onto
the central ship image fits it correctly. The final browser observation shows
**Low 1: EM Armor Hardener I**, no visible alerts, and a contained document.
Together with the confirmed slot drop, slot move and Cmd+Z checks, this closes
the fitting gesture gate. Screenshot:
`/tmp/shrike-fitting-final-manual-validation.png`.

The final source inventory was refreshed after the manual report: all 66
current Nuxt route files are represented by TODO's page families, expanded
tabs or redirects, including the market catch-all and ship/shipgroup aliases.

The completion audit inspected the current checklist, route files, comparison
reports and passing test logs listed above. All 136 checklist entries are
complete. The full Go suite, vet, API/frontend/fixture race tests and disposable
PostgreSQL integration suite pass; final fitting changes also pass 189 Bun
tests/943 assertions, frontend/fixture Go tests, Nuxt typecheck and the Go build.
The latest HTTP audit covers 314 routes/97 assets, supplemented by ten populated
dynamic examples. Later helper imports are covered by asset-integrity tests
and successful real browser loading. The physical fitting checks close the
only gate that automated input could not establish.

This is validation of the inventoried page families and recorded scenarios
in the development branch, using Chrome comparisons and automated server/
utility tests. It does not claim exhaustive coverage of every filter/data
combination or additional browser engines. The cold-load observations and
known typecheck warnings remain documented above. No production deployment
was performed as part of this migration goal.

## Go-only runtime cleanup, 2026-10-06

Removed the Nuxt application, Bun locks/tooling and separate comparison-renderer
wiring after the completed parity audit. Reusable browser sources, generated API
contracts and retained interaction/WASM tests now live in `frontend/`, using
npm/esbuild/Vitest. The isolated Go/API fixture remains available. A private
git-ignored archive preserves the dirty Nuxt source before removal at
`.data/source-archives/nuxt-20261006.tar.gz`; committed originals also
remain in Git history.

Server fitting calculations now run the original Rust v7.1.0 engine natively
through persistent JSON-line workers. Vendored calculation source matches the
upstream tag unchanged. Browser WASM/SDE versions remain unchanged. All 16
reference fits match the previous runtime across all 49 API fields, within
relative 1e-9/absolute 1e-7 tolerance, on macOS and Linux ARM64/AMD64. Protocol,
cancellation, restart and invalid-path regressions pass under the Go race detector.

Cleanup checks passed: full Go tests and vet; template formatting/generation;
179 retained browser/WASM tests; TypeScript; matching OpenAPI generation;
reproducible browser bundle/data regeneration; dependency audit; Go/frontend/
fixture race tests; Linux runtime build and version command. The runtime image
contains Go and native Rust, with neither Bun nor Node, and retains UID/GID 1000.
The read-only HTTP audit passed 314 routes and 97 assets with no failures.
Browser checks covered the retained fit, fullscreen ship/table views, mobile
width containment, live skill-profile recalculation, and populated advanced
search filtering. These are cleanup regression checks; the broader completed
page/interaction comparisons and their limitations are documented above.

## Shared native/WASM dogma upgrade, 2026-10-06

Upgraded both calculation targets to upstream 13.6.0 and the matching
12.3579973.0 FlatBuffer. The audit and retained/retired extensions are documented
in `docs/architecture/dogma-engine.md`. A private source archive preserves the
old 7.1 implementation before replacing it.

Validation completed:

- 189 browser/helper/WASM tests across 59 files, TypeScript and WASM source/artifact
  checksums pass. This includes native/browser agreement across 16 reference fits
  and all 49 projected API fields, independent WASM factories, ammunition/script
  behavior, ancillary charges, remote totals, rigs and subsystems.
- Full Go tests and vet pass. Dogma protocol/cancellation/recovery, the new Go
  converter and frontend tests pass under the race detector. Final CLI/converter
  checks cover the custom output path without changing the repository manifest.
- The Go generator rebuilt CCP SDE build 3579973 in 4.481 seconds after download.
  Its `sde.dat` matches the pinned upstream release byte for byte; all 16 native
  reference calculations match the current golden results on this generated file.
- Linux ARM64 runtime and AMD64 native worker builds pass. Both workers match
  all 16 fits/49 fields within relative 1e-9 / absolute 1e-7. The runtime runs as
  UID/GID 1000, includes the data/licenses, and contains neither Bun nor Node.
- Real Chrome on the updated 4200 preview loads the complex Abaddon fit, recalculates
  all-V/all-III profiles, applies Scorch L across the high rack, restores the prior
  statistics with Undo, and switches between ship/table views. The editor has no
  captured console errors or warnings. A 390x844 viewport contains the document
  without horizontal overflow. The viewport override was reset after checking.
- HTTP checks confirm the preview serves the exact committed FlatBuffer and WASM
  bytes with the expected content types. Rust/template formatting and CI YAML
  checks pass. Focused Go lint passes; upstream lint exclusions are scoped and
  documented in the converter provenance. These upgrade checks supplement the earlier full route/parity
  audit; they do not repeat or expand that audit's coverage claims.

Evidence: `/tmp/shrike-dogma-upgrade-browser-tests.log`,
`/tmp/shrike-dogma-upgrade-typecheck.log`,
`/tmp/shrike-dogma-upgrade-go-tests.log`, `/tmp/shrike-dogma-upgrade-vet.log`,
`/tmp/shrike-dogma-upgrade-race.log`,
`/tmp/shrike-dogma-upgrade-final-cli-tests.log`,
`/tmp/shrike-dogma-upgrade-final-wrapper-tests.log`,
`/tmp/shrike-dogma-upgrade-lint.log`,
`/tmp/shrike-dogma-upgrade-data-build.log`,
`/tmp/shrike-dogma-upgrade-docker-final.log`,
`/tmp/shrike-dogma-upgrade-amd64.log`,
`/tmp/shrike-dogma-13.6-arm64.jsonl`, `/tmp/shrike-dogma-13.6-amd64.jsonl`,
`/tmp/shrike-dogma-13.6-desktop.png`, `/tmp/shrike-dogma-13.6-mobile.png`.

The local preview uses the upgraded build. No production deployment or fitting
statistics mutation was performed. Production catalogue metric filters and
histograms require a backfill/distribution rebuild for 13.6.0+ek.1 / 3579973
when the upgrade is deployed; old statistics retain their original version tags.

## Post-migration cleanup audit, 2026-10-07

Consolidated editor document conversion in `frontend/src/fit/document.ts`.
Its generated `fit-document.js` uses the existing `toServer.ts` save serializer
instead of a second handwritten implementation. Regression tests assert the
actual saved API rows and round-trip slots, states, charges, grouped drones,
cargo and visibility; they no longer compare two copies of the serializer.
The bundle retains its public module URL and exports.

Batched dogma stat reads now resolve the cached metadata once and read all
requested fields synchronously, avoiding per-field promises. The native/WASM
calculation adapter and committed WASM are unchanged. Replaced the stale
Protobuf data README and corrected old source-path comments.

Validation: all 189 browser/WASM/helper tests across 59 files, TypeScript,
frontend Go tests, Go build, WASM source/artifact checksums and `git diff --check`
pass. The updated 4200 preview returns HTTP 200 for the editor and serves the
exact generated fitting document bundle. Logs:
`/tmp/shrike-cleanup-20261007-browser-tests.log`,
`/tmp/shrike-cleanup-20261007-typecheck.log`,
`/tmp/shrike-cleanup-20261007-frontend-tests.log`,
`/tmp/shrike-cleanup-20261007-build.log`,
`/tmp/shrike-cleanup-20261007-assets.log`.

The remaining architectural priority is replacing eleven in-process HTTP handler
dispatch sites across ten frontend files with typed services shared by pages and
public APIs. This should be done by page family, preserving authorization, domain
scoping, validation and caching. Other useful follow-ups are consolidating browser
controller sources/build watches and auditing compatibility aliases against
remaining browser, MCP and external consumers. Neither API removal nor a broad
service refactor is part of this small cleanup pass.


## Coalition activity and browser build/controller consolidation, 2026-10-07

The coalition directory renders membership and navigation separately from the
30-day kill/loss/value/efficiency aggregation. Its combat fragment preserves
cards, filtering and focus, and exposes loading/failure/retry state. The existing
public directory API retains complete totals. A cached typed overview avoids
killmail/attacker scans; a membership-count query optimization preserves exact
member/alliance/system counts. Controlled uncached reads measured 736ms overview
versus 2.14s complete directory; an early HTTP cold page was 648ms. A later live
cold read was about 3.28s, so these are observations, not a fixed latency promise.
Membership reads can still make a cold overview slower; warm responses do not
prove cold performance. Activity no longer blocks document rendering.

The local PostgreSQL integration fixture verifies membership counts, deleted
characters, systems, multi-alliance kill deduplication and the 30-day cutoff.
Frontend tests distinguish overview/fragment failures. Delayed/failed browser
checks preserve filtering while totals arrive; Retry works without rebuilding
cards. The mobile check at 390x844 (379px document) has no horizontal overflow.

All authored browser controllers now live under frontend/src/browser and compile
through one TypeScript/esbuild graph. The common site.js entry combines the shell,
account, announcements, submission, navigation feedback and home behavior;
page-specific entry points retain feature loading. Shared chunks carry common
helpers/singletons. Sixteen duplicated HTML escape implementations and duplicate
fuzzy-search/theme-map definitions now share typed implementations. Roam
aggregation and fitting document interchange use typed canonical models.

Stimulus 3.2.2 now supplies actions/targets/lifecycle for select menus, categories,
legacy attackers and coalition activity. One application is shared across page
bundles. Tests cover keyboard selection, outside click, focus, dynamic fragments,
reconnection without duplicate listeners, async cancellation and obsolete
responses. These are initial framework conversions, not a claim that every
controller has been rewritten. Strict TypeScript includes noImplicitAny; 87
legacy JavaScript controllers remain explicitly listed in browser-legacy.json.
New JavaScript controllers and TypeScript suppression directives fail the source
policy test. Full conversion remains open in TODO.md.

make dev starts the browser watcher and Air together. Generated chunk writes
signal one completed graph through a marker, preventing overlapping Go rebuilds.
Real Chrome observed automatic page reload after two source edits/restoration;
they took 29s and 19s and caused one Go rebuild per save. There is no element HMR,
and embedding browser assets means Go still recompiles for browser-only edits.
Temporary source exports were removed and all dev watcher/Air children stopped.

Production output is minified, reproducible and contains no source maps. All
document scripts use ESM except the synchronous appearance bootstrap. The final
graph is 934,668 JavaScript bytes across 118 public entries and 76 shared chunks,
about 25% below the original 1,250,543 bytes. The static common-shell dependency
graph is 61,890 bytes/11 requests; shell plus fitting editor is 202,137 bytes/23
requests. These are uncompressed JS totals excluding dynamic modules, CSS, data
and WASM. Docker builds browser assets in its Node stage and embeds them in Go;
its final runtime is UID 1000 with no Node/npm/Bun executables.

Validation: 205 browser/WASM tests in 64 files, strict TypeScript, make build,
full Go tests/vet, frontend race checks, focused lint (0 issues), Rust/WASM source
checksums, production bundle reproducibility and Docker build pass. The read-only
HTTP inventory checks 315 route samples and 92 directly referenced assets with
zero HTTP/asset failures; nine dynamic route patterns require IDs and remain
outside that inventory, as documented in its report. This inventory supplements
rather than replaces the earlier functional route audit.

Chrome checks on the minified build cover fitting import, all-III/all-V
recalculation, ship/table/fullscreen rendering and fuzzy search; shared search;
category filtering; desktop/mobile admin menu selection and keyboard behavior;
coalition filtering/activity; and lazy map initialization with Zarzakh scope.
No captured console errors/warnings were present on these checks. Expired
sessions were refreshed only in the owned loopback fixture database; the
retained 4450 fixture API and its seeded data were not restarted. The 4200 preview
and 4451 fixture facade now use the updated production bundles.

Evidence: /tmp/shrike-coalition-20261007-integration.log,
/tmp/shrike-coalition-20261007-race.log,
/tmp/shrike-coalition-20261007-live-query-times-optimized.log,
/tmp/shrike-coalitions-activity-pending.png,
/tmp/shrike-browser-dev-smoke-final.log,
/tmp/shrike-browser-final-tests.log, /tmp/shrike-browser-final-typecheck.log,
/tmp/shrike-browser-production-go-build.log,
/tmp/shrike-browser-final-go.log, /tmp/shrike-browser-final-vet.log,
/tmp/shrike-browser-final-race.log, /tmp/shrike-browser-final-lint.log,
/tmp/shrike-browser-final-docker.log, /tmp/shrike-browser-repro-build.log,
/tmp/shrike-browser-final-http.json,
/tmp/shrike-browser-coalitions-mobile.png.


## Active TypeScript goal: shared shell/account conversion checkpoint

Seven original JS modules have been converted to strict TS: shell, account,
notifications, tracker-state, reload, reload-channel and reload-worker. Legacy
inventory now contains 80 modules. Shell/account/notification DOM bindings use
Stimulus; shared state and transport remain ordinary typed modules. The generated
auth contract replaces the loose browser user shape, tracker types now derive
from generated responses, and the response schemas describe the already-present
reply target_slug and database-enforced tracker/role enums. Wire payloads and
production data are unchanged. API/types regenerated with make gen-api-client.

Tests cover shared auth, both login preference forms, shortcuts/mobile/background
inert state, reconnection, search cancellation and stale responses, kill navigation
sequence capture, retained notification focus, account-change cancellation,
tracker errors, relay restoration/logout and malformed envelopes. A retained-row
focus failure was fixed with feature-detected moveBefore for retained siblings
and focus restoration on insertion fallback. Secure wss inputs stay secure.
Tracker reads abort on account changes and preserve independent resource errors.

215 tests in 66 files and strict TS pass. Frontend/API Go tests and focused lint
pass. Browser checks confirm populated account/avatar/entity/balance fields, all
five reply destinations alongside tracker alerts, shared search, Escape, and
mobile focus/menu containment. However, repeated balance refresh in Chrome showed
a failure/pending state despite valid fixture API data. This issue is not yet
resolved and this family is not considered fully validated.

Current review processes: owned preview 4205 PID 49697; traced fixture facade
4451 PID 55824, backend retained at 4450 PID 59303. The 4200 preview PID 8086 still
has the previous accepted build. Do not restart the retained 4450 fixture API.
A temporary balance console warning was removed from source but is embedded in
/tmp/shrike-ts-shell-facade-trace for diagnosis. Optional fixture request tracing
uses SHRIKE_FIXTURE_TRACE=1 and logs only method/path/start/end. The trace showed
auth/me, announcements, ws/announcements and __dev/reload but did not receive
subsequent tracker/balance requests after one refresh; investigate request/stream
queuing and component lifecycle rather than assuming the backend balance failed.
Browser binding typedShellCheck is on 4451/settings; temporary viewport was reset.

Evidence: /tmp/shrike-ts-shell-all-tests.log,
/tmp/shrike-ts-shell-controller-tests.log, /tmp/shrike-ts-shell-typecheck.log,
/tmp/shrike-ts-shell-go-tests.log, /tmp/shrike-ts-shell-lint.log,
/tmp/shrike-ts-shell-api-generation.log, /tmp/shrike-ts-shell-facade-trace.log,
/tmp/shrike-ts-shell-mobile.png. Rebuild production assets after diagnostic source
changes before artifact checks. Full goal and full-family validation remain open.


## Shared TypeScript components and picker checkpoint, 2026-10-07

Eleven of the original 87 JS modules are now checked TypeScript; 76 remain in
frontend/browser-legacy.json. Added navigation-progress, diagnostics, announcements
and search-picker to the previous seven. All four use Stimulus for event bindings
and disconnect cleanup. Existing rendering strings, HTML/CSS, generated API wire
contracts, per-page bundles and Go/templ rendering are preserved.

Navigation feedback releases its observer/native-navigation subscriptions and
loading tasks on disconnect. Tests cover native programmatic rows, canceled
fallback clicks, modifier/anchor exclusions, keyboard rows, hidden fragments and
removal. In Chrome, an injected 2.5s /_fragments/campaigns delay showed Loading…,
the top progress bar and navigation-pending on the clicked Conflicts button; all
cleared on completion. The delay rule was cleared afterward.

Diagnostics reads native timing entries, distinguishes counts from durations,
keeps request paths without queries and releases its observer/load timer. The
browser panel populated server parts, browser timing and a measured resource
request. Announcements retain ticker colors/markup/animation, mandatory modal
acknowledgment, local dismissal/collapse state and snapshot/live reconciliation.
Incoming relay data is validated before rendering; archived/expired events use
an ID-only typed shape. Disconnect and visibility suspension cancel stale reads.
Disposable relay checks confirmed live ticker insertion, collapse/expand and
modal Escape prevention plus acknowledgment. Temporary ticker was archived.

Search-picker keeps the parent initialization/update/close APIs while Stimulus
connects inserted markup and removes listeners on detach. Parent options set
before connection remain effective. Hide/outside input/disconnect abort searches
and ignore late results. The picker preserves its original placeholder on
reconnection. Chrome /trackers showed the owner result disabled, ArrowDown moved
to Fixture Reader, Escape closed results and restored input focus. At 390x844,
results ended at x358 within the 379px content viewport; no horizontal overflow.
No tracker mutation was needed for these checks.

The earlier account balance stall did not recur in a fresh tab: repeated desktop
and mobile menu opens and a reload showed 96k ISK, and traced requests reached and
completed in the fixture API. No cause was proven; retain repeated balance/relay
checks in the final whole-site audit rather than claiming a diagnosed fix. The
temporary catch warning is absent from the rebuilt preview/facade.

Validation: 227 browser/helper/WASM tests in 69 files; strict TS; Go frontend and
frontend-parity tests; production asset build and template generation. After the
last two small fixes, the 10 picker/progress/diagnostics/artifact tests and strict
TS passed again. Production graph: 118 entries, 78 shared chunks, 944508 total JS
bytes. This is site-wide output, not a per-page transfer measurement.

Current review: /tmp/shrike-ts-picker-preview on 4205 and
/tmp/shrike-ts-picker-facade on 4451, with optional method/path-only tracing.
The retained fixture backend on 4450 was not restarted. 4200 retains the previous
accepted build; user 4000 was untouched. No Docker rebuild or final whole-site
audit was claimed for this batch, and no production mutation/deployment occurred.
Next conversion work: tracker pages/dashboard/private killfeed, including their
remaining row-renderer dependency, then settings and entity/tool families.

Evidence: /tmp/shrike-ts-picker-all-tests.log,
/tmp/shrike-ts-picker-final-tests.log, /tmp/shrike-ts-picker-typecheck.log,
/tmp/shrike-ts-picker-go-tests.log, /tmp/shrike-ts-picker-build.log,
/tmp/shrike-ts-picker-templ.log, /tmp/shrike-ts-picker-facade.log,
/tmp/shrike-ts-global-diagnostics.png, /tmp/shrike-ts-global-mobile.png,
/tmp/shrike-ts-global-fragment-loading.png, /tmp/shrike-ts-picker-trackers.png,
/tmp/shrike-ts-picker-mobile.png.


## Typed tracker family checkpoint, 2026-10-07

Converted kill-row, private-killfeed, trackers and tracker-dashboard to checked
TS. Inventory now has 72 JS modules remaining, 15 of the initial 87 converted.
The row renderer uses the generated REST KilllistRow contract; unknown relay
frames are normalized into its checked fields before entering rendering.
Item-only final blows retain item links/icons, fixed portrait geometry and the
users involvement icon. Malformed dates/IDs never become live rows.

Stimulus owns private feed, tracker page and dashboard event/subscription
lifecycles. Parent initialize/refresh/destroy feed APIs remain available.
Superseded requests cannot replace newer content; detach aborts requests and
releases sockets/subscriptions. BFCache restoration retries interrupted feeds.
Embedded tracker/dashboard seeds are validated; generated tracker target enums
and canonical dashboard widget types constrain mutations. Shared helpers handle
account errors/login destination, template cloning, option creation and retained
row moves. Go markup, CSS, public API wire payloads and server rendering are
preserved. No API schema/engine changes were needed for this family.

Tests use markup extracted from the disposable local Go fixture and verify
tracker toggles/duplicate rejection/session expiry, dashboard widget moves,
text/enabled/kill-type saves, unsubscribe/cancellation and stale fragments.
Private-feed tests cover cursors, queued arrivals on later pages, loss coloring,
newer-ID-only insertion, duplicate/malformed frames, retry and BFCache.

Chrome on 4451: selected the owner tracker, changed page size, injected a live
fixture kill and saw its loss color, item-only final-blow icon and 3 involved.
Forced the disposable relay to disconnect; the live dot reconnected. Created
the owned fixture user's dashboard with a text widget moved to the top and
50/50 columns; reload retained both, and Cancel discarded an unsaved note.
The configured dashboard rendered its top lists, valuable kills and killlist.
A 503 fault on /_fragments/tracker/killmails displayed its Retry control and
released aria-busy; clearing the rule and retrying restored rows. Fault rule
was cleared. Desktop/390px checks had no console errors or horizontal overflow
(379px document width/scrollWidth). Fixture account settings were changed only
in the retained disposable database; production received no mutations.

Validation: 237 browser/helper/WASM tests in 71 files, strict TS, frontend/parity
Go tests, templates and production asset build. Browser graph: 118 entries,
79 chunks, 950365 total JS bytes. Full-site/Go/Docker/reproducibility audit still
remains at the end of the complete 87-module migration.

Current processes: owned /tmp/shrike-ts-tracker-preview on 4205 PID 70499;
/tmp/shrike-ts-tracker-facade on 4451 PID 70501; retained 4450 fixture backend
PID 59303 was not restarted. 4200 still holds the earlier accepted preview and
user 4000 was untouched. New Chrome task trackerVerify is ephemeral; temporary
viewport was reset. Next source family: account/settings tools and entity picker.

Evidence: /tmp/shrike-ts-tracker-typecheck.log,
/tmp/shrike-ts-tracker-controller-tests.log, /tmp/shrike-ts-tracker-all-tests.log,
/tmp/shrike-ts-tracker-go-tests.log, /tmp/shrike-ts-tracker-build.log,
/tmp/shrike-ts-tracker-templ.log, /tmp/shrike-ts-tracker-facade.log,
/tmp/shrike-ts-tracker-live-desktop.png, /tmp/shrike-ts-tracker-mobile.png,
/tmp/shrike-ts-dashboard-desktop.png, /tmp/shrike-ts-dashboard-mobile.png.


## Account tools TypeScript checkpoint, 2026-10-07

Converted entity-search-select, esi and wallet: 18 of the original 87 modules
are now checked TS, 69 remain. Settings and domain-editor parent modules still
need conversion. Stimulus now owns entity picker, shared admin/account ESI log
and public wallet bindings/cleanup. Their legacy factory APIs remain available
for the unconverted parent modules. Reconnection avoids duplicated ESI menus.
The nonbubbling settings/admin section visibility events retain one explicit
section listener, removed on disconnect.

The ESI generated response schema now documents six already-returned optional
metadata fields: endpoint entity ID/name, requesting character ID/name, and
new/stored item ID arrays. API/types regenerated; no response payload/SQL changed.
EsiLogRow derives from that contract. Embedded log/chart seeds are checked before
use. Admin search uses a typed character/corporation picker. Manual ESI requests
set aria-busy for site-wide loading feedback; background polling stays quiet.
Hide/detach cancels requests/timers, and interrupted filters retry on restoration.
Wallet preserves date formatting, query/history behavior and layout while
aborting superseded/detached reads and avoiding duplicate error notices.

Eight new tests cover detached picker setup, selection/clear, cancellation and
late responses; wallet errors/history safety; ESI metadata/pending IDs, retry,
section restoration, silent polling/page pauses and 24-hour chart geometry.
All 245 browser/helper/WASM tests in 72 files and strict TS pass. Go API/frontend/
parity tests pass; templates and production assets regenerated. Production graph
remains 118 entries/79 chunks, now 955833 site-wide JS bytes.

Chrome: admin request 74 showed entity/requesting pilot, duration, source, stored
killmail 2000000001 link and pending 2199999999. Entity/status filtering yielded
six error rows. Account log paged to 2/2 and paused polling; switching Overview
and back retained that state and exactly three filter menus. A local 503 fault
produced Retry and cleared aria-busy; after clearing it, Retry loaded page 2/2.
The public wallet's Division 2 query produced the empty journal; Back restored
All divisions and the two imported entries. Desktop/390px checks passed without
console errors or outer horizontal overflow; wallet table uses its existing
inner horizontal scroll.

Use distinct origins for fixture and public checks: account-review.localhost:4451
for owned account tests; localhost:4205 for public reads. Ports do not isolate
cookies. During this pass, opening public auth/me on 127.0.0.1:4205 with the
fixture cookie cleared the invalid cookie, so the existing fixture tab later
reported Authentication required. Fresh-cookie API read returned 200 and the
separate-host retry passed. authService.resolvePrincipal clears invalid cookies.
This was a comparison-session issue, not a diagnosed ESI request failure.
Refresh the viewport capability for a new browser task and verify actual DOM
width before accepting screenshots; an old handle did not resize the new tab.
The final wallet phone screenshot was verified at 390px. Overrides were reset.

Current owned preview /tmp/shrike-ts-account-preview on 4205 PID 78215 and facade
/tmp/shrike-ts-account-facade on 4451 PID 78216. Retained fixture backend on 4450
was not restarted. 4200 retains an older accepted build and user 4000 was untouched.
No production mutations/deployment occurred. Fault rule was cleared.
Next: settings/preferences/appearance/sessions/descriptions/comments and domains.
Full-site, Go/Docker and reproducibility audits remain required after all modules
are converted.

Evidence: /tmp/shrike-ts-account-api-generation.log,
/tmp/shrike-ts-account-typecheck.log, /tmp/shrike-ts-account-controller-tests.log,
/tmp/shrike-ts-account-all-tests.log, /tmp/shrike-ts-account-go-tests.log,
/tmp/shrike-ts-account-build.log, /tmp/shrike-ts-account-templ.log,
/tmp/shrike-ts-esi-detail-desktop.png, /tmp/shrike-ts-esi-admin-mobile.png,
/tmp/shrike-ts-esi-account-desktop.png, /tmp/shrike-ts-esi-account-mobile.png,
/tmp/shrike-ts-wallet-mobile.png.


## Settings TypeScript checkpoint, 2026-10-07

Accepted settings.ts and six scoped Stimulus controllers: preferences, appearance,
sessions, bio editors, private account wallet and comments. 19/87 original source
modules are converted, 68 remain. Domain settings are still legacy JavaScript.
The shared settings component owns JSON/HTML requests through body consumption,
loading state, abort cleanup and transient timers. Draft panels remain cached in
DOM; root tab navigation aborts superseded fragments and preserves history/Back.
Theme snapshots survive reconnects, and saves acknowledge the submitted values
without discarding newer local edits. Comments release disabled controls when a
pagehide cancels a load, preserve their error SVG and maintain pagination metadata.

Corrected the generated user-preferences-update-compat response schema to describe
its existing defaultTabs response rather than the account-preferences wrapper.
Wire payload and SQL behavior are unchanged. Regenerated API types/templates/assets.
All 261 tests in 73 files and strict TS pass; 16 new Go-markup-based regression
tests cover keyboard/outside dismissal, cookies, timezone events, drafts/Back,
fragment supersession/retry, save races and detach, session confirmations/errors,
copy feedback, wallet failure/retry and comments cursor/dedupe/delete/error cleanup.
Go API/frontend/parity tests pass. Production assets: 118 entries, 80 shared chunks,
960498 total site-wide JS bytes; utility CSS remains 119790 bytes.

Chrome on account-review.localhost:4451: saved Character default tab as Losses;
previewed/saved Gallente Green; bio draft survived switching to Appearance and
Back, then submitted in the named disposable account. Sessions rendered current
Chrome/macOS plus two older devices. A local 503 wallet fragment fault showed the
error, cleared aria-busy and restored Next; clearing it allowed Page 2. Comments
Load more produced 37 unique cards and End of list. At 390x844, preferences dropdown
and all six panels fit without outer horizontal overflow. Console warnings/errors
were empty. Phone screenshots visually checked; viewport reset and review tab closed.
No production mutations. Fixture fault rule cleared; retained backend 4450 untouched.

Owned current processes: /tmp/shrike-ts-settings-preview on 4205 PID 97065 and
/tmp/shrike-ts-settings-facade on 4451 PID 97067. 4200 remains an older preview;
user 4000 untouched. Named account tests use account-review.localhost, public
reads use localhost:4205, since ports alone do not isolate cookies.
Next: settings-domains.js and its domain editor types. Final full-site, Go/Docker
and bundle reproducibility audit remains required after all original modules.

Evidence: /tmp/shrike-ts-settings-api-generation.log,
/tmp/shrike-ts-settings-typecheck.log, /tmp/shrike-ts-settings-controller-tests.log,
/tmp/shrike-ts-settings-all-tests.log, /tmp/shrike-ts-settings-go-tests.log,
/tmp/shrike-ts-settings-build.log, /tmp/shrike-ts-settings-templ.log,
/tmp/shrike-ts-settings-facade.log, /tmp/shrike-ts-settings-preferences-mobile.png,
/tmp/shrike-ts-settings-appearance-mobile.png.


## Domain editor TypeScript checkpoint, 2026-10-07

Accepted settings-domains.ts: 20/87 original authored JS modules are now converted,
67 remain. Settings root imports the domain controller instead of calling a manual
initializer. Stimulus owns click/input/change/drop/page lifecycle bindings, while
one explicit section visibility listener is removed on disconnect (the existing
settings-visibility event does not bubble). Requests, including multipart uploads
through JSON body consumption, use the shared settings owner. Search debounce and
transient feedback stop on hide/detach; retained drafts survive reconnection.
Interrupted subdomain checks resume after a persisted pageshow.

Replaced all explicit any types in domainEditor.ts with generated-model-derived
editable fields, snapshots and patches. Domain seeds validate entity IDs, theme
fields, navigation, widget types, campaigns and assets. Widget campaign IDs/titles
and older theme keys remain preserved. Numeric entity IDs now work alongside the
legacy prefixed search form; the old implementation produced NaN for numeric IDs.
The domain widget predicate includes campaigns (the tracker catalog deliberately
does not). Saves still rebase against submitted values while newer edits stay
local; managed image fields and paired campaign audience IDs retain their behavior.

All 269 browser/helper/WASM tests in 74 files and strict TypeScript pass. Eight new
regressions cover seeds/numeric IDs, new creation, save races, nested menus/widgets,
paired campaign audiences, disconnect/stale searches, persisted-page restoration
and multipart/drop cancellation/duplicate protection. happy-dom aliases
PageTransitionEvent to Event, so its persisted field is provided in that test;
drop tests likewise supply the missing dataTransfer property. Existing settings,
JSON transport and domain-patch tests pass. Go API/frontend/parity checks pass;
templates and production assets rebuilt: 118 entries, 82 shared chunks, 966623
site-wide JS bytes and 119790 utility CSS bytes.

Chrome at account-review.localhost:4451: edited owned fixture domain 9104. Entity
search added Fixture Red; a site-name draft survived Preferences and Back including
?edit=9104, then saved. Added/saved a text widget and nested Fixture menu -> Roam
report link. Added a private fixture campaign; Members/Everyone toggled correctly
and the saved audience remained Members. Uploaded /tmp/shrike-ts-domain-upload.png
(128x64 generated test PNG) through the native Chrome chooser: Backgrounds showed
1/8 and Pending. The extension setFiles path lacked Allow access to file URLs;
its documented setting was reported, but the native picker worked without any
permission change. No production data was mutated and no external domain was opened.

All five editor panes at 390x844 fit without outer horizontal overflow and console
warnings/errors were empty. A full reload confirmed the saved site name, text widget
and nested menu URL. Screenshot sizing was wrong immediately after reload with the
old viewport override; a fresh tab/capability set after navigation produced a
visually verified 390px phone screenshot. Viewport reset and temporary tabs closed.
The earlier balance stall remains unproven/on the final whole-site audit checklist.

Current owned preview: /tmp/shrike-ts-domains-preview, 4205 PID 4376; fixture facade:
/tmp/shrike-ts-domains-facade, 4451 PID 4378. Retained fixture backend 4450 PID 59303
was not restarted. User 4000 and older preview 4200 untouched. Local fault rule is
clear. Next: admin page/controller family. Goal remains active; removal of allowJs,
remaining 67 modules and final all-site/Go/Docker/reproducibility audits are pending.

Evidence: /tmp/shrike-ts-domains-typecheck.log,
/tmp/shrike-ts-domains-controller-tests.log, /tmp/shrike-ts-domains-all-tests.log,
/tmp/shrike-ts-domains-go-tests.log, /tmp/shrike-ts-domains-build.log,
/tmp/shrike-ts-domains-templ.log, /tmp/shrike-ts-domains-facade.log,
/tmp/shrike-ts-domains-upload-desktop.png, /tmp/shrike-ts-domains-navigation-mobile.png.


## Admin navigation and review TypeScript checkpoint, 2026-10-07

Accepted admin.ts and admin-review.ts: 22/87 original modules converted, 65 remain.
Admin navigation, user directory, user-detail forms, ESI summaries and both review
queues now use checked TS/Stimulus. Remaining admin announcements/blog/campaigns/
domains/River/wallet modules still use their inventoried legacy factories. The
separate admin domain-detail page remains legacy until that module is converted.

Extracted RequestComponent from settings: owned JSON/HTML/multipart work covers
body reads, rejects detached/pre-aborted starts, normalizes canceled late failures,
clears loading and owns transient timers. Settings retains its compatibility alias.
FragmentTabsController shares settings/admin history, retained panel/draft caches,
supersession, wrong-section rejection, login gates and retry. Attention shortcuts
keep their original style. Admin 403 errors remain distinct from expired sessions.

User searches debounce/cancel/retry and retain query/input state. User actions use
operation tokens so an old finally cannot unlock newer work after reconnection;
unrelated Discord drafts survive refresh, and persisted pageshow reconciles an
interrupted action via GET without repeating the POST. Review actions validate
their queue/action combination, block duplicate submissions, preserve matching
media nodes, cancel on hide/detach, resume interrupted filters, and clear delete
confirmation when leaving the tab. Approval/hide/restore/delete API routes unchanged.

Corrected adminOverviewSchema to document the existing attention counters and
requests24h/errors24h. The handler does not return comments/moderation aggregates,
so those legacy schema properties are optional, while attention is required.
Generated API artifacts regenerated; no response payload or SQL behavior changed.

Chrome revealed an SSR registration-order problem in the first implementation:
the summary controller could connect before a custom parent scope was registered.
Replaced the scope registry with Stimulus outlets. The actual OutletConnected
callback supplies a typed parent (hasOutlet only proves a DOM match). A regression
unloads/registers the parent after the child mounts. The rebuilt browser now loads
summary data on initial SSR. HTTP trace shows one /api/admin/esi request per document;
a dev auto-reload produced a second document after restart, while cached ESI tab
navigation produced no further summary request. No repeated generated ESI controls.

All 285 browser/helper/WASM tests in 75 files and strict TypeScript pass. Sixteen
new admin tests cover summaries/cache/late-parent startup, tabs/Back/supersession,
search/error/cancellation, user form validation/tokens/drafts/read reconciliation,
review media/filter/actions/confirmation/visibility, and query-seed validation.
Fixtures retain real Go markup; iframe src is about:blank in offline unit fixtures
to avoid external page loads. Go API/frontend/parity tests pass, including template
controller mount checks. Templates/assets regenerated: 118 entries, 84 shared chunks,
969324 site-wide JS bytes, 119757 utility CSS bytes. Full final audits remain pending.

Chrome on account-review.localhost:4451: user filtering found one Fixture Reader;
Overview/Back retained the search and Enter on the row opened its detail. An invalid
Discord ID showed the server's 15-22 digit rule and restored controls; a dummy
numeric ID saved and was then cleared. Admin-role changes were exercised with
mocked requests/unit tests; no live role grant was clicked. Approved the disposable
TypeScript fixture bio: pending 9->8, bios 4->3, and Approved showed the saved body.
A local 403 moderation fragment fault showed Admin access required, cleared busy,
and successfully retried after clearing the rule. Hiding/restoring fixture comment
07 worked; restore clears its flag, so it leaves Flagged and appears in All active
as OK. Delete was armed once (not submitted), then leaving/Back disarmed it.

390x844 checks passed for user detail, user directory/Page 2 (six rows), overview,
moderation, comments and ESI: no outer horizontal overflow, no console warnings/errors.
Pointer hover on Users after Back looked selected in a screenshot, but DOM active
and aria-current correctly identified Comments. The moderation badge is an overview
snapshot: it changed from 9 to 8 when the overview fragment refreshed; preserve this
existing behavior for now and review whole-site count freshness during final audit.
Screenshots visually inspected; viewport reset and temporary admin tab closed.
No production mutations/deployment. All writes were named disposable fixture rows.

Current owned /tmp/shrike-ts-admin-preview: 4205 PID 19358; /tmp/shrike-ts-admin-facade:
4451 PID 19360. Retained 4450 fixture backend PID 59303 untouched. User 4000 and older
4200 preview untouched. Fault rule cleared. Next remaining admin editor modules;
goal active. allowJs/legacy exception removal and final all-site/Go/Docker/bundle
reproducibility audits remain required after the remaining 65 modules are converted.

Evidence: /tmp/shrike-ts-admin-api-generation.log,
/tmp/shrike-ts-admin-typecheck.log, /tmp/shrike-ts-admin-controller-tests.log,
/tmp/shrike-ts-admin-all-tests.log, /tmp/shrike-ts-admin-go-tests.log,
/tmp/shrike-ts-admin-build.log, /tmp/shrike-ts-admin-templ.log,
/tmp/shrike-ts-admin-facade.log, /tmp/shrike-ts-admin-user-mobile.png,
/tmp/shrike-ts-admin-users-mobile.png, /tmp/shrike-ts-admin-comments-mobile.png,
/tmp/shrike-ts-admin-moderation-mobile.png.


### Admin editors and date picker TypeScript checkpoint, 2026-10-07

Converted date-time-picker and all six remaining admin modules: announcements,
blog, campaigns, domains (directory/detail), River and private corporation wallet.
29/87 original modules are checked TypeScript; 58 handwritten browser JS modules
remain, exactly matching browser-legacy.json. Admin no longer imports legacy
factories or owns a manual child-disposal map. Shared AdminSectionComponent owns
visibility handling; RequestComponent owns requests through body reads and timers.
Typed generated contracts cover campaign/River/domain/wallet actions and editor
create responses; JSON seeds are narrowed and normalized at their boundary.

Editors retain newer input during a save; successful creation updates its ID so
another save updates the same record. Select controls are removed on disconnect.
Archive/delete/clear confirmations disarm on timeout and tab exit. Interrupting
mutations cancels local requests and reconciles via reads rather than replaying
writes. Domain refresh preserves preview nodes and newer rejection notes, and
rejects a fragment for a different domain. River retains structured arguments as
plain text, supports independent queue/job retries and scopes dialog cleanup.
Wallet reload restores focused controls, keeps one division menu, cancels delayed
sync reads when hidden, and suppresses duplicate settlement/authorization work.

DateTimePicker now uses Stimulus actions while retaining the compatibility entry
for callers awaiting conversion. Global zone preference preserves the UTC instant.
Calendar days/hour/minute updates bubble to owner forms. Non-bubbling dialog cancel
is captured so Escape also clears date popups. Chrome exposed an existing phone
issue: below-trigger calendars were clipped by scrolling modals. The reusable
floating-panel helper now flips upwards and caps/scrolls the panel when necessary.
A regression models clipping ancestors; actual 390px bounds were x37..341,
y226..654, fully inside the modal. Date selection and Done were exercised.

Validation: strict TypeScript; 308 browser/helper/WASM tests in 76 files, including
23 new editor/lifecycle tests; Go frontend/API/parity tests; Go page-controller
mount contracts; regenerated templ/assets and preview/facade builds all pass.
Production build: 118 entries, 89 shared chunks, 985262 total JS bytes, 119782 CSS
bytes. Offline HTML fixtures come from Go rendering. A synthetic claimed-wallet
state is used only inside its mocked request test; no live financial action ran.

Chrome on account-review.localhost:4451 verified required-title validation,
UTC/local display, Escape cleanup, fixture blog draft create/update (#48), tag
normalization, campaign search (Paused -> PausedCamp0001), domain filters/detail
and lazy file previews, River inspection/plain text and completed-state filtering,
wallet Page 2/403/error/retry/focus. 390x844 checks covered all six sections plus
standalone domain detail, with no outer horizontal overflow or console warnings/
errors. The rebuilt mobile calendar was visually inspected. No production writes,
role grants, job deletion, wallet authorization or settlement action ran. Blog
writes used the retained named disposable fixture only; fault rule is cleared.
Viewport reset and temporary browser tab closed. Screenshots:
/tmp/shrike-ts-admin-editors-calendar-mobile.png.

Owned preview /tmp/shrike-ts-admin-editors-preview runs on 4205; facade
/tmp/shrike-ts-admin-editors-facade on 4451. Retained fixture backend on 4450,
user 4000 and older 4200 preview untouched. Public reads use localhost:4205;
private fixtures use account-review.localhost:4451 so auth cookies do not mix.
Logs: /tmp/shrike-ts-admin-editors-{typecheck,all-tests,go-tests,templ,build,
facade,preview,rebuild}.log. Goal remains active; advanced search is next.
allowJs removal and full-site/build/Docker/reproducibility audits remain pending.


### Advanced search and saved searches TypeScript checkpoint, 2026-10-07

Converted advanced-search-state, saved-searches and advanced-search. Accepted
progress is 32/87 original modules, with 55 handwritten JavaScript modules remaining.
The filter helpers moved into canonical utils/advancedSearch.ts behind a browser
entry. They narrow unknown SSR/link data into explicit entity/item/location/time/
option models, retain source chip order and readable links, and serialize the
existing API fields. SavedSearchDocument now uses the generated API type rather
than Record<string,any>. Browser storage is validated; shorthand ISK, legacy
items, explicit defaults, display names and fitting grouping/drilldowns survive.
Null preset is stored as the equivalent empty API timeRange object.

Advanced search preserves the original template-driven tag/suggestion rendering,
filter controls, 300ms lookup and 500ms numeric debounce, fitting drilldowns,
pagination/cursors and Back. Typed UI functions retain state; Stimulus owns DOM/
window actions and cleanup, with no per-row listeners. Shared RequestComponent
owns network reads through response bodies and timers. Reconnection/BFCache reads
reconcile interrupted results; stale responses cannot replace newer results.
Changing either custom date endpoint applies the range (the old source only reacted
to To). Mixed normal/group lookup results deduplicate entity IDs; Chrome exposed
this with the fixture's overlapping responses.

SavedSearchesController handles popover/outside/Escape/layout, browser/account
save, revision-aware rename/replace/import and explicit deletion confirmation.
It uses scoped current/load events to ask the advanced-search owner for state;
no manual initialization adapter or parent registry remains. A newer name entered
during a save survives and uses the returned ID/revision on its next save.
Identity changes/disconnect cancel scoped requests; interrupted writes are not
replayed. Failed refreshes clear on success. Browser imports retain their copy.

Strict TypeScript, all 326 browser/helper/WASM tests in 78 files, Go frontend/API/
parity tests, templ generation and preview/facade builds pass. Eighteen new tests
cover state boundaries/chips/dedup, saved-search lifecycle/revisions/storage/errors,
filters/links/date changes/Back/fitting/supersession and reconnect. Asset generation
and Go embed checks must run sequentially: an overlapping first Go run caught
missing chunks while the builder replaced them. The final Go checks ran after
asset generation and passed. Production assets: 118 entries, 90 shared chunks,
992726 total JS bytes, 119782 CSS bytes. Browser legacy list matches source files.

The local comparison facade lacked AdvancedSearchReader. Added a read-only remote
adapter to the retained fixture API for labels and killlist/advanced. Its test checks
GET-only requests, no session transmission, query controls and cursor decoding.
Retained backend/database on 4450 was not restarted. This changes test tooling only.
Offline root HTML originated from Go rendering; lifecycle attrs match templ and
Go controller-mount assertions. Browser tests used account-review.localhost:4451.

Chrome desktop/390px: initial SSR performed no unfiltered results query; selected
Fixture Owner as victim, saved and renamed an account search, cleared/loaded it,
and Back restored the empty state. Lowsec/fitting grouping and empty results worked.
A 503 fragment fault showed the source failure feedback and released aria-busy;
the rule was cleared and a reload recovered. The rebuilt lookup displayed one
Fixture Owner suggestion. Anonymous browser save stored canonical 500m filters;
logging back into the fixture owner and importing saved the document and kept the
browser copy. Account save/rename/import were writes to the named disposable
fixture only. No live deletion, production write, job action or financial action.
Both document and popover had no outer horizontal overflow at 390px. Final popover
bounds x16..374, y365..785; screenshot visually inspected. No console warnings/errors.
Viewport reset and temporary tab closed. Screenshot:
/tmp/shrike-ts-advanced-saved-mobile.png.

Current owned binaries: /tmp/shrike-ts-advanced-preview on 4205 and
/tmp/shrike-ts-advanced-facade on 4451. Retained 4450, user 4000 and older 4200
untouched. Fault rule cleared. Logs: /tmp/shrike-ts-advanced-{typecheck,all-tests,
controller-tests,go-tests,templ,build,facade-build,facade,preview}.log and
/tmp/shrike-ts-saved-controller-tests.log. Goal remains active; comment components
are next. allowJs/legacy exceptions and final whole-site/build/Docker audits remain.


### Comment/media TypeScript checkpoint, 2026-10-07

Converted comment-stream, klipy-picker, comment-input, comments-feed and
comment-thread. Accepted progress is 37/87 original browser modules, with 50
handwritten JavaScript files remaining, matching browser-legacy.json. All five
compile as strict TypeScript without suppression/any casts. CommentInput and
thread rendering retain the original markup/classes. Stimulus owns actions,
request lifetime, timers, dialog cleanup and page restoration. The thread's
strong document registry and MutationObserver cleanup are removed; compatibility
initializeCommentThreads only adds declarative attributes for unconverted callers.
Go kill/blog/battle/campaign/fitting templates declare the new thread scopes.

The comment stream reuses relaySocketCore rather than a second reconnect transport.
One socket serves all mounted threads; the last subscriber disposes it. Unknown
frames are narrowed, minimal production events merge into typed models and keep
cached Markdown. Editing a live-only card still fetches authoritative Markdown.
Reconnect refreshes recent roots, reconciles missed deletions in the proven query
range, and preserves pagination rather than requesting another page automatically.
Scoping retains target type/id/slug/domain. Deep replies, removed descendants,
source ordering, permissions, hash scrolling and live indicators are retained.

Editor requests retain newer text entered during posting, including inline replies.
Selection/focus survive live tree updates. Formatting, shortcuts, write/preview tabs,
validation, embed URLs/thumbnails and server Markdown rendering retain their source
behavior. Dialogs share reference-counted scroll locking; all close paths release
ownership and abort requests. GIF insertion closes its modal before focusing the
editor. Media/favorite JSON is narrowed; unsafe URLs are skipped. Klipy retains
24-item pages, 300ms debounce, provider errors, favorites/storage sync and keyboard
insertion. Thread reports use generated enum contracts and a scoped dialog controller.

Feed sorting uses placeElement for retained cards. Superseded reads cannot replace
current rows; failures retain displayed cards. Internal HTML fragments now carry
an explicit data-comments-fragment boundary. Existing public JSON API contracts,
queries and SQL are unchanged. Source/API comment text search uses pg_trgm body_md
similarity, not substring matching: TypeScript returned two of the three named
root comments, and the browser matched the API exactly.

Strict TypeScript, all 351 browser/helper/WASM tests in 82 files, Go frontend/API/
parity tests, templ generation, assets and preview/facade builds pass. Twenty-five
new tests cover composer/media/stream/thread/feed lifecycle and reconnection. The
initial two failing composer assertions copied a getter into a static value in the
test helper; keeping the actual handle resolved them, with implementation unchanged.
Production assets: 118 entries, 91 shared chunks, 1006215 total JS bytes and 119988 CSS
bytes. Asset generation completed before the final Go embed checks/builds.

Added media-fixtures support to the read-only comparison facade, reusing the
existing deterministic provider/relay. Its fake GIF responses use assets served by
the facade itself; tests verify pagination, errors and no provider/retained-API
calls. Real local auth, Markdown, comment writes/edits and reports still use the
retained named disposable database. The backend on 4450 was not restarted.

Chrome on account-review.localhost:4451: two live killmail tabs saw create/edit/reply
updates. New root #21096 arrived without Markdown; edit fetched the original bold
Markdown and saved the update. Another live root left an active reply's exact text,
0..42 selection and focus intact. Posting the reply appeared in the other tab.
A delayed 1500ms POST preserved newer root-composer input. All writes were fixture
rows, including report against fixture comment #38; no production write/deployment,
real financial action, moderation grant or live deletion ran.

Real formatting/preview, GIF 24->48 paging, favorites, keyboard insertion/focus,
provider-error recovery/Clear, image insertion and Escape cleanup pass. Closed
non-comment shell/announcement dialogs are inert; no comment dialog remained.
The GIF dialog fit the verified 390x844 tab at x16..374,y95..749. An earlier viewport
capability affected the other tab, so the desktop capture was discarded and a fresh
phone tab/capability was used. Thread/image controls and feed had no outer horizontal
overflow. Feed 30->60 paging, oldest sorting, search, no-match clearing and 503/error/
retained-card/retry recovery pass. All three tabs reported no console warnings/errors.
Viewport reset, fault rule cleared and temporary tabs closed. Screenshots inspected:
/tmp/shrike-ts-comments-gif-mobile.png and /tmp/shrike-ts-comments-feed-mobile.png.

Current owned /tmp/shrike-ts-comments-preview: 4205 PID 54335;
/tmp/shrike-ts-comments-facade: 4451 PID 54336 (-faults -media-fixtures).
Retained 4450 PID 59303, user 4000 and older 4200 untouched. Logs:
/tmp/shrike-ts-comments-{typecheck,tests,all-tests,go-tests,templ,build,
facade-build,facade,preview}.log. Goal remains active; campaign components are next.
allowJs/legacy exceptions and the full-site/build/Docker/reproducibility audit remain.


### Campaign TypeScript checkpoint, 2026-10-08

All seven campaign modules are checked TypeScript: creator state, directory,
activity chart, prize controls, owner editor, creator and detail tabs. Their
original JavaScript files and legacy exceptions are gone. The inventory is now
44/87 converted, with 43 JavaScript modules remaining. Typed entity/location
boundaries, editor/draft models and generated API request/response contracts
preserve the existing Go DOM, wire format, CSS and theme behaviour. Go declares
Stimulus actions directly; redundant campaign initializer factories are removed.

The family uses owned requests, scoped event bindings, safe fragment boundaries,
supersession guards and disposed polling. Campaign directory search/paging and
Back restore their controls; pending searches reconcile on page return. Failed
killfeed/tab reads retain content and expose Retry, without stale loading text.
Owner saves keep newer drafts rather than reload them away. Interrupted writes
never replay automatically. Contribution amounts typed during fragment reads
survive refresh, exact wallet-max strings are retained, and retries keep their
nonce through a failed post-success refresh. Changed creation payloads get a new
funding nonce; ambiguous successful creation responses require checking existing
campaigns. Claim/contribution failure, serialization, interruption and GET-only
reconciliation paths are covered by mocked requests. The shared RequestComponent
now counts busy requests across controllers on one root, including cancellation
and late completions.

Validation: strict TypeScript, 380 browser/WASM tests in 84 files (29 new
regressions), generated browser artifact checks, Go frontend/API/fixture tests,
production browser build and both Go binaries pass. Tests include real Go/templ
fixtures, malformed boundaries, draft retention, auth expiry, side movement,
loading/errors/retry, URL history, cursor paging, polling cleanup and chart
semantics. Browser output: 118 entries, 92 shared chunks, 1,018,998 JS bytes;
utility CSS 119,988 bytes. No allowJs/suppression/blanket-any exception was added.

Desktop Chrome creates an unfunded private campaign bR2DgOznWSDVhe in the retained
named disposable fixture, loads its owner editor and verifies save failure and a
1,500ms delayed successful save. The API retained "TypeScript campaign submitted"
while the editor retained "TypeScript campaign newer draft" and displayed the
newer-unsaved-changes status. Four-side metric switching/hover totals, intel and
comment fragments, Back, failed tab reads and Retry, side filters and 10-row
selection pass. Directory page 1->2, filtering to the new campaign and Back pass.
Prize display/Escape and funded-rule locks pass. No production mutation, actual
funding/claim, live deletion, role grant or deployment ran.

Phone checks use a verified 390x844 viewport. Area selection, Jita picking, date
selection/time controls, fixed-end/prize validation, detail/intel/comments and
prize display fit without outer horizontal overflow. Calendars now leave space
for the fixed navbar outside native dialogs (the panel is x25..329,y56..397).
Chart coordinates adapt to available width so labels remain readable and fewer
x-axis labels overlap; tooltip bounds stay inside the phone. Resize observers are
disposed. Prize rows use a second status line on phones; dialog/body width equals
scroll width (356px), and all payment statuses remain visible. The 390px prize
dialog stays between x16 and x374. Existing scrolling-modal placement tests still pass.
Browser console warning/error checks are empty. Screenshots inspected:
/tmp/shrike-ts-campaign-saved-draft.png, /tmp/shrike-ts-campaign-desktop.png,
/tmp/shrike-ts-campaign-calendar-mobile.png, /tmp/shrike-ts-campaign-chart-mobile.png
and /tmp/shrike-ts-campaign-prize-mobile.png. Viewport reset, fault rule cleared
and both temporary tabs closed; the final build was reloaded before cleanup.

Current owned preview: /tmp/shrike-ts-campaign-preview, 4205 PID 68703,
exec session 9456 (dev --port 4205). Facade: /tmp/shrike-ts-campaign-facade,
4451 PID 68719, exec session 20098 (-backend http://127.0.0.1:4450
-public http://127.0.0.1:4205 -faults -media-fixtures). Retained 4450 PID 59303,
user 4000 and older 4200 are untouched. Logs:
/tmp/shrike-ts-campaign-{typecheck,tests,all-tests,go-tests,templ,build,
artifact-tests,preview-build,facade-build,preview,facade}.log.

The goal remains active. Battle/war modules are next. allowJs/legacy removal,
remaining conversions and the full-site desktop/mobile, fitting/map, live/error/
focus/navigation/performance, production build/Docker/reproducibility audit remain.
The historical Chrome balance stall remains unproven and on the final audit list.


### Paused battle TypeScript work, 2026-10-08

User asked to pause for the day. Last accepted checkpoint: campaign family,
44/87 modules, 380 browser/WASM tests in 84 files, preview/facade PIDs 68703/68719
and retained fixture PID 59303. No browser tabs or viewport overrides remain.

WIP sources: battle-composition.ts, battle-replay.ts and shared/battle-model.ts.
The original two .js files were removed from browser-legacy.json (41 JS files
remain), but these conversions are not accepted until family/browser validation.
Composition keeps initial SSR image nodes and has typed row/filter/sort models.
Replay keeps the original SVG geometry/UI, owns events through Stimulus and uses
requestAnimationFrame only during playback instead of an always-running idle
interval. Pointer capture, visibility and disposal are scoped. hasReplayPosition
is now a real position type guard; runtime semantics are unchanged. Temporary
initializeBattleComposition/initializeBattleReplay bridges keep the legacy
battle-detail parent callable until it is converted. No Go template scopes have
been added for these two yet, and generated assets/binaries have not been rebuilt.

Strict TypeScript and the existing composition/replay math tests pass. Logs:
/tmp/shrike-ts-battle-typecheck.log and /tmp/shrike-ts-battle-tests.log. No new
regression/browser validation ran. Do not report 46 validated modules or reuse
the campaign evidence for battle DOM/behaviour.

Next: port battle-detail, battle-generator and battles directory; add the actual
Go template controller/action attributes and remove both temporary bridges.
Preserve the original sorting, draft/wire models, filtered-loss cursor behavior,
replay selection/geometry/feed and independent battle-map geography/markers.
Generator helpers currently use any in generatorSeedTeams; replace that with a
typed team model. Test request supersession, errors/retry, auth, retained drafts,
history/focus, replay playback/panning/layers/cleanup, generated previews/saves
and list/map/filter behaviour on desktop and phone before accepting the family.
Keep all production writes/deployments excluded; use the named disposable fixture.

Potential source entry points read this turn: src/utils/battleGenerator.ts,
src/utils/map/replay.ts, src/utils/battleExplorer.ts, Go battle_analysis.go,
battle_detail.go, battle_replay.go and battle_generator.go. Public API generated
models live in frontend/shared/api/types.gen.ts. The remaining battle JS sources
are unchanged and available for direct ports.

### Battle TypeScript checkpoint, 2026-10-08

Accepted 49/87 modules; 38 authored JavaScript files remain. This supersedes the
paused battle WIP checkpoint. Converted battle-composition, battle-replay,
battle-detail, battle-generator and battles directly from their existing sources.
Temporary parent bridges are removed; Go templates declare scoped Stimulus actions.
No layout redesign, API replacement, production write or deployment ran.

Shared boundaries validate composition rows, replay timestamps/coordinates,
landmarks/sides, generator participants and the complete preview/save payload.
HTML request errors preserve structured backend messages. Component requests own
body reads, cancel on exit and ignore stale responses. Composition retains initial
SSR image nodes. Replay has no idle animation loop; playback, visibility and
pointer capture have explicit lifecycles. Generator lists retain relevant nodes
and bound their caches to the current window; saves retain newer drafts and never
automatically repeat interrupted writes. Maps independently load geography and
markers and retain the existing renderer if a results fragment fails.

Validation: strict TypeScript; 405 browser/WASM tests in 85 files, including 25
new battle controller regressions using captured Go component markup; relevant
Go frontend/API/parity tests; production browser assets, artifact checks and Go
preview/facade builds. Final production graph: 118 entries, 93 shared chunks,
1,030,828 JavaScript bytes; utility CSS 119,988 bytes. These are whole-site totals,
not a per-page transfer or latency measurement. Logs use /tmp/shrike-ts-battle-*.
The DOM tests compensate for happy-dom's table-template parsing limitation by
restoring the raw captured template HTML, without changing browser markup.

Live Chrome desktop checks used populated public battle 1172169 (75 losses).
Composition search, aggregate/group views, sorting/reset and tab fragments worked.
Timeline measure/side filters worked; replay play/pause/speed/scrub, feed selection,
focus/fit/layers and native dragging worked. Dragging changed the SVG viewBox from
-17.649/-10.589 to -22.703/-13.311 with the same span. The killfeed switched to
10 rows, paged to page 2 using its cursor, then returned to page 1 with limit 100
and all 75 losses. Relevant error/warning logs were empty.

Generator creation used only the retained disposable local fixture (4450/4451).
It loaded real fixture entities and rendered a two-side report. A delayed local
save created battle #1 while the editor retained "TypeScript battle newer draft"
and displayed the saved link with its unsaved-changes notice. No production
report was created. Mocked regressions cover full preview payload preservation,
returned-ID reuse, session expiry, interrupted/ambiguous saves, duplicate submit,
late error bodies and stale previews. Calendar/search-picker implementations were
unchanged; their existing shared regressions remain part of the full suite.

390x844 checks covered generator side moves/image fallbacks/preview, composition
views with locally scrolling tables, replay playback/scrub and map region/system
navigation. Document widths were 379px, with no page-wide horizontal overflow.
Phone map surface was 363px wide. A phone screenshot is retained temporarily at
/tmp/shrike-ts-battle-phone-map.png. Map geography became usable independently of
markers; injected marker failure kept its canvas, and injected wormhole-geography
failure recovered on Retry. Region/system selection and browser Back worked.
A crowded all-time overview mouse click once selected a neighboring region;
keyboard selection and subsequent system clicks selected the intended targets.
Recheck overlapping marker hit targets in the final full-site map audit.

The preview is on 4205 and the disposable facade is on 4451; user servers and the
retained fixture were not restarted. Fault injection is cleared; review tabs are
closed and viewport overrides reset. Next: war/faction-war modules. allowJs/legacy
removal, remaining conversions and the full-site desktop/mobile, fitting/map,
WebSocket/error/focus/navigation/performance and Docker/reproducibility audit remain.
The historical Chrome balance stall is still unproven and remains on that audit.

Final runtime: preview PID 7651 (/tmp/shrike-ts-battle-preview, session 41953),
facade PID 7654 (/tmp/shrike-ts-battle-facade, session 22936); retained fixture
PID 59303 was untouched. Restart smoke checks returned 200 for the current public
battle 1172179 composition fragment and generator markup. The earlier public
battle 1172169 was removed by upstream while this check ran (its API also returns
404); use current list IDs for later live checks. Captured test data remains stable.

### War family TypeScript work in progress, 2026-10-08

Four sources (wars, war-detail, faction-war, faction-war-map) are now TypeScript
and removed from browser-legacy.json; 34 JavaScript files remain. The accepted
checkpoint remains 49/87 until desktop/phone browser validation is complete.
Go templates declare scoped actions; shared war-model and FragmentSections own
unknown-data boundaries, named reads, cancellation, retry and retained fragments.
Faction map SVG geometry/colors/territory helpers remain the existing implementation;
hover, zoom, panning and pointer capture are now scoped. The map no longer accumulates
window/per-node listeners on repeated mounts. Interrupted side loss reads reconcile
when returning to their combat tab; cache hits cancel competing reads.

Strict TypeScript, 427 browser/WASM tests in 86 files (22 new war regressions),
relevant Go frontend/API/parity tests and production asset/Go builds pass. Artifact
checks pass. Graph: 118 entries, 96 shared chunks, 1,038,775 JS bytes; utility CSS
119,988 bytes. Directory member sorting now uses member_count, and a short final
page retains Previous while disabling Next. Captured Go snapshots use public war
760892 and Caldari/Gallente data, including 90 systems; no production write ran.
The comparison facade now has a public GET-only WarReader with transport tests.

Browser gate is pending: creating the review tab and navigating its retained handle
both timed out on Chrome focus emulation. Inventory confirmed a live blank tab
351724865. Native access to that blank tab was rejected by browser URL policy
because its URL was invalid. No workaround to that rejected action ran. A connector
reconnect or user-opened http://localhost:4205/wars was requested asynchronously.
Do not count these four conversions as accepted or claim desktop/mobile validation.

Runtime: /tmp/shrike-ts-war-preview dev --port 4205 (session 27082) and
/tmp/shrike-ts-war-facade -backend http://127.0.0.1:4205 -public
http://127.0.0.1:4205 -faults (session 35337). Facade is temporarily public-read-only
for war comparisons; the disposable database/fixture PID 59303 remains untouched.
Restore facade backend to 4450 after war browser validation, without restarting
that retained fixture. The earlier owned preview/facade sessions had stopped during
the interruption; missing listeners were verified before restarting them.


Roam follow-up WIP: roam-composition.ts and a typed readRoamComposition boundary
are in place; 33 JS files remain. The existing aggregation/sorting implementation
is unchanged. Go declares the new scope, while a temporary declaration-only
initializeRoamComposition bridge keeps the still-JS roam-report parent callable.
Stimulus owns events; initial SSR image nodes remain intact. Strict checks and
five composition helper tests pass (two new boundary regressions). These are
preliminary helper checks, not DOM/browser acceptance. Generated running preview
still contains the war checkpoint; Roam assets/binaries have not been rebuilt.

Next independent work: convert roam-report, with scoped form/report controllers,
owned tab/map requests, typed form and API contracts, draft persistence and
retention during asynchronous saves, clipboard/reset/copy cleanup, lazy map
loading and stable loss paging. Remove the temporary composition bridge once
that parent is ported. Add real Go DOM regressions and local-only creation/update
checks, then complete desktop/phone/browser checks when Chrome is available.
The accepted count remains 49/87; do not count the five WIP modules as validated.

Current listeners: preview PID 30840 (session 27082), public comparison facade
PID 30844 (session 35337), retained disposable fixture PID 59303. Browser review
blank tab 351724865 remains for connector recovery; no viewport override was set
during this war turn. Keep the existing Chrome browser binding (ID 1; warBrowser
in the current REPL), and do not repeat the URL-policy-rejected blank-tab access.

### Roam family TypeScript work in progress, 2026-10-08

Both Roam modules now use strict TypeScript. The declaration-only composition
bridge is removed; Go declares scoped report/composition/form actions. The native
form sits in a display:contents lifecycle wrapper, retaining its layout and form
semantics. Owned requests cancel through response-body reads. Tabs preserve hero
images, expose read-only retry, synchronize metadata and handle Back. Loss role
and limit changes combine pending choices without reusing a stale cursor.

Create/update preserves newer drafts during both POST/PUT and subsequent HTML
refresh. Ambiguous/interrupted writes cannot automatically resubmit. Clipboard
completion, copy timers and map callbacks are scoped; map geography uses shared
typed scope guards and one retryable lazy renderer import. Battle directory uses
the same import helper. Dynamic Roam icon changes use the canonical iconHref
helper rather than deriving a sprite path from an individual SVG.

Strict checks and 447 tests in 87 files pass, including 18 new Roam DOM regressions.
Relevant Go frontend/API/parity tests and production assets/binaries build;
artifact/source checks pass. Graph: 118 entries, 97 shared chunks, 1,047,450 JS
bytes; utility CSS 119,988 bytes. Captured Go fixtures are local-only. Real API
creation and update retained the report ID/created_at and changed fixture losses
from three to four. No production writes ran.

32 JavaScript modules remain. Six War/Roam WIP modules still need live desktop
and phone validation; accepted checkpoint stays 49/87. A fresh browser inventory
was empty; opening a new valid local review URL again timed out on focus emulation.
The previously rejected invalid blank-tab action was not repeated or bypassed.
Continue independent conversions while the requested connector recovery is pending.

The latest preview/facade binaries are /tmp/shrike-ts-roam-preview and
/tmp/shrike-ts-roam-facade. Facade backend is restored to the retained disposable
fixture on 4450, with public GET-only comparisons directed to 4205. User servers
and retained fixture PID 59303 remain untouched. The preview's production Valkey
connection is currently timing out; do not claim a fresh live-data/reconnection
check until that dependency is reachable. The dynamic-icon audit, crowded map
marker hit targets and historical Chrome balance stall remain final-audit items.

### Scans, legacy archive and status WIP, 2026-10-08

Three additional original modules are strict TypeScript with Go-declared Stimulus
scopes. Scans have typed nested API boundaries, owned identity/analysis/save/login
requests and clipboard generations. Failed analysis/rendering retains the last
usable result; saving is disabled during analysis. Expired save responses cancel
older session reads. Interrupted saves never repeat automatically; the existing
API's deterministic input hash/upsert permits an explicit retry of the same scan.
Real local D-Scan/Local Scan analysis, fragment rendering, save, read and saved-page
checks passed against the retained disposable fixture. No production save ran.

The archive retains existing filters/presets and cursor paging. Independent kill/
top reads cancel through body consumption, preserve existing content on failure,
provide one Retry and reconcile interrupted reads on page return. Old pagination
cannot carry a cursor into new filters/sort/limit. Scoped autocomplete disposes
debounces/reads and cannot let one field's delayed blur cancel a newer field.
Mobile tab/history state preserves the query. Captured markup uses the existing
Go testLegacyReader; no archive database mutation was needed.

Status preserves its panel markup/formatting with normalized typed input models.
The shared relay transport now owns visibility, reconnect/backoff and disposal;
naturally closed sockets release callbacks too. Returning from bfcache starts
one new stream and retains the last snapshot. Invalid frames/nested metrics cannot
break charts. Hour tooltips survive new snapshots and support focus/tap. Tests use
an actual Go-marshalled Status payload and the real shared transport with controlled
sockets; real production delivery and phone layout remain browser gates.

The dynamic-icon audit found cached individual SVG paths reused as sprite bases.
Converted battle composition/replay/generator, campaign creator/prize, search
picker and advanced search now use iconHref. Two regressions begin with individual
SVG paths and verify canonical replacement icons. Remaining JavaScript icon code
will be checked as those families are converted.

Strict TypeScript and 490 tests in 90 files pass. Relevant frontend/API/status/
parity Go tests, production assets/facade build and source/artifact checks pass.
Graph: 118 entries, 97 shared chunks, 1,059,532 JS bytes; CSS 119,988 bytes.
Temporary fixture-capture package tests were removed. 29 handwritten JavaScript
modules remain; nine War/Roam/scans/archive/status modules await live desktop/
phone validation. Accepted count stays 49/87; the goal remains active.

Runtime: /tmp/shrike-ts-status-facade on 4451, session 77337, backend 4450 and
public origin 4205. Retained fixture PID 59303 and user servers remain untouched.
The rebuilt production preview exited because its primary PostgreSQL connection
timed out. Direct two-second checks found all three Tailscale services unreachable;
a user connectivity check was requested asynchronously. Chrome still times out on
focus emulation when opening a new valid review URL. Continue independent source/
local-fixture work, then start a new preview and resume the pending browser gates
when those dependencies recover. Do not claim fresh live data or browser proof.

### API docs and MCP TypeScript WIP, 2026-10-08

Both documentation entries are typed with Go-declared scoped actions. Docs no
longer duplicates the OpenAPI parser: src/models/openapiDocs accepts unknown,
checks document/operation boundaries and resolves local schema, parameter, body
and response references with cycle protection. Its any declarations and test cast
are removed. Tests render every operation in the generated document, plus a small
controlled document for card/transport regressions.

Endpoint fields remain lazy. Dedicated scoped controllers own controls, entity
suggestions, clipboard work and requests. Picked IDs apply to both path and query
parameters. Missing paths and invalid JSON prevent requests; an unset enum stays
out of the query. Shared URL helpers encode path values; curl quoting preserves
the raw body draft, including incomplete text, independently of Copy URL.
RequestComponent.response owns HTTP error inspection and body/stream consumption.
Streams stop at ten seconds/200,000 bytes and release their reader. Page/card exit
cancels work; interrupted writes do not repeat. Filtering through no matches
retains expanded field drafts on reconnect.

Read-only inspection of the retained pre-conversion docs bundle on 4450 confirmed
all original CSS classes remain in the port, including docs-param-field. Original
path labels/entity labels/enum placeholder text are retained. This source/DOM
comparison is not a live desktop/phone screenshot or CSS-layout acceptance.

MCP retains the registered Go catalog, configuration text, grouping/counts and
lazy output schemas. Repeated opening uses cached markup. Closing/unmounting
cancels a pending body; failures offer one Retry. Page return resumes open reads;
clipboard feedback/timers cannot update a newer click or returned page. The Go
catalog-error view remains usable for copying config. No MCP tool ran in production.

Full suite: 516 tests in 92 files. The final 39 docs/MCP/model tests pass after
copy/class refinements; strict TypeScript, relevant Go frontend/API/parity checks,
production assets/preview/facade builds and artifact checks pass. Graph: 118 entries,
97 shared chunks, 1,067,995 JS bytes; CSS 119,988 bytes. Temporary capture tests were
removed. Fixtures document their Go provenance and their DOM-only limits.

27 handwritten JS modules remain; eleven typed War/Roam/scans/archive/status/docs/
MCP modules await live browser validation. Accepted count stays 49/87; goal active.
Continue with the remaining source families and retain these pending browser gates.
allowJs/legacy removal and the final full-site mobile/theme/navigation/focus/live/
errors/fitting/map/performance/Docker/reproducibility audit remain outstanding.

Runtime: /tmp/shrike-ts-docs-mcp-facade, port 4451, session 40514, backend 4450,
public origin 4205. HTTP smoke checks returned 200 for docs/MCP/scans/status typed
scopes and the current docs bundle. Retained fixture PID 59303 was untouched.
/tmp/shrike-ts-docs-mcp-preview is built but not running; all three production
Tailscale services remain unreachable on a fresh two-second check. Chrome's review
tab focus emulation is also unresolved; no new browser proof is claimed. Restore
the public preview/connector when available, without restarting the retained fixture
or changing user servers. Existing crowded-map and Chrome-balance audit items remain.

Connectivity follow-up: the local Tailscale CLI reports Running and Self.Online=true.
None of the three configured service addresses, nor the expected Knas/EK service
names, appears in its visible peer inventory. This suggests the selected tailnet
or service addresses need checking; it does not prove which side changed. No
connection settings or environment values were modified. Latest facade PID 63732.

Next source family: graph.js and graph-render.js. Preserve the existing formatter
WASM/helpers and the nine intelligence tools' rendered markup. They have been
inspected but not edited; 27 JS modules is the exact source checkpoint.

### Graph, rankings and statistics TypeScript WIP, 2026-10-08

Four further original sources are typed/scoped: graph, graph-render, rankings,
statistics. 23 handwritten JS modules remain; fifteen typed modules still await
live browser/phone acceptance. Accepted count stays 49/87. The previous goal turn
made concrete progress; the source/automation work continues despite the pending
browser and production-connectivity gates.

Graph retains all nine tools and their existing formatter/markup implementation.
Typed projections preserve nested path affiliations, coalition cooperation and
census character totals. Scoped search components own debounce, focus/selection
and outside/tab cleanup. Hidden reads cancel, cache hits cancel competing requests,
and stale bodies cannot overwrite a new tool/type. Expanded coalition DOM survives
tabs. Failures retain usable results where available and offer retry. Per-tool
caches retain the original eight-entry bound. Existing renderer tests plus sixteen
new DOM/boundary regressions pass.

The graph OpenAPI schema described three wire fields incorrectly. It now declares
nested corporation/alliance affiliations, cooperation and totals.characters;
the generated document/client is refreshed. A Go contract test executes the actual
path/coalition/census HTTP serializers against controlled graph records and empty
name lookups, then checks emitted keys against the published schema. Payloads/routes
did not change, and no database or production Memgraph connection was used.

Rankings preserves SSR podium/images during selection reads and validates the
returned entity/window markers. It retains failed results, exposes one Retry,
synchronizes Back and preserves unrelated history state. Hourly polling is scoped,
silent, non-overlapping and paused while hidden. Foreground return before the
deadline avoids an unnecessary query; detach/pagehide disposes the timer. Nine
DOM regressions pass.

Statistics preserves warm SSR nodes without duplicate startup queries and caches
actual nodes across period switches. The four-request limit now uses owned active
leases; cancellation releases slots immediately, and a stale finalizer cannot
decrement a newer lease. Hidden section/old-period reads and queued jobs cancel.
Loaded key/period markers are validated. Expired caches retain usable rows on
failure, with explicit retry; period and population freshness remain 30/60 seconds.
Thirteen regressions cover bounds, slot release, caches, errors, Back, reduced-motion
jumps and page return/detach. Cold/warm Go fixtures and fragments are documented.

Full suite: 554 tests in 95 files, with strict TypeScript, relevant Go frontend/API/
parity checks, production assets/preview/facade builds and source/artifact checks
passing. Graph: 118 entries, 97 shared chunks, 1,078,681 JS bytes; CSS 119,988 bytes.
Temporary capture tests are removed and diff whitespace checks are clean.

Runtime: /tmp/shrike-ts-statistics-facade on 4451, PID 82206, session 21715;
backend 4450, public origin 4205. HTTP smoke checks confirm graph/statistics typed
scopes return 200. Retained fixture PID 59303 and user servers remain untouched.
/tmp/shrike-ts-statistics-preview is built but not running while the configured
production addresses remain unreachable. Chrome focus emulation remains unresolved;
no desktop/phone proof is claimed for these families. The facade's API document
still comes from the retained older backend; the new schema is verified offline
and in the newly built preview, without restarting that fixture.

Next source family: market.js, then the related universe/item interactions. The
market module has been read but not edited. Preserve the existing order sorting,
region/day/security filters, lazy category tree, sidebar and ship/item search.
Existing market glue also handles item Market fragments on universe pages, so
declare its components on reusable Go fragments and avoid a global page factory.
Remaining conversions, allowJs/legacy removal and the full-site mobile/theme/
navigation/focus/errors/live/fitting/map/performance/Docker/reproducibility audit
remain required; the goal stays active.


### Market TypeScript WIP, 2026-10-08

The original market source is now typed with three scoped controllers and shared
filter/sort/search boundaries. 22 handwritten JS modules remain; sixteen typed
modules await browser/phone acceptance. Accepted progress stays 49/87.

Existing order tables, chart markup, sidebar wording and reusable item panels are
retained. Sorting keeps the existing rows/directions and uses exact integer order
IDs for ties. Region/day/security selections combine while reads are pending;
only matching identity/view/filters may commit. Sorting performed during a read
is reapplied to the returned books. Failed requests retain content and expose one
Retry. Standalone URL history preserves other state; entity-item Back is owned by
the parent tab navigator and does not start a duplicate market request.

Lazy category trees validate their parent marker, cache exact nodes, cancel reads
when a parent closes and resume visible open nodes after page return. Search owns
its debounce and parallel ship/item reads, deduplicates numeric type IDs and keeps
names as text. Errors differ from empty results. Escape restores input focus and
cancels reads; late bodies cannot reopen the dropdown after hiding or detach.

Sixteen new DOM/boundary tests use HTML captured through normal Go handlers with
the existing controlled testUniverseReader. Fixture provenance is recorded in
frontend/test/fixtures/market/README.md. The temporary capture test is removed.
No production records were changed. Full suite: 570 tests in 96 files; strict TS,
relevant Go market/item/schema checks, full frontend/parity tests, source/artifact
checks and production asset/preview/facade builds pass. Production graph: 118
entries, 97 shared chunks, 1,083,542 JS bytes; CSS 119,988 bytes.

Runtime: /tmp/shrike-ts-market-facade on 4451, PID 87303, session 77904,
backend 4450/public origin 4205. HTTP smoke confirms graph/statistics scopes return
200. Market fixture captures exercise real Go HTTP handlers; this facade does not
configure a UniverseReader and is not claimed as market page evidence. The
retained fixture PID 59303/user servers are untouched. The preview binary is built
at /tmp/shrike-ts-market-preview but not running while production connectivity is
unavailable. Chrome currently lists no review tabs; live desktop/phone acceptance
remains pending. Source conversion can continue independently.

Next source family: universe.js, currently unchanged. Port its existing item/group
tab navigation, image dialog, cursor kill paging, fitting filters/supporting
fragments and preferred-tab behavior. Preserve the source and shared helpers.
Remaining conversions, allowJs/legacy removal and the full mobile/theme/navigation/
focus/error/live/fitting/map/performance/Docker/reproducibility audit remain required.


### Universe/item TypeScript WIP, 2026-10-08

Universe is now strict TypeScript with scoped item/group navigation and item-fitting
controllers. 66/87 original sources are typed; 21 JS remain and seventeen typed
modules await desktop/phone acceptance. Accepted checkpoint remains 49/87.

Existing Go HTML/classes, persistent hero and image dimensions, tab links, fitting
families/slots, histograms and filters are retained. Tab reads validate kind/id/tab,
retain the current content on failure and expose Retry. Metadata/canonical/active
links and history state update after commit. Shared session/preferred-tab behavior
ignores late results after explicit navigation or detach. Paging preserves cursor
history, validates kind/page/limit and clears cursors when a limit changes. Dialogs
use the larger image source and close on pagehide/detach.

Fitting filters combine pending selections and retain newer form drafts/open
families. Supporting panels load independently, validate type/query markers, keep
exact DOM during refresh and expose separate Retry controls. Scope disconnect
cancels reads; stale bodies cannot write filters after Back changes the route.
The same route guard was added to item-market and has its own regression. Number
shorthand parsing and repair metric sorting now share strict helpers with the
remaining fitting-search source; that source is still inventoried as JavaScript.

Twenty-four new item/group DOM/lifecycle tests plus one market route regression
pass. Captured normal Go HTTP pages/fragments use the controlled testUniverseReader;
provenance is in frontend/test/fixtures/universe/README.md. No production database
was read/changed for captures. The unit harness restores option[selected] values
because happy-dom parses the captured NPC default incorrectly (angels despite a
selected guristas attribute); production markup/selection logic is unchanged.
External module/style loaders are omitted in unit mounts. Browser layout/selection
acceptance remains separate and pending.

Full suite: 595 tests in 97 files, strict TypeScript, targeted Go item/group/market/
fitting/schema checks, full frontend/parity checks, source/artifact checks and
production asset/preview/facade builds pass. Browser graph: 118 entries, 100 shared
chunks, 1,088,664 JS bytes; CSS 119,988 bytes. Temporary capture tests are removed;
diff whitespace checks are clean. No browser acceptance is claimed for WIP modules.

Runtime: /tmp/shrike-ts-universe-facade on 4451, PID 91482, session 59349;
backend 4450/public origin 4205. HTTP smoke confirms graph/statistics typed scopes
return 200. The retained fixture PID 59303 and user servers are untouched. Item/
group handler evidence comes from captures because the facade has no UniverseReader.
/tmp/shrike-ts-universe-preview is built but not running while production services
remain unreachable. Chrome has no review tabs and the earlier connector problem
is unresolved. Pending user requests remain open; independent conversion continues.

Next source family: system.js, currently unchanged. It handles system/region/
constellation tab navigation, valuable selections, cursor kills and battle paging.
Reuse the lifecycle/fragment/focus/filter helpers and preserve source behavior.
Remaining conversions, allowJs/legacy removal and the full mobile/theme/navigation/
focus/error/live/fitting/map/performance/Docker/reproducibility audit remain required.
The goal stays active.


### Location/blog TypeScript checkpoint, 2026-10-08

Previous goal turn made progress: market and universe sources converted, meaningful
regressions/builds recorded. This pass adds system and blog. 68/87 original sources
are now strict TS; 19 JS remain. Blog browser acceptance raises the accepted count
to 50/87, with eighteen typed modules still awaiting browser validation.

System/region/constellation behavior keeps existing templates/classes, persistent
hero/valuable panels, tab metadata/preferences/Back, cursor kills, battle paging
and chart hour/readout handling. Location uses the shared fragment loader with
root replacement; shared KillPages now supplies item/group/location cursor logic.
No extra wrapper DOM is introduced. Kind/id/tab/page/category markers validate
responses. Returning to the currently displayed tab/category cancels a competing
read. Reads/errors/retry are independent, hidden/detached reads cancel, page return
resumes interrupted intent and stale text cannot commit. Twenty-eight new location/
helper tests and existing universe tests pass. Go captures extend controlled
readers with two recorded activity hours, one kill/valuable entry and fifty battles;
provenance is in frontend/test/fixtures/location/README.md.

Blog retains the source listing/detail markup and local date formats. Scoped reads
cover headers/text, retain existing cards and cursor on failure, disable one active
pagination button and resume canceled work on page return. Unexpected fragment
roots are rejected; URL deduplication retains existing card nodes. Dates initialize
through card/detail scopes on insertion. Nine new DOM/lifecycle tests use captured
Go blog HTML, with provenance in frontend/test/fixtures/blog/README.md. Capture tests
are removed; no production records were read or changed for fixtures.

Chrome's previous browser 1 became unavailable. Fresh inventory showed a reconnected
Chrome browser 2; its documented connector now opens/inspects local pages correctly.
No user tabs or settings were modified. Agent comparison tabs used the retained
JavaScript Go renderer on 4450 and the rebuilt TypeScript renderer on 4451. Desktop
initial twenty and terminal forty-five cards match exactly in order/text/dates and
width/height. A local 300ms/503 pagination fault retained all twenty cards and busy
feedback; clearing it and retrying restored the matching forty-five. Local faults
are now clear.

Phone checks used the existing same-origin fixture frame at 390x844, because the
browser-scoped viewport override applied only to the reference tab. Actual frame
client/scroll widths both measure 379px; no horizontal overflow. Rich detail header,
cover/body/short-long date rendering match visually; tagged updates lists match all
twenty-two cards and dimensions after pagination, with terminal controls hidden.
Back restores rich detail with both long dates. No browser console errors were
recorded. Temporary viewport override was reset. Screenshot:
/tmp/shrike-ts-blog-phone.jpg. Blog is accepted; location browser acceptance is still
pending while the configured production services are unreachable.

Full suite: 632 tests in 99 files; strict TypeScript, targeted system/region/entity/
item/group/blog Go checks, full frontend/parity checks, source/artifact checks and
production asset/preview/facade builds pass. Asset graph: 118 entries, 101 shared
chunks, 1,091,684 JS bytes; CSS 119,988 bytes. Diff whitespace checks are clean.

Runtime: /tmp/shrike-ts-location-blog-facade on 4451, PID 97698, session 59609,
backend 4450/public origin 4205. Retained fixture PID 59303 and user ports are
untouched. /tmp/shrike-ts-location-blog-preview is built, not running. Fresh TCP
checks still cannot reach the remote PostgreSQL, Valkey or Memgraph services. Existing local blog/account/API fixtures remain responsive.

Next: browser backlog against local fixtures where available, plus search.js source
conversion. Search/post sources have been inspected; their conversion and shared kill submission migration remain pending.
Remaining conversions, allowJs/legacy removal and the complete mobile/theme/
navigation/focus/errors/live/fitting/map/performance/Docker/reproducibility audit
remain required; the goal stays active.


### General search TypeScript checkpoint, 2026-10-08

Previous goal turn made progress: location/blog conversion and blog browser
acceptance. This pass converts search. 69/87 original sources are now strict TS;
18 JS remain. Search acceptance brings validated progress to 51/87, while eighteen
previously typed modules still await desktop/phone acceptance.

Existing eleven entity types, card classes/structure, image variants, affiliations,
filter labels/counts, suggestions and result statistics are preserved. Runtime
projections use generated SearchResponse fields and validate unknown JSON rather
than asserting whole payloads. Kind/ID guards exclude invalid routes; market search
and the accepted picker now share the same EVE ID parser. Filters retain actual
card/image nodes. Stimulus owns delegated actions and the request component owns
debounce/body lifetimes. Clear cancels work, retains other URL/history state and
restores input focus. Failures retain usable cards with a separate Retry; empty
results remain distinct. Hidden/detached work cancels; page return resumes only
interrupted searches. Back restores the query and ignores other routes.

Fifteen new DOM/model regressions plus existing picker/market tests pass. Go search
captures execute the real sorter/serializer against controlled rows for all eleven
kinds and corporation/alliance name joins. A persistent contract test checks type
precedence, emitted identifiers/counters and affiliations. Its fake rows implement
pgx RowScanner as required by current RowToMap. Fixture provenance lives in
frontend/test/fixtures/search/README.md; temporary capture tests are removed.
No production records were read or changed for captures or comparisons.

The retained binary's older fault helper accepts but ignores response overrides,
so direct 4450 comparison initially received only its eight simple fixture hits.
A reusable -reference-port mode now serves retained HTML/CSS/JS via an anonymous
GET/HEAD-only proxy on 4452, sharing modern local read fixtures with 4451. It strips
credentials and authentication response cookies, never relays mutation methods,
and has explicit proxy tests. Retained PID 59303 was not restarted. This compares
the actual original JavaScript against the converted controller with one Go-produced
payload; no duplicate implementation was invented for the reference.

Chrome desktop comparison matches all eleven cards exactly in order/text, primary
and affiliation links, image URLs and width/height. Every entity filter returns the
same content/routes as the original; All restores all eleven. A delayed 503 search
retains the eleven usable cards with one Retry; restoring reads recovers. Clear
restores empty input/suggestions and focus. The retained CSS snapshot still uses a
fixed blue character badge; current branch CSS already uses the selected primary
color. Both computed primary variables are purple (#a855f7); the new badge follows
that variable. No CSS changed in this conversion. Native tab heights differed, so
responsive comparisons use identical fixture frames rather than comparing footer
positions from different window heights.

390x844 same-origin phone frames match all eleven card texts/routes/dimensions;
character filtering returns one matching card. Actual client and scroll widths are
both 379px, without horizontal overflow. Empty response displays the expected
message with zero cards and no error panel. Back restores the cleared desktop
search; Forward restores the phone query. No browser console errors were recorded.
Screenshot: /tmp/shrike-ts-search-phone.jpg. Local faults on 4450/4451 are cleared.
No temporary browser viewport override was applied in this pass.

Full suite: 647 tests in 100 files, strict TypeScript, relevant Go frontend/API/
reference checks, source/artifact checks and production assets/preview/facade builds
pass. Asset graph: 118 entries, 102 shared chunks, 1,095,083 JS bytes; CSS 119,988
bytes. Diff whitespace checks are clean. Runtime:
/tmp/shrike-ts-search-facade-v2, PID 32015/session 18531, current renderer 4451 and
read-only original renderer 4452; backend 4450/public origin 4205. The preview
binary /tmp/shrike-ts-search-preview is built but not running while production
connectivity remains unavailable. User servers and retained fixture remain intact.

Next: shared kill-submit/post sources, then remaining entity/home/coalition/kill/
fitting/map sources, with browser backlog where local data permits. allowJs/legacy
removal and the complete mobile/theme/navigation/focus/error/live/fitting/map/
performance/Docker/reproducibility audit remain required. Goal active.


## Kill submission/post TypeScript checkpoint, 2026-10-08

The prior search checkpoint is superseded by 53/87 accepted, 71/87 sources typed,
16 JS remaining, eighteen typed browser-pending modules. Kill-submit and post are
accepted together. The retained original JS/HTML/CSS on 4452 remains the reference;
no production records or jobs were changed. Pure in-memory -submission-fixtures
handles /api/killmail/post, recognized synthetic exists/detail requests and a shared
local WS relay before both proxies. Other reference mutations remain blocked.
Tests cover isolation, invalid config, deduplication and acknowledgment/rejection.

21 new submission regressions cover original counts/links/twenty-link cap,
single-existing redirect, empty/invalid input, global paste exclusions, clipboard
single flight/errors/detach, unchanged/newer drafts, delayed responses, overlapping
attempts, pending persistence and non-replayed POST reconciliation, relay all-topic
subscription/dedup/reconnect, hidden/stale/nonoverlapping poll reads, detail failure,
notice expiry, safe rendering and retained keyboard focus/images. All 668 tests in
101 files pass; strict TS, Go frontend/facade tests, source/artifact checks and
production assets/binary build pass. No check suppression was introduced.

Chrome desktop: form dimensions and buttons match exactly (1198px width, 695.25px
result form height; original 16px base type), mixed accepted/existing counters and
links match, delayed queue disables textarea/clipboard/button, rejected batch
retains draft, failed API clears result and restores usable controls. New transport
shows the API error message; original transport uses its generic Failed to queue.
Live WS notice text/image/value matches the retained renderer. Navigation to About,
reload and Back preserve pending/ready state; fixture call count confirms no repeat
POST. 390x844 frames: 363px form, 329px textarea, 288px ready notice, client/scroll
width both 379px, matching source with no horizontal overflow. Dropped WS reconnects
in the new controller; original falls back to polling. Both deliver matching fallback
notices. Final built-asset phone smoke verified focus remains on First Pilot when
Second Pilot arrives. No browser console errors. Screenshot:
/tmp/shrike-ts-submission-phone.jpg. Clipboard denial/global paste and interrupted
POST cases are deterministic DOM transport tests rather than live OS clipboard tests.

Assets: 118 entries, 103 shared chunks, 1,101,838 total production JS bytes; CSS
119,988 bytes. User 4000/4200 servers untouched. Retained PID 59303/4450 unchanged.
Current /tmp/shrike-ts-submission-facade PID 42341/session 23602 serves 4451/current
and 4452/reference with -backend http://127.0.0.1:4450 -public http://127.0.0.1:4205
-faults -media-fixtures -submission-fixtures -reference-port 4452. Public 4205 remains
offline while production connectivity is unavailable. Browser tabs closed; no
viewport override applied. Local fault rule is clear. Submission fixtures remain
pure memory and reset on restart.

Continue with coalition directory/editor, entity, home, kill detail, fitting and map
sources, plus eighteen pending browser families where fixtures permit. Full final
mobile/themes/navigation/focus/loading/error/live/fitting/map/performance/build/
Docker/reproducibility audit and allowJs/legacy removal remain required. Goal active.


## Coalition TypeScript checkpoint, 2026-10-08

The preceding submission checkpoint is superseded by 54/87 accepted, 72/87 sources
typed and 15 JS remaining. Eighteen earlier typed modules still await browser
acceptance. The prior goal turn made verified progress (submission conversion and
browser acceptance); this turn converts and accepts coalitions. Goal remains active.

Coalitions use a Go-declared Stimulus scope. Shared generated API projections,
search IDs, session transport, request lifetime and fragment focus are reused.
Picker reads debounce/cancel; stale bodies, wrong types/IDs and duplicate choices
are rejected. Login destinations require HTTPS. Creation/save is single flight,
retains newer drafts, and validates response identity/revision. Refreshed detail
keeps the actual newer form and image nodes while updating its published baseline.
Interrupted write or post-save HTML reconciles with GET on return, never automatic
PATCH replay. Reload errors keep a retry button. Templates/CSS classes remain intact.

Go capture fixtures use testCoalitionReader via actual HTTP routes; temporary
capture test removed. 19 new DOM regressions cover filtering, auth, picker races,
source payload/revision, create, late drafts during PATCH and HTML, image retention,
conflict/invalid identity, stale fragments, detach/page return, safe login, activity
integration and browser-restored query on pageshow. Existing 4 activity tests pass.
All 687 browser/WASM tests in 102 files, strict TS, Go frontend/facade tests,
source/artifact checks, production assets and final Go binary build pass. CSS equals
the archive byte-for-byte (SHA256 553764b63a965bb68d59d4087031cb943afefb46c05e9127355e1c0780f45d5f).

The retained binary never registered coalition page routes and its coalition JS
predates the activity split. The explicit -reference-coalitions option renders
actual Go components with the untouched two scripts from
.data/source-archives/browser-before-ts-pipeline-20261007.tar.gz. Only those exact
named files are read/served under /__fixture/coalition-source/. CSS and accepted
shell assets use the current unchanged component styles; older retained assets
remain available through the anonymous read-only reference proxy. Archive-selection,
credential/cookie/mutation isolation, asset delegation and remote reader tests pass.
A comparison-only meta removal prevents the generated page's current dev-instance
ID from repeatedly reloading against the retained process's old reload endpoint.
This correction fixed interrupted reference navigations; application reload stays
unchanged. The facade overview adapter reads the disposable API (which lacks an
overview endpoint), so it is not production cold-load/performance evidence.

The retained disposable fixture started Oct 5 and its 24-hour owner session had
expired. Its two PostgreSQL connections identify the sole active
shrike_frontend_parity_84b93bb35495 database in local shrike-go-postgres-1. The exact
owner session selected by /__fixture/account was renewed locally after checking
UUID, database name, character 4242 and Fixture Owner; no other sessions renewed,
no credentials printed, no renderer restart. Browser then created only the named
TypeScript Fixture Coalition (slug typescript-fixture-coalition), alliance 70121,
and changed its description once. Seeded coalitions and production were untouched.
That fixture has real members, combat totals and create/update edit history.

Chrome desktop: both scripts match populated card text/routes, 591x309 card size,
16px base font, winter filter/empty state, nonblocking activity loading and errors/
retry. Actual fixture create redirects to detail with correct member and history;
delayed PATCH disables Save while permitting newer description edits, then updates
revision/history and retains the newer draft and its focus. Public detail statistics
and member text match the archived script. Original and new Back both initially
restored query text while showing all cards; the new pageshow fix now returns the
single filtered card with query typescript. Final built-asset check passed.
390x844 frames match detail statistics exactly (175.5x78 each), root 363px and
client/scroll width both 379px. Phone editor removal disables Save with zero members;
closing/reopening restores the published member list. Editor and directory have no
horizontal overflow. No JS console errors in current/reference checks. Screenshot:
/tmp/shrike-ts-coalition-phone.jpg. Reference edit writes remain blocked; auth/write
comparisons use controlled DOM transport and live current disposable API tests.

Final runtime /tmp/shrike-ts-coalition-facade-final PID 49802/session 48548:
-backend http://127.0.0.1:4450 -public http://127.0.0.1:4205 -faults -media-fixtures
-submission-fixtures -reference-coalitions -reference-port 4452. Current renderer
4451, reference 4452; retained PID 59303/4450 unchanged. User 4000/4200 servers remain
untouched. 4205 remains offline. All comparison tabs closed, no temporary viewport
overrides. Fault rule cleared. Assets: 118 entries, 103 shared chunks, 1,105,693
production JS bytes and 119,988 CSS bytes. Diff whitespace clean.

Next: entity/home/kill detail, ten fitting modules and two map modules, plus eighteen
pending browser families. allowJs/legacy removal and full mobile/theme/navigation/
focus/error/loading/live/fitting/map/performance/build/reproducibility/Docker audit
remain outstanding. Do not mark the goal complete yet.


## Entity TypeScript WIP, 2026-10-08

Previous goal turn was verified progress (coalition acceptance). This turn ports
the full 671-line original entity module to strict TS scopes. 73/87 sources typed,
14 JS remain, 54/87 accepted; nineteen typed browser-pending modules now include
entity. No production data, local fixture rows, user listeners or retained renderer
changed in this turn. Goal active, not complete.

EntityNavigationController owns persistent tabs, preference resolution, metadata,
history and portrait/legacy dialogs. EntityValuable/ShipClasses scopes retain the
source classes/routes/formatting and image sizes; requests cancel on removal or
page hide. EntityContentController owns the replaceable tab's dashboard, intel,
faction/militia, top lists, achievements, membership/corporation/conflict tables,
kill paging, live relay and relative-time work. Go declares actions/scopes. Shared
request/fragment/relay/paging/search/format helpers replace manual lifecycle code.
Member choices combine during pending reads; wrong fragment markers cannot commit.
Organization intel can arrive after stats replacement and fill all fresh slots.
The source's complete top-period options (including hourly fractions/alltime) are
preserved. Portrait dialogs retain natural image sizing, including legacy images.
Empty militia 200 is accepted; failures surface visibly in the originally hidden
slot with Retry. Events during kill reads retain their pending refresh count.

27 regression tests mount real Go captures for all four entity kinds and fragments:
widgets/formatting/empty/retry/races; persistent portrait/card nodes; tab metadata,
history, cancel/back/query/modified clicks/default preferences; legacy dialog and
space-image fallback; dashboard/intel races and slots; fractional top slices;
combined membership filters/dropdown keyboard/outside; corporation/war/campaign
controls; achievements; opaque cursors/limits; WS topics/role/group/coalescing/
pending-counter/reconnect/hide; responsive lazy sidebar; faction empty/failure;
detach and runtime model guards. Go captures use controlled test readers with a
fixed Rifter row, large opaque decimal cursor, ally and recent join; temporary
capture tests removed, provenance in frontend/test/fixtures/entities/README.md.

Full suite: 714 tests in 103 files. Strict TS, focused entity tests, Go frontend/
facade tests, source/artifact checks and production assets pass. Final small popup
sizing and group-filter checks were rerun in the focused 27-test family. Assets:
118 entries, 103 shared chunks, 1,111,527 production JS bytes; CSS 119,988 bytes.
Browser acceptance has not run yet. Existing running coalition facade remains
/tmp/shrike-ts-coalition-facade-final PID 49802/session 48548, ports 4451/4452 and
backend retained 4450. It embeds the previous accepted asset checkpoint and does
not register entity page readers. Newly built entity preview/facade binaries are
not started; do not mistake the running facade for entity validation.

Resume with populated entity browser comparisons. Extend the local facade with an
EntityReader using the retained disposable API (or a clearly labeled Go-controlled
fixture); use the pre-migration archive's entity.js and its imported helpers as the
reference if the retained binary lacks routes. Protect PID 59303 and user ports
4000/4200. The archived source and full Go templates exist: reuse them faithfully.
Desktop/phone tabs, tables, portraits, period controls, paging, live updates,
keyboard/focus/history/error/retry and themes still require browser acceptance.
Then port home, kill detail, ten fitting modules and two map modules, and finish the
nineteen-family browser backlog plus full mobile/theme/navigation/focus/loading/
error/live/fitting/map/performance/build/reproducibility/Docker audit. allowJs and
legacy removal remain required when no handwritten JS remains.


## Entity browser acceptance, 2026-10-08

The preceding source-only WIP is superseded: 55/87 accepted, 73/87 typed, 14 JS
remain; eighteen other typed families await browser acceptance. Previous goal turn
was verified progress (full entity source conversion and tests). This pass accepts
entity after real Chrome comparisons. Full goal/final audit still outstanding.

The local facade now optionally opens the existing disposable database using
TEST_DATABASE_URL only when host is loopback and name begins shrike_frontend_parity_.
It forces default_transaction_read_only=on and verifies transaction_read_only before
using api.NewHomepageReader with the fixture database wrapper. This reuses the
actual Go entity queries/renderers, not a second implementation. No records or
jobs were written. The seed lacks NPC catalogue names; missing factions 500001/
500003 receive only visibly named Fixture Faction identities. All remaining
combat/warzone/militia methods are the real read-only reader. Other missing entities
remain missing. Guards have unit tests. Database credential stayed in process env,
never printed or written to files. Existing retained fixture PID 59303 unchanged.

The coalition reference helper is consolidated into reusable named component
archive/renderer helpers. Entity reference serves exactly entity.js, fragment.js,
browser-request.js and default-tab.js from the pre-migration archive. All four
new Stimulus scopes are stripped from reference component HTML, preserving
classes/structure. Current assets/styles remain available; reference API requests
are anonymous/read-only and its current-process dev-instance meta is removed to
prevent reload loops against the old retained endpoint. Isolation/selection tests
pass. No new browser source JS was added; original scripts stay inside the private
archive and opt-in fixture transport.

Chrome desktop comparisons used real Fixture Red Pilot 70131, corporation 70111,
alliance 70121 and the two fixture factions. Valuable cards match text/routes and
139.5px dimensions, with eight real kill rows, source limits and connected badges.
Character intel matches playstyle/ships/targets; periods select 30d. All 162
achievement cards match, Completed/In Progress show zero and Combat shows six.
Portrait dialogs use the same 512 source URL and close correctly. Alliance member
search/dropdown/Escape returns focus to summary; filters/sort/corporation selection
match and corporation 70111 selects two members. Four corporation rows and sort
name/asc match. Organization intel/stats match. Top slices expose all 10/11 periods
including 6h=0.25. Battle tab has one real battle; war/campaign empty states and war
sort=isk match. Failed delayed tab retains campaigns; new Retry recovers members.
Original source requires clicking the tab again. Faction warzone/militia text
matches; Caldari shows Fixture Red Fleet, Amarr has no visible militia/error.
The original leaves a hidden empty placeholder; new code removes that placeholder.

Live relay test emits only local frames, never database writes. A 503 on live
refresh proves both subscribers initiate fresh reads and keep eight usable rows.
Two frames one second apart during a ten-second read show +1 new while pending.
After success the new controller retains that later arrival; original clears it.
An explicit fresh read clears the new counter. Forced relay disconnect recovers
both connections. Losses retains limit 10; Back asynchronously restores kills and
limit 10. Final console checks record no JS errors. The real fixture has fewer
than ten rows for these scopes, so no live second-page claim: opaque cursor/page/
limit behavior remains verified with real Go DOM fixtures. Legacy portrait choice
is likewise a DOM test, while live current-image open/close is browser verified.

390x844 frames match alliance member text, links, 80px portrait, 363px main width,
client/scroll width 379px. Phone corporation search and Escape work. Kill rows match
at 345x68px; the desktop sidebar remains lazy on phones. Caldari warzone/militia
fits without overflow. Screenshot /tmp/shrike-ts-entity-phone.jpg records mobile
member controls. CSS equals the archived entity.css byte-for-byte. No temporary
viewport override used, all comparison tabs closed, local faults cleared.

Strict TypeScript, focused entity/source/artifact checks (30 passing), Go frontend/
facade tests and reference/read-only guards pass. The previous complete frontend
suite passed 714 tests in 103 files; browser support changes did not alter shipped
browser sources. Assets remain 118 entries, 103 chunks, 1,111,527 total JS bytes and
119,988 utility CSS bytes. Whitespace checks clean. Current runtime
/tmp/shrike-ts-entity-browser-facade-v2 PID 66924/session 45611, ports 4451/current and
4452/reference, retained backend 4450; args include -entity-database,
-reference-entities, -reference-coalitions, -submission-fixtures, -faults,
-media-fixtures and -public http://127.0.0.1:4205. That public preview remains offline;
user 4000/4200 untouched. Future launches must securely supply the guarded local
TEST_DATABASE_URL (never echo credentials).

Next: home and kill detail, ten fitting modules and two map modules. Also finish
the eighteen typed browser families and full mobile/theme/navigation/focus/loading/
error/live/fitting/map/performance/reproducibility/Docker checks. Remove allowJs and
legacy inventory when no handwritten browser JS remains. Goal active.


## Public kill-feed and kill-detail TypeScript checkpoint, 2026-10-08

57/87 original modules accepted; 75/87 typed; twelve JS remain. The previous
entity turn was verified progress. This pass converts home.js and kill-detail.js
fully and records their desktop/phone comparisons. Eighteen previously typed
modules still await browser acceptance. The complete goal and final audit remain
active; these counts do not claim every edge case was exercised live.

Home owns state on the persistent homepage/kills root. Each cold top box owns
its read/retry lifecycle separately. Shared request ownership, fragment focus,
search projections, row rendering, relay reconnect and meta-group matching replace
local boilerplate. Filters preserve the actual panel/input nodes across HTML
replacement. Cursor strings stay opaque. Top lists remain lazy on phones and
accept legitimate empty fragments. Canceled reads/timers cannot commit after
hide/disconnect; cached return resumes interrupted intents. History state/query
are preserved when switching phone hash tabs. The public feed keeps subscribing
while filtered HTML omits live-control markup, matching the original behavior.

Live frames are validated before matching filters and rendering. Empty feeds
create the existing header/row geometry for their first arrival. Existing rows
retain their nodes; duplicates/older rows are ignored and limits/day dividers are
reconciled. An arrival after a refresh starts remains in the new-count badge after
its response; explicit refresh clears it. Excluded frames do not increment it.

Kill detail retains result context and safe links, source attacker batches of
25, container-child grouping during sorting, image fallback, delayed wheel tips,
mobile tools/hash tabs and optional matchup. Escape closes dropdowns with focus
returned to the summary. Fitting uses a validated generated API projection and
the exact existing CSV/gzip v3 format, extracted into the shared fit codec. A late
compression result cannot navigate after page exit; failures restore the button.
The shared kill-navigation reader now validates unknown stored JSON.

Actual Go HTML captures drive 17 kill-detail, two real gzip codec and 16 home
regressions. They cover races, retries, cancellation, drafts/focus, all quick-filter
wire fields/security boundaries, large cursors, empty/live rows, lazy top boxes,
image errors, attacker counts/items/containers, fitting and optional matchup.
Full suite: 749 tests in 106 files; strict TS, Go frontend/facade, source/artifact
checks and production assets/binary pass. Bundle: 118 entries, 103 chunks,
1,119,641 JS bytes; utility CSS 119,988 bytes. No CSS edits; kill-detail.css equals
its source archive byte-for-byte.

Opt-in -reference-public-kills uses the existing guarded read-only local database
reader for both home and /kills routes. Reference serves exactly archived home.js,
kill-detail.js, fragment.js and kill-row.js; no archive extraction or private files
are exposed. Its Go HTML loses only the matching Stimulus scopes/dev-instance
meta. A shared-shell controller has no standalone tag, so the reference helper
now inserts its archived entry into full HTML; fragment responses get no scripts.
The new isolation test uses the actual site-only script shape. Reference API
requests remain anonymous/read-only; data credentials stay in process env.

Chrome comparison connection changed from 2 to available 1 after a runtime reset.
New ephemeral tabs only; user tabs/4000/4200 were untouched. The disposable home
rollup is empty while seven valuable cards and historical kill records exist.
Highsec filtering loads ten real records in both implementations. System search
returns the same seven Ji matches and selects Jita; location+security badge is
two, selected input readonly, and limit 10 agrees. Keyboard Enter on the explicit
kill overlay reaches /kill/2000000207 with the same result context (1/10, disabled
Previous, Next and return link). Clicking center content can correctly follow an
entity link; earlier such clicks were not evidence of kill-context navigation.

Delayed 503 reads retain rows and expose Retry in new code; original requires
clicking a choice again. Retry restores limit 25 in both. Valuable ships switches
to the same empty state. Local frames (no DB writes) prove first-row insertion in
new code while original stays empty. A frame during a ten-second unfiltered read
shows count one; after both reads complete, new retains one and original clears
to zero. Explicit fresh read clears the new badge. Forced relay disconnect
recovers subscribers. No JS console errors observed.

Phone 390x844 frames: /kills/big keeps three top placeholders until Top Lists;
selecting it removes the three legitimate empty results, and Valuable reveals
its panel. Homepage highsec rows match at 345x68; client/scroll width both 379,
seven valuable cards. Screenshot /tmp/shrike-ts-home-phone.jpg includes filtered
rows and a local relay arrival. Kill detail wheel compares at 540 desktop/363
phone; render image desktop 507.625 square, matching routes/text. Phone Kill/Tools
switches, delayed fitting failure/retry and actual compressed editor handoff/Back
were checked. Screenshot /tmp/shrike-ts-kill-detail-phone.jpg records kill layout.

Limits: no populated homepage top rollup exists locally; populated top-card and
large attacker/item/matchup cases use real Go-rendered DOM captures. Large opaque
cursor paging is likewise regression-covered rather than a second live page.
Browser Back resets homepage filters in both implementations; retain that state
as a final-audit improvement rather than claiming it works already. Final full
mobile/theme/navigation/focus/error/live/fitting/map/performance/reproducibility/
Docker audit still required. Remove allowJs and legacy inventory only at zero.

Current preview: /tmp/shrike-ts-public-kills-browser-facade-v3, PID 78866/session
39071, current 4451 and reference 4452; backend 4450 retained. Same guarded flags
as v2 plus -reference-public-kills. Secure TEST_DATABASE_URL required on launch.
Latest asset/binary embeds the filtered-feed transport lifecycle correction;
35 focused tests passed again after the final test refinement. Kill route returns
200. Fault rules cleared, all comparison tabs closed, user listeners untouched.


## Fitting TypeScript WIP, 2026-10-08

The preceding public-kill pass was verified progress. This pass ports eight more
original files completely: fit-capacity, fit-resources, fit-stats, fit-variants,
fit-wheel, fit-search-state, fits-catalog and fit-detail. 83/87 source typed, four
JS remain; accepted stays 57/87 with twenty-six typed files pending browser checks.
Fitting browser acceptance will run after the editor/search controllers are typed.

The SDE decoder's seven any-valued maps now have concrete types for type metadata,
physical nullable values, attributes/defaults, effects/modifiers, categories, groups
and market groups. These are the actual FlatBuffer fields already decoded; engine
calculation rules and serialized data are unchanged. Shared fitting render types
connect helpers to the existing Fit/ComputedItem/FitStats models. Capacity retains
strategic cruiser subsystem fallback, slot clamps and validity/hardpoint checks.
Bays narrow drone/cargo documents before state/quantity access. Wheel preserves all
31 slots, SVG paths/angles/viewBoxes, placeholder geometry, charge icons, selected/
preview/offline states and hardpoint/usage meters. Statistics retain capacitor,
weapon range, EWAR grouping, remote/tank/resistance/targeting/navigation/drone
content and remembered expansion. Suggestions retain overlap/ranking/resource/
hardpoint filtering, duplicate-module counts and observed/custom deltas.

Fit-search-state now checks URL JSON as unknown with typed role/type comparisons,
metric/sort/state models, shared numeric/EVE ID guards and eight-filter limits.
The original API query field names, zero bounds and NPC profiles round-trip.
Catalogue navigation uses Go-declared Stimulus and shared FragmentSections; sections
remain independent, retain usable cards on failure and restore query/history/hash
and choices through navigation. Repeated active choices leave a pending read
running. Saved fitting rating/comments have separate Go scopes. Ratings validate
seed/response data against generated API projections, retain source star/summary/
clear/preview behavior and use owned requests. Page exit prevents late session
lookups from initiating a write. Interrupted/ambiguous PUT/DELETE requires GET
reconciliation before another write, never replay. Comment import/mount stays lazy,
closed or detached sections ignore late imports and cached return resumes pending
work; child comment controllers own their existing reads/WS cleanup.

Real Go route captures under frontend/test/fixtures/fitting mount the actual new
scopes/actions. Their temporary capture test used a local in-process HTTP stub and
was removed. No database rows/jobs or production services changed.

Ten renderer/SDE regressions use real pinned 13.6.0 engine calculations and output
from the four original pure JS renderers. The initial differential run passed
byte-for-byte HTML/JSON comparisons. Original output is now saved in
render-source.json (152,270 bytes): normal tests require no private .data archive
and never regenerate expected output from the new implementation. Temporary
VM/tar capture code was removed. Tests also assert meaningful wheel slots/icons,
resource thresholds/clamps, drone states, EWAR/expansion, empty/structure views,
suggestion constraints and comparison actions. Sixteen Go-HTML-backed controller
regressions cover independent requests, races, retained cards/retry, history,
cancellation, rating auth/redirect/single-flight/clear/reconciliation and lazy
comment import/cleanup/retry. Existing capacity/search-state checks now use actual
typed models; two further URL-validation regressions cover malformed requirements
and invalid/zero statistics.

Strict TypeScript, all 777 tests in 108 files, relevant Go frontend/facade checks
and whitespace checks pass. The four remaining original JS files are fit-editor,
fits-search, map-live and map-page. Complete their full scope, fitting family
browser acceptance, twenty-six typed browser backlogs and the full final mobile/
theme/history/focus/error/loading/live/fitting/map/performance/reproducibility/
Docker audit. Remove allowJs/legacy inventory only at zero. Goal active.

Production asset/artifact checks passed: 118 entries, 103 chunks, 1,123,782 JS
bytes and unchanged 119,988 utility CSS bytes. Current comparison preview is
/tmp/shrike-ts-fitting-browser-facade, PID 85300/session 96991, 4451/current and
4452/reference; retained backend 4450 unchanged. Same guarded read-only database
flags as the public-kill checkpoint. /fits returns 200. Browser fitting acceptance
has not run this pass; source/test progress must not be reported as browser parity.
No temporary capture test or archive-execution test dependency remains.


## Fitting search TypeScript WIP, 2026-10-08

The preceding fitting helper/catalogue/detail turn was verified progress.
fits-search now has a Go-declared Stimulus scope with owned reads/timers and native
popover capture actions. 84/87 source typed, three JS remain; accepted remains
57/87, twenty-seven typed files pending live browser acceptance. Fitting family
acceptance remains scheduled after the editor conversion.

Initial state, role metadata, availability counts and fragment result metadata
enter as unknown and project into the existing generated API/filter models.
Malformed IDs/operators/counts, wrong availability hulls and wrong offsets do
not gain trusted types. Shared EVE ID/search results, metric/sort parsing, request
ownership, DOM template cloning and state-preserving moves are reused. Original
CSS/classes, icon sprite, source query keys and Go component markup remain.

Results, distributions, availability, item picker and global retries have
independent cancellation. Sorting reloads results while useful profile/availability
reads continue. Filter/hull changes cancel superseded work. Old cards remain on
failed updates; paging Retry retains the exact offset and appends to existing
row/image nodes. Loading status and aria-busy cover reads. New requirement/statistic
updates retain component/image nodes and keyboard focus; availability updates
restore focus on a surviving role. Empty/selected hull states, popular cards,
source picker event, role/type requirements, duplicate/eight-filter limits, bounded
count/operator controls, numeric shorthand, all metrics/NPC/capacitor fields,
distribution buckets, native choices/popover position/keyboard navigation and
URL/Back behavior are kept. Pagehide/disconnect cancels every request/debounce,
closes menus/pickers and ignores late responses. Cached return resumes unfinished
work without resetting drafts or valid rows.

Six new Go route/fragment captures use a local in-process stub: unselected and
selected Rifter pages, results at offsets 0/24, DPS profiles and popular hulls.
Temporary capture test removed. Seventeen regressions mount these actual
Go scopes/actions. The one initial failure was a visibility-stub mistake: it
pretended hidden retry buttons were visible in an unselected-hull popover.
The corrected keyboard test uses the selected page, respects hidden ancestors
and proves disabled choices are skipped. No production code was changed to make
that test pass. Normal tests require no private source archive.

Strict TypeScript, 17 focused regressions, all 794 tests in 109 files and Go
frontend/facade checks pass. No database rows/jobs or production services changed.
Browser search/editor/fullscreen/native-drag/keyboard comparisons have not run
this pass; do not count this source-only progress as browser acceptance. Next:
fit-editor, map-live/map-page, fitting browser acceptance and twenty-seven typed
browser backlogs, then the original full mobile/theme/history/focus/error/loading/
live/fitting/map/performance/reproducibility/Docker audit and allowJs/legacy removal
at zero. Goal active.

Latest assets/binary pass production artifact checks: 118 entries, 103 chunks,
1,132,063 JS bytes, unchanged 119,988 utility CSS bytes. Current preview is
/tmp/shrike-ts-fitting-search-browser-facade, PID 90059/session 97529; 4451/current,
4452/reference, backend 4450 retained, same guarded read-only fixture flags.
/fits/search?ship=587 returns 200. No temporary capture test remains. Fitting
browser acceptance is still pending; user 4000/4200 and production are untouched.

## Editor boundary work in progress, 2026-10-08

The previous fitting-search turn was verified progress. This pass establishes
checked editor state/API/document boundaries and wires them into the existing
live module; it is partial editor work, not a completed source conversion.
Source count remains 84/87, accepted 57/87, three JS files remain. Do not inflate
these counts because typed helpers now exist.

frontend/src/fit/editor-model.ts defines loaded editor state, catalog/slot/context
and observed/saved-document projections using existing Fit/SdeData/FitStats and
generated API models. It retains the original engine input (including Preview to
Active and grouped active/passive drones), workspace defaults/URL preference and
column widths. Incoming saved rows, observed families and bulk prices are checked
as unknown. Legacy missing quantity/charge and legitimate empty Go slice JSON
normalize to their existing empty/default semantics. No original calculation
rules, fit wheel/table markup, save protocol or rendering algorithms were replaced.

editor-codec.ts extracts the original v3 CSV parser and uses the shared encoder
already used by kill-detail. The editor now imports these helpers and the shared
slot constants instead of repeating encoder/decoder/engine input/JSON projection
logic. Gzip expansion is bounded at 1 MiB and the original compressed input/500
line, slot, drone and cargo bounds remain. Full share URLs also decode when fit
is after view or another query parameter; the old includes('?fit=') check missed
those links. The format and normal encoded bytes remain identical.

Thirteen new regressions cover Unicode/full-URL round trips, exact shared codec
bytes, invalid/oversized documents and compressed expansion, original parser
bounds/state fallback, engine input/drone expansion, workspace phone/explicit-view
behavior, independent histories, legacy saved rows, corrupt descriptions/items,
observed modules/charges/drones, empty Go slices and prices. All 807 tests in 110
files, strict TS and relevant Go frontend/facade checks pass. No database rows,
jobs, production services or user listeners were changed for implementation/tests.

Remaining editor work: convert its still-JS rendering/actions/state consumers and
event bindings completely, with owned asynchronous requests/timers/observers and
cleanup. Preserve cold clipboard activation, fullscreen relocation/state/scale,
wheel/table controls, catalog groups/search/hover previews, observed/custom fits,
undo/redo, context/rack/drone actions, table resize/focus, drag/drop and save flows.
Do not claim this component migrated or browser tested until that work and the
real desktop/phone interaction comparison pass. Then convert both map sources,
finish twenty-seven typed browser families and the full original final audit;
remove allowJs/legacy inventory only at zero. Goal active.

Production artifact checks pass: 118 entries, 104 chunks, 1,135,560 JS bytes;
utility CSS remains 119,988 bytes. Current preview is
/tmp/shrike-ts-editor-boundaries-browser-facade, PID 94360/session 73159,
4451/current and 4452/reference, retained backend 4450 unchanged, same guarded
read-only fixture flags. /fits/create returns 200. No editor browser acceptance
was performed here. The still-JS module is 1,305 lines and now calls the shared
checked boundaries; finish its actual component/lifecycle conversion next.

### Typed editor and live/AIID feed checkpoint, 2026-10-08

The remaining editor rendering and interaction code was lifted from the existing
source into `fit/editor-runtime.ts` with real Fit/SDE/engine/DOM types. Its Stimulus
owner registers the original callback bodies as actions and owns requests,
observers and timers. Fullscreen reparenting retains the editor; actual removal
cleans up. Page exit invalidates calculation, preview, codec and clipboard results.
Save/login response fields are checked, saves are single flight, newer workspace
or form drafts survive late replies, and uncertain writes are never replayed on
cached return. Twelve Go-markup/pinned 13.6.0 engine tests cover hull selection,
wheel/table, keyboard history, slot/hull drops, context charges/state/racks,
fullscreen, cold clipboard, save races and hide/show/disconnect.

Map live/AIID now shares the typed relay and checked live-kill projection. AIID
stays armed while hidden; live traffic pauses. Scope updates preserve rows and
freshness history; freshness, 60-row display/200-event buffer and near/outer alarm
bands match the original. Factory teardown removes owned rows and blocks late
reads/hover/update callbacks, preventing duplicates when recreated. Eight real-Go-
DOM tests replace the old unchecked VM harness and cover these behaviors.

Chrome source comparison uses opt-in `-reference-fittings` with the private
pre-TypeScript archive, identical current Go markup/CSS and pinned engine. The
retained binary predates later fitting fixes, so its old selection strip is not
an appropriate source checkpoint. Fixed iframe sizes avoid differing tab heights:
1440x900 fullscreen wheel is 339x339 above the resource bar in both implementations;
normalized wheel/statistics/resources match exactly. At 390x844 the wheel is
374x374, the page scroll width is 390 and remembered desktop fullscreen does not
activate. Table contents match; live table Delete and Cmd+Z restore the module;
the save dialog opens with visibility choices. No save was submitted. Both
implementations report unavailable observed fits because this isolated facade
has no populated universe reader. Do not count that error state as populated
observed-fit acceptance or declare the whole fitting family accepted.

Screenshots: `/tmp/shrike-fitting-review/editor-desktop-source-comparison.png`,
`editor-phone-current.png`, and the earlier full-window current/reference views.
The fault controls are clear. Preview remains read-only for entity/public-kill
queries, backed by the existing disposable database; user ports 4000/4200 and the
retained 4450 process are unchanged. Strict TypeScript, all 826 tests in 111 files,
Go frontend/facade tests and whitespace checks pass. Source is 86/87 typed; only
map-page.js remains. Accepted count stays 57/87, leaving twenty-nine typed browser
families open. Continue map-page, populated fitting and browser backlogs, the full
final mobile/theme/navigation/live/performance/build audit, then remove allowJs
and legacy exceptions at zero. Goal remains active.

Follow-up phone interaction also compared explicit Expand → Ship → Exit in both
implementations: fullscreen wheel is 366x366, page width stays 390, and Exit clears
the expanded class. The final production build contains 118 entries/104 shared
chunks (1,142,444 JS bytes; unchanged 119,988-byte utility CSS). Asset/source audits
and rebuilt Go tests pass. Current owned preview executable:
`/tmp/shrike-ts-editor-live-browser-facade` on 4451, with `-reference-fittings` on
4452; retained backend 4450 stays running. Only `map-page.js` remains in the legacy
inventory. Continue from that source and the recorded acceptance backlog.

### All browser sources typed; validation paused, 2026-10-09

map-page is now a real Stimulus component with the existing rendering/control
bodies, checked generated-API projections, shared scope/layer/watch parsing,
owned requests and debounce timers, delegated sovereignty row focus/image
fallback, and renderer/feed teardown. Old request/import/canvas callbacks cannot
commit, navigate or show hover cards after replacement or removal. Watch updates
retain their feed and freshness history; page exit releases it. Twelve actual-Go-
markup regression tests cover controls, URL/history, query races, invalid data,
search, anchors, alarms, sovereignty, callbacks and hide/show/disconnect.

All 87 original sources are checked. Legacy inventory and compiler JS exceptions
are deleted. AST source audit bans authored any types and suppression comments
across src, rejects JS, and requires strict/noImplicitAny. wasm-bindgen's missing
background declaration is represented by a narrow adapter ABI declaration backed
by the pinned Rust structs and real-engine tests; no wildcard JS declaration or
blanket cast was introduced. All 838 tests in 112 files and strict TS pass.
Go frontend/facade checks pass; production assets are 118 entries, 104 chunks,
1,150,671 JS bytes, with unchanged utility CSS. The full local production Docker
build and runtime Go/Rust/no-Node/no-Bun smoke pass. Image tag is
shrike:typescript-audit-20261008; no image was published or deployed.

Chrome map comparison uses -reference-maps with the four named archived original
modules; only absolute module import locations are relocated. Current/source use
identical Go markup/CSS and explicit local GET fixtures. Geography at 1440x900
matches DOM/text exactly: 1198px root, 700px stage, two canvas layers, four systems
and two regions. Security/danger and routes/systems/labels toggles match; Jita
search points at /system/30000142. Top-level canvas click reaches
/map/region/10000002 retaining base/activity/window/toggles; the actual region page
loads, and browser Back restores map controls. Initial iframe navigation showed a
blocked frame because new routes intentionally lack the fixture framing marker;
use top-level tabs for navigation checks. Sovereignty begins populated and both
implementations show one alliance with two claimed systems. Continue sovereignty
hover/search, live/AIID relay/reconnect and fixed phone layouts before acceptance.

Screenshots: /tmp/shrike-map-review/geography-current.png and geography-source.png.
Browser acceptance stays 57/87, with thirty typed modules awaiting complete
acceptance. The source conversion is complete, not the overall goal. Continue
maps, populated fitting, existing browser backlog, full final site/mobile/theme/
navigation/focus/loading/live/performance audit and bundle reproducibility.
Current preview is /tmp/shrike-ts-maps-browser-facade (4451), archived reference
4452, retained backend 4450. User listeners are unchanged. Overrides cleared and
comparison tabs closed before the explicit bedtime pause. Do not restart retained
4450 or use fixture/warm timings as production/cold performance evidence.


### Resumed map acceptance, 2026-10-09

Both original map modules are accepted, advancing browser acceptance to 59/87;
all 87 sources remain strict TypeScript and 28 need browser acceptance. Current
and archived scripts use identical Go markup/CSS and explicit local map fixtures.
1440x900 and 390x844 comparisons cover geography, sovereignty, live and AIID,
matching panels, labels, search and watch controls. The phone document stays
379px wide inside its 390px fixture with an 11px scrollbar; no page overflow.
Source/live clocks are sampled at slightly different times and are not asserted
as identical seconds. Live rows deduplicate, reconnect and retain streamed rows
through AIID watch changes. Mode exit closes both owned sockets. Empty AIID areas
show the original guidance. Delayed 503 retains the prior map and reports an error;
a successful subsequent window change recovers without duplicate canvases.

Fixed map search focus: input-to-result focus keeps its menu visible; leaving the
associated field/menu closes it after the existing delay. Three regressions use
actual Go DOM. The old script also had the unconditional blur timeout, so this is
an intentional accessibility improvement. All 841 tests in 112 files and strict
TS pass. Broad `env -u TEST_DATABASE_URL go test ./...` passes, excluding database
integration. Final canvas gesture/site audit still remains; no native touch/pinch
claim is made. Screenshots: /tmp/shrike-map-review/sovereignty-phone-current.png,
aiid-phone-current.png, aiid-desktop-current.png and live-desktop-current.png.

Next is populated fitting acceptance: the comparison facade gains a guarded
read-only UniverseReader and an in-memory named Rifter family only under the
existing opt-in reference-fittings mode. Real Go variants markup replaces the
unavailable-reader placeholder. Existing nonempty family data and read failures
are preserved. No fixture rows or production data are changed. Goal remains active.


### Populated fitting and boundary regression, 2026-10-09

The guarded opt-in fitting facade now supplies UniverseReader, routes variants
locally and fills only missing Rifter fixture metadata/family data in memory.
Original catalogue/search scripts and their named helper imports are available
through the same archived comparison renderer. Its public APIs remain GET/HEAD
only and credentials are stripped on source rendering. Current preview is
/tmp/shrike-ts-fitting-family-browser-facade, PID 23708/session 24998. Retained
4450 and user ports are unchanged. Availability is currently overridden locally.

Populated Go variants exposed charge_type_id:0 being rejected by the TypeScript
reader. It now treats zero/null/absent as no charge while rejecting invalid IDs.
Boundary and real-Go-markup/pinned-engine tests cover this plus observed fitting
preview, modified reset and custom restoration. Browser desktop comparison shows
identical suggestions, observed/custom diff, calculated stats and wheel output.
Keyboard activation was used for the preview link: its central overlap contains
the separate All variants link, so a center click opened that legitimate link.
Those two comparison-created tabs were closed. No product click bug was inferred.
Phone normal and fullscreen editor layout matches; shared CSS intentionally hides
observed variants below 1200px and fullscreen stats below 900px. Record as final
mobile usability review, not a successful phone preview interaction.

Catalogue matches populated flavor, doctrine, community and rated cards on desktop
and phone; final-blow/loss modes, corporation controls and top-rated/error retries
work. New refresh failures retain prior cards, an intentional source improvement.
Search is partial: populated Rifter result and editor handoff, min_ehp=50000 from
50k, capacitor flag, sorting, damage-control requirement addition, count=2/op='='
and fixed 390px layouts/URLs match. Result fixtures are explicit presentation and
transport scenarios, not assertions that a 3K-EHP fixture passes a 50K SQL filter.
Availability and result fixtures were applied separately; the single rule replaced
the former response. Empty real results during role checks are legitimate. Finish
items/distributions/paging/loading/retry/Back, saved detail and remaining editor
flows before accepting fitting. Formal acceptance remains 59/87.

All 843 tests in 112 files and strict TS pass after both new regressions. Go
frontend/facade tests and rebuilt production assets pass (118 entries/104 chunks,
1,151,071 JS bytes; 119,988 CSS bytes). Broad Go checks passed before local fixture
additions with TEST_DATABASE_URL unset. Current Docker audit image predates these
changes; rebuild at final completion. Screenshots are in /tmp/shrike-fitting-review,
including observed-desktop-current.png, editor-phone-charge-fixed.png,
catalogue-rated-phone-current.png and search-role-phone-current.png. Unknown
Postgres reltuples produces -1 quick estimates in the tiny fixture; add appropriate
unknown-count presentation to the final audit. No production writes occurred.


Follow-up: phone result cards, empty results after switching fixture rules,
requirement removal and delayed result failure/retry match source, retaining cards.
Current broad Go suite passes after fixture changes with database integration
explicitly disabled. Repeated production build reproduces the artifact manifest,
output hashes, utility CSS and icon data. 843/112 tests and strict TS remain the
latest full checkpoint. Browser acceptance stays 59/87. Named search response is
/tmp/shrike-fit-search-response.json; overrides are now clear and tabs closed.
Current preview remains running, goal active; no user listeners were changed.
Corrected observed preview screenshot:
/tmp/shrike-fitting-review/observed-preview-charge-fixed.png. Continue remaining
search items/profiles/paging/Back and saved/editor interactions before accepting
fitting; then the eighteen prior modules and full final audit.


### Fitting family accepted and saved-page startup fixed, 2026-10-09

Acceptance is 69/87, all 87 sources strict, eighteen older modules still pending.
The complete fitting family now has source browser comparisons plus actual Go-DOM
and pinned-engine regressions. Search exact items, DPS histogram range, 24+1 paging,
append failure/retention/retry and navigation Back pass; catalogue selection Back
also passes. Phone widths stay 379px within the 390px/11px-scrollbar fixture.

Saved page validation caught mandatory lookup of an intentionally absent shortcut
help node. It is now queried as optional and keyboard activation guarded; required
nodes remain checked. A new combined editor test uses current captured Go saved
Vexor markup and real dogma/SDE, proving module/charge, 3 active/2 stored drones,
cargo and computed stats survive initialization. Failure causes now appear in the
console as well as the existing error state. Original editor/detail/comment scripts
run against identical Go markup in a reference wrapper tested to remove duplicate
controller scopes and credentials. Desktop/phone stats/wheel/ratings/comments match.
Private and missing fit pages have matching errors. Real local rating save/clear
passes and the original page observes its aggregate. The valid existing fixture
owner account was used; the reader session had expired, and no session was extended.
Direct /auth/me browser navigation was blocked by the client and not retried or
bypassed; session ownership was verified through normal rendered owner controls.

A uniquely named private Rifter was created, updated, loaded via /fit/<id>, verified
in the disposable DB and rejected anonymously. Only that exact ID/name/fixture
owner row was removed afterward. The prior public fixture rating was cleared.
No production rows changed; broad Go checks do not use the test DB. Fixture account
is anonymous, no fault rules remain, and temporary comparison/save tabs are closed.
The user's interacted-with typed fitting preview stays open as a deliverable.

Native application dragging failed to produce a drop in both source and current.
The user confirmed all three actual drag paths in the typed preview: catalogue to
slot, moving between slots, and catalogue to ship image. Native cursor/keyboard
input interfered with their work on another monitor, so stop native desktop input;
use only scoped browser connector APIs for further UI validation. Do not lose this
constraint at compaction. Existing drop/history tests remain as regression coverage.

All 844 tests in 112 files, strict TS, built assets, frontend/facade and broad Go
checks pass; broad Go uses `env -u TEST_DATABASE_URL go test ./...`, skipping DB
integration. Assets are 118 entries/104 chunks, 1,151,143 JS bytes; CSS 119,988.
Preview /tmp/shrike-ts-fitting-saved-fix-browser-facade, PID 30361/session 77793;
4451/current, 4452/source, retained 4450 unchanged. Public 4200 now returns healthy
and its populated live homepage was observed; 4205 was only an earlier unavailable
upstream. Prepare subsequent public read-only comparisons from 4200 without
changing the user's server. Screenshots:
/tmp/shrike-fitting-review/saved-fit-phone-fixed.png,
search-page-two-phone-current.png and private-save-reload-current.png.

Remaining original modules: wars, war-detail, faction-war, faction-war-map,
roam-composition, roam-report, scans, legacy, status, docs, mcp, graph, graph-render,
rankings, statistics, market, universe and system. Full final audits and updated
Docker/dev/integration/reproducibility evidence remain required. Goal stays active.


### War family browser acceptance, 2026-10-09

 Four original modules are now
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


### Roam Report browser acceptance, 2026-10-09

 Both original report/composition
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


### Scans/docs/MCP acceptance, 2026-10-09

 Three original modules are accepted,
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


### Statistics/rankings/graph browser acceptance, 2026-10-09

 Four original
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


## Browser migration completion audit, 2026-10-09

All 87 original modules are strict TypeScript and have desktop/phone family
acceptance. The separate final audit now passes. This closes the browser migration
goal; deeper Memgraph ingestion, query design and intelligence work remain deferred.
Existing historical WIP/checkpoint notes above describe earlier states.

The final map audit reproduced the earlier ambiguous pointer hit: The Forge's
button centre was covered by The Citadel's button, and selected that neighbour.
A small native location popover now offers the actual overlapping markers before
selection. It retains geographic positions and direct keyboard/exposed-marker
activation. Native Escape restores focus; replacing markers/maps dismisses old
choices. A dialog fallback and regressions cover unsupported Popover environments.
Labels remain plain text and colours use existing theme variables. Map and
fitting-search controls share one viewport positioning helper, including scrollbar
space. This adds no API reads or per-frame layout work; hit testing runs only on
pointer activation.

Actual public all-time data rendered 660,201 matching battles / 7,859 systems,
with New Eden's 5,484 systems / 69 regions and 67 region markers. Desktop pointer
selection now explicitly reaches The Forge. At 390px the picker occupies x51–371
inside 379px content, and at 320px it occupies x8–301 inside 309px content, with
no horizontal overflow. Escape restores the physical marker's focus. Keyboard
Jita selection opens the actual fifty-card list; switching back to battle cards
removes the open picker. Current narrow fitting-search Sort and all eleven metric
choices also fit 309px. Screenshots: /tmp/shrike-final-map-review/
crowded-marker-choice.png, crowded-desktop-picker.png and crowded-phone-picker.png.

A guarded read-only local fixture pool supplies the final account/live tests;
only the owned in-memory relay is writable. Its historical kills are legitimately
outside the current homepage window, so SSR initially has no rows. Two actual
fixture kill summaries, flattened to the relay contract with timestamps changed
only in memory, verify first insertion, duplicate suppression, ordering, existing
link focus retention, involved count/people icon, forced disconnect and resubscribe,
and delivery after reconnect. Two relay connections are expected: all-kills and
the account's private tracker subscriptions. No killmail or notification was
persisted by these broadcasts.

A five-second local fragment delay/503 shows Loading, the top bar and the clicked
100-row control's pending state while retaining both live rows. Failure clears
feedback, retains both rows and exposes Retry; Retry restores the selected limit
and removes the error. Its successful empty result is the fixture's authoritative
current-window response, not fabricated live persistence. Keyboard activation of
the full-row link reaches the real fixture kill with its navigation context;
canonical metadata excludes nav. Back returns the fresh homepage, clears loading
and resubscribes. Its full-reload limit reset matches the archived home script,
which stores limits in page state rather than its URL; no new persistence feature
or stronger Back-state claim is implied. No controller connection/action errors.

Final generation audit corrected three stale generated files (editor, catalogue,
map), then confirmed all 93 generated templates reproduce unchanged. Current
catalogue -1/-1/1/0 counters display unavailable/unavailable/1/0. Regenerated map and
editor scopes initialize; actual Rifter selection renders its available slots and
calculated statistics without status/controller errors. Production Docker already
regenerates these files before compilation and its final runtime covers that state.

| Objective requirement | Current authoritative evidence |
| --- | --- |
| Convert the original 87 modules | All 87 TODO names map to existing browser .ts files; source audit finds zero authored JS. /tmp/shrike-final-objective-audit.json records the exact names. |
| Strict checks without suppression/blanket any | strict/noImplicitAny compile passes; browserSource.test.ts parses every authored src TS AST and rejects any and suppression comments. |
| Remove JS/legacy exceptions | No allowJs, checkJs or JS include; browser-legacy.json absent; source audit passes. |
| Consolidate real reusable behavior | Shared HTML/JSON/DOM/API/session/loading/fragment/fit-codec/theme/map helpers; new map/fitting-search positioning helper has two actual consumers. Pure transforms/transport remain normal typed modules. |
| Scoped lifecycle with Stimulus | One application registry; Go-declared actions/targets/outlets; RequestComponent and FragmentSections own body reads, cancellation and shared busy counts. Editor, renderer and family tests prove timers/observers/drafts/stale responses/teardown. |
| Retain Go/templ, DOM/CSS and public contracts | Every original family has source comparisons on current Go markup/data; all page rendering remains Go. Current source generation, Go tests and 385 local API/ingress integrations pass. The marker picker is the documented final UX fix. |
| Themes | Shared theme map/bootstrap, theme tests, family comparisons and final saved/custom phone fragment preview/restoration; picker uses those variables. |
| Shared shell/page/lazy bundles | Entry/import/artifact test covers the complete source graph; 118 entries, 105 chunks, 223 physical JS outputs. Appearance stays synchronous; feature renderers remain lazy. |
| Automatic development builds/reload | Real isolated make dev/Air/source watcher passes TS/CSS/template reload, failed-build retention, retry/recovery and source restoration. /tmp/shrike-dev-audit-location.json. |
| Production builds/runtime | Latest Docker image shrike:typescript-audit-20261009-markers passes eight HTTP routes, 96 referenced assets, all 223 versioned JS bodies/ETags/immutable cache headers, authored map CSS, absent dev surfaces, UID1000 Go/Rust without Node/Bun, and all 16 fitting snapshots. /tmp/shrike-final-marker-runtime.json. |
| Each family's desktop/mobile behavior | Accepted family checkpoints span shell/account/admin, forms/search/comments/campaigns/battles, public/entity feeds, fitting/map, wars/Roam/tools/docs, analysis/status/archive, market and universe/location. Final accepted count 87/87; later source changes have their targeted current checks. |
| Full interaction/error/loading/focus/navigation/live/fitting/map pass | Family evidence plus final balance/theme/reload, phone fullscreen/cold search, native-drag user confirmation, crowded maps, popover teardown, live reconnect/dedup/focus, visible loading/failure/Retry, row keyboard navigation and Back checks. Chrome desktop and 320/390/1024px viewport evidence; no physical-device/all-browser guarantee is asserted. |
| Relevant performance validation | Source/renderer lifecycle cleanup, no initial SSR-widget refetch, bounded owned reads, lazy bundles and unchanged shell/editor static budgets. Current map-picker work is O(markers) only per ambiguous click. No warm-cache or fixture timing is represented as cold production speed. /tmp/shrike-final-marker-transfer.json. |
| Automated tests/checks and reproducibility | 855 frontend tests in 113 files; strict TS; broad Go tests and vet; WASM pin/checksums; 385 DB/API/ingress tests with one explicit imported-market-data skip. 230 tracked files reproduce and all 223 output hashes/sizes verify. |
| Tracking, full scope and cleanup | TODO gates and architecture/development/review guidance are current. All owned tabs and four facade ports are closed; Docker/dev copies, owned containers/databases are cleaned. User 4200, retained 4450, shared dependencies and original .env are preserved. No production writes, commit, release or deployment. |

The imported-market integration skip is explicit: the fresh integration database
contains no imported market orders. Actual populated market browser acceptance is
recorded separately above. Broader backend/graph refinements and deployment remain
outside this completed migration goal. Final evidence inventory:
/tmp/shrike-final-objective-audit.json, /tmp/shrike-final-marker-repro.json,
/tmp/shrike-final-marker-runtime.json and /tmp/shrike-final-template-audit.json.
