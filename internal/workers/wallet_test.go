package workers

import (
	"context"
	"testing"
	"time"
)

// Expiring a reservation is a money operation, and the failure mode is silent:
// close the reservation without returning the held amount and the user's
// spendable balance shrinks a little with every expiry, permanently, with no
// error anywhere. So the assertion here is about the account balance, not about
// the reservation's status.

// Reservations cannot be deleted. A database trigger enforces it — "EK Wallet
// reservations are auditable and cannot be deleted" — which is correct for a
// financial audit trail and means these tests cannot clean up after themselves.
//
// Isolation is therefore by identity: each test owns a character id, so one
// test's undeletable rows can never be seen by another. Re-running a test does
// see its own leftovers, and that is harmless — a released reservation is
// status 3 and no longer eligible, and the one test that deliberately leaves an
// active reservation expects a failure either way.
//
// walletCharacterBase is above every real character id CCP has issued, which is
// not much headroom (real ids have reached 2.12 billion) but is enough for a
// handful of fixtures.
const walletCharacterBase int32 = 2_140_000_000

// walletCharacter returns this test's own character id.
func walletCharacter(t *testing.T, offset int32) int32 {
	t.Helper()
	return walletCharacterBase + offset
}

// seedReservation creates an account holding `reserved` with one reservation.
//
// The schema is strict about the shape, and each constraint is worth knowing:
// transaction_type must be 1 or 3, expires_at must be after created_at (so an
// already-expired fixture needs a created_at in the past), amount must be
// positive, and the lifecycle check ties status to which closure columns are
// set — status 3 requires closed_at and forbids captured_amount, which is
// exactly what the expiry writes.
//
// The account's balance is set equal to the reserved amount, because the schema
// enforces `reserved_balance <= balance` — a hold cannot exceed the funds it is
// held against. Seeding a zero balance violates that and the row is rejected.
func seedReservation(t *testing.T, d *Deps, character int32, reserved, amount string, expiresAt time.Time) int64 {
	t.Helper()
	ctx := context.Background()

	// Clear first, not only on cleanup. A previous run that failed during
	// seeding never reached its t.Cleanup registration, so its rows are still
	// here — and an orphaned reservation makes the next run release twice and
	// report a balance that looks like a bug in the code under test.
	clearWalletFixtures(ctx, d, character)

	// A wallet account belongs to a logged-in user, not merely to a known
	// character — the foreign key is against `users`. That is the right shape:
	// only somebody who has authenticated can hold a balance.
	if _, err := d.Pool.Exec(ctx, `
        INSERT INTO users (character_id, character_name, character_owner_hash, session_id, created_at, updated_at)
        VALUES ($1, 'Shrike Wallet Test', 'shrike-test-hash', gen_random_uuid(), now(), now())
        ON CONFLICT (character_id) DO NOTHING`, character); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// A failure here is a broken test, not an absent feature — the tables exist
	// in the baseline schema. Failing rather than skipping keeps a bad seed
	// from reading as "nothing to test".
	if _, err := d.Pool.Exec(ctx, `
        INSERT INTO ek_wallet_accounts (character_id, balance, reserved_balance, created_at, updated_at)
        VALUES ($1, $2::numeric, $2::numeric, now(), now())
        ON CONFLICT (character_id) DO UPDATE SET
            balance = EXCLUDED.balance, reserved_balance = EXCLUDED.reserved_balance`,
		character, reserved); err != nil {
		t.Fatalf("seed wallet account: %v", err)
	}

	var id int64
	if err := d.Pool.QueryRow(ctx, `
        INSERT INTO ek_wallet_reservations (
            character_id, external_reference, transaction_type, amount, status,
            description, expires_at, created_at, updated_at
        ) VALUES ($1, $2, 1, $3::numeric, $4, 'shrike test hold', $5,
                  now() - interval '1 day', now())
        RETURNING id`,
		character, "shrike-test-"+time.Now().Format("150405.000000"),
		amount, reservationActive, expiresAt).Scan(&id); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { clearWalletFixtures(context.Background(), d, character) })
	return id
}

