package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/battle"
	"golang.org/x/sync/errgroup"
)

const conflictOnTheFlyMinimumKills = 2

type conflictBattleWindow struct {
	BattleID    int32
	KillmailID  int64
	SystemIDs   []int32
	SystemID    int32
	RegionID    int32
	Start       time.Time
	End         time.Time
	IsCustom    bool
	Assignment  battle.TeamAssignment
	TeamIndices []int
	Kills       []battle.Killmail
	Attackers   map[int64][]battle.Attacker
}

type conflictBattleNames struct {
	Systems   map[int32]map[string]any
	Region    map[string]any
	Corps     map[int32]map[string]any
	Alliances map[int32]string
}

func registerConflictBattleRoutes(a huma.API, opts Options) {
	// /battles remains the established public list. The dashboard response has
	// different pagination, filters, domain scoping, and team summaries, so it
	// gets a canonical domain route rather than overloading that contract.
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "conflict-battles",
		Path:        "/conflicts/battles",
		Summary:     "Battle dashboard listing",
		Tags:        []string{"battles"},
	}, time.Minute, conflictBattlesHandler(opts))
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "battle-report",
		Path:        "/battle/{id}",
		Summary:     "Battle report",
		Tags:        []string{"battles"},
	}, 2*time.Minute, battleDetailHandler(opts))
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "killmail-battle-report",
		Path:        "/battle/killmail/{id}",
		Summary:     "Detect a battle around a killmail",
		Tags:        []string{"battles"},
	}, 2*time.Minute, killmailBattleHandler(opts))
}

func conflictBattlesHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		page := parseConflictBoundedInt(
			req.Query, "page", conflictDefaultPage, 1, conflictMaximumPage,
		)
		limit := parseConflictBoundedInt(req.Query, "limit", 50, 10, 50)

		domain, err := loadConflictDomainScope(ctx, opts.DB, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if domain != nil {
			if err := resolveConflictScopeCharacters(ctx, opts.DB, domain); err != nil {
				return legacyPayload{}, err
			}
			if domain.empty() {
				return jsonPayload(map[string]any{
					"battles": []map[string]any{}, "years": []map[string]any{},
					"page": page, "limit": limit,
				}), nil
			}
		}

		where := []string{"TRUE"}
		args := []any{}
		if domain != nil {
			args = append(args, domain.Corporations, domain.Alliances)
			where = append(where, fmt.Sprintf(`
				EXISTS (
					SELECT 1
					FROM battle_teams scope_team
					JOIN battle_team_members scope_member
					  ON scope_member.battle_team_id = scope_team.id
					WHERE scope_team.battle_id = b.battle_id
					  AND (
						scope_member.corporation_id = ANY($%d::int[])
						OR scope_member.alliance_id = ANY($%d::int[])
					  )
				)`, len(args)-1, len(args)))
		}
		domainWhere := append([]string(nil), where...)
		domainArgs := append([]any(nil), args...)

		if raw := strings.TrimSpace(req.Query.Get("year")); raw != "" {
			year, parseErr := strconv.Atoi(raw)
			if parseErr != nil || year < 2003 || year > 9999 {
				return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid year")
			}
			args = append(args, time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC))
			where = append(where, fmt.Sprintf("b.start_time >= $%d", len(args)))
			args = append(args, time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC))
			where = append(where, fmt.Sprintf("b.start_time < $%d", len(args)))
		}
		for _, filter := range []struct {
			query  string
			column string
		}{
			{"minKills", "b.kill_count"},
			{"regionId", "b.region_id"},
			{"systemId", "b.solar_system_id"},
		} {
			value, parseErr := parseConflictOptionalPositiveInt64(req.Query, filter.query)
			if parseErr != nil {
				return legacyPayload{}, parseErr
			}
			if value != nil {
				args = append(args, *value)
				operator := "="
				if filter.query == "minKills" {
					operator = ">="
				}
				where = append(where,
					fmt.Sprintf("%s %s $%d", filter.column, operator, len(args)))
			}
		}
		if raw := strings.TrimSpace(req.Query.Get("minIsk")); raw != "" {
			value, parseErr := strconv.ParseFloat(raw, 64)
			if parseErr != nil || value < 0 {
				return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid minIsk")
			}
			args = append(args, value)
			where = append(where,
				fmt.Sprintf("b.total_isk_destroyed >= $%d", len(args)))
		}
		if value, parseErr := parseConflictOptionalID(req.Query, "constellationId"); parseErr != nil {
			return legacyPayload{}, parseErr
		} else if value != nil {
			args = append(args, *value)
			where = append(where, fmt.Sprintf(
				`EXISTS (
					SELECT 1 FROM solar_systems scope_system
					WHERE scope_system.solar_system_id = b.solar_system_id
					  AND scope_system.constellation_id = $%d
				)`, len(args)))
		}
		if raw := strings.TrimSpace(req.Query.Get("custom")); raw != "" {
			var value bool
			switch raw {
			case "true":
				value = true
			case "false":
				value = false
			default:
				return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid custom")
			}
			args = append(args, value)
			where = append(where, fmt.Sprintf("b.is_custom = $%d", len(args)))
		}

		allianceID, err := parseConflictOptionalID(req.Query, "allianceId")
		if err != nil {
			return legacyPayload{}, err
		}
		corporationID, err := parseConflictOptionalID(req.Query, "corporationId")
		if err != nil {
			return legacyPayload{}, err
		}
		characterID, err := parseConflictOptionalID(req.Query, "characterId")
		if err != nil {
			return legacyPayload{}, err
		}
		if characterID != nil && corporationID == nil && allianceID == nil {
			row, queryErr := queryMap(ctx, opts.DB,
				`SELECT corporation_id FROM characters WHERE character_id = $1`,
				*characterID,
			)
			if queryErr != nil {
				return legacyPayload{}, queryErr
			}
			if row == nil {
				return emptyConflictBattleList(ctx, opts.DB, domainWhere, domainArgs, page, limit)
			}
			resolved, ok := int64Value(row["corporation_id"])
			if !ok || resolved <= 0 || resolved > pgInt4Max {
				return emptyConflictBattleList(ctx, opts.DB, domainWhere, domainArgs, page, limit)
			}
			value := int32(resolved)
			corporationID = &value
		}
		if corporationID != nil || allianceID != nil {
			entityWhere := make([]string, 0, 2)
			if corporationID != nil {
				args = append(args, *corporationID)
				entityWhere = append(entityWhere,
					fmt.Sprintf("entity_member.corporation_id = $%d", len(args)))
			}
			if allianceID != nil {
				args = append(args, *allianceID)
				entityWhere = append(entityWhere,
					fmt.Sprintf("entity_member.alliance_id = $%d", len(args)))
			}
			// Drizzle used AND when both explicit filters were present.
			where = append(where, `EXISTS (
				SELECT 1
				FROM battle_teams entity_team
				JOIN battle_team_members entity_member
				  ON entity_member.battle_team_id = entity_team.id
				WHERE entity_team.battle_id = b.battle_id
				  AND (`+strings.Join(entityWhere, " AND ")+`)
			)`)
		}

		args = append(args, limit, (page-1)*limit)
		rows, err := queryMaps(ctx, opts.DB, `
			SELECT b.battle_id, b.solar_system_id,
			       system.system_name AS solar_system_name,
			       system.security AS solar_system_security,
			       b.region_id, region.name AS region_name,
			       b.start_time, b.end_time,
			       COALESCE(b.duration_minutes, 0)::int AS duration_minutes,
			       COALESCE(b.kill_count, 0)::int AS kill_count,
			       COALESCE(b.total_isk_destroyed, 0)::double precision
			           AS total_isk_destroyed,
			       COALESCE(b.is_multi_party, false) AS is_multi_party,
			       COALESCE(b.is_custom, false) AS is_custom
			FROM battles b
			LEFT JOIN solar_systems system
			  ON system.solar_system_id = b.solar_system_id
			LEFT JOIN regions region ON region.region_id = b.region_id
			WHERE `+strings.Join(where, " AND ")+`
			ORDER BY b.start_time DESC, b.battle_id DESC
			LIMIT $`+strconv.Itoa(len(args)-1)+`
			OFFSET $`+strconv.Itoa(len(args)),
			args...,
		)
		if err != nil {
			return legacyPayload{}, err
		}

		years, err := loadConflictBattleYears(ctx, opts.DB, domainWhere, domainArgs)
		if err != nil {
			return legacyPayload{}, err
		}
		if err := attachConflictBattleListTeams(ctx, opts.DB, rows); err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"battles": rows, "years": years, "page": page, "limit": limit,
		}), nil
	}
}

