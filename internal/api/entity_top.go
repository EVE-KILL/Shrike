package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/stats"
	"golang.org/x/sync/errgroup"
)

const entityTopCacheTTL = 5 * time.Minute

type entityTopPeriod struct {
	Days    float64
	AllTime bool
}

func registerEntityTopRoutes(a huma.API, opts Options) {
	for _, kind := range []string{
		entityPageCharacter, entityPageCorporation, entityPageAlliance,
	} {
		registerLegacy(a, huma.Operation{
			OperationID: "entity-top-" + kind + "-compat",
			Method:      http.MethodGet,
			Path:        "/" + kind + "/{id}/top",
			Summary:     "Entity top activity lists",
			Tags:        []string{"entities", "statistics"},
		}, cacheEntityTopRoute(opts, kind, entityTopHandler(opts, kind)))
	}
}

func cacheEntityTopRoute(
	opts Options,
	fixedType string,
	next legacyHandler,
) legacyHandler {
	return routeJSONCacheBy(
		opts,
		entityTopCacheTTL,
		"public, max-age=60, s-maxage=300, stale-while-revalidate=300",
		func(req *legacyRequest) string {
			kind := fixedType
			if kind == "" {
				kind = req.Param("type")
			}
			return "entities:top:" + kind + ":" + req.Param("id") +
				"?" + req.Query.Encode()
		},
		next,
	)
}

func entityTopHandler(opts Options, fixedType string) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		kind := fixedType
		if kind == "" {
			kind = strings.TrimSpace(req.Param("type"))
		}
		entityType, ok := entityPageStatsType(kind)
		if !ok {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid entity type",
			)
		}
		id, err := parseUniverseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		slice, period := parseEntityTopParams(req)
		body, err := loadEntityTop(
			ctx, opts.DB, kind, entityType, id, slice, period,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(body), nil
	}
}

func parseEntityTopParams(req *legacyRequest) (string, entityTopPeriod) {
	slice := "left"
	if req.Query.Get("slice") == "right" {
		slice = "right"
	}
	if slice == "right" && req.Query.Get("days") == "alltime" {
		return slice, entityTopPeriod{AllTime: true}
	}
	days := float64(7)
	raw := strings.TrimSpace(req.Query.Get("days"))
	if value, err := strconv.ParseFloat(raw, 64); err == nil &&
		value > 0 && !math.IsInf(value, 0) && !math.IsNaN(value) {
		days = min(365.0, max(1.0/24.0, value))
	}
	return slice, entityTopPeriod{Days: days}
}

func entityTopEmpty() map[string]any {
	return map[string]any{
		"charactersByKills":    []map[string]any{},
		"charactersByPoints":   []map[string]any{},
		"charactersByIsk":      []map[string]any{},
		"soloKillers":          []map[string]any{},
		"corporationsByKills":  []map[string]any{},
		"shipsUsed":            []map[string]any{},
		"systems":              []map[string]any{},
		"constellations":       []map[string]any{},
		"regions":              []map[string]any{},
		"killedCorporations":   []map[string]any{},
		"killedAlliances":      []map[string]any{},
		"killedByCorporations": []map[string]any{},
		"killedByAlliances":    []map[string]any{},
		"achievementPoints":    []map[string]any{},
		"recentMembers":        []map[string]any{},
	}
}

func loadEntityTop(
	ctx context.Context,
	db Database,
	kind string,
	entityType int,
	id int64,
	slice string,
	period entityTopPeriod,
) (map[string]any, error) {
	if slice == "right" {
		return loadEntityTopRight(ctx, db, kind, entityType, id, period)
	}
	return loadEntityTopLeft(ctx, db, kind, id, period)
}

