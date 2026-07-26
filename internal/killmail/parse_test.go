package killmail

import (
	"testing"

	"github.com/eve-kill/shrike/internal/eve"
)

// A cache with just enough in it to exercise the classification rules.
func testCache() *eve.Cache {
	return eve.NewCache(eve.CacheData{
		Types: map[int32]eve.Type{
			587:   {GroupID: 25, CategoryID: 6, Name: "Rifter"},          // frigate
			645:   {GroupID: 27, CategoryID: 6, Name: "Dominix"},         // battleship
			670:   {GroupID: 29, CategoryID: 6, Name: "Capsule"},         // pod
			29990: {GroupID: 963, CategoryID: 6, Name: "Loki"},           // strategic cruiser
			17738: {GroupID: 27, CategoryID: 6, Name: "Machariel"},       // faction battleship
			35832: {GroupID: 1657, CategoryID: 65, Name: "Astrahus"},     // structure
			17619: {GroupID: 380, CategoryID: 6, Name: "Impel"},          // hauler
			22436: {GroupID: 553, CategoryID: 11, Name: "Serpentis NPC"}, // entity
			2456:  {GroupID: 100, CategoryID: 18, Name: "Warrior I"},     // drone
			// Modules
			5973: {GroupID: 46, CategoryID: 7, Name: "1MN Afterburner"},
			2873: {GroupID: 645, CategoryID: 7, Name: "Small Smartbomb"},
			483:  {GroupID: 54, CategoryID: 7, Name: "Miner I"},
		},
		Groups: map[int32]eve.Group{
			25:   {CategoryID: 6, Name: "Frigate"},
			27:   {CategoryID: 6, Name: "Battleship"},
			29:   {CategoryID: 6, Name: "Capsule"},
			963:  {CategoryID: 6, Name: "Strategic Cruiser"},
			1657: {CategoryID: 65, Name: "Citadel"},
			553:  {CategoryID: 11, Name: "Entity"},
			380:  {CategoryID: 6, Name: "Deep Space Transport"},
			46:   {CategoryID: 7, Name: "Propulsion Module"},
			645:  {CategoryID: 7, Name: "Smart Bomb"},
			54:   {CategoryID: 7, Name: "Mining Laser"},
			100:  {CategoryID: 18, Name: "Light Scout Drone"},
		},
		Dogma: map[eve.DogmaKey]float64{
			{TypeID: 587, AttributeID: eve.AttrRigSize}:     1,
			{TypeID: 645, AttributeID: eve.AttrRigSize}:     3,
			{TypeID: 17738, AttributeID: eve.AttrRigSize}:   3,
			{TypeID: 5973, AttributeID: eve.AttrMetaLevel}:  4,
			{TypeID: 5973, AttributeID: eve.AttrHeatDamage}: 1,
			{TypeID: 2873, AttributeID: eve.AttrMetaLevel}:  0,
			{TypeID: 483, AttributeID: eve.AttrMetaLevel}:   0,
		},
	})
}

// The flag ranges decide both what counts as a fit and what the points
// algorithm looks at, so their boundaries are worth pinning: an off-by-one at
// either end silently reclassifies whole slot types.
func TestSlotClassification(t *testing.T) {
	fitted := []int32{11, 34, 87, 92, 95, 125, 132, 158, 162}
	for _, f := range fitted {
		if !isFittedSlot(f) {
			t.Errorf("flag %d should be a fitted slot", f)
		}
	}
	for _, f := range []int32{0, 5, 10, 35, 86, 91, 96, 124, 133, 157, 163} {
		if isFittedSlot(f) {
			t.Errorf("flag %d should not be a fitted slot", f)
		}
	}

	// The drone bay and fighter tubes are fitted but are not module slots:
	// counting their contents toward the danger factor would score a carrier's
	// fighter stock as a dangerous fit.
	for _, f := range []int32{87, 158, 162} {
		if isModuleSlot(f) {
			t.Errorf("flag %d should not count as a module slot", f)
		}
	}
}

