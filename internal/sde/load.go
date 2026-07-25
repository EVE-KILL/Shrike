package sde

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Table declares how one JSONL member maps onto one Postgres table.
//
// Deliberately data, not code: adding a table is a literal in tables.go rather
// than a new function, which keeps the 14 straightforward imports readable side
// by side and makes them diffable against the TypeScript they came from.
type Table struct {
	// Member is the archive member without its .jsonl suffix.
	Member string

	// Name is the destination table.
	Name string

	// PK is the conflict target for the upsert.
	PK []string

	// Columns are the columns written, in the order Values returns them.
	Columns []string

	// Values converts a row to column values. Returning false skips the row —
	// used for records with no usable "_key".
	//
	// Exactly one of Values or Expand must be set.
	Values func(Row) ([]any, bool)

	// Expand is the one-to-many form: a single archive record yields any number
	// of destination rows. Several members nest their real payload in an array
	// or object — one typeDogma record carries every attribute for a type, one
	// blueprint carries every activity — so the row count in the archive bears
	// no relation to the row count in the table.
	Expand func(Row) [][]any

	// Optional marks members that may legitimately be absent from an archive.
	// CCP adds and drops members between builds.
	Optional bool
}

// LoadResult reports what a single table import did.
type LoadResult struct {
	Table    string        `json:"table"`
	Member   string        `json:"member"`
	Read     int64         `json:"read"`
	Written  int64         `json:"written"`
	Skipped  int64         `json:"skipped"`
	Duration time.Duration `json:"-"`
	Elapsed  string        `json:"elapsed"`
	Missing  bool          `json:"missing,omitempty"`
}

// Load streams a member into its table.
//
// The write path is COPY into a TEMP table followed by INSERT ... SELECT ...
// ON CONFLICT DO UPDATE, rather than batched upserts. Two reasons: COPY is
// several times faster than even large multi-row INSERTs, and the merge then
// happens in one statement so the destination table is never half-updated. The
// import is idempotent, which matters because it runs daily from a cron.
//
// Everything runs on a single dedicated connection because a TEMP table is
// session-scoped — with a pool, the COPY and the merge could otherwise land on
// different connections and the merge would not see the staged rows.
func Load(ctx context.Context, pool *pgxpool.Pool, t Table, src *Source) (LoadResult, error) {
	res := LoadResult{Table: t.Name, Member: t.Member}
	start := time.Now()

	if !src.Has(t.Member) {
		if t.Optional {
			res.Missing = true
			res.Elapsed = time.Since(start).Round(time.Millisecond).String()
			return res, nil
		}
		return res, fmt.Errorf("archive is missing required member %s.jsonl", t.Member)
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return res, fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	staging := "sde_staging_" + t.Name

	// A TEMP table is unlogged by nature and disappears with the session, so
	// there is nothing to clean up if this fails. ON COMMIT DROP is not used
	// because the COPY and merge are separate statements outside one transaction.
	if _, err := conn.Exec(ctx, fmt.Sprintf(
		`CREATE TEMP TABLE %s (LIKE public.%s INCLUDING DEFAULTS) ON COMMIT PRESERVE ROWS`,
		staging, t.Name,
	)); err != nil {
		return res, fmt.Errorf("create staging table: %w", err)
	}
	defer func() {
		// Best-effort: the session ends with the connection release anyway.
		_, _ = conn.Exec(context.WithoutCancel(ctx), "DROP TABLE IF EXISTS "+staging)
	}()

	// pgx's CopyFrom pulls from this source as it writes, so rows are produced
	// lazily and the whole member is never buffered.
	rows := &copySource{
		ctx:    ctx,
		src:    src,
		table:  t,
		result: &res,
	}

	written, err := conn.CopyFrom(ctx, pgx.Identifier{staging}, t.Columns, rows)
	if err != nil {
		if rows.err != nil {
			return res, rows.err
		}
		return res, fmt.Errorf("copy into staging: %w", err)
	}
	res.Written = written

	if _, err := conn.Exec(ctx, mergeSQL(t, staging)); err != nil {
		return res, fmt.Errorf("merge into %s: %w", t.Name, err)
	}

	res.Duration = time.Since(start)
	res.Elapsed = res.Duration.Round(time.Millisecond).String()
	return res, nil
}

// mergeSQL builds the upsert from staging into the destination.
//
// Columns that are part of the primary key are excluded from the SET clause:
// they are the conflict target, so assigning them is redundant and Postgres
// rejects it for some column types.
func mergeSQL(t Table, staging string) string {
	pk := make(map[string]bool, len(t.PK))
	for _, c := range t.PK {
		pk[c] = true
	}

	var sets []string
	for _, c := range t.Columns {
		if pk[c] {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", c, c))
	}

	cols := strings.Join(t.Columns, ", ")

	// DISTINCT ON guards against duplicate keys within a single archive member.
	// Without it a repeated _key aborts the whole statement with
	// "ON CONFLICT DO UPDATE command cannot affect row a second time".
	sel := fmt.Sprintf("SELECT DISTINCT ON (%s) %s FROM %s",
		strings.Join(t.PK, ", "), cols, staging)

	if len(sets) == 0 {
		// A table whose every column is part of the key — nothing to update.
		return fmt.Sprintf("INSERT INTO public.%s (%s) %s ON CONFLICT (%s) DO NOTHING",
			t.Name, cols, sel, strings.Join(t.PK, ", "))
	}

	return fmt.Sprintf("INSERT INTO public.%s (%s) %s ON CONFLICT (%s) DO UPDATE SET %s",
		t.Name, cols, sel, strings.Join(t.PK, ", "), strings.Join(sets, ", "))
}

// copySource adapts streaming JSONL to pgx.CopyFromSource.
//
// pgx pulls one row at a time via Next/Values, but Source.Stream pushes through
// a callback. Rather than run a goroutine and channel, Stream is driven once on
// the first Next call and rows are buffered — bounded by the member's row count,
// which peaks at 645 k narrow rows and is far cheaper than the coordination and
// error-propagation complexity of a producer goroutine.
type copySource struct {
	ctx    context.Context
	src    *Source
	table  Table
	result *LoadResult

	loaded bool
	buf    [][]any
	idx    int
	err    error
}

func (c *copySource) Next() bool {
	if !c.loaded {
		c.loaded = true
		c.err = c.src.Stream(c.ctx, c.table.Member, func(r Row) error {
			c.result.Read++

			if c.table.Expand != nil {
				rows := c.table.Expand(r)
				if len(rows) == 0 {
					c.result.Skipped++
					return nil
				}
				for _, vals := range rows {
					if len(vals) != len(c.table.Columns) {
						return fmt.Errorf("table %s: Expand returned %d items for %d columns",
							c.table.Name, len(vals), len(c.table.Columns))
					}
					c.buf = append(c.buf, vals)
				}
				return nil
			}

			vals, ok := c.table.Values(r)
			if !ok {
				c.result.Skipped++
				return nil
			}
			if len(vals) != len(c.table.Columns) {
				return fmt.Errorf("table %s: Values returned %d items for %d columns",
					c.table.Name, len(vals), len(c.table.Columns))
			}
			c.buf = append(c.buf, vals)
			return nil
		})
		if c.err != nil {
			return false
		}
	}
	if c.idx >= len(c.buf) {
		return false
	}
	c.idx++
	return true
}

func (c *copySource) Values() ([]any, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.buf[c.idx-1], nil
}

func (c *copySource) Err() error { return c.err }
