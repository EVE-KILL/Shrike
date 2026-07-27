package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/images"
)

// registerRoutes installs the whole API into one registry. The schema-linked
// routes are registered first only because their established response contract
// includes a generated $schema field; it is not an access boundary.
func registerRoutes(a huma.API, opts Options) *responseSchemaResolver {
	registerSchemaLinkedRoutes(a, opts)

	// Snapshot before the remaining routes so adding a new operation cannot
	// accidentally opt it into that established response-shape contract.
	schemas := newResponseSchemaResolver(a.OpenAPI())

	registerStatsRankingsRoute(a, opts)
	registerLegacyArchiveRoutes(a, opts)
	registerMarketRoutes(a, opts)
	registerMatchupRoute(a, opts)
	registerKillmailDetailRoutes(a, opts)
	registerKillmailSubmissionRoute(a, opts)
	registerKillboardRoutes(a, opts)
	registerMapRoutes(a, opts)
	registerBackgroundRoutes(a, opts)
	registerFittingRoutes(a, opts)
	registerScanRoutes(a, opts)
	registerUniverseRoutes(a, opts)
	registerEntityPageRoutes(a, opts)
	registerEntityTopRoutes(a, opts)
	registerFactionWarDashboardRoutes(a, opts)
	registerConflictRoutes(a, opts)
	registerCampaignRoutes(a, opts)
	registerWalletRoutes(a, opts)
	registerBlogRoutes(a, opts)
	registerAnnouncementAdminRoutes(a, opts)
	registerCommentRoutes(a, opts)
	registerModerationRoutes(a, opts)
	registerSiteConfigurationRoute(a, opts)
	registerDomainRoutes(a, opts)
	registerDomainKillboardRoutes(a, opts)
	registerSitemapRoutes(a, opts)
	registerAuthRoutes(a, opts)
	registerAccountRoutes(a, opts)
	registerAdminRoutes(a, opts)
	images.Register(a, opts.Images)

	return schemas
}

func registerSchemaLinkedRoutes(a huma.API, opts Options) {
	registerHealthRoute(a, opts)
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

func registerHealthRoute(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Database health check",
		Description: "Checks that the API can reach Postgres.",
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
