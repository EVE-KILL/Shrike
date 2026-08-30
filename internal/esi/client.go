// Package esi talks to CCP's EVE Swagger Interface.
//
// Every call runs the same pipeline in-process — cache, pause check, error
// budget, singleflight, rate limit, sequential lock, HTTP — with Redis as the
// shared substrate, so a dozen workers behave as one client against ESI's
// per-application budget.
//
// The thing to understand about ESI is that it polices two separate budgets.
// The rate limit is per endpoint family and is what the token bucket tracks. The
// *error* limit is global: exceed it and every endpoint returns 420 for a
// minute, whichever one you were abusing. That is why a 420 pauses the whole
// client rather than only the group that earned it.
package esi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/redisx"
	"github.com/redis/go-redis/v9"
)

// DefaultBaseURL matches ESI_BASE in the TypeScript client.
const DefaultBaseURL = "https://esi.evetech.net"

const (
	httpTimeout    = 10 * time.Second
	maxHTTPRetries = 3
	// maxPipelineRetries caps how many times a request re-enters the pipeline
	// after waiting for a pause, a token, or another worker's fetch. Without it
	// a saturated cluster spins instead of failing.
	maxPipelineRetries = 5
)

// httpBackoff is the wait before each HTTP retry. Deliberately not exponential
// past five seconds: ESI recovers in seconds or in minutes, and doubling into
// minutes only holds a worker hostage.
var httpBackoff = []time.Duration{200 * time.Millisecond, time.Second, 5 * time.Second}

const (
	keyPaused         = "esi:paused"
	keyErrorRemaining = "esi:error:remaining"
	keyErrorReset     = "esi:error:reset"
)

// Wait caps per attempt. Non-sequential groups hold large buckets refreshed from
// response headers, so a long wait there means something is wrong and bailing
// beats sleeping. Sequential groups have 60-second windows and hand-built
// presets, so a freshly drained bucket legitimately needs most of a window —
// capping those short would fail every trailing request in a burst.
const (
	waitCapNormal     = 10 * time.Second
	waitCapSequential = 65 * time.Second
	pauseWaitCap      = 5 * time.Second
)

func waitCap(g Group) time.Duration {
	if g.Sequential {
		return waitCapSequential
	}
	return waitCapNormal
}

// Response is the outcome of a request.
//
// Status 0 means the request never reached ESI — every retry was exhausted, or
// the pipeline gave up waiting. Callers must treat it as retryable rather than
// as a 4xx.
type Response[T any] struct {
	Data   *T     `json:"data"`
	Status int    `json:"status"`
	Group  string `json:"group,omitempty"`
	// RetryAfter is set on 420 and 429, in seconds.
	RetryAfter int `json:"retry_after,omitempty"`
	// Pages is the x-pages header, for endpoints that paginate.
	Pages int `json:"pages,omitempty"`
	// Cached reports that the body came from cache without touching ESI.
	Cached bool `json:"cached,omitempty"`
}

// OK reports whether ESI supplied usable data. A conditional request returns
// 304 when the cached representation is still current; finishGet attaches that
// cached body, so callers must treat it as a successful response too.
func (r Response[T]) OK() bool {
	return (r.Status == http.StatusOK || r.Status == http.StatusNotModified) && r.Data != nil
}

// Permanent reports whether retrying could ever produce a different answer.
// A 404 or 422 on a killmail means the id and hash do not go together; no amount
// of retrying will change that.
func (r Response[T]) Permanent() bool {
	switch r.Status {
	case http.StatusNotFound, http.StatusGone, http.StatusUnprocessableEntity, http.StatusBadRequest:
		return true
	}
	return false
}

// raw is the pipeline's output before it is typed. Cache coordination and HTTP
// retries operate on JSON bytes regardless of the response type, so decoding
// once at the typed edge keeps the shared pipeline small.
type raw struct {
	Body       json.RawMessage
	Status     int
	Group      string
	RetryAfter int
	Pages      int
	Cached     bool
}

