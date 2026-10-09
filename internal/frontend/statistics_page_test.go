package frontend

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

type statisticsTestReader struct {
	config  api.SiteConfigurationResponse
	queries []api.StatisticsQuery
	warm    map[int][]api.StatisticsEntry
	calls   int
	err     error
}

func (r *statisticsTestReader) StatisticsSite(context.Context, string) (api.SiteConfigurationResponse, error) {
	return r.config, nil
}
func (r *statisticsTestReader) StatisticsBoardsCached(_ context.Context, _ api.SiteConfigurationResponse, queries []api.StatisticsQuery) map[int][]api.StatisticsEntry {
	r.queries = queries
	return r.warm
}
func (r *statisticsTestReader) StatisticsBoard(_ context.Context, _ api.SiteConfigurationResponse, query api.StatisticsQuery) ([]api.StatisticsEntry, error) {
	r.calls++
	r.queries = []api.StatisticsQuery{query}
	return []api.StatisticsEntry{{ID: 123, Name: `Pilot <script>alert(1)</script>`, Type: "character", Count: 1234.5}}, r.err
}

func TestStatisticsDocumentDoesNotStartColdReads(t *testing.T) {
	r := &statisticsTestReader{}
	site := New(Options{Statistics: r})
	w := httptest.NewRecorder()
	site.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/stats", nil))
	if w.Code != 200 || r.calls != 0 || len(r.queries) != 34 || strings.Count(w.Body.String(), `data-stats-loaded="false"`) != 34 {
		t.Fatal("the document must render all source cards without starting any cold query")
	}
	w = httptest.NewRecorder()
	site.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/_fragments/stats/characters?period=0.25", nil))
	if w.Code != 200 || r.calls != 1 || r.queries[0].Days != 0.25 || !strings.Contains(w.Body.String(), "1,234.5") || strings.Contains(w.Body.String(), "<script>alert(1)</script>") || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("the card did not preserve the selected period, number precision, safe HTML and private cache policy")
	}
	r.err = errors.New("private SQL details")
	w = httptest.NewRecorder()
	site.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/_fragments/stats/characters", nil))
	if w.Code != 503 || strings.Contains(w.Body.String(), "private SQL") {
		t.Fatal("card failures must be isolated and use a public retry message")
	}
}

func TestStatisticsWarmCardsAndDomainScope(t *testing.T) {
	name := "Fixture board"
	r := &statisticsTestReader{config: api.SiteConfigurationResponse{IsDomainHost: true, Domain: &api.SiteDomainConfiguration{ID: 1, SiteName: &name}}, warm: map[int][]api.StatisticsEntry{0: {{ID: 123, Name: "Scoped pilot", Type: "character", Count: 9}}}}
	site := New(Options{Statistics: r})
	w := httptest.NewRecorder()
	site.ServeHTTP(w, httptest.NewRequest("GET", "http://fixture.localhost/stats?section=largest&period=90", nil))
	body := w.Body.String()
	if w.Code != 200 || len(r.queries) != 16 || !strings.Contains(body, `data-stats-filter="all"`) || !strings.Contains(body, "Scoped pilot") || !strings.Contains(body, "This killboard only") || strings.Contains(body, `data-stats-card="largest-alliance"`) || !strings.Contains(body, "noindex") {
		t.Fatal("a board must retain scoped warm tables, reject population filters and remain noindex")
	}
	w = httptest.NewRecorder()
	site.ServeHTTP(w, httptest.NewRequest("GET", "http://fixture.localhost/_fragments/stats/largest-alliance", nil))
	if w.Code != 404 || r.calls != 0 {
		t.Fatal("domain fragments must never fall through to global population rankings")
	}
	w = httptest.NewRecorder()
	site.ServeHTTP(w, httptest.NewRequest("GET", "http://fixture.localhost/_fragments/stats/unknown", nil))
	if w.Code != 404 || r.calls != 0 {
		t.Fatal("unknown cards must not reach the reader")
	}
}
