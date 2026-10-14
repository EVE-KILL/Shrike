package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicNavbarKeepsIdentityInPrivateFragmentAndIgnoresCookieHints(t *testing.T) {
	calls := 0
	handler := New(Options{Production: true, AccountAPI: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ })})
	for _, cookie := range []string{"", "ek_auth=42:PrivateFixture; evelogin=spoofed"} {
		request := httptest.NewRequest("GET", "/", nil)
		request.Header.Set("Cookie", cookie)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		body := response.Body.String()
		if response.Code != http.StatusOK || calls != 0 || strings.Contains(body, "PrivateFixture") {
			t.Fatalf("public navbar loaded or exposed private identity: status=%d calls=%d", response.Code, calls)
		}
		if !strings.Contains(body, `data-account-slot="desktop"`) || !strings.Contains(body, `data-account-slot="mobile"`) || strings.Contains(body, "data-account-avatar") || strings.Contains(body, "data-account-session") || !strings.Contains(body, "site.js") {
			t.Fatal("public page omitted the browser account shell")
		}
	}
}
