package mcpserver

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

type RouteDangerInput struct {
	From      StringOrInt64   `json:"from" doc:"Starting solar system name or id."`
	To        StringOrInt64   `json:"to" doc:"Destination solar system name or id."`
	Prefer    string          `json:"prefer,omitempty" enum:"shortest,safest,lowsec_ok" default:"shortest"`
	Avoid     []StringOrInt64 `json:"avoid,omitempty" maxItems:"20"`
	RoundTrip bool            `json:"round_trip,omitempty" default:"false"`
	Hours     float64         `json:"hours,omitempty" default:"1" minimum:"0.25" maximum:"72"`
}

type RouteHop struct {
	Step        int      `json:"step"`
	SystemID    int64    `json:"system_id"`
	SystemName  *string  `json:"system_name"`
	Security    *float64 `json:"security"`
	SecBand     string   `json:"sec_band"`
	RegionID    *int64   `json:"region_id"`
	RegionName  *string  `json:"region_name"`
	KillsWindow int64    `json:"kills_window"`
	Danger      float64  `json:"danger"`
	URL         string   `json:"url"`
}

type WorstRouteHop struct {
	SystemID    int64   `json:"system_id"`
	SystemName  *string `json:"system_name"`
	SecBand     string  `json:"sec_band"`
	KillsWindow int64   `json:"kills_window"`
	Danger      float64 `json:"danger"`
}

type RouteLeg struct {
	Jumps             int        `json:"jumps"`
	TotalKillsOnRoute int64      `json:"total_kills_on_route"`
	CrossesLowsec     bool       `json:"crosses_lowsec"`
	CrossesNullsec    bool       `json:"crosses_nullsec"`
	AverageDanger     float64    `json:"avg_danger"`
	Hops              []RouteHop `json:"hops"`
}

type RouteDangerOutput struct {
	From              SystemRef     `json:"from"`
	To                SystemRef     `json:"to"`
	Prefer            string        `json:"prefer"`
	AvoidedSystems    int           `json:"avoided_systems"`
	Jumps             int           `json:"jumps"`
	WindowHours       float64       `json:"window_hours"`
	TotalKillsOnRoute int64         `json:"total_kills_on_route"`
	CrossesLowsec     bool          `json:"crosses_lowsec"`
	CrossesNullsec    bool          `json:"crosses_nullsec"`
	AverageDanger     float64       `json:"avg_danger"`
	WorstHop          WorstRouteHop `json:"worst_hop"`
	Hops              []RouteHop    `json:"hops"`
	ReturnLeg         *RouteLeg     `json:"return_leg,omitempty"`
}

type routeGraph struct {
	adjacency map[int64][]int64
	security  map[int64]float64
	loadedAt  time.Time
}

var cachedRouteGraph struct {
	sync.RWMutex
	value routeGraph
}

func registerRouteTool(registry *Registry) error {
	return addTool(registry, ToolDefinition{
		Name: "route_danger", Title: "Assess stargate route danger",
		Description: "Build a shortest or security-weighted stargate route between two systems and annotate every hop with recent kills and danger.",
	}, func(ctx context.Context, input RouteDangerInput) (RouteDangerOutput, error) {
		return routeDanger(ctx, registry.deps, input)
	})
}

