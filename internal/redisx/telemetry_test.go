package redisx

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"
	"github.com/redis/go-redis/v9"
)

func TestRedisTelemetryPreservesMissAndExcludesKeys(t *testing.T) {
	old := sentry.CurrentHub().Clone()
	transport := &sentry.MockTransport{}
	runtime, err := telemetry.Init(telemetry.Options{Config: &config.Config{SentryDSN: "https://test@o1.ingest.sentry.io/42", SentryTracesSampleRate: 1, SentryMetrics: true, SentryLogLevel: "info"}, Version: "test", Service: "serve", Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		runtime.Close()
		sentry.CurrentHub().BindClient(old.Client())
		sentry.CurrentHub().Scope().Clear()
	})
	ctx, task := telemetry.StartTask(context.Background(), "http.server", "cache fixture")
	hook := telemetryHook{}.ProcessHook(func(_ context.Context, cmd redis.Cmder) error { cmd.SetErr(redis.Nil); return redis.Nil })
	if err := hook(ctx, redis.NewStringCmd(ctx, "get", "private-user-cache-key")); !errors.Is(err, redis.Nil) {
		t.Fatal("cache miss behavior changed")
	}
	task.End("completed", nil)
	runtime.Close()
	spans := 0
	for _, event := range transport.Events() {
		encoded, _ := json.Marshal(event)
		if strings.Contains(string(encoded), "private-user-cache-key") {
			t.Fatal("cache key leaked")
		}
		for _, span := range event.Spans {
			if span.Op == "db.redis" {
				spans++
				if span.Description != "get" || span.Status != sentry.SpanStatusOK {
					t.Fatal(span.Description, span.Status)
				}
			}
		}
	}
	if spans != 1 {
		t.Fatalf("cache spans=%d", spans)
	}
}
