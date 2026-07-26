package api

import (
	"context"
	"math"
)

func (s *adminService) overviewHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		// One snapshot replaces eleven independent round-trips while retaining
		// the exact counters and killmail reltuples estimate used by Nuxt.
		row, err := queryMap(ctx, s.opts.DB, `
			SELECT
			  (SELECT COUNT(*)::bigint FROM users) AS user_total,
			  (SELECT COUNT(*)::bigint FROM users
			   WHERE created_at >= NOW() - INTERVAL '7 days') AS user_recent,
			  COALESCE((
			    SELECT reltuples::bigint FROM pg_class
			    WHERE relname = 'killmails' LIMIT 1
			  ), 0) AS killmail_total,
			  (SELECT COUNT(*)::bigint FROM killmails
			   WHERE killmail_time >= NOW() - INTERVAL '24 hours') AS kills_24h,
			  (SELECT COUNT(*)::bigint FROM killmails
			   WHERE killmail_time >= NOW() - INTERVAL '7 days') AS kills_7d,
			  (SELECT COUNT(*)::bigint FROM esi_request_logs
			   WHERE created_at >= NOW() - INTERVAL '24 hours'
			     AND success = FALSE
			     AND (status_code IS NULL OR status_code != 304)) AS esi_errors,
			  (SELECT COUNT(*)::bigint FROM esi_request_logs
			   WHERE created_at >= NOW() - INTERVAL '24 hours') AS esi_total,
			  (SELECT COUNT(*)::bigint FROM moderation_queue
			   WHERE status = 0) AS pending_moderation,
			  (SELECT COUNT(*)::bigint FROM domain_assets
			   WHERE status = 'pending') AS pending_assets,
			  (SELECT COUNT(*)::bigint FROM campaigns
			   WHERE processing_paused = TRUE) AS paused_campaigns,
			  (SELECT COUNT(*)::bigint FROM campaigns
			   WHERE last_processing_error IS NOT NULL
			     AND processing_paused = FALSE) AS failed_campaigns`)
		if err != nil {
			return legacyPayload{}, err
		}
		totalESI := int64OrZero(row["esi_total"])
		errorsESI := int64OrZero(row["esi_errors"])
		errorRate := float64(0)
		if totalESI > 0 {
			errorRate = math.Round(
				float64(errorsESI)/float64(totalESI)*10000,
			) / 100
		}
		return accountNoStorePayload(map[string]any{
			"users": map[string]any{
				"total":    int64OrZero(row["user_total"]),
				"recent7d": int64OrZero(row["user_recent"]),
			},
			"killmails": map[string]any{
				"total":   int64OrZero(row["killmail_total"]),
				"last24h": int64OrZero(row["kills_24h"]),
				"last7d":  int64OrZero(row["kills_7d"]),
			},
			"esi": map[string]any{
				"requests24h": totalESI,
				"errors24h":   errorsESI,
				"errorRate":   errorRate,
			},
			"attention": map[string]any{
				"moderation":      int64OrZero(row["pending_moderation"]),
				"domainAssets":    int64OrZero(row["pending_assets"]),
				"campaignsPaused": int64OrZero(row["paused_campaigns"]),
				"campaignsFailed": int64OrZero(row["failed_campaigns"]),
			},
		}), nil
	}
}
