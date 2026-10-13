package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFittingReferencePreservesMarkupAndUsesOnlyNamedOriginalScripts(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Error("reference received credentials")
		}
		fmt.Fprint(w, `<body><meta name="shrike-dev-instance" content="current"><div data-controller="fit-editor" data-fit-editor><img width="564" height="564"></div><script src="/assets/current/fit-editor.js"></script></body>`)
	})
	handler := fittingReferenceRenderer(referenceRendererMustNotMutate{}, renderer, map[string][]byte{"fit-editor.js": []byte("original editor")})
	request := httptest.NewRequest("GET", "/fits/create", nil)
	request.Header.Set("Cookie", "private")
	request.Header.Set("Authorization", "private")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body := response.Body.String()
	if !strings.Contains(body, `src="/__fixture/fitting-source/fit-editor.js"`) || strings.Contains(body, `data-controller="fit-editor"`) || strings.Contains(body, "shrike-dev-instance") || !strings.Contains(body, `width="564" height="564"`) {
		t.Fatal(body)
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"POST", "/api/fit", 405}, {"GET", "/__fixture/fitting-source/private.js", 404}, {"GET", "/__fixture/fitting-source/fit-editor.js", 200}} {
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
		if response.Code != tc.status {
			t.Fatalf("%s %s: %d", tc.method, tc.path, response.Code)
		}
	}
}

func TestSavedFittingReferenceRunsOriginalEditorAndDetailWithoutDuplicateScopes(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("source renderer received credentials")
		}
		fmt.Fprint(w, `<body><div data-controller="fit-editor"><div data-controller="fit-rating"></div><section data-controller="fit-comments"><template><div data-controller="comment-thread"></div></template></section></div><script src="/assets/current/fit-editor.js"></script><script src="/assets/current/fit-detail.js"></script></body>`)
	})
	handler := fittingReferenceRenderer(referenceRendererMustNotMutate{}, renderer, map[string][]byte{"fit-editor.js": []byte("original editor"), "fit-detail.js": []byte("original detail")})
	request := httptest.NewRequest("GET", "/fit/PublicFit00001", nil)
	request.Header.Set("Cookie", "private")
	request.Header.Set("Authorization", "private")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body := response.Body.String()
	for _, entry := range []string{"fit-editor", "fit-detail"} {
		if !strings.Contains(body, `src="/__fixture/fitting-source/`+entry+`.js"`) || strings.Contains(body, `src="/assets/current/`+entry+`.js"`) {
			t.Fatal("current and original scripts mixed", body)
		}
	}
	for _, controller := range []string{"fit-editor", "fit-rating", "fit-comments", "comment-thread"} {
		if strings.Contains(body, `data-controller="`+controller+`"`) {
			t.Fatal("duplicate typed scope left in source comparison", controller)
		}
	}
}