func parseConflictOptionalPositiveInt64(
	values url.Values,
	name string,
) (*int64, error) {
	raw := strings.TrimSpace(values.Get(name))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil, apiError(http.StatusBadRequest, "Invalid "+name)
	}
	return &value, nil
}

func emptyConflictBattleList(
	ctx context.Context,
	db Database,
	where []string,
	args []any,
	page, limit int,
) (legacyPayload, error) {
	years, err := loadConflictBattleYears(ctx, db, where, args)
	if err != nil {
		return legacyPayload{}, err
	}
	return jsonPayload(map[string]any{
		"battles": []map[string]any{}, "years": years,
		"page": page, "limit": limit,
	}), nil
}

func loadConflictBattleYears(
	ctx context.Context,
	db Database,
	where []string,
	args []any,
) ([]map[string]any, error) {
	rows, err := queryMaps(ctx, db, `
		SELECT EXTRACT(YEAR FROM b.start_time)::int AS year,
		       COUNT(*)::int AS count
		FROM battles b
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY EXTRACT(YEAR FROM b.start_time)
		ORDER BY year`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []map[string]any{}
	}
	return rows, nil
}

func attachConflictBattleListTeams(
	ctx context.Context,
	db Database,
	battles []map[string]any,
) error {
	battleIDs := make([]int32, 0, len(battles))
	for _, row := range battles {
		id := conflictInt(row, "battle_id")
		if id > 0 && id <= pgInt4Max {
			battleIDs = append(battleIDs, int32(id))
		}
	}
	if len(battleIDs) == 0 {
		for _, row := range battles {
			row["teams"] = []map[string]any{}
		}
		return nil
	}
	rows, err := queryMaps(ctx, db, `
		SELECT team.battle_id, team.id AS battle_team_id, team.team_index,
		       COALESCE(team.total_kills, 0)::int AS total_kills,
		       COALESCE(team.total_losses, 0)::int AS total_losses,
		       COALESCE(team.total_isk_destroyed, 0)::double precision
		           AS total_isk_destroyed,
		       COALESCE(team.total_isk_lost, 0)::double precision
		           AS total_isk_lost,
		       member.corporation_id, corporation.name AS corporation_name,
		       member.alliance_id, alliance.name AS alliance_name,
		       COALESCE(member.kills, 0)::int AS kills,
		       COALESCE(member.losses, 0)::int AS losses,
		       COALESCE(member.isk_destroyed, 0)::double precision
		           AS isk_destroyed,
		       COALESCE(member.isk_lost, 0)::double precision AS isk_lost
		FROM battle_teams team
		LEFT JOIN battle_team_members member
		  ON member.battle_team_id = team.id
		LEFT JOIN corporations corporation
		  ON corporation.corporation_id = member.corporation_id
		LEFT JOIN alliances alliance ON alliance.alliance_id = member.alliance_id
		WHERE team.battle_id = ANY($1::int[])
		ORDER BY team.battle_id, team.team_index, member.id`,
		battleIDs,
	)
	if err != nil {
		return err
	}
	type aggregate struct {
		ID        any
		Name      any
		CorpID    any
		CorpName  any
		Kills     int64
		Losses    int64
		Destroyed float64
		Lost      float64
	}
	teamMap := map[int64]map[int64]map[string]any{}
	allianceMap := map[int64]map[int64]map[int64]*aggregate{}
	for _, row := range rows {
		battleID := conflictInt(row, "battle_id")
		teamIndex := conflictInt(row, "team_index")
		if teamMap[battleID] == nil {
			teamMap[battleID] = map[int64]map[string]any{}
			allianceMap[battleID] = map[int64]map[int64]*aggregate{}
		}
		if teamMap[battleID][teamIndex] == nil {
			teamMap[battleID][teamIndex] = map[string]any{
				"team_index":          teamIndex,
				"total_kills":         conflictInt(row, "total_kills"),
				"total_losses":        conflictInt(row, "total_losses"),
				"total_isk_destroyed": conflictFloat(row, "total_isk_destroyed"),
				"total_isk_lost":      conflictFloat(row, "total_isk_lost"),
			}
			allianceMap[battleID][teamIndex] = map[int64]*aggregate{}
		}
		corpID := conflictInt(row, "corporation_id")
		if corpID == 0 {
			continue
		}
		allianceID, hasAlliance := int64Value(row["alliance_id"])
		key := allianceID
		if !hasAlliance {
			key = 0
		}
		agg := allianceMap[battleID][teamIndex][key]
		if agg == nil {
			agg = &aggregate{
				ID:       conflictNullableID(row, "alliance_id"),
				Name:     conflictNullableString(row, "alliance_name"),
				CorpID:   conflictNullableID(row, "corporation_id"),
				CorpName: conflictNullableString(row, "corporation_name"),
			}
			allianceMap[battleID][teamIndex][key] = agg
		}
		agg.Kills += conflictInt(row, "kills")
		agg.Losses += conflictInt(row, "losses")
		agg.Destroyed += conflictFloat(row, "isk_destroyed")
		agg.Lost += conflictFloat(row, "isk_lost")
	}
	for battleID, teams := range teamMap {
		for teamIndex, team := range teams {
			aggregates := make([]*aggregate, 0, len(allianceMap[battleID][teamIndex]))
			for _, agg := range allianceMap[battleID][teamIndex] {
				aggregates = append(aggregates, agg)
			}
			sort.Slice(aggregates, func(i, j int) bool {
				left := float64(aggregates[i].Kills) + aggregates[i].Destroyed
				right := float64(aggregates[j].Kills) + aggregates[j].Destroyed
				return left > right
			})
			if len(aggregates) > 5 {
				aggregates = aggregates[:5]
			}
			top := make([]map[string]any, 0, len(aggregates))
			for _, agg := range aggregates {
				item := map[string]any{
					"alliance_id": agg.ID, "alliance_name": agg.Name,
					"corporation_id": nil, "corporation_name": nil,
					"kills": agg.Kills, "losses": agg.Losses,
					"isk_destroyed": agg.Destroyed, "isk_lost": agg.Lost,
				}
				if agg.ID == nil {
					item["corporation_id"] = agg.CorpID
					item["corporation_name"] = agg.CorpName
				}
				top = append(top, item)
			}
			team["alliance_count"] = len(allianceMap[battleID][teamIndex])
			team["corp_count"] = countConflictBattleTeamCorps(rows, battleID, teamIndex)
			team["top_alliances"] = top
		}
	}
	for _, battleRow := range battles {
		id := conflictInt(battleRow, "battle_id")
		byIndex := teamMap[id]
		indices := make([]int, 0, len(byIndex))
		for index := range byIndex {
			indices = append(indices, int(index))
		}
		sort.Ints(indices)
		teams := make([]map[string]any, 0, len(indices))
		for _, index := range indices {
			teams = append(teams, byIndex[int64(index)])
		}
		battleRow["teams"] = teams
	}
	return nil
}

