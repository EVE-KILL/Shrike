package frontend

import (
	"fmt"
	"sort"
	"strconv"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

type battlePilot struct {
	CharacterID   int64   `json:"character_id"`
	CharacterName string  `json:"character_name"`
	CorpID        int64   `json:"corporation_id"`
	CorpName      string  `json:"corporation_name"`
	AllianceID    int64   `json:"alliance_id"`
	AllianceName  string  `json:"alliance_name"`
	ShipID        int64   `json:"ship_type_id"`
	ShipName      string  `json:"ship_name"`
	GroupID       int64   `json:"ship_group_id"`
	GroupName     string  `json:"ship_group_name"`
	Lost          float64 `json:"isk_lost"`
	Damage        float64 `json:"damage_done"`
	Taken         float64 `json:"damage_taken"`
	Deaths        int64   `json:"deaths"`
	Rank          int     `json:"rank"`
	Died          bool    `json:"died"`
	Confirmed     bool    `json:"confirmed"`
}
type battleShipAggregate struct {
	Key     string  `json:"key"`
	Name    string  `json:"name"`
	ShipID  int64   `json:"ship_type_id"`
	GroupID int64   `json:"ship_group_id"`
	Count   int64   `json:"count"`
	Losses  int64   `json:"losses"`
	Lost    float64 `json:"isk_lost"`
	Damage  float64 `json:"damage_done"`
	Taken   float64 `json:"damage_taken"`
	Rank    int     `json:"rank"`
}
type battleCompositionTeam struct {
	Index       int                   `json:"team_index"`
	Individuals []battlePilot         `json:"individuals"`
	Ships       []battleShipAggregate `json:"by_ship"`
	Groups      []battleShipAggregate `json:"by_group"`
}
type battleCompositionData struct {
	Teams []battleCompositionTeam `json:"teams"`
}
type battleIntelTeam struct {
	Index     int           `json:"team_index"`
	FCs       []battlePilot `json:"fcs"`
	Logistics []battlePilot `json:"logistics"`
	Capitals  []battlePilot `json:"capitals"`
}
type battleIntelData struct {
	Teams []battleIntelTeam `json:"teams"`
}
type battleCompositionStats struct {
	Pilots, Hulls, Losses int64
	ISK                   float64
}

func battleCompositionTeamStats(t battleCompositionTeam) battleCompositionStats {
	ids := map[int64]bool{}
	s := battleCompositionStats{Hulls: int64(len(t.Ships))}
	for _, p := range t.Individuals {
		if p.CharacterID > 0 {
			ids[p.CharacterID] = true
		}
	}
	s.Pilots = int64(len(ids))
	for _, ship := range t.Ships {
		s.Losses += ship.Losses
		s.ISK += ship.Lost
	}
	return s
}
func battleCompositionGroups(data battleCompositionData) []battleShipGroup {
	groups := map[int64]string{}
	for _, t := range data.Teams {
		for _, g := range t.Groups {
			id := g.GroupID
			if id == 0 {
				id, _ = strconv.ParseInt(g.Key, 10, 64)
			}
			groups[id] = g.Name
		}
	}
	result := []battleShipGroup{}
	for id, name := range groups {
		result = append(result, battleShipGroup{id, name})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
func battleCompositionInitialRows(t battleCompositionTeam, group *int64) []battlePilot {
	names := collate.New(language.English)
	rows := []battlePilot{}
	for _, p := range t.Individuals {
		if group == nil || *group == 0 || *group == p.GroupID {
			rows = append(rows, p)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Rank != b.Rank {
			return a.Rank > b.Rank
		}
		if a.Lost != b.Lost {
			return a.Lost > b.Lost
		}
		if a.Damage != b.Damage {
			return a.Damage > b.Damage
		}
		return names.CompareString(a.CharacterName, b.CharacterName) < 0
	})
	return rows
}
func battlePilotLogo(p battlePilot) string {
	if p.AllianceID > 0 {
		return eveEntityImage("alliance", p.AllianceID, 32)
	}
	if p.CorpID > 0 {
		return eveEntityImage("corporation", p.CorpID, 32)
	}
	return ""
}
func battleTeamBorder(index int) string {
	if index == 0 {
		return "border-red-500/20"
	}
	return "border-blue-500/20"
}
func battleTeamAccent(index int) string {
	if index == 0 {
		return "text-red-400"
	}
	return "text-blue-400"
}

type battleCapitalGroup struct {
	ID          int64
	Name        string
	Count, Died int
}
type battleIntelStats struct {
	Confirmed, Possible, LogiSurvived, LogiDied, CapDied int
	Groups                                               []battleCapitalGroup
}

func battleIntelTeamStats(t battleIntelTeam) battleIntelStats {
	s := battleIntelStats{}
	for _, p := range t.FCs {
		if p.Confirmed {
			s.Confirmed++
		} else {
			s.Possible++
		}
	}
	for _, p := range t.Logistics {
		if p.Died {
			s.LogiDied++
		} else {
			s.LogiSurvived++
		}
	}
	groups := map[string]battleCapitalGroup{}
	for _, p := range t.Capitals {
		name := battleUnknown(p.GroupName)
		g := groups[name]
		g.ID = p.GroupID
		g.Name = name
		g.Count++
		if p.Died {
			g.Died++
			s.CapDied++
		}
		groups[name] = g
	}
	for _, g := range groups {
		s.Groups = append(s.Groups, g)
	}
	order := map[string]int{"Titan": 6, "Supercarrier": 5, "Lancer Dreadnought": 4, "Dreadnought": 3, "Carrier": 2, "Force Auxiliary": 1}
	sort.SliceStable(s.Groups, func(i, j int) bool {
		a, b := s.Groups[i], s.Groups[j]
		if order[a.Name] != order[b.Name] {
			return order[a.Name] > order[b.Name]
		}
		return a.Name < b.Name
	})
	return s
}
func battleLossClassLink(v battleDetailView, group int64, side int) string {
	return v.path("kills", map[string][]string{"group": {strconv.FormatInt(group, 10)}, "side": {strconv.Itoa(side)}, "from": nil, "to": nil})
}
func battleShipRender(id int64) string {
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("/images/types/%d/render?size=32", id)
}
func battleCompositionTiles(t battleCompositionTeam) []battleTile {
	s := battleCompositionTeamStats(t)
	return []battleTile{{Label: "Observed pilots", Value: fullNumber(float64(s.Pilots)), Color: "text-gray-200"}, {Label: "Hull types", Value: fullNumber(float64(s.Hulls)), Color: "text-gray-200"}, {Label: "Recorded losses", Value: fullNumber(float64(s.Losses)), Color: "text-gray-200"}, {Label: "ISK lost", Value: compactNumber(s.ISK), Color: "text-amber-200"}}
}
