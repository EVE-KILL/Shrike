package everef

import (
	"context"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/configstore"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- Market history ---

func clearPriceDay(t *testing.T, pool *pgxpool.Pool, date string, typeIDs []int32) {
	t.Helper()
	cleanup := func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM prices WHERE date = $1::date AND type_id = ANY($2::int[])`, date, typeIDs)
	}
	cleanup()
	t.Cleanup(cleanup)
}

// The fixture holds one day shaped like the real file: two ordinary rows, a row
// priced at zero, a row with no price at all, rows with empty optional columns,
// a count expressed as a float, and a row from another region.
func TestImportPriceDayFiltersAndCoerces(t *testing.T) {
	pool := testPool(t)
	const date = "2026-07-20"
	typeIDs := []int32{34, 35, 36, 587, 888888, 999999}
	clearPriceDay(t, pool, date, typeIDs)

	client := fileServer(t, map[string]string{
		"/market-history/2026/market-history-2026-07-20.csv.bz2": "market-history-2026-07-20.csv.bz2",
	})
	ctx := context.Background()

	res, err := ImportPriceDay(ctx, pool, client, date)
	if err != nil {
		t.Fatal(err)
	}
	// Seven rows in the file: one is another region, one has a zero average and
	// one has none, leaving four.
	if res.Rows != 4 {
		t.Errorf("imported %d rows, want 4", res.Rows)
	}
	if res.Missing {
		t.Error("a published day was reported as missing")
	}

	// Only The Forge is stored: keeping the other hundred-odd regions would
	// multiply the table tenfold for data nothing reads.
	var otherRegions int64
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM prices WHERE date = $1::date AND region_id <> $2`,
		date, RegionTheForge).Scan(&otherRegions); err != nil {
		t.Fatal(err)
	}
	if otherRegions != 0 {
		t.Errorf("%d rows from other regions were stored", otherRegions)
	}

	// A row with no average is a day the type was listed but never traded.
	// Storing it would put a zero into the price history, and "latest average
	// at or before this date" would then value the item at nothing.
	for _, id := range []int32{999999, 888888} {
		var n int64
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM prices WHERE date = $1::date AND type_id = $2`, date, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("type %d has a row despite no usable average", id)
		}
	}

	var average float64
	var highest, lowest *float64
	var orderCount, volume *int64
	if err := pool.QueryRow(ctx, `
        SELECT average, highest, lowest, order_count, volume
        FROM prices WHERE date = $1::date AND type_id = 34`, date).
		Scan(&average, &highest, &lowest, &orderCount, &volume); err != nil {
		t.Fatal(err)
	}
	if average != 5.50 {
		t.Errorf("average = %v, want 5.50", average)
	}
	if highest == nil || *highest != 6.00 {
		t.Errorf("highest = %v", highest)
	}
	if orderCount == nil || *orderCount != 12 {
		t.Errorf("order_count = %v", orderCount)
	}

	// Empty optional columns become NULL, matching what production holds.
	if err := pool.QueryRow(ctx, `
        SELECT order_count, volume FROM prices WHERE date = $1::date AND type_id = 35`, date).
		Scan(&orderCount, &volume); err != nil {
		t.Fatal(err)
	}
	if orderCount != nil || volume != nil {
		t.Errorf("empty columns stored as %v / %v, want NULL", orderCount, volume)
	}

	// Counts occasionally arrive as floats.
	if err := pool.QueryRow(ctx, `
        SELECT order_count FROM prices WHERE date = $1::date AND type_id = 36`, date).
		Scan(&orderCount); err != nil {
		t.Fatal(err)
	}
	if orderCount == nil || *orderCount != 7 {
		t.Errorf("a float order_count parsed to %v, want 7", orderCount)
	}
}

// Re-importing a day must not change what is stored. EVE Ref occasionally
// republishes a corrected file, and adopting it would silently reprice history
// that stored killmail values were already computed from.
func TestImportPriceDayKeepsTheFirstVersion(t *testing.T) {
	pool := testPool(t)
	const date = "2026-07-20"
	typeIDs := []int32{34, 35, 36, 587, 888888, 999999}
	clearPriceDay(t, pool, date, typeIDs)

	client := fileServer(t, map[string]string{
		"/market-history/2026/market-history-2026-07-20.csv.bz2": "market-history-2026-07-20.csv.bz2",
	})
	ctx := context.Background()

	if _, err := ImportPriceDay(ctx, pool, client, date); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE prices SET average = 1 WHERE date = $1::date AND type_id = 34`, date); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportPriceDay(ctx, pool, client, date); err != nil {
		t.Fatal(err)
	}

	var average float64
	var rows int64
	if err := pool.QueryRow(ctx, `
        SELECT count(*), max(average) FROM prices
        WHERE date = $1::date AND type_id = 34`, date).Scan(&rows, &average); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Errorf("re-importing duplicated the row (%d copies)", rows)
	}
	if average != 1 {
		t.Errorf("re-importing overwrote the stored price (%v) — history would be silently repriced", average)
	}
}

