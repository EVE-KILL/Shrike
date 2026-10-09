package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

func TestSystemPagesAndFragments(t *testing.T) {
	handler := New(Options{Production: true, Universe: testUniverseReader{}, Entities: testEntityReader{}})
	for _, tc := range []struct {
		path   string
		status int
		marker string
	}{
		{"/system/30002813", 200, `data-system-id="30002813"`},
		{"/system/30002813/kills", 200, `data-kind="system"`},
		{"/system/30002813/battles", 200, `/battle/12`},
		{"/system/30002813/map", 200, `href="/map/region/10000033"`},
		{"/_fragments/system/tab/30002813/dashboard", 200, `data-system-tab="dashboard"`},
		{"/_fragments/system/kills/30002813?limit=50&page=2&after=42", 200, `data-page="2"`},
		{"/_fragments/system/kills/30002813?page=2", 400, `Invalid page`},
		{"/_fragments/system/valuable/30002813?category=ships", 200, `data-kind="ships"`},
		{"/_fragments/system/battles/30002813?page=1", 200, `/battle/12`},
		{"/system/404", 404, `Page not found`},
		{"/system/30002813/nope", 404, `Page not found`},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+tc.path, nil))
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.marker) {
			t.Errorf("%s: status %d, marker %q present = %v", tc.path, w.Code, tc.marker, strings.Contains(w.Body.String(), tc.marker))
		}
	}
}

func TestSystemActivityChartKeepsMissingHourSlot(t *testing.T) {
	base := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	chart := systemActivityChart([]api.UniverseSystemActivity{
		{Timestamp: base, ShipJumps: 20, ShipKills: 5},
		{Timestamp: base.Add(-2 * time.Hour), ShipJumps: 10, ShipKills: 2},
	})
	if len(chart.Hours) != 24 || chart.Missing != 22 || chart.Hours[22].Recorded ||
		chart.Hours[22].Time.Hour() != 11 || len(chart.Rows) != 2 ||
		chart.Rows[0].Total != 30 || chart.Rows[0].Peak != 20 {
		t.Fatalf("unexpected activity chart: %+v", chart)
	}
}
