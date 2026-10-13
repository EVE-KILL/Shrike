package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"
)

// StatisticsEntry retains the richer population-ranking fields that are not
// part of the homepage's TopEntry contract.
type StatisticsEntry struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Type          string     `json:"type"`
	Count         float64    `json:"count"`
	ISK           float64    `json:"isk"`
	Security      float64    `json:"sec"`
	Members       int64      `json:"member_count"`
	Delta1        int64      `json:"delta_1d"`
	Delta7        int64      `json:"delta_7d"`
	Delta30       int64      `json:"delta_30d"`
	Delta         int64      `json:"delta"`
	AverageSec    float64    `json:"avg_sec_status"`
	WeightedScore float64    `json:"weighted_score"`
	Founded       *time.Time `json:"date_founded"`
	TotalPoints   float64    `json:"total_points"`
	AveragePoints float64    `json:"avg_points"`
}

type StatisticsQuery struct {
	DataType   string
	Section    string
	EntityType string
	Rank       string
	Direction  string
	Days       float64
}

func (r *HomepageReader) StatisticsSite(ctx context.Context, host string) (SiteConfigurationResponse, error) {
	if r == nil || r.db == nil {
		return SiteConfigurationResponse{}, fmt.Errorf("statistics database unavailable")
	}
	service := siteConfigurationService{db: r.db, cache: r.cache, commit: r.commit}
	result, _, err := service.resolve(ctx, normalizeRequestHost(host))
	return result, err
}

func statisticsQueryValues(query StatisticsQuery) (url.Values, error) {
	if query.Days <= 0 || query.Days > 90 || math.IsNaN(query.Days) || math.IsInf(query.Days, 0) {
		return nil, fmt.Errorf("invalid statistics period")
	}
	values := url.Values{"limit": {"10"}, "days": {strconv.FormatFloat(query.Days, 'g', -1, 64)}}
	if query.DataType != "" {
		if _, ok := validGlobalStatsTypes[query.DataType]; !ok || query.Section != "" {
			return nil, fmt.Errorf("invalid statistics type")
		}
		values.Set("dataType", query.DataType)
		return values, nil
	}
	if query.EntityType != "alliance" && query.EntityType != "corporation" && query.EntityType != "character" {
		return nil, fmt.Errorf("invalid statistics entity")
	}
	switch query.Section {
	case "largest":
		if query.EntityType == "character" {
			return nil, fmt.Errorf("invalid largest statistics entity")
		}
	case "security":
		if query.EntityType == "character" || query.Rank != "pirate" && query.Rank != "carebear" {
			return nil, fmt.Errorf("invalid security statistics ranking")
		}
		values.Set("rank", query.Rank)
	case "growth":
		if query.EntityType == "character" || query.Direction != "growing" && query.Direction != "shrinking" {
			return nil, fmt.Errorf("invalid growth statistics direction")
		}
		values.Set("direction", query.Direction)
	case "newest", "achievements":
	default:
		return nil, fmt.Errorf("invalid population statistics section")
	}
	values.Set("section", query.Section)
	values.Set("entityType", query.EntityType)
	return values, nil
}

func statisticsBoardCacheName(site SiteConfigurationResponse, values url.Values) string {
	scope := "global"
	if domain := site.Domain; domain != nil {
		// A scope change cannot reuse a previous board's result, even during
		// the site configuration's cache lifetime.
		entities, _ := json.Marshal(domain.EntityIDs)
		scope = fmt.Sprintf("domain:%d:%x", domain.ID, sha256.Sum256(entities))
	}
	return "statistics:v2:" + scope + ":" + values.Encode()
}

// StatisticsBoardsCached reads warm cards in one cache batch without starting
// cold queries or background work. Missing cards load through HTML fragments.
func (r *HomepageReader) StatisticsBoardsCached(ctx context.Context, site SiteConfigurationResponse, queries []StatisticsQuery) map[int][]StatisticsEntry {
	result := map[int][]StatisticsEntry{}
	if r == nil || r.cache == nil {
		return result
	}
	keys := []string{}
	indexes := map[int]string{}
	for i, query := range queries {
		values, err := statisticsQueryValues(query)
		if err != nil || site.Domain != nil && (query.Section != "" || !hasDomainStatisticsType(query.DataType)) {
			continue
		}
		key := r.cacheKey(statisticsBoardCacheName(site, values))
		keys = append(keys, key)
		indexes[i] = key
	}
	entries := r.cache.LoadMany(ctx, keys)
	for i, key := range indexes {
		if entry, ok := entries[key]; ok {
			var rows []StatisticsEntry
			if json.Unmarshal(entry.Body, &rows) == nil {
				result[i] = rows
			}
		}
	}
	return result
}

func (r *HomepageReader) StatisticsBoard(ctx context.Context, site SiteConfigurationResponse, query StatisticsQuery) ([]StatisticsEntry, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("statistics database unavailable")
	}
	values, err := statisticsQueryValues(query)
	if err != nil {
		return nil, err
	}
	if site.Domain != nil && (query.Section != "" || !hasDomainStatisticsType(query.DataType)) {
		return nil, apiError(404, "Population rankings are global only")
	}
	ttl := time.Minute
	if query.Days < 1 {
		ttl = 30 * time.Second
	}
	return cachedHomepage(ctx, r, statisticsBoardCacheName(site, values), ttl, func(ctx context.Context) ([]StatisticsEntry, error) {
		var payload legacyPayload
		var err error
		if domain := site.Domain; domain != nil {
			// Retain /custom/stats' period parsing and precomputed scope rules.
			req := &legacyRequest{Query: values}
			days := boundedQueryInt(req, "days", 7, 1, 90)
			scope := &domainEntityScope{Characters: domain.EntityIDs.CharacterIDs, Corporations: domain.EntityIDs.CorporationIDs, Alliances: domain.EntityIDs.AllianceIDs}
			var rows []map[string]any
			rows, err = loadDomainStatistics(ctx, r.db, scope, query.DataType, time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02"), 10)
			if err == nil {
				rows, err = attachGlobalStatsPalettes(ctx, r.db, rows)
			}
			payload = jsonPayload(map[string]any{"entries": nonNilUniverseRows(rows)})
		} else if query.DataType != "" {
			payload, err = globalStatisticsHandler(Options{DB: r.db})(ctx, &legacyRequest{Query: values})
		} else {
			payload, err = statsRankingsHandler(Options{DB: r.db})(ctx, &legacyRequest{Query: values})
		}
		if err != nil {
			return nil, err
		}
		var result struct {
			Entries []StatisticsEntry `json:"entries"`
		}
		payload.Body = normalizeJSON(payload.Body)
		if err = decodeLegacyPagePayload(payload, &result); err != nil {
			return nil, err
		}
		return result.Entries, nil
	})
}
