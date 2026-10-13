package frontend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/killtype"
)

type coldTopReaderStub struct{ homepageReaderStub }

func (coldTopReaderStub) TopKillsCached(context.Context, string, string, int, int) ([]api.TopEntry, bool) {
	return nil, false
}

func TestKillsPagesShareValidatedTypesAndWidgets(t *testing.T) {
	handler := New(Options{Homepage: homepageReaderStub{}})
	for _, kind := range killtype.Types {
		if killsPageTitle(kind) == "Kills" {
			t.Errorf("%s lacks a descriptive title", kind)
		}
		request := httptest.NewRequest(http.MethodGet, "/kills/"+kind, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("%s returned %d", kind, response.Code)
			continue
		}
		body := response.Body.String()
		for _, want := range []string{
			"data-kind=\"" + kind + "\"", "data-kills-page", "id=\"kill-list\"",
			"id=\"top-panel\"", "id=\"valuable-panel\"",
			"/kills/" + kind,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%s lacks %q", kind, want)
			}
		}
	}
	for _, path := range []string{"/kills/unknown", "/kills/top", "/kills/most-valuable"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Errorf("%s returned %d, want 404", path, response.Code)
		}
	}
}

func TestKillsPageFragmentsKeepTheirTypeAndMobileMode(t *testing.T) {
	handler := New(Options{Homepage: homepageReaderStub{}})
	request := func(path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}
	fragment := request("/_fragments/killlist?type=big&page=2&limit=25")
	if fragment.Code != http.StatusOK ||
		!strings.Contains(fragment.Body.String(), "data-kind=\"big\"") ||
		!strings.Contains(fragment.Body.String(), "data-page=\"2\"") ||
		strings.Contains(fragment.Body.String(), "kill-filter-panel") {
		t.Fatalf("typed fragment = %d, body=%s", fragment.Code, fragment.Body.String())
	}
	if bad := request("/_fragments/killlist?type=not-a-type"); bad.Code != http.StatusBadRequest {
		t.Errorf("unknown type fragment = %d, want 400", bad.Code)
	}
	if bad := request("/_fragments/killlist?type=big&quick=1&filters=%7B%7D"); bad.Code != http.StatusBadRequest {
		t.Errorf("filters on typed list = %d, want 400", bad.Code)
	}
}

func TestColdKillsPageDefersOnlyTopBoxes(t *testing.T) {
	handler := New(Options{Homepage: coldTopReaderStub{}})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/kills/big", nil))
	if page.Code != http.StatusOK || strings.Count(page.Body.String(), "data-top-pending=") != 3 ||
		!strings.Contains(page.Body.String(), "data-kill-id=") ||
		!strings.Contains(page.Body.String(), "valuable-card") {
		t.Fatalf("cold page status=%d, pending=%d", page.Code, strings.Count(page.Body.String(), "data-top-pending="))
	}
	fragment := httptest.NewRecorder()
	handler.ServeHTTP(fragment, httptest.NewRequest(http.MethodGet, "/_fragments/kills/top?type=big&dataType=corporations", nil))
	if fragment.Code != http.StatusOK || !strings.Contains(fragment.Body.String(), "Example corporations") ||
		strings.Contains(fragment.Body.String(), "<!DOCTYPE") {
		t.Fatalf("top fragment status=%d", fragment.Code)
	}
}

func TestKillsPageDoesNotShowGlobalDataOnTenantHost(t *testing.T) {
	handler := New(Options{Production: true, Homepage: homepageReaderStub{}})
	request := httptest.NewRequest(http.MethodGet, "https://board.example/kills/solo", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "data-kill-id=") ||
		strings.Contains(response.Header().Get("Cache-Control"), "public") {
		t.Fatalf("tenant placeholder status=%d cache=%s", response.Code, response.Header().Get("Cache-Control"))
	}
}

func TestKillsPagesAreInProductionSitemap(t *testing.T) {
	handler := New(Options{Production: true, Homepage: homepageReaderStub{}})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "https://eve-kill.com/kills/solo", nil))
	if page.Code != http.StatusOK || page.Header().Get("X-Robots-Tag") != "index, follow" ||
		!strings.Contains(page.Body.String(), `href="https://eve-kill.com/kills/solo"`) {
		t.Fatalf("solo page status=%d robots=%s", page.Code, page.Header().Get("X-Robots-Tag"))
	}
	sitemap := httptest.NewRecorder()
	handler.ServeHTTP(sitemap, httptest.NewRequest(http.MethodGet, "https://eve-kill.com/sitemap.xml", nil))
	if sitemap.Code != http.StatusOK || strings.Count(sitemap.Body.String(), "/kills/") != len(killtype.Types) {
		t.Fatalf("sitemap status=%d category entries=%d, want %d", sitemap.Code,
			strings.Count(sitemap.Body.String(), "/kills/"), len(killtype.Types))
	}
}