func countConflictBattleTeamCorps(
	rows []map[string]any,
	battleID, teamIndex int64,
) int {
	count := 0
	for _, row := range rows {
		if conflictInt(row, "battle_id") == battleID &&
			conflictInt(row, "team_index") == teamIndex &&
			conflictInt(row, "corporation_id") != 0 {
			count++
		}
	}
	return count
}

func battleDetailHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseConflictID(req)
		if err != nil {
			return legacyPayload{}, err
		}
		window, battleRow, err := loadSavedConflictBattleWindow(ctx, opts.DB, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if window == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "Battle not found")
		}
		result, err := buildBattleDetail(ctx, opts, *window, battleRow, true)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(result), nil
	}
}

func killmailBattleHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		killmailID, err := parseID(req.Param("id"))
		if err != nil || killmailID <= 0 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid ID")
		}
		window, redirectID, err := detectConflictKillmailBattle(
			ctx, opts.DB, killmailID,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if redirectID != 0 {
			return jsonPayload(map[string]any{
				"redirect":  fmt.Sprintf("/api/battle/%d", redirectID),
				"battle_id": redirectID,
			}), nil
		}
		result, err := buildBattleDetail(ctx, opts, *window, nil, false)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(result), nil
	}
}

func detectConflictKillmailBattle(
	ctx context.Context,
	db Database,
	killmailID int64,
) (*conflictBattleWindow, int32, error) {
	seed, err := queryMap(ctx, db, `
		SELECT killmail_id, solar_system_id, region_id, killmail_time
		FROM killmails WHERE killmail_id = $1 LIMIT 1`,
		killmailID,
	)
	if err != nil {
		return nil, 0, err
	}
	if seed == nil {
		return nil, 0, apiError(http.StatusNotFound, "Killmail not found")
	}
	systemID := int32(conflictInt(seed, "solar_system_id"))
	regionID := int32(conflictInt(seed, "region_id"))
	seedTime, ok := seed["killmail_time"].(time.Time)
	if !ok {
		return nil, 0, fmt.Errorf("killmail %d has an invalid timestamp", killmailID)
	}
	saved, err := queryMap(ctx, db, `
			SELECT battle_id FROM battles
			WHERE solar_system_id = $1
			  AND start_time <= $2 AND end_time >= $2
			ORDER BY start_time DESC, battle_id DESC LIMIT 1`,
		systemID, seedTime,
	)
	if err != nil {
		return nil, 0, err
	}
	if saved != nil {
		id := conflictInt(saved, "battle_id")
		return nil, int32(id), nil
	}

	candidateStart := seedTime.Add(-2 * time.Hour)
	candidateEnd := seedTime.Add(2 * time.Hour)
	kills, err := loadConflictBattleKillmails(
		ctx, db, []int32{systemID}, candidateStart, candidateEnd, false,
	)
	if err != nil {
		return nil, 0, err
	}
	if len(kills) < conflictOnTheFlyMinimumKills {
		return nil, 0, apiError(
			http.StatusNotFound,
			"Not enough kills in the area to form a battle",
		)
	}
	attackers, err := loadConflictBattleAttackers(
		ctx, db, []int32{systemID}, candidateStart, candidateEnd,
	)
	if err != nil {
		return nil, 0, err
	}
	refined := battle.RefineBoundaries(
		kills, conflictOnTheFlyMinimumKills, &seedTime,
	)
	if refined == nil {
		return nil, 0, apiError(
			http.StatusNotFound,
			"Could not detect a battle from this killmail",
		)
	}
	kills = conflictKillsWithin(kills, refined.Start, refined.End)
	if len(kills) == 0 {
		return nil, 0, apiError(http.StatusNotFound, "Battle is empty")
	}
	assignment := battle.AssignTeams(kills, attackers)
	if len(assignment.CorpTeam) == 0 {
		return nil, 0, apiError(
			http.StatusNotFound,
			"Could not detect battle sides from this killmail",
		)
	}
	window := conflictBattleWindow{
		KillmailID: killmailID, SystemIDs: []int32{systemID},
		SystemID: systemID, RegionID: regionID,
		Start: refined.Start, End: refined.End,
		Assignment: assignment, TeamIndices: []int{0, 1},
		Kills: kills, Attackers: attackers,
	}
	return &window, 0, nil
}

