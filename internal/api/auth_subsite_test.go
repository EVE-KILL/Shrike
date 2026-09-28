package api

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type loginDomainDB struct{ stubDatabase }

func (loginDomainDB) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	return loginDomainRow(args[0] == "vip")
}

type loginDomainRow bool

func (r loginDomainRow) Scan(dest ...any) error {
	*dest[0].(*bool) = bool(r)
	return nil
}

func TestSubsiteLoginRoundTrip(t *testing.T) {
	for _, endpoint := range []string{"/auth/login", "/auth/eve/start"} {
		t.Run(endpoint, func(t *testing.T) {
			rig := newAuthTestRig(t)
			rig.production = true
			rig.domainDB = loginDomainDB{}
			handler := rig.handler(t)
			jar, err := cookiejar.New(nil)
			if err != nil {
				t.Fatal(err)
			}
			request := func(target string, withCookies bool) *httptest.ResponseRecorder {
				t.Helper()
				req := httptest.NewRequest(http.MethodGet, target, nil)
				if withCookies {
					for _, cookie := range jar.Cookies(req.URL) {
						req.AddCookie(cookie)
					}
				}
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				if withCookies {
					jar.SetCookies(req.URL, rec.Result().Cookies())
				}
				return rec
			}
			path := "/corporation/98630834/battles?page=2#recent"
			start := request("https://vip.eve-kill.com"+endpoint+
				"?redirect="+url.QueryEscape(path)+"&charKm=0&corpKm=1&delay=6", true)
			hop := start.Header().Get("Location")
			if endpoint == "/auth/login" {
				var body struct {
					URL string `json:"url"`
				}
				decodeResponse(t, start, &body)
				hop = body.URL
				if start.Code != http.StatusOK {
					t.Fatalf("login URL: %d %s", start.Code, start.Body.String())
				}
			} else if start.Code != http.StatusFound {
				t.Fatalf("start: %d %s", start.Code, start.Body.String())
			}
			if !strings.HasPrefix(hop, "https://eve-kill.com/auth/eve/start?") ||
				len(start.Result().Cookies()) != 0 || len(rig.flows.items) != 0 {
				t.Fatalf("subsite must only hop to main: %q, cookies=%v", hop, start.Result().Cookies())
			}
			main := request(hop, true)
			if main.Code != http.StatusFound {
				t.Fatalf("main start: %d %s", main.Code, main.Body.String())
			}
			flowCookie := cookieWithPrefix(main.Result().Cookies(), oauthFlowCookiePrefix)
			if flowCookie == nil || flowCookie.Domain != "" || !flowCookie.HttpOnly || !flowCookie.Secure {
				t.Fatalf("main binding cookie: %#v", flowCookie)
			}
			authorization, err := url.Parse(main.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			callback := "/auth/callback?code=good&state=" + url.QueryEscape(authorization.Query().Get("state"))
			wrong := request("https://eve-kill.com"+callback, false)
			if wrong.Code != http.StatusBadRequest {
				t.Fatalf("unbound callback: %d", wrong.Code)
			}
			// A real cookie jar must not send the main site's binding to VIP.
			wrongHost := request("https://vip.eve-kill.com"+callback, true)
			if wrongHost.Code != http.StatusBadRequest {
				t.Fatalf("binding leaked to subsite: %d", wrongHost.Code)
			}
			complete := request("https://eve-kill.com"+callback, true)
			if complete.Code != http.StatusFound {
				t.Fatalf("callback: %d %s", complete.Code, complete.Body.String())
			}
			want := appendLoginMarker("https://vip.eve-kill.com" + path)
			if got := complete.Header().Get("Location"); got != want {
				t.Fatalf("return = %q, want %q", got, want)
			}
			if rig.store.completed.Delay != 6 || len(rig.oauth.scopes) != 2 {
				t.Fatalf("login options lost: %#v, %v", rig.store.completed, rig.oauth.scopes)
			}
			for _, host := range []string{"eve-kill.com", "vip.eve-kill.com", "another.eve-kill.com"} {
				u, _ := url.Parse("https://" + host + "/")
				if cookieNamed(jar.Cookies(u), authSessionCookie) == nil ||
					cookieNamed(jar.Cookies(u), authHintCookie) == nil {
					t.Errorf("login not shared with %s", host)
				}
			}
			if replay := request("https://eve-kill.com"+callback, true); replay.Code != http.StatusBadRequest {
				t.Fatalf("callback replay accepted: %d", replay.Code)
			}
		})
	}
}

func TestLoginReturnHostRejectsUntrustedDestinations(t *testing.T) {
	for _, host := range []string{
		"evil.example", "eve-kill.com.evil.example", "vip.eve-kill.com@evil.example",
		"vip.eve-kill.com:8443", "vip.eve-kill.com/path", "unknown.eve-kill.com",
		"nested.vip.eve-kill.com", "vip.eve-kill.com\\evil", "vip.eve-kill.com\r\n",
	} {
		t.Run(host, func(t *testing.T) {
			rig := newAuthTestRig(t)
			rig.production = true
			rig.domainDB = loginDomainDB{}
			req := httptest.NewRequest(http.MethodGet,
				"https://eve-kill.com/auth/eve/start?returnHost="+url.QueryEscape(host), nil)
			rec := httptest.NewRecorder()
			rig.handler(t).ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest || len(rig.flows.items) != 0 {
				t.Fatalf("untrusted destination accepted: %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestSubsiteLoginUsesActualHost(t *testing.T) {
	rig := newAuthTestRig(t)
	rig.production = true
	rig.domainDB = loginDomainDB{}
	req := httptest.NewRequest(http.MethodGet,
		"https://vip.eve-kill.com/auth/eve/start?returnHost=evil.example", nil)
	req.Header.Set("X-Forwarded-Host", "evil.example")
	rec := httptest.NewRecorder()
	rig.handler(t).ServeHTTP(rec, req)
	hop, err := url.Parse(rec.Header().Get("Location"))
	if err != nil || rec.Code != http.StatusFound || hop.Query().Get("returnHost") != "vip.eve-kill.com" {
		t.Fatalf("incorrect hop: %d %s", rec.Code, rec.Body.String())
	}
}

func TestMainSiteLoginStaysDirect(t *testing.T) {
	rig := newAuthTestRig(t)
	rig.production = true
	req := httptest.NewRequest(http.MethodGet,
		"https://eve-kill.com/auth/eve/start?redirect=%2Fsettings", nil)
	req.Header.Set("X-Forwarded-Host", "vip.eve-kill.com")
	rec := httptest.NewRecorder()
	rig.handler(t).ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || len(rig.flows.items) != 1 {
		t.Fatalf("main login did not start directly: %d %s", rec.Code, rec.Body.String())
	}
	for _, item := range rig.flows.items {
		if item.flow.ReturnTo != "/settings" {
			t.Fatalf("main return path changed: %q", item.flow.ReturnTo)
		}
	}
}
