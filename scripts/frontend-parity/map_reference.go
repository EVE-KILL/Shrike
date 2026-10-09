package main

import (
	"net/http"
	"regexp"
	"strings"
)

var mapPageModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/map-page\.js"`)

func mapReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte) http.Handler {
	// Original absolute module URLs must point at their named archived helpers.
	// Only import locations change; all original behavior and markup is retained.
	sources := make(map[string][]byte, len(scripts))
	for name, body := range scripts {
		sources[name] = body
	}
	body := string(sources["map-page.js"])
	for _, name := range []string{"map-canvas.js", "map-live.js", "map-aiid-audio.js"} {
		body = strings.ReplaceAll(body, "'/assets/"+name+"'", "'/__fixture/map-source/"+name+"'")
	}
	sources["map-page.js"] = []byte(body)
	return componentReferenceRenderer(reference, renderer, sources, "map", mapPageModuleScript,
		func(path string) bool { return path == "/map" }, []string{"map-page"}, "map-page.js")
}
