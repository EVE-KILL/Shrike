package api

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/jackc/pgx/v5"
)

const trackedDashboardConfigKey = "tracked_dashboard"

type trackedDashboardConfigBody struct {
	Widgets SiteDomainWidgets `json:"widgets"`
}

func registerTrackedKillboardRoutes(
	a huma.API,
	service *trackerAccountService,
	requiredSession []map[string][]string,
) {
	registerLegacy(a, huma.Operation{
		OperationID: "account-tracked-config",
		Method:      http.MethodGet,
		Path:        "/me/tracked/config",
		Summary:     "Personal tracked-killboard configuration",
		Description: "Returns the private dashboard layout. A missing saved configuration signals first-visit onboarding.",
		Tags:        []string{"account", "trackers", "killboard"},
		Security:    requiredSession,
	}, service.trackedConfigHandler())
	registerLegacy(a, documentJSONBody[trackedDashboardConfigBody](a, huma.Operation{
		OperationID: "account-tracked-config-update",
		Method:      http.MethodPut,
		Path:        "/me/tracked/config",
		Summary:     "Save personal tracked-killboard configuration",
		Description: "Stores widget placement independently of tracker and notification settings.",
		Tags:        []string{"account", "trackers", "killboard"},
		Security:    requiredSession,
	}), service.saveTrackedConfigHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "account-tracked-summary",
		Method:      http.MethodGet,
		Path:        "/me/tracked/summary",
		Summary:     "Personal tracked-killboard summary",
		Description: "Summarizes the unique killmails recorded by this account's trackers during the requested window.",
		Tags:        []string{"account", "trackers", "killboard"},
		Security:    requiredSession,
	}, service.trackedSummaryHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "account-tracked-killmails",
		Method:      http.MethodGet,
		Path:        "/me/tracked/killmails",
		Summary:     "Personal tracked killmail list",
		Description: "Combines recorded events across the account's current trackers and deduplicates killmails matched by more than one tracker.",
		Tags:        []string{"account", "trackers", "killboard", "killmails"},
		Security:    requiredSession,
	}, service.trackedKillmailsHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "account-tracked-statistics",
		Method:      http.MethodGet,
		Path:        "/me/tracked/stats",
		Summary:     "Personal tracked-killboard statistics",
		Description: "Builds leaderboards and most-valuable lists from unique killmails recorded by the account's trackers.",
		Tags:        []string{"account", "trackers", "killboard", "statistics"},
		Security:    requiredSession,
	}, service.trackedStatisticsHandler())
}

func defaultTrackedDashboardWidgets() SiteDomainWidgets {
	ratio := "250px_1fr"
	return SiteDomainWidgets{
		Top: []SiteDomainWidget{{Type: "mostValuable", Enabled: true}},
		Left: []SiteDomainWidget{
			{Type: "entityInfo", Enabled: true},
			{Type: "topCharacters", Enabled: true},
			{Type: "topCorporations", Enabled: true},
			{Type: "topAlliances", Enabled: true},
			{Type: "topShips", Enabled: true},
			{Type: "topSystems", Enabled: true},
			{Type: "topRegions", Enabled: true},
		},
		Right:       []SiteDomainWidget{{Type: "killList", Enabled: true}},
		ColumnRatio: &ratio,
	}
}

func (s *trackerAccountService) trackedConfigHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		var raw []byte
		err = s.db.QueryRow(ctx, `
			SELECT value FROM user_config
			WHERE character_id = $1 AND key = $2`,
			principal.CharacterID, trackedDashboardConfigKey).Scan(&raw)
		if errors.Is(err, pgx.ErrNoRows) {
			return accountNoStorePayload(map[string]any{
				"configured": false, "widgets": defaultTrackedDashboardWidgets(),
			}), nil
		}
		if err != nil {
			return legacyPayload{}, err
		}
		var stored trackedDashboardConfigBody
		if err := json.Unmarshal(raw, &stored); err != nil {
			return legacyPayload{}, err
		}
		widgets, valid := sanitizeTrackedDashboardWidgets(stored.Widgets)
		if !valid {
			widgets = defaultTrackedDashboardWidgets()
		}
		return accountNoStorePayload(map[string]any{
			"configured": true, "widgets": widgets,
		}), nil
	}
}

