package mcpserver

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type EntityKillsInput struct {
	Entity StringOrInt64 `json:"entity" jsonschema:"Entity name or numeric identifier."`
	Type   *EntityType   `json:"type,omitempty" enum:"character,corporation,alliance,ship,system,region,constellation,faction" jsonschema:"Entity type, recommended for identifiers and ambiguous names."`
	Role   string        `json:"role,omitempty" enum:"kills,losses,all" default:"all" jsonschema:"Return kills made, losses suffered, or both."`
	Limit  int           `json:"limit,omitempty" minimum:"1" maximum:"50" default:"20" jsonschema:"Maximum number of killmails."`
	Before *int64        `json:"before,omitempty" jsonschema:"Return killmails older than this identifier."`
	From   *string       `json:"from,omitempty" jsonschema:"Inclusive ISO date or timestamp lower bound."`
	To     *string       `json:"to,omitempty" jsonschema:"Exclusive ISO date or timestamp upper bound."`
}

type KillmailSystem struct {
	ID         int64    `json:"id"`
	Name       *string  `json:"name"`
	Security   *float64 `json:"security"`
	RegionID   *int64   `json:"region_id"`
	RegionName *string  `json:"region_name"`
}

type KillmailParticipant struct {
	CharacterID       *int64  `json:"character_id"`
	CharacterName     *string `json:"character_name"`
	CorporationID     *int64  `json:"corporation_id"`
	CorporationName   *string `json:"corporation_name"`
	CorporationTicker *string `json:"corporation_ticker"`
	AllianceID        *int64  `json:"alliance_id"`
	AllianceName      *string `json:"alliance_name"`
	AllianceTicker    *string `json:"alliance_ticker"`
	ShipTypeID        *int64  `json:"ship_type_id"`
	ShipName          *string `json:"ship_name"`
}

type KillmailSummary struct {
	KillmailID    int64                `json:"killmail_id"`
	URL           string               `json:"url"`
	Time          *time.Time           `json:"time,omitempty"`
	TotalValue    *float64             `json:"total_value,omitempty"`
	AttackerCount *int64               `json:"attacker_count,omitempty"`
	IsNPC         *bool                `json:"is_npc,omitempty"`
	IsSolo        *bool                `json:"is_solo,omitempty"`
	System        *KillmailSystem      `json:"system,omitempty"`
	Victim        *KillmailParticipant `json:"victim,omitempty"`
	FinalBlow     *KillmailParticipant `json:"final_blow"`
}

type EntityKillsOutput struct {
	Entity     Entity            `json:"entity"`
	Count      *int              `json:"count,omitempty"`
	NextBefore *int64            `json:"next_before"`
	Kills      []KillmailSummary `json:"kills"`
}

func registerEntityKillsTool(registry *Registry) error {
	return addTool(registry, ToolDefinition{
		Name:  "entity_kills",
		Title: "List entity killmails",
		Description: "Return recent killmails involving an entity, newest first. " +
			"Filter between kills, losses, or both and paginate with before.",
	}, func(ctx context.Context, input EntityKillsInput) (EntityKillsOutput, error) {
		return entityKills(ctx, registry.deps, input)
	})
}

