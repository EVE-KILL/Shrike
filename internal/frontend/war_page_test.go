package frontend

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type testWarReader struct {
	query, killQuery, memberQuery url.Values
	host                          string
}

type blockedFactionReader struct {
	*testWarReader
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
	fail    bool
}

func (r *blockedFactionReader) FactionWars(ctx context.Context) (api.FactionWarsData, error) {
	r.calls.Add(1)
	if r.started != nil {
		r.started <- struct{}{}
	}
	if r.release != nil {
		select {
		case <-r.release:
		case <-ctx.Done():
			return api.FactionWarsData{}, ctx.Err()
		}
	}
	if r.fail {
		return api.FactionWarsData{}, errors.New("database unavailable")
	}
	return r.testWarReader.FactionWars(ctx)
}

func TestWarsDocumentDoesNotWaitForFactionAggregates(t *testing.T) {
	reader := &blockedFactionReader{testWarReader: &testWarReader{}, started: make(chan struct{}, 1), release: make(chan struct{})}
	handler := New(Options{Wars: reader})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer close(reader.release)
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/wars", nil).WithContext(ctx))
		done <- w
	}()
	select {
	case w := <-done:
		if w.Code != 200 || !strings.Contains(w.Body.String(), "/war/42") || !strings.Contains(w.Body.String(), `data-wars-faction-content aria-busy="true"`) {
			t.Fatalf("initial wars response status=%d", w.Code)
		}
		if reader.calls.Load() != 0 {
			t.Fatal("document invoked the deferred faction query")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("war document waited for faction aggregates")
	}
	fragment := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/_fragments/wars/faction", nil).WithContext(ctx))
		fragment <- w
	}()
	select {
	case <-reader.started:
	case <-ctx.Done():
		t.Fatal("fragment did not load faction data")
	}
	// The query is still waiting, while the document and its list have completed.
	reader.release <- struct{}{}
	select {
	case w := <-fragment:
		body := w.Body.String()
		if w.Code != 200 || !strings.Contains(body, "10 kills") || !strings.Contains(body, "20 kills") || !strings.Contains(body, `data-wars-faction-loaded`) || strings.Contains(body, "wars-header") {
			t.Fatalf("faction fragment status=%d", w.Code)
		}
		if w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("fragment cache policy missing")
		}
	case <-ctx.Done():
		t.Fatal("fragment did not complete")
	}
}

func TestWarsFactionFailureAndTenantIsolation(t *testing.T) {
	reader := &blockedFactionReader{testWarReader: &testWarReader{}, fail: true}
	handler := New(Options{Wars: reader})
	get := func(url string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		return w
	}
	if w := get("http://localhost/wars"); w.Code != 200 || reader.calls.Load() != 0 {
		t.Fatalf("faction failure affected directory: %d", w.Code)
	}
	if w := get("http://localhost/_fragments/wars/faction"); w.Code != http.StatusServiceUnavailable {
		t.Errorf("failed fragment status=%d", w.Code)
	}
	reader.fail = false
	if w := get("http://localhost/_fragments/wars/faction"); w.Code != 200 {
		t.Errorf("retry status=%d", w.Code)
	}
	before := reader.calls.Load()
	if w := get("http://custom.localhost/wars"); w.Code != 200 || strings.Contains(w.Body.String(), "data-wars-faction-content") {
		t.Errorf("tenant directory status=%d", w.Code)
	}
	if w := get("http://custom.localhost/_fragments/wars/faction"); w.Code != 404 || reader.calls.Load() != before {
		t.Fatal("tenant queried global faction data")
	}
}

func (r *testWarReader) WarDirectory(_ context.Context, query url.Values, host string) (api.WarDirectoryData, error) {
	r.query, r.host = query, host
	return api.WarDirectoryData{Page: 1, Limit: 50, Wars: []api.EntityWarEntry{{
		WarID: 42, Aggressor: api.EntityWarSide{ID: 99000001, Type: "alliance", Name: "Aggressor"},
		Defender: api.EntityWarSide{ID: 98000001, Type: "corporation", Name: "Defender"},
	}}}, nil
}
func (*testWarReader) WarOverview(context.Context) (api.WarOverviewData, error) {
	return api.WarOverviewData{ActiveWars: 15, FinishedWars: 200, EligibleCorps: 100}, nil
}
func (*testWarReader) FactionWars(context.Context) (api.FactionWarsData, error) {
	var data api.FactionWarsData
	data.CaldariVsGallente.Caldari.Kills = 10
	data.CaldariVsGallente.Gallente.Kills = 20
	return data, nil
}
func (*testWarReader) WarDetail(_ context.Context, id int64) (api.WarDetailData, bool, error) {
	if id != 42 {
		return api.WarDetailData{}, false, nil
	}
	return api.WarDetailData{War: api.WarDetail{WarID: 42, Started: "2026-09-23T18:57:00Z",
		Aggressor: api.EntityWarSide{ID: 99000001, Type: "alliance", Name: "Aggressor"},
		Defender:  api.EntityWarSide{ID: 98000001, Type: "corporation", Name: "Defender"}},
		Stats: api.WarDetailStats{TotalKills: 1, TotalValue: 1_000_000_000}}, true, nil
}
func (r *testWarReader) WarKilllist(_ context.Context, _ int64, query url.Values) (api.KilllistPage, error) {
	r.killQuery = query
	return api.KilllistPage{Kills: []api.KilllistRow{{KillmailID: 123, ShipName: "Raitaru", ShipTypeID: 35825,
		KillmailTime: time.Date(2026, 9, 23, 19, 22, 0, 0, time.UTC), VictimCorporationID: 98000001, VictimCorporationName: "Defender"}}, TotalPages: 1}, nil
}
func (*testWarReader) WarStats(context.Context, int64) (api.WarStatsData, error) {
	return api.WarStatsData{Combined: api.WarLeaderboard{TopCorporations: []api.WarLeaderboardEntry{{ID: 98000001, Name: "Defender", Kills: 1}}}}, nil
}
func (r *testWarReader) WarMembers(_ context.Context, _ int64, query url.Values) (api.WarMembersData, error) {
	r.memberQuery = query
	return api.WarMembersData{Members: []api.WarMember{{CharacterID: 12345, CharacterName: "Pilot", Side: "aggressor"}}}, nil
}
func (*testWarReader) WarIntel(context.Context, int64) (api.WarIntelData, error) {
	var data api.WarIntelData
	data.Summary.Kills = 1
	data.TopSystems = []api.WarIntelSystem{{SystemID: 30003014, SystemName: "Bereye", Kills: 1}}
	data.SecurityBreakdown = []api.WarIntelSecurity{{SecClass: "highsec", Kills: 1}}
	return data, nil
}

