package mcpserver

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

type SystemPulseInput struct {
	System StringOrInt64 `json:"system" doc:"System name or id."`
	Hours  float64       `json:"hours,omitempty" default:"6" minimum:"0.5" maximum:"168"`
	Since  *string       `json:"since,omitempty" doc:"ISO datetime lower bound."`
	Until  *string       `json:"until,omitempty" doc:"ISO datetime upper bound."`
	TopN   int           `json:"top_n,omitempty" default:"5" minimum:"1" maximum:"20"`
}

type SystemRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type SystemPulseTotals struct {
	Kills          int64   `json:"kills"`
	PVPKills       int64   `json:"pvp_kills"`
	SoloKills      int64   `json:"solo_kills"`
	AttackersTotal int64   `json:"attackers_total"`
	ISKDestroyed   float64 `json:"isk_destroyed"`
}

type OrganizationActivity struct {
	ID     int64   `json:"id"`
	Name   *string `json:"name"`
	Ticker *string `json:"ticker"`
	Kills  int64   `json:"kills"`
}

type ShipLossActivity struct {
	TypeID  *int64  `json:"type_id"`
	Name    *string `json:"name"`
	Losses  int64   `json:"losses"`
	ISKLost float64 `json:"isk_lost"`
}

type SystemPulseOutput struct {
	System               SystemRef              `json:"system"`
	Window               TimeWindow             `json:"window"`
	WindowHours          *float64               `json:"window_hours"`
	Totals               SystemPulseTotals      `json:"totals"`
	HeatScore            float64                `json:"heat_score"`
	TopAttackerCorps     []OrganizationActivity `json:"top_attacker_corps"`
	TopAttackerAlliances []OrganizationActivity `json:"top_attacker_alliances"`
	TopVictimShips       []ShipLossActivity     `json:"top_victim_ships"`
}

type ExpensiveLossesInput struct {
	Days              int            `json:"days,omitempty" default:"30" minimum:"1" maximum:"365"`
	RegionID          *int64         `json:"region_id,omitempty"`
	SystemID          *int64         `json:"system_id,omitempty"`
	ShipTypeID        *int64         `json:"ship_type_id,omitempty"`
	VictimAllianceID  *int64         `json:"victim_alliance_id,omitempty"`
	VictimCharacterID *int64         `json:"victim_character_id,omitempty"`
	VS                *StringOrInt64 `json:"vs,omitempty" doc:"Character, corporation, or alliance present as attacker."`
	MinValue          *float64       `json:"min_value,omitempty"`
	Limit             int            `json:"limit,omitempty" default:"20" minimum:"1" maximum:"50"`
}

type LossSystem struct {
	ID         int64   `json:"id"`
	Name       *string `json:"name"`
	RegionID   *int64  `json:"region_id"`
	RegionName *string `json:"region_name"`
}

type LossShip struct {
	TypeID *int64  `json:"type_id"`
	Name   *string `json:"name"`
}

type ExpensiveLoss struct {
	KillmailID int64          `json:"killmail_id"`
	URL        string         `json:"url"`
	Time       *time.Time     `json:"time"`
	TotalValue float64        `json:"total_value"`
	System     LossSystem     `json:"system"`
	VictimShip LossShip       `json:"victim_ship"`
	Victim     KillmailVictim `json:"victim"`
}

type ExpensiveLossesOutput struct {
	WindowDays int             `json:"window_days"`
	Count      int             `json:"count"`
	Kills      []ExpensiveLoss `json:"kills"`
}

type DossierInput struct {
	Entity StringOrInt64 `json:"entity" doc:"Character name or id."`
	Type   *EntityType   `json:"type,omitempty" enum:"character"`
	Format string        `json:"format,omitempty" enum:"json,summary" default:"json"`
}

type DossierArchetypeLastSeen struct {
	LastFCSeen      *string `json:"last_fc_seen"`
	LastSuperKill   *string `json:"last_super_kill"`
	LastBlopsSeen   *string `json:"last_blops_seen"`
	LastCapitalKill *string `json:"last_capital_kill"`
	LastLogiSeen    *string `json:"last_logi_seen"`
}

