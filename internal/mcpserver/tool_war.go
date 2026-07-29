package mcpserver

import (
	"context"
	"fmt"
	"math"
	"time"
)

type WarReportInput struct {
	A          StringOrInt64 `json:"a"`
	B          StringOrInt64 `json:"b"`
	Since      *string       `json:"since,omitempty"`
	Until      *string       `json:"until,omitempty"`
	TopSystems int           `json:"top_systems,omitempty" default:"10" minimum:"1" maximum:"50"`
	TopBattles int           `json:"top_battles,omitempty" default:"5" minimum:"1" maximum:"20"`
}

type WarTotals struct {
	AKillsB       int64   `json:"a_kills_b"`
	BKillsA       int64   `json:"b_kills_a"`
	AISKDestroyed float64 `json:"a_isk_destroyed"`
	BISKDestroyed float64 `json:"b_isk_destroyed"`
	TotalKills    int64   `json:"total_kills"`
	TotalISK      float64 `json:"total_isk"`
	Leader        string  `json:"leader"`
	AISKShare     float64 `json:"a_isk_share"`
}

type WarTimelineDay struct {
	PeriodStart time.Time `json:"period_start"`
	AKillsB     int64     `json:"a_kills_b"`
	BKillsA     int64     `json:"b_kills_a"`
	AISKOnB     float64   `json:"a_isk_on_b"`
	BISKOnA     float64   `json:"b_isk_on_a"`
}

type ContestedSystem struct {
	SolarSystemID int64   `json:"solar_system_id"`
	SystemName    *string `json:"system_name"`
	AKillsB       int64   `json:"a_kills_b"`
	BKillsA       int64   `json:"b_kills_a"`
	TotalKills    int64   `json:"total_kills"`
	TotalISK      float64 `json:"total_isk"`
}

type BattleAllianceSummary struct {
	AllianceID   int64   `json:"alliance_id"`
	Name         *string `json:"name"`
	Ticker       *string `json:"ticker"`
	ISKDestroyed float64 `json:"isk_destroyed"`
	Kills        int64   `json:"kills"`
	Losses       int64   `json:"losses"`
}

type BattleTeamSummary struct {
	TeamIndex     int                     `json:"team_index"`
	TopAlliances  []BattleAllianceSummary `json:"top_alliances"`
	AllianceCount int                     `json:"alliance_count"`
}

type BattleSystemSummary struct {
	ID         int64   `json:"id"`
	Name       *string `json:"name"`
	RegionID   *int64  `json:"region_id"`
	RegionName *string `json:"region_name"`
}

type WarBattleSummary struct {
	BattleID          int64               `json:"battle_id"`
	URL               string              `json:"url"`
	StartTime         *time.Time          `json:"start_time"`
	EndTime           *time.Time          `json:"end_time"`
	DurationMinutes   int64               `json:"duration_minutes"`
	KillCount         int64               `json:"kill_count"`
	TotalISKDestroyed float64             `json:"total_isk_destroyed"`
	IsMultiParty      bool                `json:"is_multi_party"`
	System            BattleSystemSummary `json:"system"`
	Teams             []BattleTeamSummary `json:"teams"`
}

type WarReportOutput struct {
	A                   Entity             `json:"a"`
	B                   Entity             `json:"b"`
	Window              TimeWindow         `json:"window"`
	Totals              WarTotals          `json:"totals"`
	TimelineDaily       []WarTimelineDay   `json:"timeline_daily"`
	TopContestedSystems []ContestedSystem  `json:"top_contested_systems"`
	RecentBattles       []WarBattleSummary `json:"recent_battles"`
}

func registerWarTool(registry *Registry) error {
	return addTool(registry, ToolDefinition{
		Name: "war_report", Title: "Build a head-to-head war report",
		Description: "Head-to-head report between two characters, corporations, or alliances, including directional kills and ISK, a daily timeline, contested systems, and shared battles.",
	}, func(ctx context.Context, input WarReportInput) (WarReportOutput, error) {
		return warReport(ctx, registry.deps, input)
	})
}

