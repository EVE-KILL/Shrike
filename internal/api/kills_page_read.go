package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/eve-kill/shrike/internal/killtype"
)

const (
	KillCategoryTopDays  = 7
	KillCategoryTopLimit = 10
)

func killsTopCacheName(kind, dataType string, days, limit int) string {
	return fmt.Sprintf("kills-top:%s:%s:%d:%d", kind, dataType, days, limit)
}

// KillCategoryTopCached returns the last successful boards for the initial page.
func (r *HomepageReader) KillCategoryTopCached(ctx context.Context, kind string) map[string][]TopEntry {
	if r == nil {
		return nil
	}
	boards, _ := r.preparedKillCategoryTop(ctx, kind)
	return boards
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
	if days == KillCategoryTopDays && limit == KillCategoryTopLimit {
		boards, err := r.preparedKillCategoryTop(ctx, kind)
		if err != nil {
			return nil, err
		}
		entries, ok := boards[dataType]
		if !ok {
			return nil, apiError(http.StatusServiceUnavailable, "Leaderboard is being prepared")
		}
		return entries, nil
	}
	key := killsTopCacheName(kind, dataType, days, limit)
	return cachedHomepage(ctx, r, key, 5*time.Minute, func(ctx context.Context) ([]TopEntry, error) {
		return loadTopKillEntries(ctx, r.db, kind, dataType, limit, time.Now().UTC().AddDate(0, 0, -days))
	})
}

func loadTopKillEntries(ctx context.Context, db Database, kind, dataType string, limit int, since time.Time) ([]TopEntry, error) {
	rows, err := loadTopKills(ctx, db, kind, dataType, limit, since)
	if err != nil {
		return nil, err
	}
	rows, err = attachGlobalStatsPalettes(ctx, db, rows)
	if err != nil {
		return nil, err
	}
	return decodeHomepageRows[TopEntry](nonNilUniverseRows(rows))
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