func (s *trackerAccountService) saveTrackedConfigHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[trackedDashboardConfigBody](req, accountBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		widgets, valid := sanitizeTrackedDashboardWidgets(body.Widgets)
		if !valid {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Dashboard needs at least one valid enabled widget")
		}
		raw, err := json.Marshal(trackedDashboardConfigBody{Widgets: widgets})
		if err != nil {
			return legacyPayload{}, err
		}
		if _, err := s.db.Exec(ctx, `
			INSERT INTO user_config (character_id, key, value, updated_at)
			VALUES ($1, $2, $3::jsonb, now())
			ON CONFLICT (character_id, key) DO UPDATE SET
			  value = excluded.value, updated_at = excluded.updated_at`,
			principal.CharacterID, trackedDashboardConfigKey, raw); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"configured": true, "widgets": widgets,
		}), nil
	}
}

func sanitizeTrackedDashboardWidgets(input SiteDomainWidgets) (SiteDomainWidgets, bool) {
	allowed := map[string]bool{
		"mostValuable": true, "killList": true, "topCharacters": true,
		"topCorporations": true, "topAlliances": true, "topShips": true,
		"topSystems": true, "topRegions": true, "entityInfo": true,
		"textBlock": true,
	}
	seen := map[string]bool{}
	enabled := 0
	total := 0
	sanitize := func(items []SiteDomainWidget) []SiteDomainWidget {
		result := make([]SiteDomainWidget, 0, min(len(items), 10))
		for _, widget := range items {
			if len(result) >= 10 || total >= 20 || !allowed[widget.Type] {
				continue
			}
			if widget.Type != "textBlock" && seen[widget.Type] {
				continue
			}
			seen[widget.Type] = true
			if widget.Type == "killList" {
				kind := "latest"
				if widget.KilllistType != nil {
					if _, ok := killtype.Predicates()[*widget.KilllistType]; ok {
						kind = *widget.KilllistType
					}
				}
				widget.KilllistType = &kind
			}
			if widget.Type == "textBlock" && widget.Content != nil {
				content := strings.TrimSpace(*widget.Content)
				runes := []rune(content)
				if len(runes) > 2000 {
					content = string(runes[:2000])
				}
				widget.Content = &content
			}
			if widget.Enabled {
				enabled++
			}
			total++
			result = append(result, widget)
		}
		return result
	}
	result := SiteDomainWidgets{
		Top: sanitize(input.Top), Left: sanitize(input.Left), Right: sanitize(input.Right),
	}
	ratio := "250px_1fr"
	if input.ColumnRatio != nil {
		switch *input.ColumnRatio {
		case "250px_1fr", "300px_1fr", "1fr_1fr", "1fr_2fr", "1fr_3fr":
			ratio = *input.ColumnRatio
		}
	}
	result.ColumnRatio = &ratio
	return result, total > 0 && enabled > 0
}

func (s *trackerAccountService) trackedSummaryHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		days := boundedQueryInt(req, "days", 7, 1, 90)
		since := time.Now().UTC().AddDate(0, 0, -days)
		row, err := queryMap(ctx, s.db, `
			WITH tracked_kills AS MATERIALIZED (
				SELECT DISTINCT event.killmail_id
				FROM entity_tracker_events event
				JOIN entity_trackers tracker
				  ON tracker.tracker_id = event.tracker_id
				WHERE tracker.character_id = $1
				  AND event.occurred_at >= $2
			)
			SELECT
				(SELECT COUNT(*) FROM entity_trackers
				 WHERE character_id = $1)::bigint AS tracker_count,
				(SELECT COUNT(*) FROM entity_trackers
				 WHERE character_id = $1 AND enabled)::bigint AS active_tracker_count,
				COUNT(k.killmail_id)::bigint AS kill_count,
				COALESCE(SUM(k.total_value), 0)::double precision AS total_value,
				MAX(k.killmail_time) AS last_kill_at,
				$3::integer AS window_days
			FROM tracked_kills tracked
			JOIN killmails k ON k.killmail_id = tracked.killmail_id`,
			principal.CharacterID, since, days)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(row), nil
	}
}

