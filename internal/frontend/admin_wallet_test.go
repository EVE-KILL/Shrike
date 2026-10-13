package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminWalletFragmentPrivacyQueryAndStoredText(t *testing.T) {
	for _, identity := range []string{"admin", "reader", "anonymous"} {
		t.Run(identity, func(t *testing.T) {
			api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Fatal("wallet rendering wrote data")
				}
				if r.URL.Path == "/auth/me" {
					var user *settingsUser
					if identity != "anonymous" {
						user = &settingsUser{ID: 42, Admin: identity == "admin"}
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"user": user})
					return
				}
				if identity != "admin" {
					t.Fatal("wallet read before authorization")
				}
				if r.URL.Path != "/api/admin/wallet" || r.URL.Query().Get("page") != "2" || r.URL.Query().Get("division") != "3" || r.URL.Query().Has("limit") {
					t.Fatal("changed or unbounded wallet query")
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"corporation": map[string]any{"corporation_id": 98779905, "name": "<script>corporation</script>"}, "totalBalance": "1234.50", "page": 2, "journal": []map[string]any{{"division": 3, "journal_id": 9, "description": "<script>description</script>", "ref_type": "player_donation", "amount": "NaN"}}, "prizeSettlements": []any{}, "walletReferences": []any{}})
			})
			handler := New(Options{Production: true, AccountAPI: api})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/_fragments/admin/wallet?page=2&division=3&limit=999999&wallet_auth=ok&message=%3Cscript%3Ecallback%3C%2Fscript%3E", nil))
			want := 200
			if identity == "reader" {
				want = 403
			}
			if identity == "anonymous" {
				want = 401
			}
			if w.Code != want || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
				t.Fatal("wallet fragment could expose private data")
			}
			if identity == "admin" {
				body := w.Body.String()
				if strings.Contains(body, "<script>") || !strings.Contains(body, "&lt;script&gt;description&lt;/script&gt;") || !strings.Contains(body, "&lt;script&gt;callback&lt;/script&gt;") || !strings.Contains(body, "Page 2") {
					t.Fatal("wallet text, callback, or displayed page diverged")
				}
			}
		})
	}
}
