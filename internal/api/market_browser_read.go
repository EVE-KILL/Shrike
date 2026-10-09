package api

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

type UniverseMarketTreeNode struct {
	ID       int64                    `json:"id"`
	Name     string                   `json:"name"`
	Slug     string                   `json:"slug"`
	ParentID *int64                   `json:"parent_id"`
	HasTypes bool                     `json:"has_types"`
	IconID   *int64                   `json:"icon_id"`
	Children []UniverseMarketTreeNode `json:"children"`
}

type UniverseMarketGroupItem struct {
	ID              int64    `json:"type_id"`
	Name            string   `json:"name"`
	GroupID         int64    `json:"group_id"`
	MarketGroupID   int64    `json:"market_group_id"`
	CategoryID      int64    `json:"category_id"`
	MetaGroupID     *int64   `json:"meta_group_id"`
	IsShip          bool     `json:"is_ship"`
	UniverseAverage *float64 `json:"universe_average"`
	UniverseVolume  *int64   `json:"universe_volume"`
	JitaSell        *float64 `json:"jita_sell"`
}

func (r *HomepageReader) UniverseMarketTree(ctx context.Context) ([]UniverseMarketTreeNode, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("market tree reader unavailable")
	}
	return cachedHomepage(ctx, r, "universe-market-tree", 6*time.Hour, func(ctx context.Context) ([]UniverseMarketTreeNode, error) {
		payload, err := marketTreeHandler(Options{DB: r.db})(ctx, &legacyRequest{})
		if err != nil {
			return nil, err
		}
		var decoded struct {
			Groups []UniverseMarketTreeNode `json:"groups"`
		}
		decoded, err = decodeLegacyPayload[struct {
			Groups []UniverseMarketTreeNode `json:"groups"`
		}](payload)
		return decoded.Groups, err
	})
}

func (r *HomepageReader) UniverseMarketGroupItems(ctx context.Context, id int64) ([]UniverseMarketGroupItem, error) {
	if r == nil || r.db == nil || id < 1 || id > 2_147_483_647 {
		return nil, fmt.Errorf("invalid market group")
	}
	key := "universe-market-group-items:" + strconv.FormatInt(id, 10)
	return cachedHomepage(ctx, r, key, 6*time.Hour, func(ctx context.Context) ([]UniverseMarketGroupItem, error) {
		payload, err := marketGroupItemsHandler(Options{DB: r.db})(ctx, &legacyRequest{Params: map[string]string{"id": strconv.FormatInt(id, 10)}})
		if err != nil {
			return nil, err
		}
		decoded, err := decodeLegacyPayload[struct {
			Items []UniverseMarketGroupItem `json:"items"`
		}](payload)
		return decoded.Items, err
	})
}