// Client is the full ESI client. Safe for concurrent use.
type Client struct {
	BaseURL   string
	UserAgent string
	HTTP      *http.Client

	limiter *RateLimiter
	coord   *Coordination
	cache   *Cache
	coordis *redis.Client
	owned   []*redis.Client

	// maxWait overrides the per-attempt sleep ceiling when non-zero, and backoff
	// overrides the HTTP retry ladder. Only tests set them: exercising a drained
	// bucket or a failing server honestly means sleeping the real intervals,
	// which is a minute of wall clock for properties provable in milliseconds.
	maxWait time.Duration
	backoff []time.Duration
}

// retryBackoff is the wait before the nth HTTP retry.
func (c *Client) retryBackoff(attempt int) time.Duration {
	ladder := c.backoff
	if ladder == nil {
		ladder = httpBackoff
	}
	return ladder[min(attempt-1, len(ladder)-1)]
}

// waitCapFor is how long one attempt may sleep before giving up and re-entering
// the pipeline.
func (c *Client) waitCapFor(g Group) time.Duration {
	if c.maxWait > 0 {
		return c.maxWait
	}
	return waitCap(g)
}

// New builds a fully coordinated client from configuration.
func New(cfg *config.Config) *Client {
	sharedRedis := redisx.New(cfg)

	return &Client{
		BaseURL:   DefaultBaseURL,
		UserAgent: UserAgent(cfg),
		HTTP:      &http.Client{Timeout: httpTimeout},
		limiter:   NewRateLimiter(sharedRedis),
		coord:     NewCoordination(sharedRedis),
		cache:     NewCache(sharedRedis),
		coordis:   sharedRedis,
		owned:     []*redis.Client{sharedRedis},
	}
}

// NewForTest builds a client against dependencies the caller supplies.
//
// Exported because the packages that sit on top of this one need a real
// pipeline — real bucket, real singleflight — pointed at a fake ESI, and the
// fields that would take are unexported. It does not adopt the Redis clients:
// Close leaves them open for whoever created them.
func NewForTest(baseURL, userAgent string, coordination, cache *redis.Client) *Client {
	return &Client{
		BaseURL:   baseURL,
		UserAgent: userAgent,
		HTTP:      &http.Client{Timeout: httpTimeout},
		limiter:   NewRateLimiter(coordination),
		coord:     NewCoordination(coordination),
		cache:     NewCache(cache),
		coordis:   coordination,
	}
}

// UserAgent identifies us to CCP, which their acceptable-use policy requires.
func UserAgent(cfg *config.Config) string {
	if cfg != nil && cfg.ESIUserAgent != "" {
		return cfg.ESIUserAgent
	}
	return "shrike/dev (+https://github.com/EVE-KILL/Shrike)"
}

// Close releases the Redis connections the client opened.
func (c *Client) Close() {
	for _, r := range c.owned {
		_ = r.Close()
	}
}

// Limiter exposes the rate limiter for diagnostics.
func (c *Client) Limiter() *RateLimiter { return c.limiter }

// --- Typed entry points ---

// Get performs a cached, coordinated, rate-limited GET.
func Get[T any](ctx context.Context, c *Client, path string) (Response[T], error) {
	r, err := c.doGet(ctx, path)
	return typed[T](r, err)
}

// GetAuthenticated performs a GET with a character's access token.
//
// Authenticated responses are per-character and are neither cached nor
// deduplicated — two characters asking the same question get different answers,
// and a shared cache keyed by URL would hand one character another's data.
func GetAuthenticated[T any](ctx context.Context, c *Client, path, accessToken string) (Response[T], error) {
	r, err := c.doAuthenticated(ctx, path, accessToken)
	return typed[T](r, err)
}

// Post performs a POST. Used by /characters/affiliation/, which is a read
// expressed as a write because the id list does not fit in a URL.
func Post[T any](ctx context.Context, c *Client, path string, body any) (Response[T], error) {
	r, err := c.doPost(ctx, path, body)
	return typed[T](r, err)
}

