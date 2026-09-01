package killmail

import "testing"

func TestAllocatePointsDamageAndParticipation(t *testing.T) {
	shares := AllocatePoints(100, 1_000, []PointParticipant{
		{CharacterID: 1, DamageDone: 75},
		{CharacterID: 2, DamageDone: 25},
		{CharacterID: 3},
		{CharacterID: 4},
		{CharacterID: 0, DamageDone: 1_000_000},
	})

	want := map[int32]int64{1: 70, 2: 25, 3: 3, 4: 2}
	for id, points := range want {
		if shares[id] != points {
			t.Errorf("character %d received %d, want %d", id, shares[id], points)
		}
	}
	assertConserved(t, 100, shares)
}

func TestAllocatePointsMergesDuplicateCharacters(t *testing.T) {
	shares := AllocatePoints(11, 1_000, []PointParticipant{
		{CharacterID: 7, DamageDone: 20},
		{CharacterID: 7, DamageDone: 30, FinalBlow: true},
		{CharacterID: 8, DamageDone: 50},
	})
	if len(shares) != 2 {
		t.Fatalf("received %d character shares, want 2", len(shares))
	}
	assertConserved(t, 11, shares)
}

func TestAllocatePointsAllZeroDamageIsEqualAndDeterministic(t *testing.T) {
	shares := AllocatePoints(2, 1_000, []PointParticipant{
		{CharacterID: 30}, {CharacterID: 10}, {CharacterID: 20},
	})
	if shares[10] != 1 || shares[20] != 1 || shares[30] != 0 {
		t.Fatalf("zero-damage shares = %#v, want lowest ids to win equal remainders", shares)
	}
	assertConserved(t, 2, shares)
}

func TestAllocatePointsSoloGetsEntirePool(t *testing.T) {
	shares := AllocatePoints(137, 1_000, []PointParticipant{{CharacterID: 42}})
	if shares[42] != 137 {
		t.Fatalf("solo share = %d, want 137", shares[42])
	}
}

func TestAllocatePointsNoPlayersLeavesPoolUnallocated(t *testing.T) {
	shares := AllocatePoints(100, 1_000, []PointParticipant{{CharacterID: 0, DamageDone: 100}})
	if len(shares) != 0 {
		t.Fatalf("NPC-only allocation = %#v, want empty", shares)
	}
}

func assertConserved(t *testing.T, pool int64, shares map[int32]int64) {
	t.Helper()
	var total int64
	for _, points := range shares {
		total += points
	}
	if total != pool {
		t.Fatalf("allocated %d points from pool %d", total, pool)
	}
}
