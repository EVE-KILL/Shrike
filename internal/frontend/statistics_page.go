package frontend

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type statisticsSection struct {
	Key, Label, Icon, Blurb string
	Period                  bool
}

// Card order, descriptions and responsive grids come from stats.vue.
var statisticsSections = []statisticsSection{
	{"activity", "Activity", "crosshair", "Who is killing the most", true},
	{"isk", "ISK", "coins", "Value destroyed and lost", true},
	{"pvp", "PvP", "swords", "Solo work and points earned", true},
	{"locations", "Locations", "map-pinned", "Where the fighting happens", true},
	{"ships", "Ships", "rocket", "What flies and what dies", true},
	{"largest", "Largest", "users", "By member count", false},
	{"security", "Security", "scale", "Sec status extremes", false},
	{"growth", "Growth", "trending-up", "Gaining and shedding members", false},
	{"newest", "Newest", "sparkles", "Recently founded", false},
	{"achievements", "Achievements", "trophy", "Top achievement scores", false},
}

type statisticsPeriod struct {
	Label string
	Days  float64
}

var statisticsPeriods = []statisticsPeriod{{"1h", 1.0 / 24}, {"6h", 0.25}, {"12h", 0.5}, {"1d", 1}, {"2d", 2}, {"3d", 3}, {"4d", 4}, {"5d", 5}, {"6d", 6}, {"7d", 7}, {"14d", 14}, {"30d", 30}, {"45d", 45}, {"90d", 90}}

type statisticsCardSpec struct {
	Key, Section, Kind, Title, Icon, Image, Format, Color string
	Query                                                 api.StatisticsQuery
}

func statisticsCardSpecs() []statisticsCardSpec {
	cards := []statisticsCardSpec{}
	for _, card := range []struct{ Key, Label, Image, Section, Icon, Format string }{
		{"characters", "Top Killers", "character", "activity", "user-round", ""},
		{"corporations", "Top Corporations", "corporation", "activity", "building-2", ""},
		{"alliances", "Top Alliances", "alliance", "activity", "shield", ""},
		{"isk_destroyers_chars", "Top ISK Destroyers", "character", "isk", "user-round", "isk"},
		{"isk_destroyers_corps", "Top ISK Corps", "corporation", "isk", "building-2", "isk"},
		{"isk_destroyers_alliances", "Top ISK Alliances", "alliance", "isk", "shield", "isk"},
		{"biggest_losers", "Biggest ISK Losers", "character", "isk", "trending-down", "isk"},
		{"solo_killers", "Top Solo Killers", "character", "pvp", "user-round", ""},
		{"top_points", "Top Points Earners", "character", "pvp", "star", ""},
		{"systems", "Top Systems", "", "locations", "map-pin", ""},
		{"regions", "Top Regions", "", "locations", "map", ""},
		{"dangerous_systems", "Most Dangerous Systems", "", "locations", "skull", "isk"},
		{"deadliest_regions", "Deadliest Regions", "", "locations", "skull", "isk"},
		{"ships", "Top Ships (Kills)", "ship", "ships", "rocket", ""},
		{"most_used_ships", "Most Used Ships", "ship", "ships", "rocket", ""},
		{"most_destroyed_ships", "Most Destroyed Ships", "ship", "ships", "flame", ""},
	} {
		cards = append(cards, statisticsCardSpec{Key: card.Key, Section: card.Section, Kind: "period", Title: card.Label, Icon: card.Icon, Image: card.Image, Format: card.Format, Color: "text-blue-400/80", Query: api.StatisticsQuery{DataType: card.Key, Days: 7}})
	}
	add := func(section, entity, variant, title, color string) {
		key := section + "-" + entity
		if variant != "" {
			key += "-" + variant
		}
		query := api.StatisticsQuery{Section: section, EntityType: entity, Days: 7}
		if section == "security" {
			query.Rank = variant
		}
		if section == "growth" {
			query.Direction = variant
		}
		cards = append(cards, statisticsCardSpec{Key: key, Section: section, Kind: section, Title: title, Image: entity, Color: color, Query: query})
	}
	for _, entity := range []string{"alliance", "corporation"} {
		add("largest", entity, "", "Largest "+statisticsEntityLabel(entity), "text-blue-400/80")
	}
	for _, rank := range []string{"pirate", "carebear"} {
		for _, entity := range []string{"alliance", "corporation"} {
			add("security", entity, rank, "Most "+statisticsTitle(rank)+" "+statisticsEntityLabel(entity), killBoolClass(rank == "pirate", "text-red-400/80", "text-cyan-400/80"))
		}
	}
	for _, rank := range []string{"pirate", "carebear"} {
		cards = append(cards, statisticsCardSpec{Key: "security-character-" + rank, Section: "security", Kind: "security-character", Title: "Most " + statisticsTitle(rank) + " Characters", Image: "character", Color: killBoolClass(rank == "pirate", "text-red-400/80", "text-cyan-400/80"), Query: api.StatisticsQuery{DataType: rank + "_characters", Days: 7}})
	}
	for _, direction := range []string{"growing", "shrinking"} {
		for _, entity := range []string{"alliance", "corporation"} {
			add("growth", entity, direction, "Fastest "+statisticsTitle(direction)+" "+statisticsEntityLabel(entity)+" — 7d", killBoolClass(direction == "growing", "text-isk/80", "text-red-400/80"))
		}
	}
	for _, entity := range []string{"alliance", "corporation", "character"} {
		add("newest", entity, "", "Newest "+statisticsEntityLabel(entity), "text-amber-400/80")
	}
	for _, entity := range []string{"character", "corporation", "alliance"} {
		add("achievements", entity, "", "Top Achievement "+statisticsEntityLabel(entity), "text-purple-400/80")
	}
	return cards
}

