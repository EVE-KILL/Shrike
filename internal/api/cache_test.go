package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// An ETag only has to be stable for identical bytes and different for
// different ones. Clients treat it as opaque, so the algorithm is ours; these
// are the only two properties worth asserting.
func TestResponseETagIsStableAndContentDependent(t *testing.T) {
	t.Parallel()

	body := []byte(`{"data":[1,2,3]}`)
	if responseETag(body) != responseETag(body) {
		t.Error("the same body produced two different ETags")
	}
	if responseETag(body) == responseETag([]byte(`{"data":[1,2,4]}`)) {
		t.Error("different bodies produced the same ETag")
	}
	if got := responseETag(body); !strings.HasPrefix(got, `W/"`) ||
		!strings.HasSuffix(got, `"`) {
		t.Errorf("ETag %q is not a quoted weak validator", got)
	}
}

// The cache stores the content type with the bytes rather than assuming JSON.
// /killmails/{id}/eft returns an EFT fitting block as text/plain; when the hit
// path assumed application/json, those bytes came back as text while the entry
// was cold and as JSON once it was warm, so the header flipped on cache state.
func TestCacheHitPreservesANonJSONContentType(t *testing.T) {
	t.Parallel()

	entry := cachedResponse{
		ContentType: "text/plain; charset=utf-8",
		Body:        []byte("[Drake, Killmail #1]\n\nHeavy Missile Launcher II\n"),
	}
	response := httptest.NewRecorder()
	writeCacheHit(response,
		httptest.NewRequest(http.MethodGet, "http://api.example/killmails/1/eft", nil),
		entry, responseCachePolicy{TTL: time.Hour}, "/schemas/killmail-eft-response.json")

	if got := response.Header().Get("Content-Type"); got != entry.ContentType {
		t.Errorf("Content-Type = %q, want %q", got, entry.ContentType)
	}
	// $schema must not be injected into a body that is not JSON.
	if response.Body.String() != string(entry.Body) {
		t.Errorf("body was rewritten: %q", response.Body)
	}
}

func TestResponseCacheServesStaleWhileRefreshing(t *testing.T) {
	cache := newResponseCache(nil, 4096)
	request := httptest.NewRequest(
		http.MethodGet, "http://api.example/stats?dataType=characters", nil,
	)
	key := cacheKey("test", request)
	cache.local.Put(key, cachedResponse{
		ContentType: "text/plain",
		Body:        []byte("stale"),
		FreshUntil:  time.Now().Add(-time.Minute),
	}, time.Minute, time.Now())

	refreshed := make(chan struct{})
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("fresh"))
		close(refreshed)
	})
	schemas := &responseSchemaResolver{routes: []responseSchemaRoute{{
		method: http.MethodGet, path: "/stats",
	}}}

	response := httptest.NewRecorder()
	responseCache(cache, schemas, "test", next).ServeHTTP(response, request)
	if got := response.Body.String(); got != "stale" {
		t.Fatalf("body = %q, want stale", got)
	}
	select {
	case <-refreshed:
	case <-time.After(time.Second):
		t.Fatal("background refresh did not run")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		entry, ok := cache.Load(context.Background(), key)
		if ok && string(entry.Body) == "fresh" {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("fresh response was not stored")
}

// Two requests that differ only in query string must not share an entry, and
// Shrike's keyspace must not collide with the TypeScript API's `api:` prefix on
// the shared Valkey — otherwise each implementation serves the other's bodies.
func TestCacheKeyIsQueryAndBuildScoped(t *testing.T) {
	t.Parallel()

	req := func(target string) *http.Request {
		return httptest.NewRequest(http.MethodGet, "http://api.example"+target, nil)
	}
	a := cacheKey("abc123", req("/killmails?limit=1"))
	b := cacheKey("abc123", req("/killmails?limit=50"))
	if a == b {
		t.Error("different query strings produced the same cache key")
	}
	if cacheKey("abc123", req("/killmails")) == cacheKey("def456", req("/killmails")) {
		t.Error("different builds produced the same cache key")
	}
	for _, key := range []string{a, b} {
		if strings.HasPrefix(key, "api:") {
			t.Errorf("key %q collides with the TypeScript API's keyspace", key)
		}
	}
}

func TestCacheHitAndMissHaveIdenticalResponseMetadata(t *testing.T) {
	schemas := &responseSchemaResolver{routes: []responseSchemaRoute{{
		method: http.MethodGet, path: "/search",
		schemaPath: "/schemas/search-response.json",
	}}}
	body := []byte(`{"hits":[]}`)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})

	miss := httptest.NewRecorder()
	crossOriginAPI(responseCache(nil, schemas, "test", next)).ServeHTTP(
		miss, httptest.NewRequest(http.MethodGet, "http://api.example/search", nil),
	)

	hit := httptest.NewRecorder()
	crossOriginAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCacheHit(w, r, cachedResponse{
			ContentType: "application/json", Body: body,
		}, responseCachePolicy{TTL: time.Minute}, "/schemas/search-response.json")
	})).ServeHTTP(
		hit, httptest.NewRequest(http.MethodGet, "http://api.example/search", nil),
	)

	for _, name := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Expose-Headers",
		"Content-Type",
		"ETag",
		"Link",
	} {
		if got, want := hit.Header().Get(name), miss.Header().Get(name); got != want {
			t.Errorf("%s differs: HIT %q, MISS %q", name, got, want)
		}
	}
	if got := hit.Header().Get("Access-Control-Allow-Methods"); got !=
		"GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods = %q", got)
	}
	if got := hit.Header().Get("Access-Control-Allow-Headers"); got !=
		"Content-Type, If-None-Match, Last-Event-ID" {
		t.Errorf("Access-Control-Allow-Headers = %q", got)
	}
	if got := hit.Header().Get("Access-Control-Expose-Headers"); got !=
		"ETag, Link, X-Cache" {
		t.Errorf("Access-Control-Expose-Headers = %q", got)
	}
	if hit.Body.String() != miss.Body.String() {
		t.Errorf("HIT body %q differs from MISS body %q", hit.Body, miss.Body)
	}
	if !strings.Contains(hit.Body.String(),
		`"$schema":"http://api.example/schemas/search-response.json"`) {
		t.Errorf("response is missing $schema: %s", hit.Body)
	}
}

