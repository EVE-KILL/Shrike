package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

func registerEntityRoutes(a huma.API, opts Options) {
	registerEntityLists(a, opts)
	registerCharacterDetail(a, opts)
	registerCorporationDetail(a, opts)
	registerAllianceDetail(a, opts)
	registerEntityMembers(a, opts)
	registerEntityKillmailRoutes(a, opts)
	registerEntityStatsRoutes(a, opts)
}

func registerEntityLists(a huma.API, opts Options) {
	for _, spec := range []struct {
		id          string
		path        string
		summary     string
		table       string
		idColumn    string
		selectCols  string
		extraFilter map[string]string
	}{
		{
			"characters", "/characters", "Characters", "characters", "character_id",
			"character_id, name, corporation_id, alliance_id, faction_id, security_status, last_active",
			map[string]string{"corporation_id": "corporation_id", "alliance_id": "alliance_id"},
		},
		{
			"corporations", "/corporations", "Corporations", "corporations", "corporation_id",
			"corporation_id, name, ticker, alliance_id, faction_id, member_count, date_founded",
			map[string]string{"alliance_id": "alliance_id"},
		},
		{
			"alliances", "/alliances", "Alliances", "alliances", "alliance_id",
			"alliance_id, name, ticker, faction_id, corporation_count, member_count, date_founded",
			nil,
		},
	} {
		registerLegacy(a, huma.Operation{
			OperationID: spec.id,
			Method:      http.MethodGet,
			Path:        spec.path,
			Summary:     spec.summary,
			Tags:        []string{spec.id},
		}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			limit := boundedQueryInt(req, "limit", 50, 1, 1000)
			name := strings.TrimSpace(req.Query.Get("name"))
			where := []string{"deleted IS NOT TRUE"}
			args := []any{}
			if name != "" {
				args = append(args, name+"%")
				where = append(where, fmt.Sprintf("name ILIKE $%d", len(args)))
			}
			for queryName, column := range spec.extraFilter {
				if value, ok := optionalQueryNumber(req, queryName); ok && value != 0 {
					args = append(args, value)
					where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
				}
			}
			if after, ok := optionalQueryNumber(req, "after"); ok && after != 0 {
				args = append(args, after)
				if name != "" {
					index := len(args)
					where = append(where, fmt.Sprintf(
						"(name, %s) > ((SELECT name FROM %s WHERE %s = $%d), $%d)",
						spec.idColumn, spec.table, spec.idColumn, index, index,
					))
				} else {
					where = append(where, fmt.Sprintf("%s > $%d", spec.idColumn, len(args)))
				}
			}
			query := "SELECT " + spec.selectCols + " FROM " + spec.table +
				" WHERE " + strings.Join(where, " AND ")
			if name != "" {
				query += " ORDER BY name ASC, " + spec.idColumn + " ASC"
			} else {
				query += " ORDER BY " + spec.idColumn + " ASC"
			}
			args = append(args, limit+1)
			query += fmt.Sprintf(" LIMIT $%d", len(args))
			rows, err := queryMaps(ctx, opts.DB, query, args...)
			if err != nil {
				return legacyPayload{}, err
			}
			return paginatedRows(rows, limit, spec.idColumn), nil
		})

		registerLegacy(a, huma.Operation{
			OperationID: spec.id + "-count",
			Method:      http.MethodGet,
			Path:        spec.path + "/count",
			Summary:     "Estimated " + strings.ToLower(spec.summary) + " count",
			Tags:        []string{spec.id},
		}, func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
			count, err := estimatedRows(ctx, opts.DB, spec.table)
			if err != nil {
				return legacyPayload{}, err
			}
			return jsonPayload(map[string]any{"count": count}), nil
		})
	}
}