// A day EVE Ref has not published is reported, not raised: the most recent day
// is routinely absent for several hours and an import must carry on.
func TestImportPriceDayReportsAnUnpublishedDay(t *testing.T) {
	pool := testPool(t)
	client := fileServer(t, map[string]string{}) // serves 404 for everything

	res, err := ImportPriceDay(context.Background(), pool, client, "2026-07-21")
	if err != nil {
		t.Fatalf("an unpublished day returned an error: %v", err)
	}
	if !res.Missing {
		t.Error("an unpublished day was not reported as missing")
	}
	if res.Rows != 0 {
		t.Errorf("wrote %d rows for a day that does not exist", res.Rows)
	}
}

// The bookmark is what a resumed import reads, so it must not advance past a day
// that was never imported.
func TestImportPricesBookmarksOnlyPublishedDays(t *testing.T) {
	pool := testPool(t)
	const date = "2026-07-20"
	typeIDs := []int32{34, 35, 36, 587, 888888, 999999}
	clearPriceDay(t, pool, date, typeIDs)
	ctx := context.Background()

	before, err := configstore.Get(ctx, pool, configstore.KeyPricesLastDate)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = configstore.Set(context.Background(), pool, configstore.KeyPricesLastDate, before)
	})

	client := fileServer(t, map[string]string{
		"/market-history/2026/market-history-2026-07-20.csv.bz2": "market-history-2026-07-20.csv.bz2",
	})

	// The second day is not published.
	total, err := ImportPrices(ctx, pool, client, []string{date, "2026-07-21"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if total.Failed != 1 {
		t.Errorf("failed = %d, want 1 unpublished day", total.Failed)
	}

	got, err := configstore.Get(ctx, pool, configstore.KeyPricesLastDate)
	if err != nil {
		t.Fatal(err)
	}
	if got != date {
		t.Errorf("bookmark = %q, want %q — it must not advance past a day that was never imported", got, date)
	}
}

// --- Insurance ---

func TestImportInsuranceReplacesTheSnapshot(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	// Snapshot the real table so the test can restore it: insurance is a full
	// replace, so there is no way to scope this to reserved ids.
	var before int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM insurance_prices`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before == 0 {
		t.Skip("no insurance data to restore afterwards")
	}
	saved, err := pool.Query(ctx, `SELECT type_id, level_name, cost, payout FROM insurance_prices`)
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		typeID    int32
		level     string
		cost, pay float64
	}
	var rows []row
	for saved.Next() {
		var r row
		if err := saved.Scan(&r.typeID, &r.level, &r.cost, &r.pay); err != nil {
			saved.Close()
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	saved.Close()

	t.Cleanup(func() {
		ctx := context.Background()
		if _, err := pool.Exec(ctx, `DELETE FROM insurance_prices`); err != nil {
			t.Fatalf("restore insurance: %v", err)
		}
		for _, r := range rows {
			if _, err := pool.Exec(ctx,
				`INSERT INTO insurance_prices (type_id, level_name, cost, payout) VALUES ($1,$2,$3,$4)
                 ON CONFLICT DO NOTHING`, r.typeID, r.level, r.cost, r.pay); err != nil {
				t.Fatalf("restore insurance: %v", err)
			}
		}
	})

	client := jsonServer(t, map[string]string{
		"/insurance-prices/insurance-prices-latest.json": `[
            {"type_id": 587, "levels": [
                {"name": "Basic", "cost": 100.5, "payout": 1000.5},
                {"name": "Platinum", "cost": 900.0, "payout": 5000.0}
            ]},
            {"type_id": 588, "levels": [{"name": "Basic", "cost": 1.0, "payout": 2.0}]}
        ]`,
	})

	res, err := ImportInsurance(ctx, pool, client)
	if err != nil {
		t.Fatal(err)
	}
	if res.Rows != 3 {
		t.Errorf("wrote %d rows, want 3", res.Rows)
	}

	// A replace, not a merge: a policy CCP withdrew has to disappear, and an
	// upsert would leave it behind forever.
	var total int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM insurance_prices`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Errorf("the table holds %d rows after a 3-row snapshot — it was merged, not replaced", total)
	}

	var cost, payout float64
	if err := pool.QueryRow(ctx,
		`SELECT cost, payout FROM insurance_prices WHERE type_id = 587 AND level_name = 'Platinum'`).
		Scan(&cost, &payout); err != nil {
		t.Fatal(err)
	}
	if cost != 900.0 || payout != 5000.0 {
		t.Errorf("stored cost=%v payout=%v", cost, payout)
	}
}