// clearWalletFixtures resets what can be reset.
//
// Reservations are deliberately not deleted — the trigger forbids it. The
// account balance is reset instead, which is what each test actually depends
// on, and stale reservations are made ineligible by closing them.
func clearWalletFixtures(ctx context.Context, d *Deps, character int32) {
	_, _ = d.Pool.Exec(ctx, `
        UPDATE ek_wallet_reservations
        SET status = $2, close_reason = 'test reset', closed_at = now(), updated_at = now()
        WHERE character_id = $1 AND status = $3`, character, reservationExpired, reservationActive)
	_, _ = d.Pool.Exec(ctx, `DELETE FROM ek_wallet_accounts WHERE character_id = $1`, character)
	_, _ = d.Pool.Exec(ctx, `DELETE FROM users WHERE character_id = $1`, character)
}

func reservedBalance(t *testing.T, d *Deps, character int32) string {
	t.Helper()
	var v string
	if err := d.Pool.QueryRow(context.Background(),
		`SELECT reserved_balance::text FROM ek_wallet_accounts WHERE character_id = $1`,
		character).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func reservationStatus(t *testing.T, d *Deps, id int64) int16 {
	t.Helper()
	var s int16
	if err := d.Pool.QueryRow(context.Background(),
		`SELECT status FROM ek_wallet_reservations WHERE id = $1`, id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// The assertion that matters: the held amount comes back.
func TestExpiredReservationReleasesTheHeldBalance(t *testing.T) {
	d := testDeps(t)
	ctx := context.Background()
	character := walletCharacter(t, 1)

	id := seedReservation(t, d, character, "100.00", "40.00", time.Now().Add(-time.Minute))

	report, err := d.cronEkWalletReservationExpiry(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report == "" {
		t.Fatal("an expired reservation produced no report — it was not processed")
	}

	if got := reservedBalance(t, d, character); got != "60.00" {
		t.Errorf("reserved_balance = %s, want 60.00 — the hold was closed without "+
			"returning the funds, so the user permanently loses 40.00 of "+
			"spendable balance", got)
	}
	if got := reservationStatus(t, d, id); got != reservationExpired {
		t.Errorf("reservation status = %d, want %d", got, reservationExpired)
	}
}

// A reservation that has not expired must be left entirely alone.
func TestUnexpiredReservationIsUntouched(t *testing.T) {
	d := testDeps(t)
	ctx := context.Background()
	character := walletCharacter(t, 2)

	id := seedReservation(t, d, character, "100.00", "40.00", time.Now().Add(time.Hour))

	if _, err := d.cronEkWalletReservationExpiry(ctx); err != nil {
		t.Fatal(err)
	}
	if got := reservedBalance(t, d, character); got != "100.00" {
		t.Errorf("reserved_balance = %s, want 100.00 — funds were released early", got)
	}
	if got := reservationStatus(t, d, id); got != reservationActive {
		t.Errorf("an unexpired reservation was closed (status %d)", got)
	}
}

// Running twice must not release twice. The re-read under lock is what prevents
// it, and without that a retry after a partial failure would double-refund.
func TestExpiryIsIdempotent(t *testing.T) {
	d := testDeps(t)
	ctx := context.Background()
	character := walletCharacter(t, 3)

	seedReservation(t, d, character, "100.00", "40.00", time.Now().Add(-time.Minute))

	for i := range 3 {
		if _, err := d.cronEkWalletReservationExpiry(ctx); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	if got := reservedBalance(t, d, character); got != "60.00" {
		t.Errorf("reserved_balance = %s after three runs, want 60.00 — the hold was "+
			"released more than once", got)
	}
}

// If the account holds less than the reservation claims, the two records
// already disagree. Refusing is right: clamping to zero would silently paper
// over an inconsistency in somebody's money.
func TestRefusesToReleaseMoreThanIsHeld(t *testing.T) {
	d := testDeps(t)
	ctx := context.Background()
	character := walletCharacter(t, 4)

	seedReservation(t, d, character, "10.00", "40.00", time.Now().Add(-time.Minute))

	_, err := d.cronEkWalletReservationExpiry(ctx)
	if err == nil {
		t.Fatal("a reservation claiming more than the account holds was released — " +
			"the balance would have gone negative")
	}
	if got := reservedBalance(t, d, character); got != "10.00" {
		t.Errorf("reserved_balance = %s, want it left at 10.00", got)
	}
}
