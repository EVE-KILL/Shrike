package api

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// AdvancedSearchFit is the existing FitPreview contract shared by search and
// later fitting catalogue pages. The slot and drone types are already used by
// the item fitting reader.
type AdvancedSearchFit struct {
	KillmailID    int64               `json:"killmail_id"`
	KillmailTime  time.Time           `json:"killmail_time"`
	ShipTypeID    int64               `json:"victim_ship_type_id"`
	ShipName      string              `json:"victim_ship_name"`
	TotalValue    float64             `json:"total_value"`
	AttackerCount int                 `json:"attacker_count"`
	Modules       []UniverseFitModule `json:"modules"`
	Drones        []UniverseFitDrone  `json:"drones"`
	Count         int                 `json:"count"`
	Hash          string              `json:"hash"`
	DedupMode     string              `json:"dedup_mode"`
}

type AdvancedSearchResult struct {
	Kills   []KilllistRow       `json:"kills"`
	Fits    []AdvancedSearchFit `json:"fits"`
	HasMore bool                `json:"hasMore"`
	Cursor  *int64              `json:"cursor"`
}

// AdvancedSearch delegates to the established query parser and loaders. Page
// and API reads have the same filters, sort cursor, fitting grouping, and TTL.
func (r *HomepageReader) AdvancedSearch(ctx context.Context, query url.Values) (AdvancedSearchResult, error) {
	if r == nil || r.db == nil {
		return AdvancedSearchResult{}, fmt.Errorf("search database is not configured")
	}
	return cachedHomepage(ctx, r, "advanced-search:"+query.Encode(), advancedKilllistCacheTTL, func(ctx context.Context) (AdvancedSearchResult, error) {
		payload, err := advancedKilllistHandler(Options{DB: r.db})(ctx, &legacyRequest{Query: query})
		if err != nil {
			return AdvancedSearchResult{}, err
		}
		return decodeLegacyPayload[AdvancedSearchResult](payload)
	})
}
