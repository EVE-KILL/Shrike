package frontend

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

type domainReaderStub struct {
	statisticsTestReader
	hosts map[string]*api.SiteDomainConfiguration
	err   error
}

func (r *domainReaderStub) StatisticsSite(_ context.Context, host string) (api.SiteConfigurationResponse, error) {
	return api.SiteConfigurationResponse{IsDomainHost: true, Domain: r.hosts[host]}, r.err
}
func testDomain(name string) *api.SiteDomainConfiguration {
	transparent := true
	ratio := "250px_1fr"
	text := "Safe text <script>bad</script>"
	preset := "triglavian"
	return &api.SiteDomainConfiguration{SiteName: &name, Entities: []api.SiteDomainEntity{{Type: "corporation", ID: 42, Name: "Scoped Corp"}}, Theme: api.SiteDomainTheme{TransparentBanner: &transparent, DefaultThemePreset: &preset, DefaultThemeOverrides: map[string]string{"brandPrimary": "#b91c1c"}}, Backgrounds: []string{"/images/domains/background/27"}, Widgets: api.SiteDomainWidgets{Top: []api.SiteDomainWidget{{Type: "mostValuable", Enabled: true}}, Left: []api.SiteDomainWidget{{Type: "topCharacters", Enabled: true}, {Type: "textBlock", Enabled: true, Content: &text}, {Type: "topAlliances", Enabled: false}}, Right: []api.SiteDomainWidget{{Type: "killList", Enabled: true}}, ColumnRatio: &ratio}}
}
func TestDomainPagesKeepHostsIsolatedAndWidgetsDeferred(t *testing.T) {
	reader := &domainReaderStub{hosts: map[string]*api.SiteDomainConfiguration{"void.eve-kill.com": testDomain("VOID Killboard"), "board.example.org": testDomain("Other Board")}}
	apiCalls := 0
	handler := New(Options{Production: true, Homepage: homepageReaderStub{}, Statistics: reader, AccountAPI: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { apiCalls++ })})
	for _, host := range []string{"void.eve-kill.com", "board.example.org"} {
		for _, path := range []string{"/", "/kills/solo", "/about"} {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "https://"+host+path, nil))
			body := w.Body.String()
			if w.Code != 200 || !strings.Contains(body, domainSiteName(reader.hosts[host])) || !strings.Contains(body, "noindex") || strings.Contains(body, "site is being rebuilt") || strings.Contains(body, "Example characters") {
				t.Fatalf("host=%s path=%s status=%d", host, path, w.Code)
			}
			if w.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal("tenant pages must not enter the apex shared cache")
			}
			if path == "/" && (!strings.Contains(body, `data-domain-widget="topCharacters"`) || strings.Contains(body, `data-domain-widget="topAlliances"`) || !strings.Contains(body, `data-domain-background="/images/domains/background/27"`) || strings.Contains(body, "<script>bad</script>")) {
				t.Fatal("configured widgets or safe theme bootstrapping missing")
			}
		}
	}
	if apiCalls != 0 {
		t.Fatal("cold widgets must not delay initial HTML")
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "https://unknown.eve-kill.com/", nil))
	if w.Code != 404 || !strings.Contains(w.Body.String(), "No Killboard Found") {
		t.Fatal("unknown hosts must not receive global kills")
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "https://eve-kill.com/", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Example characters") || strings.Contains(w.Body.String(), "VOID Killboard") {
		t.Fatal("apex must preserve the original homepage")
	}
	reader.err = errors.New("private database details")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "https://void.eve-kill.com/", nil))
	if w.Code != 500 || strings.Contains(w.Body.String(), reader.err.Error()) {
		t.Fatal("configuration failures must not expose database details or global data")
	}
}
func TestDomainFragmentsForwardHostAndFilters(t *testing.T) {
	reader := &domainReaderStub{hosts: map[string]*api.SiteDomainConfiguration{"void.eve-kill.com": testDomain("VOID")}}
	var address *url.URL
	handler := New(Options{Homepage: homepageReaderStub{}, Statistics: reader, AccountAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "void.eve-kill.com" {
			t.Error("lost tenant host")
		}
		address = r.URL
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "killlist") {
			_ = json.NewEncoder(w).Encode(api.KilllistPage{Kills: []api.KilllistRow{{KillmailID: 99, ShipName: "Scoped Ship"}}})
		} else {
			_, _ = w.Write([]byte(`{"entries":[]}`))
		}
	})})
	for _, tt := range []struct{ path, endpoint, filter string }{
		{"/_fragments/domain/widgets/valuable?kind=ships", "/api/custom/stats", "most_valuable_ships"},
		{"/_fragments/domain/widgets/valuable?type=solo", "/api/custom/kills/most-valuable", "solo"},
		{"/_fragments/domain/widgets/topCharacters?type=solo", "/api/custom/kills/top", "solo"},
		{"/_fragments/domain/killlist?type=highsec&limit=100&after=123&page=2", "/api/custom/killlist", "highsec"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "https://void.eve-kill.com"+tt.path, nil))
		if w.Code != 200 || address.Path != tt.endpoint || !strings.Contains(address.RawQuery, tt.filter) {
			t.Fatalf("%s status=%d API=%v", tt.path, w.Code, address)
		}
		if strings.Contains(tt.path, "killlist") && (address.Query().Get("after") != "123" || address.Query().Get("limit") != "100") {
			t.Fatal("lost cursor or limit")
		}
	}
	for _, path := range []string{"/_fragments/domain/widgets/valuable?kind=unsafe", "/_fragments/domain/widgets/unknown", "/_fragments/domain/widgets/topCharacters?type=unsafe", "/_fragments/domain/killlist?limit=101", "/_fragments/domain/killlist?type=unsafe"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "https://void.eve-kill.com"+path, nil))
		if w.Code != 400 && w.Code != 404 {
			t.Fatalf("invalid fragment accepted: %s", path)
		}
	}
	request := httptest.NewRequest("GET", "https://void.eve-kill.com/system/30000142", nil)
	ctx := context.WithValue(request.Context(), domainPageKey{}, &domainPageContext{Config: reader.hosts[request.Host], Request: request})
	instance := &site{options: Options{AccountAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		address = r.URL
		if r.Host != request.Host {
			t.Error("lost location tenant")
		}
		_, _ = w.Write([]byte(`{"kills":[]}`))
	})}}
	cursor := int64(123)
	_, err := instance.locationKilllist(ctx, "system", 30000142, 25, &cursor)
	if err != nil || address.Path != "/api/custom/system/30000142/killlist" || address.Query().Get("after") != "123" {
		t.Fatalf("location read not scoped: %v %v", address, err)
	}
}
