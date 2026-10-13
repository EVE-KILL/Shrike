package frontend

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type fitCatalogStats struct {
	Families  int64 `json:"fittings_known"`
	Killmails int64 `json:"killmails_analyzed"`
	Community int64 `json:"community_fits"`
	Ratings   int64 `json:"ratings_cast"`
}
type fitCatalogFamily struct {
	api.UniverseFitFamily
	ShipID       int64    `json:"ship_type_id"`
	ShipName     string   `json:"ship_name"`
	RankingCount int64    `json:"ranking_count"`
	HullCost     *float64 `json:"hull_cost"`
}
type fitCatalogDoctrine struct {
	api.UniverseFitFamily
	EntityID    int64    `json:"entity_id"`
	EntityName  string   `json:"entity_name"`
	ShipID      int64    `json:"ship_type_id"`
	ShipName    string   `json:"ship_name"`
	TotalLosses int64    `json:"total_losses"`
	Uses        int64    `json:"doctrine_uses"`
	Share       float64  `json:"doctrine_share"`
	HullCost    *float64 `json:"hull_cost"`
}
type fitCatalogHull struct {
	ShipID   int64  `json:"ship_type_id"`
	ShipName string `json:"ship_name"`
	Uses     int64  `json:"total_uses"`
	Families int64  `json:"fit_count"`
}
type communityFitCard struct {
	ID        string    `json:"fit_id"`
	Name      string    `json:"name"`
	ShipID    int64     `json:"ship_type_id"`
	ShipName  string    `json:"ship_name"`
	OwnerID   int64     `json:"owner_character_id"`
	OwnerName string    `json:"owner_name"`
	Rating    *float64  `json:"rating_avg"`
	Ratings   int       `json:"rating_count"`
	Created   time.Time `json:"created_at"`
	Modules   int       `json:"module_count"`
}
type fitCatalogView struct {
	Flavor, Doctrine string
	Stats            fitCatalogStats
	Flavors          struct {
		Days     int                `json:"window_days"`
		Families []fitCatalogFamily `json:"families"`
	}
	Doctrines struct {
		Days    int                  `json:"window_days"`
		Entries []fitCatalogDoctrine `json:"doctrines"`
	}
	Popular struct {
		Ships []fitCatalogHull `json:"ships"`
	}
	Community struct {
		Fits []communityFitCard `json:"fits"`
	}
	TopRated struct {
		Fits []communityFitCard `json:"fits"`
	}
	Errors map[string]bool
	Now    time.Time
}

func fitCatalogSelection(q url.Values) (string, string) {
	flavor, doctrine := q.Get("flavor"), q.Get("doctrines")
	if flavor != "final_blows" && flavor != "losses" {
		flavor = "kills"
	}
	if doctrine != "corporation" {
		doctrine = "alliance"
	}
	return flavor, doctrine
}

// Consume the same cached catalogue APIs in-process, preserving their source
// queries, validation and invalidation. There are no browser JSON renderers.
func (s *site) fittingAPIRead(r *http.Request, path string, q url.Values, out any) error {
	request := r.Clone(r.Context())
	address := *r.URL
	address.Path, address.RawPath, address.RawQuery = path, "", q.Encode()
	request.URL, request.Method, request.RequestURI, request.Body, request.ContentLength = &address, http.MethodGet, "", nil, 0
	response := httptest.NewRecorder()
	s.options.FittingsAPI.ServeHTTP(response, request)
	if response.Code != 200 {
		return fittingAPIError{response.Code}
	}
	return json.Unmarshal(response.Body.Bytes(), out)
}

type fittingAPIError struct{ Status int }

func (e fittingAPIError) Error() string { return fmt.Sprintf("fitting API returned %d", e.Status) }

func (s *site) readFitCatalogPart(r *http.Request, view *fitCatalogView, part string) error {
	path, q, out := "", url.Values{}, any(nil)
	switch part {
	case "stats":
		path, out = "quick-stats", &view.Stats
	case "flavors":
		path, out = "flavors-of-the-week", &view.Flavors
		q.Set("mode", view.Flavor)
	case "doctrines":
		path, out = "top-alliance-doctrines", &view.Doctrines
		q.Set("entity_type", view.Doctrine)
	case "popular":
		path, out = "popular-ships", &view.Popular
		q.Set("mode", "kills")
	case "community":
		path, out = "community-latest", &view.Community
	case "top-rated":
		path, out = "top-rated", &view.TopRated
	default:
		return ErrNotFound
	}
	return s.fittingAPIRead(r, "/api/fits/"+path, q, out)
}

