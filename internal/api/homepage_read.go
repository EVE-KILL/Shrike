package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/killtype"
)

// HomepageReader exposes the existing public read paths to HTML pages. API
// handlers and pages use the same SQL and enrichment functions.
type HomepageReader struct {
	db     Database
	graph  GraphDatabase
	cache  *ResponseCache
	commit string
}

func NewHomepageReader(db Database) *HomepageReader {
	return &HomepageReader{db: db}
}

func newCachedHomepageReader(db Database, graph GraphDatabase, cache *ResponseCache, commit string) *HomepageReader {
	if commit == "" {
		commit = "dev"
	}
	return &HomepageReader{db: db, graph: graph, cache: cache, commit: commit}
}

type TopEntry struct {
	ID      int64       `json:"id"`
	Name    string      `json:"name"`
	Count   float64     `json:"count"`
	Type    string      `json:"type"`
	Palette *TopPalette `json:"palette"`
}

type TopPalette struct {
	MainColor      string `json:"main_color"`
	SecondaryColor string `json:"secondary_color"`
	TertiaryColor  string `json:"tertiary_color"`
}

type ValuableKill struct {
	KillmailID            int64   `json:"killmail_id"`
	ShipTypeID            int64   `json:"ship_type_id"`
	ShipName              string  `json:"ship_name"`
	TotalValue            float64 `json:"total_value"`
	VictimCharacterName   string  `json:"victim_character_name"`
	VictimCorporationName string  `json:"victim_corporation_name"`
	VictimAllianceName    string  `json:"victim_alliance_name"`
}

type KilllistRow struct {
	KillmailID               int64     `json:"killmail_id"`
	KillmailTime             time.Time `json:"killmail_time"`
	TotalValue               float64   `json:"total_value"`
	AttackerCount            int       `json:"attacker_count"`
	IsNPC                    bool      `json:"is_npc"`
	IsSolo                   bool      `json:"is_solo"`
	ShipTypeID               int64     `json:"ship_type_id"`
	ShipName                 string    `json:"ship_name"`
	ShipGroupID              int64     `json:"ship_group_id"`
	ShipGroupName            string    `json:"ship_group_name"`
	VictimCharacterID        int64     `json:"victim_character_id"`
	VictimCharacterName      string    `json:"victim_character_name"`
	VictimCorporationID      int64     `json:"victim_corporation_id"`
	VictimCorporationName    string    `json:"victim_corporation_name"`
	VictimAllianceID         int64     `json:"victim_alliance_id"`
	VictimAllianceName       string    `json:"victim_alliance_name"`
	FinalBlowCharacterID     int64     `json:"final_blow_character_id"`
	FinalBlowCharacterName   string    `json:"final_blow_character_name"`
	FinalBlowCorporationID   int64     `json:"final_blow_corporation_id"`
	FinalBlowCorporationName string    `json:"final_blow_corporation_name"`
	FinalBlowAllianceID      int64     `json:"final_blow_alliance_id"`
	FinalBlowAllianceName    string    `json:"final_blow_alliance_name"`
	FinalBlowShipTypeID      int64     `json:"final_blow_ship_type_id"`
	FinalBlowShipName        string    `json:"final_blow_ship_name"`
	SolarSystemID            int64     `json:"solar_system_id"`
	SolarSystemName          string    `json:"solar_system_name"`
	SolarSystemSecurity      *float64  `json:"solar_system_security"`
	RegionID                 int64     `json:"region_id"`
	RegionName               string    `json:"region_name"`
}

type KilllistPage struct {
	Kills      []KilllistRow `json:"kills"`
	HasMore    bool          `json:"hasMore"`
	Cursor     *int64        `json:"cursor"`
	TotalPages int           `json:"totalPages"`
}

