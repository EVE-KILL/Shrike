package frontend

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type warSideIDs struct {
	aggressorCorps, aggressorAlliances map[int64]bool
	defenderCorps, defenderAlliances   map[int64]bool
}

type warDetailView struct {
	ID    int64
	Data  api.WarDetailData
	Kills api.KilllistPage
	Sides warSideIDs
}

func warSides(war api.WarDetail) warSideIDs {
	sides := warSideIDs{map[int64]bool{}, map[int64]bool{}, map[int64]bool{}, map[int64]bool{}}
	add := func(kind string, id int64, defender bool) {
		if id <= 0 {
			return
		}
		if defender {
			if kind == "alliance" {
				sides.defenderAlliances[id] = true
			} else {
				sides.defenderCorps[id] = true
			}
		} else {
			if kind == "alliance" {
				sides.aggressorAlliances[id] = true
			} else {
				sides.aggressorCorps[id] = true
			}
		}
	}
	add(war.Aggressor.Type, war.Aggressor.ID, false)
	add(war.Defender.Type, war.Defender.ID, true)
	for _, ally := range war.Allies {
		add(ally.Type, ally.ID, true)
	}
	return sides
}

func warKillRowClass(kill api.KilllistRow, index int, sides *warSideIDs) string {
	base := killRowClass(index)
	if sides == nil {
		return base
	}
	if sides.aggressorAlliances[kill.VictimAllianceID] || sides.aggressorCorps[kill.VictimCorporationID] {
		return base + " war-victim-aggressor"
	}
	if sides.defenderAlliances[kill.VictimAllianceID] || sides.defenderCorps[kill.VictimCorporationID] {
		return base + " war-victim-defender"
	}
	return base
}

func warIDFromPath(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id < 1 {
		return 0, ErrNotFound
	}
	return id, nil
}

func warIDsCSV(values map[int64]bool) string {
	ids := make([]int64, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	return strings.Join(parts, ",")
}

func warKillQuery(data api.WarDetailData, sides warSideIDs, side string, page, limit int) url.Values {
	query := url.Values{"page": {strconv.Itoa(page)}, "limit": {strconv.Itoa(limit)}}
	start := data.War.Started
	if start == "" {
		start = data.War.Declared
	}
	if start != "" {
		query.Set("warStart", start)
	}
	if data.War.Finished != "" {
		query.Set("warEnd", data.War.Finished)
	}
	switch side {
	case "aggressor":
		if corps := warIDsCSV(sides.defenderCorps); corps != "" {
			query.Set("warSideCorps", corps)
		}
		if alliances := warIDsCSV(sides.defenderAlliances); alliances != "" {
			query.Set("warSideAlliances", alliances)
		}
	case "defender":
		if corps := warIDsCSV(sides.aggressorCorps); corps != "" {
			query.Set("warSideCorps", corps)
		}
		if alliances := warIDsCSV(sides.aggressorAlliances); alliances != "" {
			query.Set("warSideAlliances", alliances)
		}
	}
	return query
}

func warTitle(war api.WarDetail) string {
	return war.Aggressor.Name + " vs " + war.Defender.Name + " — War #" + warID(war.WarID)
}

func warMetadata(data api.WarDetailData) Metadata {
	war := data.War
	status := "Active"
	if war.Finished != "" {
		status = "Finished"
	}
	return Metadata{
		Title: warTitle(war), OGTitle: warTitle(war), CanonicalPath: "/war/" + warID(war.WarID), Indexable: true,
		Description: fmt.Sprintf("War #%d — %s vs %s. %s. %d ships destroyed, %s ISK destroyed — EVE-KILL.", war.WarID, war.Aggressor.Name, war.Defender.Name, status, data.Stats.TotalKills, warISK(data.Stats.TotalValue)),
		Image:       eveEntityImage(war.Aggressor.Type, war.Aggressor.ID, 256),
		Stylesheets: []string{"war-detail.css"}, ModuleScripts: []string{"war-detail.js"},
		StructuredData: warStructuredData(war),
	}
}

func warStructuredData(war api.WarDetail) []any {
	url := "https://eve-kill.com/war/" + warID(war.WarID)
	event := map[string]any{"@context": "https://schema.org", "@type": "Event", "name": warTitle(war), "url": url,
		"eventAttendanceMode": "https://schema.org/OnlineEventAttendanceMode",
		"location":            map[string]any{"@type": "VirtualLocation", "url": "https://www.eveonline.com"},
		"organizer":           map[string]any{"@type": "Organization", "name": war.Aggressor.Name, "url": "https://eve-kill.com" + warPartyLink(war.Aggressor.Type, war.Aggressor.ID)},
		"attendee":            []any{map[string]any{"@type": "Organization", "name": war.Aggressor.Name, "url": "https://eve-kill.com" + warPartyLink(war.Aggressor.Type, war.Aggressor.ID)}, map[string]any{"@type": "Organization", "name": war.Defender.Name, "url": "https://eve-kill.com" + warPartyLink(war.Defender.Type, war.Defender.ID)}},
	}
	if war.Started != "" {
		event["startDate"] = war.Started
	}
	if war.Finished != "" {
		event["endDate"] = war.Finished
		event["eventStatus"] = "https://schema.org/EventCancelled"
	} else {
		event["eventStatus"] = "https://schema.org/EventScheduled"
	}
	return []any{map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": []any{
		map[string]any{"@type": "ListItem", "position": 1, "name": "Home", "item": "https://eve-kill.com/"},
		map[string]any{"@type": "ListItem", "position": 2, "name": "Wars", "item": "https://eve-kill.com/wars"},
		map[string]any{"@type": "ListItem", "position": 3, "name": "War #" + warID(war.WarID), "item": url},
	}}, event}
}

