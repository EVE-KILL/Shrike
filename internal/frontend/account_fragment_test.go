package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
	"golang.org/x/net/html"
)

func TestAccountFragmentRendersPrivateResponsiveIdentity(t *testing.T) {
	for _, admin := range []bool{false, true} {
		t.Run(map[bool]string{false: "member", true: "admin"}[admin], func(t *testing.T) {
			calls := 0
			account := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/auth/me" || r.URL.RawQuery != "" || r.Host != "void.eve-kill.com" || r.Header.Get("Cookie") != "evelogin=fixture-session" {
					t.Fatalf("session read lost its original host/cookie or accepted fragment query: %s %s %s", r.Method, r.URL, r.Host)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"user": map[string]any{
					"characterId": 4242, "characterName": `Pilot <script>alert("identity")</script>`, "isAdmin": admin,
					"corporationId": 10, "corporationName": "Corp & Friends", "allianceId": 20, "allianceName": "Alliance",
					"lastSeenNotificationId": 81, "characterOwnerHash": "private-owner-hash", "accessToken": "private-token",
					"settings": map[string]any{"defaultTabs": map[string]any{"character": "losses"}, "theme": map[string]any{"primary": "#aabbcc"}, "secret": "private-settings"},
				}})
			})
			boards := &domainReaderStub{hosts: map[string]*api.SiteDomainConfiguration{"void.eve-kill.com": testDomain("VOID")}}
			handler := New(Options{Production: true, Statistics: boards, AccountAPI: account})
			r := httptest.NewRequest("GET", "https://void.eve-kill.com/_fragments/account?characterId=7", nil)
			r.Header.Set("Cookie", "evelogin=fixture-session")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			body := w.Body.String()
			if w.Code != 200 || calls != 1 || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
				t.Fatalf("fragment status/cache/reads: %d %s %d", w.Code, w.Header().Get("Cache-Control"), calls)
			}
			for _, want := range []string{`data-account-content="desktop"`, `data-account-content="mobile"`, `href="/character/4242"`, `href="/corporation/10"`, `href="/alliance/20"`, `/characters/4242/portrait?size=64`, `/corporations/10/logo?size=32`, `width="32" height="32"`, `data-notifications`, `data-account-logout`} {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q", want)
				}
			}
			if strings.Contains(body, `href="/admin"`) != admin {
				t.Fatal("admin link visibility did not come from authoritative session")
			}
			for _, secret := range []string{"private-owner-hash", "private-token", "private-settings", `<script>alert`, `data-account-guest`} {
				if strings.Contains(body, secret) {
					t.Errorf("fragment exposed %q", secret)
				}
			}
			root, err := html.Parse(strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			var seed string
			var walk func(*html.Node)
			walk = func(node *html.Node) {
				for _, attr := range node.Attr {
					if attr.Key == "data-account-session" {
						seed = attr.Val
					}
				}
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					walk(child)
				}
			}
			walk(root)
			var user navbarUser
			if err := json.Unmarshal([]byte(seed), &user); err != nil || user.CharacterID != 4242 || user.LastSeenNotificationID != 81 || user.Settings.DefaultTabs["character"] != "losses" || user.CharacterName != `Pilot <script>alert("identity")</script>` {
				t.Fatalf("escaped bootstrap did not preserve display/preferences data: %v", err)
			}
			unknown := httptest.NewRecorder()
			handler.ServeHTTP(unknown, httptest.NewRequest("GET", "https://unknown.eve-kill.com/_fragments/account", nil))
			if unknown.Code != 404 || calls != 1 || strings.Contains(unknown.Body.String(), "data-account-session") || unknown.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal("account fragment bypassed custom-board host validation")
			}
		})
	}
}

func TestAccountFragmentAnonymousExpiredFailureAndHead(t *testing.T) {
	for _, test := range []struct {
		name, method, response string
		apiStatus, wantStatus  int
		clear                  bool
	}{
		{"anonymous", "GET", `{"user":null}`, 200, 200, false},
		{"expired", "GET", `{"user":null}`, 200, 200, true},
		{"head", "HEAD", `{"user":null}`, 200, 200, false},
		{"unavailable", "GET", `private database details`, 503, 503, true},
		{"malformed", "GET", `broken json`, 200, 503, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := New(Options{Production: true, AccountAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.clear {
					http.SetCookie(w, &http.Cookie{Name: "evelogin", Value: "", MaxAge: -1, Path: "/", HttpOnly: true, Secure: true})
				}
				w.WriteHeader(test.apiStatus)
				_, _ = w.Write([]byte(test.response))
			})})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(test.method, canonicalOrigin+"/_fragments/account", nil))
			if w.Code != test.wantStatus || w.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatalf("status/cache: %d %s", w.Code, w.Header().Get("Cache-Control"))
			}
			if (len(w.Result().Cookies()) != 0) != test.clear {
				t.Fatal("auth cookie lifecycle headers were lost")
			}
			if test.method == "HEAD" {
				if w.Body.Len() != 0 || w.Header().Get("Content-Length") == "" {
					t.Fatal("HEAD did not match GET headers with an empty body")
				}
			} else if test.wantStatus == 200 {
				if strings.Count(w.Body.String(), "data-account-guest") != 2 || strings.Contains(w.Body.String(), "data-account-avatar") || strings.Contains(w.Body.String(), "data-notifications") {
					t.Fatal("guest fragment included authenticated controls")
				}
			} else if strings.Contains(w.Body.String(), test.response) {
				t.Fatal("failure exposed internal API details")
			}
		})
	}
}

func TestAccountFragmentLeavesPublicHomepageCacheIndependent(t *testing.T) {
	calls := 0
	handler := New(Options{Production: true, Homepage: homepageReaderStub{}, AccountAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"user":null}`))
	})})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", canonicalOrigin+"/", nil))
	if !strings.Contains(w.Header().Get("Cache-Control"), "s-maxage=30") || calls != 0 || strings.Contains(w.Body.String(), "data-account-session") {
		t.Fatal("public HTML became personalized or lost shared cache eligibility")
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", canonicalOrigin+"/_fragments/account", nil))
	if w.Code != 200 || calls != 1 || w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("fragment inherited public page caching")
	}
}
