package mcpserver

import (
	"context"
	"fmt"
	"strings"
)

type KillsWithInput struct {
	Entity       StringOrInt64  `json:"entity"`
	Type         *EntityType    `json:"type,omitempty" enum:"character"`
	Partner      StringOrInt64  `json:"partner"`
	EntityShip   *StringOrInt64 `json:"entity_ship,omitempty"`
	PartnerShip  *StringOrInt64 `json:"partner_ship,omitempty"`
	VictimShip   *StringOrInt64 `json:"victim_ship,omitempty"`
	VictimEntity *StringOrInt64 `json:"victim_entity,omitempty"`
	System       *StringOrInt64 `json:"system,omitempty"`
	Region       *StringOrInt64 `json:"region,omitempty"`
	From         *string        `json:"from,omitempty"`
	To           *string        `json:"to,omitempty"`
	GroupBy      string         `json:"group_by,omitempty" enum:"none,victim_ship,system,region,month,partner_ship,entity_ship" default:"none"`
	Limit        int            `json:"limit,omitempty" default:"10" minimum:"1" maximum:"50"`
}

type NamedShip struct {
	TypeID int64  `json:"type_id"`
	Name   string `json:"name"`
}

type KillsWithFilters struct {
	EntityShip   *NamedShip `json:"entity_ship"`
	PartnerShip  *NamedShip `json:"partner_ship"`
	VictimShip   *NamedShip `json:"victim_ship"`
	VictimEntity *Entity    `json:"victim_entity"`
	System       *Entity    `json:"system"`
	Region       *Entity    `json:"region"`
	From         *string    `json:"from"`
	To           *string    `json:"to"`
}

type KillsWithTotals struct {
	Kills        int64   `json:"kills"`
	ISKDestroyed float64 `json:"isk_destroyed"`
}

type KillsWithBreakdown struct {
	Month        *string `json:"month,omitempty"`
	TypeID       *int64  `json:"type_id,omitempty"`
	SystemID     *int64  `json:"system_id,omitempty"`
	RegionID     *int64  `json:"region_id,omitempty"`
	Name         *string `json:"name,omitempty"`
	Kills        int64   `json:"kills"`
	ISKDestroyed float64 `json:"isk_destroyed"`
}

type KillsWithOutput struct {
	Entity    Entity               `json:"entity"`
	Partner   Entity               `json:"partner"`
	Filters   KillsWithFilters     `json:"filters"`
	Totals    KillsWithTotals      `json:"totals"`
	GroupBy   string               `json:"group_by,omitempty"`
	Breakdown []KillsWithBreakdown `json:"breakdown,omitempty"`
}

func registerKillsWithTool(registry *Registry) error {
	return addTool(registry, ToolDefinition{
		Name: "kills_with", Title: "Count shared kills",
		Description: "Count killmails where two characters were both attackers, with optional ship, victim, location, time, and breakdown filters.",
	}, func(ctx context.Context, input KillsWithInput) (KillsWithOutput, error) {
		return killsWith(ctx, registry.deps, input)
	})
}

