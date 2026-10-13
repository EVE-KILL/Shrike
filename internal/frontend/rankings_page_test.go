package frontend

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

type testRankingsReader struct{ entity, window string }

func (r *testRankingsReader) Rankings(_ context.Context, entity, window string) ([]api.RankingEntry, error) {
	r.entity, r.window = entity, window
	return []api.RankingEntry{
		{EntityID: 5, Name: "First", OverallRank: 1, EveKillRating: 1000},
		{EntityID: 6, Name: "Second", OverallRank: 2, EveKillRating: 900},
		{EntityID: 7, Name: "Third", OverallRank: 3, EveKillRating: 800},
		{EntityID: 8, Name: "Fourth", OverallRank: 4, EveKillRating: 700},
	}, nil
}

func TestRankingsPageSelectionAndLinks(t *testing.T) {
	reader := &testRankingsReader{}
	handler := New(Options{Rankings: reader})
	for _, tt := range []struct{ path, entity, window, link string }{
		{"/rankings", "character", "all_time", "/character/5"},
		{"/rankings?entity=ship&window=weekly", "ship", "weekly", "/item/5"},
		{"/_fragments/rankings?entity=region&window=hourly", "region", "hourly", "/region/5"},
		{"/rankings?entity=invalid&window=invalid", "character", "all_time", "/character/5"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", tt.path, nil))
		if w.Code != 200 || reader.entity != tt.entity || reader.window != tt.window || !strings.Contains(w.Body.String(), tt.link) ||
			!strings.Contains(w.Body.String(), "Fourth") {
			t.Errorf("%s: status=%d selection=%s/%s link=%v", tt.path, w.Code, reader.entity, reader.window, strings.Contains(w.Body.String(), tt.link))
		}
	}
}
