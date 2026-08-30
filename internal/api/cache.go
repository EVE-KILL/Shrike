package api

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type responseCachePolicy struct {
	TTL          time.Duration
	CacheControl string
}

var (
	numericPath = regexp.MustCompile(`/[0-9]+(?:/|$)`)
	entityBase  = regexp.MustCompile(`^/(characters|corporations|alliances)(?:/|$)`)
)

func responseCache(
	cache *ResponseCache,
	schemas *responseSchemaResolver,
	build string,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		schemaPath, routeMatched := schemas.lookup(r.Method, r.URL.Path)
		if !routeMatched {
			next.ServeHTTP(w, r)
			return
		}
		policy, ok := cachePolicy(r)
		if !ok || r.Method != http.MethodGet {
			if schemaPath == "" || r.URL.Path == "/feed/stream" {
				next.ServeHTTP(w, r)
				return
			}
			writeUncachedResponse(w, r, next, schemaPath)
			return
		}
		key := cacheKey(build, r)
		entry, fresh, found := cacheLoadState(r.Context(), cache, key)
		if found && fresh {
			writeCacheHit(w, r, entry, policy, schemaPath)
			return
		}
		staleTTL := min(policy.TTL, 10*time.Minute)
		if found {
			cache.Refresh(key, func() (any, error) {
				refreshCtx, cancel := context.WithTimeout(
					context.WithoutCancel(r.Context()), 2*time.Minute,
				)
				defer cancel()
				refreshRequest := r.Clone(refreshCtx)
				recorder := newCacheRecorder()
				next.ServeHTTP(recorder, refreshRequest)
				result := collapsedResponse{
					status: recorder.status,
					header: recorder.header.Clone(),
					body:   append([]byte(nil), recorder.body.Bytes()...),
				}
				if result.status == http.StatusOK {
					cache.StoreStale(refreshCtx, key, cachedResponse{
						ContentType: result.header.Get("Content-Type"),
						Body:        result.body,
					}, policy.TTL, staleTTL)
				}
				return result, nil
			})
			writeCacheHit(w, r, entry, policy, schemaPath)
			return
		}

		value, err, _ := cache.LoadOnce(r.Context(), key, func() (any, error) {
			recorder := newCacheRecorder()
			next.ServeHTTP(recorder, r)
			result := collapsedResponse{
				status: recorder.status,
				header: recorder.header.Clone(),
				body:   append([]byte(nil), recorder.body.Bytes()...),
			}
			if result.status == http.StatusOK {
				cache.StoreStale(context.WithoutCancel(r.Context()), key, cachedResponse{
					ContentType: result.header.Get("Content-Type"),
					Body:        result.body,
				}, policy.TTL, staleTTL)
			}
			return result, nil
		})
		if err != nil {
			return
		}
		result := value.(collapsedResponse)
		cachedBody := result.body
		copyResponseHeaders(w.Header(), result.header)
		w.Header().Set("X-Cache", "MISS")
		body := cachedBody
		if result.status >= 200 && result.status < 300 {
			body = addResponseSchema(r, w.Header(), body, schemaPath)
		}
		if result.status == http.StatusOK {
			etag := responseETag(body)
			w.Header().Set("ETag", etag)
			if policy.CacheControl != "" {
				w.Header().Set("Cache-Control", policy.CacheControl)
			}
			if ifNoneMatch(r.Header.Get("If-None-Match"), etag) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		w.WriteHeader(result.status)
		_, _ = w.Write(body)
	})
}

type collapsedResponse struct {
	status int
	header http.Header
	body   []byte
}

// cachedResponse is what gets stored: the bytes plus the content type that
// described them.
//
// The content type has to be stored, not assumed. It used to be assumed —
// the hit path set application/json unconditionally — and /killmails/{id}/eft
// returns an EFT fitting block as text/plain, so the same bytes came back as
// text while the entry was cold and as JSON once it was warm. The header
// flipped on cache state alone, which is worse than either answer.
type cachedResponse struct {
	ContentType string
	Body        []byte
	FreshUntil  time.Time
}

// cacheKey namespaces entries to Shrike and to this build.
//
// The TypeScript API caches the same routes under `api:{path}{query}` on the
// same Valkey. Sharing that keyspace meant whichever process answered first
// populated the entry and the other served it verbatim — so during a
// side-by-side cutover each implementation would hand back the other's bodies,
// and any behavioural difference between them would surface at random. Worse
// for us, it made the two look identical under test for exactly that reason.
//
// The build is in the key because a response shape is a property of the code
// that produced it. A deploy that changes a payload must not inherit the old
// one, and a rolling deploy must not have two versions fighting over one entry.
func cacheKey(build string, r *http.Request) string {
	if build == "" {
		build = "dev"
	}
	key := "shrike:api:" + build + ":" + r.URL.Path
	if r.URL.RawQuery != "" {
		key += "?" + r.URL.RawQuery
	}
	return key
}

