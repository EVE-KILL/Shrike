package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Public HTML and JSON share the authoritative faction warfare loaders.
type FactionWarSideData struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	CorpID       int64   `json:"corpId"`
	Kills        int64   `json:"kills"`
	Losses       int64   `json:"losses"`
	ISKDestroyed float64 `json:"isk_destroyed"`
	ISKLost      float64 `json:"isk_lost"`
}
type FactionWarShip struct {
	ID    int64  `json:"ship_type_id"`
	Name  string `json:"ship_name"`
	Kills int64  `json:"kills"`
}
type FactionWarData struct {
	Matchup  string             `json:"matchup"`
	Days     int                `json:"days"`
	Side1    FactionWarSideData `json:"side1"`
	Side2    FactionWarSideData `json:"side2"`
	TopShips []FactionWarShip   `json:"topShips"`
	Overview FactionWarOverview `json:"overview"`
}
type FactionWarFactionStats struct {
	Pilots    int64 `json:"pilots"`
	Systems   int64 `json:"systems_controlled"`
	WeekVP    int64 `json:"vp_last_week"`
	WeekKills int64 `json:"kills_last_week"`
}
type FactionWarLeaderboardEntry struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Kills           int64  `json:"kills"`
	CorporationName string `json:"corporation_name"`
}
type FactionWarLeaderboardSides struct {
	Side1 []FactionWarLeaderboardEntry `json:"side1"`
	Side2 []FactionWarLeaderboardEntry `json:"side2"`
}
type FactionWarFlip struct {
	SystemID   int64  `json:"solar_system_id"`
	SystemName string `json:"system_name"`
	OldFaction string `json:"old_faction_name"`
	NewFaction string `json:"new_faction_name"`
	Time       string `json:"flipped_at"`
}
type FactionWarFlipDay struct {
	Day   string           `json:"day"`
	Items []FactionWarFlip `json:"items"`
}
type FactionWarOverview struct {
	FactionStats struct {
		Side1 *FactionWarFactionStats `json:"side1"`
		Side2 *FactionWarFactionStats `json:"side2"`
	} `json:"factionStats"`
	Warzone struct {
		Total int64                     `json:"total_systems"`
		Side1 factionWarSystemBreakdown `json:"side1"`
		Side2 factionWarSystemBreakdown `json:"side2"`
	} `json:"warzone"`
	FlipDays     []FactionWarFlipDay `json:"flipDays"`
	Leaderboards struct {
		Characters   FactionWarLeaderboardSides `json:"characters"`
		Corporations FactionWarLeaderboardSides `json:"corporations"`
	} `json:"leaderboards"`
}
type FactionWarSystem struct {
	ID                int64   `json:"solar_system_id"`
	Name              string  `json:"system_name"`
	X                 float64 `json:"x"`
	Y                 float64 `json:"y"`
	Security          float64 `json:"security"`
	RegionID          int64   `json:"region_id"`
	RegionName        string  `json:"region_name"`
	ConstellationName string  `json:"constellation_name"`
	OccupierID        int64   `json:"occupier_faction_id"`
	Contested         string  `json:"contested"`
	VP                int64   `json:"victory_points"`
	Threshold         int64   `json:"victory_points_threshold"`
	Kills             int64   `json:"kills_24h"`
	ISK               float64 `json:"isk_24h"`
}
type FactionWarCelestial struct {
	SystemID int64   `json:"system_id"`
	GroupID  int64   `json:"group_id"`
	X        float64 `json:"x"`
	Z        float64 `json:"z"`
}
type FactionWarSystems struct {
	Systems    []FactionWarSystem    `json:"systems"`
	Jumps      [][]int64             `json:"jumps"`
	Celestials []FactionWarCelestial `json:"celestials"`
}

func factionPageRequest(slug string, query url.Values) *legacyRequest {
	return &legacyRequest{Params: map[string]string{"matchup": slug}, Query: query}
}
func validFactionMatchup(slug string) bool {
	return slug == "caldari-vs-gallente" || slug == "amarr-vs-minmatar"
}

