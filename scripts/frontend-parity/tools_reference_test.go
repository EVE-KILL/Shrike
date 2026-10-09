package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestToolsReferenceUsesArchivedControllersAndLimitsPrivateWrites(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("source received credentials")
		}
		name := "scans"
		if r.URL.Path == "/docs" {
			name = "docs"
		}
		if r.URL.Path == "/mcp" {
			name = "mcp"
		}
		fmt.Fprintf(w, `<div data-controller="%s">Original markup</div><script src="/assets/current/%s.js"></script>`, name, name)
	})
	privateCalls := 0
	privateAPI := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("source forwarded credentials")
		}
		privateCalls++
		w.WriteHeader(204)
	})
	handler := toolsReferenceRenderer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(405) }), renderer, privateAPI, map[string][]byte{})
	for _, entry := range []struct{ path, name string }{{"/tools/dscan", "scans"}, {"/tools/localscan", "scans"}, {"/docs", "docs"}, {"/mcp", "mcp"}} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", entry.path, nil)
		r.Header.Set("Cookie", "private")
		handler.ServeHTTP(w, r)
		if !strings.Contains(w.Body.String(), `/__fixture/tools-source/`+entry.name+`.js`) || strings.Contains(w.Body.String(), `data-controller="`+entry.name+`"`) {
			t.Fatal("duplicate or missing script", w.Body.String())
		}
	}
	for _, path := range []string{"/api/tools/dscan", "/api/tools/localscan", "/api/tools/dscan/save", "/api/tools/localscan/save"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", path, nil)
		r.Header.Set("Cookie", "private")
		handler.ServeHTTP(w, r)
		if w.Code != 204 {
			t.Fatal("private scan not forwarded")
		}
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_fragments/scans/dscan", nil)
	r.Header.Set("Cookie", "private")
	handler.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Original markup") {
		t.Fatal("analysis result fragment was not rendered")
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("POST", "/api/fit", nil))
	if w.Code != 405 || privateCalls != 4 {
		t.Fatal("expanded private write surface")
	}
}