func warReport(ctx context.Context, deps Dependencies, input WarReportInput) (WarReportOutput, error) {
	a, err := resolveEntity(ctx, deps, input.A, nil)
	if err != nil || a == nil {
		if err != nil {
			return WarReportOutput{}, err
		}
		return WarReportOutput{}, fmt.Errorf("could not resolve a")
	}
	b, err := resolveEntity(ctx, deps, input.B, nil)
	if err != nil || b == nil {
		if err != nil {
			return WarReportOutput{}, err
		}
		return WarReportOutput{}, fmt.Errorf("could not resolve b")
	}
	aAttacker, aVictim := organizationAttackerColumns[a.Type], organizationVictimColumns[a.Type]
	bAttacker, bVictim := organizationAttackerColumns[b.Type], organizationVictimColumns[b.Type]
	if aAttacker == "" || bAttacker == "" {
		return WarReportOutput{}, fmt.Errorf("both sides must be characters, corporations, or alliances")
	}
	since, until, err := parseVSWindow(input.Since, input.Until, 30)
	if err != nil {
		return WarReportOutput{}, err
	}
	topSystems, topBattles := input.TopSystems, input.TopBattles
	if topSystems == 0 {
		topSystems = 10
	}
	if topBattles == 0 {
		topBattles = 5
	}
	topSystems, topBattles = clamp(topSystems, 1, 50), clamp(topBattles, 1, 20)
	directionQuery := func(victimColumn, attackerColumn string) string {
		return fmt.Sprintf(`
			SELECT COUNT(*)::bigint AS kills, COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed
			FROM killmails k WHERE k.%s = $1 AND k.killmail_time >= $3 AND k.killmail_time <= $4
			  AND EXISTS (SELECT 1 FROM killmail_attackers ka WHERE ka.killmail_id = k.killmail_id AND ka.%s = $2)`,
			victimColumn, attackerColumn)
	}
	aOnB, err := queryMaps(ctx, deps.DB, directionQuery(bVictim, aAttacker), b.ID, a.ID, since, until)
	if err != nil {
		return WarReportOutput{}, err
	}
	bOnA, err := queryMaps(ctx, deps.DB, directionQuery(aVictim, bAttacker), a.ID, b.ID, since, until)
	if err != nil {
		return WarReportOutput{}, err
	}
	union := fmt.Sprintf(`
		SELECT period_start, SUM(a_kills_b)::bigint AS a_kills_b, SUM(b_kills_a)::bigint AS b_kills_a,
		       SUM(a_isk_on_b)::double precision AS a_isk_on_b, SUM(b_isk_on_a)::double precision AS b_isk_on_a
		FROM (
			SELECT date_trunc('day', k.killmail_time)::date AS period_start, 1 AS a_kills_b, 0 AS b_kills_a,
			       COALESCE(k.total_value, 0) AS a_isk_on_b, 0 AS b_isk_on_a
			FROM killmails k WHERE k.%s = $1 AND k.killmail_time >= $3 AND k.killmail_time <= $4
			  AND EXISTS (SELECT 1 FROM killmail_attackers ka WHERE ka.killmail_id = k.killmail_id AND ka.%s = $2)
			UNION ALL
			SELECT date_trunc('day', k.killmail_time)::date, 0, 1, 0, COALESCE(k.total_value, 0)
			FROM killmails k WHERE k.%s = $2 AND k.killmail_time >= $3 AND k.killmail_time <= $4
			  AND EXISTS (SELECT 1 FROM killmail_attackers ka WHERE ka.killmail_id = k.killmail_id AND ka.%s = $1)
		) directions GROUP BY period_start ORDER BY period_start`, bVictim, aAttacker, aVictim, bAttacker)
	timelineRows, err := queryMaps(ctx, deps.DB, union, b.ID, a.ID, since, until)
	if err != nil {
		return WarReportOutput{}, err
	}
	systemsQuery := fmt.Sprintf(`
		SELECT solar_system_id, system_name, SUM(a_kills_b)::bigint AS a_kills_b,
		       SUM(b_kills_a)::bigint AS b_kills_a, SUM(total_kills)::bigint AS total_kills,
		       SUM(total_isk)::double precision AS total_isk
		FROM (
			SELECT k.solar_system_id, s.system_name, 1 AS a_kills_b, 0 AS b_kills_a,
			       1 AS total_kills, COALESCE(k.total_value, 0) AS total_isk
			FROM killmails k LEFT JOIN solar_systems s ON s.solar_system_id = k.solar_system_id
			WHERE k.%s = $1 AND k.killmail_time >= $3 AND k.killmail_time <= $4
			  AND EXISTS (SELECT 1 FROM killmail_attackers ka WHERE ka.killmail_id = k.killmail_id AND ka.%s = $2)
			UNION ALL
			SELECT k.solar_system_id, s.system_name, 0, 1, 1, COALESCE(k.total_value, 0)
			FROM killmails k LEFT JOIN solar_systems s ON s.solar_system_id = k.solar_system_id
			WHERE k.%s = $2 AND k.killmail_time >= $3 AND k.killmail_time <= $4
			  AND EXISTS (SELECT 1 FROM killmail_attackers ka WHERE ka.killmail_id = k.killmail_id AND ka.%s = $1)
		) directions WHERE solar_system_id IS NOT NULL
		GROUP BY solar_system_id, system_name ORDER BY total_kills DESC LIMIT $5`,
		bVictim, aAttacker, aVictim, bAttacker)
	systemRows, err := queryMaps(ctx, deps.DB, systemsQuery, b.ID, a.ID, since, until, topSystems)
	if err != nil {
		return WarReportOutput{}, err
	}
	battles := []WarBattleSummary{}
	if a.Type != EntityCharacter && b.Type != EntityCharacter {
		battles, err = loadWarBattles(ctx, deps, *a, *b, since, until, topBattles)
		if err != nil {
			return WarReportOutput{}, err
		}
	}
	ab, ba := firstMap(aOnB), firstMap(bOnA)
	aKills, bKills := valueInt64(ab["kills"]), valueInt64(ba["kills"])
	aISK, bISK := valueFloat64(ab["isk_destroyed"]), valueFloat64(ba["isk_destroyed"])
	totalISK := aISK + bISK
	leader := "tied"
	if aISK > bISK {
		leader = "a"
	} else if bISK > aISK {
		leader = "b"
	}
	output := WarReportOutput{
		A: a.Public(deps.BaseURL), B: b.Public(deps.BaseURL),
		Window:        TimeWindow{Since: since.Format(time.RFC3339Nano), Until: until.Format(time.RFC3339Nano)},
		Totals:        WarTotals{AKillsB: aKills, BKillsA: bKills, AISKDestroyed: aISK, BISKDestroyed: bISK, TotalKills: aKills + bKills, TotalISK: totalISK, Leader: leader},
		TimelineDaily: []WarTimelineDay{}, TopContestedSystems: []ContestedSystem{}, RecentBattles: battles,
	}
	if totalISK > 0 {
		output.Totals.AISKShare = math.Round(aISK/totalISK*10000) / 100
	}
	for _, row := range timelineRows {
		period := nullableTime(row["period_start"])
		if period != nil {
			output.TimelineDaily = append(output.TimelineDaily, WarTimelineDay{PeriodStart: *period, AKillsB: valueInt64(row["a_kills_b"]), BKillsA: valueInt64(row["b_kills_a"]), AISKOnB: valueFloat64(row["a_isk_on_b"]), BISKOnA: valueFloat64(row["b_isk_on_a"])})
		}
	}
	for _, row := range systemRows {
		output.TopContestedSystems = append(output.TopContestedSystems, ContestedSystem{SolarSystemID: valueInt64(row["solar_system_id"]), SystemName: nullableString(row["system_name"]), AKillsB: valueInt64(row["a_kills_b"]), BKillsA: valueInt64(row["b_kills_a"]), TotalKills: valueInt64(row["total_kills"]), TotalISK: valueFloat64(row["total_isk"])})
	}
	return output, nil
}

