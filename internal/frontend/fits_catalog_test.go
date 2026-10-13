package frontend

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestFittingCatalogueDistinguishesUnknownEstimatesFromZero(t *testing.T) {
	view := fitCatalogView{Stats: fitCatalogStats{Families: -1, Killmails: -1, Community: 0, Ratings: 12}}
	var rendered bytes.Buffer
	if err := fitCatalogPart(view, "stats").Render(context.Background(), &rendered); err != nil {
		t.Fatal(err)
	}
	body := rendered.String()
	if strings.Contains(body, ">-1<") || strings.Count(body, "Count unavailable") != 2 || strings.Count(body, ">—<") != 2 || !strings.Contains(body, ">0<") || !strings.Contains(body, ">12<") {
		t.Fatal("unknown estimates were displayed as negative or zero counts", body)
	}
}

func TestFittingCatalogueFeedsAndScopedControls(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]string{}
	handler := New(Options{FittingsAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Host != "eve-kill.com" || r.Header.Get("Cookie") != "session=fixture" {
			t.Error("catalogue request lost its identity or method")
		}
		mu.Lock()
		seen[r.URL.Path] = r.URL.RawQuery
		mu.Unlock()
		switch r.URL.Path {
		case "/api/fits/quick-stats":
			_, _ = w.Write([]byte(`{"fittings_known":1200000,"killmails_analyzed":3000,"community_fits":1,"ratings_cast":2}`))
		case "/api/fits/flavors-of-the-week":
			_, _ = w.Write([]byte(`{"window_days":7,"families":[{"family_hash":"family","ship_type_id":587,"ship_name":"Rifter","ranking_count":45,"total_uses":12,"variant_count":3,"fit_cost":10000,"hull_cost":100000,"modules":[{"slot_group":1,"type_id":123,"ordinal":9,"charge_type_id":321}],"drones":[{"type_id":2454,"quantity":7}]}]}`))
		case "/api/fits/top-alliance-doctrines":
			_, _ = w.Write([]byte(`{"window_days":30,"doctrines":[{"entity_id":42,"entity_name":"Test corp","ship_type_id":587,"ship_name":"Rifter","total_losses":100,"doctrine_uses":30,"doctrine_share":30,"modules":[]}]}`))
		case "/api/fits/popular-ships":
			_, _ = w.Write([]byte(`{"ships":[{"ship_type_id":587,"ship_name":"Rifter","total_uses":100,"fit_count":3}]}`))
		case "/api/fits/community-latest", "/api/fits/top-rated":
			_, _ = w.Write([]byte(`{"fits":[{"fit_id":"fixture-fit","name":"<script>unsafe()</script>","ship_type_id":587,"ship_name":"Rifter","owner_character_id":42,"owner_name":"Owner","rating_avg":4.5,"rating_count":2,"created_at":"2026-09-29T00:00:00Z","module_count":5}]}`))
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
	page := request("/fits?flavor=losses&doctrines=corporation")
	if page.Code != 200 {
		t.Fatalf("catalogue status %d", page.Code)
	}
	for _, text := range []string{"Ship Fittings", "1.2M", "3.0K", "Test corp", "45 losses", "3 variants", "4.5", "/fit/fixture-fit", "/item/587/fittings", "/fits/create?fit=v3"} {
		if !strings.Contains(page.Body.String(), text) {
			t.Errorf("missing %s", text)
		}
	}
	if strings.Contains(page.Body.String(), "<script>unsafe()") || !strings.Contains(page.Body.String(), "&lt;script&gt;") {
		t.Fatal("unescaped community fit name")
	}
	if len(seen) != 6 || seen["/api/fits/flavors-of-the-week"] != "mode=losses" || seen["/api/fits/top-alliance-doctrines"] != "entity_type=corporation" {
		t.Fatal("source selection did not reach all six feeds")
	}
	mu.Lock()
	clear(seen)
	mu.Unlock()
	fragment := request("/_fragments/fits/catalog/flavors?flavor=final_blows")
	if fragment.Code != 200 || strings.Contains(fragment.Body.String(), "<header") || !strings.Contains(fragment.Body.String(), "45 final blows") || len(seen) != 1 {
		t.Fatal("flavor change refreshed unrelated content")
	}
}

func TestFittingCatalogueFailureIsIsolated(t *testing.T) {
	handler := New(Options{FittingsAPI: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/fits/flavors-of-the-week" {
			http.Error(w, "private failure", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest("GET", "/fits", nil))
	if page.Code != 200 || !strings.Contains(page.Body.String(), `data-fits-retry="flavors"`) || !strings.Contains(page.Body.String(), "Be the first to share a fit") || strings.Contains(page.Body.String(), "private failure") {
		t.Fatal("one failed catalogue feed took down unrelated sections or leaked its error")
	}
	fragment := httptest.NewRecorder()
	handler.ServeHTTP(fragment, httptest.NewRequest("GET", "/_fragments/fits/catalog/not-a-part", nil))
	if fragment.Code != 404 {
		t.Fatal("unknown catalogue part was accepted")
	}
}
