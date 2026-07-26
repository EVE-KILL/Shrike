package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/sync/errgroup"
)

type coalitionSide struct {
	Label        string
	Alliances    []int32
	Corporations []int32
}

type coalitionTotals struct {
	Kills        int64
	Losses       int64
	SoloKills    int64
	NPCLosses    int64
	ISKDestroyed float64
	ISKLost      float64
	Points       int64
	FinalBlows   int64
}

type coalitionPairwise struct {
	AKillsCount  int64
	AKillsISK    float64
	ALossesCount int64
	ALossesISK   float64
}

func registerCoalitionRoute(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "coalition-stats",
		Method:      http.MethodPost,
		Path:        "/coalitions/stats",
		Summary:     "Coalition versus coalition statistics",
		Tags:        []string{"stats"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		body, err := parseCoalitionBody(req)
		if err != nil {
			return legacyPayload{}, err
		}
		sideA, err := normalizeCoalitionSide(ctx, opts.DB, body.SideA)
		if err != nil {
			return legacyPayload{}, err
		}
		sideB, err := normalizeCoalitionSide(ctx, opts.DB, body.SideB)
		if err != nil {
			return legacyPayload{}, err
		}
		if overlap := coalitionOverlap(sideA, sideB); overlap != "" {
			return legacyPayload{}, apiError(http.StatusBadRequest, overlap)
		}
		output, err := loadCoalitionStats(
			ctx, opts.DB, sideA, sideB,
			body.Mode, body.PeriodDays, body.From, body.To,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(output), nil
	})
}

type coalitionRequest struct {
	SideA      coalitionSide
	SideB      coalitionSide
	Mode       string
	PeriodDays int
	From       string
	To         string
}

func parseCoalitionBody(req *legacyRequest) (coalitionRequest, error) {
	var body map[string]any
	decoder := json.NewDecoder(req.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return coalitionRequest{}, apiError(http.StatusBadRequest, "Invalid JSON body")
	}
	sideA, err := parseCoalitionSide(body["sideA"], "sideA")
	if err != nil {
		return coalitionRequest{}, err
	}
	sideB, err := parseCoalitionSide(body["sideB"], "sideB")
	if err != nil {
		return coalitionRequest{}, err
	}
	result := coalitionRequest{SideA: sideA, SideB: sideB}
	if raw, ok := body["date"]; ok && jsTruthy(raw) {
		date, ok := raw.(string)
		if !ok || !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(date) {
			return coalitionRequest{}, apiError(
				http.StatusBadRequest, "date must be YYYY-MM-DD",
			)
		}
		result.Mode, result.PeriodDays = "daily", 1
		result.From, result.To = date, date
		return result, nil
	}
	days := 30
	if raw, ok := body["days"]; ok {
		if value, valid := jsNumber(raw); valid && value != 0 {
			days = int(value)
		}
	}
	days = max(1, min(90, days))
	now := time.Now().UTC()
	result.Mode, result.PeriodDays = "lookback", days
	result.From = now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	result.To = now.Format("2006-01-02")
	return result, nil
}

func parseCoalitionSide(raw any, name string) (coalitionSide, error) {
	value, ok := raw.(map[string]any)
	if !ok {
		return coalitionSide{}, apiError(http.StatusBadRequest, name+" required")
	}
	alliances, err := coalitionIDs(value["alliances"], name+".alliances")
	if err != nil {
		return coalitionSide{}, err
	}
	corporations, err := coalitionIDs(value["corporations"], name+".corporations")
	if err != nil {
		return coalitionSide{}, err
	}
	if len(alliances)+len(corporations) == 0 {
		return coalitionSide{}, apiError(
			http.StatusBadRequest,
			name+" must include at least one alliance or corporation",
		)
	}
	if len(alliances)+len(corporations) > 100 {
		return coalitionSide{}, apiError(
			http.StatusBadRequest, name+" exceeds 100 entities",
		)
	}
	label := name
	if text, ok := value["label"].(string); ok {
		if len(text) > 120 {
			text = text[:120]
		}
		label = text
	}
	return coalitionSide{
		Label: label, Alliances: alliances, Corporations: corporations,
	}, nil
}

