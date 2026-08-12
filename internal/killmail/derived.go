package killmail

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/jackc/pgx/v5"
)

// The two effects that write to Postgres directly rather than enqueueing work.
//
// Both are additive — they add one to a counter or move a timestamp forward —
// which is what makes them belong under the ledger. Running either twice is a
// wrong number, not a duplicate row that a unique constraint would catch.

// Subject builds the classifier input from a parsed killmail.
//
// The security rating and the victim hull's meta group are not on the killmail
// row, so they come from the SDE cache. When the cache does not know the system
// — a newly added one, or a partial load — the region on the killmail row
// stands in, which is enough for the region-based subsets even though the
// security-based ones have to be skipped.
func Subject(cache *eve.Cache, km Killmail) killtype.Subject {
	s := killtype.Subject{
		RegionID:          km.RegionID,
		IsSolo:            km.IsSolo,
		IsNPC:             km.IsNPC,
		TotalValue:        km.TotalValue,
		HasTotalValue:     true,
		VictimShipGroupID: km.VictimShipGroupID,
	}

	if cache != nil {
		if sys, ok := cache.System(km.SolarSystemID); ok {
			s.Security = sys.Security
			s.HasSecurity = true
			if sys.RegionID != 0 {
				s.RegionID = sys.RegionID
			}
		}
		if km.VictimShipGroupID != 0 {
			s.HasVictimGroup = true
			if g, ok := cache.Group(km.VictimShipGroupID); ok {
				s.GroupCategoryID = g.CategoryID
			}
		}
		if km.VictimShipTypeID != 0 {
			if t, ok := cache.Type(km.VictimShipTypeID); ok {
				s.HasVictimShip = true
				s.MetaGroupID = t.MetaGroupID
			}
		}
	}

	return s
}

// BumpRollup adds this killmail to every daily subset it belongs to.
//
// Rows are sorted into primary-key order before the insert. Two workers
// bumping overlapping subsets in different orders would deadlock, and at the
// rate killmails arrive that is a certainty rather than a risk.
func BumpRollup(ctx context.Context, tx pgx.Tx, types []string, killmailTime string) error {
	if len(types) == 0 {
		return nil
	}

	sorted := append([]string(nil), types...)
	sort.Strings(sorted)

	values := make([]string, 0, len(sorted))
	args := make([]any, 0, len(sorted)+1)
	args = append(args, killmailTime)
	for i, t := range sorted {
		values = append(values, fmt.Sprintf("($1::date, $%d::text, 1)", i+2))
		args = append(args, t)
	}

	_, err := tx.Exec(ctx, fmt.Sprintf(`
        INSERT INTO kills_daily_count (date, type, count)
        VALUES %s
        ON CONFLICT (date, type) DO UPDATE
          SET count = kills_daily_count.count + EXCLUDED.count`,
		strings.Join(values, ", ")), args...)
	if err != nil {
		return fmt.Errorf("bump daily rollup: %w", err)
	}
	return nil
}

// UTCDateKey is the calendar day a killmail counts towards.
//
// UTC because the whole game runs on it: the client's "Today" header is EVE
// time, and a rollup keyed to anything else would disagree with it for part of
// every day.
func UTCDateKey(p Parsed) string {
	return p.Killmail.KillmailTime.UTC().Format("2006-01-02")
}

// UpdateLastActive moves each participating character's last_active forward.
//
// It also refreshes security_status from the attacker rows, which is not a
// tangent: ESI reports it on every attacker, so taking it here costs nothing
// and saves a /characters/{id}/ call per ganker — the population whose
// security status changes most and who would otherwise be refetched constantly.
//
// Only moves forward. Killmails arrive out of order and an old one must not
// drag a character's activity backwards.
func UpdateLastActive(ctx context.Context, tx pgx.Tx, p Parsed) error {
	// A character on several attacker rows is one entry: the map both
	// deduplicates and keeps the last security status seen, matching what the
	// TypeScript does.
	secByChar := make(map[int32]*float64, len(p.Attackers)+1)
	order := make([]int32, 0, len(p.Attackers)+1)

	remember := func(id int32, sec *float64) {
		if id == 0 {
			return
		}
		if _, seen := secByChar[id]; !seen {
			order = append(order, id)
		}
		secByChar[id] = sec
	}

	// The victim contributes activity but no security status — the mail does
	// not carry theirs.
	remember(p.Killmail.VictimCharacterID, nil)
	for _, a := range p.Attackers {
		remember(a.CharacterID, a.SecurityStatus)
	}

	if len(order) == 0 {
		return nil
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })

	ids := make([]int32, 0, len(order))
	secs := make([]*float64, 0, len(order))
	for _, id := range order {
		ids = append(ids, id)
		secs = append(secs, secByChar[id])
	}

	killTime := p.Killmail.KillmailTime.UTC()
	// UPDATE ... FROM does not guarantee the order in which it locks rows.
	// Lock every participating character by primary key first so concurrent
	// killmail and achievement workers cannot take overlapping rows in
	// opposite orders and deadlock.
	if _, err := tx.Exec(ctx, `
        SELECT character_id
        FROM characters
        WHERE character_id = ANY($1::int[])
        ORDER BY character_id
        FOR UPDATE`, ids); err != nil {
		return fmt.Errorf("lock characters for last_active update: %w", err)
	}

	_, err := tx.Exec(ctx, `
        UPDATE characters c
        SET last_active = $3::timestamptz,
            security_status = COALESCE(v.sec, c.security_status)
        FROM unnest($1::int[], $2::real[]) AS v(character_id, sec)
        WHERE c.character_id = v.character_id
          AND (c.last_active IS NULL OR c.last_active < $3::timestamptz)`,
		ids, secs, killTime)
	if err != nil {
		return fmt.Errorf("update last_active: %w", err)
	}
	return nil
}
