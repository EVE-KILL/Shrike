package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/eve-kill/shrike/internal/battle"
	"github.com/go-chi/chi/v5"
)

func TestLegacyRequestHostUsesAuthoritativeHumaHost(t *testing.T) {
	router := chi.NewRouter()
	a := humachi.New(router, huma.DefaultConfig("host-test", "test"))
	var got string
	registerLegacy(a, huma.Operation{
		OperationID: "host-test",
		Method:      http.MethodGet,
		Path:        "/host",
	}, func(_ context.Context, req *legacyRequest) (legacyPayload, error) {
		got = legacyRequestHost(req)
		return jsonPayload(map[string]bool{"ok": true}), nil
	})

	request := httptest.NewRequest(http.MethodGet, "http://tenant.example/host", nil)
	request.Host = "real-tenant.example"
	request.Header.Set("X-Forwarded-Host", "spoofed.example")
	router.ServeHTTP(httptest.NewRecorder(), request)

	if got != "real-tenant.example" {
		t.Fatalf("legacy host = %q, want authoritative Request.Host", got)
	}
}

func TestConflictRoutesRegisterBattleWarAndGeneratorSurface(t *testing.T) {
	a := humachi.New(
		chi.NewRouter(),
		huma.DefaultConfig("conflict-test", "test"),
	)
	registerConflictRoutes(a, Options{})

	gets := []string{
		"/conflicts/wars",
		"/conflicts/battles",
		"/wars/stats",
		"/wars/eligible",
		"/war/{id}",
		"/war/{id}/dashboard",
		"/war/{id}/stats",
		"/war/{id}/members",
		"/war/{id}/intel",
		"/war/{id}/killlist",
		"/battle/{id}",
		"/battle/killmail/{id}",
		"/battle/{id}/composition",
		"/battle/{id}/intel",
		"/battle/{id}/killlist",
		"/battle/{id}/most-valuable",
		"/battle/{id}/timeline",
		"/battle/killmail/{id}/composition",
		"/battle/killmail/{id}/intel",
		"/battle/killmail/{id}/killlist",
		"/battle/killmail/{id}/most-valuable",
		"/battle/killmail/{id}/timeline",
	}
	for _, path := range gets {
		item := a.OpenAPI().Paths[path]
		if item == nil || item.Get == nil {
			t.Errorf("GET %s was not registered", path)
		}
	}
	for _, path := range []string{
		"/battle/generator/entities",
		"/battle/generator/preview",
		"/battle/generator/save",
	} {
		item := a.OpenAPI().Paths[path]
		if item == nil || item.Post == nil {
			t.Errorf("POST %s was not registered", path)
		}
	}
	save := a.OpenAPI().Paths["/battle/generator/save"].Post
	if !reflect.DeepEqual(
		save.Security,
		[]map[string][]string{{"eveSession": {}}},
	) {
		t.Fatalf("save security = %#v", save.Security)
	}
}

func TestParseConflictGeneratorWindowValidatesAndNormalizes(t *testing.T) {
	systems, start, end, err := parseConflictGeneratorWindow(
		conflictBattleGeneratorWindow{
			SystemIDs: []int32{30000142, 30000144},
			StartTime: "2026-07-01T10:00:00Z",
			EndTime:   "2026-07-01T12:00:00Z",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(systems, []int32{30000142, 30000144}) {
		t.Fatalf("systems = %v", systems)
	}
	if !start.Equal(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)) ||
		!end.Equal(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("window = %s - %s", start, end)
	}

	for name, body := range map[string]conflictBattleGeneratorWindow{
		"duplicate systems": {
			SystemIDs: []int32{30000142, 30000142},
			StartTime: "2026-07-01T10:00:00Z",
			EndTime:   "2026-07-01T12:00:00Z",
		},
		"reversed": {
			SystemIDs: []int32{30000142},
			StartTime: "2026-07-01T12:00:00Z",
			EndTime:   "2026-07-01T10:00:00Z",
		},
		"invalid time": {
			SystemIDs: []int32{30000142},
			StartTime: "yesterdayish",
			EndTime:   "2026-07-01T10:00:00Z",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := parseConflictGeneratorWindow(body); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestBuildBattleCompositionMatchesShipAndGroupBuckets(t *testing.T) {
	assignment := battle.TeamAssignment{
		CorpTeam: map[int32]int{10: 0, 20: 1},
	}
	result := buildBattleComposition(
		[]map[string]any{
			{
				"character_id": int32(1), "corporation_id": int32(10),
				"ship_type_id": int32(100), "ship_name": "First",
				"ship_group_id": int32(30), "ship_group_name": "Battleship",
				"damage_done": int64(50), "damage_taken": int64(10),
				"deaths": int32(1), "isk_lost": float64(100),
			},
			{
				"character_id": int32(2), "corporation_id": int32(10),
				"ship_type_id": int32(100), "ship_name": "First",
				"ship_group_id": int32(30), "ship_group_name": "Battleship",
				"damage_done": int64(75), "damage_taken": int64(20),
				"deaths": int32(0), "isk_lost": float64(0),
			},
			{
				"character_id": int32(3), "corporation_id": int32(999),
				"ship_type_id": int32(200), "ship_group_id": int32(31),
			},
		},
		assignment,
		[]int{0, 1},
	)
	teams := result["teams"].([]map[string]any)
	if got := len(teams[0]["individuals"].([]map[string]any)); got != 2 {
		t.Fatalf("team 0 individuals = %d", got)
	}
	ships := teams[0]["by_ship"].([]map[string]any)
	if len(ships) != 1 ||
		conflictInt(ships[0], "count") != 2 ||
		conflictInt(ships[0], "losses") != 1 ||
		conflictInt(ships[0], "damage_done") != 125 {
		t.Fatalf("ship aggregate = %#v", ships)
	}
	if got := len(teams[1]["individuals"].([]map[string]any)); got != 0 {
		t.Fatalf("team 1 individuals = %d", got)
	}
}

func TestValidateConflictSaveTeamsRejectsCrossSideCorporation(t *testing.T) {
	teams := []conflictBattleSaveTeam{
		{Alliances: []conflictBattleSaveAlliance{{
			Corporations: []conflictBattleSaveCorporation{
				{CorporationID: 42},
			},
		}}},
		{Alliances: []conflictBattleSaveAlliance{{
			Corporations: []conflictBattleSaveCorporation{
				{CorporationID: 42},
			},
		}}},
	}
	if err := validateConflictSaveTeams(teams); err == nil {
		t.Fatal("expected duplicate-side corporation error")
	}
}

func TestConflictBattleAssignmentEntitiesDeduplicatesAlliances(t *testing.T) {
	corps, alliances := conflictBattleAssignmentEntities(
		battle.TeamAssignment{
			CorpTeam: map[int32]int{10: 0, 11: 0, 20: 1},
			CorpAlliance: map[int32]int32{
				10: 100, 11: 100, 20: 200,
			},
		},
		0,
	)
	if !reflect.DeepEqual(corps, []int32{10, 11}) ||
		!reflect.DeepEqual(alliances, []int32{100}) {
		t.Fatalf("entities = corps %v alliances %v", corps, alliances)
	}
}