func (r *HomepageReader) Top(ctx context.Context, dataType string, days, limit int) ([]TopEntry, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("homepage database is not configured")
	}
	if _, ok := validGlobalStatsTypes[dataType]; !ok || days < 1 || days > 90 || limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid top list options")
	}
	key := fmt.Sprintf("top:%s:%d:%d", dataType, days, limit)
	return cachedHomepage(ctx, r, key, 5*time.Minute, func(ctx context.Context) ([]TopEntry, error) {
		since := time.Now().AddDate(0, 0, -days).UTC().Format("2006-01-02")
		rows, err := loadGlobalTopList(ctx, r.db, dataType, since, limit)
		if err == nil {
			rows, err = attachGlobalStatsPalettes(ctx, r.db, rows)
		}
		if err != nil {
			return nil, err
		}
		return decodeHomepageRows[TopEntry](rows)
	})
}

func (r *HomepageReader) MostValuable(ctx context.Context, kind string, days, limit int) ([]ValuableKill, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("homepage database is not configured")
	}
	if kind != "kills" && kind != "ships" && kind != "structures" || days < 1 || days > 90 || limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid valuable kills options")
	}
	key := fmt.Sprintf("valuable:%s:%d:%d", kind, days, limit)
	return cachedHomepage(ctx, r, key, 5*time.Minute, func(ctx context.Context) ([]ValuableKill, error) {
		rows, err := loadMostValuable(ctx, r.db, "most_valuable_"+kind, time.Now().AddDate(0, 0, -days), limit)
		if err != nil {
			return nil, err
		}
		return decodeHomepageRows[ValuableKill](rows)
	})
}

func (r *HomepageReader) LatestKills(ctx context.Context, page, limit int) (KilllistPage, error) {
	return r.Killlist(ctx, "latest", page, limit)
}

// Killlist reads any validated /kills/{type} subset through the same rollup
// and enrichment path as the public killlist API.
func (r *HomepageReader) Killlist(ctx context.Context, kind string, page, limit int) (KilllistPage, error) {
	if err := r.validateKilllist(kind, page, limit); err != nil {
		return KilllistPage{}, err
	}
	key := fmt.Sprintf("killlist:%s:%d:%d", kind, page, limit)
	return cachedHomepage(ctx, r, key, time.Minute, func(ctx context.Context) (KilllistPage, error) {
		return r.loadKilllist(ctx, kind, page, limit)
	})
}

// LatestKillsFresh bypasses a one-minute cache entry after a live notification.
// Limit explicit refreshes to one per ten seconds per key and process, even if
// many visitors click the badge together. It updates the same cache key so the
// next HTML request receives that page.
func (r *HomepageReader) LatestKillsFresh(ctx context.Context, page, limit int) (KilllistPage, error) {
	return r.KilllistFresh(ctx, "latest", page, limit)
}

func (r *HomepageReader) KilllistFresh(ctx context.Context, kind string, page, limit int) (KilllistPage, error) {
	if err := r.validateKilllist(kind, page, limit); err != nil {
		return KilllistPage{}, err
	}
	key := fmt.Sprintf("killlist:%s:%d:%d", kind, page, limit)
	cacheKey := r.cacheKey(key)
	if r.cache != nil {
		if entry, _, ok := r.cache.LoadState(ctx, cacheKey); ok && time.Until(entry.FreshUntil) > 50*time.Second {
			var result KilllistPage
			if err := json.Unmarshal(entry.Body, &result); err == nil {
				return result, nil
			}
		}
	}
	value, err, _ := r.cache.LoadOnce(ctx, cacheKey+":fresh", func() (any, error) {
		return loadAndCacheHomepage(ctx, r.cache, cacheKey, time.Minute,
			func(ctx context.Context) (KilllistPage, error) { return r.loadKilllist(ctx, kind, page, limit) })
	})
	if err != nil {
		return KilllistPage{}, err
	}
	return value.(KilllistPage), nil
}

func (r *HomepageReader) validateKilllist(kind string, page, limit int) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("homepage database is not configured")
	}
	if _, known := killtype.Predicates()[kind]; !known || page < 1 || limit < 1 || limit > 100 {
		return fmt.Errorf("invalid killlist options")
	}
	return nil
}

