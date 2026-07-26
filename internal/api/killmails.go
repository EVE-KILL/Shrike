package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const killlistSelect = `
	SELECT
		k.killmail_id, k.killmail_hash, k.killmail_time,
		COALESCE(k.total_value, 0) AS total_value,
		COALESCE(k.attacker_count, 0) AS attacker_count,
		COALESCE(k.is_npc, false) AS is_npc,
		COALESCE(k.is_solo, false) AS is_solo,
		k.victim_ship_type_id AS ship_type_id,
		ship.name AS ship_name,
		ship_group.name AS ship_group_name,
		k.victim_character_id, victim_character.name AS victim_character_name,
		k.victim_corporation_id, victim_corporation.name AS victim_corporation_name,
		k.victim_alliance_id, victim_alliance.name AS victim_alliance_name,
		final_blow.character_id AS final_blow_character_id,
		final_character.name AS final_blow_character_name,
		final_blow.corporation_id AS final_blow_corporation_id,
		final_corporation.name AS final_blow_corporation_name,
		final_blow.alliance_id AS final_blow_alliance_id,
		final_alliance.name AS final_blow_alliance_name,
		k.solar_system_id, system.system_name AS solar_system_name,
		system.security AS solar_system_security,
		k.region_id, region.name AS region_name
	FROM killmails k
	LEFT JOIN LATERAL (
		SELECT character_id, corporation_id, alliance_id
		FROM killmail_attackers
		WHERE killmail_id = k.killmail_id AND final_blow = true
		ORDER BY attacker_index
		LIMIT 1
	) final_blow ON true
	LEFT JOIN inv_types ship ON ship.type_id = k.victim_ship_type_id
	LEFT JOIN inv_groups ship_group ON ship_group.group_id = k.victim_ship_group_id
	LEFT JOIN characters victim_character ON victim_character.character_id = k.victim_character_id
	LEFT JOIN corporations victim_corporation ON victim_corporation.corporation_id = k.victim_corporation_id
	LEFT JOIN alliances victim_alliance ON victim_alliance.alliance_id = k.victim_alliance_id
	LEFT JOIN characters final_character ON final_character.character_id = final_blow.character_id
	LEFT JOIN corporations final_corporation ON final_corporation.corporation_id = final_blow.corporation_id
	LEFT JOIN alliances final_alliance ON final_alliance.alliance_id = final_blow.alliance_id
	LEFT JOIN solar_systems system ON system.solar_system_id = k.solar_system_id
	LEFT JOIN regions region ON region.region_id = k.region_id`

func registerKillmailRoutes(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "killmails",
		Method:      http.MethodGet,
		Path:        "/killmails",
		Summary:     "Filtered killmail list",
		Tags:        []string{"killmails"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		page := parsePagination(req.Query)
		where, args := killmailTypeConditions(req.Query.Get("type"))

		factions := parseCommaInt32(req.Query.Get("victimFactions"))
		if len(factions) > 0 {
			args = append(args, factions)
			where = append(where, fmt.Sprintf("k.victim_faction_id = ANY($%d::int[])", len(args)))
		}
		return loadKilllistPage(ctx, opts.DB, where, args, page)
	})

	registerLegacy(a, huma.Operation{
		OperationID: "killmails-count",
		Method:      http.MethodGet,
		Path:        "/killmails/count",
		Summary:     "Estimated killmail count",
		Tags:        []string{"killmails"},
	}, func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		count, err := estimatedRows(ctx, opts.DB, "killmails")
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"count": count}), nil
	})

	registerLegacy(a, killmailIDOperation(
		"killmail-esi", "/killmails/{id}/esi", "ESI-compatible killmail",
	), func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		body, found, err := loadKillmailESI(ctx, opts.DB, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if !found {
			return legacyPayload{}, apiError(http.StatusNotFound, "Killmail not found")
		}
		return jsonPayload(body), nil
	})

	registerLegacy(a, killmailIDOperation(
		"killmail-fitting", "/killmails/{id}/fitting", "Killmail fitting by slot",
	), func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		body, found, err := loadKillmailFitting(ctx, opts.DB, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if !found {
			return legacyPayload{}, apiError(http.StatusNotFound, "Killmail not found")
		}
		return jsonPayload(body), nil
	})

	registerLegacy(a, huma.Operation{
		OperationID: "killmail-eft",
		Method:      http.MethodGet,
		Path:        "/killmails/{id}/eft",
		Summary:     "EFT-format killmail fitting",
		Tags:        []string{"killmails"},
		Parameters:  idParameter(),
		Responses: map[string]*huma.Response{
			"200": {
				Description: "EFT fitting",
				Content: map[string]*huma.MediaType{
					"text/plain": {Schema: &huma.Schema{Type: huma.TypeString}},
				},
			},
		},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		body, found, err := loadKillmailEFT(ctx, opts.DB, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if !found {
			return legacyPayload{}, apiError(http.StatusNotFound, "Killmail not found")
		}
		return legacyPayload{
			ContentType: "text/plain; charset=utf-8",
			RawBody:     []byte(body),
		}, nil
	})

	registerLegacy(a, killmailIDOperation(
		"killmail", "/killmails/{id}", "Full killmail detail",
	), func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		body, found, err := loadKillmailDetail(ctx, opts.DB, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if !found {
			return legacyPayload{}, apiError(http.StatusNotFound, "Killmail not found")
		}
		return jsonPayload(body), nil
	})
}

