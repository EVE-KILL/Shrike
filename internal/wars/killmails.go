package wars

import (
	"context"
	"fmt"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Allies and the killmails fought under a war.

// StoreAllies records the third parties who joined a war.
//
// Insert-only: allies are appended to a war and never removed. war_allies has
// only a serial primary key, so ON CONFLICT cannot deduplicate the logical
// (war, alliance, corporation) identity; guard it explicitly instead.
func StoreAllies(ctx context.Context, pool *pgxpool.Pool, warID int32, w esi.War) (int64, error) {
	if len(w.Allies) == 0 {
		return 0, nil
	}

	var written int64
	for _, a := range w.Allies {
		if a.AllianceID == 0 && a.CorporationID == 0 {
			continue
		}
		tag, err := pool.Exec(ctx, `
            INSERT INTO war_allies (war_id, alliance_id, corporation_id)
            SELECT $1, $2, $3
            WHERE NOT EXISTS (
                SELECT 1 FROM war_allies
                WHERE war_id = $1
                  AND alliance_id IS NOT DISTINCT FROM $2::integer
                  AND corporation_id IS NOT DISTINCT FROM $3::integer
            )`,
			warID, nullID(a.AllianceID), nullID(a.CorporationID))
		if err != nil {
			return written, fmt.Errorf("store ally for war %d: %w", warID, err)
		}
		written += tag.RowsAffected()
	}
	return written, nil
}

// MissingKillmails walks a war's killmail list and returns the ones that still
// need war processing.
//
// ESI pages these, and a long war has thousands. The page loop stops on the
// first empty page rather than trusting a total, because ESI does not report
// one for this endpoint.
func MissingKillmails(ctx context.Context, pool *pgxpool.Pool, client *esi.Client, warID int32) ([]killmail.Ref, error) {
	var out []killmail.Ref

	for page := 1; ; page++ {
		res, err := esi.FetchWarKillmails(ctx, client, warID, page)
		if err != nil {
			return out, err
		}
		// A war with no killmail list at all answers 404 on the first page,
		// which is not an error — plenty of declared wars see no fighting.
		if res.Status == 400 || res.Status == 404 || res.Status == 422 {
			return out, nil
		}
		if !res.OK() || res.Data == nil {
			return out, fmt.Errorf("ESI returned %d for war %d killmail page %d", res.Status, warID, page)
		}
		refs := *res.Data
		if len(refs) == 0 {
			return out, nil
		}

		ids := make([]int64, 0, len(refs))
		for _, r := range refs {
			ids = append(ids, r.KillmailID)
		}

		stored, err := storedWarStates(ctx, pool, ids)
		if err != nil {
			return out, err
		}
		for _, r := range refs {
			if needsWarReplay(warID, stored[r.KillmailID]) {
				out = append(out, killmail.Ref{KillmailID: r.KillmailID, KillmailHash: r.KillmailHash})
			}
		}
	}
}

type storedWarState struct {
	warID     int32
	completed killmail.Effect
}

func needsWarReplay(warID int32, state storedWarState) bool {
	return state.warID != warID ||
		!killmail.IsComplete(state.completed, killmail.EffectWarInteractions)
}

func storedWarStates(ctx context.Context, pool *pgxpool.Pool, ids []int64) (map[int64]storedWarState, error) {
	rows, err := pool.Query(ctx, `
        SELECT k.killmail_id, coalesce(k.war_id, 0),
               coalesce(p.effects_completed, 0)
        FROM killmails k
        LEFT JOIN killmail_processing p USING (killmail_id)
        WHERE k.killmail_id = ANY($1::bigint[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]storedWarState, len(ids))
	for rows.Next() {
		var id int64
		var state storedWarState
		if err := rows.Scan(&id, &state.warID, &state.completed); err != nil {
			return nil, err
		}
		out[id] = state
	}
	return out, rows.Err()
}
