package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// Database is the read-only subset of pgxpool.Pool used by API handlers.
// Keeping it as an interface makes the HTTP contract testable without a live
// Postgres server; *pgxpool.Pool satisfies it directly in production.
type Database interface {
	Ping(context.Context) error
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type legacyRequest struct {
	Huma  huma.Context
	Query url.Values
	Body  io.Reader
}

func (r *legacyRequest) Param(name string) string {
	return r.Huma.Param(name)
}

type legacyPayload struct {
	Status      int
	ContentType string
	Body        any
	RawBody     []byte
	Headers     http.Header
}

type legacyHandler func(context.Context, *legacyRequest) (legacyPayload, error)

type legacyAPIError struct {
	Status  int
	Message string
}

func (e *legacyAPIError) Error() string {
	return e.Message
}

func apiError(status int, message string) error {
	return &legacyAPIError{Status: status, Message: message}
}

// registerLegacy mounts a compatibility handler through Huma's adapter while
// retaining control of validation and serialization. Huma's default request
// validation errors use RFC 9457 problem details; the API being replaced has
// always returned {"error":"..."} and existing consumers depend on that.
func registerLegacy(a huma.API, op huma.Operation, handler legacyHandler) {
	if op.DefaultStatus == 0 {
		op.DefaultStatus = http.StatusOK
	}
	if op.Extensions == nil {
		op.Extensions = make(map[string]any)
	}
	if _, exists := op.Extensions["x-audience"]; !exists {
		op.Extensions["x-audience"] = operationAudience(op)
	}
	op.SkipValidateParams = true
	op.SkipValidateBody = true
	if op.Responses == nil {
		op.Responses = legacyResponses(op.DefaultStatus)
	}

	a.OpenAPI().AddOperation(&op)
	a.Adapter().Handle(&op, func(hctx huma.Context) {
		requestURL := hctx.URL()
		req := &legacyRequest{
			Huma:  hctx,
			Query: requestURL.Query(),
			Body:  hctx.BodyReader(),
		}
		payload, err := handler(hctx.Context(), req)
		if err != nil {
			writeLegacyError(hctx, err)
			return
		}
		writeLegacyPayload(hctx, payload)
	})
}

func operationAudience(op huma.Operation) string {
	for _, tag := range op.Tags {
		switch strings.ToLower(tag) {
		case "admin", "administration":
			return "admin"
		case "auth", "account":
			return "account"
		}
	}
	for _, requirement := range op.Security {
		if len(requirement) == 0 {
			// An empty alternative means authentication is optional. Keep the
			// operation in the public catalogue unless its domain tags above
			// explicitly identify it as an account operation.
			return "public"
		}
	}
	if len(op.Security) > 0 {
		return "account"
	}
	return "public"
}

func legacyRequestHost(req *legacyRequest) string {
	if req != nil && req.Huma != nil {
		// Host is a dedicated net/http request field rather than an ordinary
		// header. Huma exposes it directly; Header("Host") silently returns
		// empty with the Chi adapter. Prefer the authoritative request host so
		// a client-supplied forwarding header cannot redefine tenant scope.
		if host := req.Huma.Host(); host != "" {
			return host
		}
		// Retain a compatibility fallback for unusual embeddings that do not
		// populate Host.
		host := firstForwarded(req.Huma.Header("X-Forwarded-Host"))
		if host == "" {
			host = req.Huma.Header("Host")
		}
		return host
	}
	return ""
}

func legacyResponses(status int) map[string]*huma.Response {
	return map[string]*huma.Response{
		strconv.Itoa(status): {
			Description: http.StatusText(status),
			Content: map[string]*huma.MediaType{
				"application/json": {
					Schema: &huma.Schema{
						Type:                 huma.TypeObject,
						AdditionalProperties: true,
					},
				},
			},
		},
	}
}

func writeLegacyPayload(ctx huma.Context, payload legacyPayload) {
	status := payload.Status
	if status == 0 {
		status = http.StatusOK
	}
	contentType := payload.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	for name, values := range payload.Headers {
		for i, value := range values {
			if i == 0 {
				ctx.SetHeader(name, value)
			} else {
				ctx.AppendHeader(name, value)
			}
		}
	}
	ctx.SetHeader("Content-Type", contentType)
	ctx.SetStatus(status)

	if payload.RawBody != nil {
		_, _ = ctx.BodyWriter().Write(payload.RawBody)
		return
	}
	body, err := json.Marshal(normalizeJSON(payload.Body))
	if err != nil {
		writeLegacyError(ctx, fmt.Errorf("marshal response: %w", err))
		return
	}
	_, _ = ctx.BodyWriter().Write(body)
}

func writeLegacyError(ctx huma.Context, err error) {
	status := http.StatusInternalServerError
	message := "Internal server error"
	var apiErr *legacyAPIError
	if errors.As(err, &apiErr) {
		status = apiErr.Status
		message = apiErr.Message
	} else {
		log.Error().Err(err).Msg("API request failed")
	}
	body, marshalErr := json.Marshal(map[string]string{"error": message})
	if marshalErr != nil {
		body = []byte(`{"error":"Internal server error"}`)
		status = http.StatusInternalServerError
	}
	ctx.SetHeader("Content-Type", "application/json")
	ctx.SetStatus(status)
	_, _ = ctx.BodyWriter().Write(body)
}

// crossOriginAPI applies the API-host transport policy to every response,
// independent of whether it came from a handler, Redis, or a conditional
// request. Payload compatibility does not require preserving the legacy
// server's cache-dependent CORS quirks.
func crossOriginAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set(
			"Access-Control-Allow-Methods",
			"GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS",
		)
		w.Header().Set(
			"Access-Control-Allow-Headers",
			"Content-Type, If-None-Match, Last-Event-ID",
		)
		w.Header().Set("Access-Control-Expose-Headers", "ETag, Link, X-Cache")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func jsonPayload(body any) legacyPayload {
	return legacyPayload{Body: body}
}

func parseID(raw string) (int64, error) {
	// JavaScript's Number(), used by the old parseId(), accepts surrounding
	// whitespace but rejects NaN and zero. API IDs are integral Postgres
	// values, so reject fractions before they become a database type error.
	raw = strings.TrimSpace(raw)
	number, err := strconv.ParseFloat(raw, 64)
	if err != nil || number == 0 || math.IsNaN(number) ||
		math.IsInf(number, 0) || math.Trunc(number) != number ||
		number < math.MinInt64 || number > math.MaxInt64 {
		return 0, apiError(http.StatusBadRequest, "Invalid ID")
	}
	return int64(number), nil
}

const pgInt4Max = int64(math.MaxInt32)

type pagination struct {
	Limit  int
	After  *int64
	Before *int64
}

func parsePagination(query url.Values) pagination {
	limit := numberOr(query.Get("limit"), 50)
	if limit < 10 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	after := positiveInt4(query.Get("after"))
	before := positiveInt4(query.Get("before"))
	if before != nil {
		after = nil
	}
	return pagination{Limit: int(limit), After: after, Before: before}
}

func numberOr(raw string, fallback float64) float64 {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || n == 0 || math.IsNaN(n) {
		return fallback
	}
	return n
}

func positiveInt4(raw string) *int64 {
	n := numberOr(raw, 0)
	if n <= 0 || math.IsInf(n, 0) || math.IsNaN(n) {
		return nil
	}
	value := int64(math.Floor(n))
	if value > pgInt4Max {
		value = pgInt4Max
	}
	return &value
}

func queryMaps(ctx context.Context, db Database, query string, args ...any) ([]map[string]any, error) {
	if db == nil {
		return nil, fmt.Errorf("API database is not configured")
	}
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToMap)
}

