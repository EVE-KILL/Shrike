package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/graph"
	"github.com/eve-kill/shrike/internal/mcpserver"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var flagMCPPort int

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run the Model Context Protocol server",
	Long: `Runs the public EVE-KILL Model Context Protocol server.

The streamable HTTP endpoint is /mcp. The process is stateless and exposes
read-only tools backed by Postgres and Memgraph.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		port := cfg.Port
		if cmd.Flags().Changed("port") {
			port = flagMCPPort
		}
		return RunService(cmd, "mcp", func(ctx context.Context) error {
			pool, err := db.New(ctx, cfg)
			if err != nil {
				return fmt.Errorf("connect MCP database: %w", err)
			}
			defer pool.Close()

			var graphClient *graph.Client
			graphCtx, cancelGraph := context.WithTimeout(ctx, 3*time.Second)
			graphClient, graphErr := graph.Connect(graphCtx, cfg.MemgraphURL)
			cancelGraph()
			if graphErr != nil {
				log.Warn().Err(graphErr).Msg(
					"Memgraph unavailable; graph-backed MCP tools will return errors",
				)
			} else {
				defer graphClient.Close(context.Background()) //nolint:errcheck
			}

			handler, err := mcpserver.Handler(mcpserver.Dependencies{
				DB:      pool,
				Graph:   graphClient,
				BaseURL: "https://eve-kill.com",
			}, ui.Version, nil)
			if err != nil {
				return fmt.Errorf("build MCP server: %w", err)
			}

			mux := http.NewServeMux()
			mux.Handle("/mcp", handler)
			mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				if err := pool.Ping(r.Context()); err != nil {
					http.Error(w, "database unavailable", http.StatusServiceUnavailable)
					return
				}
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, _ = w.Write([]byte("ok"))
			})
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]string{
					"name":      "evekill-mcp",
					"transport": "streamable-http",
					"endpoint":  "/mcp",
				})
			})

			server := &http.Server{
				Addr:              fmt.Sprintf(":%d", port),
				Handler:           mux,
				ReadHeaderTimeout: 10 * time.Second,
				ReadTimeout:       2 * time.Minute,
				WriteTimeout:      2 * time.Minute,
				IdleTimeout:       5 * time.Minute,
				MaxHeaderBytes:    64 << 10,
			}
			errs := make(chan error, 1)
			go func() {
				errs <- server.ListenAndServe()
			}()
			log.Info().
				Str("url", fmt.Sprintf("http://localhost:%d/mcp", port)).
				Msg("MCP server listening")

			select {
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(
					context.Background(),
					10*time.Second,
				)
				defer cancel()
				return server.Shutdown(shutdownCtx)
			case err := <-errs:
				if errors.Is(err, http.ErrServerClosed) {
					return nil
				}
				return err
			}
		})
	},
}

func init() {
	mcpCmd.Flags().IntVar(&flagMCPPort, "port", 0, "Port to listen on (overrides PORT)")
}
