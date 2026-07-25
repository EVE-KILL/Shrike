// Package migrate wraps goose with the safety rules this database needs.
//
// The hazard being guarded against is specific and expensive. Production already
// carries all 102 tables of migration 00001, created by drizzle and tracked in
// drizzle.__drizzle_migrations. Goose knows nothing about that ledger, so a plain
// `goose up` against production would try to CREATE TABLE over a live 285 GB
// schema. Everything here exists to make that impossible rather than merely
// discouraged.
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// BaselineVersion is the migration that captures the pre-existing schema. Any
// database that already has the schema must have exactly this version stamped,
// never executed.
const BaselineVersion int64 = 1

// ErrNeedsBaseline is returned when a database already contains the schema but
// has no goose ledger. Applying migrations in that state would attempt to
// recreate live tables.
var ErrNeedsBaseline = errors.New("database has an existing schema but no goose ledger")

// ErrAlreadyBaselined is returned when baselining a database that goose already
// tracks, which would otherwise silently rewrite history.
var ErrAlreadyBaselined = errors.New("database already has a goose ledger")

// ErrEmptySchema is returned when baselining a database with no schema. Stamping
// there would permanently skip table creation and leave an unusable database.
var ErrEmptySchema = errors.New("database has no existing schema to baseline")

// Migration is one entry in a migration plan.
type Migration struct {
	Version int64     `json:"version"`
	Source  string    `json:"source"`
	Applied bool      `json:"applied"`
	At      time.Time `json:"applied_at,omitzero"`
}

// State describes what goose knows about a database versus what exists on disk.
type State struct {
	HasLedger    bool        `json:"has_ledger"`
	HasSchema    bool        `json:"has_schema"`
	TableCount   int         `json:"table_count"`
	DBVersion    int64       `json:"db_version"`
	Migrations   []Migration `json:"migrations"`
	NeedsBaselin bool        `json:"needs_baseline"`
}

func init() {
	goose.SetBaseFS(migrations.FS)
	goose.SetDialect("postgres")
	// goose logs to stdout by default, which would corrupt --json output.
	goose.SetLogger(goose.NopLogger())
}

// sqlDB adapts the pgx pool to database/sql, which is the only interface goose
// accepts. It shares the pool's configuration rather than opening a second,
// separately-tuned connection path.
func sqlDB(pool *pgxpool.Pool) *sql.DB {
	return stdlib.OpenDBFromPool(pool)
}

// Inspect reports the migration state without changing anything. It is the basis
// for both `db:status` and the guard inside Apply.
func Inspect(ctx context.Context, pool *pgxpool.Pool) (*State, error) {
	st := &State{}

	// Count user tables, ignoring goose's own ledger and the two pre-existing
	// drizzle ledgers — their presence says nothing about whether the
	// application schema exists.
	err := pool.QueryRow(ctx, `
        SELECT count(*)
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_type = 'BASE TABLE'
          AND table_name NOT IN ('goose_db_version', '__evekill_schema_migrations')
    `).Scan(&st.TableCount)
	if err != nil {
		return nil, fmt.Errorf("count tables: %w", err)
	}
	st.HasSchema = st.TableCount > 0

	if err := pool.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema = 'public' AND table_name = 'goose_db_version'
        )
    `).Scan(&st.HasLedger); err != nil {
		return nil, fmt.Errorf("check ledger: %w", err)
	}

	// A schema that exists without a ledger is precisely the production case,
	// and the one that must never be migrated over.
	st.NeedsBaselin = st.HasSchema && !st.HasLedger

	sdb := sqlDB(pool)
	defer sdb.Close()

	if st.HasLedger {
		v, verr := goose.GetDBVersionContext(ctx, sdb)
		if verr != nil {
			return nil, fmt.Errorf("read db version: %w", verr)
		}
		st.DBVersion = v
	}

	// Collect what is on disk, marking each against the ledger.
	applied := map[int64]time.Time{}
	if st.HasLedger {
		rows, qerr := pool.Query(ctx, `
            SELECT version_id, tstamp FROM goose_db_version
            WHERE is_applied ORDER BY version_id
        `)
		if qerr != nil {
			return nil, fmt.Errorf("read ledger: %w", qerr)
		}
		defer rows.Close()
		for rows.Next() {
			var v int64
			var at time.Time
			if serr := rows.Scan(&v, &at); serr != nil {
				return nil, serr
			}
			applied[v] = at
		}
		if rows.Err() != nil {
			return nil, rows.Err()
		}
	}

	files, err := goose.CollectMigrations(".", 0, goose.MaxVersion)
	if err != nil {
		return nil, fmt.Errorf("collect migrations: %w", err)
	}
	for _, m := range files {
		at, ok := applied[m.Version]
		st.Migrations = append(st.Migrations, Migration{
			Version: m.Version,
			Source:  m.Source,
			Applied: ok,
			At:      at,
		})
	}
	return st, nil
}

// Apply runs every pending migration. It refuses when the database has a schema
// but no ledger, because that is the production shape and applying there would
// try to recreate live tables.
func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	st, err := Inspect(ctx, pool)
	if err != nil {
		return err
	}
	if st.NeedsBaselin {
		return fmt.Errorf("%w: found %d existing tables; run `shrike db:baseline --apply` first",
			ErrNeedsBaseline, st.TableCount)
	}

	sdb := sqlDB(pool)
	defer sdb.Close()
	return goose.UpContext(ctx, sdb, ".")
}

// Baseline stamps the baseline version as applied without executing it, which is
// how a database that already has the schema joins goose's history.
//
// It refuses on an empty database: stamping there would skip table creation
// forever and leave a database that can never be migrated into a working state.
func Baseline(ctx context.Context, pool *pgxpool.Pool) error {
	st, err := Inspect(ctx, pool)
	if err != nil {
		return err
	}
	if st.HasLedger {
		return fmt.Errorf("%w (at version %d)", ErrAlreadyBaselined, st.DBVersion)
	}
	if !st.HasSchema {
		return fmt.Errorf("%w: run `shrike db:migrate --apply` to create it instead", ErrEmptySchema)
	}

	sdb := sqlDB(pool)
	defer sdb.Close()

	// Creating the ledger through goose rather than by hand keeps the table
	// definition owned by the library, so it stays correct across upgrades.
	if _, err := goose.EnsureDBVersionContext(ctx, sdb); err != nil {
		return fmt.Errorf("create ledger: %w", err)
	}

	// EnsureDBVersion seeds version 0. Record the baseline as applied on top.
	if _, err := pool.Exec(ctx, `
        INSERT INTO goose_db_version (version_id, is_applied, tstamp)
        VALUES ($1, true, now())
    `, BaselineVersion); err != nil {
		return fmt.Errorf("stamp baseline: %w", err)
	}
	return nil
}

// Pending returns the migrations that Apply would run.
func (s *State) Pending() []Migration {
	var out []Migration
	for _, m := range s.Migrations {
		if !m.Applied {
			out = append(out, m)
		}
	}
	return out
}
