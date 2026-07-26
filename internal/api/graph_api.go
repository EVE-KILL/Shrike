package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func graphHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		mode := strings.TrimSpace(req.Query.Get("mode"))
		if mode == "" {
			mode = "path_finder"
		}
		switch mode {
		case "path_finder":
			return graphPathFinder(ctx, opts, req, mode)
		case "coalitions":
			return graphCoalitions(ctx, opts, mode)
		case "rivalries":
			return graphRivalries(ctx, opts, req, mode)
		case "entity_intel":
			return graphEntityIntel(ctx, opts, req, mode)
		case "hunting_grounds":
			return graphHuntingGrounds(ctx, opts, req, mode)
		case "hot_zones":
			return graphHotZones(ctx, opts, mode)
		case "migration":
			return graphMigration(ctx, opts, req, mode)
		case "spy_check":
			return graphSpyCheck(ctx, opts, req, mode)
		case "census":
			return graphCensus(ctx, opts, req, mode)
		default:
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Unknown graph mode: "+mode,
			)
		}
	}
}

func graphPathFinder(
	ctx context.Context,
	opts Options,
	req *legacyRequest,
	mode string,
) (legacyPayload, error) {
	from := graphQueryID(req.Query.Get("fromId"))
	to := graphQueryID(req.Query.Get("toId"))
	if from == 0 || to == 0 {
		return jsonPayload(map[string]any{"path": nil, "mode": mode}), nil
	}
	if from == to {
		return jsonPayload(map[string]any{
			"path": nil, "error": "Same character", "mode": mode,
		}), nil
	}
	graph, err := requireGraph(opts)
	if err != nil {
		return legacyPayload{}, err
	}
	rows, err := graph.Read(ctx, `
		MATCH (a:Character {id: $from}), (b:Character {id: $to})
		MATCH path = (a)-[:FLEW_WITH *BFS ..6]-(b)
		RETURN [node IN nodes(path) | node.id] AS node_ids,
		       [edge IN relationships(path) | edge.weight] AS weights
		LIMIT 1`, map[string]any{"from": from, "to": to})
	if err != nil {
		return legacyPayload{}, err
	}
	if len(rows) == 0 {
		return jsonPayload(map[string]any{"path": nil, "mode": mode}), nil
	}
	nodeIDs := graphInt64Slice(rows[0]["node_ids"])
	weights := graphInt64Slice(rows[0]["weights"])
	nodeRows, err := graph.Read(ctx, `
		UNWIND $ids AS cid
		MATCH (node:Character {id: cid})
		RETURN node.id AS id, node.corporation_id AS corp,
		       node.alliance_id AS alliance`,
		map[string]any{"ids": nodeIDs},
	)
	if err != nil {
		return legacyPayload{}, err
	}
	nodesByID := map[int64]map[string]any{}
	corpSet := map[int64]struct{}{}
	allianceSet := map[int64]struct{}{}
	for _, row := range nodeRows {
		id := int64OrZero(row["id"])
		nodesByID[id] = row
		addGraphID(corpSet, row["corp"])
		addGraphID(allianceSet, row["alliance"])
	}
	characterNames, err := loadGraphNames(
		ctx, opts.DB, "character", nodeIDs,
	)
	if err != nil {
		return legacyPayload{}, err
	}
	corporationNames, err := loadGraphNames(
		ctx, opts.DB, "corporation", sortedInt64Set(corpSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	allianceNames, err := loadGraphNames(
		ctx, opts.DB, "alliance", sortedInt64Set(allianceSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}

	nodes := make([]map[string]any, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		row := nodesByID[id]
		corpID := int64OrZero(row["corp"])
		allianceID := int64OrZero(row["alliance"])
		var corporation any
		if corpID != 0 {
			corporation = map[string]any{
				"id": corpID, "name": nilIfEmpty(corporationNames[corpID]),
			}
		}
		var alliance any
		if allianceID != 0 {
			alliance = map[string]any{
				"id": allianceID, "name": nilIfEmpty(allianceNames[allianceID]),
			}
		}
		nodes = append(nodes, map[string]any{
			"id":          id,
			"name":        fallbackName(characterNames[id], "Character", id),
			"corporation": corporation,
			"alliance":    alliance,
		})
	}
	edges := make([]map[string]any, 0, len(weights))
	for _, weight := range weights {
		edges = append(edges, map[string]any{"weight": weight})
	}
	return jsonPayload(map[string]any{
		"path": map[string]any{
			"nodes": nodes, "edges": edges, "hops": len(weights),
		},
		"mode": mode,
	}), nil
}

type graphAlliancePair struct {
	A, B  int64
	Score int64
}

type graphPairKey struct {
	A, B int64
}

func graphCoalitions(
	ctx context.Context,
	opts Options,
	mode string,
) (legacyPayload, error) {
	graph, err := requireGraph(opts)
	if err != nil {
		return legacyPayload{}, err
	}
	rows, err := graph.Read(ctx, `
		MATCH (attacker:Character)-[edge:KILLED]->(victim:Character)
		WHERE attacker.alliance_id IS NOT NULL
		  AND victim.alliance_id IS NOT NULL
		  AND attacker.alliance_id <> victim.alliance_id
		WITH attacker.alliance_id AS attacker,
		     victim.alliance_id AS enemy, sum(edge.weight) AS kills
		WHERE kills >= 50
		RETURN attacker, enemy, kills`, nil)
	if err != nil {
		return legacyPayload{}, err
	}
	profiles := map[int64]map[int64]int64{}
	for _, row := range rows {
		attacker := int64OrZero(row["attacker"])
		enemy := int64OrZero(row["enemy"])
		if attacker == 0 || enemy == 0 {
			continue
		}
		if profiles[attacker] == nil {
			profiles[attacker] = map[int64]int64{}
		}
		profiles[attacker][enemy] = int64OrZero(row["kills"])
	}
	active := []int64{}
	for id, profile := range profiles {
		total := int64(0)
		for _, kills := range profile {
			total += kills
		}
		if total >= 2000 {
			active = append(active, id)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		left, right := graphProfileTotal(profiles[active[i]]),
			graphProfileTotal(profiles[active[j]])
		if left == right {
			return active[i] < active[j]
		}
		return left > right
	})

	allied := []graphAlliancePair{}
	hostile := map[graphPairKey]struct{}{}
	for i, a := range active {
		for _, b := range active[i+1:] {
			profileA, profileB := profiles[a], profiles[b]
			mutual := profileA[b] + profileB[a]
			if mutual == 0 && len(profileA) < 3 && len(profileB) < 3 {
				continue
			}
			enemySet := map[int64]struct{}{}
			for enemy := range profileA {
				enemySet[enemy] = struct{}{}
			}
			for enemy := range profileB {
				enemySet[enemy] = struct{}{}
			}
			delete(enemySet, a)
			delete(enemySet, b)
			shared := int64(0)
			for enemy := range enemySet {
				shared += minInt64(profileA[enemy], profileB[enemy])
			}
			if shared < 500 {
				continue
			}
			ratio := math.Inf(1)
			if mutual > 0 {
				ratio = float64(shared) / float64(mutual)
			}
			if ratio >= 4 {
				allied = append(allied, graphAlliancePair{
					A: a, B: b, Score: shared,
				})
			}
			if ratio < 3 {
				hostile[orderedGraphPair(a, b)] = struct{}{}
			}
		}
	}
	sort.Slice(allied, func(i, j int) bool {
		if allied[i].Score != allied[j].Score {
			return allied[i].Score > allied[j].Score
		}
		if allied[i].A != allied[j].A {
			return allied[i].A < allied[j].A
		}
		return allied[i].B < allied[j].B
	})

	union := newGraphUnion()
	for _, pair := range allied {
		union.add(pair.A)
		union.add(pair.B)
	}
	for _, pair := range allied {
		membersA := union.members(pair.A)
		membersB := union.members(pair.B)
		blocked := false
		for _, a := range membersA {
			for _, b := range membersB {
				if _, exists := hostile[orderedGraphPair(a, b)]; exists {
					blocked = true
					break
				}
			}
			if blocked {
				break
			}
		}
		if !blocked {
			union.join(pair.A, pair.B)
		}
	}
	components := union.components()
	coalitions := make([][]int64, 0, len(components))
	for _, members := range components {
		if len(members) >= 2 {
			coalitions = append(coalitions, members)
		}
	}
	sort.Slice(coalitions, func(i, j int) bool {
		if len(coalitions[i]) != len(coalitions[j]) {
			return len(coalitions[i]) > len(coalitions[j])
		}
		return coalitions[i][0] < coalitions[j][0]
	})
	score := map[int64]int64{}
	for _, pair := range allied {
		score[pair.A] += pair.Score
		score[pair.B] += pair.Score
	}
	allianceSet := map[int64]struct{}{}
	for _, coalition := range coalitions {
		for _, id := range coalition {
			allianceSet[id] = struct{}{}
		}
	}
	names, err := loadGraphNames(
		ctx, opts.DB, "alliance", sortedInt64Set(allianceSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	output := make([]map[string]any, 0, len(coalitions))
	for index, coalition := range coalitions {
		alliances := make([]map[string]any, 0, len(coalition))
		for _, id := range coalition {
			alliances = append(alliances, map[string]any{
				"id": id, "name": fallbackName(names[id], "Alliance", id),
				"cooperation": score[id],
			})
		}
		sort.Slice(alliances, func(i, j int) bool {
			left := int64OrZero(alliances[i]["cooperation"])
			right := int64OrZero(alliances[j]["cooperation"])
			if left == right {
				return int64OrZero(alliances[i]["id"]) <
					int64OrZero(alliances[j]["id"])
			}
			return left > right
		})
		output = append(output, map[string]any{
			"id": index + 1, "alliances": alliances,
		})
	}
	return jsonPayload(map[string]any{
		"coalitions": output, "mode": mode,
	}), nil
}

func graphRivalries(
	ctx context.Context,
	opts Options,
	req *legacyRequest,
	mode string,
) (legacyPayload, error) {
	graph, err := requireGraph(opts)
	if err != nil {
		return legacyPayload{}, err
	}
	entityType := strings.TrimSpace(req.Query.Get("entityType"))
	if entityType == "" {
		entityType = "alliance"
	}
	limit := boundedQueryInt(req, "limit", 50, 10, 100)
	if entityType == "character" {
		rows, err := graph.Read(ctx, `
			MATCH (a:Character)-[first:KILLED]->(b:Character),
			      (b)-[second:KILLED]->(a)
			WHERE a.id < b.id
			  AND (
			    a.alliance_id IS NULL OR b.alliance_id IS NULL
			    OR a.alliance_id <> b.alliance_id
			  )
			RETURN a.id AS char_a, a.corporation_id AS corp_a,
			       a.alliance_id AS alliance_a,
			       b.id AS char_b, b.corporation_id AS corp_b,
			       b.alliance_id AS alliance_b,
			       first.weight + second.weight AS mutual_kills,
			       first.isk_destroyed + second.isk_destroyed AS total_isk
			ORDER BY mutual_kills DESC
			LIMIT $limit`, map[string]any{"limit": int64(limit)})
		if err != nil {
			return legacyPayload{}, err
		}
		characters, corporations, alliances := graphRivalryIDs(rows)
		charNames, err := loadGraphNames(ctx, opts.DB, "character", characters)
		if err != nil {
			return legacyPayload{}, err
		}
		corpNames, err := loadGraphNames(ctx, opts.DB, "corporation", corporations)
		if err != nil {
			return legacyPayload{}, err
		}
		allianceNames, err := loadGraphNames(ctx, opts.DB, "alliance", alliances)
		if err != nil {
			return legacyPayload{}, err
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			a, b := int64OrZero(row["char_a"]), int64OrZero(row["char_b"])
			aCorp, bCorp := int64OrZero(row["corp_a"]), int64OrZero(row["corp_b"])
			aAlliance := int64OrZero(row["alliance_a"])
			bAlliance := int64OrZero(row["alliance_b"])
			items = append(items, map[string]any{
				"entity_a": map[string]any{
					"id": a, "name": fallbackName(charNames[a], "Character", a),
					"corp_name":     nilIfEmpty(corpNames[aCorp]),
					"alliance_name": nilIfEmpty(allianceNames[aAlliance]),
				},
				"entity_b": map[string]any{
					"id": b, "name": fallbackName(charNames[b], "Character", b),
					"corp_name":     nilIfEmpty(corpNames[bCorp]),
					"alliance_name": nilIfEmpty(allianceNames[bAlliance]),
				},
				"mutual_kills": int64OrZero(row["mutual_kills"]),
				"total_isk":    float64OrZero(row["total_isk"]),
			})
		}
		return jsonPayload(map[string]any{
			"items": items, "entityType": entityType, "mode": mode,
		}), nil
	}

	rows, err := graph.Read(ctx, `
		MATCH (attacker:Character)-[edge:KILLED]->(victim:Character)
		WHERE attacker.alliance_id IS NOT NULL
		  AND victim.alliance_id IS NOT NULL
		  AND attacker.alliance_id <> victim.alliance_id
		WITH CASE WHEN attacker.alliance_id < victim.alliance_id
		       THEN attacker.alliance_id ELSE victim.alliance_id END AS lo,
		     CASE WHEN attacker.alliance_id < victim.alliance_id
		       THEN victim.alliance_id ELSE attacker.alliance_id END AS hi,
		     sum(edge.weight) AS total_kills,
		     sum(edge.isk_destroyed) AS total_isk
		RETURN lo, hi, total_kills, total_isk
		ORDER BY total_kills DESC
		LIMIT $limit`, map[string]any{"limit": int64(limit)})
	if err != nil {
		return legacyPayload{}, err
	}
	allianceSet := map[int64]struct{}{}
	for _, row := range rows {
		addGraphID(allianceSet, row["lo"])
		addGraphID(allianceSet, row["hi"])
	}
	names, err := loadGraphNames(
		ctx, opts.DB, "alliance", sortedInt64Set(allianceSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		a, b := int64OrZero(row["lo"]), int64OrZero(row["hi"])
		items = append(items, map[string]any{
			"entity_a": map[string]any{
				"id": a, "name": fallbackName(names[a], "Alliance", a),
			},
			"entity_b": map[string]any{
				"id": b, "name": fallbackName(names[b], "Alliance", b),
			},
			"mutual_kills": int64OrZero(row["total_kills"]),
			"total_isk":    float64OrZero(row["total_isk"]),
		})
	}
	return jsonPayload(map[string]any{
		"items": items, "entityType": entityType, "mode": mode,
	}), nil
}

func graphEntityIntel(
	ctx context.Context,
	opts Options,
	req *legacyRequest,
	mode string,
) (legacyPayload, error) {
	id := graphQueryID(req.Query.Get("entityId"))
	entityType, property, prefix := graphEntityType(req)
	if id == 0 {
		return jsonPayload(map[string]any{
			"allies":  []map[string]any{},
			"enemies": []map[string]any{},
			"mode":    mode,
		}), nil
	}
	graph, err := requireGraph(opts)
	if err != nil {
		return legacyPayload{}, err
	}
	outbound, err := graph.Read(ctx, fmt.Sprintf(`
		MATCH (attacker:Character)-[edge:KILLED]->(victim:Character)
		WHERE attacker.%[1]s = $id AND victim.%[1]s IS NOT NULL
		  AND victim.%[1]s <> $id
		WITH victim.%[1]s AS other, sum(edge.weight) AS kills,
		     sum(edge.isk_destroyed) AS isk
		WHERE kills >= 10
		RETURN other, kills, isk`, property), map[string]any{"id": id})
	if err != nil {
		return legacyPayload{}, err
	}
	inbound, err := graph.Read(ctx, fmt.Sprintf(`
		MATCH (attacker:Character)-[edge:KILLED]->(victim:Character)
		WHERE victim.%[1]s = $id AND attacker.%[1]s IS NOT NULL
		  AND attacker.%[1]s <> $id
		WITH attacker.%[1]s AS other, sum(edge.weight) AS kills,
		     sum(edge.isk_destroyed) AS isk
		WHERE kills >= 10
		RETURN other, kills, isk`, property), map[string]any{"id": id})
	if err != nil {
		return legacyPayload{}, err
	}
	targetKills := graphCountMap(outbound, "other", "kills")
	targetLosses := graphCountMap(inbound, "other", "kills")
	mutual := map[int64]int64{}
	for other, kills := range targetKills {
		mutual[other] += kills
	}
	for other, kills := range targetLosses {
		mutual[other] += kills
	}
	enemyIDs := []int64{}
	for other, kills := range targetKills {
		if kills >= 50 {
			enemyIDs = append(enemyIDs, other)
		}
	}
	sharedScores := map[int64]int64{}
	if len(enemyIDs) > 0 {
		shared, err := graph.Read(ctx, fmt.Sprintf(`
			MATCH (attacker:Character)-[edge:KILLED]->(victim:Character)
			WHERE victim.%[1]s IN $enemies
			  AND attacker.%[1]s IS NOT NULL
			  AND attacker.%[1]s <> $id
			WITH attacker.%[1]s AS other, victim.%[1]s AS enemy,
			     sum(edge.weight) AS kills
			WHERE kills >= 20
			RETURN other, enemy, kills`, property), map[string]any{
			"enemies": enemyIDs, "id": id,
		})
		if err != nil {
			return legacyPayload{}, err
		}
		for _, row := range shared {
			other, enemy := int64OrZero(row["other"]), int64OrZero(row["enemy"])
			sharedScores[other] += minInt64(
				targetKills[enemy], int64OrZero(row["kills"]),
			)
		}
	}
	allies := []map[string]any{}
	for other, shared := range sharedScores {
		if shared < 200 {
			continue
		}
		ratio := math.Inf(1)
		if mutual[other] > 0 {
			ratio = float64(shared) / float64(mutual[other])
		}
		if ratio >= 4 {
			allies = append(allies, map[string]any{
				"id": other, "shared_enemy_kills": shared,
				"mutual_kills": mutual[other],
			})
		}
	}
	sort.Slice(allies, func(i, j int) bool {
		return int64OrZero(allies[i]["shared_enemy_kills"]) >
			int64OrZero(allies[j]["shared_enemy_kills"])
	})
	if len(allies) > 20 {
		allies = allies[:20]
	}
	enemies := []map[string]any{}
	for other, total := range mutual {
		if total >= 50 {
			enemies = append(enemies, map[string]any{
				"id": other, "kills_given": targetKills[other],
				"kills_taken": targetLosses[other], "total": total,
			})
		}
	}
	sort.Slice(enemies, func(i, j int) bool {
		return int64OrZero(enemies[i]["total"]) >
			int64OrZero(enemies[j]["total"])
	})
	if len(enemies) > 20 {
		enemies = enemies[:20]
	}
	peerSet := map[int64]struct{}{}
	for _, list := range [][]map[string]any{allies, enemies} {
		for _, row := range list {
			addGraphID(peerSet, row["id"])
		}
	}
	names, err := loadGraphNames(
		ctx, opts.DB, entityType, sortedInt64Set(peerSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	for _, row := range allies {
		other := int64OrZero(row["id"])
		row["name"] = fallbackName(names[other], prefix, other)
	}
	for _, row := range enemies {
		other := int64OrZero(row["id"])
		row["name"] = fallbackName(names[other], prefix, other)
	}
	return jsonPayload(map[string]any{
		"allies": allies, "enemies": enemies,
		"mode": mode, "entityType": entityType,
	}), nil
}

func graphHuntingGrounds(
	ctx context.Context,
	opts Options,
	req *legacyRequest,
	mode string,
) (legacyPayload, error) {
	id := graphQueryID(req.Query.Get("entityId"))
	entityType, property, _ := graphEntityType(req)
	if id == 0 {
		return jsonPayload(map[string]any{
			"systems": []map[string]any{}, "mode": mode,
		}), nil
	}
	graph, err := requireGraph(opts)
	if err != nil {
		return legacyPayload{}, err
	}
	rows, err := graph.Read(ctx, fmt.Sprintf(`
		MATCH (character:Character)-[edge:OPERATED_IN]->(system:SolarSystem)
		WHERE character.%s = $id AND edge.last_seen > $cutoff
		WITH system.id AS system_id,
		     count(DISTINCT character) AS active_chars,
		     max(edge.last_seen) AS latest
		RETURN system_id, active_chars, latest
		ORDER BY active_chars DESC
		LIMIT 30`, property), map[string]any{
		"id":     id,
		"cutoff": javascriptTimestamp(time.Now().UTC().Add(-24 * time.Hour)),
	})
	if err != nil {
		return legacyPayload{}, err
	}
	systemSet := map[int64]struct{}{}
	for _, row := range rows {
		addGraphID(systemSet, row["system_id"])
	}
	names, err := loadGraphNames(
		ctx, opts.DB, "system", sortedInt64Set(systemSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	systems := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		systemID := int64OrZero(row["system_id"])
		systems = append(systems, map[string]any{
			"id":                systemID,
			"name":              fallbackName(names[systemID], "System", systemID),
			"active_characters": int64OrZero(row["active_chars"]),
			"latest_activity":   row["latest"],
		})
	}
	return jsonPayload(map[string]any{
		"systems": systems, "mode": mode, "entityType": entityType,
	}), nil
}

func graphHotZones(
	ctx context.Context,
	opts Options,
	mode string,
) (legacyPayload, error) {
	graph, err := requireGraph(opts)
	if err != nil {
		return legacyPayload{}, err
	}
	rows, err := graph.Read(ctx, `
		MATCH (character:Character)-[edge:OPERATED_IN]->(system:SolarSystem)
		WHERE edge.last_seen > $cutoff
		  AND character.alliance_id IS NOT NULL
		WITH system.id AS system_id,
		     character.alliance_id AS alliance,
		     count(DISTINCT character) AS chars
		WITH system_id, count(DISTINCT alliance) AS alliance_count,
		     sum(chars) AS total_chars
		WHERE alliance_count >= 3
		RETURN system_id, alliance_count, total_chars
		ORDER BY total_chars DESC
		LIMIT 30`, map[string]any{
		"cutoff": javascriptTimestamp(time.Now().UTC().Add(-24 * time.Hour)),
	})
	if err != nil {
		return legacyPayload{}, err
	}
	systemSet := map[int64]struct{}{}
	for _, row := range rows {
		addGraphID(systemSet, row["system_id"])
	}
	names, err := loadGraphNames(
		ctx, opts.DB, "system", sortedInt64Set(systemSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	systems := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := int64OrZero(row["system_id"])
		systems = append(systems, map[string]any{
			"id": id, "name": fallbackName(names[id], "System", id),
			"alliances":  int64OrZero(row["alliance_count"]),
			"characters": int64OrZero(row["total_chars"]),
		})
	}
	return jsonPayload(map[string]any{"systems": systems, "mode": mode}), nil
}

func graphMigration(
	ctx context.Context,
	opts Options,
	req *legacyRequest,
	mode string,
) (legacyPayload, error) {
	id := graphQueryID(req.Query.Get("entityId"))
	if id == 0 {
		return jsonPayload(map[string]any{
			"departed": []map[string]any{},
			"joined":   []map[string]any{}, "mode": mode,
		}), nil
	}
	graph, err := requireGraph(opts)
	if err != nil {
		return legacyPayload{}, err
	}
	departed, err := graph.Read(ctx, `
		MATCH (character:Character)-[membership:MEMBER_OF]->
		      (corporation:Corporation {id: $id})
		WHERE character.corporation_id <> $id
		RETURN character.id AS char_id,
		       character.corporation_id AS current_corp,
		       membership.last_seen AS left_before
		ORDER BY membership.last_seen DESC
		LIMIT 30`, map[string]any{"id": id})
	if err != nil {
		return legacyPayload{}, err
	}
	joined, err := graph.Read(ctx, `
		MATCH (character:Character)-[membership:MEMBER_OF]->
		      (corporation:Corporation {id: $id})
		WHERE character.corporation_id = $id
		MATCH (character)-[:MEMBER_OF]->(other:Corporation)
		WHERE other.id <> $id
		WITH DISTINCT character, membership
		RETURN character.id AS char_id,
		       membership.first_seen AS joined_at
		ORDER BY membership.first_seen DESC
		LIMIT 30`, map[string]any{"id": id})
	if err != nil {
		return legacyPayload{}, err
	}
	joinedIDs := []int64{}
	for _, row := range joined {
		joinedIDs = append(joinedIDs, int64OrZero(row["char_id"]))
	}
	previous := []map[string]any{}
	if len(joinedIDs) > 0 {
		previous, err = graph.Read(ctx, `
			UNWIND $ids AS cid
			MATCH (character:Character {id: cid})-[membership:MEMBER_OF]->
			      (corporation:Corporation)
			WHERE corporation.id <> $corpId
			RETURN character.id AS char_id,
			       corporation.id AS prev_corp,
			       membership.last_seen AS left_at
			ORDER BY membership.last_seen DESC`, map[string]any{
			"ids": joinedIDs, "corpId": id,
		})
		if err != nil {
			return legacyPayload{}, err
		}
	}
	previousByCharacter := map[int64]map[string]any{}
	for _, row := range previous {
		characterID := int64OrZero(row["char_id"])
		if previousByCharacter[characterID] == nil {
			previousByCharacter[characterID] = row
		}
	}
	characterSet := map[int64]struct{}{}
	corporationSet := map[int64]struct{}{id: {}}
	for _, row := range departed {
		addGraphID(characterSet, row["char_id"])
		addGraphID(corporationSet, row["current_corp"])
	}
	for _, row := range joined {
		characterID := int64OrZero(row["char_id"])
		characterSet[characterID] = struct{}{}
		if previous := previousByCharacter[characterID]; previous != nil {
			addGraphID(corporationSet, previous["prev_corp"])
		}
	}
	characterNames, err := loadGraphNames(
		ctx, opts.DB, "character", sortedInt64Set(characterSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	corporationNames, err := loadGraphNames(
		ctx, opts.DB, "corporation", sortedInt64Set(corporationSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	departedOutput := make([]map[string]any, 0, len(departed))
	for _, row := range departed {
		characterID := int64OrZero(row["char_id"])
		corporationID := int64OrZero(row["current_corp"])
		departedOutput = append(departedOutput, map[string]any{
			"id": characterID,
			"name": fallbackName(
				characterNames[characterID], "Character", characterID,
			),
			"current_corp": map[string]any{
				"id":   corporationID,
				"name": nilIfEmpty(corporationNames[corporationID]),
			},
			"last_seen": row["left_before"],
		})
	}
	joinedOutput := make([]map[string]any, 0, len(joined))
	for _, row := range joined {
		characterID := int64OrZero(row["char_id"])
		var previousCorporation any
		if previous := previousByCharacter[characterID]; previous != nil {
			corporationID := int64OrZero(previous["prev_corp"])
			previousCorporation = map[string]any{
				"id":   corporationID,
				"name": nilIfEmpty(corporationNames[corporationID]),
			}
		}
		joinedOutput = append(joinedOutput, map[string]any{
			"id": characterID,
			"name": fallbackName(
				characterNames[characterID], "Character", characterID,
			),
			"joined_at":     row["joined_at"],
			"previous_corp": previousCorporation,
		})
	}
	return jsonPayload(map[string]any{
		"departed": departedOutput, "joined": joinedOutput,
		"corp_name": nilIfEmpty(corporationNames[id]), "mode": mode,
	}), nil
}

func graphSpyCheck(
	ctx context.Context,
	opts Options,
	req *legacyRequest,
	mode string,
) (legacyPayload, error) {
	id := graphQueryID(req.Query.Get("entityId"))
	entityType, property, _ := graphEntityType(req)
	if id == 0 {
		return jsonPayload(map[string]any{
			"suspects": []map[string]any{}, "mode": mode,
		}), nil
	}
	graph, err := requireGraph(opts)
	if err != nil {
		return legacyPayload{}, err
	}
	rows, err := graph.Read(ctx, fmt.Sprintf(`
		MATCH (character:Character)-[edge:FLEW_WITH]-(other:Character)
		WHERE character.%[1]s = $id AND other.%[1]s IS NOT NULL
		  AND other.%[1]s <> $id
		WITH character, other.%[1]s AS enemy_group,
		     sum(edge.weight) AS flights
		WHERE flights >= 10
		RETURN character.id AS char_id,
		       character.corporation_id AS corp_id,
		       enemy_group, flights
		ORDER BY flights DESC
		LIMIT 50`, property), map[string]any{"id": id})
	if err != nil {
		return legacyPayload{}, err
	}
	type suspectData struct {
		Corporation int64
		Connections []map[string]any
	}
	suspectsByID := map[int64]*suspectData{}
	characterSet := map[int64]struct{}{}
	corporationSet := map[int64]struct{}{}
	groupSet := map[int64]struct{}{}
	for _, row := range rows {
		characterID := int64OrZero(row["char_id"])
		corporationID := int64OrZero(row["corp_id"])
		groupID := int64OrZero(row["enemy_group"])
		if suspectsByID[characterID] == nil {
			suspectsByID[characterID] = &suspectData{Corporation: corporationID}
		}
		suspectsByID[characterID].Connections = append(
			suspectsByID[characterID].Connections,
			map[string]any{
				"id": groupID, "flights": int64OrZero(row["flights"]),
			},
		)
		characterSet[characterID] = struct{}{}
		if corporationID != 0 {
			corporationSet[corporationID] = struct{}{}
		}
		if groupID != 0 {
			groupSet[groupID] = struct{}{}
		}
	}
	characterNames, err := loadGraphNames(
		ctx, opts.DB, "character", sortedInt64Set(characterSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	corporationNames, err := loadGraphNames(
		ctx, opts.DB, "corporation", sortedInt64Set(corporationSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	groupNames, err := loadGraphNames(
		ctx, opts.DB, entityType, sortedInt64Set(groupSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	suspects := make([]map[string]any, 0, len(suspectsByID))
	for characterID, data := range suspectsByID {
		sort.Slice(data.Connections, func(i, j int) bool {
			return int64OrZero(data.Connections[i]["flights"]) >
				int64OrZero(data.Connections[j]["flights"])
		})
		total := int64(0)
		for _, connection := range data.Connections {
			total += int64OrZero(connection["flights"])
			groupID := int64OrZero(connection["id"])
			connection["name"] = fallbackName(
				groupNames[groupID], "Group", groupID,
			)
		}
		connections := data.Connections
		if len(connections) > 5 {
			connections = connections[:5]
		}
		suspects = append(suspects, map[string]any{
			"id": characterID,
			"name": fallbackName(
				characterNames[characterID], "Character", characterID,
			),
			"corp_name":   nilIfEmpty(corporationNames[data.Corporation]),
			"connections": connections, "total_flights": total,
		})
	}
	sort.Slice(suspects, func(i, j int) bool {
		return int64OrZero(suspects[i]["total_flights"]) >
			int64OrZero(suspects[j]["total_flights"])
	})
	if len(suspects) > 30 {
		suspects = suspects[:30]
	}
	return jsonPayload(map[string]any{
		"suspects": suspects, "mode": mode, "entityType": entityType,
	}), nil
}

func graphCensus(
	ctx context.Context,
	opts Options,
	req *legacyRequest,
	mode string,
) (legacyPayload, error) {
	id := graphQueryID(req.Query.Get("entityId"))
	if id == 0 {
		return jsonPayload(map[string]any{
			"corps":  []map[string]any{},
			"totals": map[string]any{}, "mode": mode,
		}), nil
	}
	graph, err := requireGraph(opts)
	if err != nil {
		return legacyPayload{}, err
	}
	rows, err := graph.Read(ctx, `
		MATCH (character:Character)-[:MEMBER_OF]->(corporation:Corporation)
		WHERE corporation.alliance_id = $id
		RETURN corporation.id AS corp_id,
		       count(character) AS total,
		       sum(CASE WHEN character.last_fc_seen > $cutoff
		           THEN 1 ELSE 0 END) AS fcs,
		       sum(CASE WHEN character.last_logi_seen > $cutoff
		           THEN 1 ELSE 0 END) AS logis,
		       sum(CASE WHEN character.last_capital_kill > $cutoff
		           THEN 1 ELSE 0 END) AS caps,
		       sum(CASE WHEN character.last_super_kill > $cutoff
		           THEN 1 ELSE 0 END) AS supers,
		       sum(CASE WHEN character.last_blops_seen > $cutoff
		           THEN 1 ELSE 0 END) AS droppers
		ORDER BY total DESC`, map[string]any{
		"id": id,
		"cutoff": javascriptTimestamp(
			time.Now().UTC().Add(-90 * 24 * time.Hour),
		),
	})
	if err != nil {
		return legacyPayload{}, err
	}
	corporationSet := map[int64]struct{}{}
	for _, row := range rows {
		addGraphID(corporationSet, row["corp_id"])
	}
	names, err := loadGraphNames(
		ctx, opts.DB, "corporation", sortedInt64Set(corporationSet),
	)
	if err != nil {
		return legacyPayload{}, err
	}
	totals := map[string]any{
		"characters": int64(0), "fcs": int64(0), "logis": int64(0),
		"caps": int64(0), "supers": int64(0), "droppers": int64(0),
	}
	corporations := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		corporationID := int64OrZero(row["corp_id"])
		for _, key := range []string{
			"fcs", "logis", "caps", "supers", "droppers",
		} {
			totals[key] = int64OrZero(totals[key]) + int64OrZero(row[key])
		}
		totals["characters"] = int64OrZero(totals["characters"]) +
			int64OrZero(row["total"])
		corporations = append(corporations, map[string]any{
			"id":       corporationID,
			"name":     fallbackName(names[corporationID], "Corp", corporationID),
			"total":    int64OrZero(row["total"]),
			"fcs":      int64OrZero(row["fcs"]),
			"logis":    int64OrZero(row["logis"]),
			"caps":     int64OrZero(row["caps"]),
			"supers":   int64OrZero(row["supers"]),
			"droppers": int64OrZero(row["droppers"]),
		})
	}
	return jsonPayload(map[string]any{
		"corps": corporations, "totals": totals, "mode": mode,
	}), nil
}

func requireGraph(opts Options) (GraphDatabase, error) {
	if opts.Graph == nil {
		return nil, apiError(
			http.StatusServiceUnavailable, "Graph database is not configured",
		)
	}
	return opts.Graph, nil
}

func graphQueryID(raw string) int64 {
	number, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || number == 0 || math.IsNaN(number) ||
		math.IsInf(number, 0) || math.Trunc(number) != number ||
		number < math.MinInt64 || number > math.MaxInt64 {
		return 0
	}
	return int64(number)
}

func graphEntityType(req *legacyRequest) (string, string, string) {
	if strings.TrimSpace(req.Query.Get("entityType")) == "corporation" {
		return "corporation", "corporation_id", "Corp"
	}
	return "alliance", "alliance_id", "Alliance"
}

func loadGraphNames(
	ctx context.Context,
	db Database,
	kind string,
	ids []int64,
) (map[int64]string, error) {
	result := map[int64]string{}
	if len(ids) == 0 {
		return result, nil
	}
	specs := map[string]struct {
		Table, ID, Name string
	}{
		"character":   {"characters", "character_id", "name"},
		"corporation": {"corporations", "corporation_id", "name"},
		"alliance":    {"alliances", "alliance_id", "name"},
		"system":      {"solar_systems", "solar_system_id", "system_name"},
	}
	spec, ok := specs[kind]
	if !ok {
		return result, nil
	}
	rows, err := queryMaps(ctx, db, fmt.Sprintf(
		`SELECT %s::bigint AS id, %s AS name
		 FROM %s WHERE %s = ANY($1::int[])`,
		spec.ID, spec.Name, spec.Table, spec.ID,
	), int64sToInt32(ids))
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[int64OrZero(row["id"])] = stringOrEmpty(row["name"])
	}
	return result, nil
}

func graphInt64Slice(value any) []int64 {
	switch values := value.(type) {
	case []int64:
		return append([]int64(nil), values...)
	case []int32:
		result := make([]int64, len(values))
		for i, value := range values {
			result[i] = int64(value)
		}
		return result
	case []any:
		result := make([]int64, 0, len(values))
		for _, value := range values {
			if id, ok := int64Value(value); ok {
				result = append(result, id)
			}
		}
		return result
	default:
		return []int64{}
	}
}

func addGraphID(target map[int64]struct{}, value any) {
	if id := int64OrZero(value); id != 0 {
		target[id] = struct{}{}
	}
}

func graphProfileTotal(profile map[int64]int64) int64 {
	total := int64(0)
	for _, kills := range profile {
		total += kills
	}
	return total
}

func orderedGraphPair(a, b int64) graphPairKey {
	if a > b {
		a, b = b, a
	}
	return graphPairKey{A: a, B: b}
}

type graphUnion struct {
	parent map[int64]int64
	groups map[int64]map[int64]struct{}
}

func newGraphUnion() *graphUnion {
	return &graphUnion{
		parent: map[int64]int64{},
		groups: map[int64]map[int64]struct{}{},
	}
}

func (u *graphUnion) add(id int64) {
	if _, exists := u.parent[id]; exists {
		return
	}
	u.parent[id] = id
	u.groups[id] = map[int64]struct{}{id: {}}
}

func (u *graphUnion) root(id int64) int64 {
	parent, exists := u.parent[id]
	if !exists {
		u.add(id)
		return id
	}
	if parent != id {
		u.parent[id] = u.root(parent)
	}
	return u.parent[id]
}

func (u *graphUnion) members(id int64) []int64 {
	root := u.root(id)
	result := make([]int64, 0, len(u.groups[root]))
	for member := range u.groups[root] {
		result = append(result, member)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func (u *graphUnion) join(a, b int64) {
	rootA, rootB := u.root(a), u.root(b)
	if rootA == rootB {
		return
	}
	if len(u.groups[rootA]) < len(u.groups[rootB]) {
		rootA, rootB = rootB, rootA
	}
	u.parent[rootB] = rootA
	for member := range u.groups[rootB] {
		u.parent[member] = rootA
		u.groups[rootA][member] = struct{}{}
	}
	delete(u.groups, rootB)
}

func (u *graphUnion) components() map[int64][]int64 {
	result := map[int64][]int64{}
	for id := range u.parent {
		root := u.root(id)
		result[root] = append(result[root], id)
	}
	for root := range result {
		sort.Slice(result[root], func(i, j int) bool {
			return result[root][i] < result[root][j]
		})
	}
	return result
}

func graphRivalryIDs(
	rows []map[string]any,
) ([]int64, []int64, []int64) {
	characters := map[int64]struct{}{}
	corporations := map[int64]struct{}{}
	alliances := map[int64]struct{}{}
	for _, row := range rows {
		addGraphID(characters, row["char_a"])
		addGraphID(characters, row["char_b"])
		addGraphID(corporations, row["corp_a"])
		addGraphID(corporations, row["corp_b"])
		addGraphID(alliances, row["alliance_a"])
		addGraphID(alliances, row["alliance_b"])
	}
	return sortedInt64Set(characters),
		sortedInt64Set(corporations),
		sortedInt64Set(alliances)
}
