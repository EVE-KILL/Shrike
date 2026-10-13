Go-rendered archive page, killlist and top sidebar captured on 2026-10-08 with
the existing internal/frontend testLegacyReader. The reader supplies historical
kill 123, archive counters and top entries without a database. A temporary package
test called the normal HTTP handlers; it was removed after capture. The full page
was reduced to its data-legacy-page root without changing markup.

DOM regressions cover presets, filters, cursors, independent reads, retries,
autocomplete and mobile tab/history state. Live browser/mobile validation remains
a separate gate. No production writes or remote archive data were used.
