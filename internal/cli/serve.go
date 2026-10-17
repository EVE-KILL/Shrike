package cli

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/frontend"
	"github.com/eve-kill/shrike/internal/graph"
	"github.com/eve-kill/shrike/internal/images"
	"github.com/eve-kill/shrike/internal/ingress"
	"github.com/eve-kill/shrike/internal/objectstore"
	"github.com/eve-kill/shrike/internal/redisx"
	"github.com/eve-kill/shrike/internal/requeststats"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/eve-kill/shrike/internal/ui"
	shrikewebsocket "github.com/eve-kill/shrike/internal/websocket"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// serve and dev route all HTTP surfaces directly to Go handlers.
var (
	flagServePort int
	flagDevPort   int
)

type siteMode struct{ Name string }

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the HTTP server",
	Long: `Starts the HTTP listener in the foreground and runs until interrupted.

Requests are routed by hostname and path to one of Shrike's surfaces:

  /health                                liveness, on any hostname
  /api, /api/*                           public and signed-in API
  /auth, /auth/*                         browser authentication
  /images, /images/*                     the image server
  /ws, /ws/*                             live event streams
  everything else                        the Go frontend

All surfaces run in process. No frontend runtime or private SSR listener is
started. In development, serve uses local HTTPS; dev uses plain HTTP.
SIGINT and SIGTERM gracefully stop the listener.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSite(cmd, siteMode{Name: "serve"}, flagServePort)
	},
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run the Go site over HTTP for local development",
	Long:  "Runs the same Go handlers as serve over plain HTTP. Use make dev for automatic rebuilds.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSite(cmd, siteMode{Name: "dev"}, flagDevPort)
	},
}

func runSite(cmd *cobra.Command, mode siteMode, portFlag int) error {
	if err := requireConfig(); err != nil {
		return err
	}

	port := cfg.Port
	if cmd.Flags().Changed("port") {
		port = portFlag
	}

	return RunService(cmd, mode.Name, func(ctx context.Context) error {
		domainAssets, err := newDomainAssetStorage(cfg)
		if err != nil {
			return err
		}
		imageStorage, err := newImageStorage(cfg)
		if err != nil {
			return err
		}
		primaryPool, err := db.New(ctx, cfg)
		if err != nil {
			return fmt.Errorf("connect API primary database: %w", err)
		}
		defer primaryPool.Close()

		readPool := primaryPool
		if cfg.DatabaseReadURL != cfg.DatabaseURL {
			readPool, err = db.NewRead(ctx, cfg)
			if err != nil {
				return fmt.Errorf("connect API read database: %w", err)
			}
			defer readPool.Close()
		}

		imageService := images.New(images.Options{
			Store: imageStorage, UserAgent: cfg.ESIUserAgent,
			CacheBytes: cfg.ImageCacheBytes,
			Social:     images.PostgresSocialLoader{DB: readPool},
		})

		var graphClient *graph.Client
		graphCtx, cancelGraph := context.WithTimeout(ctx, 3*time.Second)
		graphClient, graphErr := graph.Connect(graphCtx, cfg.MemgraphURL)
		cancelGraph()
		if graphErr != nil {
			log.Warn().Err(graphErr).Msg(
				"memgraph unavailable; character graph intelligence disabled",
			)
		} else {
			defer func() { _ = graphClient.Close(context.Background()) }()
		}

		sharedRedis := redisx.New(cfg)
		defer sharedRedis.Close() //nolint:errcheck

		wsServer := shrikewebsocket.New(
			sharedRedis,
			log.With().Str("subsystem", "websocket").Logger(),
		)
		wsServer.Start(ctx)
		defer wsServer.Close()

		// A Redis notification can beat physical replication, so live feed
		// hydration deliberately reads from the primary.
		feed := api.NewFeedManager(primaryPool, sharedRedis)
		feed.Start(ctx)

		opts := api.Options{
			Version: ui.Version, Commit: ui.Commit,
			DB: readPool, Primary: primaryPool, PrimaryPool: primaryPool,
			Graph: graphClient, Feed: feed, Cache: sharedRedis,
			ResponseCacheBytes: cfg.APICacheBytes,
			DomainAssets:       domainAssets,
			Images:             imageService,
			RequestGuard:       api.NewRequestGuard(),
			OpenAIAPIKey:       cfg.OpenAIAPIKey,
			KlipyAPIKey:        cfg.KlipyAPIKey,
			Auth: api.AuthOptions{
				ClientID: cfg.EVEClientID, ClientSecret: cfg.EVEClientSecret,
				StateSecret: cfg.EVEClientSecret, CallbackURL: cfg.EVECallbackURL,
				UserAgent: cfg.ESIUserAgent, Production: cfg.IsProduction(),
			},
		}
		apiService := api.New(opts)
		defer apiService.Close()
		// Published posts should be visible on the Go page as soon as the
		// existing blog API (which also reads primary) can return them.
		blogReader := api.NewHomepageReader(primaryPool)
		siteHandler := withRequestDeadline(apiService.Site(), 30*time.Second)
		if !cfg.IsProduction() {
			siteHandler = requeststats.Middleware(siteHandler)
		}
		surfaces := map[string]http.Handler{
			ingress.SurfaceSameOrigin: siteHandler,
			ingress.SurfaceWS:         wsServer,
			ingress.SurfaceFrontend:   withRequestDeadline(frontend.New(frontend.Options{Production: cfg.IsProduction(), Homepage: apiService.Homepage(), Entities: apiService.Homepage(), Universe: apiService.Homepage(), Wallet: apiService.Homepage(), Legacy: apiService.Homepage(), Rankings: apiService.Homepage(), Statistics: apiService.Homepage(), Scans: apiService.Homepage(), Coalitions: apiService.Homepage(), Wars: apiService.Homepage(), FactionWars: apiService.Homepage(), BattleGenerator: apiService.Homepage(), Killmail: apiService.Homepage(), Blog: blogReader, BlogPreviewAPI: apiService.Site(), CommentsAPI: apiService.Site(), SitemapAPI: apiService.Site(), CampaignsAPI: apiService.Site(), Sessions: apiService.Session(), AccountAPI: apiService.Site(), ConflictsAPI: apiService.Site(), FittingsAPI: apiService.Site(), RoamReports: blogReader, AdvancedSearch: apiService.Homepage()}), 30*time.Second),
		}

		for name, handler := range surfaces {
			surfaces[name] = telemetry.HTTP(name, handler)
		}
		manager := ingress.New(surfaces, log.With().Str("subsystem", "ingress").Logger())
		localHTTPS := mode.Name != "dev" && !cfg.IsProduction()

		if err := manager.Start(ctx, ingress.Config{
			Address:      fmt.Sprintf(":%d", port),
			LegacyDomain: cfg.LegacyDomain,
			DataDir:      cfg.DataDir,
			LogLevel:     cfg.LogLevel,
			LocalHTTPS:   localHTTPS,
		}); err != nil {
			return fmt.Errorf("starting embedded Caddy ingress: %w", err)
		}
		defer func() { _ = manager.Close() }()

		for _, r := range manager.Status().Routes {
			log.Info().Str("match", r.Match).Str("surface", r.Surface).Msg("route")
		}
		scheme := "http"
		if localHTTPS {
			scheme = "https"
		}
		log.Info().
			Str("url", fmt.Sprintf("%s://localhost:%d", scheme, port)).
			Msg("site listening")

		<-ctx.Done()
		return nil
	})
}

// withRequestDeadline ensures a slow or abandoned HTTP request cannot hold a
// database connection indefinitely. The SSE feed is intentionally long-lived
// and manages its own cancellation through the client connection.
func withRequestDeadline(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/feed/stream" || r.URL.Path == "/__dev/reload" {
			next.ServeHTTP(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newImageStorage(cfg *config.Config) (images.ObjectStore, error) {
	if cfg.ImageStoragePath != "" {
		store, err := objectstore.NewFileStore(cfg.ImageStoragePath, 32<<20)
		if err != nil {
			return nil, fmt.Errorf("configure filesystem image storage: %w", err)
		}
		return store, nil
	}
	if cfg.B2ImagesPartiallyConfigured() {
		return nil, fmt.Errorf(
			"configure B2_ENDPOINT, B2_IMAGES_BUCKET, B2_KEY_ID, and B2_APP_KEY together",
		)
	}
	if !cfg.B2ImagesConfigured() {
		return nil, nil
	}
	store, err := objectstore.NewS3Store(objectstore.S3Options{
		Endpoint:        cfg.B2Endpoint,
		Bucket:          cfg.B2ImagesBucket,
		Region:          objectstore.BackblazeRegion,
		AccessKeyID:     cfg.B2KeyID,
		SecretAccessKey: cfg.B2AppKey,
		MaximumBytes:    32 << 20,
		DisableCache:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("configure image object storage: %w", err)
	}
	return store, nil
}

func newDomainAssetStorage(cfg *config.Config) (api.DomainAssetStorage, error) {
	if cfg.B2PartiallyConfigured() {
		return nil, fmt.Errorf(
			"configure all of B2_ENDPOINT, B2_MEDIA_BUCKET, B2_KEY_ID, and B2_APP_KEY",
		)
	}
	if !cfg.B2Configured() {
		return nil, nil
	}

	store, err := objectstore.NewS3Store(objectstore.S3Options{
		Endpoint:        cfg.B2Endpoint,
		Bucket:          cfg.B2MediaBucket,
		Region:          objectstore.BackblazeRegion,
		AccessKeyID:     cfg.B2KeyID,
		SecretAccessKey: cfg.B2AppKey,
		MaximumBytes:    8 << 20,
	})
	if err != nil {
		return nil, fmt.Errorf("configure custom-domain object storage: %w", err)
	}
	return store, nil
}

func init() {
	serveCmd.Flags().IntVar(&flagServePort, "port", 0, "Port to listen on (overrides PORT)")
	devCmd.Flags().IntVar(&flagDevPort, "port", 0, "Port to listen on (overrides PORT)")
}
