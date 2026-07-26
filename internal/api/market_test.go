package api

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

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
	url url.URL
}

func (f fakeHumaContext) URL() url.URL { return f.url }

type humaContextStub interface {
	huma.Context
}
