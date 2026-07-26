package battle

import (
	"testing"
	"time"
)

// Battle detection is heuristic, so the tests are about the filters rather than
// about exact output. Each filter exists because without it a specific wrong
// answer appears — an unrelated corporation on a side, one team's kills not
// matching the other's losses — and those are what is asserted.

var base = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

// killsAt builds n killmails inside one segment.
func killsAt(startID int64, at time.Time, n int, victimCorp int32) []Killmail {
	out := make([]Killmail, n)
	for i := range out {
		out[i] = Killmail{
			KillmailID:          startID + int64(i),
			KillmailTime:        at.Add(time.Duration(i) * time.Second),
			SolarSystemID:       30000142,
			RegionID:            10000002,
			TotalValue:          10_000_000,
			VictimCorporationID: victimCorp,
		}
	}
	return out
}

// A burst of kills is a battle; a trickle is not.
func TestBoundariesNeedAnActiveSegment(t *testing.T) {
	busy := killsAt(1, base, MinKillsPerSegment, 100)
	if RefineBoundaries(busy, MinKillsPerSegment, nil) == nil {
		t.Error("a segment at the threshold was not detected as a battle")
	}

	quiet := killsAt(1, base, MinKillsPerSegment-1, 100)
	if w := RefineBoundaries(quiet, MinKillsPerSegment, nil); w != nil {
		t.Errorf("a segment below the threshold was detected as a battle: %+v", w)
	}
}

// A long quiet stretch ends a battle, so two fights hours apart in one system
// are two battles rather than one very long one.
func TestQuietSplitsBattles(t *testing.T) {
	var kms []Killmail
	kms = append(kms, killsAt(1, base, 6, 100)...)
	// Well past MaxInactiveSegments of quiet.
	later := base.Add(time.Duration(MaxInactiveSegments+3) * SegmentDuration)
	kms = append(kms, killsAt(100, later, 6, 100)...)

	first := RefineBoundaries(kms, MinKillsPerSegment, nil)
	if first == nil {
		t.Fatal("no battle detected")
	}
	if !first.End.Before(later) {
		t.Errorf("the first battle ran to %s, past the gap — two separate fights "+
			"were merged into one", first.End)
	}

	// Asking for the later time selects the second fight.
	at := later.Add(time.Minute)
	second := RefineBoundaries(kms, MinKillsPerSegment, &at)
	if second == nil {
		t.Fatal("the second battle was not found")
	}
	if !second.Start.After(first.End) {
		t.Errorf("the second battle started at %s, not after the first ended at %s",
			second.Start, first.End)
	}
}

// A required time that falls in no active segment means the kill was not part
// of a battle at all.
func TestRequiredTimeOutsideAnyBattle(t *testing.T) {
	kms := killsAt(1, base, 6, 100)
	outside := base.Add(4 * time.Hour)
	if w := RefineBoundaries(kms, MinKillsPerSegment, &outside); w != nil {
		t.Errorf("a time outside every battle returned %+v", w)
	}
}

// twoSided builds a fight between corp 100 and corp 200.
func twoSided() ([]Killmail, map[int64][]Attacker) {
	var kms []Killmail
	atts := map[int64][]Attacker{}

	for i := range 6 {
		id := int64(1 + i)
		kms = append(kms, Killmail{
			KillmailID: id, KillmailTime: base.Add(time.Duration(i) * time.Second),
			SolarSystemID: 30000142, RegionID: 10000002,
			TotalValue: 100_000_000, VictimCorporationID: 100, VictimAllianceID: 1000,
		})
		atts[id] = []Attacker{
			{KillmailID: id, CharacterID: 900, CorporationID: 200, AllianceID: 2000,
				DamageDone: 1000, FinalBlow: true},
		}
	}
	return kms, atts
}

func TestTeamsAreOpposed(t *testing.T) {
	kms, atts := twoSided()
	a := AssignTeams(kms, atts)

	if len(a.CorpTeam) != 2 {
		t.Fatalf("assigned %d corporations, want 2", len(a.CorpTeam))
	}
	if a.CorpTeam[100] == a.CorpTeam[200] {
		t.Error("the attacker and the victim were put on the same side")
	}
	if a.MultiParty {
		t.Error("a straightforward two-sided fight was flagged multi-party")
	}
}

// The killwhore filter is what stops an opportunist dragging their whole
// corporation into a fight they had nothing to do with.
func TestKillwhoresAreExcluded(t *testing.T) {
	kms, atts := twoSided()
	for id := range atts {
		atts[id] = append(atts[id], Attacker{
			KillmailID: id, CharacterID: 999, CorporationID: 300,
			// Well under the threshold against the 1000 above.
			DamageDone: 10,
		})
	}

	a := AssignTeams(kms, atts)
	if _, present := a.CorpTeam[300]; present {
		t.Error("a corporation that did 1% of the damage was placed on a side — " +
			"every passing opportunist would become a belligerent")
	}
}

