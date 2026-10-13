package frontend

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCampaignDirectorySSRAndFragmentKeepSiteScope(t *testing.T) {
	var received []string
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/campaigns" || r.Host != "eve-kill.com" || r.Header.Get("Cookie") != "session=test" {
			t.Errorf("campaign API request: path=%q host=%q cookie=%q", r.URL.Path, r.Host, r.Header.Get("Cookie"))
		}
		received = append(received, r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"campaigns":[{"campaign_id":"abc123","name":"Campaign Match","mode":"area","status":1,"location":{"systemIds":[30000163]},"creator":{"name":"Pilot"},"totals":{"killCount":4,"iskDestroyed":1000000000},"sides":[]}],"hasMore":true,"page":2,"total":49,"counts":{"active":2,"archived":6,"private":1}}`))
	})
	handler := New(Options{Production: true, CampaignsAPI: apiHandler})
	request := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, canonicalOrigin+path, nil)
		r.Header.Set("Cookie", "session=test")
		handler.ServeHTTP(w, r)
		return w
	}
	filters := "?status=archived&mode=area&q=F%C3%B8%C3%B8&page=2"
	page := request("/campaigns" + filters)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Campaign Match") || !strings.Contains(page.Body.String(), "Page 2 of 3") || !strings.Contains(page.Body.String(), "data-campaign-status=\"archived\"") {
		t.Fatalf("campaign page status=%d body has card=%v", page.Code, strings.Contains(page.Body.String(), "Campaign Match"))
	}
	fragment := request("/_fragments/campaigns" + filters)
	if fragment.Code != http.StatusOK || !strings.Contains(fragment.Body.String(), "Campaign Match") || strings.Contains(fragment.Body.String(), "campaigns-header") {
		t.Fatalf("campaign fragment status=%d contains card=%v", fragment.Code, strings.Contains(fragment.Body.String(), "Campaign Match"))
	}
	for _, raw := range received {
		query, err := url.ParseQuery(raw)
		if err != nil || query.Get("status") != "archived" || query.Get("mode") != "area" || query.Get("q") != "Føø" || query.Get("page") != "2" {
			t.Fatalf("forwarded campaign query %q: %v", raw, err)
		}
	}
	if bad := request("/_fragments/campaigns?page=invalid"); bad.Code != http.StatusBadRequest {
		t.Fatalf("bad fragment status=%d", bad.Code)
	}
	if bad := request("/campaigns?status=unknown"); bad.Code != http.StatusNotFound {
		t.Fatalf("bad page status=%d", bad.Code)
	}
}

func TestCampaignDirectorySearchLimitIsRuneSafe(t *testing.T) {
	filters, err := parseCampaignDirectoryFilters(url.Values{"q": {strings.Repeat("ø", 101)}})
	if err != nil || len([]rune(filters.Query)) != 100 {
		t.Fatalf("query length=%d, error=%v", len([]rune(filters.Query)), err)
	}
}

func TestCampaignDirectoryKeepsTenantHostAndBadge(t *testing.T) {
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "tenant.example.com" {
			t.Errorf("campaign API host = %q", r.Host)
		}
		_, _ = w.Write([]byte(`{"campaigns":[],"hasMore":false,"page":1,"total":0,"counts":{"active":0,"archived":0,"private":0}}`))
	})
	handler := New(Options{Production: true, CampaignsAPI: apiHandler})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://tenant.example.com/campaigns", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "This killboard only") {
		t.Fatalf("tenant page status=%d badge=%v", w.Code, strings.Contains(w.Body.String(), "This killboard only"))
	}
}
