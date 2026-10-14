package telemetry

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/rand/v2"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
	"github.com/rs/zerolog"
)

// LogHook carries event context through zerolog's JSON writer without keeping
// a global map of requests or leaking one goroutine's scope into another.
type LogHook struct{}

func (LogHook) Run(event *zerolog.Event, _ zerolog.Level, _ string) {
	if !Enabled() {
		return
	}
	if span := sentry.SpanFromContext(event.GetCtx()); span != nil {
		event.Str("_sentry_trace_id", span.TraceID.String()).Str("_sentry_span_id", span.SpanID.String())
	}
}

type LogWriter struct{}

func (w LogWriter) Write(data []byte) (int, error) {
	var entry map[string]any
	if json.Unmarshal(data, &entry) != nil {
		return len(data), nil
	}
	level, _ := entry[zerolog.LevelFieldName].(string)
	parsed, err := zerolog.ParseLevel(level)
	if err != nil {
		return len(data), nil
	}
	return w.WriteLevel(parsed, data)
}

func (LogWriter) WriteLevel(level zerolog.Level, data []byte) (int, error) {
	r := active.Load()
	if r == nil {
		return len(data), nil
	}
	var fields map[string]any
	if json.Unmarshal(data, &fields) != nil {
		return len(data), nil
	}
	// Access lines contain raw IPs/referrers. The HTTP middleware emits their
	// structured, privacy-safe equivalent on the request's own trace.
	if fields["subsystem"] == "ingress" {
		return len(data), nil
	}
	message, _ := fields[zerolog.MessageFieldName].(string)
	hub := sentry.CurrentHub().Clone()
	if trace, ok := fields["_sentry_trace_id"].(string); ok {
		spanID, _ := fields["_sentry_span_id"].(string)
		span := &sentry.Span{}
		if tid, err := hex.DecodeString(trace); err == nil && len(tid) == len(span.TraceID) {
			copy(span.TraceID[:], tid)
			if sid, err := hex.DecodeString(spanID); err == nil && len(sid) == len(span.SpanID) {
				copy(span.SpanID[:], sid)
			}
			hub.Scope().SetSpan(span)
		}
	}
	ctx := sentry.SetHubOnContext(context.Background(), hub)
	r.redact.data(fields)
	delete(fields, zerolog.MessageFieldName)
	delete(fields, zerolog.LevelFieldName)
	delete(fields, zerolog.TimestampFieldName)
	if r.logs && level >= r.logLevel && (level >= zerolog.WarnLevel || rand.Float64() < r.logSampleRate) { //nolint:gosec // Sampling is not a security decision.
		logger := sentry.NewLogger(ctx)
		for key, value := range fields {
			switch value := value.(type) {
			case string:
				logger.SetAttributes(attribute.String(key, value))
			case bool:
				logger.SetAttributes(attribute.Bool(key, value))
			case float64:
				logger.SetAttributes(attribute.Float64(key, value))
			}
		}
		switch level {
		case zerolog.TraceLevel:
			logger.Trace().Emit(message)
		case zerolog.DebugLevel:
			logger.Debug().Emit(message)
		case zerolog.InfoLevel:
			logger.Info().Emit(message)
		case zerolog.WarnLevel:
			logger.Warn().Emit(message)
		case zerolog.ErrorLevel:
			logger.Error().Emit(message)
		case zerolog.FatalLevel, zerolog.PanicLevel:
			logger.LFatal().Emit(message)
		}
	}
	if level >= zerolog.ErrorLevel && fields["sentry_handled"] != true {
		event := sentry.NewEvent()
		event.Message = message
		event.Logger = "zerolog"
		event.Level = sentry.LevelError
		event.Contexts["log"] = fields
		if level >= zerolog.FatalLevel {
			event.Level = sentry.LevelFatal
		}
		if value, ok := fields[zerolog.ErrorFieldName].(string); ok {
			event.Exception = []sentry.Exception{{Type: "LoggedError", Value: value, Stacktrace: sentry.NewStacktrace()}}
		}
		hub.CaptureEvent(event)
	}
	if level >= zerolog.FatalLevel {
		hub.Flush(FlushTimeout)
	}
	return len(data), nil
}

func (r *Runtime) filterLog(entry *sentry.Log) *sentry.Log {
	if !r.logs {
		return nil
	}
	entry.Body = r.redact.text(entry.Body)
	r.redact.attributes(entry.Attributes)
	return entry
}

// Observe emits one domain outcome without copying an operation's input/body.
func Observe(ctx context.Context, name string, elapsed time.Duration, err error, attrs ...string) {
	result := "ok"
	if err != nil {
		result = "error"
	}
	opts := Attributes(append(attrs, "outcome", result)...)
	Count(ctx, name+".count", 1, opts)
	Duration(ctx, name+".duration", elapsed, opts)
}
