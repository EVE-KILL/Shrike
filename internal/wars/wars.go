// Package wars fetches and stores war declarations.
//
// A war is a small, slow-changing record with one awkward property: it stays
// interesting after it ends. Killmails reference war ids indefinitely, so a war
// finished in 2013 still has to resolve to a name and two belligerents.
//
// Three things therefore drive what gets refetched: wars ESI has that we do
// not, wars still running (or only just finished, since the final tallies land
// after the finish), and wars referenced by stored killmails whose metadata was
// never imported — which historical killmail backfills leave behind routinely.
package wars

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Refresh stores one war and returns what ESI said about it.
//
// The decoded war comes back so the caller can record its allies without
// fetching it a second time. A nil war with a nil error means ESI does not
// recognise the id — see below.
func Refresh(ctx context.Context, pool *pgxpool.Pool, client *esi.Client, warID int32) (*esi.War, error) {
	res, err := esi.FetchWar(ctx, client, warID)
	if err != nil {
		return nil, err
	}

	// A war id ESI does not recognise is a fact, not a failure. Historical
	// killmails reference ids CCP has since removed, and retrying them forever
	// spends the request budget on a permanent answer.
	if res.Status == 404 || res.Status == 422 {
		return nil, nil
	}
	if !res.OK() || res.Data == nil {
		return nil, fmt.Errorf("ESI returned %d for war %d", res.Status, warID)
	}

	w := *res.Data
	_, err = pool.Exec(ctx, `
        INSERT INTO wars (
            war_id, declared, started, finished, retracted, mutual, open_for_allies,
            aggressor_alliance_id, aggressor_corporation_id,
            aggressor_isk_destroyed, aggressor_ships_killed,
            defender_alliance_id, defender_corporation_id,
            defender_isk_destroyed, defender_ships_killed,
            created_at, updated_at
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,now(),now())
        ON CONFLICT (war_id) DO UPDATE SET
            declared = EXCLUDED.declared,
            started = EXCLUDED.started,
            finished = EXCLUDED.finished,
            retracted = EXCLUDED.retracted,
            mutual = EXCLUDED.mutual,
            open_for_allies = EXCLUDED.open_for_allies,
            aggressor_alliance_id = EXCLUDED.aggressor_alliance_id,
            aggressor_corporation_id = EXCLUDED.aggressor_corporation_id,
            aggressor_isk_destroyed = EXCLUDED.aggressor_isk_destroyed,
            aggressor_ships_killed = EXCLUDED.aggressor_ships_killed,
            defender_alliance_id = EXCLUDED.defender_alliance_id,
            defender_corporation_id = EXCLUDED.defender_corporation_id,
            defender_isk_destroyed = EXCLUDED.defender_isk_destroyed,
            defender_ships_killed = EXCLUDED.defender_ships_killed,
            updated_at = now()`,
		warID, esiTime(w.Declared), esiTime(w.Started), esiTime(w.Finished), esiTime(w.Retracted),
		w.Mutual, w.OpenForAllies,
		nullID(w.Aggressor.AllianceID), nullID(w.Aggressor.CorporationID),
		w.Aggressor.IskDestroyed, w.Aggressor.ShipsKilled,
		nullID(w.Defender.AllianceID), nullID(w.Defender.CorporationID),
		w.Defender.IskDestroyed, w.Defender.ShipsKilled)
	if err != nil {
		return nil, fmt.Errorf("upsert war %d: %w", warID, err)
	}
	return &w, nil
}

// Discover reports which wars need fetching.
type Discover struct {
	// New are wars ESI lists that are not stored.
	New []int32
	// Active are stored wars that have not finished, or finished recently
	// enough that the final tallies may still be moving.
	Active []int32
	// Missing are wars referenced by stored killmails but never imported.
	Missing []int32
}

// Total is how many fetches the discovery implies.
func (d Discover) Total() int { return len(d.New) + len(d.Active) + len(d.Missing) }

// MissingRepairBatch bounds how many orphaned war references are repaired per
// run. Historical backfills can leave tens of thousands; taking a hundred an
// hour clears the backlog steadily without ever competing with live work.
const MissingRepairBatch = 100

// Find works out what to fetch.
func Find(ctx context.Context, pool *pgxpool.Pool, client *esi.Client) (Discover, error) {
	var out Discover

	res, err := esi.FetchWarList(ctx, client, 0)
	if err != nil {
		return out, err
	}
	if !res.OK() || res.Data == nil {
		return out, fmt.Errorf("ESI returned %d for the war list", res.Status)
	}

	latest := *res.Data
	if len(latest) > 0 {
		rows, err := pool.Query(ctx,
			`SELECT war_id FROM wars WHERE war_id = ANY($1::int[])`, latest)
		if err != nil {
			return out, err
		}
		known := make(map[int32]bool, len(latest))
		for rows.Next() {
			var id int32
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return out, err
			}
			known[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return out, err
		}

		for _, id := range latest {
			if !known[id] {
				out.New = append(out.New, id)
			}
		}
	}

	// Finished wars are re-checked for a day afterwards: ESI keeps adjusting
	// the ISK and ship tallies after the war closes.
	out.Active, err = queryIDs(ctx, pool, `
        SELECT war_id FROM wars
        WHERE finished IS NULL OR finished >= now() - interval '24 hours'`)
	if err != nil {
		return out, err
	}

	out.Missing, err = queryIDs(ctx, pool, `
        SELECT DISTINCT k.war_id
        FROM killmails k
        LEFT JOIN wars w ON w.war_id = k.war_id
        WHERE k.war_id IS NOT NULL AND w.war_id IS NULL
        ORDER BY k.war_id
        LIMIT $1`, MissingRepairBatch)
	return out, err
}

func queryIDs(ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) ([]int32, error) {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func nullID(v int32) any {
	if v == 0 {
		return nil
	}
	return v
}

// esiTime maps an absent or unparseable timestamp to NULL rather than to the
// zero time, which Postgres would store as the year 1.
func esiTime(s string) any {
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return nil
}
