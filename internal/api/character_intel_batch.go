package api

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const defaultBatchIntelDays = 90

type batchIntelRequest struct {
	CharacterIDs requestList[int64] `json:"character_ids" minItems:"1" maxItems:"100" doc:"Characters to inspect, at most 100 unique IDs."`
	Days         int                `json:"days,omitempty" minimum:"1" maximum:"90" default:"90" doc:"Look-back window in days, from 1 through 90."`
}

func registerBatchCharacterIntelRoute(a huma.API, opts Options) {
	registerLegacyJSON(a, huma.Operation{
		OperationID: "character-intel-batch",
		Method:      http.MethodPost,
		Path:        "/characters/intel",
		Summary:     "Batch character intelligence profiles",
		Tags:        []string{"characters"},
	}, analyzeBodyLimit, func(ctx context.Context, req *legacyRequest, body *batchIntelRequest) (legacyPayload, error) {
		if len(body.CharacterIDs) == 0 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "character_ids must be a non-empty array")
		}
		ids := make([]int32, 0, len(body.CharacterIDs))
		requestIDs := make([]int64, 0, len(body.CharacterIDs))
		seen := map[int64]struct{}{}
		for _, id := range body.CharacterIDs {
			if id <= 0 || id > math.MaxInt32 {
				return legacyPayload{}, apiError(http.StatusBadRequest, "All character_ids must be positive integers")
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			requestIDs = append(requestIDs, id)
			ids = append(ids, int32(id))
		}
		if len(ids) > 100 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Maximum 100 unique characters per request")
		}
		days := body.Days
		if days == 0 {
			days = defaultBatchIntelDays
		}
		queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		data, missing, err := loadCharacterIntelBatch(queryCtx, opts, requestIDs, ids, days)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"data": data, "not_found": missing, "days": days}), nil
	})
}

func loadCharacterIntelBatch(ctx context.Context, opts Options, requestIDs []int64, ids []int32, days int) ([]map[string]any, []int64, error) {
	results, err := queryMapsConcurrent(ctx, opts.DB,
		databaseQuery{SQL: `SELECT character_id FROM characters WHERE character_id = ANY($1::int[])`, Args: []any{ids}},
		databaseQuery{SQL: `
			SELECT a.character_id,
			 count(*) FILTER (WHERE k.attacker_count=1) AS solo,
			 count(*) FILTER (WHERE k.attacker_count BETWEEN 2 AND 5) AS small_gang,
			 count(*) FILTER (WHERE k.attacker_count BETWEEN 6 AND 15) AS mid_gang,
			 count(*) FILTER (WHERE k.attacker_count BETWEEN 16 AND 50) AS fleet,
			 count(*) FILTER (WHERE k.attacker_count>50) AS blob,
			 count(*) AS total,
			 round(avg(k.attacker_count)::numeric,1)::double precision AS avg_fleet_size,
			 count(*) FILTER (WHERE a.ship_type_id=45534) AS monitor_appearances,
			 count(*) FILTER (WHERE a.damage_done=0 AND k.attacker_count>=10) AS zero_dmg_fleet,
			 count(*) AS total_appearances
			FROM killmail_attackers a JOIN killmails k ON k.killmail_id=a.killmail_id
			WHERE a.character_id=ANY($1::int[]) AND a.killmail_time>now()-make_interval(days=>$2)
			GROUP BY a.character_id`, Args: []any{ids, days}},
		databaseQuery{SQL: batchIntelShipsFlownSQL, Args: []any{ids, days}},
		databaseQuery{SQL: batchIntelShipsLostSQL, Args: []any{ids, days}},
		databaseQuery{SQL: batchIntelTargetsSQL, Args: []any{ids, days}},
		databaseQuery{SQL: batchIntelAwoxSQL, Args: []any{ids, days}},
		databaseQuery{SQL: batchIntelBaitSQL, Args: []any{ids, days}},
		databaseQuery{SQL: batchIntelArchetypeSQL, Args: []any{ids, days}},
	)
	if err != nil {
		return nil, nil, err
	}
	found := map[int64]struct{}{}
	for _, row := range results[0] {
		id, _ := int64Value(row["character_id"])
		found[id] = struct{}{}
	}
	graphs := loadGraphIntelBatch(ctx, opts.Graph, ids)
	charIDs, corpIDs, allianceIDs := batchGraphNameIDs(graphs)
	charNames, corpNames, allianceNames, err := intelEntityNames(ctx, opts.DB, charIDs, corpIDs, allianceIDs)
	if err != nil {
		return nil, nil, err
	}
	playstyles := batchRowsByID(results[1], "character_id")
	shipsFlown := batchListsByID(results[2], "character_id")
	shipsLost := batchListsByID(results[3], "character_id")
	targets := batchListsByID(results[4], "character_id")
	awox := batchRowsByID(results[5], "character_id")
	bait := batchRowsByID(results[6], "character_id")
	archetypes := batchRowsByID(results[7], "character_id")
	data := make([]map[string]any, 0, len(found))
	missing := []int64{}
	for _, id := range requestIDs {
		if _, ok := found[id]; !ok {
			missing = append(missing, id)
			continue
		}
		data = append(data, assembleCharacterIntel(id, days,
			playstyles[id], shipsFlown[id], shipsLost[id], targets[id], awox[id], bait[id], archetypes[id], graphs[id],
			charNames, corpNames, allianceNames))
	}
	return data, missing, nil
}

