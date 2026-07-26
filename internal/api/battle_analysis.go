package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/battle"
)

var conflictShipGroupRank = map[string]int{
	"Flag Cruiser": 110, "Command Ship": 100, "Force Auxiliary": 95,
	"Titan": 94, "Supercarrier": 93, "Lancer Dreadnought": 92,
	"Dreadnought": 91, "Carrier": 90, "Black Ops": 80, "Marauder": 79,
	"Battleship": 78, "Strategic Cruiser": 70, "Combat Battlecruiser": 65,
	"Command Destroyer": 64, "Heavy Assault Cruiser": 60, "Logistics": 59,
	"Recon Ship": 58, "Heavy Interdiction Cruiser": 57, "Cruiser": 50,
	"Battlecruiser": 49, "Tactical Destroyer": 45, "Assault Frigate": 40,
	"Interceptor": 39, "Interdictor": 38, "Stealth Bomber": 37,
	"Logistics Frigate": 36, "Electronic Attack Ship": 35, "Covert Ops": 34,
	"Destroyer": 30, "Frigate": 20, "Industrial": 15, "Mining Barge": 12,
	"Exhumer": 12, "Shuttle": 5, "Capsule": 1,
}

var conflictCapitalGroups = map[int64]struct{}{
	547: {}, 659: {}, 30: {}, 485: {}, 1538: {}, 4594: {},
}

var conflictLogisticsGroups = map[int64]struct{}{832: {}, 1527: {}}

func registerBattleAnalysisRoutes(a huma.API, opts Options) {
	type route struct {
		name, suffix, summary string
		ttl                   time.Duration
		handler               func(Options, bool) legacyHandler
	}
	routes := []route{
		{"composition", "composition", "Battle ship composition", time.Hour, battleCompositionHandler},
		{"intel", "intel", "Battle role intelligence", 5 * time.Minute, battleIntelHandler},
		{"killlist", "killlist", "Battle killmails", 2 * time.Minute, battleKilllistHandler},
		{"most-valuable", "most-valuable", "Most valuable battle losses", 5 * time.Minute, battleMostValuableHandler},
		{"timeline", "timeline", "Complete battle timeline", 2 * time.Minute, battleTimelineHandler},
	}
	for _, route := range routes {
		route := route
		registerConflictCachedGET(a, opts, huma.Operation{
			OperationID: "battle-report-" + route.name,
			Path:        "/battle/{id}/" + route.suffix,
			Summary:     route.summary,
			Tags:        []string{"battles"},
		}, route.ttl, route.handler(opts, false))
		registerConflictCachedGET(a, opts, huma.Operation{
			OperationID: "killmail-battle-" + route.name,
			Path:        "/battle/killmail/{id}/" + route.suffix,
			Summary:     route.summary + " detected around a killmail",
			Tags:        []string{"battles"},
		}, route.ttl, route.handler(opts, true))
	}
}

func resolveConflictBattleWindow(
	ctx context.Context,
	db Database,
	req *legacyRequest,
	killmailMode bool,
) (*conflictBattleWindow, error) {
	id, err := parseID(req.Param("id"))
	if err != nil || id <= 0 {
		return nil, apiError(http.StatusBadRequest, "Invalid ID")
	}
	if !killmailMode {
		if id > pgInt4Max {
			return nil, apiError(http.StatusBadRequest, "Invalid ID")
		}
		window, _, err := loadSavedConflictBattleWindow(ctx, db, int32(id))
		if err != nil {
			return nil, err
		}
		if window == nil {
			return nil, apiError(http.StatusNotFound, "Battle not found")
		}
		return window, nil
	}
	window, redirectID, err := detectConflictKillmailBattle(ctx, db, id)
	if err != nil {
		return nil, err
	}
	if redirectID != 0 {
		saved, _, loadErr := loadSavedConflictBattleWindow(ctx, db, redirectID)
		if loadErr != nil {
			return nil, loadErr
		}
		if saved == nil {
			return nil, apiError(http.StatusNotFound, "Battle not found")
		}
		return saved, nil
	}
	return window, nil
}

func battleKilllistHandler(opts Options, killmailMode bool) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		window, err := resolveConflictBattleWindow(
			ctx, opts.DB, req, killmailMode,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return loadConflictKilllist(
			ctx, opts.DB, req.Query,
			[]string{
				"k.solar_system_id = ANY($1::int[])",
				"k.killmail_time >= $2",
				"k.killmail_time <= $3",
			},
			[]any{window.SystemIDs, window.Start, window.End},
		)
	}
}

