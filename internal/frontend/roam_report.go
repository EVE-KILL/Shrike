package frontend

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

var roamTabs = []entityTab{
	{ID: "information", Label: "Information", Icon: "layout-dashboard"},
	{ID: "kills", Label: "Kills", Icon: "crosshair"},
	{ID: "systems", Label: "Systems", Icon: "map"},
	{ID: "intel", Label: "Intel", Icon: "scan-eye"},
	{ID: "composition", Label: "Composition", Icon: "users"},
}

type roamView struct {
	Data        api.RoamReport
	Tab         string
	Kills       api.KilllistPage
	KillOptions killListOptions
	KillRole    string
}

type roamFormView struct {
	Names, Start, End string
	Edit              bool
}
type roamMetric struct{ Label, Value, Detail, Icon, Tone string }

func roamPositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return min(value, 1000000)
}

func validRoamReportID(id string) bool {
	return len(id) == 32 && strings.Trim(id, "0123456789abcdef") == ""
}
func validRoamTab(tab string) bool {
	return member(tab, "information", "kills", "systems", "intel", "composition")
}
func roamTabPath(id, tab string) string {
	path := "/tools/roam-report/" + id
	if tab != "information" {
		path += "/" + tab
	}
	return path
}
func roamNumber(n int) string { return fullNumber(float64(n)) }
func roamDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2 Jan 2006, 15:04 EVE")
}
func roamClock(t time.Time) string { return t.UTC().Format("15:04") }
func roamPlural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
func roamForm(report api.RoamReport) roamFormView {
	names := make([]string, 0, len(report.Roster)+len(report.Unresolved))
	for _, pilot := range report.Roster {
		names = append(names, pilot.Name)
	}
	names = append(names, report.Unresolved...)
	return roamFormView{Names: strings.Join(names, "\n"), Start: report.StartTime.UTC().Format("2006-01-02T15:04"), End: report.EndTime.UTC().Format("2006-01-02T15:04"), Edit: true}
}

func roamMetadata(id, tab string) Metadata {
	meta := Metadata{Title: "Roam Report", OGTitle: "Roam Report — EVE-KILL", CanonicalPath: "/tools/roam-report", Indexable: true,
		Description: "Turn an EVE Online fleet list into a shareable report of recorded fights, losses, targets, and systems.",
		Stylesheets: []string{"utilities.css", "map-canvas.css", "roam-report.css"}, ModuleScripts: []string{"roam-report.js"}}
	if id != "" {
		meta.CanonicalPath = roamTabPath(id, tab)
		meta.Description = "A fleet combat report with recorded kills, systems, pilot composition and intel."
		if tab != "information" {
			for _, item := range roamTabs {
				if item.ID == tab {
					meta.Title += " — " + item.Label
				}
			}
		}
	}
	return meta
}

func (s *site) roamCreatePage(_ *http.Request) (Page, error) {
	end := time.Now().UTC().Truncate(time.Minute)
	form := roamFormView{Start: end.Add(-3 * time.Hour).Format("2006-01-02T15:04"), End: end.Format("2006-01-02T15:04")}
	return Page{Metadata: roamMetadata("", "information"), Content: roamCreatePage(form)}, nil
}

func roamReadStatus(err error) int {
	var status interface{ GetStatus() int }
	if errors.As(err, &status) {
		return status.GetStatus()
	}
	return http.StatusInternalServerError
}

func (s *site) loadRoamView(r *http.Request, id, tab string) (roamView, int, error) {
	view := roamView{Tab: tab}
	report, found, err := s.options.RoamReports.RoamReport(r.Context(), id)
	if err != nil {
		return view, roamReadStatus(err), err
	}
	if !found {
		return view, http.StatusNotFound, ErrNotFound
	}
	view.Data = report
	if view.Data.ID != id {
		return view, http.StatusBadGateway, fmt.Errorf("roam report ID did not match")
	}
	if tab == "kills" {
		role := r.URL.Query().Get("role")
		if !member(role, "kills", "losses", "all") {
			role = "kills"
		}
		view.KillRole = role
		page, limit := roamPositiveInt(r.URL.Query().Get("page"), 1), roamPositiveInt(r.URL.Query().Get("limit"), 50)
		if !member(strconv.Itoa(limit), "10", "25", "50", "100") {
			limit = 50
		}
		query := url.Values{"role": {role}, "page": {strconv.Itoa(page)}, "limit": {strconv.Itoa(limit)}}
		if cursor := r.URL.Query().Get("after"); cursor != "" {
			query.Set("after", cursor)
		}
		view.Kills, err = s.options.RoamReports.RoamReportKilllist(r.Context(), report, query)
		if err != nil {
			return view, roamReadStatus(err), err
		}
		losses := make(map[int64]bool, len(view.Data.Roster))
		for _, pilot := range view.Data.Roster {
			losses[int64(pilot.CharacterID)] = true
		}
		view.KillOptions = killListOptions{Kind: "roam", Page: page, Limit: limit, LossCharacters: losses}
	}
	return view, http.StatusOK, nil
}

