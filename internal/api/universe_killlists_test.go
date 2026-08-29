package api

import (
	"reflect"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/stats"
)

func TestLocationKilllistRouteInventoryUsesDomainNames(t *testing.T) {
	want := []locationKilllistRoute{
		{
			Name: "region", Canonical: "/universe/regions/{id}/killmails",
			Alias: "/region/{id}/killlist", Column: "region_id",
			EntityType: stats.EntityRegion,
		},
		{
			Name:      "constellation",
			Canonical: "/universe/constellations/{id}/killmails",
			Alias:     "/constellation/{id}/killlist",
			Column:    "constellation_id", EntityType: stats.EntitySystem,
			MemberQuery: `
			SELECT solar_system_id FROM solar_systems
			WHERE constellation_id = $2`,
		},
		{
			Name: "system", Canonical: "/universe/systems/{id}/killmails",
			Alias: "/system/{id}/killlist", Column: "solar_system_id",
			EntityType: stats.EntitySystem,
		},
	}
	if !reflect.DeepEqual(locationKilllistRoutes, want) {
		t.Fatalf("location killlist routes = %#v, want %#v",
			locationKilllistRoutes, want)
	}
	for _, route := range locationKilllistRoutes {
		if strings.Contains(strings.ToLower(route.Name), "private") ||
			strings.HasPrefix(route.Canonical, "/api/") {
			t.Errorf("route leaks access/transport naming: %#v", route)
		}
	}
}

func TestLocationStatsScopeUsesStableStatsEnums(t *testing.T) {
	where, args := locationStatsScope(locationKilllistRoutes[0], 10000002)
	if where != "entity_type = $1 AND entity_id = $2" ||
		!reflect.DeepEqual(args, []any{stats.EntityRegion, int64(10000002)}) {
		t.Errorf("region scope = %q, %#v", where, args)
	}

	where, args = locationStatsScope(locationKilllistRoutes[1], 20000020)
	if !strings.Contains(where, "constellation_id = $2") ||
		!reflect.DeepEqual(args, []any{stats.EntitySystem, int64(20000020)}) {
		t.Errorf("constellation scope = %q, %#v", where, args)
	}
}

func TestUniverseItemKilllistQueryMatchesVictimAndFittedSlots(t *testing.T) {
	after := int64(123456)
	query, args := universeItemKilllistQuery(34, &after, 50, true)
	for _, fragment := range []string{
		"WITH item_kills AS MATERIALIZED",
		"victim_ship_type_id = $1",
		"type_id = $1",
		"parent_index IS NULL",
		"flag_id BETWEEN 11 AND 34",
		"flag_id BETWEEN 92 AND 99",
		"flag_id BETWEEN 125 AND 132",
		"flag_id = 87",
		"killmail_id < $2",
		"LIMIT $3",
		"bounded_item_kills AS MATERIALIZED",
		"FROM item_kills",
		"JOIN killmails k ON k.killmail_id = item_kills.killmail_id",
		"FROM bounded_item_kills k",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("item query missing %q:\n%s", fragment, query)
		}
	}
	if !reflect.DeepEqual(args, []any{int64(34), after, 51}) {
		t.Errorf("item args = %#v", args)
	}
	if strings.Contains(query, "FROM killmails k\n\tLEFT JOIN LATERAL") {
		t.Errorf("item enrichment is not driven by the bounded relation:\n%s", query)
	}

	shipQuery, shipArgs := universeItemKilllistQuery(587, nil, 100, false)
	if !strings.Contains(shipQuery, "k.victim_ship_type_id = $1") ||
		strings.Contains(shipQuery, "killmail_items") ||
		!strings.Contains(shipQuery, "LIMIT $2") {
		t.Errorf("ship query is not victim-only:\n%s", shipQuery)
	}
	if !reflect.DeepEqual(shipArgs, []any{int64(587), 101}) {
		t.Errorf("ship args = %#v", shipArgs)
	}
}

func TestOptionalPositiveInt64RejectsDatabaseBreakingCursors(t *testing.T) {
	if value, err := optionalPositiveInt64(""); err != nil || value != nil {
		t.Errorf("empty cursor = %v, %v", value, err)
	}
	value, err := optionalPositiveInt64(" 123 ")
	if err != nil || value == nil || *value != 123 {
		t.Errorf("valid cursor = %v, %v", value, err)
	}
	for _, raw := range []string{"0", "-1", "1.5", "NaN", "9223372036854775808"} {
		if _, err := optionalPositiveInt64(raw); err == nil {
			t.Errorf("cursor %q was accepted", raw)
		}
	}
}