func killsWith(ctx context.Context, deps Dependencies, input KillsWithInput) (KillsWithOutput, error) {
	anchor, err := resolveCharacter(ctx, deps, input.Entity, input.Type)
	if err != nil {
		return KillsWithOutput{}, err
	}
	partner, err := resolveCharacter(ctx, deps, input.Partner, entityTypePointer(EntityCharacter))
	if err != nil {
		return KillsWithOutput{}, err
	}
	if anchor.ID == partner.ID {
		return KillsWithOutput{}, fmt.Errorf("anchor and partner must be different characters")
	}
	entityShip, err := optionalTypedEntity(ctx, deps, input.EntityShip, []EntityType{EntityShip}, "entity_ship")
	if err != nil {
		return KillsWithOutput{}, err
	}
	partnerShip, err := optionalTypedEntity(ctx, deps, input.PartnerShip, []EntityType{EntityShip}, "partner_ship")
	if err != nil {
		return KillsWithOutput{}, err
	}
	victimShip, err := optionalTypedEntity(ctx, deps, input.VictimShip, []EntityType{EntityShip}, "victim_ship")
	if err != nil {
		return KillsWithOutput{}, err
	}
	victimEntity, err := optionalTypedEntity(ctx, deps, input.VictimEntity, []EntityType{EntityCharacter, EntityCorporation, EntityAlliance}, "victim_entity")
	if err != nil {
		return KillsWithOutput{}, err
	}
	system, err := optionalTypedEntity(ctx, deps, input.System, []EntityType{EntitySystem}, "system")
	if err != nil {
		return KillsWithOutput{}, err
	}
	region, err := optionalTypedEntity(ctx, deps, input.Region, []EntityType{EntityRegion}, "region")
	if err != nil {
		return KillsWithOutput{}, err
	}
	arguments := []any{anchor.ID, partner.ID}
	attackerFilters, killmailFilters := []string{"a1.character_id = $1", "a2.character_id = $2", "a2.attacker_index != a1.attacker_index"}, []string{}
	add := func(target *[]string, expression string, value any) {
		arguments = append(arguments, value)
		*target = append(*target, fmt.Sprintf(expression, len(arguments)))
	}
	if entityShip != nil {
		add(&attackerFilters, "a1.ship_type_id = $%d", entityShip.ID)
	}
	if partnerShip != nil {
		add(&attackerFilters, "a2.ship_type_id = $%d", partnerShip.ID)
	}
	if input.From != nil {
		add(&attackerFilters, "a1.killmail_time >= $%d", *input.From)
	}
	if input.To != nil {
		add(&attackerFilters, "a1.killmail_time < $%d", *input.To)
	}
	if victimShip != nil {
		add(&killmailFilters, "killmail.victim_ship_type_id = $%d", victimShip.ID)
	}
	if victimEntity != nil {
		add(&killmailFilters, "killmail."+organizationVictimColumns[victimEntity.Type]+" = $%d", victimEntity.ID)
	}
	if system != nil {
		add(&killmailFilters, "killmail.solar_system_id = $%d", system.ID)
	}
	if region != nil {
		add(&killmailFilters, "killmail.region_id = $%d", region.ID)
	}
	cte := fmt.Sprintf(`
		WITH shared AS (
			SELECT DISTINCT ON (a1.killmail_id) a1.killmail_id,
			       a1.ship_type_id AS entity_ship_type_id, a2.ship_type_id AS partner_ship_type_id
			FROM killmail_attackers a1 JOIN killmail_attackers a2 ON a2.killmail_id = a1.killmail_id
			WHERE %s ORDER BY a1.killmail_id DESC
		)`, strings.Join(attackerFilters, " AND "))
	where := ""
	if len(killmailFilters) > 0 {
		where = " WHERE " + strings.Join(killmailFilters, " AND ")
	}
	totalsRows, err := queryMaps(ctx, deps.DB, cte+`
		SELECT count(*)::bigint AS kills, COALESCE(SUM(killmail.total_value), 0)::double precision AS isk_destroyed
		FROM shared JOIN killmails killmail ON killmail.killmail_id = shared.killmail_id`+where, arguments...)
	if err != nil {
		return KillsWithOutput{}, err
	}
	output := KillsWithOutput{
		Entity: anchor.Public(deps.BaseURL), Partner: partner.Public(deps.BaseURL),
		Filters: KillsWithFilters{
			EntityShip: publicNamedShip(entityShip), PartnerShip: publicNamedShip(partnerShip),
			VictimShip: publicNamedShip(victimShip), VictimEntity: publicEntity(victimEntity, deps.BaseURL),
			System: publicEntity(system, deps.BaseURL), Region: publicEntity(region, deps.BaseURL), From: input.From, To: input.To,
		},
		Totals: KillsWithTotals{Kills: valueInt64(firstMap(totalsRows)["kills"]), ISKDestroyed: valueFloat64(firstMap(totalsRows)["isk_destroyed"])},
	}
	groupBy := input.GroupBy
	if groupBy == "" {
		groupBy = "none"
	}
	if groupBy != "none" && output.Totals.Kills > 0 {
		limit := input.Limit
		if limit == 0 {
			limit = 10
		}
		output.GroupBy = groupBy
		output.Breakdown, err = killsWithBreakdown(ctx, deps, cte, where, arguments, groupBy, clamp(limit, 1, 50))
		if err != nil {
			return KillsWithOutput{}, err
		}
	}
	return output, nil
}

