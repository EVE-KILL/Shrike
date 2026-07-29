package mcpserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

var entityTable = map[EntityType]struct {
	table, id, name, ticker string
}{
	EntityCharacter:     {"characters", "character_id", "name", ""},
	EntityCorporation:   {"corporations", "corporation_id", "name", "ticker"},
	EntityAlliance:      {"alliances", "alliance_id", "name", "ticker"},
	EntitySystem:        {"solar_systems", "solar_system_id", "system_name", ""},
	EntityRegion:        {"regions", "region_id", "name", ""},
	EntityConstellation: {"constellations", "constellation_id", "constellation_name", ""},
	EntityFaction:       {"factions", "faction_id", "name", ""},
}

func resolveEntity(
	ctx context.Context,
	deps Dependencies,
	input StringOrInt64,
	typeHint *EntityType,
) (*ResolvedEntity, error) {
	raw := strings.TrimSpace(input.String())
	if raw == "" {
		return nil, nil
	}
	if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
		if typeHint != nil {
			return lookupEntityByID(ctx, deps, id, *typeHint)
		}
		return lookupEntityByAnyID(ctx, deps, id)
	}
	return fuzzyResolveEntity(ctx, deps, raw, typeHint)
}

func lookupEntityByAnyID(
	ctx context.Context,
	deps Dependencies,
	id int64,
) (*ResolvedEntity, error) {
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT id, name, ticker, type
		FROM (
			SELECT character_id AS id, name, NULL::text AS ticker,
			       'character'::text AS type, 1 AS priority
			FROM characters WHERE character_id = $1
			UNION ALL
			SELECT corporation_id, name, ticker, 'corporation', 2
			FROM corporations WHERE corporation_id = $1
			UNION ALL
			SELECT alliance_id, name, ticker, 'alliance', 3
			FROM alliances WHERE alliance_id = $1
			UNION ALL
			SELECT solar_system_id, system_name, NULL::text, 'system', 4
			FROM solar_systems WHERE solar_system_id = $1
			UNION ALL
			SELECT region_id, name, NULL::text, 'region', 5
			FROM regions WHERE region_id = $1
			UNION ALL
			SELECT constellation_id, constellation_name, NULL::text,
			       'constellation', 6
			FROM constellations WHERE constellation_id = $1
			UNION ALL
			SELECT faction_id, name, NULL::text, 'faction', 7
			FROM factions WHERE faction_id = $1
			UNION ALL
			SELECT t.type_id, t.name, NULL::text,
			       CASE WHEN g.category_id = 6 THEN 'ship' ELSE 'item' END, 8
			FROM inv_types t
			JOIN inv_groups g ON g.group_id = t.group_id
			WHERE t.type_id = $1 AND t.published IS TRUE
		) candidates
		ORDER BY priority
		LIMIT 1`, id)
	if err != nil {
		return nil, err
	}
	return resolvedFromMap(firstMap(rows)), nil
}

func lookupEntityByID(
	ctx context.Context,
	deps Dependencies,
	id int64,
	entityType EntityType,
) (*ResolvedEntity, error) {
	if entityType == EntityShip || entityType == EntityItem {
		categoryComparison := "="
		if entityType == EntityItem {
			categoryComparison = "!="
		}
		rows, err := queryMaps(ctx, deps.DB, fmt.Sprintf(`
			SELECT t.type_id AS id, t.name, NULL::text AS ticker, $2::text AS type
			FROM inv_types t
			JOIN inv_groups g ON g.group_id = t.group_id
			WHERE t.type_id = $1 AND t.published IS TRUE
			  AND g.category_id %s 6
			LIMIT 1`, categoryComparison), id, string(entityType))
		if err != nil {
			return nil, err
		}
		return resolvedFromMap(firstMap(rows)), nil
	}
	table, ok := entityTable[entityType]
	if !ok {
		return nil, nil
	}
	ticker := "NULL::text"
	if table.ticker != "" {
		ticker = table.ticker
	}
	rows, err := queryMaps(ctx, deps.DB, fmt.Sprintf(`
		SELECT %s AS id, %s AS name, %s AS ticker, $2::text AS type
		FROM %s
		WHERE %s = $1
		LIMIT 1`,
		table.id, table.name, ticker, table.table, table.id,
	), id, string(entityType))
	if err != nil {
		return nil, err
	}
	return resolvedFromMap(firstMap(rows)), nil
}

func fuzzyResolveEntity(
	ctx context.Context,
	deps Dependencies,
	query string,
	typeHint *EntityType,
) (*ResolvedEntity, error) {
	hint := ""
	if typeHint != nil {
		hint = string(*typeHint)
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT id, name, ticker, type
		FROM (
			SELECT character_id::bigint AS id, name::text AS name,
			       NULL::text AS ticker, 'character'::text AS type,
			       similarity(name, $1)::double precision AS score,
			       7 AS type_rank,
			       (SELECT COALESCE(SUM(kills), 0)
			          FROM stats
			         WHERE entity_type = 0 AND entity_id = character_id
			           AND period_type = 2)::bigint AS activity
			FROM characters
			WHERE ($3 = '' OR $3 = 'character') AND name ILIKE $2
			UNION ALL
			SELECT corporation_id, name, ticker, 'corporation',
			       GREATEST(similarity(name, $1), similarity(ticker, $1)),
			       6, COALESCE(member_count, 0)::bigint
			FROM corporations
			WHERE ($3 = '' OR $3 = 'corporation')
			  AND (name ILIKE $2 OR ticker ILIKE $2)
			UNION ALL
			SELECT alliance_id, name, ticker, 'alliance',
			       GREATEST(similarity(name, $1), similarity(ticker, $1)),
			       5, COALESCE(member_count, 0)::bigint
			FROM alliances
			WHERE ($3 = '' OR $3 = 'alliance')
			  AND (name ILIKE $2 OR ticker ILIKE $2)
			UNION ALL
			SELECT t.type_id, t.name, NULL::text,
			       CASE WHEN g.category_id = 6 THEN 'ship' ELSE 'item' END,
			       similarity(t.name, $1), 1, 0
			FROM inv_types t
			JOIN inv_groups g ON g.group_id = t.group_id
			WHERE t.published IS TRUE AND t.name ILIKE $2
			  AND ($3 = '' OR ($3 = 'ship' AND g.category_id = 6)
			       OR ($3 = 'item' AND g.category_id != 6))
			UNION ALL
			SELECT solar_system_id, system_name, NULL::text, 'system',
			       similarity(system_name, $1), 3, 0
			FROM solar_systems
			WHERE ($3 = '' OR $3 = 'system') AND system_name ILIKE $2
			UNION ALL
			SELECT region_id, name, NULL::text, 'region',
			       similarity(name, $1), 2, 0
			FROM regions
			WHERE ($3 = '' OR $3 = 'region') AND name ILIKE $2
			UNION ALL
			SELECT constellation_id, constellation_name, NULL::text,
			       'constellation', similarity(constellation_name, $1), 2, 0
			FROM constellations
			WHERE ($3 = '' OR $3 = 'constellation')
			  AND constellation_name ILIKE $2
			UNION ALL
			SELECT faction_id, name, NULL::text, 'faction',
			       similarity(name, $1), 4, 0
			FROM factions
			WHERE ($3 = '' OR $3 = 'faction') AND name ILIKE $2
		) candidates
		ORDER BY score DESC, type_rank, activity DESC
		LIMIT 1`, query, query+"%", hint)
	if err != nil {
		return nil, err
	}
	return resolvedFromMap(firstMap(rows)), nil
}

func resolvedFromMap(row map[string]any) *ResolvedEntity {
	if row == nil {
		return nil
	}
	return &ResolvedEntity{
		ID:     valueInt64(row["id"]),
		Name:   valueString(row["name"]),
		Type:   EntityType(valueString(row["type"])),
		Ticker: nullableString(row["ticker"]),
	}
}
