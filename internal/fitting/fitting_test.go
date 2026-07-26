package fitting

import (
	"testing"

	"github.com/eve-kill/shrike/internal/eve"
)

// The hash is a stored contract. Roughly 400,000 fits exist under hashes the
// TypeScript produced, so a change to the serialisation — a different
// separator, a different sort, charges creeping into the digest — orphans all
// of them silently. Every test here is ultimately about that.

// testCache builds a static-data cache with just enough to classify the types
// these tests use.
func testCache() *eve.Cache {
	return eve.NewCache(eve.CacheData{
		Types: map[int32]eve.Type{
			// Hull.
			587: {GroupID: 25, Name: "Rifter"},
			// Modules, two of them meta variants of 500.
			500: {GroupID: 60, Name: "125mm Gatling AutoCannon I"},
			501: {GroupID: 60, Name: "125mm Gatling AutoCannon II", VariationParentTypeID: 500},
			502: {GroupID: 60, Name: "Small Shield Extender I"},
			// Charge.
			600: {GroupID: 70, Name: "EMP S"},
			// Drone.
			700: {GroupID: 80, Name: "Hobgoblin I"},
		},
		Groups: map[int32]eve.Group{
			25: {CategoryID: categoryShip},
			60: {CategoryID: categoryModule},
			70: {CategoryID: categoryCharge},
			80: {CategoryID: categoryDrone},
		},
	})
}

// top builds a fitted (non-nested) item.
func top(typeID, flag int32) Item {
	return Item{TypeID: typeID, FlagID: flag, ParentIndex: -1}
}

func TestSlotGroupRanges(t *testing.T) {
	cases := []struct {
		flag int32
		want int32
	}{
		{27, SlotHigh}, {34, SlotHigh},
		{19, SlotMed}, {26, SlotMed},
		{11, SlotLow}, {18, SlotLow},
		{92, SlotRig}, {99, SlotRig},
		{125, SlotSubsystem}, {132, SlotSubsystem},
		{87, SlotDrone},
		// Outside every range — cargo, fighter tubes, unknown.
		{5, 0}, {35, 0}, {100, 0}, {159, 0},
	}
	for _, c := range cases {
		if got := SlotGroupForFlag(c.flag); got != c.want {
			t.Errorf("flag %d mapped to slot %d, want %d", c.flag, got, c.want)
		}
	}
}

// The same modules in a different order must hash identically, or a fit's
// identity would depend on the order ESI happened to list its items.
func TestHashIsOrderIndependent(t *testing.T) {
	c := testCache()

	a := Extract(c, 587, []Item{top(500, 27), top(502, 19), top(500, 28)})
	b := Extract(c, 587, []Item{top(500, 28), top(500, 27), top(502, 19)})

	if a == nil || b == nil {
		t.Fatal("a fitted hull produced no fit")
	}
	if a.FitHash != b.FitHash {
		t.Errorf("the same modules in a different order hashed differently:\n  %s\n  %s",
			a.FitHash, b.FitHash)
	}
}

// Charges are deliberately outside the hash: two ships with identical modules
// and different ammo are the same fit. Changing that would invalidate every
// stored hash.
func TestChargesDoNotChangeTheFitHash(t *testing.T) {
	c := testCache()

	bare := Extract(c, 587, []Item{top(500, 27)})
	loaded := Extract(c, 587, []Item{top(500, 27), top(600, 27)})

	if bare == nil || loaded == nil {
		t.Fatal("no fit extracted")
	}
	if bare.FitHash != loaded.FitHash {
		t.Error("loading ammo changed the fit hash — every stored fit would be " +
			"orphaned the first time one was re-extracted with its charge")
	}

	// But the charge is still recorded on the item, which is what the render
	// payload needs.
	if len(loaded.Items) != 1 || loaded.Items[0].ChargeTypeID != 600 {
		t.Errorf("the loaded charge was not attached to the module: %+v", loaded.Items)
	}
	if bare.Items[0].ChargeTypeID != 0 {
		t.Errorf("an unloaded module reported charge %d", bare.Items[0].ChargeTypeID)
	}
}