func cacheLoad(ctx context.Context, cache *ResponseCache, key string) (cachedResponse, bool) {
	if cache == nil {
		return cachedResponse{}, false
	}
	return cache.Load(ctx, key)
}

func cacheLoadState(
	ctx context.Context,
	cache *ResponseCache,
	key string,
) (cachedResponse, bool, bool) {
	if cache == nil {
		return cachedResponse{}, false, false
	}
	return cache.LoadState(ctx, key)
}

func cacheStore(
	ctx context.Context,
	cache *ResponseCache,
	key string,
	entry cachedResponse,
	ttl time.Duration,
) {
	if cache == nil {
		return
	}
	cache.Store(ctx, key, entry, ttl)
}

func writeUncachedResponse(
	w http.ResponseWriter,
	r *http.Request,
	next http.Handler,
	schemaPath string,
) {
	recorder := newCacheRecorder()
	next.ServeHTTP(recorder, r)
	copyResponseHeaders(w.Header(), recorder.header)
	body := recorder.body.Bytes()
	if recorder.status >= 200 && recorder.status < 300 {
		body = addResponseSchema(r, w.Header(), body, schemaPath)
	}
	w.WriteHeader(recorder.status)
	_, _ = w.Write(body)
}

func cachePolicy(r *http.Request) (responseCachePolicy, bool) {
	policy, ok := cachePolicyFor(r.URL.Path)
	if !ok {
		return responseCachePolicy{}, false
	}
	// A branch that names its own directives has a reason to diverge from its
	// storage lifetime. Everything else expires at the edge exactly when the
	// stored entry does.
	if policy.CacheControl == "" {
		policy.CacheControl = sharedCacheControl(policy.TTL)
	}
	return policy, true
}

// sharedCacheControl derives the response directives from the lifetime the
// entry is stored under, so Cloudflare and the origin expire together.
//
// These routes used to return a TTL and no directives at all. The origin
// refreshed on schedule, but with no s-maxage Cloudflare fell back to the zone
// default edge lifetime, so the tier in front of the origin was the one tier
// whose expiry nobody had chosen. Deriving both from one number is what keeps
// them from drifting apart as TTLs change.
//
// max-age is a quarter of the shared lifetime so a client revalidates well
// before the shared copy goes stale. That ratio matches the entity page routes.
func sharedCacheControl(ttl time.Duration) string {
	seconds := int64(ttl / time.Second)
	if seconds <= 0 {
		return ""
	}
	return fmt.Sprintf(
		"public, max-age=%d, s-maxage=%d, stale-while-revalidate=%d",
		maxInt64(30, seconds/4), seconds, seconds,
	)
}

func cachePolicyFor(path string) (responseCachePolicy, bool) {
	switch path {
	case "/characters/analyze", "/characters/intel", "/characters/stats", "/corporations/stats",
		"/alliances/stats", "/coalitions/stats", "/killmails/search":
		return responseCachePolicy{}, false
	}
	immutable := "public, max-age=3600, s-maxage=31536000"
	sde := "public, max-age=86400, s-maxage=31536000, stale-while-revalidate=86400"
	switch {
	case path == "/killmails/count":
		return responseCachePolicy{TTL: 10 * time.Minute}, true
	case strings.HasPrefix(path, "/killmails/") &&
		(pathHasSuffix(path, "/esi", "/fitting", "/eft") ||
			numericPath.MatchString(strings.TrimPrefix(path, "/killmails"))):
		return responseCachePolicy{TTL: time.Hour, CacheControl: immutable}, true
	case path == "/killmails":
		return responseCachePolicy{TTL: 30 * time.Second}, true
	case path == "/history/latest":
		return responseCachePolicy{TTL: time.Minute}, true
	case path == "/history":
		return responseCachePolicy{TTL: 5 * time.Minute}, true
	case strings.HasPrefix(path, "/history/"):
		return responseCachePolicy{TTL: time.Hour}, true
	case entityBase.MatchString(path):
		switch {
		case strings.HasSuffix(path, "/count"):
			return responseCachePolicy{TTL: 10 * time.Minute}, true
		case strings.HasSuffix(path, "/intel"):
			return responseCachePolicy{TTL: time.Hour}, true
		case strings.Contains(path, "/stats"):
			return responseCachePolicy{TTL: 10 * time.Minute}, true
		case pathHasSuffix(path, "/kills", "/losses"):
			return responseCachePolicy{TTL: 30 * time.Second}, true
		case pathHasSuffix(path, "/members", "/corporations"):
			return responseCachePolicy{TTL: 5 * time.Minute}, true
		default:
			segments := strings.Count(strings.Trim(path, "/"), "/")
			if segments == 0 {
				return responseCachePolicy{TTL: 30 * time.Second}, true
			}
			if segments == 1 {
				return responseCachePolicy{TTL: 5 * time.Minute}, true
			}
		}
	case strings.HasPrefix(path, "/ships/"):
		return responseCachePolicy{TTL: 10 * time.Minute}, true
	case path == "/stats":
		return responseCachePolicy{TTL: 5 * time.Minute}, true
	case path == "/search":
		return responseCachePolicy{TTL: time.Minute}, true
	case strings.HasPrefix(path, "/battles"):
		policy := responseCachePolicy{TTL: 5 * time.Minute}
		if strings.Count(strings.Trim(path, "/"), "/") == 1 {
			if _, err := strconv.ParseInt(
				strings.TrimPrefix(path, "/battles/"), 10, 64,
			); err == nil {
				policy.CacheControl = immutable
			}
		}
		return policy, true
	case strings.HasPrefix(path, "/wars"):
		return responseCachePolicy{TTL: 5 * time.Minute}, true
	case strings.HasSuffix(path, "/kills") &&
		(strings.HasPrefix(path, "/sde/systems/") ||
			strings.HasPrefix(path, "/sde/regions/")):
		return responseCachePolicy{TTL: 30 * time.Second}, true
	case isSDEPath(path):
		return responseCachePolicy{TTL: 24 * time.Hour, CacheControl: sde}, true
	default:
		return responseCachePolicy{}, false
	}
	return responseCachePolicy{}, false
}

