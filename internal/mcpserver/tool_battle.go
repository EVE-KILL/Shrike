package mcpserver

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

type BattleReportInput struct {
	BattleID int64  `json:"battle_id" minimum:"1"`
	Level    string `json:"level,omitempty" enum:"alliance,corp" default:"alliance"`
	Format   string `json:"format,omitempty" enum:"json,summary" default:"json"`
}

type BattleMember struct {
	CorporationID     *int64  `json:"corporation_id,omitempty"`
	CorporationName   *string `json:"corporation_name,omitempty"`
	CorporationTicker *string `json:"corporation_ticker,omitempty"`
	AllianceID        *int64  `json:"alliance_id"`
	AllianceName      *string `json:"alliance_name,omitempty"`
	AllianceTicker    *string `json:"alliance_ticker,omitempty"`
	CorporationCount  int     `json:"corporation_count,omitempty"`
	Kills             int64   `json:"kills"`
	Losses            int64   `json:"losses"`
	ISKDestroyed      float64 `json:"isk_destroyed"`
	ISKLost           float64 `json:"isk_lost"`
}

type BattleTeam struct {
	TeamIndex         int            `json:"team_index"`
	TotalKills        int64          `json:"total_kills"`
	TotalLosses       int64          `json:"total_losses"`
	TotalISKDestroyed float64        `json:"total_isk_destroyed"`
	TotalISKLost      float64        `json:"total_isk_lost"`
	Members           []BattleMember `json:"members"`
}

type BattleReportOutput struct {
	BattleID          int64                `json:"battle_id"`
	URL               string               `json:"url"`
	Summary           string               `json:"summary,omitempty"`
	StartTime         *time.Time           `json:"start_time,omitempty"`
	EndTime           *time.Time           `json:"end_time,omitempty"`
	DurationMinutes   int64                `json:"duration_minutes,omitempty"`
	KillCount         int64                `json:"kill_count,omitempty"`
	TotalISKDestroyed float64              `json:"total_isk_destroyed,omitempty"`
	IsMultiParty      bool                 `json:"is_multi_party,omitempty"`
	IsCustom          bool                 `json:"is_custom,omitempty"`
	Level             string               `json:"level,omitempty"`
	System            *BattleSystemSummary `json:"system,omitempty"`
	Teams             []BattleTeam         `json:"teams,omitempty"`
}

type FindBattlesInput struct {
	RegionID     *int64          `json:"region_id,omitempty"`
	SystemID     *int64          `json:"system_id,omitempty"`
	MinKills     *int64          `json:"min_kills,omitempty" minimum:"1"`
	MinISK       *float64        `json:"min_isk,omitempty"`
	Since        *string         `json:"since,omitempty"`
	Until        *string         `json:"until,omitempty"`
	Participants []StringOrInt64 `json:"participants,omitempty" maxItems:"6"`
	Opposing     bool            `json:"opposing,omitempty" default:"false"`
	Sort         string          `json:"sort,omitempty" enum:"isk,kills,recent,intensity" default:"isk"`
	Limit        int             `json:"limit,omitempty" default:"20" minimum:"1" maximum:"50"`
}

type ResolvedParticipant struct {
	ID   int64      `json:"id"`
	Name string     `json:"name"`
	Type EntityType `json:"type"`
}

type BattleTopAlliance struct {
	AllianceID int64   `json:"alliance_id"`
	Name       *string `json:"name"`
	Ticker     *string `json:"ticker"`
}

type FoundBattle struct {
	BattleID              int64               `json:"battle_id"`
	URL                   string              `json:"url"`
	StartTime             *time.Time          `json:"start_time"`
	EndTime               *time.Time          `json:"end_time"`
	DurationMinutes       int64               `json:"duration_minutes"`
	KillCount             int64               `json:"kill_count"`
	TotalISKDestroyed     float64             `json:"total_isk_destroyed"`
	IntensityISKPerMinute *float64            `json:"intensity_isk_per_minute"`
	IsMultiParty          bool                `json:"is_multi_party"`
	CorporationsInvolved  *int64              `json:"corporations_involved"`
	AlliancesInvolved     *int64              `json:"alliances_involved"`
	TopAllianceByISK      *BattleTopAlliance  `json:"top_alliance_by_isk"`
	System                BattleSystemSummary `json:"system"`
}

