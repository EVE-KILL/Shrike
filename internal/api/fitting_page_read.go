package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type UniverseFitModule struct {
	SlotGroup    int    `json:"slot_group"`
	Ordinal      int    `json:"ordinal"`
	TypeID       int64  `json:"type_id"`
	Name         string `json:"name"`
	ChargeTypeID int64  `json:"charge_type_id"`
	ChargeName   string `json:"charge_name"`
}

type UniverseFitDrone struct {
	TypeID   int64  `json:"type_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type UniverseFitAlliance struct {
	ID                  int64   `json:"alliance_id"`
	Name                string  `json:"name"`
	Uses                int     `json:"uses"`
	PctOfAllianceLosses float64 `json:"pct_of_alliance_losses"`
}

type UniverseFitSecurityContext struct {
	Name string  `json:"name"`
	Pct  float64 `json:"pct"`
}
type UniverseFitRegionContext struct {
	ID   int64   `json:"region_id"`
	Name string  `json:"name"`
	Pct  float64 `json:"pct"`
}
type UniverseFitContext struct {
	SecurityDistribution []UniverseFitSecurityContext `json:"security_distribution"`
	TopRegion            *UniverseFitRegionContext    `json:"top_region"`
	MedianAttackers      float64                      `json:"median_attackers"`
	MedianLossValue      float64                      `json:"median_loss_value"`
}

type UniverseFitStats struct {
	EHP        *float64 `json:"ehp"`
	DPS        *float64 `json:"dps"`
	Alpha      *float64 `json:"alpha"`
	Speed      *float64 `json:"speed"`
	Align      *float64 `json:"align"`
	Repair     *float64 `json:"repair"`
	NPCEHP     *float64 `json:"npc_ehp"`
	NPCProfile string   `json:"npc_profile"`
}

type UniverseFitFamily struct {
	FamilyHash       string                `json:"family_hash"`
	CanonicalFitHash string                `json:"canonical_fit_hash"`
	TotalUses        int                   `json:"total_uses"`
	CanonicalUses    int                   `json:"canonical_uses"`
	VariantCount     int                   `json:"variant_count"`
	LastUsed         time.Time             `json:"last_used"`
	FitCost          float64               `json:"fit_cost"`
	Modules          []UniverseFitModule   `json:"modules"`
	Drones           []UniverseFitDrone    `json:"drones"`
	TopAlliances     []UniverseFitAlliance `json:"top_alliances"`
	Context          *UniverseFitContext   `json:"context"`
	Stats            *UniverseFitStats     `json:"stats"`
}

type UniverseFitFamilies struct {
	ShipTypeID int64               `json:"ship_type_id"`
	WindowDays int                 `json:"window_days"`
	IsRareHull bool                `json:"is_rare_hull"`
	HullCost   *float64            `json:"hull_cost"`
	Families   []UniverseFitFamily `json:"families"`
}

type UniverseFitMetaGroup struct {
	ID        int64   `json:"group_id"`
	Name      string  `json:"name"`
	KillCount int64   `json:"kill_count"`
	Pct       float64 `json:"pct"`
}
type UniverseFitMeta struct {
	ShipTypeID int64                  `json:"ship_type_id"`
	WindowDays int                    `json:"window_days"`
	TotalKills int64                  `json:"total_kills"`
	Groups     []UniverseFitMetaGroup `json:"groups"`
}
type UniverseFitDistributionBucket struct {
	Bucket           int     `json:"bucket"`
	LowerBound       float64 `json:"lower_bound"`
	UpperBound       float64 `json:"upper_bound"`
	ObservationCount int64   `json:"observation_count"`
}
type UniverseFitDistributionMetric struct {
	Metric           string                          `json:"metric"`
	ObservationCount int64                           `json:"observation_count"`
	Median           float64                         `json:"median"`
	P10              float64                         `json:"p10"`
	P90              float64                         `json:"p90"`
	Buckets          []UniverseFitDistributionBucket `json:"buckets"`
}
type UniverseFitDistributions struct {
	ShipTypeID int64                           `json:"ship_type_id"`
	WindowDays int                             `json:"window_days"`
	Metrics    []UniverseFitDistributionMetric `json:"metrics"`
}

func (r *HomepageReader) UniverseFitFamilies(ctx context.Context, shipID int64, filters url.Values) (UniverseFitFamilies, error) {
	if r == nil || r.db == nil || shipID < 1 || shipID > 2_147_483_647 {
		return UniverseFitFamilies{}, fmt.Errorf("invalid ship type")
	}
	key := "universe-fit-families:" + strconv.FormatInt(shipID, 10) + ":" + filters.Encode()
	return cachedHomepage(ctx, r, key, 5*time.Minute, func(ctx context.Context) (UniverseFitFamilies, error) {
		payload, err := shipFittingFamiliesHandler(Options{DB: r.db})(ctx, &legacyRequest{Params: map[string]string{"id": strconv.FormatInt(shipID, 10)}, Query: filters})
		if err != nil {
			return UniverseFitFamilies{}, err
		}
		return decodeLegacyPayload[UniverseFitFamilies](payload)
	})
}

func (r *HomepageReader) UniverseFitMeta(ctx context.Context, shipID int64, filters url.Values) (UniverseFitMeta, error) {
	if r == nil || r.db == nil || shipID < 1 || shipID > 2_147_483_647 {
		return UniverseFitMeta{}, fmt.Errorf("invalid ship type")
	}
	key := "universe-fit-meta:" + strconv.FormatInt(shipID, 10) + ":" + filters.Encode()
	return cachedHomepage(ctx, r, key, 5*time.Minute, func(ctx context.Context) (UniverseFitMeta, error) {
		payload, err := shipFittingMetadataHandler(Options{DB: r.db})(ctx, &legacyRequest{Params: map[string]string{"id": strconv.FormatInt(shipID, 10)}, Query: filters})
		if err != nil {
			return UniverseFitMeta{}, err
		}
		return decodeLegacyPayload[UniverseFitMeta](payload)
	})
}

func (r *HomepageReader) UniverseFitDistributions(ctx context.Context, shipID int64, filters url.Values) (UniverseFitDistributions, error) {
	if r == nil || r.db == nil || shipID < 1 || shipID > 2_147_483_647 {
		return UniverseFitDistributions{}, fmt.Errorf("invalid ship type")
	}
	query := url.Values{}
	for key, values := range filters {
		query[key] = append([]string(nil), values...)
	}
	query.Set("days", "90")
	key := "universe-fit-distributions:" + strconv.FormatInt(shipID, 10) + ":" + query.Encode()
	return cachedHomepage(ctx, r, key, 30*time.Minute, func(ctx context.Context) (UniverseFitDistributions, error) {
		payload, err := shipFittingDistributionsHandler(Options{DB: r.db})(ctx, &legacyRequest{Params: map[string]string{"id": strconv.FormatInt(shipID, 10)}, Query: query})
		if err != nil {
			return UniverseFitDistributions{}, err
		}
		return decodeLegacyPayload[UniverseFitDistributions](payload)
	})
}