func killmailIDOperation(id, path, summary string) huma.Operation {
	return huma.Operation{
		OperationID: id,
		Method:      http.MethodGet,
		Path:        path,
		Summary:     summary,
		Tags:        []string{"killmails"},
		Parameters:  idParameter(),
	}
}

func idParameter() []*huma.Param {
	return []*huma.Param{{
		Name:     "id",
		In:       "path",
		Required: true,
		Schema:   &huma.Schema{Type: huma.TypeString},
	}}
}

func estimatedRows(ctx context.Context, db Database, table string) (int64, error) {
	// Table names are internal constants, never request input.
	row, err := queryMap(ctx, db, `
		SELECT GREATEST(c.reltuples::bigint, COALESCE(s.n_live_tup, 0))::bigint AS count
		FROM pg_class c
		LEFT JOIN pg_stat_user_tables s ON s.relid = c.oid
		WHERE c.relname = $1 AND c.relkind = 'r'`, table)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	count, _ := int64Value(row["count"])
	return count, nil
}

func loadKilllistPage(
	ctx context.Context,
	db Database,
	where []string,
	args []any,
	page pagination,
) (legacyPayload, error) {
	// The cursor is applied here rather than by each caller. It used to be the
	// caller's job, and /sde/systems/{id}/kills and /sde/regions/{id}/kills
	// simply did not do it — they flipped the sort order for `before` but never
	// filtered, so every page returned the same first rows and a client
	// paginating those endpoints walked forever. Applying it once, next to the
	// ORDER BY it has to agree with, removes the chance of a third caller
	// forgetting.
	if page.After != nil {
		args = append(args, *page.After)
		where = append(where, fmt.Sprintf("k.killmail_id > $%d", len(args)))
	}
	if page.Before != nil {
		args = append(args, *page.Before)
		where = append(where, fmt.Sprintf("k.killmail_id < $%d", len(args)))
	}

	query := killlistSelect
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	order := "ASC"
	if page.Before != nil {
		order = "DESC"
	}
	args = append(args, page.Limit+1)
	query += fmt.Sprintf(" ORDER BY k.killmail_id %s LIMIT $%d", order, len(args))
	rows, err := queryMaps(ctx, db, query, args...)
	if err != nil {
		return legacyPayload{}, err
	}
	return paginatedRows(rows, page.Limit, "killmail_id"), nil
}