func loadWarBattles(ctx context.Context, deps Dependencies, a, b ResolvedEntity, since, until time.Time, limit int) ([]WarBattleSummary, error) {
	aColumn, bColumn := "alliance_id", "alliance_id"
	if a.Type == EntityCorporation {
		aColumn = "corporation_id"
	}
	if b.Type == EntityCorporation {
		bColumn = "corporation_id"
	}
	query := fmt.Sprintf(`
		SELECT battle.battle_id, battle.solar_system_id, system.system_name, battle.region_id,
		       region.name AS region_name, battle.start_time, battle.end_time, battle.duration_minutes,
		       battle.kill_count, battle.total_isk_destroyed, battle.is_multi_party
		FROM battles battle
		LEFT JOIN solar_systems system ON system.solar_system_id = battle.solar_system_id
		LEFT JOIN regions region ON region.region_id = battle.region_id
		WHERE battle.start_time >= $3 AND battle.start_time <= $4
		  AND EXISTS (SELECT 1 FROM battle_team_members member JOIN battle_teams team ON team.id = member.battle_team_id WHERE team.battle_id = battle.battle_id AND member.%s = $1)
		  AND EXISTS (SELECT 1 FROM battle_team_members member JOIN battle_teams team ON team.id = member.battle_team_id WHERE team.battle_id = battle.battle_id AND member.%s = $2)
		  AND (SELECT COUNT(DISTINCT team.team_index) FROM battle_team_members member JOIN battle_teams team ON team.id = member.battle_team_id
		       WHERE team.battle_id = battle.battle_id AND (member.%s = $1 OR member.%s = $2)) >= 2
		ORDER BY battle.total_isk_destroyed DESC NULLS LAST LIMIT $5`, aColumn, bColumn, aColumn, bColumn)
	rows, err := queryMaps(ctx, deps.DB, query, a.ID, b.ID, since, until, limit)
	if err != nil {
		return nil, err
	}
	battleIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		battleIDs = append(battleIDs, valueInt64(row["battle_id"]))
	}
	teamsByBattle, err := loadBattleAllianceTeams(ctx, deps, battleIDs)
	if err != nil {
		return nil, err
	}
	output := make([]WarBattleSummary, 0, len(rows))
	for _, row := range rows {
		id := valueInt64(row["battle_id"])
		output = append(output, WarBattleSummary{
			BattleID: id, URL: entityURL(deps.BaseURL, EntityType("battle"), id),
			StartTime: nullableTime(row["start_time"]), EndTime: nullableTime(row["end_time"]),
			DurationMinutes: valueInt64(row["duration_minutes"]), KillCount: valueInt64(row["kill_count"]),
			TotalISKDestroyed: valueFloat64(row["total_isk_destroyed"]), IsMultiParty: valueBool(row["is_multi_party"]),
			System: BattleSystemSummary{ID: valueInt64(row["solar_system_id"]), Name: nullableString(row["system_name"]), RegionID: nullableInt64(row["region_id"]), RegionName: nullableString(row["region_name"])},
			Teams:  teamsByBattle[id],
		})
	}
	return output, nil
}

