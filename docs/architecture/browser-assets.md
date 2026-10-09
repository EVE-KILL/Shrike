# Browser source and builds

Go/templ owns documents and HTML fragments. Browser behavior is an ES module
project under `frontend/`, compiled with TypeScript/esbuild. Stimulus manages
component targets, actions and connection/disconnection on server-rendered HTML;
there is no client renderer or page server. Node is a development/build dependency only.

## Source ownership

- `frontend/src/browser`: page controllers, shared browser primitives and entry
  points. All authored controllers compile as `.ts` through the same graph.
- `frontend/src/fit`, `frontend/src/utils`, `frontend/src/models`,
  `frontend/shared`: shared document conversions, models and pure helpers.
- `frontend/dogma`: the shared Rust engine's WASM facade and fitting metadata.
- `frontend/scripts`: asset builders and Go metadata generation.
- `internal/frontend/static`: generated JavaScript plus styles, sprites and public
  media embedded by Go. Edit JavaScript in the source project, not this directory.

`site.js` combines the common shell, navigation feedback, account, announcements,
killmail submission and homepage behavior. Other pages load their feature entry
points separately. Shared dependencies become generated
chunks, so session state and other module singletons remain shared across page
widgets. Absolute `/assets/*.js` dynamic imports resolve to source modules during
building. Page-specific tools such as maps retain lazy loading. The appearance
bootstrap is compiled separately as a small synchronous IIFE to apply theme and
background preferences before paint; all other document scripts are ES modules.

`frontend/browser-artifacts.json` records every generated module/map with its
checksum and size. Builds remove obsolete owned chunks; unexpected edits to an
obsolete generated file stop cleanup. Tests validate checksums, entry coverage and
relative module dependencies. Only generated files are removed.

## Development

Install Node 24+ and run `npm --prefix frontend ci --ignore-scripts` once.
`make dev` starts the source watcher and Air together on `DEV_PORT` (default 4000).
There is no second HTTP server. Changes to browser sources, styles, relevant Go
page sources and the icon sprite rebuild the assets; Air embeds them and the
existing reload channel refreshes open pages. Saves are debounced and builds are
queued. Unchanged files keep their timestamps to avoid reload loops.
The complete browser build writes one development marker before Air rebuilds;
individual generated chunks/maps do not start overlapping Go builds. Reloads
currently include Go recompilation because assets are embedded: the measured
source-edit smoke took about 29 seconds in this checkout. This is automatic page
reload, not element-level HMR.

Development output is readable and includes external source maps. Production
builds are minified and remove owned development maps. `npm --prefix frontend
run dev` starts only the asset watcher; `npm --prefix frontend run build` builds
production assets. Existing Makefile asset-group targets remain available;
JavaScript groups rebuild the complete shared graph, while CSS can build alone.

## Production and checks

`make build` compiles browser assets, templates, the native worker and Go. Docker
has an isolated Node build stage, then embeds its output in Go. The final image
contains Go/native Rust and data; it contains neither Node nor Bun.

Run browser tests and TypeScript checks, Go tests/vet, and the committed-WASM
checksum check. CI regenerates assets/Go metadata and checks the artifact manifest
for drift. Source-level tests cover behavior; the artifact test covers shipping
integrity, and browser checks cover integration.

## Consolidation review, 2026-10-07

The previous tree had 123 JavaScript assets and 15,066 lines. The source inventory
is retained in `.data/browser-inventory-20261007.json`; a private source archive
preserves the original tree before movement. The shared/minified graph reduced
JavaScript from 1,250,543 bytes to about 935,000 bytes (about 25%), including Stimulus. These are totals
across the site, not a claim about each page's transfer size or load time; fitting
WASM/data sizes are unchanged. The production common-shell graph is 61,890 bytes
across 11 requests; shell plus editor is 202,137 bytes across 23 requests. These
are uncompressed static imports, excluding optional dynamic imports, appearance,
development diagnostics, data, CSS and WASM. A single all-site bundle would make
each page download every feature.

