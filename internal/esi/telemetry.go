package esi

import (
	"context"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"
)

func beginRequest(ctx context.Context, method, group string, authenticated bool) (context.Context, func(raw, error)) {
	ctx, span := telemetry.StartSpan(ctx, "ccp.esi", method+" "+group)
	started := time.Now()
	return ctx, func(result raw, err error) {
		attrs := telemetry.Attributes("group", group, "method", method, "status", strconv.Itoa(result.Status),
			"cached", strconv.FormatBool(result.Cached), "authenticated", strconv.FormatBool(authenticated))
		telemetry.Count(ctx, "esi.requests", 1, attrs)
		telemetry.Duration(ctx, "esi.duration", time.Since(started), attrs)
		if result.RetryAfter > 0 {
			telemetry.Gauge(ctx, "esi.retry_after", float64(result.RetryAfter), attrs, telemetry.WithUnit(sentry.UnitSecond))
		}
		if span != nil {
			span.SetData("http.response.status_code", result.Status)
			span.SetData("cache.hit", result.Cached)
			span.SetData("esi.group", group)
			span.SetData("esi.authenticated", authenticated)
			if err != nil {
				span.Status = sentry.SpanStatusInternalError
			} else if result.Status == 0 {
				span.Status = sentry.SpanStatusResourceExhausted
			} else {
				span.Status = sentry.HTTPtoSpanStatus(result.Status)
			}
			span.Finish()
		}
	}
}

func waitMeasured(ctx context.Context, reason, group string, wait time.Duration) error {
	ctx, span := telemetry.StartSpan(ctx, "ccp.esi.wait", reason)
	started := time.Now()
	err := sleep(ctx, wait)
	telemetry.Count(ctx, "esi.waits", 1, telemetry.Attributes("reason", reason, "group", group))
	telemetry.Duration(ctx, "esi.wait", time.Since(started), telemetry.Attributes("reason", reason, "group", group))
	telemetry.FinishSpan(span, err)
	return err
}
