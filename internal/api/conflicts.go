package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/eve"
	"golang.org/x/sync/errgroup"
)

const (
	conflictDefaultPage       = 1
	conflictMaximumPage       = 10000
	conflictDefaultLimit      = 50
	conflictMaximumLimit      = 100
	conflictMaximumMembers    = 500
	conflictMaximumEligible   = 500
	conflictMaximumSystems    = 5
	conflictMaximumBodyBytes  = 2 << 20
	conflictShipCategory      = 6
	conflictStructureCategory = 65
)

// registerConflictRoutes installs the site conflict surface. Saved
// and on-the-fly battle aliases deliberately share the same window handlers;
// only the way a window and team roster are resolved differs.
func registerConflictRoutes(a huma.API, opts Options) {
	registerConflictWarRoutes(a, opts)
	registerConflictBattleRoutes(a, opts)
	registerBattleAnalysisRoutes(a, opts)
	registerBattleGeneratorRoutes(a, opts)
}

func registerConflictWarRoutes(a huma.API, opts Options) {
	// /wars is the established cursor API. Keep its contract stable and expose
	// the richer page/dashboard listing under the conflict domain instead of
	// silently changing the meaning of the public endpoint.
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "conflict-wars",
		Path:        "/conflicts/wars",
		Summary:     "War dashboard listing",
		Tags:        []string{"wars"},
	}, time.Minute, conflictWarsHandler(opts))
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "wars-overview-stats",
		Path:        "/wars/stats",
		Summary:     "War overview counters",
		Tags:        []string{"wars"},
	}, 5*time.Minute, warCountersHandler(opts))
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "wars-eligible",
		Path:        "/wars/eligible",
		Summary:     "War-eligible corporations and alliances",
		Tags:        []string{"wars"},
	}, 10*time.Minute, warEligibleHandler(opts))
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "war-dashboard-detail",
		Path:        "/war/{id}",
		Summary:     "War dashboard detail",
		Tags:        []string{"wars"},
	}, 2*time.Minute, warDetailHandler(opts))
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "war-dashboard",
		Path:        "/war/{id}/dashboard",
		Summary:     "Complete war dashboard",
		Tags:        []string{"wars"},
	}, 2*time.Minute, warDashboardHandler(opts))
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "war-leaderboards",
		Path:        "/war/{id}/stats",
		Summary:     "War leaderboards and sides",
		Tags:        []string{"wars"},
	}, 2*time.Minute, warStatsHandler(opts))
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "war-members",
		Path:        "/war/{id}/members",
		Summary:     "Active war members",
		Tags:        []string{"wars"},
	}, 5*time.Minute, warMembersHandler(opts))
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "war-intel",
		Path:        "/war/{id}/intel",
		Summary:     "War activity intelligence",
		Tags:        []string{"wars"},
	}, 5*time.Minute, warIntelHandler(opts))
	registerConflictCachedGET(a, opts, huma.Operation{
		OperationID: "war-killlist",
		Path:        "/war/{id}/killlist",
		Summary:     "War killmails",
		Tags:        []string{"wars", "killmails"},
	}, 2*time.Minute, warKilllistHandler(opts))
}

func registerConflictCachedGET(
	a huma.API,
	opts Options,
	op huma.Operation,
	ttl time.Duration,
	handler legacyHandler,
) {
	op.Method = http.MethodGet
	cacheControl := fmt.Sprintf(
		"public, max-age=30, s-maxage=%d, stale-while-revalidate=60",
		int(ttl/time.Second),
	)
	registerLegacy(a, op, routeJSONCacheBy(
		opts,
		ttl,
		cacheControl,
		conflictCacheKey,
		handler,
	))
}

func conflictCacheKey(req *legacyRequest) string {
	host := conflictRequestHost(req)
	u := req.Huma.URL()
	return host + u.RequestURI()
}

func conflictRequestHost(req *legacyRequest) string {
	return normalizeRequestHost(legacyRequestHost(req))
}

func normalizeRequestHost(raw string) string {
	host := raw
	host = strings.ToLower(strings.TrimSpace(host))
	if parsed, err := netSplitHostPortLoose(host); err == nil {
		host = parsed
	}
	if len(host) > 255 {
		host = host[:255]
	}
	return host
}

// customDomainHostQuery centralizes the tenant-host interpretation shared by
// site bootstrap, campaigns, conflict views, and the custom killboard.
//
// domainHost remains true for malformed or unknown custom-domain-looking
// hosts so the renderer can noindex them. An empty predicate means there is
// intentionally no database lookup to perform.
func customDomainHostQuery(host string) (
	predicate string,
	value string,
	domainHost bool,
) {
	if !isPossibleCustomDomain(host) {
		return "", "", false
	}
	switch {
	case strings.HasSuffix(host, ".eve-kill.com"):
		value = strings.TrimSuffix(host, ".eve-kill.com")
		if value == "" || strings.Contains(value, ".") {
			return "", "", true
		}
		return "domain.subdomain = $1", value, true
	case strings.HasSuffix(host, ".localhost"):
		value = strings.TrimSuffix(host, ".localhost")
		if value == "" || strings.Contains(value, ".") {
			return "", "", true
		}
		return "domain.subdomain = $1", value, true
	default:
		return "LOWER(domain.custom_hostname) = $1", host, true
	}
}

func netSplitHostPortLoose(host string) (string, error) {
	// Avoid importing net just to handle the overwhelmingly common host:port
	// form. Bracketed IPv6 is never a custom domain and may remain unchanged.
	if strings.Count(host, ":") != 1 {
		return host, nil
	}
	name, port, found := strings.Cut(host, ":")
	if !found {
		return host, nil
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return host, err
	}
	return name, nil
}

type conflictEntityScope struct {
	Characters   []int32
	Corporations []int32
	Alliances    []int32
}

func (s *conflictEntityScope) empty() bool {
	return len(s.Characters) == 0 &&
		len(s.Corporations) == 0 &&
		len(s.Alliances) == 0
}

func loadConflictDomainScope(
	ctx context.Context,
	db Database,
	req *legacyRequest,
) (*conflictEntityScope, error) {
	host := conflictRequestHost(req)
	if !isPossibleCustomDomain(host) {
		return nil, nil
	}

	predicate, key, domainHost := customDomainHostQuery(host)
	if !domainHost || predicate == "" {
		return nil, nil
	}
	var query string
	if predicate == "domain.subdomain = $1" {
		query = `
			SELECT entity->>'type' AS entity_type,
			       CASE WHEN entity->>'id' ~ '^[0-9]+$'
			            THEN (entity->>'id')::bigint END AS entity_id
			FROM custom_domains domain
			LEFT JOIN LATERAL jsonb_array_elements(domain.entities) entity ON true
			WHERE domain.active IS TRUE AND domain.subdomain = $1`
	} else {
		query = `
			SELECT entity->>'type' AS entity_type,
			       CASE WHEN entity->>'id' ~ '^[0-9]+$'
			            THEN (entity->>'id')::bigint END AS entity_id
			FROM custom_domains domain
			LEFT JOIN LATERAL jsonb_array_elements(domain.entities) entity ON true
			WHERE domain.active IS TRUE
			  AND LOWER(domain.custom_hostname) = $1`
	}
	rows, err := queryMaps(ctx, db, query, key)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	scope := &conflictEntityScope{}
	for _, row := range rows {
		value, ok := int64Value(row["entity_id"])
		if !ok || value <= 0 || value > pgInt4Max {
			continue
		}
		id := int32(value)
		switch conflictString(row, "entity_type") {
		case "character":
			scope.Characters = append(scope.Characters, id)
		case "corporation":
			scope.Corporations = append(scope.Corporations, id)
		case "alliance":
			scope.Alliances = append(scope.Alliances, id)
		}
	}
	scope.Characters = uniqueConflictIDs(scope.Characters)
	scope.Corporations = uniqueConflictIDs(scope.Corporations)
	scope.Alliances = uniqueConflictIDs(scope.Alliances)
	return scope, nil
}

