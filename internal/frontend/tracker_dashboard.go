package frontend

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

// Generated from the same catalog that Nuxt's configurator and widgets use.
//
//go:embed static/tracker-widgets.json
var trackerWidgetCatalogJSON []byte

type trackerWidgetDefinition struct{ Type, Title, Icon, DataType string }

var trackerWidgetCatalog = func() []trackerWidgetDefinition {
	var catalog []trackerWidgetDefinition
	if err := json.Unmarshal(trackerWidgetCatalogJSON, &catalog); err != nil {
		panic(err)
	}
	return catalog
}()

func trackerWidgetDefinitionFor(kind string) trackerWidgetDefinition {
	for _, definition := range trackerWidgetCatalog {
		if definition.Type == kind {
			return definition
		}
	}
	return trackerWidgetDefinition{}
}

type trackerDashboardConfig struct {
	Configured bool                  `json:"configured"`
	Widgets    api.SiteDomainWidgets `json:"widgets"`
}
type trackerDashboardSummary struct {
	Trackers int64      `json:"tracker_count"`
	Active   int64      `json:"active_tracker_count"`
	Kills    int64      `json:"kill_count"`
	Value    float64    `json:"total_value"`
	LastKill *time.Time `json:"last_kill_at"`
}
type trackerDashboardView struct {
	User                                     *settingsUser
	Config                                   trackerDashboardConfig
	Trackers                                 []entityTracker
	Limit                                    int
	Summary                                  trackerDashboardSummary
	ConfigError, TrackersError, SummaryError bool
}

func (s *site) loadTrackerDashboard(r *http.Request) (trackerDashboardView, error) {
	user, err := s.settingsIdentity(r)
	view := trackerDashboardView{User: user, Limit: 50}
	if err != nil || user == nil {
		return view, err
	}
	// Only the small account/configuration reads block the document. Each
	// aggregate widget has its own private HTML fragment and failure boundary.
	var group sync.WaitGroup
	group.Add(3)
	go func() {
		defer group.Done()
		view.ConfigError = s.accountRead(r, "/api/me/tracked/config", &view.Config) != nil
	}()
	go func() {
		defer group.Done()
		var data struct {
			Trackers []entityTracker `json:"trackers"`
			Limit    int             `json:"limit"`
		}
		view.TrackersError = s.accountRead(r, "/api/me/trackers", &data) != nil
		if !view.TrackersError {
			view.Trackers, view.Limit = data.Trackers, data.Limit
		}
	}()
	go func() {
		defer group.Done()
		view.SummaryError = s.accountRead(r, "/api/me/tracked/summary?days=7", &view.Summary) != nil
	}()
	group.Wait()
	return view, nil
}
func (s *site) trackerDashboardPage(r *http.Request) (Page, error) {
	view, err := s.loadTrackerDashboard(r)
	if err != nil {
		return Page{}, err
	}
	return Page{Metadata: Metadata{Title: "Tracker", CanonicalPath: "/tracker", Stylesheets: []string{"utility-page.css", "tracker-dashboard.css"}, ModuleScripts: []string{"tracker-dashboard.js"}}, Content: trackerDashboardPageContent(view)}, nil
}
func (s *site) trackerDashboardFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	view, err := s.loadTrackerDashboard(r)
	if err != nil {
		s.trackerFragmentError(w, err)
		return
	}
	if view.User == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	s.renderFragment(w, r, trackerDashboardBody(view))
}
func dashboardSection(widgets api.SiteDomainWidgets, section string) []api.SiteDomainWidget {
	switch section {
	case "top":
		return widgets.Top
	case "left":
		return widgets.Left
	default:
		return widgets.Right
	}
}
func dashboardHasSection(widgets api.SiteDomainWidgets, section string) bool {
	for _, widget := range dashboardSection(widgets, section) {
		if widget.Enabled {
			return true
		}
	}
	return false
}
func dashboardColumns(widgets api.SiteDomainWidgets) string {
	if !dashboardHasSection(widgets, "left") || !dashboardHasSection(widgets, "right") {
		return "--dashboard-columns:1fr"
	}
	ratio := "250px_1fr"
	if widgets.ColumnRatio != nil {
		switch *widgets.ColumnRatio {
		case "250px_1fr", "300px_1fr", "1fr_1fr", "1fr_2fr", "1fr_3fr":
			ratio = *widgets.ColumnRatio
		}
	}
	return "--dashboard-columns:" + strings.Replace(ratio, "_", " ", 1)
}
func dashboardKillType(widget api.SiteDomainWidget) string {
	if widget.KilllistType != nil && validKillsPageType(*widget.KilllistType) {
		return *widget.KilllistType
	}
	return "latest"
}
func (s *site) trackerDashboardStatsFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	definition := trackerWidgetDefinitionFor(r.PathValue("widget"))
	if definition.DataType == "" {
		http.NotFound(w, r)
		return
	}
	var data struct {
		Entries []api.TopEntry `json:"entries"`
	}
	if err := s.accountRead(r, "/api/me/tracked/stats?"+url.Values{"dataType": {definition.DataType}, "limit": {"10"}, "days": {"7"}}.Encode(), &data); err != nil {
		s.trackerFragmentError(w, err)
		return
	}
	s.renderFragment(w, r, topBox(topBoxView{Title: definition.Title, DataType: definition.DataType, Entries: data.Entries, ShowFooter: true}))
}
func (s *site) trackerDashboardValuableFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	kind := r.URL.Query().Get("kind")
	if kind != "kills" && kind != "ships" && kind != "structures" {
		http.Error(w, "Invalid kind", 400)
		return
	}
	var data struct {
		Entries []api.ValuableKill `json:"entries"`
	}
	if err := s.accountRead(r, "/api/me/tracked/stats?"+url.Values{"dataType": {"most_valuable_" + kind}, "limit": {"8"}, "days": {"7"}}.Encode(), &data); err != nil {
		s.trackerFragmentError(w, err)
		return
	}
	s.renderFragment(w, r, mostValuable(data.Entries, kind, false))
}
func (s *site) trackerDashboardKilllistFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	q := r.URL.Query()
	limit, err := fragmentInt(q.Get("limit"), 50, 10, 100)
	if err != nil {
		http.Error(w, "Invalid limit", 400)
		return
	}
	page, err := fragmentInt(q.Get("page"), 1, 1, 100000)
	if err != nil {
		http.Error(w, "Invalid page", 400)
		return
	}
	kind := q.Get("type")
	if kind == "" {
		kind = "latest"
	}
	if !validKillsPageType(kind) {
		http.Error(w, "Invalid killlist type", 400)
		return
	}
	params := url.Values{"limit": {strconv.Itoa(limit)}, "type": {kind}}
	if after := q.Get("after"); after != "" {
		params.Set("after", after)
	}
	var data api.KilllistPage
	if err := s.accountRead(r, "/api/me/tracked/killmails?"+params.Encode(), &data); err != nil {
		s.trackerFragmentError(w, err)
		return
	}
	s.renderFragment(w, r, killList(data, killListOptions{Kind: kind, Page: page, Limit: limit, Live: q.Get("live") == "1"}))
}
