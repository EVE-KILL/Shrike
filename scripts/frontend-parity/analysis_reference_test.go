package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnalysisReferencePreservesDataMarkupAndRemovesTypedScopes(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" {
			t.Fatal("source received credentials")
		}
		name := map[string]string{"/stats": "statistics", "/rankings": "rankings", "/graph": "graph"}[r.URL.Path]
		fmt.Fprintf(w, `<div data-controller="%s"><div data-controller="graph-search"></div><script type="application/json">{"source":"same data"}</script></div><script src="/assets/current/%s.js"></script>`, name, name)
	})
	handler := analysisReferenceRenderer(referenceRendererMustNotMutate{}, renderer, map[string][]byte{})
	for _, entry := range []struct{ path, name string }{{"/stats", "statistics"}, {"/rankings", "rankings"}, {"/graph", "graph"}} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", entry.path, nil)
		r.Header.Set("Cookie", "private")
		handler.ServeHTTP(w, r)
		body := w.Body.String()
		if strings.Contains(body, `data-controller="`+entry.name+`"`) || !strings.Contains(body, `/__fixture/analysis-source/`+entry.name+`.js`) || !strings.Contains(body, `{"source":"same data"}`) {
			t.Fatal("changed source or duplicate scope", body)
		}
		if entry.name == "graph" && strings.Contains(body, `data-controller="graph-search"`) {
			t.Fatal("duplicate search")
		}
	}
}
