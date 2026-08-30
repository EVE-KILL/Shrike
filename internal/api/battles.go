package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const battleListColumns = `
	b.battle_id, b.solar_system_id, s.system_name,
	b.region_id, r.name AS region_name, b.start_time, b.end_time,
	b.duration_minutes, b.kill_count, b.total_isk_destroyed,
	b.is_multi_party, b.is_custom`

func registerBattleRoutes(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "battles",
		Method:      http.MethodGet,
		Path:        "/battles",
		Summary:     "Battle reports",
		Tags:        []string{"battles"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		page, limit, order, startAfter, startBefore, err := parseBattleFilters(req)
		if err != nil {
			return legacyPayload{}, err
		}
		rawSort := req.Query.Get("sort")
		if rawSort == "" {
			rawSort = "battle_id"
		}
		sortColumn := map[string]string{
			"battle_id":           "b.battle_id",
			"total_isk_destroyed": "b.total_isk_destroyed",
			"kill_count":          "b.kill_count",
			"start_time":          "b.start_time",
		}[rawSort]
		if sortColumn == "" {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid sort: "+rawSort)
		}
		where, args := battleDateConditions(startAfter, startBefore)
		query := `SELECT ` + battleListColumns + `
			FROM battles b
			LEFT JOIN solar_systems s ON s.solar_system_id = b.solar_system_id
			LEFT JOIN regions r ON r.region_id = b.region_id`
		if len(where) > 0 {
			query += " WHERE " + strings.Join(where, " AND ")
		}
		args = append(args, limit+1, (page-1)*limit)
		query += fmt.Sprintf(
			" ORDER BY %s %s NULLS LAST LIMIT $%d OFFSET $%d",
			sortColumn, order, len(args)-1, len(args),
		)
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return battlePage(rows, page, limit), nil
	})

	for _, spec := range []struct {
		id, path, summary, column string
	}{
		{"corporation-battles", "/battles/corporation/{id}", "Corporation battles", "corporation_id"},
		{"alliance-battles", "/battles/alliance/{id}", "Alliance battles", "alliance_id"},
	} {
		registerLegacy(a, entityIDOperation(spec.id, spec.path, spec.summary, "battles"),
			func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
				id, err := parseID(req.Param("id"))
				if err != nil {
					return legacyPayload{}, err
				}
				return loadEntityBattles(ctx, opts.DB, spec.column, id, req)
			})
	}

	registerLegacy(a, entityIDOperation(
		"battle", "/battles/{id}", "Battle report detail", "battles",
	), func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		battle, err := queryMap(ctx, opts.DB, `
			SELECT b.battle_id, b.solar_system_id, s.system_name,
			       s.security AS system_security, b.region_id,
			       r.name AS region_name, b.start_time, b.end_time,
			       b.duration_minutes, b.kill_count, b.total_isk_destroyed,
			       b.is_multi_party, b.is_custom
			FROM battles b
			LEFT JOIN solar_systems s ON s.solar_system_id = b.solar_system_id
			LEFT JOIN regions r ON r.region_id = b.region_id
			WHERE b.battle_id = $1 LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if battle == nil {
			return foundOr404(nil, "Battle not found"), nil
		}
		teams, err := queryMaps(ctx, opts.DB,
			`SELECT * FROM battle_teams WHERE battle_id = $1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		teamValues := make([]any, 0, len(teams))
		for _, team := range teams {
			teamValues = append(teamValues, team["id"])
		}
		teamIDs := int32Slice(teamValues...)
		members := []map[string]any{}
		if len(teamIDs) > 0 {
			members, err = queryMaps(ctx, opts.DB, `
				SELECT m.battle_team_id, m.corporation_id,
				       c.name AS corporation_name, c.ticker AS corporation_ticker,
				       m.alliance_id, a.name AS alliance_name,
				       a.ticker AS alliance_ticker,
				       COALESCE(m.kills, 0) AS kills,
				       COALESCE(m.losses, 0) AS losses,
				       COALESCE(m.isk_destroyed, 0) AS isk_destroyed,
				       COALESCE(m.isk_lost, 0) AS isk_lost
				FROM battle_team_members m
				LEFT JOIN corporations c ON c.corporation_id = m.corporation_id
				LEFT JOIN alliances a ON a.alliance_id = m.alliance_id
				WHERE m.battle_team_id = ANY($1::int[])`, teamIDs)
			if err != nil {
				return legacyPayload{}, err
			}
		}
		membersByTeam := map[int64][]map[string]any{}
		for _, member := range members {
			teamID, _ := int64Value(member["battle_team_id"])
			delete(member, "battle_team_id")
			membersByTeam[teamID] = append(membersByTeam[teamID], member)
		}
		teamOutput := make([]map[string]any, 0, len(teams))
		for _, team := range teams {
			teamID, _ := int64Value(team["id"])
			teamMembers := membersByTeam[teamID]
			if teamMembers == nil {
				teamMembers = []map[string]any{}
			}
			teamOutput = append(teamOutput, map[string]any{
				"team_index":          team["team_index"],
				"total_kills":         zeroIfNil(team["total_kills"]),
				"total_losses":        zeroIfNil(team["total_losses"]),
				"total_isk_destroyed": zeroIfNil(team["total_isk_destroyed"]),
				"total_isk_lost":      zeroIfNil(team["total_isk_lost"]),
				"members":             teamMembers,
			})
		}
		return jsonPayload(map[string]any{
			"battle": battle,
			"teams":  teamOutput,
		}), nil
	})
}

