package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCategoriesRenderCanonicalLinksAndCounts(t *testing.T) {
	handler := New(Options{Homepage: homepageReaderStub{}})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/categories", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("categories status = %d", w.Code)
	}
	for _, want := range []string{
		"Killmail Categories", "1,200", "/kills/highsec",
		"/advancedsearch?q=", "data-category-search=", "categories.js",
	} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("categories page missing %q", want)
		}
	}
	if strings.Index(w.Body.String(), "Space</h2>") > strings.Index(w.Body.String(), "Engagement</h2>") {
		t.Fatal("category groups are out of Nuxt display order")
	}
}
