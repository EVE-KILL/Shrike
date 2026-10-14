// Package logging wires zerolog. Logs always go to stderr so that a command's
// actual output on stdout stays machine-readable — `shrike doctor --json | jq`
// must not have log lines spliced into it.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/eve-kill/shrike/internal/telemetry"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init configures the global logger. jsonFormat selects structured output for
// production log shipping; otherwise we use the human console writer.
func Init(level string, jsonFormat bool) {
	parsed, err := zerolog.ParseLevel(level)
	if err != nil {
		parsed = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(parsed)

	var output io.Writer = os.Stderr
	if !jsonFormat {
		output = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	}
	if telemetry.Enabled() {
		output = zerolog.MultiLevelWriter(output, telemetry.LogWriter{})
	}
	log.Logger = zerolog.New(output).Hook(telemetry.LogHook{}).With().Timestamp().Logger()
	slog.SetDefault(Slog(log.Logger))
}

// ThirdParty adapts libraries that insist on their own logger (go-redis, for
// one) onto zerolog at debug level. Without it, go-redis writes connection-pool
// retry chatter straight to stderr with its own timestamps, which corrupts both
// our log format and any command's human-facing output.
//
// The method set is chosen to satisfy redis.Logging structurally, so this
// package needs no dependency on the client libraries it tames.
type ThirdParty struct{}

func (ThirdParty) Printf(ctx context.Context, format string, v ...any) {
	log.Debug().Ctx(ctx).Msgf(format, v...)
}