func loadSavedConflictBattleWindow(
	ctx context.Context,
	db Database,
	id int32,
) (*conflictBattleWindow, map[string]any, error) {
	row, err := queryMap(ctx, db, `
		SELECT battle_id, solar_system_id, region_id, start_time, end_time,
		       COALESCE(duration_minutes, 0)::int AS duration_minutes,
		       COALESCE(kill_count, 0)::int AS kill_count,
		       COALESCE(total_isk_destroyed, 0)::double precision
		           AS total_isk_destroyed,
		       COALESCE(is_multi_party, false) AS is_multi_party,
		       COALESCE(is_custom, false) AS is_custom
		FROM battles WHERE battle_id = $1 LIMIT 1`,
		id,
	)
	if err != nil || row == nil {
		return nil, row, err
	}
	start, startOK := row["start_time"].(time.Time)
	end, endOK := row["end_time"].(time.Time)
	if !startOK || !endOK {
		return nil, row, fmt.Errorf("battle %d has invalid timestamps", id)
	}
	assignment, indices, err := loadSavedConflictBattleAssignment(ctx, db, id)
	if err != nil {
		return nil, row, err
	}
	isCustom, _ := row["is_custom"].(bool)
	return &conflictBattleWindow{
		BattleID:  id,
		SystemID:  int32(conflictInt(row, "solar_system_id")),
		SystemIDs: []int32{int32(conflictInt(row, "solar_system_id"))},
		RegionID:  int32(conflictInt(row, "region_id")),
		Start:     start, End: end, IsCustom: isCustom,
		Assignment: assignment, TeamIndices: indices,
	}, row, nil
}

