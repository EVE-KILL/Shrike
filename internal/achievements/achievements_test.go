package achievements

import (
	"strings"
	"testing"
)

// Achievement ids are the primary key of entity_achievements and appear in
// URLs, so they are a contract: a generated id that changes silently orphans
// every row already awarded under the old one.

func TestGeneratedIDsAreStableAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range All {
		if d.ID == "" {
			t.Errorf("achievement %q has no id", d.Name)
		}
		if seen[d.ID] {
			t.Errorf("duplicate achievement id %q — one of the two would be "+
				"unreachable and its awarded rows orphaned", d.ID)
		}
		seen[d.ID] = true
	}

	// A sample of the generated ids, pinned. These exist in production; a
	// change to the id format would strand them.
	for _, want := range []string{
		"babys_first_kill", "veteran_killer", "solo_first_blood",
		"highsec_hunter", "lowsec_prowler", "nullsec_warrior", "bling_hunter",
		"frigates_killer_10", "frigates_killer_1000", "frigates_loser_50",
		"capitals_killer_1", "industrials_loser_10",
	} {
		if !seen[want] {
			t.Errorf("achievement %q is missing — rows awarded under it would be orphaned", want)
		}
	}
}

// Every definition needs a positive threshold, or the completion arithmetic
// divides by zero.
func TestThresholdsArePositive(t *testing.T) {
	for _, d := range All {
		if d.Threshold <= 0 {
			t.Errorf("%s has threshold %d — the tier calculation divides by it", d.ID, d.Threshold)
		}
		if d.BasePoints <= 0 {
			t.Errorf("%s is worth %d base points", d.ID, d.BasePoints)
		}
	}
}

// Losses subtract, kills add. Getting the sign wrong would make dying the best
// way to climb the leaderboard.
func TestLossAchievementsAreNegative(t *testing.T) {
	var kills, losses int
	for _, d := range All {
		switch d.Trigger {
		case TriggerShipLosses:
			losses++
			if !d.Negative {
				t.Errorf("%s is a loss achievement but adds points", d.ID)
			}
			if d.SignedBasePoints() >= 0 {
				t.Errorf("%s has signed points %d, want negative", d.ID, d.SignedBasePoints())
			}
		case TriggerShipKills:
			kills++
			if d.Negative {
				t.Errorf("%s is a kill achievement but subtracts points", d.ID)
			}
		}
	}
	if kills == 0 || losses == 0 {
		t.Fatalf("generated %d kill and %d loss achievements", kills, losses)
	}
}

// The security bands must tile the whole range without gaps or overlap — a gap
// means a kill earns no location badge, an overlap means it earns two.
func TestSecurityBandsTileWithoutOverlap(t *testing.T) {
	bands := ByTrigger[TriggerKillsBySecurity]
	if len(bands) != 3 {
		t.Fatalf("expected three security bands, got %d", len(bands))
	}

	for _, sec := range []float64{1.0, 0.9, 0.5, 0.4, 0.1, 0.0, -0.5, -1.0} {
		var matched []string
		for _, d := range bands {
			if sec >= d.MinSec && sec < d.MaxSec {
				matched = append(matched, d.ID)
			}
		}
		if len(matched) != 1 {
			t.Errorf("security %.1f matched %v, want exactly one band", sec, matched)
		}
	}
}

