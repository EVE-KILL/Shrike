// Package pgbulk loads many rows at once, the only way that is fast enough for
// the importers.
//
// The shape is always the same: COPY into an unindexed temp table, then one
// INSERT ... SELECT into the real one. COPY skips the per-row parse and plan
// that INSERT pays, and the temp table has no indexes or constraints to
// maintain while it fills; the single INSERT that follows is what pays for
// those, once, in bulk. Against batched INSERTs this is five to ten times
// faster on the row counts the archives produce.
package pgbulk

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Conn is the subset of pgx a bulk load needs. Both *pgx.Conn and pgx.Tx
// satisfy it, so a load can run inside a caller's transaction or on its own.
type Conn interface {
	CopyFrom(ctx context.Context, table pgx.Identifier, columns []string, src pgx.CopyFromSource) (int64, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// FlushAt is how many rows accumulate before a COPY is issued. High enough that
// the per-COPY overhead disappears, low enough that a day's archive does not
// have to fit in memory twice.
const FlushAt = 50_000

// Copier buffers rows and copies them into a staging table.
type Copier struct {
	ctx     context.Context
	conn    Conn
	table   string
	columns []string

	buf     [][]any
	written int64
}

// NewCopier starts a buffered copy into table.
func NewCopier(ctx context.Context, conn Conn, table string, columns []string) *Copier {
	return &Copier{ctx: ctx, conn: conn, table: table, columns: columns}
}

// Add queues one row, flushing when the buffer fills.
func (c *Copier) Add(row []any) error {
	if len(row) != len(c.columns) {
		return fmt.Errorf("row has %d values for %d columns", len(row), len(c.columns))
	}
	c.buf = append(c.buf, row)
	if len(c.buf) >= FlushAt {
		return c.Flush()
	}
	return nil
}

// Flush writes whatever is buffered. Callers must call it once at the end.
func (c *Copier) Flush() error {
	if len(c.buf) == 0 {
		return nil
	}
	n, err := c.conn.CopyFrom(c.ctx, pgx.Identifier{c.table}, c.columns, pgx.CopyFromRows(c.buf))
	if err != nil {
		return fmt.Errorf("copy into %s: %w", c.table, err)
	}
	c.written += n
	c.buf = c.buf[:0]
	return nil
}

// Written reports how many rows have reached the staging table.
func (c *Copier) Written() int64 { return c.written }

// StagingTx creates a temp table shaped like a real one, scoped to the
// transaction.
//
// ON COMMIT DROP is what makes it safe, and the reason is worth stating: a temp
// table belongs to the *session*, not the connection handle. Creating one with
// PRESERVE ROWS inside a transaction that then commits leaves it behind on
// whichever pooled connection ran it, and the next caller to draw that
// connection fails with "relation already exists". Behind pgbouncer in session
// pooling that leak outlives the process. Letting the commit drop it removes
// the whole class of problem — and there is nothing to clean up on the error
// path either, because the rollback drops it too.
func StagingTx(ctx context.Context, tx Conn, name, like string) error {
	if _, err := tx.Exec(ctx, fmt.Sprintf(
		`CREATE TEMP TABLE %s (LIKE public.%s INCLUDING DEFAULTS) ON COMMIT DROP`,
		name, like)); err != nil {
		return fmt.Errorf("create staging table %s: %w", name, err)
	}
	return nil
}

// Staging creates a session-scoped temp table for a load that is not inside a
// transaction, and returns a function that drops it.
//
// The caller must call that function — see StagingTx for what happens
// otherwise. It uses a context detached from the caller's so that cleanup still
// runs when the import was cancelled, which is exactly when a leftover temp
// table is most annoying.
func Staging(ctx context.Context, conn Conn, name, like string) (func(), error) {
	// IF NOT EXISTS covers the one case the drop cannot: a previous run on this
	// connection that was killed between creating the table and dropping it.
	if _, err := conn.Exec(ctx, fmt.Sprintf(
		`CREATE TEMP TABLE IF NOT EXISTS %s (LIKE public.%s INCLUDING DEFAULTS) ON COMMIT PRESERVE ROWS`,
		name, like)); err != nil {
		return nil, fmt.Errorf("create staging table %s: %w", name, err)
	}
	// ...and if it did already exist, it may still hold that run's rows.
	if _, err := conn.Exec(ctx, "TRUNCATE "+name); err != nil {
		return nil, fmt.Errorf("truncate staging table %s: %w", name, err)
	}
	return func() {
		_, _ = conn.Exec(context.WithoutCancel(ctx), "DROP TABLE IF EXISTS "+name)
	}, nil
}

// Conflict says what a merge does when a row already exists.
type Conflict int

const (
	// DoNothing keeps what is already stored. Correct when the existing row is
	// at least as good as the incoming one — re-importing an archive, say.
	DoNothing Conflict = iota
	// DoUpdate overwrites every non-key column. Correct when the source is
	// authoritative and the stored row may be stale.
	DoUpdate
)

// MergeSQL builds the INSERT ... SELECT that moves staged rows into the real
// table.
//
// The SELECT is DISTINCT ON the key because a single archive can carry the same
// key twice, and without the guard DoUpdate aborts the whole statement with
// "ON CONFLICT DO UPDATE command cannot affect row a second time".
func MergeSQL(table, staging string, columns, pk []string, mode Conflict) string {
	cols := strings.Join(columns, ", ")
	sel := fmt.Sprintf("SELECT DISTINCT ON (%s) %s FROM %s",
		strings.Join(pk, ", "), cols, staging)

	if mode == DoNothing {
		return fmt.Sprintf("INSERT INTO public.%s (%s) %s ON CONFLICT (%s) DO NOTHING",
			table, cols, sel, strings.Join(pk, ", "))
	}

	isKey := make(map[string]bool, len(pk))
	for _, c := range pk {
		isKey[c] = true
	}
	var sets []string
	for _, c := range columns {
		if !isKey[c] {
			sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", c, c))
		}
	}
	if len(sets) == 0 {
		// Every column is part of the key, so there is nothing an update could
		// change.
		return fmt.Sprintf("INSERT INTO public.%s (%s) %s ON CONFLICT (%s) DO NOTHING",
			table, cols, sel, strings.Join(pk, ", "))
	}
	return fmt.Sprintf("INSERT INTO public.%s (%s) %s ON CONFLICT (%s) DO UPDATE SET %s",
		table, cols, sel, strings.Join(pk, ", "), strings.Join(sets, ", "))
}
