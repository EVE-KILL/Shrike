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
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

const battleDirectoryPageSize = 50

type battleDirectoryFilters struct {
	View       string
	Hours      int
	Year       int
	MinKills   int
	MinISK     string
	Custom     bool
	EntityType string
	EntityID   int64
	EntityName string
	Page       int
}

type battleDirectoryEntry struct {
	BattleID            int64                 `json:"battle_id"`
	SolarSystemID       int64                 `json:"solar_system_id"`
	SolarSystemName     string                `json:"solar_system_name"`
	SolarSystemSecurity *float64              `json:"solar_system_security"`
	RegionID            *int64                `json:"region_id"`
	RegionName          string                `json:"region_name"`
	StartTime           string                `json:"start_time"`
	DurationMinutes     int                   `json:"duration_minutes"`
	KillCount           int                   `json:"kill_count"`
	TotalISKDestroyed   float64               `json:"total_isk_destroyed"`
	IsMultiParty        bool                  `json:"is_multi_party"`
	IsCustom            bool                  `json:"is_custom"`
	Teams               []battleDirectoryTeam `json:"teams"`
}

type battleDirectoryTeam struct {
	TotalKills        int                         `json:"total_kills"`
	TotalISKDestroyed float64                     `json:"total_isk_destroyed"`
	AllianceCount     int                         `json:"alliance_count"`
	TopAlliances      []battleDirectoryTeamEntity `json:"top_alliances"`
}

type battleDirectoryTeamEntity struct {
	AllianceID      *int64 `json:"alliance_id"`
	AllianceName    string `json:"alliance_name"`
	CorporationID   *int64 `json:"corporation_id"`
	CorporationName string `json:"corporation_name"`
}

type battleDirectoryYear struct {
	Year  int `json:"year"`
	Count int `json:"count"`
}

type battleDirectoryData struct {
	Battles []battleDirectoryEntry `json:"battles"`
	Years   []battleDirectoryYear  `json:"years"`
	Page    int                    `json:"page"`
	Limit   int                    `json:"limit"`
}

type battleDirectoryView struct {
	Filters    battleDirectoryFilters
	Data       battleDirectoryData
	DomainMode bool
}

