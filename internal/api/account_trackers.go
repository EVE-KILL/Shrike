package api

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const maxAccountTrackers = 50

var trackerTargetTypes = map[string]bool{
	"character": true, "corporation": true, "alliance": true,
	"system": true, "constellation": true, "region": true,
}

type trackerAccountService struct {
	auth *authService
	db   MutationDatabase
	err  error
}

type trackerCreateBody struct {
	TargetType           string `json:"target_type" enum:"character,corporation,alliance,system,constellation,region"`
	TargetID             int64  `json:"target_id" minimum:"1"`
	NotificationsEnabled bool   `json:"notifications_enabled,omitempty"`
}

type trackerUpdateBody struct {
	Enabled              *bool `json:"enabled,omitempty"`
	NotificationsEnabled *bool `json:"notifications_enabled,omitempty"`
}

type trackerNotificationsReadBody struct {
	All bool `json:"all,omitempty" default:"true" doc:"Marks every unread tracker notification when true."`
}

type trackerTarget struct {
	Name   string
	Ticker *string
}

func newTrackerAccountService(opts Options) *trackerAccountService {
	db, err := mutationDatabase(opts)
	return &trackerAccountService{auth: newAuthService(opts), db: db, err: err}
}

func registerTrackerAccountRoutes(
	a huma.API,
	opts Options,
	requiredSession []map[string][]string,
) {
	service := newTrackerAccountService(opts)
	registerTrackedKillboardRoutes(a, service, requiredSession)
	registerLegacy(a, huma.Operation{
		OperationID: "account-trackers",
		Method:      http.MethodGet,
		Path:        "/me/trackers",
		Summary:     "Current account entity trackers",
		Description: "Lists persistent trackers independently of their optional notification delivery setting.",
		Tags:        []string{"account", "trackers"},
		Security:    requiredSession,
	}, service.listHandler())
	registerLegacy(a, documentJSONBody[trackerCreateBody](a, huma.Operation{
		OperationID:   "account-tracker-create",
		Method:        http.MethodPost,
		Path:          "/me/trackers",
		Summary:       "Create an entity tracker",
		Description:   "Creates a tracker for future matching killmails. Notification delivery is optional and disabled by default.",
		Tags:          []string{"account", "trackers"},
		Security:      requiredSession,
		DefaultStatus: http.StatusCreated,
	}), service.createHandler())
	registerLegacy(a, documentJSONBody[trackerUpdateBody](a, huma.Operation{
		OperationID: "account-tracker-update",
		Method:      http.MethodPatch,
		Path:        "/me/trackers/{id}",
		Summary:     "Update an entity tracker",
		Description: "Pauses or resumes tracking and independently toggles future notifications.",
		Tags:        []string{"account", "trackers"},
		Security:    requiredSession,
	}), service.updateHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "account-tracker-delete",
		Method:      http.MethodDelete,
		Path:        "/me/trackers/{id}",
		Summary:     "Delete an entity tracker",
		Tags:        []string{"account", "trackers"},
		Security:    requiredSession,
	}, service.deleteHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "account-tracker-killmails",
		Method:      http.MethodGet,
		Path:        "/me/trackers/{id}/killmails",
		Summary:     "Killmails recorded by one tracker",
		Description: "Returns only events observed after the tracker was created; enabling notifications does not change tracking history.",
		Tags:        []string{"account", "trackers", "killmails"},
		Security:    requiredSession,
	}, service.killmailsHandler())
	registerLegacy(a, huma.Operation{
		OperationID: "account-tracker-notifications",
		Method:      http.MethodGet,
		Path:        "/me/tracker-notifications",
		Summary:     "Tracker notifications for this account",
		Description: "Lists notification records created only by trackers whose notification toggle was enabled when the killmail arrived.",
		Tags:        []string{"account", "notifications", "trackers"},
		Security:    requiredSession,
	}, service.notificationsHandler())
	registerLegacy(a, documentJSONBody[trackerNotificationsReadBody](a, huma.Operation{
		OperationID: "account-tracker-notifications-read",
		Method:      http.MethodPost,
		Path:        "/me/tracker-notifications/read",
		Summary:     "Mark tracker notifications as read",
		Tags:        []string{"account", "notifications", "trackers"},
		Security:    requiredSession,
	}), service.markNotificationsReadHandler())
}

func (s *trackerAccountService) principal(
	ctx context.Context,
	req *legacyRequest,
) (*Principal, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.db == nil {
		return nil, apiError(http.StatusServiceUnavailable, "Tracker storage is not configured")
	}
	return s.auth.requirePrincipal(ctx, req)
}