func coalitionIDs(raw any, field string) ([]int32, error) {
	if raw == nil {
		return []int32{}, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, apiError(http.StatusBadRequest, field+" must be an array")
	}
	result := []int32{}
	seen := map[int32]struct{}{}
	for _, rawID := range values {
		number, valid := jsNumber(rawID)
		if !valid || number <= 0 || math.Trunc(number) != number {
			return nil, apiError(http.StatusBadRequest,
				fmt.Sprintf("%s contains invalid id: %v", field, rawID))
		}
		id := int32(number)
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result, nil
}

func normalizeCoalitionSide(
	ctx context.Context,
	db Database,
	side coalitionSide,
) (coalitionSide, error) {
	if len(side.Alliances) == 0 || len(side.Corporations) == 0 {
		return side, nil
	}
	rows, err := queryMaps(ctx, db, `
		SELECT corporation_id, alliance_id FROM corporations
		WHERE corporation_id = ANY($1::int[])`, side.Corporations)
	if err != nil {
		return coalitionSide{}, err
	}
	membership := map[int32]int32{}
	for _, row := range rows {
		corporation, _ := int64Value(row["corporation_id"])
		alliance, ok := int64Value(row["alliance_id"])
		if ok {
			membership[int32(corporation)] = int32(alliance)
		}
	}
	listed := map[int32]struct{}{}
	for _, id := range side.Alliances {
		listed[id] = struct{}{}
	}
	filtered := []int32{}
	for _, id := range side.Corporations {
		alliance, belongs := membership[id]
		_, covered := listed[alliance]
		if !belongs || !covered {
			filtered = append(filtered, id)
		}
	}
	side.Corporations = filtered
	return side, nil
}

func coalitionOverlap(a, b coalitionSide) string {
	allianceSet := map[int32]struct{}{}
	for _, id := range a.Alliances {
		allianceSet[id] = struct{}{}
	}
	for _, id := range b.Alliances {
		if _, exists := allianceSet[id]; exists {
			return fmt.Sprintf("Alliance %d appears on both sides", id)
		}
	}
	corporationSet := map[int32]struct{}{}
	for _, id := range a.Corporations {
		corporationSet[id] = struct{}{}
	}
	for _, id := range b.Corporations {
		if _, exists := corporationSet[id]; exists {
			return fmt.Sprintf("Corporation %d appears on both sides", id)
		}
	}
	return ""
}

func loadCoalitionStats(
	ctx context.Context,
	db Database,
	sideA, sideB coalitionSide,
	mode string,
	periodDays int,
	from, to string,
) (map[string]any, error) {
	var (
		pairwise           coalitionPairwise
		totalsA, totalsB   coalitionTotals
		shipsA, shipsB     []map[string]any
		systemsA, regionsA []int64
		systemsB, regionsB []int64
		daily              []map[string]any
	)
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		pairwise, err = loadCoalitionPairwise(
			groupCtx, db, sideA, sideB, from, to,
		)
		return
	})
	group.Go(func() (err error) {
		totalsA, err = loadCoalitionTotals(groupCtx, db, sideA, from, to)
		return
	})
	group.Go(func() (err error) {
		totalsB, err = loadCoalitionTotals(groupCtx, db, sideB, from, to)
		return
	})
	group.Go(func() (err error) {
		shipsA, err = coalitionTopShips(groupCtx, db, sideA, from, to)
		return
	})
	group.Go(func() (err error) {
		shipsB, err = coalitionTopShips(groupCtx, db, sideB, from, to)
		return
	})
	group.Go(func() (err error) {
		systemsA, err = coalitionLocations(
			groupCtx, db, sideA, "solar_system_id", from, to,
		)
		return
	})
	group.Go(func() (err error) {
		regionsA, err = coalitionLocations(
			groupCtx, db, sideA, "region_id", from, to,
		)
		return
	})
	group.Go(func() (err error) {
		systemsB, err = coalitionLocations(
			groupCtx, db, sideB, "solar_system_id", from, to,
		)
		return
	})
	group.Go(func() (err error) {
		regionsB, err = coalitionLocations(
			groupCtx, db, sideB, "region_id", from, to,
		)
		return
	})
	group.Go(func() (err error) {
		daily, err = coalitionDaily(
			groupCtx, db, sideA, sideB, from, to,
		)
		return
	})
	if err := group.Wait(); err != nil {
		return nil, err
	}
	clashed := intersectCoalitionLocations(systemsA, systemsB)
	if len(clashed) > 50 {
		clashed = clashed[:50]
	}
	return map[string]any{
		"mode": mode, "period_days": periodDays, "from": from, "to": to,
		"sideA": coalitionSideBlock(
			sideA, totalsA,
			coalitionVersus(pairwise.AKillsCount, pairwise.ALossesCount,
				pairwise.AKillsISK, pairwise.ALossesISK),
			shipsA, len(systemsA), len(regionsA),
		),
		"sideB": coalitionSideBlock(
			sideB, totalsB,
			coalitionVersus(pairwise.ALossesCount, pairwise.AKillsCount,
				pairwise.ALossesISK, pairwise.AKillsISK),
			shipsB, len(systemsB), len(regionsB),
		),
		"clashed_systems": map[string]any{
			"count":      len(intersectCoalitionLocations(systemsA, systemsB)),
			"system_ids": clashed,
		},
		"daily": daily,
	}, nil
}

