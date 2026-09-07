# Saved searches and killmail navigation

Account saved searches are private and follow the signed-in character, consistent with existing account preferences.

## Contract

- `GET /me/saved-searches` lists up to 50 searches, ordered by update time and ID, newest first.
- `POST /me/saved-searches` creates a search with a name and versioned document.
- `PUT /me/saved-searches/{id}` replaces the name and document when the supplied revision matches.
- `DELETE /me/saved-searches/{id}?revision=N` deletes the owned search when its revision matches.

All responses use `no-store`. Writes require a same-origin request and an authenticated session.
Names contain 1–100 characters and are unique within an account, ignoring case.
An account-level transaction lock enforces the 50-search limit during concurrent creation.
A stale revision or conflicting name returns HTTP 409 without overwriting stored data.

Version 1 records filters, sort, time range, killmail or fitting view, fitting deduplication, and optional exact-fit or family hash.
The existing advanced-search query builder validates executable filters.
Location and entity names are display hints. Query execution uses IDs.
Rolling date presets resolve when the search runs. Custom dates remain absolute.
Documents are bounded by request and serialized-size limits.

## Browser storage

Anonymous saves retain the `evekill-saved-searches` localStorage key.
The reader accepts the previous name, filters, and date format.
Account import is explicit and handles one search at a time.
The user can change the import name. Existing account searches are never overwritten.
Import keeps the browser copy until the user deletes it.
Existing `/advancedsearch?q=...` links remain supported.

## Killmail navigation

Kill-list links carry an opaque browser context ID in `nav`.
The context stores only the current page's ordered killmail IDs, return URL, label, and creation time.
Previous and Next follow that displayed order, including non-time search sorts.
They stop at page boundaries. The reader returns to the list to select another page.
No new server query endpoint or SQL state is exposed.

Contexts live in sessionStorage, expire after two hours, and are limited to ten pages of at most 100 IDs each.
A missing context shows a direct-link fallback. Killmail URLs without `nav` keep their existing behavior.
Context URLs are intended for browsing within a tab; shared links do not require the context.
