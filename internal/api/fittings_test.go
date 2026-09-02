package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func TestFittingRoutesRegisterCanonicalAndFrontendAliases(t *testing.T) {
	mux := http.NewServeMux()
	a := humago.New(mux, huma.DefaultConfig("test", "test"))
	registerFittingRoutes(a, Options{})

	for _, route := range []struct {
		method, path string
	}{
		{http.MethodPost, "/fittings"},
		{http.MethodPost, "/fit"},
		{http.MethodGet, "/fittings/{id}"},
		{http.MethodGet, "/fit/{fit_id}"},
		{http.MethodPatch, "/fittings/{id}"},
		{http.MethodDelete, "/fittings/{id}"},
		{http.MethodPut, "/fittings/{id}/rating"},
		{http.MethodGet, "/fittings/community/latest"},
		{http.MethodGet, "/fits/community-latest"},
		{http.MethodGet, "/fittings/trending"},
		{http.MethodGet, "/fits/flavors-of-the-week"},
		{http.MethodGet, "/fittings/search"},
		{http.MethodGet, "/fits/search"},
		{http.MethodGet, "/fittings/ships/{id}/families"},
		{http.MethodGet, "/item/{id}/fittings"},
		{http.MethodGet, "/fittings/ships/{id}/metadata"},
		{http.MethodGet, "/item/{id}/fit-meta"},
	} {
		item := a.OpenAPI().Paths[route.path]
		if item == nil {
			t.Errorf("%s %s path is missing", route.method, route.path)
			continue
		}
		var exists bool
		switch route.method {
		case http.MethodGet:
			exists = item.Get != nil
		case http.MethodPost:
			exists = item.Post != nil
		case http.MethodPatch:
			exists = item.Patch != nil
		case http.MethodPut:
			exists = item.Put != nil
		case http.MethodDelete:
			exists = item.Delete != nil
		}
		if !exists {
			t.Errorf("%s %s operation is missing", route.method, route.path)
		}
	}
}

func TestTrendingFittingsModeContract(t *testing.T) {
	mux := http.NewServeMux()
	a := humago.New(mux, huma.DefaultConfig("test", "test"))
	registerFittingRoutes(a, Options{})

	for _, path := range []string{"/fittings/trending", "/fits/flavors-of-the-week"} {
		operation := a.OpenAPI().Paths[path].Get
		if operation == nil || len(operation.Parameters) != 1 {
			t.Fatalf("%s parameters = %#v", path, operation)
		}
		parameter := operation.Parameters[0]
		if parameter.Name != "mode" || parameter.Schema.Default != "kills" ||
			!reflect.DeepEqual(parameter.Schema.Enum, []any{"kills", "final_blows", "losses"}) {
			t.Errorf("%s mode parameter = %#v", path, parameter)
		}
	}

	_, err := trendingFittingsHandler(Options{})(context.Background(), &legacyRequest{
		Query: url.Values{"mode": {"invalid"}},
	})
	var apiErr *legacyAPIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("invalid mode error = %#v", err)
	}
}

func TestDoctrineEntityTypeContract(t *testing.T) {
	mux := http.NewServeMux()
	a := humago.New(mux, huma.DefaultConfig("test", "test"))
	registerFittingRoutes(a, Options{})

	for _, path := range []string{"/fittings/doctrines/alliances", "/fits/top-alliance-doctrines"} {
		operation := a.OpenAPI().Paths[path].Get
		if operation == nil || len(operation.Parameters) != 1 {
			t.Fatalf("%s parameters = %#v", path, operation)
		}
		parameter := operation.Parameters[0]
		if parameter.Name != "entity_type" || parameter.Schema.Default != "alliance" ||
			!reflect.DeepEqual(parameter.Schema.Enum, []any{"alliance", "corporation"}) {
			t.Errorf("%s entity_type parameter = %#v", path, parameter)
		}
	}

	_, err := allianceDoctrineHandler(Options{})(context.Background(), &legacyRequest{
		Query: url.Values{"entity_type": {"invalid"}},
	})
	var apiErr *legacyAPIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("invalid entity type error = %#v", err)
	}
}

// Decoding from JSON rather than building the struct by hand keeps the test on
// the path a request actually takes, so a tag or a field type that stops
// matching the wire format fails here.
func decodeFittingCreate(t *testing.T, payload string) (*fittingCreateBody, error) {
	t.Helper()
	return decodeJSONBody[fittingCreateBody](
		&legacyRequest{Body: strings.NewReader(payload)}, fittingBodyLimit,
	)
}

