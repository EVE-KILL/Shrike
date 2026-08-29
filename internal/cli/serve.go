package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

var (
	flagServePort     int
	flagDevPort       int
	flagDevRenderer   string
	flagDevAPIAddress string
)

// siteMode is the difference between `serve` and `dev`.
//
// Everything else about the two commands is identical, and deliberately so: a
// development stack that wires its surfaces differently from production stops
// being evidence about production.
type siteMode struct {
	// Name labels the service in logs and telemetry.
	Name string

	// DevRenderer is the host:port of an already-running `nuxt dev`. When it
	// is empty the command supervises the built Bun renderer instead, which is
	// what production does.
	DevRenderer string

	// DevAPIAddress is the loopback host:port that serves the private API to
	// that renderer. Only read when DevRenderer is set.
	DevAPIAddress string
}

// supervised reports whether this mode owns the renderer process.
func (m siteMode) supervised() bool { return m.DevRenderer == "" }

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
socket is public; browser API calls remain same-origin.

In development, Caddy serves trusted HTTPS on localhost using its internal CA.
The first run may ask permission to add that CA to the system and browser trust
stores. Production origins remain plain HTTP behind Cloudflare.

SIGINT (Ctrl+C) and SIGTERM both trigger a graceful shutdown: the listener
stops accepting, in-flight requests are given time to finish, and the
supervised renderer is drained. Kubernetes needs no special handling beyond
its default SIGTERM.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSite(cmd, siteMode{Name: "serve"}, flagServePort)
	},
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run the HTTP server against a separate `nuxt dev` renderer",
	Long: `Starts the same listener as serve, but proxies the frontend to a Nuxt
development server you run yourself.

serve supervises the built renderer and reaches it over a Unix socket. The Nuxt
development server cannot offer a socket, so dev proxies to --renderer over
loopback instead. Every other surface — /health, /api, /auth, /images, /ws — is
wired exactly as production wires it.

dev does not start or stop the renderer. That separation is the point: a Go
rebuild restarts this process only, and the development server keeps its state
and its hot module replacement connections across every restart.

Nuxt server-side rendering cannot use the production socket either: the
development server runs under Node, whose fetch has no way to dial one. dev
therefore serves the same private API on --api-addr, and the development server
reaches it through NUXT_API_ORIGIN. ` + "`make dev`" + ` wires all of this.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// The flag wins, then the environment, then the Nuxt default port.
		// air owns the process arguments, so the environment is how `make dev`
		// passes a port through without depending on how air splits a string.
		upstream := strings.TrimSpace(flagDevRenderer)
		if upstream == "" {
			upstream = strings.TrimSpace(os.Getenv("SHRIKE_DEV_RENDERER"))
		}
		if upstream == "" {
			upstream = "127.0.0.1:3000"
		}
		if _, _, err := net.SplitHostPort(upstream); err != nil {
			return fmt.Errorf("renderer address must be host:port: %w", err)
		}

		apiAddress := strings.TrimSpace(flagDevAPIAddress)
		if apiAddress == "" {
			apiAddress = strings.TrimSpace(os.Getenv("SHRIKE_DEV_API_ADDR"))
		}
		if apiAddress == "" {
			apiAddress = "127.0.0.1:4002"
		}
		return runSite(cmd, siteMode{
			Name:          "dev",
			DevRenderer:   upstream,
			DevAPIAddress: apiAddress,
		}, flagDevPort)
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

		sharedRedis := redisx.New(cfg)
		defer sharedRedis.Close() //nolint:errcheck

		wsServer := shrikewebsocket.New(
			sharedRedis,
			log.With().Str("subsystem", "websocket").Logger(),
		)
		wsServer.Start(ctx)
		defer wsServer.Close()

		feed := api.NewFeedManager(pool, sharedRedis)
		feed.Start(ctx)

		opts := api.Options{
			Version: ui.Version, Commit: ui.Commit,
			DB: pool, Graph: graphClient, Feed: feed, Cache: sharedRedis,
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
		siteHandler := withRequestDeadline(apiService.Site(), 30*time.Second)
		surfaces := map[string]http.Handler{
			ingress.SurfaceSameOrigin: siteHandler,
			ingress.SurfaceWS:         wsServer,
		}

		// dev never resolves an entrypoint. Doing so would fail the whole
		// command on a missing web/.output, which is the normal state of a
		// checkout whose frontend has only ever been run from source.
		entrypoint, superviseRenderer := "", false
		if mode.supervised() {
			entrypoint, superviseRenderer, err = renderer.ResolveEntrypoint(cfg.NuxtEntrypoint)
			if err != nil {
				return err
			}
		}

		apiSocket := cfg.APISocket
		if apiSocket == "" && superviseRenderer {
			apiSocket = processSocketPath("shrike-api")
		}

		// The private surface carries server-side rendering traffic. serve puts
		// it on a Unix socket, which is private by construction: no port, no
		// interface.
		//
		// dev cannot use one. `nuxt dev` runs under Node, and Node's fetch
		// ignores the `unix` request option that Bun implements and that
		// web/shared/utils/serverApi.ts depends on — the request resolves
		// shrike.internal through DNS and fails. dev therefore serves the same
		// handler on loopback, and Nuxt reaches it through NUXT_API_ORIGIN.
		var privateAPI *unixhttp.Server
		var privateAPIDone <-chan struct{}
		switch {
		case !mode.supervised():
			privateAPI, err = unixhttp.ListenLoopback(
				mode.DevAPIAddress, devRendererHost(siteHandler),
			)
			if err != nil {
				return fmt.Errorf("start development SSR API: %w", err)
			}
		case apiSocket != "":
			privateAPI, err = unixhttp.Listen(apiSocket, siteHandler)
			if err != nil {
				return fmt.Errorf("start private SSR API: %w", err)
			}
		}
		if privateAPI != nil {
			defer func() {
				if closeErr := privateAPI.Close(); closeErr != nil {
					log.Warn().Err(closeErr).Msg("private SSR API did not close cleanly")
				}
			}()
			privateAPIDone = privateAPI.Done()
			log.Info().Str("address", privateAPI.Addr()).Msg("private SSR API listening")
		}

		// A NUXT_SOCKET left in the environment names the production
		// renderer, which dev is not running. Ingress rejects a socket and
		// an address together rather than guessing which one is live.
		nuxtSocket := cfg.NuxtSocket
		if !mode.supervised() {
			nuxtSocket = ""
		}
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
		localHTTPS := !cfg.IsProduction()

		if err := manager.Start(ctx, ingress.Config{
			Address:     fmt.Sprintf(":%d", port),
			DataDir:     cfg.DataDir,
			LogLevel:    cfg.LogLevel,
			LocalHTTPS:  localHTTPS,
			NuxtSocket:  nuxtSocket,
			NuxtAddress: mode.DevRenderer,
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
}

// withRequestDeadline ensures a slow or abandoned HTTP request cannot hold a
// database connection indefinitely. The SSE feed is intentionally long-lived
// and manages its own cancellation through the client connection.
func withRequestDeadline(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/feed/stream" {
			next.ServeHTTP(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// devRendererHost restores the browser's host on development server-side
// rendering requests.
//
// Tenant scope comes from the request host, and the API reads it from the
// authoritative Host field rather than a forwarding header so that no client
// can claim another board by sending one. `nuxt dev` runs under Node, whose
// fetch drops a caller-supplied Host — the name is forbidden to scripts — so
// the renderer's only surviving copy is X-Forwarded-Host, and every board
// answered with the apex site. Production keeps the header intact, because
// the renderer runs under Bun.
//
// This wrapper is dev-only and stays that way. The listener it fronts is
// loopback and serves one client, the renderer; the public listener still
// ignores the header.
func devRendererHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded, _, _ := strings.Cut(r.Header.Get("X-Forwarded-Host"), ",")
		if forwarded = strings.TrimSpace(forwarded); forwarded != "" {
			r = r.Clone(r.Context())
			r.Host = forwarded
		}
		next.ServeHTTP(w, r)
	})
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
	devCmd.Flags().StringVar(
		&flagDevRenderer, "renderer", "",
		"host:port of the running `nuxt dev` server "+
			"(default $SHRIKE_DEV_RENDERER, then 127.0.0.1:3000)",
	)
	devCmd.Flags().StringVar(
		&flagDevAPIAddress, "api-addr", "",
		"loopback host:port serving the private API to `nuxt dev` "+
			"(default $SHRIKE_DEV_API_ADDR, then 127.0.0.1:4002)",
	)
}
