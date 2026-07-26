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
// Insert-only: allies are appended to a war and never removed, so a conflict
// means we already had this one.
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
            VALUES ($1, $2, $3)
            ON CONFLICT DO NOTHING`,
			warID, nullID(a.AllianceID), nullID(a.CorporationID))
		if err != nil {
			return written, fmt.Errorf("store ally for war %d: %w", warID, err)
		}
		written += tag.RowsAffected()
	}
	return written, nil
}

// MissingKillmails walks a war's killmail list and returns the ones not stored.
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
		if res.Status == 404 || res.Status == 422 {
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

		stored, err := storedIDs(ctx, pool, ids)
		if err != nil {
			return out, err
		}
		for _, r := range refs {
			if !stored[r.KillmailID] {
				out = append(out, killmail.Ref{KillmailID: r.KillmailID, KillmailHash: r.KillmailHash})
			}
		}
	}
}

func storedIDs(ctx context.Context, pool *pgxpool.Pool, ids []int64) (map[int64]bool, error) {
	rows, err := pool.Query(ctx,
		`SELECT killmail_id FROM killmails WHERE killmail_id = ANY($1::bigint[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]bool, len(ids))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
