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
	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/eve-kill/shrike/internal/stats"
)

const (
	domainKilllistCacheTTL     = time.Minute
	domainTopKillsCacheTTL     = 5 * time.Minute
	domainStatisticsCacheTTL   = time.Hour
	domainCandidateMinimumRows = 250
)

// domainEntityScope deliberately aliases the scope already used by
// domain-aware conflict reads. That loader owns the active-domain and
// Host/subdomain/custom-hostname lookup rules; these handlers only decide how
// the configured entities constrain their data.
type domainEntityScope = conflictEntityScope

type domainKillboardService struct {
	db  Database
	now func() time.Time
}

type domainLocation struct {
	name   string
	column string
}

var domainLocations = []domainLocation{
	{name: "region", column: "region_id"},
	{name: "constellation", column: "constellation_id"},
	{name: "system", column: "solar_system_id"},
}

func registerDomainKillboardRoutes(a huma.API, opts Options) {
	service := &domainKillboardService{db: opts.DB, now: time.Now}

	registerDomainRead(a, opts, huma.Operation{
		OperationID: "domain-killlist",
		Method:      http.MethodGet,
		Path:        "/custom/killlist",
		Summary:     "Custom-domain killmail list",
		Tags:        []string{"domains", "killboard", "killmails"},
	}, domainKilllistCacheTTL, service.killlistHandler(domainLocation{}))

	for _, location := range domainLocations {
		location := location
		registerDomainRead(a, opts, huma.Operation{
			OperationID: "domain-" + location.name + "-killlist",
			Method:      http.MethodGet,
			Path:        "/custom/" + location.name + "/{id}/killlist",
			Summary: "Custom-domain killmails in this " +
				location.name,
			Tags: []string{"domains", "universe", "killmails"},
		}, domainKilllistCacheTTL, service.killlistHandler(location))
	}

	registerDomainRead(a, opts, huma.Operation{
		OperationID: "domain-kills-most-valuable",
		Method:      http.MethodGet,
		Path:        "/custom/kills/most-valuable",
		Summary:     "Custom-domain most valuable kills",
		Tags:        []string{"domains", "killboard", "statistics"},
	}, domainTopKillsCacheTTL, service.mostValuableHandler())

	registerDomainRead(a, opts, huma.Operation{
		OperationID: "domain-kills-top",
		Method:      http.MethodGet,
		Path:        "/custom/kills/top",
		Summary:     "Custom-domain top killers and locations",
		Tags:        []string{"domains", "killboard", "statistics"},
	}, domainTopKillsCacheTTL, service.topKillsHandler())

	registerDomainRead(a, opts, huma.Operation{
		OperationID: "domain-statistics",
		Method:      http.MethodGet,
		Path:        "/custom/stats",
		Summary:     "Custom-domain statistics",
		Tags:        []string{"domains", "statistics"},
	}, domainStatisticsCacheTTL, service.statisticsHandler())
}

