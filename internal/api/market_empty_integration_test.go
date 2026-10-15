package api

import (
	"net/url"
	"testing"
	"time"
)

func TestEmptyMarketBookUsesCommittedSourceSnapshot(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_market_empty_")
	when := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, "INSERT INTO inv_types(type_id,name,published) VALUES(587,'Rifter',true)"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO market_source_files(source_path,dataset,size_bytes,rows_imported,file_time) VALUES('/market-orders/market-orders-latest.v3.csv.bz2','market-orders',0,0,$1)", when); err != nil {
		t.Fatal(err)
	}
	handler := marketItemOrdersHandler(Options{DB: pool})
	request := &legacyRequest{Params: map[string]string{"id": "587"}, Query: url.Values{}}
	payload, err := handler(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	body := payload.Body.(map[string]any)
	if snapshot, ok := body["snapshot_at"].(time.Time); !ok || !snapshot.Equal(when) {
		t.Fatalf("empty item lost the committed book timestamp: %v", body["snapshot_at"])
	}
	if len(body["sellers"].([]map[string]any)) != 0 || len(body["buyers"].([]map[string]any)) != 0 {
		t.Fatal("empty item invented orders")
	}
	if _, err := pool.Exec(ctx, "DELETE FROM market_source_files"); err != nil {
		t.Fatal(err)
	}
	payload, err = handler(ctx, request)
	if err != nil || payload.Body.(map[string]any)["snapshot_at"] != nil {
		t.Fatalf("an unimported book must return a null snapshot: %v", err)
	}
}

func TestShortSearchStillRanksCategoriesAndBoundsResults(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_search_short_")
	if _, err := pool.Exec(ctx, "INSERT INTO characters(character_id,name) SELECT 1000+i,'Alpha '||i::text FROM generate_series(1,80) i"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO factions(faction_id,name) VALUES(500003,'Alpha Republic')"); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"Al", "Alpha"} {
		for _, limit := range []string{"10", "-1"} {
			payload, err := searchHandler(Options{DB: pool})(ctx, &legacyRequest{Query: url.Values{"q": {query}, "limit": {limit}}})
			if err != nil {
				t.Fatal(err)
			}
			body := payload.Body.(map[string]any)
			hits := body["hits"].([]map[string]any)
			want := 10
			if limit == "-1" {
				want = 1
			}
			if len(hits) != want || hits[0]["type"] != "faction" {
				t.Fatalf("query=%s limit=%s lost the bounded page or faction prefix ranking: %v", query, limit, hits)
			}
		}
	}
}
