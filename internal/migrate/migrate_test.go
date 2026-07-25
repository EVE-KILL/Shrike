package migrate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests exercise the guards that stand between a `db:migrate` and the
// production schema. They need a real Postgres because the guards are queries
// against information_schema, not pure logic — mocking them would test nothing.
//
// Point TEST_DATABASE_URL at any throwaway superuser-capable Postgres; the
// local docker-compose stack is the default. Skipped when unreachable so
// `go test ./...` still passes without docker running.
const defaultTestDSN = "postgresql://evekill:evekill@127.0.0.1:5432/evekill"

func adminDSN(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return defaultTestDSN
}

// scratchDB creates an empty database and returns a pool for it. Each test gets
// its own so they cannot interfere, and it is dropped on cleanup.
func scratchDB(t *testing.T, name string) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	admin, err := pgxpool.New(ctx, adminDSN(t))
	if err != nil {
		t.Skipf("no test database available: %v", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		t.Skipf("no test database available: %v", err)
	}
	defer admin.Close()

	dbName := "evekill_migtest_" + name
	// Drop first: a previous crashed run may have left it behind.
	if _, err := admin.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)); err != nil {
		t.Fatalf("drop scratch db: %v", err)
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
		t.Fatalf("create scratch db: %v", err)
	}

	pool, err := pgxpool.New(ctx, swapDBName(adminDSN(t), dbName))
	if err != nil {
		t.Fatalf("connect to scratch db: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleanupAdmin, err := pgxpool.New(ctx, adminDSN(t))
		if err != nil {
			return
		}
		defer cleanupAdmin.Close()
		_, _ = cleanupAdmin.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
	})
	return pool
}

func swapDBName(dsn, name string) string {
	if i := strings.LastIndex(dsn, "/"); i >= 0 {
		// Preserve any query string (sslmode etc.) that follows the db name.
		rest := dsn[i+1:]
		if q := strings.Index(rest, "?"); q >= 0 {
			return dsn[:i+1] + name + rest[q:]
		}
		return dsn[:i+1] + name
	}
	return dsn
}

// The central safety property: a database that already has a schema but no
// goose ledger must never be migrated. That is production's exact shape, where
// migration 1 would try to CREATE TABLE over live data.
func TestApplyRefusesExistingSchemaWithoutLedger(t *testing.T) {
	ctx := context.Background()
	pool := scratchDB(t, "needsbaseline")

	// Stand in for the drizzle-created schema.
	if _, err := pool.Exec(ctx, `CREATE TABLE pretend_existing (id int primary key)`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}

	st, err := Inspect(ctx, pool)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if !st.NeedsBaselin {
		t.Fatalf("NeedsBaseline = false; want true (tables=%d, ledger=%v)", st.TableCount, st.HasLedger)
	}

	err = Apply(ctx, pool)
	if !errors.Is(err, ErrNeedsBaseline) {
		t.Fatalf("Apply error = %v; want ErrNeedsBaseline", err)
	}

	// And it must have changed nothing on the way out.
	var n int
	if err := pool.QueryRow(ctx, `
        SELECT count(*) FROM information_schema.tables
        WHERE table_schema='public' AND table_type='BASE TABLE'
    `).Scan(&n); err != nil {
		t.Fatalf("recount: %v", err)
	}
	if n != 1 {
		t.Fatalf("table count = %d; want 1 — Apply modified a schema it should have refused", n)
	}
}

// Baselining an empty database would stamp table creation as done and leave a
// database that can never be migrated into a working state.
func TestBaselineRefusesEmptyDatabase(t *testing.T) {
	ctx := context.Background()
	pool := scratchDB(t, "emptybaseline")

	err := Baseline(ctx, pool)
	if !errors.Is(err, ErrEmptySchema) {
		t.Fatalf("Baseline error = %v; want ErrEmptySchema", err)
	}
}

// Baselining twice would rewrite history and desynchronise the ledger.
func TestBaselineIsNotRepeatable(t *testing.T) {
	ctx := context.Background()
	pool := scratchDB(t, "doublebaseline")

	if _, err := pool.Exec(ctx, `CREATE TABLE pretend_existing (id int primary key)`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}

	if err := Baseline(ctx, pool); err != nil {
		t.Fatalf("first Baseline: %v", err)
	}

	st, err := Inspect(ctx, pool)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if st.DBVersion != BaselineVersion {
		t.Fatalf("DBVersion = %d; want %d", st.DBVersion, BaselineVersion)
	}
	if len(st.Pending()) != 0 {
		t.Fatalf("Pending = %d; want 0 after baselining", len(st.Pending()))
	}

	if err := Baseline(ctx, pool); !errors.Is(err, ErrAlreadyBaselined) {
		t.Fatalf("second Baseline error = %v; want ErrAlreadyBaselined", err)
	}
}

// After baselining, Apply must be a safe no-op rather than executing migration 1.
func TestApplyAfterBaselineDoesNotRunBaseline(t *testing.T) {
	ctx := context.Background()
	pool := scratchDB(t, "applyafterbaseline")

	if _, err := pool.Exec(ctx, `CREATE TABLE pretend_existing (id int primary key)`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	if err := Baseline(ctx, pool); err != nil {
		t.Fatalf("Baseline: %v", err)
	}
	if err := Apply(ctx, pool); err != nil {
		t.Fatalf("Apply after baseline: %v", err)
	}

	// Migration 1 creates ~102 tables. If it had run, this would not be 1.
	var n int
	if err := pool.QueryRow(ctx, `
        SELECT count(*) FROM information_schema.tables
        WHERE table_schema='public' AND table_type='BASE TABLE'
          AND table_name <> 'goose_db_version'
    `).Scan(&n); err != nil {
		t.Fatalf("recount: %v", err)
	}
	if n != 1 {
		t.Fatalf("table count = %d; want 1 — the baseline was executed despite being stamped", n)
	}
}

// The happy path: a genuinely fresh database gets the full schema.
func TestApplyOnFreshDatabaseCreatesSchema(t *testing.T) {
	ctx := context.Background()
	pool := scratchDB(t, "fresh")

	if err := Apply(ctx, pool); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx, `
        SELECT count(*) FROM information_schema.tables
        WHERE table_schema='public' AND table_type='BASE TABLE'
          AND table_name <> 'goose_db_version'
    `).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	// Exact count is asserted against production separately; here we only need
	// to know the baseline actually ran rather than silently doing nothing.
	if n < 100 {
		t.Fatalf("table count = %d; want the full schema (~101)", n)
	}

	var hasTrgm bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='pg_trgm')`).Scan(&hasTrgm); err != nil {
		t.Fatalf("check pg_trgm: %v", err)
	}
	if !hasTrgm {
		t.Fatal("pg_trgm missing — entity search would fail on a fresh database")
	}
}