func isPossibleCustomDomain(host string) bool {
	switch host {
	case "", "eve-kill.com", "www.eve-kill.com",
		"zkillboard.co", "www.zkillboard.co",
		"localhost", "127.0.0.1", "::1":
		return false
	}
	if !strings.Contains(host, ".") {
		return false
	}
	parts := strings.Split(host, ".")
	if len(parts) == 4 {
		ipv4 := true
		for _, part := range parts {
			value, err := strconv.Atoi(part)
			if err != nil || value < 0 || value > 255 {
				ipv4 = false
				break
			}
		}
		if ipv4 {
			return false
		}
	}
	return true
}

func uniqueConflictIDs(ids []int32) []int32 {
	seen := make(map[int32]struct{}, len(ids))
	result := make([]int32, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func resolveConflictScopeCharacters(
	ctx context.Context,
	db Database,
	scope *conflictEntityScope,
) error {
	if scope == nil || len(scope.Characters) == 0 {
		return nil
	}
	rows, err := queryMaps(ctx, db, `
		SELECT corporation_id, alliance_id
		FROM characters
		WHERE character_id = ANY($1::int[])`,
		scope.Characters,
	)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if value, ok := int64Value(row["corporation_id"]); ok &&
			value > 0 && value <= pgInt4Max {
			scope.Corporations = append(scope.Corporations, int32(value))
		}
		if value, ok := int64Value(row["alliance_id"]); ok &&
			value > 0 && value <= pgInt4Max {
			scope.Alliances = append(scope.Alliances, int32(value))
		}
	}
	scope.Corporations = uniqueConflictIDs(scope.Corporations)
	scope.Alliances = uniqueConflictIDs(scope.Alliances)
	return nil
}

func parseConflictBoundedInt(
	query url.Values,
	name string,
	fallback, minimum, maximum int,
) int {
	return parseFactionWarBoundedInt(
		query.Get(name),
		fallback,
		minimum,
		maximum,
	)
}

func parseConflictOptionalID(query url.Values, name string) (*int32, error) {
	raw := strings.TrimSpace(query.Get(name))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value <= 0 {
		return nil, apiError(http.StatusBadRequest, "Invalid "+name)
	}
	id := int32(value)
	return &id, nil
}

func parseConflictID(req *legacyRequest) (int32, error) {
	id, err := parseID(req.Param("id"))
	if err != nil || id <= 0 || id > pgInt4Max {
		return 0, apiError(http.StatusBadRequest, "Invalid ID")
	}
	return int32(id), nil
}

func decodeConflictBody(req *legacyRequest, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(req.Body, conflictMaximumBodyBytes+1))
	if err := decoder.Decode(destination); err != nil {
		return apiError(http.StatusBadRequest, "Invalid request body")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return apiError(http.StatusBadRequest, "Invalid request body")
	}
	return nil
}

func conflictInt(row map[string]any, key string) int64 {
	value, _ := int64Value(row[key])
	return value
}

func conflictFloat(row map[string]any, key string) float64 {
	value, _ := float64Value(row[key])
	return value
}

func conflictString(row map[string]any, key string) string {
	value, _ := stringValue(row[key])
	return value
}

func conflictNullableID(row map[string]any, key string) any {
	if row[key] == nil {
		return nil
	}
	value, ok := int64Value(row[key])
	if !ok || value == 0 {
		return nil
	}
	return value
}

func conflictNullableString(row map[string]any, key string) any {
	if row[key] == nil {
		return nil
	}
	value, ok := stringValue(row[key])
	if !ok {
		return nil
	}
	return value
}

func conflictNullableFloat(row map[string]any, key string) any {
	if row[key] == nil {
		return nil
	}
	value, ok := float64Value(row[key])
	if !ok {
		return nil
	}
	return value
}

func conflictWarsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		page := parseConflictBoundedInt(
			req.Query, "page", conflictDefaultPage, 1, conflictMaximumPage,
		)
		limit := parseConflictBoundedInt(
			req.Query, "limit", conflictDefaultLimit, 1, conflictMaximumLimit,
		)
		allianceID, err := parseConflictOptionalID(req.Query, "allianceId")
		if err != nil {
			return legacyPayload{}, err
		}
		corporationID, err := parseConflictOptionalID(req.Query, "corporationId")
		if err != nil {
			return legacyPayload{}, err
		}
		characterID, err := parseConflictOptionalID(req.Query, "characterId")
		if err != nil {
			return legacyPayload{}, err
		}

		scope, err := loadConflictDomainScope(ctx, opts.DB, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if err := resolveConflictScopeCharacters(ctx, opts.DB, scope); err != nil {
			return legacyPayload{}, err
		}
		if scope != nil && scope.empty() {
			return conflictEmptyWars(page, limit), nil
		}

		entityScope := &conflictEntityScope{}
		if allianceID != nil {
			entityScope.Alliances = append(entityScope.Alliances, *allianceID)
		}
		if corporationID != nil {
			entityScope.Corporations = append(entityScope.Corporations, *corporationID)
		}
		if characterID != nil && allianceID == nil && corporationID == nil {
			row, err := queryMap(ctx, opts.DB, `
				SELECT corporation_id, alliance_id
				FROM characters
				WHERE character_id = $1
				LIMIT 1`,
				*characterID,
			)
			if err != nil {
				return legacyPayload{}, err
			}
			if row == nil {
				return conflictEmptyWars(page, limit), nil
			}
			if value, ok := int64Value(row["corporation_id"]); ok && value > 0 {
				entityScope.Corporations = append(entityScope.Corporations, int32(value))
			}
			if value, ok := int64Value(row["alliance_id"]); ok && value > 0 {
				entityScope.Alliances = append(entityScope.Alliances, int32(value))
			}
			if entityScope.empty() {
				return conflictEmptyWars(page, limit), nil
			}
		}

		where := make([]string, 0, 10)
		args := make([]any, 0, 12)
		if scope != nil {
			where = append(where, conflictWarScopeCondition(scope, &args))
		}
		if !entityScope.empty() {
			where = append(where, conflictWarScopeCondition(entityScope, &args))
		}
		upcoming := req.Query.Get("upcoming") == "true"
		finishedOnly := req.Query.Get("finished") == "true"
		if upcoming {
			where = append(where, "w.started > NOW()")
		} else if finishedOnly {
			where = append(where, "w.finished IS NOT NULL")
		} else {
			where = append(where, "(w.started IS NULL OR w.started <= NOW())")
		}
		if req.Query.Get("ongoing") == "true" {
			where = append(where, "w.finished IS NULL")
		}
		if req.Query.Get("mutual") == "true" {
			where = append(where, "w.mutual IS TRUE")
		}
		if req.Query.Get("hasActivity") == "true" {
			where = append(where,
				"(COALESCE(w.aggressor_isk_destroyed, 0) > 0 OR "+
					"COALESCE(w.defender_isk_destroyed, 0) > 0)")
		}
		if req.Query.Get("hasKills") == "true" {
			where = append(where,
				"(COALESCE(w.aggressor_ships_killed, 0) > 0 OR "+
					"COALESCE(w.defender_ships_killed, 0) > 0)")
		}
		if req.Query.Get("hasAllies") == "true" {
			where = append(where,
				"EXISTS (SELECT 1 FROM war_allies wa WHERE wa.war_id = w.war_id)")
		}

		orderBy := "w.war_id DESC"
		if upcoming {
			orderBy = "w.started ASC"
		} else {
			switch req.Query.Get("sort") {
			case "kills":
				orderBy = `(COALESCE(w.aggressor_ships_killed, 0) +
					COALESCE(w.defender_ships_killed, 0)) DESC, w.war_id DESC`
			case "isk":
				orderBy = `(COALESCE(w.aggressor_isk_destroyed, 0) +
					COALESCE(w.defender_isk_destroyed, 0)) DESC, w.war_id DESC`
			default:
				if finishedOnly {
					orderBy = "w.finished DESC NULLS LAST, w.war_id DESC"
				}
			}
		}

		args = append(args, limit, (page-1)*limit)
		query := `
			SELECT w.war_id, w.declared, w.started, w.finished,
			       w.mutual, w.open_for_allies,
			       w.aggressor_alliance_id, w.aggressor_corporation_id,
			       COALESCE(w.aggressor_isk_destroyed, 0)::double precision
			           AS aggressor_isk_destroyed,
			       COALESCE(w.aggressor_ships_killed, 0)::bigint
			           AS aggressor_ships_killed,
			       w.defender_alliance_id, w.defender_corporation_id,
			       COALESCE(w.defender_isk_destroyed, 0)::double precision
			           AS defender_isk_destroyed,
			       COALESCE(w.defender_ships_killed, 0)::bigint
			           AS defender_ships_killed,
			       aa.name AS aggressor_alliance_name,
			       aa.ticker AS aggressor_alliance_ticker,
			       ac.name AS aggressor_corporation_name,
			       ac.ticker AS aggressor_corporation_ticker,
			       da.name AS defender_alliance_name,
			       da.ticker AS defender_alliance_ticker,
			       dc.name AS defender_corporation_name,
			       dc.ticker AS defender_corporation_ticker
			FROM wars w
			LEFT JOIN alliances aa ON aa.alliance_id = w.aggressor_alliance_id
			LEFT JOIN corporations ac ON ac.corporation_id = w.aggressor_corporation_id
			LEFT JOIN alliances da ON da.alliance_id = w.defender_alliance_id
			LEFT JOIN corporations dc ON dc.corporation_id = w.defender_corporation_id
			WHERE ` + strings.Join(where, " AND ") +
			` ORDER BY ` + orderBy +
			fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		wars := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			wars = append(wars, map[string]any{
				"war_id":          conflictInt(row, "war_id"),
				"declared":        row["declared"],
				"started":         row["started"],
				"finished":        row["finished"],
				"mutual":          falseIfNil(row["mutual"]),
				"open_for_allies": falseIfNil(row["open_for_allies"]),
				"aggressor":       conflictWarEntityFromRow(row, "aggressor"),
				"defender":        conflictWarEntityFromRow(row, "defender"),
			})
		}
		return jsonPayload(map[string]any{
			"wars": wars, "page": page, "limit": limit,
		}), nil
	}
}

