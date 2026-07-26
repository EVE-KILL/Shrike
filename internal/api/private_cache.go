package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// privateJSONCache gives selected anonymous frontend reads the same shared
// Valkey cache used by the public API. It is opt-in per handler: session,
// custom-domain, and authenticated responses must never accidentally become
// shared merely because they use GET.
func privateJSONCache(
	opts Options,
	ttl time.Duration,
	cacheControl string,
	next legacyHandler,
) legacyHandler {
	return privateJSONCacheBy(opts, ttl, cacheControl, nil, next)
}

func privateJSONCacheBy(
	opts Options,
	ttl time.Duration,
	cacheControl string,
	keyFor func(*legacyRequest) string,
	next legacyHandler,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		url := req.Huma.URL()
		keyRequest := &http.Request{URL: &url}
		key := "web:" + cacheKey(opts.Commit, keyRequest)
		if keyFor != nil {
			build := opts.Commit
			if build == "" {
				build = "dev"
			}
			key = "shrike:web-api:" + build + ":" + keyFor(req)
		}
		if entry, ok := cacheLoad(ctx, opts.Cache, key); ok {
			headers := make(http.Header)
			headers.Set("X-Cache", "HIT")
			if cacheControl != "" {
				headers.Set("Cache-Control", cacheControl)
			}
			return legacyPayload{
				RawBody:     entry.Body,
				ContentType: entry.ContentType,
				Headers:     headers,
			}, nil
		}

		payload, err := next(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		status := payload.Status
		if status == 0 {
			status = http.StatusOK
		}
		if status != http.StatusOK {
			return payload, nil
		}

		body := payload.RawBody
		if body == nil {
			body, err = json.Marshal(normalizeJSON(payload.Body))
			if err != nil {
				return legacyPayload{}, err
			}
		}
		contentType := payload.ContentType
		if contentType == "" {
			contentType = "application/json"
		}
		cacheStore(context.WithoutCancel(ctx), opts.Cache, key, cachedResponse{
			ContentType: contentType,
			Body:        body,
		}, ttl)

		if payload.Headers == nil {
			payload.Headers = make(http.Header)
		}
		payload.Headers.Set("X-Cache", "MISS")
		if cacheControl != "" {
			payload.Headers.Set("Cache-Control", cacheControl)
		}
		payload.RawBody = body
		payload.Body = nil
		payload.ContentType = contentType
		return payload, nil
	}
}