func loadCoalitionPairwise(
	ctx context.Context,
	db Database,
	a, b coalitionSide,
	from, to string,
) (coalitionPairwise, error) {
	aOnB, err := coalitionPairDirection(ctx, db, a, b, from, to)
	if err != nil {
		return coalitionPairwise{}, err
	}
	bOnA, err := coalitionPairDirection(ctx, db, b, a, from, to)
	if err != nil {
		return coalitionPairwise{}, err
	}
	return coalitionPairwise{
		AKillsCount: aOnB.Kills, AKillsISK: aOnB.ISKDestroyed,
		ALossesCount: bOnA.Kills, ALossesISK: bOnA.ISKDestroyed,
	}, nil
}

func coalitionPairDirection(
	ctx context.Context,
	db Database,
	attacker, victim coalitionSide,
	from, to string,
) (coalitionTotals, error) {
	args := []any{}
	attackerMatch := coalitionSideSQL(attacker, "a", false, &args)
	args = append(args, from, to)
	fromArg, toArg := len(args)-1, len(args)
	victimMatch := coalitionSideSQL(victim, "k", true, &args)
	row, err := queryMap(ctx, db, `
		WITH side_kms AS MATERIALIZED (
		  SELECT DISTINCT a.killmail_id
		  FROM killmail_attackers a
		  WHERE `+attackerMatch+`
		    AND a.killmail_time >= $`+fmt.Sprint(fromArg)+`::date
		    AND a.killmail_time < ($`+fmt.Sprint(toArg)+`::date + interval '1 day')
		)
		SELECT COUNT(*)::bigint AS kills,
		       COALESCE(SUM(k.total_value), 0)::double precision AS isk
		FROM side_kms s
		JOIN killmails k ON k.killmail_id = s.killmail_id
		WHERE `+victimMatch, args...)
	if err != nil {
		return coalitionTotals{}, err
	}
	kills, _ := int64Value(row["kills"])
	isk, _ := float64Value(row["isk"])
	return coalitionTotals{Kills: kills, ISKDestroyed: isk}, nil
}

func loadCoalitionTotals(
	ctx context.Context,
	db Database,
	side coalitionSide,
	from, to string,
) (coalitionTotals, error) {
	args := []any{}
	attackerMatch := coalitionSideSQL(side, "a", false, &args)
	args = append(args, from, to)
	kills, err := queryMap(ctx, db, `
		WITH side_kms AS MATERIALIZED (
		  SELECT a.killmail_id, bool_or(a.final_blow) AS had_final_blow
		  FROM killmail_attackers a
		  WHERE `+attackerMatch+`
		    AND a.killmail_time >= $`+fmt.Sprint(len(args)-1)+`::date
		    AND a.killmail_time < ($`+fmt.Sprint(len(args))+`::date + interval '1 day')
		  GROUP BY a.killmail_id
		)
		SELECT COUNT(*)::bigint AS kills,
		       COUNT(*) FILTER (WHERE k.is_solo = true)::bigint AS solo_kills,
		       COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed,
		       COALESCE(SUM(k.points), 0)::bigint AS points,
		       COUNT(*) FILTER (WHERE s.had_final_blow = true)::bigint AS final_blows
		FROM side_kms s JOIN killmails k ON k.killmail_id = s.killmail_id`,
		args...)
	if err != nil {
		return coalitionTotals{}, err
	}
	lossArgs := []any{}
	victimMatch := coalitionSideSQL(side, "k", true, &lossArgs)
	lossArgs = append(lossArgs, from, to)
	losses, err := queryMap(ctx, db, `
		SELECT COUNT(*)::bigint AS losses,
		       COUNT(*) FILTER (WHERE k.is_npc = true)::bigint AS npc_losses,
		       COALESCE(SUM(k.total_value), 0)::double precision AS isk_lost
		FROM killmails k
		WHERE `+victimMatch+`
		  AND k.killmail_time >= $`+fmt.Sprint(len(lossArgs)-1)+`::date
		  AND k.killmail_time < ($`+fmt.Sprint(len(lossArgs))+`::date + interval '1 day')`,
		lossArgs...)
	if err != nil {
		return coalitionTotals{}, err
	}
	result := coalitionTotals{}
	result.Kills, _ = int64Value(kills["kills"])
	result.SoloKills, _ = int64Value(kills["solo_kills"])
	result.ISKDestroyed, _ = float64Value(kills["isk_destroyed"])
	result.Points, _ = int64Value(kills["points"])
	result.FinalBlows, _ = int64Value(kills["final_blows"])
	result.Losses, _ = int64Value(losses["losses"])
	result.NPCLosses, _ = int64Value(losses["npc_losses"])
	result.ISKLost, _ = float64Value(losses["isk_lost"])
	return result, nil
}

