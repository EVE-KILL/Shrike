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

The valuation changes affect presentation only.
Stored kill values include the hull and fitted items, including drones and fighter tubes.
The breakdown uses the prices returned for individual items and the hull.
These values can differ; the UI must identify their different meanings.

## Verification

The review used the local Go development server with existing database reads.
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

The `.templ` detector needs explicit coverage or the configured extension.
Without that configuration, a directory scan can silently omit templates.
Static scans cannot evaluate dynamic templ attributes or generated image URLs reliably.
Rendered checks remain necessary.
The final 18 page/viewport scans reported no low-contrast text findings.
This is sampled detector evidence, not an accessibility certification.

## Failure modes and remaining work

Detector occurrences repeat shared components and live data across routes.
Do not turn occurrence counts into a count of separate defects.
Development diagnostics intentionally use compact text and are not production content.
Semantic heatmap colors, compact data groups, and fitting diagrams require contextual review.

The following work remains outside this pass:

- Reorganize Advanced Search choices for newcomers while retaining expert filters and linked queries.
- Review mobile fitting-toolbar action groups and disclosure before changing the established workflow.
- Extend semantic theme tokens through remaining route-specific surfaces and verify custom light palettes.
- Run authenticated save/post workflows against disposable accounts and data.
- Test physical touch devices, screen readers, text zoom, and additional expanded/error states.

This review does not certify every route or all accessibility requirements.

## Source

- `internal/frontend`: rendered components, shared shell, and route styles.
- `frontend/src/browser`: controllers and shared interaction helpers.
- `frontend/src/utils/map/goFrontendCanvas.ts`: shared canvas renderer.
- `frontend/src/utils/themeData.ts`: supported theme properties and presets.
- `frontend/test`: browser controller, rendering, and calculation regression tests.
