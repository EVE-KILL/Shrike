package frontend

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBattleDirectorySSRAndFilterFragment(t *testing.T) {
	var battleQuery url.Values
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "eve-kill.com" || r.Header.Get("Cookie") != "session=test" {
			t.Errorf("battle API scope host=%q cookie=%q", r.Host, r.Header.Get("Cookie"))
		}
		switch r.URL.Path {
		case "/api/site":
			_, _ = w.Write([]byte(`{"domain":null,"isDomainHost":false}`))
		case "/api/entity/resolve":
			if r.URL.Query().Get("type") != "system" || r.URL.Query().Get("id") != "30000142" {
				t.Errorf("entity resolve query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"name":"Jita"}`))
		case "/api/conflicts/battles":
			battleQuery = r.URL.Query()
			_, _ = w.Write([]byte(`{"page":2,"limit":50,"years":[{"year":2025,"count":42}],"battles":[{"battle_id":123,"solar_system_id":30000142,"solar_system_name":"Jita","solar_system_security":0.9,"region_id":10000002,"region_name":"The Forge","start_time":"2026-09-27T12:00:00Z","duration_minutes":65,"kill_count":25,"total_isk_destroyed":12300000000,"is_multi_party":true,"teams":[{"total_kills":20,"total_isk_destroyed":10000000000,"alliance_count":1,"top_alliances":[{"alliance_id":99003581,"alliance_name":"Fraternity."}]},{"total_kills":5,"total_isk_destroyed":2300000000,"alliance_count":1,"top_alliances":[{"corporation_id":98598862,"corporation_name":"Test Corp"}]}]}]}`))
		default:
			http.NotFound(w, r)
		}
	})
	handler := New(Options{Production: true, ConflictsAPI: apiHandler})
	request := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, canonicalOrigin+path, nil)
		r.Header.Set("Cookie", "session=test")
		handler.ServeHTTP(w, r)
		return w
	}
	path := "/battles?hours=24&minKills=25&minIsk=10b&custom=1&eType=system&eId=30000142&page=2"
	page := request(path)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Jita") || !strings.Contains(page.Body.String(), `/battle/123`) || !strings.Contains(page.Body.String(), `data-battles-page="1"`) || !strings.Contains(page.Body.String(), "1h 5m") {
		t.Fatalf("battle page status=%d missing content", page.Code)
	}
	if strings.Contains(page.Body.String(), "This killboard only") || !strings.Contains(strings.Split(page.Body.String(), "battles-view-tabs")[0], "Jita") {
		t.Fatal("global page must show its resolved entity chip before the tabs, without a domain badge")
	}
	for key, want := range map[string]string{"hours": "24", "minKills": "25", "minIsk": "10000000000", "custom": "true", "systemId": "30000142", "page": "2", "limit": "50"} {
		if got := battleQuery.Get(key); got != want {
			t.Errorf("API query %s=%q, want %q", key, got, want)
		}
	}
	fragment := request("/_fragments" + path)
	if fragment.Code != http.StatusOK || !strings.Contains(fragment.Body.String(), `data-battles-results`) || !strings.Contains(fragment.Body.String(), `data-battles-chips`) || strings.Contains(fragment.Body.String(), "<header") {
		t.Fatalf("battle fragment status=%d did not stay scoped", fragment.Code)
	}
	mapPage := request("/battles?view=map")
	if mapPage.Code != http.StatusOK || !strings.Contains(mapPage.Body.String(), `data-battles-map-panel`) || strings.Contains(mapPage.Body.String(), `id="battle-panel-list"`) {
		t.Fatalf("battle map page status=%d panel mismatch", mapPage.Code)
	}
	for _, invalid := range []string{"hours=25", "minIsk=bogus", "eType=system", "page=-1"} {
		if got := request("/battles?" + invalid).Code; got != http.StatusNotFound {
			t.Errorf("invalid battle filter %q status=%d", invalid, got)
		}
	}
}

func TestBattleDirectoryBadgeRequiresConfiguredDomain(t *testing.T) {
	for _, configured := range []bool{false, true} {
		handler := New(Options{ConflictsAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Host != "unknown.example" {
				t.Fatal("site bootstrap lost the request host")
			}
			switch r.URL.Path {
			case "/api/site":
				if configured {
					_, _ = w.Write([]byte(`{"domain":{"id":42},"isDomainHost":true}`))
				} else {
					_, _ = w.Write([]byte(`{"domain":null,"isDomainHost":true}`))
				}
			case "/api/conflicts/battles":
				_, _ = w.Write([]byte(`{"battles":[],"years":[]}`))
			default:
				http.NotFound(w, r)
			}
		})})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://unknown.example/battles", nil))
		if response.Code != 200 || strings.Contains(response.Body.String(), "This killboard only") != configured {
			t.Fatal("badge did not follow the authoritative site configuration")
		}
	}
}
