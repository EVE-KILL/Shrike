package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

type WarDetailData struct {
	War   WarDetail      `json:"war"`
	Stats WarDetailStats `json:"stats"`
}

type WarDetail struct {
	WarID         int64         `json:"war_id"`
	Declared      string        `json:"declared"`
	Started       string        `json:"started"`
	Finished      string        `json:"finished"`
	Mutual        bool          `json:"mutual"`
	OpenForAllies bool          `json:"open_for_allies"`
	Aggressor     EntityWarSide `json:"aggressor"`
	Defender      EntityWarSide `json:"defender"`
	Allies        []WarAlly     `json:"allies"`
}

type WarAlly struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Kills        int64   `json:"kills"`
	Losses       int64   `json:"losses"`
	ISKDestroyed float64 `json:"isk_destroyed"`
	ISKLost      float64 `json:"isk_lost"`
}

type WarDetailStats struct {
	TotalKills int64        `json:"total_kills"`
	TotalValue float64      `json:"total_value"`
	TopShips   []WarTopShip `json:"top_ships"`
}

type WarTopShip struct {
	ShipTypeID int64  `json:"ship_type_id"`
	ShipName   string `json:"ship_name"`
	Count      int64  `json:"count"`
}

type WarLeaderboardEntry struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Kills int64  `json:"kills"`
	Side  string `json:"side"`
}

type WarLeaderboard struct {
	TopCharacters   []WarLeaderboardEntry `json:"topCharacters"`
	TopCorporations []WarLeaderboardEntry `json:"topCorporations"`
	TopAlliances    []WarLeaderboardEntry `json:"topAlliances"`
}

type WarStatsData struct {
	Combined  WarLeaderboard `json:"combined"`
	Aggressor WarLeaderboard `json:"aggressor"`
	Defender  WarLeaderboard `json:"defender"`
	TopShips  []WarTopShip   `json:"topShips"`
}

type WarMembersData struct {
	Members []WarMember `json:"members"`
	Count   int64       `json:"count"`
	Limit   int64       `json:"limit"`
}

type WarMember struct {
	CharacterID     int64   `json:"character_id"`
	CharacterName   string  `json:"character_name"`
	CorporationID   int64   `json:"corporation_id"`
	CorporationName string  `json:"corporation_name"`
	AllianceID      int64   `json:"alliance_id"`
	AllianceName    string  `json:"alliance_name"`
	TopShipTypeID   int64   `json:"top_ship_type_id"`
	TopShipName     string  `json:"top_ship_name"`
	TopShipCount    int64   `json:"top_ship_count"`
	Kills           int64   `json:"kills"`
	Losses          int64   `json:"losses"`
	ISKDestroyed    float64 `json:"isk_destroyed"`
	ISKLost         float64 `json:"isk_lost"`
	Side            string  `json:"side"`
}

type WarIntelData struct {
	Summary struct {
		Kills          int64   `json:"kills"`
		ISKDestroyed   float64 `json:"isk_destroyed"`
		Systems        int64   `json:"systems"`
		Constellations int64   `json:"constellations"`
		Regions        int64   `json:"regions"`
	} `json:"summary"`
	TopSystems          []WarIntelSystem    `json:"top_systems"`
	TopConstellations   []WarIntelLocation  `json:"top_constellations"`
	TopRegions          []WarIntelLocation  `json:"top_regions"`
	ShipsDestroyed      []WarIntelShip      `json:"ships_destroyed"`
	ShipsUsed           []WarIntelShip      `json:"ships_used"`
	ShipGroupsDestroyed []WarIntelShipGroup `json:"ship_groups_destroyed"`
	SecurityBreakdown   []WarIntelSecurity  `json:"security_breakdown"`
}

type WarIntelSystem struct {
	SystemID     int64    `json:"system_id"`
	SystemName   string   `json:"system_name"`
	RegionName   string   `json:"region_name"`
	Security     *float64 `json:"security"`
	Kills        int64    `json:"kills"`
	ISKDestroyed float64  `json:"isk_destroyed"`
}

