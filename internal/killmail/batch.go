package killmail

import (
	"context"
	"fmt"

	"github.com/eve-kill/shrike/internal/pgbulk"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Bulk insertion, for the archive importers.
//
// Insert writes one killmail in its own transaction, which is right for a live
// mail arriving off the queue and hopeless for a daily archive of twenty
// thousand. InsertBatch is the same rows through COPY instead.

var (
	killmailColumns = []string{
		"killmail_id", "killmail_time", "killmail_hash",
		"solar_system_id", "constellation_id", "region_id",
		"position_x", "position_y", "position_z",
		"victim_character_id", "victim_corporation_id", "victim_alliance_id",
		"victim_faction_id", "victim_ship_type_id", "victim_ship_group_id",
		"victim_damage_taken",
		"total_value", "fitted_value", "dropped_value", "destroyed_value",
		"points", "attacker_count", "is_npc", "is_solo", "war_id", "blob",
	}
	attackerColumns = []string{
		"killmail_id", "killmail_time", "attacker_index",
		"character_id", "corporation_id", "alliance_id", "faction_id",
		"ship_type_id", "ship_group_id", "weapon_type_id",
		"damage_done", "points", "final_blow", "security_status",
	}
	itemColumns = []string{
		"killmail_id", "item_index", "type_id", "flag_id",
		"quantity_dropped", "quantity_destroyed", "singleton", "parent_index",
	}
)

// BatchResult reports what a bulk insert wrote.
type BatchResult struct {
	Killmails int64
	Attackers int64
	Items     int64
}

// InsertBatch writes many parsed killmails at once.
//
// Conflicts are ignored rather than updated: a killmail is immutable once
// stored, and the archives overlap heavily with what the live queue has already
// picked up. That makes re-running an import safe and cheap.
//
// The three tables go in together under one transaction, so a failure never
// leaves attackers without their killmail.
//
// Archive imports deliberately do not create killmail_processing rows. The TS
// importers treat historical rows as already imported data, not newly-arrived
// queue work. Creating a zeroed ledger here would make the entire archive look
// like a pending derived-effects backlog.
func InsertBatch(ctx context.Context, pool *pgxpool.Pool, batch []*Parsed) (BatchResult, error) {
	var res BatchResult
	if len(batch) == 0 {
		return res, nil
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

	const (
		kmStaging   = "km_staging_killmails"
		attStaging  = "km_staging_attackers"
		itemStaging = "km_staging_items"
	)

	for _, s := range []struct{ name, like string }{
		{kmStaging, "killmails"},
		{attStaging, "killmail_attackers"},
		{itemStaging, "killmail_items"},
	} {
		if err := pgbulk.StagingTx(ctx, tx, s.name, s.like); err != nil {
			return res, err
		}
	}

	kmw := pgbulk.NewCopier(ctx, tx, kmStaging, killmailColumns)
	attw := pgbulk.NewCopier(ctx, tx, attStaging, attackerColumns)
	itemw := pgbulk.NewCopier(ctx, tx, itemStaging, itemColumns)

	for _, p := range batch {
		km := p.Killmail
		var posX, posY, posZ any
		if km.Position != nil {
			posX, posY, posZ = km.Position.X, km.Position.Y, km.Position.Z
		}

		if err := kmw.Add([]any{
			km.KillmailID, km.KillmailTime, km.KillmailHash,
			km.SolarSystemID, nullID(km.ConstellationID), nullID(km.RegionID),
			posX, posY, posZ,
			nullID(km.VictimCharacterID), nullID(km.VictimCorporationID), nullID(km.VictimAllianceID),
			nullID(km.VictimFactionID), nullID(km.VictimShipTypeID), nullID(km.VictimShipGroupID),
			nullID(km.VictimDamageTaken),
			km.TotalValue, km.FittedValue, km.DroppedValue, km.DestroyedValue,
			km.Points, km.AttackerCount, km.IsNPC, km.IsSolo, nullID(km.WarID), km.Blob,
		}); err != nil {
			return res, err
		}
		for _, a := range p.Attackers {
			if err := attw.Add([]any{
				a.KillmailID, a.KillmailTime, a.AttackerIndex,
				nullID(a.CharacterID), nullID(a.CorporationID), nullID(a.AllianceID), nullID(a.FactionID),
				nullID(a.ShipTypeID), nullID(a.ShipGroupID), nullID(a.WeaponTypeID),
				a.DamageDone, a.Points, a.FinalBlow, a.SecurityStatus,
			}); err != nil {
				return res, err
			}
		}
		for _, it := range p.Items {
			if err := itemw.Add([]any{
				it.KillmailID, it.ItemIndex, it.TypeID, it.FlagID,
				it.QuantityDropped, it.QuantityDestroyed, it.Singleton, nullIndex(it.ParentIndex),
			}); err != nil {
				return res, err
			}
		}
	}

	for _, w := range []*pgbulk.Copier{kmw, attw, itemw} {
		if err := w.Flush(); err != nil {
			return res, err
		}
	}

	// Killmails first: the attacker and item rows are meaningless without it.
	for _, m := range []struct {
		table, staging string
		columns, pk    []string
	}{
		{"killmails", kmStaging, killmailColumns, []string{"killmail_id"}},
		{"killmail_attackers", attStaging, attackerColumns, []string{"killmail_id", "attacker_index"}},
		{"killmail_items", itemStaging, itemColumns, []string{"killmail_id", "item_index"}},
	} {
		if _, err := tx.Exec(ctx, pgbulk.MergeSQL(m.table, m.staging, m.columns, m.pk, pgbulk.DoNothing)); err != nil {
			return res, fmt.Errorf("merge %s: %w", m.table, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return res, err
	}

	res.Killmails = kmw.Written()
	res.Attackers = attw.Written()
	res.Items = itemw.Written()
	return res, nil
}

// AssignWars sets war_id on killmails that are already stored.
//
// The bulk insert ignores conflicts, so a kill the live queue picked up before
// the war archive named it keeps a null war_id. Only a null is filled: a
// killmail already attributed to a war must not be moved to a different one, and
// the archives do occasionally disagree.
func AssignWars(ctx context.Context, pool *pgxpool.Pool, batch []*Parsed) (int64, error) {
	ids := make([]int32, 0, len(batch))
	warIDs := make([]int32, 0, len(batch))
	for _, p := range batch {
		if p.Killmail.WarID == 0 {
			continue
		}
		ids = append(ids, int32(p.Killmail.KillmailID))
		warIDs = append(warIDs, p.Killmail.WarID)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	tag, err := pool.Exec(ctx, `
        UPDATE killmails k
        SET war_id = v.war_id
        FROM unnest($1::int[], $2::int[]) AS v(killmail_id, war_id)
        WHERE k.killmail_id = v.killmail_id
          AND (k.war_id IS NULL OR k.war_id = 0)`, ids, warIDs)
	if err != nil {
		return 0, fmt.Errorf("assign war ids: %w", err)
	}
	return tag.RowsAffected(), nil
}
