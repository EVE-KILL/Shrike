package frontend

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLegacyPageRedirects(t *testing.T) {
	handler := New(Options{})
	for _, tt := range []struct{ path, location string }{
		{"/ship/11186", "/item/11186"},
		{"/ship/11186/kills", "/item/11186/kills"},
		{"/shipgroup/25", "/market/ships"},
		{"/shipgroup/25/kills", "/market/ships"},
		{"/wars/755805", "/war/755805"},
		{"/killmail/138664454", "/kill/138664454"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if w.Code != http.StatusMovedPermanently || w.Header().Get("Location") != tt.location {
				t.Fatalf("redirect = %d %q, want 301 %q", w.Code, w.Header().Get("Location"), tt.location)
			}
		})
	}
	for _, path := range []string{"/ship/nope", "/wars/nope", "/killmail/0"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("invalid redirect %q returned %d", path, w.Code)
		}
	}
}
