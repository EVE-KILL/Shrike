package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
	"github.com/eve-kill/shrike/internal/renderer"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/eve-kill/shrike/internal/unixhttp"
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
  everything else                        the supervised Nuxt renderer

Every Go-owned surface shares the frontend origin, including tenant domains.
In a source checkout, serve finds web/.output/server/index.mjs automatically.
NUXT_ENTRYPOINT can select an explicit build. Caddy talks to Nuxt over
NUXT_SOCKET, while Nuxt SSR talks back to Go over SHRIKE_API_SOCKET. Neither
socket is public; browser API calls remain same-origin HTTP.

SIGINT (Ctrl+C) and SIGTERM both trigger a graceful shutdown: the listener
stops accepting, in-flight requests are given time to finish, and the
supervised renderer is drained. Kubernetes needs no special handling beyond
its default SIGTERM.`,
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
				RequestGuard: api.NewRequestGuard(),
				OpenAIAPIKey: cfg.OpenAIAPIKey,
				KlipyAPIKey:  cfg.KlipyAPIKey,
				Auth: api.AuthOptions{
					ClientID: cfg.EVEClientID, ClientSecret: cfg.EVEClientSecret,
					StateSecret: cfg.EVEClientSecret, CallbackURL: cfg.EVECallbackURL,
					UserAgent: cfg.ESIUserAgent, Production: cfg.IsProduction(),
				},
			}
			apiService := api.New(opts)
			siteHandler := apiService.Site()
			surfaces := map[string]http.Handler{
				ingress.SurfaceSameOrigin: siteHandler,
				ingress.SurfaceWS:         wsServer,
			}

			entrypoint, superviseRenderer, err := renderer.ResolveEntrypoint(cfg.NuxtEntrypoint)
			if err != nil {
				return err
			}

			apiSocket := cfg.APISocket
			if superviseRenderer && apiSocket == "" {
				apiSocket = processSocketPath("shrike-api")
			}
			var privateAPI *unixhttp.Server
			var privateAPIDone <-chan struct{}
			if apiSocket != "" {
				privateAPI, err = unixhttp.Listen(apiSocket, siteHandler)
				if err != nil {
					return fmt.Errorf("start private SSR API: %w", err)
				}
				defer func() {
					if closeErr := privateAPI.Close(); closeErr != nil {
						log.Warn().Err(closeErr).Msg("private SSR API did not close cleanly")
					}
				}()
				privateAPIDone = privateAPI.Done()
				log.Info().Str("socket", apiSocket).Msg("private SSR API listening")
			}

			nuxtSocket := cfg.NuxtSocket
			var nuxt *renderer.Process
			var nuxtDone <-chan struct{}
			if superviseRenderer {
				if apiSocket == "" {
					return fmt.Errorf("supervised Nuxt renderer requires a private API socket")
				}
				if nuxtSocket == "" {
					nuxtSocket = processSocketPath("shrike-nuxt")
				}
				nuxt, err = renderer.Start(renderer.Options{
					Entrypoint: entrypoint,
					Socket:     nuxtSocket,
					APISocket:  apiSocket,
				})
				if err != nil {
					return err
				}
				defer func() {
					if closeErr := nuxt.Close(); closeErr != nil {
						log.Warn().Err(closeErr).Msg("Nuxt renderer did not close cleanly")
					}
				}()
				nuxtDone = nuxt.Done()
				log.Info().
					Str("entrypoint", entrypoint).
					Str("socket", nuxtSocket).
					Msg("Nuxt renderer ready")
			}

			manager := ingress.New(surfaces, log.With().Str("subsystem", "ingress").Logger())

			if err := manager.Start(ctx, ingress.Config{
				Address:    fmt.Sprintf(":%d", port),
				DataDir:    cfg.DataDir,
				LogLevel:   cfg.LogLevel,
				NuxtSocket: nuxtSocket,
			}); err != nil {
				return fmt.Errorf("starting embedded Caddy ingress: %w", err)
			}
			defer func() { _ = manager.Close() }()

			for _, r := range manager.Status().Routes {
				log.Info().Str("match", r.Match).Str("surface", r.Surface).Msg("route")
			}
			log.Info().Int("port", port).Msg("http listening")

			// All three accept loops are one service. Losing either private
			// dependency is fatal, so Kubernetes can restart the whole unit
			// instead of leaving Caddy alive and returning 502s.
			select {
			case <-ctx.Done():
				return nil
			case <-nuxtDone:
				if exitErr := nuxt.Err(); exitErr != nil {
					return fmt.Errorf("Nuxt renderer stopped: %w", exitErr)
				}
				return fmt.Errorf("Nuxt renderer stopped unexpectedly")
			case <-privateAPIDone:
				if serveErr := privateAPI.Err(); serveErr != nil {
					return fmt.Errorf("private SSR API stopped: %w", serveErr)
				}
				return fmt.Errorf("private SSR API stopped unexpectedly")
			}
		})
	},
}

func processSocketPath(name string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d.sock", name, os.Getpid()))
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

func newImageStorage(cfg *config.Config) (images.ObjectStore, error) {
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
