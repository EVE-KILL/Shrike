package logging

import (
	"context"
	"log/slog"
	"strings"

	"github.com/rs/zerolog"
)

// Slog preserves River/library structured fields in the same zerolog/Sentry
// pipeline as application logs; stdout remains reserved for command output.
func Slog(logger zerolog.Logger) *slog.Logger { return slog.New(slogHandler{logger: logger}) }

type slogHandler struct {
	logger zerolog.Logger
	attrs  []slog.Attr
	group  string
}

func slogLevel(level slog.Level) zerolog.Level {
	switch {
	case level >= slog.LevelError:
		return zerolog.ErrorLevel
	case level >= slog.LevelWarn:
		return zerolog.WarnLevel
	case level >= slog.LevelInfo:
		return zerolog.InfoLevel
	default:
		return zerolog.DebugLevel
	}
}
func (h slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return slogLevel(level) >= zerolog.GlobalLevel() && slogLevel(level) >= h.logger.GetLevel()
}
func (h slogHandler) Handle(ctx context.Context, record slog.Record) error {
	event := h.logger.WithLevel(slogLevel(record.Level)).Ctx(ctx)
	var appendAttr func(slog.Attr, string)
	appendAttr = func(attr slog.Attr, group string) {
		attr.Value = attr.Value.Resolve()
		if attr.Equal(slog.Attr{}) {
			return
		}
		key := attr.Key
		if group != "" && key != "" {
			key = group + "." + key
		}
		if attr.Value.Kind() == slog.KindGroup {
			for _, nested := range attr.Value.Group() {
				appendAttr(nested, strings.Trim(key, "."))
			}
			return
		}
		if err, ok := attr.Value.Any().(error); ok {
			event.Str(key, err.Error())
		} else {
			event.Interface(key, attr.Value.Any())
		}
	}
	for _, attr := range h.attrs {
		appendAttr(attr, "")
	}
	record.Attrs(func(attr slog.Attr) bool { appendAttr(attr, h.group); return true })
	event.Msg(record.Message)
	return nil
}
func (h slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	merged := append([]slog.Attr(nil), h.attrs...)
	if h.group != "" {
		attrs = []slog.Attr{{Key: h.group, Value: slog.GroupValue(attrs...)}}
	}
	merged = append(merged, attrs...)
	h.attrs = merged
	return h
}
func (h slogHandler) WithGroup(group string) slog.Handler {
	if group == "" {
		return h
	}
	if h.group == "" {
		h.group = group
	} else {
		h.group += "." + group
	}
	return h
}
