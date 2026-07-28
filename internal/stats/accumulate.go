package stats

import (
	"time"
)

// Turning one killmail into counter updates.
//
// The fan-out is large and worth being explicit about: a kill with fifty
// attackers produces rows for fifty characters, their distinct corporations and
// alliances, every distinct hull flown, the victim's four identities, and three
// location levels — then the same again split across five or six breakdown
// dimensions each. A large fight can be several thousand counter updates from a
// single killmail.
//
// Two rules govern the whole thing:
//
//   - Attacker-side aggregates count once per distinct entity, not once per
//     attacker. A corporation that brought five members to a kill gets one
//     kill, not five. Characters are already distinct.
//   - Everything is accumulated into a map keyed by its primary key before any
//     SQL runs. Postgres rejects an ON CONFLICT statement that touches the same
//     row twice in one command, and that collision is routine rather than
//     exotic: the victim's corporation appearing among the attackers is an
//     internal mishap, and attacker and victim flying the same hull is common.

// Killmail is what the accumulator needs about a kill.
//
// Re-read from the database rather than carried on the job, so the job payload
// stays a single id and the handler is trivially replay-safe.
type Killmail struct {
	KillmailID      int64
	KillmailTime    time.Time
	SolarSystemID   int32
	ConstellationID int32
	RegionID        int32

	VictimCharacterID   int32
	VictimCorporationID int32
	VictimAllianceID    int32
	VictimFactionID     int32
	VictimShipTypeID    int32
	VictimDamageTaken   int64

	TotalValue    float64
	Points        int64
	AttackerCount int64
	IsNPC         bool
	IsSolo        bool
}

// Attacker is one participant.
type Attacker struct {
	CharacterID   int32
	CorporationID int32
	AllianceID    int32
	FactionID     int32
	ShipTypeID    int32
	DamageDone    int64
	FinalBlow     bool
}

// StatsKey identifies a stats row within one period.
type StatsKey struct {
	EntityType EntityType
	EntityID   int32
}

// BreakdownKey identifies a breakdown row within one period.
type BreakdownKey struct {
	EntityType  EntityType
	EntityID    int32
	DimCategory DimCategory
	DimID       int32
}

// Accumulator collects the counter updates for one or more killmails.
//
// Reusable across a batch: accumulating a day of killmails and merging once is
// dramatically cheaper than merging per kill, because the same character
// appears on many kills and collapses to a single row.
type Accumulator struct {
	Stats      map[StatsKey]*Row
	Breakdowns map[BreakdownKey]*Breakdown
}

// NewAccumulator returns an empty accumulator.
func NewAccumulator() *Accumulator {
	return &Accumulator{
		Stats:      map[StatsKey]*Row{},
		Breakdowns: map[BreakdownKey]*Breakdown{},
	}
}

func (a *Accumulator) stat(t EntityType, id int32) *Row {
	k := StatsKey{t, id}
	r, ok := a.Stats[k]
	if !ok {
		r = &Row{}
		a.Stats[k] = r
	}
	return r
}

func (a *Accumulator) breakdown(t EntityType, id int32, cat DimCategory, dim int32) *Breakdown {
	k := BreakdownKey{t, id, cat, dim}
	b, ok := a.Breakdowns[k]
	if !ok {
		b = &Breakdown{}
		a.Breakdowns[k] = b
	}
	return b
}

