package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Real search sorting/serialization and affiliation joins over controlled rows.
// This contract does not require a PostgreSQL connection.
type searchResponseDB struct{ stubDatabase }
type searchResponseRows struct {
	pgx.Rows
	columns []string
	rows    [][]any
	at      int
}

func (r *searchResponseRows) Close()                 {}
func (r *searchResponseRows) Err() error             { return nil }
func (r *searchResponseRows) Next() bool             { r.at++; return r.at <= len(r.rows) }
func (r *searchResponseRows) Values() ([]any, error) { return r.rows[r.at-1], nil }
func (r *searchResponseRows) Scan(destinations ...any) error {
	if len(destinations) == 1 {
		if scanner, ok := destinations[0].(pgx.RowScanner); ok {
			return scanner.ScanRow(r)
		}
	}
	for i, destination := range destinations {
		target, ok := destination.(*any)
		if !ok {
			return fmt.Errorf("unexpected controlled search scan")
		}
		*target = r.rows[r.at-1][i]
	}
	return nil
}
func (r *searchResponseRows) FieldDescriptions() []pgconn.FieldDescription {
	fields := make([]pgconn.FieldDescription, len(r.columns))
	for i, name := range r.columns {
		fields[i].Name = name
	}
	return fields
}
func (searchResponseDB) Query(_ context.Context, query string, _ ...any) (pgx.Rows, error) {
	if strings.Contains(query, "SELECT * FROM (") {
		rows := [][]any{}
		for _, entity := range []struct {
			kind string
			id   int64
			name string
		}{{"character", 4242, "Fixture Pilot"}, {"corporation", 10, "Fixture Corp"}, {"alliance", 70121, "Fixture Alliance"}, {"faction", 500003, "Fixture Faction"}, {"system", 30002813, "Fixture System"}, {"region", 10000042, "Fixture Region"}, {"constellation", 20000396, "Fixture Constellation"}, {"ship", 587, "Fixture Rifter"}, {"item", 34, "Fixture Tritanium"}, {"shipgroup", 25, "Fixture Frigates"}, {"group", 52, "Fixture Module Group"}} {
			var corp, alliance any
			if entity.kind == "character" {
				corp = int64(10)
			}
			if entity.kind == "character" || entity.kind == "corporation" {
				alliance = int64(70121)
			}
			rows = append(rows, []any{entity.id, entity.name, "FIX", entity.kind, corp, alliance, float64(0), float64(.9)})
		}
		return &searchResponseRows{columns: []string{"entity_id", "name", "ticker", "type", "corporation_id", "alliance_id", "weight", "score"}, rows: rows}, nil
	}
	if strings.Contains(query, "FROM corporations WHERE") {
		return &searchResponseRows{columns: []string{"id", "name", "ticker"}, rows: [][]any{{int64(10), "Fixture Corp", "FC"}}}, nil
	}
	if strings.Contains(query, "FROM alliances WHERE") {
		return &searchResponseRows{columns: []string{"id", "name", "ticker"}, rows: [][]any{{int64(70121), "Fixture Alliance", "FA"}}}, nil
	}
	return nil, fmt.Errorf("unexpected controlled search query")
}
func searchResponseFixture(t *testing.T) []byte {
	t.Helper()
	service := New(Options{DB: searchResponseDB{}})
	response := httptest.NewRecorder()
	service.Site().ServeHTTP(response, httptest.NewRequest("GET", "/api/search?q=fixture", nil))
	if response.Code != 200 {
		t.Fatalf("search status %d: %s", response.Code, response.Body.String())
	}
	return response.Body.Bytes()
}
func TestSearchResponsePreservesTypesAndAffiliations(t *testing.T) {
	var body struct {
		Hits   []map[string]any `json:"hits"`
		Counts map[string]int   `json:"entityCounts"`
		Total  int              `json:"total"`
	}
	if err := json.Unmarshal(searchResponseFixture(t), &body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 11 || len(body.Hits) != 11 || len(body.Counts) != 11 {
		t.Fatalf("search counts %+v", body)
	}
	if body.Hits[0]["type"] != "ship" || body.Hits[0]["id"] != "ship_587" {
		t.Fatalf("search universe precedence %+v", body.Hits[0])
	}
	for _, hit := range body.Hits {
		kind, _ := hit["type"].(string)
		if body.Counts[kind] != 1 {
			t.Errorf("count for %s", kind)
		}
		if kind == "character" && (hit["corporation_name"] != "Fixture Corp" || hit["corporation_ticker"] != "FC" || hit["alliance_name"] != "Fixture Alliance" || hit["alliance_ticker"] != "FA") {
			t.Errorf("character affiliations %+v", hit)
		}
	}
}