func batchRowsByID(rows []map[string]any, key string) map[int64]map[string]any {
	result := map[int64]map[string]any{}
	for _, row := range rows {
		id, _ := int64Value(row[key])
		result[id] = row
	}
	return result
}

func batchListsByID(rows []map[string]any, key string) map[int64][]map[string]any {
	result := map[int64][]map[string]any{}
	for _, row := range rows {
		id, _ := int64Value(row[key])
		result[id] = append(result[id], row)
	}
	return result
}

const batchIntelShipsFlownSQL = `
WITH counts AS MATERIALIZED (
 SELECT a.character_id,a.ship_type_id,count(*)::int AS count
 FROM killmail_attackers a WHERE a.character_id=ANY($1::int[])
 AND a.killmail_time>now()-make_interval(days=>$2) AND a.ship_type_id IS NOT NULL
 GROUP BY a.character_id,a.ship_type_id
), ranked AS (
 SELECT counts.*,row_number() OVER(PARTITION BY character_id ORDER BY count DESC) AS rank FROM counts
)
SELECT r.character_id,r.ship_type_id,t.name AS ship_name,r.count FROM ranked r
JOIN inv_types t ON t.type_id=r.ship_type_id WHERE r.rank<=5`

const batchIntelShipsLostSQL = `
WITH counts AS MATERIALIZED (
 SELECT k.victim_character_id AS character_id,k.victim_ship_type_id AS ship_type_id,count(*)::int AS count
 FROM killmails k WHERE k.victim_character_id=ANY($1::int[])
 AND k.killmail_time>now()-make_interval(days=>$2) AND k.victim_ship_type_id IS NOT NULL
 GROUP BY k.victim_character_id,k.victim_ship_type_id
), ranked AS (
 SELECT counts.*,row_number() OVER(PARTITION BY character_id ORDER BY count DESC) AS rank FROM counts
)
SELECT r.character_id,r.ship_type_id,t.name AS ship_name,r.count FROM ranked r
JOIN inv_types t ON t.type_id=r.ship_type_id WHERE r.rank<=5`

const batchIntelTargetsSQL = `
WITH counts AS MATERIALIZED (
 SELECT atk.character_id,k.victim_alliance_id AS alliance_id,count(*)::int AS count
 FROM killmail_attackers atk JOIN killmails k ON k.killmail_id=atk.killmail_id
 WHERE atk.character_id=ANY($1::int[]) AND atk.killmail_time>now()-make_interval(days=>$2)
 AND k.victim_alliance_id IS NOT NULL GROUP BY atk.character_id,k.victim_alliance_id
), ranked AS (
 SELECT counts.*,row_number() OVER(PARTITION BY character_id ORDER BY count DESC) AS rank FROM counts
)
SELECT r.character_id,r.alliance_id,a.name AS alliance_name,r.count FROM ranked r
LEFT JOIN alliances a ON a.alliance_id=r.alliance_id WHERE r.rank<=10`

