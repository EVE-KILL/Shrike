package frontend

import (
	"sort"

	"github.com/eve-kill/shrike/internal/api"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

type roamCompositionPilot struct {
	CharacterID     int32  `json:"character_id"`
	Name            string `json:"name"`
	CorporationName string `json:"corporation_name"`
}
type roamCompositionRow struct {
	Pilot   roamCompositionPilot `json:"pilot"`
	Ship    *api.RoamShip        `json:"ship"`
	ISKLost float64              `json:"iskLost"`
}
type roamGroup struct {
	ID   int32
	Name string
}
type roamCompositionView struct {
	Report                    api.RoamReport
	Rows                      []roamCompositionRow
	SortedRows                []roamCompositionRow
	Groups                    []roamGroup
	ObservedPilots, HullTypes int
}

// Normalize observations once in Go. The client filters these same rows; it
// does not reconstruct losses or infer unobserved ships from the roster.
func buildRoamComposition(report api.RoamReport) roamCompositionView {
	view := roamCompositionView{Report: report, Rows: []roamCompositionRow{}, Groups: []roamGroup{}}
	type key struct{ pilot, ship int32 }
	losses := map[key]float64{}
	for _, engagement := range report.Engagements {
		for _, kill := range engagement.Killmails {
			if kill.Role == "loss" {
				losses[key{kill.VictimCharacterID, kill.VictimShipTypeID}] += kill.TotalValue
			}
		}
	}
	hulls := map[int32]bool{}
	groups := map[int32]string{}
	for _, pilot := range report.Roster {
		identity := roamCompositionPilot{pilot.CharacterID, pilot.Name, pilot.CorporationName}
		if len(pilot.Ships) == 0 {
			view.Rows = append(view.Rows, roamCompositionRow{Pilot: identity})
			continue
		}
		view.ObservedPilots++
		for _, ship := range pilot.Ships {
			view.Rows = append(view.Rows, roamCompositionRow{Pilot: identity, Ship: &ship, ISKLost: losses[key{pilot.CharacterID, ship.ShipTypeID}]})
			hulls[ship.ShipTypeID] = true
			if ship.ShipGroupID != 0 {
				groups[ship.ShipGroupID] = ship.ShipGroupName
			}
		}
	}
	view.HullTypes = len(hulls)
	for id, name := range groups {
		view.Groups = append(view.Groups, roamGroup{id, name})
	}
	names := collate.New(language.English)
	sort.SliceStable(view.Groups, func(i, j int) bool { return names.CompareString(view.Groups[i].Name, view.Groups[j].Name) < 0 })
	view.SortedRows = append([]roamCompositionRow{}, view.Rows...)
	sort.SliceStable(view.SortedRows, func(i, j int) bool {
		a, b := view.SortedRows[i], view.SortedRows[j]
		var ak, bk int
		var ad, bd int64
		if a.Ship != nil {
			ak, ad = a.Ship.Killmails, a.Ship.DamageDone
		}
		if b.Ship != nil {
			bk, bd = b.Ship.Killmails, b.Ship.DamageDone
		}
		if ak != bk {
			return ak > bk
		}
		if ad != bd {
			return ad > bd
		}
		return names.CompareString(a.Pilot.Name, b.Pilot.Name) < 0
	})
	return view
}
func roamRowShipID(row roamCompositionRow) int64 {
	if row.Ship == nil {
		return 0
	}
	return int64(row.Ship.ShipTypeID)
}
func roamRowShipName(row roamCompositionRow) string {
	if row.Ship == nil {
		return ""
	}
	return row.Ship.ShipName
}
func roamRowGroupName(row roamCompositionRow) string {
	if row.Ship == nil {
		return ""
	}
	return row.Ship.ShipGroupName
}
func roamRowObservations(row roamCompositionRow) string {
	if row.Ship == nil {
		return "—"
	}
	return roamNumber(row.Ship.Killmails)
}
func roamRowLosses(row roamCompositionRow) string {
	if row.Ship == nil || row.Ship.Losses == 0 {
		return "—"
	}
	return roamNumber(row.Ship.Losses)
}
func roamRowLost(row roamCompositionRow) string {
	if row.ISKLost == 0 {
		return "—"
	}
	return compactNumber(row.ISKLost)
}
func roamRowDamage(row roamCompositionRow) string {
	if row.Ship == nil {
		return "—"
	}
	return fullNumber(float64(row.Ship.DamageDone))
}

type roamHull struct {
	Pilot api.RoamPilot
	Ship  api.RoamShip
}
type roamHullSection struct {
	Title, Icon, Description, Tone string
	Rows                           []roamHull
}
type roamDestroyedShip struct {
	ID    int32
	Name  string
	Count int
	ISK   float64
}
type roamLossThreat struct {
	Name, Corporation string
	Losses            int
	ISK               float64
}
type roamIntelView struct {
	Report    api.RoamReport
	Sections  []roamHullSection
	Pilots    []api.RoamPilot
	Destroyed []roamDestroyedShip
	Threats   []roamLossThreat
}

func buildRoamIntel(report api.RoamReport) roamIntelView {
	view := roamIntelView{Report: report, Sections: []roamHullSection{
		{"Command hulls", "crown", "Command-class ships on matching killmails; this does not identify the FC.", "text-yellow-400", nil},
		{"Logistics", "heart-pulse", "Logistics hulls observed on matching killmails.", "text-emerald-400", nil},
		{"Capitals", "ship", "Capital hulls observed on matching killmails.", "text-amber-400", nil},
	}}
	for _, pilot := range report.Roster {
		if pilot.KillParticipations > 0 || pilot.Losses > 0 {
			view.Pilots = append(view.Pilots, pilot)
		}
		for _, ship := range pilot.Ships {
			if member(ship.ShipGroupName, "Flag Cruiser", "Command Ship", "Command Destroyer") {
				view.Sections[0].Rows = append(view.Sections[0].Rows, roamHull{pilot, ship})
			}
			if member(ship.ShipGroupName, "Logistics", "Logistics Frigate", "Force Auxiliary") {
				view.Sections[1].Rows = append(view.Sections[1].Rows, roamHull{pilot, ship})
			}
			if member(ship.ShipGroupName, "Titan", "Supercarrier", "Lancer Dreadnought", "Dreadnought", "Carrier", "Force Auxiliary") {
				view.Sections[2].Rows = append(view.Sections[2].Rows, roamHull{pilot, ship})
			}
		}
	}
	sort.SliceStable(view.Pilots, func(i, j int) bool {
		a, b := view.Pilots[i], view.Pilots[j]
		if a.KillParticipations != b.KillParticipations {
			return a.KillParticipations > b.KillParticipations
		}
		return a.DamageDone > b.DamageDone
	})
	view.Pilots = view.Pilots[:min(12, len(view.Pilots))]
	ships := map[int32]*roamDestroyedShip{}
	threats := map[[2]string]*roamLossThreat{}
	for _, engagement := range report.Engagements {
		for _, kill := range engagement.Killmails {
			if kill.Role == "kill" && kill.VictimShipTypeID > 0 {
				entry := ships[kill.VictimShipTypeID]
				if entry == nil {
					name := kill.VictimShipName
					if name == "" {
						name = "Unknown ship"
					}
					entry = &roamDestroyedShip{ID: kill.VictimShipTypeID, Name: name}
					ships[kill.VictimShipTypeID] = entry
				}
				entry.Count++
				entry.ISK += kill.TotalValue
			}
			if kill.Role == "loss" && !kill.FriendlyFire && kill.FinalBlowCharacterName != "" {
				key := [2]string{kill.FinalBlowCharacterName, kill.FinalBlowCorpName}
				entry := threats[key]
				if entry == nil {
					entry = &roamLossThreat{Name: key[0], Corporation: key[1]}
					threats[key] = entry
				}
				entry.Losses++
				entry.ISK += kill.TotalValue
			}
		}
	}
	// Preserve first-observed order when metrics tie, as the Vue Map does.
	seenShips := map[int32]bool{}
	seenThreats := map[[2]string]bool{}
	for _, engagement := range report.Engagements {
		for _, kill := range engagement.Killmails {
			if entry := ships[kill.VictimShipTypeID]; kill.Role == "kill" && entry != nil && !seenShips[entry.ID] {
				view.Destroyed = append(view.Destroyed, *entry)
				seenShips[entry.ID] = true
			}
			key := [2]string{kill.FinalBlowCharacterName, kill.FinalBlowCorpName}
			if entry := threats[key]; kill.Role == "loss" && !kill.FriendlyFire && entry != nil && !seenThreats[key] {
				view.Threats = append(view.Threats, *entry)
				seenThreats[key] = true
			}
		}
	}
	sort.SliceStable(view.Destroyed, func(i, j int) bool {
		a, b := view.Destroyed[i], view.Destroyed[j]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		return a.ISK > b.ISK
	})
	sort.SliceStable(view.Threats, func(i, j int) bool {
		a, b := view.Threats[i], view.Threats[j]
		if a.Losses != b.Losses {
			return a.Losses > b.Losses
		}
		return a.ISK > b.ISK
	})
	view.Destroyed = view.Destroyed[:min(8, len(view.Destroyed))]
	view.Threats = view.Threats[:min(8, len(view.Threats))]
	return view
}
func roamHasSpecialists(view roamIntelView) bool {
	for _, section := range view.Sections {
		if len(section.Rows) > 0 {
			return true
		}
	}
	return false
}