func killmailTypeConditions(raw string) ([]string, []any) {
	kind := raw
	if kind == "" {
		kind = "latest"
	}
	condition := map[string]string{
		"highsec":        "k.solar_system_id IN (SELECT solar_system_id FROM solar_systems WHERE security >= 0.45)",
		"lowsec":         "k.solar_system_id IN (SELECT solar_system_id FROM solar_systems WHERE security > 0.0 AND security < 0.45)",
		"nullsec":        "k.solar_system_id IN (SELECT solar_system_id FROM solar_systems WHERE security <= 0.0 AND region_id < 11000000)",
		"wspace":         "k.region_id >= 11000001 AND k.region_id <= 11000033",
		"abyssal":        "k.region_id >= 12000001 AND k.region_id <= 12000005",
		"pochven":        "k.region_id = 10000070",
		"jove":           "k.region_id IN (10000004, 10000017, 10000019)",
		"solo":           "k.is_solo = true",
		"npc":            "k.is_npc = true",
		"big":            "k.total_value >= 1000000000",
		"5b":             "k.total_value >= 5000000000",
		"10b":            "k.total_value >= 10000000000",
		"frigates":       "k.victim_ship_group_id IN (25,324,830,831,834,893,1283,1527)",
		"destroyers":     "k.victim_ship_group_id IN (420,1305,1534)",
		"cruisers":       "k.victim_ship_group_id IN (26,358,832,833,906,894,963,1972)",
		"battlecruisers": "k.victim_ship_group_id IN (419,1201,540)",
		"battleships":    "k.victim_ship_group_id IN (27,898,900)",
		"capitals":       "k.victim_ship_group_id IN (547,485,1538,883,659,30,4594)",
		"supercarriers":  "k.victim_ship_group_id = 659",
		"titans":         "k.victim_ship_group_id = 30",
		"freighters":     "k.victim_ship_group_id IN (513,902)",
		"citadels":       "k.victim_ship_group_id IN (SELECT group_id FROM inv_groups WHERE category_id = 65)",
		"t1":             "k.victim_ship_type_id IN (SELECT type_id FROM inv_types WHERE COALESCE(meta_group_id, 1) = 1)",
		"t2":             "k.victim_ship_type_id IN (SELECT type_id FROM inv_types WHERE meta_group_id = 2)",
		"t3":             "k.victim_ship_type_id IN (SELECT type_id FROM inv_types WHERE meta_group_id = 14)",
		"faction":        "k.victim_ship_type_id IN (SELECT type_id FROM inv_types WHERE meta_group_id = 4)",
	}[kind]
	if condition == "" {
		return nil, nil
	}
	return []string{condition}, nil
}

func parseCommaInt32(raw string) []int32 {
	parts := strings.Split(raw, ",")
	out := make([]int32, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || value == 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		if value < math.MinInt32 || value > math.MaxInt32 {
			continue
		}
		out = append(out, int32(value))
	}
	return out
}

func loadKillmailESI(ctx context.Context, db Database, id int64) (map[string]any, bool, error) {
	killmail, err := queryMap(ctx, db, `SELECT * FROM killmails WHERE killmail_id = $1 LIMIT 1`, id)
	if err != nil || killmail == nil {
		return nil, false, err
	}
	attackers, err := queryMaps(ctx, db,
		`SELECT * FROM killmail_attackers WHERE killmail_id = $1 ORDER BY attacker_index`, id)
	if err != nil {
		return nil, false, err
	}
	items, err := queryMaps(ctx, db,
		`SELECT * FROM killmail_items WHERE killmail_id = $1 ORDER BY item_index`, id)
	if err != nil {
		return nil, false, err
	}
	return formatKillmailESI(killmail, attackers, items), true, nil
}

