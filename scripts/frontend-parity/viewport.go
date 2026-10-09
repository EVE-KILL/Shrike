package main

import (
	"bytes"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
)

// A fixed-size, same-origin frame tests real mobile media queries when browser
// viewport emulation is unavailable. Framing is permitted only in this local
// fixture and only for the explicitly marked comparison request.
func fixtureFrameHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("__fixture_frame") != "1" {
			next.ServeHTTP(w, r)
			return
		}
		response := httptest.NewRecorder()
		next.ServeHTTP(response, r)
		for name, values := range response.Header() {
			for _, value := range values {
				if strings.EqualFold(name, "Content-Security-Policy") {
					value = strings.ReplaceAll(value, "frame-ancestors 'none'", "frame-ancestors 'self'")
				}
				if strings.EqualFold(name, "X-Frame-Options") {
					value = "SAMEORIGIN"
				}
				w.Header().Add(name, value)
			}
		}
		body := response.Body.Bytes()
		if r.URL.Query().Get("__fixture_audio") == "1" {
			body = bytes.Replace(body, []byte("<head>"), []byte(`<head><script src="/__fixture/audio-trace.js"></script>`), 1)
			w.Header().Del("Content-Length")
		}
		w.WriteHeader(response.Code)
		_, _ = w.Write(body)
	})
}
func fixtureViewportPath(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "\\\r\n") {
		return "", fmt.Errorf("mobile preview must use a relative page path")
	}
	q := u.Query()
	q.Set("__fixture_frame", "1")
	u.RawQuery = q.Encode()
	return u.String(), nil
}
func fixtureViewportPage(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/__fixture/viewport" {
		return false
	}
	path, err := fixtureViewportPath(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return true
	}
	if r.URL.Query().Get("audio") == "1" {
		u, _ := url.Parse(path)
		q := u.Query()
		q.Set("__fixture_audio", "1")
		u.RawQuery = q.Encode()
		path = u.String()
	}
	width, _ := strconv.Atoi(r.URL.Query().Get("width"))
	height, _ := strconv.Atoi(r.URL.Query().Get("height"))
	if width < 280 || width > 1920 {
		width = 393
	}
	if height < 320 || height > 1600 {
		height = 852
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = fmt.Fprintf(w, `<!doctype html><html><head><title>Mobile frontend comparison</title></head><body style="margin:24px;background:#252525;color:#ddd;font-family:system-ui"><p>%d × %d · <a style="color:#93c5fd" href="%s">Open page</a></p><iframe title="Mobile preview" src="%s" width="%d" height="%d" style="border:1px solid #555;display:block"></iframe></body></html>`, width, height, html.EscapeString(r.URL.Query().Get("path")), html.EscapeString(path), width, height)
	return true
}