func loadEntityTopLeft(
	ctx context.Context,
	db Database,
	kind string,
	id int64,
	period entityTopPeriod,
) (map[string]any, error) {
	output := entityTopEmpty()
	if kind == entityPageCharacter || period.AllTime {
		return output, nil
	}
	since := entityTopSince(period.Days, time.Now().UTC())
	attackerColumn := entityTopAttackerColumn(kind)
	memberColumn := "corporation_id"
	if kind == entityPageAlliance {
		memberColumn = "alliance_id"
	}
	falseQuery := `
		SELECT NULL::bigint AS id, NULL::text AS name,
		       NULL::double precision AS count
		WHERE false`
	queries := []databaseQuery{
		{
			SQL: fmt.Sprintf(`
				SELECT attacker.character_id AS id, character.name,
				       COUNT(*)::bigint AS count
				FROM killmail_attackers attacker
				JOIN characters character
				  ON character.character_id = attacker.character_id
				WHERE attacker.%s = $1
				  AND attacker.killmail_time >= $2
				  AND attacker.character_id IS NOT NULL
				GROUP BY attacker.character_id, character.name
				ORDER BY count DESC
				LIMIT 10`, attackerColumn),
			Args: []any{id, since},
		},
		{
			SQL: fmt.Sprintf(`
				SELECT attacker.character_id AS id, character.name,
				       COALESCE(SUM(attacker.points), 0)::bigint AS count
				FROM killmail_attackers attacker
				JOIN characters character
				  ON character.character_id = attacker.character_id
				WHERE attacker.%s = $1
				  AND attacker.killmail_time >= $2
				  AND attacker.character_id IS NOT NULL
				GROUP BY attacker.character_id, character.name
				ORDER BY count DESC
				LIMIT 10`, attackerColumn),
			Args: []any{id, since},
		},
		{
			SQL: fmt.Sprintf(`
				SELECT attacker.character_id AS id, character.name,
				       COALESCE(SUM(killmail.total_value), 0)::double precision
				         AS count
				FROM killmail_attackers attacker
				JOIN killmails killmail
				  ON killmail.killmail_id = attacker.killmail_id
				JOIN characters character
				  ON character.character_id = attacker.character_id
				WHERE attacker.%s = $1
				  AND attacker.killmail_time >= $2
				  AND attacker.character_id IS NOT NULL
				  AND attacker.final_blow = true
				GROUP BY attacker.character_id, character.name
				ORDER BY count DESC
				LIMIT 10`, attackerColumn),
			Args: []any{id, since},
		},
		{
			SQL: fmt.Sprintf(`
				SELECT attacker.character_id AS id, character.name,
				       COUNT(*)::bigint AS count
				FROM killmail_attackers attacker
				JOIN killmails killmail
				  ON killmail.killmail_id = attacker.killmail_id
				JOIN characters character
				  ON character.character_id = attacker.character_id
				WHERE attacker.%s = $1
				  AND attacker.killmail_time >= $2
				  AND attacker.character_id IS NOT NULL
				  AND attacker.final_blow = true
				  AND killmail.is_solo = true
				GROUP BY attacker.character_id, character.name
				ORDER BY count DESC
				LIMIT 10`, attackerColumn),
			Args: []any{id, since},
		},
		{SQL: falseQuery},
		{
			SQL: fmt.Sprintf(`
				SELECT character_id AS id, name,
				       achievement_points::bigint AS count
				FROM characters
				WHERE %s = $1 AND achievement_points > 0
				ORDER BY achievement_points DESC
				LIMIT 10`, memberColumn),
			Args: []any{id},
		},
		{SQL: falseQuery},
	}
	if kind == entityPageAlliance {
		queries[4] = databaseQuery{
			SQL: `
				SELECT attacker.corporation_id AS id, corporation.name,
				       corporation.palette,
				       COUNT(DISTINCT attacker.killmail_id)::bigint AS count
				FROM killmail_attackers attacker
				JOIN corporations corporation
				  ON corporation.corporation_id = attacker.corporation_id
				WHERE attacker.alliance_id = $1
				  AND attacker.killmail_time >= $2
				  AND attacker.corporation_id IS NOT NULL
				GROUP BY attacker.corporation_id, corporation.name,
				         corporation.palette
				ORDER BY count DESC
				LIMIT 10`,
			Args: []any{id, since},
		}
	}
	if kind == entityPageCorporation {
		queries[6] = databaseQuery{
			SQL: `
				SELECT history.character_id AS id, character.name,
				       EXTRACT(EPOCH FROM history.start_date)::bigint AS count
				FROM character_corporation_history history
				JOIN characters character
				  ON character.character_id = history.character_id
				WHERE history.corporation_id = $1
				ORDER BY history.start_date DESC
				LIMIT 10`,
			Args: []any{id},
		}
	}
	results, err := queryMapsConcurrent(ctx, db, queries...)
	if err != nil {
		return nil, err
	}
	keys := []string{
		"charactersByKills", "charactersByPoints", "charactersByIsk",
		"soloKillers", "corporationsByKills", "achievementPoints",
		"recentMembers",
	}
	for i, key := range keys {
		output[key] = normalizeEntityTopRows(results[i])
	}
	for _, key := range []string{
		"charactersByKills", "charactersByPoints", "charactersByIsk",
		"soloKillers", "achievementPoints", "recentMembers",
	} {
		rows := output[key].([]map[string]any)
		if err := attachEntityTopPalettes(ctx, db, rows, "character"); err != nil {
			return nil, err
		}
	}
	return output, nil
}

