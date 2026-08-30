// Package api builds Shrike's HTTP API.
//
// The API, authentication, and image routes share one Huma registry and one
// OpenAPI document on the main eve-kill.com origin. API operations live below
// /api, images below /images, and browser OAuth below /auth.
package api

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/eve-kill/shrike/internal/images"
	"github.com/eve-kill/shrike/internal/mcpserver"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Options carries what the surfaces need from the process.
//
// Deliberately not the whole *config.Config: a surface that wants a database
// pool should be handed a pool, so the dependency shows up in the signature
// rather than being fished out of a bag at request time.
type Options struct {
	Version string
	Commit  string
	DB      Database
	// Primary is used for mutations and reads that require read-after-write
	// consistency. PrimaryPool is the same connection exposed concretely for
	// River, whose client requires *pgxpool.Pool.
	Primary     MutationDatabase
	PrimaryPool *pgxpool.Pool
	Graph       GraphDatabase
	Feed        *FeedManager
	Cache       *redis.Client
	// ResponseCacheBytes bounds the process-local L1. Cache remains the shared
	// Valkey client used as L2 and by non-response coordination features.
	ResponseCacheBytes int64
	responseCache      *ResponseCache

	Auth         AuthOptions
	DomainAssets DomainAssetStorage
	Images       *images.Service
	RequestGuard *RequestGuard
	OpenAIAPIKey string
	KlipyAPIKey  string
}

type GraphDatabase interface {
	Read(context.Context, string, map[string]any) ([]map[string]any, error)
}

// Service is one API registry exposed through the main-site transport.
type Service struct {
	site     http.Handler
	document *huma.OpenAPI
}

type sameOriginPrefixContextKey struct{}

const apiIntroduction = `EVE-KILL turns EVE Online combat data into a queryable, near-real-time history of New Eden.
Use it to explore killmails, characters, corporations, alliances, wars, battles, campaigns,
fittings, statistics, market data, and the static universe behind them. Live consumers can
follow the feed instead of repeatedly polling list endpoints.

## One API, three namespaces

- **/api** contains the data API used by the website and third-party applications.
- **/auth** contains browser-based EVE SSO flows.
- **/images** serves cached character, corporation, alliance, type, map, and social images.

Most read operations are public. Operations that act on an account, character, corporation,
custom domain, or administrative resource declare their session requirement individually.
In Scalar, the lock icon and each operation's **Security** section are authoritative.

## Contract and data freshness

This is a single, continuously evolving API: there are no versioned paths, compatibility
dates, or separate public/private catalogues. We make a best effort to keep existing clients
working, prefer additive changes, and document authentication per operation. Corrections to
incorrect calculations or malformed data may intentionally change output.

Killmails and derived statistics are eventually consistent. A newly ingested event can appear
before every leaderboard, campaign, graph, or aggregate has caught up, and historical values
can be corrected when better source data becomes available.

## Responsible use

Every /api request must send an identifying **User-Agent**; use an application name and a way
to contact its operator. The origin allows 600 API requests per client IP in each one-minute
window and reports the current budget through **RateLimit-Limit**,
**RateLimit-Remaining**, and **RateLimit-Reset**. Reuse responses according to their cache
headers, prefer the live feed for continuous consumption, and back off when a response includes
**Retry-After** or returns **429 Too Many Requests**.`

// New builds the shared API once.
func New(opts Options) *Service {
	mux := chi.NewRouter()
	opts.responseCache = NewResponseCache(opts.Cache, opts.ResponseCacheBytes)

	cfg := huma.DefaultConfig("EVE-KILL API", opts.Version)
	cfg.DocsRenderer = huma.DocsRendererScalar
	cfg.Info.Description = apiIntroduction
	cfg.Info.Contact = &huma.Contact{
		Name: "EVE-KILL on GitHub",
		URL:  "https://github.com/eve-kill",
	}

	a := humachi.New(mux, cfg)
	a.OpenAPI().Servers = []*huma.Server{
		{URL: "/api", Description: "EVE-KILL API"},
	}
	schemas := registerRoutes(a, opts)
	setRootNamespaceServers(a.OpenAPI())
	applyOperationTags(a.OpenAPI())
	applyTagMetadata(a.OpenAPI())
	applyOperationDescriptions(a.OpenAPI())
	applyOperationParameters(a.OpenAPI())
	registerLegacyMethodGuards(mux)
	mcpTransport, err := mcpserver.Handler(mcpserver.Dependencies{
		DB: opts.DB, Graph: opts.Graph, BaseURL: "https://eve-kill.com",
	}, opts.Version, nil)
	if err != nil {
		panic("register MCP transport: " + err.Error())
	}
	mux.Handle("/mcp", mcpTransport)
	mux.HandleFunc("/", legacyFallback)
	mux.NotFound(legacyFallback)

	cached := responseCache(opts.responseCache, schemas, opts.Commit, mux)
	site := sitePaths(cached)
	if opts.RequestGuard != nil {
		site = opts.RequestGuard.Wrap(site)
	}
	return &Service{site: site, document: a.OpenAPI()}
}

