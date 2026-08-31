package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/eve"
)

func loadEntityPageMembers(
	ctx context.Context,
	opts Options,
	kind string,
	id int64,
	req *legacyRequest,
) (any, error) {
	page := max(entityPageQueryInt(req.Query.Get("page"), 1), 1)
	limit := min(max(entityPageQueryInt(req.Query.Get("limit"), 100), 1), 200)
	offset64 := min(int64(page-1)*int64(limit), math.MaxInt32)
	sortBy := req.Query.Get("sort")
	orderBy := "name ASC"
	switch sortBy {
	case "last_active":
		orderBy = "last_active DESC NULLS LAST"
	case "security_status":
		orderBy = "security_status DESC NULLS LAST"
	}

	args := []any{id}
	where := ""
	switch kind {
	case entityPageCorporation:
		where = "corporation_id = $1"
	case entityPageAlliance:
		where = "alliance_id = $1"
	default:
		return nil, apiError(http.StatusBadRequest, "Invalid entity type")
	}
	corpFilter := int64(entityPageQueryInt(req.Query.Get("corporation_id"), 0))
	if kind == entityPageAlliance && corpFilter != 0 {
		args = append(args, corpFilter)
		where += fmt.Sprintf(" AND corporation_id = $%d", len(args))
	}
	activityDays := entityPageQueryInt(req.Query.Get("activity"), 0)
	if activityDays > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -minInt(activityDays, 100000))
		args = append(args, cutoff)
		where += fmt.Sprintf(" AND last_active >= $%d", len(args))
	}
	args = append(args, limit, offset64)
	limitParam, offsetParam := len(args)-1, len(args)

	rows, err := queryMaps(ctx, opts.DB, fmt.Sprintf(`
		WITH filtered AS (
			SELECT character_id, name, security_status, last_active,
			       corporation_id
			FROM characters
			WHERE %s
		), totals AS (
			SELECT COUNT(*)::bigint AS total FROM filtered
		), page AS (
			SELECT * FROM filtered
			ORDER BY %s
			LIMIT $%d OFFSET $%d
		), recent AS (
			SELECT s.entity_id AS character_id,
			       COALESCE(SUM(s.kills), 0)::bigint AS kills,
			       COALESCE(SUM(s.losses), 0)::bigint AS losses
			FROM stats s
			WHERE s.entity_type = 0 AND s.period_type = 0
			  AND s.period_start >= CURRENT_DATE - 90
			  AND s.entity_id IN (SELECT character_id FROM page)
			GROUP BY s.entity_id
		)
		SELECT p.character_id, p.name,
		       COALESCE(p.security_status, 0)::real AS security_status,
		       p.last_active, p.corporation_id,
		       COALESCE(r.kills, 0)::bigint AS kills_90d,
		       COALESCE(r.losses, 0)::bigint AS losses_90d,
		       totals.total
		FROM totals
		LEFT JOIN page p ON true
		LEFT JOIN recent r ON r.character_id = p.character_id
		ORDER BY %s`,
		where, orderBy, limitParam, offsetParam, orderBy,
	), args...)
	if err != nil {
		return nil, err
	}

	total := int64(0)
	characterIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		total = int64OrZero(row["total"])
		if characterID := int64OrZero(row["character_id"]); characterID > 0 {
			characterIDs = append(characterIDs, characterID)
		}
	}
	roles := loadEntityMemberRoles(ctx, opts.Graph, characterIDs)
	members := make([]map[string]any, 0, len(characterIDs))
	for _, row := range rows {
		characterID := int64OrZero(row["character_id"])
		if characterID == 0 {
			continue
		}
		role := roles[characterID]
		member := map[string]any{
			"character_id":     characterID,
			"name":             row["name"],
			"security_status":  row["security_status"],
			"last_active":      row["last_active"],
			"kills_90d":        int64OrZero(row["kills_90d"]),
			"losses_90d":       int64OrZero(row["losses_90d"]),
			"is_fc":            role["is_fc"],
			"is_logi":          role["is_logi"],
			"is_capital_pilot": role["is_capital_pilot"],
		}
		if kind == entityPageAlliance {
			member["corporation_id"] = row["corporation_id"]
		}
		members = append(members, member)
	}
	return map[string]any{
		"members": members, "total": total, "page": page, "limit": limit,
	}, nil
}