func entityTopSince(days float64, now time.Time) time.Time {
	if days < 1 {
		minutes := max(1, int(math.Floor(days*24*60+0.5)))
		return now.Add(-time.Duration(minutes) * time.Minute)
	}
	wholeDays := max(1, int(math.Floor(days+0.5)))
	return now.AddDate(0, 0, -wholeDays)
}

func entityTopAttackerColumn(kind string) string {
	switch kind {
	case entityPageCharacter:
		return "character_id"
	case entityPageCorporation:
		return "corporation_id"
	default:
		return "alliance_id"
	}
}

func entityTopVictimColumn(kind string) string {
	switch kind {
	case entityPageCharacter:
		return "victim_character_id"
	case entityPageCorporation:
		return "victim_corporation_id"
	default:
		return "victim_alliance_id"
	}
}

func normalizeEntityTopRows(rows []map[string]any) []map[string]any {
	output := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := int64OrZero(row["id"])
		if id == 0 {
			continue
		}
		name := stringOrEmpty(row["name"])
		if name == "" {
			name = "Unknown"
		}
		count := float64OrZero(row["count"])
		item := map[string]any{"id": id, "name": name, "count": count}
		if palette, exists := row["palette"]; exists {
			item["palette"] = palette
		}
		output = append(output, item)
	}
	return output
}