func formatKillmailESI(
	killmail map[string]any,
	attackerRows []map[string]any,
	itemRows []map[string]any,
) map[string]any {
	victim := map[string]any{
		"damage_taken": zeroIfNil(killmail["victim_damage_taken"]),
		"items":        make([]map[string]any, 0, len(itemRows)),
		"position": map[string]any{
			"x": zeroIfNil(killmail["position_x"]),
			"y": zeroIfNil(killmail["position_y"]),
			"z": zeroIfNil(killmail["position_z"]),
		},
	}
	copyIfNotNil(victim, killmail, "character_id", "victim_character_id")
	copyIfNotNil(victim, killmail, "corporation_id", "victim_corporation_id")
	copyIfNotNil(victim, killmail, "alliance_id", "victim_alliance_id")
	copyIfNotNil(victim, killmail, "faction_id", "victim_faction_id")
	copyIfNotNil(victim, killmail, "ship_type_id", "victim_ship_type_id")

	items := victim["items"].([]map[string]any)
	for _, item := range itemRows {
		items = append(items, map[string]any{
			"item_type_id":       item["type_id"],
			"flag":               item["flag_id"],
			"quantity_dropped":   zeroIfNil(item["quantity_dropped"]),
			"quantity_destroyed": zeroIfNil(item["quantity_destroyed"]),
			"singleton":          zeroIfNil(item["singleton"]),
		})
	}
	victim["items"] = items

	attackers := make([]map[string]any, 0, len(attackerRows))
	for _, row := range attackerRows {
		attacker := map[string]any{
			"damage_done":     zeroIfNil(row["damage_done"]),
			"final_blow":      falseIfNil(row["final_blow"]),
			"security_status": zeroIfNil(row["security_status"]),
		}
		for _, key := range []string{
			"character_id", "corporation_id", "alliance_id", "faction_id",
			"ship_type_id", "weapon_type_id",
		} {
			if row[key] != nil {
				attacker[key] = row[key]
			}
		}
		attackers = append(attackers, attacker)
	}

	body := map[string]any{
		"killmail_id":     killmail["killmail_id"],
		"killmail_hash":   killmail["killmail_hash"],
		"killmail_time":   normalizeJSON(killmail["killmail_time"]),
		"solar_system_id": killmail["solar_system_id"],
		"victim":          victim,
		"attackers":       attackers,
	}
	if killmail["war_id"] != nil {
		body["war_id"] = killmail["war_id"]
	}
	return body
}

func copyIfNotNil(dst, src map[string]any, dstKey, srcKey string) {
	if src[srcKey] != nil {
		dst[dstKey] = src[srcKey]
	}
}

func zeroIfNil(value any) any {
	if value == nil {
		return 0
	}
	return value
}

func falseIfNil(value any) any {
	if value == nil {
		return false
	}
	return value
}

type fittingItem struct {
	TypeID   int64
	Name     string
	Quantity int64
}

func loadKillmailFitting(
	ctx context.Context,
	db Database,
	id int64,
) (map[string]any, bool, error) {
	ship, items, typeNames, found, err := loadFittingData(ctx, db, id)
	if err != nil || !found {
		return nil, found, err
	}
	slots := fittingSlots(items, typeNames)
	body := map[string]any{
		"killmail_id": id,
		"ship": map[string]any{
			"type_id": ship,
			"name":    knownName(typeNames, ship),
		},
	}
	for name, slotItems := range slots {
		values := make([]map[string]any, 0, len(slotItems))
		for _, item := range slotItems {
			values = append(values, map[string]any{
				"type_id":  item.TypeID,
				"name":     item.Name,
				"quantity": item.Quantity,
			})
		}
		body[name] = values
	}
	return body, true, nil
}

func loadKillmailEFT(
	ctx context.Context,
	db Database,
	id int64,
) (string, bool, error) {
	ship, items, typeNames, found, err := loadFittingData(ctx, db, id)
	if err != nil || !found {
		return "", found, err
	}
	slots := fittingSlots(items, typeNames)
	lines := []string{fmt.Sprintf("[%s, Killmail #%d]", knownName(typeNames, ship), id)}
	for _, slot := range []string{"subsystem", "high", "mid", "low", "rig", "service"} {
		slotItems := slots[slot]
		if len(slotItems) == 0 {
			continue
		}
		lines = append(lines, "")
		for _, item := range slotItems {
			for i := int64(0); i < item.Quantity; i++ {
				lines = append(lines, item.Name)
			}
		}
	}
	for _, slot := range []string{"drone", "fighter", "cargo", "implant", "booster", "fuel"} {
		slotItems := slots[slot]
		if len(slotItems) == 0 {
			continue
		}
		lines = append(lines, "", "")
		for _, item := range slotItems {
			if item.Quantity > 1 {
				lines = append(lines, fmt.Sprintf("%s x%d", item.Name, item.Quantity))
			} else {
				lines = append(lines, item.Name)
			}
		}
	}
	return strings.Join(lines, "\n"), true, nil
}