func registerDomainRead(
	a huma.API,
	opts Options,
	operation huma.Operation,
	ttl time.Duration,
	handler legacyHandler,
) {
	cacheControl := fmt.Sprintf(
		"public, max-age=%d, s-maxage=%d, stale-while-revalidate=%d",
		max(30, int(ttl.Seconds()/5)),
		int(ttl.Seconds()),
		int(ttl.Seconds()),
	)
	cached := routeJSONCacheBy(
		opts,
		ttl,
		cacheControl,
		domainReadCacheKey,
		handler,
	)
	registerLegacy(a, operation, func(
		ctx context.Context,
		req *legacyRequest,
	) (legacyPayload, error) {
		payload, err := cached(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		if payload.Headers == nil {
			payload.Headers = make(http.Header)
		}
		payload.Headers.Add("Vary", "Host")
		return payload, nil
	})
}

func domainReadCacheKey(req *legacyRequest) string {
	url := req.Huma.URL()
	return domainReadCacheKeyFor(
		conflictRequestHost(req),
		url.RequestURI(),
	)
}

func domainReadCacheKeyFor(host, requestURI string) string {
	return "domain:" + host + ":" + requestURI
}

func (s *domainKillboardService) requireScope(
	ctx context.Context,
	req *legacyRequest,
) (*domainEntityScope, error) {
	scope, err := loadConflictDomainScope(ctx, s.db, req)
	if err != nil {
		return nil, err
	}
	if scope == nil {
		return nil, apiError(
			http.StatusBadRequest,
			"This endpoint requires a custom domain context",
		)
	}
	return scope, nil
}

func (s *domainKillboardService) killlistHandler(
	location domainLocation,
) legacyHandler {
	return func(
		ctx context.Context,
		req *legacyRequest,
	) (legacyPayload, error) {
		scope, err := s.requireScope(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		var locationID *int64
		if location.column != "" {
			id, idErr := parseUniverseID(req.Param("id"))
			if idErr != nil {
				return legacyPayload{}, idErr
			}
			locationID = &id
		}
		limit := boundedQueryInt(req, "limit", 50, 10, 100)
		after, err := optionalPositiveInt64(req.Query.Get("after"))
		if err != nil {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid after",
			)
		}
		kind := ""
		if location.column == "" {
			kind = strings.TrimSpace(req.Query.Get("type"))
			if kind == "" {
				kind = "latest"
			}
		}

		query, args := buildDomainKilllistQuery(
			scope,
			kind,
			location.column,
			locationID,
			after,
			limit,
		)
		rows, err := queryMaps(ctx, s.db, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, hasMore, cursor, err := finishUniverseKilllist(
			ctx, s.db, rows, limit,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		// The TypeScript custom killlist SELECT intentionally omitted the hash,
		// unlike the ordinary global list. Preserve that established payload.
		for _, row := range rows {
			delete(row, "killmail_hash")
		}
		return jsonPayload(map[string]any{
			"kills": rows, "hasMore": hasMore, "cursor": cursor,
		}), nil
	}
}

type domainSQLArgs struct {
	values []any
}

func (a *domainSQLArgs) bind(value any) string {
	a.values = append(a.values, value)
	return "$" + strconv.Itoa(len(a.values))
}

func buildDomainKilllistQuery(
	scope *domainEntityScope,
	kind string,
	locationColumn string,
	locationID *int64,
	after *int64,
	limit int,
) (string, []any) {
	args := &domainSQLArgs{}
	afterArg := ""
	if after != nil {
		afterArg = args.bind(*after)
	}
	locationArg := ""
	if locationColumn != "" && locationID != nil {
		locationArg = args.bind(*locationID)
	}
	legLimit := max(limit*5, domainCandidateMinimumRows)
	legs := make([]string, 0, 6)

	addLegs := func(
		ids []int32,
		victimColumn string,
		attackerColumn string,
	) {
		if len(ids) == 0 {
			return
		}
		idsArg := args.bind(ids)
		victimWhere := []string{
			victimColumn + " = ANY(" + idsArg + "::int[])",
		}
		if afterArg != "" {
			victimWhere = append(
				victimWhere, "killmail_id < "+afterArg,
			)
		}
		if locationArg != "" {
			victimWhere = append(
				victimWhere,
				locationColumn+" = "+locationArg,
			)
		}
		legs = append(legs, fmt.Sprintf(`
			(
				SELECT killmail_id
				FROM killmails
				WHERE %s
				ORDER BY killmail_id DESC
				LIMIT %d
			)`, strings.Join(victimWhere, " AND "), legLimit))

		attackerWhere := []string{
			attackerColumn + " = ANY(" + idsArg + "::int[])",
		}
		if afterArg != "" {
			attackerWhere = append(attackerWhere, `
				killmail_time <= (
					SELECT killmail_time FROM killmails
					WHERE killmail_id = `+afterArg+`
				)`)
		}
		legs = append(legs, fmt.Sprintf(`
			(
				SELECT killmail_id
				FROM killmail_attackers
				WHERE %s
				ORDER BY killmail_time DESC
				LIMIT %d
			)`, strings.Join(attackerWhere, " AND "), legLimit))
	}

	if scope != nil {
		addLegs(scope.Characters, "victim_character_id", "character_id")
		addLegs(
			scope.Corporations,
			"victim_corporation_id",
			"corporation_id",
		)
		addLegs(scope.Alliances, "victim_alliance_id", "alliance_id")
	}
	candidates := "SELECT NULL::bigint AS killmail_id WHERE false"
	if len(legs) != 0 {
		candidates = strings.Join(legs, " UNION ")
	}

	where := []string{
		"k.killmail_id IN (SELECT killmail_id FROM candidates)",
	}
	if predicate, known := killtype.Predicates()[kind]; known &&
		predicate != "TRUE" {
		where = append(where, predicate)
	}
	if locationArg != "" {
		where = append(
			where, "k."+locationColumn+" = "+locationArg,
		)
	}
	if afterArg != "" {
		where = append(where, "k.killmail_id < "+afterArg)
	}
	limitArg := args.bind(limit + 1)
	query := `
		WITH candidates AS MATERIALIZED (` + candidates + `)
		` + campaignKilllistSelect + `
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY k.killmail_id DESC
		LIMIT ` + limitArg
	return query, args.values
}

func (s *domainKillboardService) mostValuableHandler() legacyHandler {
	return func(
		ctx context.Context,
		req *legacyRequest,
	) (legacyPayload, error) {
		scope, err := s.requireScope(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		kind := strings.TrimSpace(req.Query.Get("type"))
		if kind == "" {
			kind = "latest"
		}
		predicate := "TRUE"
		if value, known := killtype.Predicates()[kind]; known {
			predicate = value
		}
		limit := boundedQueryInt(req, "limit", 7, 1, 20)
		days := boundedQueryInt(
			req, "days", 7, 1, math.MaxInt32,
		)
		rows, err := loadDomainMostValuable(
			ctx,
			s.db,
			scope,
			s.now().UTC().AddDate(0, 0, -days),
			limit,
			predicate,
			0,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"entries": nonNilUniverseRows(rows),
		}), nil
	}
}

func (s *domainKillboardService) topKillsHandler() legacyHandler {
	return func(
		ctx context.Context,
		req *legacyRequest,
	) (legacyPayload, error) {
		scope, err := s.requireScope(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		kind := strings.TrimSpace(req.Query.Get("type"))
		if kind == "" {
			kind = "latest"
		}
		dataType := strings.TrimSpace(req.Query.Get("dataType"))
		if dataType == "" {
			dataType = "characters"
		}
		limit := boundedQueryInt(req, "limit", 10, 1, 50)
		days := boundedQueryInt(
			req, "days", 7, 1, math.MaxInt32,
		)
		since := s.now().UTC().
			AddDate(0, 0, -days).
			Format("2006-01-02")
		rows, err := loadDomainTopKills(
			ctx, s.db, scope, kind, dataType, since, limit,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"entries": nonNilUniverseRows(rows),
		}), nil
	}
}

var validDomainStatisticsTypes = map[string]struct{}{
	"characters": {}, "corporations": {}, "alliances": {},
	"ships": {}, "systems": {}, "regions": {},
	"isk_destroyers_chars": {}, "isk_destroyers_corps": {},
	"isk_destroyers_alliances": {}, "solo_killers": {},
	"top_points": {}, "dangerous_systems": {},
	"deadliest_regions": {}, "most_used_ships": {},
	"most_destroyed_ships": {}, "biggest_losers": {},
}

func (s *domainKillboardService) statisticsHandler() legacyHandler {
	return func(
		ctx context.Context,
		req *legacyRequest,
	) (legacyPayload, error) {
		scope, err := s.requireScope(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		dataType := strings.TrimSpace(req.Query.Get("dataType"))
		if dataType == "" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Missing dataType parameter",
			)
		}
		limit := boundedQueryInt(req, "limit", 10, 1, 100)
		days := boundedQueryInt(req, "days", 7, 1, 90)
		now := s.now().UTC()

		var entries []map[string]any
		switch {
		case hasDomainStatisticsType(dataType):
			entries, err = loadDomainStatistics(
				ctx,
				s.db,
				scope,
				dataType,
				now.AddDate(0, 0, -days).Format("2006-01-02"),
				limit,
			)
			if err == nil {
				entries, err = attachGlobalStatsPalettes(
					ctx, s.db, entries,
				)
			}
		case strings.HasPrefix(dataType, "most_valuable_"):
			category := 0
			switch dataType {
			case "most_valuable_ships":
				category = 6
			case "most_valuable_structures":
				category = 65
			}
			entries, err = loadDomainMostValuable(
				ctx,
				s.db,
				scope,
				now.AddDate(0, 0, -days),
				limit,
				"TRUE",
				category,
			)
		default:
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"Unknown dataType: "+dataType,
			)
		}
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"entries": nonNilUniverseRows(entries),
		}), nil
	}
}