func coalitionTopShips(
	ctx context.Context,
	db Database,
	side coalitionSide,
	from, to string,
) ([]map[string]any, error) {
	args := []any{}
	match := coalitionSideSQL(side, "a", false, &args)
	args = append(args, from, to)
	rows, err := queryMaps(ctx, db, `
		SELECT a.ship_type_id, COALESCE(t.name, 'Unknown') AS ship_name,
		       COUNT(*)::bigint AS count
		FROM killmail_attackers a
		LEFT JOIN inv_types t ON t.type_id = a.ship_type_id
		WHERE `+match+` AND a.ship_type_id IS NOT NULL
		  AND a.killmail_time >= $`+fmt.Sprint(len(args)-1)+`::date
		  AND a.killmail_time < ($`+fmt.Sprint(len(args))+`::date + interval '1 day')
		GROUP BY a.ship_type_id, t.name
		ORDER BY count DESC LIMIT 10`, args...)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		count, _ := int64Value(row["count"])
		row["count"] = count
	}
	return rows, nil
}

func coalitionLocations(
	ctx context.Context,
	db Database,
	side coalitionSide,
	column, from, to string,
) ([]int64, error) {
	args := []any{}
	attacker := coalitionSideSQL(side, "a", false, &args)
	args = append(args, from, to)
	fromArg, toArg := len(args)-1, len(args)
	victim := coalitionSideSQL(side, "k", true, &args)
	rows, err := queryMaps(ctx, db, `
		WITH side_kms AS MATERIALIZED (
		  SELECT DISTINCT a.killmail_id FROM killmail_attackers a
		  WHERE `+attacker+`
		    AND a.killmail_time >= $`+fmt.Sprint(fromArg)+`::date
		    AND a.killmail_time < ($`+fmt.Sprint(toArg)+`::date + interval '1 day')
		)
		SELECT DISTINCT id FROM (
		  SELECT k.`+column+` AS id FROM side_kms s
		  JOIN killmails k ON k.killmail_id = s.killmail_id
		  WHERE k.`+column+` IS NOT NULL
		  UNION
		  SELECT k.`+column+` AS id FROM killmails k
		  WHERE `+victim+`
		    AND k.killmail_time >= $`+fmt.Sprint(fromArg)+`::date
		    AND k.killmail_time < ($`+fmt.Sprint(toArg)+`::date + interval '1 day')
		    AND k.`+column+` IS NOT NULL
		) u`, args...)
	if err != nil {
		return nil, err
	}
	result := make([]int64, 0, len(rows))
	for _, row := range rows {
		id, ok := int64Value(row["id"])
		if ok && id > 0 {
			result = append(result, id)
		}
	}
	return result, nil
}

func coalitionDaily(
	ctx context.Context,
	db Database,
	a, b coalitionSide,
	from, to string,
) ([]map[string]any, error) {
	aRows, err := coalitionDailyDirection(ctx, db, a, b, from, to)
	if err != nil {
		return nil, err
	}
	bRows, err := coalitionDailyDirection(ctx, db, b, a, from, to)
	if err != nil {
		return nil, err
	}
	aMap := coalitionDailyMap(aRows)
	bMap := coalitionDailyMap(bRows)
	start, _ := time.Parse("2006-01-02", from)
	end, _ := time.Parse("2006-01-02", to)
	result := []map[string]any{}
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		key := date.Format("2006-01-02")
		aValue, bValue := aMap[key], bMap[key]
		result = append(result, map[string]any{
			"date":         key,
			"a_vs_b_kills": zeroMapInt(aValue, "kills"),
			"a_vs_b_isk":   zeroMapFloat(aValue, "isk"),
			"b_vs_a_kills": zeroMapInt(bValue, "kills"),
			"b_vs_a_isk":   zeroMapFloat(bValue, "isk"),
		})
	}
	return result, nil
}