// NPCs shoot everyone, so counting them creates edges between corporations that
// never fought.
func TestNPCAttackersAreExcluded(t *testing.T) {
	kms, atts := twoSided()
	for id := range atts {
		atts[id] = append(atts[id], Attacker{
			KillmailID: id, CorporationID: 400, FactionID: 500001,
			CharacterID: 0, // no character: an NPC
			DamageDone:  5000,
		})
	}

	a := AssignTeams(kms, atts)
	if _, present := a.CorpTeam[400]; present {
		t.Error("an NPC attacker was placed on a side")
	}
}

// Corporations in one alliance fought together, whatever the greedy pass did.
func TestAllianceCohesion(t *testing.T) {
	var kms []Killmail
	atts := map[int64][]Attacker{}

	// Two corporations of alliance 2000 kill members of alliance 1000, and one
	// of them also takes a loss — enough to tempt the greedy pass to split them.
	for i := range 6 {
		id := int64(1 + i)
		attackerCorp := int32(200)
		if i%2 == 1 {
			attackerCorp = 201
		}
		kms = append(kms, Killmail{
			KillmailID: id, KillmailTime: base.Add(time.Duration(i) * time.Second),
			SolarSystemID: 30000142, TotalValue: 100_000_000,
			VictimCorporationID: 100, VictimAllianceID: 1000,
		})
		atts[id] = []Attacker{{
			KillmailID: id, CharacterID: 900, CorporationID: attackerCorp,
			AllianceID: 2000, DamageDone: 1000, FinalBlow: true,
		}}
	}

	a := AssignTeams(kms, atts)
	if a.CorpTeam[200] != a.CorpTeam[201] {
		t.Error("two corporations of the same alliance were split across sides")
	}
	if a.CorpTeam[200] == a.CorpTeam[100] {
		t.Error("the alliance ended up on the same side as its victims")
	}
}

// The exclusion that keeps the arithmetic honest: a corporation with no side
// contributes to neither team, so one side's kills cannot exceed the other's
// losses.
func TestUnassignedCorpsAreNotCounted(t *testing.T) {
	kms, atts := twoSided()

	// A structure lands the final blow on one kill while contributing almost
	// no damage. The real attacker is still there and still did the work, so
	// the structure is filtered out as a killwhore and has no side — which is
	// exactly the case the exclusion exists for.
	atts[1] = append(atts[1], Attacker{
		KillmailID: 1, CorporationID: 999, DamageDone: 1, FinalBlow: true,
	})
	// The original attacker keeps the damage but loses the final blow.
	atts[1][0].FinalBlow = false

	a := AssignTeams(kms, atts)
	if _, placed := a.CorpTeam[999]; placed {
		t.Fatal("the killwhoring structure was placed on a side, so this test is " +
			"not exercising the exclusion it claims to")
	}

	teams := ComputeTeamStats(kms, atts, a)

	total := teams[0].Kills + teams[1].Kills
	losses := teams[0].Losses + teams[1].Losses
	if total > losses {
		t.Errorf("%d kills against %d losses — an unassigned killer inflated a "+
			"side with no matching loss", total, losses)
	}
	// The kill whose final-blower has no side is credited to nobody. That is
	// the intended trade: an uncredited kill is better than one credited to a
	// participant who was not really in the fight.
	if total != losses-1 {
		t.Errorf("%d kills against %d losses — expected exactly one uncredited kill",
			total, losses)
	}
	for _, team := range teams {
		for _, e := range team.Entries {
			if e.CorporationID == 999 {
				t.Error("a corporation with no side appears in a team")
			}
		}
	}
}

func TestAllianceFallbackAttributesFilteredCorporations(t *testing.T) {
	kms, atts := twoSided()

	// This corporation is below the interaction threshold and therefore has no
	// direct corp-team placement, but the rest of its alliance is firmly on
	// the attacking side.
	atts[1] = append(atts[1], Attacker{
		KillmailID:    1,
		CharacterID:   999,
		CorporationID: 300,
		AllianceID:    2000,
		DamageDone:    1,
		FinalBlow:     true,
	})
	atts[1][0].FinalBlow = false

	a := AssignTeams(kms, atts)
	if _, placed := a.CorpTeam[300]; placed {
		t.Fatal("filtered corporation unexpectedly received a direct team")
	}

	teams := ComputeTeamStats(kms, atts, a)
	attackerTeam := a.CorpTeam[200]
	if teams[attackerTeam].Kills != int64(len(kms)) {
		t.Fatalf("attacking alliance received %d kills, want %d with alliance fallback",
			teams[attackerTeam].Kills, len(kms))
	}

	var filtered *TeamEntry
	for i := range teams[attackerTeam].Entries {
		if teams[attackerTeam].Entries[i].CorporationID == 300 {
			filtered = &teams[attackerTeam].Entries[i]
			break
		}
	}
	if filtered == nil {
		t.Fatal("alliance-sided filtered corporation is absent from team members")
	}
	if filtered.AllianceID != 2000 || filtered.Kills != 1 {
		t.Errorf("filtered corporation stats = %+v, want alliance 2000 and one kill", *filtered)
	}
}

