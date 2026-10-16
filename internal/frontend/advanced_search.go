package frontend

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/api"
)

type advancedOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
type advancedOptionGroup struct {
	Key     string           `json:"key"`
	Label   string           `json:"label"`
	Options []advancedOption `json:"options"`
}

var advancedOptionGroups = []advancedOptionGroup{
	{"security", "Security Space", []advancedOption{{"highsec", "Highsec"}, {"lowsec", "Lowsec"}, {"nullsec", "Nullsec"}, {"wspace", "W-Space"}, {"abyssal", "Abyssal"}, {"pochven", "Pochven"}}},
	{"attackerCount", "Attacker Count", []advancedOption{{"solo", "Solo"}, {"2+", "2+"}, {"5+", "5+"}, {"10+", "10+"}, {"20+", "20+"}, {"50+", "50+"}, {"100+", "100+"}}},
	{"attackerType", "Attacker Type", []advancedOption{{"ganked", "Ganked"}, {"npc", "NPC"}}},
	{"iskValue", "ISK Value", []advancedOption{{"1b", "1B+"}, {"5b", "5B+"}, {"10b", "10B+"}, {"50b", "50B+"}, {"100b", "100B+"}}},
	{"shipCategory", "Ship Category", []advancedOption{{"frigates", "Frigates"}, {"destroyers", "Destroyers"}, {"cruisers", "Cruisers"}, {"battlecruisers", "Battlecruisers"}, {"battleships", "Battleships"}, {"capitals", "Capitals"}, {"supercarriers", "Supercarriers"}, {"titans", "Titans"}, {"supercapitals", "Super Capitals"}, {"freighters", "Freighters"}, {"citadels", "Structures"}}},
}
var advancedTechOptions = []advancedOption{{"t1", "T1"}, {"t2", "T2"}, {"t3", "T3"}, {"faction", "Faction"}}
var advancedTimeOptions = []advancedOption{{"today", "Today"}, {"yesterday", "Yesterday"}, {"24h", "24h"}, {"7d", "7 Days"}, {"30d", "30 Days"}, {"90d", "90 Days"}, {"thisWeek", "This Week"}, {"thisMonth", "This Month"}}
var advancedSortOptions = []advancedOption{{"killmail_time", "Time"}, {"total_value", "ISK Value"}, {"attacker_count", "Attackers"}}

type advancedAmount string

func (a *advancedAmount) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*a = ""
		return nil
	}
	var value string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
	} else {
		var number json.Number
		if err := json.Unmarshal(data, &number); err != nil {
			return err
		}
		value = number.String()
	}
	*a = advancedAmount(value)
	return nil
}

type advancedSearchEntity struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Role    string `json:"role,omitempty"`
	Exclude bool   `json:"exclude,omitempty"`
}
type advancedSearchItem struct {
	TypeID  int64  `json:"typeId,omitempty"`
	GroupID int64  `json:"groupId,omitempty"`
	Name    string `json:"name"`
	Slot    string `json:"slot"`
	Side    string `json:"side"`
}
type advancedSearchLocation struct {
	SecurityTypes     []string `json:"securityTypes"`
	SystemID          int64    `json:"systemId,omitempty"`
	SystemName        string   `json:"systemName,omitempty"`
	RegionID          int64    `json:"regionId,omitempty"`
	RegionName        string   `json:"regionName,omitempty"`
	ConstellationID   int64    `json:"constellationId,omitempty"`
	ConstellationName string   `json:"constellationName,omitempty"`
}
type advancedSearchTimeRange struct {
	Preset *string `json:"preset,omitempty"`
	From   string  `json:"from,omitempty"`
	To     string  `json:"to,omitempty"`
}
type advancedSearchSort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}
type advancedSearchFilters struct {
	Label         string                            `json:"label"`
	Entities      map[string][]advancedSearchEntity `json:"entities"`
	Items         []advancedSearchItem              `json:"items"`
	Location      advancedSearchLocation            `json:"location"`
	TimeRange     advancedSearchTimeRange           `json:"timeRange"`
	AttackerCount string                            `json:"attackerCount"`
	AttackerType  string                            `json:"attackerType"`
	ISKValue      string                            `json:"iskValue"`
	ISKMin        advancedAmount                    `json:"iskMin"`
	ISKMax        advancedAmount                    `json:"iskMax"`
	ShipCategory  string                            `json:"shipCategory"`
	TechLevel     string                            `json:"techLevel"`
	Sort          advancedSearchSort                `json:"sort"`
}
type advancedSearchState struct {
	Filters     advancedSearchFilters `json:"filters"`
	View        string                `json:"view"`
	Dedup       string                `json:"dedup"`
	Hash        string                `json:"hash"`
	HashMode    string                `json:"hashMode"`
	ShowResults bool                  `json:"showResults"`
}
type advancedSearchView struct {
	State       advancedSearchState
	Labels      []api.LabelEntry
	Result      api.AdvancedSearchResult
	Error       string
	Page, Limit int
}
type advancedChip struct{ Key, Group, Label string }

