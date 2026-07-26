// Package configstore reads and writes the `config` table, a two-column
// key/value store the importers use as a bookmark.
//
// It is deliberately the same table, with the same keys, that the TypeScript
// services already use — an importer rewritten in Go must resume where the one
// it replaces stopped, not start again from 2007.
package configstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Keys currently in use. Listing them here rather than beside each importer
// makes an accidental collision visible, and the table is shared with the
// TypeScript services so the names are not ours alone to change.
const (
	KeySDEBuild          = "sde:buildNumber"
	KeyPricesLastDate    = "everef:prices:last_date"
	KeySovereigntyDate   = "everef:sovereignty:last_date"
	KeyKillmailsLastDate = "everef:killmails:last_date"
	KeyWarsLastYear      = "everef:wars:last_year"
	KeyWarsLastDaily     = "everef:wars:last_daily"

	// KeyR2Z2Sequence is the zKillboard R2Z2 feed cursor. Spelled exactly as
	// the TypeScript listener spells it, so the Go one resumes from where that
	// process stopped instead of bootstrapping at the head of the feed and
	// leaving every killmail in between unfetched.
	KeyR2Z2Sequence = "r2z2:last_sequence"
)

// Get returns a value, or the empty string when the key is unset.
//
// Absent and empty are deliberately not distinguished: every caller treats an
// unset bookmark as "nothing imported yet", and an empty string means the same.
func Get(ctx context.Context, pool *pgxpool.Pool, key string) (string, error) {
	var value string
	err := pool.QueryRow(ctx, `SELECT value FROM config WHERE key = $1`, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// Set stores a value.
func Set(ctx context.Context, pool *pgxpool.Pool, key, value string) error {
	_, err := pool.Exec(ctx, `
        INSERT INTO config (key, value) VALUES ($1, $2)
        ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, value)
	return err
}
