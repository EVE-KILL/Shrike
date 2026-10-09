package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/killtype"
)

func killsTopCacheName(kind, dataType string, days, limit int) string {
	return fmt.Sprintf("kills-top:%s:%s:%d:%d", kind, dataType, days, limit)
}

// TopKillsCached is a nonblocking read for page assembly. Cold or expired
// leaderboards are fetched through their HTML fragments after the main page
// has rendered; a warm one remains part of the initial HTML.
func (r *HomepageReader) TopKillsCached(ctx context.Context, kind, dataType string, days, limit int) ([]TopEntry, bool) {
	if r == nil || r.cache == nil {
		return nil, false
	}
	key := r.cacheKey(killsTopCacheName(kind, dataType, days, limit))
	entry, fresh, ok := r.cache.LoadState(ctx, key)
	if !ok || !fresh {
		return nil, false
	}
	var result []TopEntry
	if err := json.Unmarshal(entry.Body, &result); err != nil {
		return nil, false
	}
	return result, true
}

// TopKills and MostValuableKills are the typed reads for /kills/{type} pages.
// They call the same query helpers as the JSON endpoints and reuse the shared
// homepage response cache rather than making an internal HTTP request.
func (r *HomepageReader) TopKills(ctx context.Context, kind, dataType string, days, limit int) ([]TopEntry, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("killboard database is not configured")
	}
	if _, known := killtype.Predicates()[kind]; !known ||
		(dataType != "characters" && dataType != "corporations" && dataType != "alliances") ||
		days < 1 || days > 365 || limit < 1 || limit > 50 {
		return nil, fmt.Errorf("invalid killboard top list options")
	}
	key := killsTopCacheName(kind, dataType, days, limit)
	return cachedHomepage(ctx, r, key, 5*time.Minute, func(ctx context.Context) ([]TopEntry, error) {
		rows, err := loadTopKills(ctx, r.db, kind, dataType, limit, time.Now().UTC().AddDate(0, 0, -days))
		if err == nil {
			rows, err = attachGlobalStatsPalettes(ctx, r.db, rows)
		}
		if err != nil {
			return nil, err
		}
		return decodeHomepageRows[TopEntry](rows)
	})
}

func (r *HomepageReader) MostValuableKills(ctx context.Context, kind string, days, limit int) ([]ValuableKill, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("killboard database is not configured")
	}
	if _, known := killtype.Predicates()[kind]; !known || days < 1 || days > 365 || limit < 1 || limit > 20 {
		return nil, fmt.Errorf("invalid killboard most valuable options")
	}
	if kind == "latest" {
		return r.MostValuable(ctx, "kills", days, limit)
	}
	key := fmt.Sprintf("kills-valuable:%s:%d:%d", kind, days, limit)
	return cachedHomepage(ctx, r, key, 5*time.Minute, func(ctx context.Context) ([]ValuableKill, error) {
		rows, err := loadMostValuableKills(ctx, r.db, kind, days, limit)
		if err != nil {
			return nil, err
		}
		return decodeHomepageRows[ValuableKill](rows)
	})
}
