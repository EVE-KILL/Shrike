# Entity component captures

These fixtures were generated through the real Go page/fragment routes with
`testEntityReader` from `internal/frontend/entity_page_test.go`. The temporary
capture reader added a Rifter kill row with a fixed UTC timestamp and an opaque
decimal cursor (`9007199254740993`) to exercise paging without JS precision loss.
It returned an empty list for other ship groups. The temporary capture test was
removed after generation. No database or production data was used.

The organization intel capture adds a controlled ally and recent join through the
same Go route, exercising all three independent intel slots after stats replacement.

The directory includes character, corporation, alliance and faction pages plus
real Go table, tab, dashboard, intel, top-list and faction fragments. The empty
`faction-militia.html` is intentional: the Go route returns an empty successful
body when there are no militia rows. Tests mount these component scopes after
stripping shell scripts/styles. Controlled responses may change only request
markers (page/sort/filter/day) to simulate delayed or competing reads.
