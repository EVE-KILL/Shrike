package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiagnosticsOnlyInDevelopment(t *testing.T) {
	for _, production := range []bool{false, true} {
		w := httptest.NewRecorder()
		New(Options{Production: production}).ServeHTTP(w, httptest.NewRequest("GET", canonicalOrigin+"/", nil))
		hasPanel := strings.Contains(w.Body.String(), `id="dev-diagnostics"`)
		hasScript := strings.Contains(w.Body.String(), "/diagnostics.js")
		hasTiming := w.Header().Get("Server-Timing") != ""
		if hasPanel == production || hasScript == production || hasTiming == production {
			t.Fatalf("production=%v: panel=%v script=%v timing=%v", production, hasPanel, hasScript, hasTiming)
		}
		if !production && !strings.Contains(w.Header().Get("Server-Timing"), `dbcount;desc="0"`) {
			t.Fatal("static homepage should perform no SQL")
		}
	}
}

func TestErrorPageRetainsDiagnostics(t *testing.T) {
	w := httptest.NewRecorder()
	New(Options{}).ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/missing", nil))
	if w.Code != http.StatusNotFound || w.Header().Get("Server-Timing") == "" || !strings.Contains(w.Body.String(), `id="dev-diagnostics"`) {
		t.Fatal("missing error page diagnostics")
	}
}
