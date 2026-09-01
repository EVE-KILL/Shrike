package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/sync/errgroup"
)

// coalitionSideBody is the wire shape of one side. jsInt keeps the numeric
// strings the previous API accepted while the parsed result is a proper int32,
// which is what the alliance and corporation columns hold.
var coalitionDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type coalitionSideBody struct {
	Label        string  `json:"label,omitempty" maxLength:"120" doc:"Display name for this side. Truncated at 120 characters."`
	Alliances    []jsInt `json:"alliances,omitempty" doc:"Alliance IDs on this side."`
	Corporations []jsInt `json:"corporations,omitempty" doc:"Corporation IDs on this side."`
}

// coalitionRequestBody is the documented request schema.
type coalitionRequestBody struct {
	SideA coalitionSideBody `json:"sideA" doc:"First coalition. Needs at least one alliance or corporation."`
	SideB coalitionSideBody `json:"sideB" doc:"Second coalition. Needs at least one alliance or corporation."`
	Date  string            `json:"date,omitempty" format:"date" pattern:"^\\d{4}-\\d{2}-\\d{2}$" doc:"Restrict to a single day. Takes precedence over days."`
	Days  *jsInt            `json:"days,omitempty" doc:"Lookback window ending today, clamped to 1-90. Defaults to 30."`
}

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
	registerLegacyJSON(a, huma.Operation{
		OperationID: "coalition-stats",
		Method:      http.MethodPost,
		Path:        "/coalitions/stats",
		Summary:     "Coalition versus coalition statistics",
		Tags:        []string{"stats"},
	}, defaultBodyLimit, func(
		ctx context.Context, req *legacyRequest, wire *coalitionRequestBody,
	) (legacyPayload, error) {
		body, err := parseCoalitionBody(wire)
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

func parseCoalitionBody(wire *coalitionRequestBody) (coalitionRequest, error) {
	sideA, err := parseCoalitionSide(wire.SideA, "sideA")
	if err != nil {
		return coalitionRequest{}, err
	}
	sideB, err := parseCoalitionSide(wire.SideB, "sideB")
	if err != nil {
		return coalitionRequest{}, err
	}
	result := coalitionRequest{SideA: sideA, SideB: sideB}

	// A date pins the window to one day and wins over days, matching the
	// original precedence.
	if wire.Date != "" {
		if !coalitionDatePattern.MatchString(wire.Date) {
			return coalitionRequest{}, apiError(
				http.StatusBadRequest, "date must be YYYY-MM-DD",
			)
		}
		result.Mode, result.PeriodDays = "daily", 1
		result.From, result.To = wire.Date, wire.Date
		return result, nil
	}

	// Zero was previously indistinguishable from absent, because the check was
	// `value != 0`. Keeping that means a caller sending 0 still gets 30.
	days := 30
	if wire.Days != nil && int64(*wire.Days) != 0 {
		days = int(*wire.Days)
	}
	days = max(1, min(90, days))
	now := time.Now().UTC()
	result.Mode, result.PeriodDays = "lookback", days
	result.From = now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	result.To = now.Format("2006-01-02")
	return result, nil
}

func parseCoalitionSide(side coalitionSideBody, name string) (coalitionSide, error) {
	alliances, err := coalitionIDs(side.Alliances, name+".alliances")
	if err != nil {
		return coalitionSide{}, err
	}
	corporations, err := coalitionIDs(side.Corporations, name+".corporations")
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
	if side.Label != "" {
		label = side.Label
		if len(label) > 120 {
			label = label[:120]
		}
	}
	return coalitionSide{
		Label: label, Alliances: alliances, Corporations: corporations,
	}, nil
}

func coalitionIDs(values []jsInt, field string) ([]int32, error) {
	result := []int32{}
	seen := map[int32]struct{}{}
	for _, value := range values {
		id := int64(value)
		if id <= 0 || id > math.MaxInt32 {
			return nil, apiError(http.StatusBadRequest,
				fmt.Sprintf("%s contains invalid id: %d", field, id))
		}
		narrowed := int32(id)
		if _, exists := seen[narrowed]; !exists {
			seen[narrowed] = struct{}{}
			result = append(result, narrowed)
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
		  SELECT a.killmail_id, bool_or(a.final_blow) AS had_final_blow,
		         sum(a.points)::bigint AS points
		  FROM killmail_attackers a
		  WHERE `+attackerMatch+`
		    AND a.killmail_time >= $`+fmt.Sprint(len(args)-1)+`::date
		    AND a.killmail_time < ($`+fmt.Sprint(len(args))+`::date + interval '1 day')
		  GROUP BY a.killmail_id
		)
		SELECT COUNT(*)::bigint AS kills,
		       COUNT(*) FILTER (WHERE k.is_solo = true)::bigint AS solo_kills,
		       COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed,
		       COALESCE(SUM(s.points), 0)::bigint AS points,
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