func TestValidateCreateFittingNormalizesAndValidatesItems(t *testing.T) {
	body, err := decodeFittingCreate(t, `{
		"ship_type_id": 587,
		"name": "  Rifter PvP  ",
		"description": "",
		"visibility": 3,
		"items": [
			{"slot_group":1,"ordinal":0,"type_id":2001,"state":2,"charge_type_id":null},
			{"slot_group":6,"ordinal":0,"type_id":2002,"state":0,"quantity":5}
		]
	}`)
	if err != nil {
		t.Fatal(err)
	}

	got, err := validateCreateFitting(body)
	if err != nil {
		t.Fatal(err)
	}
	if *got.ShipTypeID != 587 || *got.Name != "Rifter PvP" ||
		got.Description != nil || *got.Visibility != 3 {
		t.Errorf("validated header = %#v", got)
	}
	if len(got.Items) != 2 || got.Items[0].Quantity != 1 ||
		got.Items[1].Quantity != 5 {
		t.Errorf("validated items = %#v", got.Items)
	}

	// A module slot may only hold one of a thing.
	modules, err := decodeFittingCreate(t, `{
		"ship_type_id": 587, "name": "Rifter PvP", "visibility": 3,
		"items": [{"slot_group":1,"ordinal":0,"type_id":2001,"state":2,"quantity":2}]
	}`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = validateCreateFitting(modules)
	var apiErr *legacyAPIError
	if !errors.As(err, &apiErr) || apiErr.Status != 422 ||
		!strings.Contains(apiErr.Message, "must be 1 for module slots") {
		t.Fatalf("module quantity error = %#v", err)
	}
}

func TestValidateFittingItemsRejectsDuplicateSlots(t *testing.T) {
	body, err := decodeFittingCreate(t, `{
		"ship_type_id": 587, "name": "Dupe", "visibility": 0,
		"items": [
			{"slot_group":2,"ordinal":3,"type_id":10,"state":1},
			{"slot_group":2,"ordinal":3,"type_id":11,"state":1}
		]
	}`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = validateFittingItems(body.Items)
	var apiErr *legacyAPIError
	if !errors.As(err, &apiErr) ||
		apiErr.Message != "Duplicate slot_group=2 ordinal=3 at items[1]" {
		t.Fatalf("duplicate error = %#v", err)
	}
}

// An absent description leaves the stored text alone; an explicit null clears
// it. The fitting editor relies on both.
func TestFittingUpdateSeparatesAbsentFromClearedDescription(t *testing.T) {
	decode := func(payload string) *fittingUpdateBody {
		t.Helper()
		body, err := decodeJSONBody[fittingUpdateBody](
			&legacyRequest{Body: strings.NewReader(payload)}, fittingBodyLimit,
		)
		if err != nil {
			t.Fatal(err)
		}
		return body
	}

	patch, err := validateUpdateFitting(decode(`{"name":"Renamed"}`))
	if err != nil {
		t.Fatal(err)
	}
	if patch.HasDescription {
		t.Error("an absent description was treated as an update")
	}

	patch, err = validateUpdateFitting(decode(`{"description":null}`))
	if err != nil {
		t.Fatal(err)
	}
	if !patch.HasDescription || patch.Description != nil {
		t.Errorf("null description = %#v, want a cleared update", patch.Description)
	}
}

func TestFittingSearchFilterParserPreservesFrontendContract(t *testing.T) {
	got, err := parseFittingSearchFilters(`[
		{"role_id":"armor-rep","op":">=","count":2},
		{"type_id":2048,"type_name":"Damage Control II","op":"=","count":1}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	want := []fittingSearchFilter{
		{Kind: "role", RoleID: "armor-rep", Op: ">=", Count: 2},
		{
			Kind: "type", TypeID: 2048, TypeName: "Damage Control II",
			Op: "=", Count: 1,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filters = %#v, want %#v", got, want)
	}
	for raw, message := range map[string]string{
		`{}`: "filters must be an array",
		`[{"role_id":"nope","op":"=","count":1}]`:       `unknown role_id "nope"`,
		`[{"role_id":"armor-rep","op":"!=","count":1}]`: `unsupported op "!="`,
		`[{"type_id":1,"op":"=","count":17}]`:           "count must be an integer 0..16",
	} {
		_, err := parseFittingSearchFilters(raw)
		var apiErr *legacyAPIError
		if !errors.As(err, &apiErr) || apiErr.Message != message {
			t.Errorf("parse %s error = %#v, want %q", raw, err, message)
		}
	}
}

func TestFittingSearchPaginationMatchesJavaScriptClamp(t *testing.T) {
	req := &legacyRequest{Query: url.Values{
		"limit":  {"12.9"},
		"offset": {"-4"},
	}}
	if got := fittingSearchPageNumber(req, "limit", 24, 1, 50); got != 12 {
		t.Errorf("limit = %d, want 12", got)
	}
	if got := fittingSearchPageNumber(req, "offset", 0, 0, 1<<31-1); got != 0 {
		t.Errorf("offset = %d, want 0", got)
	}
}

func TestFittingSearchSupportsSharedStatisticFilters(t *testing.T) {
	req := &legacyRequest{Query: url.Values{
		"sort":             {"npc_ehp"},
		"npc_profile":      {"guristas"},
		"min_npc_ehp":      {"150000"},
		"max_speed":        {"2200"},
		"min_armor_repair": {"100"},
	}}
	statQuery, err := parseFittingStatQuery(req)
	if err != nil {
		t.Fatal(err)
	}
	query, args, err := buildFittingSearchQuery(req, 29990, nil, nil, statQuery, 24, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"stats.max_velocity <=", "stats.armor_effective_repair >=", "'npc_profile', 'guristas'", "'npc_ehp'"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("query does not contain %q", fragment)
		}
	}
	if len(args) != 6 {
		t.Fatalf("args = %#v, want ship, limit, offset, and three statistic bounds", args)
	}
}

func TestGenerateFittingIDUsesFrontendLengthAndAlphabet(t *testing.T) {
	id, err := generateFittingID()
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != fittingIDLength {
		t.Fatalf("fit ID length = %d, want %d", len(id), fittingIDLength)
	}
	for _, character := range id {
		if !strings.ContainsRune(fittingIDAlphabet, character) {
			t.Fatalf("fit ID %q contains invalid character %q", id, character)
		}
	}
}
