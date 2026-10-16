# Frontend UI review

This reference records the October 2026 UI review, its fixes, and its validation limits.
It preserves the existing EVE-KILL interface while improving readability and operation.

## Contract

Newcomers and experienced players are equally primary audiences.
Keep useful data density, recognizable EVE imagery, custom themes, and existing workflows.
Use [product context](../../PRODUCT.md) and [design conventions](../../DESIGN.md) for future changes.

## Defaults

Impeccable uses code-first development for this project.
Its shared configuration enables design hooks and recognizes `.templ` as HTML templates.
Codex hook manifests, local consent, detector caches, and raw critique snapshots remain local.
The installed skill is linked locally; it is not a production dependency.

`make dev DEV_PORT=4200` rebuilds templates and browser assets, then reloads open pages.
Production bundles must be rebuilt before committing generated assets.

## Review results

Two independent assessments reviewed visual usability and technical behavior before fixes.
The baseline usability score was 26/40; the technical score was 12/20.
These scores describe the baseline, not a post-change certification.

The review retained the existing font, space backgrounds, compact kill rows, semantic colors, and fitting wheel.
It produced these changes:

- Readable supporting labels, mobile kill details, profile statistics, and market links.
- Theme-aware neutral colors and brighter semantic values in killmail item tables.
- Larger mobile controls without expanding desktop data rows.
- Profile navigation before showcase panels, with all mobile labels visible.
- Compact entity headers with identity and metadata grouped together, and affiliation logos side by side.
- Character origin details and ratings visible on mobile, with readable wrapping for affiliation names and tickers.
- Associated valuable-list tabs with arrow, Home, End, and native activation keys.
- Map zoom buttons, keyboard pan/zoom/reset, center inspection, and keyboard instructions.
- Clearer spotlight focus, default-action navigation, result announcements, and empty-state guidance.
- An explicit fitting statistics prompt before hull selection or completed calculation.
- Distinct labels for recorded hull-and-fit values, module values, and the item breakdown subtotal.
- Advanced Search keeps common time/space choices visible and groups specialist controls behind a native More filters disclosure.
- Shared links and saved searches reveal their specialist criteria; collapsing filters keeps result sorting and view controls available.
- Custom ranges with only an end date survive restoration and appear in applied criteria.
- Date-picker popups stay inside narrow viewport edges.
- A compact mobile fitting toolbar keeps Undo, Redo, Paste and Save available, with secondary actions in a native More disclosure.
- Larger mobile catalogue, view, skill and comment controls; fitting dialogs use 16px mobile input text.
- Fitting catalogue tabs share keyboard focus behavior and label their retained panel.
- Failed fitting-data downloads expose an explicit retry, preserving the URL and typed query; failed downloads are not cached permanently.
- Shared theme application covers first paint and live previews, including native control appearance and legacy neutral utility colors.
- Neutral text, borders and interface surfaces follow custom colors across route styles, while original dark surface defaults remain the fallback.
- Saved-search refreshes retain the last account rows during slow or failed reads and disable mutations until refreshed.
- Live map signals become static under reduced motion, retain their expiry, and release timers/listeners on teardown.

Advanced Search accepts open-ended custom ranges in both the UI and API.
Omitting a timeframe still defaults to 30 days; explicitly clearing it removes the time bound.
Malformed dates and reversed ranges remain invalid.

The valuation changes affect presentation only.
Stored kill values include the hull and fitted items, including drones and fighter tubes.
The breakdown uses the prices returned for individual items and the hull.
These values can differ; the UI must identify their different meanings.

## Verification

The initial review used the local Go development server with existing database reads.
It did not submit killmails, save fittings, post comments, or change production records.

Browser coverage included home, latest kills, character, corporation, alliance, kill detail, advanced search, market, map, fitting, FAQ, and API documentation.
Responsive checks included 320px, 390px, and 768px widths, plus desktop layouts.
The expanded fitting wheel remained separate from statistics at 1280px.
The mobile profile navigation moved from approximately 1033px to 703px below the document start,
while restoring the rating and character origin information previously hidden on mobile.
The character header above the statistics shrank from 322px to approximately 205px on desktop.
Character, corporation, and alliance headers fit at 320px and 768px, including ratings and metadata.