func conflictEmptyWars(page, limit int) legacyPayload {
	return jsonPayload(map[string]any{
		"wars": []map[string]any{}, "page": page, "limit": limit,
	})
}

func conflictWarScopeCondition(
	scope *conflictEntityScope,
	args *[]any,
) string {
	parts := make([]string, 0, 6)
	if len(scope.Alliances) > 0 {
		*args = append(*args, scope.Alliances)
		n := len(*args)
		parts = append(parts,
			fmt.Sprintf("w.aggressor_alliance_id = ANY($%d::int[])", n),
			fmt.Sprintf("w.defender_alliance_id = ANY($%d::int[])", n),
			fmt.Sprintf(`EXISTS (
				SELECT 1 FROM war_allies scoped_wa
				WHERE scoped_wa.war_id = w.war_id
				  AND scoped_wa.alliance_id = ANY($%d::int[])
			)`, n),
		)
	}
	if len(scope.Corporations) > 0 {
		*args = append(*args, scope.Corporations)
		n := len(*args)
		parts = append(parts,
			fmt.Sprintf("w.aggressor_corporation_id = ANY($%d::int[])", n),
			fmt.Sprintf("w.defender_corporation_id = ANY($%d::int[])", n),
			fmt.Sprintf(`EXISTS (
				SELECT 1 FROM war_allies scoped_wa
				WHERE scoped_wa.war_id = w.war_id
				  AND scoped_wa.corporation_id = ANY($%d::int[])
			)`, n),
		)
	}
	if len(parts) == 0 {
		return "FALSE"
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

func conflictWarEntityFromRow(row map[string]any, prefix string) map[string]any {
	allianceID, alliance := int64Value(row[prefix+"_alliance_id"])
	corporationID, corporation := int64Value(row[prefix+"_corporation_id"])
	entity := map[string]any{
		"id":            int64(0),
		"name":          "Unknown",
		"ticker":        "?",
		"type":          "corporation",
		"isk_destroyed": conflictFloat(row, prefix+"_isk_destroyed"),
		"ships_killed":  conflictInt(row, prefix+"_ships_killed"),
	}
	nameKey := prefix + "_corporation_name"
	tickerKey := prefix + "_corporation_ticker"
	if alliance && allianceID != 0 {
		entity["id"] = allianceID
		entity["type"] = "alliance"
		nameKey = prefix + "_alliance_name"
		tickerKey = prefix + "_alliance_ticker"
	} else if corporation && corporationID != 0 {
		entity["id"] = corporationID
	}
	if name := conflictString(row, nameKey); name != "" {
		entity["name"] = name
	}
	if ticker := conflictString(row, tickerKey); ticker != "" {
		entity["ticker"] = ticker
	}
	return entity
}

func warCountersHandler(opts Options) legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		row, err := queryMap(ctx, opts.DB, `
			SELECT
				(SELECT COUNT(*)::int FROM wars
				 WHERE finished IS NULL AND (started IS NULL OR started <= NOW()))
				    AS active_wars,
				(SELECT COUNT(*)::int FROM wars WHERE finished IS NOT NULL)
				    AS finished_wars,
				(SELECT COUNT(*)::int FROM wars WHERE started > NOW())
				    AS upcoming_wars,
				(SELECT COUNT(*)::int FROM corporations
				 WHERE war_eligible IS TRUE AND corporation_id > 1999999
				   AND deleted IS NOT TRUE) AS eligible_corps,
				(SELECT COUNT(*)::int FROM alliances alliance
				 WHERE alliance.deleted IS NOT TRUE
				   AND EXISTS (
					   SELECT 1 FROM corporations corporation
					   WHERE corporation.alliance_id = alliance.alliance_id
					     AND corporation.war_eligible IS TRUE
					     AND corporation.deleted IS NOT TRUE
				   )) AS eligible_alliances`)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"activeWars":        conflictInt(row, "active_wars"),
			"finishedWars":      conflictInt(row, "finished_wars"),
			"upcomingWars":      conflictInt(row, "upcoming_wars"),
			"eligibleCorps":     conflictInt(row, "eligible_corps"),
			"eligibleAlliances": conflictInt(row, "eligible_alliances"),
		}), nil
	}
}

func warEligibleHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		kind := "corporations"
		entityType := entityCorporation
		if req.Query.Get("type") == "alliances" {
			kind = "alliances"
			entityType = entityAlliance
		}
		fromDate := time.Now().UTC().AddDate(0, 0, -90).Format("2006-01-02")
		var query string
		if kind == "corporations" {
			query = `
				WITH candidates AS MATERIALIZED (
					SELECT corporation_id AS id, name, ticker,
					       COALESCE(member_count, 0)::int AS member_count,
					       alliance_id
					FROM corporations
					WHERE war_eligible IS TRUE
					  AND corporation_id > 1999999
					  AND deleted IS NOT TRUE
					ORDER BY member_count DESC NULLS LAST, corporation_id
					LIMIT 500
				),
				aggregated AS (
					SELECT entity_id,
					       COALESCE(SUM(kills), 0)::bigint AS kills,
					       COALESCE(SUM(losses), 0)::bigint AS losses,
					       COALESCE(SUM(isk_destroyed), 0)::double precision
					           AS isk_destroyed,
					       COALESCE(SUM(isk_lost), 0)::double precision AS isk_lost
					FROM stats
					WHERE entity_type = $1 AND period_type = 0
					  AND period_start >= $2::date
					  AND entity_id IN (SELECT id FROM candidates)
					GROUP BY entity_id
				)
				SELECT candidate.*, alliance.name AS alliance_name,
				       COALESCE(stat.kills, 0)::bigint AS kills,
				       COALESCE(stat.losses, 0)::bigint AS losses,
				       COALESCE(stat.isk_destroyed, 0)::double precision
				           AS isk_destroyed,
				       COALESCE(stat.isk_lost, 0)::double precision AS isk_lost
				FROM candidates candidate
				LEFT JOIN alliances alliance
				  ON alliance.alliance_id = candidate.alliance_id
				LEFT JOIN aggregated stat ON stat.entity_id = candidate.id
				ORDER BY candidate.member_count DESC, candidate.id`
		} else {
			query = `
				WITH latest_snap AS (
					SELECT MAX(date) AS date
					FROM entity_snapshots
					WHERE entity_type = 'alliance'
				),
				candidates AS MATERIALIZED (
					SELECT alliance.alliance_id AS id,
					       alliance.name, alliance.ticker,
					       (SELECT COUNT(*)::int
					        FROM corporations corporation
					        WHERE corporation.alliance_id = alliance.alliance_id
					          AND corporation.deleted IS NOT TRUE)
					           AS corporation_count,
					       COALESCE(snapshot.member_count, 0)::int AS member_count
					FROM alliances alliance
					LEFT JOIN entity_snapshots snapshot
					  ON snapshot.entity_type = 'alliance'
					 AND snapshot.entity_id = alliance.alliance_id
					 AND snapshot.date = (SELECT date FROM latest_snap)
					WHERE alliance.deleted IS NOT TRUE
					  AND EXISTS (
						  SELECT 1 FROM corporations corporation
						  WHERE corporation.alliance_id = alliance.alliance_id
						    AND corporation.war_eligible IS TRUE
						    AND corporation.deleted IS NOT TRUE
					  )
					ORDER BY member_count DESC NULLS LAST, alliance.alliance_id
					LIMIT 500
				),
				aggregated AS (
					SELECT entity_id,
					       COALESCE(SUM(kills), 0)::bigint AS kills,
					       COALESCE(SUM(losses), 0)::bigint AS losses,
					       COALESCE(SUM(isk_destroyed), 0)::double precision
					           AS isk_destroyed,
					       COALESCE(SUM(isk_lost), 0)::double precision AS isk_lost
					FROM stats
					WHERE entity_type = $1 AND period_type = 0
					  AND period_start >= $2::date
					  AND entity_id IN (SELECT id FROM candidates)
					GROUP BY entity_id
				)
				SELECT candidate.*,
				       COALESCE(stat.kills, 0)::bigint AS kills,
				       COALESCE(stat.losses, 0)::bigint AS losses,
				       COALESCE(stat.isk_destroyed, 0)::double precision
				           AS isk_destroyed,
				       COALESCE(stat.isk_lost, 0)::double precision AS isk_lost
				FROM candidates candidate
				LEFT JOIN aggregated stat ON stat.entity_id = candidate.id
				ORDER BY candidate.member_count DESC, candidate.id`
		}
		rows, err := queryMaps(ctx, opts.DB, query, entityType, fromDate)
		if err != nil {
			return legacyPayload{}, err
		}
		entries := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			name := conflictString(row, "name")
			if name == "" {
				name = "Unknown"
			}
			entry := map[string]any{
				"id":            conflictInt(row, "id"),
				"name":          name,
				"ticker":        conflictString(row, "ticker"),
				"member_count":  conflictInt(row, "member_count"),
				"kills":         conflictInt(row, "kills"),
				"losses":        conflictInt(row, "losses"),
				"isk_destroyed": conflictFloat(row, "isk_destroyed"),
				"isk_lost":      conflictFloat(row, "isk_lost"),
			}
			if kind == "corporations" {
				entry["alliance_id"] = conflictNullableID(row, "alliance_id")
				entry["alliance_name"] = conflictNullableString(row, "alliance_name")
			} else {
				entry["corporation_count"] = conflictInt(row, "corporation_count")
			}
			entries = append(entries, entry)
		}
		return jsonPayload(map[string]any{
			"entries": entries,
			"type":    kind,
			"limit":   conflictMaximumEligible,
		}), nil
	}
}

type conflictWarSides struct {
	War             map[string]any
	AggressorCorps  []int32
	AggressorAllies []int32
	DefenderCorps   []int32
	DefenderAllies  []int32
}

func loadConflictWarSides(
	ctx context.Context,
	db Database,
	warID int32,
) (conflictWarSides, error) {
	rows, err := queryMapsConcurrent(
		ctx,
		db,
		databaseQuery{
			SQL: `
				SELECT war.*,
				       aa.name AS aggressor_alliance_name,
				       aa.ticker AS aggressor_alliance_ticker,
				       ac.name AS aggressor_corporation_name,
				       ac.ticker AS aggressor_corporation_ticker,
				       da.name AS defender_alliance_name,
				       da.ticker AS defender_alliance_ticker,
				       dc.name AS defender_corporation_name,
				       dc.ticker AS defender_corporation_ticker
				FROM wars war
				LEFT JOIN alliances aa
				  ON aa.alliance_id = war.aggressor_alliance_id
				LEFT JOIN corporations ac
				  ON ac.corporation_id = war.aggressor_corporation_id
				LEFT JOIN alliances da
				  ON da.alliance_id = war.defender_alliance_id
				LEFT JOIN corporations dc
				  ON dc.corporation_id = war.defender_corporation_id
				WHERE war.war_id = $1
				LIMIT 1`,
			Args: []any{warID},
		},
		databaseQuery{
			SQL: `
				SELECT alliance_id, corporation_id
				FROM war_allies
				WHERE war_id = $1`,
			Args: []any{warID},
		},
	)
	if err != nil {
		return conflictWarSides{}, err
	}
	if len(rows[0]) == 0 {
		return conflictWarSides{}, apiError(http.StatusNotFound, "War not found")
	}
	war := rows[0][0]
	result := conflictWarSides{War: war}
	if value, ok := int64Value(war["aggressor_corporation_id"]); ok && value > 0 {
		result.AggressorCorps = append(result.AggressorCorps, int32(value))
	}
	if value, ok := int64Value(war["aggressor_alliance_id"]); ok && value > 0 {
		result.AggressorAllies = append(result.AggressorAllies, int32(value))
	}
	if value, ok := int64Value(war["defender_corporation_id"]); ok && value > 0 {
		result.DefenderCorps = append(result.DefenderCorps, int32(value))
	}
	if value, ok := int64Value(war["defender_alliance_id"]); ok && value > 0 {
		result.DefenderAllies = append(result.DefenderAllies, int32(value))
	}
	for _, ally := range rows[1] {
		if value, ok := int64Value(ally["corporation_id"]); ok && value > 0 {
			result.DefenderCorps = append(result.DefenderCorps, int32(value))
		}
		if value, ok := int64Value(ally["alliance_id"]); ok && value > 0 {
			result.DefenderAllies = append(result.DefenderAllies, int32(value))
		}
	}
	result.AggressorCorps = uniqueConflictIDs(result.AggressorCorps)
	result.AggressorAllies = uniqueConflictIDs(result.AggressorAllies)
	result.DefenderCorps = uniqueConflictIDs(result.DefenderCorps)
	result.DefenderAllies = uniqueConflictIDs(result.DefenderAllies)
	return result, nil
}

func warDetailHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		warID, err := parseConflictID(req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := loadConflictWarDetail(ctx, opts.DB, warID)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(body), nil
	}
}

