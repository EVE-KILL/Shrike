package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// EntityDetail gives Go-rendered pages the same consolidated data as the
// public /api/{character,corporation,alliance,faction}/{id} routes. A missing
// entity returns nil, matching the frontend's 404 handling.
func (r *HomepageReader) EntityDetail(ctx context.Context, kind string, id int64) (map[string]any, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("entity database is not configured")
	}
	if (kind != entityPageCharacter && kind != entityPageCorporation &&
		kind != entityPageAlliance && kind != entityPageFaction) || id < 1 || id > 2_147_483_647 {
		return nil, fmt.Errorf("invalid entity page identity")
	}
	key := fmt.Sprintf("entity-detail:%s:%d", kind, id)
	return cachedHomepage(ctx, r, key, 2*time.Minute, func(ctx context.Context) (map[string]any, error) {
		value, err := loadEntityPageDetail(ctx, Options{DB: r.db}, kind, id, nil)
		if err != nil {
			var apiErr *legacyAPIError
			if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
				return nil, nil
			}
			return nil, err
		}
		result, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid entity detail result")
		}
		return result, nil
	})
}

// EntityKilllist reuses the scoped killmail query and enrichment behind the
// public entity endpoint, including its numbered-page and cursor behavior.
func (r *HomepageReader) EntityKilllist(ctx context.Context, kind string, id int64, role string, page, limit int, after *int64, shipGroup int64) (KilllistPage, error) {
	if r == nil || r.db == nil {
		return KilllistPage{}, fmt.Errorf("entity database is not configured")
	}
	if (kind != entityPageCharacter && kind != entityPageCorporation && kind != entityPageAlliance && kind != entityPageFaction) ||
		(role != "combined" && role != "kills" && role != "losses") || id < 1 || id > 2_147_483_647 ||
		page < 1 || page > 1_000_000 || limit < 10 || limit > 100 || shipGroup < 0 || shipGroup > 2_147_483_647 {
		return KilllistPage{}, fmt.Errorf("invalid entity killlist options")
	}
	if after != nil && *after < 1 {
		return KilllistPage{}, fmt.Errorf("invalid entity killlist cursor")
	}
	var cursor int64
	if after != nil {
		cursor = *after
	}
	key := fmt.Sprintf("entity-killlist:%s:%d:%s:%d:%d:%d:%d", kind, id, role, page, limit, cursor, shipGroup)
	return cachedHomepage(ctx, r, key, time.Minute, func(ctx context.Context) (KilllistPage, error) {
		query := url.Values{"role": {role}, "limit": {strconv.Itoa(limit)}}
		if page > 1 {
			query.Set("page", strconv.Itoa(page))
		}
		if after != nil {
			query.Set("after", strconv.FormatInt(*after, 10))
		}
		if shipGroup > 0 {
			query.Set("ship_group", strconv.FormatInt(shipGroup, 10))
		}
		value, err := loadEntityPageKilllist(ctx, Options{DB: r.db}, kind, id, &legacyRequest{Query: query})
		if err != nil {
			return KilllistPage{}, err
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return KilllistPage{}, err
		}
		var result KilllistPage
		if err := json.Unmarshal(encoded, &result); err != nil {
			return KilllistPage{}, err
		}
		return result, nil
	})
}

type EntityTopCollection struct {
	Characters   []TopEntry `json:"characters"`
	Corporations []TopEntry `json:"corporations"`
	Alliances    []TopEntry `json:"alliances"`
}

type EntityTopLists struct {
	Killed   EntityTopCollection `json:"killed"`
	KilledBy EntityTopCollection `json:"killedBy"`
}

type EntityDashboardStats struct {
	Kills          int64   `json:"kills"`
	Losses         int64   `json:"losses"`
	Efficiency     int64   `json:"efficiency"`
	ISKDestroyed   float64 `json:"isk_destroyed"`
	ISKLost        float64 `json:"isk_lost"`
	ISKEfficiency  int64   `json:"isk_efficiency"`
	SoloKills      int64   `json:"solo_kills"`
	SoloLosses     int64   `json:"solo_losses"`
	NPCLosses      int64   `json:"npc_losses"`
	BlobFactor     float64 `json:"blob_factor"`
	ActiveTimezone string  `json:"active_timezone"`
	FinalBlows     int64   `json:"final_blows"`
	Points         int64   `json:"points"`
}

