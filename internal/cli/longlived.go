package cli

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eve-kill/shrike/internal/ui"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// errInterrupted marks a shutdown caused by Ctrl+C. Execute maps it to exit
// code 130 (128 + SIGINT), the shell convention, so an interactive interrupt
// does not read as a crash.
//
// SIGTERM deliberately has no sentinel: it is an *instructed* shutdown, so
// obeying it is success and RunService returns nil. Exiting non-zero there
// would make every Kubernetes rolling update show a failed container.
var errInterrupted = errors.New("interrupted")

// shutdownGrace bounds how long a service gets to drain after cancellation.
// Kubernetes sends SIGKILL 30s after SIGTERM by default; finishing well inside
// that keeps us from being killed mid-drain.
const shutdownGrace = 15 * time.Second

// RunService is the single contract for every long-running command — `serve`,
// `work`, `cron start`. It owns signal handling and the drain deadline so each
// service only has to respect ctx and return.
//
// The passed function must return when ctx is cancelled. If it does not return
// within shutdownGrace we stop waiting and let the process exit; a worker stuck
// on an unresponsive socket must not block the pod's termination.
func RunService(cmd *cobra.Command, name string, run func(ctx context.Context) error) error {
	// A dedicated channel rather than signal.NotifyContext, because the exit
	// code depends on *which* signal arrived and NotifyContext discards that.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigc)

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	var received os.Signal
	go func() {
		select {
		case sig := <-sigc:
			received = sig
			cancel()
		case <-ctx.Done():
		}
	}()

	log.Info().Str("service", name).Msg("starting")

	done := make(chan error, 1)
	go func() { done <- run(ctx) }()

	// interruptResult maps the signal that stopped us onto a return value.
	interruptResult := func() error {
		if received == syscall.SIGINT {
			return errInterrupted
		}
		return nil
	}

	select {
	case err := <-done:
		// Returned on its own — either finished or failed, unrelated to signals.
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		log.Info().Str("service", name).Msg("stopped")
		return nil

	case <-ctx.Done():
		// A blank line separates the shutdown log from whatever the service was
		// printing, and from the ^C the terminal just echoed.
		ui.Newline()
		log.Info().Str("service", name).
			Str("signal", signalName(received)).
			Str("grace", shutdownGrace.String()).
			Msg("shutdown signal received, draining")

		select {
		case err := <-done:
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			log.Info().Str("service", name).Msg("stopped cleanly")
			return interruptResult()

		case <-time.After(shutdownGrace):
			log.Warn().Str("service", name).
				Str("grace", shutdownGrace.String()).
				Msg("drain deadline exceeded, exiting anyway")
			return interruptResult()
		}
	}
}

func signalName(sig os.Signal) string {
	if sig == nil {
		// Reachable when the parent context is cancelled rather than a signal
		// arriving — currently only in tests, but worth not printing "<nil>".
		return "context"
	}
	return sig.String()
}