func typed[T any](r raw, err error) (Response[T], error) {
	out := Response[T]{
		Status:     r.Status,
		Group:      r.Group,
		RetryAfter: r.RetryAfter,
		Pages:      r.Pages,
		Cached:     r.Cached,
	}
	if err != nil || len(r.Body) == 0 {
		return out, err
	}
	var data T
	if err := json.Unmarshal(r.Body, &data); err != nil {
		return out, fmt.Errorf("decode ESI response: %w", err)
	}
	out.Data = &data
	return out, nil
}

// --- The GET pipeline ---

func (c *Client) doGet(ctx context.Context, path string) (raw, error) {
	url := c.fullURL(path)
	group := ResolveGroup(url)
	etag := ""

	for attempt := 0; attempt <= maxPipelineRetries; attempt++ {
		// 1. Cache. A fresh entry ends the request without any coordination:
		// no tokens spent, no claim taken, no ESI contact.
		cached := c.cache.Get(ctx, url)
		if cached.Fresh(time.Now()) {
			return raw{Body: cached.Data, Status: cached.Status, Group: group.Name, Cached: true}, nil
		}
		// A stale entry still carries an ETag worth sending.
		if etag == "" && cached != nil {
			etag = cached.ETag
		}

		// 2. Global pause, set by anyone who saw a 420.
		if wait := c.pauseRemaining(ctx); wait > 0 {
			if err := sleep(ctx, min(wait, pauseWaitCap)); err != nil {
				return raw{Group: group.Name}, err
			}
			continue
		}

		// 3. Error-budget pacing. Distinct from the rate limit: this throttles
		// on how many errors we have already caused, before ESI cuts us off.
		if delay := c.errorBudgetDelay(ctx); delay > 0 {
			if err := sleep(ctx, delay); err != nil {
				return raw{Group: group.Name}, err
			}
		}

		// 4. Singleflight. Losing means another worker is already fetching this
		// exact URL; wait for them and re-enter, which will hit their cache fill.
		claim, err := c.coord.TryClaim(ctx, url)
		if err != nil {
			return raw{Group: group.Name}, err
		}
		if claim == "" {
			c.coord.WaitForClaim(ctx, url)
			continue
		}

		// 5. Rate limit. Releasing the claim before sleeping matters: holding it
		// would block every other worker for the whole wait, turning a rate
		// limit into a cluster-wide stall.
		wait, err := c.limiter.Acquire(ctx, group, TokenCost)
		if err != nil {
			c.coord.ReleaseClaim(ctx, url, claim)
			return raw{Group: group.Name}, err
		}
		if wait > 0 {
			c.coord.ReleaseClaim(ctx, url, claim)
			if err := sleep(ctx, min(wait, c.waitCapFor(group))); err != nil {
				return raw{Group: group.Name}, err
			}
			continue
		}

		result, err := c.executeGuarded(ctx, group, func() (httpResult, error) {
			return c.execute(ctx, http.MethodGet, url, nil, etag, "")
		})
		if err != nil {
			c.coord.ReleaseClaim(ctx, url, claim)
			return raw{Group: group.Name}, err
		}
		if result == nil {
			// Sequential lock timed out. Retryable, not an error.
			c.coord.ReleaseClaim(ctx, url, claim)
			return raw{Status: 0, Group: group.Name}, nil
		}

		// The claim is held until after the response is cached. Releasing it
		// first leaves a window in which a waiting worker wakes, finds the cache
		// still empty, and fetches the same URL over again — which is the exact
		// duplication singleflight exists to prevent.
		out, ferr := c.finishGet(ctx, url, group, etag, cached, *result)
		c.coord.ReleaseClaim(ctx, url, claim)
		return out, ferr
	}

	// Every attempt was spent waiting rather than fetching.
	return raw{Status: 0, Group: group.Name}, nil
}