func battleTimelineHandler(opts Options, killmailMode bool) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		window, err := resolveConflictBattleWindow(
			ctx, opts.DB, req, killmailMode,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, err := queryMaps(ctx, opts.DB,
			conflictKilllistSelect+`
			WHERE k.solar_system_id = ANY($1::int[])
			  AND k.killmail_time >= $2 AND k.killmail_time <= $3
			ORDER BY k.killmail_time, k.killmail_id`,
			window.SystemIDs, window.Start, window.End,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if err := enrichConflictKilllist(ctx, opts.DB, rows); err != nil {
			return legacyPayload{}, err
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		return jsonPayload(map[string]any{"kills": rows}), nil
	}
}

func battleMostValuableHandler(
	opts Options,
	killmailMode bool,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		window, err := resolveConflictBattleWindow(
			ctx, opts.DB, req, killmailMode,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		limit := parseConflictBoundedInt(req.Query, "limit", 8, 1, 32)
		dataType := strings.TrimSpace(req.Query.Get("dataType"))
		if dataType == "" {
			dataType = "most_valuable_kills"
		}
		category := 0
		switch dataType {
		case "most_valuable_kills":
		case "most_valuable_ships":
			category = conflictShipCategory
		case "most_valuable_structures":
			category = conflictStructureCategory
		default:
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid dataType",
			)
		}
		where := []string{
			"k.solar_system_id = ANY($1::int[])",
			"k.killmail_time >= $2",
			"k.killmail_time <= $3",
		}
		args := []any{window.SystemIDs, window.Start, window.End}
		if category != 0 {
			args = append(args, category)
			where = append(where, fmt.Sprintf(`
				EXISTS (
					SELECT 1 FROM inv_groups category_group
					WHERE category_group.group_id = k.victim_ship_group_id
					  AND category_group.category_id = $%d
				)`, len(args)))
		}
		if raw := strings.TrimSpace(req.Query.Get("team")); raw != "" {
			team, parseErr := strconv.Atoi(raw)
			if parseErr != nil || (team != 0 && team != 1) {
				return legacyPayload{}, apiError(
					http.StatusBadRequest, "Invalid team",
				)
			}
			other := 1 - team
			corps, alliances := conflictBattleAssignmentEntities(
				window.Assignment, other,
			)
			if len(corps) == 0 && len(alliances) == 0 {
				return jsonPayload(map[string]any{"entries": []map[string]any{}}), nil
			}
			args = append(args, corps, alliances)
			where = append(where, fmt.Sprintf(
				"(k.victim_corporation_id = ANY($%d::int[]) OR "+
					"k.victim_alliance_id = ANY($%d::int[]))",
				len(args)-1, len(args),
			))
		}
		candidateLimit := limit * 6
		if candidateLimit < 32 {
			candidateLimit = 32
		}
		args = append(args, candidateLimit)
		rows, err := queryMaps(ctx, opts.DB, `
			SELECT k.killmail_id, k.killmail_hash, k.killmail_time,
			       k.victim_ship_type_id AS ship_type_id,
			       COALESCE(type.name, 'Unknown') AS ship_name,
			       COALESCE(k.total_value, 0)::double precision AS total_value,
			       k.victim_character_id, character.name AS victim_character_name,
			       corporation.name AS victim_corporation_name,
			       alliance.name AS victim_alliance_name
			FROM killmails k
			INNER JOIN inv_types type ON type.type_id = k.victim_ship_type_id
			LEFT JOIN characters character
			  ON character.character_id = k.victim_character_id
			LEFT JOIN corporations corporation
			  ON corporation.corporation_id = k.victim_corporation_id
			LEFT JOIN alliances alliance
			  ON alliance.alliance_id = k.victim_alliance_id
			WHERE `+strings.Join(where, " AND ")+`
			ORDER BY k.total_value DESC NULLS LAST, k.killmail_id DESC
			LIMIT $`+strconv.Itoa(len(args)),
			args...,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		typeIDs := make([]int32, 0, len(rows))
		for _, row := range rows {
			if id := int32(conflictInt(row, "ship_type_id")); id != 0 {
				typeIDs = append(typeIDs, id)
			}
		}
		deltas, err := loadConflictPriceDeltas(
			ctx, opts.DB, typeIDs, window.End,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		for _, row := range rows {
			id := int32(conflictInt(row, "ship_type_id"))
			row["total_value"] = conflictFloat(row, "total_value") + deltas[id]
		}
		sort.Slice(rows, func(i, j int) bool {
			left := conflictFloat(rows[i], "total_value")
			right := conflictFloat(rows[j], "total_value")
			if left != right {
				return left > right
			}
			return conflictInt(rows[i], "killmail_id") >
				conflictInt(rows[j], "killmail_id")
		})
		if len(rows) > limit {
			rows = rows[:limit]
		}
		for _, row := range rows {
			delete(row, "victim_corporation_id")
		}
		return jsonPayload(map[string]any{"entries": rows}), nil
	}
}

func loadConflictPriceDeltas(
	ctx context.Context,
	db Database,
	typeIDs []int32,
	at time.Time,
) (map[int32]float64, error) {
	typeIDs = uniqueConflictIDs(typeIDs)
	if len(typeIDs) == 0 {
		return map[int32]float64{}, nil
	}
	rows, err := queryMapsConcurrent(ctx, db,
		databaseQuery{
			SQL: `SELECT DISTINCT ON (type_id) type_id, price
			      FROM custom_prices
			      WHERE type_id = ANY($1::int[])
			      ORDER BY type_id, date DESC`,
			Args: []any{typeIDs},
		},
		databaseQuery{
			SQL: `SELECT DISTINCT ON (type_id) type_id, average
			      FROM prices
			      WHERE type_id = ANY($1::int[])
			        AND region_id = 10000002
			        AND date <= $2::date
			      ORDER BY type_id, date DESC`,
			Args: []any{typeIDs, at},
		},
	)
	if err != nil {
		return nil, err
	}
	market := map[int32]float64{}
	for _, row := range rows[1] {
		market[int32(conflictInt(row, "type_id"))] =
			conflictFloat(row, "average")
	}
	deltas := map[int32]float64{}
	for _, row := range rows[0] {
		id := int32(conflictInt(row, "type_id"))
		delta := conflictFloat(row, "price") - market[id]
		if delta != 0 {
			deltas[id] = delta
		}
	}
	return deltas, nil
}

func conflictBattleAssignmentEntities(
	assignment battle.TeamAssignment,
	team int,
) ([]int32, []int32) {
	corporations := []int32{}
	alliances := []int32{}
	for corporationID, assigned := range assignment.CorpTeam {
		if assigned != team {
			continue
		}
		corporations = append(corporations, corporationID)
		if allianceID := assignment.CorpAlliance[corporationID]; allianceID != 0 {
			alliances = append(alliances, allianceID)
		}
	}
	corporations = uniqueConflictIDs(corporations)
	alliances = uniqueConflictIDs(alliances)
	sort.Slice(corporations, func(i, j int) bool {
		return corporations[i] < corporations[j]
	})
	sort.Slice(alliances, func(i, j int) bool {
		return alliances[i] < alliances[j]
	})
	return corporations, alliances
}

func battleCompositionHandler(
	opts Options,
	killmailMode bool,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		window, err := resolveConflictBattleWindow(
			ctx, opts.DB, req, killmailMode,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, err := queryMaps(ctx, opts.DB, `
			WITH battle_kills AS MATERIALIZED (
				SELECT killmail_id, victim_character_id,
				       victim_corporation_id, victim_alliance_id,
				       victim_ship_type_id, victim_ship_group_id,
				       total_value, victim_damage_taken
				FROM killmails
				WHERE solar_system_id = ANY($1::int[])
				  AND killmail_time >= $2 AND killmail_time <= $3
			),
			attackers_agg AS (
				SELECT attacker.character_id, attacker.corporation_id,
				       attacker.alliance_id, attacker.ship_type_id,
				       attacker.ship_group_id,
				       SUM(COALESCE(attacker.damage_done, 0))::bigint
				           AS damage_done
				FROM killmail_attackers attacker
				JOIN battle_kills kill
				  ON kill.killmail_id = attacker.killmail_id
				WHERE attacker.character_id IS NOT NULL
				  AND attacker.ship_type_id IS NOT NULL
				  AND attacker.killmail_time >= $2
				  AND attacker.killmail_time <= $3
				GROUP BY attacker.character_id, attacker.corporation_id,
				         attacker.alliance_id, attacker.ship_type_id,
				         attacker.ship_group_id
			),
			losses_agg AS (
				SELECT victim_character_id AS character_id,
				       victim_corporation_id AS corporation_id,
				       victim_alliance_id AS alliance_id,
				       victim_ship_type_id AS ship_type_id,
				       victim_ship_group_id AS ship_group_id,
				       COUNT(*)::int AS deaths,
				       SUM(COALESCE(total_value, 0))::double precision AS isk_lost,
				       SUM(COALESCE(victim_damage_taken, 0))::bigint
				           AS damage_taken
				FROM battle_kills
				WHERE victim_character_id IS NOT NULL
				  AND victim_ship_type_id IS NOT NULL
				GROUP BY victim_character_id, victim_corporation_id,
				         victim_alliance_id, victim_ship_type_id,
				         victim_ship_group_id
			),
			combined AS (
				SELECT COALESCE(attacker.character_id, loss.character_id)
				           AS character_id,
				       COALESCE(attacker.corporation_id, loss.corporation_id)
				           AS corporation_id,
				       COALESCE(attacker.alliance_id, loss.alliance_id)
				           AS alliance_id,
				       COALESCE(attacker.ship_type_id, loss.ship_type_id)
				           AS ship_type_id,
				       COALESCE(attacker.ship_group_id, loss.ship_group_id)
				           AS ship_group_id,
				       COALESCE(attacker.damage_done, 0)::bigint AS damage_done,
				       COALESCE(loss.damage_taken, 0)::bigint AS damage_taken,
				       COALESCE(loss.deaths, 0)::int AS deaths,
				       COALESCE(loss.isk_lost, 0)::double precision AS isk_lost
				FROM attackers_agg attacker
				FULL OUTER JOIN losses_agg loss
				  ON loss.character_id = attacker.character_id
				 AND loss.ship_type_id = attacker.ship_type_id
			)
			SELECT combined.*, character.name AS character_name,
			       corporation.name AS corporation_name,
			       alliance.name AS alliance_name, type.name AS ship_name,
			       ship_group.name AS ship_group_name
			FROM combined
			LEFT JOIN characters character
			  ON character.character_id = combined.character_id
			LEFT JOIN corporations corporation
			  ON corporation.corporation_id = combined.corporation_id
			LEFT JOIN alliances alliance
			  ON alliance.alliance_id = combined.alliance_id
			LEFT JOIN inv_types type ON type.type_id = combined.ship_type_id
			LEFT JOIN inv_groups ship_group
			  ON ship_group.group_id = combined.ship_group_id`,
			window.SystemIDs, window.Start, window.End,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(buildBattleComposition(
			rows, window.Assignment, window.TeamIndices,
		)), nil
	}
}

func buildBattleComposition(
	rows []map[string]any,
	assignment battle.TeamAssignment,
	indices []int,
) map[string]any {
	teamRows := map[int][]map[string]any{}
	for _, row := range rows {
		corpID := int32(conflictInt(row, "corporation_id"))
		team, ok := assignment.CorpTeam[corpID]
		if !ok {
			continue
		}
		groupName := conflictString(row, "ship_group_name")
		row["team_index"] = team
		row["rank"] = conflictShipGroupRank[groupName]
		row["isk_lost"] = conflictFloat(row, "isk_lost")
		row["damage_done"] = conflictInt(row, "damage_done")
		row["damage_taken"] = conflictInt(row, "damage_taken")
		row["deaths"] = conflictInt(row, "deaths")
		teamRows[team] = append(teamRows[team], row)
	}
	if len(indices) == 0 {
		indices = []int{0, 1}
	}
	indices = append([]int(nil), indices...)
	sort.Ints(indices)
	teams := make([]map[string]any, 0, len(indices))
	for _, index := range indices {
		individuals := teamRows[index]
		sort.Slice(individuals, func(i, j int) bool {
			if conflictInt(individuals[i], "rank") != conflictInt(individuals[j], "rank") {
				return conflictInt(individuals[i], "rank") >
					conflictInt(individuals[j], "rank")
			}
			if conflictFloat(individuals[i], "isk_lost") !=
				conflictFloat(individuals[j], "isk_lost") {
				return conflictFloat(individuals[i], "isk_lost") >
					conflictFloat(individuals[j], "isk_lost")
			}
			if conflictFloat(individuals[i], "damage_done") !=
				conflictFloat(individuals[j], "damage_done") {
				return conflictFloat(individuals[i], "damage_done") >
					conflictFloat(individuals[j], "damage_done")
			}
			left := conflictString(individuals[i], "character_name")
			right := conflictString(individuals[j], "character_name")
			if left != right {
				return left < right
			}
			return conflictInt(individuals[i], "character_id") <
				conflictInt(individuals[j], "character_id")
		})
		byShip := aggregateBattleComposition(individuals, true)
		byGroup := aggregateBattleComposition(individuals, false)
		teams = append(teams, map[string]any{
			"team_index": index, "individuals": individuals,
			"by_ship": byShip, "by_group": byGroup,
		})
	}
	return map[string]any{"teams": teams, "team_count": len(teams)}
}

func aggregateBattleComposition(
	individuals []map[string]any,
	byShip bool,
) []map[string]any {
	keyName, nameName := "ship_group_id", "ship_group_name"
	if byShip {
		keyName, nameName = "ship_type_id", "ship_name"
	}
	aggregates := map[int64]map[string]any{}
	for _, individual := range individuals {
		id, ok := int64Value(individual[keyName])
		if !ok || id == 0 {
			continue
		}
		current := aggregates[id]
		if current == nil {
			current = map[string]any{
				"key":   strconv.FormatInt(id, 10),
				"name":  conflictString(individual, nameName),
				keyName: id,
				"count": int64(0), "losses": int64(0),
				"isk_lost": float64(0), "damage_done": int64(0),
				"damage_taken": int64(0),
				"rank":         conflictInt(individual, "rank"),
			}
			if byShip {
				current["ship_group_id"] = conflictNullableID(
					individual, "ship_group_id",
				)
			}
			if current["name"] == "" {
				current["name"] = "Unknown"
			}
			aggregates[id] = current
		}
		current["count"] = conflictInt(current, "count") + 1
		current["losses"] = conflictInt(current, "losses") +
			conflictInt(individual, "deaths")
		current["isk_lost"] = conflictFloat(current, "isk_lost") +
			conflictFloat(individual, "isk_lost")
		current["damage_done"] = conflictInt(current, "damage_done") +
			conflictInt(individual, "damage_done")
		current["damage_taken"] = conflictInt(current, "damage_taken") +
			conflictInt(individual, "damage_taken")
	}
	out := make([]map[string]any, 0, len(aggregates))
	for _, row := range aggregates {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if conflictInt(out[i], "rank") != conflictInt(out[j], "rank") {
			return conflictInt(out[i], "rank") > conflictInt(out[j], "rank")
		}
		if conflictInt(out[i], "count") != conflictInt(out[j], "count") {
			return conflictInt(out[i], "count") > conflictInt(out[j], "count")
		}
		if conflictFloat(out[i], "isk_lost") !=
			conflictFloat(out[j], "isk_lost") {
			return conflictFloat(out[i], "isk_lost") >
				conflictFloat(out[j], "isk_lost")
		}
		return conflictString(out[i], "key") <
			conflictString(out[j], "key")
	})
	return out
}

func battleIntelHandler(opts Options, killmailMode bool) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		window, err := resolveConflictBattleWindow(
			ctx, opts.DB, req, killmailMode,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, err := loadBattleIntelPilots(
			ctx, opts.DB, *window,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		graphFCs := loadBattleGraphFCs(ctx, opts, rows)
		return jsonPayload(formatBattleIntel(
			rows, window.Assignment, window.TeamIndices, graphFCs,
		)), nil
	}
}

func loadBattleIntelPilots(
	ctx context.Context,
	db Database,
	window conflictBattleWindow,
) ([]map[string]any, error) {
	return queryMaps(ctx, db, `
		WITH battle_kills AS MATERIALIZED (
			SELECT killmail_id, victim_character_id,
			       victim_corporation_id, victim_alliance_id,
			       victim_ship_type_id, victim_ship_group_id,
			       victim_damage_taken
			FROM killmails
			WHERE solar_system_id = ANY($1::int[])
			  AND killmail_time >= $2 AND killmail_time <= $3
		),
		attacker_ships AS (
			SELECT attacker.character_id, attacker.corporation_id,
			       attacker.alliance_id, attacker.ship_type_id,
			       attacker.ship_group_id,
			       SUM(COALESCE(attacker.damage_done, 0))::bigint
			           AS ship_damage
			FROM killmail_attackers attacker
			JOIN battle_kills kill ON kill.killmail_id = attacker.killmail_id
			WHERE attacker.character_id IS NOT NULL
			  AND attacker.ship_type_id IS NOT NULL
			  AND attacker.killmail_time >= $2
			  AND attacker.killmail_time <= $3
			GROUP BY attacker.character_id, attacker.corporation_id,
			         attacker.alliance_id, attacker.ship_type_id,
			         attacker.ship_group_id
		),
		attackers AS (
			SELECT DISTINCT ON (character_id)
			       character_id, corporation_id, alliance_id,
			       ship_type_id, ship_group_id, ship_damage,
			       SUM(ship_damage) OVER (PARTITION BY character_id)
			           AS damage_done
			FROM attacker_ships
			ORDER BY character_id, ship_damage DESC, ship_type_id
		),
		victims AS (
			SELECT DISTINCT ON (victim_character_id)
			       victim_character_id AS character_id,
			       victim_corporation_id AS corporation_id,
			       victim_alliance_id AS alliance_id,
			       victim_ship_type_id AS ship_type_id,
			       victim_ship_group_id AS ship_group_id,
			       0::bigint AS damage_done
			FROM battle_kills
			WHERE victim_character_id IS NOT NULL
			  AND victim_ship_type_id IS NOT NULL
			  AND NOT EXISTS (
				SELECT 1 FROM attackers
				WHERE attackers.character_id =
				      battle_kills.victim_character_id
			  )
			ORDER BY victim_character_id,
			         victim_damage_taken DESC NULLS LAST
		),
		participants AS (
			SELECT character_id, corporation_id, alliance_id,
			       ship_type_id, ship_group_id, damage_done
			FROM attackers
			UNION ALL
			SELECT * FROM victims
		)
		SELECT participant.*, character.name AS character_name,
		       corporation.name AS corporation_name,
		       alliance.name AS alliance_name, type.name AS ship_name,
		       ship_group.name AS ship_group_name,
		       EXISTS (
				SELECT 1 FROM battle_kills death
				WHERE death.victim_character_id = participant.character_id
		       ) AS died
		FROM participants participant
		LEFT JOIN characters character
		  ON character.character_id = participant.character_id
		LEFT JOIN corporations corporation
		  ON corporation.corporation_id = participant.corporation_id
		LEFT JOIN alliances alliance
		  ON alliance.alliance_id = participant.alliance_id
		LEFT JOIN inv_types type ON type.type_id = participant.ship_type_id
		LEFT JOIN inv_groups ship_group
		  ON ship_group.group_id = participant.ship_group_id`,
		window.SystemIDs, window.Start, window.End,
	)
}

func loadBattleGraphFCs(
	ctx context.Context,
	opts Options,
	rows []map[string]any,
) map[int64]struct{} {
	result := map[int64]struct{}{}
	if opts.Graph == nil {
		return result
	}
	ids := make([]int64, 0, len(rows))
	seen := map[int64]struct{}{}
	for _, row := range rows {
		id := conflictInt(row, "character_id")
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	cutoff := time.Now().UTC().
		Add(-90 * 24 * time.Hour).
		Format("2006-01-02T15:04:05.000Z")
	for start := 0; start < len(ids); start += 5000 {
		end := start + 5000
		if end > len(ids) {
			end = len(ids)
		}
		graphRows, err := opts.Graph.Read(ctx, `
			MATCH (character:Character)
			WHERE character.id IN $ids
			  AND character.last_fc_seen > $cutoff
			RETURN character.id AS id`,
			map[string]any{"ids": ids[start:end], "cutoff": cutoff},
		)
		if err != nil {
			// Graph intelligence is an enhancement; PostgreSQL battle data
			// remains usable while Memgraph is unavailable.
			return result
		}
		for _, row := range graphRows {
			if id := conflictInt(row, "id"); id != 0 {
				result[id] = struct{}{}
			}
		}
	}
	return result
}

func formatBattleIntel(
	rows []map[string]any,
	assignment battle.TeamAssignment,
	indices []int,
	graphFCs map[int64]struct{},
) map[string]any {
	allianceTeam := battleAllianceTeams(assignment)
	sideOf := func(corpID, allianceID int32) (int, bool) {
		if team, ok := assignment.CorpTeam[corpID]; corpID != 0 && ok {
			return team, true
		}
		team, ok := allianceTeam[allianceID]
		return team, allianceID != 0 && ok
	}
	type bucket struct {
		all  map[int64]map[string]any
		fcs  map[int64]map[string]any
		logi map[int64]map[string]any
		caps map[int64]map[string]any
	}
	buckets := map[int]*bucket{}
	for _, index := range indices {
		buckets[index] = &bucket{
			all:  map[int64]map[string]any{},
			fcs:  map[int64]map[string]any{},
			logi: map[int64]map[string]any{},
			caps: map[int64]map[string]any{},
		}
	}
	for _, row := range rows {
		team, ok := sideOf(
			int32(conflictInt(row, "corporation_id")),
			int32(conflictInt(row, "alliance_id")),
		)
		if !ok || buckets[team] == nil {
			continue
		}
		id := conflictInt(row, "character_id")
		pilot := map[string]any{
			"character_id":     id,
			"character_name":   conflictNullableString(row, "character_name"),
			"corporation_id":   conflictNullableID(row, "corporation_id"),
			"corporation_name": conflictNullableString(row, "corporation_name"),
			"alliance_id":      conflictNullableID(row, "alliance_id"),
			"alliance_name":    conflictNullableString(row, "alliance_name"),
			"ship_type_id":     conflictNullableID(row, "ship_type_id"),
			"ship_name":        conflictNullableString(row, "ship_name"),
			"ship_group_id":    conflictNullableID(row, "ship_group_id"),
			"ship_group_name":  conflictNullableString(row, "ship_group_name"),
			"damage_done":      conflictInt(row, "damage_done"),
			"died":             boolValueOrFalse(row["died"]),
		}
		buckets[team].all[id] = pilot
		groupID := conflictInt(row, "ship_group_id")
		if groupID == 1972 {
			buckets[team].fcs[id] = pilot
		}
		if _, exists := conflictLogisticsGroups[groupID]; exists {
			buckets[team].logi[id] = pilot
		}
		if _, exists := conflictCapitalGroups[groupID]; exists {
			buckets[team].caps[id] = pilot
		}
	}
	indices = append([]int(nil), indices...)
	sort.Ints(indices)
	teams := make([]map[string]any, 0, len(indices))
	for _, index := range indices {
		current := buckets[index]
		if current == nil {
			current = &bucket{
				all:  map[int64]map[string]any{},
				fcs:  map[int64]map[string]any{},
				logi: map[int64]map[string]any{},
				caps: map[int64]map[string]any{},
			}
		}
		fcs := []map[string]any{}
		for _, pilot := range current.fcs {
			copy := cloneConflictMap(pilot)
			copy["confirmed"] = true
			fcs = append(fcs, copy)
		}
		for id, pilot := range current.all {
			if _, confirmed := current.fcs[id]; confirmed {
				continue
			}
			if _, logistics := current.logi[id]; logistics {
				continue
			}
			if _, known := graphFCs[id]; !known {
				continue
			}
			copy := cloneConflictMap(pilot)
			copy["confirmed"] = false
			fcs = append(fcs, copy)
		}
		sort.Slice(fcs, func(i, j int) bool {
			left := boolValueOrFalse(fcs[i]["confirmed"])
			right := boolValueOrFalse(fcs[j]["confirmed"])
			if left != right {
				return left
			}
			if conflictInt(fcs[i], "damage_done") !=
				conflictInt(fcs[j], "damage_done") {
				return conflictInt(fcs[i], "damage_done") >
					conflictInt(fcs[j], "damage_done")
			}
			return conflictInt(fcs[i], "character_id") <
				conflictInt(fcs[j], "character_id")
		})
		logistics := conflictSortedPilots(current.logi)
		capitals := conflictSortedPilots(current.caps)
		teams = append(teams, map[string]any{
			"team_index": index, "fcs": fcs,
			"logistics": logistics, "capitals": capitals,
		})
	}
	return map[string]any{"teams": teams}
}

func cloneConflictMap(value map[string]any) map[string]any {
	out := make(map[string]any, len(value)+1)
	for key, item := range value {
		out[key] = item
	}
	return out
}

func conflictSortedPilots(values map[int64]map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if conflictInt(out[i], "damage_done") !=
			conflictInt(out[j], "damage_done") {
			return conflictInt(out[i], "damage_done") >
				conflictInt(out[j], "damage_done")
		}
		return conflictInt(out[i], "character_id") <
			conflictInt(out[j], "character_id")
	})
	return out
}
