package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchPagePreservesAndEscapesQuery(t *testing.T) {
	handler := New(Options{Production: true})
	for _, tt := range []struct {
		url, want string
	}{
		{canonicalOrigin + "/search", `value=""`},
		{canonicalOrigin + "/search?q=Karbowiak", `value="Karbowiak"`},
		{canonicalOrigin + "/search?q=%C3%86gir", `value="Ægir"`},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.url, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), tt.want) || !strings.Contains(w.Body.String(), "search.js") {
			t.Fatalf("%s: status=%d missing %s or search script", tt.url, w.Code, tt.want)
		}
	}
	malicious := httptest.NewRecorder()
	handler.ServeHTTP(malicious, httptest.NewRequest(http.MethodGet, canonicalOrigin+`/search?q=%22%3E%3Cscript%3Ealert(1)%3C%2Fscript%3E`, nil))
	if malicious.Code != http.StatusOK || strings.Contains(malicious.Body.String(), `"><script>alert(1)</script>`) {
		t.Fatal("search query was not HTML escaped")
	}
}