func hasDomainStatisticsType(dataType string) bool {
	_, ok := validDomainStatisticsTypes[dataType]
	return ok
}

type domainHeadlineStatisticsSpec struct {
	entityType stats.EntityType
	table      string
	idColumn   string
	entityKind string
	metric     string
}

var domainHeadlineStatistics = map[string]domainHeadlineStatisticsSpec{
	"characters": {
		stats.EntityCharacter, "characters", "character_id",
		"character", "kills",
	},
	"corporations": {
		stats.EntityCorporation, "corporations", "corporation_id",
		"corporation", "kills",
	},
	"alliances": {
		stats.EntityAlliance, "alliances", "alliance_id",
		"alliance", "kills",
	},
	"isk_destroyers_chars": {
		stats.EntityCharacter, "characters", "character_id",
		"character", "isk_destroyed",
	},
	"isk_destroyers_corps": {
		stats.EntityCorporation, "corporations", "corporation_id",
		"corporation", "isk_destroyed",
	},
	"isk_destroyers_alliances": {
		stats.EntityAlliance, "alliances", "alliance_id",
		"alliance", "isk_destroyed",
	},
	"solo_killers": {
		stats.EntityCharacter, "characters", "character_id",
		"character", "solo_kills",
	},
	"top_points": {
		stats.EntityCharacter, "characters", "character_id",
		"character", "points",
	},
	"biggest_losers": {
		stats.EntityCharacter, "characters", "character_id",
		"character", "isk_lost",
	},
}