Request/error handling, session lookup, loading state, focus-preserving fragments,
default-tab selection, coalition activity and icon selection now have typed
primitives. The editor's save/load and EFT helpers use canonical TypeScript
implementations. Passive coalition/campaign/comment widgets share the document's
session lookup; explicit reauthentication checks remain fresh. Failed lookups are
retryable. Combat fragments preserve existing cards, filter state and geometry.
Sixteen duplicate HTML escape implementations now use one typed helper, the
fitting search shares its canonical fuzzy matcher, and theme bootstrap/settings
share one variable map. Roam aggregation has typed pilot/ship/aggregate models.

All **87 original source modules now pass strict TypeScript and automated
checks**. No authored browser JavaScript remains; `allowJs`, `checkJs`, the JS
include and the legacy inventory have been removed. The source audit rejects new
JS, explicit any types and suppression comments across authored src TypeScript.
All 87 modules have completed family desktop/mobile browser acceptance. The
separate final cross-site audit has passed, with its concrete gates and evidence
in TODO.md and the frontend review. The settings entry imports scoped
controllers for preferences, appearance, sessions, bios, private wallet and comments;
the domain editor owns its bindings, uploads and searches. Domain patch helpers
use generated models and explicit snapshots rather than unchecked any values.
The fitting editor preserves the original render/interaction functions behind
an owned Stimulus component. Requests, clocks, ResizeObservers and global DOM
actions stop on exit; fullscreen moves retain the same workspace. Save responses
cannot overwrite newer drafts, and interrupted writes are not automatically
replayed. The live/AIID feed uses the existing shared relay transport, retains
the original freshness/alarm rules and disposes its rows, reads and timers.
Tests prevent adding JavaScript or bypassing TypeScript checks; the compiler
requires strict and noImplicitAny. The Rust background glue has a specific typed
application ABI declaration matching the pinned adapter structs; generated
FlatBuffers/wasm-bindgen vendor code remains external to authored browser logic.

## Controller pattern

`src/browser/controllers/application.ts` starts one shared Stimulus application
and registers a controller once. Each page imports only the controllers it uses.
Go templates declare `data-controller`, targets and actions; Stimulus attaches
to newly inserted fragments and removes event bindings when their scope leaves
the document. Select menus, category filtering, legacy attacker controls and
coalition activity now use this lifecycle. Controller-owned asynchronous work
must abort on disconnect and ignore obsolete responses after reconnection.
`RequestComponent` owns requests through body consumption and cancels timers;
`FragmentTabsController` shares admin/settings panel caching, history and retry.
Stimulus outlets coordinate the admin ESI summary with its parent, including
SSR startup where child modules can register before the parent.
`AdminSectionComponent` shares hidden-panel request cancellation and restoration.
Admin editors use scoped menus/dialogs, typed payloads and operation tokens; newer
drafts survive in-flight saves. The shared date picker preserves UTC values while
changing the display zone; floating panels respect scrolling modal boundaries.
Advanced search retains the source rendering/filter behavior through typed UI
functions; Stimulus owns all DOM/window actions, request cancellation and timers.
Saved searches communicate through scoped current/load events rather than a
manual initializer or parent registry. Canonical filter models and saved documents
use checked fields and generated API contracts; stored documents contain no any
values. Account changes, revisions and interrupted requests retain their scope.