// Add folds one killmail into the accumulator.
func (a *Accumulator) Add(km Killmail, attackers []Attacker) {
	v := km.TotalValue
	kmTime := km.KillmailTime.Unix()

	// Distinct attacker entities. The maps are what enforce "once per entity" —
	// a corporation with five members on the mail is one entry here.
	charDamage := map[int32]int64{}
	charShip := map[int32]int32{}
	charCorp := map[int32]int32{}
	charAlliance := map[int32]int32{}
	corps := map[int32]bool{}
	alliances := map[int32]bool{}
	factions := map[int32]bool{}
	shipTypes := map[int32]bool{}

	var fbChar, fbCorp, fbAlliance, fbFaction int32

	for _, at := range attackers {
		if at.CharacterID != 0 {
			charDamage[at.CharacterID] += at.DamageDone
			if at.ShipTypeID != 0 {
				charShip[at.CharacterID] = at.ShipTypeID
			}
			charCorp[at.CharacterID] = at.CorporationID
			charAlliance[at.CharacterID] = at.AllianceID
		}
		if at.CorporationID != 0 {
			corps[at.CorporationID] = true
		}
		if at.AllianceID != 0 {
			alliances[at.AllianceID] = true
		}
		if at.FactionID != 0 {
			factions[at.FactionID] = true
		}
		if at.ShipTypeID != 0 {
			shipTypes[at.ShipTypeID] = true
		}
		if at.FinalBlow {
			fbChar, fbCorp, fbAlliance, fbFaction = at.CharacterID, at.CorporationID,
				at.AllianceID, at.FactionID
		}
	}

	// --- Attacker-side headline counters ---

	for charID, damage := range charDamage {
		r := a.stat(EntityCharacter, charID)
		r.Kills++
		r.IskDestroyed += v
		r.Points += km.Points
		r.DamageDealt += damage
		// The gang size on every kill, summed — divided by kills it gives the
		// blob factor, which is why it accumulates rather than being averaged
		// here.
		r.SumAttackerCount += km.AttackerCount
		if km.IsSolo {
			r.SoloKills++
		}
		if charID == fbChar {
			r.FinalBlows++
		}
	}

	for corpID := range corps {
		r := a.stat(EntityCorporation, corpID)
		r.Kills++
		r.IskDestroyed += v
		r.Points += km.Points
		if km.IsSolo {
			r.SoloKills++
		}
		if corpID == fbCorp {
			r.FinalBlows++
		}
	}

	for allyID := range alliances {
		r := a.stat(EntityAlliance, allyID)
		r.Kills++
		r.IskDestroyed += v
		r.Points += km.Points
		if km.IsSolo {
			r.SoloKills++
		}
		if allyID == fbAlliance {
			r.FinalBlows++
		}
	}

	for factionID := range factions {
		r := a.stat(EntityFaction, factionID)
		r.Kills++
		r.IskDestroyed += v
		r.Points += km.Points
		if km.IsSolo {
			r.SoloKills++
		}
		if factionID == fbFaction {
			r.FinalBlows++
		}
	}

	// One kill per distinct hull brought, so a fleet of twenty of the same ship
	// credits that hull once.
	for shipID := range shipTypes {
		r := a.stat(EntityShip, shipID)
		r.Kills++
		r.IskDestroyed += v
	}

	// --- Victim-side headline counters ---

	if id := km.VictimCharacterID; id != 0 {
		r := a.stat(EntityCharacter, id)
		r.Losses++
		r.IskLost += v
		r.DamageTaken += km.VictimDamageTaken
		if km.IsSolo {
			r.SoloLosses++
		}
		if km.IsNPC {
			r.NPCLosses++
		}
	}
	for _, e := range []struct {
		t  EntityType
		id int32
	}{
		{EntityCorporation, km.VictimCorporationID},
		{EntityAlliance, km.VictimAllianceID},
		{EntityFaction, km.VictimFactionID},
	} {
		if e.id == 0 {
			continue
		}
		r := a.stat(e.t, e.id)
		r.Losses++
		r.IskLost += v
		if km.IsSolo {
			r.SoloLosses++
		}
		if km.IsNPC {
			r.NPCLosses++
		}
	}
	if id := km.VictimShipTypeID; id != 0 {
		r := a.stat(EntityShip, id)
		r.Losses++
		r.IskLost += v
	}

	// --- Locations ---
	//
	// Every kill is both a kill and a loss from a location's point of view, so
	// one of the two has to be picked as the convention. Kills is the choice,
	// and every location-level read depends on it being consistent.

	if id := km.SolarSystemID; id != 0 {
		r := a.stat(EntitySystem, id)
		r.Kills++
		r.IskDestroyed += v
		if km.IsNPC {
			r.NPCLosses++
		}
	}
	for _, e := range []struct {
		t  EntityType
		id int32
	}{
		{EntityConstellation, km.ConstellationID},
		{EntityRegion, km.RegionID},
	} {
		if e.id == 0 {
			continue
		}
		r := a.stat(e.t, e.id)
		r.Kills++
		r.IskDestroyed += v
	}

	// --- Attacker-side breakdowns ---

	killDim := func(t EntityType, id int32, cat DimCategory, dim int32) {
		if id == 0 || dim == 0 {
			return
		}
		b := a.breakdown(t, id, cat, dim)
		b.Kills++
		b.IskDestroyed += v
		if markerAfter(kmTime, km.KillmailID, b.LastKillmailTime, b.LastKillmailID) {
			b.LastKillmailTime, b.LastKillmailID = kmTime, km.KillmailID
		}
	}

	for charID := range charDamage {
		killDim(EntityCharacter, charID, DimShipFlown, charShip[charID])
		killDim(EntityCharacter, charID, DimSystem, km.SolarSystemID)
		killDim(EntityCharacter, charID, DimConstellation, km.ConstellationID)
		killDim(EntityCharacter, charID, DimRegion, km.RegionID)
		killDim(EntityCharacter, charID, DimKilledCorporation, km.VictimCorporationID)
		killDim(EntityCharacter, charID, DimKilledAlliance, km.VictimAllianceID)
	}

	// The organisation-level breakdowns mirror the character ones. SHIP_FLOWN
	// is one contribution per distinct hull the organisation brought to this
	// killmail, matching the authoritative TS catchup/backfill SQL. The TS live
	// writer accidentally emits one contribution per member and relies on its
	// merge to sum them, which means a catchup later changes the same daily row.
	// Deduplicating here keeps live ingest and rebuilds convergent.
	for corpID := range corps {
		killDim(EntityCorporation, corpID, DimSystem, km.SolarSystemID)
		killDim(EntityCorporation, corpID, DimConstellation, km.ConstellationID)
		killDim(EntityCorporation, corpID, DimRegion, km.RegionID)
		killDim(EntityCorporation, corpID, DimKilledCorporation, km.VictimCorporationID)
		killDim(EntityCorporation, corpID, DimKilledAlliance, km.VictimAllianceID)
		seenShips := map[int32]bool{}
		for charID, shipID := range charShip {
			if charCorp[charID] == corpID && !seenShips[shipID] {
				seenShips[shipID] = true
				killDim(EntityCorporation, corpID, DimShipFlown, shipID)
			}
		}
	}

	for allyID := range alliances {
		killDim(EntityAlliance, allyID, DimSystem, km.SolarSystemID)
		killDim(EntityAlliance, allyID, DimConstellation, km.ConstellationID)
		killDim(EntityAlliance, allyID, DimRegion, km.RegionID)
		killDim(EntityAlliance, allyID, DimKilledCorporation, km.VictimCorporationID)
		killDim(EntityAlliance, allyID, DimKilledAlliance, km.VictimAllianceID)
		seenShips := map[int32]bool{}
		for charID, shipID := range charShip {
			if charAlliance[charID] == allyID && !seenShips[shipID] {
				seenShips[shipID] = true
				killDim(EntityAlliance, allyID, DimShipFlown, shipID)
			}
		}
	}

	// --- Victim-side breakdowns ---

	lossDim := func(t EntityType, id int32, cat DimCategory, dim int32) {
		if id == 0 || dim == 0 {
			return
		}
		b := a.breakdown(t, id, cat, dim)
		b.Losses++
		b.IskLost += v
		if markerAfter(kmTime, km.KillmailID, b.LastKillmailTime, b.LastKillmailID) {
			b.LastKillmailTime, b.LastKillmailID = kmTime, km.KillmailID
		}
	}

	for _, e := range []struct {
		t  EntityType
		id int32
	}{
		{EntityCharacter, km.VictimCharacterID},
		{EntityCorporation, km.VictimCorporationID},
		{EntityAlliance, km.VictimAllianceID},
	} {
		if e.id == 0 {
			continue
		}
		lossDim(e.t, e.id, DimShipLost, km.VictimShipTypeID)
		lossDim(e.t, e.id, DimSystem, km.SolarSystemID)
		lossDim(e.t, e.id, DimConstellation, km.ConstellationID)
		lossDim(e.t, e.id, DimRegion, km.RegionID)
		for corpID := range corps {
			lossDim(e.t, e.id, DimDiesToCorporation, corpID)
		}
		for allyID := range alliances {
			lossDim(e.t, e.id, DimDiesToAlliance, allyID)
		}
	}
}

