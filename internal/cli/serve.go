package cli

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/graph"
	"github.com/eve-kill/shrike/internal/images"
	"github.com/eve-kill/shrike/internal/ingress"
	"github.com/eve-kill/shrike/internal/objectstore"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/redisx"
	"github.com/eve-kill/shrike/internal/ui"
	shrikewebsocket "github.com/eve-kill/shrike/internal/websocket"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// serve runs the HTTP front door: an embedded Caddy listener in front of
// Shrike's surfaces, with everything it does not recognise handed to the Nuxt
// renderer over a Unix socket.
//
// The surfaces are built here rather than inside the ingress package so the
// two stay independent — ingress routes to names and knows nothing about Huma,
// api builds handlers and knows nothing about Caddy. This function is the only
// place that has to know both.

var flagServePort int

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
  everything else                        the Nuxt renderer, over NUXT_SOCKET

Every Go-owned surface shares the frontend origin, including tenant domains.
With NUXT_SOCKET unset, unmatched requests get a 404 rather than being proxied
to a renderer that is not running.

SIGINT (Ctrl+C) and SIGTERM both trigger a graceful shutdown: the listener
stops accepting, in-flight requests are given time to finish, then the process
exits. Kubernetes needs no special handling beyond its default SIGTERM.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		port := cfg.Port
		if cmd.Flags().Changed("port") {
			port = flagServePort
		}

		return RunService(cmd, "serve", func(ctx context.Context) error {
			domainAssets, err := newDomainAssetStorage(cfg)
			if err != nil {
				return err
			}
			imageStorage, err := newImageStorage(cfg)
			if err != nil {
				return err
			}
			pool, err := db.New(ctx, cfg)
			if err != nil {
				return fmt.Errorf("connect API database: %w", err)
			}
			defer pool.Close()

			imageQueue, err := queue.New(queue.Options{Pool: pool})
			if err != nil {
				return fmt.Errorf("configure image refresh queue: %w", err)
			}
			imageService := images.New(images.Options{
				Store: imageStorage, UserAgent: cfg.ESIUserAgent,
				CacheBytes: cfg.ImageCacheBytes,
				Refresh:    imageRefreshDispatcher{Queue: imageQueue},
				Social:     images.PostgresSocialLoader{DB: pool},
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

			queueRedis := redisx.Coordination(cfg)
			defer queueRedis.Close() //nolint:errcheck
			cacheRedis := redisx.Cache(cfg)
			defer cacheRedis.Close() //nolint:errcheck

			wsServer := shrikewebsocket.New(
				queueRedis,
				log.With().Str("subsystem", "websocket").Logger(),
			)
			wsServer.Start(ctx)
			defer wsServer.Close()

			feed := api.NewFeedManager(pool, queueRedis)
			feed.Start(ctx)

			opts := api.Options{
				Version: ui.Version, Commit: ui.Commit,
				DB: pool, Graph: graphClient, Feed: feed, Cache: cacheRedis,
				DomainAssets: domainAssets,
				Images:       imageService,
				Auth: api.AuthOptions{
					ClientID: cfg.EVEClientID, ClientSecret: cfg.EVEClientSecret,
					StateSecret: cfg.EVEClientSecret, CallbackURL: cfg.EVECallbackURL,
					UserAgent: cfg.ESIUserAgent, Production: cfg.IsProduction(),
				},
			}
			apiService := api.New(opts)
			surfaces := map[string]http.Handler{
				ingress.SurfaceSameOrigin: apiService.Site(),
				ingress.SurfaceWS:         wsServer,
			}
			manager := ingress.New(surfaces, log.With().Str("subsystem", "ingress").Logger())

			if err := manager.Start(ctx, ingress.Config{
				Address:    fmt.Sprintf(":%d", port),
				DataDir:    cfg.DataDir,
				LogLevel:   cfg.LogLevel,
				NuxtSocket: cfg.NuxtSocket,
			}); err != nil {
				return fmt.Errorf("starting embedded Caddy ingress: %w", err)
			}
			defer func() { _ = manager.Close() }()

			for _, r := range manager.Status().Routes {
				log.Info().Str("match", r.Match).Str("surface", r.Surface).Msg("route")
			}
			log.Info().Int("port", port).Msg("http listening")

			// Caddy owns its own accept loops and shutdown, so this only has to
			// stay alive until RunService cancels and then let the deferred
			// Close drain. RunService bounds how long that is allowed to take.
			<-ctx.Done()
			return nil
		})
	},
}

type imageRefreshDispatcher struct {
	Queue *queue.Client
}

func (d imageRefreshDispatcher) EnqueueImageRefresh(
	ctx context.Context,
	kind images.EntityKind,
	id int64,
) error {
	_, err := queue.Dispatch(ctx, d.Queue, queue.ImageRefreshArgs{
		EntityKind: string(kind),
		EntityID:   id,
	}, queue.Live)
	return err
}

func newImageStorage(cfg *config.Config) (*objectstore.S3Store, error) {
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
}
