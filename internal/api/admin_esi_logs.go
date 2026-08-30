package api

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type adminESILogFilters struct {
	CharacterID, CorporationID *float64
	Search, Source, Status     string
	EndpointType               string
	HasNew                     bool
	AfterID                    *float64
	Page, Limit                int
}

func parseAdminESILogFilters(req *legacyRequest) adminESILogFilters {
	filters := adminESILogFilters{
		Search:       strings.TrimSpace(req.Query.Get("search")),
		Source:       strings.TrimSpace(req.Query.Get("source")),
		Status:       req.Query.Get("status"),
		EndpointType: req.Query.Get("endpoint_type"),
		HasNew: req.Query.Get("has_new") == "1" ||
			req.Query.Get("has_new") == "true",
		Page:  adminBoundedNumber(req.Query.Get("page"), 1, 1, math.MaxInt32),
		Limit: adminBoundedNumber(req.Query.Get("limit"), 50, 1, 100),

		CharacterID:   optionalJavaScriptNumber(req.Query.Get("character_id")),
		CorporationID: optionalJavaScriptNumber(req.Query.Get("corporation_id")),
		AfterID:       optionalJavaScriptNumber(req.Query.Get("after_id"))}
	return filters
}

func optionalJavaScriptNumber(raw string) *float64 {
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(value) || value == 0 {
		return nil
	}
	return &value
}

func (s *adminService) esiLogsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		filters := parseAdminESILogFilters(req)
		where, args := buildAdminESILogWhere(filters)
		if filters.AfterID != nil {
			args = append(args, *filters.AfterID)
			where = appendAdminESICondition(
				where, fmt.Sprintf("log.id > $%d", len(args)),
			)
			rows, err := queryMaps(
				ctx, s.opts.DB, adminESILogPollingSQL(where), args...,
			)
			if err != nil {
				return legacyPayload{}, err
			}
			return accountNoStorePayload(map[string]any{
				"rows": nonNilAdminRows(rows), "newRows": true,
			}), nil
		}

		args = append(args, filters.Limit)
		limitPlaceholder := len(args)
		offset := (filters.Page - 1) * filters.Limit
		args = append(args, offset)
		offsetPlaceholder := len(args)
		rows, err := queryMaps(
			ctx, s.opts.DB,
			adminESILogPageSQL(where, limitPlaceholder, offsetPlaceholder),
			args...,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		total := int64(0)
		sources := []string{}
		if len(rows) > 0 {
			total = int64OrZero(rows[0]["_total"])
			sources = adminStringSlice(rows[0]["_sources"])
		}
		logs := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			if row["id"] == nil {
				continue
			}
			delete(row, "_total")
			delete(row, "_sources")
			logs = append(logs, row)
		}
		pages := int64(math.Ceil(float64(total) / float64(filters.Limit)))
		return accountNoStorePayload(map[string]any{
			"rows": logs, "total": total,
			"page": filters.Page, "limit": filters.Limit,
			"pages": pages, "sources": sources,
		}), nil
	}
}

func buildAdminESILogWhere(
	filters adminESILogFilters,
) (string, []any) {
	conditions := make([]string, 0, 8)
	args := make([]any, 0, 8)
	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(
			conditions, fmt.Sprintf(condition, len(args)),
		)
	}
	if filters.CharacterID != nil {
		add("log.character_id = $%d", *filters.CharacterID)
	}
	if filters.CorporationID != nil {
		add(`log.character_id IN (
			SELECT character_id FROM characters
			WHERE corporation_id = $%d
		)`, *filters.CorporationID)
	}
	if filters.Search != "" {
		add("log.endpoint ILIKE $%d", "%"+filters.Search+"%")
	}
	if filters.Source != "" {
		add("log.source = $%d", filters.Source)
	}
	switch filters.Status {
	case "success":
		conditions = append(conditions,
			"(log.success = TRUE OR log.status_code = 304)")
	case "error":
		conditions = append(conditions,
			"log.success = FALSE AND "+
				"(log.status_code IS NULL OR log.status_code != 304)")
	}
	switch filters.EndpointType {
	case "character":
		conditions = append(
			conditions, "log.endpoint ILIKE '%/characters/%'",
		)
	case "corporation":
		conditions = append(
			conditions, "log.endpoint ILIKE '%/corporations/%'",
		)
	}
	if filters.HasNew {
		conditions = append(conditions, "log.new_items >= 1")
	}
	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

