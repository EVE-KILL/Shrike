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
	Images       *images.Service
}

type GraphDatabase interface {
	Read(context.Context, string, map[string]any) ([]map[string]any, error)
}

// Service is one API registry exposed through the main-site transport.
type Service struct {
	site http.Handler
}

type sameOriginPrefixContextKey struct{}

// New builds the shared API once.
func New(opts Options) *Service {
	mux := chi.NewRouter()

	cfg := huma.DefaultConfig("EVE-KILL API", opts.Version)
	cfg.DocsRenderer = huma.DocsRendererScalar
	cfg.Info.Description = "The API powering EVE-KILL. Public data, signed-in " +
		"account operations, and administration share one best-effort stable " +
		"contract; authentication requirements are documented per operation."

	a := humachi.New(mux, cfg)
	a.OpenAPI().Servers = []*huma.Server{
		{URL: "/api", Description: "EVE-KILL API"},
	}
	schemas := registerRoutes(a, opts)
	setRootNamespaceServers(a.OpenAPI())
	registerLegacyMethodGuards(mux)
	mux.HandleFunc("/", legacyFallback)
	mux.NotFound(legacyFallback)

	cached := responseCache(opts.Cache, schemas, opts.Commit, mux)
	return &Service{site: sitePaths(cached)}
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

// Site serves /api, /auth, /images, and /health on the frontend origin.
func (s *Service) Site() http.Handler {
	return s.site
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
			r.URL.Path == "/health":
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
