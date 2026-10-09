package main

import (
	"net/http"
	"regexp"
	"strings"
)

func analysisReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte) http.Handler {
	for _, entry := range []struct {
		name        string
		controllers []string
		match       func(string) bool
	}{
		{"statistics", []string{"statistics"}, func(path string) bool { return path == "/stats" || strings.HasPrefix(path, "/_fragments/stats/") }},
		{"rankings", []string{"rankings"}, func(path string) bool { return path == "/rankings" || path == "/_fragments/rankings" }},
		{"graph", []string{"graph", "graph-search"}, func(path string) bool { return path == "/graph" }},
	} {
		reference = componentReferenceRenderer(reference, renderer, scripts, "analysis", regexp.MustCompile(`src="/assets/[^/"<>]+/`+entry.name+`\.js"`), entry.match, entry.controllers, entry.name+".js")
	}
	return reference
}
