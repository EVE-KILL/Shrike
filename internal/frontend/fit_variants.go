package frontend

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// The expanded editor loads observed fits after its interactive workspace is
// ready. Reuse the same direct reader and classifier as the ship fittings tab.
func (s *site) fitVariantsFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	item, found, err := s.options.Universe.UniverseItem(r.Context(), id)
	if err != nil {
		http.Error(w, "Could not load ship", http.StatusServiceUnavailable)
		return
	}
	if !found || !item.Item.IsShip {
		http.NotFound(w, r)
		return
	}
	families, err := s.options.Universe.UniverseFitFamilies(r.Context(), id, url.Values{})
	if err != nil {
		http.Error(w, "Could not load observed fits", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, fitVariantsList(item.Item, families))
}

func fitFamilySearchURL(hash string, days int, fits bool) string {
	if days < 1 {
		days = 90
	}
	query := url.Values{
		"q":  {fmt.Sprintf(`{"timeRange":{"preset":"%dd"}}`, days)},
		"dh": {hash},
		"dm": {"family"},
	}
	if fits {
		query.Set("view", "fits")
		query.Set("dedup", "exact")
	}
	return "/advancedsearch?" + query.Encode()
}

func fitObservedAgo(lastUsed time.Time) string {
	days := int(time.Since(lastUsed).Hours() / 24)
	if days <= 0 {
		return "today"
	}
	return fmt.Sprintf("%dd ago", days)
}
