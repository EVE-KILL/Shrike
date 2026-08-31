package killtype

import (
	"slices"
	"testing"
	"time"
)

// Classify decides what the navigation counts say. It has to agree with
// Predicates, which decides what the lists behind those numbers contain — a
// kill counted here but not selected there is a page that says 41 and shows 40.

func has(types []string, want string) bool { return slices.Contains(types, want) }

// Every kill is in `latest`, which is why the predicate for it is TRUE.
func TestEverythingIsLatest(t *testing.T) {
	if !has(Classify(Subject{}), "latest") {
		t.Error("a killmail with nothing set is not in `latest`")
	}
}

func TestSecurityBuckets(t *testing.T) {
	cases := []struct {
		security float64
		region   int32
		want     string
	}{
		{0.9, 10000002, "highsec"},
		{0.5, 10000002, "highsec"},
		// 0.45 rounds to 0.5 in the client, so it is highsec — the boundary the
		// game itself uses for CONCORD response.
		{0.45, 10000002, "highsec"},
		{0.44, 10000002, "lowsec"},
		{0.1, 10000002, "lowsec"},
		{0.0, 10000002, "nullsec"},
		{-1.0, 10000002, "nullsec"},
	}
	for _, c := range cases {
		got := Classify(Subject{Security: c.security, HasSecurity: true, RegionID: c.region})
		if !has(got, c.want) {
			t.Errorf("security %.2f in region %d classified as %v, want %s",
				c.security, c.region, got, c.want)
		}
	}
}

// Wormhole and abyssal space have non-positive security but are not nullsec:
// their region ids sit above the nullsec ceiling, which is what excludes them.
// Without that, every w-space kill would inflate the nullsec count.
func TestWormholeAndAbyssalAreNotNullsec(t *testing.T) {
	for _, region := range []int32{WSpaceRegionMin, WSpaceRegionMax, AbyssalRegionMin, AbyssalRegionMax} {
		got := Classify(Subject{Security: -0.5, HasSecurity: true, RegionID: region})
		if has(got, "nullsec") {
			t.Errorf("region %d classified as nullsec: %v", region, got)
		}
	}
}

// Pochven and the Jove regions are counted as nullsec *as well as* their own
// bucket, because their region ids are below the nullsec ceiling.
//
// That is deliberate rather than an oversight — a Pochven kill is a kill in
// negative-security space and the nullsec page shows it — and it is asserted
// here because it looks like a bug and someone will eventually try to "fix" it.
// The predicates say the same, so changing one without the other is what would
// actually break: the count and the list would stop agreeing.
func TestPochvenAndJoveAreAlsoNullsec(t *testing.T) {
	for _, region := range append([]int32{PochvenRegionID}, JoveRegionIDs...) {
		got := Classify(Subject{Security: -0.5, HasSecurity: true, RegionID: region})
		if !has(got, "nullsec") {
			t.Errorf("region %d is no longer counted as nullsec (%v) — if that is "+
				"intended, the SQL predicate has to change with it", region, got)
		}
	}
}

func TestRegionBuckets(t *testing.T) {
	cases := []struct {
		region int32
		want   string
	}{
		{PochvenRegionID, "pochven"},
		{11_000_001, "wspace"},
		{11_000_033, "wspace"},
		{12_000_001, "abyssal"},
		{10_000_004, "jove"},
	}
	for _, c := range cases {
		if got := Classify(Subject{RegionID: c.region}); !has(got, c.want) {
			t.Errorf("region %d classified as %v, want %s", c.region, got, c.want)
		}
	}
}

// The ISK buckets are cumulative, not exclusive: a 10b kill is also 5b and big.
func TestValueBucketsAreCumulative(t *testing.T) {
	got := Classify(Subject{TotalValue: TenBISK, HasTotalValue: true})
	for _, want := range []string{"big", "5b", "10b"} {
		if !has(got, want) {
			t.Errorf("a %d ISK kill is not in %s: %v", TenBISK, want, got)
		}
	}

	got = Classify(Subject{TotalValue: BigISK, HasTotalValue: true})
	if !has(got, "big") {
		t.Error("a 1b kill is not in `big`")
	}
	if has(got, "5b") || has(got, "10b") {
		t.Errorf("a 1b kill was counted in a higher bucket: %v", got)
	}
}