type FindBattlesOutput struct {
	Count                int                   `json:"count"`
	Battles              []FoundBattle         `json:"battles"`
	ParticipantsResolved []ResolvedParticipant `json:"participants_resolved,omitempty"`
	OpposingRequired     bool                  `json:"opposing_required,omitempty"`
}

func registerBattleTools(registry *Registry) error {
	if err := addTool(registry, ToolDefinition{
		Name: "battle_report", Title: "Get a battle report",
		Description: "Break down one battle by team and alliance or corporation, using recomputed final-blow and loss totals from raw killmails.",
	}, func(ctx context.Context, input BattleReportInput) (BattleReportOutput, error) {
		return battleReport(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	return addTool(registry, ToolDefinition{
		Name: "find_battles", Title: "Find battles",
		Description: "Find recent battles by location, size, value, time, participants, opposing-team requirement, and sort order.",
	}, func(ctx context.Context, input FindBattlesInput) (FindBattlesOutput, error) {
		return findBattles(ctx, registry.deps, input)
	})
}

type battleTotals struct {
	kills, losses         int64
	iskDestroyed, iskLost float64
}

type recomputedBattle struct {
	corporations map[int64]battleTotals
	teams        map[int]battleTotals
	killCount    int64
	iskDestroyed float64
}

func battleReport(ctx context.Context, deps Dependencies, input BattleReportInput) (BattleReportOutput, error) {
	if input.BattleID <= 0 {
		return BattleReportOutput{}, fmt.Errorf("invalid battle_id")
	}
	battleRows, err := queryMaps(ctx, deps.DB, `
		SELECT battle.*, system.system_name, system.security AS system_security, region.name AS region_name
		FROM battles battle LEFT JOIN solar_systems system ON system.solar_system_id = battle.solar_system_id
		LEFT JOIN regions region ON region.region_id = battle.region_id
		WHERE battle.battle_id = $1 LIMIT 1`, input.BattleID)
	if err != nil {
		return BattleReportOutput{}, err
	}
	battle := firstMap(battleRows)
	if battle == nil {
		return BattleReportOutput{}, fmt.Errorf("battle %d not found", input.BattleID)
	}
	teamRows, err := queryMaps(ctx, deps.DB, `SELECT * FROM battle_teams WHERE battle_id = $1 ORDER BY team_index`, input.BattleID)
	if err != nil {
		return BattleReportOutput{}, err
	}
	memberRows, err := queryMaps(ctx, deps.DB, `
		SELECT member.*, corporation.name AS corporation_name, corporation.ticker AS corporation_ticker,
		       alliance.name AS alliance_name, alliance.ticker AS alliance_ticker
		FROM battle_team_members member
		LEFT JOIN corporations corporation ON corporation.corporation_id = member.corporation_id
		LEFT JOIN alliances alliance ON alliance.alliance_id = member.alliance_id
		WHERE member.battle_team_id IN (SELECT id FROM battle_teams WHERE battle_id = $1)`, input.BattleID)
	if err != nil {
		return BattleReportOutput{}, err
	}
	recomputed, err := recomputeBattle(ctx, deps, battle, teamRows)
	if err != nil {
		return BattleReportOutput{}, err
	}
	level := input.Level
	if level == "" {
		level = "alliance"
	}
	teamMembers := map[int64][]BattleMember{}
	for _, row := range memberRows {
		corporationID := nullableInt64(row["corporation_id"])
		stats := recomputed.corporations[derefInt64(corporationID)]
		teamMembers[valueInt64(row["battle_team_id"])] = append(teamMembers[valueInt64(row["battle_team_id"])], BattleMember{
			CorporationID: corporationID, CorporationName: nullableString(row["corporation_name"]),
			CorporationTicker: nullableString(row["corporation_ticker"]), AllianceID: nullableInt64(row["alliance_id"]),
			AllianceName: nullableString(row["alliance_name"]), AllianceTicker: nullableString(row["alliance_ticker"]),
			Kills: stats.kills, Losses: stats.losses, ISKDestroyed: stats.iskDestroyed, ISKLost: stats.iskLost,
		})
	}
	output := BattleReportOutput{
		BattleID: input.BattleID, URL: entityURL(deps.BaseURL, EntityType("battle"), input.BattleID),
		StartTime: nullableTime(battle["start_time"]), EndTime: nullableTime(battle["end_time"]),
		DurationMinutes: valueInt64(battle["duration_minutes"]), KillCount: recomputed.killCount,
		TotalISKDestroyed: recomputed.iskDestroyed, IsMultiParty: valueBool(battle["is_multi_party"]),
		IsCustom: valueBool(battle["is_custom"]), Level: level,
		System: &BattleSystemSummary{ID: valueInt64(battle["solar_system_id"]), Name: nullableString(battle["system_name"]), RegionID: nullableInt64(battle["region_id"]), RegionName: nullableString(battle["region_name"])},
		Teams:  []BattleTeam{},
	}
	for _, teamRow := range teamRows {
		index, teamID := int(valueInt64(teamRow["team_index"])), valueInt64(teamRow["id"])
		members := teamMembers[teamID]
		if level == "alliance" {
			members = rollupBattleMembers(members)
		}
		sortBattleMembers(members)
		stats := recomputed.teams[index]
		output.Teams = append(output.Teams, BattleTeam{
			TeamIndex: index, TotalKills: stats.kills, TotalLosses: stats.losses,
			TotalISKDestroyed: stats.iskDestroyed, TotalISKLost: stats.iskLost, Members: members,
		})
	}
	if input.Format == "summary" {
		summary := renderBattleReportSummary(output)
		return BattleReportOutput{BattleID: output.BattleID, URL: output.URL, Summary: summary}, nil
	}
	return output, nil
}

func recomputeBattle(ctx context.Context, deps Dependencies, battle map[string]any, teamRows []map[string]any) (recomputedBattle, error) {
	systemID, battleID := valueInt64(battle["solar_system_id"]), valueInt64(battle["battle_id"])
	killmails, err := queryMaps(ctx, deps.DB, `
		SELECT killmail_id, victim_corporation_id, victim_alliance_id, victim_ship_type_id, total_value
		FROM killmails WHERE solar_system_id = $1 AND killmail_time >= $2 AND killmail_time <= $3`,
		systemID, battle["start_time"], battle["end_time"])
	if err != nil {
		return recomputedBattle{}, err
	}
	members, err := queryMaps(ctx, deps.DB, `
		SELECT member.corporation_id, member.alliance_id, team.team_index
		FROM battle_team_members member JOIN battle_teams team ON team.id = member.battle_team_id
		WHERE team.battle_id = $1`, battleID)
	if err != nil {
		return recomputedBattle{}, err
	}
	result := recomputedBattle{corporations: map[int64]battleTotals{}, teams: map[int]battleTotals{}}
	for _, row := range teamRows {
		result.teams[int(valueInt64(row["team_index"]))] = battleTotals{}
	}
	corpTeam, allianceTeam := map[int64]int{}, map[int64]int{}
	for _, row := range members {
		index := int(valueInt64(row["team_index"]))
		corpTeam[valueInt64(row["corporation_id"])] = index
		if alliance := nullableInt64(row["alliance_id"]); alliance != nil {
			allianceTeam[*alliance] = index
		}
	}
	if len(killmails) == 0 {
		return result, nil
	}
	killmailIDs, shipIDs := []int64{}, []int64{}
	for _, row := range killmails {
		killmailIDs = append(killmailIDs, valueInt64(row["killmail_id"]))
		if ship := nullableInt64(row["victim_ship_type_id"]); ship != nil {
			shipIDs = append(shipIDs, *ship)
		}
	}
	customPrices, err := queryMaps(ctx, deps.DB, `
		SELECT DISTINCT ON (type_id) type_id, price FROM custom_prices
		WHERE type_id = ANY($1) ORDER BY type_id, date DESC`, shipIDs)
	if err != nil {
		return recomputedBattle{}, err
	}
	battleDate := ""
	if endTime := nullableTime(battle["end_time"]); endTime != nil {
		battleDate = endTime.Format("2006-01-02")
	}
	marketPrices, err := queryMaps(ctx, deps.DB, `
		SELECT DISTINCT ON (type_id) type_id, average FROM prices
		WHERE type_id = ANY($1) AND region_id = 10000002 AND date <= $2::date
		ORDER BY type_id, date DESC`, shipIDs, battleDate)
	if err != nil {
		return recomputedBattle{}, err
	}
	finalBlows, err := queryMaps(ctx, deps.DB, `
		SELECT killmail_id, corporation_id, alliance_id FROM killmail_attackers
		WHERE killmail_id = ANY($1) AND final_blow = true`, killmailIDs)
	if err != nil {
		return recomputedBattle{}, err
	}
	market := map[int64]float64{}
	for _, row := range marketPrices {
		market[valueInt64(row["type_id"])] = valueFloat64(row["average"])
	}
	delta := map[int64]float64{}
	for _, row := range customPrices {
		id := valueInt64(row["type_id"])
		delta[id] = valueFloat64(row["price"]) - market[id]
	}
	finalByKill := map[int64]map[string]any{}
	for _, row := range finalBlows {
		if row["corporation_id"] != nil {
			finalByKill[valueInt64(row["killmail_id"])] = row
		}
	}
	sideOf := func(corporation, alliance *int64) (int, bool) {
		if corporation != nil {
			if team, ok := corpTeam[*corporation]; ok {
				return team, true
			}
		}
		if alliance != nil {
			team, ok := allianceTeam[*alliance]
			return team, ok
		}
		return 0, false
	}
	for _, killmail := range killmails {
		value := valueFloat64(killmail["total_value"]) + delta[valueInt64(killmail["victim_ship_type_id"])]
		victimCorp, victimAlliance := nullableInt64(killmail["victim_corporation_id"]), nullableInt64(killmail["victim_alliance_id"])
		victimTeam, victimIncluded := sideOf(victimCorp, victimAlliance)
		if victimIncluded {
			team := result.teams[victimTeam]
			team.losses, team.iskLost = team.losses+1, team.iskLost+value
			result.teams[victimTeam] = team
			if victimCorp != nil {
				corp := result.corporations[*victimCorp]
				corp.losses, corp.iskLost = corp.losses+1, corp.iskLost+value
				result.corporations[*victimCorp] = corp
			}
		}
		final := finalByKill[valueInt64(killmail["killmail_id"])]
		killerCorp, killerAlliance := nullableInt64(final["corporation_id"]), nullableInt64(final["alliance_id"])
		killerTeam, killerIncluded := sideOf(killerCorp, killerAlliance)
		if killerIncluded && killerCorp != nil {
			team := result.teams[killerTeam]
			team.kills, team.iskDestroyed = team.kills+1, team.iskDestroyed+value
			result.teams[killerTeam] = team
			corp := result.corporations[*killerCorp]
			corp.kills, corp.iskDestroyed = corp.kills+1, corp.iskDestroyed+value
			result.corporations[*killerCorp] = corp
		}
		if victimIncluded || killerIncluded {
			result.killCount++
			result.iskDestroyed += value
		}
	}
	return result, nil
}

func rollupBattleMembers(members []BattleMember) []BattleMember {
	output, index := []BattleMember{}, map[string]int{}
	for _, member := range members {
		key := fmt.Sprintf("c:%d", derefInt64(member.CorporationID))
		if member.AllianceID != nil {
			key = fmt.Sprintf("a:%d", *member.AllianceID)
		}
		if position, ok := index[key]; ok {
			output[position].Kills += member.Kills
			output[position].Losses += member.Losses
			output[position].ISKDestroyed += member.ISKDestroyed
			output[position].ISKLost += member.ISKLost
			output[position].CorporationCount++
			continue
		}
		member.CorporationCount = 1
		if member.AllianceID != nil {
			member.CorporationID, member.CorporationName, member.CorporationTicker = nil, nil, nil
		}
		index[key], output = len(output), append(output, member)
	}
	return output
}

func sortBattleMembers(members []BattleMember) {
	for i := 1; i < len(members); i++ {
		for j := i; j > 0 && members[j].Kills+members[j].Losses > members[j-1].Kills+members[j-1].Losses; j-- {
			members[j], members[j-1] = members[j-1], members[j]
		}
	}
}

func renderBattleReportSummary(output BattleReportOutput) string {
	start := "unknown time"
	if output.StartTime != nil {
		start = output.StartTime.UTC().Format("2006-01-02 15:04 UTC")
	}
	system := fmt.Sprintf("system %d", output.System.ID)
	if output.System.Name != nil {
		system = *output.System.Name
	}
	teams := []string{}
	for _, team := range output.Teams {
		labels := []string{}
		for _, member := range team.Members {
			if len(labels) >= 3 {
				break
			}
			name := member.AllianceTicker
			if name == nil {
				name = member.CorporationTicker
			}
			if name != nil {
				labels = append(labels, *name)
			}
		}
		teams = append(teams, fmt.Sprintf("Team %d (%s): %d kills / %d losses, %s destroyed", team.TeamIndex, strings.Join(labels, " / "), team.TotalKills, team.TotalLosses, formatISK(team.TotalISKDestroyed)))
	}
	return fmt.Sprintf("Battle %d on %s in %s — %d kills, %s destroyed over %dm. %s.",
		output.BattleID, start, system, output.KillCount, formatISK(output.TotalISKDestroyed), output.DurationMinutes, strings.Join(teams, ". "))
}

func findBattles(ctx context.Context, deps Dependencies, input FindBattlesInput) (FindBattlesOutput, error) {
	limit := input.Limit
	if limit == 0 {
		limit = 20
	}
	limit = clamp(limit, 1, 50)
	since := input.Since
	if since == nil {
		value := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339Nano)
		since = &value
	}
	resolved := make([]ResolvedParticipant, 0, len(input.Participants))
	filters, args := []string{"battle.start_time >= $1"}, []any{*since}
	add := func(filter string, value any) {
		args = append(args, value)
		filters = append(filters, fmt.Sprintf(filter, len(args)))
	}
	if input.Until != nil {
		add("battle.start_time <= $%d", *input.Until)
	}
	if input.RegionID != nil {
		add("battle.region_id = $%d", *input.RegionID)
	}
	if input.SystemID != nil {
		add("battle.solar_system_id = $%d", *input.SystemID)
	}
	if input.MinKills != nil {
		add("battle.kill_count >= $%d", *input.MinKills)
	}
	if input.MinISK != nil {
		add("battle.total_isk_destroyed >= $%d", *input.MinISK)
	}
	participantPredicates := []string{}
	for index, reference := range input.Participants {
		entity, err := resolveEntity(ctx, deps, reference, nil)
		if err != nil {
			return FindBattlesOutput{}, err
		}
		if entity == nil || (entity.Type != EntityCorporation && entity.Type != EntityAlliance) {
			return FindBattlesOutput{}, fmt.Errorf("participant %d did not resolve to a corporation or alliance", index)
		}
		column := "corporation_id"
		if entity.Type == EntityAlliance {
			column = "alliance_id"
		}
		args = append(args, entity.ID)
		predicate := fmt.Sprintf("member.%s = $%d", column, len(args))
		participantPredicates = append(participantPredicates, predicate)
		filters = append(filters, "EXISTS (SELECT 1 FROM battle_team_members member JOIN battle_teams team ON team.id = member.battle_team_id WHERE team.battle_id = battle.battle_id AND "+predicate+")")
		resolved = append(resolved, ResolvedParticipant{ID: entity.ID, Name: entity.Name, Type: entity.Type})
	}
	if input.Opposing && len(participantPredicates) >= 2 {
		filters = append(filters, "(SELECT COUNT(DISTINCT team.team_index) FROM battle_team_members member JOIN battle_teams team ON team.id = member.battle_team_id WHERE team.battle_id = battle.battle_id AND ("+strings.Join(participantPredicates, " OR ")+")) >= 2")
	}
	order := "battle.total_isk_destroyed DESC NULLS LAST"
	switch input.Sort {
	case "kills":
		order = "battle.kill_count DESC NULLS LAST"
	case "recent":
		order = "battle.start_time DESC"
	case "intensity":
		order = "(battle.total_isk_destroyed / NULLIF(battle.duration_minutes, 0)) DESC NULLS LAST"
	}
	args = append(args, limit)
	query := fmt.Sprintf(`
		SELECT battle.battle_id, battle.solar_system_id, system.system_name, battle.region_id,
		       region.name AS region_name, battle.start_time, battle.end_time, battle.duration_minutes,
		       battle.kill_count, battle.total_isk_destroyed, battle.is_multi_party
		FROM battles battle LEFT JOIN solar_systems system ON system.solar_system_id = battle.solar_system_id
		LEFT JOIN regions region ON region.region_id = battle.region_id
		WHERE %s ORDER BY %s LIMIT $%d`, strings.Join(filters, " AND "), order, len(args))
	rows, err := queryMaps(ctx, deps.DB, query, args...)
	if err != nil {
		return FindBattlesOutput{}, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, valueInt64(row["battle_id"]))
	}
	enrichment, err := loadBattleListEnrichment(ctx, deps, ids)
	if err != nil {
		return FindBattlesOutput{}, err
	}
	output := FindBattlesOutput{Count: len(rows), Battles: []FoundBattle{}, ParticipantsResolved: resolved, OpposingRequired: input.Opposing && len(resolved) > 0}
	for _, row := range rows {
		id, duration, isk := valueInt64(row["battle_id"]), valueInt64(row["duration_minutes"]), valueFloat64(row["total_isk_destroyed"])
		var intensity *float64
		if duration > 0 {
			value := math.Round(isk / float64(duration))
			intensity = &value
		}
		enrich := enrichment[id]
		output.Battles = append(output.Battles, FoundBattle{
			BattleID: id, URL: entityURL(deps.BaseURL, EntityType("battle"), id),
			StartTime: nullableTime(row["start_time"]), EndTime: nullableTime(row["end_time"]),
			DurationMinutes: duration, KillCount: valueInt64(row["kill_count"]), TotalISKDestroyed: isk,
			IntensityISKPerMinute: intensity, IsMultiParty: valueBool(row["is_multi_party"]),
			CorporationsInvolved: enrich.corporations, AlliancesInvolved: enrich.alliances, TopAllianceByISK: enrich.topAlliance,
			System: BattleSystemSummary{ID: valueInt64(row["solar_system_id"]), Name: nullableString(row["system_name"]), RegionID: nullableInt64(row["region_id"]), RegionName: nullableString(row["region_name"])},
		})
	}
	return output, nil
}

type battleListEnrichment struct {
	corporations, alliances *int64
	topAlliance             *BattleTopAlliance
}

func loadBattleListEnrichment(ctx context.Context, deps Dependencies, ids []int64) (map[int64]battleListEnrichment, error) {
	output := map[int64]battleListEnrichment{}
	if len(ids) == 0 {
		return output, nil
	}
	rows, err := queryMaps(ctx, deps.DB, `
		WITH top_alliance AS (
			SELECT team.battle_id, member.alliance_id, alliance.name, alliance.ticker,
			       ROW_NUMBER() OVER (PARTITION BY team.battle_id ORDER BY SUM(member.isk_destroyed) DESC NULLS LAST) AS rank
			FROM battle_team_members member JOIN battle_teams team ON team.id = member.battle_team_id
			LEFT JOIN alliances alliance ON alliance.alliance_id = member.alliance_id
			WHERE team.battle_id = ANY($1) AND member.alliance_id IS NOT NULL
			GROUP BY team.battle_id, member.alliance_id, alliance.name, alliance.ticker
		), counts AS (
			SELECT team.battle_id, COUNT(DISTINCT member.corporation_id) AS corporations,
			       COUNT(DISTINCT member.alliance_id) FILTER (WHERE member.alliance_id IS NOT NULL) AS alliances
			FROM battle_team_members member JOIN battle_teams team ON team.id = member.battle_team_id
			WHERE team.battle_id = ANY($1) GROUP BY team.battle_id
		)
		SELECT counts.*, top_alliance.alliance_id, top_alliance.name, top_alliance.ticker
		FROM counts LEFT JOIN top_alliance ON top_alliance.battle_id = counts.battle_id AND top_alliance.rank = 1`, ids)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		value := battleListEnrichment{corporations: nullableInt64(row["corporations"]), alliances: nullableInt64(row["alliances"])}
		if allianceID := nullableInt64(row["alliance_id"]); allianceID != nil {
			value.topAlliance = &BattleTopAlliance{AllianceID: *allianceID, Name: nullableString(row["name"]), Ticker: nullableString(row["ticker"])}
		}
		output[valueInt64(row["battle_id"])] = value
	}
	return output, nil
}