func routeDanger(ctx context.Context, deps Dependencies, input RouteDangerInput) (RouteDangerOutput, error) {
	from, err := resolveEntity(ctx, deps, input.From, new(EntitySystem))
	if err != nil || from == nil || from.Type != EntitySystem {
		if err != nil {
			return RouteDangerOutput{}, err
		}
		return RouteDangerOutput{}, fmt.Errorf("could not resolve from to a solar system")
	}
	to, err := resolveEntity(ctx, deps, input.To, new(EntitySystem))
	if err != nil || to == nil || to.Type != EntitySystem {
		if err != nil {
			return RouteDangerOutput{}, err
		}
		return RouteDangerOutput{}, fmt.Errorf("could not resolve to to a solar system")
	}
	graph, err := getRouteGraph(ctx, deps)
	if err != nil {
		return RouteDangerOutput{}, err
	}
	avoid := map[int64]bool{}
	for index, reference := range input.Avoid {
		system, resolveErr := resolveEntity(ctx, deps, reference, new(EntitySystem))
		if resolveErr != nil {
			return RouteDangerOutput{}, resolveErr
		}
		if system == nil || system.Type != EntitySystem {
			return RouteDangerOutput{}, fmt.Errorf("avoid[%d] did not resolve to a system", index)
		}
		avoid[system.ID] = true
	}
	prefer := input.Prefer
	if prefer == "" {
		prefer = "shortest"
	}
	path := findRoute(graph, from.ID, to.ID, prefer, avoid)
	if len(path) == 0 {
		return RouteDangerOutput{}, fmt.Errorf("no stargate route from %s to %s", from.Name, to.Name)
	}
	var returnPath []int64
	if input.RoundTrip {
		returnPath = findRoute(graph, to.ID, from.ID, prefer, avoid)
		if len(returnPath) == 0 {
			return RouteDangerOutput{}, fmt.Errorf("no return route from %s to %s", to.Name, from.Name)
		}
	}
	hours := input.Hours
	if hours == 0 {
		hours = 1
	}
	hours = math.Max(0.25, math.Min(72, hours))
	allIDs := append(append([]int64{}, path...), returnPath...)
	meta, err := loadRouteMeta(ctx, deps, allIDs, time.Now().UTC().Add(-time.Duration(hours*float64(time.Hour))))
	if err != nil {
		return RouteDangerOutput{}, err
	}
	hops := annotateRoute(deps.BaseURL, path, meta)
	leg := summarizeRoute(hops)
	worst := hops[0]
	for _, hop := range hops[1:] {
		if hop.Danger > worst.Danger {
			worst = hop
		}
	}
	output := RouteDangerOutput{
		From:   SystemRef{ID: from.ID, Name: from.Name, URL: entityURL(deps.BaseURL, EntitySystem, from.ID)},
		To:     SystemRef{ID: to.ID, Name: to.Name, URL: entityURL(deps.BaseURL, EntitySystem, to.ID)},
		Prefer: prefer, AvoidedSystems: len(avoid), Jumps: len(path) - 1, WindowHours: hours,
		TotalKillsOnRoute: leg.TotalKillsOnRoute, CrossesLowsec: leg.CrossesLowsec,
		CrossesNullsec: leg.CrossesNullsec, AverageDanger: leg.AverageDanger, Hops: hops,
		WorstHop: WorstRouteHop{SystemID: worst.SystemID, SystemName: worst.SystemName, SecBand: worst.SecBand, KillsWindow: worst.KillsWindow, Danger: worst.Danger},
	}
	if len(returnPath) > 0 {
		value := summarizeRoute(annotateRoute(deps.BaseURL, returnPath, meta))
		output.ReturnLeg = &value
	}
	return output, nil
}

func getRouteGraph(ctx context.Context, deps Dependencies) (routeGraph, error) {
	cachedRouteGraph.RLock()
	value := cachedRouteGraph.value
	cachedRouteGraph.RUnlock()
	if value.adjacency != nil && time.Since(value.loadedAt) < 24*time.Hour {
		return value, nil
	}
	edges, err := queryMaps(ctx, deps.DB, `SELECT from_solar_system_id AS f, to_solar_system_id AS t FROM solar_system_jumps`)
	if err != nil {
		return routeGraph{}, err
	}
	systems, err := queryMaps(ctx, deps.DB, `SELECT solar_system_id AS id, security FROM solar_systems`)
	if err != nil {
		return routeGraph{}, err
	}
	value = routeGraph{adjacency: map[int64][]int64{}, security: map[int64]float64{}, loadedAt: time.Now()}
	for _, edge := range edges {
		from, to := valueInt64(edge["f"]), valueInt64(edge["t"])
		value.adjacency[from] = append(value.adjacency[from], to)
		value.adjacency[to] = append(value.adjacency[to], from)
	}
	for _, system := range systems {
		value.security[valueInt64(system["id"])] = valueFloat64(system["security"])
	}
	cachedRouteGraph.Lock()
	cachedRouteGraph.value = value
	cachedRouteGraph.Unlock()
	return value, nil
}

func findRoute(graph routeGraph, from, to int64, prefer string, avoid map[int64]bool) []int64 {
	if prefer == "shortest" {
		return breadthFirstRoute(graph.adjacency, from, to, 60, avoid)
	}
	weights := map[string]float64{"highsec": 1, "lowsec": 10_000, "nullsec": 100_000_000}
	if prefer == "lowsec_ok" {
		weights["lowsec"] = 3
	}
	return safestRoute(graph, from, to, weights, avoid)
}

func breadthFirstRoute(adjacency map[int64][]int64, from, to int64, maxHops int, avoid map[int64]bool) []int64 {
	if from == to {
		return []int64{from}
	}
	queue, visited, previous := []int64{from}, map[int64]bool{from: true}, map[int64]int64{}
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		for _, neighbor := range adjacency[current] {
			if visited[neighbor] || (avoid[neighbor] && neighbor != to) {
				continue
			}
			visited[neighbor], previous[neighbor] = true, current
			if neighbor == to {
				return reconstructRoute(previous, from, to, maxHops)
			}
			queue = append(queue, neighbor)
		}
	}
	return nil
}

