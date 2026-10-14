package telemetry

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/rs/zerolog/log"
)

type requestStateKey struct{}
type requestState struct {
	captured atomic.Bool
	route    string
	surface  string
}

func requestStateFrom(ctx context.Context) *requestState {
	state, _ := ctx.Value(requestStateKey{}).(*requestState)
	return state
}

// SetRoute accepts router templates, never a query string or a concrete ID.
func SetRoute(ctx context.Context, method, pattern string) {
	state := requestStateFrom(ctx)
	if state == nil || state.surface == "frontend" {
		return
	}
	state.route = method + " " + pattern
	if transaction := sentry.TransactionFromContext(ctx); transaction != nil {
		transaction.Name = state.route
		transaction.Source = sentry.SourceRoute
	}
}

var dynamicSegment = regexp.MustCompile(`^(?:[0-9]+|[0-9a-fA-F-]{8,})$`)

func routeFallback(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return "/"
	}
	switch parts[0] {
	case "assets", "images":
		return "/" + parts[0] + "/*"
	case "market", "blog", "docs":
		if len(parts) > 1 {
			return "/" + parts[0] + "/{path}"
		}
	}
	for i, part := range parts {
		if dynamicSegment.MatchString(part) {
			parts[i] = "{id}"
		}
	}
	return "/" + strings.Join(parts, "/")
}

func quietHTTP(path string) bool {
	return path == "/health" || path == "/ready" || strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/images/")
}

func HTTP(surface string, next http.Handler) http.Handler {
	if !Enabled() {
		return next
	}
	wrapped := sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := &requestState{surface: surface}
		r = r.WithContext(context.WithValue(r.Context(), requestStateKey{}, state))
		ctx := r.Context()
		ctx = context.WithValue(ctx, captureStateKey{}, &state.captured)
		r = r.WithContext(ctx)
		transaction := sentry.TransactionFromContext(ctx)
		hubFor(ctx).Scope().SetTag("surface", surface)
		if quietHTTP(r.URL.Path) {
			transaction.Sampled = sentry.SampledFalse
		}
		started := time.Now()
		defer func() {
			if value := recover(); value != nil {
				Count(ctx, "http.requests", 1, Attributes("surface", surface, "status", "500", "route", routeFallback(r.URL.Path)))
				Duration(ctx, "http.duration", time.Since(started), Attributes("surface", surface, "status", "500"))
				transaction.Status = sentry.SpanStatusInternalError
				panic(value)
			}
		}()
		metrics := httpsnoop.CaptureMetrics(next, w, r)
		if state.route == "" {
			if r.Pattern != "" {
				state.route = r.Pattern
				if !strings.Contains(state.route, " ") {
					state.route = r.Method + " " + state.route
				}
			} else if metrics.Code != http.StatusNotFound {
				state.route = r.Method + " " + routeFallback(r.URL.Path)
			} else {
				state.route = r.Method + " /unmatched"
			}
		}
		transaction.Name = state.route
		transaction.Source = sentry.SourceRoute
		attrs := Attributes("surface", surface, "method", r.Method, "route", state.route, "status", strconv.Itoa(metrics.Code))
		Count(ctx, "http.requests", 1, attrs)
		Duration(ctx, "http.duration", metrics.Duration, attrs)
		Gauge(ctx, "http.response.size", float64(metrics.Written), attrs, WithUnit(sentry.UnitByte))
		if metrics.Code >= 500 && !state.captured.Load() && ctx.Err() == nil {
			Capture(ctx, errors.New("HTTP "+strconv.Itoa(metrics.Code)+" "+http.StatusText(metrics.Code)))
		}
		if !quietHTTP(r.URL.Path) || metrics.Code >= 500 {
			event := log.Info().Ctx(ctx)
			if metrics.Code >= 500 {
				event = log.Error().Ctx(ctx).Bool("sentry_handled", true)
			}
			event.Str("operation", "http.request").Str("surface", surface).
				Str("method", r.Method).Str("route", state.route).Int("status", metrics.Code).
				Dur("duration", metrics.Duration).Int64("response_bytes", metrics.Written).
				Str("cache_status", w.Header().Get("X-Cache")).Msg("HTTP request completed")
		}
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A WebSocket/SSE can live for days. Trace its individual operations,
		// rather than holding a server transaction and its spans for that long.
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || r.URL.Path == "/api/feed/stream" || r.URL.Path == "/__dev/reload" {
			next.ServeHTTP(w, r)
			return
		}
		wrapped.ServeHTTP(w, r)
	})
}

// Transport instruments outbound calls without forwarding cookies, bodies,
// auth headers, query strings, or trace headers to unrelated external services.
func Transport(service string, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return transport{service: service, base: base}
}

type transport struct {
	service string
	base    http.RoundTripper
}

func (t transport) RoundTrip(req *http.Request) (*http.Response, error) {
	path := routeFallback(req.URL.Path)
	if t.service == "objectstore" {
		path = "/{object}"
	}
	ctx, span := StartSpan(req.Context(), "http.client", req.Method+" "+path)
	started := time.Now()
	res, err := t.base.RoundTrip(req.WithContext(ctx))
	status := 0
	if res != nil {
		status = res.StatusCode
	}
	if span != nil {
		span.SetData("server.address", req.URL.Hostname())
		span.SetData("http.request.method", req.Method)
		span.SetData("http.response.status_code", status)
		if res != nil {
			span.Status = sentry.HTTPtoSpanStatus(status)
		}
		if err != nil {
			span.Status = sentry.SpanStatusInternalError
		}
	}
	attrs := Attributes("service", t.service, "method", req.Method, "status", strconv.Itoa(status))
	Count(ctx, "http.client.requests", 1, attrs)
	Duration(ctx, "http.client.ttfb", time.Since(started), attrs)
	finish := func(readErr error, bytes int64) {
		Duration(ctx, "http.client.duration", time.Since(started), attrs)
		Count(ctx, "http.client.response.bytes", bytes, attrs, WithUnit(sentry.UnitByte))
		if readErr != nil {
			Count(ctx, "http.client.failures", 1, attrs)
			if span != nil {
				span.Status = sentry.SpanStatusInternalError
			}
		}
		if span != nil {
			span.Finish()
		}
	}
	if err != nil || res == nil || res.Body == nil {
		finish(err, 0)
	} else {
		res.Body = &observedBody{ReadCloser: res.Body, finish: finish}
	}
	return res, err
}

type observedBody struct {
	io.ReadCloser
	finish func(error, int64)
	once   sync.Once
	bytes  atomic.Int64
}

func (body *observedBody) Read(data []byte) (int, error) {
	n, err := body.ReadCloser.Read(data)
	body.bytes.Add(int64(n))
	if err != nil {
		observedErr := err
		if errors.Is(err, io.EOF) {
			observedErr = nil
		}
		body.once.Do(func() { body.finish(observedErr, body.bytes.Load()) })
	}
	return n, err
}
func (body *observedBody) Close() error {
	err := body.ReadCloser.Close()
	body.once.Do(func() { body.finish(err, body.bytes.Load()) })
	return err
}
