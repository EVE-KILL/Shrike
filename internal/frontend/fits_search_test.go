package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func TestFittingSearchImmediateControlsAndScopedFragments(t *testing.T) {
	var mu sync.Mutex
	calls := map[string]url.Values{}
	handler := New(Options{FittingsAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Host != "eve-kill.com" || r.Header.Get("Cookie") != "session=fixture" {
			t.Error("fitting API request identity was lost")
		}
		mu.Lock()
		calls[r.URL.Path] = r.URL.Query()
		mu.Unlock()
		switch r.URL.Path {
		case "/api/fits/roles":
			_, _ = w.Write([]byte(`{"roles":[{"id":"autocannon","label":"Autocannon","icon":"lucide:rotate-cw","category":"weapon","typeCount":10}]}`))
		case "/api/fits/popular-ships":
			_, _ = w.Write([]byte(`{"window_days":30,"ships":[{"ship_type_id":587,"ship_name":"Rifter","total_uses":3000,"fit_count":15}]}`))
		case "/api/fits/search":
			_, _ = w.Write([]byte(`{"ship_type_id":587,"ship_name":"Rifter","total":48,"offset":24,"has_more":false,"fits":[{"fit_hash":"fixture-exact","family_hash":"fixture-family","ship_type_id":587,"ship_name":"Rifter","total_uses":3,"family_total_uses":7,"variant_count":2,"last_used":"2026-09-29T00:00:00Z","fit_cost":2000000,"hull_cost":1000000,"modules":[{"slot_group":1,"ordinal":4,"type_id":484,"name":"125mm Gatling AutoCannon I","charge_type_id":178,"charge_name":"Carbonized Lead S"}],"drones":[{"type_id":2454,"name":"<script>unsafe()</script>","quantity":7}],"stats":{"dps_with_reload":1200,"ehp":50123,"max_velocity":1900,"shield_effective_boost":12,"armor_effective_repair":20,"npc_ehp":55000,"npc_profile":"guristas","cap_stable":true}}]}`))
		case "/api/fits/search/distributions":
			_, _ = w.Write([]byte(`{"ship_type_id":587,"window_days":90,"metrics":[{"metric":"dps","median":200,"p10":100,"p90":300,"buckets":[{"bucket":0,"lower_bound":100,"upper_bound":200,"observation_count":1},{"bucket":1,"lower_bound":200,"upper_bound":300,"observation_count":100}]}]}`))
		default:
			http.NotFound(w, r)
		}
	})})
	request := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", canonicalOrigin+path, nil)
		r.Header.Set("Cookie", "session=fixture")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	q := url.Values{"ship": {"587"}, "ship_name": {"Rifter"}, "sort": {"dps"}, "min_dps": {"100"}, "max_dps": {"200"}, "npc_profile": {"guristas"}, "cap_stable": {"true"}, "filters": {`[{"role_id":"autocannon","op":">=","count":2}]`}}
	page := request("/fits/search?" + q.Encode())
	if page.Code != 200 || len(calls) != 2 || !strings.Contains(page.Body.String(), "Search Criteria") || !strings.Contains(page.Body.String(), "Autocannon") || !strings.Contains(page.Body.String(), "Loading fits") || strings.Contains(page.Body.String(), "wasm-unsafe-eval") {
		t.Fatal("initial search blocked on heavy results or lost its controls")
	}
	mu.Lock()
	clear(calls)
	mu.Unlock()
	q.Set("offset", "24")
	results := request("/_fragments/fits/search/results?" + q.Encode())
	if results.Code != 200 || len(calls) != 1 || results.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("results did not use one scoped API read")
	}
	apiQuery := calls["/api/fits/search"]
	for key, value := range map[string]string{"ship": "587", "sort": "dps", "min_dps": "100", "max_dps": "200", "npc_profile": "guristas", "cap_stable": "true", "offset": "24", "limit": "24"} {
		if apiQuery.Get(key) != value {
			t.Errorf("%s not preserved", key)
		}
	}
	if apiQuery.Get("ship_name") != "" {
		t.Fatal("display-only hull name leaked into API scope")
	}
	body := results.Body.String()
	for _, value := range []string{`data-fit-search-hash="fixture-exact"`, "25", "3 exact uses", "7 family uses", "1200 DPS", "50.1K EHP", "20 EHP/s", "1,900 m/s", "55K EHP vs Guristas Pirates", "Cap stable", "3.00m ISK", "7×", "Carbonized Lead S", "&lt;script&gt;", "/fits/create?fit=v3"} {
		if !strings.Contains(body, value) {
			t.Errorf("missing %s", value)
		}
	}
	if strings.Contains(body, "<script>unsafe()") {
		t.Fatal("unescaped module/drone data")
	}
	mu.Lock()
	clear(calls)
	mu.Unlock()
	profile := request("/_fragments/fits/search/distributions?" + q.Encode())
	if profile.Code != 200 || len(calls) != 1 || calls["/api/fits/search/distributions"].Get("sort") != "" || !strings.Contains(profile.Body.String(), "height:4.000000%") || !strings.Contains(profile.Body.String(), `data-fit-search-bucket-min="100"`) {
		t.Fatal("histogram scope, scaling or range controls changed")
	}
}

