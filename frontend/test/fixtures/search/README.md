Initial/query HTML was captured through normal Go search page handlers on
2026-10-08. results.json comes from the real Go /api/search sorter/serializer and
controlled rows for all eleven entity types plus corporation/alliance joins.
The persistent search_response_test.go verifies that contract; temporary capture
tests were removed. Unit mounts omit external module/style loaders.
No production records were read or changed. Desktop/mobile acceptance is separate.
