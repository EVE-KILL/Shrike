package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMapReferenceKeepsMarkupAndRelocatesOnlyNamedModuleImports(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Error("reference received credentials")
		}
		fmt.Fprint(w, `<body><div data-controller="map-page" data-map-root></div><script src="/assets/current/map-page.js"></script></body>`)
	})
	original := "import('/assets/map-live.js'); console.log('original')"
	handler := mapReferenceRenderer(referenceRendererMustNotMutate{}, renderer, map[string][]byte{"map-page.js": []byte(original), "map-live.js": []byte("original feed")})
	request := httptest.NewRequest("GET", "/map", nil)
	request.Header.Set("Cookie", "private")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), `src="/__fixture/map-source/map-page.js"`) || strings.Contains(response.Body.String(), `data-controller="map-page"`) {
		t.Fatal(response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/__fixture/map-source/map-page.js", nil))
	if response.Body.String() != "import('/__fixture/map-source/map-live.js'); console.log('original')" {
		t.Fatal(response.Body.String())
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"POST", "/api/map/scope", 405}, {"GET", "/__fixture/map-source/private.js", 404}} {
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
		if response.Code != tc.status {
			t.Fatalf("%s %s: %d", tc.method, tc.path, response.Code)
		}
	}
}
