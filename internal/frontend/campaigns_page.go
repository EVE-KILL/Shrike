package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/api"
)

const campaignDirectoryPageSize = 24

type campaignDirectoryCounts struct {
	Active   int `json:"active"`
	Archived int `json:"archived"`
	Private  int `json:"private"`
}

type campaignDirectoryData struct {
	Campaigns []api.EntityCampaignEntry `json:"campaigns"`
	HasMore   bool                      `json:"hasMore"`
	Page      int                       `json:"page"`
	Total     int                       `json:"total"`
	Counts    campaignDirectoryCounts   `json:"counts"`
}

type campaignDirectoryFilters struct {
	Status string
	Mode   string
	Query  string
	Page   int
}

type campaignDirectoryView struct {
	Filters    campaignDirectoryFilters
	Data       campaignDirectoryData
	DomainMode bool
}

func parseCampaignDirectoryFilters(values url.Values) (campaignDirectoryFilters, error) {
	filters := campaignDirectoryFilters{Status: "active", Mode: "all", Page: 1}
	if status := values.Get("status"); status != "" {
		if status != "active" && status != "archived" && status != "private" {
			return filters, fmt.Errorf("invalid campaign status")
		}
		filters.Status = status
	}
	if mode := values.Get("mode"); mode != "" {
		if mode != "all" && mode != "conflict" && mode != "area" {
			return filters, fmt.Errorf("invalid campaign mode")
		}
		filters.Mode = mode
	}
	filters.Query = strings.TrimSpace(values.Get("q"))
	if runes := []rune(filters.Query); len(runes) > 100 {
		filters.Query = string(runes[:100])
	}
	if raw := values.Get("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil || page < 1 || page > 1_000_000 {
			return filters, fmt.Errorf("invalid campaign page")
		}
		filters.Page = page
	}
	return filters, nil
}

func (s *site) readCampaignDirectory(r *http.Request, filters campaignDirectoryFilters) (campaignDirectoryData, error) {
	request := r.Clone(r.Context())
	address := *r.URL
	address.Path = "/api/campaigns"
	address.RawPath = ""
	query := url.Values{"status": {filters.Status}, "page": {strconv.Itoa(filters.Page)}}
	if filters.Mode != "all" {
		query.Set("mode", filters.Mode)
	}
	if filters.Query != "" {
		query.Set("q", filters.Query)
	}
	address.RawQuery = query.Encode()
	request.URL = &address
	request.Method = http.MethodGet
	request.RequestURI = ""
	w := httptest.NewRecorder()
	s.options.CampaignsAPI.ServeHTTP(w, request)
	if w.Code != http.StatusOK {
		return campaignDirectoryData{}, fmt.Errorf("campaigns API returned %d", w.Code)
	}
	var data campaignDirectoryData
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		return campaignDirectoryData{}, fmt.Errorf("decode campaign directory: %w", err)
	}
	if data.Campaigns == nil {
		data.Campaigns = []api.EntityCampaignEntry{}
	}
	return data, nil
}

func (s *site) campaignsPage(r *http.Request) (Page, error) {
	filters, err := parseCampaignDirectoryFilters(r.URL.Query())
	if err != nil {
		return Page{}, ErrNotFound
	}
	data, err := s.readCampaignDirectory(r, filters)
	if err != nil {
		return Page{}, err
	}
	return Page{Metadata: Metadata{
		Title: "EVE Online Campaigns", CanonicalPath: "/campaigns", Indexable: true,
		Description:    "Long-running conflict trackers on EVE-KILL — kills, losses and ISK destroyed for ongoing and historical wars in EVE Online.",
		Stylesheets:    []string{"entity.css", "campaigns.css"},
		ModuleScripts:  []string{"campaigns.js"},
		StructuredData: map[string]any{"@context": "https://schema.org", "@type": "CollectionPage", "name": "EVE Online Campaigns", "url": canonicalOrigin + "/campaigns"},
	}, Content: campaignsDirectoryPage(campaignDirectoryView{Filters: filters, Data: data, DomainMode: warsDomainMode(r.Host)})}, nil
}

func (s *site) campaignsFragment(w http.ResponseWriter, r *http.Request) {
	filters, err := parseCampaignDirectoryFilters(r.URL.Query())
	if err != nil {
		http.Error(w, "Invalid campaign filters", http.StatusBadRequest)
		return
	}
	data, err := s.readCampaignDirectory(r, filters)
	if err != nil {
		http.Error(w, "Could not load campaigns", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	s.renderFragment(w, r, campaignsDirectoryResults(campaignDirectoryView{Filters: filters, Data: data}))
}

func campaignLastPage(total int) int {
	return max(1, (total+campaignDirectoryPageSize-1)/campaignDirectoryPageSize)
}
