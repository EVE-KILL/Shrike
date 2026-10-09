package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMarketPagesAndFragments(t *testing.T) {
	handler := New(Options{Production: true, Universe: testUniverseReader{}})
	for _, tc := range []struct {
		path   string
		status int
		marker string
	}{
		{"/market", 200, "Market Explorer"},
		{"/market/ships", 200, "Frigates"},
		{"/market/ships/frigates", 200, "Other Item"},
		{"/market/ships/unique", 302, "/market/item/11186"},
		{"/market/ships/missing", 404, "Page not found"},
		{"/market/item/11186", 200, `data-market-item-id="11186"`},
		{"/market/item/404", 404, "Page not found"},
		{"/_fragments/market/item/11186?region=10000002&days=90&security=high", 200, `data-security="high"`},
		{"/_fragments/market/item/11186?region=0&days=90&security=high,null", 200, `data-region="0"`},
		{"/_fragments/market/item/11186?days=45", 400, "Invalid market filters"},
		{"/_fragments/market/tree/4", 200, "Frigates"},
		{"/_fragments/market/tree/404", 404, "404"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+tc.path, nil))
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.marker) {
			t.Errorf("%s: status %d, marker %q present = %v", tc.path, w.Code, tc.marker, strings.Contains(w.Body.String(), tc.marker))
		}
	}
	root := httptest.NewRecorder()
	handler.ServeHTTP(root, httptest.NewRequest(http.MethodGet, canonicalOrigin+"/market", nil))
	if strings.Contains(root.Body.String(), `data-market-node="5"`) {
		t.Fatal("market root eagerly rendered collapsed sidebar descendants")
	}
}
