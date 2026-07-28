package stats

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotStored means the killmail the job names is not in the database.
//
// Routine rather than exceptional: the stats job is dispatched alongside other
// follow-up work, and a killmail can be deleted between the dispatch and the
// run. Nothing to count, and nothing to retry.
var ErrNotStored = errors.New("killmail is not stored")

// Load reads what the accumulator needs about one killmail.
//
// Read back from the database rather than carried on the job so the payload is
// a single id — which keeps the queue small and makes a replay produce exactly
// the same counters as the original run, since both read the same stored row.
func Load(ctx context.Context, pool *pgxpool.Pool, killmailID int64) (Killmail, []Attacker, error) {
	var km Killmail
	var attackerCount pgtype.Int4

	err := pool.QueryRow(ctx, `
        SELECT killmail_id, killmail_time,
               coalesce(solar_system_id, 0), coalesce(constellation_id, 0), coalesce(region_id, 0),
               coalesce(victim_character_id, 0), coalesce(victim_corporation_id, 0),
               coalesce(victim_alliance_id, 0), coalesce(victim_faction_id, 0),
               coalesce(victim_ship_type_id, 0),
               coalesce(victim_damage_taken, 0),
               coalesce(total_value, 0), coalesce(points, 0), attacker_count,
               coalesce(is_npc, false), coalesce(is_solo, false)
        FROM killmails WHERE killmail_id = $1`, killmailID).
		Scan(&km.KillmailID, &km.KillmailTime,
			&km.SolarSystemID, &km.ConstellationID, &km.RegionID,
			&km.VictimCharacterID, &km.VictimCorporationID,
			&km.VictimAllianceID, &km.VictimFactionID, &km.VictimShipTypeID,
			&km.VictimDamageTaken,
			&km.TotalValue, &km.Points, &attackerCount,
			&km.IsNPC, &km.IsSolo)
	if errors.Is(err, pgx.ErrNoRows) {
		return km, nil, ErrNotStored
	}
	if err != nil {
		return km, nil, err
	}

	rows, err := pool.Query(ctx, `
        SELECT coalesce(character_id, 0), coalesce(corporation_id, 0),
               coalesce(alliance_id, 0), coalesce(faction_id, 0),
               coalesce(ship_type_id, 0), coalesce(damage_done, 0), coalesce(final_blow, false)
        FROM killmail_attackers WHERE killmail_id = $1`, killmailID)
	if err != nil {
		return km, nil, err
	}
	defer rows.Close()

	var attackers []Attacker
	for rows.Next() {
		var a Attacker
		if err := rows.Scan(&a.CharacterID, &a.CorporationID, &a.AllianceID, &a.FactionID,
			&a.ShipTypeID, &a.DamageDone, &a.FinalBlow); err != nil {
			return km, nil, err
		}
		attackers = append(attackers, a)
	}
	if err := rows.Err(); err != nil {
		return km, nil, err
	}

	// Historical/imported rows can predate attacker_count. The TypeScript
	// writer falls back to the number of stored attacker rows only when the
	// column is NULL (an explicit zero stays zero), so preserve that distinction
	// rather than coalescing NULL to zero in SQL.
	km.AttackerCount = resolvedAttackerCount(attackerCount, len(attackers))
	return km, attackers, nil
}

func resolvedAttackerCount(stored pgtype.Int4, rows int) int64 {
	if stored.Valid {
		return int64(stored.Int32)
	}
	return int64(rows)
}