type EntityShipEntry struct {
	ShipTypeID int64   `json:"ship_type_id"`
	ShipName   string  `json:"ship_name"`
	Count      float64 `json:"count"`
}

type EntityDashboard struct {
	Stats                 EntityDashboardStats `json:"stats"`
	HeatMap               map[string]int64     `json:"heatMap"`
	TopShipsUsed          []EntityShipEntry    `json:"topShipsUsed"`
	TopShipsLost          []EntityShipEntry    `json:"topShipsLost"`
	FliesWithCorporations []TopEntry           `json:"fliesWithCorporations"`
	FliesWithAlliances    []TopEntry           `json:"fliesWithAlliances"`
	DiesToCorporations    []TopEntry           `json:"diesToCorporations"`
	DiesToAlliances       []TopEntry           `json:"diesToAlliances"`
}

type EntityTopData struct {
	CharactersByKills    []TopEntry `json:"charactersByKills"`
	CharactersByPoints   []TopEntry `json:"charactersByPoints"`
	CharactersByISK      []TopEntry `json:"charactersByIsk"`
	SoloKillers          []TopEntry `json:"soloKillers"`
	CorporationsByKills  []TopEntry `json:"corporationsByKills"`
	AchievementPoints    []TopEntry `json:"achievementPoints"`
	RecentMembers        []TopEntry `json:"recentMembers"`
	ShipsUsed            []TopEntry `json:"shipsUsed"`
	Systems              []TopEntry `json:"systems"`
	Constellations       []TopEntry `json:"constellations"`
	Regions              []TopEntry `json:"regions"`
	KilledCorporations   []TopEntry `json:"killedCorporations"`
	KilledAlliances      []TopEntry `json:"killedAlliances"`
	KilledByCorporations []TopEntry `json:"killedByCorporations"`
	KilledByAlliances    []TopEntry `json:"killedByAlliances"`
}

