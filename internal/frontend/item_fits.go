package frontend

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

var fitMetricLabels = []struct{ Key, Label string }{
	{"ehp", "Effective hitpoints"}, {"dps", "Damage / second"}, {"alpha", "Alpha damage"},
	{"speed", "Maximum velocity"}, {"align", "Align time"}, {"repair", "Strongest local repair"},
	{"shield_repair", "Active shield repair"}, {"armor_repair", "Active armor repair"},
	{"hull_repair", "Active hull repair"}, {"passive_shield", "Passive shield regeneration"},
	{"npc_ehp", "EHP against NPC faction"},
}

var fitSortLabels = []struct{ Key, Label string }{
	{"observed", "Most observed"}, {"recent", "Recently seen"}, {"cheapest", "Cheapest"},
	{"expensive", "Most expensive"}, {"ehp", "Highest EHP"}, {"dps", "Highest DPS"},
	{"alpha", "Highest alpha"}, {"repair", "Highest repair"}, {"speed", "Highest speed"},
	{"align", "Fastest align"}, {"npc_ehp", "Highest NPC EHP"},
}

var fitNPCProfiles = []struct{ Key, Label string }{
	{"omni", "Omnidamage"}, {"angels", "Angel Cartel"}, {"blood-raiders", "Blood Raiders"},
	{"guristas", "Guristas Pirates"}, {"mordus", "Mordu's Legion"}, {"sansha", "Sansha's Nation"},
	{"serpentis", "Serpentis"}, {"triglavian", "Triglavian Collective"}, {"amarr", "Amarr Empire"},
	{"caldari", "Caldari State"}, {"gallente", "Gallente Federation"}, {"minmatar", "Minmatar Republic"},
}

func fitNPCLabel(key string) string {
	for _, profile := range fitNPCProfiles {
		if profile.Key == key {
			return profile.Label
		}
	}
	return key
}

func fitFilterLabel(metric, label string, filters url.Values) string {
	if metric == "npc_ehp" && filters.Get("npc_profile") != "" {
		label += " · " + fitNPCLabel(filters.Get("npc_profile"))
	}
	format := func(raw string) string {
		value, _ := strconv.ParseFloat(raw, 64)
		value = math.Round(value*1000) / 1000
		return strings.TrimRight(strings.TrimRight(fullNumber(math.Trunc(value))+fmt.Sprintf("%.3f", value-math.Trunc(value))[1:], "0"), ".")
	}
	minimum, maximum := filters.Get("min_"+metric), filters.Get("max_"+metric)
	switch {
	case minimum != "" && maximum != "":
		return label + " " + format(minimum) + "–" + format(maximum)
	case minimum != "":
		return label + " ≥ " + format(minimum)
	case maximum != "":
		return label + " ≤ " + format(maximum)
	default:
		return label
	}
}

func itemFitQuery(input url.Values) (url.Values, string, error) {
	result, sortBy := url.Values{}, "observed"
	allowedMetrics := map[string]bool{}
	for _, metric := range fitMetricLabels {
		allowedMetrics[metric.Key] = true
	}
	if raw := input.Get("sort"); raw != "" {
		valid := false
		for _, candidate := range fitSortLabels {
			if candidate.Key == raw {
				valid = true
				break
			}
		}
		if !valid {
			return nil, "", ErrNotFound
		}
		sortBy = raw
	}
	for key, values := range input {
		if len(values) != 1 {
			return nil, "", ErrNotFound
		}
		value := strings.TrimSpace(values[0])
		if key == "sort" || value == "" {
			continue
		}
		switch {
		case strings.HasPrefix(key, "min_") || strings.HasPrefix(key, "max_"):
			metric := strings.TrimPrefix(strings.TrimPrefix(key, "min_"), "max_")
			if !allowedMetrics[metric] {
				continue
			}
			number, err := strconv.ParseFloat(value, 64)
			if err != nil || number < 0 || math.IsNaN(number) || math.IsInf(number, 0) {
				return nil, "", ErrNotFound
			}
			result.Set(key, strconv.FormatFloat(number, 'f', -1, 64))
		case key == "npc_profile":
			valid := false
			for _, candidate := range fitNPCProfiles {
				if candidate.Key == value {
					valid = true
					break
				}
			}
			if !valid {
				return nil, "", ErrNotFound
			}
			result.Set(key, value)
		case key == "groups":
			parts := strings.Split(value, ",")
			if len(parts) > 12 {
				return nil, "", ErrNotFound
			}
			seen, groups := map[int64]bool{}, make([]string, 0, len(parts))
			for _, part := range parts {
				id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 32)
				if err != nil || id < 1 {
					return nil, "", ErrNotFound
				}
				if !seen[id] {
					seen[id] = true
					groups = append(groups, strconv.FormatInt(id, 10))
				}
			}
			result.Set(key, strings.Join(groups, ","))
		}
	}
	if (result.Get("min_npc_ehp") != "" || result.Get("max_npc_ehp") != "") && result.Get("npc_profile") == "" {
		result.Set("npc_profile", "guristas")
	}
	return result, sortBy, nil
}

