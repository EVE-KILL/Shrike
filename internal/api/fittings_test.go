package api

import (
	"encoding/json"
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

func TestValidateCreateFittingNormalizesAndValidatesItems(t *testing.T) {
	body := map[string]any{
		"ship_type_id": json.Number("587"),
		"name":         "  Rifter PvP  ",
		"description":  "",
		"visibility":   json.Number("3"),
		"items": []any{
			map[string]any{
				"slot_group": json.Number("1"),
				"ordinal":    json.Number("0"), "type_id": json.Number("2001"),
				"state": json.Number("2"), "charge_type_id": nil,
			},
			map[string]any{
				"slot_group": json.Number("6"),
				"ordinal":    json.Number("0"), "type_id": json.Number("2002"),
				"state": json.Number("0"), "quantity": json.Number("5"),
			},
		},
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

	body["items"] = []any{
		map[string]any{
			"slot_group": json.Number("1"),
			"ordinal":    json.Number("0"), "type_id": json.Number("2001"),
			"state": json.Number("2"), "quantity": json.Number("2"),
		},
	}
	_, err = validateCreateFitting(body)
	var apiErr *legacyAPIError
	if !errors.As(err, &apiErr) || apiErr.Status != 422 ||
		!strings.Contains(apiErr.Message, "must be 1 for module slots") {
		t.Fatalf("module quantity error = %#v", err)
	}
}

func TestValidateFittingItemsRejectsDuplicateSlots(t *testing.T) {
	item := func(typeID string) map[string]any {
		return map[string]any{
			"slot_group": json.Number("2"),
			"ordinal":    json.Number("3"), "type_id": json.Number(typeID),
			"state": json.Number("1"),
		}
	}
	_, err := validateFittingItems([]any{item("10"), item("11")})
	var apiErr *legacyAPIError
	if !errors.As(err, &apiErr) ||
		apiErr.Message != "Duplicate slot_group=2 ordinal=3 at items[1]" {
		t.Fatalf("duplicate error = %#v", err)
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
