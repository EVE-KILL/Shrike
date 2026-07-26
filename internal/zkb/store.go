package zkb

import (
	"context"
	"errors"
	"strconv"

	"github.com/eve-kill/shrike/internal/configstore"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is the production Store.
//
// Deliver is a function rather than a method so this package does not depend on
// the queue: `work:zkb` hands it a River insert, a verification command hands
// it one that parses and stores inline, and neither arrangement requires the
// listener to know which it is talking to.
type PostgresStore struct {
	Pool    *pgxpool.Pool
	Deliver func(ctx context.Context, r *Response) error
}

// Cursor reads the stored sequence.
//
// The key is the one the TypeScript listener uses, and that is deliberate: the
// Go listener has to resume where the process it replaces stopped, not restart
// at the head of the feed and leave a gap behind it.
func (s *PostgresStore) Cursor(ctx context.Context) (int64, error) {
	raw, err := configstore.Get(ctx, s.Pool, configstore.KeyR2Z2Sequence)
	if err != nil {
		return 0, err
	}
	if raw == "" {
		return 0, nil
	}
	seq, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seq < 0 {
		// A malformed bookmark is treated as absent rather than fatal. The
		// consequence is a bootstrap from the head of the feed, which is what a
		// fresh install does anyway.
		return 0, nil
	}
	return seq, nil
}

// SaveCursor persists the sequence.
func (s *PostgresStore) SaveCursor(ctx context.Context, sequence int64) error {
	return configstore.Set(ctx, s.Pool, configstore.KeyR2Z2Sequence, strconv.FormatInt(sequence, 10))
}

// Has reports whether the killmail is already stored.
func (s *PostgresStore) Has(ctx context.Context, killmailID int64) (bool, error) {
	var found bool
	err := s.Pool.QueryRow(ctx,
		`SELECT true FROM killmails WHERE killmail_id = $1`, killmailID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return found, nil
}

// Accept hands the killmail on.
func (s *PostgresStore) Accept(ctx context.Context, r *Response) error {
	return s.Deliver(ctx, r)
}
