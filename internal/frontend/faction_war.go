package frontend

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"golang.org/x/sync/errgroup"
)

type factionWarView struct {
	Data  api.FactionWarData
	Kills api.KilllistPage
}

func validFactionWar(slug string) bool {
	return member(slug, "caldari-vs-gallente", "amarr-vs-minmatar")
}
func factionWarTitle(data api.FactionWarData) string {
	return data.Side1.Name + " vs " + data.Side2.Name + " — Faction Warfare"
}

func (s *site) factionWarPage(r *http.Request) (Page, error) {
	slug := r.PathValue("matchup")
	if !validFactionWar(slug) {
		return Page{}, ErrNotFound
	}
	view := factionWarView{}
	group, ctx := errgroup.WithContext(r.Context())
	group.Go(func() (err error) { view.Data, err = s.options.FactionWars.FactionWarDashboard(ctx, slug); return })
	group.Go(func() (err error) {
		view.Kills, err = s.options.FactionWars.FactionWarKilllist(ctx, slug, "combined", 1, 50, nil)
		return
	})
	if err := group.Wait(); err != nil {
		return Page{}, err
	}
	data := view.Data
	wz := data.Overview.Warzone
	title := factionWarTitle(data)
	description := fmt.Sprintf("%s vs %s — faction warfare in EVE Online. %s holds %d systems, %s holds %d of %d. %s kills, %s ISK destroyed (30d). Warzone map, system control, leaderboards on EVE-KILL.", data.Side1.Name, data.Side2.Name, data.Side1.Name, wz.Side1.Total, data.Side2.Name, wz.Side2.Total, wz.Total, warNumber(data.Side1.Kills+data.Side2.Kills), warISK(data.Side1.ISKDestroyed+data.Side2.ISKDestroyed))
	url := "https://eve-kill.com/faction-war/" + slug
	return Page{Metadata: Metadata{Title: title, OGTitle: title + " — EVE-KILL", Description: description, CanonicalPath: "/faction-war/" + slug, Indexable: true, Image: eveEntityImage("corporation", data.Side1.CorpID, 256), Stylesheets: []string{"utilities.css", "war-detail.css", "faction-war.css"}, ModuleScripts: []string{"faction-war.js"}, StructuredData: []any{
		map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": []any{map[string]any{"@type": "ListItem", "position": 1, "name": "Home", "item": "https://eve-kill.com/"}, map[string]any{"@type": "ListItem", "position": 2, "name": "Wars", "item": "https://eve-kill.com/wars"}, map[string]any{"@type": "ListItem", "position": 3, "name": title, "item": url}}},
		map[string]any{"@context": "https://schema.org", "@type": "Event", "name": title, "url": url, "description": description, "eventStatus": "https://schema.org/EventScheduled", "eventAttendanceMode": "https://schema.org/OnlineEventAttendanceMode", "location": map[string]any{"@type": "VirtualLocation", "url": "https://www.eveonline.com"}, "organizer": []any{map[string]any{"@type": "Organization", "name": data.Side1.Name, "url": "https://eve-kill.com" + idPath("faction", data.Side1.ID)}, map[string]any{"@type": "Organization", "name": data.Side2.Name, "url": "https://eve-kill.com" + idPath("faction", data.Side2.ID)}}},
	}}, Content: factionWarContent(view), SharedCacheTTL: time.Minute}, nil
}

