package frontend

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type testLegacyReader struct{}

func (testLegacyReader) LegacyArchiveStats(context.Context) (api.LegacyArchiveStats, error) {
	return api.LegacyArchiveStats{Killmails: 436063}, nil
}
func (testLegacyReader) LegacyArchiveTop(_ context.Context, kind string, _ int) ([]api.TopEntry, error) {
	return []api.TopEntry{{ID: 1, Name: kind, Count: 10}}, nil
}
func (testLegacyReader) LegacyArchiveKills(_ context.Context, _ url.Values) (api.KilllistPage, error) {
	id := int64(123)
	return api.KilllistPage{Kills: []api.KilllistRow{{
		KillmailID: id, KillmailTime: time.Date(2007, 7, 8, 5, 44, 0, 0, time.UTC),
		ShipName: "Apocalypse", TotalValue: 93936840,
	}}, HasMore: true, Cursor: &id}, nil
}
func (testLegacyReader) LegacyArchiveKill(_ context.Context, id int64) (api.LegacyArchiveKillData, bool, error) {
	if id != 123 {
		return api.LegacyArchiveKillData{}, false, nil
	}
	return api.LegacyArchiveKillData{Kill: api.LegacyArchiveKill{
		KillmailID: id, KillmailTime: time.Date(2007, 7, 8, 5, 44, 0, 0, time.UTC),
		VictimShip: "Apocalypse", VictimName: "Pilot", TotalValue: 93936840,
	}}, true, nil
}

func TestLegacyArchiveListAndDetailRoutes(t *testing.T) {
	handler := New(Options{Legacy: testLegacyReader{}})
	for _, tt := range []struct {
		path string
		code int
		want string
	}{
		{"/legacy", 200, "436,063"},
		{"/_fragments/legacy/kills?sort=id_asc", 200, "/legacy/kill/123"},
		{"/_fragments/legacy/top?year=2003", 200, "Top Victims"},
		{"/legacy/kill/123", 200, "View original killmail text"},
		{"/legacy/kill/999", 404, "Page not found"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", tt.path, nil))
		if w.Code != tt.code || !strings.Contains(w.Body.String(), tt.want) {
			t.Errorf("%s: status=%d want=%d marker=%q", tt.path, w.Code, tt.code, tt.want)
		}
	}
}