// An empty response is far likelier to be EVE Ref having a bad day than CCP
// withdrawing every insurance policy in the game. Truncating on it would leave
// the killboard unable to value a loss until the next successful run.
func TestImportInsuranceRefusesAnEmptySnapshot(t *testing.T) {
	pool := testPool(t)
	client := jsonServer(t, map[string]string{
		"/insurance-prices/insurance-prices-latest.json": `[]`,
	})

	before, err := countRows(t, pool, "insurance_prices")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ImportInsurance(context.Background(), pool, client); err == nil {
		t.Fatal("an empty snapshot was accepted")
	}

	after, err := countRows(t, pool, "insurance_prices")
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Errorf("the table went from %d to %d rows on a refused import", before, after)
	}
}

func countRows(t *testing.T, pool *pgxpool.Pool, table string) (int64, error) {
	t.Helper()
	var n int64
	err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table).Scan(&n)
	return n, err
}

// --- Archives ---

// A war archive holds two kinds of document mixed together, plus files that are
// neither. The walk has to skip what it cannot use rather than abandon the
// archive: these are machine-generated and occasionally truncated, and one bad
// file out of twenty thousand is not a reason to lose a day.
func TestWalkArchiveSkipsUnusableMembers(t *testing.T) {
	client := fileServer(t, map[string]string{
		"/wars/history/2026/wars-2026-07-20.tar.bz2": "wars-2026-07-20.tar.bz2",
	})

	var names []string
	var decoded, undecodable int
	err := client.WalkArchive(context.Background(),
		client.url("/wars/history/2026/wars-2026-07-20.tar.bz2"),
		func(name string, data []byte) error {
			names = append(names, name)
			var probe map[string]any
			if decodeMember(data, &probe) {
				decoded++
			} else {
				undecodable++
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}

	// Four .json members; the .txt is not offered at all.
	if len(names) != 4 {
		t.Errorf("walked %d members, want 4: %v", len(names), names)
	}
	for _, name := range names {
		if strings.HasSuffix(name, ".txt") {
			t.Errorf("a non-JSON member was offered: %s", name)
		}
	}
	if decoded != 3 {
		t.Errorf("decoded %d members, want 3", decoded)
	}
	if undecodable != 1 {
		t.Errorf("%d members failed to decode, want the single truncated one", undecodable)
	}
}

// The path is how a war id is recovered from older archives, which state it
// nowhere else.
func TestWarIDFromPath(t *testing.T) {
	cases := map[string]int32{
		"./wars/700001/killmails/90000001.json": 700001,
		"wars/12345/killmails/1.json":           12345,
		"./wars/700001.json":                    0,
		"nonsense":                              0,
	}
	for input, want := range cases {
		if got := warIDFromPath(input); got != want {
			t.Errorf("warIDFromPath(%q) = %d, want %d", input, got, want)
		}
	}
}
