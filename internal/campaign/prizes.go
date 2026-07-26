package campaign

import (
	"context"
	"errors"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Prize pool settlement.
//
// A campaign with a prize pool has real ISK in it, contributed by people who
// expect it to be paid out according to the standings. Settlement is the
// moment those standings stop being live and become the thing the money is
// divided by, so it happens exactly once, under a lock, and only after the
// statistics are known to cover the campaign's full window.

// Prize pool lifecycle, stored as smallints.
const (
	PrizePoolFunding   = 0
	PrizePoolReady     = 1
	PrizePoolPaid      = 2
	PrizePoolCancelled = 3
)

// CalculatePayouts divides a pool between ranks by weight.
//
// Everything is whole ISK — the game has no fractions — which is what makes
// the first place a special case. Flooring every share loses the remainder, so
// the shares below first are floored and first takes what is left. The
// alternative, distributing the remainder proportionally, produces a pool that
// does not add up to what was funded, and a prize pool that does not add up is
// a support ticket.
func CalculatePayouts(fundedTotal float64, percentages []float64) []int64 {
	out := make([]int64, len(percentages))
	if len(percentages) == 0 {
		return out
	}

	total := int64(math.Floor(math.Max(0, fundedTotal)))
	if total == 0 {
		return out
	}

	var weight float64
	for _, p := range percentages {
		weight += math.Max(0, p)
	}
	// Every rank weighted zero: there is no basis to divide by, and paying the
	// whole pool to first because it happens to be index zero would be worse
	// than paying nothing.
	if weight <= 0 {
		return out
	}

	var assigned int64
	for i := 1; i < len(percentages); i++ {
		out[i] = int64(math.Floor(float64(total) * math.Max(0, percentages[i]) / weight))
		assigned += out[i]
	}
	out[0] = total - assigned
	return out
}

// Finalize freezes the standings and writes the payouts.
//
// Returns false when there was nothing to settle — no pool, or one already
// settled. That is the ordinary outcome of the sweep re-examining a campaign
// it has already finished with, not an error.
//
// The pool row is locked for the whole transaction. Two workers reaching a
// campaign at the same moment would otherwise both read status Funding and
// both write payouts, and the second would overwrite the first with the same
// numbers — harmless today, and exactly the kind of thing that stops being
// harmless the moment a payout step is added between them.
func Finalize(ctx context.Context, pool *pgxpool.Pool, campaignID string) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var status int16
	var fundedTotal float64
	err = tx.QueryRow(ctx, `
        SELECT status, funded_total FROM campaign_prize_pools
        WHERE campaign_id = $1 FOR UPDATE`, campaignID).Scan(&status, &fundedTotal)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if status != PrizePoolFunding {
		return false, nil
	}

	rows, err := tx.Query(ctx, `
        SELECT rank, payout_percentage FROM campaign_prize_results
        WHERE campaign_id = $1 ORDER BY rank`, campaignID)
	if err != nil {
		return false, err
	}

	var ranks []int32
	var percentages []float64
	for rows.Next() {
		var rank int32
		var pct float64
		if err := rows.Scan(&rank, &pct); err != nil {
			rows.Close()
			return false, err
		}
		ranks = append(ranks, rank)
		percentages = append(percentages, pct)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, err
	}

	payouts := CalculatePayouts(fundedTotal, percentages)
	for i, rank := range ranks {
		if _, err := tx.Exec(ctx, `
            UPDATE campaign_prize_results
            SET payout_amount = $3, updated_at = now()
            WHERE campaign_id = $1 AND rank = $2`, campaignID, rank, payouts[i]); err != nil {
			return false, err
		}
	}

	if _, err := tx.Exec(ctx, `
        UPDATE campaign_prize_pools
        SET status = $2, finalized_at = now(), updated_at = now()
        WHERE campaign_id = $1`, campaignID, PrizePoolReady); err != nil {
		return false, err
	}

	return true, tx.Commit(ctx)
}