Default colors, the existing purple preference, an Amarr accent, and custom dark-theme colors were checked.
Temporary theme changes were restored.
Keyboard checks covered valuable-list tabs, search, and map controls.
Browser controller tests cover the shared tab behavior, map camera movement, cancellation, fitting calculations, and search states.
The Advanced Search follow-up exercised a real linked pilot/ISK/end-date query,
sorting with collapsed filters, fitting mode's empty state, native keyboard disclosure,
and date-picker placement at 320px. Saved-search restoration is covered by controller tests;
no account writes were performed.

### Follow-up validation

The follow-up used a separate PostgreSQL 18.6 container bound only to loopback,
with a disposable database and owner/reader accounts. It did not modify production data.
The existing fixture service on 4450/4460 was left running; this review used 4470/4471.

Browser checks exercised private fit creation, update and readback; another account was denied access.
Account searches were saved, renamed and loaded, and remained isolated between accounts.
A six-second delay displayed local loading text and the global progress bar while retaining the saved list.
A forced 503 retained the list and disabled stale edits; a subsequent refresh restored editing.
Comment submission retained its draft after a forced failure, rendered Markdown in preview,
and persisted the successful comment and formatting through reload.
Synthetic killmail submissions exercised rejection and acceptance through the in-memory transport;
no killmail jobs or external submissions were dispatched.
Blocking the fitting catalogue download in Chrome exposed the retry state;
unblocking it and retrying restored the typed Rifter query and search focus.
An 844×390 landscape check exposed a clipped fitting-save footer. The dialog now
scrolls its body independently while keeping its title and action buttons visible.

Custom light colors were checked on character, corporation, alliance, kill detail,
market, advanced search, fitting and blog surfaces. Narrow public pages fit at 320px;
720px reflow checks also covered entity, kill detail, market and search layouts.
The 720px check approximates the layout space of a 1440px page at 200% browser zoom;
it does not certify text-only zoom behavior. Chrome's emulated OS text-scale setting
did not enlarge this desktop render, so that attempt is not counted as validation.
An additional inspector-only stylesheet doubled the computed font sizes and line heights
on Advanced Search (20px heading became 40px). Common filters and the keyboard-operated
More disclosure remained usable without horizontal document overflow at 1440px.
The temporary stylesheet was removed after the check; this is enlarged-text rendering
evidence, not a claim of native Firefox text-only zoom coverage.
Sampled accessible control names were inspected through Chrome's accessibility tree,
and reduced-motion emulation stopped CSS animations. Live-map reduced-motion expiry
and cleanup are covered by the canvas regression test.

Strict TypeScript and 893 frontend tests passed, including loader recovery, responsive disclosure,
catalogue tab behavior, retained saved-search rows and reduced-motion map expiry/teardown.
The production browser graph remains 119 entries with 107 shared chunks.
Its JavaScript totals 1,180,692 bytes, an increase of 3,483 bytes from the preceding polish commit.
Generated artifact hashes and module imports were checked after the production build.
The full Go suite passed with application and River migrations applied to the disposable
PostgreSQL container. The first opt-in run against its empty base database correctly
failed on missing tables; applying the normal test schema resolved that setup failure.

The `.templ` detector needs explicit coverage or the configured extension.
Without that configuration, a directory scan can silently omit templates.
Static scans cannot evaluate dynamic templ attributes or generated image URLs reliably.
Rendered checks remain necessary.
The initial polish pass's 18 page/viewport scans reported no low-contrast text findings.
This is sampled detector evidence, not an accessibility certification.

## Failure modes and remaining work

Detector occurrences repeat shared components and live data across routes.
Do not turn occurrence counts into a count of separate defects.
Development diagnostics intentionally use compact text and are not production content.
Semantic heatmap colors, compact data groups, and fitting diagrams require contextual review.

Physical touch devices, screen readers, actual text-only zoom and additional route-specific expanded/error states
still require further hands-on testing. Viewport emulation and accessibility-tree inspection
are not substitutes for those checks. Arbitrary custom palettes can still choose insufficient
contrast; theme support does not guarantee the contrast of every user-selected color combination.
Star-map/graph scenes, fitting-wheel artwork and EVE data colors retain their established rendering.

This review does not certify every route or all accessibility requirements.

## Source

- `internal/frontend`: rendered components, shared shell, and route styles.
- `frontend/src/browser`: controllers and shared interaction helpers.
- `frontend/src/utils/map/goFrontendCanvas.ts`: shared canvas renderer.
- `frontend/src/utils/themeData.ts`: supported theme properties and presets.
- `frontend/test`: browser controller, rendering, and calculation regression tests.
