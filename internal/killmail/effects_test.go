package killmail

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

// The ledger's entire purpose is that additive work runs once. These tests are
// about the ways it could run twice, or fail to run at all.
//
// The ids used here sit in the reserved band above every real killmail (see
// testIDOffset in store_test.go), and each test cleans up only the rows it
// created. Nothing here touches production-shaped data.

const effectsTestID int64 = testIDOffset + 990_001

func effectsCleanup(t *testing.T, ctx context.Context, ids ...int64) {
	t.Helper()
	pool := storePool(t)
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = pool.Exec(context.Background(),
				`DELETE FROM killmail_processing WHERE killmail_id = $1`, id)
			_, _ = pool.Exec(context.Background(),
				`DELETE FROM killmails WHERE killmail_id = $1`, id)
		}
	})
}

// seedLedger creates a killmail and its ledger row.
func seedLedger(t *testing.T, ctx context.Context, id int64, completed Effect) {
	t.Helper()
	pool := storePool(t)

	if _, err := pool.Exec(ctx, `
        INSERT INTO killmails (killmail_id, killmail_time, killmail_hash, solar_system_id)
        VALUES ($1, now(), 'effects-test', 30000142)
        ON CONFLICT (killmail_id) DO NOTHING`, id); err != nil {
		t.Fatalf("seed killmail: %v", err)
	}
	if _, err := pool.Exec(ctx, `
        INSERT INTO killmail_processing (killmail_id, effects_completed)
        VALUES ($1, $2)
        ON CONFLICT (killmail_id) DO UPDATE SET effects_completed = $2`,
		id, int32(completed)); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}
}

func ledgerState(t *testing.T, ctx context.Context, id int64) Effect {
	t.Helper()
	var completed int32
	err := storePool(t).QueryRow(ctx,
		`SELECT effects_completed FROM killmail_processing WHERE killmail_id = $1`, id).
		Scan(&completed)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	return Effect(completed)
}

func TestEffectRunsOnceAndRecordsItself(t *testing.T) {
	ctx := context.Background()
	pool := storePool(t)
	id := effectsTestID
	effectsCleanup(t, ctx, id)
	seedLedger(t, ctx, id, 0)

	var runs int
	work := func(ctx context.Context, tx pgx.Tx) (bool, error) {
		runs++
		return true, nil
	}

	ran, err := RunDBEffect(ctx, pool, id, EffectLastActive, work, EffectOptions{})
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if !ran || runs != 1 {
		t.Fatalf("first run: ran=%v runs=%d, want true and 1", ran, runs)
	}
	if got := ledgerState(t, ctx, id); !IsComplete(got, EffectLastActive) {
		t.Fatalf("the ledger does not record the effect: %d", got)
	}

	// The retry. This is the case the whole mechanism exists for.
	ran, err = RunDBEffect(ctx, pool, id, EffectLastActive, work, EffectOptions{})
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if ran {
		t.Error("the second run reported that it ran the effect")
	}
	if runs != 1 {
		t.Errorf("the work ran %d times — an additive effect would have "+
			"double-counted on retry", runs)
	}
}

// A failing effect leaves its bit clear, so the retry does it again. If the
// bit were set regardless, the work would be silently skipped forever.
func TestFailedEffectIsNotRecorded(t *testing.T) {
	ctx := context.Background()
	pool := storePool(t)
	id := effectsTestID + 1
	effectsCleanup(t, ctx, id)
	seedLedger(t, ctx, id, 0)

	boom := errors.New("the database fell over")
	_, err := RunDBEffect(ctx, pool, id, EffectDailyKillRollup,
		func(ctx context.Context, tx pgx.Tx) (bool, error) { return false, boom },
		EffectOptions{})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want the work's error", err)
	}
	if got := ledgerState(t, ctx, id); IsComplete(got, EffectDailyKillRollup) {
		t.Error("a failed effect was marked complete — it would never be retried")
	}
}