func statisticsEntityLabel(entity string) string {
	return map[string]string{"alliance": "Alliances", "corporation": "Corporations", "character": "Characters"}[entity]
}
func statisticsTitle(value string) string { return strings.ToUpper(value[:1]) + value[1:] }

type statisticsCardView struct {
	Spec    statisticsCardSpec
	Entries []api.StatisticsEntry
	Pending bool
}
type statisticsView struct {
	Site                      api.SiteConfigurationResponse
	Name, Filter, PeriodLabel string
	Days                      float64
	Cards                     []statisticsCardView
}

func statisticsSelection(query url.Values, domain bool) (string, float64, string) {
	filter := "all"
	for _, section := range statisticsSections {
		if section.Key == query.Get("section") && (!domain || section.Period) {
			filter = section.Key
		}
	}
	days, _ := strconv.ParseFloat(query.Get("period"), 64)
	for _, period := range statisticsPeriods {
		if days == period.Days {
			return filter, days, period.Label
		}
	}
	return filter, 7, "7d"
}
func statisticsShow(filter, section string) bool { return filter == "all" || filter == section }
func statisticsShowPeriod(filter string) bool {
	if filter == "all" {
		return true
	}
	for _, section := range statisticsSections {
		if section.Key == filter {
			return section.Period
		}
	}
	return false
}
func statisticsSectionCards(cards []statisticsCardView, key string) []statisticsCardView {
	result := []statisticsCardView{}
	for _, card := range cards {
		if card.Spec.Section == key {
			result = append(result, card)
		}
	}
	return result
}
func statisticsGrid(key string) string {
	if key == "isk" || key == "locations" {
		return "grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3"
	}
	return "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3"
}
func statisticsPeriodValue(days float64) string { return strconv.FormatFloat(days, 'g', -1, 64) }
func statisticsCardSlot(card statisticsCardView) string {
	if card.Spec.Kind == "period" {
		return statisticsPeriodValue(card.Spec.Query.Days)
	}
	return "population"
}
func statisticsFilterClass(active bool) string {
	return killBoolClass(active, "bg-blue-500/20 text-blue-400 border-blue-500/30", "text-gray-400 border-white/[0.08] hover:text-blue-400 hover:bg-blue-500/[0.04]")
}
func statisticsPeriodClass(active bool) string {
	return killBoolClass(active, "bg-blue-500/20 text-blue-400", "bg-white/[0.02] text-gray-500 hover:bg-white/[0.05] hover:text-gray-300")
}

