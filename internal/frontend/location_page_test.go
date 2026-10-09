package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegionAndConstellationRoutes(t *testing.T) {
	handler := New(Options{Production: true, Universe: testUniverseReader{}, Entities: testEntityReader{}})
	for _, tc := range []struct {
		path   string
		status int
		marker string
	}{
		{"/region/10000042", 200, `Metropolis`},
		{"/region/10000042/kills", 200, `data-kind="region"`},
		{"/region/10000042/battles", 200, `/battle/12`},
		{"/region/10000042/map", 200, `href="/map/region/10000042"`},
		{"/constellation/20000396", 200, `Agiesseson`},
		{"/constellation/20000396/kills", 200, `data-kind="constellation"`},
		{"/constellation/20000396/battles", 200, `/battle/12`},
		{"/constellation/20000396/map", 200, `href="/map/region/10000032"`},
		{"/_fragments/location/region/tab/10000042/dashboard", 200, `data-system-tab="dashboard"`},
		{"/_fragments/location/constellation/kills/20000396?limit=50&page=2&after=42", 200, `data-page="2"`},
		{"/_fragments/location/region/kills/10000042?page=2", 400, `Invalid page`},
		{"/_fragments/location/region/valuable/10000042?category=ships", 200, `data-kind="ships"`},
		{"/_fragments/location/constellation/battles/20000396?page=1", 200, `/battle/12`},
		{"/region/404", 404, `Page not found`},
		{"/constellation/404", 404, `Page not found`},
		{"/region/10000042/nope", 404, `Page not found`},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+tc.path, nil))
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.marker) {
			t.Errorf("%s: status %d, marker %q present = %v", tc.path, w.Code, tc.marker, strings.Contains(w.Body.String(), tc.marker))
		}
	}
}
