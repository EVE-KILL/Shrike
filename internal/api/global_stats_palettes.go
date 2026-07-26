package api

import (
	"context"
	"fmt"
)

func attachGlobalStatsPalettes(
	ctx context.Context,
	db Database,
	entries []map[string]any,
) ([]map[string]any, error) {
	characterIDs := []int32{}
	corporationIDs := []int32{}
	allianceIDs := []int32{}
	for _, entry := range entries {
		id := int32(int64OrZero(entry["id"]))
		switch stringOrEmpty(entry["type"]) {
		case "character":
			characterIDs = append(characterIDs, id)
		case "corporation":
			corporationIDs = append(corporationIDs, id)
		case "alliance":
			allianceIDs = append(allianceIDs, id)
		}
	}
	if len(characterIDs) == 0 &&
		len(corporationIDs) == 0 &&
		len(allianceIDs) == 0 {
		return entries, nil
	}
	rows, err := queryMaps(ctx, db, `
		SELECT 'character'::text AS entity_type,
		       character.character_id AS id,
		       corporation.palette
		FROM characters character
		LEFT JOIN corporations corporation
		  ON corporation.corporation_id = character.corporation_id
		WHERE character.character_id = ANY($1::int[])
		UNION ALL
		SELECT 'corporation'::text AS entity_type,
		       corporation.corporation_id AS id,
		       corporation.palette
		FROM corporations corporation
		WHERE corporation.corporation_id = ANY($2::int[])
		UNION ALL
		SELECT 'alliance'::text AS entity_type,
		       alliance.alliance_id AS id,
		       corporation.palette
		FROM alliances alliance
		LEFT JOIN corporations corporation
		  ON corporation.corporation_id = alliance.executor_corporation_id
		WHERE alliance.alliance_id = ANY($3::int[])`,
		characterIDs, corporationIDs, allianceIDs)
	if err != nil {
		return nil, err
	}
	return applyGlobalStatsPalettes(entries, rows), nil
}

func applyGlobalStatsPalettes(
	entries, paletteRows []map[string]any,
) []map[string]any {
	palettes := make(map[string]any, len(paletteRows))
	for _, row := range paletteRows {
		key := fmt.Sprintf(
			"%s:%d",
			stringOrEmpty(row["entity_type"]),
			int64OrZero(row["id"]),
		)
		palettes[key] = row["palette"]
	}
	for _, entry := range entries {
		entityType := stringOrEmpty(entry["type"])
		if entityType != "character" &&
			entityType != "corporation" &&
			entityType != "alliance" {
			continue
		}
		key := fmt.Sprintf("%s:%d", entityType, int64OrZero(entry["id"]))
		entry["palette"] = palettes[key]
	}
	return entries
}
