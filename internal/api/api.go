// Package api builds Shrike's HTTP API.
//
// The API hostname and eve-kill.com deliberately share one Huma registry and
// one OpenAPI document. The main site reaches the same root-path operations
// through its same-origin /api prefix; authentication is described per
// operation rather than by maintaining a second API catalogue.
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
	"github.com/go-chi/chi/v5"
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
	Graph   GraphDatabase
	Feed    *FeedManager
	Cache   *redis.Client

	Auth         AuthOptions
	DomainAssets DomainAssetStorage
}

type GraphDatabase interface {
	Read(context.Context, string, map[string]any) ([]map[string]any, error)
}

// Service is one API registry exposed through two transport adapters.
//
// APIHost serves root paths on api.eve-kill.com. SameOrigin accepts
// /api-prefixed paths on the main site and strips that transport-only prefix
// before dispatch. Both expose the same operations and OpenAPI document.
type Service struct {
	apiHost    http.Handler
	sameOrigin http.Handler
}

type sameOriginPrefixContextKey struct{}

// New builds the shared API once. Callers should retain the returned Service
// and hand its two transport adapters to ingress.
func New(opts Options) *Service {
	mux := chi.NewRouter()

	cfg := huma.DefaultConfig("EVE-KILL API", opts.Version)
	cfg.DocsRenderer = huma.DocsRendererScalar
	cfg.Info.Description = "The API powering EVE-KILL. Public data, signed-in " +
		"account operations, and administration share one best-effort stable " +
		"contract; authentication requirements are documented per operation."

	a := humachi.New(mux, cfg)
	a.OpenAPI().Servers = []*huma.Server{
		{URL: "/", Description: "API hostname"},
		{URL: "/api", Description: "Same-origin frontend"},
	}
	schemas := registerRoutes(a, opts)
	registerLegacyMethodGuards(mux)
	mux.HandleFunc("/", legacyFallback)
	mux.NotFound(legacyFallback)

	cached := responseCache(opts.Cache, schemas, opts.Commit, mux)
	return &Service{
		apiHost:    crossOriginAPI(cached),
		sameOrigin: sameOriginPrefix(cached),
	}
}

// APIHost serves the shared API at root paths.
func (s *Service) APIHost() http.Handler {
	return s.apiHost
}

// SameOrigin serves the shared API through eve-kill.com's /api prefix while
// leaving browser OAuth routes under /auth unchanged.
func (s *Service) SameOrigin() http.Handler {
	return s.sameOrigin
}

// APIHost constructs a root-path API handler. Production should normally
// retain a Service so this and SameOrigin share the same registry instance.
func APIHost(opts Options) http.Handler {
	return New(opts).APIHost()
}

// SameOrigin constructs the /api-prefixed transport used by the main site.
func SameOrigin(opts Options) http.Handler {
	return New(opts).SameOrigin()
}

func sameOriginPrefix(next http.Handler) http.Handler {
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

// Images is images.eve-kill.com. Placeholder, as WS.
func Images(Options) http.Handler {
	return notImplemented("image server")
}

func notImplemented(surface string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte(`{"title":"Not Implemented","status":501,` +
			`"detail":"the ` + surface + ` surface is routed but not yet ported"}`))
	})
}
