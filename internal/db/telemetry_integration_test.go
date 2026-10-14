package db

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"
)

func TestProductionPoolTelemetryExcludesBindValues(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	old := sentry.CurrentHub().Clone()
	transport := &sentry.MockTransport{}
	cfg := &config.Config{DatabaseURL: dsn, NodeEnv: "production", SentryDSN: "https://test@o1.ingest.sentry.io/42", SentryEnvironment: "test", SentryTracesSampleRate: 1, SentryMetrics: true, SentryLogLevel: "info"}
	runtime, err := telemetry.Init(telemetry.Options{Config: cfg, Version: "test", Service: "serve", Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		runtime.Close()
		sentry.CurrentHub().BindClient(old.Client())
		sentry.CurrentHub().Scope().Clear()
	})
	pool, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ctx, task := telemetry.StartTask(context.Background(), "http.server", "database fixture")
	var argument, literal string
	if err := pool.QueryRow(ctx, "SELECT $1::text, 'private-sql-literal'", "private-bind-value").Scan(&argument, &literal); err != nil {
		t.Fatal(err)
	}
	if argument != "private-bind-value" || literal != "private-sql-literal" {
		t.Fatal("query result changed")
	}
	task.End("completed", nil)
	runtime.Close()
	queries, waits := 0, 0
	for _, event := range transport.Events() {
		encoded, _ := json.Marshal(event)
		if strings.Contains(string(encoded), "private-") {
			t.Fatal("SQL telemetry leaked values")
		}
		for _, span := range event.Spans {
			switch span.Op {
			case "db.query":
				queries++
				if span.Description != "SELECT $1::text, ?" {
					t.Fatal(span.Description)
				}
			case "db.acquire":
				waits++
			}
		}
	}
	if queries != 1 || waits != 1 {
		t.Fatalf("queries=%d pool waits=%d", queries, waits)
	}
}
