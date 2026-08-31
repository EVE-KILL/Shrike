package killmail

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// nullID converts the zero-means-absent convention into a real NULL.
func nullID(v int32) any {
	if v == 0 {
		return nil
	}
	return v
}

// nullIndex converts the NoParent sentinel into a real NULL. Separate from
// nullID because zero is a valid index.
func nullIndex(v int32) any {
	if v < 0 {
		return nil
	}
	return v
}

// Insert writes a killmail and everything hanging off it.
//
// Returns false when the killmail already existed, which is not an error: two
// sources routinely deliver the same kill within seconds of each other, and the
// loser of that race has nothing left to do.
func Insert(ctx context.Context, pool *pgxpool.Pool, p *Parsed) (bool, error) {
	return insert(ctx, pool, p, true)
}

// InsertUntracked writes only the killmail and child rows, without creating a
// derived-effect ledger entry. It exists for debug:killmail, whose TypeScript
// implementation invokes selected effects inline with allowUntracked=true.
func InsertUntracked(ctx context.Context, pool *pgxpool.Pool, p *Parsed) (bool, error) {
	return insert(ctx, pool, p, false)
}

func insert(ctx context.Context, pool *pgxpool.Pool, p *Parsed, tracked bool) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	km := p.Killmail
	var posX, posY, posZ any
	if km.Position != nil {
		posX, posY, posZ = km.Position.X, km.Position.Y, km.Position.Z
	}

	var insertedID int64
	err = tx.QueryRow(ctx, `
        INSERT INTO killmails (
            killmail_id, killmail_time, killmail_hash,
            solar_system_id, constellation_id, region_id,
            position_x, position_y, position_z,
            victim_character_id, victim_corporation_id, victim_alliance_id,
            victim_faction_id, victim_ship_type_id, victim_ship_group_id,
            victim_damage_taken,
            total_value, fitted_value, dropped_value, destroyed_value,
            points, attacker_count, is_npc, is_solo, war_id, blob
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
            $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
        )
        ON CONFLICT (killmail_id) DO NOTHING
        RETURNING killmail_id`,
		km.KillmailID, km.KillmailTime, km.KillmailHash,
		km.SolarSystemID, nullID(km.ConstellationID), nullID(km.RegionID),
		posX, posY, posZ,
		nullID(km.VictimCharacterID), nullID(km.VictimCorporationID), nullID(km.VictimAllianceID),
		nullID(km.VictimFactionID), nullID(km.VictimShipTypeID), nullID(km.VictimShipGroupID),
		nullID(km.VictimDamageTaken),
		km.TotalValue, km.FittedValue, km.DroppedValue, km.DestroyedValue,
		km.Points, km.AttackerCount, km.IsNPC, km.IsSolo, nullID(km.WarID), km.Blob,
	).Scan(&insertedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("insert killmail: %w", err)
	}

	batch := &pgx.Batch{}
	for _, a := range p.Attackers {
		batch.Queue(`
            INSERT INTO killmail_attackers (
                killmail_id, attacker_index, character_id, corporation_id,
                alliance_id, faction_id, ship_type_id, ship_group_id,
                weapon_type_id, damage_done, final_blow, security_status,
                killmail_time
            ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
            ON CONFLICT (killmail_id, attacker_index) DO NOTHING`,
			a.KillmailID, a.AttackerIndex, nullID(a.CharacterID), nullID(a.CorporationID),
			nullID(a.AllianceID), nullID(a.FactionID), nullID(a.ShipTypeID), nullID(a.ShipGroupID),
			nullID(a.WeaponTypeID), a.DamageDone, a.FinalBlow, a.SecurityStatus,
			a.KillmailTime)
	}
	for _, it := range p.Items {
		batch.Queue(`
            INSERT INTO killmail_items (
                killmail_id, item_index, type_id, flag_id,
                quantity_dropped, quantity_destroyed, singleton, parent_index
            ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
            ON CONFLICT (killmail_id, item_index) DO NOTHING`,
			it.KillmailID, it.ItemIndex, it.TypeID, it.FlagID,
			it.QuantityDropped, it.QuantityDestroyed, it.Singleton, nullIndex(it.ParentIndex))
	}
	if tracked {
		// The ledger row that the derived-effect machinery reads.
		batch.Queue(`
            INSERT INTO killmail_processing (killmail_id, effects_completed)
            VALUES ($1, 0)
			ON CONFLICT (killmail_id) DO NOTHING`, km.KillmailID)
		// The dirty marker commits with the canonical killmail. Queue delivery is
		// an acceleration; daily maintenance can always recover this durable fact.
		batch.Queue(`
			INSERT INTO character_intel_dirty_days (activity_date, dirtied_at)
			SELECT $1::timestamptz::date, now()
			WHERE $1::timestamptz >= CURRENT_DATE - 364
			ON CONFLICT (activity_date) DO UPDATE SET dirtied_at = EXCLUDED.dirtied_at`, km.KillmailTime)
	}

	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return false, fmt.Errorf("insert killmail children: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// Delete removes a killmail and its children, for reprocessing.
func Delete(ctx context.Context, pool *pgxpool.Pool, killmailID int64) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	for _, stmt := range []string{
		`DELETE FROM killmail_items WHERE killmail_id = $1`,
		`DELETE FROM killmail_attackers WHERE killmail_id = $1`,
		`DELETE FROM killmail_processing WHERE killmail_id = $1`,
		`DELETE FROM killmails WHERE killmail_id = $1`,
	} {
		if _, err := tx.Exec(ctx, stmt, killmailID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ErrNotStored is returned by Load for a killmail the database does not have.
var ErrNotStored = errors.New("killmail is not stored")

// Load reads a killmail back out of the database in the same shape the parser
// produces, so a stored mail and a freshly parsed one can be compared field by
// field.
func Load(ctx context.Context, pool *pgxpool.Pool, killmailID int64) (*Parsed, error) {
	var km Killmail
	var constellationID, regionID *int32
	var posX, posY, posZ *float64
	var vChar, vCorp, vAlly, vFaction, vShip, vGroup, vDamage, warID *int32

	err := pool.QueryRow(ctx, `
        SELECT killmail_id, killmail_time, killmail_hash,
               solar_system_id, constellation_id, region_id,
               position_x, position_y, position_z,
               victim_character_id, victim_corporation_id, victim_alliance_id,
               victim_faction_id, victim_ship_type_id, victim_ship_group_id,
               victim_damage_taken,
               coalesce(total_value, 0), coalesce(fitted_value, 0),
               coalesce(dropped_value, 0), coalesce(destroyed_value, 0),
               coalesce(points, 0), coalesce(attacker_count, 0),
               coalesce(is_npc, false), coalesce(is_solo, false),
               war_id, coalesce(blob, false)
        FROM killmails WHERE killmail_id = $1`, killmailID).Scan(
		&km.KillmailID, &km.KillmailTime, &km.KillmailHash,
		&km.SolarSystemID, &constellationID, &regionID,
		&posX, &posY, &posZ,
		&vChar, &vCorp, &vAlly, &vFaction, &vShip, &vGroup, &vDamage,
		&km.TotalValue, &km.FittedValue, &km.DroppedValue, &km.DestroyedValue,
		&km.Points, &km.AttackerCount, &km.IsNPC, &km.IsSolo, &warID, &km.Blob)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotStored
	}
	if err != nil {
		return nil, err
	}

	km.ConstellationID = deref(constellationID)
	km.RegionID = deref(regionID)
	km.VictimCharacterID = deref(vChar)
	km.VictimCorporationID = deref(vCorp)
	km.VictimAllianceID = deref(vAlly)
	km.VictimFactionID = deref(vFaction)
	km.VictimShipTypeID = deref(vShip)
	km.VictimShipGroupID = deref(vGroup)
	km.VictimDamageTaken = deref(vDamage)
	km.WarID = deref(warID)
	if posX != nil && posY != nil && posZ != nil {
		km.Position = &ESIPosition{X: *posX, Y: *posY, Z: *posZ}
	}

	out := &Parsed{Killmail: km}

	rows, err := pool.Query(ctx, `
        SELECT attacker_index, character_id, corporation_id, alliance_id,
               faction_id, ship_type_id, ship_group_id, weapon_type_id,
               coalesce(damage_done, 0), coalesce(final_blow, false),
               security_status, killmail_time
        FROM killmail_attackers WHERE killmail_id = $1 ORDER BY attacker_index`, killmailID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		a := Attacker{KillmailID: killmailID}
		var char, corp, ally, faction, ship, group, weapon *int32
		if err := rows.Scan(&a.AttackerIndex, &char, &corp, &ally, &faction,
			&ship, &group, &weapon, &a.DamageDone, &a.FinalBlow,
			&a.SecurityStatus, &a.KillmailTime); err != nil {
			rows.Close()
			return nil, err
		}
		a.CharacterID, a.CorporationID, a.AllianceID = deref(char), deref(corp), deref(ally)
		a.FactionID, a.ShipTypeID, a.ShipGroupID = deref(faction), deref(ship), deref(group)
		a.WeaponTypeID = deref(weapon)
		out.Attackers = append(out.Attackers, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = pool.Query(ctx, `
        SELECT item_index, type_id, flag_id,
               coalesce(quantity_dropped, 0), coalesce(quantity_destroyed, 0),
               coalesce(singleton, 0), parent_index
        FROM killmail_items WHERE killmail_id = $1 ORDER BY item_index`, killmailID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		it := Item{KillmailID: killmailID}
		var parent *int32
		if err := rows.Scan(&it.ItemIndex, &it.TypeID, &it.FlagID,
			&it.QuantityDropped, &it.QuantityDestroyed, &it.Singleton, &parent); err != nil {
			return nil, err
		}
		if parent == nil {
			it.ParentIndex = NoParent
		} else {
			it.ParentIndex = *parent
		}
		out.Items = append(out.Items, it)
	}
	return out, rows.Err()
}

func deref(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}