// Items nest one level deep. The index a child records as its parent has to be
// the parent's own position in the flattened list, which only holds if the walk
// is depth-first.
func TestFlattenItemsNesting(t *testing.T) {
	items := flattenItems([]ESIItem{
		{ItemTypeID: 100, Flag: 5, QuantityDropped: 1},
		{ItemTypeID: 200, Flag: 12, QuantityDestroyed: 1, Items: []ESIItem{
			{ItemTypeID: 201, Flag: 5, QuantityDestroyed: 10},
			{ItemTypeID: 202, Flag: 5, QuantityDropped: 3},
		}},
		{ItemTypeID: 300, Flag: 5, QuantityDropped: 1},
	}, 42, nil, NoParent)

	if len(items) != 5 {
		t.Fatalf("got %d rows, want 5", len(items))
	}
	want := []struct {
		typeID int32
		parent int32
	}{
		{100, NoParent},
		{200, NoParent},
		{201, 1},
		{202, 1},
		{300, NoParent},
	}
	for i, w := range want {
		if items[i].ItemIndex != int32(i) {
			t.Errorf("row %d has index %d", i, items[i].ItemIndex)
		}
		if items[i].TypeID != w.typeID || items[i].ParentIndex != w.parent {
			t.Errorf("row %d = type %d parent %d, want type %d parent %d",
				i, items[i].TypeID, items[i].ParentIndex, w.typeID, w.parent)
		}
		if items[i].KillmailID != 42 {
			t.Errorf("row %d lost the killmail id", i)
		}
	}
}

