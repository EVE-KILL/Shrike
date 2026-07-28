package api

import (
	"strings"
	"testing"
)

func TestFactionSearchPrioritizesNamePrefixes(t *testing.T) {
	query := searchFactionPart()
	if !strings.Contains(query, "WHEN name ILIKE $2 THEN 1") {
		t.Fatalf("faction search no longer prioritizes a faction name prefix:\n%s", query)
	}
	if !strings.Contains(query, "'faction' AS type") {
		t.Fatalf("faction search no longer returns faction hits:\n%s", query)
	}
}
