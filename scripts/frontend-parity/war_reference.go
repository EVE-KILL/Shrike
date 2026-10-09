package main

import (
	"net/http"
	"regexp"
	"strings"
)

var warsModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/wars\.js"`)
var warDetailModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/war-detail\.js"`)
var factionWarModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/faction-war\.js"`)
var warDetailPath = regexp.MustCompile(`^/war/[0-9]+$`)

// Only the explicitly named pre-TypeScript modules run on the source port.
// Both implementations render identical Go markup against public GET data.
func warsReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte) http.Handler {
	pages := componentReferenceRenderer(reference, renderer, scripts, "war", warsModuleScript,
		func(path string) bool { return path == "/wars" || path == "/_fragments/wars/faction" },
		[]string{"wars"}, "wars.js")
	pages = componentReferenceRenderer(pages, renderer, scripts, "war", warDetailModuleScript,
		func(path string) bool {
			return warDetailPath.MatchString(path) || strings.HasPrefix(path, "/_fragments/war/")
		},
		[]string{"war-detail"}, "war-detail.js")
	return componentReferenceRenderer(pages, renderer, scripts, "war", factionWarModuleScript,
		func(path string) bool {
			return strings.HasPrefix(path, "/faction-war/") || strings.HasPrefix(path, "/_fragments/faction-war/")
		},
		[]string{"faction-war", "faction-war-map"}, "faction-war.js")
}
