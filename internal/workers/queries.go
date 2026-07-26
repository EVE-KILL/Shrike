package workers

import (
	"context"
	"time"

	"github.com/eve-kill/shrike/internal/entities"
	"github.com/eve-kill/shrike/internal/esi"
)

// The set-shaped queries the crons need.
//
// Every one of these answers "which of these thousands of ids is interesting?"
// in a single round trip. That is not an optimisation, it is the difference
// between a job that finishes and one that does not: the alternative shape —
// a lookup per id — turns a one-second job into a twenty-minute one and holds a
// connection for the duration.

// staleAlliances returns the alliance ids that are unknown or older than
// entities.StaleAfter.
func staleAlliances(ctx context.Context, d *Deps, ids []int32) ([]int32, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := d.Pool.Query(ctx, `
        SELECT alliance_id FROM alliances
        WHERE alliance_id = ANY($1::int[])
          AND updated_at IS NOT NULL
          AND updated_at >= $2`, ids, time.Now().Add(-entities.StaleAfter))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fresh := make(map[int32]bool, len(ids))
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		fresh[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []int32
	for _, id := range ids {
		if !fresh[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

// recentlyActiveCharacters returns characters worth re-checking for a change of
// corporation, most recently seen first.
//
// "Recently active" means recently seen on a killmail. Checking every character
// ever recorded would be millions of ids for a job that runs every minute, and
// the ones who have not appeared in months are also the ones least likely to
// have moved in a way anybody will look at.
func recentlyActiveCharacters(ctx context.Context, d *Deps, limit int) ([]int32, error) {
	rows, err := d.Pool.Query(ctx, `
        SELECT character_id FROM characters
        WHERE deleted IS NOT TRUE
          AND character_id > 0
        ORDER BY updated_at ASC NULLS FIRST
        LIMIT $1`, limit)
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

// changedAffiliations returns the characters whose stored corporation or
// alliance disagrees with what ESI just said.
//
// Only the disagreements are refetched. The bulk endpoint answers for a
// thousand characters at once, and the great majority have not moved — queuing
// all of them would turn a cheap check into a thousand full character fetches
// every minute.
func changedAffiliations(ctx context.Context, d *Deps, current []esi.Affiliation) ([]int32, error) {
	if len(current) == 0 {
		return nil, nil
	}

	ids := make([]int32, 0, len(current))
	for _, a := range current {
		ids = append(ids, a.CharacterID)
	}

	rows, err := d.Pool.Query(ctx, `
        SELECT character_id, coalesce(corporation_id, 0), coalesce(alliance_id, 0)
        FROM characters WHERE character_id = ANY($1::int[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type stored struct{ corporation, alliance int32 }
	known := make(map[int32]stored, len(ids))
	for rows.Next() {
		var id int32
		var s stored
		if err := rows.Scan(&id, &s.corporation, &s.alliance); err != nil {
			return nil, err
		}
		known[id] = s
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []int32
	for _, a := range current {
		s, seen := known[a.CharacterID]
		if !seen {
			out = append(out, a.CharacterID)
			continue
		}
		if s.corporation != a.CorporationID || s.alliance != a.AllianceID {
			out = append(out, a.CharacterID)
		}
	}
	return out, nil
}
