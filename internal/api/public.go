package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

func registerPublicAPI(a huma.API, opts Options) {
	registerPublicHealth(a, opts)
	registerHistoryRoutes(a, opts)
	registerResolveRoute(a, opts)
	registerSearchRoute(a, opts)
	registerLocationRoute(a, opts)
	registerKillmailRoutes(a, opts)
	registerKillmailSearchRoute(a, opts)
	registerEntityRoutes(a, opts)
	registerCharacterIntelRoute(a, opts)
	registerAnalyzeRoute(a, opts)
	registerBatchStatsRoutes(a, opts)
	registerCoalitionRoute(a, opts)
	registerGlobalStatsRoute(a, opts)
	registerShipRoutes(a, opts)
	registerBattleRoutes(a, opts)
	registerWarRoutes(a, opts)
	registerSDERoutes(a, opts)
	registerFeedRoutes(a, opts)
}

func registerPublicHealth(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Database health check",
		Description: "Checks that the public API can reach Postgres.",
		Tags:        []string{"health"},
	}, func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		if opts.DB == nil {
			return legacyPayload{}, context.Canceled
		}
		if err := opts.DB.Ping(ctx); err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"ok":        true,
			"timestamp": javascriptTimestamp(time.Now()),
		}), nil
	})
}
