package everef

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/pgbulk"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Insurance payouts, republished by EVE Ref whenever CCP changes them.
//
// There is no history here — only a current snapshot — so the table is replaced
// rather than merged. A ship whose insurance CCP withdrew has to disappear, and
// an upsert would leave it behind forever.

const insurancePath = "/insurance-prices/insurance-prices-latest.json"

type insuranceEntry struct {
	TypeID int32 `json:"type_id"`
	Levels []struct {
		Name   string  `json:"name"`
		Cost   float64 `json:"cost"`
		Payout float64 `json:"payout"`
	} `json:"levels"`
}

// ImportInsurance replaces the insurance table with the published snapshot.
func ImportInsurance(ctx context.Context, pool *pgxpool.Pool, client *Client) (Result, error) {
	start := time.Now()
	res := Result{Name: "insurance_prices"}

	var data []insuranceEntry
	if err := client.JSON(ctx, client.url(insurancePath), &data); err != nil {
		return res, err
	}
	if len(data) == 0 {
		// An empty snapshot would truncate the table to nothing. Refuse: EVE
		// Ref serving an empty file is far likelier than CCP withdrawing every
		// insurance policy in the game.
		return res, fmt.Errorf("insurance snapshot is empty — refusing to replace the table")
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return res, err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	// Delete and refill inside one transaction, so a reader never sees the
	// table empty and a failed import leaves the old snapshot intact.
	if _, err := tx.Exec(ctx, `DELETE FROM insurance_prices`); err != nil {
		return res, fmt.Errorf("clear insurance_prices: %w", err)
	}

	columns := []string{"type_id", "level_name", "cost", "payout"}
	const staging = "everef_staging_insurance"
	if err := pgbulk.StagingTx(ctx, tx, staging, "insurance_prices"); err != nil {
		return res, err
	}

	w := pgbulk.NewCopier(ctx, tx, staging, columns)
	for _, entry := range data {
		if entry.TypeID == 0 {
			continue
		}
		for _, level := range entry.Levels {
			if err := w.Add([]any{entry.TypeID, level.Name, level.Cost, level.Payout}); err != nil {
				return res, err
			}
		}
	}
	if err := w.Flush(); err != nil {
		return res, err
	}

	// DoNothing rather than DoUpdate: the table was just emptied, so the only
	// conflicts possible are duplicate levels within the snapshot itself.
	if _, err := tx.Exec(ctx, pgbulk.MergeSQL("insurance_prices", staging, columns,
		[]string{"type_id", "level_name"}, pgbulk.DoNothing)); err != nil {
		return res, fmt.Errorf("merge insurance_prices: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return res, err
	}

	res.Rows = w.Written()
	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}
