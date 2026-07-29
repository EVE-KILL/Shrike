package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type EntityTopInput struct {
	Entity    StringOrInt64  `json:"entity"`
	Type      *EntityType    `json:"type,omitempty" enum:"character,corporation,alliance"`
	Dimension string         `json:"dimension" enum:"ship_flown,ship_lost,system,constellation,region,dies_to_corporation,dies_to_alliance,killed_corporation,killed_alliance"`
	SortBy    string         `json:"sort_by,omitempty" enum:"kills,losses,isk_destroyed,isk_lost" default:"kills"`
	Since     *string        `json:"since,omitempty"`
	Until     *string        `json:"until,omitempty"`
	VS        *StringOrInt64 `json:"vs,omitempty" doc:"Opponent character, corporation, or alliance."`
	Limit     int            `json:"limit,omitempty" default:"10" minimum:"1" maximum:"50"`
}

type EntityTopOutput struct {
	Entity    Entity            `json:"entity"`
	VS        *Entity           `json:"vs,omitempty"`
	Dimension string            `json:"dimension"`
	SortBy    string            `json:"sort_by"`
	Window    StatsWindow       `json:"window"`
	Count     int               `json:"count"`
	Rows      []EntityBreakdown `json:"rows"`
	Warnings  []string          `json:"warnings,omitempty"`
}

type EntityTimelineInput struct {
	Entity StringOrInt64  `json:"entity"`
	Type   *EntityType    `json:"type,omitempty" enum:"character,corporation,alliance,ship,system,constellation,region"`
	Bucket string         `json:"bucket,omitempty" enum:"day,month,year" default:"month"`
	Since  *string        `json:"since,omitempty"`
	Until  *string        `json:"until,omitempty"`
	VS     *StringOrInt64 `json:"vs,omitempty" doc:"Opponent character, corporation, or alliance."`
}

type TimelineWindow struct {
	Since *string `json:"since"`
	Until *string `json:"until"`
}

type TimelineBucket struct {
	PeriodStart  time.Time `json:"period_start"`
	Kills        int64     `json:"kills"`
	Losses       int64     `json:"losses"`
	SoloKills    int64     `json:"solo_kills,omitempty"`
	SoloLosses   int64     `json:"solo_losses,omitempty"`
	FinalBlows   int64     `json:"final_blows,omitempty"`
	Points       int64     `json:"points,omitempty"`
	ISKDestroyed float64   `json:"isk_destroyed"`
	ISKLost      float64   `json:"isk_lost"`
}

type EntityTimelineOutput struct {
	Entity  Entity           `json:"entity"`
	VS      *Entity          `json:"vs,omitempty"`
	Bucket  string           `json:"bucket"`
	Window  TimelineWindow   `json:"window"`
	Count   int              `json:"count"`
	Buckets []TimelineBucket `json:"buckets"`
}

