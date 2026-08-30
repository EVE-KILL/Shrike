package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"
)

func loadEntityPageIntel(
	ctx context.Context,
	opts Options,
	kind string,
	id int64,
	req *legacyRequest,
) (any, error) {
	if kind == entityPageCharacter {
		days := min(max(entityPageQueryInt(req.Query.Get("days"), 90), 7), 365)
		return loadCharacterIntel(ctx, opts, id, days)
	}
	if kind != entityPageCorporation && kind != entityPageAlliance {
		return nil, apiError(http.StatusBadRequest, "Invalid entity type")
	}

	var active map[string]any
	var graph map[string]any
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		column := "corporation_id"
		if kind == entityPageAlliance {
			column = "alliance_id"
		}
		var err error
		active, err = queryMap(groupCtx, opts.DB, fmt.Sprintf(`
			SELECT COUNT(*) FILTER (
			         WHERE last_active >= NOW() - INTERVAL '7 days'
			       )::bigint AS days_7,
			       COUNT(*) FILTER (
			         WHERE last_active >= NOW() - INTERVAL '30 days'
			       )::bigint AS days_30,
			       COUNT(*) FILTER (
			         WHERE last_active >= NOW() - INTERVAL '90 days'
			       )::bigint AS days_90
			FROM characters WHERE %s = $1`, column), id)
		return err
	})
	group.Go(func() error {
		graph = loadOrganizationGraphIntel(groupCtx, opts, kind, id)
		return nil
	})
	if err := group.Wait(); err != nil {
		return nil, err
	}
	if graph == nil {
		graph = emptyOrganizationIntel(kind)
	}
	graph["activeMembers"] = map[string]any{
		"days_7":  int64OrZero(active["days_7"]),
		"days_30": int64OrZero(active["days_30"]),
		"days_90": int64OrZero(active["days_90"]),
	}
	return graph, nil
}

type organizationIntelConfig struct {
	Property     string
	PeerProperty string
	PeerTable    string
	PeerID       string
	PeerFallback string
}

var organizationIntelConfigs = map[string]organizationIntelConfig{
	entityPageCorporation: {
		Property: "corporation_id", PeerProperty: "corporation_id",
		PeerTable: "corporations", PeerID: "corporation_id",
		PeerFallback: "Corp",
	},
	entityPageAlliance: {
		Property: "alliance_id", PeerProperty: "alliance_id",
		PeerTable: "alliances", PeerID: "alliance_id",
		PeerFallback: "Alliance",
	},
}

func emptyOrganizationIntel(kind string) map[string]any {
	census := map[string]any{
		"total": int64(0), "fcs": int64(0), "logis": int64(0),
		"caps": int64(0), "supers": int64(0), "droppers": int64(0),
	}
	if kind == entityPageAlliance {
		census["corps"] = []map[string]any{}
	}
	return map[string]any{
		"allies":           []map[string]any{},
		"enemies":          []map[string]any{},
		"huntingGrounds":   []map[string]any{},
		"census":           census,
		"recentDepartures": []map[string]any{},
		"recentJoins":      []map[string]any{},
	}
}