func loadFittingData(
	ctx context.Context,
	db Database,
	id int64,
) (any, []map[string]any, map[int64]string, bool, error) {
	row, err := queryMap(ctx, db,
		`SELECT victim_ship_type_id FROM killmails WHERE killmail_id = $1 LIMIT 1`, id)
	if err != nil || row == nil {
		return nil, nil, nil, false, err
	}
	items, err := queryMaps(ctx, db,
		`SELECT * FROM killmail_items WHERE killmail_id = $1 ORDER BY item_index`, id)
	if err != nil {
		return nil, nil, nil, false, err
	}
	values := []any{row["victim_ship_type_id"]}
	for _, item := range items {
		values = append(values, item["type_id"])
	}
	typeIDs := int32Slice(values...)
	typeNames := map[int64]string{}
	if len(typeIDs) > 0 {
		rows, queryErr := queryMaps(ctx, db,
			`SELECT type_id, name FROM inv_types WHERE type_id = ANY($1::int[])`, typeIDs)
		if queryErr != nil {
			return nil, nil, nil, false, queryErr
		}
		for _, typeRow := range rows {
			typeID, _ := int64Value(typeRow["type_id"])
			name, _ := stringValue(typeRow["name"])
			typeNames[typeID] = name
		}
	}
	return row["victim_ship_type_id"], items, typeNames, true, nil
}

func fittingSlots(items []map[string]any, names map[int64]string) map[string][]fittingItem {
	slots := map[string][]fittingItem{}
	indexes := map[string]map[int64]int{}
	for _, row := range items {
		if row["parent_index"] != nil {
			continue
		}
		typeID, typeOK := int64Value(row["type_id"])
		flagID, flagOK := int64Value(row["flag_id"])
		dropped, _ := int64Value(row["quantity_dropped"])
		destroyed, _ := int64Value(row["quantity_destroyed"])
		quantity := dropped + destroyed
		if !typeOK || !flagOK || quantity <= 0 {
			continue
		}
		slot := categorizeFlag(flagID)
		if indexes[slot] == nil {
			indexes[slot] = map[int64]int{}
		}
		if index, exists := indexes[slot][typeID]; exists {
			slots[slot][index].Quantity += quantity
			continue
		}
		indexes[slot][typeID] = len(slots[slot])
		slots[slot] = append(slots[slot], fittingItem{
			TypeID: typeID, Name: knownName(names, typeID), Quantity: quantity,
		})
	}
	return slots
}

func knownName(names map[int64]string, value any) string {
	id, ok := int64Value(value)
	if ok {
		if name, exists := names[id]; exists && name != "" {
			return name
		}
	}
	return "Unknown"
}

func categorizeFlag(flagID int64) string {
	switch {
	case flagID >= 27 && flagID <= 34:
		return "high"
	case flagID >= 19 && flagID <= 26:
		return "mid"
	case flagID >= 11 && flagID <= 18:
		return "low"
	case flagID >= 92 && flagID <= 99:
		return "rig"
	case flagID >= 125 && flagID <= 132:
		return "subsystem"
	case flagID >= 159 && flagID <= 163:
		return "fighter"
	case flagID >= 164 && flagID <= 165:
		return "service"
	case flagID == 5:
		return "cargo"
	case flagID == 87:
		return "drone"
	case flagID == 88:
		return "booster"
	case flagID == 89:
		return "implant"
	case flagID == 90:
		return "ship_maintenance_bay"
	case flagID == 133:
		return "fuel"
	case flagID == 155:
		return "fleet"
	case flagID == 158:
		return "fighter_bay"
	case flagID >= 134 && flagID <= 151:
		return "specialized"
	default:
		return "other"
	}
}

