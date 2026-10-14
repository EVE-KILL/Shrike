// Package telemetry connects the shared Go runtime to Sentry. Instrumentation
// is a no-op without a DSN and never depends on Sentry being reachable.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/retry"
	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
	"github.com/rs/zerolog"
)

const FlushTimeout = 2 * time.Second

type Options struct {
	Config                   *config.Config
	Version, Commit, Service string
	// Transport is normally nil; tests use an in-memory transport.
	Transport sentry.Transport
}

type Runtime struct {
	client        *sentry.Client
	redact        *redactor
	logs, metrics bool
	logLevel      zerolog.Level
	logSampleRate float64
	stop          chan struct{}
	done          chan struct{}
	once          sync.Once
	mu            sync.RWMutex
	collectors    map[string]func(context.Context)
	metricBuffer  metricBuffer
}

var active atomic.Pointer[Runtime]

type captureStateKey struct{}

func Init(opts Options) (*Runtime, error) {
	cfg := opts.Config
	if cfg == nil || cfg.SentryDSN == "" {
		return nil, nil
	}
	level, err := zerolog.ParseLevel(cfg.SentryLogLevel)
	if err != nil {
		return nil, fmt.Errorf("SENTRY_LOG_LEVEL must be a valid log level")
	}
	r := &Runtime{redact: newRedactor(cfg), logs: cfg.SentryLogs, metrics: cfg.SentryMetrics,
		logLevel: level, logSampleRate: cfg.SentryLogSampleRate,
		stop: make(chan struct{}), done: make(chan struct{}), collectors: make(map[string]func(context.Context))}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn: cfg.SentryDSN, Environment: cfg.SentryEnvironment,
		Release: "shrike@" + opts.Version, Dist: opts.Commit,
		AttachStacktrace: true, EnableTracing: cfg.SentryTracesSampleRate > 0,
		TracesSampleRate: cfg.SentryTracesSampleRate, MaxSpans: 1000,
		StrictTraceContinuation: true, Transport: opts.Transport,
		DataCollection: &sentry.DataCollection{
			UserInfo: sentry.Set(false), HTTPBodies: []sentry.BodyType{},
			Cookies:     &sentry.KeyValueCollectionBehavior{Mode: sentry.CollectionOff},
			QueryParams: &sentry.KeyValueCollectionBehavior{Mode: sentry.CollectionOff},
			HTTPHeaders: &sentry.HeaderCollectionConfig{
				Request:  &sentry.KeyValueCollectionBehavior{Mode: sentry.CollectionOff},
				Response: &sentry.KeyValueCollectionBehavior{Mode: sentry.CollectionOff},
			},
		},
		BeforeSend: r.redact.event, BeforeSendTransaction: r.redact.event,
		BeforeSendLog: r.filterLog, BeforeSendMetric: r.filterMetric,
		BeforeBreadcrumb: r.redact.breadcrumb,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize Sentry: %s", r.redact.text(err.Error()))
	}
	r.client = client
	sentry.CurrentHub().BindClient(client)
	sentry.CurrentHub().ConfigureScope(func(scope *sentry.Scope) {
		scope.Clear()
		scope.SetTag("service", opts.Service)
		// Error/trace tags and log/metric attributes are separate SDK stores.
		scope.SetAttributes(attribute.String("service", opts.Service))
	})
	active.Store(r)
	go r.collect()
	return r, nil
}

func Enabled() bool { return active.Load() != nil }

func Flush() bool {
	r := active.Load()
	if r == nil {
		return true
	}
	r.flushMetrics()
	return r.client.Flush(FlushTimeout)
}

func (r *Runtime) Close() {
	if r == nil {
		return
	}
	r.once.Do(func() {
		close(r.stop)
		<-r.done
		r.client.Flush(FlushTimeout)
		r.client.Close()
		active.CompareAndSwap(r, nil)
	})
}

// RegisterCollector installs cheap runtime gauges. A returned cleanup removes
// the collector when a pool, cache or queue consumer is stopped.
func RegisterCollector(name string, collector func(context.Context)) func() {
	r := active.Load()
	if r == nil || !r.metrics {
		return func() {}
	}
	r.mu.Lock()
	r.collectors[name] = collector
	r.mu.Unlock()
	return func() { r.mu.Lock(); delete(r.collectors, name); r.mu.Unlock() }
}

