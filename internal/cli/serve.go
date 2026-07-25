package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/eve-kill/shrike/internal/ui"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// serve currently exposes only /health. It exists at this stage to prove the
// process lifecycle — bind, serve, drain on signal — that every real service
// will inherit. The Huma API mounts onto this same handler next.

var flagServePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the HTTP server",
	Long: `Starts the HTTP listener in the foreground and runs until interrupted.

SIGINT (Ctrl+C) and SIGTERM both trigger a graceful shutdown: the listener
stops accepting, in-flight requests are given time to finish, then the process
exits. Kubernetes needs no special handling beyond its default SIGTERM.

Only /health is served so far; the OpenAPI surface lands on top of it.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		port := cfg.Port
		if cmd.Flags().Changed("port") {
			port = flagServePort
		}

		mux := http.NewServeMux()
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "ok",
				"version": ui.Version,
				"commit":  ui.Commit,
			})
		})

		srv := &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second, // slowloris guard
		}

		return RunService(cmd, "serve", func(ctx context.Context) error {
			errc := make(chan error, 1)
			go func() {
				log.Info().Int("port", port).Msg("http listening")
				// ErrServerClosed is the expected result of our own Shutdown
				// call, so it is not propagated as a failure.
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					errc <- err
					return
				}
				errc <- nil
			}()

			select {
			case err := <-errc:
				// Failed on its own — almost always a bind error.
				return err
			case <-ctx.Done():
				// Shutdown gets its own timeout: ctx is already cancelled, and
				// passing it through would abort the drain immediately.
				drainCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace-time.Second)
				defer cancel()
				return srv.Shutdown(drainCtx)
			}
		})
	},
}

func init() {
	serveCmd.Flags().IntVar(&flagServePort, "port", 0, "Port to listen on (overrides PORT)")
}
