package mcpserver

import (
	"context"
	"fmt"
	"strings"
)

type ShipsUsedInput struct {
	Entity  StringOrInt64  `json:"entity"`
	Type    *EntityType    `json:"type,omitempty" enum:"character,corporation,alliance"`
	Ship    *StringOrInt64 `json:"ship,omitempty"`
	Role    string         `json:"role,omitempty" enum:"kills,losses,all" default:"all"`
	From    *string        `json:"from,omitempty"`
	To      *string        `json:"to,omitempty"`
	System  *StringOrInt64 `json:"system,omitempty"`
	Region  *StringOrInt64 `json:"region,omitempty"`
	GroupBy string         `json:"group_by,omitempty" enum:"none,ship,victim_ship,system,region,month" default:"ship"`
	Limit   int            `json:"limit,omitempty" default:"10" minimum:"1" maximum:"50"`
}

type ShipsUsedFilters struct {
	Ship   *NamedShip `json:"ship"`
	Role   string     `json:"role"`
	From   *string    `json:"from"`
	To     *string    `json:"to"`
	System *Entity    `json:"system"`
	Region *Entity    `json:"region"`
}

type ShipsUsedTotals struct {
	Kills        int64   `json:"kills"`
	Losses       int64   `json:"losses"`
	ISKDestroyed float64 `json:"isk_destroyed"`
	ISKLost      float64 `json:"isk_lost"`
}

type ShipsUsedBreakdown struct {
	Month        *string `json:"month,omitempty"`
	TypeID       *int64  `json:"type_id,omitempty"`
	SystemID     *int64  `json:"system_id,omitempty"`
	RegionID     *int64  `json:"region_id,omitempty"`
	Name         *string `json:"name,omitempty"`
	Kills        int64   `json:"kills"`
	Losses       int64   `json:"losses"`
	ISKDestroyed float64 `json:"isk_destroyed"`
	ISKLost      float64 `json:"isk_lost"`
}

type ShipsUsedOutput struct {
	Entity    Entity               `json:"entity"`
	Filters   ShipsUsedFilters     `json:"filters"`
	Totals    ShipsUsedTotals      `json:"totals"`
	GroupBy   string               `json:"group_by,omitempty"`
	Breakdown []ShipsUsedBreakdown `json:"breakdown,omitempty"`
}

func registerShipsUsedTool(registry *Registry) error {
	return addTool(registry, ToolDefinition{
		Name: "ships_used", Title: "Analyze ships used",
		Description: "Analyze kills and losses for a character, corporation, or alliance by ship, victim ship, location, or month with optional filters.",
	}, func(ctx context.Context, input ShipsUsedInput) (ShipsUsedOutput, error) {
		return shipsUsed(ctx, registry.deps, input)
	})
}

type shipsUsedQuery struct {
	entity                       ResolvedEntity
	ship, system, region         *ResolvedEntity
	attackerColumn, victimColumn string
	input                        ShipsUsedInput
}

func shipsUsed(ctx context.Context, deps Dependencies, input ShipsUsedInput) (ShipsUsedOutput, error) {
	entity, err := resolveEntity(ctx, deps, input.Entity, input.Type)
	if err != nil {
		return ShipsUsedOutput{}, err
	}
	if entity == nil || organizationAttackerColumns[entity.Type] == "" {
		return ShipsUsedOutput{}, fmt.Errorf("entity must be a character, corporation, or alliance")
	}
	ship, err := optionalTypedEntity(ctx, deps, input.Ship, []EntityType{EntityShip}, "ship")
	if err != nil {
		return ShipsUsedOutput{}, err
	}
	system, err := optionalTypedEntity(ctx, deps, input.System, []EntityType{EntitySystem}, "system")
	if err != nil {
		return ShipsUsedOutput{}, err
	}
	region, err := optionalTypedEntity(ctx, deps, input.Region, []EntityType{EntityRegion}, "region")
	if err != nil {
		return ShipsUsedOutput{}, err
	}
	role, groupBy := input.Role, input.GroupBy
	if role == "" {
		role = "all"
	}
	if groupBy == "" {
		groupBy = "ship"
	}
	if groupBy == "victim_ship" && role != "kills" {
		return ShipsUsedOutput{}, fmt.Errorf("group_by victim_ship is only valid with role kills")
	}
	query := shipsUsedQuery{
		entity: *entity, ship: ship, system: system, region: region,
		attackerColumn: organizationAttackerColumns[entity.Type], victimColumn: organizationVictimColumns[entity.Type],
		input: input,
	}
	kills, err := shipsUsedTotalsKills(ctx, deps, query)
	if err != nil {
		return ShipsUsedOutput{}, err
	}
	losses, err := shipsUsedTotalsLosses(ctx, deps, query)
	if err != nil {
		return ShipsUsedOutput{}, err
	}
	output := ShipsUsedOutput{
		Entity:  entity.Public(deps.BaseURL),
		Filters: ShipsUsedFilters{Ship: publicNamedShip(ship), Role: role, From: input.From, To: input.To, System: publicEntity(system, deps.BaseURL), Region: publicEntity(region, deps.BaseURL)},
	}
	if role == "kills" || role == "all" {
		output.Totals.Kills, output.Totals.ISKDestroyed = kills.Kills, kills.ISKDestroyed
	}
	if role == "losses" || role == "all" {
		output.Totals.Losses, output.Totals.ISKLost = losses.Losses, losses.ISKLost
	}
	if groupBy != "none" {
		limit := input.Limit
		if limit == 0 {
			limit = 10
		}
		output.GroupBy = groupBy
		output.Breakdown, err = shipsUsedBreakdown(ctx, deps, query, role, groupBy, clamp(limit, 1, 50))
		if err != nil {
			return ShipsUsedOutput{}, err
		}
	}
	return output, nil
}