// doProbe performs an uncached GET without consulting the shared pause,
// singleflight, or rate-limit state.
//
// This path exists for the Tranquility status endpoint, which is the authority
// that sets and clears the global pause. Sending that request through the
// regular pipeline creates a feedback loop: one failed check sets the pause,
// then every later check obeys it and can never observe recovery. A status
// probe is cheap, scheduled once per cluster, and has its own generous ESI
// budget, so it is both safe and necessary to let it reach ESI directly.
func (c *Client) doProbe(ctx context.Context, path string) (raw, error) {
	url := c.fullURL(path)
	group := ResolveGroup(url)

	result, err := c.execute(ctx, http.MethodGet, url, nil, "", "")
	if err != nil {
		return raw{Group: group.Name}, err
	}

	// The probe bypasses local admission control, but its response still keeps
	// the shared ESI diagnostics current.
	c.updateErrorBudget(ctx, result.Header)
	c.applyRateLimitHeaders(ctx, group, result.Header)

	out := raw{
		Status: result.Status,
		Group:  group.Name,
		Pages:  result.Pages,
	}
	switch result.Status {
	case http.StatusOK:
		out.Body = result.Body
	case statusErrorLimited:
		out.RetryAfter = c.pause(ctx, result.Header)
	case http.StatusTooManyRequests:
		out.RetryAfter = result.RetryAfter
	}
	return out, nil
}

// finishGet turns an HTTP result into a response, caching what deserves it.
func (c *Client) finishGet(ctx context.Context, url string, group Group, etag string, cached *Entry, res httpResult) (raw, error) {
	out := raw{Status: res.Status, Group: group.Name, Pages: res.Pages}

	switch res.Status {
	case http.StatusOK:
		expires, ok := ParseExpires(res.Header.Get("expires"))
		if !ok {
			// No Expires header. A minute is short enough to stay honest and
			// long enough to absorb a burst of identical requests.
			expires = time.Now().Add(time.Minute).UnixMilli()
		}
		c.cache.Set(ctx, url, &Entry{
			Data:    res.Body,
			Status:  http.StatusOK,
			Expires: expires,
			ETag:    res.Header.Get("etag"),
		})
		out.Body = res.Body

	case http.StatusNotModified:
		// ESI confirms what we already hold. Refresh its life and serve it.
		expires, ok := ParseExpires(res.Header.Get("expires"))
		if !ok {
			expires = time.Now().Add(time.Minute).UnixMilli()
		}
		newETag := res.Header.Get("etag")
		if newETag == "" {
			newETag = etag
		}
		c.cache.Touch(ctx, url, expires, newETag)
		if cached != nil {
			out.Body = cached.Data
		}

	case statusErrorLimited:
		out.RetryAfter = c.pause(ctx, res.Header)

	case http.StatusTooManyRequests:
		out.RetryAfter = res.RetryAfter
	}

	return out, nil
}

// --- Authenticated GET ---

func (c *Client) doAuthenticated(ctx context.Context, path, accessToken string) (raw, error) {
	url := c.fullURL(path)
	group := ResolveGroup(url)

	if !c.waitForPermit(ctx, group) {
		return raw{Status: 0, Group: group.Name}, nil
	}

	result, err := c.executeGuarded(ctx, group, func() (httpResult, error) {
		return c.execute(ctx, http.MethodGet, url, nil, "", accessToken)
	})
	if err != nil {
		return raw{Group: group.Name}, err
	}
	if result == nil {
		return raw{Status: 0, Group: group.Name}, nil
	}

	out := raw{Status: result.Status, Group: group.Name, Pages: result.Pages}
	switch result.Status {
	case http.StatusOK:
		out.Body = result.Body
	case statusErrorLimited:
		out.RetryAfter = c.pause(ctx, result.Header)
	case http.StatusTooManyRequests:
		out.RetryAfter = result.RetryAfter
	}
	return out, nil
}

// --- POST ---

func (c *Client) doPost(ctx context.Context, path string, body any) (raw, error) {
	url := c.fullURL(path)
	group := ResolveGroup(url)

	payload, err := json.Marshal(body)
	if err != nil {
		return raw{Group: group.Name}, fmt.Errorf("encode request body: %w", err)
	}

	if !c.waitForPermit(ctx, group) {
		return raw{Status: 0, Group: group.Name}, nil
	}

	result, err := c.executeGuarded(ctx, group, func() (httpResult, error) {
		return c.execute(ctx, http.MethodPost, url, payload, "", "")
	})
	if err != nil {
		return raw{Group: group.Name}, err
	}
	if result == nil {
		return raw{Status: 0, Group: group.Name}, nil
	}

	out := raw{Status: result.Status, Group: group.Name}
	switch result.Status {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		out.Body = result.Body
	case statusErrorLimited:
		out.RetryAfter = c.pause(ctx, result.Header)
	}
	return out, nil
}

