package frontend

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type fitSearchShip struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type fitRequirement struct {
	Kind     string `json:"kind"`
	RoleID   string `json:"role_id,omitempty"`
	TypeID   int64  `json:"type_id,omitempty"`
	TypeName string `json:"type_name,omitempty"`
	Op       string `json:"op"`
	Count    int    `json:"count"`
}
type fitStatisticFilter struct {
	Metric     string   `json:"metric"`
	Min        *float64 `json:"min"`
	Max        *float64 `json:"max"`
	NPCProfile string   `json:"npcProfile,omitempty"`
}
type fitSearchState struct {
	Ship       *fitSearchShip       `json:"ship"`
	Filters    []fitRequirement     `json:"filters"`
	Statistics []fitStatisticFilter `json:"statistics"`
	Sort       string               `json:"sort"`
	CapStable  bool                 `json:"capStable"`
}
type fitSearchRole struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Category    string `json:"category"`
	TypeCount   int    `json:"typeCount"`
}
type fitSearchStatistics struct {
	DPS           *float64 `json:"dps_with_reload"`
	EHP           *float64 `json:"ehp"`
	ShieldRepair  *float64 `json:"shield_effective_boost"`
	ArmorRepair   *float64 `json:"armor_effective_repair"`
	HullRepair    *float64 `json:"hull_effective_repair"`
	PassiveShield *float64 `json:"passive_shield_effective"`
	Speed         *float64 `json:"max_velocity"`
	NPCEHP        *float64 `json:"npc_ehp"`
	NPCProfile    string   `json:"npc_profile"`
	CapStable     bool     `json:"cap_stable"`
}
type fitSearchResult struct {
	api.UniverseFitFamily
	Hash       string               `json:"fit_hash"`
	ShipID     int64                `json:"ship_type_id"`
	ShipName   string               `json:"ship_name"`
	FamilyUses int64                `json:"family_total_uses"`
	HullCost   *float64             `json:"hull_cost"`
	Statistics *fitSearchStatistics `json:"stats"`
}
type fitSearchResponse struct {
	ShipID   int64             `json:"ship_type_id"`
	ShipName string            `json:"ship_name"`
	Total    int64             `json:"total"`
	More     bool              `json:"has_more"`
	Offset   int               `json:"offset"`
	Fits     []fitSearchResult `json:"fits"`
}
type fitSearchView struct {
	State                    fitSearchState
	Roles                    []fitSearchRole
	Popular                  []fitCatalogHull
	PopularDays              int
	RolesError, PopularError bool
}

var fitSearchSorts = []struct{ Key, Label string }{{"uses", "Most used"}, {"ehp", "Highest EHP"}, {"dps", "Highest DPS"}, {"alpha", "Highest alpha"}, {"repair", "Highest repair"}, {"speed", "Highest speed"}, {"align", "Fastest align"}, {"npc_ehp", "Highest NPC EHP"}}

func fitSearchInitialState(q url.Values) fitSearchState {
	state := fitSearchState{Filters: []fitRequirement{}, Statistics: []fitStatisticFilter{}, Sort: "uses", CapStable: q.Get("cap_stable") == "true"}
	if id, err := strconv.ParseInt(q.Get("ship"), 10, 32); err == nil && id > 0 {
		state.Ship = &fitSearchShip{id, catalogShipName(q.Get("ship_name"), id)}
	}
	for _, option := range fitSearchSorts {
		if option.Key == q.Get("sort") {
			state.Sort = option.Key
		}
	}
	var raw []json.RawMessage
	if len(q.Get("filters")) <= 8192 && json.Unmarshal([]byte(q.Get("filters")), &raw) == nil {
		for _, document := range raw {
			var filter fitRequirement
			var fields map[string]json.RawMessage
			if json.Unmarshal(document, &fields) != nil || fields["count"] == nil || string(fields["count"]) == "null" || json.Unmarshal(document, &filter) != nil || filter.Count < 0 || filter.Count > 16 || !slices.Contains([]string{">=", "<=", "=", ">", "<"}, filter.Op) {
				continue
			}
			if filter.RoleID != "" {
				filter.Kind = "role"
				filter.TypeID = 0
				filter.TypeName = ""
			} else if filter.TypeID > 0 && filter.TypeID <= math.MaxInt32 {
				filter.Kind = "type"
			} else {
				continue
			}
			state.Filters = append(state.Filters, filter)
			if len(state.Filters) == 8 {
				break
			}
		}
	}
	for _, metric := range fitMetricLabels {
		min, max := fitSearchQueryNumber(q.Get("min_"+metric.Key)), fitSearchQueryNumber(q.Get("max_"+metric.Key))
		profile := ""
		if metric.Key == "npc_ehp" {
			profile = q.Get("npc_profile")
		}
		if min != nil || max != nil || profile != "" {
			state.Statistics = append(state.Statistics, fitStatisticFilter{metric.Key, min, max, profile})
		}
	}
	return state
}
func fitSearchQueryNumber(raw string) *float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || value < 0 || math.IsInf(value, 0) || math.IsNaN(value) {
		return nil
	}
	return &value
}
func (state fitSearchState) query(sort bool) url.Values {
	q := url.Values{}
	if state.Ship != nil {
		q.Set("ship", strconv.FormatInt(state.Ship.ID, 10))
	}
	if len(state.Filters) > 0 {
		body, _ := json.Marshal(state.Filters)
		q.Set("filters", string(body))
	}
	for _, filter := range state.Statistics {
		if filter.Min != nil {
			q.Set("min_"+filter.Metric, strconv.FormatFloat(*filter.Min, 'f', -1, 64))
		}
		if filter.Max != nil {
			q.Set("max_"+filter.Metric, strconv.FormatFloat(*filter.Max, 'f', -1, 64))
		}
		if filter.Metric == "npc_ehp" && filter.NPCProfile != "" {
			q.Set("npc_profile", filter.NPCProfile)
		}
	}
	if state.CapStable {
		q.Set("cap_stable", "true")
	}
	if sort {
		q.Set("sort", state.Sort)
	}
	return q
}

