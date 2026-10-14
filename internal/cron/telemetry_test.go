package cron

import (
	"context"
	"errors"
	"testing"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"
)

func TestManualCronReportsFailureOnlyAtCommandBoundary(t *testing.T) {
	for _, panics := range []bool{false, true} {
		t.Run(map[bool]string{false: "error", true: "panic"}[panics], func(t *testing.T) {
			old := sentry.CurrentHub().Clone()
			transport := &sentry.MockTransport{}
			runtime, err := telemetry.Init(telemetry.Options{Config: &config.Config{
				SentryDSN: "https://test@o1.ingest.sentry.io/42", SentryEnvironment: "test", SentryTracesSampleRate: 1,
				SentryLogLevel: "info",
			}, Transport: transport})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				runtime.Close()
				sentry.CurrentHub().BindClient(old.Client())
				sentry.CurrentHub().Scope().Clear()
			})
			boom := errors.New("manual cron failure")
			registry := NewRegistry()
			registry.MustRegister("insurance", func(context.Context) (string, error) {
				if panics {
					panic(boom)
				}
				return "", boom
			})
			func() {
				defer func() {
					value := recover()
					if panics && value != boom || !panics && value != nil {
						t.Fatalf("unexpected panic: %v", value)
					}
				}()
				defer telemetry.RecoverAndRepanic(context.Background())
				run, runErr := RunOnce(context.Background(), registry, "insurance")
				if !errors.Is(runErr, boom) {
					t.Fatalf("returned error=%v", runErr)
				}
				telemetry.Capture(run.Context, runErr)
			}()
			runtime.Close()
			failures, transactions := 0, 0
			for _, event := range transport.Events() {
				if len(event.Exception) > 0 {
					failures++
				}
				if event.Type == "transaction" && event.Transaction == "insurance" {
					transactions++
				}
			}
			if failures != 1 || transactions != 1 {
				t.Fatalf("failures=%d transactions=%d", failures, transactions)
			}
		})
	}
}