func loadKillmailDetail(
	ctx context.Context,
	db Database,
	id int64,
) (map[string]any, bool, error) {
	killmail, err := queryMap(ctx, db, `
		SELECT k.*, vc.name AS victim_character_name,
		       vco.name AS victim_corporation_name,
		       vco.palette AS victim_corporation_palette,
		       va.name AS victim_alliance_name,
		       ship.name AS victim_ship_name,
		       ship.market_group_id AS victim_ship_market_group_id,
		       ship_group.name AS victim_ship_group_name,
		       system.system_name AS solar_system_name,
		       system.security AS solar_system_security,
		       constellation.constellation_name,
		       region.name AS region_name,
		       nearest.item_id AS location_item_id,
		       nearest.item_name AS location_item_name,
		       nearest.type_id AS location_type_id,
		       nearest.group_id AS location_group_id,
		       nearest.distance AS location_distance
		FROM killmails k
		LEFT JOIN characters vc ON vc.character_id = k.victim_character_id
		LEFT JOIN corporations vco ON vco.corporation_id = k.victim_corporation_id
		LEFT JOIN alliances va ON va.alliance_id = k.victim_alliance_id
		LEFT JOIN inv_types ship ON ship.type_id = k.victim_ship_type_id
		LEFT JOIN inv_groups ship_group ON ship_group.group_id = k.victim_ship_group_id
		LEFT JOIN solar_systems system ON system.solar_system_id = k.solar_system_id
		LEFT JOIN constellations constellation
		  ON constellation.constellation_id = COALESCE(
		    system.constellation_id, k.constellation_id
		  )
		LEFT JOIN regions region ON region.region_id = k.region_id
		LEFT JOIN LATERAL (
			SELECT c.item_id, c.item_name, c.type_id, c.group_id,
			       sqrt(
			         power(c.x - k.position_x, 2) +
			         power(c.y - k.position_y, 2) +
			         power(c.z - k.position_z, 2)
			       ) AS distance
			FROM celestials c
			WHERE c.solar_system_id = k.solar_system_id
			  AND k.position_x IS NOT NULL
			  AND k.position_y IS NOT NULL
			  AND k.position_z IS NOT NULL
			  AND c.x IS NOT NULL AND c.y IS NOT NULL AND c.z IS NOT NULL
			ORDER BY distance ASC
			LIMIT 1
		) nearest ON true
		WHERE k.killmail_id = $1
		LIMIT 1`, id)
	if err != nil || killmail == nil {
		return nil, false, err
	}
	attackers, err := queryMaps(ctx, db, `
		SELECT a.character_id, c.name AS character_name,
		       a.corporation_id, co.name AS corporation_name,
		       a.alliance_id, al.name AS alliance_name,
		       a.ship_type_id, ship.name AS ship_name,
		       a.weapon_type_id, weapon.name AS weapon_name,
		       COALESCE(a.damage_done, 0) AS damage_done,
		       COALESCE(a.final_blow, false) AS final_blow,
		       a.security_status
		FROM killmail_attackers a
		LEFT JOIN characters c ON c.character_id = a.character_id
		LEFT JOIN corporations co ON co.corporation_id = a.corporation_id
		LEFT JOIN alliances al ON al.alliance_id = a.alliance_id
		LEFT JOIN inv_types ship ON ship.type_id = a.ship_type_id
		LEFT JOIN inv_types weapon ON weapon.type_id = a.weapon_type_id
		WHERE a.killmail_id = $1
		ORDER BY COALESCE(a.damage_done, 0) DESC, a.attacker_index ASC`, id)
	if err != nil {
		return nil, false, err
	}
	items, err := queryMaps(ctx, db, `
		SELECT i.item_index, i.type_id, t.name AS type_name,
		       t.group_id, g.category_id, i.flag_id,
		       f.flag_name, COALESCE(i.quantity_dropped, 0) AS quantity_dropped,
		       COALESCE(i.quantity_destroyed, 0) AS quantity_destroyed,
		       COALESCE(i.singleton, 0) AS singleton, i.parent_index,
		       EXISTS (
		           SELECT 1 FROM killmail_items child
		           WHERE child.killmail_id = i.killmail_id
		             AND child.parent_index = i.item_index
		             AND child.parent_index <> 0
		       ) AS is_container
		FROM killmail_items i
		LEFT JOIN inv_types t ON t.type_id = i.type_id
		LEFT JOIN inv_groups g ON g.group_id = t.group_id
		LEFT JOIN inv_flags f ON f.flag_id = i.flag_id
		WHERE i.killmail_id = $1
		ORDER BY i.item_index ASC`, id)
	if err != nil {
		return nil, false, err
	}

	typeValues := make([]any, 0, 1+len(attackers)*2+len(items))
	addKnownType := func(typeValue any) {
		typeValues = append(typeValues, typeValue)
	}
	addKnownType(killmail["victim_ship_type_id"])
	for _, attacker := range attackers {
		addKnownType(attacker["ship_type_id"])
		addKnownType(attacker["weapon_type_id"])
	}
	for _, item := range items {
		addKnownType(item["type_id"])
	}
	typeIDs := int32Slice(typeValues...)
	prices, err := loadTypePrices(ctx, db, typeIDs, killmail["killmail_time"])
	if err != nil {
		return nil, false, err
	}

	itemOutput := make([]map[string]any, 0, len(items))
	for _, item := range items {
		typeID, _ := int64Value(item["type_id"])
		flagID, _ := int64Value(item["flag_id"])
		dropped, _ := int64Value(item["quantity_dropped"])
		destroyed, _ := int64Value(item["quantity_destroyed"])
		price := prices[typeID]
		slot := categorizeFlag(flagID)
		if item["parent_index"] != nil {
			slot = "container_item"
		}
		itemOutput = append(itemOutput, map[string]any{
			"item_index":         item["item_index"],
			"type_id":            item["type_id"],
			"type_name":          item["type_name"],
			"group_id":           item["group_id"],
			"category_id":        item["category_id"],
			"flag_id":            item["flag_id"],
			"flag_name":          item["flag_name"],
			"quantity_dropped":   item["quantity_dropped"],
			"quantity_destroyed": item["quantity_destroyed"],
			"singleton":          item["singleton"],
			"parent_index":       item["parent_index"],
			"is_container":       item["is_container"],
			"slot":               slot,
			"price":              price,
			"total_value":        price * float64(dropped+destroyed),
		})
	}

	totalDamage := int64(0)
	for _, attacker := range attackers {
		damage, _ := int64Value(attacker["damage_done"])
		totalDamage += damage
	}

	siblings := []map[string]any{}
	if killmail["victim_character_id"] != nil {
		rows, queryErr := queryMaps(ctx, db, `
			SELECT sibling.killmail_id,
			       sibling.victim_ship_type_id AS ship_type_id,
			       sibling.victim_ship_group_id AS ship_group_id,
			       ship.name AS ship_name,
			       COALESCE(sibling.total_value, 0) AS total_value,
			       sibling.killmail_time
			FROM killmails sibling
			LEFT JOIN inv_types ship
			  ON ship.type_id = sibling.victim_ship_type_id
			WHERE sibling.victim_character_id = $1
			  AND sibling.solar_system_id = $2
			  AND sibling.killmail_time >= $3::timestamptz - interval '1 hour'
			  AND sibling.killmail_time <= $3::timestamptz + interval '1 hour'
			  AND sibling.killmail_id <> $4
			  AND sibling.victim_ship_type_id <> COALESCE($5, 0)
			ORDER BY sibling.killmail_time DESC, sibling.killmail_id DESC
			LIMIT 10`,
			killmail["victim_character_id"],
			killmail["solar_system_id"],
			killmail["killmail_time"],
			id,
			killmail["victim_ship_type_id"],
		)
		if queryErr != nil {
			return nil, false, queryErr
		}
		siblings = nonNilRows(rows)
	}

	marketPath, err := loadMarketPath(
		ctx, db, killmail["victim_ship_market_group_id"],
	)
	if err != nil {
		return nil, false, err
	}
	var location any
	if killmail["location_item_id"] != nil {
		location = map[string]any{
			"item_id":   killmail["location_item_id"],
			"item_name": killmail["location_item_name"],
			"type_id":   killmail["location_type_id"],
			"group_id":  killmail["location_group_id"],
			"distance":  killmail["location_distance"],
		}
	}
	victimShipID, _ := int64Value(killmail["victim_ship_type_id"])
	body := map[string]any{
		"killmail_id":   killmail["killmail_id"],
		"killmail_hash": killmail["killmail_hash"],
		"killmail_time": killmail["killmail_time"],
		"victim": map[string]any{
			"character_id":        killmail["victim_character_id"],
			"character_name":      killmail["victim_character_name"],
			"corporation_id":      killmail["victim_corporation_id"],
			"corporation_name":    killmail["victim_corporation_name"],
			"corporation_palette": killmail["victim_corporation_palette"],
			"alliance_id":         killmail["victim_alliance_id"],
			"alliance_name":       killmail["victim_alliance_name"],
			"ship_type_id":        killmail["victim_ship_type_id"],
			"ship_name":           killmail["victim_ship_name"],
			"ship_group_id":       killmail["victim_ship_group_id"],
			"ship_group_name":     killmail["victim_ship_group_name"],
			"ship_market_path":    marketPath,
			"damage_taken":        zeroIfNil(killmail["victim_damage_taken"]),
			"ship_price":          prices[victimShipID],
		},
		"solar_system_id":       killmail["solar_system_id"],
		"solar_system_name":     killmail["solar_system_name"],
		"solar_system_security": killmail["solar_system_security"],
		"constellation_id":      killmail["constellation_id"],
		"constellation_name":    killmail["constellation_name"],
		"region_id":             killmail["region_id"],
		"region_name":           killmail["region_name"],
		"position_x":            killmail["position_x"],
		"position_y":            killmail["position_y"],
		"position_z":            killmail["position_z"],
		"location":              location,
		"total_value":           zeroIfNil(killmail["total_value"]),
		"fitted_value":          zeroIfNil(killmail["fitted_value"]),
		"dropped_value":         zeroIfNil(killmail["dropped_value"]),
		"destroyed_value":       zeroIfNil(killmail["destroyed_value"]),
		"points":                zeroIfNil(killmail["points"]),
		"attacker_count":        zeroIfNil(killmail["attacker_count"]),
		"is_npc":                falseIfNil(killmail["is_npc"]),
		"is_solo":               falseIfNil(killmail["is_solo"]),
		"total_damage":          totalDamage,
		"attackers":             attackers,
		"items":                 itemOutput,
		"siblings":              siblings,
	}
	return body, true, nil
}

func loadTypePrices(
	ctx context.Context,
	db Database,
	typeIDs []int32,
	killmailTime any,
) (map[int64]float64, error) {
	result := map[int64]float64{}
	if len(typeIDs) == 0 {
		return result, nil
	}
	killTime, ok := killmailTime.(time.Time)
	if !ok {
		return result, nil
	}
	rows, err := queryMaps(ctx, db, `
		SELECT DISTINCT ON (type_id) type_id, average
		FROM prices
		WHERE type_id = ANY($1::int[])
		  AND region_id = 10000002
		ORDER BY type_id, ABS(date - $2::date), date DESC`, typeIDs, killTime)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		typeID, _ := int64Value(row["type_id"])
		price, _ := float64Value(row["average"])
		result[typeID] = price
	}
	custom, err := queryMaps(ctx, db, `
		SELECT DISTINCT ON (type_id) type_id, price
		FROM custom_prices
		WHERE type_id = ANY($1::int[])
		ORDER BY type_id, ABS(date - $2::date), date DESC`, typeIDs, killTime)
	if err != nil {
		return nil, err
	}
	for _, row := range custom {
		typeID, _ := int64Value(row["type_id"])
		price, _ := float64Value(row["price"])
		result[typeID] = price
	}
	return result, nil
}
