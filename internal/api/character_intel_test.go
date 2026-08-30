package api

import "testing"

func TestGraphMapList(t *testing.T) {
	input := []any{
		map[string]any{"alliance_id": int64(1), "shared_partners": int64(4)},
		"not a map",
		map[string]any{"alliance_id": int64(2), "shared_partners": int64(3)},
	}

	got := graphMapList(input)
	if len(got) != 2 {
		t.Fatalf("graphMapList returned %d rows, want 2", len(got))
	}
	if got[0]["alliance_id"] != int64(1) || got[1]["alliance_id"] != int64(2) {
		t.Fatalf("graphMapList returned unexpected rows: %#v", got)
	}
}

func TestGraphMapListReturnsStableEmptyList(t *testing.T) {
	got := graphMapList(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("graphMapList(nil) = %#v, want non-nil empty list", got)
	}
}
