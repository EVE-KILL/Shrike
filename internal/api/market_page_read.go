package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Market page reads share the same order and history handlers as the public
// API, including its region/security semantics, without an HTTP round trip.
type UniverseMarketOrder struct {
	OrderID      int64     `json:"order_id"`
	VolumeRemain int64     `json:"volume_remain"`
	MinVolume    int64     `json:"min_volume"`
	Price        float64   `json:"price"`
	LocationName string    `json:"location_name"`
	RegionName   string    `json:"region_name"`
	SystemID     int64     `json:"system_id"`
	Security     *float64  `json:"security"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type UniverseMarketRegion struct {
	ID   int64  `json:"region_id"`
	Name string `json:"name"`
}

type UniverseMarketOrders struct {
	SnapshotAt *time.Time             `json:"snapshot_at"`
	RegionID   int32                  `json:"region_id"`
	Security   []string               `json:"security"`
	Sellers    []UniverseMarketOrder  `json:"sellers"`
	Buyers     []UniverseMarketOrder  `json:"buyers"`
	Regions    []UniverseMarketRegion `json:"regions"`
}

type UniverseMarketHistoryPoint struct {
	Date    time.Time `json:"date"`
	Average *float64  `json:"average"`
	Volume  *int64    `json:"volume"`
}

type UniverseMarketHistory struct {
	RegionID int32                        `json:"region_id"`
	Days     int                          `json:"days"`
	History  []UniverseMarketHistoryPoint `json:"history"`
}

func decodeLegacyPayload[T any](payload legacyPayload) (T, error) {
	var result T
	encoded, err := json.Marshal(payload.Body)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(encoded, &result)
	return result, err
}

func (r *HomepageReader) UniverseMarketOrders(ctx context.Context, id int64, region int32, security string) (UniverseMarketOrders, error) {
	if r == nil || r.db == nil || id < 1 || id > 2_147_483_647 || region < 0 {
		return UniverseMarketOrders{}, fmt.Errorf("invalid market item")
	}
	if _, err := marketSecuritySQL(security, "system.security"); err != nil {
		return UniverseMarketOrders{}, err
	}
	security = joinMarketSecurity(normalizedMarketSecurity(security))
	key := fmt.Sprintf("universe-market-orders:%d:%d:%s", id, region, security)
	return cachedHomepage(ctx, r, key, time.Minute, func(ctx context.Context) (UniverseMarketOrders, error) {
		query := url.Values{"region_id": {strconv.FormatInt(int64(region), 10)}, "security": {security}, "limit": {"100"}}
		payload, err := marketItemOrdersHandler(Options{DB: r.db})(ctx, &legacyRequest{Params: map[string]string{"id": strconv.FormatInt(id, 10)}, Query: query})
		if err != nil {
			return UniverseMarketOrders{}, err
		}
		return decodeLegacyPayload[UniverseMarketOrders](payload)
	})
}

func joinMarketSecurity(values []string) string {
	result := ""
	for _, value := range values {
		if result != "" {
			result += ","
		}
		result += value
	}
	return result
}

func (r *HomepageReader) UniverseMarketHistory(ctx context.Context, id int64, region int32, days int) (UniverseMarketHistory, error) {
	if r == nil || r.db == nil || id < 1 || id > 2_147_483_647 || region < 0 || days != 30 && days != 90 {
		return UniverseMarketHistory{}, fmt.Errorf("invalid market history")
	}
	key := fmt.Sprintf("universe-market-history:%d:%d:%d", id, region, days)
	return cachedHomepage(ctx, r, key, 10*time.Minute, func(ctx context.Context) (UniverseMarketHistory, error) {
		query := url.Values{"region_id": {strconv.FormatInt(int64(region), 10)}, "days": {strconv.Itoa(days)}}
		payload, err := marketItemHistoryHandler(Options{DB: r.db})(ctx, &legacyRequest{Params: map[string]string{"id": strconv.FormatInt(id, 10)}, Query: query})
		if err != nil {
			return UniverseMarketHistory{}, err
		}
		return decodeLegacyPayload[UniverseMarketHistory](payload)
	})
}