func (s *trackerAccountService) listHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, err := queryMaps(ctx, s.db, `
			SELECT t.tracker_id AS id, t.target_type, t.target_id,
			       t.target_name, t.target_ticker, t.enabled,
			       t.notifications_enabled, t.created_at, t.updated_at,
			       COUNT(e.event_id)::bigint AS event_count,
			       MAX(e.occurred_at) AS last_event_at,
			       COUNT(n.notification_id) FILTER (WHERE n.read_at IS NULL)::bigint
			           AS unread_notifications
			FROM entity_trackers t
			LEFT JOIN entity_tracker_events e ON e.tracker_id = t.tracker_id
			LEFT JOIN entity_tracker_notifications n ON n.event_id = e.event_id
			WHERE t.character_id = $1
			GROUP BY t.tracker_id
			ORDER BY t.created_at DESC, t.tracker_id DESC`, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"trackers": rows, "limit": maxAccountTrackers,
		}), nil
	}
}

func (s *trackerAccountService) createHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[trackerCreateBody](req, accountBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		kind := strings.ToLower(strings.TrimSpace(body.TargetType))
		if !trackerTargetTypes[kind] || body.TargetID <= 0 || body.TargetID > math.MaxInt32 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid tracker target")
		}
		target, err := resolveTrackerTarget(ctx, s.db, kind, int32(body.TargetID))
		if errors.Is(err, pgx.ErrNoRows) {
			return legacyPayload{}, apiError(http.StatusNotFound, "Tracker target not found")
		}
		if err != nil {
			return legacyPayload{}, err
		}

		tx, err := s.db.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2)`,
			20_260_904, principal.CharacterID); err != nil {
			return legacyPayload{}, err
		}
		var count int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM entity_trackers WHERE character_id = $1`,
			principal.CharacterID).Scan(&count); err != nil {
			return legacyPayload{}, err
		}
		if count >= maxAccountTrackers {
			return legacyPayload{}, apiError(http.StatusConflict, "Tracker limit reached")
		}
		row, err := queryTrackerInsert(ctx, tx, principal.CharacterID, kind,
			int32(body.TargetID), target, body.NotificationsEnabled)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return legacyPayload{}, apiError(http.StatusConflict, "Tracker already exists")
			}
			return legacyPayload{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		return legacyPayload{Status: http.StatusCreated, ContentType: "application/json", Body: row}, nil
	}
}

func (s *trackerAccountService) updateHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := trackerPathID(req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[trackerUpdateBody](req, accountBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		if body.Enabled == nil && body.NotificationsEnabled == nil {
			return legacyPayload{}, apiError(http.StatusBadRequest, "No tracker changes supplied")
		}
		row, err := queryMap(ctx, s.db, `
			WITH updated AS (
				UPDATE entity_trackers
				SET enabled = COALESCE($3, enabled),
				    notifications_enabled = COALESCE($4, notifications_enabled),
				    updated_at = now()
				WHERE tracker_id = $1 AND character_id = $2
				RETURNING *
			)
			SELECT updated.tracker_id AS id, updated.target_type,
			       updated.target_id, updated.target_name, updated.target_ticker,
			       updated.enabled, updated.notifications_enabled,
			       updated.created_at, updated.updated_at,
			       COUNT(event.event_id)::bigint AS event_count,
			       MAX(event.occurred_at) AS last_event_at,
			       COUNT(notification.notification_id)
			           FILTER (WHERE notification.read_at IS NULL)::bigint
			           AS unread_notifications
			FROM updated
			LEFT JOIN entity_tracker_events event
			    ON event.tracker_id = updated.tracker_id
			LEFT JOIN entity_tracker_notifications notification
			    ON notification.event_id = event.event_id
			GROUP BY updated.tracker_id, updated.target_type, updated.target_id,
			         updated.target_name, updated.target_ticker, updated.enabled,
			         updated.notifications_enabled, updated.created_at,
			         updated.updated_at`,
			id, principal.CharacterID, body.Enabled, body.NotificationsEnabled)
		if row == nil && err == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "Tracker not found")
		}
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(row), nil
	}
}

func (s *trackerAccountService) deleteHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := trackerPathID(req)
		if err != nil {
			return legacyPayload{}, err
		}
		command, err := s.db.Exec(ctx, `
			DELETE FROM entity_trackers WHERE tracker_id = $1 AND character_id = $2`,
			id, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, err
		}
		if command.RowsAffected() == 0 {
			return legacyPayload{}, apiError(http.StatusNotFound, "Tracker not found")
		}
		return accountNoStorePayload(map[string]any{"deleted": true}), nil
	}
}

