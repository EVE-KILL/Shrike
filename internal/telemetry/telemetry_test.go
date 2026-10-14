package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

func testRuntime(t *testing.T) (*Runtime, *sentry.MockTransport) {
	t.Helper()
	old := sentry.CurrentHub().Clone()
	transport := &sentry.MockTransport{}
	r, err := Init(Options{Config: &config.Config{
		SentryDSN: "https://test@o1.ingest.sentry.io/42", SentryEnvironment: "test", SentryTracesSampleRate: 1,
		SentryLogs: true, SentryMetrics: true, SentryLogLevel: "info", SentryLogSampleRate: 1,
		DatabaseURL: "postgres://tester:db-password-fixture@localhost/test", EVEClientSecret: "sso-secret-fixture",
	}, Version: "test", Commit: "test-commit", Service: "test", Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(); sentry.CurrentHub().BindClient(old.Client()); sentry.CurrentHub().Scope().Clear() })
	return r, transport
}

func TestHTTPErrorPrivacyAndRouteGrouping(t *testing.T) {
	r, transport := testRuntime(t)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /character/{id}", func(w http.ResponseWriter, req *http.Request) {
		ctx, child := StartSpan(req.Context(), "db.query", NormalizeSQL("SELECT * FROM users WHERE password = 'private-bind-value' AND id = $1 -- private-comment"))
		FinishSpan(child, errors.New("query failed"))
		Capture(ctx, errors.New("connect postgres://user:db-password-fixture@localhost/test; Bearer private-token-value; sso-secret-fixture"))
		w.WriteHeader(http.StatusInternalServerError)
	})
	req := httptest.NewRequest("GET", "https://eve-kill.com/character/123?code=private-oauth-code", strings.NewReader("private-request-body"))
	req.Header.Set("Cookie", "evelogin=private-cookie")
	req.Header.Set("Authorization", "Bearer private-header-token")
	w := httptest.NewRecorder()
	HTTP("frontend", mux).ServeHTTP(w, req)
	if w.Code != 500 {
		t.Fatal(w.Code)
	}
	r.Close()
	errorEvents, transactions := 0, 0
	for _, event := range transport.Events() {
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"db-password-fixture", "private-token-value", "sso-secret-fixture", "private-oauth-code", "private-request-body", "private-cookie", "private-header-token", "private-bind-value", "private-comment"} {
			if strings.Contains(string(data), secret) {
				t.Fatalf("leaked %s in %s", secret, data)
			}
		}
		if event.Type == "transaction" {
			transactions++
			if event.Transaction != "GET /character/{id}" {
				t.Fatalf("ungrouped transaction: %q", event.Transaction)
			}
			if len(event.Spans) != 1 || event.Spans[0].Op != "db.query" {
				t.Fatalf("missing SQL child: %+v", event.Spans)
			}
		} else if len(event.Exception) > 0 {
			errorEvents++
			if len(event.Exception[0].Stacktrace.Frames) == 0 {
				t.Fatal("missing exception stack")
			}
		}
	}
	if errorEvents != 1 || transactions != 1 {
		t.Fatalf("events=%d transactions=%d", errorEvents, transactions)
	}
}

func TestLogAndMetricTraceCorrelation(t *testing.T) {
	r, transport := testRuntime(t)
	ctx, task := StartTask(context.Background(), "queue.job", "killmail", "queue", "killmails")
	logger := zerolog.New(LogWriter{}).Hook(LogHook{}).With().Timestamp().Logger()
	logger.Info().Ctx(ctx).Str("queue", "killmails").Str("access_token", "private-token").Msg("job completed")
	Duration(ctx, "job.duration", 10*time.Millisecond, Attributes("queue", "killmails"))
	task.End("completed", nil)
	r.Close()
	logs, distributions := 0, 0
	for _, event := range transport.Events() {
		for _, entry := range event.Logs {
			logs++
			if entry.TraceID != task.span.TraceID || entry.SpanID != task.span.SpanID {
				t.Fatalf("unlinked log %+v", entry)
			}
			if _, exists := entry.Attributes["access_token"]; exists {
				t.Fatal("log leaked access_token")
			}
			if entry.Body != "job completed" {
				t.Fatal(entry.Body)
			}
		}
		for _, metric := range event.Metrics {
			if metric.Name == "job.duration" {
				distributions++
				if metric.TraceID != task.span.TraceID {
					t.Fatal("unlinked duration")
				}
			}
		}
	}
	if logs != 1 || distributions != 1 {
		t.Fatalf("logs=%d distributions=%d", logs, distributions)
	}
}

func TestConcurrentTaskIsolation(t *testing.T) {
	r, transport := testRuntime(t)
	var wg sync.WaitGroup
	for _, name := range []string{"first", "second"} {
		wg.Go(func() {
			ctx, task := StartTask(context.Background(), "queue.job", name, "queue", name)
			Capture(ctx, errors.New(name))
			task.End("failed", errors.New(name))
		})
	}
	wg.Wait()
	r.Close()
	seen := map[string]bool{}
	for _, event := range transport.Events() {
		if len(event.Exception) > 0 {
			name := event.Exception[0].Value
			if event.Tags["queue"] != name {
				t.Fatalf("scope leaked: %+v", event.Tags)
			}
			seen[name] = true
		}
	}
	if len(seen) != 2 {
		t.Fatal(seen)
	}
}

