package fitting

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const fittingStoreTestDSN = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"

func fittingStorePool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = fittingStoreTestDSN
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

func TestStoreKeepsFirstFitPayloadAndRefreshesKillmailLink(t *testing.T) {
	pool := fittingStorePool(t)
	ctx := context.Background()

	const killmailID int64 = 2_140_003_000
	hashA, hashB := strings.Repeat("a", 64), strings.Repeat("b", 64)

	cleanup := func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM killmail_fittings WHERE killmail_id = $1`, killmailID)
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM fitting_items WHERE fit_hash = ANY($1::text[])`, []string{hashA, hashB})
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM fittings WHERE fit_hash = ANY($1::text[])`, []string{hashA, hashB})
	}
	cleanup()
	t.Cleanup(cleanup)

	first := &Fitting{
		FitHash: hashA, FamilyHash: hashA,
		Items: []ExtractedItem{{
			SlotGroup: SlotHigh, Ordinal: 0, TypeID: 500,
			ChargeTypeID: 600, Quantity: 1,
		}},
	}
	firstLink := Link{
		KillmailID: killmailID, ShipTypeID: 587,
		KillTime:         time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
		VictimAllianceID: 99_000_001, VictimCorporationID: 98_000_001,
	}
	if err := Store(ctx, pool, first, firstLink); err != nil {
		t.Fatal(err)
	}

	// Same hash, different non-hash payload. The first-seen representative
	// must win; otherwise charges and drones churn with every example.
	changed := &Fitting{
		FitHash: hashA, FamilyHash: hashA,
		Items: []ExtractedItem{{
			SlotGroup: SlotHigh, Ordinal: 0, TypeID: 501,
			ChargeTypeID: 601, Quantity: 9,
		}},
	}
	if err := Store(ctx, pool, changed, firstLink); err != nil {
		t.Fatal(err)
	}

	var itemType, chargeType, quantity int32
	if err := pool.QueryRow(ctx, `
        SELECT type_id, charge_type_id, quantity
        FROM fitting_items
        WHERE fit_hash = $1 AND slot_group = $2 AND ordinal = 0`,
		hashA, SlotHigh).Scan(&itemType, &chargeType, &quantity); err != nil {
		t.Fatal(err)
	}
	if itemType != 500 || chargeType != 600 || quantity != 1 {
		t.Errorf("stored representative = type %d charge %d qty %d, want 500/600/1",
			itemType, chargeType, quantity)
	}

	// A reparse can legitimately point the killmail at a different fit.
	second := &Fitting{
		FitHash: hashB, FamilyHash: hashB,
		Items: []ExtractedItem{{
			SlotGroup: SlotMed, Ordinal: 0, TypeID: 502, Quantity: 1,
		}},
	}
	secondLink := Link{
		KillmailID: killmailID, ShipTypeID: 588,
		KillTime:         time.Date(2021, 6, 7, 8, 9, 10, 0, time.UTC),
		VictimAllianceID: 99_000_002,
	}
	if err := Store(ctx, pool, second, secondLink); err != nil {
		t.Fatal(err)
	}

	var (
		gotHash        string
		gotShip        int32
		gotTime        time.Time
		gotAlliance    *int32
		gotCorporation *int32
	)
	if err := pool.QueryRow(ctx, `
        SELECT fit_hash, ship_type_id, kill_time,
               victim_alliance_id, victim_corporation_id
        FROM killmail_fittings WHERE killmail_id = $1`, killmailID).
		Scan(&gotHash, &gotShip, &gotTime, &gotAlliance, &gotCorporation); err != nil {
		t.Fatal(err)
	}
	if gotHash != hashB || gotShip != 588 || !gotTime.Equal(secondLink.KillTime) {
		t.Errorf("link = %s/%d/%s, want second fit", gotHash, gotShip, gotTime)
	}
	if gotAlliance == nil || *gotAlliance != secondLink.VictimAllianceID || gotCorporation != nil {
		t.Errorf("link affiliations = %v/%v, want %d/NULL",
			gotAlliance, gotCorporation, secondLink.VictimAllianceID)
	}
}
