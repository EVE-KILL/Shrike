package frontend

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type warsView struct {
	Tab, Sort                                string
	Page                                     int
	Mutual, HasActivity, HasKills, HasAllies bool
	DomainMode                               bool
	List                                     api.WarDirectoryData
	Overview                                 *api.WarOverviewData
}

func warsSelection(query url.Values) warsView {
	view := warsView{Tab: "wars", Sort: "recent", Page: 1}
	switch query.Get("tab") {
	case "finished", "upcoming", "eligible-corps", "eligible-alliances":
		view.Tab = query.Get("tab")
	}
	switch query.Get("sort") {
	case "kills", "isk":
		view.Sort = query.Get("sort")
	}
	if page, err := strconv.Atoi(query.Get("page")); err == nil && page > 0 && page <= 10000 {
		view.Page = page
	}
	view.Mutual = query.Get("mutual") == "true"
	view.HasActivity = query.Get("hasActivity") == "true"
	view.HasKills = query.Get("hasKills") == "true"
	view.HasAllies = query.Get("hasAllies") == "true"
	return view
}

func (v warsView) listQuery() url.Values {
	query := url.Values{"page": {strconv.Itoa(v.Page)}, "limit": {"50"}}
	if v.Tab == "upcoming" {
		query.Set("upcoming", "true")
		return query
	}
	if v.Tab == "finished" {
		query.Set("finished", "true")
	} else {
		query.Set("ongoing", "true")
	}
	query.Set("sort", v.Sort)
	for key, enabled := range map[string]bool{"mutual": v.Mutual, "hasActivity": v.HasActivity, "hasKills": v.HasKills, "hasAllies": v.HasAllies} {
		if enabled {
			query.Set(key, "true")
		}
	}
	return query
}

func warsDomainMode(host string) bool {
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	switch host {
	case "", "eve-kill.com", "www.eve-kill.com", "localhost", "127.0.0.1", "::1":
		return false
	}
	return strings.Contains(host, ".")
}

func (s *site) warsPage(r *http.Request) (Page, error) {
	view := warsSelection(r.URL.Query())
	view.DomainMode = warsDomainMode(r.Host)
	var overview api.WarOverviewData
	var overviewErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); overview, overviewErr = s.options.Wars.WarOverview(r.Context()) }()
	if view.Tab != "eligible-corps" && view.Tab != "eligible-alliances" {
		list, err := s.options.Wars.WarDirectory(r.Context(), view.listQuery(), r.Host)
		if err != nil {
			wg.Wait()
			return Page{}, err
		}
		view.List = list
	}
	wg.Wait()
	if overviewErr == nil {
		view.Overview = &overview
	}
	return Page{Metadata: Metadata{
		Title: "EVE Online Wars", OGTitle: "Wars — EVE-KILL", CanonicalPath: "/wars", Indexable: true,
		Description: "Browse active and historical wars in EVE Online — war declarations, faction warfare statistics, and conflict tracking on EVE-KILL.",
		Stylesheets: []string{"wars.css"}, ModuleScripts: []string{"wars.js"},
	}, Content: warsPageContent(view), SharedCacheTTL: time.Minute}, nil
}

// The global faction aggregates have a different cost and cache lifetime from
// the war directory. Their cold load must never hold up the document or its tabs.
func (s *site) warsFactionFragment(w http.ResponseWriter, r *http.Request) {
	if warsDomainMode(r.Host) {
		http.NotFound(w, r)
		return
	}
	data, err := s.options.Wars.FactionWars(r.Context())
	if err != nil {
		http.Error(w, "Could not load faction warfare", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	s.renderFragment(w, r, warsFactionCards(data))
}

func warEntityImage(side api.EntityWarSide) string { return eveEntityImage(side.Type, side.ID, 64) }
func warID(id int64) string                        { return strconv.FormatInt(id, 10) }
func warNumber(value int64) string                 { return fullNumber(float64(value)) }
func warISK(value float64) string                  { return compactNumber(value) }

func warTimeAgo(raw string) string {
	if raw == "" {
		return ""
	}
	when, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return ""
	}
	diff := time.Since(when)
	future := diff < 0
	if future {
		diff = -diff
	}
	minutes := int(diff.Minutes())
	value, unit := minutes, "m"
	switch {
	case minutes < 60:
	case minutes < 24*60:
		value, unit = minutes/60, "h"
	case minutes < 30*24*60:
		value, unit = minutes/(24*60), "d"
	case minutes < 365*24*60:
		value, unit = minutes/(30*24*60), "mo"
	default:
		value, unit = minutes/(365*24*60), "y"
	}
	if future {
		return fmt.Sprintf("in %d%s", value, unit)
	}
	return fmt.Sprintf("%d%s ago", value, unit)
}

func warUpcoming(raw string) bool {
	when, err := time.Parse(time.RFC3339Nano, raw)
	return err == nil && when.After(time.Now())
}

func warEfficiency(kills, losses int64) string {
	if kills+losses == 0 {
		return "50%"
	}
	return strconv.FormatInt(int64(float64(kills)/float64(kills+losses)*100+0.5), 10) + "%"
}

func warTabClass(active, tab string) string {
	if active == tab {
		return "active"
	}
	return ""
}

func warFilterClass(active bool) string {
	if active {
		return "active"
	}
	return ""
}

func warEligibleType(tab string) string {
	if tab == "eligible-alliances" {
		return "alliances"
	}
	return "corporations"
}

func warStartedLabel(tab string) string {
	if tab == "upcoming" {
		return "Starts"
	}
	return "Started"
}

func warEmptyText(tab string) string {
	switch tab {
	case "upcoming":
		return "No upcoming wars"
	case "finished":
		return "No finished wars found"
	default:
		return "No active wars found"
	}
}