// Returning false means "not yet": the work decided it cannot run, and the
// effect stays pending rather than being recorded as done.
func TestDeferredEffectStaysPending(t *testing.T) {
	ctx := context.Background()
	pool := storePool(t)
	id := effectsTestID + 2
	effectsCleanup(t, ctx, id)
	seedLedger(t, ctx, id, 0)

	ran, err := RunDBEffect(ctx, pool, id, EffectWarInteractions,
		func(ctx context.Context, tx pgx.Tx) (bool, error) { return false, nil },
		EffectOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ran {
		t.Error("a deferred effect reported that it ran")
	}
	if got := ledgerState(t, ctx, id); IsComplete(got, EffectWarInteractions) {
		t.Error("a deferred effect was marked complete — a killmail waiting for " +
			"its war would never be aggregated")
	}
}

// The work and the bit commit together. A rolled-back effect must leave no
// trace of either.
func TestWorkAndLedgerCommitTogether(t *testing.T) {
	ctx := context.Background()
	pool := storePool(t)
	id := effectsTestID + 3
	effectsCleanup(t, ctx, id, id+500)
	seedLedger(t, ctx, id, 0)

	marker := id + 500
	boom := errors.New("failed after writing")

	// Write a row, then fail. Neither the row nor the bit may survive.
	_, err := RunDBEffect(ctx, pool, id, EffectLastActive,
		func(ctx context.Context, tx pgx.Tx) (bool, error) {
			if _, err := tx.Exec(ctx, `
                INSERT INTO killmails (killmail_id, killmail_time, killmail_hash, solar_system_id)
                VALUES ($1, now(), 'rollback-marker', 30000142)`, marker); err != nil {
				return false, err
			}
			return false, boom
		}, EffectOptions{})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v", err)
	}

	var exists bool
	err = pool.QueryRow(ctx, `SELECT true FROM killmails WHERE killmail_id = $1`, marker).Scan(&exists)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Error("the failed effect's write survived — the work and the ledger " +
			"are not committing as one transaction")
	}
	if got := ledgerState(t, ctx, id); IsComplete(got, EffectLastActive) {
		t.Error("the failed effect's ledger bit survived")
	}
}

