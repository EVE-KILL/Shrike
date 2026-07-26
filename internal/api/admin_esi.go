package api

import (
	"context"
	"strings"
	"unicode/utf8"
)

func (s *adminService) esiOverviewHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		results, err := queryMapsConcurrent(ctx, s.opts.DB,
			databaseQuery{SQL: `
				SELECT DATE_TRUNC('hour', created_at)::text AS hour,
				       COUNT(*)::bigint AS total,
				       COUNT(*) FILTER (
				         WHERE success = FALSE
				           AND (status_code IS NULL OR status_code != 304)
				       )::bigint AS errors,
				       COALESCE(SUM(new_items), 0)::bigint AS new_items
				FROM esi_request_logs
				WHERE created_at >= NOW() - INTERVAL '24 hours'
				GROUP BY DATE_TRUNC('hour', created_at)
				ORDER BY DATE_TRUNC('hour', created_at)`},
			databaseQuery{SQL: `
				SELECT COUNT(*)::bigint AS request_count,
				       ROUND(AVG(request_duration_ms))::int AS avg_ms,
				       ROUND(
				         PERCENTILE_CONT(0.95) WITHIN GROUP (
				           ORDER BY request_duration_ms
				         )
				       )::int AS p95_ms
				FROM esi_request_logs
				WHERE created_at >= NOW() - INTERVAL '1 hour'`},
		)
		if err != nil {
			return legacyPayload{}, err
		}
		summary := map[string]any{
			"request_count": int64(0), "avg_ms": nil, "p95_ms": nil,
		}
		if len(results[1]) > 0 {
			summary = results[1][0]
		}
		return accountNoStorePayload(map[string]any{
			"volumeByHour": nonNilAdminRows(results[0]),
			"rateLimit": map[string]any{
				"request_count": int64OrZero(summary["request_count"]),
			},
			"responseTime": map[string]any{
				"avg_ms": summary["avg_ms"], "p95_ms": summary["p95_ms"],
			},
		}), nil
	}
}

func (s *adminService) esiEntitiesHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		query := strings.TrimSpace(req.Query.Get("q"))
		if utf8.RuneCountInString(query) < 2 {
			return accountNoStorePayload(map[string]any{
				"results": []map[string]any{},
			}), nil
		}
		rows, err := queryMaps(ctx, s.opts.DB, `
			SELECT results.id, results.name, results.type, results._entity_order
			FROM (
			  (
			    SELECT DISTINCT users.character_id AS id,
			           users.character_name AS name,
			           'character'::text AS type,
			           0 AS _entity_order
			    FROM users
			    INNER JOIN esi_request_logs log
			      ON log.character_id = users.character_id
			    WHERE users.character_name ILIKE $1 || '%'
			    LIMIT 10
			  )
			  UNION ALL
			  (
			    SELECT DISTINCT corporation.corporation_id AS id,
			           corporation.name,
			           'corporation'::text AS type,
			           1 AS _entity_order
			    FROM characters character
			    INNER JOIN corporations corporation
			      ON corporation.corporation_id = character.corporation_id
			    INNER JOIN esi_request_logs log
			      ON log.character_id = character.character_id
			    WHERE corporation.name ILIKE $1 || '%'
			    LIMIT 10
			  )
			) results
			ORDER BY results._entity_order`, query)
		if err != nil {
			return legacyPayload{}, err
		}
		for _, row := range rows {
			delete(row, "_entity_order")
		}
		return accountNoStorePayload(map[string]any{
			"results": nonNilAdminRows(rows),
		}), nil
	}
}
