package api

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/eve-kill/shrike/internal/stats"
	"github.com/go-chi/chi/v5"
)

func TestDomainKillboardRegistersFrontendRoutesAsPublic(t *testing.T) {
	a := humachi.New(
		chi.NewRouter(),
		huma.DefaultConfig("domain-killboard-test", "test"),
	)
	registerDomainKillboardRoutes(a, Options{})

	for path, operationID := range map[string]string{
		"/custom/killlist":                    "domain-killlist",
		"/custom/region/{id}/killlist":        "domain-region-killlist",
		"/custom/constellation/{id}/killlist": "domain-constellation-killlist",
		"/custom/system/{id}/killlist":        "domain-system-killlist",
		"/custom/kills/most-valuable":         "domain-kills-most-valuable",
		"/custom/kills/top":                   "domain-kills-top",
		"/custom/stats":                       "domain-statistics",
	} {
		item := a.OpenAPI().Paths[path]
		if item == nil || item.Get == nil {
			t.Errorf("GET %s was not registered", path)
			continue
		}
		if item.Get.OperationID != operationID {
			t.Errorf(
				"GET %s operation = %q, want %q",
				path,
				item.Get.OperationID,
				operationID,
			)
		}
		if len(item.Get.Security) != 0 {
			t.Errorf(
				"GET %s unexpectedly requires authentication",
				path,
			)
		}
		if got := item.Get.Extensions["x-audience"]; got != "public" {
			t.Errorf(
				"GET %s audience = %#v, want public",
				path,
				got,
			)
		}
	}
}

func TestDomainReadCacheKeySeparatesHostsAndQueries(t *testing.T) {
	first := domainReadCacheKeyFor(
		"red.eve-kill.com",
		"/custom/killlist?type=solo",
	)
	second := domainReadCacheKeyFor(
		"blue.eve-kill.com",
		"/custom/killlist?type=solo",
	)
	third := domainReadCacheKeyFor(
		"red.eve-kill.com",
		"/custom/killlist?type=latest",
	)
	if first == second {
		t.Fatal("different domain hosts share a cache key")
	}
	if first == third {
		t.Fatal("different domain queries share a cache key")
	}
}

func TestDomainKilllistQueryScopesVictimsAndAttackersBeforeOuterFilters(
	t *testing.T,
) {
	scope := &domainEntityScope{
		Characters:   []int32{90000001},
		Corporations: []int32{98000001},
		Alliances:    []int32{99000001},
	}
	locationID := int64(20000020)
	after := int64(123456789)
	query, args := buildDomainKilllistQuery(
		scope,
		"solo",
		"constellation_id",
		&locationID,
		&after,
		10,
	)

	for _, fragment := range []string{
		"WITH candidates AS MATERIALIZED",
		"victim_character_id = ANY(",
		"character_id = ANY(",
		"victim_corporation_id = ANY(",
		"corporation_id = ANY(",
		"victim_alliance_id = ANY(",
		"alliance_id = ANY(",
		"FROM killmail_attackers",
		"k.killmail_id IN (SELECT killmail_id FROM candidates)",
		"k.constellation_id = ",
		"k.killmail_id < ",
		"k.is_solo = true",
		"LIMIT 250",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("domain killlist query lacks %q:\n%s", fragment, query)
		}
	}
	if got := strings.Count(query, "constellation_id = "); got != 4 {
		t.Errorf(
			"location predicate count = %d, want three victim legs and one outer filter",
			got,
		)
	}
	for _, rawID := range []string{
		"90000001", "98000001", "99000001", "123456789", "20000020",
	} {
		if strings.Contains(query, rawID) {
			t.Errorf("domain id %s was interpolated into SQL", rawID)
		}
	}
	if len(args) != 6 ||
		args[0] != after ||
		args[1] != locationID ||
		!reflect.DeepEqual(args[2], scope.Characters) ||
		!reflect.DeepEqual(args[3], scope.Corporations) ||
		!reflect.DeepEqual(args[4], scope.Alliances) ||
		args[5] != 11 {
		t.Fatalf("domain killlist args = %#v", args)
	}
}

func TestDomainKilllistQueryCannotFallBackToGlobalData(t *testing.T) {
	query, args := buildDomainKilllistQuery(
		&domainEntityScope{},
		"latest",
		"",
		nil,
		nil,
		50,
	)
	if !strings.Contains(
		query,
		"SELECT NULL::bigint AS killmail_id WHERE false",
	) {
		t.Fatalf("empty domain query is not forced empty:\n%s", query)
	}
	// The one remaining reference is the bounded final-blow enrichment in the
	// shared SELECT, not a candidate leg capable of widening the result.
	if got := strings.Count(query, "FROM killmail_attackers"); got != 1 {
		t.Fatalf(
			"empty domain query has %d attacker references, want final-blow enrichment only:\n%s",
			got,
			query,
		)
	}
	if len(args) != 1 || args[0] != 51 {
		t.Fatalf("empty domain query args = %#v", args)
	}
}

