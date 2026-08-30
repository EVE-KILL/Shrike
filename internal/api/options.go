package api

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuthOptions configures EVE SSO and the browser session cookie.
//
// StateSecret signs browser OAuth state and may be rotated independently from
// EVE's client secret. An empty value falls back to ClientSecret for transition
// compatibility. HTTPClient is injected for deterministic OAuth tests; nil
// uses a bounded production client.
type AuthOptions struct {
	ClientID     string
	ClientSecret string
	StateSecret  string
	CallbackURL  string
	UserAgent    string
	Production   bool
	HTTPClient   *http.Client

	// The fields below are package-private seams for focused tests. Production
	// always derives them from opts.DB, opts.Cache, and the EVE configuration
	// above, so callers outside package api cannot accidentally replace an
	// authentication dependency.
	store     authStore
	flowStore oauthFlowStore
	oauth     oauthCodeClient
	now       func() time.Time
	random    io.Reader
}

// MutationDatabase is the database contract for API-owned state.
//
// Read-only handlers deliberately depend on the smaller Database interface.
// The production *pgxpool.Pool satisfies both, while focused tests do not need
// mutation methods unless the operation they exercise actually writes.
type MutationDatabase interface {
	Database
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

func mutationDatabase(opts Options) (MutationDatabase, error) {
	if opts.Primary != nil {
		return opts.Primary, nil
	}
	db, ok := opts.DB.(MutationDatabase)
	if !ok || db == nil {
		return nil, apiError(http.StatusServiceUnavailable,
			"API database is not configured")
	}
	return db, nil
}

// primaryDatabase returns the primary when one is configured and preserves the
// single-database behavior used by focused tests and older callers otherwise.
func primaryDatabase(opts Options) Database {
	if opts.Primary != nil {
		return opts.Primary
	}
	return opts.DB
}

func primaryPool(opts Options) *pgxpool.Pool {
	if opts.PrimaryPool != nil {
		return opts.PrimaryPool
	}
	pool, _ := opts.DB.(*pgxpool.Pool)
	return pool
}
