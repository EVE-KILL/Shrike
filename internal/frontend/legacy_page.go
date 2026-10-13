package frontend

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"golang.org/x/sync/errgroup"
)

type legacyPageView struct {
	Stats    api.LegacyArchiveStats
	TopBoxes []topBoxView
	Kills    api.KilllistPage
	Year     int
	Sort     string
}

var legacyTopKinds = []struct{ Kind, Title string }{
	{"characters", "Top Victims"},
	{"corporations", "Top Corporations"},
	{"alliances", "Top Alliances"},
	{"ships", "Top Ships Lost"},
	{"systems", "Top Systems"},
}

func legacyButtonClass(active bool) string {
	if active {
		return "active"
	}
	return ""
}

func legacyYearClass(active bool) string { return legacyButtonClass(active) }

func legacyDateMin(kind string) string {
	if kind == "date" {
		return "2003-01-01"
	}
	return ""
}

func legacyDateMax(kind string) string {
	if kind == "date" {
		return "2007-12-31"
	}
	return ""
}

func (s *site) legacyPage(r *http.Request) (Page, error) {
	ctx := r.Context()
	var view legacyPageView
	view.TopBoxes = make([]topBoxView, len(legacyTopKinds))
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { var err error; view.Stats, err = s.options.Legacy.LegacyArchiveStats(ctx); return err })
	group.Go(func() error {
		var err error
		view.Kills, err = s.options.Legacy.LegacyArchiveKills(ctx, url.Values{"limit": {"50"}})
		return err
	})
	for i, kind := range legacyTopKinds {
		i, kind := i, kind
		group.Go(func() error {
			entries, err := s.options.Legacy.LegacyArchiveTop(ctx, kind.Kind, 0)
			if err != nil {
				return err
			}
			view.TopBoxes[i] = topBoxView{Title: kind.Title, DataType: kind.Kind, Entries: entries}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return Page{}, err
	}
	return Page{Metadata: Metadata{
		Title: "Legacy Kills", CanonicalPath: "/legacy", Indexable: true,
		Description: "Browse 436,000+ killmails from EVE Online's early days (2003–2007).",
		OGTitle:     "Legacy Kills — EVE-KILL", Stylesheets: []string{"legacy.css"}, ModuleScripts: []string{"legacy.js"},
	}, Content: legacyPage(view), SharedCacheTTL: time.Minute}, nil
}

func (s *site) legacyKillsFragment(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	limit, _ := strconv.Atoi(values.Get("limit"))
	if limit != 10 && limit != 25 && limit != 50 && limit != 100 {
		limit = 50
	}
	values.Set("limit", strconv.Itoa(limit))
	page, _ := strconv.Atoi(values.Get("page"))
	if page < 1 {
		page = 1
	}
	kills, err := s.options.Legacy.LegacyArchiveKills(r.Context(), values)
	if err != nil {
		http.Error(w, "Could not load legacy kills", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, killList(kills, killListOptions{Kind: "legacy", Page: page, Limit: limit}))
}

func (s *site) legacyTopFragment(w http.ResponseWriter, r *http.Request) {
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	if year < 2003 || year > 2007 {
		year = 0
	}
	boxes := make([]topBoxView, len(legacyTopKinds))
	group, ctx := errgroup.WithContext(r.Context())
	for i, kind := range legacyTopKinds {
		i, kind := i, kind
		group.Go(func() error {
			entries, err := s.options.Legacy.LegacyArchiveTop(ctx, kind.Kind, year)
			if err != nil {
				return err
			}
			boxes[i] = topBoxView{Title: kind.Title, DataType: kind.Kind, Entries: entries}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		http.Error(w, "Could not load legacy top lists", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, legacyTopSidebar(boxes, year))
}
