package fitting

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Storing a fit.
//
// Three tables with different lifetimes, which is why they are separate:
// `fittings` is the identity and is written once ever, `fitting_items` is its
// contents and is rewritten when the render payload gains columns, and
// `killmail_fittings` links a fit to the kill it was seen on and is what the
// 90-day retention actually prunes.

// Link is the killmail context a fit was observed in.
type Link struct {
	KillmailID          int64
	ShipTypeID          int32
	KillTime            time.Time
	VictimAllianceID    int32
	VictimCorporationID int32
}

// Store writes a fit and links it to the killmail it came from.
func Store(ctx context.Context, pool *pgxpool.Pool, f *Fitting, link Link) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	// The identity is immutable — the same modules always produce the same
	// hash — so a conflict means we have seen this fit before and first_seen_at
	// must not move.
	if _, err := tx.Exec(ctx, `
        INSERT INTO fittings (fit_hash, ship_type_id, family_hash, first_seen_at)
        VALUES ($1, $2, $3, now())
        ON CONFLICT (fit_hash) DO NOTHING`,
		f.FitHash, link.ShipTypeID, f.FamilyHash); err != nil {
		return fmt.Errorf("store fit identity: %w", err)
	}

	// Items are upserted rather than skipped, so a re-extract can fill in
	// columns added since the fit was first seen. The ordinals are stable by
	// construction, so this overwrites in place rather than churning rows.
	for _, it := range f.Items {
		if _, err := tx.Exec(ctx, `
            INSERT INTO fitting_items (fit_hash, slot_group, ordinal, type_id, charge_type_id, quantity)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT (fit_hash, slot_group, ordinal) DO UPDATE SET
                type_id = EXCLUDED.type_id,
                charge_type_id = EXCLUDED.charge_type_id,
                quantity = EXCLUDED.quantity`,
			f.FitHash, it.SlotGroup, it.Ordinal, it.TypeID,
			nullID(it.ChargeTypeID), it.Quantity); err != nil {
			return fmt.Errorf("store fit item: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
        INSERT INTO killmail_fittings (
            killmail_id, fit_hash, ship_type_id, kill_time,
            victim_alliance_id, victim_corporation_id
        ) VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT DO NOTHING`,
		link.KillmailID, f.FitHash, link.ShipTypeID, link.KillTime,
		nullID(link.VictimAllianceID), nullID(link.VictimCorporationID)); err != nil {
		return fmt.Errorf("link fit to killmail: %w", err)
	}

	return tx.Commit(ctx)
}

func nullID(v int32) any {
	if v == 0 {
		return nil
	}
	return v
}