func safestRoute(graph routeGraph, from, to int64, weights map[string]float64, avoid map[int64]bool) []int64 {
	distances, previous, visited := map[int64]float64{from: 0}, map[int64]int64{}, map[int64]bool{}
	for {
		current, best, found := int64(0), math.Inf(1), false
		for id, distance := range distances {
			if !visited[id] && distance < best {
				current, best, found = id, distance, true
			}
		}
		if !found {
			return nil
		}
		if current == to {
			return reconstructRoute(previous, from, to, 120)
		}
		visited[current] = true
		for _, neighbor := range graph.adjacency[current] {
			if avoid[neighbor] && neighbor != to {
				continue
			}
			next := best + weights[routeSecurityBand(graph.security[neighbor])]
			old, exists := distances[neighbor]
			if !exists || next < old {
				distances[neighbor], previous[neighbor] = next, current
			}
		}
	}
}

func reconstructRoute(previous map[int64]int64, from, to int64, maxHops int) []int64 {
	path := []int64{to}
	for step := to; step != from; {
		parent, ok := previous[step]
		if !ok {
			return nil
		}
		path, step = append(path, parent), parent
	}
	if len(path)-1 > maxHops {
		return nil
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func routeSecurityBand(security float64) string {
	if security >= 0.45 {
		return "highsec"
	}
	if security > 0 {
		return "lowsec"
	}
	return "nullsec"
}

type routeMeta struct {
	name, regionName *string
	security         *float64
	regionID         *int64
	kills            int64
}

func loadRouteMeta(ctx context.Context, deps Dependencies, ids []int64, since time.Time) (map[int64]routeMeta, error) {
	output := map[int64]routeMeta{}
	if len(ids) == 0 {
		return output, nil
	}
	systems, err := queryMaps(ctx, deps.DB, `
		SELECT s.solar_system_id AS id, s.system_name, s.security, s.region_id, r.name AS region_name
		FROM solar_systems s LEFT JOIN regions r ON r.region_id = s.region_id
		WHERE s.solar_system_id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	kills, err := queryMaps(ctx, deps.DB, `
		SELECT solar_system_id AS id, COUNT(*)::bigint AS kills FROM killmails
		WHERE solar_system_id = ANY($1) AND killmail_time >= $2 GROUP BY solar_system_id`, ids, since)
	if err != nil {
		return nil, err
	}
	for _, row := range systems {
		output[valueInt64(row["id"])] = routeMeta{name: nullableString(row["system_name"]), security: nullableFloat64(row["security"]), regionID: nullableInt64(row["region_id"]), regionName: nullableString(row["region_name"])}
	}
	for _, row := range kills {
		id := valueInt64(row["id"])
		value := output[id]
		value.kills = valueInt64(row["kills"])
		output[id] = value
	}
	return output, nil
}

func annotateRoute(baseURL string, path []int64, meta map[int64]routeMeta) []RouteHop {
	output := make([]RouteHop, 0, len(path))
	for index, id := range path {
		value := meta[id]
		band := "unknown"
		if value.security != nil {
			band = routeSecurityBand(*value.security)
		}
		killHeat := math.Min(1, math.Log10(1+float64(value.kills))/math.Log10(21))
		floor := float64(0)
		if band == "lowsec" {
			floor = 0.25
		} else if band == "nullsec" {
			floor = 0.5
		}
		danger := math.Round(math.Min(1, math.Max(killHeat, floor)+killHeat*0.5)*100) / 100
		output = append(output, RouteHop{
			Step: index, SystemID: id, SystemName: value.name, Security: value.security, SecBand: band,
			RegionID: value.regionID, RegionName: value.regionName, KillsWindow: value.kills,
			Danger: danger, URL: entityURL(baseURL, EntitySystem, id),
		})
	}
	return output
}

func summarizeRoute(hops []RouteHop) RouteLeg {
	output := RouteLeg{Jumps: len(hops) - 1, Hops: hops}
	for _, hop := range hops {
		output.TotalKillsOnRoute += hop.KillsWindow
		output.CrossesLowsec = output.CrossesLowsec || hop.SecBand == "lowsec"
		output.CrossesNullsec = output.CrossesNullsec || hop.SecBand == "nullsec"
		output.AverageDanger += hop.Danger
	}
	if len(hops) > 0 {
		output.AverageDanger = math.Round(output.AverageDanger/float64(len(hops))*100) / 100
	}
	return output
}