func registerEntityExtraTools(registry *Registry) error {
	if err := addTool(registry, ToolDefinition{
		Name: "entity_top", Title: "Get an entity top breakdown",
		Description: "Top-N breakdown for a character, corporation, or alliance across ships, locations, prey, or tormentors, with optional date and opponent filters.",
	}, func(ctx context.Context, input EntityTopInput) (EntityTopOutput, error) {
		return entityTop(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	return addTool(registry, ToolDefinition{
		Name: "entity_timeline", Title: "Get an entity activity timeline",
		Description: "Kills, losses, and ISK activity over time for an entity, bucketed by day, month, or year with optional opponent filtering.",
	}, func(ctx context.Context, input EntityTimelineInput) (EntityTimelineOutput, error) {
		return entityTimeline(ctx, registry.deps, input)
	})
}

type entityTopDimension struct {
	category        int
	table, id, name string
}

var entityTopDimensions = map[string]entityTopDimension{
	"ship_flown":          {dimensionShipFlown, "inv_types", "type_id", "name"},
	"ship_lost":           {dimensionShipLost, "inv_types", "type_id", "name"},
	"system":              {dimensionSystem, "solar_systems", "solar_system_id", "system_name"},
	"constellation":       {11, "constellations", "constellation_id", "constellation_name"},
	"region":              {dimensionRegion, "regions", "region_id", "name"},
	"dies_to_corporation": {dimensionDiesToCorporation, "corporations", "corporation_id", "name"},
	"dies_to_alliance":    {22, "alliances", "alliance_id", "name"},
	"killed_corporation":  {dimensionKilledCorporation, "corporations", "corporation_id", "name"},
	"killed_alliance":     {32, "alliances", "alliance_id", "name"},
}

var organizationAttackerColumns = map[EntityType]string{
	EntityCharacter: "character_id", EntityCorporation: "corporation_id", EntityAlliance: "alliance_id",
}

var organizationVictimColumns = map[EntityType]string{
	EntityCharacter: "victim_character_id", EntityCorporation: "victim_corporation_id", EntityAlliance: "victim_alliance_id",
}

func entityTop(ctx context.Context, deps Dependencies, input EntityTopInput) (EntityTopOutput, error) {
	entity, err := resolveEntity(ctx, deps, input.Entity, input.Type)
	if err != nil {
		return EntityTopOutput{}, err
	}
	if entity == nil {
		return EntityTopOutput{}, fmt.Errorf("no entity found for %q", input.Entity.String())
	}
	statsType, ok := statsEntityType(entity.Type)
	if !ok || statsType > statsAlliance {
		return EntityTopOutput{}, fmt.Errorf("entity_top is for characters, corporations, or alliances")
	}
	dimension, ok := entityTopDimensions[input.Dimension]
	if !ok {
		return EntityTopOutput{}, fmt.Errorf("unknown dimension %q", input.Dimension)
	}
	sortBy := input.SortBy
	if sortBy == "" {
		sortBy = "kills"
	}
	allowedSort := map[string]bool{"kills": true, "losses": true, "isk_destroyed": true, "isk_lost": true}
	if !allowedSort[sortBy] {
		return EntityTopOutput{}, fmt.Errorf("invalid sort_by")
	}
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}
	limit = clamp(limit, 1, 50)
	output := EntityTopOutput{
		Entity: entity.Public(deps.BaseURL), Dimension: input.Dimension, SortBy: sortBy,
		Window: LifetimeWindow(), Rows: []EntityBreakdown{},
	}
	if (input.Since == nil) != (input.Until == nil) {
		output.Warnings = []string{"since/until must both be set to apply a time window; ignoring partial bound and returning lifetime totals."}
	}
	if input.VS != nil && input.VS.String() != "" {
		opponent, resolveErr := resolveEntity(ctx, deps, *input.VS, nil)
		if resolveErr != nil {
			return EntityTopOutput{}, resolveErr
		}
		if opponent == nil || organizationAttackerColumns[opponent.Type] == "" {
			return EntityTopOutput{}, fmt.Errorf("vs must resolve to character, corporation, or alliance")
		}
		if !map[string]bool{"ship_flown": true, "ship_lost": true, "system": true, "region": true}[input.Dimension] {
			return EntityTopOutput{}, fmt.Errorf("dimension %s is not supported with vs", input.Dimension)
		}
		window, rows, loadErr := entityTopVS(ctx, deps, *entity, *opponent, input.Dimension, sortBy, input.Since, input.Until, limit)
		if loadErr != nil {
			return EntityTopOutput{}, loadErr
		}
		public := opponent.Public(deps.BaseURL)
		output.VS, output.Window, output.Rows, output.Count = &public, window, rows, len(rows)
		return output, nil
	}
	dateBounded := input.Since != nil && input.Until != nil
	periodFilter := "b.period_type = 2"
	args := []any{statsType, entity.ID, dimension.category}
	if dateBounded {
		periodFilter = "b.period_type = 0 AND b.period_start >= $4::date AND b.period_start <= $5::date"
		args = append(args, *input.Since, *input.Until)
		output.Window = BoundedWindow(*input.Since, *input.Until)
	}
	args = append(args, limit)
	query := fmt.Sprintf(`
		SELECT b.dim_id AS id, lookup.%s AS name, SUM(b.kills)::bigint AS kills,
		       SUM(b.losses)::bigint AS losses, SUM(b.isk_destroyed)::double precision AS isk_destroyed,
		       SUM(b.isk_lost)::double precision AS isk_lost
		FROM stats_breakdowns b LEFT JOIN %s lookup ON lookup.%s = b.dim_id
		WHERE b.entity_type = $1 AND b.entity_id = $2 AND %s AND b.dim_category = $3
		GROUP BY b.dim_id, lookup.%s ORDER BY SUM(b.%s) DESC LIMIT $%d`,
		dimension.name, dimension.table, dimension.id, periodFilter, dimension.name, sortBy, len(args))
	rows, err := queryMaps(ctx, deps.DB, query, args...)
	if err != nil {
		return EntityTopOutput{}, err
	}
	output.Rows = breakdownRows(rows)
	output.Count = len(output.Rows)
	return output, nil
}

func entityTimeline(ctx context.Context, deps Dependencies, input EntityTimelineInput) (EntityTimelineOutput, error) {
	entity, err := resolveEntity(ctx, deps, input.Entity, input.Type)
	if err != nil {
		return EntityTimelineOutput{}, err
	}
	if entity == nil {
		return EntityTimelineOutput{}, fmt.Errorf("no entity found for %q", input.Entity.String())
	}
	bucket := input.Bucket
	if bucket == "" {
		bucket = "month"
	}
	if bucket != "day" && bucket != "month" && bucket != "year" {
		return EntityTimelineOutput{}, fmt.Errorf("invalid bucket")
	}
	output := EntityTimelineOutput{
		Entity: entity.Public(deps.BaseURL), Bucket: bucket,
		Window: TimelineWindow{Since: input.Since, Until: input.Until}, Buckets: []TimelineBucket{},
	}
	if input.VS != nil && input.VS.String() != "" {
		if organizationAttackerColumns[entity.Type] == "" {
			return EntityTimelineOutput{}, fmt.Errorf("vs is only supported for character, corporation, or alliance subjects")
		}
		opponent, resolveErr := resolveEntity(ctx, deps, *input.VS, nil)
		if resolveErr != nil {
			return EntityTimelineOutput{}, resolveErr
		}
		if opponent == nil || organizationAttackerColumns[opponent.Type] == "" {
			return EntityTimelineOutput{}, fmt.Errorf("vs must resolve to character, corporation, or alliance")
		}
		window, buckets, loadErr := entityTimelineVS(ctx, deps, *entity, *opponent, bucket, input.Since, input.Until)
		if loadErr != nil {
			return EntityTimelineOutput{}, loadErr
		}
		public := opponent.Public(deps.BaseURL)
		output.VS, output.Window, output.Buckets, output.Count = &public, window, buckets, len(buckets)
		return output, nil
	}
	statsType, ok := statsEntityType(entity.Type)
	if !ok {
		return EntityTimelineOutput{}, fmt.Errorf("stats not tracked for %s", entity.Type)
	}
	periodType := map[string]int{"day": 0, "month": 1, "year": 2}[bucket]
	since := input.Since
	if bucket == "day" && since == nil {
		value := time.Now().UTC().Add(-90 * 24 * time.Hour).Format("2006-01-02")
		since = &value
		output.Window.Since = since
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT period_start, kills, losses, solo_kills, solo_losses, final_blows,
		       points, isk_destroyed, isk_lost
		FROM stats
		WHERE entity_type = $1 AND entity_id = $2 AND period_type = $3
		  AND ($4::date IS NULL OR period_start >= $4)
		  AND ($5::date IS NULL OR period_start <= $5)
		ORDER BY period_start ASC`, statsType, entity.ID, periodType, since, input.Until)
	if err != nil {
		return EntityTimelineOutput{}, err
	}
	output.Buckets = timelineRows(rows)
	output.Count = len(output.Buckets)
	return output, nil
}

func statsEntityType(entityType EntityType) (int, bool) {
	value, ok := map[EntityType]int{
		EntityCharacter: statsCharacter, EntityCorporation: statsCorporation,
		EntityAlliance: statsAlliance, EntityShip: statsShip, EntitySystem: statsSystem,
		EntityConstellation: statsConstellation, EntityRegion: statsRegion,
	}[entityType]
	return value, ok
}

func entityTopVS(ctx context.Context, deps Dependencies, entity, opponent ResolvedEntity, dimension, sortBy string, sinceInput, untilInput *string, limit int) (StatsWindow, []EntityBreakdown, error) {
	since, until, err := parseVSWindow(sinceInput, untilInput, 90)
	if err != nil {
		return StatsWindow{}, nil, err
	}
	entityAttacker, entityVictim := organizationAttackerColumns[entity.Type], organizationVictimColumns[entity.Type]
	opponentAttacker, opponentVictim := organizationAttackerColumns[opponent.Type], organizationVictimColumns[opponent.Type]
	var query string
	switch dimension {
	case "ship_flown":
		order := "COUNT(*)"
		if sortBy == "isk_destroyed" {
			order = "SUM(k.total_value)"
		}
		query = fmt.Sprintf(`
			SELECT a.ship_type_id AS id, t.name, COUNT(*)::bigint AS kills, 0::bigint AS losses,
			       COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed, 0::double precision AS isk_lost
			FROM killmails k
			JOIN killmail_attackers a ON a.killmail_id = k.killmail_id AND a.%s = $1
			LEFT JOIN inv_types t ON t.type_id = a.ship_type_id
			WHERE k.%s = $2 AND k.killmail_time >= $3 AND k.killmail_time <= $4
			  AND a.ship_type_id IS NOT NULL
			GROUP BY a.ship_type_id, t.name ORDER BY %s DESC NULLS LAST LIMIT $5`, entityAttacker, opponentVictim, order)
	case "ship_lost":
		order := "COUNT(*)"
		if sortBy == "isk_lost" {
			order = "SUM(k.total_value)"
		}
		query = fmt.Sprintf(`
			SELECT k.victim_ship_type_id AS id, t.name, 0::bigint AS kills, COUNT(*)::bigint AS losses,
			       0::double precision AS isk_destroyed, COALESCE(SUM(k.total_value), 0)::double precision AS isk_lost
			FROM killmails k LEFT JOIN inv_types t ON t.type_id = k.victim_ship_type_id
			WHERE k.%s = $1 AND k.killmail_time >= $3 AND k.killmail_time <= $4
			  AND k.victim_ship_type_id IS NOT NULL
			  AND EXISTS (SELECT 1 FROM killmail_attackers a WHERE a.killmail_id = k.killmail_id AND a.%s = $2)
			GROUP BY k.victim_ship_type_id, t.name ORDER BY %s DESC NULLS LAST LIMIT $5`, entityVictim, opponentAttacker, order)
	default:
		dimColumn, table, idColumn, nameColumn := "solar_system_id", "solar_systems", "solar_system_id", "system_name"
		if dimension == "region" {
			dimColumn, table, idColumn, nameColumn = "region_id", "regions", "region_id", "name"
		}
		query = fmt.Sprintf(`
			SELECT events.%s AS id, lookup.%s AS name, SUM(events.kills)::bigint AS kills,
			       SUM(events.losses)::bigint AS losses, SUM(events.isk_destroyed)::double precision AS isk_destroyed,
			       SUM(events.isk_lost)::double precision AS isk_lost
			FROM (
				SELECT k.%s, 1 AS kills, 0 AS losses, COALESCE(k.total_value, 0) AS isk_destroyed, 0 AS isk_lost
				FROM killmails k WHERE k.%s = $2 AND k.killmail_time >= $3 AND k.killmail_time <= $4
				  AND EXISTS (SELECT 1 FROM killmail_attackers a WHERE a.killmail_id = k.killmail_id AND a.%s = $1)
				UNION ALL
				SELECT k.%s, 0, 1, 0, COALESCE(k.total_value, 0)
				FROM killmails k WHERE k.%s = $1 AND k.killmail_time >= $3 AND k.killmail_time <= $4
				  AND EXISTS (SELECT 1 FROM killmail_attackers a WHERE a.killmail_id = k.killmail_id AND a.%s = $2)
			) events LEFT JOIN %s lookup ON lookup.%s = events.%s
			WHERE events.%s IS NOT NULL GROUP BY events.%s, lookup.%s
			ORDER BY SUM(events.%s) DESC LIMIT $5`,
			dimColumn, nameColumn, dimColumn, opponentVictim, entityAttacker, dimColumn,
			entityVictim, opponentAttacker, table, idColumn, dimColumn, dimColumn, dimColumn, nameColumn, sortBy)
	}
	rows, err := queryMaps(ctx, deps.DB, query, entity.ID, opponent.ID, since, until, limit)
	return BoundedWindow(since.Format(time.RFC3339Nano), until.Format(time.RFC3339Nano)), breakdownRows(rows), err
}

func entityTimelineVS(ctx context.Context, deps Dependencies, entity, opponent ResolvedEntity, bucket string, sinceInput, untilInput *string) (TimelineWindow, []TimelineBucket, error) {
	defaultDays := map[string]int{"day": 90, "month": 365, "year": 1825}[bucket]
	since, until, err := parseVSWindow(sinceInput, untilInput, defaultDays)
	if err != nil {
		return TimelineWindow{}, nil, err
	}
	trunc := bucket
	entityAttacker, entityVictim := organizationAttackerColumns[entity.Type], organizationVictimColumns[entity.Type]
	opponentAttacker, opponentVictim := organizationAttackerColumns[opponent.Type], organizationVictimColumns[opponent.Type]
	kills, err := queryMaps(ctx, deps.DB, fmt.Sprintf(`
		SELECT date_trunc('%s', k.killmail_time)::date AS period_start, COUNT(*)::bigint AS kills,
		       COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed
		FROM killmails k WHERE k.%s = $2 AND k.killmail_time >= $3 AND k.killmail_time <= $4
		  AND EXISTS (SELECT 1 FROM killmail_attackers a WHERE a.killmail_id = k.killmail_id AND a.%s = $1)
		GROUP BY period_start ORDER BY period_start`, trunc, opponentVictim, entityAttacker),
		entity.ID, opponent.ID, since, until)
	if err != nil {
		return TimelineWindow{}, nil, err
	}
	losses, err := queryMaps(ctx, deps.DB, fmt.Sprintf(`
		SELECT date_trunc('%s', k.killmail_time)::date AS period_start, COUNT(*)::bigint AS losses,
		       COALESCE(SUM(k.total_value), 0)::double precision AS isk_lost
		FROM killmails k WHERE k.%s = $1 AND k.killmail_time >= $3 AND k.killmail_time <= $4
		  AND EXISTS (SELECT 1 FROM killmail_attackers a WHERE a.killmail_id = k.killmail_id AND a.%s = $2)
		GROUP BY period_start ORDER BY period_start`, trunc, entityVictim, opponentAttacker),
		entity.ID, opponent.ID, since, until)
	if err != nil {
		return TimelineWindow{}, nil, err
	}
	combined := map[string]TimelineBucket{}
	for _, row := range kills {
		period := nullableTime(row["period_start"])
		if period != nil {
			combined[period.Format("2006-01-02")] = TimelineBucket{PeriodStart: *period, Kills: valueInt64(row["kills"]), ISKDestroyed: valueFloat64(row["isk_destroyed"])}
		}
	}
	for _, row := range losses {
		period := nullableTime(row["period_start"])
		if period == nil {
			continue
		}
		key := period.Format("2006-01-02")
		value := combined[key]
		value.PeriodStart, value.Losses, value.ISKLost = *period, valueInt64(row["losses"]), valueFloat64(row["isk_lost"])
		combined[key] = value
	}
	keys := make([]string, 0, len(combined))
	for key := range combined {
		keys = append(keys, key)
	}
	sortStrings(keys)
	buckets := make([]TimelineBucket, 0, len(keys))
	for _, key := range keys {
		buckets = append(buckets, combined[key])
	}
	sinceText, untilText := since.Format(time.RFC3339Nano), until.Format(time.RFC3339Nano)
	return TimelineWindow{Since: &sinceText, Until: &untilText}, buckets, nil
}

func parseVSWindow(sinceInput, untilInput *string, defaultDays int) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	since, until := now.Add(-time.Duration(defaultDays)*24*time.Hour), now
	var err error
	if sinceInput != nil {
		since, err = time.Parse(time.RFC3339, *sinceInput)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid since: %w", err)
		}
	}
	if untilInput != nil {
		until, err = time.Parse(time.RFC3339, *untilInput)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid until: %w", err)
		}
	}
	return since, until, nil
}

func breakdownRows(rows []map[string]any) []EntityBreakdown {
	output := make([]EntityBreakdown, 0, len(rows))
	for _, row := range rows {
		output = append(output, EntityBreakdown{
			ID: valueInt64(row["id"]), Name: nullableString(row["name"]), Kills: valueInt64(row["kills"]),
			Losses: valueInt64(row["losses"]), ISKDestroyed: valueFloat64(row["isk_destroyed"]), ISKLost: valueFloat64(row["isk_lost"]),
		})
	}
	return output
}

func timelineRows(rows []map[string]any) []TimelineBucket {
	output := make([]TimelineBucket, 0, len(rows))
	for _, row := range rows {
		period := nullableTime(row["period_start"])
		if period == nil {
			continue
		}
		output = append(output, TimelineBucket{
			PeriodStart: *period, Kills: valueInt64(row["kills"]), Losses: valueInt64(row["losses"]),
			SoloKills: valueInt64(row["solo_kills"]), SoloLosses: valueInt64(row["solo_losses"]),
			FinalBlows: valueInt64(row["final_blows"]), Points: valueInt64(row["points"]),
			ISKDestroyed: valueFloat64(row["isk_destroyed"]), ISKLost: valueFloat64(row["isk_lost"]),
		})
	}
	return output
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && strings.Compare(values[j], values[j-1]) < 0; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
