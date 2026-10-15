package queue

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/retry"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

func TestUpstreamRetriesKeepTheirBudgetAndReportTerminalOrUnexpectedErrors(t *testing.T) {
	old := sentry.CurrentHub().Clone()
	transport := &sentry.MockTransport{}
	runtime, err := telemetry.Init(telemetry.Options{Config: &config.Config{
		SentryDSN: "https://test@o1.ingest.sentry.io/42", SentryEnvironment: "test",
		SentryTracesSampleRate: 1, SentryLogLevel: "info",
	}, Version: "test", Service: "queues", Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		runtime.Close()
		sentry.CurrentHub().BindClient(old.Client())
		sentry.CurrentHub().Scope().Clear()
	})
	middleware := newJobLogMiddleware(zerolog.Nop())
	upstream := retry.Upstream(502, errors.New("ESI returned 502"))
	timeout := &url.Error{Op: "Get", URL: "https://esi.evetech.net/", Err: context.DeadlineExceeded}
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	expected := 0
	for _, tc := range []struct {
		name     string
		ctx      context.Context
		err      error
		terminal bool
		capture  bool
	}{
		{"HTTP retry", t.Context(), upstream, false, false},
		{"transport retry", t.Context(), timeout, false, false},
		{"unexpected first failure", t.Context(), errors.New("database failed"), false, true},
		{"mixed independent failure", t.Context(), errors.Join(upstream, errors.New("database failed")), false, true},
		{"exhausted upstream", t.Context(), upstream, true, true},
		{"worker's own deadline", expired, timeout, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			job := testJobRow()
			job.Attempt, job.MaxAttempts = 1, 3
			if tc.terminal {
				job.Attempt = job.MaxAttempts
			}
			attempt := job.Attempt
			got := middleware.Work(tc.ctx, job, func(context.Context) error { return tc.err })
			if got != tc.err || job.Attempt != attempt {
				t.Fatal("reporting changed the returned failure or retry budget")
			}
			if tc.capture {
				expected++
			}
			count := 0
			for _, event := range transport.Events() {
				if len(event.Exception) > 0 {
					count++
				}
			}
			if count != expected {
				t.Fatalf("exception events=%d, want %d", count, expected)
			}
		})
	}
}