func (s *site) fitsCatalogPage(r *http.Request) (Page, error) {
	flavor, doctrine := fitCatalogSelection(r.URL.Query())
	view := fitCatalogView{Flavor: flavor, Doctrine: doctrine, Now: time.Now(), Errors: map[string]bool{}}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, part := range []string{"stats", "flavors", "doctrines", "popular", "community", "top-rated"} {
		wg.Add(1)
		go func(part string) {
			defer wg.Done()
			if s.readFitCatalogPart(r, &view, part) != nil {
				mu.Lock()
				view.Errors[part] = true
				mu.Unlock()
			}
		}(part)
	}
	wg.Wait()
	return Page{Metadata: Metadata{Title: "EVE Online Ship Fittings", Description: "Browse popular ship fittings from EVE Online killmails, share your own fits, and build new ones with the in-browser fitting tool.", CanonicalPath: "/fits", Indexable: true, Stylesheets: []string{"utilities.css", "utility-page.css", "fits-catalog.css"}, ModuleScripts: []string{"fits-catalog.js"}}, Content: fitsCatalogPage(view)}, nil
}

func (s *site) fitsCatalogFragment(w http.ResponseWriter, r *http.Request) {
	part := r.PathValue("part")
	flavor, doctrine := fitCatalogSelection(r.URL.Query())
	view := fitCatalogView{Flavor: flavor, Doctrine: doctrine, Now: time.Now(), Errors: map[string]bool{}}
	if err := s.readFitCatalogPart(r, &view, part); err != nil {
		if err == ErrNotFound {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Unable to load fittings", http.StatusServiceUnavailable)
		}
		return
	}
	s.renderFragment(w, r, fitCatalogPart(view, part))
}

func catalogShipName(name string, id int64) string {
	if name != "" {
		return name
	}
	return "Type " + strconv.FormatInt(id, 10)
}
func catalogCount(n int64) string {
	if n < 0 {
		return "—"
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fullNumber(float64(n))
}

func catalogStatDetail(n int64, detail string) string {
	if n < 0 {
		return "Count unavailable"
	}
	return detail
}
func catalogFlavorMetric(mode string) string {
	if mode == "final_blows" {
		return "final blows"
	}
	return mode
}
func catalogFlavorURL(view fitCatalogView, fit fitCatalogFamily) string {
	name := fit.ShipName
	if name == "" {
		name = "Community"
	}
	days := view.Flavors.Days
	if days == 0 {
		days = 7
	}
	return observedFitEditorURL(fit.ShipID, name+" Fit", fmt.Sprintf("Flavor of the week — %d %s in the last %d days. Representative fit observed on %d losses.", fit.RankingCount, catalogFlavorMetric(view.Flavor), days, fit.TotalUses), fit.Modules, fit.Drones)
}
func catalogDoctrineName(view fitCatalogView, fit fitCatalogDoctrine) string {
	if fit.EntityName != "" {
		return fit.EntityName
	}
	kind := "Alliance"
	if view.Doctrine == "corporation" {
		kind = "Corporation"
	}
	return fmt.Sprintf("%s %d", kind, fit.EntityID)
}
func catalogDoctrineURL(view fitCatalogView, fit fitCatalogDoctrine) string {
	name := fit.EntityName
	if name == "" {
		name = "Doctrine"
	}
	entity := fit.EntityName
	if entity == "" {
		entity = "Entity"
	}
	days := view.Doctrines.Days
	if days == 0 {
		days = 30
	}
	return observedFitEditorURL(fit.ShipID, strings.TrimSpace(name+" "+fit.ShipName), fmt.Sprintf("%s doctrine — %d losses (%s%% of their activity) in the last %d days.", entity, fit.Uses, strconv.FormatFloat(fit.Share, 'f', -1, 64), days), fit.Modules, fit.Drones)
}
func catalogShipRender(id int64, size int) string {
	return fmt.Sprintf("/images/types/%d/render?size=%d", id, size)
}
func catalogRatingOn(avg *float64, index int) bool {
	return avg != nil && int(math.Round(*avg)) >= index
}
func catalogRating(avg *float64) string {
	if avg == nil {
		return "–"
	}
	return fmt.Sprintf("%.1f", *avg)
}
func catalogAge(now, stamp time.Time) string {
	if stamp.IsZero() {
		return ""
	}
	mins := int(now.Sub(stamp).Minutes())
	if mins < 1 {
		return "just now"
	}
	if mins < 60 {
		return fmt.Sprintf("%dm ago", mins)
	}
	hours := mins / 60
	if hours < 24 {
		return fmt.Sprintf("%dh ago", hours)
	}
	days := hours / 24
	if days < 30 {
		return fmt.Sprintf("%dd ago", days)
	}
	months := days / 30
	if months < 12 {
		return fmt.Sprintf("%dmo ago", months)
	}
	return fmt.Sprintf("%dy ago", days/365)
}
