package mcpserver

import (
	"context"
	"fmt"
	"math"
	"time"
)

type CoalitionGraphInput struct {
	Since              *string        `json:"since,omitempty" doc:"ISO datetime lower bound. Default 30 days ago."`
	Until              *string        `json:"until,omitempty" doc:"ISO datetime upper bound. Default now."`
	MinEdgeWeight      int            `json:"min_edge_weight,omitempty" default:"3" minimum:"1"`
	MinAllianceBattles int            `json:"min_alliance_battles,omitempty" default:"5" minimum:"1"`
	FocusAlliance      *StringOrInt64 `json:"focus_alliance,omitempty" doc:"Alliance name or id for an ego graph."`
	LimitEdges         int            `json:"limit_edges,omitempty" default:"100" minimum:"1" maximum:"500"`
}

type AllianceNode struct {
	AllianceID int64   `json:"alliance_id"`
	Name       *string `json:"name"`
	Ticker     *string `json:"ticker"`
}

type CoalitionEdge struct {
	A             AllianceNode `json:"a"`
	B             AllianceNode `json:"b"`
	AlliedBattles int64        `json:"allied_battles"`
	EnemyBattles  int64        `json:"enemy_battles"`
	TotalBattles  int64        `json:"total_battles"`
}

type CoalitionFocus struct {
	AllianceID int64   `json:"alliance_id"`
	Name       *string `json:"name"`
}

type TimeWindow struct {
	Since string `json:"since"`
	Until string `json:"until"`
}

type CoalitionGraphOutput struct {
	Window      TimeWindow      `json:"window"`
	Focus       *CoalitionFocus `json:"focus,omitempty"`
	NodeCount   int             `json:"node_count"`
	EdgeCount   int             `json:"edge_count"`
	AlliedEdges []CoalitionEdge `json:"allied_edges"`
	EnemyEdges  []CoalitionEdge `json:"enemy_edges"`
	MixedEdges  []CoalitionEdge `json:"mixed_edges"`
	Nodes       []AllianceNode  `json:"nodes"`
}

type CharacterHistoryInput struct {
	Entity StringOrInt64 `json:"entity" doc:"Character name or id."`
	Since  *string       `json:"since,omitempty" doc:"ISO datetime lower bound. Default all history."`
}

type IDNameTicker struct {
	ID     int64   `json:"id"`
	Name   *string `json:"name"`
	Ticker *string `json:"ticker"`
}

type CharacterRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type MembershipPeriod struct {
	FirstSeen        time.Time     `json:"first_seen"`
	LastSeen         time.Time     `json:"last_seen"`
	DurationDays     int           `json:"duration_days"`
	ObservationCount int           `json:"observation_count"`
	Corporation      *IDNameTicker `json:"corporation"`
	Alliance         *IDNameTicker `json:"alliance"`
}

type membershipRun struct {
	corpID, allianceID *int64
	first, last        time.Time
	count              int
}

type CharacterHistoryOutput struct {
	Character        CharacterRef       `json:"character"`
	ObservationCount int                `json:"observation_count"`
	PeriodCount      int                `json:"period_count"`
	Periods          []MembershipPeriod `json:"periods"`
	Notes            string             `json:"notes,omitempty"`
}

type PilotEfficiencyInput struct {
	Entity     StringOrInt64 `json:"entity" doc:"Character name or id."`
	ShipTypeID *int64        `json:"ship_type_id,omitempty" doc:"Restrict to events involving this ship type."`
	Since      *string       `json:"since,omitempty" doc:"ISO datetime lower bound. Default 90 days ago."`
	Until      *string       `json:"until,omitempty" doc:"ISO datetime upper bound. Default now."`
}

type PilotEfficiencyTotals struct {
	Kills            int64    `json:"kills"`
	Losses           int64    `json:"losses"`
	KillLossRatio    *float64 `json:"kill_loss_ratio"`
	ISKDestroyed     float64  `json:"isk_destroyed"`
	ISKLost          float64  `json:"isk_lost"`
	ISKRatio         *float64 `json:"isk_ratio"`
	ISKEfficiencyPct float64  `json:"isk_efficiency_pct"`
	SoloKills        int64    `json:"solo_kills"`
	SoloRatePct      float64  `json:"solo_rate_pct"`
	FinalBlows       int64    `json:"final_blows"`
	AvgGangOnKills   float64  `json:"avg_gang_on_kills"`
	AvgGangOnLosses  float64  `json:"avg_gang_on_losses"`
}