func (r *HomepageReader) loadKilllist(ctx context.Context, kind string, page, limit int) (KilllistPage, error) {
	payload, err := loadNumberedKilllist(ctx, r.db, kind, killtype.Predicates()[kind], page, limit)
	if err != nil {
		return KilllistPage{}, err
	}
	return decodeKilllistPayload(payload)
}

// FilteredKills uses the same advanced query parser and SQL as the public API.
// The filters value is the advanced endpoint's JSON contract.
func (r *HomepageReader) FilteredKills(
	ctx context.Context, filters string, after *int64, limit int,
) (KilllistPage, error) {
	if r == nil || r.db == nil {
		return KilllistPage{}, fmt.Errorf("homepage database is not configured")
	}
	if limit < 1 || limit > 100 || len(filters) > 4096 {
		return KilllistPage{}, fmt.Errorf("invalid filtered killlist options")
	}
	values := url.Values{"limit": {strconv.Itoa(limit)}, "filters": {filters}}
	if after != nil {
		values.Set("after", strconv.FormatInt(*after, 10))
	}
	query, err := parseAdvancedKilllistQuery(&legacyRequest{Query: values}, time.Now().UTC())
	if err != nil {
		return KilllistPage{}, err
	}
	payload, err := loadAdvancedKills(ctx, r.db, query)
	if err != nil {
		return KilllistPage{}, err
	}
	return decodeKilllistPayload(payload)
}

func decodeKilllistPayload(payload legacyPayload) (KilllistPage, error) {
	encoded, err := json.Marshal(payload.Body)
	if err != nil {
		return KilllistPage{}, err
	}
	var result KilllistPage
	if err := json.Unmarshal(encoded, &result); err != nil {
		return KilllistPage{}, err
	}
	return result, nil
}

func cachedHomepage[T any](
	ctx context.Context, reader *HomepageReader, name string, ttl time.Duration,
	load func(context.Context) (T, error),
) (T, error) {
	if reader.cache == nil {
		return load(ctx)
	}
	key := reader.cacheKey(name)
	decode := func(entry cachedResponse) (T, error) {
		var result T
		err := json.Unmarshal(entry.Body, &result)
		return result, err
	}
	if entry, fresh, ok := reader.cache.LoadState(ctx, key); ok {
		if result, err := decode(entry); err == nil {
			if !fresh {
				reader.cache.Refresh(key, func() (any, error) {
					refreshCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					defer cancel()
					return loadAndCacheHomepage(refreshCtx, reader.cache, key, ttl, load)
				})
			}
			return result, nil
		}
	}
	value, err, _ := reader.cache.LoadOnce(ctx, key, func() (any, error) {
		if entry, fresh, ok := reader.cache.LoadState(ctx, key); ok && fresh {
			if result, err := decode(entry); err == nil {
				return result, nil
			}
		}
		return loadAndCacheHomepage(ctx, reader.cache, key, ttl, load)
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return value.(T), nil
}

func (r *HomepageReader) cacheKey(name string) string {
	return "shrike:homepage:v1:" + r.commit + ":" + name
}

func loadAndCacheHomepage[T any](
	ctx context.Context, cache *ResponseCache, key string, ttl time.Duration,
	load func(context.Context) (T, error),
) (T, error) {
	result, err := load(ctx)
	if err != nil {
		return result, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		var zero T
		return zero, err
	}
	if cache != nil {
		cache.StoreStale(context.WithoutCancel(ctx), key, cachedResponse{
			ContentType: "application/json", Body: encoded,
		}, ttl, ttl)
	}
	return result, nil
}

func decodeHomepageRows[T any](rows []map[string]any) ([]T, error) {
	encoded, err := json.Marshal(rows)
	if err != nil {
		return nil, err
	}
	var result []T
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, err
	}
	return result, nil
}