// The kill-side collection: everyone on the mail is credited with the hull,
// but only the final blow gets the value and location badges.
func TestCollectCreditsTheRightAttackers(t *testing.T) {
	km := Killmail{
		TotalValue:        2_000_000_000,
		SystemSecurity:    0.9,
		HasSecurity:       true,
		IsSolo:            false,
		VictimShipGroupID: 25, // a frigate
		VictimCharacterID: 100,
		Attackers: []Attacker{
			{CharacterID: 1, FinalBlow: true},
			{CharacterID: 2},
		},
	}

	byChar := map[int32]map[string]bool{}
	for _, a := range collect(km) {
		if byChar[a.characterID] == nil {
			byChar[a.characterID] = map[string]bool{}
		}
		byChar[a.characterID][a.def.ID] = true
	}

	// Both attackers get the frigate kill.
	for _, id := range []int32{1, 2} {
		if !byChar[id]["frigates_killer_10"] {
			t.Errorf("attacker %d was not credited with the frigate kill", id)
		}
	}

	// Only the final blow gets the value and location badges.
	if !byChar[1]["bling_hunter"] || !byChar[1]["highsec_hunter"] {
		t.Error("the final blow was not credited with the value and location badges")
	}
	if byChar[2]["bling_hunter"] || byChar[2]["highsec_hunter"] {
		t.Error("an attacker without the final blow was credited with value or " +
			"location badges — every gang member would earn them")
	}

	// The victim gets the loss.
	if !byChar[100]["frigates_loser_10"] {
		t.Error("the victim was not credited with the frigate loss")
	}
}

// NPC kills must not award value or location badges, or ratting would earn the
// same badges as player combat.
func TestNPCKillsEarnNoValueOrLocationBadges(t *testing.T) {
	km := Killmail{
		TotalValue: 5_000_000_000, SystemSecurity: 0.9, HasSecurity: true,
		IsNPC: true, VictimShipGroupID: 25,
		Attackers: []Attacker{{CharacterID: 1, FinalBlow: true}},
	}

	for _, a := range collect(km) {
		if a.def.Trigger == TriggerKillsByValue || a.def.Trigger == TriggerKillsBySecurity {
			t.Errorf("an NPC kill awarded %q", a.def.ID)
		}
	}
}

// A character appearing twice on one mail, or being both attacker and victim,
// must produce one row per achievement — Postgres rejects an ON CONFLICT that
// touches the same row twice.
func TestMergeCollapsesDuplicates(t *testing.T) {
	def := All[0]
	merged := mergeAwards([]award{
		{characterID: 5, def: def, delta: 1},
		{characterID: 5, def: def, delta: 1},
		{characterID: 7, def: def, delta: 1},
	})

	if len(merged) != 2 {
		t.Fatalf("merged to %d rows, want 2 — a repeated key would make the "+
			"upsert fail outright", len(merged))
	}
	for _, a := range merged {
		if a.characterID == 5 && a.delta != 2 {
			t.Errorf("the duplicated award summed to %d, want 2", a.delta)
		}
	}
}

// Sorted by (character, achievement) so concurrent writers take row locks in
// the same order and cannot deadlock on a large fight.
func TestMergeOutputIsOrdered(t *testing.T) {
	merged := mergeAwards([]award{
		{characterID: 9, def: All[2], delta: 1},
		{characterID: 3, def: All[1], delta: 1},
		{characterID: 9, def: All[0], delta: 1},
		{characterID: 3, def: All[0], delta: 1},
	})

	for i := 1; i < len(merged); i++ {
		prev, cur := merged[i-1], merged[i]
		if prev.characterID > cur.characterID ||
			(prev.characterID == cur.characterID && prev.def.ID > cur.def.ID) {
			t.Fatalf("output is not ordered: (%d,%s) before (%d,%s)",
				prev.characterID, prev.def.ID, cur.characterID, cur.def.ID)
		}
	}
}

// An attacker with no character is an NPC and earns nothing.
func TestNPCAttackersEarnNothing(t *testing.T) {
	km := Killmail{
		VictimShipGroupID: 25, IsSolo: true,
		Attackers: []Attacker{{CharacterID: 0, FinalBlow: true}},
	}
	for _, a := range collect(km) {
		if a.characterID == 0 {
			t.Errorf("an award was made to character 0 (%s)", a.def.ID)
		}
	}
}

// Names and descriptions are user-visible; an empty one renders as a blank row.
func TestEveryAchievementIsDescribed(t *testing.T) {
	for _, d := range All {
		if strings.TrimSpace(d.Name) == "" || strings.TrimSpace(d.Description) == "" {
			t.Errorf("%s has an empty name or description", d.ID)
		}
		if strings.TrimSpace(d.Category) == "" {
			t.Errorf("%s has no category", d.ID)
		}
	}
}