func (s *site) factionWarFragment(w http.ResponseWriter, r *http.Request) {
	slug, part := r.PathValue("matchup"), r.PathValue("part")
	if !validFactionWar(slug) || !member(part, "kills", "sidebar", "members", "intel", "systems") {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	w.Header().Set("Cache-Control", "private, no-store")
	fail := func() { http.Error(w, "Could not load faction warfare", http.StatusServiceUnavailable) }
	switch part {
	case "kills":
		side := q.Get("side")
		if !member(side, "combined", "side1", "side2") {
			http.Error(w, "Invalid side", 400)
			return
		}
		page, err := fragmentInt(q.Get("page"), 1, 1, 10000)
		if err != nil {
			http.Error(w, "Invalid page", 400)
			return
		}
		limit, err := fragmentInt(q.Get("limit"), 50, 10, 100)
		if err != nil {
			http.Error(w, "Invalid limit", 400)
			return
		}
		var after *int64
		if raw := q.Get("after"); raw != "" {
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || id < 1 {
				http.Error(w, "Invalid cursor", 400)
				return
			}
			after = &id
		}
		if page > 1 && after == nil {
			http.Error(w, "Missing cursor", http.StatusBadRequest)
			return
		}
		data, err := s.options.FactionWars.FactionWarKilllist(r.Context(), slug, side, page, limit, after)
		if err != nil {
			fail()
			return
		}
		s.renderFragment(w, r, killList(data, killListOptions{Kind: "faction-war", Page: page, Limit: limit, Filtered: true}))
	case "sidebar":
		side, period := q.Get("side"), q.Get("period")
		if !member(side, "combined", "side1", "side2") || !member(period, "yesterday", "last_week", "active_total") {
			http.Error(w, "Invalid leaderboard selection", 400)
			return
		}
		data, err := s.options.FactionWars.FactionWarDashboard(r.Context(), slug)
		if err != nil {
			fail()
			return
		}
		if period != "last_week" {
			overview, err := s.options.FactionWars.FactionWarOverview(r.Context(), slug, period)
			if err != nil {
				fail()
				return
			}
			data.Overview = overview
		}
		s.renderFragment(w, r, factionWarSidebar(data, side, period))
	case "members":
		if !member(q.Get("side"), "combined", "aggressor", "defender") || !member(q.Get("sort"), "activity", "kills", "losses", "isk") {
			http.Error(w, "Invalid member selection", 400)
			return
		}
		data, err := s.options.FactionWars.FactionWarMembers(r.Context(), slug, q)
		if err != nil {
			fail()
			return
		}
		s.renderFragment(w, r, warMembersResults(data, "No faction pilots found in the last 30 days"))
	case "intel":
		data, err := s.options.FactionWars.FactionWarIntel(r.Context(), slug)
		if err != nil {
			fail()
			return
		}
		s.renderFragment(w, r, warIntelContent(data))
	case "systems":
		data, err := s.options.FactionWars.FactionWarSystems(r.Context(), slug)
		if err != nil {
			fail()
			return
		}
		s.renderFragment(w, r, factionWarSystemsContent(data))
	}
}

func factionWarTabs(data api.FactionWarData) []entityTab {
	return []entityTab{{"combined", "Both Sides", "layers"}, {"side1", data.Side1.Name, "swords"}, {"side2", data.Side2.Name, "shield"}, {"members", "Members", "users"}, {"intel", "Intel", "radar"}, {"map", "Warzone Map", "map"}, {"systems", "Systems", "list"}}
}
func factionWarShare(a, b float64) int {
	if a+b <= 0 {
		return 50
	}
	return int(math.Floor(a/(a+b)*100 + 0.5))
}
func factionWarControlPct(overview api.FactionWarOverview) int {
	if overview.Warzone.Total <= 0 {
		return 50
	}
	return int(math.Floor(float64(overview.Warzone.Side1.Total)/float64(overview.Warzone.Total)*100 + .5))
}
func factionWarRoundedPct(system api.FactionWarSystem) string {
	return strconv.Itoa(int(math.Floor(factionWarPct(system) + .5)))
}
func factionWarPeriod(period string) string {
	switch period {
	case "yesterday":
		return "Yesterday"
	case "active_total":
		return "All Time"
	default:
		return "This Week"
	}
}
func factionWarLeaders(sides api.FactionWarLeaderboardSides, side string) []api.FactionWarLeaderboardEntry {
	if side == "side1" {
		return sides.Side1
	}
	if side == "side2" {
		return sides.Side2
	}
	rows := append(append([]api.FactionWarLeaderboardEntry{}, sides.Side1...), sides.Side2...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Kills > rows[j].Kills })
	return rows[:min(10, len(rows))]
}
func factionWarFlipDay(day string) string {
	date, err := time.Parse("2006-01-02", day)
	if err != nil {
		return day
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	label := date.Format("2 Jan")
	switch int(today.Sub(date).Hours() / 24) {
	case 0:
		return "Today — " + label
	case 1:
		return "Yesterday — " + label
	}
	return label
}
func factionWarFlipTime(raw string) string {
	when, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return ""
	}
	return when.UTC().Format("15:04")
}
func factionWarColor(id int64) string {
	switch id {
	case 500001:
		return "#3b82f6"
	case 500002:
		return "#ef4444"
	case 500003:
		return "#f59e0b"
	case 500004:
		return "#22c55e"
	}
	return "#64748b"
}
func factionWarShort(id int64) string {
	switch id {
	case 500001:
		return "Caldari"
	case 500002:
		return "Minmatar"
	case 500003:
		return "Amarr"
	case 500004:
		return "Gallente"
	}
	return "?"
}
func factionWarPct(system api.FactionWarSystem) float64 {
	if system.Threshold <= 0 {
		return 0
	}
	return math.Min(100, float64(system.VP)/float64(system.Threshold)*100)
}
func factionWarSystemRows(data api.FactionWarSystems) []api.FactionWarSystem {
	rows := append([]api.FactionWarSystem{}, data.Systems...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Kills > rows[j].Kills })
	return rows
}
func factionWarStatusClass(status string) string {
	switch status {
	case "vulnerable":
		return "bg-red-500/20 text-red-400"
	case "contested":
		return "bg-yellow-500/20 text-yellow-400"
	case "captured":
		return "bg-purple-500/20 text-purple-400"
	}
	return "bg-green-500/10 text-green-400/70"
}
func factionWarVPClass(status string) string {
	if status == "vulnerable" {
		return "bg-red-500"
	}
	if status == "contested" {
		return "bg-yellow-500"
	}
	return "bg-white/20"
}
func factionWarSecurity(sec float64) string {
	if sec >= .5 {
		return "text-green-400"
	}
	if sec > 0 {
		return "text-amber-400"
	}
	return "text-red-400"
}
