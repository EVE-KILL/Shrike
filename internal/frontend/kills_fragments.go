package frontend

import "net/http"

func (s *site) killsTopFragment(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("type")
	dataType := r.URL.Query().Get("dataType")
	title := topKillsTitle(kind, dataType)
	if !validKillsPageType(kind) || title == "" {
		http.Error(w, "Invalid top list", http.StatusBadRequest)
		return
	}
	entries, err := s.options.Homepage.TopKills(r.Context(), kind, dataType, 7, 10)
	if err != nil {
		http.Error(w, "Could not load top list", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, topBox(topBoxView{
		Title: title, DataType: dataType, Limit: 10, Entries: entries, ShowFooter: true,
	}))
}
