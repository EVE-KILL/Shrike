package frontend

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type locationPageView struct {
	Kind          string
	ID            int64
	Tab           string
	Name          string
	RegionID      int64
	RegionName    string
	Description   string
	Dominant      *api.UniverseLocationSovereignty
	Region        api.UniverseRegionData
	Constellation api.UniverseConstellationData
	Valuable      []api.ValuableKill
	Kills         api.KilllistPage
	Battles       api.EntityBattlesData
}

func (s *site) loadLocationPage(ctx context.Context, kind string, id int64, tab string, valuable bool) (locationPageView, error) {
	if kind != "region" && kind != "constellation" || id < 1 || id > 2_147_483_647 {
		return locationPageView{}, ErrNotFound
	}
	if tab == "" {
		tab = "dashboard"
	}
	if tab != "dashboard" && tab != "kills" && tab != "battles" && tab != "map" {
		return locationPageView{}, ErrNotFound
	}
	view := locationPageView{Kind: kind, ID: id, Tab: tab}
	if kind == "region" {
		data, found, err := s.options.Universe.UniverseRegion(ctx, id)
		if err != nil {
			return locationPageView{}, err
		}
		if !found {
			return locationPageView{}, ErrNotFound
		}
		view.Region = data
		view.Name = data.Region.Name
		view.RegionID = id
		view.RegionName = data.Region.Name
		view.Description = entityBioHTML("", data.Region.Description)
		if len(data.SovDistribution) > 0 {
			view.Dominant = &view.Region.SovDistribution[0]
		}
	} else {
		data, found, err := s.options.Universe.UniverseConstellation(ctx, id)
		if err != nil {
			return locationPageView{}, err
		}
		if !found {
			return locationPageView{}, ErrNotFound
		}
		view.Constellation = data
		view.Name = data.Constellation.ConstellationName
		view.RegionID = data.Constellation.RegionID
		view.RegionName = data.Constellation.RegionName
		if len(data.SovDistribution) > 0 {
			view.Dominant = &view.Constellation.SovDistribution[0]
		}
	}
	if valuable {
		kills, err := s.options.Universe.UniverseLocationValuable(ctx, kind, id, "kills")
		if err != nil {
			return locationPageView{}, err
		}
		view.Valuable = kills
	}
	switch tab {
	case "kills":
		kills, err := s.locationKilllist(ctx, kind, id, 50, nil)
		view.Kills = kills
		if err != nil {
			return locationPageView{}, err
		}
	case "battles":
		if s.options.Entities != nil {
			battles, err := s.options.Entities.EntityBattles(ctx, kind, id, 1)
			view.Battles = battles
			if err != nil {
				return locationPageView{}, err
			}
		}
	}
	return view, nil
}