func entityKills(
	ctx context.Context,
	deps Dependencies,
	input EntityKillsInput,
) (EntityKillsOutput, error) {
	resolved, err := resolveEntity(ctx, deps, input.Entity, input.Type)
	if err != nil {
		return EntityKillsOutput{}, fmt.Errorf("resolve entity: %w", err)
	}
	if resolved == nil {
		return EntityKillsOutput{}, fmt.Errorf("no entity found for %q", input.Entity.String())
	}
	role := input.Role
	if role == "" {
		role = "all"
	}
	if role != "kills" && role != "losses" && role != "all" {
		return EntityKillsOutput{}, fmt.Errorf("unsupported role %q", role)
	}
	limit := input.Limit
	if limit == 0 {
		limit = 20
	}
	limit = clamp(limit, 1, 50)
	legs, err := entityKillLegs(resolved.Type, resolved.ID, role)
	if err != nil {
		return EntityKillsOutput{}, err
	}

	seen := map[int64]bool{}
	ids := make([]int64, 0, limit*len(legs))
	for _, leg := range legs {
		query := leg.query
		args := []any{resolved.ID}
		next := 2
		if input.Before != nil {
			query += fmt.Sprintf(" AND k.killmail_id < $%d", next)
			args = append(args, *input.Before)
			next++
		}
		if input.From != nil {
			query += fmt.Sprintf(" AND k.killmail_time >= $%d", next)
			args = append(args, *input.From)
			next++
		}
		if input.To != nil {
			query += fmt.Sprintf(" AND k.killmail_time < $%d", next)
			args = append(args, *input.To)
			next++
		}
		query += fmt.Sprintf(" ORDER BY k.killmail_id DESC LIMIT $%d", next)
		args = append(args, limit)
		rows, queryErr := queryMaps(ctx, deps.DB, query, args...)
		if queryErr != nil {
			return EntityKillsOutput{}, fmt.Errorf("load entity killmails: %w", queryErr)
		}
		for _, row := range rows {
			id := valueInt64(row["killmail_id"])
			if id > 0 && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	if len(ids) > limit {
		ids = ids[:limit]
	}
	output := EntityKillsOutput{
		Entity: resolved.Public(deps.BaseURL),
		Kills:  []KillmailSummary{},
	}
	if len(ids) == 0 {
		return output, nil
	}
	output.Kills, err = hydrateKillmailSummaries(ctx, deps, ids)
	if err != nil {
		return EntityKillsOutput{}, err
	}
	count := len(output.Kills)
	output.Count = &count
	if len(ids) == limit {
		output.NextBefore = &ids[len(ids)-1]
	}
	return output, nil
}

type entityKillLeg struct {
	query string
}

func entityKillLegs(
	entityType EntityType,
	_ int64,
	role string,
) ([]entityKillLeg, error) {
	victimColumns := map[EntityType]string{
		EntityCharacter:   "victim_character_id",
		EntityCorporation: "victim_corporation_id",
		EntityAlliance:    "victim_alliance_id",
		EntityFaction:     "victim_faction_id",
	}
	attackerColumns := map[EntityType]string{
		EntityCharacter:   "character_id",
		EntityCorporation: "corporation_id",
		EntityAlliance:    "alliance_id",
		EntityFaction:     "faction_id",
	}
	var legs []entityKillLeg
	if victimColumn := victimColumns[entityType]; victimColumn != "" {
		if role != "kills" {
			legs = append(legs, entityKillLeg{query: fmt.Sprintf(
				"SELECT k.killmail_id FROM killmails k WHERE k.%s = $1",
				victimColumn,
			)})
		}
		if role != "losses" {
			legs = append(legs, entityKillLeg{query: fmt.Sprintf(
				`SELECT k.killmail_id
				   FROM killmail_attackers attacker
				   JOIN killmails k ON k.killmail_id = attacker.killmail_id
				  WHERE attacker.%s = $1`,
				attackerColumns[entityType],
			)})
		}
	} else {
		column := map[EntityType]string{
			EntityShip:          "victim_ship_type_id",
			EntitySystem:        "solar_system_id",
			EntityConstellation: "constellation_id",
			EntityRegion:        "region_id",
		}[entityType]
		if column != "" {
			legs = append(legs, entityKillLeg{query: fmt.Sprintf(
				"SELECT k.killmail_id FROM killmails k WHERE k.%s = $1",
				column,
			)})
		}
	}
	if len(legs) == 0 {
		return nil, fmt.Errorf("no queryable leg for %s in role %s", entityType, role)
	}
	return legs, nil
}

func hydrateKillmailSummaries(
	ctx context.Context,
	deps Dependencies,
	ids []int64,
) ([]KillmailSummary, error) {
	killmails, err := queryMaps(ctx, deps.DB, `
		SELECT
			k.killmail_id, k.killmail_time, k.total_value, k.attacker_count,
			k.is_npc, k.is_solo,
			k.victim_character_id, k.victim_corporation_id, k.victim_alliance_id,
			k.victim_ship_type_id, k.solar_system_id, k.region_id,
			character.name AS victim_character_name,
			corporation.name AS victim_corporation_name,
			corporation.ticker AS victim_corporation_ticker,
			alliance.name AS victim_alliance_name,
			alliance.ticker AS victim_alliance_ticker,
			ship.name AS victim_ship_name,
			system.system_name AS solar_system_name,
			system.security AS solar_system_security,
			region.name AS region_name
		FROM killmails k
		LEFT JOIN characters character
		       ON character.character_id = k.victim_character_id
		LEFT JOIN corporations corporation
		       ON corporation.corporation_id = k.victim_corporation_id
		LEFT JOIN alliances alliance ON alliance.alliance_id = k.victim_alliance_id
		LEFT JOIN inv_types ship ON ship.type_id = k.victim_ship_type_id
		LEFT JOIN solar_systems system ON system.solar_system_id = k.solar_system_id
		LEFT JOIN regions region ON region.region_id = k.region_id
		WHERE k.killmail_id = ANY($1::bigint[])`, ids)
	if err != nil {
		return nil, fmt.Errorf("hydrate killmails: %w", err)
	}
	finalBlows, err := queryMaps(ctx, deps.DB, `
		SELECT attacker.killmail_id, attacker.character_id,
		       attacker.corporation_id, attacker.alliance_id, attacker.ship_type_id,
		       character.name AS character_name,
		       corporation.name AS corporation_name,
		       corporation.ticker AS corporation_ticker,
		       alliance.name AS alliance_name,
		       alliance.ticker AS alliance_ticker,
		       ship.name AS ship_name
		FROM killmail_attackers attacker
		LEFT JOIN characters character ON character.character_id = attacker.character_id
		LEFT JOIN corporations corporation
		       ON corporation.corporation_id = attacker.corporation_id
		LEFT JOIN alliances alliance ON alliance.alliance_id = attacker.alliance_id
		LEFT JOIN inv_types ship ON ship.type_id = attacker.ship_type_id
		WHERE attacker.killmail_id = ANY($1::bigint[])
		  AND attacker.final_blow IS TRUE`, ids)
	if err != nil {
		return nil, fmt.Errorf("hydrate final blows: %w", err)
	}
	killmailByID := make(map[int64]map[string]any, len(killmails))
	for _, row := range killmails {
		killmailByID[valueInt64(row["killmail_id"])] = row
	}
	finalByID := make(map[int64]map[string]any, len(finalBlows))
	for _, row := range finalBlows {
		finalByID[valueInt64(row["killmail_id"])] = row
	}
	result := make([]KillmailSummary, 0, len(ids))
	for _, id := range ids {
		row := killmailByID[id]
		summary := KillmailSummary{
			KillmailID: id,
			URL:        killmailURL(deps.BaseURL, id),
		}
		if row == nil {
			result = append(result, summary)
			continue
		}
		summary.Time = nullableTime(row["killmail_time"])
		summary.TotalValue = nullableFloat64(row["total_value"])
		summary.AttackerCount = nullableInt64(row["attacker_count"])
		isNPC, isSolo := valueBool(row["is_npc"]), valueBool(row["is_solo"])
		summary.IsNPC, summary.IsSolo = &isNPC, &isSolo
		summary.System = &KillmailSystem{
			ID:         valueInt64(row["solar_system_id"]),
			Name:       nullableString(row["solar_system_name"]),
			Security:   nullableFloat64(row["solar_system_security"]),
			RegionID:   nullableInt64(row["region_id"]),
			RegionName: nullableString(row["region_name"]),
		}
		summary.Victim = participantFromMap(row, "victim_")
		if final := finalByID[id]; final != nil {
			summary.FinalBlow = participantFromMap(final, "")
		}
		result = append(result, summary)
	}
	return result, nil
}

func participantFromMap(row map[string]any, prefix string) *KillmailParticipant {
	return &KillmailParticipant{
		CharacterID:       nullableInt64(row[prefix+"character_id"]),
		CharacterName:     nullableString(row[prefix+"character_name"]),
		CorporationID:     nullableInt64(row[prefix+"corporation_id"]),
		CorporationName:   nullableString(row[prefix+"corporation_name"]),
		CorporationTicker: nullableString(row[prefix+"corporation_ticker"]),
		AllianceID:        nullableInt64(row[prefix+"alliance_id"]),
		AllianceName:      nullableString(row[prefix+"alliance_name"]),
		AllianceTicker:    nullableString(row[prefix+"alliance_ticker"]),
		ShipTypeID:        nullableInt64(row[prefix+"ship_type_id"]),
		ShipName:          nullableString(row[prefix+"ship_name"]),
	}
}

func stringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
