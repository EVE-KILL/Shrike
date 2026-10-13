package frontend

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type rankingChoice struct{ Value, Label string }

var rankingEntities = []rankingChoice{{"character", "Characters"}, {"corporation", "Corporations"}, {"alliance", "Alliances"}, {"ship", "Ships"}, {"system", "Systems"}, {"region", "Regions"}}
var rankingWindows = []rankingChoice{{"hourly", "1h"}, {"weekly", "7d"}, {"fourteen_days", "14d"}, {"thirty_days", "30d"}, {"ninety_days", "90d"}, {"one_eighty_days", "180d"}, {"one_year", "365d"}, {"all_time", "All Time"}}

type rankingsView struct {
	Entity, Window, EntityLabel, WindowLabel string
	Entries                                  []api.RankingEntry
}

func rankingSelection(values url.Values) (string, string) {
	entity, window := values.Get("entity"), values.Get("window")
	valid := false
	for _, choice := range rankingEntities {
		if choice.Value == entity {
			valid = true
			break
		}
	}
	if !valid {
		entity = "character"
	}
	valid = false
	for _, choice := range rankingWindows {
		if choice.Value == window {
			valid = true
			break
		}
	}
	if !valid {
		window = "all_time"
	}
	return entity, window
}
func rankingLabel(choices []rankingChoice, value string) string {
	for _, choice := range choices {
		if choice.Value == value {
			return choice.Label
		}
	}
	return ""
}
func (s *site) rankingsView(r *http.Request) (rankingsView, error) {
	entity, window := rankingSelection(r.URL.Query())
	entries, err := s.options.Rankings.Rankings(r.Context(), entity, window)
	if err != nil {
		return rankingsView{}, err
	}
	return rankingsView{Entity: entity, Window: window, EntityLabel: rankingLabel(rankingEntities, entity), WindowLabel: rankingLabel(rankingWindows, window), Entries: entries}, nil
}
func (s *site) rankingsPage(r *http.Request) (Page, error) {
	view, err := s.rankingsView(r)
	if err != nil {
		return Page{}, err
	}
	path := "/rankings"
	if r.URL.Query().Has("entity") || r.URL.Query().Has("window") {
		path += fmt.Sprintf("?entity=%s&window=%s", url.QueryEscape(view.Entity), url.QueryEscape(view.Window))
	}
	return Page{Metadata: Metadata{
		Title: "EVE-KILL Rankings", CanonicalPath: path, Indexable: true,
		Description: "Explore live hourly, weekly, 90-day, and all-time EVE-KILL combat rankings.",
		Stylesheets: []string{"rankings.css"}, ModuleScripts: []string{"rankings.js"},
	}, Content: rankingsPage(view), SharedCacheTTL: time.Minute}, nil
}
func (s *site) rankingsFragment(w http.ResponseWriter, r *http.Request) {
	view, err := s.rankingsView(r)
	if err != nil {
		http.Error(w, "Could not load rankings", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, rankingsResults(view))
}
func rankingEntityPath(entity string, id int64) string {
	if entity == "ship" {
		entity = "item"
	}
	return idPath(entity, id)
}
func rankingEntityImage(entity string, id int64, size int) string {
	switch entity {
	case "character", "corporation", "alliance":
		return eveEntityImage(entity, id, size)
	case "ship":
		return fmt.Sprintf("/images/types/%d/icon?size=%d", id, size)
	case "system", "region":
		return fmt.Sprintf("/images/%ss/%d?size=%d", entity, id, size)
	}
	return ""
}
func rankingButtonClass(active bool, kind string) string {
	classes := []string{"ranking-filter"}
	if active {
		classes = append(classes, "active", kind)
	}
	return strings.Join(classes, " ")
}
func rankingNumber(value int64) string { return fullNumber(float64(value)) }
func rankingRank(value int64) string   { return strconv.FormatInt(value, 10) }

func rankingPodiumClass(entry api.RankingEntry) string {
	if entry.OverallRank == 1 {
		return "ranking-podium-card first"
	}
	return "ranking-podium-card"
}