const batchIntelAwoxSQL = `
SELECT atk.character_id,count(*)::int AS awox_kills
FROM killmail_attackers atk JOIN killmails k ON k.killmail_id=atk.killmail_id
WHERE atk.character_id=ANY($1::int[]) AND atk.killmail_time>now()-make_interval(days=>$2)
AND atk.alliance_id IS NOT NULL AND atk.alliance_id=k.victim_alliance_id
AND atk.corporation_id!=k.victim_corporation_id GROUP BY atk.character_id`

const batchIntelBaitSQL = `
WITH requested AS (SELECT unnest($1::int[]) AS character_id),
cheap AS MATERIALIZED (
 SELECT victim_character_id AS character_id,killmail_id,killmail_time,solar_system_id
 FROM killmails WHERE victim_character_id=ANY($1::int[])
 AND killmail_time>now()-make_interval(days=>$2) AND total_value<50000000
), baited AS (
 SELECT DISTINCT c.character_id,c.killmail_id FROM cheap c JOIN killmails f
 ON f.solar_system_id=c.solar_system_id
 AND f.killmail_time BETWEEN c.killmail_time AND c.killmail_time+interval '5 minutes'
 AND f.killmail_id!=c.killmail_id AND f.attacker_count>=2
), cc AS (SELECT character_id,count(*)::int AS cheap_deaths FROM cheap GROUP BY character_id),
bc AS (SELECT character_id,count(*)::int AS baited_deaths FROM baited GROUP BY character_id)
SELECT r.character_id,coalesce(cc.cheap_deaths,0) AS cheap_deaths,coalesce(bc.baited_deaths,0) AS baited_deaths
FROM requested r LEFT JOIN cc USING(character_id) LEFT JOIN bc USING(character_id)`

const batchIntelArchetypeSQL = `
WITH requested AS (SELECT unnest($1::int[]) AS character_id),
activity AS (
 SELECT atk.character_id,count(*) FILTER(WHERE s.security>=0.5)::int AS hs_kms,
 count(*) FILTER(WHERE s.security>0 AND s.security<0.5)::int AS ls_kms,
 count(*) FILTER(WHERE s.security<=0)::int AS ns_kms,count(*)::int AS total_kms_90d,
 count(*) FILTER(WHERE s.security>=0.5 AND atk.security_status < -5.0)::int AS gank_kms
 FROM killmail_attackers atk JOIN killmails k ON k.killmail_id=atk.killmail_id
 JOIN solar_systems s ON s.solar_system_id=k.solar_system_id
 WHERE atk.character_id=ANY($1::int[]) AND atk.killmail_time>now()-make_interval(days=>$2) GROUP BY atk.character_id
), losses AS (
 SELECT victim_character_id AS character_id,count(*)::int AS losses_90d FROM killmails
 WHERE victim_character_id=ANY($1::int[]) AND killmail_time>now()-make_interval(days=>$2) GROUP BY victim_character_id
), cyno AS (
 SELECT k.victim_character_id AS character_id,count(DISTINCT k.killmail_id)::int AS cyno_losses_90d
 FROM killmails k JOIN killmail_items ki ON ki.killmail_id=k.killmail_id
 WHERE k.victim_character_id=ANY($1::int[]) AND k.killmail_time>now()-make_interval(days=>$2)
 AND ki.type_id IN(21096,28646,52694) AND ki.flag_id BETWEEN 11 AND 34 GROUP BY k.victim_character_id
)
SELECT r.character_id,coalesce(a.hs_kms,0) AS hs_kms,coalesce(a.ls_kms,0) AS ls_kms,
coalesce(a.ns_kms,0) AS ns_kms,coalesce(a.total_kms_90d,0) AS total_kms_90d,
coalesce(a.gank_kms,0) AS gank_kms,coalesce(l.losses_90d,0) AS losses_90d,
coalesce(c.cyno_losses_90d,0) AS cyno_losses_90d,coalesce(c.cyno_losses_90d,0) AS cyno_deaths,
coalesce(ch.birthday>now()-interval '180 days',false) AS is_new_char
FROM requested r LEFT JOIN activity a USING(character_id) LEFT JOIN losses l USING(character_id)
LEFT JOIN cyno c USING(character_id) LEFT JOIN characters ch USING(character_id)`

