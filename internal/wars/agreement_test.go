package wars_test

import (
	"context"
	"os"
	"testing"

	"github.com/eve-kill/shrike/internal/wars"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Two implementations, one answer.
//
// Contributions runs per killmail as it arrives and cannot see the whole war.
// Rebuild runs set-based over every killmail at once and cannot run per
// killmail. Both write war_interactions, and a war page shows whichever ran
// last — so if they disagree, the numbers change depending on whether a war was
// live-aggregated or repaired, which is indistinguishable from corruption.
//
// This is the test that keeps them honest: rebuild the table, snapshot it,
// replay the same killmails one at a time through the incremental path, and
// require the two to be identical.

const agreementDSN = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"

func agreementPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = agreementDSN
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type interactionRow struct {
	WarID      int32
	Side       int16
	Category   int16
	TargetType int16
	TargetID   int32
	Count      int32
	ISK        float64
}

func snapshot(t *testing.T, ctx context.Context, q pgx.Tx) map[interactionRow]bool {
	t.Helper()
	rows, err := q.Query(ctx, `
        SELECT war_id, side, category, target_type, target_id, count, isk_value
        FROM war_interactions`)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	defer rows.Close()

	out := map[interactionRow]bool{}
	for rows.Next() {
		var r interactionRow
		if err := rows.Scan(&r.WarID, &r.Side, &r.Category,
			&r.TargetType, &r.TargetID, &r.Count, &r.ISK); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[r] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	return out
}

func TestIncrementalAgreesWithRebuild(t *testing.T) {
	ctx := context.Background()
	pool := agreementPool(t)

	var warKills int64
	if err := pool.QueryRow(ctx, `
        SELECT count(*) FROM killmails k JOIN wars w ON w.war_id = k.war_id`).
		Scan(&warKills); err != nil {
		t.Fatalf("count war killmails: %v", err)
	}
	if warKills == 0 {
		t.Skip("no war killmails in the database to compare against")
	}

	// Everything happens in one transaction that is always rolled back, so the
	// real table is never modified — this test reads production-shaped data and
	// must not be able to damage it.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // deliberate: never commit

	// The set-based answer, computed inside the transaction.
	if err := rebuildInTx(ctx, tx); err != nil {
		t.Fatalf("set-based rebuild: %v", err)
	}
	setBased := snapshot(t, ctx, tx)
	if len(setBased) == 0 {
		t.Skip("the rebuild produced no rows")
	}

	// Now the incremental answer over the same killmails.
	if _, err := tx.Exec(ctx, `DELETE FROM war_interactions`); err != nil {
		t.Fatalf("clear: %v", err)
	}

	rows, err := tx.Query(ctx, `
        SELECT k.killmail_id FROM killmails k
        JOIN wars w ON w.war_id = k.war_id
        ORDER BY k.killmail_id`)
	if err != nil {
		t.Fatalf("list war killmails: %v", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("scan id: %v", err)
		}
		ids = append(ids, id)
	}
	rows.Close()

	for _, id := range ids {
		if _, err := wars.AggregateKillmail(ctx, tx, id); err != nil {
			t.Fatalf("aggregate killmail %d: %v", id, err)
		}
	}
	incremental := snapshot(t, ctx, tx)

	// Compared on the logical key, so a row present in both but counted
	// differently reports as one disagreement rather than as two unrelated rows.
	type key struct {
		WarID                      int32
		Side, Category, TargetType int16
		TargetID                   int32
	}
	index := func(rows map[interactionRow]bool) map[key]interactionRow {
		out := make(map[key]interactionRow, len(rows))
		for r := range rows {
			out[key{r.WarID, r.Side, r.Category, r.TargetType, r.TargetID}] = r
		}
		return out
	}
	a, b := index(setBased), index(incremental)

	var missing, extra, differing int
	for k, want := range a {
		got, ok := b[k]
		if !ok {
			missing++
			if missing <= 5 {
				t.Errorf("only the rebuild produced %+v", want)
			}
			continue
		}
		if got.Count != want.Count || !iskEqual(got.ISK, want.ISK) {
			differing++
			if differing <= 8 {
				t.Errorf("war %d side %d cat %d target %d/%d: rebuild says %d kills / %v ISK, "+
					"incremental says %d / %v",
					k.WarID, k.Side, k.Category, k.TargetType, k.TargetID,
					want.Count, want.ISK, got.Count, got.ISK)
			}
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			extra++
			if extra <= 5 {
				t.Errorf("only the incremental path produced %+v", b[k])
			}
		}
	}

	if missing > 0 || extra > 0 || differing > 0 {
		t.Fatalf("the two aggregations disagree over %d killmails: %d rows only in the "+
			"rebuild, %d only in the incremental path, %d counted differently — a war's "+
			"numbers would change depending on which path last touched it",
			len(ids), missing, extra, differing)
	}

	t.Logf("both paths agree exactly: %d rows across %d war killmails", len(setBased), len(ids))
}

// iskEqual compares two ISK totals for practical equality.
//
// isk_value is double precision and the two paths reach the same total by
// different routes: the rebuild sums a group in one SQL aggregate, the
// incremental path adds one killmail at a time over hundreds of transactions.
// Floating-point addition is not associative, so the last bit or two differ on
// any row with enough kills behind it. That is arithmetic, not disagreement.
//
// The tolerance is relative because the values span from thousands to hundreds
// of billions, and a fixed epsilon would be either meaningless at the top of
// that range or useless at the bottom. Anything above it is a real difference:
// a single miscounted killmail moves a total by millions, which is many orders
// of magnitude beyond accumulated rounding.
func iskEqual(a, b float64) bool {
	if a == b {
		return true
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	scale := a
	if scale < 0 {
		scale = -scale
	}
	if scale < 1 {
		return diff < 1e-6
	}
	return diff/scale < 1e-9
}

// rebuildInTx is the set-based aggregation, run inside a caller's transaction.
//
// Rebuild takes its own transaction and an exclusive lock, neither of which a
// test that must roll back can use. The SQL is the same.
func rebuildInTx(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM war_interactions`); err != nil {
		return err
	}
	return wars.AggregateAllInTx(ctx, tx)
}
