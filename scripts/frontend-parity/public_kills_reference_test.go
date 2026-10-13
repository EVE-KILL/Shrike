package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicKillsReferenceUsesOriginalScriptsAndAnonymousGoHTML(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Error("credentials leaked to reference")
		}
		fmt.Fprint(w, `<body><meta name="shrike-dev-instance" content="current"><div class="homepage" data-controller="home"><section data-controller="home-top"></section></div><section class="kill-detail" data-controller="kill-detail"></section><script src="/assets/current/site.js"></script><script src="/assets/current/kill-detail.js"></script></body>`)
	})
	handler := publicKillsReferenceRenderer(referenceRendererMustNotMutate{}, renderer, map[string][]byte{"home.js": []byte("original home"), "kill-detail.js": []byte("original kill")})
	for _, path := range []string{"/", "/kills/big", "/kill/2000000001"} {
		request := httptest.NewRequest("GET", path, nil)
		request.Header.Set("Cookie", "private")
		request.Header.Set("Authorization", "private")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		body := response.Body.String()
		expected := "home.js"
		controller := `data-controller="home"`
		if strings.HasPrefix(path, "/kill/") {
			expected = "kill-detail.js"
			controller = `data-controller="kill-detail"`
		}
		if !strings.Contains(body, `src="/__fixture/public-kills-source/`+expected+`"`) || strings.Contains(body, controller) || strings.Contains(body, "shrike-dev-instance") {
			t.Fatal(body)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("POST", "/api/killmail/2000000001/fit", nil))
	if response.Code != 405 {
		t.Fatal("reference write accepted")
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/__fixture/public-kills-source/private.js", nil))
	if response.Code != 404 {
		t.Fatal("unnamed archive file exposed")
	}
}