func loadOrganizationGraphIntel(
	ctx context.Context,
	opts Options,
	kind string,
	id int64,
) map[string]any {
	config, ok := organizationIntelConfigs[kind]
	if !ok || opts.Graph == nil {
		return emptyOrganizationIntel(kind)
	}
	cutoff7 := time.Now().UTC().AddDate(0, 0, -7).
		Format("2006-01-02T15:04:05.000Z")
	cutoff90 := time.Now().UTC().AddDate(0, 0, -90).
		Format("2006-01-02T15:04:05.000Z")
	params := map[string]any{"id": id}

	outboundSQL := fmt.Sprintf(`
		MATCH (atk:Character)-[r:KILLED]->(vic:Character)
		WHERE atk.%[1]s = $id AND vic.%[1]s IS NOT NULL
		  AND vic.%[1]s <> $id
		WITH vic.%[1]s AS other, sum(r.weight) AS kills,
		     sum(r.isk_destroyed) AS isk
		WHERE kills >= 10
		RETURN other, kills, isk`, config.Property)
	inboundSQL := fmt.Sprintf(`
		MATCH (atk:Character)-[r:KILLED]->(vic:Character)
		WHERE vic.%[1]s = $id AND atk.%[1]s IS NOT NULL
		  AND atk.%[1]s <> $id
		WITH atk.%[1]s AS other, sum(r.weight) AS kills,
		     sum(r.isk_destroyed) AS isk
		WHERE kills >= 10
		RETURN other, kills, isk`, config.Property)
	huntingSQL := fmt.Sprintf(`
		MATCH (c:Character)-[r:OPERATED_IN]->(s:SolarSystem)
		WHERE c.%s = $id AND r.last_seen > $cutoff
		WITH s.id AS system_id, count(DISTINCT c) AS active_chars,
		     max(r.last_seen) AS latest
		RETURN system_id, active_chars, latest
		ORDER BY active_chars DESC
		LIMIT 10`, config.Property)
	censusSQL := `
		MATCH (c:Character)-[:MEMBER_OF]->(corp:Corporation {id: $id})
		RETURN count(c) AS total,
		       sum(CASE WHEN c.last_fc_seen > $cutoff THEN 1 ELSE 0 END) AS fcs,
		       sum(CASE WHEN c.last_logi_seen > $cutoff THEN 1 ELSE 0 END) AS logis,
		       sum(CASE WHEN c.last_capital_kill > $cutoff THEN 1 ELSE 0 END) AS caps,
		       sum(CASE WHEN c.last_super_kill > $cutoff THEN 1 ELSE 0 END) AS supers,
		       sum(CASE WHEN c.last_blops_seen > $cutoff THEN 1 ELSE 0 END) AS droppers`
	departedSQL := `
		MATCH (c:Character)-[r:MEMBER_OF]->(corp:Corporation {id: $id})
		WHERE c.corporation_id <> $id
		RETURN c.id AS char_id, c.corporation_id AS current_corp,
		       r.last_seen AS left_at
		ORDER BY r.last_seen DESC
		LIMIT 10`
	joinedSQL := `
		MATCH (c:Character)-[r:MEMBER_OF]->(corp:Corporation {id: $id})
		WHERE c.corporation_id = $id
		MATCH (c)-[r2:MEMBER_OF]->(other:Corporation)
		WHERE other.id <> $id
		WITH DISTINCT c, r, r2, other
		ORDER BY r.first_seen DESC
		RETURN c.id AS char_id, r.first_seen AS joined_at,
		       other.id AS prev_corp
		LIMIT 10`
	if kind == entityPageAlliance {
		censusSQL = `
			MATCH (c:Character)-[:MEMBER_OF]->(corp:Corporation)
			WHERE corp.alliance_id = $id
			RETURN corp.id AS corp_id, count(c) AS total,
			       sum(CASE WHEN c.last_fc_seen > $cutoff THEN 1 ELSE 0 END) AS fcs,
			       sum(CASE WHEN c.last_logi_seen > $cutoff THEN 1 ELSE 0 END) AS logis,
			       sum(CASE WHEN c.last_capital_kill > $cutoff THEN 1 ELSE 0 END) AS caps,
			       sum(CASE WHEN c.last_super_kill > $cutoff THEN 1 ELSE 0 END) AS supers,
			       sum(CASE WHEN c.last_blops_seen > $cutoff THEN 1 ELSE 0 END) AS droppers
			ORDER BY total DESC`
		departedSQL = `
			MATCH (c:Character)-[r:MEMBER_OF]->(corp:Corporation)
			WHERE corp.alliance_id = $id AND c.alliance_id <> $id
			RETURN c.id AS char_id, c.corporation_id AS current_corp,
			       r.last_seen AS left_at
			ORDER BY r.last_seen DESC
			LIMIT 10`
		joinedSQL = `
			MATCH (c:Character)-[r:MEMBER_OF]->(corp:Corporation)
			WHERE corp.alliance_id = $id AND c.alliance_id = $id
			MATCH (c)-[r2:MEMBER_OF]->(other:Corporation)
			WHERE other.alliance_id IS NULL OR other.alliance_id <> $id
			WITH DISTINCT c, r
			ORDER BY r.first_seen DESC
			RETURN c.id AS char_id, r.first_seen AS joined_at
			LIMIT 10`
	}

	queries := []struct {
		sql    string
		params map[string]any
	}{
		{outboundSQL, params}, {inboundSQL, params},
		{huntingSQL, map[string]any{"id": id, "cutoff": cutoff7}},
		{censusSQL, map[string]any{"id": id, "cutoff": cutoff90}},
		{departedSQL, params}, {joinedSQL, params},
	}
	results := make([][]map[string]any, len(queries))
	group, groupCtx := errgroup.WithContext(ctx)
	for i, query := range queries {
		group.Go(func() error {
			var err error
			results[i], err = opts.Graph.Read(groupCtx, query.sql, query.params)
			return err
		})
	}
	if group.Wait() != nil {
		return emptyOrganizationIntel(kind)
	}

	targetKills := graphCountMap(results[0], "other", "kills")
	targetLosses := graphCountMap(results[1], "other", "kills")
	mutual := make(map[int64]int64, len(targetKills)+len(targetLosses))
	for target, kills := range targetKills {
		mutual[target] += kills
	}
	for target, losses := range targetLosses {
		mutual[target] += losses
	}
	enemyIDs := []int64{}
	for target, kills := range targetKills {
		if kills >= 50 {
			enemyIDs = append(enemyIDs, target)
		}
	}

	sharedScores := map[int64]int64{}
	if len(enemyIDs) > 0 {
		sharedSQL := fmt.Sprintf(`
			MATCH (atk:Character)-[r:KILLED]->(vic:Character)
			WHERE vic.%[1]s IN $enemies
			  AND atk.%[1]s IS NOT NULL AND atk.%[1]s <> $id
			WITH atk.%[1]s AS other, vic.%[1]s AS enemy,
			     sum(r.weight) AS kills
			WHERE kills >= 20
			RETURN other, enemy, kills`, config.PeerProperty)
		sharedRows, err := opts.Graph.Read(ctx, sharedSQL, map[string]any{
			"enemies": enemyIDs, "id": id,
		})
		if err != nil {
			return emptyOrganizationIntel(kind)
		}
		for _, row := range sharedRows {
			other := int64OrZero(row["other"])
			enemy := int64OrZero(row["enemy"])
			kills := int64OrZero(row["kills"])
			sharedScores[other] += minInt64(targetKills[enemy], kills)
		}
	}

	allies := []map[string]any{}
	for other, shared := range sharedScores {
		if shared < 200 {
			continue
		}
		mutualKills := mutual[other]
		ratio := math.Inf(1)
		if mutualKills > 0 {
			ratio = float64(shared) / float64(mutualKills)
		}
		if ratio >= 4 {
			allies = append(allies, map[string]any{
				"id": other, "shared_enemy_kills": shared,
				"mutual_kills": mutualKills,
			})
		}
	}
	sort.Slice(allies, func(i, j int) bool {
		return int64OrZero(allies[i]["shared_enemy_kills"]) >
			int64OrZero(allies[j]["shared_enemy_kills"])
	})
	if len(allies) > 10 {
		allies = allies[:10]
	}
	enemies := []map[string]any{}
	for other, total := range mutual {
		if total < 50 {
			continue
		}
		enemies = append(enemies, map[string]any{
			"id": other, "kills_given": targetKills[other],
			"kills_taken": targetLosses[other], "total": total,
		})
	}
	sort.Slice(enemies, func(i, j int) bool {
		return int64OrZero(enemies[i]["total"]) > int64OrZero(enemies[j]["total"])
	})
	if len(enemies) > 10 {
		enemies = enemies[:10]
	}

	peerIDs := map[int64]bool{}
	corpIDs := map[int64]bool{}
	charIDs := map[int64]bool{}
	systemIDs := map[int64]bool{}
	for _, list := range [][]map[string]any{allies, enemies} {
		for _, row := range list {
			peerIDs[int64OrZero(row["id"])] = true
		}
	}
	for _, row := range results[2] {
		systemIDs[int64OrZero(row["system_id"])] = true
	}
	if kind == entityPageAlliance {
		for _, row := range results[3] {
			corpIDs[int64OrZero(row["corp_id"])] = true
		}
	}
	for _, row := range results[4] {
		charIDs[int64OrZero(row["char_id"])] = true
		corpIDs[int64OrZero(row["current_corp"])] = true
	}
	for _, row := range results[5] {
		charIDs[int64OrZero(row["char_id"])] = true
		if kind == entityPageCorporation {
			corpIDs[int64OrZero(row["prev_corp"])] = true
		}
	}
	nameResults, err := resolveOrganizationIntelNames(
		ctx, opts.DB, config, peerIDs, corpIDs, charIDs, systemIDs,
	)
	if err != nil {
		return emptyOrganizationIntel(kind)
	}
	peerNames, corpNames := nameResults[0], nameResults[1]
	charNames, systemNames := nameResults[2], nameResults[3]
	for _, row := range allies {
		other := int64OrZero(row["id"])
		row["name"] = fallbackName(peerNames[other], config.PeerFallback, other)
	}
	for _, row := range enemies {
		other := int64OrZero(row["id"])
		row["name"] = fallbackName(peerNames[other], config.PeerFallback, other)
	}
	hunting := make([]map[string]any, 0, len(results[2]))
	for _, row := range results[2] {
		systemID := int64OrZero(row["system_id"])
		hunting = append(hunting, map[string]any{
			"id":                systemID,
			"name":              fallbackName(systemNames[systemID], "System", systemID),
			"active_characters": int64OrZero(row["active_chars"]),
		})
	}
	census := map[string]any{
		"total": int64(0), "fcs": int64(0), "logis": int64(0),
		"caps": int64(0), "supers": int64(0), "droppers": int64(0),
	}
	for _, row := range results[3] {
		for _, key := range []string{"total", "fcs", "logis", "caps", "supers", "droppers"} {
			census[key] = int64OrZero(census[key]) + int64OrZero(row[key])
		}
	}
	departures := make([]map[string]any, 0, len(results[4]))
	for _, row := range results[4] {
		charID := int64OrZero(row["char_id"])
		corpID := int64OrZero(row["current_corp"])
		departures = append(departures, map[string]any{
			"id":   charID,
			"name": fallbackName(charNames[charID], "Character", charID),
			"current_corp": map[string]any{
				"id": corpID, "name": nilIfEmpty(corpNames[corpID]),
			},
			"left_at": row["left_at"],
		})
	}
	joins := make([]map[string]any, 0, len(results[5]))
	for _, row := range results[5] {
		charID := int64OrZero(row["char_id"])
		item := map[string]any{
			"id":        charID,
			"name":      fallbackName(charNames[charID], "Character", charID),
			"joined_at": row["joined_at"],
		}
		if kind == entityPageCorporation {
			corpID := int64OrZero(row["prev_corp"])
			item["previous_corp"] = map[string]any{
				"id": corpID, "name": nilIfEmpty(corpNames[corpID]),
			}
		}
		joins = append(joins, item)
	}
	return map[string]any{
		"allies": allies, "enemies": enemies, "huntingGrounds": hunting,
		"census": census, "recentDepartures": departures, "recentJoins": joins,
	}
}