func loadBattleAllianceTeams(ctx context.Context, deps Dependencies, battleIDs []int64) (map[int64][]BattleTeamSummary, error) {
	output := map[int64][]BattleTeamSummary{}
	if len(battleIDs) == 0 {
		return output, nil
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT team.battle_id, team.team_index, member.alliance_id, alliance.name, alliance.ticker,
		       SUM(member.isk_destroyed) AS isk, SUM(member.kills) AS kills, SUM(member.losses) AS losses
		FROM battle_team_members member JOIN battle_teams team ON team.id = member.battle_team_id
		LEFT JOIN alliances alliance ON alliance.alliance_id = member.alliance_id
		WHERE team.battle_id = ANY($1) AND member.alliance_id IS NOT NULL
		GROUP BY team.battle_id, team.team_index, member.alliance_id, alliance.name, alliance.ticker
		ORDER BY team.battle_id, team.team_index, SUM(member.isk_destroyed) DESC`, battleIDs)
	if err != nil {
		return nil, err
	}
	type key struct {
		battleID  int64
		teamIndex int
	}
	grouped := map[key][]BattleAllianceSummary{}
	for _, row := range rows {
		k := key{valueInt64(row["battle_id"]), int(valueInt64(row["team_index"]))}
		grouped[k] = append(grouped[k], BattleAllianceSummary{AllianceID: valueInt64(row["alliance_id"]), Name: nullableString(row["name"]), Ticker: nullableString(row["ticker"]), ISKDestroyed: valueFloat64(row["isk"]), Kills: valueInt64(row["kills"]), Losses: valueInt64(row["losses"])})
	}
	for key, alliances := range grouped {
		count := len(alliances)
		if len(alliances) > 3 {
			alliances = alliances[:3]
		}
		output[key.battleID] = append(output[key.battleID], BattleTeamSummary{TeamIndex: key.teamIndex, TopAlliances: alliances, AllianceCount: count})
	}
	return output, nil
}