func shipsUsedTotalsKills(ctx context.Context, deps Dependencies, query shipsUsedQuery) (ShipsUsedTotals, error) {
	where, args := shipsAttackerWhere(query)
	rows, err := queryMaps(ctx, deps.DB, fmt.Sprintf(`
		WITH kills AS (
			SELECT DISTINCT attacker.killmail_id FROM killmail_attackers attacker
			JOIN killmails killmail ON killmail.killmail_id = attacker.killmail_id WHERE %s
		)
		SELECT count(*)::bigint AS kills, COALESCE(SUM(killmail.total_value), 0)::double precision AS isk
		FROM kills JOIN killmails killmail ON killmail.killmail_id = kills.killmail_id`, where), args...)
	if err != nil {
		return ShipsUsedTotals{}, err
	}
	row := firstMap(rows)
	return ShipsUsedTotals{Kills: valueInt64(row["kills"]), ISKDestroyed: valueFloat64(row["isk"])}, nil
}

func shipsUsedTotalsLosses(ctx context.Context, deps Dependencies, query shipsUsedQuery) (ShipsUsedTotals, error) {
	where, args := shipsVictimWhere(query)
	rows, err := queryMaps(ctx, deps.DB, fmt.Sprintf(`
		SELECT count(*)::bigint AS losses, COALESCE(SUM(killmail.total_value), 0)::double precision AS isk
		FROM killmails killmail WHERE %s`, where), args...)
	if err != nil {
		return ShipsUsedTotals{}, err
	}
	row := firstMap(rows)
	return ShipsUsedTotals{Losses: valueInt64(row["losses"]), ISKLost: valueFloat64(row["isk"])}, nil
}

func shipsUsedBreakdown(ctx context.Context, deps Dependencies, query shipsUsedQuery, role, groupBy string, limit int) ([]ShipsUsedBreakdown, error) {
	parts, args := []string{}, []any{}
	addPart := func(sql string, partArgs []any) {
		offset := len(args)
		for index := len(partArgs); index >= 1; index-- {
			sql = strings.ReplaceAll(sql, fmt.Sprintf("$%d", index), fmt.Sprintf("$%d", index+offset))
		}
		parts, args = append(parts, sql), append(args, partArgs...)
	}
	if role == "kills" || role == "all" {
		where, whereArgs := shipsAttackerWhere(query)
		dedupeColumns := "attacker.killmail_id"
		if groupBy == "ship" {
			dedupeColumns += ", attacker.ship_type_id"
		}
		addPart(fmt.Sprintf(`
			WITH deduped AS (
				SELECT DISTINCT %s FROM killmail_attackers attacker
				JOIN killmails killmail ON killmail.killmail_id = attacker.killmail_id WHERE %s
			)
			SELECT %s AS key, count(*)::bigint AS kills, 0::bigint AS losses,
			       COALESCE(SUM(killmail.total_value), 0)::double precision AS isk_destroyed,
			       0::double precision AS isk_lost
			FROM deduped JOIN killmails killmail ON killmail.killmail_id = deduped.killmail_id
			%s GROUP BY %s`, dedupeColumns, where, dedupedGroupExpression(groupBy), groupNullFilter(groupBy, dedupedGroupExpression(groupBy)), dedupedGroupExpression(groupBy)), whereArgs)
	}
	if role == "losses" || role == "all" {
		where, whereArgs := shipsVictimWhere(query)
		key := shipsGroupExpression(groupBy, false)
		addPart(fmt.Sprintf(`
			SELECT %s AS key, 0::bigint AS kills, count(*)::bigint AS losses,
			       0::double precision AS isk_destroyed,
			       COALESCE(SUM(killmail.total_value), 0)::double precision AS isk_lost
			FROM killmails killmail WHERE %s %s GROUP BY %s`, key, where, groupAndNullFilter(groupBy, key), key), whereArgs)
	}
	args = append(args, limit)
	rows, err := queryMaps(ctx, deps.DB, fmt.Sprintf(`
		SELECT key, SUM(kills)::bigint AS kills, SUM(losses)::bigint AS losses,
		       SUM(isk_destroyed)::double precision AS isk_destroyed, SUM(isk_lost)::double precision AS isk_lost
		FROM (%s) events GROUP BY key ORDER BY SUM(kills + losses) DESC LIMIT $%d`,
		strings.Join(parts, " UNION ALL "), len(args)), args...)
	if err != nil {
		return nil, err
	}
	names := map[int64]*string{}
	if groupBy != "month" {
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, valueInt64(row["key"]))
		}
		table, id, name := "inv_types", "type_id", "name"
		if groupBy == "system" {
			table, id, name = "solar_systems", "solar_system_id", "system_name"
		} else if groupBy == "region" {
			table, id, name = "regions", "region_id", "name"
		}
		nameRows, loadErr := queryMaps(ctx, deps.DB, fmt.Sprintf("SELECT %s AS id, %s AS name FROM %s WHERE %s = ANY($1)", id, name, table, id), ids)
		if loadErr != nil {
			return nil, loadErr
		}
		for _, row := range nameRows {
			names[valueInt64(row["id"])] = nullableString(row["name"])
		}
	}
	output := make([]ShipsUsedBreakdown, 0, len(rows))
	for _, row := range rows {
		item := ShipsUsedBreakdown{Kills: valueInt64(row["kills"]), Losses: valueInt64(row["losses"]), ISKDestroyed: valueFloat64(row["isk_destroyed"]), ISKLost: valueFloat64(row["isk_lost"])}
		if groupBy == "month" {
			item.Month = nullableString(row["key"])
		} else {
			id := valueInt64(row["key"])
			item.Name = names[id]
			if groupBy == "system" {
				item.SystemID = &id
			} else if groupBy == "region" {
				item.RegionID = &id
			} else {
				item.TypeID = &id
			}
		}
		output = append(output, item)
	}
	return output, nil
}