// AddFactions folds only faction headline counters into the accumulator.
//
// A faction-only backfill deliberately avoids constructing every other entity
// and breakdown row just to discard them. Keep this in lockstep with the
// faction portions of Add: one attacker-side kill per distinct faction on the
// mail, plus one victim-side loss.
func (a *Accumulator) AddFactions(km Killmail, attackers []Attacker) {
	factions := make(map[int32]struct{})
	var finalBlowFactionID int32
	for _, attacker := range attackers {
		if attacker.FactionID != 0 {
			factions[attacker.FactionID] = struct{}{}
		}
		if attacker.FinalBlow {
			finalBlowFactionID = attacker.FactionID
		}
	}

	for factionID := range factions {
		row := a.stat(EntityFaction, factionID)
		row.Kills++
		row.IskDestroyed += km.TotalValue
		row.Points += km.Points
		if km.IsSolo {
			row.SoloKills++
		}
		if factionID == finalBlowFactionID {
			row.FinalBlows++
		}
	}

	if km.VictimFactionID == 0 {
		return
	}
	row := a.stat(EntityFaction, km.VictimFactionID)
	row.Losses++
	row.IskLost += km.TotalValue
	if km.IsSolo {
		row.SoloLosses++
	}
	if km.IsNPC {
		row.NPCLosses++
	}
}

// KeepEntityTypes removes aggregate rows outside an explicitly selected set.
// A nil/empty selection means all entity types, preserving the normal live and
// all-entity backfill behavior.
func (a *Accumulator) KeepEntityTypes(entityTypes []EntityType) {
	if len(entityTypes) == 0 {
		return
	}
	keep := make(map[EntityType]struct{}, len(entityTypes))
	for _, entityType := range entityTypes {
		keep[entityType] = struct{}{}
	}
	for key := range a.Stats {
		if _, ok := keep[key.EntityType]; !ok {
			delete(a.Stats, key)
		}
	}
	for key := range a.Breakdowns {
		if _, ok := keep[key.EntityType]; !ok {
			delete(a.Breakdowns, key)
		}
	}
}

// Empty reports whether anything was accumulated.
func (a *Accumulator) Empty() bool {
	return len(a.Stats) == 0 && len(a.Breakdowns) == 0
}