// --- Shared pipeline pieces ---

// waitForPermit blocks until both the global pause and the token bucket allow a
// request. False means it gave up.
func (c *Client) waitForPermit(ctx context.Context, group Group) bool {
	for attempt := 0; attempt <= maxPipelineRetries; attempt++ {
		if wait := c.pauseRemaining(ctx); wait > 0 {
			if err := sleep(ctx, min(wait, pauseWaitCap)); err != nil {
				return false
			}
			continue
		}
		wait, err := c.limiter.Acquire(ctx, group, TokenCost)
		if err != nil {
			return false
		}
		if wait == 0 {
			if delay := c.errorBudgetDelay(ctx); delay > 0 {
				if err := sleep(ctx, delay); err != nil {
					return false
				}
			}
			return true
		}
		if err := sleep(ctx, min(wait, c.waitCapFor(group))); err != nil {
			return false
		}
	}
	return false
}

// executeGuarded takes the sequential lock when the group needs it, runs the
// request, then feeds both budgets from the response headers.
//
// A nil result with a nil error means the sequential lock could not be had —
// the caller should report a retryable outcome rather than a failure.
func (c *Client) executeGuarded(ctx context.Context, group Group, run func() (httpResult, error)) (*httpResult, error) {
	if group.Sequential {
		token, err := c.coord.AcquireSequential(ctx, group.Name)
		if err != nil {
			return nil, err
		}
		if token == "" {
			return nil, nil
		}
		defer c.coord.ReleaseSequential(ctx, group.Name, token)
	}

	result, err := run()
	if err != nil {
		return nil, err
	}

	c.updateErrorBudget(ctx, result.Header)
	c.applyRateLimitHeaders(ctx, group, result.Header)
	return &result, nil
}

// statusErrorLimited is ESI's 420: the global error budget is spent.
const statusErrorLimited = 420

// pause halts the whole client for as long as ESI says, and reports the same
// figure to the caller.
func (c *Client) pause(ctx context.Context, header http.Header) int {
	reset := 60
	if v, err := strconv.Atoi(header.Get("x-esi-error-limit-reset")); err == nil && v > 0 {
		reset = v
	}
	_ = c.coordis.Set(ctx, keyPaused, "1", time.Duration(reset)*time.Second).Err()
	return reset
}

func (c *Client) pauseRemaining(ctx context.Context) time.Duration {
	ttl, err := c.coordis.PTTL(ctx, keyPaused).Result()
	if err != nil || ttl <= 0 {
		return 0
	}
	return ttl
}

// errorBudgetDelay spreads what is left of the error budget over what is left of
// its window, so a client that has been erroring slows down before ESI stops it.
func (c *Client) errorBudgetDelay(ctx context.Context) time.Duration {
	vals, err := c.coordis.MGet(ctx, keyErrorRemaining, keyErrorReset).Result()
	if err != nil || len(vals) != 2 {
		return 0
	}

	remaining := 100
	if n, ok := parseRedisInt(vals[0]); ok {
		remaining = int(n)
	}
	if remaining >= 80 {
		return 0
	}
	if remaining < 10 {
		// Nearly spent. Five seconds a request is close enough to a halt to let
		// the window expire without tipping over it.
		return 5 * time.Second
	}

	resetAt := time.Now().Add(time.Minute)
	if n, ok := parseRedisInt(vals[1]); ok {
		resetAt = time.UnixMilli(n)
	}
	secondsLeft := max(1.0, time.Until(resetAt).Seconds())
	return time.Duration(secondsLeft/float64(remaining)*1000) * time.Millisecond
}

