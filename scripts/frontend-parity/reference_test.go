package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestReferenceRendererRetainsPageAndAssetsWithoutCredentialsOrMutations(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Error("reference forwarded browser credentials")
		}
		if r.Host != "review.localhost" || r.Header.Get("X-Forwarded-For") != "192.0.2.1" {
			t.Error("reference lost the board host or trusted the incoming forwarding header")
		}
		w.Header().Add("Set-Cookie", "unexpected=credential")
		_, _ = w.Write([]byte("retained " + r.URL.RequestURI()))
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	handler := referenceRenderer(target)
	for _, path := range []string{"/search?q=Fixture", "/assets/retained/search.js"} {
		request := httptest.NewRequest("GET", path, nil)
		request.Host = "review.localhost"
		request.Header.Set("Cookie", "private=session")
		request.Header.Set("Authorization", "private")
		request.Header.Set("X-Forwarded-For", "203.0.113.99")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != 200 || response.Body.String() != "retained "+path || response.Header().Get("Set-Cookie") != "" {
			t.Fatalf("reference altered retained page/assets or cookies: %+v", response)
		}
	}
	for _, method := range []string{"POST", "PATCH", "PUT", "DELETE"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, "/api/fixture", strings.NewReader("draft")))
		if response.Code != 405 {
			t.Errorf("reference relayed %s", method)
		}
	}
	if calls != 2 {
		t.Fatalf("mutations reached retained renderer: %d upstream requests", calls)
	}
}
