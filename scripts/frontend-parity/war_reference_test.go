package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWarReferencePreservesMarkupAndSuppressesDuplicateTypedScopes(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("source renderer received credentials")
		}
		entry, controller := "wars", "wars"
		if strings.HasPrefix(r.URL.Path, "/war/") {
			entry, controller = "war-detail", "war-detail"
		}
		if strings.HasPrefix(r.URL.Path, "/faction-war/") {
			entry, controller = "faction-war", "faction-war"
		}
		fmt.Fprintf(w, `<body><div data-controller="%s"><div data-controller="faction-war-map"><svg viewBox="0 0 100 100"></svg></div></div><script src="/assets/current/%s.js"></script></body>`, controller, entry)
	})
	handler := warsReferenceRenderer(referenceRendererMustNotMutate{}, renderer, map[string][]byte{"wars.js": []byte("original")})
	for _, item := range []struct{ path, entry string }{{"/wars", "wars"}, {"/war/763266", "war-detail"}, {"/faction-war/caldari-vs-gallente", "faction-war"}} {
		request := httptest.NewRequest("GET", item.path, nil)
		request.Header.Set("Cookie", "private")
		request.Header.Set("Authorization", "private")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		body := response.Body.String()
		if !strings.Contains(body, `src="/__fixture/war-source/`+item.entry+`.js"`) || strings.Contains(body, `data-controller="`+item.entry+`"`) || !strings.Contains(body, `viewBox="0 0 100 100"`) {
			t.Fatal("source markup or scripts changed", item.path, body)
		}
		if item.entry == "faction-war" && strings.Contains(body, `data-controller="faction-war-map"`) {
			t.Fatal("duplicate lazy map controller")
		}
	}
}