func locationMetadata(view locationPageView) Metadata {
	path := fmt.Sprintf("/%s/%d", view.Kind, view.ID)
	title := view.Name
	if view.Tab != "dashboard" {
		title += " (" + map[string]string{"kills": "Kills", "battles": "Battles", "map": "Map"}[view.Tab] + ")"
		path += "/" + view.Tab
	}
	image := fmt.Sprintf("/images/%ss/%d", view.Kind, view.ID)
	description := view.Name + " — "
	if view.Kind == "region" {
		description += fmt.Sprintf("%d constellations, %d systems", view.Region.Region.ConstellationCount, view.Region.Region.SystemCount)
	} else {
		description += fmt.Sprintf("%s — %d systems", view.RegionName, len(view.Constellation.Systems))
	}
	stats := locationStatistics(view)
	description += fmt.Sprintf(". 7d: %d kills, %d pod kills", stats.Kills, stats.PodKills)
	if stats.TotalValue > 0 {
		description += ", " + compactNumber(stats.TotalValue) + " ISK destroyed"
	}
	description += " — EVE-KILL."
	crumbs := []any{map[string]any{"@type": "ListItem", "position": 1, "name": "Home", "item": canonicalOrigin + "/"}}
	if view.Kind == "constellation" {
		crumbs = append(crumbs, map[string]any{"@type": "ListItem", "position": 2, "name": view.RegionName, "item": canonicalOrigin + fmt.Sprintf("/region/%d", view.RegionID)})
	}
	crumbs = append(crumbs, map[string]any{"@type": "ListItem", "position": len(crumbs) + 1, "name": view.Name, "item": canonicalOrigin + fmt.Sprintf("/%s/%d", view.Kind, view.ID)})
	return Metadata{Title: title, Description: description, CanonicalPath: path, Image: image, ImageAlt: view.Name, Indexable: true,
		StructuredData: map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": crumbs}}
}

func locationImage(view locationPageView, size int) string {
	return fmt.Sprintf("/images/%ss/%d?size=%d", view.Kind, view.ID, size)
}

func locationDominantName(value *api.UniverseLocationSovereignty) string {
	if value == nil {
		return ""
	}
	if value.AllianceID > 0 {
		return value.AllianceName
	}
	if value.FactionID > 0 {
		return value.FactionName
	}
	return ""
}

func locationDominantImage(value *api.UniverseLocationSovereignty, size int) string {
	if value == nil {
		return ""
	}
	if value.AllianceID > 0 {
		return eveEntityImage("alliance", value.AllianceID, size)
	}
	if value.FactionID > 0 {
		return eveEntityImage("corporation", value.FactionID, size)
	}
	return ""
}

func locationDominantHref(value *api.UniverseLocationSovereignty) string {
	if value != nil && value.AllianceID > 0 {
		return idPath("alliance", value.AllianceID)
	}
	return ""
}

func locationStatistics(view locationPageView) api.UniverseLocationStats {
	if view.Kind == "region" {
		return view.Region.Stats
	}
	return view.Constellation.Stats
}

func (s *site) registerLocationPages(mux *http.ServeMux) {
	for _, kind := range []string{"region", "constellation"} {
		load := func(r *http.Request) (Page, error) {
			id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
			if err != nil {
				return Page{}, ErrNotFound
			}
			view, err := s.loadLocationPage(r.Context(), kind, id, r.PathValue("tab"), true)
			if err != nil {
				return Page{}, err
			}
			return Page{Metadata: locationMetadata(view), Content: locationPage(view), SharedCacheTTL: 2 * time.Minute}, nil
		}
		mux.Handle("GET /"+kind+"/{id}", s.pageHandler(load))
		mux.Handle("GET /"+kind+"/{id}/{tab}", s.pageHandler(load))
	}
	mux.HandleFunc("GET /_fragments/location/{kind}/tab/{id}/{tab}", s.locationTabFragment)
	mux.HandleFunc("GET /_fragments/location/{kind}/kills/{id}", s.locationKillsFragment)
	mux.HandleFunc("GET /_fragments/location/{kind}/valuable/{id}", s.locationValuableFragment)
	mux.HandleFunc("GET /_fragments/location/{kind}/battles/{id}", s.locationBattlesFragment)
}

func (s *site) locationTabFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	view, err := s.loadLocationPage(r.Context(), r.PathValue("kind"), id, r.PathValue("tab"), false)
	if err != nil {
		if err == ErrNotFound {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Could not load location", http.StatusServiceUnavailable)
		}
		return
	}
	s.renderFragment(w, r, locationMain(view))
}

func (s *site) locationKillsFragment(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || kind != "region" && kind != "constellation" || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	limit, err := fragmentInt(r.URL.Query().Get("limit"), 50, 10, 100)
	page, pageErr := fragmentInt(r.URL.Query().Get("page"), 1, 1, 1_000_000)
	if err != nil || pageErr != nil {
		http.Error(w, "Invalid page or limit", http.StatusBadRequest)
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
	kills, err := s.locationKilllist(r.Context(), kind, id, limit, after)
	if err != nil {
		http.Error(w, "Could not load kills", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, killList(kills, killListOptions{Kind: kind, Page: page, Limit: limit}))
}

func (s *site) locationValuableFragment(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	category := r.URL.Query().Get("category")
	if err != nil || kind != "region" && kind != "constellation" || id < 1 || id > 2_147_483_647 || category != "kills" && category != "ships" && category != "structures" {
		http.NotFound(w, r)
		return
	}
	kills, err := s.options.Universe.UniverseLocationValuable(r.Context(), kind, id, category)
	if err != nil {
		http.Error(w, "Could not load valuable kills", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, mostValuable(kills, category, false))
}

func (s *site) locationBattlesFragment(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	page, pageErr := fragmentInt(r.URL.Query().Get("page"), 1, 1, 1_000_000)
	if err != nil || kind != "region" && kind != "constellation" || id < 1 || id > 2_147_483_647 || pageErr != nil || s.options.Entities == nil {
		http.NotFound(w, r)
		return
	}
	battles, err := s.options.Entities.EntityBattles(r.Context(), kind, id, page)
	if err != nil {
		http.Error(w, "Could not load battles", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, entityBattles(kind, id, battles))
}
