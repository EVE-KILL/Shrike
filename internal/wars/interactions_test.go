package wars

import "testing"

// The war aggregation exists to answer "how did this war go", and the only way
// it can lie is by counting one dead ship more than once. Every test here is
// some form of that question.

func membership() Membership {
	return Membership{
		AggressorAllianceID:    1000,
		AggressorCorporationID: 100,
		DefenderAllianceIDs:    map[int32]bool{2000: true},
		DefenderCorporationIDs: map[int32]bool{200: true, 201: true},
	}
}

func TestSideRecognisesBothBelligerents(t *testing.T) {
	m := membership()

	if side, ok := m.Side(100, 1000); !ok || side != SideAggressor {
		t.Errorf("the aggressor was not recognised: side=%d ok=%v", side, ok)
	}
	if side, ok := m.Side(200, 2000); !ok || side != SideDefender {
		t.Errorf("the defender was not recognised: side=%d ok=%v", side, ok)
	}
	// An ally added to the war after it started is a defender too.
	if side, ok := m.Side(201, 0); !ok || side != SideDefender {
		t.Errorf("a war ally was not recognised as a defender: side=%d ok=%v", side, ok)
	}
	if _, ok := m.Side(999, 9999); ok {
		t.Error("an uninvolved third party was placed on a side")
	}
}

// A corporation in the aggressor alliance is an aggressor even if the
// corporation id itself is not the one named on the war.
func TestSideMatchesOnAllianceAlone(t *testing.T) {
	m := membership()
	if side, ok := m.Side(555, 1000); !ok || side != SideAggressor {
		t.Errorf("an alliance member was not matched by its alliance: side=%d ok=%v", side, ok)
	}
}

// The central guarantee: one killmail is one kill, whatever the fleet size.
func TestFleetDoesNotMultiplyTheKill(t *testing.T) {
	m := membership()
	victim := Victim{CharacterID: 900, CorporationID: 200, AllianceID: 2000}

	solo := Contributions(victim, []Attacker{
		{CharacterID: 1, CorporationID: 100, AllianceID: 1000, FinalBlow: true},
	}, m)

	// The same kill, but forty aggressors were on the mail.
	var fleet []Attacker
	for i := range 40 {
		fleet = append(fleet, Attacker{
			CharacterID:   int32(1 + i),
			CorporationID: 100,
			AllianceID:    1000,
			FinalBlow:     i == 0,
		})
	}
	many := Contributions(victim, fleet, m)

	if len(solo) != len(many) {
		t.Errorf("one attacker produced %d contributions and forty produced %d — "+
			"a fleet kill would count as forty kills", len(solo), len(many))
	}
}

// Every contribution must be distinct, or the same row is incremented twice
// for one killmail.
func TestContributionsAreDistinct(t *testing.T) {
	m := membership()
	victim := Victim{CharacterID: 900, CorporationID: 200, AllianceID: 2000}

	// Both aggressor and defender corporations shot; the victim is a defender.
	got := Contributions(victim, []Attacker{
		{CharacterID: 1, CorporationID: 100, AllianceID: 1000, FinalBlow: true},
		{CharacterID: 2, CorporationID: 201, AllianceID: 2000},
	}, m)

	seen := map[Contribution]bool{}
	for _, c := range got {
		if seen[c] {
			t.Errorf("duplicate contribution %+v — its row would be incremented twice", c)
		}
		seen[c] = true
	}
}

// The combined side is the war's own total and must always be present.
func TestCombinedSideAlwaysCounts(t *testing.T) {
	m := membership()
	victim := Victim{CharacterID: 900, CorporationID: 200, AllianceID: 2000}

	// A neutral third party landed the kill: on no side, but still in the war.
	got := Contributions(victim, []Attacker{
		{CharacterID: 7, CorporationID: 777, AllianceID: 7777, FinalBlow: true},
	}, m)

	var combinedKills int
	for _, c := range got {
		if c.Side == SideCombined && c.Category == CategoryKilled {
			combinedKills++
		}
	}
	if combinedKills != 3 {
		t.Errorf("a neutral's kill produced %d combined killed rows, want 3 "+
			"(character, corporation, alliance) — a kill in the war was lost "+
			"because nobody on a side made it", combinedKills)
	}

	// And it credited no side, because the killer was on neither.
	for _, c := range got {
		if c.Side != SideCombined && c.Category == CategoryKilled {
			t.Errorf("a neutral's kill was credited to side %d", c.Side)
		}
	}
}

