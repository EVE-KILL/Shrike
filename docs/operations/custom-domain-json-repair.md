# Custom-domain JSON repair, 2026-09-22

VIP's homepage returned HTTP 200 but rendered an empty kill list because
`/api/custom/killlist`, `/api/custom/stats`, and domain-scoped battles failed
with PostgreSQL `22023: cannot extract elements from a scalar`.

## Cause and evidence

Five `custom_domains` rows (`vip`, `tremaljack`, `coldfront`, `3sum`, `npe`)
stored `entities`, `theme`, `navbar_links`, and `widgets` as JSON strings
containing JSON documents. The other 29 rows had the expected array/object
types. The domain scope query in `internal/api/conflicts.go` calls
`jsonb_array_elements(entities)` directly, which rejects JSON strings. The
configuration reader's Go JSON helpers accept strings, explaining why the
board's title and appearance could load while its data queries failed.

The affected rows were last updated April 6–8, 2026. In the predecessor
EVE-KILL repository, frontend writes used `drizzle-orm/bun-sql` until commit
`ebe48cc003f54f0b2306ce348e0ee89550d1bd57` on April 8 at 15:45 UTC, which
replaced it with `drizzle-orm/postgres-js`. Every affected row predates this
switch; later rows use the correct shapes. This strongly implicates the old
Drizzle/Bun SQL JSON serialization path, matching
[Bun issue 28819](https://github.com/oven-sh/bun/issues/28819). The original
write requests and historical container were not replayed, so attribution
to that driver is evidence-based rather than a reproduction of those writes.

The malformed records survived subsequent migrations because there was no
database constraint on the JSON document shapes. The physical move to knas
did not require converting these values.

## Repair and prevention

Migration `00027_custom_domain_json_shapes.sql` decodes string documents once
and adds a validated constraint requiring arrays for entities/navigation and
objects for themes/widgets. Invalid JSON aborts the repair; existing valid
documents remain unchanged. The migration supports adoption after an incident
repair and does not re-encode documents on rollback.

The migration's Up SQL was applied transactionally to knas on September 22,
after saving the original five configurations locally and testing against a
temporary table containing all 34 configurations. Each of the four updates
changed five rows. The Goose ledger was left unchanged so the next normal
migration run can adopt the idempotent repair.

Verification: VIP's kill list, seven statistics variants, and scoped battles
returned HTTP 200; the homepage contained kill links and no SSR API errors.
The coldfront, npe, and tremaljack kill lists also returned data. 3sum's JSON
error was resolved, but its kill-list query subsequently hit a separate
context deadline; its endpoint recovery is not claimed here.

`TestCustomDomainJSONRepair` verifies preservation of document contents,
repeatability, rejection of double encoding in all four columns, and a
non-destructive Down migration. It and the full fresh-schema migration test
passed against disposable PostgreSQL 18.6.