func registerCharacterDetail(a huma.API, opts Options) {
	registerLegacy(a, entityIDOperation("character", "/characters/{id}", "Character detail", "characters"),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			character, err := queryMap(ctx, opts.DB, `
				SELECT c.character_id, c.name, c.description, c.birthday, c.gender,
				       COALESCE(c.security_status, 0) AS security_status,
				       c.title, c.last_active,
				       c.corporation_id, co.name AS corporation_name,
				       co.ticker AS corporation_ticker,
				       c.alliance_id, a.name AS alliance_name,
				       a.ticker AS alliance_ticker,
				       c.faction_id, f.name AS faction_name,
				       r.race_name, b.bloodline_name
				FROM characters c
				LEFT JOIN corporations co ON co.corporation_id = c.corporation_id
				LEFT JOIN alliances a ON a.alliance_id = c.alliance_id
				LEFT JOIN factions f ON f.faction_id = c.faction_id
				LEFT JOIN races r ON r.race_id = c.race_id
				LEFT JOIN bloodlines b ON b.bloodline_id = c.bloodline_id
				WHERE c.character_id = $1 LIMIT 1`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			if character == nil {
				return foundOr404(nil, "Character not found"), nil
			}

			alltime, err := loadEntityStats(ctx, opts.DB, entityCharacter, id, "alltime")
			if err != nil {
				return legacyPayload{}, err
			}
			recent, err := loadEntityStats(ctx, opts.DB, entityCharacter, id, "90d")
			if err != nil {
				return legacyPayload{}, err
			}
			flown, lost, systems, err := loadStandardBreakdowns(ctx, opts.DB, entityCharacter, id, "alltime")
			if err != nil {
				return legacyPayload{}, err
			}
			topShips, topSystems, err := assembleTopBreakdowns(ctx, opts.DB, flown, lost, systems)
			if err != nil {
				return legacyPayload{}, err
			}

			history, err := queryMaps(ctx, opts.DB, `
				SELECT h.record_id, h.corporation_id, h.start_date,
				       co.name AS corporation_name, co.ticker AS corporation_ticker
				FROM character_corporation_history h
				LEFT JOIN corporations co ON co.corporation_id = h.corporation_id
				WHERE h.character_id = $1
				ORDER BY h.start_date DESC`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			stintStats := map[int64]map[string]any{}
			if len(history) > 0 {
				rows, queryErr := queryMaps(ctx, opts.DB, `
					WITH stints AS (
						SELECT corporation_id, start_date,
						       LEAD(start_date) OVER (ORDER BY start_date) AS end_date,
						       ROW_NUMBER() OVER (ORDER BY start_date DESC) AS rn
						FROM character_corporation_history
						WHERE character_id = $1
					),
					daily AS (
						SELECT s.rn, COALESCE(SUM(d.kills), 0) AS kills,
						       COALESCE(SUM(d.losses), 0) AS losses
						FROM stints s
						LEFT JOIN stats d
						  ON d.entity_type = 0 AND d.entity_id = $1
						 AND d.period_type = 0
						 AND d.period_start >= GREATEST(s.start_date::date, (CURRENT_DATE - INTERVAL '365 days')::date)
						 AND d.period_start < COALESCE(s.end_date::date, CURRENT_DATE + INTERVAL '1 day')
						GROUP BY s.rn
					),
					monthly AS (
						SELECT s.rn, COALESCE(SUM(m.kills), 0) AS kills,
						       COALESCE(SUM(m.losses), 0) AS losses
						FROM stints s
						LEFT JOIN stats m
						  ON m.entity_type = 0 AND m.entity_id = $1
						 AND m.period_type = 1
						 AND m.period_start >= GREATEST(s.start_date::date, (CURRENT_DATE - INTERVAL '18 months')::date)
						 AND m.period_start < LEAST(COALESCE(s.end_date::date, CURRENT_DATE + INTERVAL '1 day'), (CURRENT_DATE - INTERVAL '365 days')::date)
						GROUP BY s.rn
					),
					yearly AS (
						SELECT s.rn, COALESCE(SUM(y.kills), 0) AS kills,
						       COALESCE(SUM(y.losses), 0) AS losses
						FROM stints s
						LEFT JOIN stats y
						  ON y.entity_type = 0 AND y.entity_id = $1
						 AND y.period_type = 2
						 AND y.period_start >= s.start_date::date
						 AND y.period_start < LEAST(COALESCE(s.end_date::date, CURRENT_DATE + INTERVAL '1 day'), (CURRENT_DATE - INTERVAL '18 months')::date)
						GROUP BY s.rn
					)
					SELECT d.rn,
					       (d.kills + m.kills + y.kills)::int AS kills,
					       (d.losses + m.losses + y.losses)::int AS losses
					FROM daily d
					JOIN monthly m ON m.rn = d.rn
					JOIN yearly y ON y.rn = d.rn
					ORDER BY d.rn`, id)
				if queryErr != nil {
					return legacyPayload{}, queryErr
				}
				for _, row := range rows {
					rn, _ := int64Value(row["rn"])
					stintStats[rn] = row
				}
			}
			corporationHistory := make([]map[string]any, 0, len(history))
			for index, row := range history {
				name := row["corporation_name"]
				if name == nil {
					name = "Unknown"
				}
				ticker := row["corporation_ticker"]
				if ticker == nil {
					ticker = "???"
				}
				stats := stintStats[int64(index+1)]
				kills, losses := any(0), any(0)
				if stats != nil {
					kills, losses = stats["kills"], stats["losses"]
				}
				corporationHistory = append(corporationHistory, map[string]any{
					"corporation_id":     row["corporation_id"],
					"corporation_name":   name,
					"corporation_ticker": ticker,
					"start_date":         row["start_date"],
					"kills":              kills,
					"losses":             losses,
				})
			}

			return jsonPayload(map[string]any{
				"character":          character,
				"stats":              scalarStatsMap(alltime, true),
				"recentStats":        recentStatsMap(recent),
				"topShips":           topShips,
				"topSystems":         topSystems,
				"corporationHistory": corporationHistory,
			}), nil
		})
}

func registerCorporationDetail(a huma.API, opts Options) {
	registerLegacy(a, entityIDOperation("corporation", "/corporations/{id}", "Corporation detail", "corporations"),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			corporation, err := queryMap(ctx, opts.DB, `
				SELECT c.corporation_id, c.name, c.ticker, c.description,
				       c.date_founded, c.url, COALESCE(c.member_count, 0) AS member_count,
				       COALESCE(c.tax_rate, 0) AS tax_rate,
				       COALESCE(c.war_eligible, false) AS war_eligible,
				       c.ceo_id, ceo.name AS ceo_name,
				       c.creator_id, creator.name AS creator_name,
				       c.alliance_id, a.name AS alliance_name, a.ticker AS alliance_ticker,
				       c.faction_id, f.name AS faction_name
				FROM corporations c
				LEFT JOIN characters ceo ON ceo.character_id = c.ceo_id
				LEFT JOIN characters creator ON creator.character_id = c.creator_id
				LEFT JOIN alliances a ON a.alliance_id = c.alliance_id
				LEFT JOIN factions f ON f.faction_id = c.faction_id
				WHERE c.corporation_id = $1 LIMIT 1`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			if corporation == nil {
				return foundOr404(nil, "Corporation not found"), nil
			}
			alltime, err := loadEntityStats(ctx, opts.DB, entityCorporation, id, "alltime")
			if err != nil {
				return legacyPayload{}, err
			}
			recent, err := loadEntityStats(ctx, opts.DB, entityCorporation, id, "90d")
			if err != nil {
				return legacyPayload{}, err
			}
			history, err := queryMaps(ctx, opts.DB, `
				SELECT h.alliance_id, a.name AS alliance_name,
				       a.ticker AS alliance_ticker, h.start_date
				FROM corporation_alliance_history h
				LEFT JOIN alliances a ON a.alliance_id = h.alliance_id
				WHERE h.corporation_id = $1
				ORDER BY h.start_date DESC`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			for _, row := range history {
				if row["alliance_id"] != nil && row["alliance_name"] == nil {
					row["alliance_name"] = "Unknown"
				}
			}
			return jsonPayload(map[string]any{
				"corporation":     corporation,
				"stats":           scalarStatsMap(alltime, false),
				"recentStats":     recentStatsMap(recent),
				"allianceHistory": history,
			}), nil
		})
}

func registerAllianceDetail(a huma.API, opts Options) {
	registerLegacy(a, entityIDOperation("alliance", "/alliances/{id}", "Alliance detail", "alliances"),
		func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
			id, err := parseID(req.Param("id"))
			if err != nil {
				return legacyPayload{}, err
			}
			alliance, err := queryMap(ctx, opts.DB, `
				SELECT a.alliance_id, a.name, a.ticker, a.date_founded,
				       COALESCE(a.corporation_count, 0) AS corporation_count,
				       COALESCE(a.member_count, 0) AS member_count,
				       a.creator_id, creator.name AS creator_name,
				       a.executor_corporation_id,
				       executor.name AS executor_name,
				       executor.ticker AS executor_ticker,
				       a.faction_id, f.name AS faction_name
				FROM alliances a
				LEFT JOIN characters creator ON creator.character_id = a.creator_id
				LEFT JOIN corporations executor ON executor.corporation_id = a.executor_corporation_id
				LEFT JOIN factions f ON f.faction_id = a.faction_id
				WHERE a.alliance_id = $1 LIMIT 1`, id)
			if err != nil {
				return legacyPayload{}, err
			}
			if alliance == nil {
				return foundOr404(nil, "Alliance not found"), nil
			}
			alltime, err := loadEntityStats(ctx, opts.DB, entityAlliance, id, "alltime")
			if err != nil {
				return legacyPayload{}, err
			}
			recent, err := loadEntityStats(ctx, opts.DB, entityAlliance, id, "90d")
			if err != nil {
				return legacyPayload{}, err
			}
			return jsonPayload(map[string]any{
				"alliance":    alliance,
				"stats":       scalarStatsMap(alltime, false),
				"recentStats": recentStatsMap(recent),
			}), nil
		})
}

func registerEntityMembers(a huma.API, opts Options) {
	for _, spec := range []struct {
		id, path, summary, table, idColumn, filterColumn, selectColumns, cursorColumn string
	}{
		{
			"corporation-members", "/corporations/{id}/members", "Corporation members",
			"characters", "id", "corporation_id",
			"character_id, name, alliance_id, security_status, last_active", "character_id",
		},
		{
			"alliance-corporations", "/alliances/{id}/corporations", "Alliance corporations",
			"corporations", "id", "alliance_id",
			"corporation_id, name, ticker, member_count, ceo_id, date_founded", "corporation_id",
		},
		{
			"alliance-members", "/alliances/{id}/members", "Alliance members",
			"characters", "id", "alliance_id",
			"character_id, name, corporation_id, security_status, last_active", "character_id",
		},
	} {
		registerLegacy(a, entityIDOperation(spec.id, spec.path, spec.summary, strings.Split(spec.path, "/")[1]),
			func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
				id, err := parseID(req.Param(spec.idColumn))
				if err != nil {
					return legacyPayload{}, err
				}
				limit := boundedQueryInt(req, "limit", 50, 1, 1000)
				where := spec.filterColumn + " = $1"
				args := []any{id}
				if after, ok := optionalQueryNumber(req, "after"); ok && after != 0 {
					args = append(args, after)
					where += fmt.Sprintf(" AND %s > $2", spec.cursorColumn)
				}
				args = append(args, limit+1)
				rows, err := queryMaps(ctx, opts.DB,
					"SELECT "+spec.selectColumns+" FROM "+spec.table+
						" WHERE "+where+" ORDER BY "+spec.cursorColumn+
						fmt.Sprintf(" ASC LIMIT $%d", len(args)),
					args...,
				)
				if err != nil {
					return legacyPayload{}, err
				}
				return paginatedRows(rows, limit, spec.cursorColumn), nil
			})
	}
}