func (s *site) fitsSearchPage(r *http.Request) (Page, error) {
	view := fitSearchView{State: fitSearchInitialState(r.URL.Query())}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		var data struct {
			Roles []fitSearchRole `json:"roles"`
		}
		view.RolesError = s.fittingAPIRead(r, "/api/fits/roles", url.Values{}, &data) != nil
		view.Roles = data.Roles
	}()
	go func() {
		defer wg.Done()
		var data struct {
			Ships []fitCatalogHull `json:"ships"`
			Days  int              `json:"window_days"`
		}
		view.PopularError = s.fittingAPIRead(r, "/api/fits/popular-ships", url.Values{}, &data) != nil
		view.Popular = data.Ships
		if len(view.Popular) > 12 {
			view.Popular = view.Popular[:12]
		}
		view.PopularDays = data.Days
		if view.PopularDays == 0 {
			view.PopularDays = 30
		}
	}()
	wg.Wait()
	return Page{Metadata: Metadata{Title: "Search EVE Online Ship Fits", OGTitle: "Search Fits — EVE-KILL", Description: "Search EVE Online ship fittings by hull and modules. Filter by armor reps, missile launchers, tackle and more — see real fits flown over the last 90 days.", CanonicalPath: "/fits/search", Indexable: true, Stylesheets: []string{"utilities.css", "utility-page.css", "fits-search.css"}, ModuleScripts: []string{"fits-search.js"}}, Content: fitsSearchPage(view)}, nil
}

func (s *site) fitsSearchFragment(w http.ResponseWriter, r *http.Request) {
	state := fitSearchInitialState(r.URL.Query())
	part := r.PathValue("part")
	if part == "popular" {
		var view fitSearchView
		var data struct {
			Ships []fitCatalogHull `json:"ships"`
			Days  int              `json:"window_days"`
		}
		if s.fittingAPIRead(r, "/api/fits/popular-ships", url.Values{}, &data) != nil {
			http.Error(w, "Unable to load popular hulls", http.StatusServiceUnavailable)
			return
		}
		view.Popular = data.Ships
		if len(view.Popular) > 12 {
			view.Popular = view.Popular[:12]
		}
		view.PopularDays = data.Days
		s.renderFragment(w, r, fitSearchPopular(view))
		return
	}
	if state.Ship == nil {
		http.Error(w, "Choose a ship", 400)
		return
	}
	q := state.query(part == "results")
	switch part {
	case "results":
		q.Set("limit", "24")
		offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 32)
		if offset < 0 {
			offset = 0
		}
		q.Set("offset", strconv.FormatInt(offset, 10))
		var data fitSearchResponse
		if err := s.fittingAPIRead(r, "/api/fits/search", q, &data); err != nil {
			http.Error(w, "Unable to load fitting results", http.StatusServiceUnavailable)
			return
		}
		s.renderFragment(w, r, fitSearchResults(data, time.Now()))
	case "distributions":
		var data api.UniverseFitDistributions
		if s.fittingAPIRead(r, "/api/fits/search/distributions", q, &data) != nil {
			http.Error(w, "Unable to load fit profiles", http.StatusServiceUnavailable)
			return
		}
		s.renderFragment(w, r, fitSearchProfiles(data, state))
	default:
		http.NotFound(w, r)
	}
}