type domainBreakdownStatisticsSpec struct {
	category   stats.DimCategory
	metric     string
	table      string
	idColumn   string
	nameColumn string
	entityKind string
}

var domainBreakdownStatistics = map[string]domainBreakdownStatisticsSpec{
	"ships": {
		stats.DimShipFlown, "kills", "inv_types",
		"type_id", "name", "ship",
	},
	"most_used_ships": {
		stats.DimShipFlown, "kills", "inv_types",
		"type_id", "name", "ship",
	},
	"most_destroyed_ships": {
		stats.DimShipLost, "losses", "inv_types",
		"type_id", "name", "ship",
	},
	"systems": {
		stats.DimSystem, "kills", "solar_systems",
		"solar_system_id", "system_name", "system",
	},
	"dangerous_systems": {
		stats.DimSystem, "kills", "solar_systems",
		"solar_system_id", "system_name", "system",
	},
	"regions": {
		stats.DimRegion, "kills", "regions",
		"region_id", "name", "region",
	},
	"deadliest_regions": {
		stats.DimRegion, "kills", "regions",
		"region_id", "name", "region",
	},
}

func loadDomainTopKills(
	ctx context.Context,
	db Database,
	scope *domainEntityScope,
	kind string,
	dataType string,
	since string,
	limit int,
) ([]map[string]any, error) {
	if spec, ok := domainHeadlineStatistics[dataType]; ok &&
		(dataType == "characters" ||
			dataType == "corporations" ||
			dataType == "alliances") {
		spec.metric = "kills"
		if kind == "solo" {
			spec.metric = "solo_kills"
		}
		return queryDomainHeadlineStatistics(
			ctx, db, scope, spec, since, limit,
		)
	}
	spec, ok := domainBreakdownStatistics[dataType]
	if !ok || dataType != "ships" &&
		dataType != "systems" &&
		dataType != "regions" {
		return []map[string]any{}, nil
	}
	// The established /custom/kills/top contract applies the solo switch only
	// to headline entity rows. Ship and location breakdowns remain all kills.
	spec.metric = "kills"
	return queryDomainBreakdownStatistics(
		ctx, db, scope, spec, since, limit,
	)
}

func loadDomainStatistics(
	ctx context.Context,
	db Database,
	scope *domainEntityScope,
	dataType string,
	since string,
	limit int,
) ([]map[string]any, error) {
	if spec, ok := domainHeadlineStatistics[dataType]; ok {
		return queryDomainHeadlineStatistics(
			ctx, db, scope, spec, since, limit,
		)
	}
	if spec, ok := domainBreakdownStatistics[dataType]; ok {
		return queryDomainBreakdownStatistics(
			ctx, db, scope, spec, since, limit,
		)
	}
	return []map[string]any{}, nil
}

