package main

import (
	"net/http"
	"regexp"
	"strings"
)

var entityModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/entity\.js"`)
var entityPagePath = regexp.MustCompile(`^/(character|corporation|alliance|faction)/[0-9]+(?:/[a-z]+)?$`)

func entityReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte) http.Handler {
	return componentReferenceRenderer(reference, renderer, scripts, "entity", entityModuleScript,
		func(path string) bool {
			return entityPagePath.MatchString(path) || strings.HasPrefix(path, "/_fragments/entity/")
		},
		[]string{"entity-navigation", "entity-content", "entity-valuable", "entity-classes"}, "entity.js")
}