func TestFittingSearchURLStateValidationAndFailures(t *testing.T) {
	q := url.Values{"ship": {"587"}, "sort": {"unsupported"}, "filters": {`[{"role_id":"autocannon","op":"=","count":0},{"type_id":2048,"type_name":"Damage Control II","op":"<=","count":2},{"role_id":"missing-count","op":"="},{"role_id":"bad-count","op":"=","count":17}]`}, "min_align": {".5"}, "max_ehp": {"NaN"}, "npc_profile": {"angels"}}
	state := fitSearchInitialState(q)
	if state.Sort != "uses" || state.Ship.Name != "Type 587" || len(state.Filters) != 2 || len(state.Statistics) != 2 || state.Statistics[0].Metric != "align" || *state.Statistics[0].Min != .5 {
		t.Fatal("URL hydration changed filter bounds or type/role wire contract")
	}
	var roundTrip []fitRequirement
	if json.Unmarshal([]byte(state.query(true).Get("filters")), &roundTrip) != nil || roundTrip[1].Kind != "type" || roundTrip[0].Count != 0 {
		t.Fatal("URL filter conversion changed")
	}
	handler := New(Options{FittingsAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "secret query failure", http.StatusServiceUnavailable)
	})})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest("GET", "/fits/search", nil))
	if page.Code != 200 || !strings.Contains(page.Body.String(), "data-fit-search-retry-popular") || strings.Contains(page.Body.String(), "secret query") {
		t.Fatal("metadata failure broke the search shell")
	}
	fragment := httptest.NewRecorder()
	handler.ServeHTTP(fragment, httptest.NewRequest("GET", "/_fragments/fits/search/results?ship=587", nil))
	if fragment.Code != 503 || strings.Contains(fragment.Body.String(), "secret query") {
		t.Fatal("fragment failure did not remain recoverable")
	}
}

func TestFittingSearchAndLandingAppearOnlyWhenAvailableInSitemap(t *testing.T) {
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) })
	for _, test := range []struct {
		enabled bool
		want    bool
	}{{false, false}, {true, true}} {
		options := Options{Production: true}
		if test.enabled {
			options.FittingsAPI = apiHandler
		}
		response := httptest.NewRecorder()
		New(options).ServeHTTP(response, httptest.NewRequest("GET", canonicalOrigin+"/sitemap.xml", nil))
		body := response.Body.String()
		if response.Code != 200 || strings.Contains(body, "<loc>"+canonicalOrigin+"/fits</loc>") != test.want || strings.Contains(body, "<loc>"+canonicalOrigin+"/fits/search</loc>") != test.want || strings.Contains(body, "<loc>"+canonicalOrigin+"/fits/create</loc>") {
			t.Fatal("fitting sitemap does not match published indexable routes")
		}
	}
}