func TestDetectSolo(t *testing.T) {
	player := ESIAttacker{CharacterID: 1}
	other := ESIAttacker{CharacterID: 2}
	npc := ESIAttacker{ShipTypeID: 22436}

	cases := []struct {
		name      string
		attackers []ESIAttacker
		want      bool
	}{
		{"one player", []ESIAttacker{player}, true},
		{"one player among rats", []ESIAttacker{player, npc, npc}, true},
		{"two players", []ESIAttacker{player, other}, false},
		{"one rat", []ESIAttacker{npc}, true},
		{"rat gang", []ESIAttacker{npc, npc}, false},
		{"nobody", nil, false},
	}
	for _, c := range cases {
		if got := detectSolo(c.attackers); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDetectNPC(t *testing.T) {
	cache := testCache()
	cases := []struct {
		name      string
		attackers []ESIAttacker
		want      bool
	}{
		{"pure rats", []ESIAttacker{{ShipTypeID: 22436}}, true},
		{"one capsuleer", []ESIAttacker{{CharacterID: 1, ShipTypeID: 587}}, false},
		{"capsuleer hiding among rats", []ESIAttacker{{ShipTypeID: 22436}, {CharacterID: 1}}, false},
		// An empty attacker flying a player hull is a disconnected capsuleer,
		// not an NPC.
		{"player hull, no pilot", []ESIAttacker{{ShipTypeID: 587}}, false},
		// An unknown type cannot be classified, so it does not veto.
		{"unknown hull", []ESIAttacker{{ShipTypeID: 999999}}, true},
		{"no attackers at all", nil, false},
	}
	for _, c := range cases {
		if got := detectNPC(cache, c.attackers); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// An attacker with no hull but a hull for a weapon is flying that hull.
func TestResolveAttackerShip(t *testing.T) {
	cache := testCache()
	cases := []struct {
		name string
		att  ESIAttacker
		want int32
	}{
		{"ship wins over weapon", ESIAttacker{ShipTypeID: 587, WeaponTypeID: 645}, 587},
		{"hull as weapon", ESIAttacker{WeaponTypeID: 645}, 645},
		{"module as weapon", ESIAttacker{WeaponTypeID: 5973}, 0},
		{"unknown weapon", ESIAttacker{WeaponTypeID: 999999}, 0},
		{"nothing", ESIAttacker{}, 0},
	}
	for _, c := range cases {
		if got := resolveAttackerShip(cache, c.att); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestRigSize(t *testing.T) {
	cache := testCache()
	// Strategic cruisers carry no rigSize attribute and are pinned at cruiser.
	if got := rigSize(cache, 29990); got != 2 {
		t.Errorf("strategic cruiser rig size = %v, want 2", got)
	}
	if got := rigSize(cache, 645); got != 3 {
		t.Errorf("battleship rig size = %v, want 3", got)
	}
	// An unknown type must not score as a capital.
	if got := rigSize(cache, 999999); got != 1 {
		t.Errorf("unknown rig size = %v, want the default of 1", got)
	}
}

// The scoring shortcuts each bail out for a different reason; all three return
// exactly 1 and none of them may be reached by accident.
func TestPointsShortcuts(t *testing.T) {
	cache := testCache()

	if got := calculatePoints(cache, &ESIKillmail{}); got != 1 {
		t.Errorf("no victim ship: got %d, want 1", got)
	}

	structureOnMail := &ESIKillmail{
		Victim:    ESIVictim{ShipTypeID: 645},
		Attackers: []ESIAttacker{{CharacterID: 1, ShipTypeID: 35832}},
	}
	if got := calculatePoints(cache, structureOnMail); got != 1 {
		t.Errorf("structure among attackers: got %d, want 1", got)
	}

	ratsOnly := &ESIKillmail{
		Victim:    ESIVictim{ShipTypeID: 645},
		Attackers: []ESIAttacker{{ShipTypeID: 22436}},
	}
	if got := calculatePoints(cache, ratsOnly); got != 1 {
		t.Errorf("no player attackers: got %d, want 1", got)
	}
}

// Points never drop below 1, and a bigger gang always scores no better than a
// smaller one on the same kill.
func TestPointsGangPenalty(t *testing.T) {
	cache := testCache()
	fit := []ESIItem{
		{ItemTypeID: 5973, Flag: 12, QuantityDestroyed: 1},
		{ItemTypeID: 5973, Flag: 13, QuantityDestroyed: 1},
		{ItemTypeID: 2873, Flag: 27, QuantityDestroyed: 1},
	}

	score := func(n int) int32 {
		attackers := make([]ESIAttacker, n)
		for i := range attackers {
			attackers[i] = ESIAttacker{CharacterID: int32(i + 1), ShipTypeID: 587}
		}
		return calculatePoints(cache, &ESIKillmail{
			Victim:    ESIVictim{ShipTypeID: 645, Items: fit},
			Attackers: attackers,
		})
	}

	solo, gang, blob := score(1), score(5), score(50)
	if solo < gang || gang < blob {
		t.Errorf("expected a monotonic gang penalty, got solo=%d gang=%d blob=%d", solo, gang, blob)
	}
	if blob < 1 {
		t.Errorf("points floor breached: %d", blob)
	}
}

// A mining module subtracts from the danger factor where a smartbomb adds to
// it, which is the whole reason the sign is tracked.
func TestPointsDangerFactorSign(t *testing.T) {
	cache := testCache()
	base := func(items []ESIItem) int32 {
		return calculatePoints(cache, &ESIKillmail{
			Victim:    ESIVictim{ShipTypeID: 645, Items: items},
			Attackers: []ESIAttacker{{CharacterID: 1, ShipTypeID: 587}},
		})
	}

	combat := base([]ESIItem{{ItemTypeID: 2873, Flag: 27, QuantityDestroyed: 4}})
	mining := base([]ESIItem{{ItemTypeID: 483, Flag: 27, QuantityDestroyed: 4}})
	if combat <= mining {
		t.Errorf("a combat fit should outscore a mining fit, got %d vs %d", combat, mining)
	}

	// Cargo is not a fit: the same modules in the hold must not count.
	cargo := base([]ESIItem{{ItemTypeID: 2873, Flag: 5, QuantityDestroyed: 4}})
	if cargo >= combat {
		t.Errorf("cargo modules scored as a fit: %d vs fitted %d", cargo, combat)
	}
}
