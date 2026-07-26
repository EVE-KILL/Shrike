package stats

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The counters are sums, which makes every bug here cumulative and silent: a
// kill counted twice is not visible on the kill, only on a total that is wrong
// by one forever. So the tests assert the arithmetic directly rather than
// checking that something was written.

const statsTestDSN = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"

// Isolation is by period, not by id band — and that distinction cost real data
// to learn.
//
// The obvious approach, reserving entity ids above anything CCP has issued,
// does not work here: entity_id is an int32 and EVE character ids have already
// reached 2,124,556,013, leaving no headroom below the 2,147,483,647 ceiling.
// A cleanup of `entity_id >= 2_100_000_000` deletes the stats of every modern
// character, which is exactly what happened — 10,991 of them on this database.
//
// testPeriod is a date no killmail can carry, so every row these tests write is
// unambiguously theirs and the cleanup can be scoped to it. Ids are then free to
// be realistic, which they need to be: the accumulator's behaviour depends on
// NPC-corporation thresholds and ship group membership.
var testPeriod = time.Date(1970, 1, 1, 12, 0, 0, 0, time.UTC)

// testIDBase keeps in-memory fixtures clear of real ids for readability. It is
// NOT an isolation mechanism — see above.
const testIDBase int32 = 2_100_000_000

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = statsTestDSN
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// clearTestStats removes only rows in the test period.
//
// Scoped by period_start rather than by entity id, because no real killmail can
// land on testPeriod and every real one would otherwise be at risk.
func clearTestStats(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	day := testPeriod.Format("2006-01-02")
	clear := func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM stats WHERE period_start = $1::date`, day)
		_, _ = pool.Exec(ctx, `DELETE FROM stats_breakdowns WHERE period_start = $1::date`, day)
	}
	clear()
	t.Cleanup(clear)
}

// A fight where one corporation brings three members, two of them in the same
// hull, against a victim from another corporation.
func fleetKill() (Killmail, []Attacker) {
	km := Killmail{
		KillmailID:          137_000_001,
		KillmailTime:        testPeriod,
		SolarSystemID:       30000142,
		ConstellationID:     20000020,
		RegionID:            10000002,
		VictimCharacterID:   testIDBase + 100,
		VictimCorporationID: testIDBase + 200,
		VictimAllianceID:    testIDBase + 300,
		VictimShipTypeID:    17738,
		VictimDamageTaken:   9000,
		TotalValue:          1_000_000_000,
		Points:              25,
		AttackerCount:       3,
	}
	attackers := []Attacker{
		{CharacterID: testIDBase + 1, CorporationID: testIDBase + 10, AllianceID: testIDBase + 20,
			ShipTypeID: 587, DamageDone: 5000, FinalBlow: true},
		{CharacterID: testIDBase + 2, CorporationID: testIDBase + 10, AllianceID: testIDBase + 20,
			ShipTypeID: 587, DamageDone: 3000},
		{CharacterID: testIDBase + 3, CorporationID: testIDBase + 10, AllianceID: testIDBase + 20,
			ShipTypeID: 621, DamageDone: 1000},
	}
	return km, attackers
}

// The central rule: an organisation counts one kill however many members it
// brought. Counting per attacker would inflate every corporation's kill count
// by its average fleet size.
func TestOrganisationsCountOneKillPerKillmail(t *testing.T) {
	km, attackers := fleetKill()
	a := NewAccumulator()
	a.Add(km, attackers)

	corp := a.Stats[StatsKey{EntityCorporation, testIDBase + 10}]
	if corp == nil {
		t.Fatal("the attacking corporation has no stats row")
	}
	if corp.Kills != 1 {
		t.Errorf("the corporation counted %d kills for one killmail with three of its "+
			"members on it, want 1", corp.Kills)
	}
	if corp.IskDestroyed != km.TotalValue {
		t.Errorf("the corporation counted %.0f ISK destroyed, want %.0f — counting per "+
			"attacker would treble it", corp.IskDestroyed, km.TotalValue)
	}

	ally := a.Stats[StatsKey{EntityAlliance, testIDBase + 20}]
	if ally == nil || ally.Kills != 1 {
		t.Errorf("the alliance counted %v kills, want 1", ally)
	}

	// Characters are already distinct, so each gets its own kill.
	for i := int32(1); i <= 3; i++ {
		c := a.Stats[StatsKey{EntityCharacter, testIDBase + i}]
		if c == nil || c.Kills != 1 {
			t.Errorf("character %d counted %v kills, want 1", testIDBase+i, c)
		}
	}
}

// Only the attacker who landed the final blow is credited with it.
func TestFinalBlowIsCreditedOnce(t *testing.T) {
	km, attackers := fleetKill()
	a := NewAccumulator()
	a.Add(km, attackers)

	var total int64
	for i := int32(1); i <= 3; i++ {
		total += a.Stats[StatsKey{EntityCharacter, testIDBase + i}].FinalBlows
	}
	if total != 1 {
		t.Errorf("%d characters were credited with the final blow, want exactly 1", total)
	}
	if got := a.Stats[StatsKey{EntityCharacter, testIDBase + 1}].FinalBlows; got != 1 {
		t.Errorf("the attacker with final_blow set was credited %d times", got)
	}
}

// Damage is per character, not shared.
func TestDamageIsAttributedPerCharacter(t *testing.T) {
	km, attackers := fleetKill()
	a := NewAccumulator()
	a.Add(km, attackers)

	want := map[int32]int64{testIDBase + 1: 5000, testIDBase + 2: 3000, testIDBase + 3: 1000}
	for id, w := range want {
		if got := a.Stats[StatsKey{EntityCharacter, id}].DamageDealt; got != w {
			t.Errorf("character %d dealt %d damage, want %d", id, got, w)
		}
	}

	if got := a.Stats[StatsKey{EntityCharacter, km.VictimCharacterID}].DamageTaken; got != km.VictimDamageTaken {
		t.Errorf("the victim took %d damage, want %d", got, km.VictimDamageTaken)
	}
}

// A hull is credited once per killmail however many pilots brought it —
// otherwise a fleet of twenty identical ships makes that hull look twenty times
// as popular as it was.
func TestDistinctHullsAreCountedOnce(t *testing.T) {
	km, attackers := fleetKill()
	a := NewAccumulator()
	a.Add(km, attackers)

	// Two attackers flew a Rifter (587), one flew something else.
	rifter := a.Stats[StatsKey{EntityShip, 587}]
	if rifter == nil || rifter.Kills != 1 {
		t.Errorf("the hull flown by two attackers counted %v kills, want 1", rifter)
	}

	// The corporation's ship-flown breakdown does NOT collapse the same way,
	// and that is deliberate fidelity rather than a bug here.
	//
	// The TypeScript pushes one breakdown row per matching member and its merge
	// sums them (`existing.kills += r.kills`), so a hull flown by two members
	// records 2 kills — while the corporation's headline row records 1. The two
	// numbers disagree about what "a kill" means: the headline counts killmails,
	// the ship-flown breakdown counts pilot-kills.
	//
	// It is reproduced exactly because the stats port is verified against the
	// TypeScript as the oracle, and changing it would silently alter every
	// stored corporation and alliance ship breakdown. Worth revisiting once the
	// numbers have been compared end to end — but as a decision, not as a
	// side effect of a rewrite.
	b := a.Breakdowns[BreakdownKey{EntityCorporation, testIDBase + 10, DimShipFlown, 587}]
	if b == nil {
		t.Fatal("the corporation has no ship-flown breakdown")
	}
	if b.Kills != 2 {
		t.Errorf("the corporation's ship-flown breakdown counted %d kills for a hull "+
			"two of its members flew, want 2 — matching the TypeScript, which sums "+
			"per-member rows here even though the headline row counts 1", b.Kills)
	}
}

// The victim's hull is a loss for that hull, and the attackers' hulls are kills
// — the same entity type counting in both directions.
func TestVictimShipCountsAsALoss(t *testing.T) {
	km, attackers := fleetKill()
	a := NewAccumulator()
	a.Add(km, attackers)

	victimHull := a.Stats[StatsKey{EntityShip, km.VictimShipTypeID}]
	if victimHull == nil || victimHull.Losses != 1 {
		t.Errorf("the victim's hull counted %v losses, want 1", victimHull)
	}
	if victimHull.Kills != 0 {
		t.Errorf("the victim's hull counted %d kills", victimHull.Kills)
	}
}

// Attacker and victim flying the same hull is common, and it must produce one
// row with both counters rather than two rows — Postgres rejects an ON CONFLICT
// statement that touches the same row twice.
func TestSharedKeysCollapseToOneRow(t *testing.T) {
	km, attackers := fleetKill()
	// The victim is flying the same hull as two of the attackers.
	km.VictimShipTypeID = 587

	a := NewAccumulator()
	a.Add(km, attackers)

	row := a.Stats[StatsKey{EntityShip, 587}]
	if row == nil {
		t.Fatal("no row for the shared hull")
	}
	if row.Kills != 1 || row.Losses != 1 {
		t.Errorf("the shared hull counted %d kills and %d losses, want 1 and 1 — both "+
			"sides must land on one row", row.Kills, row.Losses)
	}

	// And the same for a corporation that killed one of its own.
	km2, attackers2 := fleetKill()
	km2.VictimCorporationID = testIDBase + 10
	a2 := NewAccumulator()
	a2.Add(km2, attackers2)

	corp := a2.Stats[StatsKey{EntityCorporation, testIDBase + 10}]
	if corp.Kills != 1 || corp.Losses != 1 {
		t.Errorf("a corporation that killed its own member counted %d kills and %d "+
			"losses, want 1 and 1", corp.Kills, corp.Losses)
	}
}

// Solo and NPC flags land on both sides.
func TestSoloAndNPCFlags(t *testing.T) {
	km, attackers := fleetKill()
	km.IsSolo, km.IsNPC = true, true
	attackers = attackers[:1]
	km.AttackerCount = 1

	a := NewAccumulator()
	a.Add(km, attackers)

	if got := a.Stats[StatsKey{EntityCharacter, testIDBase + 1}].SoloKills; got != 1 {
		t.Errorf("the solo attacker counted %d solo kills, want 1", got)
	}
	victim := a.Stats[StatsKey{EntityCharacter, km.VictimCharacterID}]
	if victim.SoloLosses != 1 {
		t.Errorf("the victim counted %d solo losses, want 1", victim.SoloLosses)
	}
	if victim.NPCLosses != 1 {
		t.Errorf("the victim counted %d NPC losses, want 1", victim.NPCLosses)
	}
}

// Locations count a kill for every killmail — a convention every location-level
// read depends on, since a kill is both a kill and a loss from the system's
// point of view.
func TestLocationsCountKills(t *testing.T) {
	km, attackers := fleetKill()
	a := NewAccumulator()
	a.Add(km, attackers)

	for _, e := range []struct {
		t  EntityType
		id int32
	}{
		{EntitySystem, km.SolarSystemID},
		{EntityConstellation, km.ConstellationID},
		{EntityRegion, km.RegionID},
	} {
		r := a.Stats[StatsKey{e.t, e.id}]
		if r == nil || r.Kills != 1 {
			t.Errorf("entity type %d id %d counted %v kills, want 1", e.t, e.id, r)
		}
		if r.Losses != 0 {
			t.Errorf("entity type %d counted %d losses — locations count kills only",
				e.t, r.Losses)
		}
	}
}

// A zero id is absent, not entity zero. Writing rows for it would accumulate a
// meaningless bucket that outgrows every real entity.
func TestZeroIDsProduceNoRows(t *testing.T) {
	km, attackers := fleetKill()
	km.VictimAllianceID = 0
	km.ConstellationID = 0
	attackers[2].AllianceID = 0

	a := NewAccumulator()
	a.Add(km, attackers)

	for key := range a.Stats {
		if key.EntityID == 0 {
			t.Errorf("a stats row was created for entity id 0 (type %d)", key.EntityType)
		}
	}
	for key := range a.Breakdowns {
		if key.EntityID == 0 || key.DimID == 0 {
			t.Errorf("a breakdown row was created with a zero id: %+v", key)
		}
	}
}

// Accumulating two killmails must sum rather than replace.
func TestAccumulatingTwoKillmailsSums(t *testing.T) {
	a := NewAccumulator()

	km1, at1 := fleetKill()
	a.Add(km1, at1)

	km2, at2 := fleetKill()
	km2.KillmailID = 137_000_002
	km2.KillmailTime = km1.KillmailTime.Add(time.Hour)
	a.Add(km2, at2)

	corp := a.Stats[StatsKey{EntityCorporation, testIDBase + 10}]
	if corp.Kills != 2 {
		t.Errorf("two killmails produced %d kills, want 2", corp.Kills)
	}
	if corp.IskDestroyed != km1.TotalValue*2 {
		t.Errorf("two killmails produced %.0f ISK, want %.0f", corp.IskDestroyed, km1.TotalValue*2)
	}

	// The last-killmail marker tracks the newer kill, which is what "last seen"
	// reads from.
	b := a.Breakdowns[BreakdownKey{EntityCharacter, testIDBase + 1, DimShipFlown, 587}]
	if b.LastKillmailID != km2.KillmailID {
		t.Errorf("last_killmail_id = %d, want the newer kill %d", b.LastKillmailID, km2.KillmailID)
	}
}

// An older killmail arriving late must not move "last seen" backwards.
func TestLastKillmailNeverMovesBackwards(t *testing.T) {
	a := NewAccumulator()

	newer, at := fleetKill()
	newer.KillmailID = 137_000_002
	newer.KillmailTime = testPeriod
	a.Add(newer, at)

	older, at2 := fleetKill()
	older.KillmailID = 137_000_001
	older.KillmailTime = newer.KillmailTime.Add(-24 * time.Hour)
	a.Add(older, at2)

	b := a.Breakdowns[BreakdownKey{EntityCharacter, testIDBase + 1, DimShipFlown, 587}]
	if b.LastKillmailID != newer.KillmailID {
		t.Errorf("last_killmail_id = %d after a late older kill, want the newer %d — "+
			"\"last seen\" moved backwards", b.LastKillmailID, newer.KillmailID)
	}
}

// --- Writing ---

// The merge adds to what is stored rather than replacing it, which is what
// makes the daily row an accumulating total.
func TestWriteAccumulates(t *testing.T) {
	pool := testPool(t)
	clearTestStats(t, pool)
	ctx := context.Background()

	km, attackers := fleetKill()
	day := km.KillmailTime

	for range 2 {
		a := NewAccumulator()
		a.Add(km, attackers)
		if _, err := Write(ctx, pool, a, day); err != nil {
			t.Fatal(err)
		}
	}

	var kills int64
	var isk float64
	if err := pool.QueryRow(ctx, `
        SELECT kills, isk_destroyed FROM stats
        WHERE entity_type = $1 AND entity_id = $2 AND period_type = $3 AND period_start = $4`,
		int16(EntityCorporation), testIDBase+10, int16(PeriodDaily), testPeriod.Format("2006-01-02")).
		Scan(&kills, &isk); err != nil {
		t.Fatal(err)
	}

	if kills != 2 {
		t.Errorf("two merges of one kill produced %d kills, want 2 — the merge is "+
			"replacing rather than accumulating", kills)
	}
	if isk != km.TotalValue*2 {
		t.Errorf("isk_destroyed = %.0f, want %.0f", isk, km.TotalValue*2)
	}
}

// The breakdown merge has to survive the same collision the stats merge does.
func TestWriteBreakdowns(t *testing.T) {
	pool := testPool(t)
	clearTestStats(t, pool)
	ctx := context.Background()

	km, attackers := fleetKill()
	a := NewAccumulator()
	a.Add(km, attackers)

	res, err := Write(ctx, pool, a, km.KillmailTime)
	if err != nil {
		t.Fatal(err)
	}
	if res.Breakdowns == 0 {
		t.Fatal("no breakdown rows were written")
	}

	var kills int64
	var lastID int64
	if err := pool.QueryRow(ctx, `
        SELECT kills, last_killmail_id FROM stats_breakdowns
        WHERE entity_type = $1 AND entity_id = $2 AND period_type = $3
          AND period_start = $4 AND dim_category = $5 AND dim_id = $6`,
		int16(EntityCharacter), testIDBase+1, int16(PeriodDaily),
		testPeriod.Format("2006-01-02"), int16(DimShipFlown), 587).
		Scan(&kills, &lastID); err != nil {
		t.Fatal(err)
	}
	if kills != 1 {
		t.Errorf("the ship-flown breakdown has %d kills, want 1", kills)
	}
	if lastID != km.KillmailID {
		t.Errorf("last_killmail_id = %d, want %d", lastID, km.KillmailID)
	}
}

// --- Derived metrics ---

func TestDerivedMetrics(t *testing.T) {
	r := Row{
		Kills: 75, Losses: 25, SoloKills: 15, SoloLosses: 5, NPCLosses: 10,
		IskDestroyed: 900, IskLost: 100,
		DamageDealt: 7500, SumAttackerCount: 300,
	}
	d := Derive(r)

	cases := []struct {
		name string
		got  float64
		want float64
	}{
		{"efficiency", float64(d.Efficiency), 75},
		{"isk_efficiency", float64(d.IskEfficiency), 90},
		{"solo_ratio", float64(d.SoloRatio), 20},
		{"solo_loss_ratio", float64(d.SoloLossRatio), 20},
		{"npc_loss_ratio", float64(d.NPCLossRatio), 40},
		{"blob_factor", d.BlobFactor, 4},
		{"avg_damage_per_kill", float64(d.AvgDamagePerKill), 100},
		{"danger_ratio", d.DangerRatio, 3},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// An entity with no activity divides by zero everywhere; every metric must come
// back as zero rather than NaN, which JSON cannot encode.
func TestDerivedMetricsOnAnEmptyRow(t *testing.T) {
	d := Derive(Row{})
	if d.Efficiency != 0 || d.BlobFactor != 0 || d.DangerRatio != 0 || d.AvgDamagePerKill != 0 {
		t.Errorf("an empty row produced %+v, want zeros", d)
	}
}

// Kills with no losses is an infinite ratio, which JSON cannot represent — so
// it is reported as -1 rather than producing a document the frontend cannot
// parse.
func TestDangerRatioWithNoLosses(t *testing.T) {
	d := Derive(Row{Kills: 10})
	if d.DangerRatio != -1 {
		t.Errorf("danger_ratio with no losses = %v, want -1 (JSON has no infinity)", d.DangerRatio)
	}
}
