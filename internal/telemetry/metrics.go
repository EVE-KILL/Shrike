package telemetry

import (
	"context"
	"math/rand/v2"
	"sort"
	"strings"
	"sync"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
)

const (
	maxMetricSeries    = 2048
	maxDurationSamples = 64
)

type MetricOption struct {
	attrs []string
	unit  string
}

func Attributes(values ...string) MetricOption { return MetricOption{attrs: values} }
func WithUnit(unit string) MetricOption        { return MetricOption{unit: unit} }

type metricSample struct {
	value float64
	trace sentry.TraceID
	span  sentry.SpanID
}
type metricSeries struct {
	kind, name, unit string
	attrs            []string
	count            int64
	value            float64
	observations     int64
	samples          []metricSample
}
type metricBuffer struct {
	mu      sync.Mutex
	series  map[string]*metricSeries
	dropped int64
}

// Counters remain exact; gauges retain the latest value. Durations use a
// bounded uniform reservoir per series to preserve latency distributions
// without shipping an envelope entry for every query/cache lookup.
func (b *metricBuffer) add(ctx context.Context, kind, name string, value float64, count int64, opts []MetricOption) {
	var attrs []string
	unit := ""
	for _, option := range opts {
		attrs = append(attrs, option.attrs...)
		if option.unit != "" {
			unit = option.unit
		}
	}
	// Fixed call sites use consistent order; sort pairs to merge equivalent
	// labels even when a future caller passes them in a different order.
	pairs := make([]string, 0, len(attrs)/2)
	for i := 0; i+1 < len(attrs); i += 2 {
		pairs = append(pairs, attrs[i]+"\x00"+attrs[i+1])
	}
	sort.Strings(pairs)
	key := kind + "\x00" + name + "\x00" + unit + "\x00" + strings.Join(pairs, "\x00")
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.series == nil {
		b.series = make(map[string]*metricSeries)
	}
	series := b.series[key]
	if series == nil {
		if len(b.series) >= maxMetricSeries {
			b.dropped++
			return
		}
		series = &metricSeries{kind: kind, name: name, unit: unit, attrs: attrs}
		b.series[key] = series
	}
	series.count += count
	series.value = value
	series.observations++
	if kind != "distribution" {
		return
	}
	sample := metricSample{value: value}
	if span := sentry.SpanFromContext(ctx); span != nil {
		sample.trace, sample.span = span.TraceID, span.SpanID
	}
	if len(series.samples) < maxDurationSamples {
		series.samples = append(series.samples, sample)
	} else if index := rand.Int64N(series.observations); index < maxDurationSamples { //nolint:gosec // Reservoir sampling is not a security decision.
		series.samples[index] = sample
	}
}

func (r *Runtime) flushMetrics() {
	if !r.metrics {
		return
	}
	b := &r.metricBuffer
	b.mu.Lock()
	series, dropped := b.series, b.dropped
	b.series = nil
	b.dropped = 0
	b.mu.Unlock()
	ctx := context.Background()
	meter := sentry.NewMeter(ctx)
	for _, value := range series {
		attrs := make([]attribute.Builder, 0, len(value.attrs)/2)
		for i := 0; i+1 < len(value.attrs); i += 2 {
			attrs = append(attrs, attribute.String(value.attrs[i], value.attrs[i+1]))
		}
		opts := []sentry.MeterOption{sentry.WithAttributes(attrs...), sentry.WithUnit(value.unit)}
		switch value.kind {
		case "counter":
			meter.Count(value.name, value.count, opts...)
		case "gauge":
			meter.Gauge(value.name, value.value, opts...)
		case "distribution":
			for _, sample := range value.samples {
				hub := sentry.CurrentHub().Clone()
				hub.Scope().SetSpan(&sentry.Span{TraceID: sample.trace, SpanID: sample.span})
				sentry.NewMeter(sentry.SetHubOnContext(ctx, hub)).Distribution(value.name, sample.value, opts...)
			}
		}
	}
	if dropped > 0 {
		meter.Count("telemetry.metrics.dropped_observations", dropped)
	}
}