func loadItemFits(ctx context.Context, reader UniverseReader, id int64, query url.Values) (itemPageView, error) {
	filters, sortBy, err := itemFitQuery(query)
	if err != nil {
		return itemPageView{}, err
	}
	families, err := reader.UniverseFitFamilies(ctx, id, filters)
	if err != nil {
		return itemPageView{}, err
	}
	view := itemPageView{ID: id, Tab: "fittings", Fits: families, FitQuery: filters, FitSort: sortBy}
	sortFitFamilies(view.Fits.Families, sortBy)
	return view, nil
}

func sortFitFamilies(families []api.UniverseFitFamily, sortBy string) {
	stat := func(f api.UniverseFitFamily) float64 {
		if f.Stats == nil {
			return -1
		}
		var value *float64
		switch sortBy {
		case "ehp":
			value = f.Stats.EHP
		case "dps":
			value = f.Stats.DPS
		case "alpha":
			value = f.Stats.Alpha
		case "repair":
			value = f.Stats.Repair
		case "speed":
			value = f.Stats.Speed
		case "align":
			value = f.Stats.Align
		case "npc_ehp":
			value = f.Stats.NPCEHP
		}
		if value == nil {
			return -1
		}
		return *value
	}
	sort.SliceStable(families, func(i, j int) bool {
		a, b := families[i], families[j]
		switch sortBy {
		case "recent":
			return a.LastUsed.After(b.LastUsed)
		case "cheapest":
			return a.FitCost < b.FitCost
		case "expensive":
			return a.FitCost > b.FitCost
		case "align":
			left, right := stat(a), stat(b)
			if left < 0 {
				return false
			}
			if right < 0 {
				return true
			}
			return left < right
		case "ehp", "dps", "alpha", "repair", "speed", "npc_ehp":
			return stat(a) > stat(b)
		default:
			return a.TotalUses > b.TotalUses
		}
	})
}

func (s *site) itemFitsFragment(w http.ResponseWriter, r *http.Request) {
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
	view, err := loadItemFits(r.Context(), s.options.Universe, id, r.URL.Query())
	if err != nil {
		if err == ErrNotFound {
			http.Error(w, "Invalid fitting filters", http.StatusBadRequest)
		} else {
			http.Error(w, "Could not load fits", http.StatusServiceUnavailable)
		}
		return
	}
	view.Data = item
	s.renderFragment(w, r, itemFitsPanel(view))
}

func (s *site) itemFitMetaFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	filters, _, err := itemFitQuery(r.URL.Query())
	if err != nil {
		http.Error(w, "Invalid fitting filters", http.StatusBadRequest)
		return
	}
	meta, err := s.options.Universe.UniverseFitMeta(r.Context(), id, filters)
	if err != nil {
		http.Error(w, "Could not load module usage", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, itemFitMeta(meta, filters, id))
}

func (s *site) itemFitDistributionsFragment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	filters, _, err := itemFitQuery(r.URL.Query())
	if err != nil {
		http.Error(w, "Invalid fitting filters", http.StatusBadRequest)
		return
	}
	data, err := s.options.Universe.UniverseFitDistributions(r.Context(), id, filters)
	if err != nil {
		http.Error(w, "Could not load fit profiles", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, itemFitDistributions(data, filters, id))
}

func fitLastSeen(value time.Time) string {
	days := int(time.Since(value).Hours() / 24)
	if days <= 0 {
		return "today"
	}
	if days == 1 {
		return "yesterday"
	}
	return fmt.Sprintf("%dd ago", days)
}

func fitMetricValue(metric string, value float64) string {
	if metric == "speed" {
		return fullNumber(value) + " m/s"
	}
	if value >= 1e6 {
		return fmt.Sprintf("%.1fm", value/1e6)
	}
	if value >= 1e3 {
		if value >= 100_000 {
			return fmt.Sprintf("%.0fk", value/1e3)
		}
		return fmt.Sprintf("%.1fk", value/1e3)
	}
	if value >= 100 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.1f", value)
}

