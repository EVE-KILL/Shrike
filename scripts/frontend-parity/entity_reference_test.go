package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEntityReferenceRetainsComponentsAndRunsOnlyArchivedControllers(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Error("reference leaked credentials")
		}
		fmt.Fprint(w, `<meta name="shrike-dev-instance" content="current"><div class="entity-page" data-controller="entity-navigation"><section class="entity-main" data-controller="entity-content">Members</section><div data-controller="entity-valuable"></div><details data-controller="entity-classes"></details></div><script src="/assets/current/entity.js"></script>`)
	})
	handler := entityReferenceRenderer(referenceRendererMustNotMutate{}, renderer, map[string][]byte{"entity.js": []byte("original")})
	request := httptest.NewRequest("GET", "/alliance/70121/members", nil)
	request.Header.Set("Cookie", "private=session")
	request.Header.Set("Authorization", "private")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), `src="/__fixture/entity-source/entity.js"`) || !strings.Contains(response.Body.String(), `class="entity-main"`) || strings.Contains(response.Body.String(), `data-controller="entity-`) || strings.Contains(response.Body.String(), "shrike-dev-instance") {
		t.Fatal(response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("POST", "/api/entities/alliance/70121", nil))
	if response.Code != 405 {
		t.Fatal("reference mutation was not rejected")
	}
}

type referenceRendererMustNotMutate struct{}

func (referenceRendererMustNotMutate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		w.WriteHeader(405)
	} else {
		w.WriteHeader(404)
	}
}