func setRootNamespaceServers(document *huma.OpenAPI) {
	for path, item := range document.Paths {
		if path != "/auth" && !strings.HasPrefix(path, "/auth/") {
			continue
		}
		for _, operation := range []*huma.Operation{
			item.Get,
			item.Put,
			item.Post,
			item.Delete,
			item.Options,
			item.Head,
			item.Patch,
			item.Trace,
		} {
			if operation != nil {
				operation.Servers = []*huma.Server{{
					URL:         "/",
					Description: "EVE-KILL authentication",
				}}
			}
		}
	}
}

// Site serves /api, /auth, /images, /health, and /ready on the frontend origin.
func (s *Service) Site() http.Handler {
	return s.site
}

// OpenAPI returns the generated contract owned by this service.
//
// The document is complete immediately after New returns: route registration
// never invokes a handler or touches its database dependencies, which lets the
// CLI emit the frontend contract without starting the server.
func (s *Service) OpenAPI() *huma.OpenAPI {
	return s.document
}

// Site constructs the main-origin handler.
func Site(opts Options) http.Handler {
	return New(opts).Site()
}

func sitePaths(next http.Handler) http.Handler {
	prefixed := apiPrefix(next)
	publicAPI := apiPrefix(crossOriginAPI(next))
	publicImages := crossOriginAPI(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		clone := r.Clone(context.WithValue(
			r.Context(),
			sameOriginPrefixContextKey{},
			"/api",
		))
		next.ServeHTTP(w, clone)
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api", strings.HasPrefix(r.URL.Path, "/api/"):
			inner := strings.TrimPrefix(r.URL.Path, "/api")
			if inner == "/images" || strings.HasPrefix(inner, "/images/") ||
				inner == "/auth" || strings.HasPrefix(inner, "/auth/") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"Not found"}`))
				return
			}
			publicAPI.ServeHTTP(w, r)
		case r.URL.Path == "/images", strings.HasPrefix(r.URL.Path, "/images/"):
			publicImages.ServeHTTP(w, r)
		case r.URL.Path == "/auth", strings.HasPrefix(r.URL.Path, "/auth/"),
			r.URL.Path == "/health", r.URL.Path == "/ready":
			prefixed.ServeHTTP(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"Not found"}`))
		}
	})
}

func apiPrefix(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/api":
			path = "/"
		case strings.HasPrefix(path, "/api/"):
			path = strings.TrimPrefix(path, "/api")
		}

		clone := r.Clone(context.WithValue(
			r.Context(), sameOriginPrefixContextKey{}, "/api",
		))
		requestURL := new(url.URL)
		*requestURL = *r.URL
		requestURL.Path = path
		if requestURL.RawPath != "" {
			requestURL.RawPath = strings.TrimPrefix(requestURL.RawPath, "/api")
			if requestURL.RawPath == "" {
				requestURL.RawPath = "/"
			}
		}
		clone.URL = requestURL
		if path == "/docs" {
			recorder := newCacheRecorder()
			next.ServeHTTP(recorder, clone)
			body := bytes.ReplaceAll(
				recorder.body.Bytes(),
				[]byte(`data-url="/openapi.json"`),
				[]byte(`data-url="/api/openapi.json"`),
			)
			copyResponseHeaders(w.Header(), recorder.header)
			w.Header().Del("Content-Length")
			w.WriteHeader(recorder.status)
			_, _ = w.Write(body)
			return
		}
		next.ServeHTTP(w, clone)
	})
}

func registerLegacyMethodGuards(mux chi.Router) {
	for path, message := range map[string]string{
		"/characters/analyze": "Method not allowed",
		"/characters/stats":   "POST only",
		"/corporations/stats": "POST only",
		"/alliances/stats":    "POST only",
		"/coalitions/stats":   "POST only",
		"/killmails/search":   "POST only",
	} {
		mux.MethodFunc(http.MethodGet, path, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			body, _ := json.Marshal(map[string]string{"error": message})
			_, _ = w.Write(body)
		})
	}
}

//go:embed api_index.json
var apiIndexJSON []byte

func legacyFallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Not found"}`))
		return
	}
	var body map[string]any
	if err := json.Unmarshal(apiIndexJSON, &body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"Internal server error"}`))
		return
	}
	protocol := firstForwarded(r.Header.Get("X-Forwarded-Proto"))
	if protocol == "" {
		if r.TLS != nil {
			protocol = "https"
		} else {
			protocol = "http"
		}
	}
	host := firstForwarded(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	prefix, _ := r.Context().Value(sameOriginPrefixContextKey{}).(string)
	body["baseUrl"] = protocol + "://" + host + strings.TrimSuffix(prefix, "/")
	encoded, err := json.Marshal(body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"Internal server error"}`))
		return
	}
	_, _ = w.Write(encoded)
}

func firstForwarded(value string) string {
	if before, _, found := strings.Cut(value, ","); found {
		value = before
	}
	return strings.TrimSpace(value)
}

// WS is a lightweight placeholder used by ingress-focused tests. Production
// supplies the real same-origin /ws handler from internal/websocket.
func WS(Options) http.Handler {
	return notImplemented("websocket")
}

func notImplemented(surface string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte(`{"title":"Not Implemented","status":501,` +
			`"detail":"the ` + surface + ` surface is routed but not yet ported"}`))
	})
}