func loadEntityMemberRoles(
	ctx context.Context,
	graph GraphDatabase,
	ids []int64,
) map[int64]map[string]bool {
	out := make(map[int64]map[string]bool, len(ids))
	for _, id := range ids {
		out[id] = map[string]bool{
			"is_fc": false, "is_logi": false, "is_capital_pilot": false,
		}
	}
	if graph == nil || len(ids) == 0 {
		return out
	}
	rows, err := graph.Read(ctx, `
		UNWIND $ids AS cid
		MATCH (c:Character {id: cid})
		RETURN c.id AS id,
		       c.last_fc_seen > $cutoff AS is_fc,
		       c.last_logi_seen > $cutoff AS is_logi,
		       c.last_capital_kill > $cutoff AS is_capital_pilot`,
		map[string]any{
			"ids": ids,
			"cutoff": time.Now().UTC().AddDate(0, 0, -90).
				Format("2006-01-02T15:04:05.000Z"),
		},
	)
	if err != nil {
		return out
	}
	for _, row := range rows {
		id := int64OrZero(row["id"])
		if id == 0 {
			continue
		}
		out[id] = map[string]bool{
			"is_fc":            boolOrFalse(row["is_fc"]),
			"is_logi":          boolOrFalse(row["is_logi"]),
			"is_capital_pilot": boolOrFalse(row["is_capital_pilot"]),
		}
	}
	return out
}

func boolOrFalse(value any) bool {
	out, _ := boolValue(value)
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func loadEntityPageCorporations(
	ctx context.Context,
	opts Options,
	_ string,
	id int64,
	req *legacyRequest,
) (any, error) {
	orderColumn := "member_count"
	if req.Query.Get("sort") == "name" {
		orderColumn = "name"
	}
	direction := "DESC"
	if req.Query.Get("dir") == "asc" {
		direction = "ASC"
	}
	rows, err := queryMaps(ctx, opts.DB, fmt.Sprintf(`
		SELECT corporation_id, name, ticker,
		       COALESCE(member_count, 0)::int AS member_count, palette
		FROM corporations
		WHERE alliance_id = $1
		ORDER BY %s %s`, orderColumn, direction), id)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"corporations": nonNilEntityPageRows(rows),
		"total":        len(rows),
	}, nil
}

func loadEntityPageMostValuable(
	ctx context.Context,
	opts Options,
	kind string,
	id int64,
	req *legacyRequest,
) (any, error) {
	entityColumn := map[string]string{
		entityPageCharacter:   "character_id",
		entityPageCorporation: "corporation_id",
		entityPageAlliance:    "alliance_id",
	}[kind]
	if entityColumn == "" {
		return nil, apiError(http.StatusBadRequest, "Invalid entity type")
	}
	dataType := req.Query.Get("dataType")
	if dataType == "" {
		dataType = "most_valuable_kills"
	}
	limit := min(max(entityPageQueryInt(req.Query.Get("limit"), 8), 1), 32)
	days := min(max(entityPageQueryInt(req.Query.Get("days"), 7), 1), 30)
	category := 0
	switch dataType {
	case "most_valuable_ships":
		category = 6
	case "most_valuable_structures":
		category = 65
	}
	args := []any{id, time.Now().UTC().AddDate(0, 0, -days)}
	categorySQL := ""
	if category != 0 {
		args = append(args, category)
		categorySQL = fmt.Sprintf(`
			AND k.victim_ship_group_id IN (
				SELECT group_id FROM inv_groups WHERE category_id = $%d
			)`, len(args))
	}
	args = append(args, limit)
	rows, err := queryMaps(ctx, opts.DB, fmt.Sprintf(`
		WITH involving AS MATERIALIZED (
			SELECT DISTINCT killmail_id
			FROM killmail_attackers
			WHERE %s = $1 AND killmail_time >= $2
		)
		SELECT k.killmail_id, k.killmail_hash,
		       k.victim_ship_type_id AS ship_type_id,
		       COALESCE(t.name, 'Unknown') AS ship_name,
		       COALESCE(k.total_value, 0)::double precision AS total_value,
		       k.victim_character_id, ch.name AS victim_character_name,
		       co.name AS victim_corporation_name,
		       al.name AS victim_alliance_name
		FROM killmails k
		JOIN involving i USING (killmail_id)
		JOIN inv_types t ON t.type_id = k.victim_ship_type_id
		LEFT JOIN characters ch ON ch.character_id = k.victim_character_id
		LEFT JOIN corporations co ON co.corporation_id = k.victim_corporation_id
		LEFT JOIN alliances al ON al.alliance_id = k.victim_alliance_id
		WHERE true %s
		ORDER BY k.total_value DESC
		LIMIT $%d`, entityColumn, categorySQL, len(args)), args...)
	if err != nil {
		return nil, err
	}
	return map[string]any{"entries": nonNilEntityPageRows(rows)}, nil
}

