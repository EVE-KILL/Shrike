package workers

import (
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/killmail"
)

// FLEW_WITH is a pairwise expansion, so ordering matters.

func TestFlewWithPairsAreSymmetricAndOrdered(t *testing.T) {
	pairs := flewWithPairs([]int32{30, 10, 20})

	// Three participants make three pairs, not six: the relationship is
	// symmetric and storing both directions would double every weight.
	if len(pairs) != 3 {
		t.Fatalf("3 participants produced %d pairs, want 3", len(pairs))
	}
	for _, p := range pairs {
		if p.Lo >= p.Hi {
			t.Errorf("pair (%d, %d) is not ordered low-high, so the same "+
				"relationship could be stored twice under two keys", p.Lo, p.Hi)
		}
	}
}

func TestFlewWithNeedsTwoParticipants(t *testing.T) {
	if got := flewWithPairs([]int32{10}); got != nil {
		t.Errorf("a solo attacker produced %v", got)
	}
	if got := flewWithPairs(nil); got != nil {
		t.Errorf("no attackers produced %v", got)
	}
}

// Duplicates and NPCs must not create phantom pairs.
func TestFlewWithIgnoresDuplicatesAndNPCs(t *testing.T) {
	pairs := flewWithPairs([]int32{10, 10, 20, 0, 0})
	if len(pairs) != 1 {
		t.Fatalf("two distinct characters produced %d pairs, want 1", len(pairs))
	}
	if pairs[0].Lo != 10 || pairs[0].Hi != 20 {
		t.Errorf("pair = (%d, %d), want (10, 20)", pairs[0].Lo, pairs[0].Hi)
	}
}

// Large fleets must retain every pair, matching the TypeScript graph worker.
// Silently omitting the relationship for the largest fights changes the
// derived graph precisely where it is most useful.
func TestFlewWithIncludesLargeFleets(t *testing.T) {
	const participants = 101
	ids := make([]int32, participants)
	for i := range ids {
		ids[i] = int32(i + 1)
	}
	want := participants * (participants - 1) / 2
	if got := len(flewWithPairs(ids)); got != want {
		t.Errorf("%d participants produced %d pairs, want %d", participants, got, want)
	}
}

func TestBuildGraphKillmailMatchesRelationshipFilters(t *testing.T) {
	at := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	parsed := &killmail.Parsed{
		Killmail: killmail.Killmail{
			KillmailID:          99,
			KillmailTime:        at,
			SolarSystemID:       30_000_142,
			VictimCharacterID:   5_000_001,
			VictimCorporationID: 10_000_001,
			VictimAllianceID:    99,
			VictimShipGroupID:   graphSupercarrierGroup,
			TotalValue:          1_000_000,
		},
		Attackers: []killmail.Attacker{
			// NPC character.
			{CharacterID: 3_999_999, CorporationID: 10_000_002, DamageDone: 100},
			// NPC corporation.
			{CharacterID: 6_000_001, CorporationID: 1_000_100, DamageDone: 100},
			// Smartbomb damage does not prove these characters flew together.
			{CharacterID: 6_000_002, CorporationID: 10_000_002, WeaponTypeID: 55, DamageDone: 100},
			// Below five percent and not in a support/capital whitelist group.
			{CharacterID: 6_000_003, CorporationID: 10_000_002, DamageDone: 1},
			// Monitor is retained even at low damage and stamps the FC timestamp.
			{CharacterID: 6_000_004, CorporationID: 10_000_002, AllianceID: 10, ShipGroupID: graphMonitorGroup, DamageDone: 1},
			// Ordinary meaningful attacker.
			{CharacterID: 6_000_005, CorporationID: 10_000_003, AllianceID: 10, DamageDone: 100, FinalBlow: true},
			// A player attacker on the victim's alliance still gets KILLED and
			// OPERATED_IN, but is not claimed as a fleetmate of the other side.
			{CharacterID: 6_000_006, CorporationID: 10_000_004, AllianceID: 99, DamageDone: 100},
		},
	}

	got, ok := buildGraphKillmail(parsed, map[int32]bool{55: true})
	if !ok {
		t.Fatal("player killmail was dropped")
	}
	if len(got.Characters) != 4 {
		t.Fatalf("characters = %d, want victim plus 3 eligible attackers", len(got.Characters))
	}
	if len(got.Killed) != 3 {
		t.Fatalf("KILLED edges = %d, want one per eligible attacker", len(got.Killed))
	}
	if len(got.FlewWith) != 1 {
		t.Fatalf("FLEW_WITH edges = %d, want only the two same-side attackers", len(got.FlewWith))
	}

	byID := make(map[int64]time.Time)
	for _, ch := range got.Characters {
		byID[ch.ID] = ch.LastFCSeen
		if ch.ID == int64(parsed.Killmail.VictimCharacterID) && ch.LastSuperKill != at {
			t.Errorf("victim supercarrier timestamp = %v, want %v", ch.LastSuperKill, at)
		}
	}
	if byID[6_000_004] != at {
		t.Errorf("monitor FC timestamp = %v, want %v", byID[6_000_004], at)
	}
}

func TestBuildGraphKillmailDropsNPCKills(t *testing.T) {
	_, ok := buildGraphKillmail(&killmail.Parsed{
		Killmail: killmail.Killmail{IsNPC: true},
	}, nil)
	if ok {
		t.Fatal("NPC killmail was admitted to the relationship graph")
	}
}
