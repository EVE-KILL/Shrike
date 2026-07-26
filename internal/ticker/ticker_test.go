package ticker

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
)

// The ticker is decoration, so these tests are not about exact wording. They
// are about the two things that would actually be wrong: announcing the same
// event twice under different guises, and announcing so much that the ticker
// stops being a signal.

func testCache() *eve.Cache {
	return eve.NewCache(eve.CacheData{
		Types: map[int32]eve.Type{
			671:  {Name: "Erebus", GroupID: GroupTitan},
			670:  {Name: "Capsule", GroupID: GroupCapsule},
			587:  {Name: "Rifter", GroupID: 25},
			2048: {Name: "Damage Control II", MetaGroupID: 2},
			// An officer module.
			14_042: {Name: "Draclira's Modified EM Ward Field", MetaGroupID: MetaOfficer},
		},
		Systems: map[int32]eve.System{
			30000142: {Name: "Jita", RegionID: 10000002, Security: 0.9},
		},
		Regions: map[int32]eve.Region{
			10000002: {Name: "The Forge"},
		},
	})
}

func kill(shipType, shipGroup int32, value float64, items ...killmail.Item) killmail.Parsed {
	return killmail.Parsed{
		Killmail: killmail.Killmail{
			KillmailID:        123456,
			SolarSystemID:     30000142,
			VictimShipTypeID:  shipType,
			VictimShipGroupID: shipGroup,
			TotalValue:        value,
		},
		Items: items,
	}
}

func TestOrdinaryKillsAreNotAnnounced(t *testing.T) {
	e := &Emitter{Cache: testCache()}
	// A Rifter worth ten million: the overwhelming majority of killmails.
	if spec, found := e.killmailSpec(kill(587, 25, 10_000_000)); found {
		t.Errorf("an ordinary kill was announced as %q — the ticker would become "+
			"a second killfeed", spec.Title)
	}
}

func TestTitanIsAnnouncedRegardlessOfValue(t *testing.T) {
	e := &Emitter{Cache: testCache()}
	// Deliberately below every ISK threshold: the hull is the story.
	spec, found := e.killmailSpec(kill(671, GroupTitan, 1_000_000))
	if !found {
		t.Fatal("a titan loss was not announced")
	}
	if !strings.HasPrefix(spec.Title, "Titan down") {
		t.Errorf("a titan was announced as %q", spec.Title)
	}
	if !strings.Contains(spec.Title, "Erebus") {
		t.Errorf("the announcement %q does not name the hull", spec.Title)
	}
}

// Significance order: a titan is also expensive and may also be officer-fit,
// and the announcement it gets must be the titan one.
func TestMostSignificantReasonWins(t *testing.T) {
	e := &Emitter{Cache: testCache()}

	// A titan that is also over every value threshold and officer-fit.
	titan := kill(671, GroupTitan, 200_000_000_000,
		killmail.Item{TypeID: 14_042, FlagID: 27, ParentIndex: killmail.NoParent})
	spec, found := e.killmailSpec(titan)
	if !found {
		t.Fatal("a titan loss was not announced")
	}
	if !strings.HasPrefix(spec.Title, "Titan down") {
		t.Errorf("a titan that was also high-value and officer-fit was announced "+
			"as %q — the most significant reason must win", spec.Title)
	}

	// A pod expensive enough to clear the high-value threshold is announced as
	// high-value, not as a pod: 25B of implants is the story, not the capsule.
	pod := kill(670, GroupCapsule, 30_000_000_000)
	spec, found = e.killmailSpec(pod)
	if !found {
		t.Fatal("an expensive pod was not announced")
	}
	if !strings.Contains(spec.Title, "destroyed —") {
		t.Errorf("a 30B pod was announced as %q, want the high-value form", spec.Title)
	}
}