func parseBattleFilters(req *legacyRequest) (
	page, limit int,
	order string,
	startAfter, startBefore *time.Time,
	err error,
) {
	page = boundedQueryInt(req, "page", 1, 1, 500)
	limit = boundedQueryInt(req, "limit", 20, 1, 100)
	order = "DESC"
	if req.Query.Get("order") == "asc" {
		order = "ASC"
	}
	if raw := req.Query.Get("start_after"); raw != "" {
		value, parseErr := parseJavaScriptDate(raw)
		if parseErr != nil {
			err = apiError(http.StatusBadRequest, "Invalid start_after")
			return
		}
		startAfter = &value
	}
	if raw := req.Query.Get("start_before"); raw != "" {
		value, parseErr := parseJavaScriptDate(raw)
		if parseErr != nil {
			err = apiError(http.StatusBadRequest, "Invalid start_before")
			return
		}
		startBefore = &value
	}
	return
}

func parseJavaScriptDate(raw string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02 15:04:05",
	} {
		if value, err := time.Parse(layout, raw); err == nil {
			return value, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date")
}

func battleDateConditions(after, before *time.Time) ([]string, []any) {
	where, args := []string{}, []any{}
	if after != nil {
		args = append(args, *after)
		where = append(where, fmt.Sprintf("b.start_time > $%d", len(args)))
	}
	if before != nil {
		args = append(args, *before)
		where = append(where, fmt.Sprintf("b.start_time < $%d", len(args)))
	}
	return where, args
}

func loadEntityBattles(
	ctx context.Context,
	db Database,
	column string,
	entityID int64,
	req *legacyRequest,
) (legacyPayload, error) {
	page, limit, order, startAfter, startBefore, err := parseBattleFilters(req)
	if err != nil {
		return legacyPayload{}, err
	}
	sortKey := req.Query.Get("sort")
	if sortKey == "" {
		sortKey = "battle_id"
	}
	sortExpression := map[string]string{
		"battle_id":           "b.battle_id",
		"total_isk_destroyed": "COALESCE(SUM(m.isk_destroyed), 0)",
		"kill_count":          "COALESCE(SUM(m.kills), 0)",
		"losses":              "COALESCE(SUM(m.losses), 0)",
		"start_time":          "b.start_time",
	}[sortKey]
	if sortExpression == "" {
		return legacyPayload{}, apiError(
			http.StatusBadRequest,
			"Invalid sort: "+req.Query.Get("sort"),
		)
	}
	where, args := []string{"m." + column + " = $1"}, []any{entityID}
	if startAfter != nil {
		args = append(args, *startAfter)
		where = append(where, fmt.Sprintf("b.start_time > $%d", len(args)))
	}
	if startBefore != nil {
		args = append(args, *startBefore)
		where = append(where, fmt.Sprintf("b.start_time < $%d", len(args)))
	}
	args = append(args, limit+1, (page-1)*limit)
	rows, err := queryMaps(ctx, db, `
		SELECT `+battleListColumns+`,
		       COALESCE(SUM(m.kills), 0)::bigint AS entity_kills,
		       COALESCE(SUM(m.losses), 0)::bigint AS entity_losses,
		       COALESCE(SUM(m.isk_destroyed), 0)::double precision AS entity_isk_destroyed
		FROM battle_team_members m
		INNER JOIN battle_teams t ON t.id = m.battle_team_id
		INNER JOIN battles b ON b.battle_id = t.battle_id
		LEFT JOIN solar_systems s ON s.solar_system_id = b.solar_system_id
		LEFT JOIN regions r ON r.region_id = b.region_id
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY b.battle_id, s.system_name, r.name
		ORDER BY `+sortExpression+` `+order+` NULLS LAST`+
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)),
		args...,
	)
	if err != nil {
		return legacyPayload{}, err
	}
	for _, row := range rows {
		row["entity_kills"] = integerJSONAsString(row["entity_kills"])
		row["entity_losses"] = integerJSONAsString(row["entity_losses"])
	}
	return battlePage(rows, page, limit), nil
}

func integerJSONAsString(value any) string {
	if integer, ok := int64Value(value); ok {
		return fmt.Sprintf("%d", integer)
	}
	return fmt.Sprint(zeroIfNil(value))
}

func battlePage(rows []map[string]any, page, limit int) legacyPayload {
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	return jsonPayload(map[string]any{
		"data": rows,
		"pagination": map[string]any{
			"page": page, "limit": limit, "hasMore": hasMore,
		},
	})
}
