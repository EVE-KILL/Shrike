package api

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

type graphIntel struct {
	FleetPartners   []map[string]any
	GroupsFlownWith []map[string]any
	BridgeScore     int64
	Timestamps      map[string]string
}

func registerCharacterIntelRoute(a huma.API, opts Options) {
	registerLegacy(a, entityIDOperation(
		"character-intel", "/characters/{id}/intel",
		"Character intelligence profile", "characters",
	), func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		days := int(math.Min(365, math.Max(
			7, numberOr(req.Query.Get("days"), 365),
		)))
		found, err := queryMap(ctx, opts.DB,
			`SELECT character_id FROM characters
			 WHERE character_id = $1 LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if found == nil {
			return foundOr404(nil, "Character not found"), nil
		}
		body, err := loadCharacterIntel(ctx, opts, id, days)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(body), nil
	})
}

func loadCharacterIntel(
	ctx context.Context,
	opts Options,
	id int64,
	days int,
) (map[string]any, error) {
	playstyle, err := queryMap(ctx, opts.DB, `
		SELECT count(*) FILTER (WHERE k.attacker_count = 1) AS solo,
		       count(*) FILTER (WHERE k.attacker_count BETWEEN 2 AND 5) AS small_gang,
		       count(*) FILTER (WHERE k.attacker_count BETWEEN 6 AND 15) AS mid_gang,
		       count(*) FILTER (WHERE k.attacker_count BETWEEN 16 AND 50) AS fleet,
		       count(*) FILTER (WHERE k.attacker_count > 50) AS blob,
		       count(*) AS total,
		       round(avg(k.attacker_count)::numeric, 1)::double precision AS avg_fleet_size,
		       count(*) FILTER (WHERE a.ship_type_id = 45534) AS monitor_appearances,
		       count(*) FILTER (
		         WHERE a.damage_done = 0 AND k.attacker_count >= 10
		       ) AS zero_dmg_fleet,
		       count(*) AS total_appearances
		FROM killmail_attackers a
		JOIN killmails k ON k.killmail_id = a.killmail_id
		WHERE a.character_id = $1
		  AND a.killmail_time > now() - make_interval(days => $2)`, id, days)
	if err != nil {
		return nil, err
	}
	shipsFlown, err := queryMaps(ctx, opts.DB, `
		SELECT a.ship_type_id, t.name AS ship_name, count(*)::int AS count
		FROM killmail_attackers a
		JOIN inv_types t ON t.type_id = a.ship_type_id
		WHERE a.character_id = $1
		  AND a.killmail_time > now() - make_interval(days => $2)
		  AND a.ship_type_id IS NOT NULL
		GROUP BY a.ship_type_id, t.name ORDER BY count DESC LIMIT 5`, id, days)
	if err != nil {
		return nil, err
	}
	shipsLost, err := queryMaps(ctx, opts.DB, `
		SELECT k.victim_ship_type_id AS ship_type_id,
		       t.name AS ship_name, count(*)::int AS count
		FROM killmails k
		JOIN inv_types t ON t.type_id = k.victim_ship_type_id
		WHERE k.victim_character_id = $1
		  AND k.killmail_time > now() - make_interval(days => $2)
		  AND k.victim_ship_type_id IS NOT NULL
		GROUP BY k.victim_ship_type_id, t.name
		ORDER BY count DESC LIMIT 5`, id, days)
	if err != nil {
		return nil, err
	}
	targets, err := queryMaps(ctx, opts.DB, `
		SELECT k.victim_alliance_id AS alliance_id,
		       a.name AS alliance_name, count(*)::int AS count
		FROM killmail_attackers atk
		JOIN killmails k ON k.killmail_id = atk.killmail_id
		LEFT JOIN alliances a ON a.alliance_id = k.victim_alliance_id
		WHERE atk.character_id = $1
		  AND atk.killmail_time > now() - make_interval(days => $2)
		  AND k.victim_alliance_id IS NOT NULL
		GROUP BY k.victim_alliance_id, a.name
		ORDER BY count DESC LIMIT 10`, id, days)
	if err != nil {
		return nil, err
	}
	awox, err := queryMap(ctx, opts.DB, `
		SELECT count(*)::int AS awox_kills
		FROM killmail_attackers atk
		JOIN killmails k ON k.killmail_id = atk.killmail_id
		WHERE atk.character_id = $1
		  AND atk.killmail_time > now() - make_interval(days => $2)
		  AND atk.alliance_id IS NOT NULL
		  AND atk.alliance_id = k.victim_alliance_id
		  AND atk.corporation_id != k.victim_corporation_id`, id, days)
	if err != nil {
		return nil, err
	}
	bait, err := queryMap(ctx, opts.DB, `
		WITH cheap_deaths AS (
		  SELECT killmail_id, killmail_time, solar_system_id FROM killmails
		  WHERE victim_character_id = $1
		    AND killmail_time > now() - make_interval(days => $2)
		    AND total_value < 50000000
		),
		baited AS (
		  SELECT DISTINCT cd.killmail_id FROM cheap_deaths cd
		  JOIN killmails followup
		    ON followup.solar_system_id = cd.solar_system_id
		   AND followup.killmail_time BETWEEN cd.killmail_time
		       AND cd.killmail_time + interval '5 minutes'
		   AND followup.killmail_id != cd.killmail_id
		   AND followup.attacker_count >= 2
		)
		SELECT (SELECT count(*)::int FROM cheap_deaths) AS cheap_deaths,
		       (SELECT count(*)::int FROM baited) AS baited_deaths`, id, days)
	if err != nil {
		return nil, err
	}
	archetype, err := queryMap(ctx, opts.DB, `
		WITH atk_kms AS (
		  SELECT s.security, atk.security_status
		  FROM killmail_attackers atk
		  JOIN killmails k ON k.killmail_id = atk.killmail_id
		  JOIN solar_systems s ON s.solar_system_id = k.solar_system_id
		  WHERE atk.character_id = $1
		    AND atk.killmail_time > now() - interval '90 days'
		), cyno_counts AS (
		  SELECT count(DISTINCT k.killmail_id) FILTER (
		           WHERE k.killmail_time > now() - interval '90 days'
		         )::int AS cyno_losses_90d,
		         count(DISTINCT k.killmail_id) FILTER (
		           WHERE k.killmail_time > now() - make_interval(days => $2)
		         )::int AS cyno_deaths
		  FROM killmails k
		  JOIN killmail_items ki ON ki.killmail_id = k.killmail_id
		  WHERE k.victim_character_id = $1
		    AND k.killmail_time > now() - make_interval(days => greatest($2, 90))
		    AND ki.type_id IN (21096, 28646, 52694)
		    AND ki.flag_id BETWEEN 11 AND 34
		)
		SELECT count(*) FILTER (WHERE security >= 0.5)::int AS hs_kms,
		       count(*) FILTER (WHERE security > 0 AND security < 0.5)::int AS ls_kms,
		       count(*) FILTER (WHERE security <= 0)::int AS ns_kms,
		       count(*)::int AS total_kms_90d,
		       count(*) FILTER (
		         WHERE security >= 0.5 AND security_status < -5.0
		       )::int AS gank_kms,
		       (SELECT min(killmail_time)::text FROM killmail_attackers
		         WHERE character_id = $1) AS first_km_time,
		       (SELECT count(*)::int FROM killmails
		         WHERE victim_character_id = $1
		           AND killmail_time > now() - interval '90 days') AS losses_90d,
		       (SELECT coalesce(
		           birthday > now() - interval '180 days', false
		         ) FROM characters WHERE character_id = $1) AS is_new_char,
		       max(cyno_counts.cyno_losses_90d) AS cyno_losses_90d,
		       max(cyno_counts.cyno_deaths) AS cyno_deaths
		FROM atk_kms CROSS JOIN cyno_counts`, id, days)
	if err != nil {
		return nil, err
	}
	graph := loadGraphIntel(ctx, opts.Graph, id)
	return finishCharacterIntel(ctx, opts.DB, id, days, playstyle, shipsFlown,
		shipsLost, targets, awox, bait, archetype, graph)
}

func finishCharacterIntel(
	ctx context.Context, db Database, id int64, days int,
	playstyle map[string]any, shipsFlown, shipsLost, targets []map[string]any,
	awox, bait, archetype map[string]any, graph graphIntel,
) (map[string]any, error) {
	allianceIDs := append(
		graphFieldIDs(graph.FleetPartners, "alliance_id"),
		graphFieldIDs(graph.GroupsFlownWith, "alliance_id")...,
	)
	charNames, corpNames, allianceNames, err := intelEntityNames(
		ctx, db,
		graphFieldIDs(graph.FleetPartners, "id"),
		graphFieldIDs(graph.FleetPartners, "corp_id"),
		allianceIDs,
	)
	if err != nil {
		return nil, err
	}
	return assembleCharacterIntel(id, days, playstyle, shipsFlown, shipsLost,
		targets, awox, bait, archetype, graph, charNames, corpNames, allianceNames), nil
}

func assembleCharacterIntel(
	id int64, days int,
	playstyle map[string]any, shipsFlown, shipsLost, targets []map[string]any,
	awox, bait, archetype map[string]any, graph graphIntel,
	charNames, corpNames, allianceNames map[int64]string,
) map[string]any {
	total, _ := int64Value(playstyle["total"])
	percentDenominator := total
	if percentDenominator == 0 {
		percentDenominator = 1
	}
	playstyleOutput := map[string]any{
		"solo":           percentageField(playstyle["solo"], percentDenominator),
		"small_gang":     percentageField(playstyle["small_gang"], percentDenominator),
		"mid_gang":       percentageField(playstyle["mid_gang"], percentDenominator),
		"fleet":          percentageField(playstyle["fleet"], percentDenominator),
		"blob":           percentageField(playstyle["blob"], percentDenominator),
		"avg_fleet_size": numberField(playstyle["avg_fleet_size"]),
		"total_kills":    total,
	}
	dominant := "Solo"
	maxPercent, _ := int64Value(playstyleOutput["solo"])
	for _, style := range []struct {
		Label string
		Key   string
	}{
		{"Small Gang", "small_gang"}, {"Mid Gang", "mid_gang"},
		{"Fleet", "fleet"}, {"Blob", "blob"},
	} {
		value, _ := int64Value(playstyleOutput[style.Key])
		if value > maxPercent {
			maxPercent, dominant = value, style.Label
		}
	}

	monitor, _ := int64Value(playstyle["monitor_appearances"])
	zeroDamage, _ := int64Value(playstyle["zero_dmg_fleet"])
	appearances, _ := int64Value(playstyle["total_appearances"])
	if appearances == 0 {
		appearances = 1
	}
	likelihood := "None"
	switch {
	case monitor >= 3:
		likelihood = "High"
	case monitor >= 1 || float64(zeroDamage)/float64(appearances) > 0.3:
		likelihood = "Medium"
	case float64(zeroDamage)/float64(appearances) > 0.15:
		likelihood = "Low"
	}
	tags := deriveIntelTags(graph.Timestamps, archetype)

	baited, _ := int64Value(bait["baited_deaths"])
	awoxKills, _ := int64Value(awox["awox_kills"])
	cynoDeaths, _ := int64Value(archetype["cyno_deaths"])
	return map[string]any{
		"character_id": id, "days": days,
		"playstyle": playstyleOutput, "dominant_style": dominant,
		"tags": tags,
		"fc": map[string]any{
			"likelihood": likelihood, "monitor_appearances": monitor,
		},
		"capital_pilot": containsString(tags, "CAPITAL"),
		"is_logi":       containsString(tags, "LOGI"),
		"ships_flown":   intelCountList(shipsFlown, "ship_type_id", "ship_name"),
		"ships_lost":    intelCountList(shipsLost, "ship_type_id", "ship_name"),
		"targets":       intelTargets(targets),
		"fleet_partners": intelFleetPartners(
			graph.FleetPartners, charNames, corpNames, allianceNames,
		),
		"groups_flown_with": intelGroups(graph.GroupsFlownWith, allianceNames),
		"awox_kills":        awoxKills, "cyno_deaths": cynoDeaths,
		"bait": intelBaitLevel(baited), "bait_count": baited,
		"bridge_score": graph.BridgeScore,
	}
}

func loadGraphIntel(
	ctx context.Context,
	graph GraphDatabase,
	id int64,
) graphIntel {
	empty := graphIntel{Timestamps: emptyGraphTimestamps()}
	if graph == nil {
		return empty
	}
	partners, err := graph.Read(ctx, `
		MATCH (c:Character {id: $id})-[r:FLEW_WITH]-(p:Character)
		RETURN p.id AS id, p.corporation_id AS corp_id,
		       p.alliance_id AS alliance_id, r.weight AS weight
		ORDER BY r.weight DESC LIMIT 10`, map[string]any{"id": id})
	if err != nil {
		return empty
	}
	groupRows, err := graph.Read(ctx, `
		MATCH (c:Character {id: $id})-[:FLEW_WITH]-(p:Character)
		WHERE p.alliance_id IS NOT NULL
		WITH p.alliance_id AS alliance_id,
		     count(DISTINCT p) AS shared_partners
		ORDER BY shared_partners DESC
		WITH collect({
		  alliance_id: alliance_id,
		  shared_partners: shared_partners
		}) AS groups
		RETURN size(groups) AS cnt, groups[..10] AS groups`, map[string]any{"id": id})
	if err != nil {
		return empty
	}
	timestampRows, err := graph.Read(ctx, `
		MATCH (c:Character {id: $id})
		RETURN c.last_fc_seen AS last_fc_seen,
		       c.last_super_kill AS last_super_kill,
		       c.last_blops_seen AS last_blops_seen,
		       c.last_capital_kill AS last_capital_kill,
		       c.last_logi_seen AS last_logi_seen`, map[string]any{"id": id})
	if err != nil {
		return empty
	}
	timestamps := emptyGraphTimestamps()
	if len(timestampRows) > 0 {
		for key := range timestamps {
			timestamps[key] = intelTimestamp(timestampRows[0][key])
		}
	}
	score := int64(0)
	groups := []map[string]any{}
	if len(groupRows) > 0 {
		score, _ = int64Value(groupRows[0]["cnt"])
		groups = graphMapList(groupRows[0]["groups"])
	}
	return graphIntel{
		FleetPartners: partners, GroupsFlownWith: groups,
		BridgeScore: score, Timestamps: timestamps,
	}
}

func graphMapList(value any) []map[string]any {
	values, ok := value.([]any)
	if !ok {
		return []map[string]any{}
	}
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if row, ok := value.(map[string]any); ok {
			result = append(result, row)
		}
	}
	return result
}

func emptyGraphTimestamps() map[string]string {
	return map[string]string{
		"last_fc_seen": "", "last_super_kill": "", "last_blops_seen": "",
		"last_capital_kill": "", "last_logi_seen": "",
	}
}

func intelTimestamp(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case time.Time:
		return javascriptTimestamp(v)
	default:
		return ""
	}
}

func deriveIntelTags(timestamps map[string]string, counters map[string]any) []string {
	cutoff := javascriptTimestamp(time.Now().AddDate(0, 0, -90))
	tags := []string{}
	for _, tag := range []struct {
		Key string
		Tag string
	}{
		{"last_fc_seen", "FC"}, {"last_super_kill", "SUPER"},
		{"last_blops_seen", "DROPPER"}, {"last_capital_kill", "CAPITAL"},
		{"last_logi_seen", "LOGI"},
	} {
		if timestamps[tag.Key] != "" && timestamps[tag.Key] > cutoff {
			tags = append(tags, tag.Tag)
		}
	}
	total, _ := int64Value(counters["total_kms_90d"])
	highsec, _ := int64Value(counters["hs_kms"])
	losses, _ := int64Value(counters["losses_90d"])
	ganks, _ := int64Value(counters["gank_kms"])
	cynos, _ := int64Value(counters["cyno_losses_90d"])
	isNew, _ := boolValue(counters["is_new_char"])
	if total >= 10 && float64(highsec)/float64(total) >= 0.75 {
		tags = append(tags, "HIGHSEC")
	}
	if isNew && losses > total &&
		!containsString(tags, "CAPITAL") && !containsString(tags, "SUPER") {
		tags = append(tags, "NEWBIE")
	}
	if ganks >= 3 {
		tags = append(tags, "GANKER")
	}
	if cynos >= 2 {
		tags = append(tags, "CYNO")
	}
	return tags
}

func percentageField(value any, total int64) int64 {
	number, _ := int64Value(value)
	return int64(math.Round(float64(number) / float64(total) * 100))
}

func numberField(value any) float64 {
	number, _ := float64Value(value)
	return number
}

func intelCountList(rows []map[string]any, idKey, nameKey string) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{
			"id": row[idKey], "name": row[nameKey], "count": row["count"],
		})
	}
	return result
}

func intelTargets(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id, _ := int64Value(row["alliance_id"])
		name, _ := stringValue(row["alliance_name"])
		if name == "" {
			name = fmt.Sprintf("Alliance %d", id)
		}
		result = append(result, map[string]any{
			"id": id, "name": name, "count": row["count"],
		})
	}
	return result
}

func graphFieldIDs(rows []map[string]any, field string) []int32 {
	values := make([]any, 0, len(rows))
	for _, row := range rows {
		if row[field] != nil {
			values = append(values, row[field])
		}
	}
	return int32Slice(values...)
}

func intelEntityNames(
	ctx context.Context,
	db Database,
	characterIDs, corporationIDs, allianceIDs []int32,
) (map[int64]string, map[int64]string, map[int64]string, error) {
	characters := map[int64]string{}
	corporations := map[int64]string{}
	alliances := map[int64]string{}
	if len(characterIDs) == 0 && len(corporationIDs) == 0 && len(allianceIDs) == 0 {
		return characters, corporations, alliances, nil
	}
	rows, err := queryMaps(ctx, db, `
		SELECT 'character' AS kind, character_id AS id, name
		FROM characters WHERE character_id = ANY($1::int[])
		UNION ALL
		SELECT 'corporation', corporation_id, name
		FROM corporations WHERE corporation_id = ANY($2::int[])
		UNION ALL
		SELECT 'alliance', alliance_id, name
		FROM alliances WHERE alliance_id = ANY($3::int[])`,
		characterIDs, corporationIDs, allianceIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, row := range rows {
		id, _ := int64Value(row["id"])
		name, _ := stringValue(row["name"])
		kind, _ := stringValue(row["kind"])
		switch kind {
		case "character":
			characters[id] = name
		case "corporation":
			corporations[id] = name
		case "alliance":
			alliances[id] = name
		}
	}
	return characters, corporations, alliances, nil
}

func intelFleetPartners(
	rows []map[string]any,
	charNames, corpNames, allianceNames map[int64]string,
) []map[string]any {
	if len(rows) > 10 {
		rows = rows[:10]
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id, _ := int64Value(row["id"])
		corpID, hasCorp := int64Value(row["corp_id"])
		allianceID, hasAlliance := int64Value(row["alliance_id"])
		name := charNames[id]
		if name == "" {
			name = fmt.Sprintf("Character %d", id)
		}
		var corpName, allianceName any
		if hasCorp {
			if resolved := corpNames[corpID]; resolved != "" {
				corpName = resolved
			}
		}
		if hasAlliance {
			if resolved := allianceNames[allianceID]; resolved != "" {
				allianceName = resolved
			}
		}
		result = append(result, map[string]any{
			"id": id, "name": name, "count": row["weight"],
			"corp_name": corpName, "alliance_name": allianceName,
		})
	}
	return result
}

func intelGroups(
	rows []map[string]any,
	allianceNames map[int64]string,
) []map[string]any {
	if len(rows) > 10 {
		rows = rows[:10]
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id, _ := int64Value(row["alliance_id"])
		name := allianceNames[id]
		if name == "" {
			name = fmt.Sprintf("Alliance %d", id)
		}
		result = append(result, map[string]any{
			"id": id, "name": name, "count": row["shared_partners"],
		})
	}
	return result
}

func intelBaitLevel(count int64) string {
	switch {
	case count >= 10:
		return "High"
	case count >= 5:
		return "Medium"
	case count >= 2:
		return "Low"
	default:
		return "None"
	}
}

func containsString(values []string, expected string) bool {
	return slices.Contains(values, expected)
}
