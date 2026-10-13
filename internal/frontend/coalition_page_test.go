package frontend

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

type testCoalitionReader struct{}

func (testCoalitionReader) CoalitionDirectoryOverview(context.Context) (api.CoalitionDirectoryData, error) {
	return api.CoalitionDirectoryData{
		StatsWindowDays: 30,
		Coalitions:      []api.CoalitionSummary{{CoalitionID: 1, Slug: "example-coalition", Name: "Example Coalition", AllianceCount: 1, MemberCount: 1234}},
	}, nil
}
func (reader testCoalitionReader) CoalitionDirectory(ctx context.Context) (api.CoalitionDirectoryData, error) {
	data, err := reader.CoalitionDirectoryOverview(ctx)
	data.Coalitions[0].Kills = 1234567
	data.Coalitions[0].Losses = 11
	data.Coalitions[0].ISKDestroyed = 1000000000
	return data, err
}
func (testCoalitionReader) CoalitionDetail(_ context.Context, slug string) (api.CoalitionDetailData, bool, error) {
	if slug != "example-coalition" {
		return api.CoalitionDetailData{}, false, nil
	}
	return api.CoalitionDetailData{
		StatsWindowDays: 30,
		Coalition:       api.CoalitionSummary{CoalitionID: 1, Slug: slug, Name: "Example Coalition", AllianceCount: 1, MemberCount: 1234, Revision: 2},
		Alliances:       []api.CoalitionAlliance{{AllianceID: 99000001, Name: "Test Alliance", Ticker: "TEST", MemberCount: 1234}},
		Edits:           []api.CoalitionEdit{{EditID: 1, EditorCharacterName: "EVE-KILL", Action: "seed", Summary: "Initial import"}},
	}, true, nil
}

func TestCoalitionDirectoryAndDetailRoutes(t *testing.T) {
	handler := New(Options{Coalitions: testCoalitionReader{}})
	for _, tt := range []struct {
		path    string
		status  int
		markers []string
	}{
		{"/coalitions", 200, []string{"Example Coalition", "1,234", "data-coalition-filter", "data-coalition-form=\"create\""}},
		{"/coalitions/example-coalition", 200, []string{"Example Coalition", "/alliance/99000001", "data-coalition-revision=\"2\"", "data-coalition-form=\"edit\""}},
		{"/_fragments/coalitions/example-coalition", 200, []string{"Test Alliance", "Initial import"}},
		{"/coalitions/unknown", 404, []string{"Page not found"}},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", tt.path, nil))
		if w.Code != tt.status {
			t.Errorf("%s status=%d want=%d", tt.path, w.Code, tt.status)
		}
		for _, marker := range tt.markers {
			if !strings.Contains(w.Body.String(), marker) {
				t.Errorf("%s missing %q", tt.path, marker)
			}
		}
	}
}

func TestCoalitionSourceRejectsUnsafeURLs(t *testing.T) {
	unsafe := "javascript:alert(1)"
	if got := coalitionSafeURL(&unsafe); got != "" {
		t.Fatalf("unsafe source URL: %q", got)
	}
}

type failedCoalitionActivityReader struct{ testCoalitionReader }

func (failedCoalitionActivityReader) CoalitionDirectory(context.Context) (api.CoalitionDirectoryData, error) {
	return api.CoalitionDirectoryData{}, errors.New("slow activity database unavailable")
}

func TestCoalitionDirectoryDoesNotWaitForActivity(t *testing.T) {
	handler := New(Options{Coalitions: failedCoalitionActivityReader{}})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest("GET", "/coalitions", nil))
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Example Coalition") || !strings.Contains(page.Body.String(), "Loading combat statistics") || !strings.Contains(page.Body.String(), "data-coalition-activity=\"1\"") {
		t.Errorf("directory without activity: status=%d", page.Code)
	}
	fragment := httptest.NewRecorder()
	handler.ServeHTTP(fragment, httptest.NewRequest("GET", "/_fragments/coalitions/activity", nil))
	if fragment.Code != 503 {
		t.Errorf("failed activity status=%d", fragment.Code)
	}
}

func TestCoalitionActivityFragmentContainsOnlyTotals(t *testing.T) {
	handler := New(Options{Coalitions: testCoalitionReader{}})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest("GET", "/coalitions", nil))
	if strings.Contains(page.Body.String(), "1,234,567") {
		t.Fatal("page computed combat totals during SSR")
	}
	fragment := httptest.NewRecorder()
	handler.ServeHTTP(fragment, httptest.NewRequest("GET", "/_fragments/coalitions/activity", nil))
	if fragment.Code != 200 || !strings.Contains(fragment.Body.String(), "1,234,567") || strings.Contains(fragment.Body.String(), "coalition-card-intro") {
		t.Fatalf("activity fragment: status=%d body=%s", fragment.Code, fragment.Body.String())
	}
}