func (s *site) loadWarDetailPage(r *http.Request) (Page, error) {
	id, err := warIDFromPath(r)
	if err != nil {
		return Page{}, err
	}
	data, found, err := s.options.Wars.WarDetail(r.Context(), id)
	if err != nil {
		return Page{}, err
	}
	if !found {
		return Page{}, ErrNotFound
	}
	sides := warSides(data.War)
	kills, err := s.options.Wars.WarKilllist(r.Context(), id, warKillQuery(data, sides, "combined", 1, 50))
	if err != nil {
		return Page{}, err
	}
	return Page{Metadata: warMetadata(data), Content: warDetailContent(warDetailView{ID: id, Data: data, Kills: kills, Sides: sides}), SharedCacheTTL: 2 * time.Minute}, nil
}

func (s *site) warFragmentDetail(w http.ResponseWriter, r *http.Request) (int64, api.WarDetailData, warSideIDs, bool) {
	id, err := warIDFromPath(r)
	if err != nil {
		http.NotFound(w, r)
		return 0, api.WarDetailData{}, warSideIDs{}, false
	}
	data, found, err := s.options.Wars.WarDetail(r.Context(), id)
	if err != nil {
		http.Error(w, "Could not load war", http.StatusServiceUnavailable)
		return 0, api.WarDetailData{}, warSideIDs{}, false
	}
	if !found {
		http.NotFound(w, r)
		return 0, api.WarDetailData{}, warSideIDs{}, false
	}
	return id, data, warSides(data.War), true
}