// Exactly one announcement per killmail — the function returns a single Spec,
// so this checks the boundary cases produce one rather than none.
func TestEachSignificantKillProducesOneAnnouncement(t *testing.T) {
	e := &Emitter{Cache: testCache()}

	cases := []struct {
		name string
		p    killmail.Parsed
	}{
		{"titan", kill(671, GroupTitan, 1)},
		{"supercarrier", kill(671, GroupSupercarrier, 1)},
		{"high value", kill(587, 25, ThresholdHighValue)},
		{"officer fit", kill(587, 25, ThresholdOfficerFit,
			killmail.Item{TypeID: 14_042, FlagID: 27, ParentIndex: killmail.NoParent})},
		{"expensive pod", kill(670, GroupCapsule, ThresholdExpensivePod)},
	}

	for _, c := range cases {
		spec, found := e.killmailSpec(c.p)
		if !found {
			t.Errorf("%s was not announced", c.name)
			continue
		}
		if spec.Title == "" || spec.Expires == 0 {
			t.Errorf("%s produced an incomplete announcement: %+v", c.name, spec)
		}
		if spec.ID != -c.p.Killmail.KillmailID {
			t.Errorf("%s announcement id = %d, want %d (the negated killmail id)",
				c.name, spec.ID, -c.p.Killmail.KillmailID)
		}
	}
}

// One ISK below a threshold is not an announcement. Worth asserting because an
// off-by-one here silently changes how much the ticker says.
func TestThresholdsAreInclusiveLowerBounds(t *testing.T) {
	e := &Emitter{Cache: testCache()}

	if _, found := e.killmailSpec(kill(587, 25, ThresholdHighValue-1)); found {
		t.Error("a kill one ISK below the high-value threshold was announced")
	}
	if _, found := e.killmailSpec(kill(587, 25, ThresholdHighValue)); !found {
		t.Error("a kill exactly at the high-value threshold was not announced")
	}
	if _, found := e.killmailSpec(kill(670, GroupCapsule, ThresholdExpensivePod-1)); found {
		t.Error("a pod one ISK below the threshold was announced")
	}
}

// An officer module below the officer-fit ISK floor is not announced: cheap
// hulls carrying one expensive module are not the story.
func TestOfficerFitNeedsTheValueFloorToo(t *testing.T) {
	e := &Emitter{Cache: testCache()}
	p := kill(587, 25, ThresholdOfficerFit-1,
		killmail.Item{TypeID: 14_042, FlagID: 27, ParentIndex: killmail.NoParent})
	if _, found := e.killmailSpec(p); found {
		t.Error("an officer-fit kill below the ISK floor was announced")
	}
}

func TestOfficerModulesAreOnlyCountedInFittedSlots(t *testing.T) {
	e := &Emitter{Cache: testCache()}

	fitted := kill(587, 25, 5_000_000_000,
		killmail.Item{TypeID: 14_042, FlagID: 27, ParentIndex: killmail.NoParent})
	if !e.hasOfficerModules(fitted) {
		t.Error("an officer module in a high slot was not detected")
	}

	// Flag 5 is the cargo hold: freight, not a fit.
	cargo := kill(587, 25, 5_000_000_000,
		killmail.Item{TypeID: 14_042, FlagID: 5, ParentIndex: killmail.NoParent})
	if e.hasOfficerModules(cargo) {
		t.Error("an officer module in the cargo hold counted as an officer fit — " +
			"a hauler carrying one is not flying one")
	}

	// Inside a container: also not fitted.
	nested := kill(587, 25, 5_000_000_000,
		killmail.Item{TypeID: 14_042, FlagID: 27, ParentIndex: 0})
	if e.hasOfficerModules(nested) {
		t.Error("an officer module inside a container counted as an officer fit")
	}
}

func TestOrdinaryModulesAreNotOfficer(t *testing.T) {
	e := &Emitter{Cache: testCache()}
	p := kill(587, 25, 5_000_000_000,
		killmail.Item{TypeID: 2048, FlagID: 27, ParentIndex: killmail.NoParent})
	if e.hasOfficerModules(p) {
		t.Error("a tech II module was detected as officer meta")
	}
}

// Each category has its own id band, and two announcements must never collide.
func TestIDBandsDoNotCollide(t *testing.T) {
	// A killmail id and a battle id that would collide if the bands overlapped.
	killID := -int64(2_000_000_000)
	battleID := BattleIDBase - 1
	warID := WarIDBase - 1

	if killID == battleID || battleID == warID || killID == warID {
		t.Errorf("id bands collide: kill=%d battle=%d war=%d", killID, battleID, warID)
	}

	// A war's start and end must not share an id, or the end overwrites the start.
	start := WarIDBase - 700
	end := WarIDBase - 700 + WarEndedOffset
	if start == end {
		t.Error("a war's start and end announcements share an id")
	}
}