func registerEntityKillmailRoutes(a huma.API, opts Options) {
	for _, entity := range []string{"character", "corporation", "alliance"} {
		for _, role := range []string{"kills", "losses"} {
			entity, role := entity, role
			plural := entity + "s"
			registerLegacy(a, entityIDOperation(
				entity+"-"+role,
				"/"+plural+"/{id}/"+role,
				strings.Title(entity)+" "+role,
				plural,
			), func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
				id, err := parseID(req.Param("id"))
				if err != nil {
					return legacyPayload{}, err
				}
				return loadEntityKillmailPage(ctx, opts.DB, entity, id, role, parsePagination(req.Query))
			})
		}
	}
}

func entityIDOperation(id, path, summary, tag string) huma.Operation {
	return huma.Operation{
		OperationID: id,
		Method:      http.MethodGet,
		Path:        path,
		Summary:     summary,
		Tags:        []string{tag},
		Parameters:  idParameter(),
	}
}

func recentStatsMap(stats entityStats) map[string]any {
	return map[string]any{
		"kills":         stats.Kills,
		"losses":        stats.Losses,
		"isk_destroyed": stats.ISKDestroyed,
		"isk_lost":      stats.ISKLost,
	}
}

func loadStandardBreakdowns(
	ctx context.Context,
	db Database,
	entityType int,
	entityID int64,
	window string,
) ([]entityBreakdown, []entityBreakdown, []entityBreakdown, error) {
	flown, err := loadEntityBreakdowns(ctx, db, entityType, entityID, dimShipFlown, window, 10, "kills")
	if err != nil {
		return nil, nil, nil, err
	}
	lost, err := loadEntityBreakdowns(ctx, db, entityType, entityID, dimShipLost, window, 50, "losses")
	if err != nil {
		return nil, nil, nil, err
	}
	systems, err := loadEntityBreakdowns(ctx, db, entityType, entityID, dimSystem, window, 10, "kills")
	if err != nil {
		return nil, nil, nil, err
	}
	return flown, lost, systems, nil
}