func loadEntityPageShipClasses(
	ctx context.Context,
	opts Options,
	kind string,
	id int64,
	_ *legacyRequest,
) (any, error) {
	entityType, ok := entityPageStatsType(kind)
	if !ok {
		return nil, apiError(http.StatusBadRequest, "Invalid entity type")
	}
	rows, err := queryMaps(ctx, opts.DB, `
		SELECT t.group_id,
		       COALESCE(g.name, 'Group ' || t.group_id::text) AS group_name,
		       SUM(b.losses)::bigint AS losses,
		       COALESCE(SUM(b.isk_lost), 0)::double precision AS isk_lost
		FROM stats_breakdowns b
		JOIN inv_types t ON t.type_id = b.dim_id
		LEFT JOIN inv_groups g ON g.group_id = t.group_id
		WHERE b.entity_type = $1 AND b.entity_id = $2
		  AND b.dim_category = 1 AND b.period_type = 2
		GROUP BY t.group_id, g.name
		HAVING SUM(b.losses) > 0
		ORDER BY SUM(b.losses) DESC`, entityType, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"groups": nonNilEntityPageRows(rows)}, nil
}

func loadEntityPageTopLists(
	ctx context.Context,
	opts Options,
	kind string,
	id int64,
	req *legacyRequest,
) (any, error) {
	entityType, ok := entityPageStatsType(kind)
	if !ok {
		return nil, apiError(http.StatusBadRequest, "Invalid entity type")
	}
	days := 7
	if _, present := req.Query["days"]; present {
		days = entityPageQueryInt(req.Query.Get("days"), 0)
	}
	window := entityPageDaysWindow(days)
	periodType, fromDate := statsWindow(window)
	args := []any{
		entityType, id, periodType,
		[]int16{
			dimKilledCorporation, dimKilledAlliance,
			dimDiesToCorporation, dimDiesToAlliance,
		},
	}
	dateSQL := ""
	if fromDate != "" {
		args = append(args, fromDate)
		dateSQL = fmt.Sprintf(" AND b.period_start >= $%d::date", len(args))
	}
	breakdownSQL := fmt.Sprintf(`
		WITH grouped AS (
			SELECT b.dim_category, b.dim_id,
			       COALESCE(SUM(b.kills), 0)::bigint AS kills,
			       COALESCE(SUM(b.losses), 0)::bigint AS losses,
			       COALESCE(SUM(b.isk_destroyed), 0)::double precision AS isk_destroyed,
			       COALESCE(SUM(b.isk_lost), 0)::double precision AS isk_lost
			FROM stats_breakdowns b
			WHERE b.entity_type = $1 AND b.entity_id = $2
			  AND b.period_type = $3
			  AND b.dim_category = ANY($4::smallint[]) %s
			GROUP BY b.dim_category, b.dim_id
		), ranked AS (
			SELECT grouped.*,
			       ROW_NUMBER() OVER (
			         PARTITION BY dim_category
			         ORDER BY CASE
			           WHEN dim_category IN (21, 22) THEN losses ELSE kills
			         END DESC, dim_id
			       ) AS rank
			FROM grouped
		)
		SELECT r.*,
		       CASE WHEN r.dim_category IN (31, 21)
		            THEN COALESCE(c.name, 'Unknown')
		            ELSE COALESCE(a.name, 'Unknown') END AS name,
		       CASE WHEN r.dim_category IN (31, 21) THEN c.palette
		            ELSE executor.palette END AS palette
		FROM ranked r
		LEFT JOIN corporations c
		  ON r.dim_category IN (31, 21) AND c.corporation_id = r.dim_id
		LEFT JOIN alliances a
		  ON r.dim_category IN (32, 22) AND a.alliance_id = r.dim_id
		LEFT JOIN corporations executor
		  ON executor.corporation_id = a.executor_corporation_id
		WHERE r.rank <= 10
		ORDER BY r.dim_category, r.rank`, dateSQL)
	queries := []databaseQuery{{SQL: breakdownSQL, Args: args}}
	if kind == entityPageCharacter {
		since := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
		if days > 0 {
			since = time.Now().UTC().AddDate(0, 0, -minInt(days, 100000))
		}
		queries = append(queries, databaseQuery{
			SQL: `
				WITH killed AS (
					SELECT 0 AS direction, k.victim_character_id::bigint AS id,
					       COUNT(DISTINCT k.killmail_id)::bigint AS count,
					       COALESCE(SUM(k.total_value), 0)::double precision AS isk_value
					FROM killmail_attackers actor
					JOIN killmails k ON k.killmail_id = actor.killmail_id
					WHERE actor.character_id = $1
					  AND actor.killmail_time >= $2
					  AND k.victim_character_id IS NOT NULL
					GROUP BY k.victim_character_id
					ORDER BY count DESC
					LIMIT 10
				), killed_by AS (
					SELECT 1 AS direction, actor.character_id::bigint AS id,
					       COUNT(DISTINCT actor.killmail_id)::bigint AS count,
					       COALESCE(SUM(k.total_value), 0)::double precision AS isk_value
					FROM killmails k
					JOIN killmail_attackers actor ON actor.killmail_id = k.killmail_id
					WHERE k.victim_character_id = $1
					  AND k.killmail_time >= $2
					  AND actor.character_id IS NOT NULL
					GROUP BY actor.character_id
					ORDER BY count DESC
					LIMIT 10
				), combined AS (
					SELECT * FROM killed UNION ALL SELECT * FROM killed_by
				)
				SELECT combined.*, COALESCE(ch.name, 'Unknown') AS name,
				       corp.palette
				FROM combined
				LEFT JOIN characters ch ON ch.character_id = combined.id
				LEFT JOIN corporations corp
				  ON corp.corporation_id = ch.corporation_id
				ORDER BY direction, count DESC`,
			Args: []any{id, since},
		})
	}
	results, err := queryMapsConcurrent(ctx, opts.DB, queries...)
	if err != nil {
		return nil, err
	}
	killed := map[string][]map[string]any{
		"characters": {}, "corporations": {}, "alliances": {},
	}
	killedBy := map[string][]map[string]any{
		"characters": {}, "corporations": {}, "alliances": {},
	}
	for _, row := range results[0] {
		category := int64OrZero(row["dim_category"])
		item := map[string]any{
			"id": row["dim_id"], "name": row["name"],
			"palette": row["palette"],
		}
		switch category {
		case dimKilledCorporation:
			item["count"], item["isk_value"] = row["kills"], row["isk_destroyed"]
			killed["corporations"] = append(killed["corporations"], item)
		case dimKilledAlliance:
			item["count"], item["isk_value"] = row["kills"], row["isk_destroyed"]
			killed["alliances"] = append(killed["alliances"], item)
		case dimDiesToCorporation:
			item["count"], item["isk_value"] = row["losses"], row["isk_lost"]
			killedBy["corporations"] = append(killedBy["corporations"], item)
		case dimDiesToAlliance:
			item["count"], item["isk_value"] = row["losses"], row["isk_lost"]
			killedBy["alliances"] = append(killedBy["alliances"], item)
		}
	}
	if len(results) > 1 {
		for _, row := range results[1] {
			item := map[string]any{
				"id": row["id"], "name": row["name"],
				"count": row["count"], "isk_value": row["isk_value"],
				"palette": row["palette"],
			}
			if int64OrZero(row["direction"]) == 0 {
				killed["characters"] = append(killed["characters"], item)
			} else {
				killedBy["characters"] = append(killedBy["characters"], item)
			}
		}
	}
	return map[string]any{"killed": killed, "killedBy": killedBy}, nil
}

