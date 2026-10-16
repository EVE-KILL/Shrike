package frontend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type homepageReaderStub struct{}

func (homepageReaderStub) SavedScan(context.Context, string, string) (json.RawMessage, bool, error) {
	return nil, false, nil
}

func (homepageReaderStub) Labels(context.Context) ([]api.LabelEntry, error) {
	return []api.LabelEntry{
		{ID: "highsec", Name: "Highsec", Category: "Space", Description: "Highsec kills", Count: 1200, ViewURL: "/kills/highsec", SearchFilters: map[string]any{"label": "highsec"}},
		{ID: "solo", Name: "Solo", Category: "Engagement", Description: "Solo kills", Count: 500, ViewURL: "/kills/solo", SearchFilters: map[string]any{"label": "solo"}},
	}, nil
}

func (homepageReaderStub) Top(_ context.Context, kind string, _, _ int) ([]api.TopEntry, error) {
	return []api.TopEntry{{ID: 42, Name: "Example " + kind, Count: 1_234, Type: kind}}, nil
}

func (homepageReaderStub) MostValuable(_ context.Context, kind string, _, _ int) ([]api.ValuableKill, error) {
	return []api.ValuableKill{{KillmailID: 11, ShipTypeID: 670, ShipName: "Valuable " + kind, TotalValue: 1_000_000}}, nil
}

func (homepageReaderStub) LatestKills(_ context.Context, _, limit int) (api.KilllistPage, error) {
	kills := make([]api.KilllistRow, limit)
	for index := range kills {
		kills[index] = api.KilllistRow{
			KillmailID: int64(index + 1), KillmailTime: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC),
			ShipTypeID: 670, ShipName: "Capsule", SolarSystemID: 30000142, SolarSystemName: "Jita",
			AttackerCount: 3,
		}
	}
	return api.KilllistPage{Kills: kills, HasMore: true, TotalPages: 10}, nil
}

func (homepageReaderStub) LatestKillsFresh(ctx context.Context, page, limit int) (api.KilllistPage, error) {
	return homepageReaderStub{}.LatestKills(ctx, page, limit)
}

func (homepageReaderStub) Killlist(ctx context.Context, _ string, page, limit int) (api.KilllistPage, error) {
	return homepageReaderStub{}.LatestKills(ctx, page, limit)
}

func (homepageReaderStub) KilllistFresh(ctx context.Context, kind string, page, limit int) (api.KilllistPage, error) {
	return homepageReaderStub{}.Killlist(ctx, kind, page, limit)
}

func (homepageReaderStub) TopKills(ctx context.Context, _, dataType string, days, limit int) ([]api.TopEntry, error) {
	return homepageReaderStub{}.Top(ctx, dataType, days, limit)
}

func (homepageReaderStub) TopKillsCached(ctx context.Context, kind, dataType string, days, limit int) ([]api.TopEntry, bool) {
	entries, _ := homepageReaderStub{}.TopKills(ctx, kind, dataType, days, limit)
	return entries, true
}

func (homepageReaderStub) MostValuableKills(ctx context.Context, kind string, days, limit int) ([]api.ValuableKill, error) {
	return homepageReaderStub{}.MostValuable(ctx, kind, days, limit)
}

func (homepageReaderStub) FilteredKills(_ context.Context, _ string, _ *int64, limit int) (api.KilllistPage, error) {
	return homepageReaderStub{}.LatestKills(context.Background(), 1, limit)
}

func TestHomepageRendersDataAndReusableFragments(t *testing.T) {
	handler := New(Options{Homepage: homepageReaderStub{}})
	request := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}

	page := request("/")
	if page.Code != http.StatusOK {
		t.Fatalf("homepage status = %d: %s", page.Code, page.Body.String())
	}
	for _, text := range []string{
		"Valuable kills", "Example characters", "Example corporations", "Example alliances",
		"Example factions", "Example ships", "Example systems", "Example regions",
		"data-kill-id=\"1\"", "https://images.evetech.net/characters/42/portrait?size=64",
		"aria-label=\"3 involved\"", "icons.svg#users",
		`id="valuable-tab-kills" aria-controls="valuable-panel" tabindex="0"`,
		`id="valuable-tab-ships" aria-controls="valuable-panel" tabindex="-1"`,
		`role="tabpanel" aria-labelledby="valuable-tab-kills"`,
		`keydown->home#keydown`,
	} {
		if !strings.Contains(page.Body.String(), text) {
			t.Errorf("homepage lacks %q", text)
		}
	}
	if count := strings.Count(page.Body.String(), "data-kill-id="); count != 50 {
		t.Errorf("homepage kill rows = %d, want 50", count)
	}
	if strings.Contains(page.Body.String(), "_nuxt") {
		t.Error("homepage still contains Nuxt assets")
	}

	valuable := request("/_fragments/most-valuable?kind=ships")
	if valuable.Code != http.StatusOK || !strings.Contains(valuable.Body.String(), "Valuable ships") ||
		strings.Contains(valuable.Body.String(), "<!doctype") {
		t.Fatalf("valuable fragment = %d: %s", valuable.Code, valuable.Body.String())
	}
	list := request("/_fragments/killlist?page=2&limit=10")
	if list.Code != http.StatusOK || strings.Count(list.Body.String(), "data-kill-id=") != 10 ||
		!strings.Contains(list.Body.String(), "data-page=\"2\"") {
		t.Fatalf("killlist fragment = %d, rows %d", list.Code, strings.Count(list.Body.String(), "data-kill-id="))
	}
	filtered := request("/_fragments/killlist?quick=1&filters=%7B%22location%22%3A%7B%22securityTypes%22%3A%5B%22highsec%22%5D%7D%7D&limit=10")
	if filtered.Code != http.StatusOK || !strings.Contains(filtered.Body.String(), "data-filtered=\"true\"") {
		t.Fatalf("filtered fragment = %d", filtered.Code)
	}
	if bad := request("/_fragments/most-valuable?kind=other"); bad.Code != http.StatusBadRequest {
		t.Errorf("invalid tab status = %d, want 400", bad.Code)
	}
	if bad := request("/_fragments/killlist?page=2&fresh=1"); bad.Code != http.StatusBadRequest {
		t.Errorf("fresh reload of a deep page = %d, want 400", bad.Code)
	}
}
