package telemetry

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
)

type Task struct {
	Context context.Context
	span    *sentry.Span
	started time.Time
	op      string
	attrs   []string
}

// StartTask gives concurrent workers separate hubs, breadcrumbs and scopes.
func StartTask(ctx context.Context, operation, name string, attrs ...string) (context.Context, *Task) {
	return StartTaskWithTrace(ctx, operation, name, TraceCarrier{}, attrs...)
}

type TraceCarrier struct {
	Trace   string `json:"trace"`
	Baggage string `json:"baggage"`
}

func Propagation(ctx context.Context) TraceCarrier {
	if !Enabled() {
		return TraceCarrier{}
	}
	if span := sentry.SpanFromContext(ctx); span != nil {
		return TraceCarrier{Trace: span.ToSentryTrace(), Baggage: span.ToBaggage()}
	}
	return TraceCarrier{}
}

func StartTaskWithTrace(ctx context.Context, operation, name string, carrier TraceCarrier, attrs ...string) (context.Context, *Task) {
	if !Enabled() {
		return ctx, nil
	}
	hub := hubFor(ctx).Clone()
	hub.Scope().SetRequest(nil)
	hub.Scope().SetSpan(nil)
	ctx = sentry.SetHubOnContext(ctx, hub)
	var span *sentry.Span
	if sentry.SpanFromContext(ctx) != nil {
		span = sentry.StartSpan(ctx, operation, sentry.WithDescription(name))
	} else {
		ctx = context.WithValue(ctx, captureStateKey{}, &atomic.Bool{})
		span = sentry.StartTransaction(ctx, name, sentry.WithOpName(operation), sentry.WithTransactionSource(sentry.SourceTask), sentry.ContinueTrace(hub, carrier.Trace, carrier.Baggage))
	}
	for i := 0; i+1 < len(attrs); i += 2 {
		span.SetData(attrs[i], attrs[i+1])
		hub.Scope().SetTag(attrs[i], attrs[i+1])
	}
	return span.Context(), &Task{Context: span.Context(), span: span, started: time.Now(), op: operation, attrs: attrs}
}

func (task *Task) End(outcome string, err error) {
	if task == nil {
		return
	}
	opts := Attributes(append(task.attrs, "outcome", outcome)...)
	Count(task.Context, task.op+".count", 1, opts)
	Duration(task.Context, task.op+".duration", time.Since(task.started), opts)
	task.span.SetData("outcome", outcome)
	FinishSpan(task.span, err)
}
