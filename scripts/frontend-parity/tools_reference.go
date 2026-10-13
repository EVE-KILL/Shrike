package main

import (
	"net/http"
	"regexp"
	"strings"
)

func toolsReferenceRenderer(reference, renderer, privateAPI http.Handler, scripts map[string][]byte) http.Handler {
	for _, entry := range []struct {
		name        string
		controllers []string
		match       func(string) bool
	}{
		{"scans", []string{"scans"}, func(path string) bool {
			return path == "/tools/dscan" || path == "/tools/localscan" || strings.HasPrefix(path, "/tools/dscan/") || strings.HasPrefix(path, "/tools/localscan/") || strings.HasPrefix(path, "/_fragments/scans/")
		}},
		{"docs", []string{"docs", "docs-endpoint"}, func(path string) bool { return path == "/docs" }},
		{"mcp", []string{"mcp"}, func(path string) bool { return path == "/mcp" || strings.HasPrefix(path, "/_fragments/mcp/") }},
	} {
		reference = componentReferenceRenderer(reference, renderer, scripts, "tools", regexp.MustCompile(`src="/assets/[^/"<>]+/`+entry.name+`\.js"`), entry.match, entry.controllers, entry.name+".js")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && (r.URL.Path == "/_fragments/scans/dscan" || r.URL.Path == "/_fragments/scans/localscan") {
			request := r.Clone(r.Context())
			request.Header.Del("Cookie")
			request.Header.Del("Authorization")
			// This renders supplied analysis JSON; it performs no write.
			renderer.ServeHTTP(w, request)
			return
		}
		// This opt-in mode is rejected unless its API is the separate private
		// fixture. Permit only scan analysis/save contracts on the source port.
		if r.Method == http.MethodPost && (r.URL.Path == "/api/tools/dscan" || r.URL.Path == "/api/tools/localscan" || r.URL.Path == "/api/tools/dscan/save" || r.URL.Path == "/api/tools/localscan/save") {
			request := r.Clone(r.Context())
			request.Header.Del("Cookie")
			request.Header.Del("Authorization")
			privateAPI.ServeHTTP(w, request)
			return
		}
		reference.ServeHTTP(w, r)
	})
}