func fitSearchRoleName(roles []fitSearchRole, filter fitRequirement) string {
	if filter.Kind == "type" {
		return catalogShipName(filter.TypeName, filter.TypeID)
	}
	for _, role := range roles {
		if role.ID == filter.RoleID {
			return role.Label
		}
	}
	return filter.RoleID
}
func fitSearchRoleIcon(roles []fitSearchRole, filter fitRequirement) string {
	for _, role := range roles {
		if role.ID == filter.RoleID {
			return strings.TrimPrefix(role.Icon, "lucide:")
		}
	}
	return "circle"
}
func fitSearchEditorURL(fit fitSearchResult) string {
	name := fit.ShipName
	if name == "" {
		name = "Community"
	}
	return observedFitEditorURL(fit.ShipID, name+" Fit", fmt.Sprintf("Killmail-derived fit — %d recorded uses in the last 90 days.", fit.TotalUses), fit.Modules, fit.Drones)
}
func fitSearchRepair(stats *fitSearchStatistics) float64 {
	if stats == nil {
		return 0
	}
	result := float64(0)
	for _, v := range []*float64{stats.ShieldRepair, stats.ArmorRepair, stats.HullRepair, stats.PassiveShield} {
		if v != nil {
			result = math.Max(result, *v)
		}
	}
	return result
}
func fitSearchInsights(fit fitSearchResult) []string {
	parts := fitContextParts(fit.Context)
	if len(parts) > 3 {
		parts = parts[:3]
	}
	return parts
}
func fitSearchNumber(value float64) string {
	raw := strconv.FormatFloat(math.Round(value*1000)/1000, 'f', -1, 64)
	parts := strings.SplitN(raw, ".", 2)
	integer, _ := strconv.ParseFloat(parts[0], 64)
	result := fullNumber(integer)
	if len(parts) > 1 {
		result += "." + parts[1]
	}
	return result
}
func fitSearchCompact(value float64) string {
	for _, unit := range []struct {
		Threshold float64
		Suffix    string
	}{{1e12, "T"}, {1e9, "B"}, {1e6, "M"}, {1e3, "K"}} {
		if math.Round(value/unit.Threshold*10)/10 >= 1 {
			return strconv.FormatFloat(math.Round(value/unit.Threshold*10)/10, 'f', -1, 64) + unit.Suffix
		}
	}
	return fitSearchNumber(value)
}
func fitSearchStatLabel(filter fitStatisticFilter) string {
	label := filter.Metric
	for _, option := range fitMetricLabels {
		if option.Key == filter.Metric {
			label = option.Label
		}
	}
	if filter.Metric == "npc_ehp" {
		label += " · " + fitSearchNPCLabel(filter.NPCProfile)
	}
	if filter.Min != nil && filter.Max != nil {
		return label + " " + fitSearchNumber(*filter.Min) + "–" + fitSearchNumber(*filter.Max)
	}
	if filter.Min != nil {
		return label + " ≥ " + fitSearchNumber(*filter.Min)
	}
	if filter.Max != nil {
		return label + " ≤ " + fitSearchNumber(*filter.Max)
	}
	return label
}
func fitSearchNPCLabel(profile string) string {
	for _, option := range fitNPCProfiles {
		if option.Key == profile {
			return option.Label
		}
	}
	return profile
}
func fitSearchDraftMetric(state fitSearchState) string {
	if len(state.Statistics) > 0 {
		return state.Statistics[len(state.Statistics)-1].Metric
	}
	return "ehp"
}
func fitSearchDraftNPC(state fitSearchState) string {
	if len(state.Statistics) > 0 && state.Statistics[len(state.Statistics)-1].NPCProfile != "" {
		return state.Statistics[len(state.Statistics)-1].NPCProfile
	}
	return "guristas"
}
func fitSearchChoiceLabel(options []struct{ Key, Label string }, key string) string {
	for _, option := range options {
		if option.Key == key {
			return option.Label
		}
	}
	return key
}
func fitSearchBucketHeight(bucket api.UniverseFitDistributionBucket, rows []api.UniverseFitDistributionBucket) string {
	if bucket.ObservationCount == 0 {
		return "0%"
	}
	max := int64(1)
	for _, row := range rows {
		if row.ObservationCount > max {
			max = row.ObservationCount
		}
	}
	return fmt.Sprintf("%.6f%%", math.Max(4, float64(bucket.ObservationCount)/float64(max)*100))
}
func fitSearchBucketActive(state fitSearchState, metric string, bucket api.UniverseFitDistributionBucket) bool {
	for _, filter := range state.Statistics {
		if filter.Metric == metric && filter.Min != nil && filter.Max != nil && *filter.Min == bucket.LowerBound && *filter.Max == bucket.UpperBound {
			return true
		}
	}
	return false
}
func fitSearchModules(fit fitSearchResult, slot int) []api.UniverseFitModule {
	if slot == 6 {
		rows := []api.UniverseFitModule{}
		for index, drone := range fit.Drones {
			quantity := ""
			if drone.Quantity > 1 {
				quantity = strconv.Itoa(drone.Quantity) + "×"
			}
			rows = append(rows, api.UniverseFitModule{SlotGroup: 6, Ordinal: index, TypeID: drone.TypeID, Name: drone.Name, ChargeName: quantity})
		}
		return rows
	}
	return fitFamilyModules(fit.UniverseFitFamily, slot)
}
func fitSearchSlotLabel(slot int) string {
	return []string{"", "High", "Mid", "Low", "Rigs", "Subsystems", "Drones"}[slot]
}
