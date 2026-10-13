package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/frontend"
)

var _ frontend.UniverseReader = remoteUniverse{}

func TestRemoteUniversePreservesShipScopeCursorsAndMarketFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Error("universe reader forwarded credentials or mutation")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/ship/587/killlist", "/api/item/34/killlist":
			if r.URL.Query().Get("limit") != "10" || r.URL.Query().Get("after") != "123" {
				t.Error("lost item cursor")
			}
			_, _ = w.Write([]byte(`{"kills":[],"cursor":99,"hasMore":true}`))
		case "/api/market/items/34/orders":
			if r.URL.Query().Get("region_id") != "10000002" || r.URL.Query().Get("security") != "high,null" || r.URL.Query().Get("limit") != "100" {
				t.Error("lost market scope")
			}
			_, _ = w.Write([]byte(`{"sellers":[{"order_id":42,"price":4.5}],"buyers":[],"regions":[]}`))
		case "/api/market/items/34/history":
			if r.URL.Query().Get("region_id") != "10000002" || r.URL.Query().Get("days") != "90" {
				t.Error("lost history scope")
			}
			_, _ = w.Write([]byte(`{"region_id":10000002,"days":90,"history":[]}`))
		case "/api/killlist/advanced":
			var filters struct {
				Entities struct {
					Victim []struct {
						ID   int64
						Type string
					}
				}
				TimeRange struct{ Preset string }
			}
			if json.Unmarshal([]byte(r.URL.Query().Get("filters")), &filters) != nil || len(filters.Entities.Victim) != 1 || filters.Entities.Victim[0].ID != 25 || filters.Entities.Victim[0].Type != "shipgroup" || filters.TimeRange.Preset != "90d" {
				t.Error("lost group loss/time filters")
			}
			_, _ = w.Write([]byte(`{"kills":[],"hasMore":false}`))
		case "/api/fittings/ships/587/distributions":
			if r.URL.Query().Get("days") != "90" || r.URL.Query().Get("security") != "low" {
				t.Error("lost distribution query")
			}
			_, _ = w.Write([]byte(`{"metrics":[]}`))
		case "/api/universe/types/9":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Error("unexpected route", r.URL.Path)
		}
	}))
	defer server.Close()
	r, ctx, after := remoteUniverse{server.URL}, context.Background(), int64(123)
	for _, item := range []struct {
		id   int64
		ship bool
	}{{587, true}, {34, false}} {
		data, err := r.UniverseItemKills(ctx, item.id, item.ship, 10, &after)
		if err != nil || !data.HasMore || data.Cursor == nil || *data.Cursor != 99 {
			t.Fatal(data, err)
		}
	}
	if orders, err := r.UniverseMarketOrders(ctx, 34, 10000002, "high,null"); err != nil || len(orders.Sellers) != 1 || orders.Sellers[0].Price != 4.5 {
		t.Fatal(orders, err)
	}
	if history, err := r.UniverseMarketHistory(ctx, 34, 10000002, 90); err != nil || history.Days != 90 {
		t.Fatal(history, err)
	}
	if _, err := r.UniverseGroupKills(ctx, 25, 50, nil); err != nil {
		t.Fatal(err)
	}
	query := url.Values{"days": {"30"}, "security": {"low"}}
	if _, err := r.UniverseFitDistributions(ctx, 587, query); err != nil || query.Get("days") != "30" {
		t.Fatal("changed caller query", query, err)
	}
	if _, found, err := r.UniverseItem(ctx, 9); err != nil || found {
		t.Fatal(found, err)
	}
}

func TestUniverseReferenceLoadsCooperatingOriginalModulesWithoutTypedDuplicates(t *testing.T) {
	renderer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" {
			t.Fatal("source received credentials")
		}
		_, _ = w.Write([]byte(`<div data-controller="universe"><div data-controller="item-market"></div><div data-controller="location"></div></div><script src="/assets/current/universe.js"></script><script src="/assets/current/market.js"></script><script type="application/json">{"same":"data"}</script>`))
	})
	scripts := map[string][]byte{}
	h := universeReferenceRenderer(referenceRendererMustNotMutate{}, renderer, scripts)
	for _, path := range []string{"/market", "/market/item/34", "/item/587/market", "/group/25", "/system/30000142", "/region/10000002", "/constellation/20000020", "/_fragments/item/market/34"} {
		w := httptest.NewRecorder()
		request := httptest.NewRequest("GET", path, nil)
		request.Header.Set("Cookie", "private")
		h.ServeHTTP(w, request)
		if strings.Contains(w.Body.String(), `data-controller="universe"`) || strings.Contains(w.Body.String(), `data-controller="item-market"`) || strings.Contains(w.Body.String(), `data-controller="location"`) || !strings.Contains(w.Body.String(), `/__fixture/universe-source/explorer.js`) || !strings.Contains(w.Body.String(), `{"same":"data"}`) {
			t.Fatal(w.Body.String())
		}
	}
	if !strings.Contains(string(scripts["explorer.js"]), "import './system.js'") {
		t.Fatal("missing source cooperation")
	}
}

func TestRemoteLocationBattlesUseRealAnonymousPublicQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Header.Get("Cookie") != "" || r.URL.Path != "/api/conflicts/battles" || r.URL.Query().Get("systemId") != "30000142" || r.URL.Query().Get("page") != "2" || r.URL.Query().Get("limit") != "50" {
			t.Error("changed location battle scope")
		}
		_, _ = w.Write([]byte(`{"page":2,"battles":[]}`))
	}))
	defer server.Close()
	r := remoteLocationEntities{api.NewHomepageReader(nil), server.URL}
	data, err := r.EntityBattles(context.Background(), "system", 30000142, 2)
	if err != nil || data.Page != 2 {
		t.Fatal(data, err)
	}
}
