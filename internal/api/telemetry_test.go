package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
)

func TestTelemetryUsesTemplatesForHumaValidationAndCachedResponses(t *testing.T) {
	old := sentry.CurrentHub().Clone()
	transport := &sentry.MockTransport{}
	runtime, err := telemetry.Init(telemetry.Options{Config: &config.Config{
		SentryDSN: "https://test@o1.ingest.sentry.io/42", SentryEnvironment: "test", SentryTracesSampleRate: 1,
		SentryLogLevel: "info", SentryMetrics: true,
	}, Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		runtime.Close()
		sentry.CurrentHub().BindClient(old.Client())
		sentry.CurrentHub().Scope().Clear()
	})
	router := chi.NewRouter()
	a := humachi.New(router, huma.DefaultConfig("test", "test"))
	type input struct {
		Slug string `path:"slug" maxLength:"16"`
	}
	type output struct {
		Body string
	}
	for _, path := range []string{"/coalitions/{slug}", "/auth/characters/{slug}", "/images/types/{slug}"} {
		huma.Register(a, huma.Operation{OperationID: path, Method: "GET", Path: path}, func(context.Context, *input) (*output, error) {
			return &output{Body: "ok"}, nil
		})
	}
	for _, tc := range []struct {
		path   string
		cached bool
		status int
	}{
		{"/api/coalitions/private-slug", false, 200},
		{"/api/coalitions/another-slug", true, 200},
		{"/api/coalitions/private-slug-too-long", false, 422},
		{"/auth/characters/private-slug", false, 200},
		{"/images/types/private-slug", true, 200},
		{"/api/private-unmatched-slug", false, 404},
	} {
		next := http.Handler(router)
		if tc.cached {
			// A response-cache hit returns before Huma or chi serves the request.
			next = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
		}
		handler := telemetry.HTTP("api", sitePaths(routeTelemetry(router, next)))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", tc.path, nil))
		if response.Code != tc.status {
			t.Fatalf("%s: status=%d body=%s", tc.path, response.Code, response.Body.String())
		}
	}
	runtime.Close()
	names := map[string]int{}
	for _, event := range transport.Events() {
		if event.Type == "transaction" {
			names[event.Transaction]++
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), "private-slug") || strings.Contains(string(encoded), "another-slug") || strings.Contains(string(encoded), "private-unmatched-slug") {
			t.Fatal("concrete path value leaked into telemetry")
		}
	}
	if names["GET /api/coalitions/{slug}"] != 3 || names["GET /auth/characters/{slug}"] != 1 || names["GET /images/types/{slug}"] != 0 {
		t.Fatalf("transaction names=%v", names)
	}
}