func (r *Runtime) collect() {
	defer close(r.done)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	metricsTicker := time.NewTicker(15 * time.Second)
	defer metricsTicker.Stop()
	for {
		select {
		case <-r.stop:
			r.flushMetrics()
			return
		case <-metricsTicker.C:
			r.flushMetrics()
		case <-ticker.C:
			if !r.metrics {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			Gauge(ctx, "runtime.goroutines", float64(runtime.NumGoroutine()))
			Gauge(ctx, "runtime.heap", float64(mem.HeapAlloc), WithUnit(sentry.UnitByte))
			Gauge(ctx, "runtime.heap.objects", float64(mem.HeapObjects))
			Gauge(ctx, "runtime.gc.cycles", float64(mem.NumGC))
			r.mu.RLock()
			collectors := make([]func(context.Context), 0, len(r.collectors))
			for _, collect := range r.collectors {
				collectors = append(collectors, collect)
			}
			r.mu.RUnlock()
			for _, collect := range collectors {
				collect(ctx)
			}
			cancel()
		}
	}
}

func hubFor(ctx context.Context) *sentry.Hub {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		return hub
	}
	return sentry.CurrentHub()
}

func Capture(ctx context.Context, err error) *sentry.EventID {
	if err == nil || !Enabled() || errors.Is(err, context.Canceled) {
		return nil
	}
	if _, deferred := errors.AsType[*retry.Deferred](err); deferred {
		return nil
	}
	if captured, ok := ctx.Value(captureStateKey{}).(*atomic.Bool); ok && captured.Swap(true) {
		return nil
	}
	if state := requestStateFrom(ctx); state != nil {
		state.captured.Store(true)
	}
	hub := hubFor(ctx).Clone()
	if span := sentry.SpanFromContext(ctx); span != nil {
		hub.Scope().SetSpan(span)
	}
	return hub.CaptureException(err)
}

func Breadcrumb(ctx context.Context, category, message string, data map[string]any) {
	if !Enabled() {
		return
	}
	// A contextless background operation must not contaminate other tasks.
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.AddBreadcrumb(&sentry.Breadcrumb{Category: category, Message: message, Level: sentry.LevelError, Data: data}, nil)
	}
}

// RecoverAndRepanic preserves the original crash behavior after delivery.
func RecoverAndRepanic(ctx context.Context) {
	if value := recover(); value != nil {
		if Enabled() {
			hubFor(ctx).RecoverWithContext(ctx, value)
			hubFor(ctx).Flush(FlushTimeout)
		}
		panic(value)
	}
}

func StartSpan(ctx context.Context, operation, description string) (context.Context, *sentry.Span) {
	if !Recording(ctx) {
		return ctx, nil
	}
	span := sentry.StartSpan(ctx, operation, sentry.WithDescription(description))
	return span.Context(), span
}

func Recording(ctx context.Context) bool {
	if !Enabled() {
		return false
	}
	span := sentry.SpanFromContext(ctx)
	return span != nil && span.Sampled == sentry.SampledTrue
}

func FinishSpan(span *sentry.Span, err error) {
	if span == nil {
		return
	}
	span.Status = sentry.SpanStatusOK
	if err != nil {
		span.Status = sentry.SpanStatusInternalError
		if _, deferred := errors.AsType[*retry.Deferred](err); deferred {
			span.Status = sentry.SpanStatusResourceExhausted
		}
		if errors.Is(err, context.Canceled) {
			span.Status = sentry.SpanStatusCanceled
		}
		if errors.Is(err, context.DeadlineExceeded) {
			span.Status = sentry.SpanStatusDeadlineExceeded
		}
	}
	span.Finish()
}

func Count(ctx context.Context, name string, n int64, opts ...MetricOption) {
	if r := active.Load(); r != nil && r.metrics {
		r.metricBuffer.add(ctx, "counter", name, float64(n), n, opts)
	}
}
func Gauge(ctx context.Context, name string, value float64, opts ...MetricOption) {
	if r := active.Load(); r != nil && r.metrics {
		r.metricBuffer.add(ctx, "gauge", name, value, 0, opts)
	}
}
func Duration(ctx context.Context, name string, elapsed time.Duration, opts ...MetricOption) {
	if r := active.Load(); r != nil && r.metrics {
		r.metricBuffer.add(ctx, "distribution", name, float64(elapsed)/float64(time.Millisecond), 0, append(opts, WithUnit(sentry.UnitMillisecond)))
	}
}

func (r *Runtime) filterMetric(metric *sentry.Metric) *sentry.Metric {
	if !r.metrics {
		return nil
	}
	r.redact.attributes(metric.Attributes)
	return metric
}
