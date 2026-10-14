package images

import (
	"bytes"
	"image"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestUnavailableRenderFallsBackAndRechecksAfterExpiry(t *testing.T) {
	for _, status := range []int{400, 404} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			icon := solidPNG(t, 64, 64, colorValue(30, 100, 180))
			render := solidPNG(t, 512, 512, colorValue(100, 180, 30))
			available := false
			renderCalls, iconCalls := 0, 0
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/types/24497/render" {
					renderCalls++
					if available {
						_, _ = w.Write(render)
					} else {
						w.WriteHeader(status)
					}
				} else {
					iconCalls++
					_, _ = w.Write(icon)
				}
			}))
			defer origin.Close()
			now := time.Now()
			store := newMemoryStore()
			service := New(Options{Store: store, UpstreamURL: origin.URL, Now: func() time.Time { return now }})
			for _, size := range []int{32, 64} {
				result, err := service.Type(t.Context(), 24497, "overlayrender", size, "webp")
				if err != nil {
					t.Fatal(err)
				}
				cfg, format, err := image.DecodeConfig(bytes.NewReader(result.Body))
				if err != nil || format != "webp" || cfg.Width != size || cfg.Height != size {
					t.Fatalf("fallback = %s %dx%d, err=%v", format, cfg.Width, cfg.Height, err)
				}
			}
			if renderCalls != 1 || iconCalls != 1 {
				t.Fatalf("render calls=%d icon calls=%d", renderCalls, iconCalls)
			}
			now = now.Add(25 * time.Hour)
			available = true
			result, err := service.Type(t.Context(), 24497, "render", 0, "")
			if err != nil || !bytes.Equal(result.Body, render) || renderCalls != 2 {
				t.Fatalf("render did not recover after expiry: calls=%d err=%v", renderCalls, err)
			}
		})
	}
}

func TestMissingIconStillReturnsNotFound(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/types/24497/render" {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer origin.Close()
	service := New(Options{Store: newMemoryStore(), UpstreamURL: origin.URL})
	_, err := service.Type(t.Context(), 24497, "render", 0, "")
	if status, _ := asStatus(err); status != http.StatusNotFound {
		t.Fatalf("missing icon returned %v", err)
	}
}

func TestImageUpstreamFailureIsReportedOnceWithOriginalCause(t *testing.T) {
	oldHub := sentry.CurrentHub().Clone()
	oldLogger := log.Logger
	transport := &sentry.MockTransport{}
	runtime, err := telemetry.Init(telemetry.Options{Config: &config.Config{
		SentryDSN: "https://test@o1.ingest.sentry.io/42", SentryEnvironment: "test",
		SentryTracesSampleRate: 1, SentryLogs: true, SentryLogLevel: "info", SentryLogSampleRate: 1,
	}, Service: "test", Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		runtime.Close()
		sentry.CurrentHub().BindClient(oldHub.Client())
		sentry.CurrentHub().Scope().Clear()
		log.Logger = oldLogger
	})
	log.Logger = zerolog.New(telemetry.LogWriter{}).Hook(telemetry.LogHook{})
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer origin.Close()
	service := New(Options{Store: newMemoryStore(), UpstreamURL: origin.URL})
	mux := chi.NewRouter()
	Register(humachi.New(mux, huma.DefaultConfig("images", "test")), service)
	recorder := httptest.NewRecorder()
	telemetry.HTTP("images", mux).ServeHTTP(recorder, httptest.NewRequest("GET", "/images/types/24497/overlayrender", nil))
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("upstream outage returned %d", recorder.Code)
	}
	runtime.Close()
	failures := 0
	for _, event := range transport.Events() {
		if len(event.Exception) > 0 {
			failures++
			if event.Exception[0].Value != "Type image origin returned 503" {
				t.Fatalf("lost original error: %s", event.Exception[0].Value)
			}
		}
	}
	if failures != 1 {
		t.Fatalf("failure events=%d, want one", failures)
	}
}