func (s *site) warKillsFragment(w http.ResponseWriter, r *http.Request) {
	id, data, sides, ok := s.warFragmentDetail(w, r)
	if !ok {
		return
	}
	side := r.URL.Query().Get("side")
	if side != "combined" && side != "aggressor" && side != "defender" {
		http.Error(w, "Invalid side", http.StatusBadRequest)
		return
	}
	page, err := fragmentInt(r.URL.Query().Get("page"), 1, 1, 10000)
	if err != nil {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	limit, err := fragmentInt(r.URL.Query().Get("limit"), 50, 10, 100)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}
	kills, err := s.options.Wars.WarKilllist(r.Context(), id, warKillQuery(data, sides, side, page, limit))
	if err != nil {
		http.Error(w, "Could not load war kills", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, killList(kills, killListOptions{Kind: "war", Page: page, Limit: limit, WarSides: &sides}))
}

func (s *site) warStatsFragment(w http.ResponseWriter, r *http.Request) {
	id, _, _, ok := s.warFragmentDetail(w, r)
	if !ok {
		return
	}
	side := r.URL.Query().Get("side")
	if side != "combined" && side != "aggressor" && side != "defender" {
		http.Error(w, "Invalid side", http.StatusBadRequest)
		return
	}
	data, err := s.options.Wars.WarStats(r.Context(), id)
	if err != nil {
		http.Error(w, "Could not load war stats", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, warSidebar(data, side))
}

func (s *site) warMembersFragment(w http.ResponseWriter, r *http.Request) {
	id, _, _, ok := s.warFragmentDetail(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	side, sort := query.Get("side"), query.Get("sort")
	if side != "combined" && side != "aggressor" && side != "defender" {
		http.Error(w, "Invalid side", http.StatusBadRequest)
		return
	}
	if sort != "activity" && sort != "kills" && sort != "losses" && sort != "isk" {
		http.Error(w, "Invalid sort", http.StatusBadRequest)
		return
	}
	apiQuery := url.Values{"side": {side}, "sort": {sort}}
	for _, kind := range []string{"corporationId", "allianceId"} {
		if raw := query.Get(kind); raw != "" {
			id, err := strconv.ParseInt(raw, 10, 32)
			if err != nil || id < 1 {
				http.Error(w, "Invalid entity", http.StatusBadRequest)
				return
			}
			apiQuery.Set(kind, raw)
		}
	}
	data, err := s.options.Wars.WarMembers(r.Context(), id, apiQuery)
	if err != nil {
		http.Error(w, "Could not load war members", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, warMembersResults(data))
}

func (s *site) warIntelFragment(w http.ResponseWriter, r *http.Request) {
	id, _, _, ok := s.warFragmentDetail(w, r)
	if !ok {
		return
	}
	data, err := s.options.Wars.WarIntel(r.Context(), id)
	if err != nil {
		http.Error(w, "Could not load war intel", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, warIntelContent(data))
}

func warDate(raw string) string {
	when, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return ""
	}
	return when.UTC().Format("Jan 2, 2006")
}

func warISKShare(war api.WarDetail) int {
	total := war.Aggressor.ISKDestroyed + war.Defender.ISKDestroyed
	if total <= 0 {
		return 50
	}
	return int(war.Aggressor.ISKDestroyed/total*100 + 0.5)
}

func warPartyLink(kind string, id int64) string {
	if kind != "alliance" {
		kind = "corporation"
	}
	return idPath(kind, id)
}
func warPartyImage(kind string, id int64, size int) string {
	if kind != "alliance" {
		kind = "corporation"
	}
	return eveEntityImage(kind, id, size)
}
func warSideName(side string) string {
	if side == "aggressor" {
		return "Aggressor"
	}
	if side == "defender" {
		return "Defender"
	}
	return "Both Sides"
}

func warLeaderboardSide(selected, entry string) string {
	if selected == "aggressor" || selected == "defender" {
		return selected
	}
	if entry == "aggressor" || entry == "defender" {
		return entry
	}
	return "combined"
}
func warLeaderboardImage(kind string, id int64) string { return eveEntityImage(kind, id, 64) }
func warMemberOrg(member api.WarMember) string {
	if member.AllianceName != "" {
		return member.AllianceName
	}
	return member.CorporationName
}
func warMemberEmptyLabel(labels []string) string {
	if len(labels) > 0 && labels[0] != "" {
		return labels[0]
	}
	return "No members found"
}
func warMemberCount(value int64) string {
	if value == 0 {
		return "—"
	}
	return warNumber(value)
}
func warMemberISK(value float64) string {
	if value == 0 {
		return "—"
	}
	return warISK(value)
}
func warSecurityLabel(class string) string {
	switch class {
	case "highsec":
		return "High-sec"
	case "lowsec":
		return "Low-sec"
	case "nullsec":
		return "Null-sec"
	case "wormhole":
		return "Wormhole / Pochven"
	}
	return "Unknown"
}
func warSecurityClass(class string) string {
	switch class {
	case "highsec":
		return "high"
	case "lowsec":
		return "low"
	case "nullsec":
		return "null"
	case "wormhole":
		return "wormhole"
	}
	return ""
}
func warSystemSecurity(value *float64) string {
	if value == nil {
		return "?"
	}
	return fmt.Sprintf("%.1f", *value)
}