func TestWarEndedBodyReportsNoFighting(t *testing.T) {
	if body := WarEndedBody("A", "B", 0, 0, false); !strings.Contains(body, "without a shot") {
		t.Errorf("a war with no kills was summarised as %q", body)
	}
	if body := WarEndedBody("A", "B", 0, 0, true); !strings.Contains(body, "Retracted") {
		t.Errorf("a retracted war with no kills was summarised as %q", body)
	}
}

// Within ten percent is a draw. Declaring a winner there would be
// editorialising over noise.
func TestWarEndedBodyCallsCloseResultsEven(t *testing.T) {
	body := WarEndedBody("Aggressor", "Defender", 105, 100, false)
	if !strings.Contains(body, "Evenly matched") {
		t.Errorf("a 5%% lead was summarised as %q, want an even result", body)
	}

	body = WarEndedBody("Aggressor", "Defender", 1000, 100, false)
	if !strings.Contains(body, "Aggressor came out ahead") {
		t.Errorf("a decisive result was summarised as %q", body)
	}
}

func TestWarEndedBodyNamesTheRightLeader(t *testing.T) {
	body := WarEndedBody("Aggressor", "Defender", 100, 1000, false)
	if !strings.HasPrefix(body, "Defender came out ahead") {
		t.Errorf("the defender destroyed ten times as much but the summary was %q", body)
	}
}

func TestFormatISK(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{1_500_000_000_000, "1.5T"},
		{25_000_000_000, "25.0B"},
		{1_000_000_000, "1.0B"},
		{900_000_000, "900M"},
		{1_000_000, "1M"},
		{999_999, "999,999"},
		{1234, "1,234"},
		{0, "0"},
	}
	for _, c := range cases {
		if got := FormatISK(c.in); got != c.want {
			t.Errorf("FormatISK(%.0f) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSlotGroups(t *testing.T) {
	cases := []struct {
		flag int32
		want int32
	}{
		{27, 1}, {34, 1}, // high
		{19, 2}, {26, 2}, // mid
		{11, 3}, {18, 3}, // low
		{92, 4},  // rig
		{125, 5}, // subsystem
		{87, 6},  // drone bay
		{5, 0},   // cargo
		{0, 0},
	}
	for _, c := range cases {
		if got := SlotGroup(c.flag); got != c.want {
			t.Errorf("SlotGroup(%d) = %d, want %d", c.flag, got, c.want)
		}
	}
}

// A nil emitter is the CLI case and must be inert rather than a panic.
func TestNilEmitterIsInert(t *testing.T) {
	var e *Emitter
	ctx := context.Background()
	e.Emit(ctx, Spec{ID: 1, Expires: time.Minute})
	e.Expire(ctx, 1)
	e.EvaluateKillmail(ctx, kill(671, GroupTitan, 1))
	e.BattleExpired(ctx, 1)
}

// An emitter with no relay and no Redis is the other degraded case.
func TestEmitterWithoutOutputsIsInert(t *testing.T) {
	e := &Emitter{Cache: testCache()}
	ctx := context.Background()
	e.Emit(ctx, Spec{ID: 1, Expires: time.Minute})
	e.EvaluateKillmail(ctx, kill(671, GroupTitan, 1))
	e.TQOffline(ctx, "down")
	e.TQOnline(ctx, "up")
	e.WarStarted(ctx, 1, "A", "B", false, false)
	e.WarEnded(ctx, 1, "A", "B", 1, 2, false)
	e.BattleStarted(ctx, 1, "Jita", "The Forge", 200, 1e12)
}

// The location line is what tells a reader where the fight was; an unknown
// system must not render as an empty string.
func TestLocationTextFallsBack(t *testing.T) {
	e := &Emitter{Cache: testCache()}
	if got := e.locationText(30000142); got != "Jita, The Forge" {
		t.Errorf("locationText = %q, want \"Jita, The Forge\"", got)
	}
	if got := e.locationText(99999999); got != "Unknown" {
		t.Errorf("an unknown system rendered as %q, want \"Unknown\"", got)
	}
}
