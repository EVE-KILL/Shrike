package api

import (
	"context"
	"fmt"
	"strings"
)

func loadEntityKillmailPage(
	ctx context.Context,
	db Database,
	entityType string,
	entityID int64,
	role string,
	page pagination,
) (legacyPayload, error) {
	var idRows []map[string]any
	var err error

	if role == "losses" {
		column := map[string]string{
			"character":   "victim_character_id",
			"corporation": "victim_corporation_id",
			"alliance":    "victim_alliance_id",
		}[entityType]
		where := []string{column + " = $1"}
		args := []any{entityID}
		if page.After != nil {
			args = append(args, *page.After)
			where = append(where, fmt.Sprintf("killmail_id > $%d", len(args)))
		}
		if page.Before != nil {
			args = append(args, *page.Before)
			where = append(where, fmt.Sprintf("killmail_id < $%d", len(args)))
		}
		order := "ASC"
		if page.Before != nil {
			order = "DESC"
		}
		args = append(args, page.Limit+1)
		idRows, err = queryMaps(ctx, db,
			`SELECT killmail_id FROM killmails WHERE `+
				strings.Join(where, " AND ")+
				fmt.Sprintf(" ORDER BY killmail_id %s LIMIT $%d", order, len(args)),
			args...,
		)
	} else {
		column := map[string]string{
			"character":   "character_id",
			"corporation": "corporation_id",
			"alliance":    "alliance_id",
		}[entityType]
		reverse := page.Before != nil
		cursor := page.After
		if reverse {
			cursor = page.Before
		}
		args := []any{entityID}
		cursorSQL := ""
		if cursor != nil {
			cursorRow, queryErr := queryMap(ctx, db,
				`SELECT killmail_time FROM killmails WHERE killmail_id = $1 LIMIT 1`,
				*cursor,
			)
			if queryErr != nil {
				return legacyPayload{}, queryErr
			}
			if cursorRow != nil {
				args = append(args, cursorRow["killmail_time"], *cursor)
				operator := ">"
				if reverse {
					operator = "<"
				}
				cursorSQL = fmt.Sprintf(
					" AND (killmail_time, killmail_id) %s ($2::timestamptz, $3)",
					operator,
				)
			}
		}
		order := "ASC"
		if reverse {
			order = "DESC"
		}
		args = append(args, page.Limit+1)
		idRows, err = queryMaps(ctx, db, fmt.Sprintf(`
			WITH attacker_kills AS MATERIALIZED (
				SELECT DISTINCT killmail_id, killmail_time
				FROM killmail_attackers
				WHERE %s = $1%s
				ORDER BY killmail_time %s, killmail_id %s
				LIMIT $%d
			)
			SELECT ak.killmail_id
			FROM attacker_kills ak
			ORDER BY ak.killmail_id %s`,
			column, cursorSQL, order, order, len(args), order,
		), args...)
	}
	if err != nil {
		return legacyPayload{}, err
	}

	hasMore := len(idRows) > page.Limit
	if hasMore {
		idRows = idRows[:page.Limit]
	}
	if len(idRows) == 0 {
		return jsonPayload(map[string]any{
			"data": []any{},
			"pagination": map[string]any{
				"hasMore": false,
				"cursor":  nil,
			},
		}), nil
	}
	ids := make([]int32, 0, len(idRows))
	for _, row := range idRows {
		id, _ := int64Value(row["killmail_id"])
		ids = append(ids, int32(id))
	}
	data, err := loadKillmailsESI(ctx, db, ids)
	if err != nil {
		return legacyPayload{}, err
	}
	return jsonPayload(map[string]any{
		"data": data,
		"pagination": map[string]any{
			"hasMore": hasMore,
			"cursor":  ids[len(ids)-1],
		},
	}), nil
}

func loadKillmailsESI(
	ctx context.Context,
	db Database,
	ids []int32,
) ([]map[string]any, error) {
	killmails, err := queryMaps(ctx, db,
		`SELECT * FROM killmails WHERE killmail_id = ANY($1::int[])`, ids)
	if err != nil {
		return nil, err
	}
	attackers, err := queryMaps(ctx, db,
		`SELECT * FROM killmail_attackers
		 WHERE killmail_id = ANY($1::int[])
		 ORDER BY killmail_id, attacker_index`, ids)
	if err != nil {
		return nil, err
	}
	items, err := queryMaps(ctx, db,
		`SELECT * FROM killmail_items
		 WHERE killmail_id = ANY($1::int[])
		 ORDER BY killmail_id, item_index`, ids)
	if err != nil {
		return nil, err
	}
	killmailMap := make(map[int64]map[string]any, len(killmails))
	for _, row := range killmails {
		id, _ := int64Value(row["killmail_id"])
		killmailMap[id] = row
	}
	attackerMap := groupByKillmail(attackers)
	itemMap := groupByKillmail(items)

	result := make([]map[string]any, 0, len(ids))
	for _, rawID := range ids {
		id := int64(rawID)
		killmail := killmailMap[id]
		if killmail == nil {
			continue
		}
		result = append(result, formatKillmailESI(killmail, attackerMap[id], itemMap[id]))
	}
	return result, nil
}

func groupByKillmail(rows []map[string]any) map[int64][]map[string]any {
	result := map[int64][]map[string]any{}
	for _, row := range rows {
		id, _ := int64Value(row["killmail_id"])
		result[id] = append(result[id], row)
	}
	return result
}