type EntityAchievement struct {
	ID              string  `json:"achievement_id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	Rarity          string  `json:"rarity"`
	Type            string  `json:"type"`
	PointsModifier  string  `json:"points_modifier"`
	CurrentCount    int64   `json:"current_count"`
	Threshold       int64   `json:"threshold"`
	NextThreshold   int64   `json:"next_threshold"`
	Level           int64   `json:"level"`
	MaxLevel        int64   `json:"max_level"`
	LevelThresholds []int64 `json:"level_thresholds"`
	IsCompleted     bool    `json:"is_completed"`
	Points          int64   `json:"points"`
	CompletedAt     string  `json:"completed_at"`
}

type EntityMember struct {
	CharacterID    int64   `json:"character_id"`
	CorporationID  int64   `json:"corporation_id"`
	Name           string  `json:"name"`
	SecurityStatus float64 `json:"security_status"`
	LastActive     string  `json:"last_active"`
	Kills90D       int64   `json:"kills_90d"`
	Losses90D      int64   `json:"losses_90d"`
	IsFC           bool    `json:"is_fc"`
	IsLogi         bool    `json:"is_logi"`
	IsCapitalPilot bool    `json:"is_capital_pilot"`
}

type EntityMembersData struct {
	Members []EntityMember `json:"members"`
	Total   int64          `json:"total"`
	Page    int            `json:"page"`
	Limit   int            `json:"limit"`
}

type EntityCorporation struct {
	CorporationID int64       `json:"corporation_id"`
	Name          string      `json:"name"`
	Ticker        string      `json:"ticker"`
	MemberCount   int64       `json:"member_count"`
	Palette       *TopPalette `json:"palette"`
}

type EntityCorporationsData struct {
	Corporations []EntityCorporation `json:"corporations"`
	Total        int64               `json:"total"`
}

type EntityBattleEntry struct {
	BattleID            int64   `json:"battle_id"`
	SolarSystemID       int64   `json:"solar_system_id"`
	SolarSystemName     string  `json:"solar_system_name"`
	SolarSystemSecurity float64 `json:"solar_system_security"`
	RegionID            int64   `json:"region_id"`
	RegionName          string  `json:"region_name"`
	StartTime           string  `json:"start_time"`
	DurationMinutes     int64   `json:"duration_minutes"`
	KillCount           int64   `json:"kill_count"`
	TotalISKDestroyed   float64 `json:"total_isk_destroyed"`
	IsMultiParty        bool    `json:"is_multi_party"`
	Teams               []struct {
		AllianceCount     int64   `json:"alliance_count"`
		TotalKills        int64   `json:"total_kills"`
		TotalISKDestroyed float64 `json:"total_isk_destroyed"`
		TopAlliances      []struct {
			AllianceID      int64  `json:"alliance_id"`
			AllianceName    string `json:"alliance_name"`
			CorporationID   int64  `json:"corporation_id"`
			CorporationName string `json:"corporation_name"`
		} `json:"top_alliances"`
	} `json:"teams"`
}

type EntityBattlesData struct {
	Battles []EntityBattleEntry `json:"battles"`
	Page    int                 `json:"page"`
}

type EntityWarSide struct {
	ID           int64   `json:"id"`
	Type         string  `json:"type"`
	Name         string  `json:"name"`
	ISKDestroyed float64 `json:"isk_destroyed"`
	ShipsKilled  int64   `json:"ships_killed"`
}

type EntityWarEntry struct {
	WarID         int64         `json:"war_id"`
	Aggressor     EntityWarSide `json:"aggressor"`
	Defender      EntityWarSide `json:"defender"`
	Started       string        `json:"started"`
	Finished      string        `json:"finished"`
	Mutual        bool          `json:"mutual"`
	OpenForAllies bool          `json:"open_for_allies"`
}

type EntityWarsData struct {
	Wars []EntityWarEntry `json:"wars"`
	Page int              `json:"page"`
}

type EntityCampaignSide struct {
	SideIndex    int64       `json:"side_index"`
	Name         string      `json:"name"`
	Kills        int64       `json:"kills"`
	Losses       int64       `json:"losses"`
	ISKDestroyed float64     `json:"isk_destroyed"`
	ISKLost      float64     `json:"isk_lost"`
	Palette      *TopPalette `json:"palette"`
	Entities     []struct {
		EntityType int   `json:"entity_type"`
		EntityID   int64 `json:"entity_id"`
	} `json:"entities"`
}

type EntityCampaignEntry struct {
	CampaignID  string `json:"campaign_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
	Status      int    `json:"status"`
	Visibility  int    `json:"visibility"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Creator     struct {
		CharacterID int64  `json:"character_id"`
		Name        string `json:"name"`
	} `json:"creator"`
	Totals *struct {
		KillCount    int64   `json:"killCount"`
		ISKDestroyed float64 `json:"iskDestroyed"`
	} `json:"totals"`
	Location struct {
		SystemIDs        []int64 `json:"systemIds"`
		ConstellationIDs []int64 `json:"constellationIds"`
		RegionIDs        []int64 `json:"regionIds"`
	} `json:"location"`
	Sides []EntityCampaignSide `json:"sides"`
}

type EntityCampaignsData struct {
	Campaigns []EntityCampaignEntry `json:"campaigns"`
	HasMore   bool                  `json:"hasMore"`
	Page      int                   `json:"page"`
}

type EntityIntelPeer struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Count            int64  `json:"count"`
	CorpName         string `json:"corp_name"`
	AllianceName     string `json:"alliance_name"`
	SharedEnemyKills int64  `json:"shared_enemy_kills"`
	Total            int64  `json:"total"`
	ActiveCharacters int64  `json:"active_characters"`
	LeftAt           string `json:"left_at"`
	JoinedAt         string `json:"joined_at"`
}

type EntityIntelData struct {
	Playstyle struct {
		Solo         int64   `json:"solo"`
		SmallGang    int64   `json:"small_gang"`
		MidGang      int64   `json:"mid_gang"`
		Fleet        int64   `json:"fleet"`
		Blob         int64   `json:"blob"`
		TotalKills   int64   `json:"total_kills"`
		AvgFleetSize float64 `json:"avg_fleet_size"`
	} `json:"playstyle"`
	FC struct {
		Likelihood string `json:"likelihood"`
	} `json:"fc"`
	DominantStyle string            `json:"dominant_style"`
	CapitalPilot  bool              `json:"capital_pilot"`
	IsLogi        bool              `json:"is_logi"`
	AWOXKills     int64             `json:"awox_kills"`
	CynoDeaths    int64             `json:"cyno_deaths"`
	Bait          string            `json:"bait"`
	BaitCount     int64             `json:"bait_count"`
	ShipsFlown    []TopEntry        `json:"ships_flown"`
	ShipsLost     []TopEntry        `json:"ships_lost"`
	Targets       []TopEntry        `json:"targets"`
	FleetPartners []EntityIntelPeer `json:"fleet_partners"`
	ActiveMembers struct {
		Days7  int64 `json:"days_7"`
		Days30 int64 `json:"days_30"`
		Days90 int64 `json:"days_90"`
	} `json:"activeMembers"`
	Census struct {
		Total int64 `json:"total"`
		FCs   int64 `json:"fcs"`
		Logis int64 `json:"logis"`
		Caps  int64 `json:"caps"`
	} `json:"census"`
	Allies           []EntityIntelPeer `json:"allies"`
	Enemies          []EntityIntelPeer `json:"enemies"`
	HuntingGrounds   []EntityIntelPeer `json:"huntingGrounds"`
	RecentDepartures []EntityIntelPeer `json:"recentDepartures"`
	RecentJoins      []EntityIntelPeer `json:"recentJoins"`
}

type EntityFactionSideStats struct {
	SystemsControlled int64 `json:"systems_controlled"`
	Pilots            int64 `json:"pilots"`
	KillsYesterday    int64 `json:"kills_yesterday"`
	KillsLastWeek     int64 `json:"kills_last_week"`
	VPLastWeek        int64 `json:"vp_last_week"`
}

type EntityFactionZoneStats struct {
	Contested  int64 `json:"contested"`
	Vulnerable int64 `json:"vulnerable"`
}

type EntityFactionFlip struct {
	SolarSystemID int64  `json:"solar_system_id"`
	SystemName    string `json:"system_name"`
	NewFactionID  int64  `json:"new_faction_id"`
	OldFactionID  int64  `json:"old_faction_id"`
	FlippedAt     string `json:"flipped_at"`
}

type EntityFactionOverview struct {
	Matchup      string
	Side         int
	FactionStats struct {
		Side1 EntityFactionSideStats `json:"side1"`
		Side2 EntityFactionSideStats `json:"side2"`
	} `json:"factionStats"`
	Warzone struct {
		TotalSystems int64                  `json:"total_systems"`
		Side1        EntityFactionZoneStats `json:"side1"`
		Side2        EntityFactionZoneStats `json:"side2"`
	} `json:"warzone"`
	FlipDays []struct {
		Items []EntityFactionFlip `json:"items"`
	} `json:"flipDays"`
	Leaderboards struct {
		Characters struct {
			Side1 []struct {
				ID    int64  `json:"id"`
				Name  string `json:"name"`
				Kills int64  `json:"kills"`
			} `json:"side1"`
			Side2 []struct {
				ID    int64  `json:"id"`
				Name  string `json:"name"`
				Kills int64  `json:"kills"`
			} `json:"side2"`
		} `json:"characters"`
	} `json:"leaderboards"`
}

type EntityFactionMilitiaCorp struct {
	ID     int64
	Name   string
	Pilots int
}

func (r *HomepageReader) EntityFactionMilitiaCorps(ctx context.Context, id int64) ([]EntityFactionMilitiaCorp, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("faction database is not configured")
	}
	var matchup factionWarMatchup
	var side string
	for _, candidate := range factionWarMatchups {
		if int64(candidate.Side1.ID) == id {
			matchup, side = candidate, "aggressor"
			break
		}
		if int64(candidate.Side2.ID) == id {
			matchup, side = candidate, "defender"
			break
		}
	}
	if side == "" {
		return nil, fmt.Errorf("faction has no warzone")
	}
	key := fmt.Sprintf("faction-militia-corps:%d", id)
	return cachedHomepage(ctx, r, key, 10*time.Minute, func(ctx context.Context) ([]EntityFactionMilitiaCorp, error) {
		options := factionWarMemberOptions{Days: 30, Limit: 500, Side: side, Sort: "activity"}
		query, args := factionWarMembersQuery(matchup, options, time.Now().UTC().Add(-30*24*time.Hour))
		rows, err := queryMaps(ctx, r.db, query, args...)
		if err != nil {
			return nil, err
		}
		byCorp := make(map[int64]*EntityFactionMilitiaCorp)
		for _, row := range rows {
			corpID := factionWarInt(row, "corporation_id")
			if corpID == 0 {
				continue
			}
			corp := byCorp[corpID]
			if corp == nil {
				corp = &EntityFactionMilitiaCorp{ID: corpID, Name: factionWarString(row, "corp_name")}
				byCorp[corpID] = corp
			}
			corp.Pilots++
		}
		result := make([]EntityFactionMilitiaCorp, 0, len(byCorp))
		for _, corp := range byCorp {
			result = append(result, *corp)
		}
		sort.Slice(result, func(i, j int) bool {
			if result[i].Pilots != result[j].Pilots {
				return result[i].Pilots > result[j].Pilots
			}
			return result[i].ID < result[j].ID
		})
		return result[:min(10, len(result))], nil
	})
}

func (r *HomepageReader) EntityFactionOverview(ctx context.Context, id int64) (EntityFactionOverview, error) {
	if r == nil || r.db == nil {
		return EntityFactionOverview{}, fmt.Errorf("faction database is not configured")
	}
	var matchup factionWarMatchup
	side := 0
	for _, candidate := range factionWarMatchups {
		if int64(candidate.Side1.ID) == id {
			matchup, side = candidate, 1
			break
		}
		if int64(candidate.Side2.ID) == id {
			matchup, side = candidate, 2
			break
		}
	}
	if side == 0 {
		return EntityFactionOverview{}, fmt.Errorf("faction has no warzone")
	}
	key := fmt.Sprintf("faction-overview:%d", id)
	return cachedHomepage(ctx, r, key, 2*time.Minute, func(ctx context.Context) (EntityFactionOverview, error) {
		value, err := loadFactionWarOverview(ctx, r.db, matchup, "last_week")
		if err != nil {
			return EntityFactionOverview{}, err
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return EntityFactionOverview{}, err
		}
		result := EntityFactionOverview{Matchup: matchup.Slug, Side: side}
		err = json.Unmarshal(encoded, &result)
		return result, err
	})
}

func (r *HomepageReader) EntityIntel(ctx context.Context, kind string, id int64, days int) (EntityIntelData, error) {
	if r == nil || r.db == nil || (kind != entityPageCharacter && kind != entityPageCorporation && kind != entityPageAlliance) ||
		id < 1 || id > 2_147_483_647 || (days != 0 && days != 14 && days != 30 && days != 90 && days != 180 && days != 365) {
		return EntityIntelData{}, fmt.Errorf("invalid entity intel options")
	}
	if kind != entityPageCharacter {
		days = 90
	}
	key := fmt.Sprintf("entity-intel:%s:%d:%d", kind, id, days)
	return cachedHomepage(ctx, r, key, time.Hour, func(ctx context.Context) (EntityIntelData, error) {
		query := url.Values{"days": {strconv.Itoa(days)}}
		value, err := loadEntityPageIntel(ctx, Options{DB: r.db, Graph: r.graph}, kind, id, &legacyRequest{Query: query})
		if err != nil {
			return EntityIntelData{}, err
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return EntityIntelData{}, err
		}
		var result EntityIntelData
		err = json.Unmarshal(encoded, &result)
		return result, err
	})
}

func entityConflictRequest(query url.Values) *legacyRequest {
	return &legacyRequest{Query: query}
}

func entityConflictDecode[T any](payload legacyPayload) (T, error) {
	var result T
	encoded, err := json.Marshal(payload.Body)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(encoded, &result)
	return result, err
}

func (r *HomepageReader) EntityBattles(ctx context.Context, kind string, id int64, page int) (EntityBattlesData, error) {
	if r == nil || r.db == nil || (kind != entityPageCharacter && kind != entityPageCorporation && kind != entityPageAlliance && kind != "system" && kind != "region" && kind != "constellation") || id < 1 || id > 2_147_483_647 || page < 1 || page > 1_000_000 {
		return EntityBattlesData{}, fmt.Errorf("invalid entity battle options")
	}
	key := fmt.Sprintf("entity-battles:%s:%d:%d", kind, id, page)
	return cachedHomepage(ctx, r, key, time.Minute, func(ctx context.Context) (EntityBattlesData, error) {
		query := url.Values{"page": {strconv.Itoa(page)}, "limit": {"50"}, kind + "Id": {strconv.FormatInt(id, 10)}}
		payload, err := conflictBattlesHandler(Options{DB: r.db})(ctx, entityConflictRequest(query))
		if err != nil {
			return EntityBattlesData{}, err
		}
		return entityConflictDecode[EntityBattlesData](payload)
	})
}

func (r *HomepageReader) EntityWars(ctx context.Context, kind string, id int64, page int, sortBy string) (EntityWarsData, error) {
	if r == nil || r.db == nil || (kind != entityPageCharacter && kind != entityPageCorporation && kind != entityPageAlliance) || id < 1 || id > 2_147_483_647 || page < 1 || page > 1_000_000 || (sortBy != "recent" && sortBy != "kills" && sortBy != "isk") {
		return EntityWarsData{}, fmt.Errorf("invalid entity war options")
	}
	key := fmt.Sprintf("entity-wars:%s:%d:%d:%s", kind, id, page, sortBy)
	return cachedHomepage(ctx, r, key, time.Minute, func(ctx context.Context) (EntityWarsData, error) {
		query := url.Values{"page": {strconv.Itoa(page)}, "limit": {"50"}, "sort": {sortBy}, kind + "Id": {strconv.FormatInt(id, 10)}}
		payload, err := conflictWarsHandler(Options{DB: r.db})(ctx, entityConflictRequest(query))
		if err != nil {
			return EntityWarsData{}, err
		}
		return entityConflictDecode[EntityWarsData](payload)
	})
}

func (r *HomepageReader) EntityCampaigns(ctx context.Context, kind string, id int64, page int) (EntityCampaignsData, error) {
	if r == nil || r.db == nil || (kind != entityPageCharacter && kind != entityPageCorporation && kind != entityPageAlliance) || id < 1 || id > 2_147_483_647 || page < 1 || page > 1_000_000 {
		return EntityCampaignsData{}, fmt.Errorf("invalid entity campaign options")
	}
	key := fmt.Sprintf("entity-campaigns:%s:%d:%d", kind, id, page)
	return cachedHomepage(ctx, r, key, time.Minute, func(ctx context.Context) (EntityCampaignsData, error) {
		query := url.Values{"page": {strconv.Itoa(page)}, "entityType": {kind}, "entityId": {strconv.FormatInt(id, 10)}}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://eve-kill.com/api/campaigns?"+query.Encode(), nil)
		if err != nil {
			return EntityCampaignsData{}, err
		}
		legacy := &legacyRequest{Huma: humago.NewContext(nil, request, httptest.NewRecorder()), Query: query}
		// Public campaign cards can share the established listing logic. A
		// session-aware reader will be needed before private campaigns have parity.
		service := &campaignService{auth: &authService{now: time.Now}, db: r.db}
		payload, err := service.campaignListHandler()(ctx, legacy)
		if err != nil {
			return EntityCampaignsData{}, err
		}
		return entityConflictDecode[EntityCampaignsData](payload)
	})
}

func (r *HomepageReader) EntityMembers(ctx context.Context, kind string, id int64, page int, sortBy string, activity int, corporationID int64) (EntityMembersData, error) {
	if r == nil || r.db == nil || (kind != entityPageCorporation && kind != entityPageAlliance) ||
		id < 1 || id > 2_147_483_647 || page < 1 || page > 1_000_000 ||
		(sortBy != "name" && sortBy != "last_active" && sortBy != "security_status") ||
		(activity != 0 && activity != 7 && activity != 30 && activity != 90) ||
		corporationID < 0 || corporationID > 2_147_483_647 || kind == entityPageCorporation && corporationID != 0 {
		return EntityMembersData{}, fmt.Errorf("invalid entity member options")
	}
	key := fmt.Sprintf("entity-members:%s:%d:%d:%s:%d:%d", kind, id, page, sortBy, activity, corporationID)
	return cachedHomepage(ctx, r, key, 2*time.Minute, func(ctx context.Context) (EntityMembersData, error) {
		query := url.Values{"page": {strconv.Itoa(page)}, "limit": {"100"}, "sort": {sortBy}}
		if activity > 0 {
			query.Set("activity", strconv.Itoa(activity))
		}
		if corporationID > 0 {
			query.Set("corporation_id", strconv.FormatInt(corporationID, 10))
		}
		value, err := loadEntityPageMembers(ctx, Options{DB: r.db, Graph: r.graph}, kind, id, &legacyRequest{Query: query})
		if err != nil {
			return EntityMembersData{}, err
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return EntityMembersData{}, err
		}
		var result EntityMembersData
		err = json.Unmarshal(encoded, &result)
		return result, err
	})
}

func (r *HomepageReader) EntityCorporations(ctx context.Context, id int64, sortBy, direction string) (EntityCorporationsData, error) {
	if r == nil || r.db == nil || id < 1 || id > 2_147_483_647 ||
		(sortBy != "name" && sortBy != "members") || (direction != "asc" && direction != "desc") {
		return EntityCorporationsData{}, fmt.Errorf("invalid alliance corporation options")
	}
	key := fmt.Sprintf("entity-corporations:%d:%s:%s", id, sortBy, direction)
	return cachedHomepage(ctx, r, key, 2*time.Minute, func(ctx context.Context) (EntityCorporationsData, error) {
		query := url.Values{"sort": {sortBy}, "dir": {direction}}
		value, err := loadEntityPageCorporations(ctx, Options{DB: r.db}, entityPageAlliance, id, &legacyRequest{Query: query})
		if err != nil {
			return EntityCorporationsData{}, err
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return EntityCorporationsData{}, err
		}
		var result EntityCorporationsData
		err = json.Unmarshal(encoded, &result)
		return result, err
	})
}

// EntityAchievements returns the character's award progress, including
// unstarted definitions so the filter controls have the full catalogue.
func (r *HomepageReader) EntityAchievements(ctx context.Context, id int64) ([]EntityAchievement, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("entity database is not configured")
	}
	if id < 1 || id > 2_147_483_647 {
		return nil, fmt.Errorf("invalid character ID")
	}
	key := fmt.Sprintf("entity-achievements:%d", id)
	return cachedHomepage(ctx, r, key, 2*time.Minute, func(ctx context.Context) ([]EntityAchievement, error) {
		value, err := loadEntityPageAchievements(ctx, Options{DB: r.db}, entityPageCharacter, id, nil)
		if err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		var result struct {
			Achievements []EntityAchievement `json:"achievements"`
		}
		if err := json.Unmarshal(encoded, &result); err != nil {
			return nil, err
		}
		return result.Achievements, nil
	})
}

// EntityTop reads one independently selectable Top-tab slice.
func (r *HomepageReader) EntityTop(ctx context.Context, kind string, id int64, slice, days string) (EntityTopData, error) {
	if r == nil || r.db == nil {
		return EntityTopData{}, fmt.Errorf("entity database is not configured")
	}
	entityType, ok := entityPageStatsType(kind)
	if !ok || id < 1 || id > 2_147_483_647 || (slice != "left" && slice != "right") {
		return EntityTopData{}, fmt.Errorf("invalid entity top options")
	}
	period := entityTopPeriod{}
	//nolint:misspell // alltime is the established public query value.
	if days == "alltime" && slice == "right" {
		period.AllTime = true
	} else {
		value, err := strconv.ParseFloat(days, 64)
		if err != nil || value < 1.0/24 || value > 365 {
			return EntityTopData{}, fmt.Errorf("invalid entity top period")
		}
		period.Days = value
	}
	key := fmt.Sprintf("entity-top:%s:%d:%s:%s", kind, id, slice, days)
	return cachedHomepage(ctx, r, key, 5*time.Minute, func(ctx context.Context) (EntityTopData, error) {
		value, err := loadEntityTop(ctx, r.db, kind, entityType, id, slice, period)
		if err != nil {
			return EntityTopData{}, err
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return EntityTopData{}, err
		}
		var result EntityTopData
		if err := json.Unmarshal(encoded, &result); err != nil {
			return EntityTopData{}, err
		}
		return result, nil
	})
}

// EntityDashboard uses the same windowed stats/breakdown loader as Nuxt's
// dashboard API, with a small cache for repeat page and period requests.
func (r *HomepageReader) EntityDashboard(ctx context.Context, kind string, id int64, days int) (EntityDashboard, error) {
	if r == nil || r.db == nil {
		return EntityDashboard{}, fmt.Errorf("entity database is not configured")
	}
	entityType, ok := entityPageStatsType(kind)
	if !ok || id < 1 || id > 2_147_483_647 ||
		(days != 0 && days != 14 && days != 30 && days != 90 && days != 180 && days != 365) {
		return EntityDashboard{}, fmt.Errorf("invalid entity dashboard options")
	}
	key := fmt.Sprintf("entity-dashboard:%s:%d:%d", kind, id, days)
	return cachedHomepage(ctx, r, key, 2*time.Minute, func(ctx context.Context) (EntityDashboard, error) {
		value, err := loadEntityDashboardStats(ctx, r.db, kind, entityType, id, days, entityPageDaysWindow(days))
		if err != nil {
			return EntityDashboard{}, err
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return EntityDashboard{}, err
		}
		var result EntityDashboard
		if err := json.Unmarshal(encoded, &result); err != nil {
			return EntityDashboard{}, err
		}
		return result, nil
	})
}

// EntityTopLists serves the lazy desktop sidebars on entity kill tabs.
func (r *HomepageReader) EntityTopLists(ctx context.Context, kind string, id int64, days int) (EntityTopLists, error) {
	if r == nil || r.db == nil {
		return EntityTopLists{}, fmt.Errorf("entity database is not configured")
	}
	if (kind != entityPageCharacter && kind != entityPageCorporation && kind != entityPageAlliance) ||
		id < 1 || id > 2_147_483_647 || days < 0 || days > 365 {
		return EntityTopLists{}, fmt.Errorf("invalid entity top list options")
	}
	key := fmt.Sprintf("entity-top-lists:%s:%d:%d", kind, id, days)
	return cachedHomepage(ctx, r, key, 30*time.Minute, func(ctx context.Context) (EntityTopLists, error) {
		query := url.Values{"days": {strconv.Itoa(days)}}
		value, err := loadEntityPageTopLists(ctx, Options{DB: r.db}, kind, id, &legacyRequest{Query: query})
		if err != nil {
			return EntityTopLists{}, err
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return EntityTopLists{}, err
		}
		var result EntityTopLists
		if err := json.Unmarshal(encoded, &result); err != nil {
			return EntityTopLists{}, err
		}
		return result, nil
	})
}
