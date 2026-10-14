package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/retry"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog"
)

func TestJobPropagationAndFailure(t *testing.T) {
	old := sentry.CurrentHub().Clone()
	transport := &sentry.MockTransport{}
	runtime, err := telemetry.Init(telemetry.Options{Config: &config.Config{
		SentryDSN: "https://test@o1.ingest.sentry.io/42", SentryEnvironment: "test", SentryTracesSampleRate: 1,
		SentryLogLevel: "info", SentryMetrics: true,
	}, Version: "test", Service: "queues", Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		runtime.Close()
		sentry.CurrentHub().BindClient(old.Client())
		sentry.CurrentHub().Scope().Clear()
	})
	ctx, task := telemetry.StartTask(context.Background(), "http.server", "origin")
	origin := sentry.SpanFromContext(ctx)
	params := []*rivertype.JobInsertParams{{Metadata: []byte(`{"custom":true}`), EncodedArgs: []byte(`{"killmail_hash":"private-payload"}`)}}
	middleware := newJobLogMiddleware(zerolog.Nop())
	_, err = middleware.InsertMany(ctx, params, func(context.Context) ([]*rivertype.JobInsertResult, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(params[0].Metadata, &metadata); err != nil {
		t.Fatal(err)
	}
	if string(metadata["custom"]) != "true" {
		t.Fatal("existing metadata was lost")
	}
	var carrier telemetry.TraceCarrier
	if err := json.Unmarshal(metadata["_sentry"], &carrier); err != nil {
		t.Fatal(err)
	}
	if carrier.Trace == "" {
		t.Fatal("trace metadata missing")
	}
	task.End("completed", nil)
	job := testJobRow()
	job.Metadata = params[0].Metadata
	boom := errors.New("fixture job failure")
	err = middleware.Work(context.Background(), job, func(ctx context.Context) error {
		if span := sentry.SpanFromContext(ctx); span == nil || span.TraceID != origin.TraceID {
			t.Fatal("job did not continue the originating trace")
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatal("instrumentation changed the returned error")
	}
	for _, queue := range []string{"killmail", CronQueue} {
		job.Queue = queue
		err = middleware.Work(t.Context(), job, func(context.Context) error {
			return fmt.Errorf("fetch deferred: %w", &retry.Deferred{Operation: "esi", Reason: "admission", After: time.Minute})
		})
		if snooze, ok := errors.AsType[*river.JobSnoozeError](err); !ok || snooze.Duration < time.Minute {
			t.Fatalf("expected snooze, got %v", err)
		}
	}
	job.Queue = "killmail"
	runtime.Close()
	failures := 0
	for _, event := range transport.Events() {
		if len(event.Exception) > 0 {
			failures++
			if event.Tags["queue"] != job.Queue {
				t.Fatal("queue context missing")
			}
		}
	}
	if failures != 1 {
		t.Fatalf("failure events=%d", failures)
	}
}