type DossierPlaystyle struct {
	Dominant      string  `json:"dominant"`
	SoloPct       int     `json:"solo_pct"`
	SmallGangPct  int     `json:"small_gang_pct"`
	MidGangPct    int     `json:"mid_gang_pct"`
	FleetPct      int     `json:"fleet_pct"`
	BlobPct       int     `json:"blob_pct"`
	AvgFleetSize  float64 `json:"avg_fleet_size"`
	TotalKills90D int64   `json:"total_kills_90d"`
}

type DossierTopShip struct {
	TypeID int64   `json:"type_id"`
	Name   *string `json:"name"`
	Kills  int64   `json:"kills"`
}

type DossierTopSystem struct {
	SystemID int64   `json:"system_id"`
	Name     *string `json:"name"`
	Kills    int64   `json:"kills"`
}

type DossierWingmate struct {
	CharacterID int64   `json:"character_id"`
	Name        *string `json:"name"`
	SharedKills int64   `json:"shared_kills"`
	URL         string  `json:"url"`
}

type DossierOutput struct {
	Entity            Entity                    `json:"entity"`
	Summary           string                    `json:"summary,omitempty"`
	Lifetime          *EntityLifetime           `json:"lifetime,omitempty"`
	ArchetypeTags     []string                  `json:"archetype_tags,omitempty"`
	ArchetypeLastSeen *DossierArchetypeLastSeen `json:"archetype_last_seen,omitempty"`
	Playstyle90D      *DossierPlaystyle         `json:"playstyle_90d,omitempty"`
	TopShips          []DossierTopShip          `json:"top_ships,omitempty"`
	TopSystems        []DossierTopSystem        `json:"top_systems,omitempty"`
	TopWingmates      []DossierWingmate         `json:"top_wingmates,omitempty"`
}

type KillmailStoryFacts struct {
	Time              time.Time         `json:"time"`
	TotalValue        float64           `json:"total_value"`
	IsSolo            bool              `json:"is_solo"`
	IsNPC             bool              `json:"is_npc"`
	AttackerCount     int64             `json:"attacker_count"`
	System            KillmailLocation  `json:"system"`
	VictimShip        *string           `json:"victim_ship"`
	Victim            *string           `json:"victim"`
	VictimAffiliation *string           `json:"victim_affiliation"`
	FinalBlow         *KillmailAttacker `json:"final_blow"`
}

type KillmailStoryOutput struct {
	KillmailID int64              `json:"killmail_id"`
	URL        string             `json:"url"`
	Story      string             `json:"story"`
	Facts      KillmailStoryFacts `json:"facts"`
}

