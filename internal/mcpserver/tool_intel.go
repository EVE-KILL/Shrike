package mcpserver

import (
	"context"
	"fmt"
)

type CharacterIntelInput struct {
	Entity StringOrInt64 `json:"entity" doc:"Character name or id."`
	Type   *EntityType   `json:"type,omitempty" enum:"character" doc:"Must be character if specified."`
	Limit  int           `json:"limit,omitempty" default:"15" minimum:"1" maximum:"50"`
}

type IntelPartner struct {
	CharacterID     int64   `json:"character_id"`
	CharacterName   *string `json:"character_name"`
	CorporationID   *int64  `json:"corporation_id"`
	CorporationName *string `json:"corporation_name"`
	AllianceID      *int64  `json:"alliance_id"`
	AllianceName    *string `json:"alliance_name"`
	SharedKills     int64   `json:"shared_kills"`
	FirstSeen       *string `json:"first_seen"`
	LastSeen        *string `json:"last_seen"`
	URL             string  `json:"url"`
}

type FliesWithOutput struct {
	Entity     Entity         `json:"entity"`
	WindowDays int            `json:"window_days"`
	Count      int            `json:"count"`
	Partners   []IntelPartner `json:"partners"`
}

type IntelSystem struct {
	SystemID   int64    `json:"system_id"`
	SystemName *string  `json:"system_name"`
	RegionID   *int64   `json:"region_id"`
	RegionName *string  `json:"region_name"`
	Security   *float64 `json:"security"`
	Kills      int64    `json:"kills"`
	LastSeen   *string  `json:"last_seen"`
	URL        string   `json:"url"`
}

type HuntsInOutput struct {
	Entity     Entity        `json:"entity"`
	WindowDays int           `json:"window_days"`
	Count      int           `json:"count"`
	Systems    []IntelSystem `json:"systems"`
}

type IntelCharacter struct {
	CharacterID     int64   `json:"character_id"`
	CharacterName   *string `json:"character_name"`
	CorporationID   *int64  `json:"corporation_id"`
	CorporationName *string `json:"corporation_name"`
	AllianceID      *int64  `json:"alliance_id"`
	AllianceName    *string `json:"alliance_name"`
	Count           int64   `json:"count"`
	ISKDestroyed    float64 `json:"isk_destroyed"`
	FinalBlows      int64   `json:"final_blows"`
	LastSeen        *string `json:"last_seen"`
	URL             string  `json:"url"`
}

type CharacterIntelOutput struct {
	Entity     Entity           `json:"entity"`
	WindowDays int              `json:"window_days"`
	Count      int              `json:"count"`
	Characters []IntelCharacter `json:"characters"`
}

type CompareInput struct {
	A StringOrInt64 `json:"a" doc:"First character name or id."`
	B StringOrInt64 `json:"b" doc:"Second character name or id."`
}

type HeadToHead struct {
	Count        int64    `json:"count"`
	ISKDestroyed *float64 `json:"isk_destroyed,omitempty"`
	FinalBlows   *int64   `json:"final_blows,omitempty"`
	LastSeen     *string  `json:"last_seen,omitempty"`
}

type SharedSystem struct {
	SystemID   int64   `json:"system_id"`
	SystemName *string `json:"system_name"`
	RegionName *string `json:"region_name"`
	AKillsSeen int64   `json:"a_kills_seen"`
	BKillsSeen int64   `json:"b_kills_seen"`
}

type CompareOutput struct {
	A               Entity         `json:"a"`
	B               Entity         `json:"b"`
	WindowDays      int            `json:"window_days"`
	AKilledB        HeadToHead     `json:"a_killed_b"`
	BKilledA        HeadToHead     `json:"b_killed_a"`
	SharedWingmates int64          `json:"shared_wingmates"`
	SharedSystems   []SharedSystem `json:"shared_systems"`
}