// An untracked killmail — one stored before the ledger existed — is skipped
// rather than run, unless the caller says otherwise.
func TestUntrackedKillmailIsSkipped(t *testing.T) {
	ctx := context.Background()
	pool := storePool(t)
	id := effectsTestID + 4
	effectsCleanup(t, ctx, id)

	// A killmail with no ledger row.
	if _, err := pool.Exec(ctx, `
        INSERT INTO killmails (killmail_id, killmail_time, killmail_hash, solar_system_id)
        VALUES ($1, now(), 'untracked', 30000142)
        ON CONFLICT (killmail_id) DO NOTHING`, id); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var runs int
	work := func(ctx context.Context, tx pgx.Tx) (bool, error) { runs++; return true, nil }

	if _, err := RunDBEffect(ctx, pool, id, EffectLastActive, work, EffectOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runs != 0 {
		t.Error("an untracked killmail ran an effect with no way to record it — " +
			"it would run again on every retry")
	}

	if _, err := RunDBEffect(ctx, pool, id, EffectLastActive, work,
		EffectOptions{AllowUntracked: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runs != 1 {
		t.Errorf("AllowUntracked did not run the effect (runs=%d)", runs)
	}
}

// Attaching a war to a stored killmail clears exactly one bit.
func TestWarAttachmentClearsOnlyTheWarBit(t *testing.T) {
	ctx := context.Background()
	pool := storePool(t)
	id := effectsTestID + 5
	effectsCleanup(t, ctx, id)
	seedLedger(t, ctx, id, AllEffects)

	const warID int32 = 999_999_001
	got, err := Prepare(ctx, pool, id, warID)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	if !got.Exists || !got.Tracked {
		t.Fatalf("prepare reported exists=%v tracked=%v", got.Exists, got.Tracked)
	}
	if got.WarID != warID {
		t.Errorf("war id = %d, want %d", got.WarID, warID)
	}
	if got.Pending() != EffectWarInteractions {
		t.Errorf("pending effects = %d, want only the war bit (%d) — attaching a "+
			"war must not reprocess the whole killmail",
			got.Pending(), EffectWarInteractions)
	}
	if stored := ledgerState(t, ctx, id); IsComplete(stored, EffectWarInteractions) {
		t.Error("the war bit is still set in the database")
	}
}

// A killmail predating the ledger gets one written, marked as having done
// everything but the war — which is accurate, because it did.
func TestWarAttachmentBackfillsAnUntrackedKillmail(t *testing.T) {
	ctx := context.Background()
	pool := storePool(t)
	id := effectsTestID + 6
	effectsCleanup(t, ctx, id)

	if _, err := pool.Exec(ctx, `
        INSERT INTO killmails (killmail_id, killmail_time, killmail_hash, solar_system_id)
        VALUES ($1, now(), 'legacy', 30000142)
        ON CONFLICT (killmail_id) DO NOTHING`, id); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := Prepare(ctx, pool, id, 999_999_002)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if !got.Tracked {
		t.Fatal("no ledger row was created")
	}
	if got.Completed != AllEffectsExceptWar {
		t.Errorf("completed = %d, want %d", got.Completed, AllEffectsExceptWar)
	}
	if stored := ledgerState(t, ctx, id); stored != AllEffectsExceptWar {
		t.Errorf("stored ledger = %d, want %d", stored, AllEffectsExceptWar)
	}
}

// War archives assign war_id while deliberately leaving historical kills out
// of the effects ledger. Replaying that same war must adopt the row instead of
// returning early forever with the war interaction still missing.
func TestWarReplayBackfillsAnUntrackedAlreadyAssignedKillmail(t *testing.T) {
	ctx := context.Background()
	pool := storePool(t)
	id := effectsTestID + 8
	effectsCleanup(t, ctx, id)

	const warID int32 = 999_999_003
	if _, err := pool.Exec(ctx,
		`DELETE FROM killmail_processing WHERE killmail_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
        INSERT INTO killmails (
            killmail_id, killmail_time, killmail_hash, solar_system_id, war_id
        ) VALUES ($1, now(), 'archive', 30000142, $2)
        ON CONFLICT (killmail_id) DO UPDATE SET war_id = EXCLUDED.war_id`,
		id, warID); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := Prepare(ctx, pool, id, warID)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if !got.Tracked || got.Completed != AllEffectsExceptWar {
		t.Errorf("tracked/completed = %v/%d, want true/%d",
			got.Tracked, got.Completed, AllEffectsExceptWar)
	}
	if stored := ledgerState(t, ctx, id); stored != AllEffectsExceptWar {
		t.Errorf("stored ledger = %d, want %d", stored, AllEffectsExceptWar)
	}
}

// Two wars cannot both own a killmail.
func TestConflictingWarIsRefused(t *testing.T) {
	ctx := context.Background()
	pool := storePool(t)
	id := effectsTestID + 7
	effectsCleanup(t, ctx, id)
	seedLedger(t, ctx, id, 0)

	if _, err := pool.Exec(ctx,
		`UPDATE killmails SET war_id = $2 WHERE killmail_id = $1`, id, 111); err != nil {
		t.Fatalf("seed war: %v", err)
	}

	if _, err := Prepare(ctx, pool, id, 222); !errors.Is(err, ErrWarConflict) {
		t.Errorf("error = %v, want ErrWarConflict — silently reassigning would "+
			"move the kill's ISK from one war's totals to another", err)
	}
}

// A killmail we have never seen reports so rather than erroring.
func TestPrepareOnMissingKillmail(t *testing.T) {
	ctx := context.Background()
	got, err := Prepare(context.Background(), storePool(t), testIDOffset+999_999, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Exists {
		t.Error("a killmail that does not exist was reported as existing")
	}
	_ = ctx
}

func TestDoneAndPending(t *testing.T) {
	full := Tracked{Completed: AllEffects}
	if !full.Done() {
		t.Error("a killmail with every effect complete is not Done")
	}
	if full.Pending() != 0 {
		t.Errorf("pending = %d for a complete killmail", full.Pending())
	}

	partial := Tracked{Completed: AllEffectsExceptWar}
	if partial.Done() {
		t.Error("a killmail missing the war effect reported as Done")
	}
	if partial.Pending() != EffectWarInteractions {
		t.Errorf("pending = %d, want %d", partial.Pending(), EffectWarInteractions)
	}
}

// The bits are a stored contract shared with the TypeScript backend. Changing
// one silently reinterprets every row in production.
func TestEffectBitsAreStable(t *testing.T) {
	cases := []struct {
		effect Effect
		want   int32
	}{
		{EffectLastActive, 1},
		{EffectWarInteractions, 2},
		{EffectDailyKillRollup, 4},
		{EffectAchievementsDispatched, 8},
		{EffectFitDispatched, 16},
		{EffectStatsDispatched, 32},
		{EffectGraphDispatched, 64},
		{EffectEntitiesEnsured, 128},
		{EffectEventPublished, 256},
		{EffectTickerEvaluated, 512},
		{EffectStatsWritten, 1024},
	}
	for _, c := range cases {
		if int32(c.effect) != c.want {
			t.Errorf("effect bit changed to %d, want %d — production rows and the "+
				"TypeScript backend both read these values", int32(c.effect), c.want)
		}
	}
	if int32(AllEffects) != 2047 {
		t.Errorf("AllEffects = %d, want 2047 (bits 0-10)", int32(AllEffects))
	}
}