func loadConflictWarDetail(
	ctx context.Context,
	db Database,
	warID int32,
) (map[string]any, error) {
	sides, err := loadConflictWarSides(ctx, db, warID)
	if err != nil {
		return nil, err
	}
	since := time.Date(2003, 1, 1, 0, 0, 0, 0, time.UTC)
	if value, ok := sides.War["started"].(time.Time); ok {
		since = value
	} else if value, ok := sides.War["declared"].(time.Time); ok {
		since = value
	}
	rows, err := queryMapsConcurrent(
		ctx,
		db,
		databaseQuery{
			SQL: `
				SELECT wa.alliance_id, wa.corporation_id,
				       COALESCE(alliance.name, corporation.name, 'Unknown') AS name,
				       COALESCE(MAX(interaction.count)
				           FILTER (WHERE interaction.category = 1), 0)::bigint AS kills,
				       COALESCE(MAX(interaction.isk_value)
				           FILTER (WHERE interaction.category = 1), 0)::double precision
				           AS isk_destroyed,
				       COALESCE(MAX(interaction.count)
				           FILTER (WHERE interaction.category = 0), 0)::bigint AS losses,
				       COALESCE(MAX(interaction.isk_value)
				           FILTER (WHERE interaction.category = 0), 0)::double precision
				           AS isk_lost
				FROM war_allies wa
				LEFT JOIN alliances alliance ON alliance.alliance_id = wa.alliance_id
				LEFT JOIN corporations corporation
				  ON corporation.corporation_id = wa.corporation_id
				LEFT JOIN war_interactions interaction
				  ON interaction.war_id = wa.war_id
				 AND interaction.side = 1
				 AND interaction.category IN (0, 1)
				 AND (
					   (wa.alliance_id IS NOT NULL
					    AND interaction.target_type = 2
					    AND interaction.target_id = wa.alliance_id)
					OR (wa.corporation_id IS NOT NULL
					    AND interaction.target_type = 1
					    AND interaction.target_id = wa.corporation_id)
				 )
				WHERE wa.war_id = $1
				GROUP BY wa.id, wa.alliance_id, wa.corporation_id,
				         alliance.name, corporation.name
				ORDER BY wa.id`,
			Args: []any{warID},
		},
		databaseQuery{
			SQL: `
				SELECT COUNT(*)::bigint AS total_kills,
				       COALESCE(SUM(total_value), 0)::double precision AS total_value
				FROM killmails
				WHERE war_id = $1 AND killmail_time >= $2`,
			Args: []any{warID, since},
		},
		databaseQuery{
			SQL: `
				SELECT k.victim_ship_type_id AS ship_type_id,
				       COALESCE(type.name, 'Unknown') AS ship_name,
				       COUNT(*)::bigint AS count
				FROM killmails k
				LEFT JOIN inv_types type ON type.type_id = k.victim_ship_type_id
				WHERE k.war_id = $1 AND k.killmail_time >= $2
				GROUP BY k.victim_ship_type_id, type.name
				ORDER BY count DESC, k.victim_ship_type_id
				LIMIT 10`,
			Args: []any{warID, since},
		},
	)
	if err != nil {
		return nil, err
	}
	allies := make([]map[string]any, 0, len(rows[0]))
	for _, row := range rows[0] {
		kind := "corporation"
		id := conflictInt(row, "corporation_id")
		if row["alliance_id"] != nil {
			kind = "alliance"
			id = conflictInt(row, "alliance_id")
		}
		allies = append(allies, map[string]any{
			"id":            id,
			"name":          conflictString(row, "name"),
			"type":          kind,
			"kills":         conflictInt(row, "kills"),
			"losses":        conflictInt(row, "losses"),
			"isk_destroyed": conflictFloat(row, "isk_destroyed"),
			"isk_lost":      conflictFloat(row, "isk_lost"),
		})
	}
	topShips := make([]map[string]any, 0, len(rows[2]))
	for _, row := range rows[2] {
		topShips = append(topShips, map[string]any{
			"ship_type_id": conflictInt(row, "ship_type_id"),
			"ship_name":    conflictString(row, "ship_name"),
			"count":        conflictInt(row, "count"),
		})
	}
	totals := map[string]any{}
	if len(rows[1]) > 0 {
		totals = rows[1][0]
	}
	war := map[string]any{
		"war_id":          conflictInt(sides.War, "war_id"),
		"declared":        sides.War["declared"],
		"started":         sides.War["started"],
		"finished":        sides.War["finished"],
		"retracted":       sides.War["retracted"],
		"mutual":          falseIfNil(sides.War["mutual"]),
		"open_for_allies": falseIfNil(sides.War["open_for_allies"]),
		"aggressor":       conflictWarEntityFromRow(sides.War, "aggressor"),
		"defender":        conflictWarEntityFromRow(sides.War, "defender"),
		"allies":          allies,
	}
	return map[string]any{
		"war": war,
		"stats": map[string]any{
			"total_kills": conflictInt(totals, "total_kills"),
			"total_value": conflictFloat(totals, "total_value"),
			"top_ships":   topShips,
		},
	}, nil
}

func warStatsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		warID, err := parseConflictID(req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := loadConflictWarStats(ctx, opts.DB, warID)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(body), nil
	}
}

func loadConflictWarStats(
	ctx context.Context,
	db Database,
	warID int32,
) (map[string]any, error) {
	sides, err := loadConflictWarSides(ctx, db, warID)
	if err != nil {
		return nil, err
	}
	since := time.Date(2003, 1, 1, 0, 0, 0, 0, time.UTC)
	if value, ok := sides.War["started"].(time.Time); ok {
		since = value
	} else if value, ok := sides.War["declared"].(time.Time); ok {
		since = value
	}
	rows, err := queryMapsConcurrent(
		ctx,
		db,
		databaseQuery{
			SQL: `
				WITH ranked AS (
					SELECT interaction.side, interaction.target_type,
					       interaction.target_id AS id,
					       COALESCE(character.name, corporation.name,
					                alliance.name, 'Unknown') AS name,
					       interaction.count,
					       interaction.isk_value,
					       ROW_NUMBER() OVER (
						       PARTITION BY interaction.side, interaction.target_type
						       ORDER BY interaction.count DESC,
						                interaction.target_id
					       ) AS rank
					FROM war_interactions interaction
					LEFT JOIN characters character
					  ON interaction.target_type = 0
					 AND character.character_id = interaction.target_id
					LEFT JOIN corporations corporation
					  ON interaction.target_type = 1
					 AND corporation.corporation_id = interaction.target_id
					LEFT JOIN alliances alliance
					  ON interaction.target_type = 2
					 AND alliance.alliance_id = interaction.target_id
					WHERE interaction.war_id = $1
					  AND interaction.category = 0
					  AND interaction.side IN (0, 1, 2)
					  AND interaction.target_type IN (0, 1, 2)
				)
				SELECT side, target_type, id, name, count, isk_value
				FROM ranked
				WHERE rank <= 10
				ORDER BY side, target_type, count DESC, id`,
			Args: []any{warID},
		},
		databaseQuery{
			SQL: `
				SELECT k.victim_ship_type_id AS ship_type_id,
				       COALESCE(type.name, 'Unknown') AS ship_name,
				       COUNT(*)::bigint AS count
				FROM killmails k
				LEFT JOIN inv_types type ON type.type_id = k.victim_ship_type_id
				WHERE k.war_id = $1 AND k.killmail_time >= $2
				GROUP BY k.victim_ship_type_id, type.name
				ORDER BY count DESC, k.victim_ship_type_id
				LIMIT 10`,
			Args: []any{warID, since},
		},
	)
	if err != nil {
		return nil, err
	}
	type leaderboard struct {
		Characters   []map[string]any
		Corporations []map[string]any
		Alliances    []map[string]any
	}
	boards := map[int64]*leaderboard{
		0: {Characters: []map[string]any{}, Corporations: []map[string]any{}, Alliances: []map[string]any{}},
		1: {Characters: []map[string]any{}, Corporations: []map[string]any{}, Alliances: []map[string]any{}},
		2: {Characters: []map[string]any{}, Corporations: []map[string]any{}, Alliances: []map[string]any{}},
	}
	for _, row := range rows[0] {
		board := boards[conflictInt(row, "side")]
		if board == nil {
			continue
		}
		entry := map[string]any{
			"id":        conflictInt(row, "id"),
			"name":      conflictString(row, "name"),
			"kills":     conflictInt(row, "count"),
			"isk_value": conflictFloat(row, "isk_value"),
		}
		switch conflictInt(row, "target_type") {
		case 0:
			board.Characters = append(board.Characters, entry)
		case 1:
			board.Corporations = append(board.Corporations, entry)
		case 2:
			board.Alliances = append(board.Alliances, entry)
		}
	}
	boardMap := func(board *leaderboard) map[string]any {
		return map[string]any{
			"topCharacters":   board.Characters,
			"topCorporations": board.Corporations,
			"topAlliances":    board.Alliances,
		}
	}
	topShips := make([]map[string]any, 0, len(rows[1]))
	for _, row := range rows[1] {
		topShips = append(topShips, map[string]any{
			"ship_type_id": conflictInt(row, "ship_type_id"),
			"ship_name":    conflictString(row, "ship_name"),
			"count":        conflictInt(row, "count"),
		})
	}
	return map[string]any{
		"combined":  boardMap(boards[0]),
		"aggressor": boardMap(boards[1]),
		"defender":  boardMap(boards[2]),
		"topShips":  topShips,
		"sides": map[string]any{
			"aggressor": map[string]any{
				"corporations": sides.AggressorCorps,
				"alliances":    sides.AggressorAllies,
			},
			"defender": map[string]any{
				"corporations": sides.DefenderCorps,
				"alliances":    sides.DefenderAllies,
			},
		},
	}, nil
}

func warDashboardHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		warID, err := parseConflictID(req)
		if err != nil {
			return legacyPayload{}, err
		}
		var detail, stats map[string]any
		queries, queryCtx := errgroup.WithContext(ctx)
		queries.Go(func() (err error) {
			detail, err = loadConflictWarDetail(queryCtx, opts.DB, warID)
			return err
		})
		queries.Go(func() (err error) {
			stats, err = loadConflictWarStats(queryCtx, opts.DB, warID)
			return err
		})
		if err := queries.Wait(); err != nil {
			return legacyPayload{}, err
		}
		detail["leaderboards"] = stats
		return jsonPayload(detail), nil
	}
}

type conflictWarMemberOptions struct {
	Side          string
	Sort          string
	Limit         int
	CorporationID *int32
	AllianceID    *int32
}

func parseConflictWarMemberOptions(query url.Values) (conflictWarMemberOptions, error) {
	options := conflictWarMemberOptions{
		Side: query.Get("side"),
		Sort: query.Get("sort"),
		Limit: parseConflictBoundedInt(
			query, "limit", conflictMaximumMembers, 1, conflictMaximumMembers,
		),
	}
	switch options.Side {
	case "aggressor", "defender":
	default:
		options.Side = "combined"
	}
	switch options.Sort {
	case "kills", "losses", "isk":
	default:
		options.Sort = "activity"
	}
	var err error
	options.CorporationID, err = parseConflictOptionalID(query, "corporationId")
	if err != nil {
		return conflictWarMemberOptions{}, err
	}
	options.AllianceID, err = parseConflictOptionalID(query, "allianceId")
	return options, err
}

func warMembersHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		warID, err := parseConflictID(req)
		if err != nil {
			return legacyPayload{}, err
		}
		options, err := parseConflictWarMemberOptions(req.Query)
		if err != nil {
			return legacyPayload{}, err
		}
		sides, err := loadConflictWarSides(ctx, opts.DB, warID)
		if err != nil {
			return legacyPayload{}, err
		}
		query, args := conflictWarMembersQuery(warID, sides, options)
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		members := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			characterID := conflictInt(row, "character_id")
			name := conflictString(row, "character_name")
			if name == "" {
				name = fmt.Sprintf("Character %d", characterID)
			}
			side := "defender"
			if conflictInt(row, "side") == 1 {
				side = "aggressor"
			}
			members = append(members, map[string]any{
				"character_id":       characterID,
				"character_name":     name,
				"side":               side,
				"corporation_id":     conflictNullableID(row, "corporation_id"),
				"corporation_name":   conflictNullableString(row, "corp_name"),
				"corporation_ticker": conflictNullableString(row, "corp_ticker"),
				"alliance_id":        conflictNullableID(row, "alliance_id"),
				"alliance_name":      conflictNullableString(row, "alliance_name"),
				"alliance_ticker":    conflictNullableString(row, "alliance_ticker"),
				"kills":              conflictInt(row, "kills"),
				"losses":             conflictInt(row, "losses"),
				"isk_destroyed":      conflictFloat(row, "isk_destroyed"),
				"isk_lost":           conflictFloat(row, "isk_lost"),
				"top_ship_type_id":   conflictNullableID(row, "top_ship_type_id"),
				"top_ship_name":      conflictNullableString(row, "top_ship_name"),
				"top_ship_count":     conflictInt(row, "top_ship_count"),
			})
		}
		return jsonPayload(map[string]any{
			"war_id":  warID,
			"side":    options.Side,
			"count":   len(members),
			"limit":   options.Limit,
			"members": members,
		}), nil
	}
}

func conflictWarMembersQuery(
	warID int32,
	sides conflictWarSides,
	options conflictWarMemberOptions,
) (string, []any) {
	args := []any{
		warID,
		sides.AggressorCorps,
		sides.AggressorAllies,
		sides.DefenderCorps,
		sides.DefenderAllies,
	}
	filters := make([]string, 0, 3)
	switch options.Side {
	case "aggressor":
		filters = append(filters, "latest.side = 1")
	case "defender":
		filters = append(filters, "latest.side = 2")
	}
	if options.CorporationID != nil {
		args = append(args, *options.CorporationID)
		filters = append(filters,
			fmt.Sprintf("latest.corporation_id = $%d", len(args)))
	}
	if options.AllianceID != nil {
		args = append(args, *options.AllianceID)
		filters = append(filters,
			fmt.Sprintf("latest.alliance_id = $%d", len(args)))
	}
	orderBy := "(COALESCE(kills.kills, 0) + COALESCE(losses.losses, 0)) DESC"
	switch options.Sort {
	case "kills":
		orderBy = "COALESCE(kills.kills, 0) DESC"
	case "losses":
		orderBy = "COALESCE(losses.losses, 0) DESC"
	case "isk":
		orderBy = `(COALESCE(kills.isk_destroyed, 0) +
			COALESCE(losses.isk_lost, 0)) DESC`
	}
	where := ""
	if len(filters) > 0 {
		where = "WHERE " + strings.Join(filters, " AND ")
	}
	args = append(args, options.Limit)
	return fmt.Sprintf(`
		WITH war_kms AS MATERIALIZED (
			SELECT killmail_id, killmail_time, total_value,
			       victim_character_id, victim_corporation_id,
			       victim_alliance_id, victim_ship_type_id
			FROM killmails
			WHERE war_id = $1
		),
		participants AS (
			SELECT attacker.character_id,
			       attacker.corporation_id,
			       attacker.alliance_id,
			       attacker.ship_type_id,
			       kill.killmail_id,
			       kill.killmail_time,
			       kill.total_value,
			       'attacker' AS role,
			       CASE
				       WHEN attacker.corporation_id = ANY($2::int[])
				         OR attacker.alliance_id = ANY($3::int[]) THEN 1
				       WHEN attacker.corporation_id = ANY($4::int[])
				         OR attacker.alliance_id = ANY($5::int[]) THEN 2
				       ELSE 0
			       END AS side
			FROM war_kms kill
			JOIN killmail_attackers attacker
			  ON attacker.killmail_id = kill.killmail_id
			WHERE attacker.character_id IS NOT NULL
			UNION ALL
			SELECT victim_character_id, victim_corporation_id,
			       victim_alliance_id, victim_ship_type_id,
			       killmail_id, killmail_time, total_value,
			       'victim' AS role,
			       CASE
				       WHEN victim_corporation_id = ANY($2::int[])
				         OR victim_alliance_id = ANY($3::int[]) THEN 1
				       WHEN victim_corporation_id = ANY($4::int[])
				         OR victim_alliance_id = ANY($5::int[]) THEN 2
				       ELSE 0
			       END AS side
			FROM war_kms
			WHERE victim_character_id IS NOT NULL
		),
		sided AS MATERIALIZED (
			SELECT * FROM participants WHERE side IN (1, 2)
		),
		latest AS (
			SELECT DISTINCT ON (character_id, side)
			       character_id, side, corporation_id, alliance_id
			FROM sided
			ORDER BY character_id, side, killmail_time DESC, killmail_id DESC
		),
		kills AS (
			SELECT character_id, side,
			       COUNT(DISTINCT killmail_id)::int AS kills,
			       COALESCE(SUM(total_value), 0)::double precision
			           AS isk_destroyed
			FROM sided WHERE role = 'attacker'
			GROUP BY character_id, side
		),
		losses AS (
			SELECT character_id, side,
			       COUNT(*)::int AS losses,
			       COALESCE(SUM(total_value), 0)::double precision AS isk_lost
			FROM sided WHERE role = 'victim'
			GROUP BY character_id, side
		),
		top_ship AS (
			SELECT DISTINCT ON (character_id, side)
			       character_id, side, ship_type_id, ship_count
			FROM (
				SELECT character_id, side, ship_type_id,
				       COUNT(*)::int AS ship_count
				FROM sided
				WHERE role = 'attacker' AND ship_type_id IS NOT NULL
				GROUP BY character_id, side, ship_type_id
			) ships
			ORDER BY character_id, side, ship_count DESC, ship_type_id
		)
		SELECT latest.character_id, latest.side,
		       latest.corporation_id, latest.alliance_id,
		       COALESCE(kills.kills, 0)::int AS kills,
		       COALESCE(losses.losses, 0)::int AS losses,
		       COALESCE(kills.isk_destroyed, 0)::double precision
		           AS isk_destroyed,
		       COALESCE(losses.isk_lost, 0)::double precision AS isk_lost,
		       top_ship.ship_type_id AS top_ship_type_id,
		       COALESCE(top_ship.ship_count, 0)::int AS top_ship_count,
		       type.name AS top_ship_name,
		       character.name AS character_name,
		       corporation.name AS corp_name,
		       corporation.ticker AS corp_ticker,
		       alliance.name AS alliance_name,
		       alliance.ticker AS alliance_ticker
		FROM latest
		LEFT JOIN kills
		  ON kills.character_id = latest.character_id
		 AND kills.side = latest.side
		LEFT JOIN losses
		  ON losses.character_id = latest.character_id
		 AND losses.side = latest.side
		LEFT JOIN top_ship
		  ON top_ship.character_id = latest.character_id
		 AND top_ship.side = latest.side
		LEFT JOIN inv_types type ON type.type_id = top_ship.ship_type_id
		LEFT JOIN characters character
		  ON character.character_id = latest.character_id
		LEFT JOIN corporations corporation
		  ON corporation.corporation_id = latest.corporation_id
		LEFT JOIN alliances alliance
		  ON alliance.alliance_id = latest.alliance_id
		%s
		ORDER BY %s, latest.character_id
		LIMIT $%d`,
		where, orderBy, len(args),
	), args
}

func warIntelHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		warID, err := parseConflictID(req)
		if err != nil {
			return legacyPayload{}, err
		}
		if _, err := loadConflictWarSides(ctx, opts.DB, warID); err != nil {
			return legacyPayload{}, err
		}
		rows, err := loadConflictIntelRows(ctx, opts.DB, "k.war_id = $1", []any{warID})
		if err != nil {
			return legacyPayload{}, err
		}
		body := conflictIntelResponse(rows)
		body["war_id"] = warID
		return jsonPayload(body), nil
	}
}

func loadConflictIntelRows(
	ctx context.Context,
	db Database,
	predicate string,
	args []any,
) ([][]map[string]any, error) {
	query := func(sql string) databaseQuery {
		return databaseQuery{SQL: sql, Args: args}
	}
	return queryMapsConcurrent(
		ctx,
		db,
		query(`
			SELECT COUNT(*)::int AS kills,
			       COALESCE(SUM(k.total_value), 0)::double precision
			           AS isk_destroyed,
			       COUNT(DISTINCT k.solar_system_id)::int AS systems,
			       COUNT(DISTINCT k.constellation_id)::int AS constellations,
			       COUNT(DISTINCT k.region_id)::int AS regions
			FROM killmails k WHERE `+predicate),
		query(`
			SELECT k.solar_system_id AS system_id,
			       system.system_name, system.security,
			       system.region_id, region.name AS region_name,
			       COUNT(*)::int AS kills,
			       COALESCE(SUM(k.total_value), 0)::double precision
			           AS isk_destroyed
			FROM killmails k
			LEFT JOIN solar_systems system
			  ON system.solar_system_id = k.solar_system_id
			LEFT JOIN regions region ON region.region_id = system.region_id
			WHERE `+predicate+`
			GROUP BY k.solar_system_id, system.system_name, system.security,
			         system.region_id, region.name
			ORDER BY kills DESC, k.solar_system_id LIMIT 20`),
		query(`
			SELECT k.constellation_id, constellation.constellation_name,
			       constellation.region_id, region.name AS region_name,
			       COUNT(*)::int AS kills,
			       COALESCE(SUM(k.total_value), 0)::double precision
			           AS isk_destroyed
			FROM killmails k
			LEFT JOIN constellations constellation
			  ON constellation.constellation_id = k.constellation_id
			LEFT JOIN regions region
			  ON region.region_id = constellation.region_id
			WHERE `+predicate+` AND k.constellation_id IS NOT NULL
			GROUP BY k.constellation_id, constellation.constellation_name,
			         constellation.region_id, region.name
			ORDER BY kills DESC, k.constellation_id LIMIT 20`),
		query(`
			SELECT k.region_id, region.name AS region_name,
			       COUNT(*)::int AS kills,
			       COALESCE(SUM(k.total_value), 0)::double precision
			           AS isk_destroyed
			FROM killmails k
			LEFT JOIN regions region ON region.region_id = k.region_id
			WHERE `+predicate+` AND k.region_id IS NOT NULL
			GROUP BY k.region_id, region.name
			ORDER BY kills DESC, k.region_id LIMIT 20`),
		query(`
			SELECT k.victim_ship_type_id AS ship_type_id,
			       type.name AS ship_name, type.group_id,
			       group_data.name AS group_name,
			       COUNT(*)::int AS count,
			       COALESCE(SUM(k.total_value), 0)::double precision
			           AS isk_destroyed
			FROM killmails k
			LEFT JOIN inv_types type ON type.type_id = k.victim_ship_type_id
			LEFT JOIN inv_groups group_data
			  ON group_data.group_id = type.group_id
			WHERE `+predicate+` AND k.victim_ship_type_id IS NOT NULL
			GROUP BY k.victim_ship_type_id, type.name,
			         type.group_id, group_data.name
			ORDER BY count DESC, k.victim_ship_type_id LIMIT 20`),
		query(`
			SELECT attacker.ship_type_id, type.name AS ship_name,
			       type.group_id, group_data.name AS group_name,
			       COUNT(*)::int AS count
			FROM killmails k
			JOIN killmail_attackers attacker
			  ON attacker.killmail_id = k.killmail_id
			LEFT JOIN inv_types type ON type.type_id = attacker.ship_type_id
			LEFT JOIN inv_groups group_data
			  ON group_data.group_id = type.group_id
			WHERE `+predicate+` AND attacker.ship_type_id IS NOT NULL
			GROUP BY attacker.ship_type_id, type.name,
			         type.group_id, group_data.name
			ORDER BY count DESC, attacker.ship_type_id LIMIT 20`),
		query(`
			SELECT type.group_id, group_data.name AS group_name,
			       COUNT(*)::int AS count,
			       COALESCE(SUM(k.total_value), 0)::double precision
			           AS isk_destroyed
			FROM killmails k
			LEFT JOIN inv_types type ON type.type_id = k.victim_ship_type_id
			LEFT JOIN inv_groups group_data
			  ON group_data.group_id = type.group_id
			WHERE `+predicate+` AND type.group_id IS NOT NULL
			GROUP BY type.group_id, group_data.name
			ORDER BY count DESC, type.group_id LIMIT 20`),
		query(`
			SELECT CASE
				       WHEN system.security >= 0.5 THEN 'highsec'
				       WHEN system.security > 0 AND system.security < 0.5
				           THEN 'lowsec'
				       WHEN system.security <= 0 AND system.security > -1
				           THEN 'nullsec'
				       WHEN system.security <= -1 THEN 'wormhole'
				       ELSE 'unknown'
			       END AS sec_class,
			       COUNT(*)::int AS kills,
			       COALESCE(SUM(k.total_value), 0)::double precision
			           AS isk_destroyed
			FROM killmails k
			LEFT JOIN solar_systems system
			  ON system.solar_system_id = k.solar_system_id
			WHERE `+predicate+`
			GROUP BY sec_class ORDER BY kills DESC, sec_class`),
	)
}

func conflictIntelResponse(rows [][]map[string]any) map[string]any {
	summary := map[string]any{}
	if len(rows) > 0 && len(rows[0]) > 0 {
		summary = rows[0][0]
	}
	return map[string]any{
		"summary": map[string]any{
			"kills":          conflictInt(summary, "kills"),
			"isk_destroyed":  conflictFloat(summary, "isk_destroyed"),
			"systems":        conflictInt(summary, "systems"),
			"constellations": conflictInt(summary, "constellations"),
			"regions":        conflictInt(summary, "regions"),
		},
		"top_systems":           factionWarIntelSystems(rows[1]),
		"top_constellations":    factionWarIntelConstellations(rows[2]),
		"top_regions":           factionWarIntelRegions(rows[3]),
		"ships_destroyed":       factionWarIntelShips(rows[4], true),
		"ships_used":            factionWarIntelShips(rows[5], false),
		"ship_groups_destroyed": factionWarIntelShipGroups(rows[6]),
		"security_breakdown":    factionWarIntelSecurity(rows[7]),
	}
}

