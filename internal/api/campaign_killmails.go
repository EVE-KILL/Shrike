package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	campaignengine "github.com/eve-kill/shrike/internal/campaign"
)

const campaignKilllistSelect = `
	SELECT
		k.killmail_id, k.killmail_hash, k.killmail_time,
		COALESCE(k.total_value, 0) AS total_value,
		COALESCE(k.attacker_count, 0) AS attacker_count,
		COALESCE(k.is_npc, false) AS is_npc,
		COALESCE(k.is_solo, false) AS is_solo,
		k.victim_ship_type_id AS ship_type_id,
		ship.name AS ship_name,
		k.victim_ship_group_id AS ship_group_id,
		ship_group.name AS ship_group_name,
		ship.market_group_id AS _ship_market_group_id,
		ship.meta_group_id,
		k.victim_character_id,
		victim_character.name AS victim_character_name,
		k.victim_corporation_id,
		victim_corporation.name AS victim_corporation_name,
		k.victim_alliance_id,
		victim_alliance.name AS victim_alliance_name,
		k.victim_faction_id,
		final_blow.character_id AS final_blow_character_id,
		final_character.name AS final_blow_character_name,
		final_blow.corporation_id AS final_blow_corporation_id,
		final_corporation.name AS final_blow_corporation_name,
		final_blow.alliance_id AS final_blow_alliance_id,
		final_alliance.name AS final_blow_alliance_name,
		final_blow.ship_type_id AS final_blow_ship_type_id,
		final_ship.name AS final_blow_ship_name,
		k.solar_system_id, system.system_name AS solar_system_name,
		system.security AS solar_system_security,
		k.region_id, region.name AS region_name
	FROM killmails k
	LEFT JOIN LATERAL (
		SELECT character_id, corporation_id, alliance_id, ship_type_id
		FROM killmail_attackers
		WHERE killmail_id = k.killmail_id AND final_blow = true
		ORDER BY attacker_index
		LIMIT 1
	) final_blow ON true
	LEFT JOIN inv_types ship ON ship.type_id = k.victim_ship_type_id
	LEFT JOIN inv_groups ship_group ON ship_group.group_id = k.victim_ship_group_id
	LEFT JOIN characters victim_character
	  ON victim_character.character_id = k.victim_character_id
	LEFT JOIN corporations victim_corporation
	  ON victim_corporation.corporation_id = k.victim_corporation_id
	LEFT JOIN alliances victim_alliance
	  ON victim_alliance.alliance_id = k.victim_alliance_id
	LEFT JOIN characters final_character
	  ON final_character.character_id = final_blow.character_id
	LEFT JOIN corporations final_corporation
	  ON final_corporation.corporation_id = final_blow.corporation_id
	LEFT JOIN alliances final_alliance
	  ON final_alliance.alliance_id = final_blow.alliance_id
	LEFT JOIN inv_types final_ship
	  ON final_ship.type_id = final_blow.ship_type_id
	LEFT JOIN solar_systems system
	  ON system.solar_system_id = k.solar_system_id
	LEFT JOIN regions region ON region.region_id = k.region_id`

func campaignKilllistConditions(
	entityRows []map[string]any,
	side *int16,
	start, end time.Time,
	location campaignengine.Location,
) ([]string, []any) {
	args := []any{start, end}
	where := []string{
		"k.killmail_time >= $1::timestamptz",
		"k.killmail_time <= $2::timestamptz",
	}
	addArray := func(values []int32) string {
		args = append(args, values)
		return fmt.Sprintf("$%d::int[]", len(args))
	}

	locationParts := make([]string, 0, 3)
	if len(location.SystemIDs) > 0 {
		locationParts = append(locationParts,
			"k.solar_system_id = ANY("+addArray(location.SystemIDs)+")")
	}
	if len(location.ConstellationIDs) > 0 {
		locationParts = append(locationParts,
			"k.constellation_id = ANY("+addArray(location.ConstellationIDs)+")")
	}
	if len(location.RegionIDs) > 0 {
		locationParts = append(locationParts,
			"k.region_id = ANY("+addArray(location.RegionIDs)+")")
	}
	if len(locationParts) > 0 {
		where = append(where, "("+strings.Join(locationParts, " OR ")+")")
	}

	characters := []int32{}
	corporations := []int32{}
	alliances := []int32{}
	for _, row := range entityRows {
		if side != nil && int16From(row["side_index"]) != *side {
			continue
		}
		id := int32From(row["entity_id"])
		switch int16From(row["entity_type"]) {
		case campaignengine.EntityCharacter:
			characters = append(characters, id)
		case campaignengine.EntityCorporation:
			corporations = append(corporations, id)
		case campaignengine.EntityAlliance:
			alliances = append(alliances, id)
		}
	}

	attackerParts := make([]string, 0, 3)
	victimParts := make([]string, 0, 3)
	if len(characters) > 0 {
		placeholder := addArray(characters)
		attackerParts = append(attackerParts,
			"attacker.character_id = ANY("+placeholder+")")
		victimParts = append(victimParts,
			"k.victim_character_id = ANY("+placeholder+")")
	}
	if len(corporations) > 0 {
		placeholder := addArray(corporations)
		attackerParts = append(attackerParts,
			"attacker.corporation_id = ANY("+placeholder+")")
		victimParts = append(victimParts,
			"k.victim_corporation_id = ANY("+placeholder+")")
	}
	if len(alliances) > 0 {
		placeholder := addArray(alliances)
		attackerParts = append(attackerParts,
			"attacker.alliance_id = ANY("+placeholder+")")
		victimParts = append(victimParts,
			"k.victim_alliance_id = ANY("+placeholder+")")
	}

	if len(entityRows) == 0 {
		if len(locationParts) == 0 {
			where = append(where, "false")
		}
		return where, args
	}
	if len(attackerParts) == 0 && len(victimParts) == 0 {
		where = append(where, "false")
		return where, args
	}
	attackerMatch := `EXISTS (
		SELECT 1 FROM killmail_attackers attacker
		WHERE attacker.killmail_id = k.killmail_id
		  AND (` + strings.Join(attackerParts, " OR ") + `)
	)`
	victimMatch := "(" + strings.Join(victimParts, " OR ") + ")"
	if side != nil {
		// A side feed is its kills, not a duplicate of its losses.
		where = append(where, attackerMatch, "NOT "+victimMatch)
	} else {
		where = append(where, "("+attackerMatch+" OR "+victimMatch+")")
	}
	return where, args
}

func loadCampaignKilllistPage(
	ctx context.Context,
	db Database,
	where []string,
	args []any,
	limit int,
) (legacyPayload, error) {
	args = append(args, limit+1)
	query := campaignKilllistSelect +
		" WHERE " + strings.Join(where, " AND ") +
		fmt.Sprintf(
			" ORDER BY k.killmail_id DESC LIMIT $%d",
			len(args),
		)
	rows, err := queryMaps(ctx, db, query, args...)
	if err != nil {
		return legacyPayload{}, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	paths, err := loadEntityKilllistMarketPaths(ctx, db, rows)
	if err != nil {
		return legacyPayload{}, err
	}
	for _, row := range rows {
		marketGroup := int64From(row["_ship_market_group_id"])
		row["ship_market_path"] = paths[marketGroup]
		delete(row, "_ship_market_group_id")
	}
	var cursor any
	if len(rows) > 0 {
		cursor = rows[len(rows)-1]["killmail_id"]
	}
	return jsonPayload(map[string]any{
		"kills": rows, "hasMore": hasMore, "cursor": cursor,
	}), nil
}
