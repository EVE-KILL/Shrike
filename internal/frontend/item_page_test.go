package frontend

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

func TestItemPagesAndFragments(t *testing.T) {
	handler := New(Options{Production: true, Universe: testUniverseReader{}})
	for _, tc := range []struct {
		path   string
		status int
		marker string
	}{
		{"/item/11186", 200, "Test Item"},
		{"/item/11186/kills", 200, `id="kill-list"`},
		{"/item/11186/fittings", 200, `data-item-tab="fittings"`},
		{"/item/11186/fittings?min_ehp=3000&sort=ehp", 200, `data-fit-sort="ehp"`},
		{"/_fragments/item/fittings/11186?groups=52", 200, `data-fit-query="groups=52"`},
		{"/_fragments/item/fit-meta/11186", 200, "Warp Scrambler"},
		{"/_fragments/item/fit-distributions/11186", 200, "Effective Hitpoints"},
		{"/_fragments/item/fittings/11186?min_ehp=oops", 400, "Invalid fitting filters"},
		{"/item/34/market", 200, `data-item-market`},
		{"/item/34/market?region=10000002&days=90&security=high", 200, `data-security="high"`},
		{"/_fragments/item/market/34?region=10000002&days=90&security=high", 200, `The Forge price history`},
		{"/_fragments/item/market/34?region=0&days=90&security=high,null", 200, `data-region="0"`},
		{"/_fragments/item/market/34?region=-1", 400, "Invalid market filters"},
		{"/_fragments/item/market/34?days=45", 400, "Invalid market filters"},
		{"/_fragments/item/market/34?security=high,drop", 400, "Invalid market filters"},
		{"/_fragments/item/tab/11186/dashboard", 200, `data-item-tab="dashboard"`},
		{"/_fragments/item/tab/11186/kills", 200, `data-item-tab="kills"`},
		{"/_fragments/item/kills/11186?page=2&limit=10&after=42", 200, `data-page="2"`},
		{"/_fragments/item/kills/11186?page=2", 400, "Invalid page"},
		{"/item/34/fittings", 404, "Page not found"},
		{"/item/35/market", 404, "Page not found"},
		{"/item/404", 404, "Page not found"},
		{"/item/11186/other", 404, "Page not found"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+tc.path, nil))
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.marker) {
			t.Errorf("%s: status %d, marker %q present = %v", tc.path, w.Code, tc.marker, strings.Contains(w.Body.String(), tc.marker))
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+"/_fragments/item/tab/11186/dashboard", nil))
	if strings.Contains(w.Body.String(), "item-hero") || strings.Contains(w.Body.String(), "item-tabs") {
		t.Fatal("item tab fragment includes persistent header")
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+"/_fragments/item/market/34", nil))
	if !strings.Contains(w.Body.String(), `<polyline`) || !strings.Contains(w.Body.String(), `data-market-book="sellers"`) || strings.Contains(w.Body.String(), "item-hero") {
		t.Fatal("market fragment is missing SSR chart/orders or includes the persistent header")
	}
}

func TestFitEditorURLUsesCompatibleV3Payload(t *testing.T) {
	family := api.UniverseFitFamily{TotalUses: 12, Modules: []api.UniverseFitModule{
		{SlotGroup: 1, TypeID: 42, ChargeTypeID: 43}, {SlotGroup: 1, TypeID: 44},
	}, Drones: []api.UniverseFitDrone{{TypeID: 45, Quantity: 6}}}
	link := fitEditorURL(11186, family, 90)
	parsed, err := url.Parse(link)
	if err != nil || parsed.Path != "/fits/create" {
		t.Fatalf("fit editor URL = %q: %v", link, err)
	}
	encoded := strings.TrimPrefix(parsed.Query().Get("fit"), "v3:")
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	plain, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"ship,11186,Community Fit,", "module,High,1,42,Active,43", "module,High,2,44,Active,", "drone,45,5,1"} {
		if !strings.Contains(string(plain), line) {
			t.Errorf("fit payload missing %q: %s", line, plain)
		}
	}
}

func TestFitFamilyClassifierMatchesObservedShipRoles(t *testing.T) {
	for _, tc := range []struct {
		modules []api.UniverseFitModule
		want    string
	}{
		{[]api.UniverseFitModule{{Name: "Arbalest Compact Light Missile Launcher"}, {Name: "Small Ancillary Armor Repairer"}}, "Light Missile Armor Active"},
		{[]api.UniverseFitModule{{Name: "Interdiction Nullifier II"}, {Name: "Small Ancillary Armor Repairer"}}, "Travel Armor Active"},
	} {
		if got := fitFamilyName(api.UniverseFitFamily{Modules: tc.modules}, "Interceptor"); got != tc.want {
			t.Errorf("fitFamilyName(%v) = %q, want %q", tc.modules, got, tc.want)
		}
	}
}

func TestFitQueryDefaultsNPCProfileAndRejectsInvalidBounds(t *testing.T) {
	query, _, err := itemFitQuery(url.Values{"min_npc_ehp": {"5000"}})
	if err != nil || query.Get("npc_profile") != "guristas" {
		t.Fatalf("NPC filter = %v, error = %v", query, err)
	}
	for _, raw := range []url.Values{{"min_ehp": {"-1"}}, {"max_dps": {"NaN"}}, {"sort": {"unknown"}}, {"groups": {"1,0"}}} {
		if _, _, err := itemFitQuery(raw); err == nil {
			t.Errorf("accepted invalid fitting query %v", raw)
		}
	}
}

func TestItemQuantityPreservesFractionalVolume(t *testing.T) {
	if got := itemQuantity(0.01); got != "0.01" {
		t.Fatalf("itemQuantity(0.01) = %q", got)
	}
}

func TestMarketPriceKeepsFractionalISK(t *testing.T) {
	for _, tc := range []struct {
		value float64
		want  string
	}{{0.01, "0.01"}, {1234.5, "1,234.5"}, {1000000, "1,000,000"}} {
		if got := marketPrice(tc.value); got != tc.want {
			t.Errorf("marketPrice(%v) = %q, want %q", tc.value, got, tc.want)
		}
	}
}

func TestMarketAxisLabelsAreCompact(t *testing.T) {
	for _, tc := range []struct {
		value float64
		want  string
	}{{10_000_000, "10m"}, {18_383_741.82, "18.4m"}, {100_000_000, "100m"},
		{1_000_000_000, "1b"}, {100_000_000_000, "100b"}, {1_000_000_000_000, "1t"}, {100_000_000_000_000, "100t"}} {
		if got := marketAxisLabel(tc.value); got != tc.want {
			t.Errorf("marketAxisLabel(%v) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