func registerIntelTools(registry *Registry) error {
	if err := addTool(registry, ToolDefinition{
		Name: "flies_with", Title: "Find a character's frequent wingmates",
		Description: "Frequent wingmates of a character — who they most often appear alongside on killmails in the last 90 days. " +
			"Sourced from conservative same-side FLEW_WITH relationships.",
	}, func(ctx context.Context, input CharacterIntelInput) (FliesWithOutput, error) {
		return fliesWith(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name: "hunts_in", Title: "Find a character's hunting grounds",
		Description: "Preferred hunting grounds of a character — systems where they most often appear on killmails in the last 90 days. " +
			"Weight is distinct killmails observed in that system.",
	}, func(ctx context.Context, input CharacterIntelInput) (HuntsInOutput, error) {
		return huntsIn(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name: "hunted_by", Title: "Find who hunts a character",
		Description: "Characters who most often kill this character in the last 90 days. " +
			"Weight is the death count by that attacker.",
	}, func(ctx context.Context, input CharacterIntelInput) (CharacterIntelOutput, error) {
		return huntedBy(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	if err := addTool(registry, ToolDefinition{
		Name: "preys_on", Title: "Find who a character preys on",
		Description: "Characters who this character most often kills in the last 90 days. " +
			"Weight is the kill count against that victim.",
	}, func(ctx context.Context, input CharacterIntelInput) (CharacterIntelOutput, error) {
		return preysOn(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	return addTool(registry, ToolDefinition{
		Name: "compare", Title: "Compare two characters",
		Description: "Head-to-head between two characters in the last 90 days. Returns direct kills each way, " +
			"shared wingmates count, and mutual hunting grounds.",
	}, func(ctx context.Context, input CompareInput) (CompareOutput, error) {
		return compareCharacters(ctx, registry.deps, input)
	})
}

func resolveCharacter(ctx context.Context, deps Dependencies, input StringOrInt64, hint *EntityType) (*ResolvedEntity, error) {
	resolved, err := resolveEntity(ctx, deps, input, hint)
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, fmt.Errorf("no entity found for %q", input.String())
	}
	if resolved.Type != EntityCharacter {
		return nil, fmt.Errorf("intel graph is character-only; got %s", resolved.Type)
	}
	return resolved, nil
}

func fliesWith(ctx context.Context, deps Dependencies, input CharacterIntelInput) (FliesWithOutput, error) {
	entity, err := resolveCharacter(ctx, deps, input.Entity, input.Type)
	if err != nil {
		return FliesWithOutput{}, err
	}
	limit := input.Limit
	if limit == 0 {
		limit = 15
	}
	limit = clamp(limit, 1, 50)
	rows, err := graphRead(ctx, deps, `
		MATCH (c:Character {id: $id})-[r:FLEW_WITH]-(p:Character)
		RETURN p.id AS id, p.corporation_id AS corp_id, p.alliance_id AS alliance_id,
		       r.weight AS weight, r.first_seen AS first_seen, r.last_seen AS last_seen
		ORDER BY r.weight DESC LIMIT $lim`, map[string]any{"id": entity.ID, "lim": limit})
	if err != nil {
		return FliesWithOutput{}, err
	}
	output := FliesWithOutput{
		Entity: entity.Public(deps.BaseURL), WindowDays: 90,
		Count: len(rows), Partners: make([]IntelPartner, 0, len(rows)),
	}
	names, err := loadIntelNames(ctx, deps, rows, "id")
	if err != nil {
		return FliesWithOutput{}, err
	}
	for _, row := range rows {
		id := valueInt64(row["id"])
		corpID, allianceID := nullableInt64(row["corp_id"]), nullableInt64(row["alliance_id"])
		output.Partners = append(output.Partners, IntelPartner{
			CharacterID: id, CharacterName: names.characters[id],
			CorporationID: corpID, CorporationName: names.corporations[derefInt64(corpID)],
			AllianceID: allianceID, AllianceName: names.alliances[derefInt64(allianceID)],
			SharedKills: valueInt64(row["weight"]), FirstSeen: nullableString(row["first_seen"]),
			LastSeen: nullableString(row["last_seen"]), URL: entityURL(deps.BaseURL, EntityCharacter, id),
		})
	}
	return output, nil
}

func huntsIn(ctx context.Context, deps Dependencies, input CharacterIntelInput) (HuntsInOutput, error) {
	entity, err := resolveCharacter(ctx, deps, input.Entity, input.Type)
	if err != nil {
		return HuntsInOutput{}, err
	}
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}
	limit = clamp(limit, 1, 30)
	rows, err := graphRead(ctx, deps, `
		MATCH (c:Character {id: $id})-[r:OPERATED_IN]->(s:SolarSystem)
		RETURN s.id AS system_id, r.weight AS weight, r.last_seen AS last_seen
		ORDER BY r.weight DESC LIMIT $lim`, map[string]any{"id": entity.ID, "lim": limit})
	if err != nil {
		return HuntsInOutput{}, err
	}
	output := HuntsInOutput{
		Entity: entity.Public(deps.BaseURL), WindowDays: 90,
		Count: len(rows), Systems: make([]IntelSystem, 0, len(rows)),
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, valueInt64(row["system_id"]))
	}
	meta, err := loadSystemMeta(ctx, deps, ids)
	if err != nil {
		return HuntsInOutput{}, err
	}
	for _, row := range rows {
		id := valueInt64(row["system_id"])
		system := meta[id]
		output.Systems = append(output.Systems, IntelSystem{
			SystemID: id, SystemName: system.name, RegionID: system.regionID,
			RegionName: system.regionName, Security: system.security,
			Kills: valueInt64(row["weight"]), LastSeen: nullableString(row["last_seen"]),
			URL: entityURL(deps.BaseURL, EntitySystem, id),
		})
	}
	return output, nil
}

func huntedBy(ctx context.Context, deps Dependencies, input CharacterIntelInput) (CharacterIntelOutput, error) {
	return characterKills(ctx, deps, input, false)
}

func preysOn(ctx context.Context, deps Dependencies, input CharacterIntelInput) (CharacterIntelOutput, error) {
	return characterKills(ctx, deps, input, true)
}

func characterKills(ctx context.Context, deps Dependencies, input CharacterIntelInput, outbound bool) (CharacterIntelOutput, error) {
	entity, err := resolveCharacter(ctx, deps, input.Entity, input.Type)
	if err != nil {
		return CharacterIntelOutput{}, err
	}
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}
	limit = clamp(limit, 1, 30)
	query := `
		MATCH (a:Character)-[r:KILLED]->(v:Character {id: $id})
		RETURN a.id AS character_id, a.corporation_id AS corp_id, a.alliance_id AS alliance_id,
		       r.weight AS weight, r.isk_destroyed AS isk_destroyed, r.final_blows AS final_blows,
		       r.last_seen AS last_seen
		ORDER BY r.weight DESC LIMIT $lim`
	if outbound {
		query = `
			MATCH (a:Character {id: $id})-[r:KILLED]->(v:Character)
			RETURN v.id AS character_id, v.corporation_id AS corp_id, v.alliance_id AS alliance_id,
			       r.weight AS weight, r.isk_destroyed AS isk_destroyed, r.final_blows AS final_blows,
			       r.last_seen AS last_seen
			ORDER BY r.weight DESC LIMIT $lim`
	}
	rows, err := graphRead(ctx, deps, query, map[string]any{"id": entity.ID, "lim": limit})
	if err != nil {
		return CharacterIntelOutput{}, err
	}
	names, err := loadIntelNames(ctx, deps, rows, "character_id")
	if err != nil {
		return CharacterIntelOutput{}, err
	}
	output := CharacterIntelOutput{
		Entity: entity.Public(deps.BaseURL), WindowDays: 90,
		Count: len(rows), Characters: make([]IntelCharacter, 0, len(rows)),
	}
	for _, row := range rows {
		id := valueInt64(row["character_id"])
		corpID, allianceID := nullableInt64(row["corp_id"]), nullableInt64(row["alliance_id"])
		output.Characters = append(output.Characters, IntelCharacter{
			CharacterID: id, CharacterName: names.characters[id],
			CorporationID: corpID, CorporationName: names.corporations[derefInt64(corpID)],
			AllianceID: allianceID, AllianceName: names.alliances[derefInt64(allianceID)],
			Count: valueInt64(row["weight"]), ISKDestroyed: valueFloat64(row["isk_destroyed"]),
			FinalBlows: valueInt64(row["final_blows"]), LastSeen: nullableString(row["last_seen"]),
			URL: entityURL(deps.BaseURL, EntityCharacter, id),
		})
	}
	return output, nil
}

func compareCharacters(ctx context.Context, deps Dependencies, input CompareInput) (CompareOutput, error) {
	a, err := resolveCharacter(ctx, deps, input.A, new(EntityCharacter))
	if err != nil {
		return CompareOutput{}, fmt.Errorf("could not resolve a: %w", err)
	}
	b, err := resolveCharacter(ctx, deps, input.B, new(EntityCharacter))
	if err != nil {
		return CompareOutput{}, fmt.Errorf("could not resolve b: %w", err)
	}
	params := map[string]any{"a": a.ID, "b": b.ID}
	aKilledB, err := graphRead(ctx, deps, `
		MATCH (a:Character {id: $a})-[r:KILLED]->(b:Character {id: $b})
		RETURN r.weight AS weight, r.isk_destroyed AS isk, r.final_blows AS fb, r.last_seen AS last_seen`, params)
	if err != nil {
		return CompareOutput{}, err
	}
	bKilledA, err := graphRead(ctx, deps, `
		MATCH (b:Character {id: $b})-[r:KILLED]->(a:Character {id: $a})
		RETURN r.weight AS weight, r.isk_destroyed AS isk, r.final_blows AS fb, r.last_seen AS last_seen`, params)
	if err != nil {
		return CompareOutput{}, err
	}
	wingmates, err := graphRead(ctx, deps, `
		MATCH (a:Character {id: $a})-[:FLEW_WITH]-(p:Character)-[:FLEW_WITH]-(b:Character {id: $b})
		WHERE p.id <> $a AND p.id <> $b RETURN count(DISTINCT p) AS cnt`, params)
	if err != nil {
		return CompareOutput{}, err
	}
	systems, err := graphRead(ctx, deps, `
		MATCH (a:Character {id: $a})-[ra:OPERATED_IN]->(s:SolarSystem)<-[rb:OPERATED_IN]-(b:Character {id: $b})
		RETURN s.id AS sys_id, ra.weight AS a_weight, rb.weight AS b_weight
		ORDER BY (ra.weight + rb.weight) DESC LIMIT 10`, params)
	if err != nil {
		return CompareOutput{}, err
	}
	systemIDs := make([]int64, 0, len(systems))
	for _, row := range systems {
		systemIDs = append(systemIDs, valueInt64(row["sys_id"]))
	}
	meta, err := loadSystemMeta(ctx, deps, systemIDs)
	if err != nil {
		return CompareOutput{}, err
	}
	output := CompareOutput{
		A: a.Public(deps.BaseURL), B: b.Public(deps.BaseURL), WindowDays: 90,
		AKilledB: headToHead(firstMap(aKilledB)), BKilledA: headToHead(firstMap(bKilledA)),
		SharedSystems: make([]SharedSystem, 0, len(systems)),
	}
	if row := firstMap(wingmates); row != nil {
		output.SharedWingmates = valueInt64(row["cnt"])
	}
	for _, row := range systems {
		id := valueInt64(row["sys_id"])
		output.SharedSystems = append(output.SharedSystems, SharedSystem{
			SystemID: id, SystemName: meta[id].name, RegionName: meta[id].regionName,
			AKillsSeen: valueInt64(row["a_weight"]), BKillsSeen: valueInt64(row["b_weight"]),
		})
	}
	return output, nil
}

func graphRead(ctx context.Context, deps Dependencies, query string, params map[string]any) ([]map[string]any, error) {
	if deps.Graph == nil {
		return nil, fmt.Errorf("memgraph is not configured")
	}
	return deps.Graph.Read(ctx, query, params)
}

func headToHead(row map[string]any) HeadToHead {
	if row == nil {
		return HeadToHead{}
	}
	isk, blows := valueFloat64(row["isk"]), valueInt64(row["fb"])
	return HeadToHead{
		Count: valueInt64(row["weight"]), ISKDestroyed: &isk, FinalBlows: &blows,
		LastSeen: nullableString(row["last_seen"]),
	}
}

type intelNames struct {
	characters, corporations, alliances map[int64]*string
}

func loadIntelNames(ctx context.Context, deps Dependencies, rows []map[string]any, characterKey string) (intelNames, error) {
	charIDs, corpIDs, allianceIDs := make([]int64, 0, len(rows)), make([]int64, 0, len(rows)), make([]int64, 0, len(rows))
	for _, row := range rows {
		charIDs = append(charIDs, valueInt64(row[characterKey]))
		if id := nullableInt64(row["corp_id"]); id != nil {
			corpIDs = append(corpIDs, *id)
		}
		if id := nullableInt64(row["alliance_id"]); id != nil {
			allianceIDs = append(allianceIDs, *id)
		}
	}
	characters, err := loadNameMap(ctx, deps, "characters", "character_id", "name", charIDs)
	if err != nil {
		return intelNames{}, err
	}
	corporations, err := loadNameMap(ctx, deps, "corporations", "corporation_id", "name", corpIDs)
	if err != nil {
		return intelNames{}, err
	}
	alliances, err := loadNameMap(ctx, deps, "alliances", "alliance_id", "name", allianceIDs)
	return intelNames{characters, corporations, alliances}, err
}

func loadNameMap(ctx context.Context, deps Dependencies, table, idColumn, nameColumn string, ids []int64) (map[int64]*string, error) {
	output := map[int64]*string{}
	if len(ids) == 0 {
		return output, nil
	}
	allowed := map[string]bool{
		"characters.character_id.name": true, "corporations.corporation_id.name": true,
		"alliances.alliance_id.name": true,
	}
	if !allowed[table+"."+idColumn+"."+nameColumn] {
		return nil, fmt.Errorf("unsupported name lookup")
	}
	rows, err := queryMaps(ctx, deps.DB,
		fmt.Sprintf("SELECT %s AS id, %s AS name FROM %s WHERE %s = ANY($1)", idColumn, nameColumn, table, idColumn),
		ids)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		output[valueInt64(row["id"])] = nullableString(row["name"])
	}
	return output, nil
}

type systemMeta struct {
	name, regionName *string
	security         *float64
	regionID         *int64
}

func loadSystemMeta(ctx context.Context, deps Dependencies, ids []int64) (map[int64]systemMeta, error) {
	output := map[int64]systemMeta{}
	if len(ids) == 0 {
		return output, nil
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT s.solar_system_id AS id, s.system_name AS name, s.security,
		       s.region_id, r.name AS region_name
		FROM solar_systems s
		LEFT JOIN regions r ON r.region_id = s.region_id
		WHERE s.solar_system_id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		output[valueInt64(row["id"])] = systemMeta{
			name: nullableString(row["name"]), security: nullableFloat64(row["security"]),
			regionID: nullableInt64(row["region_id"]), regionName: nullableString(row["region_name"]),
		}
	}
	return output, nil
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
