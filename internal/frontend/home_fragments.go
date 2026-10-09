package frontend

import (
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/eve-kill/shrike/internal/api"
)

func (s *site) mostValuableFragment(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	if kind != "kills" && kind != "ships" && kind != "structures" {
		http.Error(w, "Invalid kind", http.StatusBadRequest)
		return
	}
	items, err := s.options.Homepage.MostValuable(r.Context(), kind, 7, 8)
	if err != nil {
		http.Error(w, "Could not load valuable kills", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, mostValuable(items, kind, false))
}

func (s *site) killlistFragment(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	kind := query.Get("type")
	if kind == "" {
		kind = "latest"
	}
	if !validKillsPageType(kind) {
		http.Error(w, "Invalid killlist type", http.StatusBadRequest)
		return
	}
	page, err := fragmentInt(query.Get("page"), 1, 1, 1_000_000)
	if err != nil {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	if query.Get("fresh") == "1" && page != 1 {
		http.Error(w, "Fresh reload is only available for the first page", http.StatusBadRequest)
		return
	}
	limit, err := fragmentInt(query.Get("limit"), 50, 1, 100)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}
	filters := query.Get("filters")
	quickFilter := kind == "latest" && query.Get("quick") == "1"
	if filters != "" && !quickFilter {
		http.Error(w, "Filters are not available for this list", http.StatusBadRequest)
		return
	}
	options := killListOptions{Kind: kind, Page: page, Limit: limit, Live: true, QuickFilter: quickFilter}
	if filters != "" {
		options.Filtered, options.Live = true, false
		var after *int64
		if raw := query.Get("after"); raw != "" {
			value, parseErr := strconv.ParseInt(raw, 10, 64)
			if parseErr != nil || value <= 0 {
				http.Error(w, "Invalid cursor", http.StatusBadRequest)
				return
			}
			after = &value
		}
		result, err := s.options.Homepage.FilteredKills(r.Context(), filters, after, limit)
		if err != nil {
			http.Error(w, "Could not load filtered kills", http.StatusServiceUnavailable)
			return
		}
		s.renderFragment(w, r, killList(result, options))
		return
	}
	var result api.KilllistPage
	if query.Get("fresh") == "1" {
		result, err = s.options.Homepage.KilllistFresh(r.Context(), kind, page, limit)
	} else {
		result, err = s.options.Homepage.Killlist(r.Context(), kind, page, limit)
	}
	if err != nil {
		http.Error(w, "Could not load latest kills", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, killList(result, options))
}

func fragmentInt(raw string, fallback, minimum, maximum int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, strconv.ErrSyntax
	}
	return value, nil
}

func (s *site) renderFragment(w http.ResponseWriter, r *http.Request, component templ.Component) {
	buffer := templ.GetBuffer()
	defer templ.ReleaseBuffer(buffer)
	if err := component.Render(r.Context(), buffer); err != nil {
		http.Error(w, "Could not render section", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Content-Length", strconv.Itoa(buffer.Len()))
	if r.Method != http.MethodHead {
		_, _ = w.Write(buffer.Bytes())
	}
}
