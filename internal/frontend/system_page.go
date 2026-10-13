package frontend

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type systemPageView struct {
	ID          int64
	Tab         string
	Data        api.UniverseSystemData
	Valuable    []api.ValuableKill
	Kills       api.KilllistPage
	Battles     api.EntityBattlesData
	Chart       systemChartView
	Transitions []systemSovereigntyTransition
}

func (s *site) loadSystemPage(ctx context.Context, id int64, tab string, valuable bool) (systemPageView, error) {
	if id < 1 || id > 2_147_483_647 {
		return systemPageView{}, ErrNotFound
	}
	if tab == "" {
		tab = "dashboard"
	}
	if tab != "dashboard" && tab != "kills" && tab != "battles" && tab != "map" {
		return systemPageView{}, ErrNotFound
	}
	data, found, err := s.options.Universe.UniverseSystem(ctx, id)
	if err != nil {
		return systemPageView{}, err
	}
	if !found {
		return systemPageView{}, ErrNotFound
	}
	view := systemPageView{ID: id, Tab: tab, Data: data, Chart: systemActivityChart(data.Activity.History), Transitions: systemSovereigntyTransitions(data.SovereigntyHistory)}
	if valuable {
		view.Valuable, err = s.options.Universe.UniverseLocationValuable(ctx, "system", id, "kills")
		if err != nil {
			return systemPageView{}, err
		}
	}
	switch tab {
	case "kills":
		view.Kills, err = s.locationKilllist(ctx, "system", id, 50, nil)
	case "battles":
		if s.options.Entities != nil {
			view.Battles, err = s.options.Entities.EntityBattles(ctx, "system", id, 1)
		}
	}
	return view, err
}

func systemMetadata(view systemPageView) Metadata {
	system := view.Data.System
	path := fmt.Sprintf("/system/%d", view.ID)
	title := system.SystemName
	if view.Tab != "dashboard" {
		title += " (" + map[string]string{"kills": "Kills", "battles": "Battles", "map": "Map"}[view.Tab] + ")"
		path += "/" + view.Tab
	}
	description := fmt.Sprintf("%s (%s) — %s, %s", system.SystemName, systemSecurityLabel(system.Security), system.ConstellationName, system.RegionName)
	if activity := view.Data.Activity.Latest; activity != nil {
		description += fmt.Sprintf(". Last hour: %d ship kills, %d pod kills, %d jumps", activity.ShipKills, activity.PodKills, activity.ShipJumps)
	}
	description += fmt.Sprintf(". 7d: %d kills — EVE-KILL.", view.Data.Stats.Kills)
	image := fmt.Sprintf("/images/systems/%d", view.ID)
	return Metadata{Title: title, Description: description, CanonicalPath: path, Image: image, ImageAlt: system.SystemName, Indexable: true,
		StructuredData: []any{
			map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": []any{
				map[string]any{"@type": "ListItem", "position": 1, "name": "Home", "item": canonicalOrigin + "/"},
				map[string]any{"@type": "ListItem", "position": 2, "name": system.RegionName, "item": canonicalOrigin + fmt.Sprintf("/region/%d", system.RegionID)},
				map[string]any{"@type": "ListItem", "position": 3, "name": system.ConstellationName, "item": canonicalOrigin + fmt.Sprintf("/constellation/%d", system.ConstellationID)},
				map[string]any{"@type": "ListItem", "position": 4, "name": system.SystemName, "item": canonicalOrigin + fmt.Sprintf("/system/%d", view.ID)},
			}},
			map[string]any{"@context": "https://schema.org", "@type": "Place", "name": system.SystemName, "url": canonicalOrigin + fmt.Sprintf("/system/%d", view.ID), "image": canonicalOrigin + image},
		}}
}

func systemSecurityLabel(security *float64) string {
	if security == nil {
		return "?"
	}
	return fmt.Sprintf("%.1f", *security)
}

func systemSecurityClass(security *float64) string {
	if security == nil {
		return "unknown"
	}
	if *security >= 0.5 {
		return "high"
	}
	if *security > 0 {
		return "low"
	}
	return "null"
}

func systemSovereigntyName(value *api.UniverseSovereignty) string {
	if value == nil {
		return ""
	}
	if value.AllianceID > 0 {
		return value.AllianceName
	}
	if value.FactionID > 0 {
		return value.FactionName
	}
	if value.CorporationID > 0 {
		return value.CorporationName
	}
	return "Unclaimed"
}