func (s *trackerAccountService) trackedKillmailsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		limit := boundedQueryInt(req, "limit", 50, 10, 100)
		after, err := optionalPositiveInt64(req.Query.Get("after"))
		if err != nil {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid after")
		}
		kind := strings.TrimSpace(req.Query.Get("type"))
		if kind == "" {
			kind = "latest"
		}
		predicate, known := killtype.Predicates()[kind]
		if !known {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Unknown killlist type")
		}
		args := []any{principal.CharacterID}
		afterClause := ""
		if after != nil {
			args = append(args, *after)
			afterClause = " AND event.killmail_id < $2"
		}
		args = append(args, limit+1)
		query := trackedBoardKilllistQuery(afterClause, predicate, len(args))
		rows, err := queryMaps(ctx, s.db, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, hasMore, cursor, err := finishUniverseKilllist(ctx, s.db, rows, limit)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"kills": rows, "hasMore": hasMore, "cursor": cursor,
		}), nil
	}
}

func trackedBoardKilllistQuery(afterClause, predicate string, limitParameter int) string {
	if predicate == "" {
		predicate = "TRUE"
	}
	return `
		WITH tracked_kills AS MATERIALIZED (
			SELECT DISTINCT event.killmail_id
			FROM entity_tracker_events event
			JOIN entity_trackers tracker
			  ON tracker.tracker_id = event.tracker_id
			JOIN killmails k ON k.killmail_id = event.killmail_id
			WHERE tracker.character_id = $1` + afterClause + `
			  AND (` + predicate + `)
			ORDER BY event.killmail_id DESC
			LIMIT $` + strconv.Itoa(limitParameter) + `
		)
	` + strings.Replace(campaignKilllistSelect,
		"FROM killmails k",
		"FROM tracked_kills tracked\n\tJOIN killmails k ON k.killmail_id = tracked.killmail_id",
		1) + `
		ORDER BY k.killmail_id DESC`
}

func (s *trackerAccountService) trackedStatisticsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		dataType := strings.TrimSpace(req.Query.Get("dataType"))
		if dataType == "" {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Missing dataType parameter")
		}
		limit := boundedQueryInt(req, "limit", 10, 1, 100)
		days := boundedQueryInt(req, "days", 7, 1, 90)
		since := time.Now().UTC().AddDate(0, 0, -days)

		var rows []map[string]any
		switch dataType {
		case "characters", "corporations", "alliances", "ships", "systems", "regions":
			rows, err = loadTrackedTopStatistics(
				ctx, s.db, principal.CharacterID, dataType, since, limit,
			)
		case "most_valuable_kills", "most_valuable_ships", "most_valuable_structures":
			rows, err = loadTrackedMostValuable(
				ctx, s.db, principal.CharacterID, dataType, since, limit,
			)
		default:
			return legacyPayload{}, apiError(http.StatusBadRequest, "Unknown dataType: "+dataType)
		}
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"entries": nonNilUniverseRows(rows)}), nil
	}
}

type trackedTopSpec struct {
	column     string
	table      string
	idColumn   string
	nameColumn string
	resultType string
	location   bool
}

var trackedTopSpecs = map[string]trackedTopSpec{
	"characters": {
		"character_id", "characters", "character_id", "name", "character", false,
	},
	"corporations": {
		"corporation_id", "corporations", "corporation_id", "name", "corporation", false,
	},
	"alliances": {
		"alliance_id", "alliances", "alliance_id", "name", "alliance", false,
	},
	"ships": {
		"ship_type_id", "inv_types", "type_id", "name", "ship", false,
	},
	"systems": {
		"solar_system_id", "solar_systems", "solar_system_id", "system_name", "system", true,
	},
	"regions": {
		"region_id", "regions", "region_id", "name", "region", true,
	},
}

