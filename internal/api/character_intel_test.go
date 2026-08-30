package api

import (
	"context"
	"strings"
	"testing"
)

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

type batchIntelGraphStub struct{}

func (batchIntelGraphStub) Read(_ context.Context, query string, _ map[string]any) ([]map[string]any, error) {
	switch {
	case strings.Contains(query, "AS partners"):
		return []map[string]any{{"cid": int64(7), "partners": []any{
			map[string]any{"id": int64(8), "corp_id": int64(9), "alliance_id": int64(10), "weight": int64(4)},
		}}}, nil
	case strings.Contains(query, "size(groups)"):
		return []map[string]any{{"cid": int64(7), "cnt": int64(2), "groups": []any{
			map[string]any{"alliance_id": int64(10), "shared_partners": int64(3)},
		}}}, nil
	default:
		return []map[string]any{{"cid": int64(7), "last_fc_seen": "2026-08-30T00:00:00.000Z"}}, nil
	}
}

func TestLoadGraphIntelBatchMapsBatchRows(t *testing.T) {
	got := loadGraphIntelBatch(context.Background(), batchIntelGraphStub{}, []int32{7, 11})
	if len(got[7].FleetPartners) != 1 || got[7].BridgeScore != 2 || len(got[7].GroupsFlownWith) != 1 {
		t.Fatalf("character 7 graph intel = %#v", got[7])
	}
	if got[7].Timestamps["last_fc_seen"] == "" {
		t.Fatalf("character 7 timestamps = %#v", got[7].Timestamps)
	}
	if got[11].Timestamps == nil || len(got[11].FleetPartners) != 0 {
		t.Fatalf("character 11 empty graph intel = %#v", got[11])
	}
}

func TestGraphMapListReturnsStableEmptyList(t *testing.T) {
	got := graphMapList(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("graphMapList(nil) = %#v, want non-nil empty list", got)
	}
}