func advancedDefaultFilters() advancedSearchFilters {
	preset := "30d"
	return advancedSearchFilters{Entities: map[string][]advancedSearchEntity{"victim": {}, "attacker": {}, "both": {}}, Items: []advancedSearchItem{}, Location: advancedSearchLocation{SecurityTypes: []string{}}, TimeRange: advancedSearchTimeRange{Preset: &preset}, Sort: advancedSearchSort{"killmail_time", "desc"}}
}

func advancedParseState(query url.Values) (advancedSearchState, error) {
	state := advancedSearchState{Filters: advancedDefaultFilters(), View: "kills", Dedup: "family"}
	raw := query.Get("q")
	if raw != "" {
		if len(raw) > 65536 {
			return state, fmt.Errorf("search link is too large")
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &fields); err != nil || fields == nil {
			return state, fmt.Errorf("the search link contains invalid filters")
		}
		if _, ok := fields["timeRange"]; ok {
			state.Filters.TimeRange = advancedSearchTimeRange{}
		}
		if err := json.Unmarshal([]byte(raw), &state.Filters); err != nil {
			return state, fmt.Errorf("the search link contains invalid filters")
		}
		for _, role := range []string{"victim", "attacker", "both"} {
			if state.Filters.Entities == nil {
				state.Filters.Entities = map[string][]advancedSearchEntity{}
			}
			if state.Filters.Entities[role] == nil {
				state.Filters.Entities[role] = []advancedSearchEntity{}
			}
		}
		for i := range state.Filters.Items {
			item := &state.Filters.Items[i]
			if item.Slot == "" {
				item.Slot = "any"
			}
			if item.Side == "" {
				item.Side = "victim"
			}
		}
		state.ShowResults = true
	}
	if query.Get("view") == "fits" {
		state.View = "fits"
		state.ShowResults = true
	}
	if member(query.Get("dedup"), "none", "exact", "family") {
		state.Dedup = query.Get("dedup")
	}
	if hash := query.Get("dh"); hash != "" && member(query.Get("dm"), "exact", "family") {
		state.Hash = hash
		state.HashMode = query.Get("dm")
		state.ShowResults = true
	}
	return state, nil
}

var advancedISKPattern = regexp.MustCompile(`(?i)^([\d.]+)\s*([bmk])?$`)
var advancedFloatPrefix = regexp.MustCompile(`^(?:\d+(?:\.\d*)?|\.\d+)`)