// A supercarrier and a titan are also capitals, matching the predicates.
func TestCapitalHullsAreAlsoCapitals(t *testing.T) {
	for _, group := range []int32{30, 659} {
		got := Classify(Subject{VictimShipGroupID: group, HasVictimGroup: true})
		if !has(got, "capitals") {
			t.Errorf("group %d is not counted in `capitals`: %v", group, got)
		}
	}

	titan := Classify(Subject{VictimShipGroupID: 30, HasVictimGroup: true})
	if !has(titan, "titans") {
		t.Errorf("group 30 is not counted in `titans`: %v", titan)
	}
	if has(titan, "supercarriers") {
		t.Errorf("a titan was counted as a supercarrier: %v", titan)
	}
}

func TestCitadelsComeFromTheCategory(t *testing.T) {
	got := Classify(Subject{
		VictimShipGroupID: 1657, HasVictimGroup: true, GroupCategoryID: CategoryIDCitadel,
	})
	if !has(got, "citadels") {
		t.Errorf("a category-65 group is not counted in `citadels`: %v", got)
	}

	got = Classify(Subject{VictimShipGroupID: 25, HasVictimGroup: true, GroupCategoryID: 6})
	if has(got, "citadels") {
		t.Errorf("a ship group was counted as a citadel: %v", got)
	}
}

// Meta group zero and one both mean T1 — the SDE leaves it null for ordinary
// hulls, and treating that as "no tier" would lose most of the kills.
func TestMissingMetaGroupIsT1(t *testing.T) {
	for _, meta := range []int32{0, 1} {
		got := Classify(Subject{MetaGroupID: meta, HasVictimShip: true})
		if !has(got, "t1") {
			t.Errorf("meta group %d is not counted as t1: %v", meta, got)
		}
	}
}

func TestTechTiers(t *testing.T) {
	cases := []struct {
		meta int32
		want string
	}{
		{MetaT2, "t2"},
		{MetaT3Strategic, "t3"},
		{MetaFaction, "faction"},
	}
	for _, c := range cases {
		got := Classify(Subject{MetaGroupID: c.meta, HasVictimShip: true})
		if !has(got, c.want) {
			t.Errorf("meta group %d classified as %v, want %s", c.meta, got, c.want)
		}
		// And into exactly one tier.
		var tiers int
		for _, tier := range []string{"t1", "t2", "t3", "faction"} {
			if has(got, tier) {
				tiers++
			}
		}
		if tiers != 1 {
			t.Errorf("meta group %d landed in %d tech tiers: %v", c.meta, tiers, got)
		}
	}
}

// An unknown hull produces no tier rather than defaulting to t1, because a
// wrong bucket is worse than an absent one for a type we cannot resolve.
func TestUnknownHullHasNoTier(t *testing.T) {
	got := Classify(Subject{HasVictimShip: false})
	for _, tier := range []string{"t1", "t2", "t3", "faction"} {
		if has(got, tier) {
			t.Errorf("an unresolvable hull was counted as %s: %v", tier, got)
		}
	}
}

