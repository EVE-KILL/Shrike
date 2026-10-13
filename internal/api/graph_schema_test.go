package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
)

// Exercise the real serializers with controlled graph records and empty name
// lookups, without a database or Memgraph connection.
type graphSchemaDB struct{ stubDatabase }
type graphSchemaRows struct{ pgx.Rows }

func (graphSchemaRows) Close()     {}
func (graphSchemaRows) Next() bool { return false }
func (graphSchemaRows) Err() error { return nil }
func (graphSchemaDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return graphSchemaRows{}, nil
}

type graphSchemaBackend struct{}

func (graphSchemaBackend) Read(_ context.Context, query string, _ map[string]any) ([]map[string]any, error) {
	switch {
	case strings.Contains(query, "AS node_ids"):
		return []map[string]any{{"node_ids": []int64{1, 2}, "weights": []int64{7}}}, nil
	case strings.Contains(query, "UNWIND $ids AS cid"):
		return []map[string]any{{"id": int64(1), "corp": int64(12), "alliance": int64(20)}, {"id": int64(2)}}, nil
	case strings.Contains(query, "RETURN attacker, enemy, kills"):
		rows := []map[string]any{}
		for _, attacker := range []int64{12, 20} {
			for _, enemy := range []int64{30, 40, 50} {
				rows = append(rows, map[string]any{"attacker": attacker, "enemy": enemy, "kills": int64(1000)})
			}
		}
		return rows, nil
	case strings.Contains(query, "AS corp_id"):
		return []map[string]any{{"corp_id": int64(12), "total": int64(3), "fcs": int64(1), "logis": int64(1), "caps": int64(1)}}, nil
	default:
		return nil, fmt.Errorf("unexpected graph query in contract test")
	}
}

func assertGraphWireFields(t *testing.T, schema *huma.Schema, value any, path string) {
	t.Helper()
	if schema == nil {
		t.Fatalf("missing schema at %s", path)
	}
	switch item := value.(type) {
	case map[string]any:
		for name, child := range item {
			if name == "$schema" {
				continue
			}
			property := schema.Properties[name]
			if property == nil {
				t.Errorf("published schema does not describe emitted field %s.%s", path, name)
				continue
			}
			assertGraphWireFields(t, property, child, path+"."+name)
		}
	case []any:
		for index, child := range item {
			assertGraphWireFields(t, schema.Items, child, fmt.Sprintf("%s[%d]", path, index))
		}
	}
}

func TestGraphResponseSchemaKeepsRenderedGoFields(t *testing.T) {
	service := New(Options{DB: graphSchemaDB{}, Graph: graphSchemaBackend{}})
	schema := service.OpenAPI().Paths["/graph"].Get.Responses["200"].Content["application/json"].Schema
	if schema == nil {
		t.Fatal("graph response is absent from the public document")
	}
	for _, tc := range []struct{ query, field string }{
		{"mode=path_finder&fromId=1&toId=2", "path"},
		{"mode=coalitions", "coalitions"},
		{"mode=census&entityId=20", "corps"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			response := httptest.NewRecorder()
			service.Site().ServeHTTP(response, httptest.NewRequest("GET", "/api/graph?"+tc.query, nil))
			if response.Code != 200 {
				t.Fatalf("graph response = %d: %s", response.Code, response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body[tc.field] == nil {
				t.Fatal("graph serializer returned no test data")
			}
			if rows, ok := body[tc.field].([]any); ok && len(rows) == 0 {
				t.Fatal("graph serializer returned an empty test list")
			}
			for _, variant := range schema.OneOf {
				if variant.Properties[tc.field] != nil {
					assertGraphWireFields(t, variant, body, "graph")
					return
				}
			}
			t.Fatal("no public graph schema variant matches the emitted mode")
		})
	}
}