func coalitionDailyDirection(
	ctx context.Context,
	db Database,
	attacker, victim coalitionSide,
	from, to string,
) ([]map[string]any, error) {
	args := []any{}
	attackerMatch := coalitionSideSQL(attacker, "a", false, &args)
	args = append(args, from, to)
	fromArg, toArg := len(args)-1, len(args)
	victimMatch := coalitionSideSQL(victim, "k", true, &args)
	return queryMaps(ctx, db, `
		WITH side_kms AS MATERIALIZED (
		  SELECT DISTINCT a.killmail_id FROM killmail_attackers a
		  WHERE `+attackerMatch+`
		    AND a.killmail_time >= $`+fmt.Sprint(fromArg)+`::date
		    AND a.killmail_time < ($`+fmt.Sprint(toArg)+`::date + interval '1 day')
		)
		SELECT (k.killmail_time::date)::text AS date,
		       COUNT(*)::bigint AS kills,
		       COALESCE(SUM(k.total_value), 0)::double precision AS isk
		FROM side_kms s
		JOIN killmails k ON k.killmail_id = s.killmail_id
		WHERE `+victimMatch+`
		GROUP BY k.killmail_time::date`, args...)
}

func coalitionSideSQL(
	side coalitionSide,
	alias string,
	victim bool,
	args *[]any,
) string {
	columnPrefix := ""
	if victim {
		columnPrefix = "victim_"
	}
	parts := []string{}
	if len(side.Alliances) > 0 {
		*args = append(*args, side.Alliances)
		parts = append(parts, fmt.Sprintf(
			"%s.%salliance_id = ANY($%d::int[])",
			alias, columnPrefix, len(*args),
		))
	}
	if len(side.Corporations) > 0 {
		*args = append(*args, side.Corporations)
		parts = append(parts, fmt.Sprintf(
			"%s.%scorporation_id = ANY($%d::int[])",
			alias, columnPrefix, len(*args),
		))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "(" + parts[0] + " OR " + parts[1] + ")"
}

func coalitionVersus(
	kills, losses int64,
	destroyed, lost float64,
) map[string]any {
	return map[string]any{
		"kills": kills, "losses": losses,
		"isk_destroyed": destroyed, "isk_lost": lost,
		"efficiency_pct":     efficiency(kills, losses),
		"isk_efficiency_pct": iskEfficiency(destroyed, lost),
	}
}

func coalitionSideBlock(
	side coalitionSide,
	totals coalitionTotals,
	versus map[string]any,
	ships []map[string]any,
	activeSystems, activeRegions int,
) map[string]any {
	return map[string]any{
		"label": side.Label,
		"entity_counts": map[string]any{
			"alliances":    len(side.Alliances),
			"corporations": len(side.Corporations),
		},
		"overall": map[string]any{
			"kills": totals.Kills, "losses": totals.Losses,
			"solo_kills": totals.SoloKills, "npc_losses": totals.NPCLosses,
			"isk_destroyed": totals.ISKDestroyed, "isk_lost": totals.ISKLost,
			"points": totals.Points, "final_blows": totals.FinalBlows,
		},
		"vs_opponent":          versus,
		"top_ships_used":       ships,
		"active_systems_count": activeSystems,
		"active_regions_count": activeRegions,
	}
}

func intersectCoalitionLocations(a, b []int64) []int64 {
	set := map[int64]struct{}{}
	for _, id := range b {
		set[id] = struct{}{}
	}
	result := []int64{}
	for _, id := range a {
		if _, exists := set[id]; exists {
			result = append(result, id)
		}
	}
	return result
}

func coalitionDailyMap(rows []map[string]any) map[string]map[string]any {
	result := map[string]map[string]any{}
	for _, row := range rows {
		date, _ := stringValue(row["date"])
		result[date] = row
	}
	return result
}

func zeroMapInt(row map[string]any, key string) int64 {
	value, _ := int64Value(row[key])
	return value
}

func zeroMapFloat(row map[string]any, key string) float64 {
	value, _ := float64Value(row[key])
	return value
}
