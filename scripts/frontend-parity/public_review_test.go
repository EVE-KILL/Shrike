package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestPublicReviewPreservesAppearanceButNeverForwardsAuthenticationOrWrites(t *testing.T) {
	reads := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads++
		if r.Header.Get("Authorization") != "" || r.Host != "review.localhost" {
			t.Errorf("auth forwarded or host lost")
		}
		if _, err := r.Cookie("evelogin"); err == nil {
			t.Error("session cookie reached public data")
		}
		if _, err := r.Cookie("ek_auth"); err == nil {
			t.Error("identity hint reached public data")
		}
		if theme, err := r.Cookie("ek_theme"); err != nil || theme.Value != "appearance" {
			t.Error("theme preference lost")
		}
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	handler := publicReviewProxy(target)
	for _, method := range []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"} {
		request := httptest.NewRequest(method, "http://review.localhost/api/user/config", nil)
		request.Header.Set("Authorization", "Bearer disposable-test")
		request.Header.Set("Cookie", "evelogin=disposable-test; ek_auth=disposable-hint; ek_theme=appearance")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		want := 405
		if method == "GET" || method == "HEAD" {
			want = 200
		}
		if response.Code != want {
			t.Errorf("%s status %d, want %d", method, response.Code, want)
		}
		if request.Header.Get("Authorization") == "" {
			t.Error("proxy mutated the caller's request")
		}
	}
	if reads != 2 {
		t.Fatalf("writes reached public upstream: %d calls", reads)
	}
}

func TestPublicReviewRejectsInvalidPorts(t *testing.T) {
	for _, port := range []int{0, 65536} {
		if err := runPublicReview(context.Background(), "http://127.0.0.1:4200", port, false); err == nil {
			t.Fatal("invalid port accepted")
		}
	}
}
