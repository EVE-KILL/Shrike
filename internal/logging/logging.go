// Package logging wires zerolog. Logs always go to stderr so that a command's
// actual output on stdout stays machine-readable — `shrike doctor --json | jq`
// must not have log lines spliced into it.
package logging

import (
	"context"
	"os"
	"time"

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

	if jsonFormat {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
		return
	}

	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}).With().Timestamp().Logger()
}

// ThirdParty adapts libraries that insist on their own logger (go-redis, for
// one) onto zerolog at debug level. Without it, go-redis writes connection-pool
// retry chatter straight to stderr with its own timestamps, which corrupts both
// our log format and any command's human-facing output.
//
// The method set is chosen to satisfy redis.Logging structurally, so this
// package needs no dependency on the client libraries it tames.
type ThirdParty struct{}

func (ThirdParty) Printf(_ context.Context, format string, v ...any) {
	log.Debug().Msgf(format, v...)
}
