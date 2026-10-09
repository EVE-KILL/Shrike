package main

import (
	"net/http"
	"regexp"
	"strings"
)

var roamModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/roam-report\.js"`)

func roamReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte) http.Handler {
	sources := make(map[string][]byte, len(scripts))
	for name, body := range scripts {
		sources[name] = body
	}
	// Change only the archived module's absolute import location.
	sources["roam-report.js"] = []byte(strings.ReplaceAll(string(sources["roam-report.js"]), "'/assets/map-canvas.js'", "'/__fixture/roam-source/map-canvas.js'"))
	return componentReferenceRenderer(reference, renderer, sources, "roam", roamModuleScript,
		func(path string) bool {
			return path == "/tools/roam-report" || strings.HasPrefix(path, "/tools/roam-report/") || strings.HasPrefix(path, "/_fragments/roam/")
		},
		[]string{"roam-report", "roam-composition", "roam-form", "date-time-picker"}, "roam-report.js")
}
