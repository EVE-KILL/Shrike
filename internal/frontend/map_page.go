package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
)

type mapPageView struct {
	Scope           string
	Base            string
	Activity        string
	Hours           int
	ShowConnections bool
	ShowSystems     bool
	ShowLabels      bool
	ShowChanges     bool
}

type mapRegionPageView struct {
	MapPageView mapPageView
	RegionID    int64
	RegionName  string
	SystemCount int
	Data        json.RawMessage
}

func parseMapPageView(r *http.Request) mapPageView {
	query := r.URL.Query()
	view := mapPageView{Scope: "new-eden", Base: "geography", Activity: "none", Hours: 24,
		ShowConnections: true, ShowSystems: true, ShowLabels: true}
	if member(query.Get("scope"), "new-eden", "zarzakh", "wormhole", "abyssal", "proving") {
		view.Scope = query.Get("scope")
	}
	legacy := query.Get("layer")
	if member(query.Get("base"), "geography", "security", "sovereignty", "live", "aiid") {
		view.Base = query.Get("base")
	} else if legacy == "security" {
		view.Base = "security"
	}
	if view.Scope != "new-eden" && member(view.Base, "sovereignty", "live", "aiid") {
		view.Base = "geography"
	}
	if member(query.Get("activity"), "none", "activity", "danger", "ship-kills", "pod-kills", "npc-kills", "jumps") {
		view.Activity = query.Get("activity")
	} else if member(legacy, "ship-kills", "pod-kills", "npc-kills", "jumps") {
		view.Activity = legacy
	}
	if hours, err := strconv.Atoi(query.Get("hours")); err == nil {
		switch hours {
		case 1, 6, 24, 168:
			view.Hours = hours
		}
	}
	view.ShowConnections = query.Get("routes") != "0"
	view.ShowSystems = query.Get("systems") != "0"
	view.ShowLabels = query.Get("labels") != "0"
	view.ShowChanges = query.Get("changes") == "1"
	return view
}

func (s *site) mapPage(r *http.Request) (Page, error) {
	view := parseMapPageView(r)
	return Page{Metadata: Metadata{
		Title: "Map", CanonicalPath: "/map", Indexable: true,
		Description:    "Interactive map of New Eden and the rest of EVE Online — explore geography, security, activity, danger, and traffic.",
		Stylesheets:    []string{"map-canvas.css", "map-page.css"},
		ModuleScripts:  []string{"map-page.js"},
		StructuredData: map[string]any{"@context": "https://schema.org", "@type": "WebPage", "name": "New Eden Map", "url": canonicalOrigin + "/map"},
	}, Content: mapIndexPage(view)}, nil
}

func (s *site) mapRegionPage(r *http.Request) (Page, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id <= 0 {
		return Page{}, ErrNotFound
	}
	view := parseMapPageView(r)
	if view.Base != "security" {
		view.Base = "geography"
	}
	request := r.Clone(r.Context())
	address := *r.URL
	address.Path = fmt.Sprintf("/api/map/region/%d", id)
	address.RawPath = ""
	address.RawQuery = url.Values{"hours": {strconv.Itoa(view.Hours)}}.Encode()
	request.URL = &address
	request.Method = http.MethodGet
	request.RequestURI = ""
	w := httptest.NewRecorder()
	s.options.ConflictsAPI.ServeHTTP(w, request)
	if w.Code == http.StatusNotFound {
		return Page{}, ErrNotFound
	}
	if w.Code != http.StatusOK {
		return Page{}, fmt.Errorf("region map API returned %d", w.Code)
	}
	var details struct {
		Region struct {
			Name string `json:"name"`
		} `json:"region"`
		Systems []json.RawMessage `json:"systems"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &details); err != nil {
		return Page{}, fmt.Errorf("decode region map: %w", err)
	}
	if details.Region.Name == "" {
		return Page{}, ErrNotFound
	}
	content := mapRegionPageView{MapPageView: view, RegionID: id, RegionName: details.Region.Name, SystemCount: len(details.Systems), Data: append(json.RawMessage(nil), w.Body.Bytes()...)}
	return Page{Metadata: Metadata{
		Title: details.Region.Name + " Map", CanonicalPath: fmt.Sprintf("/map/region/%d", id), Indexable: true,
		Description:    "Interactive map of " + details.Region.Name + " — all systems and jump connections.",
		Stylesheets:    []string{"region-map.css"},
		ModuleScripts:  []string{"region-map.js"},
		StructuredData: map[string]any{"@context": "https://schema.org", "@type": "WebPage", "name": details.Region.Name + " Map", "url": fmt.Sprintf("%s/map/region/%d", canonicalOrigin, id)},
	}, Content: regionMapPage(content)}, nil
}

func regionMapBackURL(view mapRegionPageView) string {
	query := url.Values{"base": {view.MapPageView.Base}, "activity": {view.MapPageView.Activity}, "hours": {strconv.Itoa(view.MapPageView.Hours)}}
	if !view.MapPageView.ShowConnections {
		query.Set("routes", "0")
	}
	if !view.MapPageView.ShowSystems {
		query.Set("systems", "0")
	}
	if !view.MapPageView.ShowLabels {
		query.Set("labels", "0")
	}
	return "/map?" + query.Encode()
}
