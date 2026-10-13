package main

import (
	"net/http"
	"regexp"
	"strings"
)

var fittingEditorModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/fit-editor\.js"`)
var fittingCatalogModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/fits-catalog\.js"`)
var fittingSearchModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/fits-search\.js"`)
var fittingDetailModuleScript = regexp.MustCompile(`src="/assets/[^/"<>]+/fit-detail\.js"`)
var fittingDetailPath = regexp.MustCompile(`^/fit/[a-zA-Z0-9]{14}$`)

// Compare the pre-TypeScript editor against the same Go markup, CSS, data and
// pinned engine. The retained binary predates later fitting interaction fixes.
func fittingReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte) http.Handler {
	editor := componentReferenceRenderer(reference, renderer, scripts, "fitting", fittingEditorModuleScript,
		func(path string) bool {
			return path == "/fits/create" || fittingDetailPath.MatchString(path) || strings.HasPrefix(path, "/_fragments/fits/variants/")
		},
		[]string{"fit-editor"}, "fit-editor.js")
	pages := componentReferenceRenderer(editor, renderer, scripts, "fitting", fittingCatalogModuleScript,
		func(path string) bool { return path == "/fits" || strings.HasPrefix(path, "/_fragments/fits/catalog/") },
		[]string{"fits-catalogue"}, "fits-catalog.js")
	pages = componentReferenceRenderer(pages, renderer, scripts, "fitting", fittingSearchModuleScript,
		func(path string) bool {
			return path == "/fits/search" || strings.HasPrefix(path, "/_fragments/fits/search/")
		},
		[]string{"fits-search", "search-picker"}, "fits-search.js")
	return componentReferenceRenderer(pages, editor, scripts, "fitting", fittingDetailModuleScript,
		fittingDetailPath.MatchString,
		[]string{"fit-rating", "fit-comments", "comment-thread"}, "fit-detail.js")
}