type WarIntelLocation struct {
	ConstellationID   int64   `json:"constellation_id"`
	ConstellationName string  `json:"constellation_name"`
	RegionID          int64   `json:"region_id"`
	RegionName        string  `json:"region_name"`
	Kills             int64   `json:"kills"`
	ISKDestroyed      float64 `json:"isk_destroyed"`
}

type WarIntelShip struct {
	ShipTypeID   int64   `json:"ship_type_id"`
	ShipName     string  `json:"ship_name"`
	GroupID      int64   `json:"group_id"`
	GroupName    string  `json:"group_name"`
	Count        int64   `json:"count"`
	ISKDestroyed float64 `json:"isk_destroyed"`
}

type WarIntelShipGroup struct {
	GroupID      int64   `json:"group_id"`
	GroupName    string  `json:"group_name"`
	Count        int64   `json:"count"`
	ISKDestroyed float64 `json:"isk_destroyed"`
}

type WarIntelSecurity struct {
	SecClass     string  `json:"sec_class"`
	Kills        int64   `json:"kills"`
	ISKDestroyed float64 `json:"isk_destroyed"`
}

func validWarPageID(id int64) bool { return id > 0 && id <= pgInt4Max }

func warPageRequest(id int64, query url.Values) *legacyRequest {
	return &legacyRequest{Params: map[string]string{"id": strconv.FormatInt(id, 10)}, Query: query}
}

func (r *HomepageReader) WarDetail(ctx context.Context, id int64) (WarDetailData, bool, error) {
	if r == nil || r.db == nil || !validWarPageID(id) {
		return WarDetailData{}, false, nil
	}
	result, err := cachedHomepage(ctx, r, fmt.Sprintf("war-detail:%d", id), 2*time.Minute, func(ctx context.Context) (WarDetailData, error) {
		payload, err := warDetailHandler(Options{DB: r.db})(ctx, warPageRequest(id, nil))
		if err != nil {
			return WarDetailData{}, err
		}
		return entityConflictDecode[WarDetailData](payload)
	})
	if err != nil {
		var apiErr *legacyAPIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return WarDetailData{}, false, nil
		}
		return WarDetailData{}, false, err
	}
	return result, true, nil
}

func (r *HomepageReader) WarKilllist(ctx context.Context, id int64, query url.Values) (KilllistPage, error) {
	if r == nil || r.db == nil || !validWarPageID(id) {
		return KilllistPage{}, fmt.Errorf("invalid war")
	}
	return cachedHomepage(ctx, r, fmt.Sprintf("war-killlist:%d:%s", id, query.Encode()), 2*time.Minute, func(ctx context.Context) (KilllistPage, error) {
		payload, err := warKilllistHandler(Options{DB: r.db})(ctx, warPageRequest(id, query))
		if err != nil {
			return KilllistPage{}, err
		}
		return entityConflictDecode[KilllistPage](payload)
	})
}

func (r *HomepageReader) WarStats(ctx context.Context, id int64) (WarStatsData, error) {
	if r == nil || r.db == nil || !validWarPageID(id) {
		return WarStatsData{}, fmt.Errorf("invalid war")
	}
	return cachedHomepage(ctx, r, fmt.Sprintf("war-stats:%d", id), 2*time.Minute, func(ctx context.Context) (WarStatsData, error) {
		payload, err := warStatsHandler(Options{DB: r.db})(ctx, warPageRequest(id, nil))
		if err != nil {
			return WarStatsData{}, err
		}
		return entityConflictDecode[WarStatsData](payload)
	})
}

func (r *HomepageReader) WarMembers(ctx context.Context, id int64, query url.Values) (WarMembersData, error) {
	if r == nil || r.db == nil || !validWarPageID(id) {
		return WarMembersData{}, fmt.Errorf("invalid war")
	}
	return cachedHomepage(ctx, r, fmt.Sprintf("war-members:%d:%s", id, query.Encode()), 5*time.Minute, func(ctx context.Context) (WarMembersData, error) {
		payload, err := warMembersHandler(Options{DB: r.db})(ctx, warPageRequest(id, query))
		if err != nil {
			return WarMembersData{}, err
		}
		return entityConflictDecode[WarMembersData](payload)
	})
}

