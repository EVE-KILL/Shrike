package frontend

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/api"
)

func (s *site) entityBattlesFragment(w http.ResponseWriter, r *http.Request) {
	kind, id, page, ok := entityConflictFragmentParams(w, r)
	if !ok {
		return
	}
	data, err := s.options.Entities.EntityBattles(r.Context(), kind, id, page)
	if err != nil {
		http.Error(w, "Could not load battles", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, entityBattles(kind, id, data))
}

func (s *site) entityWarsFragment(w http.ResponseWriter, r *http.Request) {
	kind, id, page, ok := entityConflictFragmentParams(w, r)
	if !ok {
		return
	}
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "recent"
	}
	if sortBy != "recent" && sortBy != "kills" && sortBy != "isk" {
		http.Error(w, "Invalid war sort", http.StatusBadRequest)
		return
	}
	data, err := s.options.Entities.EntityWars(r.Context(), kind, id, page, sortBy)
	if err != nil {
		http.Error(w, "Could not load wars", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, entityWars(kind, id, data, sortBy))
}

func (s *site) entityCampaignsFragment(w http.ResponseWriter, r *http.Request) {
	kind, id, page, ok := entityConflictFragmentParams(w, r)
	if !ok {
		return
	}
	data, err := s.options.Entities.EntityCampaigns(r.Context(), kind, id, page)
	if err != nil {
		http.Error(w, "Could not load campaigns", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, entityCampaigns(kind, id, data))
}

func entityConflictFragmentParams(w http.ResponseWriter, r *http.Request) (string, int64, int, bool) {
	kind := r.PathValue("kind")
	if kind != "character" && kind != "corporation" && kind != "alliance" {
		http.NotFound(w, r)
		return "", 0, 0, false
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return "", 0, 0, false
	}
	page, err := fragmentInt(r.URL.Query().Get("page"), 1, 1, 1_000_000)
	if err != nil {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return "", 0, 0, false
	}
	return kind, id, page, true
}

func entityBattleTeamTitle(battle api.EntityBattleEntry, index int) string {
	if index >= len(battle.Teams) {
		return ""
	}
	team := battle.Teams[index]
	if len(team.TopAlliances) == 0 {
		return fmt.Sprintf("Team %d", index+1)
	}
	entry := team.TopAlliances[0]
	if entry.AllianceName != "" {
		return entry.AllianceName
	}
	if entry.CorporationName != "" {
		return entry.CorporationName
	}
	return fmt.Sprintf("Team %d", index+1)
}

func entityBattleDuration(minutes int64) string {
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%dh", minutes/60)
	}
	return fmt.Sprintf("%dh %dm", minutes/60, minutes%60)
}

func entityBattleShare(battle api.EntityBattleEntry) string {
	if len(battle.Teams) != 2 {
		return "width:50%"
	}
	left, right := battle.Teams[0].TotalISKDestroyed, battle.Teams[1].TotalISKDestroyed
	if left+right <= 0 {
		return "width:50%"
	}
	return fmt.Sprintf("width:%.0f%%", 100*left/(left+right))
}

func entityBattleSecurity(value float64) string {
	if value >= 0.5 {
		return "sec-green"
	}
	if value > 0 {
		return "sec-yellow"
	}
	return "sec-red"
}

func entityWarImage(side api.EntityWarSide) string {
	return eveEntityImage(side.Type, side.ID, 64)
}

func campaignIsArea(campaign api.EntityCampaignEntry) bool {
	return campaign.Mode == "area" || len(campaign.Sides) == 0
}

func campaignSideColors(campaign api.EntityCampaignEntry) []string {
	if accents := campaignSidePaletteAccents(campaign); accents != nil {
		return accents
	}
	fixed := []string{"#ef4444", "#3b82f6", "#059669", "#ea580c"}
	fallback := make([]string, len(campaign.Sides))
	for index, side := range campaign.Sides {
		if int(side.SideIndex) < len(fixed) && side.SideIndex >= 0 {
			fallback[index] = fixed[side.SideIndex]
		} else {
			fallback[index] = "#6b7280"
		}
	}
	return fallback
}

func campaignSidePaletteAccents(campaign api.EntityCampaignEntry) []string {
	accents := make([]string, len(campaign.Sides))
	hues := make([]float64, len(campaign.Sides))
	valid := len(campaign.Sides) > 0
	for index, side := range campaign.Sides {
		accent := paletteAccent(side.Palette)
		if accent == "" {
			valid = false
			continue
		}
		accents[index] = accent
		rgb, _ := parsePaletteColor(accent)
		hues[index] = rgbToOKLCH(rgb).h * 180 / math.Pi
	}
	if valid {
		for i := range hues {
			for j := i + 1; j < len(hues); j++ {
				distance := math.Abs(hues[i] - hues[j])
				if math.Min(distance, 360-distance) < 60 {
					valid = false
				}
			}
		}
	}
	if valid {
		return accents
	}
	return nil
}

func campaignRailStyle(campaign api.EntityCampaignEntry) string {
	if campaignIsArea(campaign) {
		return "background-color:#f59e0b"
	}
	colors := campaignSideColors(campaign)
	if len(campaign.Sides) == 1 {
		return "background-color:" + colors[0]
	}
	var stops []string
	for index, color := range colors {
		stops = append(stops, fmt.Sprintf("%s %.2f%%, %s %.2f%%", color, 100*float64(index)/float64(len(campaign.Sides)), color, 100*float64(index+1)/float64(len(campaign.Sides))))
	}
	return "background-image:linear-gradient(to bottom," + strings.Join(stops, ",") + ")"
}

func campaignSideStyle(campaign api.EntityCampaignEntry, index int) string {
	return "color:" + campaignSideColors(campaign)[index]
}

func campaignSideBarStyle(campaign api.EntityCampaignEntry, side api.EntityCampaignSide, index int) string {
	var total float64
	for _, candidate := range campaign.Sides {
		total += candidate.ISKDestroyed
	}
	share := 100 / float64(max(len(campaign.Sides), 1))
	if total > 0 {
		share = 100 * side.ISKDestroyed / total
	}
	return fmt.Sprintf("width:%.2f%%;background-color:%s", share, campaignSideColors(campaign)[index])
}

func campaignOneSidedShare(side api.EntityCampaignSide) float64 {
	total := side.ISKDestroyed + side.ISKLost
	if total <= 0 {
		return 0
	}
	return 100 * side.ISKDestroyed / total
}

func campaignEntityImage(entityType int, id int64) string {
	switch entityType {
	case 2:
		return eveEntityImage("alliance", id, 64)
	case 1:
		return eveEntityImage("corporation", id, 64)
	default:
		return eveEntityImage("character", id, 64)
	}
}

func campaignDateRange(campaign api.EntityCampaignEntry) string {
	end := "ongoing"
	if campaign.EndTime != "" {
		end = entityDate(campaign.EndTime)
	}
	return entityDate(campaign.StartTime) + " — " + end
}

func campaignAreaLabel(campaign api.EntityCampaignEntry) string {
	var pieces []string
	for _, item := range []struct {
		count int
		name  string
	}{{len(campaign.Location.SystemIDs), "system"}, {len(campaign.Location.ConstellationIDs), "constellation"}, {len(campaign.Location.RegionIDs), "region"}} {
		if item.count > 0 {
			name := item.name
			if item.count > 1 {
				name += "s"
			}
			pieces = append(pieces, fmt.Sprintf("%d %s", item.count, name))
		}
	}
	if len(pieces) == 0 {
		return "Scope pending"
	}
	return strings.Join(pieces, " · ")
}
