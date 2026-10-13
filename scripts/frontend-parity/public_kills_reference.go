package main

import (
	"net/http"
	"regexp"
	"strings"
)

var homeModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/home\.js"`)
var killDetailModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/kill-detail\.js"`)

// Both implementations read identical disposable data and use actual Go HTML.
// The archive supplies exactly the original controllers and their two helpers.
func publicKillsReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte) http.Handler {
	feeds := componentReferenceRenderer(reference, renderer, scripts, "public-kills", homeModuleScript,
		func(path string) bool {
			return path == "/" || strings.HasPrefix(path, "/kills/") || path == "/_fragments/killlist" || path == "/_fragments/most-valuable" || path == "/_fragments/kills/top"
		},
		[]string{"home", "home-top"}, "home.js")
	return componentReferenceRenderer(feeds, renderer, scripts, "public-kills", killDetailModuleScript,
		func(path string) bool { return strings.HasPrefix(path, "/kill/") }, []string{"kill-detail"}, "kill-detail.js")
}