func isSDEPath(path string) bool {
	if !strings.HasPrefix(path, "/sde/") {
		return false
	}
	path = strings.TrimPrefix(path, "/sde")
	for _, prefix := range []string{
		"/systems", "/regions", "/constellations", "/factions", "/races",
		"/bloodlines", "/types", "/groups", "/categories", "/market-groups",
		"/meta-groups", "/flags", "/celestials", "/npc-corporations",
		"/stations", "/station-operations", "/structures", "/sovereignty",
		"/prices", "/custom-prices",
	} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func pathHasSuffix(path string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

func writeCacheHit(
	w http.ResponseWriter,
	r *http.Request,
	entry cachedResponse,
	policy responseCachePolicy,
	schemaPath string,
) {
	w.Header().Set("Content-Type", entry.ContentType)
	// addResponseSchema is a no-op for anything that is not JSON, so a text/plain
	// body passes through untouched.
	body := addResponseSchema(r, w.Header(), entry.Body, schemaPath)
	etag := responseETag(body)
	w.Header().Set("X-Cache", "HIT")
	if ifNoneMatch(r.Header.Get("If-None-Match"), etag) {
		w.Header().Set("ETag", etag)
		if policy.CacheControl != "" {
			w.Header().Set("Cache-Control", policy.CacheControl)
		}
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("ETag", etag)
	if policy.CacheControl != "" {
		w.Header().Set("Cache-Control", policy.CacheControl)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// responseETag derives a weak validator from the body.
//
// The algorithm is ours to choose. An ETag is opaque to clients — they store
// whatever arrives and echo it back — so it only has to be stable for identical
// bytes and different for different ones. This previously reimplemented Bun's
// wyhash so validators would survive the cutover unchanged; that is ninety
// lines of bit-twiddling to save every client exactly one full-body
// revalidation, once, on the day we deploy.
func responseETag(body []byte) string {
	sum := fnv.New64a()
	_, _ = sum.Write(body)
	return `W/"` + strconv.FormatUint(sum.Sum64(), 36) + `"`
}

// ifNoneMatch implements weak comparison for the validators Shrike emits.
// Browsers and shared caches may combine several validators into one header,
// and `*` means any current representation.
func ifNoneMatch(header, current string) bool {
	if header == "" {
		return false
	}
	current = strings.TrimPrefix(strings.TrimSpace(current), "W/")
	for candidate := range strings.SplitSeq(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" {
			return true
		}
		candidate = strings.TrimPrefix(candidate, "W/")
		if candidate == current {
			return true
		}
	}
	return false
}

type cacheRecorder struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newCacheRecorder() *cacheRecorder {
	return &cacheRecorder{header: http.Header{}, status: http.StatusOK}
}

func (r *cacheRecorder) Header() http.Header {
	return r.header
}

func (r *cacheRecorder) WriteHeader(status int) {
	if r.status == http.StatusOK {
		r.status = status
	}
}

func (r *cacheRecorder) Write(body []byte) (int, error) {
	return r.body.Write(body)
}

func copyResponseHeaders(destination, source http.Header) {
	for name, values := range source {
		for _, value := range values {
			destination.Add(name, value)
		}
	}
}