func attachEntityTopPalettes(
	ctx context.Context,
	db Database,
	rows []map[string]any,
	kind string,
) error {
	if len(rows) == 0 {
		return nil
	}
	idSet := map[int32]struct{}{}
	for _, row := range rows {
		id := int64OrZero(row["id"])
		if id > 0 && id <= math.MaxInt32 {
			idSet[int32(id)] = struct{}{}
		}
	}
	ids := make([]int32, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	var query string
	switch kind {
	case "character":
		query = `
			SELECT character.character_id AS id, corporation.palette
			FROM characters character
			LEFT JOIN corporations corporation
			  ON corporation.corporation_id = character.corporation_id
			WHERE character.character_id = ANY($1::int[])`
	case "corporation":
		query = `
			SELECT corporation_id AS id, palette
			FROM corporations WHERE corporation_id = ANY($1::int[])`
	case "alliance":
		query = `
			SELECT alliance.alliance_id AS id, corporation.palette
			FROM alliances alliance
			LEFT JOIN corporations corporation
			  ON corporation.corporation_id =
			     alliance.executor_corporation_id
			WHERE alliance.alliance_id = ANY($1::int[])`
	default:
		return nil
	}
	paletteRows, err := queryMaps(ctx, db, query, ids)
	if err != nil {
		return err
	}
	palettes := map[int64]any{}
	for _, row := range paletteRows {
		palettes[int64OrZero(row["id"])] = row["palette"]
	}
	for _, row := range rows {
		row["palette"] = palettes[int64OrZero(row["id"])]
	}
	return nil
}

type entityTopRightData struct {
	Ships              []entityBreakdown
	SystemKills        []entityBreakdown
	SystemLosses       []entityBreakdown
	RegionKills        []entityBreakdown
	RegionLosses       []entityBreakdown
	KilledCorporations []entityBreakdown
	KilledAlliances    []entityBreakdown
	DiedCorporations   []entityBreakdown
	DiedAlliances      []entityBreakdown
	Constellations     []map[string]any
}

func loadEntityTopRight(
	ctx context.Context,
	db Database,
	kind string,
	entityType int,
	id int64,
	period entityTopPeriod,
) (map[string]any, error) {
	if !period.AllTime && period.Days < 1 {
		return loadEntityTopRightSubDay(ctx, db, kind, id, period.Days)
	}
	output := entityTopEmpty()
	window := entityTopStatsWindow(period)
	data, err := loadEntityTopRightBreakdowns(
		ctx, db, entityType, id, window,
	)
	if err != nil {
		return nil, err
	}
	output["shipsUsed"], err = entityTopBreakdownRows(
		ctx, db, data.Ships, "kills", "type",
	)
	if err != nil {
		return nil, err
	}
	output["systems"], err = entityTopMergedBreakdownRows(
		ctx, db, data.SystemKills, data.SystemLosses, "system",
	)
	if err != nil {
		return nil, err
	}
	output["regions"], err = entityTopMergedBreakdownRows(
		ctx, db, data.RegionKills, data.RegionLosses, "region",
	)
	if err != nil {
		return nil, err
	}
	output["constellations"] = normalizeEntityTopRows(data.Constellations)
	for _, item := range []struct {
		Key, Metric, Kind string
		Rows              []entityBreakdown
	}{
		{"killedCorporations", "kills", "corporation", data.KilledCorporations},
		{"killedAlliances", "kills", "alliance", data.KilledAlliances},
		{"killedByCorporations", "losses", "corporation", data.DiedCorporations},
		{"killedByAlliances", "losses", "alliance", data.DiedAlliances},
	} {
		rows, err := entityTopBreakdownRows(
			ctx, db, item.Rows, item.Metric, item.Kind,
		)
		if err != nil {
			return nil, err
		}
		output[item.Key] = rows
	}
	return output, nil
}

func entityTopStatsWindow(period entityTopPeriod) string {
	if period.AllTime {
		return "alltime"
	}
	switch {
	case period.Days <= 1:
		return "1d"
	case period.Days <= 7:
		return "7d"
	case period.Days <= 14:
		return "14d"
	case period.Days <= 30:
		return "30d"
	case period.Days <= 90:
		return "90d"
	case period.Days <= 180:
		return "180d"
	default:
		return "365d"
	}
}

func loadEntityTopRightBreakdowns(
	ctx context.Context,
	db Database,
	entityType int,
	id int64,
	window string,
) (entityTopRightData, error) {
	var data entityTopRightData
	group, groupCtx := errgroup.WithContext(ctx)
	load := func(
		target *[]entityBreakdown,
		dimension stats.DimCategory,
		limit int,
		order string,
	) {
		group.Go(func() (err error) {
			*target, err = loadEntityBreakdowns(
				groupCtx, db, entityType, id, int(dimension),
				window, limit, order,
			)
			return err
		})
	}
	load(&data.Ships, stats.DimShipFlown, 10, "kills")
	load(&data.SystemKills, stats.DimSystem, 20, "kills")
	load(&data.SystemLosses, stats.DimSystem, 20, "losses")
	load(&data.RegionKills, stats.DimRegion, 20, "kills")
	load(&data.RegionLosses, stats.DimRegion, 20, "losses")
	load(&data.KilledCorporations, stats.DimKilledCorporation, 10, "kills")
	load(&data.KilledAlliances, stats.DimKilledAlliance, 10, "kills")
	load(&data.DiedCorporations, stats.DimDiesToCorporation, 10, "losses")
	load(&data.DiedAlliances, stats.DimDiesToAlliance, 10, "losses")
	group.Go(func() (err error) {
		periodType, fromDate := statsWindow(window)
		args := []any{
			entityType, id, stats.DimSystem, periodType,
		}
		dateFilter := ""
		if fromDate != "" {
			args = append(args, fromDate)
			dateFilter = fmt.Sprintf(
				" AND breakdown.period_start >= $%d::date", len(args),
			)
		}
		data.Constellations, err = queryMaps(groupCtx, db, fmt.Sprintf(`
			SELECT system.constellation_id AS id,
			       constellation.constellation_name AS name,
			       COALESCE(SUM(
			         breakdown.kills + breakdown.losses
			       ), 0)::bigint AS count
			FROM stats_breakdowns breakdown
			JOIN solar_systems system
			  ON system.solar_system_id = breakdown.dim_id
			JOIN constellations constellation
			  ON constellation.constellation_id = system.constellation_id
			WHERE breakdown.entity_type = $1
			  AND breakdown.entity_id = $2
			  AND breakdown.dim_category = $3
			  AND breakdown.period_type = $4%s
			GROUP BY system.constellation_id,
			         constellation.constellation_name
			ORDER BY count DESC
			LIMIT 10`, dateFilter), args...)
		return err
	})
	if err := group.Wait(); err != nil {
		return entityTopRightData{}, err
	}
	return data, nil
}

func entityTopBreakdownRows(
	ctx context.Context,
	db Database,
	breakdowns []entityBreakdown,
	metric, kind string,
) ([]map[string]any, error) {
	rows := make([]map[string]any, 0, len(breakdowns))
	for _, breakdown := range breakdowns {
		count := breakdown.Kills
		if metric == "losses" {
			count = breakdown.Losses
		}
		rows = append(rows, map[string]any{
			"id": breakdown.DimID, "count": float64(count),
		})
	}
	if err := attachEntityTopNames(ctx, db, rows, kind); err != nil {
		return nil, err
	}
	return rows, nil
}

func entityTopMergedBreakdownRows(
	ctx context.Context,
	db Database,
	kills, losses []entityBreakdown,
	kind string,
) ([]map[string]any, error) {
	counts := map[int64]int64{}
	for _, row := range kills {
		counts[row.DimID] = row.Kills + row.Losses
	}
	for _, row := range losses {
		if _, exists := counts[row.DimID]; !exists {
			counts[row.DimID] = row.Kills + row.Losses
		}
	}
	rows := make([]map[string]any, 0, len(counts))
	for id, count := range counts {
		rows = append(rows, map[string]any{
			"id": id, "count": float64(count),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		left, right := float64OrZero(rows[i]["count"]),
			float64OrZero(rows[j]["count"])
		if left == right {
			return int64OrZero(rows[i]["id"]) < int64OrZero(rows[j]["id"])
		}
		return left > right
	})
	if len(rows) > 10 {
		rows = rows[:10]
	}
	if err := attachEntityTopNames(ctx, db, rows, kind); err != nil {
		return nil, err
	}
	return rows, nil
}

func attachEntityTopNames(
	ctx context.Context,
	db Database,
	rows []map[string]any,
	kind string,
) error {
	if len(rows) == 0 {
		return nil
	}
	idSet := map[int32]struct{}{}
	for _, row := range rows {
		id := int64OrZero(row["id"])
		if id > 0 && id <= math.MaxInt32 {
			idSet[int32(id)] = struct{}{}
		}
	}
	ids := make([]int32, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	specs := map[string]struct {
		Table, ID, Name string
	}{
		"type":   {"inv_types", "type_id", "name"},
		"system": {"solar_systems", "solar_system_id", "system_name"},
		"region": {"regions", "region_id", "name"},
	}
	if spec, ok := specs[kind]; ok {
		nameRows, err := queryMaps(ctx, db, fmt.Sprintf(`
			SELECT %s AS id, %s AS name
			FROM %s WHERE %s = ANY($1::int[])`,
			spec.ID, spec.Name, spec.Table, spec.ID,
		), ids)
		if err != nil {
			return err
		}
		names := map[int64]string{}
		for _, row := range nameRows {
			names[int64OrZero(row["id"])] = stringOrEmpty(row["name"])
		}
		for _, row := range rows {
			id := int64OrZero(row["id"])
			row["name"] = fallbackName(names[id], "Unknown", id)
			if names[id] == "" {
				row["name"] = "Unknown"
			}
		}
		return nil
	}
	if err := attachEntityTopPalettes(ctx, db, rows, kind); err != nil {
		return err
	}
	var table, idColumn string
	switch kind {
	case "corporation":
		table, idColumn = "corporations", "corporation_id"
	case "alliance":
		table, idColumn = "alliances", "alliance_id"
	default:
		return nil
	}
	nameRows, err := queryMaps(ctx, db, fmt.Sprintf(`
		SELECT %s AS id, name FROM %s
		WHERE %s = ANY($1::int[])`,
		idColumn, table, idColumn,
	), ids)
	if err != nil {
		return err
	}
	names := map[int64]string{}
	for _, row := range nameRows {
		names[int64OrZero(row["id"])] = stringOrEmpty(row["name"])
	}
	for _, row := range rows {
		if name := names[int64OrZero(row["id"])]; name != "" {
			row["name"] = name
		} else {
			row["name"] = "Unknown"
		}
	}
	return nil
}

func loadEntityTopRightSubDay(
	ctx context.Context,
	db Database,
	kind string,
	id int64,
	days float64,
) (map[string]any, error) {
	output := entityTopEmpty()
	since := entityTopSince(days, time.Now().UTC())
	attackerColumn := entityTopAttackerColumn(kind)
	victimColumn := entityTopVictimColumn(kind)
	queries := []databaseQuery{
		{
			SQL: fmt.Sprintf(`
				SELECT attacker.ship_type_id AS id, type.name,
				       COUNT(*)::bigint AS count
				FROM killmail_attackers attacker
				JOIN inv_types type ON type.type_id = attacker.ship_type_id
				WHERE attacker.%s = $1
				  AND attacker.killmail_time >= $2
				  AND attacker.ship_type_id IS NOT NULL
				GROUP BY attacker.ship_type_id, type.name
				ORDER BY count DESC LIMIT 10`, attackerColumn),
			Args: []any{id, since},
		},
		{
			SQL: fmt.Sprintf(`
				SELECT killmail.solar_system_id AS id,
				       system.system_name AS name,
				       COUNT(DISTINCT killmail.killmail_id)::bigint AS count
				FROM killmail_attackers attacker
				JOIN killmails killmail
				  ON killmail.killmail_id = attacker.killmail_id
				JOIN solar_systems system
				  ON system.solar_system_id = killmail.solar_system_id
				WHERE attacker.%s = $1
				  AND attacker.killmail_time >= $2
				GROUP BY killmail.solar_system_id, system.system_name
				ORDER BY count DESC LIMIT 10`, attackerColumn),
			Args: []any{id, since},
		},
		{
			SQL: fmt.Sprintf(`
				SELECT killmail.constellation_id AS id,
				       constellation.constellation_name AS name,
				       COUNT(DISTINCT killmail.killmail_id)::bigint AS count
				FROM killmail_attackers attacker
				JOIN killmails killmail
				  ON killmail.killmail_id = attacker.killmail_id
				JOIN constellations constellation
				  ON constellation.constellation_id =
				     killmail.constellation_id
				WHERE attacker.%s = $1
				  AND attacker.killmail_time >= $2
				  AND killmail.constellation_id IS NOT NULL
				GROUP BY killmail.constellation_id,
				         constellation.constellation_name
				ORDER BY count DESC LIMIT 10`, attackerColumn),
			Args: []any{id, since},
		},
		{
			SQL: fmt.Sprintf(`
				SELECT killmail.region_id AS id, region.name,
				       COUNT(DISTINCT killmail.killmail_id)::bigint AS count
				FROM killmail_attackers attacker
				JOIN killmails killmail
				  ON killmail.killmail_id = attacker.killmail_id
				JOIN regions region ON region.region_id = killmail.region_id
				WHERE attacker.%s = $1
				  AND attacker.killmail_time >= $2
				  AND killmail.region_id IS NOT NULL
				GROUP BY killmail.region_id, region.name
				ORDER BY count DESC LIMIT 10`, attackerColumn),
			Args: []any{id, since},
		},
	}
	for _, target := range []struct {
		Category   string
		TargetType string
	}{
		{"killed", "corporation"},
		{"killed", "alliance"},
		{"died", "corporation"},
		{"died", "alliance"},
	} {
		queries = append(queries, entityTopSubDayInteractionQuery(
			attackerColumn, victimColumn, target.Category,
			target.TargetType, id, since,
		))
	}
	results, err := queryMapsConcurrent(ctx, db, queries...)
	if err != nil {
		return nil, err
	}
	keys := []string{
		"shipsUsed", "systems", "constellations", "regions",
		"killedCorporations", "killedAlliances",
		"killedByCorporations", "killedByAlliances",
	}
	for i, key := range keys {
		output[key] = normalizeEntityTopRows(results[i])
	}
	for _, item := range []struct {
		Key, Kind string
	}{
		{"killedCorporations", "corporation"},
		{"killedAlliances", "alliance"},
		{"killedByCorporations", "corporation"},
		{"killedByAlliances", "alliance"},
	} {
		rows := output[item.Key].([]map[string]any)
		if err := attachEntityTopPalettes(ctx, db, rows, item.Kind); err != nil {
			return nil, err
		}
	}
	return output, nil
}

func entityTopSubDayInteractionQuery(
	attackerColumn, victimColumn, category, targetType string,
	id int64,
	since time.Time,
) databaseQuery {
	targetVictimColumn := "victim_corporation_id"
	targetAttackerColumn := "corporation_id"
	table, tableID := "corporations", "corporation_id"
	if targetType == "alliance" {
		targetVictimColumn = "victim_alliance_id"
		targetAttackerColumn = "alliance_id"
		table, tableID = "alliances", "alliance_id"
	}
	if category == "killed" {
		return databaseQuery{
			SQL: fmt.Sprintf(`
				SELECT killmail.%s AS id, target.name,
				       COUNT(DISTINCT killmail.killmail_id)::bigint AS count
				FROM killmail_attackers attacker
				JOIN killmails killmail
				  ON killmail.killmail_id = attacker.killmail_id
				LEFT JOIN %s target
				  ON target.%s = killmail.%s
				WHERE attacker.%s = $1
				  AND attacker.killmail_time >= $2
				  AND killmail.%s IS NOT NULL
				GROUP BY killmail.%s, target.name
				ORDER BY count DESC LIMIT 10`,
				targetVictimColumn, table, tableID, targetVictimColumn,
				attackerColumn, targetVictimColumn, targetVictimColumn,
			),
			Args: []any{id, since},
		}
	}
	return databaseQuery{
		SQL: fmt.Sprintf(`
			SELECT attacker.%s AS id, target.name,
			       COUNT(DISTINCT attacker.killmail_id)::bigint AS count
			FROM killmails killmail
			JOIN killmail_attackers attacker
			  ON attacker.killmail_id = killmail.killmail_id
			LEFT JOIN %s target ON target.%s = attacker.%s
			WHERE killmail.%s = $1
			  AND killmail.killmail_time >= $2
			  AND attacker.%s IS NOT NULL
			GROUP BY attacker.%s, target.name
			ORDER BY count DESC LIMIT 10`,
			targetAttackerColumn, table, tableID, targetAttackerColumn,
			victimColumn, targetAttackerColumn, targetAttackerColumn,
		),
		Args: []any{id, since},
	}
}