func roamKillRole(view roamView) string {
	if view.KillRole == "" {
		return "kills"
	}
	return view.KillRole
}
func roamOrderedSystems(report api.RoamReport) []api.RoamSystem {
	rows := append([]api.RoamSystem{}, report.Systems...)
	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].FirstSeen.Equal(rows[j].FirstSeen) {
			return rows[i].FirstSeen.Before(rows[j].FirstSeen)
		}
		return rows[i].SolarSystemID < rows[j].SolarSystemID
	})
	return rows
}

func (s *site) roamDetailPage(r *http.Request) (Page, error) {
	id, tab := r.PathValue("id"), r.PathValue("tab")
	if tab == "" {
		tab = "information"
	}
	if !validRoamReportID(id) || !validRoamTab(tab) {
		return Page{}, ErrNotFound
	}
	view, status, err := s.loadRoamView(r, id, tab)
	if err != nil {
		if status == http.StatusOK {
			return Page{}, err
		}
		return Page{Status: status, Metadata: roamMetadata(id, tab), Content: roamUnavailable()}, nil
	}
	return Page{Metadata: roamMetadata(id, tab), Content: roamDetailPage(view)}, nil
}

func (s *site) roamFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	id, tab := r.PathValue("id"), r.PathValue("tab")
	whole := tab == "page"
	if whole {
		tab = r.URL.Query().Get("tab")
		if tab == "" {
			tab = "information"
		}
	}
	if !validRoamReportID(id) || !validRoamTab(tab) {
		http.NotFound(w, r)
		return
	}
	view, status, err := s.loadRoamView(r, id, tab)
	if err != nil {
		if status == http.StatusOK {
			status = http.StatusBadGateway
		}
		http.Error(w, "Unable to load Roam Report", status)
		return
	}
	if whole {
		s.renderFragment(w, r, roamDetailPage(view))
	} else {
		s.renderFragment(w, r, roamTabContent(view))
	}
}

func roamSortedPilots(report api.RoamReport) []api.RoamPilot {
	pilots := append([]api.RoamPilot{}, report.Roster...)
	sort.SliceStable(pilots, func(i, j int) bool {
		return pilots[i].KillParticipations+pilots[i].Losses > pilots[j].KillParticipations+pilots[j].Losses
	})
	return pilots
}
func roamBusiestSystem(report api.RoamReport) *api.RoamSystem {
	var busiest *api.RoamSystem
	for i := range report.Systems {
		item := &report.Systems[i]
		if busiest == nil || item.Kills+item.Losses > busiest.Kills+busiest.Losses {
			busiest = item
		}
	}
	return busiest
}
func roamMetrics(report api.RoamReport) []roamMetric {
	s := report.Summary
	return []roamMetric{
		{"Systems", roamNumber(s.Systems), roamNumber(s.Regions) + " regions", "map-pin", "text-blue-300"},
		{"Pilots", roamNumber(len(report.Roster)), roamNumber(s.ActivePilots) + " active", "users", "text-sky-300"},
		{"Kills", roamNumber(s.Kills), roamNumber(s.CoordinatedKills) + " coordinated", "crosshair", "text-emerald-300"},
		{"Losses", roamNumber(s.Losses), roamNumber(s.FriendlyFireLosses) + " friendly fire", "shield-x", "text-rose-300"},
		{"ISK destroyed", compactNumber(s.IskDestroyed), fmt.Sprintf("%.2f%% efficiency", s.Efficiency), "trending-up", "text-emerald-300"},
		{"ISK lost", compactNumber(s.IskLost), "Recorded losses", "trending-down", "text-rose-300"},
		{"Engagements", roamNumber(s.Engagements), "Recorded fights", "swords", "text-amber-300"},
	}
}