// Drones are outside the hash too, and are bucketed by type with a bay total.
func TestDronesAreBucketedButNotHashed(t *testing.T) {
	c := testCache()

	without := Extract(c, 587, []Item{top(500, 27)})
	with := Extract(c, 587, []Item{
		top(500, 27),
		{TypeID: 700, FlagID: DroneBayFlag, ParentIndex: -1, Quantity: 3},
		{TypeID: 700, FlagID: DroneBayFlag, ParentIndex: -1, Quantity: 2},
	})

	if without.FitHash != with.FitHash {
		t.Error("drones changed the fit hash")
	}

	var droneRows int
	for _, it := range with.Items {
		if it.SlotGroup == SlotDrone {
			droneRows++
			if it.Quantity != 5 {
				t.Errorf("drone quantity = %d, want 5 — rows of the same type must "+
					"stack into a bay total", it.Quantity)
			}
		}
	}
	if droneRows != 1 {
		t.Errorf("%d drone rows, want one per type", droneRows)
	}
}

// The family hash collapses meta variants onto their T1 root, so a doctrine
// flown in T2 and T1 clusters together.
func TestFamilyHashCollapsesVariants(t *testing.T) {
	c := testCache()

	t1 := Extract(c, 587, []Item{top(500, 27)})
	t2 := Extract(c, 587, []Item{top(501, 27)}) // variation parent is 500

	if t1.FitHash == t2.FitHash {
		t.Error("two different modules produced the same exact hash")
	}
	if t1.FamilyHash != t2.FamilyHash {
		t.Error("a T2 variant did not collapse onto its T1 root — meta variants " +
			"of one doctrine would not cluster")
	}
}

// Only fitted hardware counts. Cargo is nested and must be ignored, or a hold
// full of modules would read as a fit.
func TestNestedItemsAreIgnored(t *testing.T) {
	c := testCache()

	fitted := Extract(c, 587, []Item{top(500, 27)})
	withCargo := Extract(c, 587, []Item{
		top(500, 27),
		{TypeID: 502, FlagID: 27, ParentIndex: 0}, // inside a container
	})

	if fitted.FitHash != withCargo.FitHash {
		t.Error("an item inside a container changed the fit — cargo is not a fit")
	}
}

// A hull with nothing fitted has no fit identity, and neither does one carrying
// only drones.
func TestEmptyHullsHaveNoFit(t *testing.T) {
	c := testCache()

	if f := Extract(c, 587, nil); f != nil {
		t.Error("an empty hull produced a fit")
	}
	if f := Extract(c, 587, []Item{top(600, 27)}); f != nil {
		t.Error("a hull with only a charge produced a fit")
	}
	dronesOnly := Extract(c, 587, []Item{
		{TypeID: 700, FlagID: DroneBayFlag, ParentIndex: -1, Quantity: 5},
	})
	if dronesOnly != nil {
		t.Error("a hull with only drones produced a fit — a pod with a drone bay " +
			"would get a fit identity")
	}
}

// Ordinals must be stable across re-extracts, or the stored item rows churn for
// a fit whose identity has not changed.
func TestOrdinalsAreStable(t *testing.T) {
	c := testCache()

	// Two identical modules, different ammo. The charge breaks the tie.
	items := []Item{
		top(500, 27), top(600, 27),
		top(500, 28),
	}
	first := Extract(c, 587, items)
	second := Extract(c, 587, []Item{items[2], items[1], items[0]})

	if len(first.Items) != len(second.Items) {
		t.Fatalf("different item counts: %d and %d", len(first.Items), len(second.Items))
	}
	for i := range first.Items {
		if first.Items[i] != second.Items[i] {
			t.Errorf("item %d differs between extracts:\n  %+v\n  %+v",
				i, first.Items[i], second.Items[i])
		}
	}
}

// A different hull with the same modules is a different fit.
func TestHullIsPartOfTheIdentity(t *testing.T) {
	c := testCache()

	a := Extract(c, 587, []Item{top(500, 27)})
	b := Extract(c, 588, []Item{top(500, 27)})
	if a.FitHash == b.FitHash {
		t.Error("two hulls with the same modules hashed identically")
	}
}

// Only ships get fits.
func TestIsShipType(t *testing.T) {
	c := testCache()
	if !IsShipType(c, 587) {
		t.Error("a ship was not recognised as one")
	}
	if IsShipType(c, 500) {
		t.Error("a module was recognised as a ship")
	}
	if IsShipType(c, 999999) {
		t.Error("an unknown type was recognised as a ship")
	}
}
