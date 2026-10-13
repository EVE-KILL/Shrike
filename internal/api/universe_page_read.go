package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type UniverseGroup struct {
	ID                 int64  `json:"group_id"`
	Name               string `json:"name"`
	CategoryID         int64  `json:"category_id"`
	CategoryName       string `json:"category_name"`
	Published          bool   `json:"published"`
	TypeCount          int64  `json:"type_count"`
	PublishedTypeCount int64  `json:"published_type_count"`
}

type UniverseGroupType struct {
	ID            int64  `json:"type_id"`
	Name          string `json:"name"`
	Published     bool   `json:"published"`
	MetaGroupName string `json:"meta_group_name"`
}

type UniverseGroupData struct {
	Group UniverseGroup       `json:"group"`
	Types []UniverseGroupType `json:"types"`
}

type UniverseItem struct {
	ID             int64   `json:"type_id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	GroupID        int64   `json:"group_id"`
	GroupName      string  `json:"group_name"`
	CategoryID     int64   `json:"category_id"`
	CategoryName   string  `json:"category_name"`
	MetaGroupName  string  `json:"meta_group_name"`
	IsShip         bool    `json:"is_ship"`
	Published      bool    `json:"published"`
	Volume         float64 `json:"volume"`
	PackagedVolume float64 `json:"packaged_volume"`
	Mass           float64 `json:"mass"`
	Capacity       float64 `json:"capacity"`
	BasePrice      float64 `json:"base_price"`
	PortionSize    int64   `json:"portion_size"`
	Radius         float64 `json:"radius"`
	MetaLevel      int64   `json:"meta_level"`
	TechLevel      int64   `json:"tech_level"`
}

type UniverseAttribute struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type UniverseSkill struct {
	ID    int64  `json:"type_id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type UniverseMaterial struct {
	ID       int64  `json:"type_id"`
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

type UniverseVariation struct {
	ID            int64  `json:"type_id"`
	Name          string `json:"name"`
	MetaGroupName string `json:"meta_group_name"`
}

type UniverseMarketCrumb struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type UniversePriceSummary struct {
	Latest       float64 `json:"latest"`
	LatestDate   string  `json:"latest_date"`
	Average90D   float64 `json:"average_90d"`
	Highest90D   float64 `json:"highest_90d"`
	Lowest90D    float64 `json:"lowest_90d"`
	AvgVolume90D int64   `json:"avg_volume_90d"`
}

type UniversePricePoint struct {
	Date    string  `json:"date"`
	Price   float64 `json:"price"`
	Average float64 `json:"average"`
	Volume  int64   `json:"volume"`
}

type UniverseInsurance struct {
	LevelName string  `json:"level_name"`
	Cost      float64 `json:"cost"`
	Payout    float64 `json:"payout"`
}

type UniverseItemData struct {
	Item             UniverseItem                   `json:"item"`
	ShipAttributes   map[string][]UniverseAttribute `json:"shipAttributes"`
	Attributes       []UniverseAttribute            `json:"attributes"`
	RequiredSkills   []UniverseSkill                `json:"requiredSkills"`
	Materials        []UniverseMaterial             `json:"materials"`
	Variations       []UniverseVariation            `json:"variations"`
	MarketBreadcrumb []UniverseMarketCrumb          `json:"marketBreadcrumb"`
	Pricing          struct {
		Summary       *UniversePriceSummary `json:"summary"`
		CustomSummary *UniversePriceSummary `json:"customSummary"`
		History       []UniversePricePoint  `json:"history"`
		CustomHistory []UniversePricePoint  `json:"customHistory"`
		Insurance     []UniverseInsurance   `json:"insurance"`
	} `json:"pricing"`
}

func (r *HomepageReader) UniverseItem(ctx context.Context, id int64) (UniverseItemData, bool, error) {
	if r == nil || r.db == nil || id < 1 || id > 2_147_483_647 {
		return UniverseItemData{}, false, fmt.Errorf("invalid inventory item")
	}
	data, err := cachedHomepage(ctx, r, fmt.Sprintf("universe-item:%d", id), 2*time.Minute, func(ctx context.Context) (map[string]any, error) {
		return loadUniverseType(ctx, r.db, id)
	})
	if err != nil || data == nil {
		return UniverseItemData{}, data != nil, err
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return UniverseItemData{}, false, err
	}
	var result UniverseItemData
	if err := json.Unmarshal(encoded, &result); err != nil {
		return UniverseItemData{}, false, err
	}
	return result, true, nil
}

func (r *HomepageReader) UniverseItemKills(ctx context.Context, id int64, ship bool, limit int, after *int64) (KilllistPage, error) {
	if r == nil || r.db == nil || id < 1 || id > 2_147_483_647 || limit < 5 || limit > 100 || after != nil && *after < 1 {
		return KilllistPage{}, fmt.Errorf("invalid inventory item kills")
	}
	var cursor int64
	if after != nil {
		cursor = *after
	}
	key := fmt.Sprintf("universe-item-kills:%d:%t:%d:%d", id, ship, limit, cursor)
	return cachedHomepage(ctx, r, key, time.Minute, func(ctx context.Context) (KilllistPage, error) {
		query, args := universeItemKilllistQuery(id, after, limit, !ship)
		rows, err := queryMaps(ctx, r.db, query, args...)
		if err != nil {
			return KilllistPage{}, err
		}
		rows, hasMore, next, err := finishUniverseKilllist(ctx, r.db, rows, limit)
		if err != nil {
			return KilllistPage{}, err
		}
		payload := jsonPayload(map[string]any{"kills": rows, "hasMore": hasMore, "cursor": next})
		return decodeKilllistPayload(payload)
	})
}

func (r *HomepageReader) UniverseGroup(ctx context.Context, id int64) (UniverseGroupData, bool, error) {
	if r == nil || r.db == nil || id < 1 || id > 2_147_483_647 {
		return UniverseGroupData{}, false, fmt.Errorf("invalid inventory group")
	}
	data, err := cachedHomepage(ctx, r, fmt.Sprintf("universe-group:%d", id), 2*time.Minute, func(ctx context.Context) (map[string]any, error) {
		return loadUniverseGroup(ctx, r.db, id)
	})
	if err != nil || data == nil {
		return UniverseGroupData{}, data != nil, err
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return UniverseGroupData{}, false, err
	}
	var result UniverseGroupData
	if err := json.Unmarshal(encoded, &result); err != nil {
		return UniverseGroupData{}, false, err
	}
	return result, true, nil
}

func (r *HomepageReader) UniverseGroupKills(ctx context.Context, id int64, limit int, after *int64) (KilllistPage, error) {
	if r == nil || r.db == nil || id < 1 || id > 2_147_483_647 || limit < 10 || limit > 100 || after != nil && *after < 1 {
		return KilllistPage{}, fmt.Errorf("invalid inventory group kills")
	}
	var cursor int64
	if after != nil {
		cursor = *after
	}
	key := fmt.Sprintf("universe-group-kills:%d:%d:%d", id, limit, cursor)
	return cachedHomepage(ctx, r, key, time.Minute, func(ctx context.Context) (KilllistPage, error) {
		filters := map[string]any{
			"entities":  map[string]any{"victim": []map[string]any{{"id": id, "type": "shipgroup"}}},
			"timeRange": map[string]any{"preset": "90d"},
		}
		encoded, err := json.Marshal(filters)
		if err != nil {
			return KilllistPage{}, err
		}
		query := url.Values{"filters": {string(encoded)}, "limit": {strconv.Itoa(limit)}}
		if after != nil {
			query.Set("after", strconv.FormatInt(*after, 10))
		}
		payload, err := advancedKilllistHandler(Options{DB: r.db})(ctx, &legacyRequest{Query: query})
		if err != nil {
			return KilllistPage{}, err
		}
		return decodeKilllistPayload(payload)
	})
}
