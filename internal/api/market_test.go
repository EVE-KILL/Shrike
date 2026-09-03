package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMarketSecurityFilterIsBoundedToKnownPredicates(t *testing.T) {
	cases := []struct {
		raw      string
		contains []string
		invalid  bool
	}{
		{"", []string{"TRUE"}, false},
		{"high", []string{"system.security >= 0.5"}, false},
		{"low,null", []string{"system.security > 0", "system.security <= 0"}, false},
		{"high,low,null", []string{"TRUE"}, false},
		{"drop table market_orders", nil, true},
	}
	for _, test := range cases {
		sql, err := marketSecuritySQL(test.raw, "system.security")
		if test.invalid {
			if err == nil {
				t.Errorf("%q was accepted as %q", test.raw, sql)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q returned %v", test.raw, err)
			continue
		}
		for _, fragment := range test.contains {
			if !strings.Contains(sql, fragment) {
				t.Errorf("%q produced %q without %q", test.raw, sql, fragment)
			}
		}
	}
}

func TestMarketExplorerEndpointsAgainstImportedData(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://evekill:evekill@127.0.0.1:5432/evekill"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("no test database: %v", err)
	}

	const plex = 44992
	var orders int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM market_orders WHERE type_id = $1`, plex).Scan(&orders); err != nil || orders == 0 {
		t.Skip("market explorer fixture data is not imported")
	}

	handler := apiPathHandler(Options{Version: "test", DB: pool, Primary: pool, PrimaryPool: pool})
	request := func(path string) map[string]any {
		t.Helper()
		response := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://example.test"+path, nil)
		req.Header.Set("User-Agent", "shrike-market-test/1.0")
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", path, response.Code, response.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body
	}

	book := request("/market/items/44992/orders?security=high&limit=10")
	if len(book["sellers"].([]any)) == 0 || len(book["buyers"].([]any)) == 0 {
		t.Fatal("PLEX order book did not contain both sides")
	}
	if len(book["regions"].([]any)) == 0 {
		t.Fatal("PLEX order book did not expose its global market region")
	}

	history := request("/market/items/44992/history?region_id=19000001&days=30")
	points := history["history"].([]any)
	if len(points) < 20 || len(points) > 30 {
		t.Fatalf("history points = %d, want a populated 30-day window", len(points))
	}
}

func TestBuildMarketTreePreservesHierarchyAndStableEmptyChildren(t *testing.T) {
	parent := int64(10)
	icon := int64(42)
	tree := buildMarketTree([]map[string]any{
		{
			"market_group_id": int32(20), "parent_group_id": parent,
			"name": "Zeta & Charges", "has_types": true,
		},
		{
			"market_group_id": int32(10), "parent_group_id": nil,
			"name": "Root", "has_types": false, "icon_id": icon,
		},
		{
			"market_group_id": int32(21), "parent_group_id": parent,
			"name": "Alpha", "has_types": true,
		},
	})

	if len(tree) != 1 || tree[0].ID != 10 {
		t.Fatalf("roots = %#v, want market group 10", tree)
	}
	if tree[0].IconID == nil || *tree[0].IconID != 42 {
		t.Errorf("root icon = %v, want 42", tree[0].IconID)
	}
	if got := []string{tree[0].Children[0].Name, tree[0].Children[1].Name}; !reflect.DeepEqual(got, []string{"Alpha", "Zeta & Charges"}) {
		t.Errorf("children = %v, want alphabetical order", got)
	}
	if got := tree[0].Children[1].Slug; got != "zeta-and-charges" {
		t.Errorf("slug = %q", got)
	}
	if tree[0].Children[0].Children == nil {
		t.Error("leaf children is nil; JSON must stay [] like the frontend contract")
	}
}

func TestParseBulkPriceIDs(t *testing.T) {
	got, err := parseBulkPriceIDs(" 34,35,34,nope,-1,2147483648,36 ")
	if err != nil {
		t.Fatal(err)
	}
	if want := []int32{34, 35, 36}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}

	var values string
	for i := 1; i <= maxBulkPriceTypes+1; i++ {
		if values != "" {
			values += ","
		}
		values += fmt.Sprint(i)
	}
	_, err = parseBulkPriceIDs(values)
	var apiErr *legacyAPIError
	if !errors.As(err, &apiErr) || apiErr.Status != 400 {
		t.Fatalf("257 ids error = %v, want API 400", err)
	}
}

func TestBulkPriceCacheKeyNormalizesOrderAndDuplicates(t *testing.T) {
	req := &legacyRequest{
		Huma:  fakeHumaContext{url: mustURL(t, "/prices/bulk?types=3,1,3")},
		Query: url.Values{"types": {"3,1,3"}},
	}
	if got, want := bulkPriceCacheKey(req), "/prices/bulk?types=1,3"; got != want {
		t.Fatalf("cache key = %q, want %q", got, want)
	}
}

func TestMarketGroupItemsCacheKeyIncludesResponseVersionAndGroup(t *testing.T) {
	req := &legacyRequest{Huma: fakeHumaContext{params: map[string]string{"id": "2358"}}}
	if got, want := marketGroupItemsCacheKey(req), "market-group-items:v2:2358"; got != want {
		t.Fatalf("cache key = %q, want %q", got, want)
	}
}

func mustURL(t *testing.T, raw string) url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return *parsed
}

// fakeHumaContext keeps this pure cache-key test independent of a router.
type fakeHumaContext struct {
	humaContextStub
	url    url.URL
	params map[string]string
}

func (f fakeHumaContext) URL() url.URL             { return f.url }
func (f fakeHumaContext) Param(name string) string { return f.params[name] }

type humaContextStub interface {
	huma.Context
}