// Every name Classify can produce must exist in Types, or the rollup writes a
// row the navigation never reads.
func TestEveryProducedTypeIsRegistered(t *testing.T) {
	// A subject deliberately hitting as many branches as one killmail can.
	subjects := []Subject{
		{Security: 0.9, HasSecurity: true, RegionID: 10000002, IsSolo: true, IsNPC: true,
			TotalValue: TenBISK, HasTotalValue: true,
			VictimShipGroupID: 30, HasVictimGroup: true, HasVictimShip: true, MetaGroupID: MetaT2},
		{Security: -0.9, HasSecurity: true, RegionID: 11_000_005},
		{RegionID: PochvenRegionID},
		{RegionID: 12_000_003},
		{RegionID: 10_000_017},
		{VictimShipGroupID: 1657, HasVictimGroup: true, GroupCategoryID: CategoryIDCitadel},
		{HasVictimShip: true, MetaGroupID: MetaFaction},
		{HasVictimShip: true, MetaGroupID: MetaT3Strategic},
		{Security: 0.2, HasSecurity: true, RegionID: 10000002},
	}
	for _, group := range []int32{25, 420, 26, 419, 27, 513, 659, 30, 547} {
		subjects = append(subjects, Subject{VictimShipGroupID: group, HasVictimGroup: true})
	}

	for _, s := range subjects {
		for _, name := range Classify(s) {
			if !slices.Contains(Types, name) {
				t.Errorf("Classify produced %q, which is not in Types — the rollup "+
					"would write a row nothing reads", name)
			}
		}
	}
}

// Classify must never produce the same name twice, or one killmail increments
// a counter by two.
func TestNoDuplicateTypes(t *testing.T) {
	got := Classify(Subject{
		Security: -0.5, HasSecurity: true, RegionID: 10000002,
		IsSolo: true, IsNPC: true, TotalValue: TenBISK, HasTotalValue: true,
		VictimShipGroupID: 30, HasVictimGroup: true,
		HasVictimShip: true, MetaGroupID: MetaT2,
	})

	seen := map[string]bool{}
	for _, name := range got {
		if seen[name] {
			t.Errorf("Classify produced %q twice — that kill would count double in %s", name, name)
		}
		seen[name] = true
	}
}

func TestTimezoneBucketsCoverUTCWithoutGaps(t *testing.T) {
	want := []string{
		"timezone-us-east", "timezone-us-east", "timezone-us-east", "timezone-us-east",
		"timezone-us-west", "timezone-us-west", "timezone-us-west", "timezone-us-west",
		"timezone-au", "timezone-au", "timezone-au", "timezone-au", "timezone-au", "timezone-au",
		"timezone-ru", "timezone-ru", "timezone-ru",
		"timezone-eu", "timezone-eu", "timezone-eu", "timezone-eu", "timezone-eu",
		"timezone-us-east", "timezone-us-east",
	}
	for hour := 0; hour < 24; hour++ {
		got := TimezoneBucket(time.Date(2026, time.August, 31, hour, 0, 0, 0, time.UTC))
		if got != want[hour] {
			t.Errorf("hour %02d: got %q, want %q", hour, got, want[hour])
		}
	}
}

func TestSoloCanOverlapRawAttackerBand(t *testing.T) {
	got := Classify(Subject{IsSolo: true, IsNPC: false, AttackerCount: 3})
	for _, want := range []string{"solo", "attackers-2-4", "pvp"} {
		if !slices.Contains(got, want) {
			t.Errorf("Classify() = %v, missing %q", got, want)
		}
	}

	one := Classify(Subject{IsSolo: true, IsNPC: false, AttackerCount: 1})
	if slices.Contains(one, "attackers-1") {
		t.Errorf("solo one-attacker kill unexpectedly classified as attackers-1: %v", one)
	}
}

func TestValueBandsAreMutuallyExclusive(t *testing.T) {
	cases := []struct {
		value float64
		want  string
	}{
		{BigISK - 1, "under-1b"}, {BigISK, "1b-5b"}, {FiveBISK, "5b-10b"},
		{TenBISK, "10b-100b"}, {HundredBISK, "100b-1t"}, {TrillionISK, "1t-plus"},
	}
	all := []string{"under-1b", "1b-5b", "5b-10b", "10b-100b", "100b-1t", "1t-plus"}
	for _, tc := range cases {
		got := Classify(Subject{HasTotalValue: true, TotalValue: tc.value})
		for _, label := range all {
			if slices.Contains(got, label) != (label == tc.want) {
				t.Errorf("value %.0f: classifications %v, want only %q band", tc.value, got, tc.want)
			}
		}
	}
}
