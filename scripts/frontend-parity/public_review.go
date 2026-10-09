package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// The facade preserves page/API transport for populated reviews while
// preventing account credentials or writes from reaching public data.
func publicReviewProxy(target *url.URL) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Public frontend review is read-only", http.StatusMethodNotAllowed)
			return
		}
		request := r.Clone(r.Context())
		request.Header.Del("Authorization")
		request.Header.Del("Cookie")
		// Theme/default-tab/background preferences remain testable on SSR.
		for _, cookie := range r.Cookies() {
			switch cookie.Name {
			case "ek_theme", "ek_prefs", "siteBackground":
				request.AddCookie(cookie)
			}
		}
		proxy.ServeHTTP(w, request)
	})
}

func runPublicReview(ctx context.Context, public string, port int, injectFaults bool) error {
	if port < 1 || port > 65535 {
		return errors.New("public review needs a valid loopback port")
	}
	target, err := localURL(public)
	if err != nil {
		return err
	}
	proxy := publicReviewProxy(target)
	faults := &fixtureFaults{enabled: injectFaults}
	renderer := fixtureFrameHandler(proxy)
	handler := faults.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fixtureViewportPage(w, r) || fixtureAudioTrace(w, r) {
			return
		}
		renderer.ServeHTTP(w, r)
	}))
	log.Printf("Read-only public review: %d Go", port)
	return serve(ctx, []*http.Server{{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: handler}})
}
