package entities

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Deciding what a killmail made stale.
//
// Every kill names up to a few hundred characters, corporations and alliances.
// Refetching all of them would be hopeless, and refetching none of them means a
// killboard that never learns anyone's name. The rule is: fetch what is unknown,
// fetch what is older than StaleAfter, and fetch anyone the killmail proves has
// moved — a kill newer than our record showing a different corporation is direct
// evidence that the record is wrong, whatever its age.

// Affiliation is what a killmail asserts about one character at kill time.
type Affiliation struct {
	CharacterID   int32
	CorporationID int32
	AllianceID    int32
}

// Referenced is every entity a killmail mentions.
type Referenced struct {
	Affiliations []Affiliation
	Corporations []int32
	Alliances    []int32
	// KillmailTime is when the kill happened, which decides whether the
	// killmail's view of an affiliation is newer than the stored one.
	KillmailTime time.Time
}

// Stale reports which of the referenced entities need fetching.
//
// Three queries regardless of how many entities the killmail names, because a
// large fight names thousands and per-entity checks would cost more than the
// fetches they save.
func Stale(ctx context.Context, pool *pgxpool.Pool, ref Referenced) (Cascade, error) {
	var out Cascade

	characters, err := staleCharacters(ctx, pool, ref)
	if err != nil {
		return out, err
	}
	out.Characters = characters

	corporations, err := staleByUpdatedAt(ctx, pool, "corporations", "corporation_id", filterPlayerCorps(ref.Corporations))
	if err != nil {
		return out, err
	}
	out.Corporations = corporations

	alliances, err := staleByUpdatedAt(ctx, pool, "alliances", "alliance_id", dedupe(ref.Alliances))
	if err != nil {
		return out, err
	}
	out.Alliances = alliances

	return out, nil
}

func staleCharacters(ctx context.Context, pool *pgxpool.Pool, ref Referenced) ([]int32, error) {
	// Last mention wins when a character appears more than once, which happens
	// when someone is on a mail twice — the later entry is no less correct.
	byID := make(map[int32]Affiliation, len(ref.Affiliations))
	order := make([]int32, 0, len(ref.Affiliations))
	for _, a := range ref.Affiliations {
		if a.CharacterID == 0 {
			continue
		}
		if _, seen := byID[a.CharacterID]; !seen {
			order = append(order, a.CharacterID)
		}
		byID[a.CharacterID] = a
	}
	if len(order) == 0 {
		return nil, nil
	}

	type stored struct {
		corporation int32
		alliance    int32
		updatedAt   *time.Time
		deleted     bool
	}
	known := make(map[int32]stored, len(order))

	rows, err := pool.Query(ctx, `
        SELECT character_id, coalesce(corporation_id, 0), coalesce(alliance_id, 0),
               updated_at, coalesce(deleted, false)
        FROM characters WHERE character_id = ANY($1::int[])`, order)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int32
		var s stored
		if err := rows.Scan(&id, &s.corporation, &s.alliance, &s.updatedAt, &s.deleted); err != nil {
			rows.Close()
			return nil, err
		}
		known[id] = s
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	staleBefore := time.Now().Add(-StaleAfter)
	var out []int32

	for _, id := range order {
		s, seen := known[id]
		switch {
		case !seen:
			out = append(out, id)
		case s.deleted:
			// Biomassed. No amount of fetching brings them back, and a busy
			// killboard references dead characters forever.
			continue
		case s.updatedAt == nil || s.updatedAt.Before(staleBefore):
			out = append(out, id)
		case !ref.KillmailTime.IsZero() && s.updatedAt != nil && ref.KillmailTime.After(*s.updatedAt):
			// The killmail is newer than the record and disagrees with it.
			a := byID[id]
			corpMoved := a.CorporationID != 0 && a.CorporationID != s.corporation
			allyMoved := a.AllianceID != s.alliance
			if corpMoved || allyMoved {
				out = append(out, id)
			}
		}
	}
	return out, nil
}

// staleByUpdatedAt returns the ids that are absent or older than StaleAfter.
func staleByUpdatedAt(ctx context.Context, pool *pgxpool.Pool, table, column string, ids []int32) ([]int32, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	staleBefore := time.Now().Add(-StaleAfter)
	fresh := make(map[int32]bool, len(ids))

	rows, err := pool.Query(ctx,
		`SELECT `+column+`, updated_at FROM `+table+` WHERE `+column+` = ANY($1::int[])`, ids)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int32
		var updatedAt *time.Time
		if err := rows.Scan(&id, &updatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		if updatedAt != nil && !updatedAt.Before(staleBefore) {
			fresh[id] = true
		}
	}
	rows.Close()
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

func filterPlayerCorps(ids []int32) []int32 {
	var out []int32
	seen := map[int32]bool{}
	for _, id := range ids {
		if id == 0 || seen[id] || !IsPlayerCorporation(id) {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func dedupe(ids []int32) []int32 {
	var out []int32
	seen := map[int32]bool{}
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