func registerFunTools(registry *Registry) error {
	if err := addTool(registry, ToolDefinition{
		Name: "system_pulse", Title: "Inspect system activity",
		Description: "Activity heat for a solar system over a recent or explicit historical window, including totals and top attackers and ships.",
	}, func(ctx context.Context, input SystemPulseInput) (SystemPulseOutput, error) {
		return systemPulse(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name: "expensive_losses", Title: "Find expensive losses",
		Description: "Most valuable killmails matching optional time, location, victim, ship, value, and opposing-entity filters.",
	}, func(ctx context.Context, input ExpensiveLossesInput) (ExpensiveLossesOutput, error) {
		return expensiveLosses(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name: "capsuleer_dossier", Title: "Build a capsuleer dossier",
		Description: "One-shot character intelligence report combining overview stats, archetype tags, top ships and systems, wingmates, and playstyle.",
	}, func(ctx context.Context, input DossierInput) (DossierOutput, error) {
		return capsuleerDossier(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	return addTool(registry, ToolDefinition{
		Name: "killmail_story", Title: "Narrate a killmail",
		Description: "Render one killmail as a short narrative with the victim, ship, system, attackers, final blow, value, security, and time.",
	}, func(ctx context.Context, input KillmailInput) (KillmailStoryOutput, error) {
		return killmailStory(ctx, registry.deps, input)
	})
}

func systemPulse(ctx context.Context, deps Dependencies, input SystemPulseInput) (SystemPulseOutput, error) {
	system, err := resolveEntity(ctx, deps, input.System, entityTypePointer(EntitySystem))
	if err != nil {
		return SystemPulseOutput{}, err
	}
	if system == nil || system.Type != EntitySystem {
		return SystemPulseOutput{}, fmt.Errorf("no solar system found for %q", input.System.String())
	}
	now := time.Now().UTC()
	hours := input.Hours
	if hours == 0 {
		hours = 6
	}
	hours = math.Max(0.5, math.Min(168, hours))
	since, until := now.Add(-time.Duration(hours*float64(time.Hour))), now
	historical := input.Since != nil || input.Until != nil
	if input.Since != nil {
		since, err = time.Parse(time.RFC3339, *input.Since)
		if err != nil {
			return SystemPulseOutput{}, fmt.Errorf("invalid since: %w", err)
		}
	}
	if input.Until != nil {
		until, err = time.Parse(time.RFC3339, *input.Until)
		if err != nil {
			return SystemPulseOutput{}, fmt.Errorf("invalid until: %w", err)
		}
	}
	limit := input.TopN
	if limit == 0 {
		limit = 5
	}
	limit = clamp(limit, 1, 20)
	overall, err := queryMaps(ctx, deps.DB, `
		SELECT COUNT(*)::bigint AS kills, COALESCE(SUM(attacker_count), 0)::bigint AS attackers_total,
		       COALESCE(SUM(total_value), 0)::double precision AS isk_destroyed,
		       COUNT(*) FILTER (WHERE is_npc = false)::bigint AS pvp_kills,
		       COUNT(*) FILTER (WHERE is_solo = true)::bigint AS solo_kills
		FROM killmails WHERE solar_system_id = $1 AND killmail_time >= $2 AND killmail_time <= $3`,
		system.ID, since, until)
	if err != nil {
		return SystemPulseOutput{}, err
	}
	corps, err := systemPulseOrganizations(ctx, deps, system.ID, since, until, limit, false)
	if err != nil {
		return SystemPulseOutput{}, err
	}
	alliances, err := systemPulseOrganizations(ctx, deps, system.ID, since, until, limit, true)
	if err != nil {
		return SystemPulseOutput{}, err
	}
	ships, err := queryMaps(ctx, deps.DB, `
		SELECT t.type_id, t.name, COUNT(*)::bigint AS losses,
		       COALESCE(SUM(k.total_value), 0)::double precision AS isk_lost
		FROM killmails k LEFT JOIN inv_types t ON t.type_id = k.victim_ship_type_id
		WHERE k.solar_system_id = $1 AND k.killmail_time >= $2 AND k.killmail_time <= $3
		  AND k.victim_ship_type_id IS NOT NULL
		GROUP BY t.type_id, t.name ORDER BY losses DESC LIMIT $4`, system.ID, since, until, limit)
	if err != nil {
		return SystemPulseOutput{}, err
	}
	row := firstMap(overall)
	kills, isk := valueInt64(row["kills"]), valueFloat64(row["isk_destroyed"])
	heat := math.Min(1, math.Log10(1+float64(kills))/2.5)*0.5 + math.Min(1, math.Log10(1+isk/1e9)/3)*0.5
	output := SystemPulseOutput{
		System:    SystemRef{ID: system.ID, Name: system.Name, URL: entityURL(deps.BaseURL, EntitySystem, system.ID)},
		Window:    TimeWindow{Since: since.Format(time.RFC3339Nano), Until: until.Format(time.RFC3339Nano)},
		Totals:    SystemPulseTotals{Kills: kills, PVPKills: valueInt64(row["pvp_kills"]), SoloKills: valueInt64(row["solo_kills"]), AttackersTotal: valueInt64(row["attackers_total"]), ISKDestroyed: isk},
		HeatScore: math.Round(heat*100) / 100, TopAttackerCorps: corps, TopAttackerAlliances: alliances,
		TopVictimShips: make([]ShipLossActivity, 0, len(ships)),
	}
	if !historical {
		output.WindowHours = &hours
	}
	for _, ship := range ships {
		output.TopVictimShips = append(output.TopVictimShips, ShipLossActivity{
			TypeID: nullableInt64(ship["type_id"]), Name: nullableString(ship["name"]),
			Losses: valueInt64(ship["losses"]), ISKLost: valueFloat64(ship["isk_lost"]),
		})
	}
	return output, nil
}

func systemPulseOrganizations(ctx context.Context, deps Dependencies, systemID int64, since, until time.Time, limit int, alliance bool) ([]OrganizationActivity, error) {
	idColumn, table, tableID := "corporation_id", "corporations", "corporation_id"
	if alliance {
		idColumn, table, tableID = "alliance_id", "alliances", "alliance_id"
	}
	query := fmt.Sprintf(`
		SELECT organization.%s AS id, organization.name, organization.ticker,
		       COUNT(DISTINCT a.killmail_id)::bigint AS kills
		FROM killmail_attackers a JOIN killmails k ON k.killmail_id = a.killmail_id
		LEFT JOIN %s organization ON organization.%s = a.%s
		WHERE k.solar_system_id = $1 AND k.killmail_time >= $2 AND k.killmail_time <= $3
		  AND a.%s IS NOT NULL
		GROUP BY organization.%s, organization.name, organization.ticker
		ORDER BY kills DESC LIMIT $4`, tableID, table, tableID, idColumn, idColumn, tableID)
	rows, err := queryMaps(ctx, deps.DB, query, systemID, since, until, limit)
	if err != nil {
		return nil, err
	}
	output := make([]OrganizationActivity, 0, len(rows))
	for _, row := range rows {
		output = append(output, OrganizationActivity{ID: valueInt64(row["id"]), Name: nullableString(row["name"]), Ticker: nullableString(row["ticker"]), Kills: valueInt64(row["kills"])})
	}
	return output, nil
}

func expensiveLosses(ctx context.Context, deps Dependencies, input ExpensiveLossesInput) (ExpensiveLossesOutput, error) {
	days, limit := input.Days, input.Limit
	if days == 0 {
		days = 30
	}
	days = clamp(days, 1, 365)
	if limit == 0 {
		limit = 20
	}
	limit = clamp(limit, 1, 50)
	var opponentColumn string
	var opponentID int64
	if input.VS != nil && input.VS.String() != "" {
		opponent, err := resolveEntity(ctx, deps, *input.VS, nil)
		if err != nil {
			return ExpensiveLossesOutput{}, err
		}
		if opponent == nil {
			return ExpensiveLossesOutput{}, fmt.Errorf("could not resolve vs opponent")
		}
		opponentColumn = map[EntityType]string{EntityCharacter: "character_id", EntityCorporation: "corporation_id", EntityAlliance: "alliance_id"}[opponent.Type]
		if opponentColumn == "" {
			return ExpensiveLossesOutput{}, fmt.Errorf("vs must be character, corporation, or alliance")
		}
		opponentID = opponent.ID
	}
	query := `
		SELECT k.killmail_id, k.killmail_time, k.total_value, k.victim_character_id,
		       c.name AS victim_character_name, k.victim_corporation_id,
		       co.name AS victim_corporation_name, co.ticker AS victim_corporation_ticker,
		       k.victim_alliance_id, al.name AS victim_alliance_name, al.ticker AS victim_alliance_ticker,
		       k.victim_ship_type_id, t.name AS victim_ship_name, k.solar_system_id,
		       s.system_name AS solar_system_name, k.region_id, r.name AS region_name
		FROM killmails k
		LEFT JOIN characters c ON c.character_id = k.victim_character_id
		LEFT JOIN corporations co ON co.corporation_id = k.victim_corporation_id
		LEFT JOIN alliances al ON al.alliance_id = k.victim_alliance_id
		LEFT JOIN inv_types t ON t.type_id = k.victim_ship_type_id
		LEFT JOIN solar_systems s ON s.solar_system_id = k.solar_system_id
		LEFT JOIN regions r ON r.region_id = k.region_id
		WHERE k.killmail_time >= $1
		  AND ($2::bigint IS NULL OR k.region_id = $2)
		  AND ($3::bigint IS NULL OR k.solar_system_id = $3)
		  AND ($4::bigint IS NULL OR k.victim_ship_type_id = $4)
		  AND ($5::bigint IS NULL OR k.victim_alliance_id = $5)
		  AND ($6::bigint IS NULL OR k.victim_character_id = $6)
		  AND ($7::double precision IS NULL OR k.total_value >= $7)`
	args := []any{time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour), input.RegionID, input.SystemID, input.ShipTypeID, input.VictimAllianceID, input.VictimCharacterID, input.MinValue}
	if opponentColumn != "" {
		query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM killmail_attackers a WHERE a.killmail_id = k.killmail_id AND a.%s = $8)", opponentColumn)
		args = append(args, opponentID)
	}
	query += fmt.Sprintf(" ORDER BY k.total_value DESC NULLS LAST LIMIT $%d", len(args)+1)
	args = append(args, limit)
	rows, err := queryMaps(ctx, deps.DB, query, args...)
	if err != nil {
		return ExpensiveLossesOutput{}, err
	}
	output := ExpensiveLossesOutput{WindowDays: days, Count: len(rows), Kills: make([]ExpensiveLoss, 0, len(rows))}
	for _, row := range rows {
		id := valueInt64(row["killmail_id"])
		output.Kills = append(output.Kills, ExpensiveLoss{
			KillmailID: id, URL: killmailURL(deps.BaseURL, id), Time: nullableTime(row["killmail_time"]), TotalValue: valueFloat64(row["total_value"]),
			System:     LossSystem{ID: valueInt64(row["solar_system_id"]), Name: nullableString(row["solar_system_name"]), RegionID: nullableInt64(row["region_id"]), RegionName: nullableString(row["region_name"])},
			VictimShip: LossShip{TypeID: nullableInt64(row["victim_ship_type_id"]), Name: nullableString(row["victim_ship_name"])},
			Victim: KillmailVictim{
				CharacterID: nullableInt64(row["victim_character_id"]), CharacterName: nullableString(row["victim_character_name"]),
				CorporationID: nullableInt64(row["victim_corporation_id"]), CorporationName: nullableString(row["victim_corporation_name"]),
				CorporationTicker: nullableString(row["victim_corporation_ticker"]), AllianceID: nullableInt64(row["victim_alliance_id"]),
				AllianceName: nullableString(row["victim_alliance_name"]), AllianceTicker: nullableString(row["victim_alliance_ticker"]),
			},
		})
	}
	return output, nil
}

func capsuleerDossier(ctx context.Context, deps Dependencies, input DossierInput) (DossierOutput, error) {
	character, err := resolveCharacter(ctx, deps, input.Entity, input.Type)
	if err != nil {
		return DossierOutput{}, err
	}
	overview, err := entityOverview(ctx, deps, EntityOverviewInput{Entity: IntRef(character.ID), Type: entityTypePointer(EntityCharacter)})
	if err != nil {
		return DossierOutput{}, err
	}
	playstyleRows, err := queryMaps(ctx, deps.DB, `
		SELECT count(*) FILTER (WHERE k.attacker_count = 1) AS solo,
		       count(*) FILTER (WHERE k.attacker_count BETWEEN 2 AND 5) AS small_gang,
		       count(*) FILTER (WHERE k.attacker_count BETWEEN 6 AND 15) AS mid_gang,
		       count(*) FILTER (WHERE k.attacker_count BETWEEN 16 AND 50) AS fleet,
		       count(*) FILTER (WHERE k.attacker_count > 50) AS blob, count(*) AS total,
		       round(avg(k.attacker_count)::numeric, 1) AS avg_fleet_size
		FROM killmail_attackers a JOIN killmails k ON k.killmail_id = a.killmail_id
		WHERE a.character_id = $1 AND a.killmail_time > now() - interval '90 days'`, character.ID)
	if err != nil {
		return DossierOutput{}, err
	}
	archetypeRows, err := graphRead(ctx, deps, `
		MATCH (c:Character {id: $id})
		RETURN c.last_fc_seen AS fc, c.last_super_kill AS super, c.last_blops_seen AS blops,
		       c.last_capital_kill AS capital, c.last_logi_seen AS logi`, map[string]any{"id": character.ID})
	if err != nil {
		return DossierOutput{}, err
	}
	partners, err := graphRead(ctx, deps, `
		MATCH (c:Character {id: $id})-[r:FLEW_WITH]-(p:Character)
		RETURN p.id AS id, r.weight AS weight ORDER BY r.weight DESC LIMIT 5`, map[string]any{"id": character.ID})
	if err != nil {
		return DossierOutput{}, err
	}
	names, err := loadIntelNames(ctx, deps, partners, "id")
	if err != nil {
		return DossierOutput{}, err
	}
	ps, arch := firstMap(playstyleRows), firstMap(archetypeRows)
	total := valueInt64(ps["total"])
	divisor := total
	if divisor == 0 {
		divisor = 1
	}
	style := &DossierPlaystyle{
		SoloPct:      int(math.Round(float64(valueInt64(ps["solo"])) / float64(divisor) * 100)),
		SmallGangPct: int(math.Round(float64(valueInt64(ps["small_gang"])) / float64(divisor) * 100)),
		MidGangPct:   int(math.Round(float64(valueInt64(ps["mid_gang"])) / float64(divisor) * 100)),
		FleetPct:     int(math.Round(float64(valueInt64(ps["fleet"])) / float64(divisor) * 100)),
		BlobPct:      int(math.Round(float64(valueInt64(ps["blob"])) / float64(divisor) * 100)),
		AvgFleetSize: valueFloat64(ps["avg_fleet_size"]), TotalKills90D: total,
	}
	style.Dominant = dominantPlaystyle(style)
	lastSeen := &DossierArchetypeLastSeen{LastFCSeen: nullableString(arch["fc"]), LastSuperKill: nullableString(arch["super"]), LastBlopsSeen: nullableString(arch["blops"]), LastCapitalKill: nullableString(arch["capital"]), LastLogiSeen: nullableString(arch["logi"])}
	tags := recentArchetypeTags(lastSeen)
	output := DossierOutput{
		Entity: character.Public(deps.BaseURL), Lifetime: overview.Lifetime, ArchetypeTags: tags,
		ArchetypeLastSeen: lastSeen, Playstyle90D: style, TopShips: []DossierTopShip{},
		TopSystems: []DossierTopSystem{}, TopWingmates: []DossierWingmate{},
	}
	for _, item := range overview.TopShipsFlown {
		output.TopShips = append(output.TopShips, DossierTopShip{TypeID: item.ID, Name: item.Name, Kills: item.Kills})
	}
	for _, item := range overview.TopSystems {
		output.TopSystems = append(output.TopSystems, DossierTopSystem{SystemID: item.ID, Name: item.Name, Kills: item.Kills})
	}
	for _, row := range partners {
		id := valueInt64(row["id"])
		output.TopWingmates = append(output.TopWingmates, DossierWingmate{CharacterID: id, Name: names.characters[id], SharedKills: valueInt64(row["weight"]), URL: entityURL(deps.BaseURL, EntityCharacter, id)})
	}
	if input.Format == "summary" {
		output = DossierOutput{Entity: output.Entity, Summary: renderDossierSummary(output)}
	}
	return output, nil
}

func killmailStory(ctx context.Context, deps Dependencies, input KillmailInput) (KillmailStoryOutput, error) {
	killmail, err := loadKillmail(ctx, deps, input)
	if err != nil {
		return KillmailStoryOutput{}, err
	}
	security := killmail.System.Security
	securityLabel := "unknown"
	if security != nil {
		switch {
		case *security >= 0.5:
			securityLabel = "highsec"
		case *security > 0:
			securityLabel = "lowsec"
		default:
			securityLabel = "nullsec"
		}
	}
	victim := "an NPC"
	if killmail.Victim.CharacterName != nil {
		victim = *killmail.Victim.CharacterName
	}
	ship := "ship"
	if killmail.Victim.ShipName != nil {
		ship = *killmail.Victim.ShipName
	}
	system := fmt.Sprintf("system %d", killmail.System.ID)
	if killmail.System.Name != nil {
		system = *killmail.System.Name
	}
	killer := "an attacker"
	if killmail.IsNPC {
		killer = "an NPC gang"
	} else if killmail.FinalBlow != nil && killmail.FinalBlow.CharacterName != nil {
		killer = *killmail.FinalBlow.CharacterName
	}
	gang := ""
	if killmail.AttackerCount > 1 {
		gang = fmt.Sprintf(" alongside %d others", killmail.AttackerCount-1)
	}
	solo := ""
	if killmail.IsSolo {
		solo = "solo "
	}
	story := fmt.Sprintf("On %s, %s lost a %s (%s) in %s (%s). %s landed the %sfinal blow%s.",
		killmail.Time.UTC().Format("2006-01-02 15:04:05 UTC"), victim, ship,
		formatISK(killmail.TotalValue), system, securityLabel, killer, solo, gang)
	affiliation := killmail.Victim.AllianceName
	if affiliation == nil {
		affiliation = killmail.Victim.CorporationName
	}
	return KillmailStoryOutput{
		KillmailID: killmail.KillmailID, URL: killmail.URL, Story: story,
		Facts: KillmailStoryFacts{
			Time: killmail.Time, TotalValue: killmail.TotalValue, IsSolo: killmail.IsSolo,
			IsNPC: killmail.IsNPC, AttackerCount: killmail.AttackerCount, System: killmail.System,
			VictimShip: killmail.Victim.ShipName, Victim: killmail.Victim.CharacterName,
			VictimAffiliation: affiliation, FinalBlow: killmail.FinalBlow,
		},
	}, nil
}

func recentArchetypeTags(last *DossierArchetypeLastSeen) []string {
	cutoff := time.Now().UTC().Add(-90 * 24 * time.Hour)
	candidates := []struct {
		name  string
		value *string
	}{{"FC", last.LastFCSeen}, {"SUPER", last.LastSuperKill}, {"BLOPS", last.LastBlopsSeen}, {"CAPITAL", last.LastCapitalKill}, {"LOGI", last.LastLogiSeen}}
	tags := []string{}
	for _, candidate := range candidates {
		if candidate.value == nil {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, *candidate.value)
		if err == nil && parsed.After(cutoff) {
			tags = append(tags, candidate.name)
		}
	}
	return tags
}

func dominantPlaystyle(style *DossierPlaystyle) string {
	values := []struct {
		name  string
		value int
	}{{"solo", style.SoloPct}, {"small_gang", style.SmallGangPct}, {"mid_gang", style.MidGangPct}, {"fleet", style.FleetPct}, {"blob", style.BlobPct}}
	best := values[0]
	for _, candidate := range values[1:] {
		if candidate.value > best.value {
			best = candidate
		}
	}
	return best.name
}

func renderDossierSummary(output DossierOutput) string {
	if output.Lifetime == nil || output.Playstyle90D == nil {
		return output.Entity.Name
	}
	shipNames, systemNames, wingNames := []string{}, []string{}, []string{}
	for _, ship := range output.TopShips {
		if ship.Name != nil && len(shipNames) < 3 {
			shipNames = append(shipNames, *ship.Name)
		}
	}
	for _, system := range output.TopSystems {
		if system.Name != nil && len(systemNames) < 3 {
			systemNames = append(systemNames, *system.Name)
		}
	}
	for _, wingmate := range output.TopWingmates {
		if wingmate.Name != nil && len(wingNames) < 3 {
			wingNames = append(wingNames, *wingmate.Name)
		}
	}
	return strings.TrimSpace(fmt.Sprintf(
		"%s: %d kills / %d losses, %.1f%% ISK efficiency. Plays dominantly %s (%d kills over last 90d, avg fleet %.1f). Favored ships: %s. Favored systems: %s. Flies most with %s.",
		output.Entity.Name, output.Lifetime.Kills, output.Lifetime.Losses, output.Lifetime.ISKEfficiency,
		strings.ReplaceAll(output.Playstyle90D.Dominant, "_", " "), output.Playstyle90D.TotalKills90D,
		output.Playstyle90D.AvgFleetSize, strings.Join(shipNames, ", "), strings.Join(systemNames, ", "),
		strings.Join(wingNames, ", "),
	))
}
