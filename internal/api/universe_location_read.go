package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type UniverseSystemData struct {
	System struct {
		SolarSystemID     int64    `json:"solar_system_id"`
		SystemName        string   `json:"system_name"`
		Security          *float64 `json:"security"`
		ConstellationID   int64    `json:"constellation_id"`
		ConstellationName string   `json:"constellation_name"`
		RegionID          int64    `json:"region_id"`
		RegionName        string   `json:"region_name"`
		SunTypeName       string   `json:"sun_type_name"`
	} `json:"system"`
	Celestials struct {
		Planets int64 `json:"planets"`
		Belts   int64 `json:"belts"`
	} `json:"celestials"`
	CelestialList []struct {
		ItemID   int64  `json:"item_id"`
		ItemName string `json:"item_name"`
		TypeName string `json:"type_name"`
		Category string `json:"category"`
	} `json:"celestialList"`
	Stations []struct {
		StationID       int64  `json:"station_id"`
		StationName     string `json:"station_name"`
		OperationName   string `json:"operation_name"`
		CorporationID   int64  `json:"corporation_id"`
		CorporationName string `json:"corporation_name"`
	} `json:"stations"`
	Structures []struct {
		StructureID int64  `json:"structure_id"`
		Name        string `json:"name"`
		TypeName    string `json:"type_name"`
		OwnerID     int64  `json:"owner_id"`
		OwnerName   string `json:"owner_name"`
		IsMarket    bool   `json:"is_market"`
	} `json:"structures"`
	Connections []struct {
		ToSolarSystemID int64    `json:"to_solar_system_id"`
		SystemName      string   `json:"system_name"`
		Security        *float64 `json:"security"`
		RegionID        int64    `json:"region_id"`
		IsRegional      bool     `json:"is_regional"`
	} `json:"connections"`
	Sovereignty        *UniverseSovereignty  `json:"sovereignty"`
	SovereigntyHistory []UniverseSovereignty `json:"sovereigntyHistory"`
	Stats              struct {
		Kills      int64   `json:"kills"`
		TotalValue float64 `json:"total_value"`
	} `json:"stats"`
	Activity struct {
		Latest  *UniverseSystemActivity  `json:"latest"`
		History []UniverseSystemActivity `json:"history"`
	} `json:"activity"`
}

type UniverseLocationStats struct {
	Kills      int64   `json:"kills"`
	NPCKills   int64   `json:"npc_kills"`
	PodKills   int64   `json:"pod_kills"`
	TotalValue float64 `json:"total_value"`
}

type UniverseLocationSovereignty struct {
	AllianceID   int64  `json:"alliance_id"`
	AllianceName string `json:"alliance_name"`
	FactionID    int64  `json:"faction_id"`
	FactionName  string `json:"faction_name"`
	SystemCount  int64  `json:"system_count"`
}

type UniverseLocationSystem struct {
	SolarSystemID int64    `json:"solar_system_id"`
	SystemName    string   `json:"system_name"`
	Security      *float64 `json:"security"`
	AllianceID    int64    `json:"alliance_id"`
	AllianceName  string   `json:"alliance_name"`
	FactionID     int64    `json:"faction_id"`
	FactionName   string   `json:"faction_name"`
	Kills         int64    `json:"kills"`
	TotalValue    float64  `json:"total_value"`
}

type UniverseRegionData struct {
	Region struct {
		RegionID           int64  `json:"region_id"`
		Name               string `json:"name"`
		Description        string `json:"description"`
		ConstellationCount int64  `json:"constellation_count"`
		SystemCount        int64  `json:"system_count"`
	} `json:"region"`
	Constellations []struct {
		ConstellationID   int64  `json:"constellation_id"`
		ConstellationName string `json:"constellation_name"`
		SystemCount       int64  `json:"system_count"`
		AllianceID        int64  `json:"alliance_id"`
		AllianceName      string `json:"alliance_name"`
		FactionID         int64  `json:"faction_id"`
		FactionName       string `json:"faction_name"`
	} `json:"constellations"`
	SovDistribution []UniverseLocationSovereignty `json:"sovDistribution"`
	TopSystems      []UniverseLocationSystem      `json:"topSystems"`
	Stats           UniverseLocationStats         `json:"stats"`
}

type UniverseConstellationData struct {
	Constellation struct {
		ConstellationID   int64  `json:"constellation_id"`
		ConstellationName string `json:"constellation_name"`
		RegionID          int64  `json:"region_id"`
		RegionName        string `json:"region_name"`
	} `json:"constellation"`
	Systems         []UniverseLocationSystem      `json:"systems"`
	SovDistribution []UniverseLocationSovereignty `json:"sovDistribution"`
	Stats           UniverseLocationStats         `json:"stats"`
}

type UniverseSovereignty struct {
	AllianceID      int64  `json:"alliance_id"`
	AllianceName    string `json:"alliance_name"`
	CorporationID   int64  `json:"corporation_id"`
	CorporationName string `json:"corporation_name"`
	FactionID       int64  `json:"faction_id"`
	FactionName     string `json:"faction_name"`
	Date            string `json:"date"`
}