func TestDomainHeadlineStatisticsUseTypeScriptHierarchyScope(t *testing.T) {
	scope := &domainEntityScope{
		Characters:   []int32{90000001},
		Corporations: []int32{98000001},
		Alliances:    []int32{99000001},
	}
	characterQuery, characterArgs := buildDomainHeadlineStatisticsQuery(
		scope,
		domainHeadlineStatistics["characters"],
		"2026-07-20",
		10,
	)
	for _, fragment := range []string{
		"entity.character_id = ANY(",
		"entity.corporation_id = ANY(",
		"SELECT corporation_id FROM corporations",
		"WHERE alliance_id = ANY(",
		"s.entity_type = ",
		"s.period_type = ",
	} {
		if !strings.Contains(characterQuery, fragment) {
			t.Errorf(
				"character hierarchy query lacks %q:\n%s",
				fragment,
				characterQuery,
			)
		}
	}
	for _, expected := range []any{
		stats.EntityCharacter,
		stats.PeriodDaily,
		"2026-07-20",
		scope.Characters,
		scope.Corporations,
		scope.Alliances,
		10,
	} {
		found := false
		for _, actual := range characterArgs {
			if reflect.DeepEqual(actual, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("character query args lack %#v: %#v", expected, characterArgs)
		}
	}

	corporationQuery, _ := buildDomainHeadlineStatisticsQuery(
		scope,
		domainHeadlineStatistics["corporations"],
		"2026-07-20",
		10,
	)
	if strings.Contains(corporationQuery, "entity.character_id") ||
		!strings.Contains(corporationQuery, "entity.corporation_id") ||
		!strings.Contains(corporationQuery, "entity.alliance_id") {
		t.Fatalf("corporation hierarchy scope is wrong:\n%s", corporationQuery)
	}
}

func TestDomainBreakdownsUseOnlyExactConfiguredEntities(t *testing.T) {
	scope := &domainEntityScope{
		Characters:   []int32{90000001},
		Corporations: []int32{98000001},
		Alliances:    []int32{99000001},
	}
	query, args := buildDomainBreakdownStatisticsQuery(
		scope,
		domainBreakdownStatistics["most_destroyed_ships"],
		"2026-07-20",
		25,
	)
	for _, fragment := range []string{
		"b.entity_type = ",
		"b.entity_id = ANY(",
		"b.dim_category = ",
		"SUM(b.losses)",
		"JOIN inv_types",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("breakdown query lacks %q:\n%s", fragment, query)
		}
	}
	if strings.Contains(query, "SELECT corporation_id FROM corporations") ||
		strings.Contains(query, "entity.corporation_id = ANY") {
		t.Fatalf("breakdown query expanded hierarchy:\n%s", query)
	}
	for _, ids := range []any{
		scope.Characters,
		scope.Corporations,
		scope.Alliances,
	} {
		found := false
		for _, arg := range args {
			if reflect.DeepEqual(arg, ids) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("breakdown args lack scope %#v: %#v", ids, args)
		}
	}

	empty, _ := buildDomainBreakdownStatisticsQuery(
		&domainEntityScope{},
		domainBreakdownStatistics["ships"],
		"2026-07-20",
		10,
	)
	if !strings.Contains(empty, "WHERE (false)") {
		t.Fatalf("empty breakdown scope is not forced empty:\n%s", empty)
	}
}

func TestDomainMostValuableIsAttackerOnlyAndCategoryAware(
	t *testing.T,
) {
	scope := &domainEntityScope{
		Characters:   []int32{90000001},
		Corporations: []int32{98000001},
		Alliances:    []int32{99000001},
	}
	query, args := buildDomainMostValuableQuery(
		scope,
		time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC),
		20,
		"k.is_solo = true",
		65,
	)
	for _, fragment := range []string{
		"FROM killmail_attackers attacker",
		"attacker.character_id = ANY(",
		"attacker.corporation_id = ANY(",
		"attacker.alliance_id = ANY(",
		"k.is_solo = true",
		"WHERE category_id = ",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("most-valuable query lacks %q:\n%s", fragment, query)
		}
	}
	for _, forbidden := range []string{
		"k.victim_character_id = ANY(",
		"k.victim_corporation_id = ANY(",
		"k.victim_alliance_id = ANY(",
	} {
		if strings.Contains(query, forbidden) {
			t.Errorf(
				"most-valuable query includes victim scope %q:\n%s",
				forbidden,
				query,
			)
		}
	}
	if len(args) != 6 ||
		!reflect.DeepEqual(args[1], scope.Characters) ||
		!reflect.DeepEqual(args[2], scope.Corporations) ||
		!reflect.DeepEqual(args[3], scope.Alliances) ||
		args[4] != 65 ||
		args[5] != 20 {
		t.Fatalf("most-valuable args = %#v", args)
	}
}

func TestDomainStatisticsTypesMatchFrontendContract(t *testing.T) {
	for _, dataType := range []string{
		"characters", "corporations", "alliances",
		"ships", "systems", "regions",
		"isk_destroyers_chars", "isk_destroyers_corps",
		"isk_destroyers_alliances", "solo_killers", "top_points",
		"dangerous_systems", "deadliest_regions", "most_used_ships",
		"most_destroyed_ships", "biggest_losers",
	} {
		if !hasDomainStatisticsType(dataType) {
			t.Errorf("frontend data type %q is missing", dataType)
		}
	}
	if hasDomainStatisticsType("pirate_characters") {
		t.Fatal("global-only data type leaked into the domain contract")
	}
}

func TestDomainRoutesRejectMainSiteContextBeforeDatabase(t *testing.T) {
	router := chi.NewRouter()
	a := humachi.New(
		router,
		huma.DefaultConfig("domain-killboard-test", "test"),
	)
	registerDomainKillboardRoutes(a, Options{})

	request := httptest.NewRequest(
		http.MethodGet,
		"http://eve-kill.com/custom/killlist",
		nil,
	)
	request.Header.Set("X-Forwarded-Host", "red.eve-kill.com")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest ||
		!strings.Contains(
			response.Body.String(),
			"This endpoint requires a custom domain context",
		) {
		t.Fatalf(
			"main-site domain route = %d %s",
			response.Code,
			response.Body.String(),
		)
	}
}