func killsWithBreakdown(ctx context.Context, deps Dependencies, cte, where string, arguments []any, groupBy string, limit int) ([]KillsWithBreakdown, error) {
	arguments = append(arguments, limit)
	if groupBy == "month" {
		rows, err := queryMaps(ctx, deps.DB, cte+fmt.Sprintf(`
			SELECT to_char(date_trunc('month', killmail.killmail_time), 'YYYY-MM') AS key,
			       count(*)::bigint AS kills, COALESCE(SUM(killmail.total_value), 0)::double precision AS isk
			FROM shared JOIN killmails killmail ON killmail.killmail_id = shared.killmail_id%s
			GROUP BY date_trunc('month', killmail.killmail_time)
			ORDER BY date_trunc('month', killmail.killmail_time) DESC LIMIT $%d`, where, len(arguments)), arguments...)
		if err != nil {
			return nil, err
		}
		output := make([]KillsWithBreakdown, 0, len(rows))
		for _, row := range rows {
			output = append(output, KillsWithBreakdown{Month: nullableString(row["key"]), Kills: valueInt64(row["kills"]), ISKDestroyed: valueFloat64(row["isk"])})
		}
		return output, nil
	}
	expression := map[string]string{
		"victim_ship": "killmail.victim_ship_type_id", "system": "killmail.solar_system_id",
		"region": "killmail.region_id", "partner_ship": "shared.partner_ship_type_id",
		"entity_ship": "shared.entity_ship_type_id",
	}[groupBy]
	if expression == "" {
		return nil, fmt.Errorf("invalid group_by")
	}
	rows, err := queryMaps(ctx, deps.DB, cte+fmt.Sprintf(`
		SELECT %s AS key, count(*)::bigint AS kills,
		       COALESCE(SUM(killmail.total_value), 0)::double precision AS isk
		FROM shared JOIN killmails killmail ON killmail.killmail_id = shared.killmail_id%s
		%s %s IS NOT NULL GROUP BY %s ORDER BY kills DESC LIMIT $%d`,
		expression, where, conjunction(where), expression, expression, len(arguments)), arguments...)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, valueInt64(row["key"]))
	}
	table, idColumn, nameColumn := "inv_types", "type_id", "name"
	if groupBy == "system" {
		table, idColumn, nameColumn = "solar_systems", "solar_system_id", "system_name"
	} else if groupBy == "region" {
		table, idColumn, nameColumn = "regions", "region_id", "name"
	}
	namesRows, err := queryMaps(ctx, deps.DB, fmt.Sprintf("SELECT %s AS id, %s AS name FROM %s WHERE %s = ANY($1)", idColumn, nameColumn, table, idColumn), ids)
	if err != nil {
		return nil, err
	}
	names := map[int64]*string{}
	for _, row := range namesRows {
		names[valueInt64(row["id"])] = nullableString(row["name"])
	}
	output := make([]KillsWithBreakdown, 0, len(rows))
	for _, row := range rows {
		id := valueInt64(row["key"])
		item := KillsWithBreakdown{Name: names[id], Kills: valueInt64(row["kills"]), ISKDestroyed: valueFloat64(row["isk"])}
		if groupBy == "system" {
			item.SystemID = &id
		} else if groupBy == "region" {
			item.RegionID = &id
		} else {
			item.TypeID = &id
		}
		output = append(output, item)
	}
	return output, nil
}

func optionalTypedEntity(ctx context.Context, deps Dependencies, reference *StringOrInt64, allowed []EntityType, label string) (*ResolvedEntity, error) {
	if reference == nil || reference.String() == "" {
		return nil, nil
	}
	var hint *EntityType
	if len(allowed) == 1 {
		hint = &allowed[0]
	}
	entity, err := resolveEntity(ctx, deps, *reference, hint)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("could not resolve %s", label)
	}
	for _, entityType := range allowed {
		if entity.Type == entityType {
			return entity, nil
		}
	}
	return nil, fmt.Errorf("%s resolved to %s", label, entity.Type)
}

func publicNamedShip(entity *ResolvedEntity) *NamedShip {
	if entity == nil {
		return nil
	}
	return &NamedShip{TypeID: entity.ID, Name: entity.Name}
}

func publicEntity(entity *ResolvedEntity, baseURL string) *Entity {
	if entity == nil {
		return nil
	}
	value := entity.Public(baseURL)
	return &value
}

func conjunction(where string) string {
	if where == "" {
		return "WHERE"
	}
	return "AND"
}
