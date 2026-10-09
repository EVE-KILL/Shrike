package frontend

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/eve-kill/shrike/internal/api"
)

func factionMatchupID(id int64) string {
	switch id {
	case 500001, 500004:
		return "caldari-vs-gallente"
	case 500003, 500002:
		return "amarr-vs-minmatar"
	default:
		return ""
	}
}

func (s *site) entityFactionOverviewFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || factionMatchupID(id) == "" {
		http.NotFound(w, r)
		return
	}
	data, err := s.options.Entities.EntityFactionOverview(r.Context(), id)
	if err != nil {
		http.Error(w, "Could not load warzone", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, entityFactionOverview(id, data))
}

func (s *site) entityFactionMilitiaFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || factionMatchupID(id) == "" {
		http.NotFound(w, r)
		return
	}
	corps, err := s.options.Entities.EntityFactionMilitiaCorps(r.Context(), id)
	if err != nil {
		http.Error(w, "Could not load militia corporations", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, entityFactionMilitiaCorps(corps))
}

func entityFactionMine(data api.EntityFactionOverview) api.EntityFactionSideStats {
	if data.Side == 2 {
		return data.FactionStats.Side2
	}
	return data.FactionStats.Side1
}

func entityFactionTheirs(data api.EntityFactionOverview) api.EntityFactionSideStats {
	if data.Side == 2 {
		return data.FactionStats.Side1
	}
	return data.FactionStats.Side2
}

func entityFactionZone(data api.EntityFactionOverview) api.EntityFactionZoneStats {
	if data.Side == 2 {
		return data.Warzone.Side2
	}
	return data.Warzone.Side1
}

func entityFactionHoldPercent(data api.EntityFactionOverview) int64 {
	if data.Warzone.TotalSystems < 1 {
		return 0
	}
	return 100 * entityFactionMine(data).SystemsControlled / data.Warzone.TotalSystems
}

func entityFactionHoldStyle(data api.EntityFactionOverview) string {
	return fmt.Sprintf("width:%d%%", entityFactionHoldPercent(data))
}

type factionFlipView struct {
	ID     int64
	Name   string
	At     string
	Gained bool
}

func entityFactionFlips(id int64, data api.EntityFactionOverview) []factionFlipView {
	var result []factionFlipView
	for _, day := range data.FlipDays {
		for _, flip := range day.Items {
			if flip.NewFactionID != id && flip.OldFactionID != id {
				continue
			}
			result = append(result, factionFlipView{ID: flip.SolarSystemID, Name: flip.SystemName, At: flip.FlippedAt, Gained: flip.NewFactionID == id})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].At > result[j].At })
	return result
}

func entityFactionFlipCount(flips []factionFlipView, gained bool) int {
	count := 0
	for _, flip := range flips {
		if flip.Gained == gained {
			count++
		}
	}
	return count
}

func entityFactionPilots(data api.EntityFactionOverview) []api.TopEntry {
	var rows []struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Kills int64  `json:"kills"`
	}
	if data.Side == 2 {
		rows = data.Leaderboards.Characters.Side2
	} else {
		rows = data.Leaderboards.Characters.Side1
	}
	result := make([]api.TopEntry, 0, min(len(rows), 10))
	for _, row := range rows[:min(len(rows), 10)] {
		result = append(result, api.TopEntry{ID: row.ID, Name: row.Name, Count: float64(row.Kills)})
	}
	return result
}