func shipsAttackerWhere(query shipsUsedQuery) (string, []any) {
	parts, args := []string{fmt.Sprintf("attacker.%s = $1", query.attackerColumn)}, []any{query.entity.ID}
	addQueryFilter(&parts, &args, "attacker.killmail_time >= $%d", query.input.From)
	addQueryFilter(&parts, &args, "attacker.killmail_time < $%d", query.input.To)
	if query.ship != nil {
		addQueryFilter(&parts, &args, "attacker.ship_type_id = $%d", &query.ship.ID)
	}
	if query.system != nil {
		addQueryFilter(&parts, &args, "killmail.solar_system_id = $%d", &query.system.ID)
	}
	if query.region != nil {
		addQueryFilter(&parts, &args, "killmail.region_id = $%d", &query.region.ID)
	}
	return strings.Join(parts, " AND "), args
}

func shipsVictimWhere(query shipsUsedQuery) (string, []any) {
	parts, args := []string{fmt.Sprintf("killmail.%s = $1", query.victimColumn)}, []any{query.entity.ID}
	addQueryFilter(&parts, &args, "killmail.killmail_time >= $%d", query.input.From)
	addQueryFilter(&parts, &args, "killmail.killmail_time < $%d", query.input.To)
	if query.ship != nil {
		addQueryFilter(&parts, &args, "killmail.victim_ship_type_id = $%d", &query.ship.ID)
	}
	if query.system != nil {
		addQueryFilter(&parts, &args, "killmail.solar_system_id = $%d", &query.system.ID)
	}
	if query.region != nil {
		addQueryFilter(&parts, &args, "killmail.region_id = $%d", &query.region.ID)
	}
	return strings.Join(parts, " AND "), args
}

func addQueryFilter[T any](parts *[]string, args *[]any, expression string, value *T) {
	if value == nil {
		return
	}
	*args = append(*args, *value)
	*parts = append(*parts, fmt.Sprintf(expression, len(*args)))
}

func shipsGroupExpression(groupBy string, attacker bool) string {
	switch groupBy {
	case "ship":
		if attacker {
			return "attacker.ship_type_id"
		}
		return "killmail.victim_ship_type_id"
	case "victim_ship":
		return "killmail.victim_ship_type_id"
	case "system":
		return "killmail.solar_system_id"
	case "region":
		return "killmail.region_id"
	default:
		return "to_char(date_trunc('month', killmail.killmail_time), 'YYYY-MM')"
	}
}

func dedupedGroupExpression(groupBy string) string {
	if groupBy == "ship" {
		return "deduped.ship_type_id"
	}
	return shipsGroupExpression(groupBy, true)
}

func groupNullFilter(groupBy, expression string) string {
	if groupBy == "month" {
		return ""
	}
	return "WHERE " + expression + " IS NOT NULL"
}

func groupAndNullFilter(groupBy, expression string) string {
	if groupBy == "month" {
		return ""
	}
	return "AND " + expression + " IS NOT NULL"
}