type UniverseSystemActivity struct {
	Timestamp time.Time `json:"timestamp"`
	ShipJumps int64     `json:"ship_jumps"`
	ShipKills int64     `json:"ship_kills"`
	NPCKills  int64     `json:"npc_kills"`
	PodKills  int64     `json:"pod_kills"`
}

func (r *HomepageReader) UniverseSystem(ctx context.Context, id int64) (UniverseSystemData, bool, error) {
	if r == nil || r.db == nil || id < 1 || id > 2_147_483_647 {
		return UniverseSystemData{}, false, fmt.Errorf("invalid solar system")
	}
	data, err := cachedHomepage(ctx, r, fmt.Sprintf("universe-system:%d", id), 2*time.Minute, func(ctx context.Context) (map[string]any, error) {
		return loadUniverseSystem(ctx, r.db, id)
	})
	if err != nil || data == nil {
		return UniverseSystemData{}, data != nil, err
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return UniverseSystemData{}, false, err
	}
	var result UniverseSystemData
	if err := json.Unmarshal(encoded, &result); err != nil {
		return UniverseSystemData{}, false, err
	}
	return result, true, nil
}

func (r *HomepageReader) UniverseRegion(ctx context.Context, id int64) (UniverseRegionData, bool, error) {
	if r == nil || r.db == nil || id < 1 || id > 2_147_483_647 {
		return UniverseRegionData{}, false, fmt.Errorf("invalid region")
	}
	data, err := cachedHomepage(ctx, r, fmt.Sprintf("universe-region:%d", id), 2*time.Minute, func(ctx context.Context) (map[string]any, error) {
		return loadUniverseRegion(ctx, r.db, id)
	})
	if err != nil || data == nil {
		return UniverseRegionData{}, data != nil, err
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return UniverseRegionData{}, false, err
	}
	var result UniverseRegionData
	if err := json.Unmarshal(encoded, &result); err != nil {
		return UniverseRegionData{}, false, err
	}
	return result, true, nil
}

func (r *HomepageReader) UniverseConstellation(ctx context.Context, id int64) (UniverseConstellationData, bool, error) {
	if r == nil || r.db == nil || id < 1 || id > 2_147_483_647 {
		return UniverseConstellationData{}, false, fmt.Errorf("invalid constellation")
	}
	data, err := cachedHomepage(ctx, r, fmt.Sprintf("universe-constellation:%d", id), 2*time.Minute, func(ctx context.Context) (map[string]any, error) {
		return loadUniverseConstellation(ctx, r.db, id)
	})
	if err != nil || data == nil {
		return UniverseConstellationData{}, data != nil, err
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return UniverseConstellationData{}, false, err
	}
	var result UniverseConstellationData
	if err := json.Unmarshal(encoded, &result); err != nil {
		return UniverseConstellationData{}, false, err
	}
	return result, true, nil
}

func universeLocationRoute(kind string) (locationKilllistRoute, bool) {
	for _, route := range locationKilllistRoutes {
		if route.Name == kind {
			return route, true
		}
	}
	return locationKilllistRoute{}, false
}

func (r *HomepageReader) UniverseLocationKills(ctx context.Context, kind string, id int64, limit int, after *int64) (KilllistPage, error) {
	route, ok := universeLocationRoute(kind)
	if r == nil || r.db == nil || !ok || id < 1 || id > 2_147_483_647 || limit < 10 || limit > 100 || after != nil && *after < 1 {
		return KilllistPage{}, fmt.Errorf("invalid location killlist")
	}
	var cursor int64
	if after != nil {
		cursor = *after
	}
	key := fmt.Sprintf("universe-location-kills:%s:%d:%d:%d", kind, id, limit, cursor)
	return cachedHomepage(ctx, r, key, time.Minute, func(ctx context.Context) (KilllistPage, error) {
		payload, err := loadCursorLocationKilllist(ctx, r.db, route, id, after, limit)
		if err != nil {
			return KilllistPage{}, err
		}
		return decodeKilllistPayload(payload)
	})
}

func (r *HomepageReader) UniverseLocationValuable(ctx context.Context, kind string, id int64, category string) ([]ValuableKill, error) {
	route, ok := universeLocationRoute(kind)
	if r == nil || r.db == nil || !ok || id < 1 || id > 2_147_483_647 || category != "kills" && category != "ships" && category != "structures" {
		return nil, fmt.Errorf("invalid location valuable kills")
	}
	key := fmt.Sprintf("universe-location-valuable:%s:%d:%s", kind, id, category)
	return cachedHomepage(ctx, r, key, 5*time.Minute, func(ctx context.Context) ([]ValuableKill, error) {
		query := url.Values{"limit": {"8"}, "days": {"7"}}
		if category != "kills" {
			query.Set("dataType", "most_valuable_"+category)
		}
		payload, err := locationMostValuableHandler(Options{DB: r.db}, route)(ctx, &legacyRequest{Query: query, Params: map[string]string{"id": strconv.FormatInt(id, 10)}})
		if err != nil {
			return nil, err
		}
		result, err := entityConflictDecode[struct {
			Entries []ValuableKill `json:"entries"`
		}](payload)
		return result.Entries, err
	})
}
