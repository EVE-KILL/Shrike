package api

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var discordSnowflakePattern = regexp.MustCompile(`^[0-9]{15,22}$`)

func (s *adminService) usersHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		page := adminBoundedNumber(req.Query.Get("page"), 1, 1, math.MaxInt32)
		limit := adminBoundedNumber(req.Query.Get("limit"), 50, 1, 100)
		search := strings.TrimSpace(req.Query.Get("search"))
		sortColumn := "last_login"
		switch req.Query.Get("sort") {
		case "created_at":
			sortColumn = "created_at"
		case "character_name":
			sortColumn = "character_name"
		}
		direction := "DESC"
		if req.Query.Get("dir") == "asc" {
			direction = "ASC"
		}
		offset := (page - 1) * limit
		rows, err := queryMaps(ctx, s.opts.DB, `
			WITH matched AS (
				SELECT users.character_id, users.character_name,
				       users.character_owner_hash, users.is_admin,
				       users.last_login, users.created_at,
				       character.corporation_id,
				       corporation.name AS corporation_name,
				       character.alliance_id,
				       alliance.name AS alliance_name
				FROM users
				LEFT JOIN characters character
				  ON character.character_id = users.character_id
				LEFT JOIN corporations corporation
				  ON corporation.corporation_id = character.corporation_id
				LEFT JOIN alliances alliance
				  ON alliance.alliance_id = character.alliance_id
				WHERE $1::text = ''
				   OR users.character_name ILIKE '%' || $1 || '%'
				   OR CAST(users.character_id AS TEXT) LIKE '%' || $1 || '%'
			),
			result_count AS (
				SELECT COUNT(*)::bigint AS total FROM matched
			),
			page AS (
				SELECT * FROM matched
				ORDER BY `+sortColumn+` `+direction+`
				LIMIT $2 OFFSET $3
			)
			SELECT page.*, result_count.total AS _total
			FROM result_count
			LEFT JOIN page ON TRUE
			ORDER BY (page.character_id IS NULL),
			         page.`+sortColumn+` `+direction,
			search, limit, offset)
		if err != nil {
			return legacyPayload{}, err
		}
		total := int64(0)
		if len(rows) > 0 {
			total = int64OrZero(rows[0]["_total"])
		}
		users := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			if row["character_id"] == nil {
				continue
			}
			delete(row, "_total")
			users = append(users, row)
		}
		pages := int64(0)
		if limit > 0 {
			pages = int64(math.Ceil(float64(total) / float64(limit)))
		}
		return accountNoStorePayload(map[string]any{
			"users": users, "total": total, "page": page,
			"limit": limit, "pages": pages,
		}), nil
	}
}

func adminBoundedNumber(raw string, fallback, minimum, maximum int) int {
	number := float64(fallback)
	if strings.TrimSpace(raw) != "" {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err == nil && !math.IsNaN(parsed) && parsed != 0 {
			number = parsed
		}
	}
	if number < float64(minimum) {
		number = float64(minimum)
	}
	if number > float64(maximum) {
		number = float64(maximum)
	}
	return int(number)
}

func (s *adminService) userDetailHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		characterID, err := parseAdminCharacterID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		user, err := queryMap(ctx, s.opts.DB, `
			SELECT users.character_id, users.character_name,
			       users.character_owner_hash, users.is_admin,
			       users.discord_user_id, users.last_login,
			       users.created_at, users.updated_at,
			       character.corporation_id,
			       corporation.name AS corporation_name,
			       character.alliance_id,
			       alliance.name AS alliance_name
			FROM users
			LEFT JOIN characters character
			  ON character.character_id = users.character_id
			LEFT JOIN corporations corporation
			  ON corporation.corporation_id = character.corporation_id
			LEFT JOIN alliances alliance
			  ON alliance.alliance_id = character.alliance_id
			WHERE users.character_id = $1
			LIMIT 1`, characterID)
		if err != nil {
			return legacyPayload{}, err
		}
		if user == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "User not found")
		}
		results, err := queryMapsConcurrent(ctx, s.opts.DB,
			databaseQuery{
				SQL: `
					SELECT scopes, token_expiry, last_fetched, created_at
					FROM user_esi_tokens
					WHERE character_id = $1 LIMIT 1`,
				Args: []any{characterID},
			},
			databaseQuery{
				SQL: `
					SELECT key, value, updated_at
					FROM user_config
					WHERE character_id = $1`,
				Args: []any{characterID},
			},
			databaseQuery{
				SQL: `
					SELECT COUNT(*)::bigint AS total_requests,
					       COUNT(*) FILTER (
					         WHERE success = FALSE
					       )::bigint AS total_errors,
					       COALESCE(SUM(new_items), 0)::bigint AS total_new_items,
					       MAX(created_at)::text AS last_request
					FROM esi_request_logs
					WHERE character_id = $1`,
				Args: []any{characterID},
			},
		)
		if err != nil {
			return legacyPayload{}, err
		}
		var token any
		if len(results[0]) > 0 {
			token = results[0][0]
		}
		config := nonNilAdminRows(results[1])
		stats := map[string]any{
			"total_requests": int64(0), "total_errors": int64(0),
			"total_new_items": int64(0), "last_request": nil,
		}
		if len(results[2]) > 0 {
			stats = results[2][0]
		}
		return accountNoStorePayload(map[string]any{
			"user": user, "config": config,
			"esiToken": token, "esiStats": stats,
		}), nil
	}
}