func TestCacheNotModifiedKeepsCORSAndSchemaLink(t *testing.T) {
	body := []byte(`{"data":[]}`)
	request := httptest.NewRequest(http.MethodGet, "https://api.example/sde/regions", nil)
	decorated := addResponseSchema(
		request, http.Header{"Content-Type": []string{"application/json"}},
		body, "/schemas/sde-regions-response.json",
	)
	request.Header.Set("If-None-Match", responseETag(decorated))
	response := httptest.NewRecorder()
	crossOriginAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCacheHit(w, r, cachedResponse{
			ContentType: "application/json", Body: body,
		}, responseCachePolicy{
			TTL: time.Hour, CacheControl: "public, max-age=60",
		}, "/schemas/sde-regions-response.json")
	})).ServeHTTP(response, request)

	if response.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", response.Code)
	}
	for name, want := range map[string]string{
		"Access-Control-Allow-Origin":   "*",
		"Access-Control-Allow-Methods":  "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS",
		"Access-Control-Allow-Headers":  "Content-Type, If-None-Match, Last-Event-ID",
		"Access-Control-Expose-Headers": "ETag, Link, X-Cache",
		"Cache-Control":                 "public, max-age=60",
		"Link":                          `</schemas/sde-regions-response.json>; rel="describedBy"`,
		"X-Cache":                       "HIT",
	} {
		if got := response.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestCacheMissHonorsIfNoneMatch(t *testing.T) {
	schemas := &responseSchemaResolver{routes: []responseSchemaRoute{{
		method: http.MethodGet, path: "/search",
		schemaPath: "/schemas/search-response.json",
	}}}
	baseBody := []byte(`{"hits":[]}`)
	request := httptest.NewRequest(http.MethodGet, "http://api.example/search", nil)
	decorated := addResponseSchema(
		request, http.Header{"Content-Type": []string{"application/json"}},
		baseBody, "/schemas/search-response.json",
	)
	request.Header.Set("If-None-Match", `"other", `+responseETag(decorated))

	response := httptest.NewRecorder()
	crossOriginAPI(responseCache(nil, schemas, "test", http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(baseBody)
		},
	))).ServeHTTP(response, request)

	if response.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", response.Code)
	}
	if response.Body.Len() != 0 {
		t.Errorf("304 body = %q, want empty", response.Body)
	}
	if response.Header().Get("X-Cache") != "MISS" {
		t.Errorf("X-Cache = %q, want MISS", response.Header().Get("X-Cache"))
	}
}