func appendAdminESICondition(where, condition string) string {
	if where == "" {
		return "WHERE " + condition
	}
	return where + " AND " + condition
}

const adminESILogColumns = `
	log.id::int AS id,
	log.character_id,
	user_account.character_name,
	log.endpoint,
	log.method,
	log.status_code,
	log.success,
	log.error_message,
	log.items_returned,
	log.new_items,
	log.new_item_ids,
	log.source,
	log.request_duration_ms,
	log.esi_error_limit_remain,
	log.created_at`

const adminESILogParsedColumns = `
	CASE
	  WHEN endpoint ILIKE '%/characters/%' THEN 'character'
	  WHEN endpoint ILIKE '%/corporations/%' THEN 'corporation'
	  ELSE 'other'
	END AS endpoint_type,
	REGEXP_REPLACE(
	  REGEXP_REPLACE(
	    endpoint, '.*/(?:characters|corporations)/[0-9]+/', ''
	  ),
	  '[/?].*', ''
	) AS endpoint_action,
	(
	  REGEXP_MATCH(
	    endpoint, '/(?:characters|corporations)/([0-9]+)'
	  )
	)[1]::int AS endpoint_entity_id`

func adminESILogPollingSQL(where string) string {
	return `
		WITH page AS (
			SELECT ` + adminESILogColumns + `
			FROM esi_request_logs log
			LEFT JOIN users user_account
			  ON user_account.character_id = log.character_id
			` + where + `
			ORDER BY log.id DESC
			LIMIT 200
		),
		parsed AS (
			SELECT page.*, ` + adminESILogParsedColumns + `
			FROM page
		)
		SELECT parsed.*,
		       CASE parsed.endpoint_type
		         WHEN 'character' THEN character.name
		         WHEN 'corporation' THEN corporation.name
		         ELSE NULL
		       END AS endpoint_entity_name
		FROM parsed
		LEFT JOIN characters character
		  ON parsed.endpoint_type = 'character'
		 AND character.character_id = parsed.endpoint_entity_id
		LEFT JOIN corporations corporation
		  ON parsed.endpoint_type = 'corporation'
		 AND corporation.corporation_id = parsed.endpoint_entity_id
		ORDER BY parsed.id DESC`
}

func adminESILogPageSQL(
	where string,
	limitPlaceholder, offsetPlaceholder int,
) string {
	return fmt.Sprintf(`
		WITH result_count AS (
			SELECT COUNT(*)::bigint AS total
			FROM esi_request_logs log
			%s
		),
		source_options AS (
			SELECT COALESCE(
			  ARRAY_AGG(DISTINCT source ORDER BY source),
			  ARRAY[]::text[]
			) AS sources
			FROM esi_request_logs
			WHERE created_at >= NOW() - INTERVAL '7 days'
		),
		page AS (
			SELECT %s
			FROM esi_request_logs log
			LEFT JOIN users user_account
			  ON user_account.character_id = log.character_id
			%s
			ORDER BY log.id DESC
			LIMIT $%d OFFSET $%d
		),
		parsed AS (
			SELECT page.*, %s
			FROM page
		),
		named_page AS (
			SELECT parsed.*,
			       CASE parsed.endpoint_type
			         WHEN 'character' THEN character.name
			         WHEN 'corporation' THEN corporation.name
			         ELSE NULL
			       END AS endpoint_entity_name
			FROM parsed
			LEFT JOIN characters character
			  ON parsed.endpoint_type = 'character'
			 AND character.character_id = parsed.endpoint_entity_id
			LEFT JOIN corporations corporation
			  ON parsed.endpoint_type = 'corporation'
			 AND corporation.corporation_id = parsed.endpoint_entity_id
		)
		SELECT named_page.*,
		       result_count.total AS _total,
		       source_options.sources AS _sources
		FROM result_count
		CROSS JOIN source_options
		LEFT JOIN named_page ON TRUE
		ORDER BY named_page.id DESC NULLS LAST`,
		where, adminESILogColumns, where,
		limitPlaceholder, offsetPlaceholder,
		adminESILogParsedColumns,
	)
}

func adminStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		if typed == nil {
			return []string{}
		}
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, value := range typed {
			if text, ok := stringValue(value); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return []string{}
	}
}