func TestWarsSSRAndSelections(t *testing.T) {
	reader := &testWarReader{}
	handler := New(Options{Wars: reader})
	for _, tt := range []struct{ path, flag, value, marker string }{
		{"/wars", "ongoing", "true", "/war/42"},
		{"/wars?tab=finished&sort=isk&mutual=true&hasKills=true&page=2", "finished", "true", "data-page=\"2\""},
		{"/wars?tab=upcoming&hasAllies=true", "upcoming", "true", "Starts"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", tt.path, nil))
		if response.Code != 200 || reader.query.Get(tt.flag) != tt.value || !strings.Contains(response.Body.String(), tt.marker) {
			t.Errorf("%s: status %d, query %v, missing %q", tt.path, response.Code, reader.query, tt.marker)
		}
		for _, marker := range []string{"Active Wars", "Faction Warfare", "data-wars-filter", "data-wars-sort", "wars.js"} {
			if !strings.Contains(response.Body.String(), marker) {
				t.Errorf("%s missing %q", tt.path, marker)
			}
		}
	}
	if reader.query.Get("sort") != "" || reader.query.Get("hasAllies") != "" {
		t.Fatalf("upcoming tab should ignore filters: %v", reader.query)
	}
}

func TestWarEligibleTabAndDomainScope(t *testing.T) {
	reader := &testWarReader{}
	handler := New(Options{Wars: reader})
	request := httptest.NewRequest("GET", "http://test.localhost/wars?tab=eligible-alliances", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || !strings.Contains(response.Body.String(), "Loading war-eligible alliances") || strings.Contains(response.Body.String(), "class=\"wars-faction\"") {
		t.Fatalf("tenant eligible page: status %d", response.Code)
	}
	if reader.query != nil {
		t.Fatal("eligible tabs must load from the dedicated client API")
	}
	request = httptest.NewRequest("GET", "http://test.localhost/wars", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || reader.host != "test.localhost" {
		t.Fatalf("tenant war request lost host: %q, status %d", reader.host, response.Code)
	}
}

func TestWarDetailAndFragments(t *testing.T) {
	reader := &testWarReader{}
	handler := New(Options{Wars: reader})
	for _, test := range []struct {
		path    string
		status  int
		markers []string
	}{
		{"/war/42", 200, []string{"War #42", "Aggressor", "Defender", "/kill/123", "war-victim-defender", "data-war-tab=\"members\""}},
		{"/_fragments/war/42/stats?side=combined", 200, []string{"Top Corporations", "Defender"}},
		{"/_fragments/war/42/members?side=aggressor&sort=kills&allianceId=99000001", 200, []string{"Pilot", "/character/12345"}},
		{"/_fragments/war/42/intel", 200, []string{"Bereye", "Top Systems", "Fighting by Security"}},
		{"/_fragments/war/42/kills?side=aggressor&page=2&limit=25", 200, []string{"/kill/123", "data-page=\"2\""}},
		{"/war/0", 404, []string{"Page not found"}},
		{"/war/43", 404, []string{"Page not found"}},
		{"/_fragments/war/42/members?side=wrong&sort=kills", 400, []string{"Invalid side"}},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", test.path, nil))
		if response.Code != test.status {
			t.Errorf("%s: status %d, want %d", test.path, response.Code, test.status)
		}
		for _, marker := range test.markers {
			if !strings.Contains(response.Body.String(), marker) {
				t.Errorf("%s missing %q", test.path, marker)
			}
		}
	}
	if reader.killQuery.Get("warSideCorps") != "98000001" || reader.killQuery.Get("page") != "2" || reader.killQuery.Get("warStart") == "" {
		t.Errorf("aggressor killlist must select defender victims and date bound: %v", reader.killQuery)
	}
	if reader.memberQuery.Get("allianceId") != "99000001" || reader.memberQuery.Get("sort") != "kills" {
		t.Errorf("member filters lost: %v", reader.memberQuery)
	}
}
