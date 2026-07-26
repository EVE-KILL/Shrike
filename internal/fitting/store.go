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

	// Items are part of the first-seen representative payload. Charges and
	// drones deliberately do not participate in fit_hash, so updating these
	// rows from a later kill would make one fit's rendered contents depend on
	// whichever example happened to process last.
	for _, it := range f.Items {
		if _, err := tx.Exec(ctx, `
            INSERT INTO fitting_items (fit_hash, slot_group, ordinal, type_id, charge_type_id, quantity)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT (fit_hash, slot_group, ordinal) DO NOTHING`,
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
        ON CONFLICT (killmail_id) DO UPDATE SET
            fit_hash = EXCLUDED.fit_hash,
            ship_type_id = EXCLUDED.ship_type_id,
            kill_time = EXCLUDED.kill_time,
            victim_alliance_id = EXCLUDED.victim_alliance_id,
            victim_corporation_id = EXCLUDED.victim_corporation_id`,
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