// Every kill is somebody's loss, so the two must balance across the fight.
func TestKillsAndLossesBalance(t *testing.T) {
	kms, atts := twoSided()
	a := AssignTeams(kms, atts)
	teams := ComputeTeamStats(kms, atts, a)

	if got, want := teams[0].Kills+teams[1].Kills, int64(len(kms)); got != want {
		t.Errorf("%d kills recorded for %d killmails", got, want)
	}
	if got, want := teams[0].Losses+teams[1].Losses, int64(len(kms)); got != want {
		t.Errorf("%d losses recorded for %d killmails", got, want)
	}
	// And one side's kills are the other's losses.
	if teams[0].Kills != teams[1].Losses || teams[1].Kills != teams[0].Losses {
		t.Errorf("kills and losses do not mirror: %d/%d against %d/%d",
			teams[0].Kills, teams[0].Losses, teams[1].Kills, teams[1].Losses)
	}
}

// The whole pipeline.
func TestDetect(t *testing.T) {
	kms, atts := twoSided()

	b := Detect(kms, atts, nil)
	if b == nil {
		t.Fatal("a six-kill burst was not detected as a battle")
	}
	if b.KillCount != len(kms) {
		t.Errorf("kill count = %d, want %d", b.KillCount, len(kms))
	}
	if b.SolarSystemID != 30000142 {
		t.Errorf("solar system = %d", b.SolarSystemID)
	}
	if b.IskDestroyed != 600_000_000 {
		t.Errorf("isk destroyed = %.0f, want 600,000,000", b.IskDestroyed)
	}
	if len(b.KillmailIDs) != len(kms) {
		t.Errorf("recorded %d killmail ids for %d kills", len(b.KillmailIDs), len(kms))
	}
	if b.DurationMinutes <= 0 {
		t.Errorf("duration = %d minutes", b.DurationMinutes)
	}
}

// Too few kills is not a battle.
func TestDetectRejectsATrickle(t *testing.T) {
	kms := killsAt(1, base, 2, 100)
	if b := Detect(kms, map[int64][]Attacker{}, nil); b != nil {
		t.Errorf("two kills were detected as a battle: %+v", b)
	}
}

// The assignment must be deterministic — two runs over one fight cannot produce
// mirrored teams, because the result is stored and compared.
func TestAssignmentIsDeterministic(t *testing.T) {
	kms, atts := twoSided()

	first := AssignTeams(kms, atts)
	for range 5 {
		again := AssignTeams(kms, atts)
		for corp, team := range first.CorpTeam {
			if again.CorpTeam[corp] != team {
				t.Fatalf("corporation %d moved between sides across runs", corp)
			}
		}
	}
}

func TestDetectAllKeepsSeparateFightsInOneSystem(t *testing.T) {
	first, firstAttackers := twoSided()
	secondStart := base.Add(3 * time.Hour)

	var second []Killmail
	secondAttackers := map[int64][]Attacker{}
	for i := range 6 {
		id := int64(100 + i)
		second = append(second, Killmail{
			KillmailID: id, KillmailTime: secondStart.Add(time.Duration(i) * time.Second),
			SolarSystemID: 30000142, RegionID: 10000002,
			TotalValue: 100_000_000, VictimCorporationID: 100, VictimAllianceID: 1000,
		})
		secondAttackers[id] = []Attacker{{
			KillmailID: id, CharacterID: 900, CorporationID: 200, AllianceID: 2000,
			DamageDone: 1000, FinalBlow: true,
		}}
	}

	all := append(first, second...)
	for id, attackers := range secondAttackers {
		firstAttackers[id] = attackers
	}
	detected := DetectAll(all, firstAttackers)
	if len(detected) != 2 {
		t.Fatalf("detected %d battles, want both separated fights", len(detected))
	}
	if !detected[1].Start.After(detected[0].End) {
		t.Errorf("second fight %s..%s overlaps first %s..%s",
			detected[1].Start, detected[1].End,
			detected[0].Start, detected[0].End)
	}
}