const conflictKilllistSelect = `
	SELECT k.killmail_id, k.killmail_hash, k.killmail_time,
	       COALESCE(k.total_value, 0)::double precision AS total_value,
	       COALESCE(k.attacker_count, 0)::int AS attacker_count,
	       COALESCE(k.is_npc, false) AS is_npc,
	       COALESCE(k.is_solo, false) AS is_solo,
	       k.victim_ship_type_id AS ship_type_id,
	       ship.name AS ship_name,
	       ship.meta_group_id,
	       ship.market_group_id AS ship_market_group_id,
	       ship_group.name AS ship_group_name,
	       k.victim_character_id, victim_character.name AS victim_character_name,
	       k.victim_corporation_id,
	       victim_corporation.name AS victim_corporation_name,
	       k.victim_alliance_id, victim_alliance.name AS victim_alliance_name,
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
		WHERE killmail_id = k.killmail_id AND final_blow IS TRUE
		ORDER BY attacker_index
		LIMIT 1
	) final_blow ON true
	LEFT JOIN inv_types ship ON ship.type_id = k.victim_ship_type_id
	LEFT JOIN inv_groups ship_group
	  ON ship_group.group_id = k.victim_ship_group_id
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
	LEFT JOIN inv_types final_ship ON final_ship.type_id = final_blow.ship_type_id
	LEFT JOIN solar_systems system ON system.solar_system_id = k.solar_system_id
	LEFT JOIN regions region ON region.region_id = k.region_id`

func warKilllistHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		warID, err := parseConflictID(req)
		if err != nil {
			return legacyPayload{}, err
		}
		where := []string{"k.war_id = $1"}
		args := []any{warID}
		if raw := strings.TrimSpace(req.Query.Get("warStart")); raw != "" {
			start, err := parseJavaScriptDate(raw)
			if err != nil {
				return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid warStart")
			}
			args = append(args, start)
			where = append(where,
				fmt.Sprintf("k.killmail_time >= $%d", len(args)))
		}
		if raw := strings.TrimSpace(req.Query.Get("warEnd")); raw != "" {
			end, err := parseJavaScriptDate(raw)
			if err != nil {
				return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid warEnd")
			}
			args = append(args, end.Add(24*time.Hour))
			where = append(where,
				fmt.Sprintf("k.killmail_time <= $%d", len(args)))
		}
		corporations := parseCommaInt32(req.Query.Get("warSideCorps"))
		alliances := parseCommaInt32(req.Query.Get("warSideAlliances"))
		victims := make([]string, 0, 2)
		if len(corporations) > 0 {
			args = append(args, corporations)
			victims = append(victims,
				fmt.Sprintf("k.victim_corporation_id = ANY($%d::int[])", len(args)))
		}
		if len(alliances) > 0 {
			args = append(args, alliances)
			victims = append(victims,
				fmt.Sprintf("k.victim_alliance_id = ANY($%d::int[])", len(args)))
		}
		if len(victims) > 0 {
			where = append(where, "("+strings.Join(victims, " OR ")+")")
		}
		return loadConflictKilllist(ctx, opts.DB, req.Query, where, args)
	}
}

func loadConflictKilllist(
	ctx context.Context,
	db Database,
	queryValues url.Values,
	where []string,
	args []any,
) (legacyPayload, error) {
	limit := parseConflictBoundedInt(
		queryValues, "limit", conflictDefaultLimit, 10, conflictMaximumLimit,
	)
	page := parseConflictBoundedInt(
		queryValues, "page", 0, 0, conflictMaximumPage,
	)
	var after *int64
	if raw := strings.TrimSpace(queryValues.Get("after")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid after")
		}
		after = &value
	}

	baseArgs := append([]any(nil), args...)
	total, err := conflictKilllistTotal(ctx, db, where, baseArgs)
	if err != nil {
		return legacyPayload{}, err
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	if page >= 1 && after == nil {
		args = append(args, limit, (page-1)*limit)
		rows, err := queryMaps(ctx, db,
			conflictKilllistSelect+
				" WHERE "+strings.Join(where, " AND ")+
				" ORDER BY k.killmail_time DESC, k.killmail_id DESC"+
				fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)),
			args...,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if err := enrichConflictKilllist(ctx, db, rows); err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"kills":      rows,
			"hasMore":    page < totalPages,
			"cursor":     nil,
			"totalPages": totalPages,
		}), nil
	}

	if after != nil {
		args = append(args, *after)
		where = append(where, fmt.Sprintf("k.killmail_id < $%d", len(args)))
	}
	args = append(args, limit+1)
	rows, err := queryMaps(ctx, db,
		conflictKilllistSelect+
			" WHERE "+strings.Join(where, " AND ")+
			" ORDER BY k.killmail_time DESC, k.killmail_id DESC"+
			fmt.Sprintf(" LIMIT $%d", len(args)),
		args...,
	)
	if err != nil {
		return legacyPayload{}, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	if err := enrichConflictKilllist(ctx, db, rows); err != nil {
		return legacyPayload{}, err
	}
	var cursor any
	if len(rows) > 0 {
		cursor = rows[len(rows)-1]["killmail_id"]
	}
	response := map[string]any{
		"kills": rows, "hasMore": hasMore, "cursor": cursor,
	}
	if total > 0 {
		response["totalPages"] = totalPages
	}
	return jsonPayload(response), nil
}

func conflictKilllistTotal(
	ctx context.Context,
	db Database,
	where []string,
	args []any,
) (int64, error) {
	row, err := queryMap(ctx, db,
		`SELECT COUNT(*)::bigint AS total
		 FROM killmails k WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return 0, err
	}
	return conflictInt(row, "total"), nil
}

func enrichConflictKilllist(
	ctx context.Context,
	db Database,
	rows []map[string]any,
) error {
	if len(rows) == 0 {
		return nil
	}
	groupIDs := make([]int32, 0, len(rows))
	for _, row := range rows {
		if id := int32(conflictInt(row, "ship_market_group_id")); id > 0 {
			groupIDs = append(groupIDs, id)
		}
	}
	groupIDs = uniqueConflictIDs(groupIDs)
	if len(groupIDs) == 0 {
		for _, row := range rows {
			row["ship_market_path"] = nil
			delete(row, "ship_market_group_id")
		}
		return nil
	}
	// Walk only the handful of market groups present in this response. Loading
	// the full SDE market tree for every cache miss made small killlists pay
	// thousands of allocations for data they never used.
	marketGroups, err := queryMaps(ctx, db, `
		WITH RECURSIVE lineage AS (
			SELECT market_group_id AS origin_id, market_group_id,
			       parent_group_id, name, 0 AS depth,
			       ARRAY[market_group_id]::int[] AS visited
			FROM inv_market_groups
			WHERE market_group_id = ANY($1::int[])
			UNION ALL
			SELECT lineage.origin_id, parent.market_group_id,
			       parent.parent_group_id, parent.name,
			       lineage.depth + 1,
			       lineage.visited || parent.market_group_id
			FROM lineage
			JOIN inv_market_groups parent
			  ON parent.market_group_id = lineage.parent_group_id
			WHERE lineage.depth < 15
			  AND NOT parent.market_group_id = ANY(lineage.visited)
		)
		SELECT origin_id,
		       ARRAY_AGG(name ORDER BY depth DESC)::text[] AS names
		FROM lineage
		GROUP BY origin_id`,
		groupIDs,
	)
	if err != nil {
		return err
	}
	paths := make(map[int64]string, len(marketGroups))
	for _, row := range marketGroups {
		id, ok := int64Value(row["origin_id"])
		if !ok {
			continue
		}
		names, ok := row["names"].([]string)
		if !ok {
			if values, valuesOK := row["names"].([]any); valuesOK {
				names = make([]string, 0, len(values))
				for _, value := range values {
					if name, nameOK := stringValue(value); nameOK {
						names = append(names, name)
					}
				}
			}
		}
		segments := make([]string, 0, len(names))
		for _, name := range names {
			segments = append(segments, eve.Slugify(name))
		}
		if len(segments) == 0 {
			continue
		}
		paths[id] = "/market/" + strings.Join(segments, "/")
	}
	for _, row := range rows {
		groupID, _ := int64Value(row["ship_market_group_id"])
		if path := paths[groupID]; path != "" {
			row["ship_market_path"] = path
		} else {
			row["ship_market_path"] = nil
		}
		delete(row, "ship_market_group_id")
	}
	return nil
}