func (c *Client) updateErrorBudget(ctx context.Context, header http.Header) {
	if v := header.Get("x-esi-error-limit-remain"); v != "" {
		_ = c.coordis.Set(ctx, keyErrorRemaining, v, 0).Err()
	}
	if v := header.Get("x-esi-error-limit-reset"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			resetAt := time.Now().Add(time.Duration(secs) * time.Second).UnixMilli()
			_ = c.coordis.Set(ctx, keyErrorReset, strconv.FormatInt(resetAt, 10), 0).Err()
		}
	}
}

func (c *Client) applyRateLimitHeaders(ctx context.Context, group Group, header http.Header) {
	if !group.HeaderAuthoritative {
		return
	}
	remain := header.Get("x-ratelimit-remaining")
	if remain == "" {
		return
	}
	remainNum, err := strconv.Atoi(remain)
	if err != nil {
		return
	}

	// Most endpoints emitting x-ratelimit-remaining also emit x-ratelimit-reset.
	// Killmails are the known exception — group, limit, remaining and used, but
	// no reset — so the counter is updated alone and reset_at is left to roll
	// over naturally.
	if reset := header.Get("x-ratelimit-reset"); reset != "" {
		if secs, err := strconv.Atoi(reset); err == nil {
			_ = c.limiter.ApplyHeaders(ctx, group, remainNum, time.Duration(secs)*time.Second)
			return
		}
	}
	_ = c.limiter.ApplyRemaining(ctx, group, remainNum)
}

// --- HTTP ---

type httpResult struct {
	Status     int
	Body       json.RawMessage
	Header     http.Header
	RetryAfter int
	Pages      int
}

// execute performs the request, retrying transport failures and 5xx.
//
// 4xx are never retried: they are answers, not failures, and retrying them
// spends error budget to be told the same thing again.
func (c *Client) execute(ctx context.Context, method, url string, body []byte, etag, accessToken string) (httpResult, error) {
	var last httpResult

	for attempt := 0; attempt <= maxHTTPRetries; attempt++ {
		if attempt > 0 {
			if err := sleep(ctx, c.retryBackoff(attempt)); err != nil {
				return last, err
			}
		}

		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return last, err
		}
		req.Header.Set("User-Agent", c.UserAgent)
		req.Header.Set("Accept", "application/json")
		// ESI versions its schemas by date. Pinning a week back means a schema
		// change lands in our lap a week after it ships rather than the moment
		// CCP deploys it.
		req.Header.Set("X-Compatibility-Date", compatibilityDate())
		if accessToken != "" {
			req.Header.Set("Authorization", "Bearer "+accessToken)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}

		resp, err := c.HTTP.Do(req)
		if err != nil {
			// A cancelled context is a decision, not a transport failure.
			if ctx.Err() != nil {
				return last, ctx.Err()
			}
			last = httpResult{Header: http.Header{}}
			continue
		}

		result := httpResult{Status: resp.StatusCode, Header: resp.Header.Clone()}
		if v, err := strconv.Atoi(resp.Header.Get("x-pages")); err == nil {
			result.Pages = v
		}
		if v, err := strconv.Atoi(resp.Header.Get("retry-after")); err == nil {
			result.RetryAfter = v
		} else if resp.StatusCode == http.StatusTooManyRequests {
			result.RetryAfter = 5
		}

		switch {
		case resp.StatusCode == http.StatusOK, resp.StatusCode == http.StatusCreated:
			// Read the body before closing; a truncated read is a retryable
			// transport failure, not a bad response.
			data, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				last = result
				continue
			}
			result.Body = data
			return result, nil

		case resp.StatusCode >= 500 && resp.StatusCode < 600:
			resp.Body.Close()
			last = result
			continue

		default:
			// 3xx and 4xx: an answer. Drain so the connection is reusable.
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
			resp.Body.Close()
			return result, nil
		}
	}

	if last.Header == nil {
		last.Header = http.Header{}
	}
	return last, nil
}

func (c *Client) fullURL(path string) string {
	if strings.HasPrefix(path, "http") {
		return path
	}
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	return strings.TrimSuffix(base, "/") + path
}

// compatibilityDate is a week behind today, matching the TypeScript client.
func compatibilityDate() string {
	return time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02")
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
