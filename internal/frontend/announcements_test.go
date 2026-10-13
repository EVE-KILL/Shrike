package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnnouncementHostsNeverBlockPublicDocumentOnFeedOrIdentity(t *testing.T) {
	api := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("public SSR waited for announcement or identity API")
	})
	handler := New(Options{Production: true, AccountAPI: api})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/about", nil))
	if w.Code != 200 {
		t.Fatal("public document failed")
	}
	for _, marker := range []string{"data-announcement-banners", "data-announcement-modal", "data-announcement-ticker", "site.js", "announcements.css"} {
		if !strings.Contains(w.Body.String(), marker) {
			t.Fatalf("shared announcement host missing %s", marker)
		}
	}
}