type entityKilllistConfig struct {
	EntityType int
	Victim     string
	Attacker   string
}

var entityKilllistConfigs = map[string]entityKilllistConfig{
	entityPageCharacter:   {entityCharacter, "victim_character_id", "character_id"},
	entityPageCorporation: {entityCorporation, "victim_corporation_id", "corporation_id"},
	entityPageAlliance:    {entityAlliance, "victim_alliance_id", "alliance_id"},
	entityPageFaction:     {entityFaction, "victim_faction_id", "faction_id"},
}

type entityKilllistScopePlan struct {
	SQL      string
	Args     []any
	Order    string
	Offset   int64
	Numbered bool
}

func buildEntityKilllistScopePlan(
	config entityKilllistConfig,
	role string,
	id int64,
	limit int,
	after int64,
	page int,
	shipGroup int64,
) entityKilllistScopePlan {
	numbered := page >= 1 && after == 0 && config.EntityType >= 0 && shipGroup == 0
	order := "k.killmail_id DESC"
	offset := int64(0)
	if numbered {
		order = "k.killmail_time DESC, k.killmail_id DESC"
		offset = max(int64(page-1)*int64(limit), 0)
	}

	args := []any{id}
	lossConditions := []string{fmt.Sprintf("k.%s = $1", config.Victim)}
	attackConditions := []string{fmt.Sprintf("actor.%s = $1", config.Attacker)}
	factionAttackConditions := []string{fmt.Sprintf(`
		EXISTS (
			SELECT 1
			FROM killmail_attackers actor
			WHERE actor.killmail_id = k.killmail_id
			  AND actor.%s = $1
		)`, config.Attacker)}
	if shipGroup != 0 {
		args = append(args, shipGroup)
		parameter := fmt.Sprintf("$%d", len(args))
		lossConditions = append(lossConditions, "k.victim_ship_group_id = "+parameter)
		attackConditions = append(
			attackConditions, "actor_k.victim_ship_group_id = "+parameter,
		)
		factionAttackConditions = append(
			factionAttackConditions, "k.victim_ship_group_id = "+parameter,
		)
	}
	if after != 0 {
		args = append(args, after)
		parameter := fmt.Sprintf("$%d", len(args))
		lossConditions = append(lossConditions, "k.killmail_id < "+parameter)
		factionAttackConditions = append(
			factionAttackConditions, "k.killmail_id < "+parameter,
		)
		attackConditions = append(attackConditions, fmt.Sprintf(`
			(actor.killmail_time < (
				SELECT killmail_time FROM killmails WHERE killmail_id = %[1]s
			) OR (
			 actor.killmail_time = (
			   SELECT killmail_time FROM killmails WHERE killmail_id = %[1]s
			 ) AND actor.killmail_id < %[1]s
			))`, parameter))
	}

	attackerJoin := ""
	if shipGroup != 0 {
		attackerJoin = "JOIN killmails actor_k ON actor_k.killmail_id = actor.killmail_id"
	}

	// Bound each half before UNION and before the detail joins. The old query
	// materialized every historical killmail ID for the entity, deduplicated
	// that complete set, and only then applied LIMIT 51. Large alliances have
	// tens of millions of attacker rows, turning a first page into a 30-second
	// query. Both source indexes are newest-first, so each branch only needs
	// enough candidates to cover the requested offset and page.
	branchLimit := offset + int64(limit) + 1
	args = append(args, branchLimit)
	branchLimitParameter := len(args)
	lossScope := fmt.Sprintf(`
		SELECT k.killmail_id, k.killmail_time
		FROM killmails k
		WHERE %s
		ORDER BY k.killmail_id DESC
		LIMIT $%d`,
		strings.Join(lossConditions, " AND "), branchLimitParameter,
	)
	attackScope := ""
	if config.Attacker == "faction_id" {
		// Unlike character/corporation/alliance, attacker factions have no
		// faction+time index. Walk the newest killmails and probe their indexed
		// attacker rows instead; the semi-join stops as soon as the page is full.
		attackScope = fmt.Sprintf(`
			SELECT k.killmail_id, k.killmail_time
			FROM killmails k
			WHERE %s
			ORDER BY k.killmail_id DESC
			LIMIT $%d`,
			strings.Join(factionAttackConditions, " AND "),
			branchLimitParameter,
		)
	} else {
		attackScope = fmt.Sprintf(`
			SELECT DISTINCT ON (actor.killmail_time, actor.killmail_id)
			       actor.killmail_id, actor.killmail_time
			FROM killmail_attackers actor
			%s
			WHERE %s
			ORDER BY actor.killmail_time DESC, actor.killmail_id DESC
			LIMIT $%d`,
			attackerJoin, strings.Join(attackConditions, " AND "),
			branchLimitParameter,
		)
	}
	scope := ""
	switch role {
	case "losses":
		scope = lossScope
	case "kills":
		scope = attackScope
	default:
		scope = fmt.Sprintf(`
			(%s)
			UNION
			(%s)`, lossScope, attackScope)
	}
	return entityKilllistScopePlan{
		SQL: scope, Args: args, Order: order, Offset: offset, Numbered: numbered,
	}
}

