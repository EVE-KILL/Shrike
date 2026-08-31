package killmail_test

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestKillListPredicatePerformance runs the page-shaped first-chunk query for
// every registered kill-list type. It is opt-in because EXPLAIN ANALYZE executes
// the query and is intended for a production-like read replica, not unit tests.
//
//	RUN_QUERY_PERFORMANCE=1 TEST_DATABASE_URL=... go test ./internal/killmail \
//	  -run TestKillListPredicatePerformance -count=1 -v
func TestKillListPredicatePerformance(t *testing.T) {
	if os.Getenv("RUN_QUERY_PERFORMANCE") != "1" {
		t.Skip("RUN_QUERY_PERFORMANCE is not set")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	types := append([]string(nil), killtype.Types...)
	sort.Strings(types)
	for _, kind := range types {
		predicate := killtype.Predicates()[kind]
		query := fmt.Sprintf(`EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
			SELECT k.killmail_id
			FROM killmails k
			WHERE k.killmail_time >= current_date - interval '1 day'
			  AND k.killmail_time < current_date
			  AND %s
			ORDER BY k.killmail_time DESC, k.killmail_id DESC
			LIMIT 50`, predicate)

		rows, err := pool.Query(ctx, query)
		if err != nil {
			t.Errorf("%s: explain: %v", kind, err)
			continue
		}
		var plan []string
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				rows.Close()
				t.Fatalf("%s: scan plan: %v", kind, err)
			}
			plan = append(plan, line)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Errorf("%s: read plan: %v", kind, err)
			continue
		}

		executionMS := explainExecutionMS(plan)
		t.Logf("%-24s %8.3f ms", kind, executionMS)
		if executionMS < 0 {
			t.Errorf("%s: execution time missing from plan", kind)
		}
		if executionMS > 1_000 {
			t.Errorf("%s: daily first-page query took %.3f ms\n%s",
				kind, executionMS, strings.Join(plan, "\n"))
		}
	}
}

// TestDailyCountBackfillPerformance measures the aggregation shape used by one
// month/type backfill partition. Run separately from the page audit because it
// scans the complete previous month for every type.
func TestDailyCountBackfillPerformance(t *testing.T) {
	if os.Getenv("RUN_BACKFILL_PERFORMANCE") != "1" {
		t.Skip("RUN_BACKFILL_PERFORMANCE is not set")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	types := append([]string(nil), killtype.Types...)
	sort.Strings(types)
	for _, kind := range types {
		predicate := killtype.Predicates()[kind]
		query := fmt.Sprintf(`EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
			SELECT (k.killmail_time AT TIME ZONE 'UTC')::date, count(*)
			FROM killmails k
			WHERE k.killmail_time >= date_trunc('month', current_date) - interval '1 month'
			  AND k.killmail_time < date_trunc('month', current_date)
			  AND %s
			GROUP BY (k.killmail_time AT TIME ZONE 'UTC')::date`, predicate)

		rows, err := pool.Query(ctx, query)
		if err != nil {
			t.Errorf("%s: explain: %v", kind, err)
			continue
		}
		var plan []string
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				rows.Close()
				t.Fatalf("%s: scan plan: %v", kind, err)
			}
			plan = append(plan, line)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Errorf("%s: read plan: %v", kind, err)
			continue
		}

		executionMS := explainExecutionMS(plan)
		t.Logf("%-24s %8.3f ms", kind, executionMS)
		if executionMS < 0 {
			t.Errorf("%s: execution time missing from plan", kind)
		}
		if executionMS > 5_000 {
			t.Errorf("%s: monthly rollup query took %.3f ms\n%s",
				kind, executionMS, strings.Join(plan, "\n"))
		}
	}
}

func explainExecutionMS(plan []string) float64 {
	for _, line := range plan {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Execution Time: ") {
			continue
		}
		raw := strings.TrimSuffix(strings.TrimPrefix(line, "Execution Time: "), " ms")
		value, err := strconv.ParseFloat(raw, 64)
		if err == nil {
			return value
		}
	}
	return -1
}