func graphCountMap(rows []map[string]any, idKey, countKey string) map[int64]int64 {
	out := make(map[int64]int64, len(rows))
	for _, row := range rows {
		out[int64OrZero(row[idKey])] = int64OrZero(row[countKey])
	}
	return out
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func resolveOrganizationIntelNames(
	ctx context.Context,
	db Database,
	config organizationIntelConfig,
	peers, corps, characters, systems map[int64]bool,
) ([]map[int64]string, error) {
	sets := []struct {
		table, id, name string
		values          map[int64]bool
	}{
		{config.PeerTable, config.PeerID, "name", peers},
		{"corporations", "corporation_id", "name", corps},
		{"characters", "character_id", "name", characters},
		{"solar_systems", "solar_system_id", "system_name", systems},
	}
	queries := make([]databaseQuery, len(sets))
	for i, set := range sets {
		ids := sortedMapKeys(set.values)
		if len(ids) == 0 {
			queries[i] = databaseQuery{
				SQL: "SELECT NULL::bigint AS id, NULL::text AS name WHERE false",
			}
			continue
		}
		queries[i] = databaseQuery{
			SQL: fmt.Sprintf(
				"SELECT %s::bigint AS id, %s AS name FROM %s WHERE %s = ANY($1::int[])",
				set.id, set.name, set.table, set.id,
			),
			Args: []any{int64sToInt32(ids)},
		}
	}
	rows, err := queryMapsConcurrent(ctx, db, queries...)
	if err != nil {
		return nil, err
	}
	result := make([]map[int64]string, len(rows))
	for i, list := range rows {
		result[i] = make(map[int64]string, len(list))
		for _, row := range list {
			result[i][int64OrZero(row["id"])] = stringOrEmpty(row["name"])
		}
	}
	return result, nil
}

func int64sToInt32(values []int64) []int32 {
	out := make([]int32, 0, len(values))
	for _, value := range values {
		if value > 0 && value <= math.MaxInt32 {
			out = append(out, int32(value))
		}
	}
	return out
}

func fallbackName(value, prefix string, id int64) string {
	if value != "" {
		return value
	}
	return fmt.Sprintf("%s %d", prefix, id)
}

func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