func advancedParseISK(value string) *float64 {
	match := advancedISKPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return nil
	}
	number, err := strconv.ParseFloat(advancedFloatPrefix.FindString(match[1]), 64)
	if err != nil {
		return nil
	}
	switch strings.ToLower(match[2]) {
	case "b":
		number *= 1e9
	case "m":
		number *= 1e6
	case "k":
		number *= 1e3
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return nil
	}
	return &number
}
func advancedSerializedFilters(f advancedSearchFilters) string {
	out := map[string]any{"timeRange": f.TimeRange}
	if f.TimeRange.From == "" && f.TimeRange.To == "" && f.TimeRange.Preset == nil {
		out["timeRange"] = map[string]any{"preset": nil}
	}
	if f.Label != "" {
		out["label"] = f.Label
	}
	entities := map[string]any{}
	for _, role := range []string{"victim", "attacker", "both"} {
		rows := []map[string]any{}
		for _, entity := range f.Entities[role] {
			row := map[string]any{"id": entity.ID, "type": entity.Type, "name": entity.Name}
			if entity.Exclude {
				row["exclude"] = true
			}
			rows = append(rows, row)
		}
		if len(rows) > 0 {
			entities[role] = rows
		}
	}
	if len(entities) > 0 {
		out["entities"] = entities
	}
	if len(f.Items) > 0 {
		out["items"] = f.Items
	}
	loc := map[string]any{}
	if len(f.Location.SecurityTypes) > 0 {
		loc["securityTypes"] = f.Location.SecurityTypes
	}
	if f.Location.SystemID != 0 {
		loc["systemId"] = f.Location.SystemID
	}
	if f.Location.RegionID != 0 {
		loc["regionId"] = f.Location.RegionID
	}
	if f.Location.ConstellationID != 0 {
		loc["constellationId"] = f.Location.ConstellationID
	}
	if len(loc) > 0 {
		out["location"] = loc
	}
	for key, value := range map[string]string{"attackerCount": f.AttackerCount, "attackerType": f.AttackerType, "shipCategory": f.ShipCategory, "techLevel": f.TechLevel} {
		if value != "" {
			out[key] = value
		}
	}
	a, b := advancedParseISK(string(f.ISKMin)), advancedParseISK(string(f.ISKMax))
	if a != nil {
		out["iskMin"] = *a
	}
	if b != nil {
		out["iskMax"] = *b
	}
	if a == nil && b == nil && f.ISKValue != "" {
		out["iskValue"] = f.ISKValue
	}
	if f.Sort.Field != "killmail_time" || f.Sort.Direction != "desc" {
		out["sort"] = f.Sort
	}
	encoded, _ := json.Marshal(out)
	return string(encoded)
}
func advancedHasFilters(f advancedSearchFilters) bool {
	return f.Label != "" || len(f.Entities["victim"])+len(f.Entities["attacker"])+len(f.Entities["both"])+len(f.Items)+len(f.Location.SecurityTypes) > 0 || f.Location.SystemID+f.Location.RegionID+f.Location.ConstellationID != 0 || f.AttackerCount != "" || f.AttackerType != "" || f.ISKValue != "" || f.ISKMin != "" || f.ISKMax != "" || f.ShipCategory != "" || f.TechLevel != ""
}
func advancedExtraFilterCount(f advancedSearchFilters) int {
	count := 0
	for _, active := range []bool{f.Label != "", f.AttackerCount != "", f.AttackerType != "", f.ISKValue != "", f.ISKMin != "" || f.ISKMax != "", f.ShipCategory != "", f.TechLevel != "", f.TimeRange.From != "" || f.TimeRange.To != ""} {
		if active {
			count++
		}
	}
	return count
}
func advancedChoice(f advancedSearchFilters, key string) string {
	switch key {
	case "attackerCount":
		return f.AttackerCount
	case "attackerType":
		return f.AttackerType
	case "iskValue":
		return f.ISKValue
	case "shipCategory":
		return f.ShipCategory
	case "techLevel":
		return f.TechLevel
	case "sortField":
		return f.Sort.Field
	case "sortDirection":
		return f.Sort.Direction
	case "timePreset":
		if f.TimeRange.Preset != nil {
			return *f.TimeRange.Preset
		}
	}
	return ""
}
func advancedButtonClass(active bool) string {
	return killBoolClass(active, "bg-blue-500/20 text-link border-blue-500/40", "bg-white/[0.04] text-gray-400 border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-link")
}
func advancedOptionLabel(options []advancedOption, value string) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}
	return value
}
func advancedLocationLabel(loc advancedSearchLocation) string {
	if loc.SystemName != "" {
		return "System: " + loc.SystemName
	}
	if loc.ConstellationName != "" {
		return "Constellation: " + loc.ConstellationName
	}
	if loc.RegionName != "" {
		return "Region: " + loc.RegionName
	}
	return ""
}
func advancedActiveChips(view advancedSearchView) []advancedChip {
	f := view.State.Filters
	chips := []advancedChip{}
	if f.Label != "" {
		label := f.Label
		for _, option := range view.Labels {
			if option.ID == f.Label {
				label = option.Name
			}
		}
		chips = append(chips, advancedChip{"label", "Label", label})
	}
	for _, sec := range f.Location.SecurityTypes {
		chips = append(chips, advancedChip{"sec-" + sec, "Space", advancedOptionLabel(advancedOptionGroups[0].Options, sec)})
	}
	for _, group := range advancedOptionGroups[1:4] {
		value := advancedChoice(f, group.Key)
		if value != "" {
			label := map[string]string{"attackerCount": "Attackers", "attackerType": "Type", "iskValue": "ISK"}[group.Key]
			chips = append(chips, advancedChip{group.Key, label, advancedOptionLabel(group.Options, value)})
		}
	}
	if f.ISKMin != "" || f.ISKMax != "" {
		a, b := string(f.ISKMin), string(f.ISKMax)
		if a == "" {
			a = "0"
		}
		if b == "" {
			b = "∞"
		}
		chips = append(chips, advancedChip{"iskRange", "ISK", a + " – " + b})
	}
	if f.ShipCategory != "" {
		chips = append(chips, advancedChip{"shipCategory", "Ships", advancedOptionLabel(advancedOptionGroups[4].Options, f.ShipCategory)})
	}
	if f.TechLevel != "" {
		chips = append(chips, advancedChip{"techLevel", "Tech", advancedOptionLabel(advancedTechOptions, f.TechLevel)})
	}
	if f.TimeRange.Preset != nil && *f.TimeRange.Preset != "" {
		chips = append(chips, advancedChip{"timePreset", "Time", advancedOptionLabel(advancedTimeOptions, *f.TimeRange.Preset)})
	} else if f.TimeRange.From != "" || f.TimeRange.To != "" {
		from, to := strings.ReplaceAll(f.TimeRange.From, "T", " "), strings.ReplaceAll(f.TimeRange.To, "T", " ")
		label := from + " → " + to
		if from == "" {
			label = "Until " + to
		} else if to == "" {
			label = "From " + from
		}
		chips = append(chips, advancedChip{"timeRange", "Time", label + " EVE"})
	}
	return chips
}
func advancedRoleColor(role string, exclude bool) string {
	if exclude {
		return "bg-red-900/30 text-red-300 border-red-500/40 line-through"
	}
	switch role {
	case "victim":
		return "bg-red-500/20 text-red-400 border-red-500/30"
	case "attacker":
		return "bg-blue-500/20 text-link border-blue-500/30"
	default:
		return "bg-green-500/20 text-green-400 border-green-500/30"
	}
}
func advancedEntityLabel(role string, e advancedSearchEntity) string {
	if e.Exclude {
		return "exclude"
	}
	if e.Type == "weapon" {
		return "weapon"
	}
	if e.Type == "shipgroup" {
		return "hull class"
	}
	return role
}
func advancedEntityImage(kind string, id int64) string {
	switch kind {
	case "character", "corporation", "alliance":
		return eveEntityImage(kind, id, 32)
	case "faction":
		return eveEntityImage("corporation", id, 32)
	case "ship", "weapon", "item":
		return fmt.Sprintf("/images/types/%d/icon?size=32", id)
	}
	return ""
}
func advancedLabelCategories(labels []api.LabelEntry) []string {
	out := []string{}
	for _, label := range labels {
		if !member(label.Category, out...) {
			out = append(out, label.Category)
		}
	}
	return out
}

