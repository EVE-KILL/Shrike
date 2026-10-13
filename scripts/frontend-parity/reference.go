package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Keep the retained HTML/CSS/JS untouched while allowing modern local read
// fixtures to drive both renderers. Public comparisons never forward credentials,
// accept authentication cookies or relay mutations to the retained application.
func referenceRenderer(target *url.URL) http.Handler {
	proxy := anonymousFixtureProxy(target)
	proxy.ModifyResponse = func(response *http.Response) error {
		response.Header.Del("Set-Cookie")
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Reference renderer is read-only", http.StatusMethodNotAllowed)
			return
		}
		proxy.ServeHTTP(w, r)
	})
}

func anonymousFixtureProxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{Rewrite: func(r *httputil.ProxyRequest) {
		r.SetURL(target)
		// Retain the comparison board's Host as NewSingleHostReverseProxy did.
		r.Out.Host = r.In.Host
		r.Out.Header.Del("Cookie")
		r.Out.Header.Del("Authorization")
		r.SetXForwarded()
	}}
}