func TestMetricAggregationIsExactAndBounded(t *testing.T) {
	r, transport := testRuntime(t)
	ctx := context.Background()
	for range 10000 {
		Count(ctx, "db.query.count", 1, Attributes("pool", "primary"))
		Duration(ctx, "db.query.duration", time.Millisecond, Attributes("pool", "primary"))
	}
	Gauge(ctx, "queue.depth", 30)
	Gauge(ctx, "queue.depth", 12)
	r.Close()
	counters, durations, gauges := 0, 0, 0
	for _, event := range transport.Events() {
		for _, metric := range event.Metrics {
			switch metric.Name {
			case "db.query.count":
				counters++
				data, _ := json.Marshal(metric.Value)
				if string(data) != "10000" {
					t.Fatalf("counter=%s", data)
				}
			case "db.query.duration":
				durations++
			case "queue.depth":
				gauges++
				data, _ := json.Marshal(metric.Value)
				if string(data) != "12" {
					t.Fatalf("gauge=%s", data)
				}
			}
		}
	}
	if counters != 1 || durations != maxDurationSamples || gauges != 1 {
		t.Fatalf("counters=%d durations=%d gauges=%d", counters, durations, gauges)
	}
}

func TestHTTPPanicIsReportedAndRepanics(t *testing.T) {
	r, transport := testRuntime(t)
	handler := HTTP("frontend", http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("fixture panic") }))
	func() {
		defer func() {
			if value := recover(); value != "fixture panic" {
				t.Fatalf("panic=%v", value)
			}
		}()
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/about", nil))
	}()
	r.Close()
	count := 0
	for _, event := range transport.Events() {
		if event.Message == "fixture panic" || len(event.Exception) > 0 && event.Exception[0].Value == "fixture panic" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("panic events=%d", count)
	}
}

func TestNormalizeSQL(t *testing.T) {
	for _, tc := range []struct{ input, expected string }{
		{"SELECT name FROM inv_types WHERE type_id=$1 AND published=true", "SELECT name FROM inv_types WHERE type_id=$1 AND published=true"},
		{"SELECT 'secret''value', 95741, $tag$secret$tag$ -- private comment", "SELECT ?, ?, ?"},
		{"SELECT /* outer /* private */ comment */ $$private$$", "SELECT ?"},
		{"SELECT $tag1$private$tag1$", "SELECT ?"},
		{"SELECT E'private" + string([]byte{92}) + "'value'", "SELECT E?"},
		{"SELECT 'unterminated-private", "SELECT ?"},
	} {
		if got := NormalizeSQL(tc.input); got != tc.expected {
			t.Errorf("got %q want %q", got, tc.expected)
		}
	}
}

func TestDisabledTelemetryPreservesHandlers(t *testing.T) {
	if Enabled() {
		t.Fatal("test runtime not cleaned up")
	}
	r, err := Init(Options{Config: &config.Config{}})
	if r != nil || err != nil {
		t.Fatal(r, err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })
	w := httptest.NewRecorder()
	HTTP("frontend", handler).ServeHTTP(w, httptest.NewRequest("GET", "/about", nil))
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	ctx, span := StartSpan(context.Background(), "db.query", "SELECT 1")
	if span != nil {
		t.Fatal("orphan transaction")
	}
	Count(ctx, "test", 1)
	Duration(ctx, "test", time.Second)
	Gauge(ctx, "test", 2)
	r.Close()
}

func TestOutboundTransportPreservesHeadersAndBody(t *testing.T) {
	r, sentryTransport := testRuntime(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer private-upstream-token" {
			t.Error("authorization changed")
		}
		if req.Header.Get(sentry.SentryTraceHeader) != "" {
			t.Error("trace headers sent to external service")
		}
		data, _ := io.ReadAll(req.Body)
		if string(data) != "private-outbound-payload" {
			t.Error("outbound body changed")
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte("upstream response"))
	}))
	defer server.Close()
	ctx, task := StartTask(context.Background(), "queue.job", "upstream")
	req, err := http.NewRequestWithContext(ctx, "POST", server.URL+"/v2/oauth/token?code=private-query", strings.NewReader("private-outbound-payload"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer private-upstream-token")
	response, err := (&http.Client{Transport: Transport("ccp.sso", nil)}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil || string(data) != "upstream response" || response.StatusCode != 201 {
		t.Fatal("response changed")
	}
	task.End("completed", nil)
	r.Close()
	spans := 0
	for _, event := range sentryTransport.Events() {
		encoded, _ := json.Marshal(event)
		if strings.Contains(string(encoded), "private-") {
			t.Fatal("outbound telemetry leaked input")
		}
		for _, span := range event.Spans {
			if span.Op == "http.client" {
				spans++
				if span.Status != sentry.SpanStatusOK {
					t.Fatal(span.Status)
				}
			}
		}
	}
	if spans != 1 {
		t.Fatalf("client spans=%d", spans)
	}
}