func (r *HomepageReader) FactionWarDashboard(ctx context.Context, slug string) (FactionWarData, error) {
	return factionPageRead[FactionWarData](ctx, r, slug, nil, "dashboard", 10*time.Minute, factionWarDashboardHandler)
}
func (r *HomepageReader) FactionWarOverview(ctx context.Context, slug, period string) (FactionWarOverview, error) {
	return factionPageRead[FactionWarOverview](ctx, r, slug, url.Values{"period": {period}}, "overview", 10*time.Minute, factionWarOverviewHandler)
}
func (r *HomepageReader) FactionWarSystems(ctx context.Context, slug string) (FactionWarSystems, error) {
	return factionPageRead[FactionWarSystems](ctx, r, slug, nil, "systems", 2*time.Minute, factionWarSystemsHandler)
}
func (r *HomepageReader) FactionWarMembers(ctx context.Context, slug string, query url.Values) (WarMembersData, error) {
	return factionPageRead[WarMembersData](ctx, r, slug, query, "members", 5*time.Minute, factionWarMembersHandler)
}
func (r *HomepageReader) FactionWarIntel(ctx context.Context, slug string) (WarIntelData, error) {
	return factionPageRead[WarIntelData](ctx, r, slug, nil, "intel", 5*time.Minute, factionWarIntelHandler)
}
func factionPageRead[T any](ctx context.Context, r *HomepageReader, slug string, query url.Values, part string, ttl time.Duration, handler func(Options) legacyHandler) (T, error) {
	var zero T
	if r == nil || r.db == nil || !validFactionMatchup(slug) {
		return zero, fmt.Errorf("invalid faction matchup")
	}
	return cachedHomepage(ctx, r, "faction-page:"+slug+":"+part+":"+query.Encode(), ttl, func(ctx context.Context) (T, error) {
		payload, err := handler(Options{DB: r.db})(ctx, factionPageRequest(slug, query))
		if err != nil {
			return zero, err
		}
		return entityConflictDecode[T](payload)
	})
}

// The side denotes the attacking faction, so its killfeed contains its
// opponent's losses. It intentionally has the same unlimited history as Nuxt.
func (r *HomepageReader) FactionWarKilllist(ctx context.Context, slug, side string, page, limit int, after *int64) (KilllistPage, error) {
	if r == nil || r.db == nil || !validFactionMatchup(slug) {
		return KilllistPage{}, fmt.Errorf("invalid faction matchup")
	}
	query, err := factionWarKillQuery(slug, side, page, limit, after)
	if err != nil {
		return KilllistPage{}, err
	}
	return cachedHomepage(ctx, r, "faction-killlist:"+query.Encode(), time.Minute, func(ctx context.Context) (KilllistPage, error) {
		payload, err := killlistHandler(Options{DB: r.db})(ctx, &legacyRequest{Query: query})
		if err != nil {
			return KilllistPage{}, err
		}
		return decodeKilllistPayload(payload)
	})
}

func factionWarKillQuery(slug, side string, page, limit int, after *int64) (url.Values, error) {
	if !validFactionMatchup(slug) {
		return nil, fmt.Errorf("invalid faction matchup")
	}
	matchup, _ := factionWarMatchupFromRequest(factionPageRequest(slug, nil))
	factions := fmt.Sprintf("%d,%d", matchup.Side1.ID, matchup.Side2.ID)
	if side == "side1" {
		factions = strconv.Itoa(int(matchup.Side2.ID))
	} else if side == "side2" {
		factions = strconv.Itoa(int(matchup.Side1.ID))
	} else if side != "combined" {
		return nil, fmt.Errorf("invalid faction side")
	}
	// Match KillList.vue's filtered cursor mode: requesting page=1 would
	// trigger a costly lifetime COUNT instead of returning the newest slice.
	if page < 1 || limit < 10 || limit > 100 || page > 1 && after == nil || after != nil && *after <= 0 {
		return nil, fmt.Errorf("invalid faction killlist pagination")
	}
	query := url.Values{"type": {"latest"}, "victimFactions": {factions}, "limit": {strconv.Itoa(limit)}}
	if after != nil {
		query.Set("after", strconv.FormatInt(*after, 10))
	}
	return query, nil
}