func (s *site) statisticsPage(r *http.Request) (Page, error) {
	config, err := s.statisticsSite(r)
	if err != nil {
		return Page{}, err
	}
	view := statisticsView{Site: config, Name: "EVE-KILL"}
	if config.Domain != nil && config.Domain.SiteName != nil && *config.Domain.SiteName != "" {
		view.Name = *config.Domain.SiteName
	}
	view.Filter, view.Days, view.PeriodLabel = statisticsSelection(r.URL.Query(), config.Domain != nil)
	queries := []api.StatisticsQuery{}
	for _, spec := range statisticsCardSpecs() {
		if config.Domain != nil && spec.Kind != "period" {
			continue
		}
		if spec.Kind == "period" {
			spec.Query.Days = view.Days
		}
		queries = append(queries, spec.Query)
		view.Cards = append(view.Cards, statisticsCardView{Spec: spec, Pending: true})
	}
	for index, entries := range s.options.Statistics.StatisticsBoardsCached(r.Context(), config, queries) {
		view.Cards[index].Entries, view.Cards[index].Pending = entries, false
	}
	title := "EVE Online PvP Statistics"
	description := "EVE Online kill statistics — top pilots, corporations, alliances, ships, systems, and regions ranked by kills, ISK destroyed, and PvP activity."
	if config.Domain != nil {
		title = "Statistics — " + view.Name
		description = "Kill statistics for " + view.Name + " — top pilots, corporations, ships, systems and regions by kills and ISK destroyed."
	}
	return Page{Metadata: Metadata{Title: title, OGTitle: "Statistics — " + view.Name, Description: description, CanonicalPath: "/stats", Indexable: !config.IsDomainHost, Stylesheets: []string{"utilities.css", "utility-page.css", "statistics.css"}, ModuleScripts: []string{"statistics.js"}}, Content: statisticsContent(view), SharedCacheTTL: time.Minute}, nil
}

func (s *site) statisticsCardFragment(w http.ResponseWriter, r *http.Request) {
	config, err := s.statisticsSite(r)
	if err != nil {
		http.Error(w, "Could not load statistics", http.StatusServiceUnavailable)
		return
	}
	_, days, _ := statisticsSelection(r.URL.Query(), config.Domain != nil)
	for _, spec := range statisticsCardSpecs() {
		if spec.Key != r.PathValue("card") {
			continue
		}
		if config.Domain != nil && spec.Kind != "period" {
			http.NotFound(w, r)
			return
		}
		if spec.Kind == "period" {
			spec.Query.Days = days
		}
		entries, err := s.options.Statistics.StatisticsBoard(r.Context(), config, spec.Query)
		if err != nil {
			http.Error(w, "Could not load this table. Try again.", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Cache-Control", "private, no-store")
		s.renderFragment(w, r, statisticsCardRows(statisticsCardView{Spec: spec, Entries: entries}))
		return
	}
	http.NotFound(w, r)
}

func statisticsNumber(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0"
	}
	value = math.Round(value*1000) / 1000
	raw := strconv.FormatFloat(value, 'f', 3, 64)
	whole, fraction, _ := strings.Cut(raw, ".")
	start := 0
	if strings.HasPrefix(whole, "-") {
		start = 1
	}
	for i := len(whole) - 3; i > start; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	fraction = strings.TrimRight(fraction, "0")
	if fraction != "" {
		return whole + "." + fraction
	}
	return whole
}
func statisticsRaw(value float64) string {
	if value == 0 {
		return "0"
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
func statisticsISK(entry api.StatisticsEntry) string {
	value := entry.ISK
	if value == 0 {
		value = entry.Count
	}
	return compactNumber(value)
}
func statisticsDelta(value int64) string {
	return killBoolClass(value > 0, "+", "") + fullNumber(float64(value))
}
func statisticsDeltaColor(value int64) string {
	if value > 0 {
		return "text-green-400"
	}
	return killBoolClass(value < 0, "text-red-400", "text-gray-600")
}
func statisticsSecurityColor(value float64) string {
	switch {
	case value <= -5:
		return "text-red-500"
	case value < 0:
		return "text-red-400"
	case value == 0:
		return "text-gray-400"
	case value < 3:
		return "text-green-400"
	default:
		return "text-green-500"
	}
}
func statisticsTimeAgo(when *time.Time) string {
	if when == nil {
		return ""
	}
	days := int(math.Floor(time.Since(*when).Hours() / 24))
	switch {
	case days < 1:
		return "Today"
	case days < 30:
		return fmt.Sprintf("%dd ago", days)
	case days < 365:
		return fmt.Sprintf("%dmo ago", days/30)
	default:
		return fmt.Sprintf("%dy ago", days/365)
	}
}
