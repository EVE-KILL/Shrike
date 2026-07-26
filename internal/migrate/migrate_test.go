package migrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
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
	if len(st.Pending()) != 1 || st.Pending()[0].Version != 2 {
		t.Fatalf("Pending = %+v; want only post-baseline migration 2", st.Pending())
	}

	if err := Baseline(ctx, pool); !errors.Is(err, ErrAlreadyBaselined) {
		t.Fatalf("second Baseline error = %v; want ErrAlreadyBaselined", err)
	}
}

// After baselining, Apply must be a safe no-op rather than executing migration 1.
func TestApplyAfterBaselineDoesNotRunBaseline(t *testing.T) {
	ctx := context.Background()
	pool := scratchDB(t, "applyafterbaseline")

	if _, err := pool.Exec(ctx, `
		CREATE TABLE pretend_existing (id int primary key);
		CREATE TABLE custom_domain_campaigns (
			public_on_domain boolean DEFAULT false NOT NULL
		);
	`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	if err := Baseline(ctx, pool); err != nil {
		t.Fatalf("Baseline: %v", err)
	}
	if err := Apply(ctx, pool); err != nil {
		t.Fatalf("Apply after baseline: %v", err)
	}

	// Migration 1 creates ~102 tables. Migration 2 adopts the column already
	// applied by TypeScript and records itself without changing its definition.
	var n int
	if err := pool.QueryRow(ctx, `
        SELECT count(*) FROM information_schema.tables
        WHERE table_schema='public' AND table_type='BASE TABLE'
          AND table_name <> 'goose_db_version'
    `).Scan(&n); err != nil {
		t.Fatalf("recount: %v", err)
	}
	if n != 2 {
		t.Fatalf("table count = %d; want 2 — the baseline was executed despite being stamped", n)
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

// Goose intentionally replaces Drizzle as the migration mechanism. That is
// safe only while it produces the same application schema. Build both histories
// in disposable databases and compare PostgreSQL's normalized catalog output,
// excluding only the migration ledgers owned by each mechanism.
func TestGooseSchemaMatchesTypeScriptMigrations(t *testing.T) {
	ctx := context.Background()
	goosePool := scratchDB(t, "goose_schema")
	typeScriptPool := scratchDB(t, "typescript_schema")

	if err := Apply(ctx, goosePool); err != nil {
		t.Fatalf("apply Goose schema: %v", err)
	}
	applyTypeScriptSchema(t, ctx, typeScriptPool)

	gooseObjects := applicationSchemaObjects(t, ctx, goosePool)
	typeScriptObjects := applicationSchemaObjects(t, ctx, typeScriptPool)

	var differences []string
	for key, gooseDefinition := range gooseObjects {
		typeScriptDefinition, ok := typeScriptObjects[key]
		switch {
		case !ok:
			differences = append(differences, "- TypeScript missing "+key)
		case typeScriptDefinition != gooseDefinition:
			differences = append(differences,
				fmt.Sprintf("- %s\n  Goose:     %s\n  TypeScript: %s",
					key, gooseDefinition, typeScriptDefinition))
		}
	}
	for key := range typeScriptObjects {
		if _, ok := gooseObjects[key]; !ok {
			differences = append(differences, "- Goose missing "+key)
		}
	}
	sort.Strings(differences)
	if len(differences) > 0 {
		t.Fatalf("Goose and TypeScript application schemas differ:\n%s",
			strings.Join(differences, "\n"))
	}
}

func applyTypeScriptSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	drizzleDir := filepath.Join("..", "..", "..", "backend", "drizzle")
	manifestBytes, err := os.ReadFile(filepath.Join(
		drizzleDir,
		"baseline-2026-07-21.json",
	))
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("TypeScript backend migration bundle is not present beside the Go module")
	}
	if err != nil {
		t.Fatalf("read TypeScript migration manifest: %v", err)
	}

	var manifest struct {
		Through string `json:"through"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("decode TypeScript migration manifest: %v", err)
	}

	bootstrap, err := os.ReadFile(filepath.Join(
		drizzleDir,
		"bootstrap",
		"2026-07-21.sql",
	))
	if err != nil {
		t.Fatalf("read TypeScript bootstrap: %v", err)
	}
	if _, err := pool.Exec(
		ctx,
		stripPSQLMetaCommands(string(bootstrap)),
		pgx.QueryExecModeSimpleProtocol,
	); err != nil {
		t.Fatalf("apply TypeScript bootstrap: %v", err)
	}
	// pg_dump deliberately empties search_path for its own session. The
	// post-baseline migrations are run later by the backend on ordinary
	// application connections, whose search_path resolves unqualified objects
	// in public.
	if _, err := pool.Exec(ctx, `SET search_path TO public`); err != nil {
		t.Fatalf("restore application search path: %v", err)
	}

	entries, err := os.ReadDir(drizzleDir)
	if err != nil {
		t.Fatalf("list TypeScript migrations: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".sql" || name <= manifest.Through {
			continue
		}
		sql, err := os.ReadFile(filepath.Join(drizzleDir, name))
		if err != nil {
			t.Fatalf("read TypeScript migration %s: %v", name, err)
		}
		for _, statement := range splitTypeScriptStatements(string(sql)) {
			if _, err := pool.Exec(
				ctx,
				statement,
				pgx.QueryExecModeSimpleProtocol,
			); err != nil {
				t.Fatalf("apply TypeScript migration %s: %v", name, err)
			}
		}
	}
}

func stripPSQLMetaCommands(sql string) string {
	lines := strings.Split(sql, "\n")
	out := lines[:0]
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), `\`) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func splitTypeScriptStatements(sql string) []string {
	parts := strings.Split(sql, "--> statement-breakpoint")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if statement := strings.TrimSpace(part); statement != "" {
			out = append(out, statement)
		}
	}
	return out
}

func applicationSchemaObjects(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) map[string]string {
	t.Helper()

	const query = `
		WITH excluded_relations(name) AS (
			VALUES
				('goose_db_version'::text),
				('__evekill_schema_migrations'::text),
				('__drizzle_migrations'::text)
		)
		SELECT kind, object_name, definition
		FROM (
			SELECT
				'column'::text AS kind,
				c.relname || '.' || a.attname AS object_name,
				concat_ws('|',
					format_type(a.atttypid, a.atttypmod),
					CASE WHEN a.attnotnull THEN 'not-null' ELSE 'nullable' END,
					CASE
						WHEN a.attidentity = '' THEN NULL
						ELSE 'identity=' || a.attidentity::text
					END,
					CASE
						WHEN a.attgenerated = '' THEN NULL
						ELSE 'generated=' || a.attgenerated::text
					END,
					pg_get_expr(d.adbin, d.adrelid),
					CASE
						WHEN a.attcollation = 0 THEN NULL
						ELSE coll.collname
					END
				) AS definition
			FROM pg_attribute a
			JOIN pg_class c ON c.oid = a.attrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace
			LEFT JOIN pg_attrdef d
				ON d.adrelid = a.attrelid AND d.adnum = a.attnum
			LEFT JOIN pg_collation coll ON coll.oid = a.attcollation
			WHERE n.nspname = 'public'
			  AND c.relkind IN ('r', 'p')
			  AND a.attnum > 0
			  AND NOT a.attisdropped
			  AND c.relname NOT IN (SELECT name FROM excluded_relations)

			UNION ALL

			SELECT
				'constraint',
				c.relname || '.' || con.conname,
				pg_get_constraintdef(con.oid, true)
			FROM pg_constraint con
			JOIN pg_class c ON c.oid = con.conrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'public'
			  AND c.relname NOT IN (SELECT name FROM excluded_relations)

			UNION ALL

			SELECT
				'index',
				table_class.relname || '.' || index_class.relname,
				pg_get_indexdef(index_class.oid)
			FROM pg_index index_meta
			JOIN pg_class index_class ON index_class.oid = index_meta.indexrelid
			JOIN pg_class table_class ON table_class.oid = index_meta.indrelid
			JOIN pg_namespace n ON n.oid = table_class.relnamespace
			WHERE n.nspname = 'public'
			  AND table_class.relname NOT IN (SELECT name FROM excluded_relations)

			UNION ALL

			SELECT
				'function',
				p.proname || '(' || pg_get_function_identity_arguments(p.oid) || ')',
				pg_get_functiondef(p.oid)
			FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			LEFT JOIN pg_depend dep
				ON dep.classid = 'pg_proc'::regclass
				AND dep.objid = p.oid
				AND dep.deptype = 'e'
			WHERE n.nspname = 'public'
			  AND dep.objid IS NULL

			UNION ALL

			SELECT
				'trigger',
				c.relname || '.' || trigger.tgname,
				pg_get_triggerdef(trigger.oid, true)
			FROM pg_trigger trigger
			JOIN pg_class c ON c.oid = trigger.tgrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'public'
			  AND NOT trigger.tgisinternal
			  AND c.relname NOT IN (SELECT name FROM excluded_relations)

			UNION ALL

			SELECT
				'sequence',
				c.relname,
				concat_ws('|',
					format_type(sequence.seqtypid, NULL),
					sequence.seqstart,
					sequence.seqincrement,
					sequence.seqmax,
					sequence.seqmin,
					sequence.seqcache,
					sequence.seqcycle
				)
			FROM pg_sequence sequence
			JOIN pg_class c ON c.oid = sequence.seqrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'public'
			  AND c.relname NOT IN (
				'goose_db_version_id_seq',
				'__evekill_schema_migrations_id_seq',
				'__drizzle_migrations_id_seq'
			  )

			UNION ALL

			SELECT
				'view',
				c.relname,
				pg_get_viewdef(c.oid, true)
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'public'
			  AND c.relkind IN ('v', 'm')
		) objects
		ORDER BY kind, object_name
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		t.Fatalf("query normalized schema: %v", err)
	}
	defer rows.Close()

	objects := make(map[string]string)
	for rows.Next() {
		var kind, name, definition string
		if err := rows.Scan(&kind, &name, &definition); err != nil {
			t.Fatalf("scan normalized schema: %v", err)
		}
		objects[kind+":"+name] = definition
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read normalized schema: %v", err)
	}
	return objects
}