func registerEntityStatsRoutes(a huma.API, opts Options) {
	registerLegacy(a, entityIDOperation(
		"character-stats", "/characters/{id}/stats", "Character statistics", "characters",
	), func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		if found, err := entityExists(ctx, opts.DB, "characters", "character_id", id); err != nil {
			return legacyPayload{}, err
		} else if !found {
			return foundOr404(nil, "Character not found"), nil
		}
		statsType := req.Query.Get("type")
		if statsType == "" {
			statsType = "alltime"
		}
		var stats entityStats
		var flown, lost, systems []entityBreakdown
		var period string
		if statsType == "range" {
			from, to := req.Query.Get("from"), req.Query.Get("to")
			if from == "" || to == "" {
				return legacyPayload{}, apiError(
					http.StatusBadRequest,
					"from and to query params required for type=range",
				)
			}
			period = from + " to " + to
			stats, err = loadRangeStats(ctx, opts.DB, entityCharacter, id, from, to)
			if err == nil {
				flown, err = loadRangeBreakdowns(ctx, opts.DB, entityCharacter, id, dimShipFlown, from, to, "kills", 10)
			}
			if err == nil {
				lost, err = loadRangeBreakdowns(ctx, opts.DB, entityCharacter, id, dimShipLost, from, to, "losses", 10)
			}
			if err == nil {
				systems, err = loadRangeBreakdowns(ctx, opts.DB, entityCharacter, id, dimSystem, from, to, "kills", 10)
			}
		} else {
			window := "alltime"
			period = "alltime"
			if statsType == "weekly" {
				window, period = "7d", "weekly"
			}
			stats, err = loadEntityStats(ctx, opts.DB, entityCharacter, id, window)
			if err == nil {
				flown, lost, systems, err = loadStandardBreakdowns(
					ctx, opts.DB, entityCharacter, id, window,
				)
			}
		}
		if err != nil {
			return legacyPayload{}, err
		}
		topShips, topSystems, err := assembleTopBreakdowns(ctx, opts.DB, flown, lost, systems)
		if err != nil {
			return legacyPayload{}, err
		}
		body := scalarStatsMap(stats, true)
		body["character_id"] = id
		body["period"] = period
		body["topShips"] = topShips
		body["topSystems"] = topSystems
		return jsonPayload(body), nil
	})

	for _, spec := range []struct {
		id, path, summary, table, idColumn, notFound, membership string
		entityType                                               int
	}{
		{
			"corporation-stats-alltime", "/corporations/{id}/stats/alltime",
			"Corporation all-time statistics", "corporations", "corporation_id",
			"Corporation not found", "corporation_id", entityCorporation,
		},
		{
			"corporation-stats-weekly", "/corporations/{id}/stats/weekly",
			"Corporation weekly statistics", "corporations", "corporation_id",
			"Corporation not found", "corporation_id", entityCorporation,
		},
		{
			"alliance-stats-alltime", "/alliances/{id}/stats/alltime",
			"Alliance all-time statistics", "alliances", "alliance_id",
			"Alliance not found", "alliance_id", entityAlliance,
		},
		{
			"alliance-stats-weekly", "/alliances/{id}/stats/weekly",
			"Alliance weekly statistics", "alliances", "alliance_id",
			"Alliance not found", "alliance_id", entityAlliance,
		},
	} {
		registerLegacy(a, entityIDOperation(spec.id, spec.path, spec.summary, spec.table),
			func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
				id, err := parseID(req.Param("id"))
				if err != nil {
					return legacyPayload{}, err
				}
				found, err := entityExists(ctx, opts.DB, spec.table, spec.idColumn, id)
				if err != nil {
					return legacyPayload{}, err
				}
				if !found {
					return foundOr404(nil, spec.notFound), nil
				}
				window, period := "alltime", "alltime"
				if strings.HasSuffix(spec.path, "/weekly") {
					window, period = "7d", "weekly"
				}
				stats, err := loadEntityStats(ctx, opts.DB, spec.entityType, id, window)
				if err != nil {
					return legacyPayload{}, err
				}
				members, err := loadTopMembers(ctx, opts.DB, spec.membership, id, window)
				if err != nil {
					return legacyPayload{}, err
				}
				flown, lost, systems, err := loadStandardBreakdowns(
					ctx, opts.DB, spec.entityType, id, window,
				)
				if err != nil {
					return legacyPayload{}, err
				}
				topShips, topSystems, err := assembleTopBreakdowns(ctx, opts.DB, flown, lost, systems)
				if err != nil {
					return legacyPayload{}, err
				}
				body := scalarStatsMap(stats, false)
				if spec.entityType == entityCorporation {
					body["corporation_id"] = id
				} else {
					body["alliance_id"] = id
				}
				body["period"] = period
				body["topMembers"] = members
				body["topShips"] = topShips
				body["topSystems"] = topSystems
				return jsonPayload(body), nil
			})
	}
}

func entityExists(
	ctx context.Context,
	db Database,
	table, idColumn string,
	id int64,
) (bool, error) {
	row, err := queryMap(ctx, db,
		`SELECT `+idColumn+` FROM `+table+` WHERE `+idColumn+` = $1 LIMIT 1`,
		id,
	)
	return row != nil, err
}
