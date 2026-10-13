package api

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type RankingEntry struct {
	EntityID          int64  `json:"entity_id"`
	Name              string `json:"name"`
	EveKillRating     int64  `json:"eve_kill_rating"`
	CombatPoints      int64  `json:"combat_points"`
	AchievementPoints int64  `json:"achievement_points"`
	OverallRank       int64  `json:"overall_rank"`
}

func (r *HomepageReader) Rankings(ctx context.Context, entityType, window string) ([]RankingEntry, error) {
	allowedEntities := map[string]bool{"character": true, "corporation": true, "alliance": true, "ship": true, "system": true, "region": true}
	allowedWindows := map[string]bool{"hourly": true, "weekly": true, "fourteen_days": true, "thirty_days": true, "ninety_days": true, "one_eighty_days": true, "one_year": true, "all_time": true}
	if !allowedEntities[entityType] || !allowedWindows[window] {
		return nil, fmt.Errorf("invalid rankings selection")
	}
	ttl := time.Minute
	if window == "hourly" {
		ttl = 30 * time.Second
	}
	return cachedHomepage(ctx, r, fmt.Sprintf("rankings:%s:%s", entityType, window), ttl, func(ctx context.Context) ([]RankingEntry, error) {
		query := url.Values{"section": {"eve-kill"}, "entityType": {entityType}, "window": {window}, "limit": {"50"}}
		payload, err := statsRankingsHandler(Options{DB: r.db})(ctx, &legacyRequest{Query: query})
		if err != nil {
			return nil, err
		}
		var result struct {
			Entries []RankingEntry `json:"entries"`
		}
		if err := decodeLegacyPagePayload(payload, &result); err != nil {
			return nil, err
		}
		return result.Entries, nil
	})
}
