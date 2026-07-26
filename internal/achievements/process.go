package achievements

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Awarding achievements for one killmail.
//
// Incremental rather than recomputed: each killmail contributes a known delta
// to a known set of counters, so nothing here scans a table. That matters at
// this scale — recomputing "how many frigates has this character killed" per
// kill would be a full aggregate per attacker per killmail.

// Killmail is what the processor needs about a kill.
type Killmail struct {
	TotalValue        float64
	SystemSecurity    float64
	HasSecurity       bool
	IsNPC             bool
	IsSolo            bool
	VictimShipGroupID int32
	VictimCharacterID int32
	Attackers         []Attacker
}

// Attacker is one participant, reduced to what achievements care about.
type Attacker struct {
	CharacterID int32
	ShipGroupID int32
	FinalBlow   bool
}

// award is one pending counter increment.
type award struct {
	characterID int32
	def         Definition
	delta       int32
}

// Process applies one killmail's achievement increments.
func Process(ctx context.Context, pool *pgxpool.Pool, km Killmail) (int64, error) {
	pending := collect(km)
	if len(pending) == 0 {
		return 0, nil
	}

	merged := mergeAwards(pending)
	if err := upsert(ctx, pool, merged); err != nil {
		return 0, err
	}
	if err := syncPoints(ctx, pool, merged); err != nil {
		return 0, err
	}
	return int64(len(merged)), nil
}

// collect works out which achievements this killmail advances.
func collect(km Killmail) []award {
	var out []award

	for _, at := range km.Attackers {
		if at.CharacterID == 0 {
			continue
		}

		if at.FinalBlow {
			for _, d := range ByTrigger[TriggerFinalBlows] {
				out = append(out, award{at.CharacterID, d, 1})
			}
		}
		if km.IsSolo {
			for _, d := range ByTrigger[TriggerSoloKills] {
				out = append(out, award{at.CharacterID, d, 1})
			}
		}

		// Ship kills credit every attacker, not just the final blow: everyone
		// on the mail helped destroy the hull.
		if km.VictimShipGroupID != 0 {
			for _, d := range ByTrigger[TriggerShipKills] {
				if d.MatchesGroup(km.VictimShipGroupID) {
					out = append(out, award{at.CharacterID, d, 1})
				}
			}
		}

		// Value and location achievements go to the final blow only, and never
		// for an NPC kill — otherwise ratting would quietly award the same
		// badges as player combat.
		if at.FinalBlow && !km.IsNPC {
			for _, d := range ByTrigger[TriggerKillsByValue] {
				if km.TotalValue >= d.MinValue {
					out = append(out, award{at.CharacterID, d, 1})
				}
			}
			if km.HasSecurity {
				for _, d := range ByTrigger[TriggerKillsBySecurity] {
					if km.SystemSecurity >= d.MinSec && km.SystemSecurity < d.MaxSec {
						out = append(out, award{at.CharacterID, d, 1})
					}
				}
			}
		}
	}

	// The victim's own loss achievements.
	if km.VictimCharacterID != 0 && km.VictimShipGroupID != 0 {
		for _, d := range ByTrigger[TriggerShipLosses] {
			if d.MatchesGroup(km.VictimShipGroupID) {
				out = append(out, award{km.VictimCharacterID, d, 1})
			}
		}
	}

	return out
}

// mergeAwards collapses duplicates and orders the result.
//
// Both parts are load-bearing. Postgres rejects an ON CONFLICT statement that
// touches the same row twice, and a character can legitimately earn the same
// achievement twice from one killmail — appearing on the mail more than once,
// or being both an attacker and the victim. The ordering makes concurrent
// writers take row locks in the same sequence, which is what stops them
// deadlocking against each other on a large fight.
func mergeAwards(in []award) []award {
	type key struct {
		characterID int32
		id          string
	}
	byKey := map[key]*award{}
	for _, a := range in {
		k := key{a.characterID, a.def.ID}
		if existing, ok := byKey[k]; ok {
			existing.delta += a.delta
			continue
		}
		copied := a
		byKey[k] = &copied
	}

	out := make([]award, 0, len(byKey))
	for _, a := range byKey {
		out = append(out, *a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].characterID != out[j].characterID {
			return out[i].characterID < out[j].characterID
		}
		return out[i].def.ID < out[j].def.ID
	})
	return out
}

