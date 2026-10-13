package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

func TestSavedFitSessionSSRAndEscaping(t *testing.T) {
	session := &campaignSessionTestReader{}
	handler := New(Options{Production: true, Sessions: session, FittingsAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/fit/PublicFit00001" || r.Header.Get("Cookie") != "evelogin=fixture" || r.URL.RawQuery != "" {
			t.Fatal("saved fitting request lost its identity or leaked editor query parameters")
		}
		_, _ = w.Write([]byte(`{"fit":{"fit_id":"PublicFit00001","owner_character_id":42,"ship_type_id":626,"name":"<script>unsafe()</script>","description":"</script><script>unsafe()</script>","visibility":3,"rating_avg":4.5,"rating_count":2},"items":[{"slot_group":6,"ordinal":0,"type_id":2454,"state":2,"quantity":3}],"viewer_rating":5}`))
	})})
	request := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", canonicalOrigin+"/fit/PublicFit00001?fit=ignored", nil)
		r.AddCookie(&http.Cookie{Name: "evelogin", Value: "fixture"})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	guest := request()
	if guest.Code != 200 || guest.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("saved fit document must never enter shared caches")
	}
	body := guest.Body.String()
	for _, expected := range []string{`data-fit-owner="false"`, `data-fit-action="save" hidden`, `data-fit-action="fork" disabled`, `id="saved-fit-state" type="application/json"`, `data-target-slug="PublicFit00001"`, `data-target-type="8"`, `data-target-id="0"`, `<template data-fit-comments-template>`, `aria-pressed="true"`, `\u003c/script\u003e`} {
		if !strings.Contains(body, expected) {
			t.Errorf("missing %s", expected)
		}
	}
	if strings.Contains(body, "<script>unsafe()") {
		t.Fatal("unescaped fit contents")
	}
	session.user = &api.Principal{CharacterID: 42}
	owner := request().Body.String()
	if !strings.Contains(owner, `data-fit-owner="true"`) || strings.Contains(owner, `data-fit-action="save" hidden`) || !strings.Contains(owner, `data-fit-action="fork" hidden`) {
		t.Fatal("owner and viewer action modes did not follow the source")
	}
}

func TestSavedFitAccessErrorsAndPrivateIndexing(t *testing.T) {
	for _, status := range []int{403, 404, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			handler := New(Options{Production: true, FittingsAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "secret database details", status) })})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", canonicalOrigin+"/fit/PublicFit00001", nil))
			want := status
			if status == 503 {
				want = 500
			}
			if w.Code != want || strings.Contains(w.Body.String(), "secret database") || strings.Contains(w.Body.String(), "saved-fit-state") || !strings.Contains(w.Body.String(), "Browse Fits") || w.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
				t.Fatal("access failure leaked contents or did not render source error card")
			}
		})
	}
	handler := New(Options{Production: true, FittingsAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"fit":{"fit_id":"PrivateFit0001","ship_type_id":626,"name":"Private","visibility":0},"items":[]}`))
	})})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", canonicalOrigin+"/fit/PrivateFit0001", nil))
	if w.Code != 200 || w.Header().Get("X-Robots-Tag") != "noindex, nofollow" || w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("restricted fitting became indexable/cacheable")
	}
}

func TestSavedFitRejectsIncompleteResponse(t *testing.T) {
	handler := New(Options{FittingsAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) })})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/fit/PublicFit00001", nil))
	if w.Code != 500 || strings.Contains(w.Body.String(), "saved-fit-state") || !strings.Contains(w.Body.String(), "Failed to load the fit") {
		t.Fatal("incomplete API response produced an unusable editor")
	}
}