func loadGraphIntelBatch(ctx context.Context, graph GraphDatabase, ids []int32) map[int64]graphIntel {
	result := map[int64]graphIntel{}
	for _, id := range ids {
		result[int64(id)] = graphIntel{Timestamps: emptyGraphTimestamps()}
	}
	if graph == nil {
		return result
	}
	graphIDs := make([]int64, len(ids))
	for i, id := range ids {
		graphIDs[i] = int64(id)
	}
	params := map[string]any{"ids": graphIDs}
	partners, err := graph.Read(ctx, `
		UNWIND $ids AS cid MATCH (c:Character {id: cid})-[r:FLEW_WITH]-(p:Character)
		WITH cid,p,r ORDER BY cid,r.weight DESC
		WITH cid,collect({id:p.id,corp_id:p.corporation_id,alliance_id:p.alliance_id,weight:r.weight})[..10] AS partners
		RETURN cid,partners`, params)
	if err != nil {
		return result
	}
	groups, err := graph.Read(ctx, `
		UNWIND $ids AS cid MATCH (c:Character {id: cid})-[:FLEW_WITH]-(p:Character)
		WHERE p.alliance_id IS NOT NULL
		WITH cid,p.alliance_id AS alliance_id,count(DISTINCT p) AS shared_partners
		ORDER BY cid,shared_partners DESC
		WITH cid,collect({alliance_id:alliance_id,shared_partners:shared_partners}) AS groups
		RETURN cid,size(groups) AS cnt,groups[..10] AS groups`, params)
	if err != nil {
		return result
	}
	timestamps, err := graph.Read(ctx, `
		UNWIND $ids AS cid MATCH (c:Character {id: cid})
		RETURN cid,c.last_fc_seen AS last_fc_seen,c.last_super_kill AS last_super_kill,
		 c.last_blops_seen AS last_blops_seen,c.last_capital_kill AS last_capital_kill,
		 c.last_logi_seen AS last_logi_seen`, params)
	if err != nil {
		return result
	}
	for _, row := range partners {
		id, _ := int64Value(row["cid"])
		item := result[id]
		item.FleetPartners = graphMapList(row["partners"])
		result[id] = item
	}
	for _, row := range groups {
		id, _ := int64Value(row["cid"])
		item := result[id]
		item.BridgeScore, _ = int64Value(row["cnt"])
		item.GroupsFlownWith = graphMapList(row["groups"])
		result[id] = item
	}
	for _, row := range timestamps {
		id, _ := int64Value(row["cid"])
		item := result[id]
		for key := range item.Timestamps {
			item.Timestamps[key] = intelTimestamp(row[key])
		}
		result[id] = item
	}
	return result
}

func batchGraphNameIDs(graphs map[int64]graphIntel) ([]int32, []int32, []int32) {
	characters, corporations, alliances := []int32{}, []int32{}, []int32{}
	for _, graph := range graphs {
		characters = append(characters, graphFieldIDs(graph.FleetPartners, "id")...)
		corporations = append(corporations, graphFieldIDs(graph.FleetPartners, "corp_id")...)
		alliances = append(alliances, graphFieldIDs(graph.FleetPartners, "alliance_id")...)
		alliances = append(alliances, graphFieldIDs(graph.GroupsFlownWith, "alliance_id")...)
	}
	return characters, corporations, alliances
}