func systemSovereigntyImage(value *api.UniverseSovereignty, size int) string {
	if value == nil {
		return ""
	}
	if value.AllianceID > 0 {
		return eveEntityImage("alliance", value.AllianceID, size)
	}
	if value.FactionID > 0 {
		return eveEntityImage("corporation", value.FactionID, size)
	}
	if value.CorporationID > 0 {
		return eveEntityImage("corporation", value.CorporationID, size)
	}
	return ""
}

func systemSovereigntyID(value api.UniverseSovereignty) int64 {
	if value.AllianceID > 0 {
		return value.AllianceID
	}
	if value.CorporationID > 0 {
		return value.CorporationID
	}
	return value.FactionID
}

func systemSovereigntyKind(value api.UniverseSovereignty) string {
	if value.AllianceID > 0 {
		return "alliance"
	}
	if value.CorporationID > 0 {
		return "corporation"
	}
	return "faction"
}

type systemSovereigntyTransition struct {
	From *api.UniverseSovereignty
	To   api.UniverseSovereignty
}

func systemSovereigntyTransitions(history []api.UniverseSovereignty) []systemSovereigntyTransition {
	result := make([]systemSovereigntyTransition, 0, len(history))
	for index, current := range history {
		var previous *api.UniverseSovereignty
		if index+1 < len(history) {
			previous = &history[index+1]
		}
		if previous == nil || systemSovereigntyID(current) != systemSovereigntyID(*previous) {
			result = append(result, systemSovereigntyTransition{From: previous, To: current})
		}
	}
	return result
}

func systemSovereigntyDate(value string) string {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return parsed.UTC().Format("Jan 2, 2006")
}

func (s *site) registerSystemPages(mux *http.ServeMux) {
	load := func(r *http.Request) (Page, error) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			return Page{}, ErrNotFound
		}
		view, err := s.loadSystemPage(r.Context(), id, r.PathValue("tab"), true)
		if err != nil {
			return Page{}, err
		}
		return Page{Metadata: systemMetadata(view), Content: systemPage(view), SharedCacheTTL: 2 * time.Minute}, nil
	}
	mux.Handle("GET /system/{id}", s.pageHandler(load))
	mux.Handle("GET /system/{id}/{tab}", s.pageHandler(load))
	mux.HandleFunc("GET /_fragments/system/tab/{id}/{tab}", s.systemTabFragment)
	mux.HandleFunc("GET /_fragments/system/kills/{id}", s.systemKillsFragment)
	mux.HandleFunc("GET /_fragments/system/valuable/{id}", s.systemValuableFragment)
	mux.HandleFunc("GET /_fragments/system/battles/{id}", s.systemBattlesFragment)
}

func (s *site) systemTabFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	view, err := s.loadSystemPage(r.Context(), id, r.PathValue("tab"), false)
	if err != nil {
		if err == ErrNotFound {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Could not load system", http.StatusServiceUnavailable)
		}
		return
	}
	s.renderFragment(w, r, systemMain(view))
}

func (s *site) systemKillsFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	limit, err := fragmentInt(r.URL.Query().Get("limit"), 50, 10, 100)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}
	page, err := fragmentInt(r.URL.Query().Get("page"), 1, 1, 1_000_000)
	if err != nil {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	var after *int64
	if raw := r.URL.Query().Get("after"); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || value < 1 {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}
		after = &value
	}
	if page > 1 && after == nil {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	kills, err := s.locationKilllist(r.Context(), "system", id, limit, after)
	if err != nil {
		http.Error(w, "Could not load kills", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, killList(kills, killListOptions{Kind: "system", Page: page, Limit: limit}))
}

func (s *site) systemValuableFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	category := r.URL.Query().Get("category")
	if err != nil || id < 1 || id > 2_147_483_647 || category != "kills" && category != "ships" && category != "structures" {
		http.NotFound(w, r)
		return
	}
	kills, err := s.options.Universe.UniverseLocationValuable(r.Context(), "system", id, category)
	if err != nil {
		http.Error(w, "Could not load valuable kills", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, mostValuable(kills, category, false))
}

func (s *site) systemBattlesFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	page, pageErr := fragmentInt(r.URL.Query().Get("page"), 1, 1, 1_000_000)
	if err != nil || id < 1 || id > 2_147_483_647 || pageErr != nil {
		http.NotFound(w, r)
		return
	}
	if s.options.Entities == nil {
		http.Error(w, "Could not load battles", http.StatusServiceUnavailable)
		return
	}
	battles, err := s.options.Entities.EntityBattles(r.Context(), "system", id, page)
	if err != nil {
		http.Error(w, "Could not load battles", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, entityBattles("system", id, battles))
}
