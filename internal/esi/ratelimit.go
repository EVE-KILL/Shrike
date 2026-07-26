package esi

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// The distributed token bucket.
//
// State lives in Redis because the budget is per-application, not per-process:
// ESI counts every request the whole deployment makes. A local limiter in each
// worker would multiply the allowance by the worker count and get us 420'd.
//
// The window is fixed, not sliding, because that is what ESI does — the budget
// snaps back to full at the boundary rather than trickling in.

// TokenCost is what one request draws. Two rather than one, so that a group's
// nominal limit is reached at half the request count: headroom for the requests
// already in flight when a header snapshot was taken.
const TokenCost = 2

// luaAcquire refills the bucket if the window has rolled over, then either
// consumes the cost or reports how long to wait — atomically, so no number of
// concurrent callers can oversubscribe.
//
// KEYS: remaining, reset_at. ARGV: limit, window_ms, cost, now_ms.
// Returns 0 when acquired, otherwise milliseconds to wait.
const luaAcquire = `
local limit = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local cost = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

local remaining = tonumber(redis.call('GET', KEYS[1]))
local reset_at = tonumber(redis.call('GET', KEYS[2]))

if remaining == nil or reset_at == nil or now >= reset_at then
    remaining = limit
    reset_at = now + window_ms
    redis.call('SET', KEYS[2], reset_at, 'PX', window_ms + 5000)
end

if remaining >= cost then
    remaining = remaining - cost
    redis.call('SET', KEYS[1], remaining, 'PX', math.max(1000, reset_at - now + 5000))
    return 0
else
    redis.call('SET', KEYS[1], remaining, 'PX', math.max(1000, reset_at - now + 5000))
    return reset_at - now
end
`

// luaApplyHeaders merges ESI's own accounting back into the bucket.
//
// A newer reset boundary means the window rolled over on ESI's side, so the
// header is adopted wholesale. Otherwise a lower remaining is adopted and a
// higher one ignored: our count includes requests ESI had not yet seen when it
// wrote that header, and taking the larger number would double-spend them.
//
// KEYS: remaining, reset_at. ARGV: header_remaining, header_reset_at_ms, ttl_ms.
const luaApplyHeaders = `
local header_remaining = tonumber(ARGV[1])
local header_reset = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])

local current_remaining = tonumber(redis.call('GET', KEYS[1]))
local current_reset = tonumber(redis.call('GET', KEYS[2]))

if current_reset == nil or header_reset > current_reset + 1000 then
    redis.call('SET', KEYS[1], header_remaining, 'PX', ttl)
    redis.call('SET', KEYS[2], header_reset, 'PX', ttl)
elseif current_remaining == nil or header_remaining < current_remaining then
    redis.call('SET', KEYS[1], header_remaining, 'PX', ttl)
end
return 0
`

var (
	acquireScript = redis.NewScript(luaAcquire)
	headersScript = redis.NewScript(luaApplyHeaders)
)

// RateLimiter paces requests against a group's budget.
type RateLimiter struct {
	redis *redis.Client
}

// NewRateLimiter builds a limiter over the coordination instance.
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{redis: client}
}

func keyRemaining(group string) string { return "esi:tb:" + group + ":remaining" }
func keyResetAt(group string) string   { return "esi:tb:" + group + ":reset_at" }

// Acquire draws cost tokens, returning how long to wait when the bucket is dry.
// A zero duration means the caller may proceed.
func (r *RateLimiter) Acquire(ctx context.Context, g Group, cost int) (time.Duration, error) {
	res, err := acquireScript.Run(ctx, r.redis,
		[]string{keyRemaining(g.Name), keyResetAt(g.Name)},
		g.Limit, g.Window*1000, cost, time.Now().UnixMilli(),
	).Int64()
	if err != nil {
		return 0, err
	}
	return time.Duration(res) * time.Millisecond, nil
}

// ApplyHeaders merges what a response reported, for groups where ESI's own
// accounting is authoritative. A no-op for the hand-built groups: their headers
// do not exist, and pretending otherwise would overwrite a preset with noise.
func (r *RateLimiter) ApplyHeaders(ctx context.Context, g Group, remaining int, resetAfter time.Duration) error {
	if !g.HeaderAuthoritative {
		return nil
	}
	resetAt := time.Now().Add(resetAfter).UnixMilli()
	ttl := (resetAfter + 30*time.Second).Milliseconds()
	return headersScript.Run(ctx, r.redis,
		[]string{keyRemaining(g.Name), keyResetAt(g.Name)},
		remaining, resetAt, ttl,
	).Err()
}

// ApplyRemaining updates only the counter, for responses that report remaining
// without a reset.
//
// /killmails/{id}/{hash}/ does exactly that — group, limit, remaining and used,
// but no reset timestamp. Without this path the local bucket drains to zero with
// no feedback and then blocks every killmail until the window rolls over on its
// own.
func (r *RateLimiter) ApplyRemaining(ctx context.Context, g Group, remaining int) error {
	if !g.HeaderAuthoritative {
		return nil
	}
	ttl := time.Duration(g.Window)*time.Second + 30*time.Second
	return r.redis.Set(ctx, keyRemaining(g.Name), remaining, ttl).Err()
}

// BucketState is a group's live budget, for diagnostics.
type BucketState struct {
	Group     string    `json:"group"`
	Remaining int       `json:"remaining"`
	Limit     int       `json:"limit"`
	ResetAt   time.Time `json:"reset_at"`
	Seeded    bool      `json:"seeded"`
}

// Peek reads a group's bucket without touching it. Seeded is false when the
// bucket has never been used, which reads as "full" rather than "empty".
func (r *RateLimiter) Peek(ctx context.Context, g Group) (BucketState, error) {
	state := BucketState{Group: g.Name, Limit: g.Limit, Remaining: g.Limit}

	vals, err := r.redis.MGet(ctx, keyRemaining(g.Name), keyResetAt(g.Name)).Result()
	if err != nil {
		return state, err
	}
	if len(vals) != 2 || vals[0] == nil || vals[1] == nil {
		return state, nil
	}

	remaining, ok1 := parseRedisInt(vals[0])
	resetAt, ok2 := parseRedisInt(vals[1])
	if !ok1 || !ok2 {
		return state, nil
	}

	state.Seeded = true
	state.Remaining = int(remaining)
	state.ResetAt = time.UnixMilli(resetAt)
	return state, nil
}

func parseRedisInt(v any) (int64, bool) {
	s, ok := v.(string)
	if !ok {
		return 0, false
	}
	var n int64
	var neg bool
	for i, c := range s {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
	}
	if neg {
		n = -n
	}
	return n, true
}