func parseAdminCharacterID(raw string) (int32, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil || value <= 0 {
		return 0, apiError(http.StatusBadRequest, "Invalid character ID")
	}
	return int32(value), nil
}

// adminSetDiscordBody carries the Discord link for a user.
type adminSetDiscordBody struct {
	DiscordUserID json.RawMessage `json:"discord_user_id,omitempty" doc:"Discord user identifier, or null to unlink."`
}

func (s *adminService) setDiscordHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, true); err != nil {
			return legacyPayload{}, err
		}
		characterID, err := parseAdminCharacterID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[adminSetDiscordBody](req, 64<<10)
		if err != nil {
			return legacyPayload{}, err
		}
		var discordID *string
		if raw, exists := rawJSONField(body.DiscordUserID); exists && raw != nil {
			text := strings.TrimSpace(adminStringValue(raw))
			if text != "" {
				if !discordSnowflakePattern.MatchString(text) {
					return legacyPayload{}, apiError(
						http.StatusBadRequest,
						"discord_user_id must be a numeric snowflake (15–22 digits) or null",
					)
				}
				discordID = &text
			}
		}
		db, err := mutationDatabase(s.opts)
		if err != nil {
			return legacyPayload{}, err
		}
		updated, err := queryMap(ctx, db, `
			UPDATE users SET discord_user_id = $2
			WHERE character_id = $1
			RETURNING discord_user_id`, characterID, discordID)
		if err != nil {
			var postgresError *pgconn.PgError
			if errors.As(err, &postgresError) && postgresError.Code == "23505" {
				return legacyPayload{}, apiError(
					http.StatusConflict,
					"That Discord id is already linked to another character",
				)
			}
			return legacyPayload{}, err
		}
		if updated == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "User not found")
		}
		return accountNoStorePayload(map[string]any{
			"character_id":    characterID,
			"discord_user_id": updated["discord_user_id"],
		}), nil
	}
}

func adminStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return string(typed)
	case bool:
		return strconv.FormatBool(typed)
	default:
		// These fail the numeric-snowflake regex, matching String(object).
		return "[object Object]"
	}
}

func (s *adminService) toggleAdminHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		admin, err := s.requireAdmin(ctx, req, true)
		if err != nil {
			return legacyPayload{}, err
		}
		characterID, err := parseAdminCharacterID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		if characterID == admin.CharacterID {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Cannot change your own admin status",
			)
		}
		db, err := mutationDatabase(s.opts)
		if err != nil {
			return legacyPayload{}, err
		}
		updated, err := queryMap(ctx, db, `
			UPDATE users SET is_admin = NOT is_admin
			WHERE character_id = $1
			RETURNING is_admin`, characterID)
		if err != nil {
			return legacyPayload{}, err
		}
		if updated == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "User not found")
		}
		return accountNoStorePayload(map[string]any{
			"character_id": characterID,
			"is_admin":     updated["is_admin"],
		}), nil
	}
}

func nonNilAdminRows(rows []map[string]any) []map[string]any {
	if rows == nil {
		return []map[string]any{}
	}
	return rows
}