// upsert applies the increments in one statement.
//
// The tier arithmetic repeats a threshold achievement rather than capping it: a
// character with 250 frigate kills against a threshold of 50 has completed that
// achievement five times and is worth five times the points. GREATEST(1, ...)
// keeps a partially-complete achievement worth its base value rather than zero.
func upsert(ctx context.Context, pool *pgxpool.Pool, awards []award) error {
	values := make([]string, 0, len(awards))
	params := make([]any, 0, len(awards)*5)

	for i, a := range awards {
		base := i * 5
		values = append(values, fmt.Sprintf("($%d::int, $%d::text, $%d::int, $%d::int, $%d::int)",
			base+1, base+2, base+3, base+4, base+5))
		params = append(params, a.characterID, a.def.ID, a.delta, a.def.Threshold, a.def.SignedBasePoints())
	}

	// The placeholders are generated from the slice length; every value is
	// bound, never interpolated.
	_, err := pool.Exec(ctx, `
        INSERT INTO entity_achievements (
            entity_id, achievement_id, current_count, threshold,
            completion_tiers, is_completed, points, completed_at, last_updated
        )
        SELECT v.entity_id, v.achievement_id, v.delta, v.threshold,
               CASE WHEN v.delta >= v.threshold THEN 1 ELSE 0 END,
               v.delta >= v.threshold,
               v.signed_base * GREATEST(1, floor(v.delta::numeric / v.threshold)::int),
               CASE WHEN v.delta >= v.threshold THEN now() ELSE NULL END,
               now()
        FROM (VALUES `+strings.Join(values, ", ")+`)
             AS v(entity_id, achievement_id, delta, threshold, signed_base)
        ORDER BY v.entity_id, v.achievement_id
        ON CONFLICT (entity_id, achievement_id) DO UPDATE SET
            current_count = entity_achievements.current_count + EXCLUDED.current_count,
            completion_tiers = floor(
                (entity_achievements.current_count + EXCLUDED.current_count)::numeric
                / entity_achievements.threshold)::int,
            is_completed = (entity_achievements.current_count + EXCLUDED.current_count)
                >= entity_achievements.threshold,
            points = EXCLUDED.points * GREATEST(1, floor(
                (entity_achievements.current_count + EXCLUDED.current_count)::numeric
                / entity_achievements.threshold)::int),
            completed_at = COALESCE(entity_achievements.completed_at,
                CASE WHEN (entity_achievements.current_count + EXCLUDED.current_count)
                          >= entity_achievements.threshold THEN now() ELSE NULL END),
            last_updated = now()`, params...)
	return err
}

// syncPoints refreshes the denormalised total on each character touched.
//
// Recomputed as a SUM rather than adjusted by a delta, and that is deliberate:
// the upsert above sets `points` absolutely rather than additively, so there is
// no delta to apply. The total exists at all because the "top by achievement
// points" leaderboards would otherwise aggregate entity_achievements across
// every member of an alliance on each request.
func syncPoints(ctx context.Context, pool *pgxpool.Pool, awards []award) error {
	seen := map[int32]bool{}
	ids := make([]int32, 0, len(awards))
	for _, a := range awards {
		if !seen[a.characterID] {
			seen[a.characterID] = true
			ids = append(ids, a.characterID)
		}
	}

	_, err := pool.Exec(ctx, `
        UPDATE characters c
        SET achievement_points = COALESCE(sums.total, 0)
        FROM (
            SELECT entity_id, SUM(points)::int AS total
            FROM entity_achievements
            WHERE entity_id = ANY($1::int[])
            GROUP BY entity_id
        ) sums
        WHERE c.character_id = sums.entity_id`, ids)
	return err
}