func queryDomainHeadlineStatistics(
	ctx context.Context,
	db Database,
	scope *domainEntityScope,
	spec domainHeadlineStatisticsSpec,
	since string,
	limit int,
) ([]map[string]any, error) {
	query, args := buildDomainHeadlineStatisticsQuery(
		scope, spec, since, limit,
	)
	rows, err := queryMaps(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	normalizeDomainStatisticsRows(rows, spec.entityKind)
	return rows, nil
}

func buildDomainHeadlineStatisticsQuery(
	scope *domainEntityScope,
	spec domainHeadlineStatisticsSpec,
	since string,
	limit int,
) (string, []any) {
	args := &domainSQLArgs{}
	entityTypeArg := args.bind(spec.entityType)
	periodArg := args.bind(stats.PeriodDaily)
	sinceArg := args.bind(since)
	scopeSQL := domainHeadlineScopeSQL(
		scope, spec.entityKind, "entity", args,
	)
	limitArg := args.bind(limit)
	query := fmt.Sprintf(`
		SELECT s.entity_id AS id, entity.name,
		       COALESCE(SUM(s.%s), 0)::double precision AS count
		FROM stats s
		JOIN %s entity ON entity.%s = s.entity_id
		WHERE s.entity_type = %s
		  AND s.period_type = %s
		  AND s.period_start >= %s::date
		  AND (%s)
		GROUP BY s.entity_id, entity.name
		ORDER BY count DESC
		LIMIT %s`,
		spec.metric,
		spec.table,
		spec.idColumn,
		entityTypeArg,
		periodArg,
		sinceArg,
		scopeSQL,
		limitArg,
	)
	return query, args.values
}

func domainHeadlineScopeSQL(
	scope *domainEntityScope,
	entityKind string,
	alias string,
	args *domainSQLArgs,
) string {
	if scope == nil {
		return "false"
	}
	parts := []string{}
	switch entityKind {
	case "character":
		if len(scope.Characters) != 0 {
			parts = append(parts,
				alias+".character_id = ANY("+
					args.bind(scope.Characters)+"::int[])",
			)
		}
		if len(scope.Corporations) != 0 {
			parts = append(parts,
				alias+".corporation_id = ANY("+
					args.bind(scope.Corporations)+"::int[])",
			)
		}
		if len(scope.Alliances) != 0 {
			parts = append(parts, `
				`+alias+`.corporation_id IN (
					SELECT corporation_id FROM corporations
					WHERE alliance_id = ANY(`+
				args.bind(scope.Alliances)+`::int[])
				)`)
		}
	case "corporation":
		if len(scope.Corporations) != 0 {
			parts = append(parts,
				alias+".corporation_id = ANY("+
					args.bind(scope.Corporations)+"::int[])",
			)
		}
		if len(scope.Alliances) != 0 {
			parts = append(parts,
				alias+".alliance_id = ANY("+
					args.bind(scope.Alliances)+"::int[])",
			)
		}
	case "alliance":
		if len(scope.Alliances) != 0 {
			parts = append(parts,
				alias+".alliance_id = ANY("+
					args.bind(scope.Alliances)+"::int[])",
			)
		}
	}
	if len(parts) == 0 {
		return "false"
	}
	return strings.Join(parts, " OR ")
}

func queryDomainBreakdownStatistics(
	ctx context.Context,
	db Database,
	scope *domainEntityScope,
	spec domainBreakdownStatisticsSpec,
	since string,
	limit int,
) ([]map[string]any, error) {
	query, args := buildDomainBreakdownStatisticsQuery(
		scope, spec, since, limit,
	)
	rows, err := queryMaps(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	normalizeDomainStatisticsRows(rows, spec.entityKind)
	return rows, nil
}

func buildDomainBreakdownStatisticsQuery(
	scope *domainEntityScope,
	spec domainBreakdownStatisticsSpec,
	since string,
	limit int,
) (string, []any) {
	args := &domainSQLArgs{}
	scopeSQL := domainExactEntityScopeSQL(scope, "b", args)
	categoryArg := args.bind(spec.category)
	periodArg := args.bind(stats.PeriodDaily)
	sinceArg := args.bind(since)
	limitArg := args.bind(limit)
	query := fmt.Sprintf(`
		SELECT b.dim_id AS id, entity.%s AS name,
		       COALESCE(SUM(b.%s), 0)::double precision AS count
		FROM stats_breakdowns b
		JOIN %s entity ON entity.%s = b.dim_id
		WHERE (%s)
		  AND b.dim_category = %s
		  AND b.period_type = %s
		  AND b.period_start >= %s::date
		GROUP BY b.dim_id, entity.%s
		ORDER BY count DESC
		LIMIT %s`,
		spec.nameColumn,
		spec.metric,
		spec.table,
		spec.idColumn,
		scopeSQL,
		categoryArg,
		periodArg,
		sinceArg,
		spec.nameColumn,
		limitArg,
	)
	return query, args.values
}

func domainExactEntityScopeSQL(
	scope *domainEntityScope,
	alias string,
	args *domainSQLArgs,
) string {
	if scope == nil {
		return "false"
	}
	parts := []string{}
	for _, item := range []struct {
		entityType stats.EntityType
		ids        []int32
	}{
		{stats.EntityCharacter, scope.Characters},
		{stats.EntityCorporation, scope.Corporations},
		{stats.EntityAlliance, scope.Alliances},
	} {
		if len(item.ids) == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf(
			"(%s.entity_type = %s AND %s.entity_id = ANY(%s::int[]))",
			alias,
			args.bind(item.entityType),
			alias,
			args.bind(item.ids),
		))
	}
	if len(parts) == 0 {
		return "false"
	}
	return strings.Join(parts, " OR ")
}

func normalizeDomainStatisticsRows(
	rows []map[string]any,
	entityKind string,
) {
	for _, row := range rows {
		if stringOrEmpty(row["name"]) == "" {
			row["name"] = "Unknown"
		}
		row["count"] = float64OrZero(row["count"])
		row["type"] = entityKind
	}
}

func loadDomainMostValuable(
	ctx context.Context,
	db Database,
	scope *domainEntityScope,
	since time.Time,
	limit int,
	predicate string,
	category int,
) ([]map[string]any, error) {
	if scope == nil || scope.empty() {
		return []map[string]any{}, nil
	}
	query, args := buildDomainMostValuableQuery(
		scope, since, limit, predicate, category,
	)
	rows, err := queryMaps(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if stringOrEmpty(row["ship_name"]) == "" {
			row["ship_name"] = "Unknown"
		}
		row["total_value"] = float64OrZero(row["total_value"])
	}
	return rows, nil
}

func buildDomainMostValuableQuery(
	scope *domainEntityScope,
	since time.Time,
	limit int,
	predicate string,
	category int,
) (string, []any) {
	args := &domainSQLArgs{}
	sinceArg := args.bind(since)
	attackerScope := domainAttackerScopeSQL(scope, "attacker", args)
	categorySQL := ""
	if category != 0 {
		categorySQL = `
		  AND k.victim_ship_group_id IN (
		    SELECT group_id FROM inv_groups
		    WHERE category_id = ` + args.bind(category) + `
		  )`
	}
	if predicate == "" {
		predicate = "TRUE"
	}
	limitArg := args.bind(limit)
	query := `
		WITH attacker_kills AS MATERIALIZED (
			SELECT DISTINCT attacker.killmail_id
			FROM killmail_attackers attacker
			WHERE attacker.killmail_time >= ` + sinceArg + `::timestamptz
			  AND (` + attackerScope + `)
		)
		SELECT k.killmail_id, k.killmail_hash,
		       k.victim_ship_type_id AS ship_type_id,
		       COALESCE(ship.name, 'Unknown') AS ship_name,
		       COALESCE(k.total_value, 0)::double precision AS total_value,
		       k.victim_character_id,
		       character.name AS victim_character_name,
		       corporation.name AS victim_corporation_name,
		       alliance.name AS victim_alliance_name
		FROM killmails k
		JOIN attacker_kills scoped
		  ON scoped.killmail_id = k.killmail_id
		JOIN inv_types ship
		  ON ship.type_id = k.victim_ship_type_id
		LEFT JOIN characters character
		  ON character.character_id = k.victim_character_id
		LEFT JOIN corporations corporation
		  ON corporation.corporation_id = k.victim_corporation_id
		LEFT JOIN alliances alliance
		  ON alliance.alliance_id = k.victim_alliance_id
		WHERE k.killmail_time >= ` + sinceArg + `::timestamptz
		  AND (` + predicate + `)` + categorySQL + `
		ORDER BY k.total_value DESC
		LIMIT ` + limitArg
	return query, args.values
}

func domainAttackerScopeSQL(
	scope *domainEntityScope,
	alias string,
	args *domainSQLArgs,
) string {
	if scope == nil {
		return "false"
	}
	parts := []string{}
	for _, item := range []struct {
		column string
		ids    []int32
	}{
		{"character_id", scope.Characters},
		{"corporation_id", scope.Corporations},
		{"alliance_id", scope.Alliances},
	} {
		if len(item.ids) == 0 {
			continue
		}
		parts = append(parts,
			alias+"."+item.column+" = ANY("+
				args.bind(item.ids)+"::int[])",
		)
	}
	if len(parts) == 0 {
		return "false"
	}
	return strings.Join(parts, " OR ")
}