Keep pure transformations outside controllers and share generated API models
where practical. Preserve existing DOM/CSS and tested interaction behavior when
moving a page family. Stimulus is for browser behavior; templ continues to own
documents and fragments. Turbo/client routing is not required by this setup.
See [the official introduction](https://stimulus.hotwired.dev/handbook/introduction)
and [TypeScript controller guidance](https://stimulus.hotwired.dev/reference/using-typescript).

All 87 original handwritten modules are strict TypeScript and have completed
family-level desktop/mobile and behavior comparisons. The final site-wide and
build/runtime audit has also passed; see TODO.md and the frontend review for the
requirement-by-requirement completion evidence. Typed Go read services can separately
replace remaining in-process API/JSON adapters without changing authorization,
domain or cache policy.

Campaign components use typed entity/location guards, editor models and generated
API request/response types. Go declares Stimulus actions for the directory,
creator, owner/prize controls, detail tabs and activity chart. Requests and
background polling stop on exit; interrupted writes are never automatically
replayed. Newer editor text and contribution amounts survive successful saves
and fragment refreshes. Chart dimensions adapt to narrow screens and the shared
floating-panel helper leaves space for the fixed navbar outside native dialogs.

Controllers sharing a component root also share its request busy count. Completing or canceling one controller's request cannot clear `aria-busy` while another controller still owns an outstanding request.


Battle composition/detail/replay/generator/directory use Go-declared Stimulus
scopes. Shared typed boundaries cover composition, replay coordinates/landmarks,
participant lists and the complete preview/save document. Replay schedules frames
only during playback; visibility/disconnect stop work and release pointer capture.
The generator retains newer drafts during saves and never automatically repeats
an interrupted write. Battle maps load geography separately from markers, retain
the working renderer on fragment failures and dispose stale callbacks.


Roam Report uses scoped report/form/composition controllers. The form's lifecycle
root uses display:contents around its native form, retaining its layout while
avoiding form named-property scope aliases. Report saves retain newer drafts,
including edits made during the subsequent HTML read; interrupted writes never
automatically repeat. Battle and Roam maps share scope validation and one lazy,
retryable renderer import. Pure composition tests import their shared math module
directly, keeping calculations independent of DOM initialization.

Docs uses one checked OpenAPI adapter for the explorer and shared tests. Endpoint
cards keep their original classes and create fields only when opened. Scoped
controllers own entity searches, request/body reads, bounded stream previews and
clipboard feedback. RequestComponent.response allows inspecting HTTP errors
while retaining ownership through consumption. MCP schemas remain lazy/cached,
with close/unmount cancellation and explicit retry. Both pages preserve drafts
and read state on return; interrupted writes cannot automatically repeat.

Graph tools retain the original markup/formatter module and use typed projections
of the actual Go payloads. Scoped search controllers own debounce, selection and
cleanup; per-tool caches retain result DOM across tabs. Rankings owns its hourly
timer and suppresses duplicate refreshes while hidden/pending. Statistics retains
SSR/cached nodes and limits reads to four active requests; hiding cards or changing
periods cancels their work and releases slots immediately.


Market scopes live on reusable Go fragments. The browser-tree component owns
lazy category reads/cache and sidebar state; market search owns debounce, parallel
ship/item reads and keyboard/outside cleanup. The item-market component validates
identity/view/filter markers before commit, combines pending filter selections,
retains order sorting and handles standalone Back history. Item-page Back remains
owned by its parent tab navigator. This avoids competing reads during history
navigation. Controllers use shared request ownership and fragment focus handling;
no global market factory or manual reinitialization is required.


Item/group pages declare a universe navigation scope around their persistent hero
and tabs. Reusable fitting fragments declare an item-fits scope; supporting reads,
filters, drafts and histograms belong to that fragment and end on replacement.
Supporting HTML includes type/query markers to prevent mismatched commits. Shared
fit-filter helpers provide shorthand parsing and repair sort selection for item
filters and fitting search. New route checks prevent delayed fitting/market reads
from writing URL filters after Back moves to another tab.

## Final-audit build, 2026-10-09

The latest production graph contains 118 entries and 105 shared chunks: 223
physical JS files totaling 1,154,057 bytes, plus 120,024 bytes of utility CSS.
All 230 tracked manifest/output/CSS/icon/generated-data files reproduce exactly
(the final check includes the authored map stylesheet explicitly).
The totals cover the entire site; pages load their own entry graph.

Current static-import graphs, including the synchronous appearance bootstrap:

| Page graph | JS modules | Uncompressed JS bytes |
| --- | ---: | ---: |
| Common shell | 22 | 138,052 |
| Shell and fitting editor | 35 | 289,341 |
| Shell and map entry | 26 | 167,338 |

These are artifact sizes, excluding optional dynamic imports, data, WASM, CSS and
development diagnostics. They are not measured network transfer sizes or cold-load
timings. The earlier consolidation numbers above describe that earlier build.

The actual make dev/Air watcher passes TypeScript, CSS and template reload, failed
build retention, and successful recovery in an isolated copied workspace. The
latest Docker image passes production HTTP/assets, absence of dev surfaces, and
all sixteen native fitting snapshots. Its runtime remains Go/native Rust with no
Node or Bun. Strict TS and all 855 frontend tests pass; local API/ingress database
integrations pass 385 tests with one explicit imported-market-data skip. Final
cross-site acceptance and the objective audit are recorded as complete in TODO.md
and the frontend review.


Location pages reuse FragmentSections with explicit root replacement, retaining
existing Go fragment roots and shared focus/request/error ownership. KillPages
shares opaque cursor history and returned kind/page/limit validation between
item/group and system/region/constellation pages. Retrying independent location
sections does not restart their siblings. Blog pagination is scoped, retains
cards during reads/errors, deduplicates card URLs and stops on an empty cursor.
Card/detail date scopes initialize on fragment insertion without manual factories.


General search uses generated API projections with runtime guards for supported
entity kinds, IDs, optional affiliations and counters. Its scoped controller owns
debounce/body reads/history/error retry; filter changes reuse existing card/image
nodes. The shared search ID parser also serves market and entity pickers. Valid
search cards preserve their existing classes, image variants and entity routes.
The public error panel retains usable prior results and offers Retry.


Kill submission has a shared document controller and a scoped /post controller.
The shared component owns submission transport, durable pending IDs, reconnecting
relay subscriptions and nonoverlapping existence checks; the form owns controls,
clipboard reads and result presentation. IDs are staged before POST so interrupted
responses can reconcile with reads on return without repeating writes. Ready
notices use retained DOM nodes and one expiration timer, preserving loaded images
and keyboard focus as more kills arrive. Toast/relay fields are validated generated
API projections and are rendered through text nodes. The local parity facade's
opt-in submission fixture handles synthetic queue/exists/detail traffic in memory
before either proxy; production data and job dispatch are never involved.


Coalition directory/picker/editor behavior uses one scope per rendered component,
with shared session, search-model and request ownership. Alliance chip/card DOM is
retained during filtering and newer edits remain mounted through successful server
refreshes; published baselines update independently of those drafts. Matching slug
and nondecreasing revision are required before replacing detail HTML. Interrupted
writes/HTML reads reconcile with GET on return before another edit can be submitted.
The pageshow action reapplies the browser-restored directory query after Back.
Combat totals keep their independent fragment scope and retry behavior. The opt-in
parity reference reads only the two original coalition scripts from the private
pre-migration archive; it never serves other archive contents or relays mutations.


Entity pages now declare persistent navigation/valuable/ship-class scopes and one
scope per tab body. Fragment insertion mounts the new body without manual widget
initializers; removal ends its reads, sockets and timers. Preferences activate the
initial body only after resolving the default tab. Persistent hero/widget nodes
remain mounted during tab navigation. Shared FragmentSections, KillPages, relay
transport, search IDs and number formatting supply common behavior. Organization
intel retains its sections while refreshed stats replace their slots. Pending
member filters combine; failed reads retain working content with Retry. Empty Go
militia responses remain a successful empty state. Entity's strict source, DOM and desktop/mobile browser checks pass.


Crowded map-marker hits use a native location popover, with a dialog fallback.
Only ambiguous pointer hits show the picker; exposed markers and keyboard
activation retain direct selection. Marker geometry stays unchanged. All choices
use plain text and the current theme, respect the effective content viewport
including its scrollbar, and expire when their map/markers are replaced.
The map and fitting-search popovers share one viewport positioning helper.