type PilotActivity struct {
	Semantics          string           `json:"semantics"`
	Description        string           `json:"description"`
	ByHourUTC          map[string]int64 `json:"by_hour_utc"`
	ByDayOfWeek        map[string]int64 `json:"by_day_of_week"`
	PeakHourUTC        int              `json:"peak_hour_utc"`
	PeakHourEventCount int64            `json:"peak_hour_event_count"`
}

type ShipFilter struct {
	TypeID int64 `json:"type_id"`
}

type PilotEfficiencyOutput struct {
	Character  CharacterRef          `json:"character"`
	Window     TimeWindow            `json:"window"`
	ShipFilter *ShipFilter           `json:"ship_filter,omitempty"`
	Totals     PilotEfficiencyTotals `json:"totals"`
	Activity   PilotActivity         `json:"activity"`
}

func registerIntelAggregationTools(registry *Registry) error {
	if err := addTool(registry, ToolDefinition{
		Name: "coalition_graph", Title: "Build an alliance co-occurrence graph",
		Description: "Time-windowed alliance co-occurrence graph derived from battle teams. Edges count battles where two alliances " +
			"stood on the same or opposing teams.",
	}, func(ctx context.Context, input CoalitionGraphInput) (CoalitionGraphOutput, error) {
		return coalitionGraph(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name: "character_history", Title: "Infer character membership history",
		Description: "Inferred corporation and alliance membership history for a character, derived from ordered killmails. " +
			"Dates may be imprecise for inactive pilots.",
	}, func(ctx context.Context, input CharacterHistoryInput) (CharacterHistoryOutput, error) {
		return characterHistory(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	return addTool(registry, ToolDefinition{
		Name: "pilot_efficiency", Title: "Calculate pilot efficiency",
		Description: "Per-pilot performance stats, optionally scoped to a ship type, with ratios, gang sizes, " +
			"and UTC activity heatmaps over recent history.",
	}, func(ctx context.Context, input PilotEfficiencyInput) (PilotEfficiencyOutput, error) {
		return pilotEfficiency(ctx, registry.deps, input)
	})
}

func coalitionGraph(ctx context.Context, deps Dependencies, input CoalitionGraphInput) (CoalitionGraphOutput, error) {
	now := time.Now().UTC()
	since, until := now.Add(-30*24*time.Hour), now
	var err error
	if input.Since != nil {
		since, err = time.Parse(time.RFC3339, *input.Since)
		if err != nil {
			return CoalitionGraphOutput{}, fmt.Errorf("invalid since: %w", err)
		}
	}
	if input.Until != nil {
		until, err = time.Parse(time.RFC3339, *input.Until)
		if err != nil {
			return CoalitionGraphOutput{}, fmt.Errorf("invalid until: %w", err)
		}
	}
	minEdge, minBattles, limit := input.MinEdgeWeight, input.MinAllianceBattles, input.LimitEdges
	if minEdge == 0 {
		minEdge = 3
	}
	if minBattles == 0 {
		minBattles = 5
	}
	if limit == 0 {
		limit = 100
	}
	limit = clamp(limit, 1, 500)

	var focus *ResolvedEntity
	if input.FocusAlliance != nil && input.FocusAlliance.String() != "" {
		focus, err = resolveEntity(ctx, deps, *input.FocusAlliance, entityTypePointer(EntityAlliance))
		if err != nil {
			return CoalitionGraphOutput{}, err
		}
		if focus == nil || focus.Type != EntityAlliance {
			return CoalitionGraphOutput{}, fmt.Errorf("focus_alliance did not resolve to an alliance")
		}
	}
	focusID := int64(0)
	if focus != nil {
		focusID = focus.ID
	}
	rows, err := queryMaps(ctx, deps.DB, `
		WITH alliance_teams AS (
			SELECT DISTINCT b.battle_id, bt.team_index, btm.alliance_id
			FROM battles b
			JOIN battle_teams bt ON bt.battle_id = b.battle_id
			JOIN battle_team_members btm ON btm.battle_team_id = bt.id
			WHERE b.start_time >= $1 AND b.start_time <= $2
			  AND btm.alliance_id IS NOT NULL
		), alliance_battle_counts AS (
			SELECT alliance_id, COUNT(DISTINCT battle_id) AS battles
			FROM alliance_teams GROUP BY alliance_id
			HAVING COUNT(DISTINCT battle_id) >= $3
		), paired AS (
			SELECT a1.alliance_id AS a, a2.alliance_id AS b,
			       COUNT(*) FILTER (WHERE a1.team_index = a2.team_index)::int AS allied,
			       COUNT(*) FILTER (WHERE a1.team_index <> a2.team_index)::int AS enemy
			FROM alliance_teams a1
			JOIN alliance_teams a2 ON a1.battle_id = a2.battle_id AND a1.alliance_id < a2.alliance_id
			JOIN alliance_battle_counts c1 ON c1.alliance_id = a1.alliance_id
			JOIN alliance_battle_counts c2 ON c2.alliance_id = a2.alliance_id
			WHERE ($4::bigint = 0 OR a1.alliance_id = $4 OR a2.alliance_id = $4)
			GROUP BY a1.alliance_id, a2.alliance_id
		)
		SELECT p.a, p.b, p.allied, p.enemy, al1.name AS a_name, al1.ticker AS a_ticker,
		       al2.name AS b_name, al2.ticker AS b_ticker, (p.allied + p.enemy) AS total
		FROM paired p
		LEFT JOIN alliances al1 ON al1.alliance_id = p.a
		LEFT JOIN alliances al2 ON al2.alliance_id = p.b
		WHERE (p.allied + p.enemy) >= $5
		ORDER BY (p.allied + p.enemy) DESC LIMIT $6`,
		since, until, minBattles, focusID, minEdge, limit)
	if err != nil {
		return CoalitionGraphOutput{}, err
	}
	output := CoalitionGraphOutput{
		Window:      TimeWindow{Since: since.Format(time.RFC3339Nano), Until: until.Format(time.RFC3339Nano)},
		AlliedEdges: []CoalitionEdge{}, EnemyEdges: []CoalitionEdge{}, MixedEdges: []CoalitionEdge{},
		Nodes: []AllianceNode{},
	}
	if focus != nil {
		output.Focus = &CoalitionFocus{AllianceID: focus.ID, Name: &focus.Name}
	}
	nodes := map[int64]AllianceNode{}
	for _, row := range rows {
		a := AllianceNode{AllianceID: valueInt64(row["a"]), Name: nullableString(row["a_name"]), Ticker: nullableString(row["a_ticker"])}
		b := AllianceNode{AllianceID: valueInt64(row["b"]), Name: nullableString(row["b_name"]), Ticker: nullableString(row["b_ticker"])}
		edge := CoalitionEdge{A: a, B: b, AlliedBattles: valueInt64(row["allied"]), EnemyBattles: valueInt64(row["enemy"]), TotalBattles: valueInt64(row["total"])}
		nodes[a.AllianceID], nodes[b.AllianceID] = a, b
		switch {
		case edge.AlliedBattles > edge.EnemyBattles:
			output.AlliedEdges = append(output.AlliedEdges, edge)
		case edge.EnemyBattles > edge.AlliedBattles:
			output.EnemyEdges = append(output.EnemyEdges, edge)
		default:
			output.MixedEdges = append(output.MixedEdges, edge)
		}
	}
	for _, node := range nodes {
		output.Nodes = append(output.Nodes, node)
	}
	output.NodeCount, output.EdgeCount = len(nodes), len(rows)
	return output, nil
}

func characterHistory(ctx context.Context, deps Dependencies, input CharacterHistoryInput) (CharacterHistoryOutput, error) {
	character, err := resolveCharacter(ctx, deps, input.Entity, entityTypePointer(EntityCharacter))
	if err != nil {
		return CharacterHistoryOutput{}, err
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT killmail_time, corporation_id, alliance_id FROM (
			SELECT a.killmail_time, a.corporation_id, a.alliance_id
			FROM killmail_attackers a WHERE a.character_id = $1
			  AND ($2::timestamptz IS NULL OR a.killmail_time >= $2)
			UNION ALL
			SELECT k.killmail_time, k.victim_corporation_id, k.victim_alliance_id
			FROM killmails k WHERE k.victim_character_id = $1
			  AND ($2::timestamptz IS NULL OR k.killmail_time >= $2)
		) observations ORDER BY killmail_time ASC`, character.ID, input.Since)
	if err != nil {
		return CharacterHistoryOutput{}, err
	}
	output := CharacterHistoryOutput{
		Character:        CharacterRef{ID: character.ID, Name: character.Name, URL: entityURL(deps.BaseURL, EntityCharacter, character.ID)},
		ObservationCount: len(rows), Periods: []MembershipPeriod{},
	}
	if len(rows) == 0 {
		output.Notes = "No killmail observations found for this character."
		return output, nil
	}
	runs := []membershipRun{}
	for _, row := range rows {
		seen := nullableTime(row["killmail_time"])
		if seen == nil {
			continue
		}
		corpID, allianceID := nullableInt64(row["corporation_id"]), nullableInt64(row["alliance_id"])
		if len(runs) > 0 && sameInt64(runs[len(runs)-1].corpID, corpID) && sameInt64(runs[len(runs)-1].allianceID, allianceID) {
			runs[len(runs)-1].last, runs[len(runs)-1].count = *seen, runs[len(runs)-1].count+1
		} else {
			runs = append(runs, membershipRun{corpID: corpID, allianceID: allianceID, first: *seen, last: *seen, count: 1})
		}
	}
	corpRows, err := loadIDNameTicker(ctx, deps, "corporations", "corporation_id", runs, true)
	if err != nil {
		return CharacterHistoryOutput{}, err
	}
	allianceRows, err := loadIDNameTicker(ctx, deps, "alliances", "alliance_id", runs, false)
	if err != nil {
		return CharacterHistoryOutput{}, err
	}
	for _, run := range runs {
		output.Periods = append(output.Periods, MembershipPeriod{
			FirstSeen: run.first, LastSeen: run.last,
			DurationDays:     int(math.Round(run.last.Sub(run.first).Hours() / 24)),
			ObservationCount: run.count, Corporation: corpRows[derefInt64(run.corpID)],
			Alliance: allianceRows[derefInt64(run.allianceID)],
		})
	}
	output.PeriodCount = len(output.Periods)
	return output, nil
}

func pilotEfficiency(ctx context.Context, deps Dependencies, input PilotEfficiencyInput) (PilotEfficiencyOutput, error) {
	character, err := resolveCharacter(ctx, deps, input.Entity, entityTypePointer(EntityCharacter))
	if err != nil {
		return PilotEfficiencyOutput{}, err
	}
	now := time.Now().UTC()
	since, until := now.Add(-90*24*time.Hour), now
	if input.Since != nil {
		since, err = time.Parse(time.RFC3339, *input.Since)
		if err != nil {
			return PilotEfficiencyOutput{}, fmt.Errorf("invalid since: %w", err)
		}
	}
	if input.Until != nil {
		until, err = time.Parse(time.RFC3339, *input.Until)
		if err != nil {
			return PilotEfficiencyOutput{}, fmt.Errorf("invalid until: %w", err)
		}
	}
	var shipID *int64 = input.ShipTypeID
	kills, err := queryMaps(ctx, deps.DB, `
		SELECT COUNT(*)::bigint AS kills, COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed,
		       AVG(k.attacker_count)::double precision AS avg_attackers,
		       COUNT(*) FILTER (WHERE k.is_solo = true)::bigint AS solo_kills,
		       COUNT(*) FILTER (WHERE a.final_blow = true)::bigint AS final_blows
		FROM killmail_attackers a JOIN killmails k ON k.killmail_id = a.killmail_id
		WHERE a.character_id = $1 AND a.killmail_time >= $2 AND a.killmail_time <= $3
		  AND ($4::bigint IS NULL OR a.ship_type_id = $4)`, character.ID, since, until, shipID)
	if err != nil {
		return PilotEfficiencyOutput{}, err
	}
	losses, err := queryMaps(ctx, deps.DB, `
		SELECT COUNT(*)::bigint AS losses, COALESCE(SUM(k.total_value), 0)::double precision AS isk_lost,
		       AVG(k.attacker_count)::double precision AS avg_attackers_on_loss
		FROM killmails k
		WHERE k.victim_character_id = $1 AND k.killmail_time >= $2 AND k.killmail_time <= $3
		  AND ($4::bigint IS NULL OR k.victim_ship_type_id = $4)`, character.ID, since, until, shipID)
	if err != nil {
		return PilotEfficiencyOutput{}, err
	}
	heatmap, err := queryMaps(ctx, deps.DB, `
		SELECT EXTRACT(HOUR FROM killmail_time)::int AS hour,
		       EXTRACT(DOW FROM killmail_time)::int AS dow, COUNT(*)::int AS count
		FROM (
			SELECT a.killmail_time FROM killmail_attackers a
			WHERE a.character_id = $1 AND a.killmail_time >= $2 AND a.killmail_time <= $3
			UNION ALL
			SELECT k.killmail_time FROM killmails k
			WHERE k.victim_character_id = $1 AND k.killmail_time >= $2 AND k.killmail_time <= $3
		) events GROUP BY hour, dow`, character.ID, since, until)
	if err != nil {
		return PilotEfficiencyOutput{}, err
	}
	k, l := firstMap(kills), firstMap(losses)
	killCount, lossCount := valueInt64(k["kills"]), valueInt64(l["losses"])
	destroyed, lost := valueFloat64(k["isk_destroyed"]), valueFloat64(l["isk_lost"])
	hours, days := map[string]int64{}, map[string]int64{}
	for i := 0; i < 24; i++ {
		hours[fmt.Sprint(i)] = 0
	}
	for i := 0; i < 7; i++ {
		days[fmt.Sprint(i)] = 0
	}
	for _, row := range heatmap {
		hour, day, count := fmt.Sprint(valueInt64(row["hour"])), fmt.Sprint(valueInt64(row["dow"])), valueInt64(row["count"])
		hours[hour], days[day] = hours[hour]+count, days[day]+count
	}
	peakHour, peakCount := 0, int64(0)
	for i := 0; i < 24; i++ {
		if hours[fmt.Sprint(i)] > peakCount {
			peakHour, peakCount = i, hours[fmt.Sprint(i)]
		}
	}
	output := PilotEfficiencyOutput{
		Character: CharacterRef{ID: character.ID, Name: character.Name, URL: entityURL(deps.BaseURL, EntityCharacter, character.ID)},
		Window:    TimeWindow{Since: since.Format(time.RFC3339Nano), Until: until.Format(time.RFC3339Nano)},
		Totals: PilotEfficiencyTotals{
			Kills: killCount, Losses: lossCount, KillLossRatio: ratio(killCount, lossCount),
			ISKDestroyed: destroyed, ISKLost: lost, ISKRatio: floatRatio(destroyed, lost),
			ISKEfficiencyPct: iskEfficiency(destroyed, lost), SoloKills: valueInt64(k["solo_kills"]),
			FinalBlows: valueInt64(k["final_blows"]), AvgGangOnKills: valueFloat64(k["avg_attackers"]),
			AvgGangOnLosses: valueFloat64(l["avg_attackers_on_loss"]),
		},
		Activity: PilotActivity{Semantics: "events_count", Description: "Values are killmail events (kills + losses unioned). Measures active-in-PvP time, not kill volume.", ByHourUTC: hours, ByDayOfWeek: days, PeakHourUTC: peakHour, PeakHourEventCount: peakCount},
	}
	if killCount > 0 {
		output.Totals.SoloRatePct = math.Round(float64(output.Totals.SoloKills)/float64(killCount)*10000) / 100
	}
	if shipID != nil {
		output.ShipFilter = &ShipFilter{TypeID: *shipID}
	}
	return output, nil
}

func sameInt64(a, b *int64) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func loadIDNameTicker(ctx context.Context, deps Dependencies, table, idColumn string, runs []membershipRun, corporation bool) (map[int64]*IDNameTicker, error) {
	ids := []int64{}
	for _, run := range runs {
		id := run.allianceID
		if corporation {
			id = run.corpID
		}
		if id != nil {
			ids = append(ids, *id)
		}
	}
	output := map[int64]*IDNameTicker{}
	if len(ids) == 0 {
		return output, nil
	}
	rows, err := queryMaps(ctx, deps.DB, fmt.Sprintf("SELECT %s AS id, name, ticker FROM %s WHERE %s = ANY($1)", idColumn, table, idColumn), ids)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		id := valueInt64(row["id"])
		output[id] = &IDNameTicker{ID: id, Name: nullableString(row["name"]), Ticker: nullableString(row["ticker"])}
	}
	return output, nil
}

func ratio(numerator, denominator int64) *float64 {
	if denominator == 0 {
		if numerator == 0 {
			value := float64(0)
			return &value
		}
		return nil
	}
	value := math.Round(float64(numerator)/float64(denominator)*100) / 100
	return &value
}

func floatRatio(numerator, denominator float64) *float64 {
	if denominator == 0 {
		if numerator == 0 {
			value := float64(0)
			return &value
		}
		return nil
	}
	value := math.Round(numerator/denominator*100) / 100
	return &value
}