func (s *trackerAccountService) killmailsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := trackerPathID(req)
		if err != nil {
			return legacyPayload{}, err
		}
		var exists bool
		if err := s.db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM entity_trackers WHERE tracker_id = $1 AND character_id = $2
			)`, id, principal.CharacterID).Scan(&exists); err != nil {
			return legacyPayload{}, err
		}
		if !exists {
			return legacyPayload{}, apiError(http.StatusNotFound, "Tracker not found")
		}
		limit := boundedQueryInt(req, "limit", 50, 10, 100)
		after, err := optionalPositiveInt64(req.Query.Get("after"))
		if err != nil {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid after")
		}
		args := []any{id, principal.CharacterID}
		afterClause := ""
		if after != nil {
			args = append(args, *after)
			afterClause = " AND event.killmail_id < $3"
		}
		args = append(args, limit+1)
		query := trackerKilllistQuery(afterClause, len(args))
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

func (s *trackerAccountService) notificationsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		limit := boundedQueryInt(req, "limit", 50, 1, 100)
		rows, err := queryMaps(ctx, s.db, `
			SELECT n.notification_id AS id, n.created_at, n.read_at IS NOT NULL AS is_read,
			       t.tracker_id, t.target_type, t.target_id, t.target_name,
			       t.target_ticker, e.match_role, e.killmail_id, e.occurred_at,
			       COALESCE(k.total_value, 0) AS total_value,
			       k.victim_ship_type_id AS ship_type_id, ship.name AS ship_name,
			       k.victim_character_id, victim.name AS victim_character_name,
			       k.solar_system_id, system.system_name AS solar_system_name,
			       k.region_id, region.name AS region_name
			FROM entity_tracker_notifications n
			JOIN entity_tracker_events e ON e.event_id = n.event_id
			JOIN entity_trackers t ON t.tracker_id = e.tracker_id
			JOIN killmails k ON k.killmail_id = e.killmail_id
			LEFT JOIN inv_types ship ON ship.type_id = k.victim_ship_type_id
			LEFT JOIN characters victim ON victim.character_id = k.victim_character_id
			LEFT JOIN solar_systems system ON system.solar_system_id = k.solar_system_id
			LEFT JOIN regions region ON region.region_id = k.region_id
			WHERE n.character_id = $1
			ORDER BY n.notification_id DESC
			LIMIT $2`, principal.CharacterID, limit)
		if err != nil {
			return legacyPayload{}, err
		}
		var unread int64
		if err := s.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM entity_tracker_notifications
			WHERE character_id = $1 AND read_at IS NULL`, principal.CharacterID).Scan(&unread); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"notifications": rows, "unreadCount": unread,
		}), nil
	}
}

func (s *trackerAccountService) markNotificationsReadHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[trackerNotificationsReadBody](req, accountBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		if !body.All {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Set all to true")
		}
		command, err := s.db.Exec(ctx, `
			UPDATE entity_tracker_notifications
			SET read_at = now()
			WHERE character_id = $1 AND read_at IS NULL`, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"updated": command.RowsAffected(), "unreadCount": 0,
		}), nil
	}
}

func trackerKilllistQuery(afterClause string, limitParameter int) string {
	return `
		WITH tracked_kills AS MATERIALIZED (
			SELECT event.killmail_id
			FROM entity_tracker_events event
			JOIN entity_trackers tracker
			  ON tracker.tracker_id = event.tracker_id
			WHERE event.tracker_id = $1
			  AND tracker.character_id = $2` + afterClause + `
			ORDER BY event.killmail_id DESC
			LIMIT $` + strconv.Itoa(limitParameter) + `
		)
	` + strings.Replace(campaignKilllistSelect,
		"FROM killmails k",
		"FROM tracked_kills tracked\n\tJOIN killmails k ON k.killmail_id = tracked.killmail_id",
		1) + `
		ORDER BY k.killmail_id DESC`
}

func resolveTrackerTarget(
	ctx context.Context,
	db Database,
	targetType string,
	targetID int32,
) (trackerTarget, error) {
	queries := map[string]string{
		"character":     `SELECT name, NULL::text FROM characters WHERE character_id = $1`,
		"corporation":   `SELECT name, ticker FROM corporations WHERE corporation_id = $1`,
		"alliance":      `SELECT name, ticker FROM alliances WHERE alliance_id = $1`,
		"system":        `SELECT system_name, NULL::text FROM solar_systems WHERE solar_system_id = $1`,
		"constellation": `SELECT constellation_name, NULL::text FROM constellations WHERE constellation_id = $1`,
		"region":        `SELECT name, NULL::text FROM regions WHERE region_id = $1`,
	}
	query, ok := queries[targetType]
	if !ok {
		return trackerTarget{}, apiError(http.StatusBadRequest, "Invalid tracker target")
	}
	var target trackerTarget
	if err := db.QueryRow(ctx, query, targetID).Scan(&target.Name, &target.Ticker); err != nil {
		return trackerTarget{}, err
	}
	return target, nil
}

func queryTrackerInsert(
	ctx context.Context,
	tx pgx.Tx,
	characterID int32,
	targetType string,
	targetID int32,
	target trackerTarget,
	notificationsEnabled bool,
) (map[string]any, error) {
	return queryMap(ctx, txDatabase{tx}, `
		INSERT INTO entity_trackers (
			character_id, target_type, target_id, target_name,
			target_ticker, notifications_enabled
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING tracker_id AS id, target_type, target_id, target_name,
		          target_ticker, enabled, notifications_enabled,
		          created_at, updated_at,
		          0::bigint AS event_count, NULL::timestamptz AS last_event_at,
		          0::bigint AS unread_notifications`,
		characterID, targetType, targetID, target.Name, target.Ticker, notificationsEnabled)
}

func trackerPathID(req *legacyRequest) (int64, error) {
	id, err := strconv.ParseInt(req.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, apiError(http.StatusBadRequest, "Invalid tracker ID")
	}
	return id, nil
}