func (r *HomepageReader) WarIntel(ctx context.Context, id int64) (WarIntelData, error) {
	if r == nil || r.db == nil || !validWarPageID(id) {
		return WarIntelData{}, fmt.Errorf("invalid war")
	}
	return cachedHomepage(ctx, r, fmt.Sprintf("war-intel:%d", id), 5*time.Minute, func(ctx context.Context) (WarIntelData, error) {
		payload, err := warIntelHandler(Options{DB: r.db})(ctx, warPageRequest(id, nil))
		if err != nil {
			return WarIntelData{}, err
		}
		return entityConflictDecode[WarIntelData](payload)
	})
}

// The HTML directory and the public API use the same conflict handlers.
// Keeping the request host lets custom killboards retain their war scope.
type WarDirectoryData struct {
	Wars  []EntityWarEntry `json:"wars"`
	Page  int              `json:"page"`
	Limit int              `json:"limit"`
}

type WarOverviewData struct {
	ActiveWars        int64 `json:"activeWars"`
	FinishedWars      int64 `json:"finishedWars"`
	UpcomingWars      int64 `json:"upcomingWars"`
	EligibleCorps     int64 `json:"eligibleCorps"`
	EligibleAlliances int64 `json:"eligibleAlliances"`
}

type FactionWarSide struct {
	FactionID         int64   `json:"faction_id"`
	Name              string  `json:"name"`
	Kills             int64   `json:"kills"`
	ISKDestroyed      float64 `json:"isk_destroyed"`
	SystemsControlled int64   `json:"systems_controlled"`
	Pilots            int64   `json:"pilots"`
}

type FactionWarsData struct {
	CaldariVsGallente struct {
		Caldari  FactionWarSide `json:"caldari"`
		Gallente FactionWarSide `json:"gallente"`
	} `json:"caldariVsGallente"`
	AmarrVsMinmatar struct {
		Amarr    FactionWarSide `json:"amarr"`
		Minmatar FactionWarSide `json:"minmatar"`
	} `json:"amarrVsMinmatar"`
}

func (r *HomepageReader) WarDirectory(ctx context.Context, query url.Values, host string) (WarDirectoryData, error) {
	if r == nil || r.db == nil {
		return WarDirectoryData{}, fmt.Errorf("war directory unavailable")
	}
	host = normalizeRequestHost(host)
	key := "war-directory:" + host + ":" + query.Encode()
	return cachedHomepage(ctx, r, key, time.Minute, func(ctx context.Context) (WarDirectoryData, error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://eve-kill.com/api/conflicts/wars?"+query.Encode(), nil)
		if err != nil {
			return WarDirectoryData{}, err
		}
		request.Host = host
		legacy := &legacyRequest{Huma: humago.NewContext(nil, request, httptest.NewRecorder()), Query: query}
		payload, err := conflictWarsHandler(Options{DB: r.db})(ctx, legacy)
		if err != nil {
			return WarDirectoryData{}, err
		}
		return entityConflictDecode[WarDirectoryData](payload)
	})
}

func (r *HomepageReader) WarOverview(ctx context.Context) (WarOverviewData, error) {
	return cachedHomepage(ctx, r, "war-overview", 5*time.Minute, func(ctx context.Context) (WarOverviewData, error) {
		payload, err := warCountersHandler(Options{DB: r.db})(ctx, &legacyRequest{})
		if err != nil {
			return WarOverviewData{}, err
		}
		return entityConflictDecode[WarOverviewData](payload)
	})
}

func (r *HomepageReader) FactionWars(ctx context.Context) (FactionWarsData, error) {
	return cachedHomepage(ctx, r, "faction-wars", 10*time.Minute, func(ctx context.Context) (FactionWarsData, error) {
		payload, err := factionWarsHandler(Options{DB: r.db})(ctx, &legacyRequest{})
		if err != nil {
			return FactionWarsData{}, err
		}
		return entityConflictDecode[FactionWarsData](payload)
	})
}
