package frontend

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestMapPageRendersURLSelectedControls(t *testing.T) {
	handler := New(Options{Production: true})
	request := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+path, nil))
		return w
	}
	base := request("/map")
	for _, marker := range []string{`data-controller="map-page"`, "popstate@window->map-page#back", "pagehide@window->map-page#hide", "pageshow@window->map-page#show", "error->map-page#imageError:capture"} {
		if !strings.Contains(base.Body.String(), marker) {
			t.Errorf("map lifecycle missing %q", marker)
		}
	}
	if base.Code != http.StatusOK || !strings.Contains(base.Body.String(), `data-map-base="geography"`) || !strings.Contains(base.Body.String(), `data-map-canvas`) || !strings.Contains(base.Body.String(), `name="description"`) {
		t.Fatalf("base map status=%d missing map shell", base.Code)
	}
	filtered := request("/map?scope=wormhole&base=security&activity=danger&hours=168&routes=0&systems=0&labels=0")
	for _, marker := range []string{`data-map-scope="wormhole"`, `data-map-base="security"`, `data-map-activity="danger"`, `data-map-hours="168"`, `data-map-toggle="routes"`, `aria-pressed="false"`} {
		if !strings.Contains(filtered.Body.String(), marker) {
			t.Errorf("filtered map missing %q", marker)
		}
	}
	if filtered.Code != http.StatusOK || !regexp.MustCompile(`data-map-base-option="sovereignty"[^>]* hidden`).MatchString(filtered.Body.String()) {
		t.Fatalf("filtered map status=%d special-base scoping missing", filtered.Code)
	}
	legacy := request("/map?layer=security")
	if legacy.Code != http.StatusOK || !strings.Contains(legacy.Body.String(), `data-map-base="security"`) {
		t.Fatalf("legacy layer status=%d compatibility missing", legacy.Code)
	}
	unsupported := request("/map?scope=wormhole&base=sovereignty")
	if unsupported.Code != http.StatusOK || !strings.Contains(unsupported.Body.String(), `data-map-base="geography"`) {
		t.Fatalf("non-New Eden special base status=%d not normalized", unsupported.Code)
	}
}

func TestRegionMapSSRForwardsScopeAndEmbedsData(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "eve-kill.com" || r.Header.Get("Cookie") != "session=test" {
			t.Errorf("region map API scope host=%q cookie=%q", r.Host, r.Header.Get("Cookie"))
		}
		if r.URL.Path != "/api/map/region/10000002" || r.URL.Query().Get("hours") != "168" {
			t.Errorf("region map API route %q", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"region":{"region_id":10000002,"name":"The Forge"},"systems":[{"solar_system_id":30000142,"system_name":"Jita"}],"jumps":[],"constellations":[],"celestials":[],"externalJumps":[],"activity":[]}`))
	})
	handler := New(Options{Production: true, ConflictsAPI: api})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, canonicalOrigin+"/map/region/10000002?base=security&activity=ship-kills&hours=168", nil)
	r.Header.Set("Cookie", "session=test")
	handler.ServeHTTP(w, r)
	for _, marker := range []string{`The Forge Map`, `data-region-base="security"`, `data-region-activity="ship-kills"`, `data-region-hours="168"`, `id="region-map-data"`, `Jita`, `/region/10000002`} {
		if !strings.Contains(w.Body.String(), marker) {
			t.Errorf("region map missing %q", marker)
		}
	}
	if w.Code != http.StatusOK {
		t.Fatalf("region map status=%d", w.Code)
	}
	bad := httptest.NewRecorder()
	handler.ServeHTTP(bad, httptest.NewRequest(http.MethodGet, canonicalOrigin+"/map/region/not-a-number", nil))
	if bad.Code != http.StatusNotFound {
		t.Errorf("invalid region status=%d", bad.Code)
	}
}