func queryMap(ctx context.Context, db Database, query string, args ...any) (map[string]any, error) {
	rows, err := queryMaps(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

type databaseQuery struct {
	SQL  string
	Args []any
}

func queryMapsConcurrent(
	ctx context.Context,
	db Database,
	queries ...databaseQuery,
) ([][]map[string]any, error) {
	result := make([][]map[string]any, len(queries))
	group, groupCtx := errgroup.WithContext(ctx)
	for i, query := range queries {
		i, query := i, query
		group.Go(func() (err error) {
			result[i], err = queryMaps(groupCtx, db, query.SQL, query.Args...)
			return
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return result, nil
}

const javascriptTimestampLayout = "2006-01-02T15:04:05.000Z"

func javascriptTimestamp(value time.Time) string {
	return value.UTC().Format(javascriptTimestampLayout)
}

// normalizeJSON reproduces the observable parts of Bun/Drizzle JSON
// serialization. In particular, JS Date#toJSON always uses UTC and exactly
// three fractional digits, while encoding/json's time.Time output may retain
// a source offset and a different fractional precision.
func normalizeJSON(value any) any {
	switch v := value.(type) {
	case time.Time:
		return javascriptTimestamp(v)
	case *time.Time:
		if v == nil {
			return nil
		}
		return javascriptTimestamp(*v)
	case map[string]any:
		for key, item := range v {
			v[key] = normalizeJSON(item)
		}
		return v
	case []map[string]any:
		for i := range v {
			for key, item := range v[i] {
				v[i][key] = normalizeJSON(item)
			}
		}
		return v
	case []any:
		for i := range v {
			v[i] = normalizeJSON(v[i])
		}
		return v
	default:
		return value
	}
}

func int64Value(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

func float64Value(value any) (float64, bool) {
	switch v := value.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		n, ok := int64Value(value)
		return float64(n), ok
	}
}

func boolValue(value any) (bool, bool) {
	v, ok := value.(bool)
	return v, ok
}

func stringValue(value any) (string, bool) {
	v, ok := value.(string)
	return v, ok
}

func int32Slice(values ...any) []int32 {
	seen := map[int32]struct{}{}
	out := make([]int32, 0, len(values))
	for _, value := range values {
		n, ok := int64Value(value)
		if !ok || n < math.MinInt32 || n > math.MaxInt32 {
			continue
		}
		v := int32(n)
		if _, exists := seen[v]; exists {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
