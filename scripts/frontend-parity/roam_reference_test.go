package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoamReferenceRetainsTemplatesAndOwnsArchivedImports(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("source renderer received credentials")
		}
		w.Write([]byte(`<div data-controller="roam-report"><div data-controller="roam-form"><div data-controller="date-time-picker"></div></div><div data-controller="roam-composition"><template><tr><td>Original</td></tr></template></div></div><script src="/assets/current/roam-report.js"></script>`))
	})
	scripts := map[string][]byte{"roam-report.js": []byte(`import('/assets/map-canvas.js')`)}
	handler := roamReferenceRenderer(referenceRendererMustNotMutate{}, renderer, scripts)
	for _, path := range []string{"/tools/roam-report", "/tools/roam-report/a462a5e9c726c1e7e210d9b8452bc3a9/composition", "/_fragments/roam/a462a5e9c726c1e7e210d9b8452bc3a9/composition"} {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Cookie", "private")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		body := w.Body.String()
		if strings.Contains(body, `data-controller="roam-`) || strings.Contains(body, `data-controller="date-time-picker"`) || !strings.Contains(body, `<template><tr><td>Original</td></tr></template>`) || !strings.Contains(body, `/__fixture/roam-source/roam-report.js`) {
			t.Fatal("lost template or duplicate controller", body)
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/__fixture/roam-source/roam-report.js", nil))
	if !strings.Contains(w.Body.String(), "'/__fixture/roam-source/map-canvas.js'") || string(scripts["roam-report.js"]) != `import('/assets/map-canvas.js')` {
		t.Fatal("archived import not isolated")
	}
}