func loadTrackedTopStatistics(
	ctx context.Context,
	db Database,
	characterID int32,
	dataType string,
	since time.Time,
	limit int,
) ([]map[string]any, error) {
	spec, ok := trackedTopSpecs[dataType]
	if !ok {
		return []map[string]any{}, nil
	}
	var query string
	if spec.location {
		regionColumn := "NULL::integer"
		if dataType == "systems" {
			regionColumn = "MAX(k.region_id)"
		}
		query = `
			WITH tracked_kills AS MATERIALIZED (
				SELECT DISTINCT event.killmail_id
				FROM entity_tracker_events event
				JOIN entity_trackers tracker
				  ON tracker.tracker_id = event.tracker_id
				WHERE tracker.character_id = $1
				  AND event.occurred_at >= $2
			)
			SELECT k.` + spec.column + ` AS id,
			       COALESCE(entity.` + spec.nameColumn + `, 'Unknown') AS name,
			       COUNT(*)::bigint AS count,
			       '` + spec.resultType + `'::text AS type,
			       ` + regionColumn + ` AS region_id
			FROM tracked_kills tracked
			JOIN killmails k ON k.killmail_id = tracked.killmail_id
			LEFT JOIN ` + spec.table + ` entity
			  ON entity.` + spec.idColumn + ` = k.` + spec.column + `
			WHERE k.` + spec.column + ` IS NOT NULL
			GROUP BY k.` + spec.column + `, entity.` + spec.nameColumn + `
			ORDER BY count DESC, id
			LIMIT $3`
	} else {
		query = `
			WITH tracked_kills AS MATERIALIZED (
				SELECT DISTINCT event.killmail_id
				FROM entity_tracker_events event
				JOIN entity_trackers tracker
				  ON tracker.tracker_id = event.tracker_id
				WHERE tracker.character_id = $1
				  AND event.occurred_at >= $2
			)
			SELECT attacker.` + spec.column + ` AS id,
			       COALESCE(entity.` + spec.nameColumn + `, 'Unknown') AS name,
			       COUNT(DISTINCT attacker.killmail_id)::bigint AS count,
			       '` + spec.resultType + `'::text AS type,
			       NULL::integer AS region_id
			FROM tracked_kills tracked
			JOIN killmail_attackers attacker
			  ON attacker.killmail_id = tracked.killmail_id
			LEFT JOIN ` + spec.table + ` entity
			  ON entity.` + spec.idColumn + ` = attacker.` + spec.column + `
			WHERE attacker.` + spec.column + ` IS NOT NULL
			GROUP BY attacker.` + spec.column + `, entity.` + spec.nameColumn + `
			ORDER BY count DESC, id
			LIMIT $3`
	}
	rows, err := queryMaps(ctx, db, query, characterID, since, limit)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row["count"] = int64OrZero(row["count"])
	}
	return rows, nil
}

func loadTrackedMostValuable(
	ctx context.Context,
	db Database,
	characterID int32,
	dataType string,
	since time.Time,
	limit int,
) ([]map[string]any, error) {
	category := 0
	switch dataType {
	case "most_valuable_ships":
		category = 6
	case "most_valuable_structures":
		category = 65
	}
	categorySQL := ""
	args := []any{characterID, since}
	if category != 0 {
		args = append(args, category)
		categorySQL = `
			  AND k.victim_ship_group_id IN (
				SELECT group_id FROM inv_groups WHERE category_id = $3
			  )`
	}
	args = append(args, limit)
	query := `
		WITH tracked_kills AS MATERIALIZED (
			SELECT DISTINCT event.killmail_id
			FROM entity_tracker_events event
			JOIN entity_trackers tracker
			  ON tracker.tracker_id = event.tracker_id
			WHERE tracker.character_id = $1
			  AND event.occurred_at >= $2
		)
		SELECT k.killmail_id, k.killmail_hash,
		       k.victim_ship_type_id AS ship_type_id,
		       COALESCE(ship.name, 'Unknown') AS ship_name,
		       COALESCE(k.total_value, 0)::double precision AS total_value,
		       k.victim_character_id,
		       character.name AS victim_character_name,
		       corporation.name AS victim_corporation_name,
		       alliance.name AS victim_alliance_name
		FROM tracked_kills tracked
		JOIN killmails k ON k.killmail_id = tracked.killmail_id
		JOIN inv_types ship ON ship.type_id = k.victim_ship_type_id
		LEFT JOIN characters character
		  ON character.character_id = k.victim_character_id
		LEFT JOIN corporations corporation
		  ON corporation.corporation_id = k.victim_corporation_id
		LEFT JOIN alliances alliance
		  ON alliance.alliance_id = k.victim_alliance_id
		WHERE true` + categorySQL + `
		ORDER BY k.total_value DESC, k.killmail_id DESC
		LIMIT $` + strconv.Itoa(len(args))
	rows, err := queryMaps(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row["total_value"] = math.Max(0, float64OrZero(row["total_value"]))
	}
	return rows, nil
}