func fitMetaClass(active bool) string {
	if active {
		return "active"
	}
	return ""
}

func fitDistributionLabel(metric string) string {
	for _, item := range []struct{ Key, Label string }{{"ehp", "Effective Hitpoints"}, {"dps", "Damage / second"}, {"repair", "Effective repair / second"}, {"speed", "Maximum velocity"}} {
		if item.Key == metric {
			return item.Label
		}
	}
	return metric
}

func fitBucketHeight(bucket api.UniverseFitDistributionBucket, rows []api.UniverseFitDistributionBucket) string {
	if bucket.ObservationCount == 0 {
		return "0%"
	}
	maximum := int64(1)
	for _, row := range rows {
		if row.ObservationCount > maximum {
			maximum = row.ObservationCount
		}
	}
	return fmt.Sprintf("%.1f%%", math.Max(3, float64(bucket.ObservationCount)/float64(maximum)*100))
}

func fitGroupSelected(filters url.Values, id int64) bool {
	for _, part := range strings.Split(filters.Get("groups"), ",") {
		if part == strconv.FormatInt(id, 10) {
			return true
		}
	}
	return false
}

func fitHasFilters(filters url.Values) bool { return len(filters) > 0 }

func fitFilterCount(filters url.Values) int {
	count := 0
	if filters.Get("groups") != "" {
		count += len(strings.Split(filters.Get("groups"), ","))
	}
	for _, metric := range fitMetricLabels {
		if filters.Get("min_"+metric.Key) != "" || filters.Get("max_"+metric.Key) != "" || metric.Key == "npc_ehp" && filters.Get("npc_profile") != "" {
			count++
		}
	}
	return count
}

func fitFamilyModules(f api.UniverseFitFamily, slot int) []api.UniverseFitModule {
	result := make([]api.UniverseFitModule, 0)
	for _, module := range f.Modules {
		if module.SlotGroup == slot {
			result = append(result, module)
		}
	}
	return result
}

func fitSlotLabel(slot int) string {
	switch slot {
	case 1:
		return "High"
	case 2:
		return "Med"
	case 3:
		return "Low"
	case 4:
		return "Rig"
	case 5:
		return "Subsystem"
	}
	return ""
}

func fitEditorURL(shipID int64, family api.UniverseFitFamily, days int) string {
	return observedFitEditorURL(shipID, "Community Fit", fmt.Sprintf("Loaded from /fits — %d recorded uses in the last %d days.", family.TotalUses, days), family.Modules, family.Drones)
}

// Shared ESF v3 conversion, matching killmailFitToUiFit in the Nuxt source.
func observedFitEditorURL(shipID int64, name, description string, modules []api.UniverseFitModule, drones []api.UniverseFitDrone) string {
	var plain strings.Builder
	fmt.Fprintf(&plain, "ship,%d,%s,%s\n", shipID, name, description)
	for _, slot := range []struct {
		ID   int
		Name string
	}{{1, "High"}, {2, "Medium"}, {3, "Low"}, {4, "Rig"}, {5, "SubSystem"}} {
		index := 0
		for _, module := range modules {
			if module.SlotGroup != slot.ID {
				continue
			}
			index++
			charge := ""
			if module.ChargeTypeID > 0 {
				charge = strconv.FormatInt(module.ChargeTypeID, 10)
			}
			fmt.Fprintf(&plain, "module,%s,%d,%d,Active,%s\n", slot.Name, index, module.TypeID, charge)
		}
	}
	budget := 5
	for _, drone := range drones {
		active := min(drone.Quantity, budget)
		budget -= active
		fmt.Fprintf(&plain, "drone,%d,%d,%d\n", drone.TypeID, active, drone.Quantity-active)
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte(plain.String())); err != nil {
		return "/fits/create"
	}
	if err := writer.Close(); err != nil {
		return "/fits/create"
	}
	return "/fits/create?fit=" + url.QueryEscape("v3:"+base64.StdEncoding.EncodeToString(compressed.Bytes()))
}

func fitHullCost(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
func fitLossesLabel(count int) string {
	if count == 1 {
		return "loss"
	}
	return "losses"
}
func fitVariantsLabel(count int) string {
	if count == 1 {
		return "variant"
	}
	return "variants"
}