func parseBattleDirectoryFilters(values url.Values) (battleDirectoryFilters, error) {
	f := battleDirectoryFilters{View: "list", Page: 1}
	if view := values.Get("view"); view != "" {
		if view != "list" && view != "map" {
			return f, fmt.Errorf("invalid battle view")
		}
		f.View = view
	}
	parse := func(key string, maxValue int) (int, error) {
		value := values.Get(key)
		if value == "" {
			return 0, nil
		}
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 || parsed > maxValue {
			return 0, fmt.Errorf("invalid battle %s", key)
		}
		return parsed, nil
	}
	var err error
	if f.Page, err = parse("page", 1_000_000); err != nil {
		return f, err
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.Hours, err = parse("hours", 24*365); err != nil {
		return f, err
	}
	if f.Hours != 0 && f.Hours != 24 && f.Hours != 48 && f.Hours != 168 && f.Hours != 720 {
		return f, fmt.Errorf("invalid battle hours")
	}
	if f.Year, err = parse("year", 9999); err != nil {
		return f, err
	}
	if f.Year != 0 && f.Year < 2003 {
		return f, fmt.Errorf("invalid battle year")
	}
	if f.MinKills, err = parse("minKills", 200); err != nil {
		return f, err
	}
	if f.MinKills != 0 && f.MinKills != 10 && f.MinKills != 25 && f.MinKills != 50 && f.MinKills != 100 && f.MinKills != 200 {
		return f, fmt.Errorf("invalid battle minimum kills")
	}
	f.MinISK = values.Get("minIsk")
	if f.MinISK != "" && f.MinISK != "1b" && f.MinISK != "10b" && f.MinISK != "50b" && f.MinISK != "100b" && f.MinISK != "1t" {
		return f, fmt.Errorf("invalid battle minimum ISK")
	}
	if custom := values.Get("custom"); custom != "" && custom != "1" {
		return f, fmt.Errorf("invalid battle custom flag")
	} else {
		f.Custom = custom == "1"
	}
	f.EntityType = values.Get("eType")
	if f.EntityType != "" && f.EntityType != "alliance" && f.EntityType != "corporation" && f.EntityType != "system" && f.EntityType != "region" {
		return f, fmt.Errorf("invalid battle entity type")
	}
	if f.EntityType != "" {
		if f.EntityID, err = strconv.ParseInt(values.Get("eId"), 10, 64); err != nil || f.EntityID <= 0 || f.EntityID > math.MaxInt32 {
			return f, fmt.Errorf("invalid battle entity ID")
		}
	} else if values.Get("eId") != "" {
		return f, fmt.Errorf("battle entity ID requires a type")
	}
	return f, nil
}

func battleDirectoryAPIQuery(f battleDirectoryFilters) url.Values {
	query := url.Values{"page": {strconv.Itoa(f.Page)}, "limit": {strconv.Itoa(battleDirectoryPageSize)}}
	if f.Hours > 0 {
		query.Set("hours", strconv.Itoa(f.Hours))
	}
	if f.Year > 0 {
		query.Set("year", strconv.Itoa(f.Year))
	}
	if f.MinKills > 0 {
		query.Set("minKills", strconv.Itoa(f.MinKills))
	}
	if f.MinISK != "" {
		amount := map[string]string{"1b": "1000000000", "10b": "10000000000", "50b": "50000000000", "100b": "100000000000", "1t": "1000000000000"}[f.MinISK]
		query.Set("minIsk", amount)
	}
	if f.Custom {
		query.Set("custom", "true")
	}
	if f.EntityType != "" {
		key := map[string]string{"alliance": "allianceId", "corporation": "corporationId", "system": "systemId", "region": "regionId"}[f.EntityType]
		query.Set(key, strconv.FormatInt(f.EntityID, 10))
	}
	return query
}

func (s *site) battleDirectoryAPICall(r *http.Request, path string, query url.Values, out any) error {
	request := r.Clone(r.Context())
	address := *r.URL
	address.Path = path
	address.RawPath = ""
	address.RawQuery = query.Encode()
	request.URL = &address
	request.Method = http.MethodGet
	request.RequestURI = ""
	w := httptest.NewRecorder()
	s.options.ConflictsAPI.ServeHTTP(w, request)
	if w.Code != http.StatusOK {
		return fmt.Errorf("battle API returned %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		return fmt.Errorf("decode battle API: %w", err)
	}
	return nil
}

func (s *site) readBattleDirectory(r *http.Request, filters *battleDirectoryFilters) (battleDirectoryData, error) {
	if filters.EntityType != "" {
		var resolved struct {
			Name string `json:"name"`
		}
		q := url.Values{"type": {filters.EntityType}, "id": {strconv.FormatInt(filters.EntityID, 10)}}
		if s.battleDirectoryAPICall(r, "/api/entity/resolve", q, &resolved) == nil && resolved.Name != "" {
			filters.EntityName = resolved.Name
		} else {
			filters.EntityName = fmt.Sprintf("%s %d", strings.ToUpper(filters.EntityType[:1])+filters.EntityType[1:], filters.EntityID)
		}
	}
	var data battleDirectoryData
	if err := s.battleDirectoryAPICall(r, "/api/conflicts/battles", battleDirectoryAPIQuery(*filters), &data); err != nil {
		return data, err
	}
	if data.Battles == nil {
		data.Battles = []battleDirectoryEntry{}
	}
	return data, nil
}

func (s *site) battlesPage(r *http.Request) (Page, error) {
	filters, err := parseBattleDirectoryFilters(r.URL.Query())
	if err != nil {
		return Page{}, ErrNotFound
	}
	data, err := s.readBattleDirectory(r, &filters)
	if err != nil {
		return Page{}, err
	}
	return Page{Metadata: Metadata{
		Title: "EVE Online Battles & Battle Reports", CanonicalPath: "/battles", Indexable: true,
		Description: "Browse major battles in EVE Online — fleet engagements, battle reports, timelines, and ISK destroyed across New Eden on EVE-KILL.",
		Stylesheets: []string{"utilities.css", "utility-page.css", "battles.css", "map-canvas.css"}, ModuleScripts: []string{"battles.js"},
		StructuredData: map[string]any{"@context": "https://schema.org", "@type": "CollectionPage", "name": "EVE Online Battles", "url": canonicalOrigin + "/battles"},
	}, Content: battlesDirectoryPage(battleDirectoryView{Filters: filters, Data: data, DomainMode: s.battleDirectoryDomainMode(r)})}, nil
}

func (s *site) battleDirectoryDomainMode(r *http.Request) bool {
	// Host shape alone does not establish a configured custom killboard.
	// Reuse the cached bootstrap that Nuxt uses, preserving its null fallback.
	if s.options.Statistics != nil {
		config, err := s.options.Statistics.StatisticsSite(r.Context(), r.Host)
		return err == nil && config.Domain != nil
	}
	var config api.SiteConfigurationResponse
	return s.battleDirectoryAPICall(r, "/api/site", nil, &config) == nil && config.Domain != nil
}

func battleDirectoryEntityIcon(kind string) string {
	switch kind {
	case "alliance":
		return "shield"
	case "corporation":
		return "building-2"
	case "system":
		return "globe"
	case "region":
		return "map"
	default:
		return "search"
	}
}

func (s *site) battlesFragment(w http.ResponseWriter, r *http.Request) {
	filters, err := parseBattleDirectoryFilters(r.URL.Query())
	if err != nil {
		http.Error(w, "Invalid battle filters", http.StatusBadRequest)
		return
	}
	data, err := s.readBattleDirectory(r, &filters)
	if err != nil {
		http.Error(w, "Unable to load battles", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	s.renderFragment(w, r, battleDirectoryFragment(battleDirectoryView{Filters: filters, Data: data}))
}

func battleDirectoryHasFilters(f battleDirectoryFilters) bool {
	return f.Hours > 0 || f.Year > 0 || f.MinKills > 0 || f.MinISK != "" || f.Custom || f.EntityType != ""
}

func battleDirectoryDuration(minutes int) string {
	if minutes < 60 {
		return strconv.Itoa(minutes) + "m"
	}
	hours := minutes / 60
	if minutes%60 == 0 {
		return strconv.Itoa(hours) + "h"
	}
	return fmt.Sprintf("%dh %dm", hours, minutes%60)
}

func battleDirectorySecurity(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.1f", *value)
}

func battleDirectorySecurityClass(value *float64) string {
	if value == nil {
		return ""
	}
	if *value >= 0.5 {
		return "high"
	}
	if *value > 0 {
		return "low"
	}
	return "null"
}

func battleDirectoryTeamName(team battleDirectoryTeam, number int) string {
	if len(team.TopAlliances) == 0 {
		return fmt.Sprintf("Team %d", number)
	}
	entity := team.TopAlliances[0]
	if entity.AllianceName != "" {
		return entity.AllianceName
	}
	if entity.CorporationName != "" {
		return entity.CorporationName
	}
	return "Unknown"
}

func battleDirectoryTeamImage(entity battleDirectoryTeamEntity) string {
	if entity.AllianceID != nil {
		return eveEntityImage("alliance", *entity.AllianceID, 64)
	}
	if entity.CorporationID != nil {
		return eveEntityImage("corporation", *entity.CorporationID, 64)
	}
	return ""
}

func battleDirectoryISKShare(battle battleDirectoryEntry) int {
	if len(battle.Teams) != 2 {
		return 50
	}
	total := battle.Teams[0].TotalISKDestroyed + battle.Teams[1].TotalISKDestroyed
	if total <= 0 {
		return 50
	}
	return int(math.Round(battle.Teams[0].TotalISKDestroyed / total * 100))
}

func battleDirectoryDate(raw string) string {
	date, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return ""
	}
	return date.UTC().Format("Jan 2, 2006")
}