func loadSavedConflictBattleAssignment(
	ctx context.Context,
	db Database,
	id int32,
) (battle.TeamAssignment, []int, error) {
	rows, err := queryMaps(ctx, db, `
		SELECT team.team_index, member.corporation_id, member.alliance_id
		FROM battle_teams team
		LEFT JOIN battle_team_members member
		  ON member.battle_team_id = team.id
		WHERE team.battle_id = $1
		ORDER BY team.team_index, member.id`,
		id,
	)
	if err != nil {
		return battle.TeamAssignment{}, nil, err
	}
	assignment := battle.TeamAssignment{
		CorpTeam: map[int32]int{}, CorpAlliance: map[int32]int32{},
	}
	indexSet := map[int]struct{}{}
	for _, row := range rows {
		index := int(conflictInt(row, "team_index"))
		indexSet[index] = struct{}{}
		corporationID := int32(conflictInt(row, "corporation_id"))
		if corporationID == 0 || index < 0 || index > 1 {
			continue
		}
		assignment.CorpTeam[corporationID] = index
		allianceID := int32(conflictInt(row, "alliance_id"))
		if allianceID != 0 {
			assignment.CorpAlliance[corporationID] = allianceID
		}
	}
	indices := make([]int, 0, len(indexSet))
	for index := range indexSet {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	if len(indices) == 0 {
		indices = []int{0, 1}
	}
	return assignment, indices, nil
}

func loadConflictBattleKillmails(
	ctx context.Context,
	db Database,
	systemIDs []int32,
	start, end time.Time,
	applyCustomPrices bool,
) ([]battle.Killmail, error) {
	rows, err := queryMaps(ctx, db, `
		SELECT killmail_id, killmail_time, solar_system_id, region_id,
		       COALESCE(total_value, 0)::double precision AS total_value,
		       victim_corporation_id, victim_alliance_id, victim_faction_id,
		       victim_ship_type_id
		FROM killmails
		WHERE solar_system_id = ANY($1::int[])
		  AND killmail_time >= $2 AND killmail_time <= $3
		ORDER BY killmail_time, killmail_id`,
		systemIDs, start, end,
	)
	if err != nil {
		return nil, err
	}
	deltas := map[int32]float64{}
	if applyCustomPrices {
		types := make([]int32, 0, len(rows))
		for _, row := range rows {
			if id := int32(conflictInt(row, "victim_ship_type_id")); id != 0 {
				types = append(types, id)
			}
		}
		deltas, err = loadConflictPriceDeltas(ctx, db, types, end)
		if err != nil {
			return nil, err
		}
	}
	kills := make([]battle.Killmail, 0, len(rows))
	for _, row := range rows {
		at, ok := row["killmail_time"].(time.Time)
		if !ok {
			continue
		}
		shipTypeID := int32(conflictInt(row, "victim_ship_type_id"))
		kills = append(kills, battle.Killmail{
			KillmailID:          conflictInt(row, "killmail_id"),
			KillmailTime:        at,
			SolarSystemID:       int32(conflictInt(row, "solar_system_id")),
			RegionID:            int32(conflictInt(row, "region_id")),
			TotalValue:          conflictFloat(row, "total_value") + deltas[shipTypeID],
			VictimCorporationID: int32(conflictInt(row, "victim_corporation_id")),
			VictimAllianceID:    int32(conflictInt(row, "victim_alliance_id")),
			VictimFactionID:     int32(conflictInt(row, "victim_faction_id")),
			VictimShipTypeID:    shipTypeID,
		})
	}
	return kills, nil
}

func loadConflictBattleAttackers(
	ctx context.Context,
	db Database,
	systemIDs []int32,
	start, end time.Time,
) (map[int64][]battle.Attacker, error) {
	rows, err := queryMaps(ctx, db, `
		WITH battle_kills AS MATERIALIZED (
			SELECT killmail_id
			FROM killmails
			WHERE solar_system_id = ANY($1::int[])
			  AND killmail_time >= $2 AND killmail_time <= $3
		)
		SELECT attacker.killmail_id, attacker.character_id,
		       attacker.corporation_id, attacker.alliance_id,
		       attacker.faction_id, COALESCE(attacker.damage_done, 0)::bigint
		           AS damage_done,
		       COALESCE(attacker.final_blow, false) AS final_blow
		FROM killmail_attackers attacker
		JOIN battle_kills kill ON kill.killmail_id = attacker.killmail_id
		WHERE attacker.killmail_time >= $2
		  AND attacker.killmail_time <= $3`,
		systemIDs, start, end,
	)
	if err != nil {
		return nil, err
	}
	out := make(map[int64][]battle.Attacker)
	for _, row := range rows {
		id := conflictInt(row, "killmail_id")
		out[id] = append(out[id], battle.Attacker{
			KillmailID:    id,
			CharacterID:   int32(conflictInt(row, "character_id")),
			CorporationID: int32(conflictInt(row, "corporation_id")),
			AllianceID:    int32(conflictInt(row, "alliance_id")),
			FactionID:     int32(conflictInt(row, "faction_id")),
			DamageDone:    conflictInt(row, "damage_done"),
			FinalBlow:     boolValueOrFalse(row["final_blow"]),
		})
	}
	return out, nil
}

func conflictKillsWithin(
	kills []battle.Killmail,
	start, end time.Time,
) []battle.Killmail {
	out := make([]battle.Killmail, 0, len(kills))
	for _, kill := range kills {
		if !kill.KillmailTime.Before(start) && kill.KillmailTime.Before(end) {
			out = append(out, kill)
		}
	}
	return out
}

func buildBattleDetail(
	ctx context.Context,
	opts Options,
	window conflictBattleWindow,
	stored map[string]any,
	applyCustomPrices bool,
) (map[string]any, error) {
	kills := window.Kills
	attackers := window.Attackers
	loaders, loaderCtx := errgroup.WithContext(ctx)
	if kills == nil {
		loaders.Go(func() (err error) {
			var loaded []battle.Killmail
			loaded, err = loadConflictBattleKillmails(
				loaderCtx, opts.DB, window.SystemIDs,
				window.Start, window.End, applyCustomPrices,
			)
			kills = loaded
			return err
		})
	}
	if attackers == nil {
		loaders.Go(func() (err error) {
			var loaded map[int64][]battle.Attacker
			loaded, err = loadConflictBattleAttackers(
				loaderCtx, opts.DB, window.SystemIDs,
				window.Start, window.End,
			)
			attackers = loaded
			return err
		})
	}
	if err := loaders.Wait(); err != nil {
		return nil, err
	}
	participants := battleParticipantCounts(kills, attackers)
	teams := battle.ComputeTeamStats(kills, attackers, window.Assignment)
	ensureBattleAssignmentEntries(&teams, window.Assignment)
	restrictBattleTeamEntries(&teams, window.Assignment)

	names, err := loadBattleNames(
		ctx, opts.DB, window, teams, kills, attackers,
	)
	if err != nil {
		return nil, err
	}
	formattedTeams := formatBattleTeams(teams, names)
	teamEntities := make([]map[string]any, 0, len(formattedTeams))
	for _, team := range formattedTeams {
		corps, alliances := battleTeamEntities(team)
		teamEntities = append(teamEntities, map[string]any{
			"corps": corps, "alliances": alliances,
		})
	}
	headlineCount, headlineISK := battleSidedTotals(
		kills, attackers, window.Assignment,
	)
	system := names.Systems[window.SystemID]
	result := map[string]any{
		"battle_id": nil, "solar_system_id": window.SystemID,
		"solar_system_name":     conflictNullableString(system, "name"),
		"solar_system_security": conflictNullableFloat(system, "security"),
		"region_id":             nullableConflictInt32(window.RegionID),
		"region_name":           conflictNullableString(names.Region, "name"),
		"start_time":            window.Start, "end_time": window.End,
		"duration_minutes": int(window.End.Sub(window.Start).Minutes()),
		"kill_count":       headlineCount, "total_isk_destroyed": headlineISK,
		"is_multi_party":        window.Assignment.MultiParty,
		"is_custom":             window.IsCustom,
		"characters_involved":   conflictInt(participants, "characters"),
		"corporations_involved": conflictInt(participants, "corporations"),
		"alliances_involved":    conflictInt(participants, "alliances"),
		"total_damage":          conflictInt(participants, "total_damage"),
		"teams":                 formattedTeams, "team_entities": teamEntities,
	}
	if window.BattleID != 0 {
		result["battle_id"] = window.BattleID
		result["unsided"] = formatBattleUnsided(
			kills, attackers, window.Assignment, names,
		)
		if stored != nil {
			result["duration_minutes"] = conflictInt(
				stored, "duration_minutes",
			)
			if value, ok := stored["is_multi_party"].(bool); ok {
				result["is_multi_party"] = value
			}
		}
	} else {
		result["killmail_id"] = window.KillmailID
		// Detection knows this directly, whereas a stored battle preserves its
		// explicit multi-party flag.
		result["is_multi_party"] = window.Assignment.MultiParty
	}
	return result, nil
}

// Count the exact kills shown in the battle, not the wider candidate window
// retained by on-the-fly detection. All attackers, including unsided/NPC
// participants, contribute damage; missing entity IDs use the domain's zero sentinel.
func battleParticipantCounts(kills []battle.Killmail, attackers map[int64][]battle.Attacker) map[string]any {
	characters, corporations, alliances := map[int32]struct{}{}, map[int32]struct{}{}, map[int32]struct{}{}
	seen := map[int64]struct{}{}
	var damage int64
	for _, kill := range kills {
		if _, ok := seen[kill.KillmailID]; ok {
			continue
		}
		seen[kill.KillmailID] = struct{}{}
		for _, a := range attackers[kill.KillmailID] {
			if a.CharacterID != 0 {
				characters[a.CharacterID] = struct{}{}
			}
			if a.CorporationID != 0 {
				corporations[a.CorporationID] = struct{}{}
			}
			if a.AllianceID != 0 {
				alliances[a.AllianceID] = struct{}{}
			}
			damage += a.DamageDone
		}
	}
	return map[string]any{"characters": len(characters), "corporations": len(corporations),
		"alliances": len(alliances), "total_damage": damage}
}

func loadBattleNames(
	ctx context.Context,
	db Database,
	window conflictBattleWindow,
	teams [2]battle.Team,
	kills []battle.Killmail,
	attackers map[int64][]battle.Attacker,
) (conflictBattleNames, error) {
	corpSet := map[int32]struct{}{}
	allianceSet := map[int32]struct{}{}
	for _, team := range teams {
		for _, entry := range team.Entries {
			if entry.CorporationID != 0 {
				corpSet[entry.CorporationID] = struct{}{}
			}
			if entry.AllianceID != 0 {
				allianceSet[entry.AllianceID] = struct{}{}
			}
		}
	}
	for corpID, allianceID := range window.Assignment.CorpAlliance {
		corpSet[corpID] = struct{}{}
		if allianceID != 0 {
			allianceSet[allianceID] = struct{}{}
		}
	}
	// Include unsided participants as well. Saved battles deliberately surface
	// them rather than silently dropping third parties and newly joined corps.
	for _, kill := range kills {
		if kill.VictimCorporationID != 0 {
			corpSet[kill.VictimCorporationID] = struct{}{}
		}
		if kill.VictimAllianceID != 0 {
			allianceSet[kill.VictimAllianceID] = struct{}{}
		}
		for _, attacker := range attackers[kill.KillmailID] {
			if attacker.CorporationID != 0 {
				corpSet[attacker.CorporationID] = struct{}{}
			}
			if attacker.AllianceID != 0 {
				allianceSet[attacker.AllianceID] = struct{}{}
			}
		}
	}
	corps := conflictSetIDs(corpSet)
	alliances := conflictSetIDs(allianceSet)
	rows, err := queryMapsConcurrent(ctx, db,
		databaseQuery{
			SQL: `SELECT solar_system_id AS id, system_name AS name, security
			      FROM solar_systems WHERE solar_system_id = ANY($1::int[])`,
			Args: []any{window.SystemIDs},
		},
		databaseQuery{
			SQL:  `SELECT region_id AS id, name FROM regions WHERE region_id = $1`,
			Args: []any{window.RegionID},
		},
		databaseQuery{
			SQL: `SELECT corporation_id AS id, name, alliance_id, palette
			      FROM corporations WHERE corporation_id = ANY($1::int[])`,
			Args: []any{corps},
		},
		databaseQuery{
			SQL: `SELECT alliance_id AS id, name
			      FROM alliances WHERE alliance_id = ANY($1::int[])`,
			Args: []any{alliances},
		},
	)
	if err != nil {
		return conflictBattleNames{}, err
	}
	out := conflictBattleNames{
		Systems:   map[int32]map[string]any{},
		Region:    map[string]any{},
		Corps:     map[int32]map[string]any{},
		Alliances: map[int32]string{},
	}
	for _, row := range rows[0] {
		out.Systems[int32(conflictInt(row, "id"))] = row
	}
	if len(rows[1]) > 0 {
		out.Region = rows[1][0]
	}
	for _, row := range rows[2] {
		out.Corps[int32(conflictInt(row, "id"))] = row
	}
	for _, row := range rows[3] {
		out.Alliances[int32(conflictInt(row, "id"))] = conflictString(row, "name")
	}
	return out, nil
}

func ensureBattleAssignmentEntries(
	teams *[2]battle.Team,
	assignment battle.TeamAssignment,
) {
	existing := map[int32]struct{}{}
	for _, team := range teams {
		for _, entry := range team.Entries {
			existing[entry.CorporationID] = struct{}{}
		}
	}
	for corpID, teamIndex := range assignment.CorpTeam {
		if teamIndex < 0 || teamIndex > 1 {
			continue
		}
		if _, ok := existing[corpID]; ok {
			continue
		}
		teams[teamIndex].Entries = append(teams[teamIndex].Entries, battle.TeamEntry{
			CorporationID: corpID,
			AllianceID:    assignment.CorpAlliance[corpID],
		})
	}
}

func restrictBattleTeamEntries(
	teams *[2]battle.Team,
	assignment battle.TeamAssignment,
) {
	for teamIndex := range teams {
		filtered := teams[teamIndex].Entries[:0]
		for _, entry := range teams[teamIndex].Entries {
			if assigned, exists := assignment.CorpTeam[entry.CorporationID]; exists && assigned == teamIndex {
				filtered = append(filtered, entry)
			}
		}
		teams[teamIndex].Entries = filtered
	}
}

func formatBattleTeams(
	teams [2]battle.Team,
	names conflictBattleNames,
) []map[string]any {
	result := make([]map[string]any, 0, len(teams))
	for teamIndex, team := range teams {
		type group struct {
			Key             int64
			AllianceID      int32
			Entries         []battle.TeamEntry
			Kills, Losses   int64
			Destroyed, Lost float64
		}
		groups := map[int64]*group{}
		for _, entry := range team.Entries {
			key := int64(entry.AllianceID)
			current := groups[key]
			if current == nil {
				current = &group{Key: key, AllianceID: entry.AllianceID}
				groups[key] = current
			}
			current.Entries = append(current.Entries, entry)
			current.Kills += entry.Kills
			current.Losses += entry.Losses
			current.Destroyed += entry.IskDestroyed
			current.Lost += entry.IskLost
		}
		ordered := make([]*group, 0, len(groups))
		for _, current := range groups {
			ordered = append(ordered, current)
		}
		sort.Slice(ordered, func(i, j int) bool {
			left := float64(ordered[i].Kills) + ordered[i].Destroyed
			right := float64(ordered[j].Kills) + ordered[j].Destroyed
			if left != right {
				return left > right
			}
			return ordered[i].Key < ordered[j].Key
		})
		allianceRows := make([]map[string]any, 0, len(ordered))
		var dominantID int32
		var dominantActivity float64 = -1
		for _, current := range ordered {
			sort.Slice(current.Entries, func(i, j int) bool {
				left := float64(current.Entries[i].Kills) + current.Entries[i].IskDestroyed
				right := float64(current.Entries[j].Kills) + current.Entries[j].IskDestroyed
				if left != right {
					return left > right
				}
				return current.Entries[i].CorporationID <
					current.Entries[j].CorporationID
			})
			corporations := make([]map[string]any, 0, len(current.Entries))
			for _, entry := range current.Entries {
				corp := names.Corps[entry.CorporationID]
				activity := float64(entry.Kills+entry.Losses) +
					entry.IskDestroyed + entry.IskLost
				if activity > dominantActivity {
					dominantActivity = activity
					dominantID = entry.CorporationID
				}
				corporations = append(corporations, map[string]any{
					"corporation_id":   entry.CorporationID,
					"corporation_name": conflictNullableString(corp, "name"),
					"kills":            entry.Kills, "losses": entry.Losses,
					"isk_destroyed": entry.IskDestroyed,
					"isk_lost":      entry.IskLost,
				})
			}
			var allianceID any
			var allianceName any
			if current.AllianceID != 0 {
				allianceID = current.AllianceID
				if name := names.Alliances[current.AllianceID]; name != "" {
					allianceName = name
				}
			}
			allianceRows = append(allianceRows, map[string]any{
				"alliance_id": allianceID, "alliance_name": allianceName,
				"kills": current.Kills, "losses": current.Losses,
				"isk_destroyed": current.Destroyed, "isk_lost": current.Lost,
				"corporations": corporations,
			})
		}
		var palette any
		if corp := names.Corps[dominantID]; corp != nil {
			palette = corp["palette"]
		}
		result = append(result, map[string]any{
			"team_index":  teamIndex,
			"total_kills": team.Kills, "total_losses": team.Losses,
			"total_isk_destroyed": team.IskDestroyed,
			"total_isk_lost":      team.IskLost,
			"alliance_count":      len(groups), "corp_count": len(team.Entries),
			"alliances": allianceRows, "dominant_corp_palette": palette,
		})
	}
	return result
}

func battleTeamEntities(team map[string]any) ([]int32, []int32) {
	corps := []int32{}
	alliances := []int32{}
	rows, _ := team["alliances"].([]map[string]any)
	for _, alliance := range rows {
		if id, ok := int64Value(alliance["alliance_id"]); ok && id > 0 {
			alliances = append(alliances, int32(id))
		}
		corporations, _ := alliance["corporations"].([]map[string]any)
		for _, corporation := range corporations {
			if id, ok := int64Value(corporation["corporation_id"]); ok && id > 0 {
				corps = append(corps, int32(id))
			}
		}
	}
	return uniqueConflictIDs(corps), uniqueConflictIDs(alliances)
}

func battleSidedTotals(
	kills []battle.Killmail,
	attackers map[int64][]battle.Attacker,
	assignment battle.TeamAssignment,
) (int, float64) {
	allianceTeam := battleAllianceTeams(assignment)
	sideOf := func(corp, alliance int32) (int, bool) {
		if team, ok := assignment.CorpTeam[corp]; corp != 0 && ok {
			return team, true
		}
		team, ok := allianceTeam[alliance]
		return team, alliance != 0 && ok
	}
	count := 0
	var isk float64
	for _, kill := range kills {
		_, victimSided := sideOf(
			kill.VictimCorporationID, kill.VictimAllianceID,
		)
		killer := battleFinalBlow(attackers[kill.KillmailID])
		killerSided := false
		if killer != nil {
			_, killerSided = sideOf(killer.CorporationID, killer.AllianceID)
		}
		if victimSided || killerSided {
			count++
			isk += kill.TotalValue
		}
	}
	return count, isk
}

func battleAllianceTeams(
	assignment battle.TeamAssignment,
) map[int32]int {
	out := map[int32]int{}
	for corporationID, team := range assignment.CorpTeam {
		if allianceID := assignment.CorpAlliance[corporationID]; allianceID != 0 {
			if _, exists := out[allianceID]; !exists {
				out[allianceID] = team
			}
		}
	}
	return out
}

func battleFinalBlow(attackers []battle.Attacker) *battle.Attacker {
	var highest *battle.Attacker
	for index := range attackers {
		if attackers[index].FinalBlow {
			return &attackers[index]
		}
		if highest == nil || attackers[index].DamageDone > highest.DamageDone {
			highest = &attackers[index]
		}
	}
	return highest
}

func formatBattleUnsided(
	kills []battle.Killmail,
	attackers map[int64][]battle.Attacker,
	assignment battle.TeamAssignment,
	names conflictBattleNames,
) map[string]any {
	type stats struct {
		CorporationID, AllianceID int32
		Kills, Losses             int64
		Destroyed, Lost           float64
	}
	allianceTeams := battleAllianceTeams(assignment)
	isSided := func(corporationID, allianceID int32) bool {
		if _, ok := assignment.CorpTeam[corporationID]; corporationID != 0 && ok {
			return true
		}
		_, ok := allianceTeams[allianceID]
		return allianceID != 0 && ok
	}
	byCorp := map[int32]*stats{}
	get := func(corporationID, allianceID int32) *stats {
		if corporationID == 0 {
			return nil
		}
		value := byCorp[corporationID]
		if value == nil {
			value = &stats{CorporationID: corporationID, AllianceID: allianceID}
			byCorp[corporationID] = value
		} else if value.AllianceID == 0 {
			value.AllianceID = allianceID
		}
		return value
	}
	for _, kill := range kills {
		if !isSided(kill.VictimCorporationID, kill.VictimAllianceID) {
			if value := get(kill.VictimCorporationID, kill.VictimAllianceID); value != nil {
				value.Losses++
				value.Lost += kill.TotalValue
			}
		}
		killer := battleFinalBlow(attackers[kill.KillmailID])
		if killer != nil && !isSided(killer.CorporationID, killer.AllianceID) {
			if value := get(killer.CorporationID, killer.AllianceID); value != nil {
				value.Kills++
				value.Destroyed += kill.TotalValue
			}
		}
	}
	type group struct {
		Key             int64
		AllianceID      int32
		Items           []*stats
		Kills, Losses   int64
		Destroyed, Lost float64
	}
	groups := map[int64]*group{}
	for _, value := range byCorp {
		key := int64(value.AllianceID)
		current := groups[key]
		if current == nil {
			current = &group{Key: key, AllianceID: value.AllianceID}
			groups[key] = current
		}
		current.Items = append(current.Items, value)
		current.Kills += value.Kills
		current.Losses += value.Losses
		current.Destroyed += value.Destroyed
		current.Lost += value.Lost
	}
	ordered := make([]*group, 0, len(groups))
	for _, current := range groups {
		ordered = append(ordered, current)
	}
	sort.Slice(ordered, func(i, j int) bool {
		left := float64(ordered[i].Kills+ordered[i].Losses) +
			ordered[i].Destroyed + ordered[i].Lost
		right := float64(ordered[j].Kills+ordered[j].Losses) +
			ordered[j].Destroyed + ordered[j].Lost
		if left != right {
			return left > right
		}
		return ordered[i].Key < ordered[j].Key
	})
	allianceRows := make([]map[string]any, 0, len(ordered))
	var killsTotal, lossesTotal int64
	var destroyedTotal, lostTotal float64
	allianceCount := 0
	for _, current := range ordered {
		if current.AllianceID != 0 {
			allianceCount++
		}
		sort.Slice(current.Items, func(i, j int) bool {
			left := float64(current.Items[i].Kills+current.Items[i].Losses) +
				current.Items[i].Destroyed + current.Items[i].Lost
			right := float64(current.Items[j].Kills+current.Items[j].Losses) +
				current.Items[j].Destroyed + current.Items[j].Lost
			if left != right {
				return left > right
			}
			return current.Items[i].CorporationID <
				current.Items[j].CorporationID
		})
		corporations := make([]map[string]any, 0, len(current.Items))
		for _, value := range current.Items {
			corporations = append(corporations, map[string]any{
				"corporation_id": value.CorporationID,
				"corporation_name": conflictNullableString(
					names.Corps[value.CorporationID], "name",
				),
				"kills": value.Kills, "losses": value.Losses,
				"isk_destroyed": value.Destroyed, "isk_lost": value.Lost,
			})
		}
		var allianceID any
		var allianceName any
		if current.AllianceID != 0 {
			allianceID = current.AllianceID
			if value := names.Alliances[current.AllianceID]; value != "" {
				allianceName = value
			}
		}
		allianceRows = append(allianceRows, map[string]any{
			"alliance_id": allianceID, "alliance_name": allianceName,
			"kills": current.Kills, "losses": current.Losses,
			"isk_destroyed": current.Destroyed, "isk_lost": current.Lost,
			"corporations": corporations,
		})
		killsTotal += current.Kills
		lossesTotal += current.Losses
		destroyedTotal += current.Destroyed
		lostTotal += current.Lost
	}
	return map[string]any{
		"kills": killsTotal, "losses": lossesTotal,
		"isk_destroyed": destroyedTotal, "isk_lost": lostTotal,
		"alliance_count": allianceCount, "corp_count": len(byCorp),
		"alliances": allianceRows,
	}
}

func conflictSetIDs(values map[int32]struct{}) []int32 {
	out := make([]int32, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

func nullableConflictInt32(value int32) any {
	if value == 0 {
		return nil
	}
	return value
}

func boolValueOrFalse(value any) bool {
	result, _ := value.(bool)
	return result
}
