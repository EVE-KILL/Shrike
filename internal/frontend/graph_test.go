package frontend

import (
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestGraphDeepLinksRenderWithoutBackend(t *testing.T) {
	handler := New(Options{})
	for _, tab := range graphTabs {
		t.Run(tab.ID, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/graph?tab="+tab.ID, nil))
			if w.Code != 200 {
				t.Fatalf("status = %d", w.Code)
			}
			doc, err := html.Parse(strings.NewReader(w.Body.String()))
			if err != nil {
				t.Fatal(err)
			}
			selected, visible, tabs := 0, 0, 0
			var visit func(*html.Node)
			visit = func(n *html.Node) {
				attrs := map[string]string{}
				for _, attr := range n.Attr {
					attrs[attr.Key] = attr.Val
				}
				if attrs["role"] == "tab" {
					tabs++
					if attrs["aria-selected"] == "true" {
						selected++
						if attrs["aria-controls"] != "graph-panel-"+tab.ID || attrs["tabindex"] != "0" {
							t.Error("deep link selected the wrong tool")
						}
					}
				}
				if attrs["role"] == "tabpanel" {
					if _, hidden := attrs["hidden"]; !hidden {
						visible++
						if attrs["id"] != "graph-panel-"+tab.ID {
							t.Error("deep link displays the wrong panel")
						}
					}
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					visit(c)
				}
			}
			visit(doc)
			if selected != 1 || visible != 1 || tabs != 9 {
				t.Fatalf("selected=%d visible=%d tools=%d", selected, visible, tabs)
			}
			if strings.Contains(w.Body.String(), "_nuxt") {
				t.Error("Go graph document loads Nuxt")
			}
		})
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/graph?tab=%3Cscript%3E", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `data-tab="path_finder"`) || strings.Contains(w.Body.String(), "<script><") {
		t.Fatal("invalid tool must fall back to Path Finder")
	}
}
