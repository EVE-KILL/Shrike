package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSettingsPrivateIdentityAndOverview(t *testing.T) {
	account := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cookie, _ := r.Cookie("evelogin")
		switch r.URL.Path {
		case "/auth/me":
			if cookie == nil || cookie.Value != "owned-fixture-session" {
				_, _ = w.Write([]byte(`{"user":null}`))
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"user": map[string]any{"characterId": 42, "characterName": "Private fixture <script>name</script>", "characterOwnerHash": "private-owner-hash", "settings": map[string]any{"theme": map[string]string{"brandPrimary": "#abcdef"}}}})
		case "/api/user/overview":
			if cookie == nil || cookie.Value != "owned-fixture-session" {
				t.Error("account read lost the caller's session")
			}
			_, _ = w.Write([]byte(`{"account":{"createdAt":"2026-01-01T00:00:00Z","lastLogin":null},"esiToken":null,"esiStats":{"requests_24h":17,"total_requests":1234,"errors_24h":0,"last_request":"2026-10-01T19:00:00Z"}}`))
		case "/api/user/manageable-entities":
			_, _ = w.Write([]byte(`{"character":{"id":42,"name":"Private fixture","canEdit":true,"esi_description":"ESI fallback","custom_description":"Published bio","custom_description_format":"markdown","pending_submission":{"body":"Pending <script>draft</script>","body_format":"eve_html"}},"corporation":{"id":10,"name":"Fixture Corp","ticker":"FIX","canEdit":false,"ceo_name":"Fixture CEO"}}`))
		case "/api/user/wallet":
			if r.URL.Query().Get("page") != "2" {
				t.Error("private account reader dropped wallet pagination")
			}
			_, _ = w.Write([]byte(`{"character":{"character_id":42,"character_name":"Private fixture"},"corporation":{"corporation_id":10,"name":"Fixture Corp","ticker":"FIX"},"availableBalance":"1234.50","reservedBalance":"10.00","totalBalance":"1244.50","transactions":[{"id":9,"type":3,"amount":"-10.00","balance_after":"1244.50","description":"Fixture funding","campaign_id":"local-fixture","created_at":"2026-10-01T00:00:00Z"}],"page":2,"hasMore":false}`))
		case "/auth/token-info":
			_, _ = w.Write([]byte(`{"scopes":["esi-killmails.read_killmails.v1"],"token_expiry":"2099-01-01T00:00:00Z"}`))
		case "/api/user/esi":
			_, _ = w.Write([]byte(`{"rateLimit":{"request_count":1234},"responseTime":{"avg_ms":1234,"p95_ms":4000},"volumeByHour":[]}`))
		case "/api/user/esi-logs":
			_, _ = w.Write([]byte(`{"rows":[],"total":0,"sources":[],"pages":0}`))
		case "/api/user/comments":
			_, _ = w.Write([]byte(`{"comments":[{"id":23,"target_type":1,"target_id":100,"body_html":"<p>Private fixture body</p><script>alert(1)</script>","character_id":42,"created_at":"2026-10-01T00:00:00Z"}],"nextCursor":23}`))
		case "/api/user/domains":
			_, _ = w.Write([]byte(`{"domains":[{"id":5,"subdomain":"private-fixture-board","site_name":"Private <script>board</script>","active":true,"entities":[{"id":42,"type":"character","name":"Private fixture"}],"theme":{},"widgets":{},"navbar_links":[]}]}`))
		default:
			http.NotFound(w, r)
		}
	})
	handler := New(Options{AccountAPI: account})
	for _, path := range []string{"/settings", "/settings/appearance", "/settings/preferences", "/settings/sessions", "/settings/description", "/settings/wallet?page=2", "/settings/esi", "/settings/comments", "/settings/domains"} {
		request := httptest.NewRequest("GET", "https://eve-kill.com"+path, nil)
		request.AddCookie(&http.Cookie{Name: "evelogin", Value: "owned-fixture-session"})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, request)
		body := w.Body.String()
		if w.Code != 200 || w.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(body, `name="robots" content="noindex, nofollow"`) {
			t.Fatalf("private response %s: status=%d cache=%s", path, w.Code, w.Header().Get("Cache-Control"))
		}
		if strings.Contains(body, "<script>name</script>") || !strings.Contains(body, "&lt;script&gt;name&lt;/script&gt;") {
			t.Error("untrusted character name was not escaped")
		}
		if path == "/settings" && (!strings.Contains(body, "1,234 total") || strings.Contains(body, "Could not load your account overview")) {
			t.Error("overview mixed timestamps and numeric statistics incorrectly")
		}
		if path == "/settings/description" && (!strings.Contains(body, "Pending &lt;script&gt;draft&lt;/script&gt;") || strings.Contains(body, "ESI fallback") || strings.Contains(body, `data-settings-bio="corporation"`) || !strings.Contains(body, `data-format="eve_html"`)) {
			t.Error("bio permissions, pending draft precedence, format, or escaping differ")
		}
		if strings.HasPrefix(path, "/settings/wallet") && (!strings.Contains(body, "1,234.50 ISK") || !strings.Contains(body, `href="/campaign/local-fixture"`) || !strings.Contains(body, "-10.00 ISK")) {
			t.Error("private wallet money or campaign reference differs")
		}
		if path == "/settings/esi" && (!strings.Contains(body, "1,234") || strings.Contains(body, "4,000") || !strings.Contains(body, "Granted Scopes")) {
			t.Error("ESI metric or token formatting differs")
		}
		if path == "/settings/comments" && (!strings.Contains(body, "/kill/100#comment-23") || strings.Contains(body, "<script>alert(1)</script>") || !strings.Contains(body, "Private fixture body")) {
			t.Error("private comment targets or stored HTML are unsafe")
		}
		if path == "/settings/domains" && (!strings.Contains(body, "https://private-fixture-board.eve-kill.com") || strings.Contains(body, "<script>board</script>") || !strings.Contains(body, "Private &lt;script&gt;board&lt;/script&gt;")) {
			t.Error("private domain data or escaping differs")
		}
		controller := map[string]string{
			"/settings/preferences":   "settings-preferences",
			"/settings/appearance":    "settings-appearance",
			"/settings/sessions":      "settings-sessions",
			"/settings/description":   "settings-bio",
			"/settings/wallet?page=2": "settings-wallet",
			"/settings/comments":      "settings-comments",
			"/settings/domains":       "settings-domains",
		}[path]
		if !strings.Contains(body, `data-controller="settings"`) || (controller != "" && !strings.Contains(body, `data-controller="`+controller+`"`)) {
			t.Errorf("settings page %s lost its browser component mount", path)
		}
	}
	// Neither a stale auth hint nor another cookie can unlock private content.
	for _, cookie := range []http.Cookie{{Name: "ek_auth", Value: "42:Private fixture"}, {Name: "evelogin", Value: "expired-fixture-session"}} {
		r := httptest.NewRequest("GET", "/settings/appearance", nil)
		r.AddCookie(&cookie)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Log in with EVE Online to access your settings") || strings.Contains(w.Body.String(), "private-owner-hash") || strings.Contains(w.Body.String(), "#abcdef") {
			t.Fatal("private identity escaped the login gate")
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/_fragments/settings/preferences", nil))
	if w.Code != 401 || w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("private fragments must reject expired sessions without caching")
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/_fragments/settings-comments", nil))
	if w.Code != 401 || w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("private comment fragments exposed expired-session content")
	}
}
