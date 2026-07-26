package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// FrontendOptions contains private, frontend-only runtime configuration.
//
// Public API handlers intentionally cannot reach these values without opting
// into opts.Frontend. That makes it much harder to accidentally expose an
// authentication secret through a public response or generated schema.
type FrontendOptions struct {
	Auth AuthOptions
}

// AuthOptions configures EVE SSO and the browser session cookie.
//
// HTTPClient is injected for deterministic OAuth tests. Nil uses a bounded
// production client.
type AuthOptions struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
	UserAgent    string
	Production   bool
	HTTPClient   *http.Client
}

// MutationDatabase is the database contract for frontend-owned state.
//
// The public API deliberately depends on the smaller read-only Database
// interface. The production *pgxpool.Pool satisfies both; tests for public
// routes therefore do not need to grow fake mutation methods.
type MutationDatabase interface {
	Database
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

func mutationDatabase(opts Options) (MutationDatabase, error) {
	db, ok := opts.DB.(MutationDatabase)
	if !ok || db == nil {
		return nil, apiError(http.StatusServiceUnavailable,
			"Frontend database is not configured")
	}
	return db, nil
}

// registerPrivateAPI is the single inventory of routes owned by the
// eve-kill.com frontend. Registrars are split by domain, but a behavior is
// implemented once even when an old frontend path remains as a compatibility
// alias during migration.
func registerPrivateAPI(a huma.API, opts Options) {
	registerHealth(a, opts)
	registerHealthAt(a, opts, "frontend-health", "/api/health")

	// Search is one behavior shared by the public API and the frontend. Keep
	// the frontend's established path while web/ moves to the generated Go
	// client; both routes execute the same handler.
	registerLegacy(a, huma.Operation{
		OperationID: "frontend-search",
		Method:      http.MethodGet,
		Path:        "/api/search",
		Summary:     "Search entities and universe data",
		Tags:        []string{"search"},
	}, searchHandler(opts))

	registerPrivateMarketRoutes(a, opts)
}