func loadEntityPageKilllist(
	ctx context.Context,
	opts Options,
	kind string,
	id int64,
	req *legacyRequest,
) (any, error) {
	config, ok := entityKilllistConfigs[kind]
	if !ok {
		return nil, apiError(http.StatusBadRequest, "Invalid entity type")
	}
	role := req.Query.Get("role")
	if role == "" {
		role = "combined"
	}
	if role != "kills" && role != "losses" && role != "combined" {
		role = "combined"
	}
	limit := min(max(entityPageQueryInt(req.Query.Get("limit"), 50), 10), 100)
	after := int64(entityPageQueryInt(req.Query.Get("after"), 0))
	page := entityPageQueryInt(req.Query.Get("page"), 0)
	shipGroup := int64(entityPageQueryInt(req.Query.Get("ship_group"), 0))
	scope := buildEntityKilllistScopePlan(
		config, role, id, limit, after, page, shipGroup,
	)
	args := append(scope.Args, limit+1, scope.Offset)
	limitParameter, offsetParameter := len(args)-1, len(args)
	killSQL := fmt.Sprintf(`
		WITH scoped AS MATERIALIZED (%s)
		SELECT k.killmail_id, k.killmail_hash, k.killmail_time,
		       COALESCE(k.total_value, 0)::double precision AS total_value,
		       COALESCE(k.attacker_count, 0)::int AS attacker_count,
		       COALESCE(k.is_npc, false) AS is_npc,
		       COALESCE(k.is_solo, false) AS is_solo,
		       k.victim_ship_type_id AS ship_type_id,
		       ship.name AS ship_name,
		       k.victim_ship_group_id AS ship_group_id,
		       ship_group.name AS ship_group_name,
		       ship.market_group_id AS _ship_market_group_id,
		       ship.meta_group_id,
		       k.victim_character_id, victim_char.name AS victim_character_name,
		       k.victim_corporation_id, victim_corp.name AS victim_corporation_name,
		       k.victim_alliance_id, victim_ally.name AS victim_alliance_name,
		       k.victim_faction_id,
		       final.character_id AS final_blow_character_id,
		       final_char.name AS final_blow_character_name,
		       final.corporation_id AS final_blow_corporation_id,
		       final_corp.name AS final_blow_corporation_name,
		       final.alliance_id AS final_blow_alliance_id,
		       final_ally.name AS final_blow_alliance_name,
		       final.ship_type_id AS final_blow_ship_type_id,
		       final_ship.name AS final_blow_ship_name,
		       k.solar_system_id, system.system_name AS solar_system_name,
		       system.security AS solar_system_security,
		       k.region_id, region.name AS region_name
		FROM scoped
		JOIN killmails k USING (killmail_id)
		LEFT JOIN LATERAL (
			SELECT character_id, corporation_id, alliance_id, ship_type_id
			FROM killmail_attackers
			WHERE killmail_id = k.killmail_id AND final_blow IS TRUE
			ORDER BY attacker_index
			LIMIT 1
		) final ON true
		LEFT JOIN inv_types ship ON ship.type_id = k.victim_ship_type_id
		LEFT JOIN inv_groups ship_group
		  ON ship_group.group_id = k.victim_ship_group_id
		LEFT JOIN characters victim_char
		  ON victim_char.character_id = k.victim_character_id
		LEFT JOIN corporations victim_corp
		  ON victim_corp.corporation_id = k.victim_corporation_id
		LEFT JOIN alliances victim_ally
		  ON victim_ally.alliance_id = k.victim_alliance_id
		LEFT JOIN characters final_char
		  ON final_char.character_id = final.character_id
		LEFT JOIN corporations final_corp
		  ON final_corp.corporation_id = final.corporation_id
		LEFT JOIN alliances final_ally
		  ON final_ally.alliance_id = final.alliance_id
		LEFT JOIN inv_types final_ship ON final_ship.type_id = final.ship_type_id
		LEFT JOIN solar_systems system
		  ON system.solar_system_id = k.solar_system_id
		LEFT JOIN regions region ON region.region_id = k.region_id
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		scope.SQL, scope.Order, limitParameter, offsetParameter,
	)

	queries := []databaseQuery{{SQL: killSQL, Args: args}}
	includeTotal := config.EntityType >= 0 && shipGroup == 0
	if includeTotal {
		countColumn := "kills + losses"
		switch role {
		case "kills":
			countColumn = "kills"
		case "losses":
			countColumn = "losses"
		}
		queries = append(queries, databaseQuery{
			SQL: fmt.Sprintf(`
				SELECT COALESCE(SUM(%s), 0)::bigint AS total
				FROM stats
				WHERE entity_type = $1 AND entity_id = $2
				  AND period_type = 0`, countColumn),
			Args: []any{config.EntityType, id},
		})
	}
	results, err := queryMapsConcurrent(ctx, opts.DB, queries...)
	if err != nil {
		return nil, err
	}
	rows := results[0]
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	paths, err := loadEntityKilllistMarketPaths(ctx, opts.DB, rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		groupID := int64OrZero(row["_ship_market_group_id"])
		row["ship_market_path"] = paths[groupID]
		delete(row, "_ship_market_group_id")
	}
	var cursor any
	if len(rows) > 0 && !scope.Numbered {
		cursor = rows[len(rows)-1]["killmail_id"]
	}
	response := map[string]any{
		"kills": rows, "hasMore": hasMore, "cursor": cursor,
	}
	if includeTotal {
		total := int64OrZero(firstOrEmpty(results[1])["total"])
		if total > 0 {
			totalPages := max(int64(math.Ceil(float64(total)/float64(limit))), 1)
			response["totalPages"] = totalPages
			if scope.Numbered {
				response["hasMore"] = int64(page) < totalPages
				response["cursor"] = nil
			}
		}
	}
	return response, nil
}

func loadEntityKilllistMarketPaths(
	ctx context.Context,
	db Database,
	rows []map[string]any,
) (map[int64]any, error) {
	ids := make(map[int64]bool)
	for _, row := range rows {
		if id := int64OrZero(row["_ship_market_group_id"]); id > 0 {
			ids[id] = true
		}
	}
	out := make(map[int64]any, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rootIDs := sortedMapKeys(ids)
	pathRows, err := queryMaps(ctx, db, `
		WITH RECURSIVE ancestors AS (
			SELECT mg.market_group_id AS root_id,
			       mg.market_group_id, mg.parent_group_id, mg.name,
			       ARRAY[mg.market_group_id]::int[] AS visited, 1 AS depth
			FROM inv_market_groups mg
			WHERE mg.market_group_id = ANY($1::int[])
			UNION ALL
			SELECT child.root_id, parent.market_group_id,
			       parent.parent_group_id, parent.name,
			       child.visited || parent.market_group_id,
			       child.depth + 1
			FROM inv_market_groups parent
			JOIN ancestors child
			  ON parent.market_group_id = child.parent_group_id
			WHERE child.depth < 16
			  AND NOT parent.market_group_id = ANY(child.visited)
		)
		SELECT root_id, name, depth
		FROM ancestors
		ORDER BY root_id, depth DESC`, rootIDs)
	if err != nil {
		return nil, err
	}
	segments := make(map[int64][]string, len(ids))
	for _, row := range pathRows {
		rootID := int64OrZero(row["root_id"])
		segments[rootID] = append(
			segments[rootID], eve.Slugify(stringOrEmpty(row["name"])),
		)
	}
	for id, values := range segments {
		out[id] = "/market/" + strings.Join(values, "/")
	}
	return out, nil
}
