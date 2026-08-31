package api

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/go-chi/chi/v5"
)

func TestKillboardRegistrarRegistersFrontendAggregateRoutes(t *testing.T) {
	a := humachi.New(
		chi.NewRouter(),
		huma.DefaultConfig("killboard-test", "test"),
	)
	registerKillboardRoutes(a, Options{})

	for path, operationID := range map[string]string{
		"/killlist":            "killlist",
		"/killlist/advanced":   "killlist-advanced",
		"/kills/top":           "kills-top",
		"/kills/most-valuable": "kills-most-valuable",
		"/graph":               "graph",
	} {
		item := a.OpenAPI().Paths[path]
		if item == nil || item.Get == nil {
			t.Errorf("GET %s was not registered", path)
			continue
		}
		if item.Get.OperationID != operationID {
			t.Errorf(
				"GET %s operation = %q, want %q",
				path, item.Get.OperationID, operationID,
			)
		}
		if got := item.Get.Extensions["x-audience"]; got != "public" {
			t.Errorf("GET %s audience = %#v, want public", path, got)
		}
	}
}

func TestAdvancedTimeBoundsUseEVETime(t *testing.T) {
	now := time.Date(2026, time.July, 26, 15, 30, 0, 0, time.UTC)
	since, until, err := advancedTimeBounds(
		&advancedTimeRange{Preset: "thisWeek"}, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	wantMonday := time.Date(2026, time.July, 20, 0, 0, 0, 0, time.UTC)
	if !since.Equal(wantMonday) || until != nil {
		t.Errorf("thisWeek = %s, %v; want %s, nil", since, until, wantMonday)
	}

	since, until, err = advancedTimeBounds(&advancedTimeRange{
		From: "2026-07-21T13:00",
		To:   "2026-07-21T14:30",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if since.Location() != time.UTC ||
		since.Hour() != 13 ||
		until == nil ||
		until.Location() != time.UTC ||
		until.Hour() != 14 ||
		until.Minute() != 30 {
		t.Errorf("custom EVE bounds = %s, %v", since, until)
	}
}

func TestAdvancedQueryUsesBoundedExistsFiltersAndStableCursor(t *testing.T) {
	filters := url.QueryEscape(`{
		"entities":{
			"attacker":[{"id":99000001,"type":"alliance"}]
		},
		"items":[{
			"groupId":46,"slot":"fitted","side":"either"
		}],
		"sort":{"field":"total_value","direction":"desc"}
	}`)
	values, err := url.ParseQuery(
		"limit=500&after=123456&filters=" + filters,
	)
	if err != nil {
		t.Fatal(err)
	}
	query, err := parseAdvancedKilllistQuery(
		&legacyRequest{Query: values},
		time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if query.Limit != 100 {
		t.Errorf("limit = %d, want 100", query.Limit)
	}
	where := query.whereSQL()
	for _, fragment := range []string{
		"EXISTS (",
		"attacker.alliance_id",
		"killmail_items items",
		"items.parent_index IS NULL",
		"killmail_attackers attacker",
		"(COALESCE(k.total_value, 0), k.killmail_id) <",
		"SELECT COALESCE(total_value, 0), killmail_id",
	} {
		if !strings.Contains(where, fragment) {
			t.Errorf("advanced WHERE missing %q:\n%s", fragment, where)
		}
	}
	if strings.Contains(where, "array_agg") ||
		strings.Contains(where, "ARRAY[") {
		t.Errorf("advanced query materializes attacker IDs:\n%s", where)
	}
}

func TestAdvancedQueryUsesCanonicalLabelPredicate(t *testing.T) {
	filters := url.QueryEscape(`{"label":"attackers-10-24"}`)
	values, err := url.ParseQuery("filters=" + filters)
	if err != nil {
		t.Fatal(err)
	}
	query, err := parseAdvancedKilllistQuery(
		&legacyRequest{Query: values},
		time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query.whereSQL(), "k.attacker_count BETWEEN 10 AND 24") {
		t.Errorf("advanced label did not use canonical predicate:\n%s", query.whereSQL())
	}

	bad := url.QueryEscape(`{"label":"not-a-label"}`)
	values, _ = url.ParseQuery("filters=" + bad)
	_, err = parseAdvancedKilllistQuery(
		&legacyRequest{Query: values}, time.Now().UTC(),
	)
	apiErr, ok := err.(*legacyAPIError)
	if !ok || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("invalid label error = %#v", err)
	}
}

func TestAdvancedSearchAcceptsEveryPublicLabel(t *testing.T) {
	for _, label := range killtype.Labels {
		filters := url.QueryEscape(`{"label":"` + label.ID + `"}`)
		values, err := url.ParseQuery("filters=" + filters)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parseAdvancedKilllistQuery(
			&legacyRequest{Query: values}, time.Now().UTC(),
		); err != nil {
			t.Errorf("label %q was rejected by advanced search: %v", label.ID, err)
		}
	}
}

func TestAdvancedFitItemsPreserveSlotAndDroneShape(t *testing.T) {
	modules, drones := advancedFitItems([]map[string]any{
		{
			"type_id": int32(100), "flag_id": int32(27),
			"category_id": int32(7), "type_name": "Gun",
		},
		{
			"type_id": int32(200), "flag_id": int32(27),
			"category_id": int32(8), "type_name": "Ammo",
		},
		{
			"type_id": int32(300), "flag_id": int32(87),
			"category_id": int32(18), "type_name": "Drone",
			"quantity_dropped": int64(2), "quantity_destroyed": int64(3),
		},
	})
	if len(modules) != 1 ||
		modules[0]["slot_group"] != int64(1) ||
		modules[0]["charge_type_id"] != int32(200) {
		t.Errorf("modules = %#v", modules)
	}
	wantDrones := []map[string]any{{
		"type_id": int64(300), "name": "Drone", "quantity": int64(5),
	}}
	if !reflect.DeepEqual(drones, wantDrones) {
		t.Errorf("drones = %#v, want %#v", drones, wantDrones)
	}
}

func TestGraphModeValidationAndEmptyPathDoNotRequireDatabases(t *testing.T) {
	handler := graphHandler(Options{})
	payload, err := handler(context.Background(), &legacyRequest{
		Query: url.Values{"mode": {"path_finder"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := payload.Body.(map[string]any)
	if body["path"] != nil || body["mode"] != "path_finder" {
		t.Errorf("empty path response = %#v", body)
	}

	_, err = handler(context.Background(), &legacyRequest{
		Query: url.Values{"mode": {"not-a-mode"}},
	})
	apiErr, ok := err.(*legacyAPIError)
	if !ok ||
		apiErr.Status != http.StatusBadRequest ||
		apiErr.Message != "Unknown graph mode: not-a-mode" {
		t.Errorf("unknown mode error = %#v", err)
	}
}

func TestGraphUnionMaintainsStableComponents(t *testing.T) {
	union := newGraphUnion()
	union.join(30, 10)
	union.join(20, 30)
	union.join(50, 40)
	components := union.components()
	got := make([][]int64, 0, len(components))
	for _, members := range components {
		got = append(got, members)
	}
	sortNestedInt64(got)
	want := [][]int64{{10, 20, 30}, {40, 50}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("components = %#v, want %#v", got, want)
	}
}

func sortNestedInt64(values [][]int64) {
	for _, value := range values {
		for i := 1; i < len(value); i++ {
			for j := i; j > 0 && value[j] < value[j-1]; j-- {
				value[j], value[j-1] = value[j-1], value[j]
			}
		}
	}
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j][0] < values[j-1][0]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func TestKilllistFactionIDsRejectFractionsAndDeduplicate(t *testing.T) {
	got := parseKilllistFactionIDs("500001, 1.5, nope, 500001, -9")
	want := []int32{500001, -9}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("factions = %#v, want %#v", got, want)
	}
}

func TestFactionFilteredCursorKilllistSkipsUnboundedTotal(t *testing.T) {
	total, err := killlistCursorTotal(
		context.Background(), nil, "latest", true, 2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("faction-filtered cursor total = %d, want omitted", total)
	}
}

func TestInitialRollupKilllistUsesBoundedFirstPage(t *testing.T) {
	page, ok := killlistNumberedPage(true, 0, 0, nil)
	if !ok || page != 1 {
		t.Fatalf("initial rollup page = %d, %v; want 1, true", page, ok)
	}
	after := int64(123)
	if _, ok := killlistNumberedPage(true, 0, 0, &after); ok {
		t.Fatal("cursor request unexpectedly switched to numbered mode")
	}
	if _, ok := killlistNumberedPage(true, 2, 0, nil); ok {
		t.Fatal("faction-filtered request unexpectedly used global rollup")
	}
}