func TestResponseCacheBypassesAccountAndAdminRoutes(t *testing.T) {
	schemas := &responseSchemaResolver{routes: []responseSchemaRoute{{
		method: http.MethodGet, path: "/search",
		schemaPath: "/schemas/search-response.json",
	}}}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "private, no-store")
		_, _ = w.Write([]byte(`{"user":{"character_id":42}}`))
	})

	for _, path := range []string{"/me", "/admin/users"} {
		response := httptest.NewRecorder()
		responseCache(nil, schemas, "test", next).ServeHTTP(
			response,
			httptest.NewRequest(http.MethodGet, "https://eve-kill.com"+path, nil),
		)
		if got := response.Header().Get("Cache-Control"); got != "private, no-store" {
			t.Errorf("%s Cache-Control = %q", path, got)
		}
		for _, header := range []string{"ETag", "X-Cache", "Link"} {
			if got := response.Header().Get(header); got != "" {
				t.Errorf("%s unexpectedly received %s %q", path, header, got)
			}
		}
		if strings.Contains(response.Body.String(), `"$schema"`) {
			t.Errorf("%s account body was rewritten: %s", path, response.Body)
		}
	}
}

// Every tier has to expire on the same schedule. A shared cache that outlives
// the stored entry serves a body the origin has already replaced, so s-maxage
// tracks the TTL rather than being chosen separately from it.
func TestCachePolicySharedLifetimeMatchesStoredLifetime(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/killmails", "/killmails/count", "/history", "/history/latest",
		"/history/2026-07-27", "/characters", "/characters/1/kills",
		"/characters/1/losses", "/characters/1/count", "/characters/1/stats",
		"/characters/1/intel", "/corporations/1", "/corporations/1/members",
		"/alliances/1/corporations", "/ships/1", "/stats", "/search",
		"/battles", "/wars", "/sde/systems/1/kills",
	} {
		policy, ok := cachePolicy(
			httptest.NewRequest(http.MethodGet, "https://api.example"+path, nil),
		)
		if !ok {
			t.Errorf("%s is not cached at all", path)
			continue
		}
		want := int64(policy.TTL / time.Second)
		if want <= 0 {
			t.Errorf("%s has TTL %v", path, policy.TTL)
			continue
		}
		for _, directive := range []string{
			fmt.Sprintf("s-maxage=%d", want),
			fmt.Sprintf("stale-while-revalidate=%d", want),
		} {
			if !strings.Contains(policy.CacheControl, directive) {
				t.Errorf("%s Cache-Control %q is missing %q (TTL %v)",
					path, policy.CacheControl, directive, policy.TTL)
			}
		}
	}
}

// Killmails, battle reports and the static data export do not change once
// written, so their edge lifetime is deliberately longer than the lifetime the
// origin stores them for. The derived directives must not overwrite that.
func TestCachePolicyKeepsExplicitImmutableDirectives(t *testing.T) {
	t.Parallel()

	for path, want := range map[string]string{
		"/killmails/1":     "public, max-age=3600, s-maxage=31536000",
		"/killmails/1/eft": "public, max-age=3600, s-maxage=31536000",
		"/battles/1":       "public, max-age=3600, s-maxage=31536000",
		"/sde/systems":     "public, max-age=86400, s-maxage=31536000, stale-while-revalidate=86400",
		"/sde/types/587":   "public, max-age=86400, s-maxage=31536000, stale-while-revalidate=86400",
	} {
		policy, ok := cachePolicy(
			httptest.NewRequest(http.MethodGet, "https://api.example"+path, nil),
		)
		if !ok {
			t.Errorf("%s is not cached at all", path)
			continue
		}
		if policy.CacheControl != want {
			t.Errorf("%s Cache-Control = %q, want %q", path, policy.CacheControl, want)
		}
	}
}

// Session and moderation responses must never acquire shared directives.
func TestCachePolicyLeavesUncacheableRoutesAlone(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/me", "/admin/users", "/characters/analyze", "/killmails/search",
		"/coalitions/stats", "/health",
	} {
		if policy, ok := cachePolicy(
			httptest.NewRequest(http.MethodGet, "https://api.example"+path, nil),
		); ok {
			t.Errorf("%s became cacheable with %q", path, policy.CacheControl)
		}
	}
}

func TestIfNoneMatchWeakComparisonAndWildcard(t *testing.T) {
	current := `W/"abc"`
	for _, header := range []string{`W/"abc"`, `"abc"`, `"other", W/"abc"`, "*"} {
		if !ifNoneMatch(header, current) {
			t.Errorf("ifNoneMatch(%q, %q) = false", header, current)
		}
	}
	for _, header := range []string{"", `"other"`, `W/"abcd"`} {
		if ifNoneMatch(header, current) {
			t.Errorf("ifNoneMatch(%q, %q) = true", header, current)
		}
	}
}