func advancedSearchMetadata() Metadata {
	return Metadata{Title: "Advanced Search", OGTitle: "Advanced Search — EVE-KILL", Description: "Advanced killmail search on EVE-KILL — filter by pilots, ships, location, ISK value, security status, time range, and more.", CanonicalPath: "/advancedsearch", Indexable: true, Stylesheets: []string{"utilities.css", "advanced-search.css"}, ModuleScripts: []string{"advanced-search.js"}}
}
func (s *site) loadAdvancedSearch(r *http.Request, labels bool) (advancedSearchView, error) {
	state, err := advancedParseState(r.URL.Query())
	view := advancedSearchView{State: state, Page: roamPositiveInt(r.URL.Query().Get("page"), 1), Limit: roamPositiveInt(r.URL.Query().Get("limit"), 50)}
	if !member(strconv.Itoa(view.Limit), "10", "25", "50", "100") {
		view.Limit = 50
	}
	if err != nil {
		view.Error = err.Error()
		view.State.ShowResults = false
	}
	if labels {
		view.Labels, err = s.options.AdvancedSearch.Labels(r.Context())
		if err != nil {
			return view, err
		}
		if view.Labels == nil {
			view.Labels = []api.LabelEntry{}
		}
	}
	if r.URL.Query().Get("empty") == "1" {
		view.State.ShowResults = false
	}
	if view.State.ShowResults {
		query := url.Values{"filters": {advancedSerializedFilters(view.State.Filters)}, "limit": {strconv.Itoa(view.Limit)}}
		if view.State.View == "fits" {
			query.Set("view", "fits")
			query.Set("dedup", view.State.Dedup)
		}
		if view.State.Hash != "" {
			key := "fitHash"
			if view.State.HashMode == "family" {
				key = "familyHash"
			}
			query.Set(key, view.State.Hash)
		}
		if after := r.URL.Query().Get("after"); after != "" {
			query.Set("after", after)
		}
		view.Result, err = s.options.AdvancedSearch.AdvancedSearch(r.Context(), query)
		if err != nil {
			view.Error = "Could not load search results. Please check the filters and try again."
			return view, err
		}
	}
	return view, nil
}
func (s *site) advancedSearchPage(r *http.Request) (Page, error) {
	view, err := s.loadAdvancedSearch(r, true)
	if err != nil && view.Labels == nil {
		return Page{}, err
	}
	meta := advancedSearchMetadata()
	if r.URL.RawQuery != "" {
		meta.Indexable = false
	}
	return Page{Metadata: meta, Content: advancedSearchPage(view)}, nil
}
func (s *site) advancedSearchFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	view, err := s.loadAdvancedSearch(r, false)
	if err != nil || view.Error != "" {
		s.renderFragment(w, r, advancedSearchError(view.Error))
		return
	}
	s.renderFragment(w, r, advancedSearchResults(view))
}
