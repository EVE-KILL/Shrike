package api

import (
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func TestAdminRoutesRegisterWithRequiredSessionSecurity(t *testing.T) {
	mux := http.NewServeMux()
	a := humago.New(mux, huma.DefaultConfig("test", "test"))
	registerAdminRoutes(a, Options{})

	for _, route := range []struct {
		method, path string
	}{
		{http.MethodGet, "/admin/overview"},
		{http.MethodGet, "/admin/users"},
		{http.MethodGet, "/admin/users/{id}"},
		{http.MethodPost, "/admin/users/{id}/set-discord"},
		{http.MethodPost, "/admin/users/{id}/toggle-admin"},
		{http.MethodGet, "/admin/esi"},
		{http.MethodGet, "/admin/esi-logs"},
		{http.MethodGet, "/admin/esi-entities"},
	} {
		item := a.OpenAPI().Paths[route.path]
		if item == nil {
			t.Errorf("%s %s path is missing", route.method, route.path)
			continue
		}
		var operation *huma.Operation
		if route.method == http.MethodGet {
			operation = item.Get
		} else {
			operation = item.Post
		}
		if operation == nil {
			t.Errorf("%s %s operation is missing", route.method, route.path)
			continue
		}
		if len(operation.Security) != 1 {
			t.Errorf("%s %s security = %#v", route.method, route.path, operation.Security)
			continue
		}
		if _, ok := operation.Security[0]["eveSession"]; !ok {
			t.Errorf("%s %s does not require eveSession", route.method, route.path)
		}
	}
}

func TestAdminPaginationMatchesFrontendBounds(t *testing.T) {
	for _, test := range []struct {
		raw                string
		fallback, min, max int
		want               int
	}{
		{"", 50, 1, 100, 50},
		{"0", 50, 1, 100, 50},
		{"-5", 50, 1, 100, 1},
		{"75", 50, 1, 100, 75},
		{"500", 50, 1, 100, 100},
		{"not-a-number", 50, 1, 100, 50},
	} {
		if got := adminBoundedNumber(
			test.raw, test.fallback, test.min, test.max,
		); got != test.want {
			t.Errorf("bounded %q = %d, want %d", test.raw, got, test.want)
		}
	}
}

func TestParseAdminCharacterIDUsesFrontendErrorContract(t *testing.T) {
	for _, raw := range []string{
		"", "0", "-1", "1.5", "2147483648", "Infinity", "not-a-number",
	} {
		_, err := parseAdminCharacterID(raw)
		var apiErr *legacyAPIError
		if !errors.As(err, &apiErr) ||
			apiErr.Status != http.StatusBadRequest ||
			apiErr.Message != "Invalid character ID" {
			t.Errorf("parse %q error = %#v", raw, err)
		}
	}
	if got, err := parseAdminCharacterID(" 90000001 "); err != nil ||
		got != 90000001 {
		t.Errorf("valid ID = %v, %v", got, err)
	}
}

func TestDiscordSnowflakeValidation(t *testing.T) {
	for _, valid := range []string{
		"123456789012345",
		"1234567890123456789012",
	} {
		if !discordSnowflakePattern.MatchString(valid) {
			t.Errorf("%q should be valid", valid)
		}
	}
	for _, invalid := range []string{
		"12345678901234", "12345678901234567890123",
		"12345678901234x", "１２３４５６７８９０１２３４５",
	} {
		if discordSnowflakePattern.MatchString(invalid) {
			t.Errorf("%q should be invalid", invalid)
		}
	}
}

func TestParseAdminESILogFiltersPreservesPollingAndFilters(t *testing.T) {
	req := &legacyRequest{Query: url.Values{
		"character_id":   {"9001"},
		"corporation_id": {"9801"},
		"search":         {"  killmails  "},
		"source":         {" worker "},
		"status":         {"error"},
		"endpoint_type":  {"corporation"},
		"has_new":        {"true"},
		"after_id":       {"321"},
		"page":           {"3"},
		"limit":          {"200"},
	}}
	got := parseAdminESILogFilters(req)
	if got.CharacterID == nil || *got.CharacterID != 9001 ||
		got.CorporationID == nil || *got.CorporationID != 9801 ||
		got.AfterID == nil || *got.AfterID != 321 {
		t.Fatalf("numeric filters = %#v", got)
	}
	if got.Search != "killmails" || got.Source != "worker" ||
		got.Status != "error" || got.EndpointType != "corporation" ||
		!got.HasNew || got.Page != 3 || got.Limit != 100 {
		t.Fatalf("filters = %#v", got)
	}

	where, args := buildAdminESILogWhere(got)
	for _, fragment := range []string{
		"log.character_id = $1",
		"corporation_id = $2",
		"log.endpoint ILIKE $3",
		"log.source = $4",
		"log.success = FALSE",
		"log.endpoint ILIKE '%/corporations/%'",
		"log.new_items >= 1",
	} {
		if !strings.Contains(where, fragment) {
			t.Errorf("where %q does not contain %q", where, fragment)
		}
	}
	wantArgs := []any{float64(9001), float64(9801), "%killmails%", "worker"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Errorf("args = %#v, want %#v", args, wantArgs)
	}
}

func TestAdminESILogPageQueryKeepsCountPageAndSourcesInOneRequest(t *testing.T) {
	query := adminESILogPageSQL(
		"WHERE log.source = $1", 2, 3,
	)
	for _, fragment := range []string{
		"result_count AS",
		"source_options AS",
		"named_page AS",
		"endpoint_entity_name",
		"stored_item_ids",
		"JOIN killmails stored",
		"LIMIT $2 OFFSET $3",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("query does not contain %q", fragment)
		}
	}
	if got := strings.Count(query, "WHERE log.source = $1"); got != 2 {
		t.Errorf("filter occurrence count = %d, want 2", got)
	}
}

func TestAdminStringSliceAlwaysProducesStableArray(t *testing.T) {
	if got := adminStringSlice([]any{"queue", nil, "worker"}); !reflect.DeepEqual(got, []string{"queue", "worker"}) {
		t.Errorf("slice = %#v", got)
	}
	if got := adminStringSlice(nil); got == nil || len(got) != 0 {
		t.Errorf("nil sources = %#v, want []", got)
	}
}