// killed-by hangs off the victim's side, not the killer's: the row answers
// "what did this side lose ships to".
func TestKilledByIsAttributedToTheVictimSide(t *testing.T) {
	m := membership()
	victim := Victim{CharacterID: 900, CorporationID: 200, AllianceID: 2000} // defender

	got := Contributions(victim, []Attacker{
		{CharacterID: 1, CorporationID: 100, AllianceID: 1000, FinalBlow: true}, // aggressor
	}, m)

	var sides []int16
	for _, c := range got {
		if c.Category == CategoryKilledBy && c.Side != SideCombined {
			sides = append(sides, c.Side)
		}
	}
	if len(sides) == 0 {
		t.Fatal("no sided killed-by rows were produced")
	}
	for _, side := range sides {
		if side != SideDefender {
			t.Errorf("killed-by was attributed to side %d; the victim was a "+
				"defender, so it belongs to the defenders", side)
		}
	}
}

// Only the final blow is credited with the kill, not everyone who shot.
func TestOnlyTheFinalBlowIsCreditedWithKilledBy(t *testing.T) {
	m := membership()
	victim := Victim{CharacterID: 900, CorporationID: 200, AllianceID: 2000}

	got := Contributions(victim, []Attacker{
		{CharacterID: 1, CorporationID: 100, AllianceID: 1000, FinalBlow: true},
		{CharacterID: 2, CorporationID: 101, AllianceID: 1000},
		{CharacterID: 3, CorporationID: 102, AllianceID: 1000},
	}, m)

	for _, c := range got {
		if c.Category != CategoryKilledBy || c.TargetType != TargetCorporation {
			continue
		}
		if c.TargetID != 100 {
			t.Errorf("corporation %d was credited with the kill; only the "+
				"final-blow corporation (100) should be", c.TargetID)
		}
	}
}

// A mail with no final blow at all — which ESI does produce — must not panic
// or invent one.
func TestNoFinalBlowProducesNoKilledBy(t *testing.T) {
	m := membership()
	victim := Victim{CharacterID: 900, CorporationID: 200, AllianceID: 2000}

	got := Contributions(victim, []Attacker{
		{CharacterID: 1, CorporationID: 100, AllianceID: 1000},
	}, m)

	for _, c := range got {
		if c.Category == CategoryKilledBy {
			t.Errorf("a killed-by row %+v was produced for a mail with no final blow", c)
		}
	}
}

// Zero ids mean absent, and an absent entity must not become target id 0.
func TestZeroTargetsAreSkipped(t *testing.T) {
	m := membership()
	// An NPC victim: no character, no alliance.
	victim := Victim{CorporationID: 200}

	got := Contributions(victim, []Attacker{
		{CharacterID: 1, CorporationID: 100, AllianceID: 1000, FinalBlow: true},
	}, m)

	for _, c := range got {
		if c.TargetID == 0 {
			t.Errorf("contribution %+v targets id 0 — an absent entity became a real row", c)
		}
	}
}

// Both sides on one mail credits both, once each. This happens when a third
// party's ship dies in crossfire.
func TestBothSidesOnOneMailEachCountOnce(t *testing.T) {
	m := membership()
	victim := Victim{CharacterID: 900, CorporationID: 777, AllianceID: 7777}

	got := Contributions(victim, []Attacker{
		{CharacterID: 1, CorporationID: 100, AllianceID: 1000, FinalBlow: true},
		{CharacterID: 2, CorporationID: 200, AllianceID: 2000},
	}, m)

	counts := map[int16]int{}
	for _, c := range got {
		if c.Category == CategoryKilled {
			counts[c.Side]++
		}
	}
	// Three targets each (character, corporation, alliance) for combined,
	// aggressor and defender.
	for _, side := range []int16{SideCombined, SideAggressor, SideDefender} {
		if counts[side] != 3 {
			t.Errorf("side %d has %d killed rows, want 3", side, counts[side])
		}
	}
}
