package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"golang.org/x/sync/errgroup"
)

type entityTab struct {
	ID, Label, Icon string
}

var entityCommonTabs = []entityTab{
	{"dashboard", "Dashboard", "activity"},
	{"combined", "Combined", "tags"},
	{"kills", "Kills", "trophy"},
	{"losses", "Losses", "swords"},
	{"top", "Top", "list-ordered"},
	{"battles", "Battles", "swords"},
	{"wars", "Wars", "flag"},
	{"campaigns", "Campaigns", "radio"},
}

func entityTabs(kind string) []entityTab {
	if kind == "faction" {
		return append([]entityTab(nil), entityCommonTabs[:4]...)
	}
	tabs := append([]entityTab(nil), entityCommonTabs...)
	switch kind {
	case "character":
		return append(tabs, entityTab{"history", "History", "archive"}, entityTab{"achievements", "Achievements", "trophy"})
	case "corporation":
		return append(tabs, entityTab{"members", "Members", "users"}, entityTab{"history", "History", "archive"})
	case "alliance":
		return append(tabs, entityTab{"corporations", "Corporations", "network"}, entityTab{"members", "Members", "users"})
	default:
		return nil
	}
}

type entityIdentity struct {
	Name                   string          `json:"name"`
	Ticker                 string          `json:"ticker"`
	Title                  string          `json:"title"`
	Description            string          `json:"description"`
	CustomDescriptionHTML  string          `json:"custom_description_html"`
	Birthday               string          `json:"birthday"`
	DateFounded            string          `json:"date_founded"`
	LastActive             string          `json:"last_active"`
	RaceName               string          `json:"race_name"`
	BloodlineName          string          `json:"bloodline_name"`
	SecurityStatus         float64         `json:"security_status"`
	Palette                *api.TopPalette `json:"palette"`
	CorporationID          int64           `json:"corporation_id"`
	CorporationName        string          `json:"corporation_name"`
	CorporationTicker      string          `json:"corporation_ticker"`
	AllianceID             int64           `json:"alliance_id"`
	AllianceName           string          `json:"alliance_name"`
	AllianceTicker         string          `json:"alliance_ticker"`
	FactionID              int64           `json:"faction_id"`
	FactionName            string          `json:"faction_name"`
	ExecutorCorporationID  int64           `json:"executor_corporation_id"`
	ExecutorName           string          `json:"executor_name"`
	ExecutorTicker         string          `json:"executor_ticker"`
	CEOId                  int64           `json:"ceo_id"`
	CEOName                string          `json:"ceo_name"`
	CreatorID              int64           `json:"creator_id"`
	CreatorName            string          `json:"creator_name"`
	MemberCount            int64           `json:"member_count"`
	CorporationCount       int64           `json:"corporation_count"`
	StationCount           int64           `json:"station_count"`
	StationSystemCount     int64           `json:"station_system_count"`
	MilitiaCorporationID   int64           `json:"militia_corporation_id"`
	MilitiaCorporationName string          `json:"militia_corporation_name"`
	SolarSystemID          int64           `json:"solar_system_id"`
	SolarSystemName        string          `json:"solar_system_name"`
	TaxRate                float64         `json:"tax_rate"`
	LPTaxRate              float64         `json:"lp_tax_rate"`
	WarEligible            bool            `json:"war_eligible"`
	State                  string          `json:"state"`
}

type entityStats struct {
	Kills, Losses, SoloKills, NPCLosses, FinalBlows, Points int64
	ISKDestroyed, ISKLost                                   float64
	Efficiency, ISKEfficiency                               int64
}

// The database's public JSON uses snake-case field names.
func (s *entityStats) UnmarshalJSON(data []byte) error {
	var raw struct {
		Kills         int64   `json:"kills"`
		Losses        int64   `json:"losses"`
		SoloKills     int64   `json:"solo_kills"`
		NPCLosses     int64   `json:"npc_losses"`
		FinalBlows    int64   `json:"final_blows"`
		Points        int64   `json:"points"`
		ISKDestroyed  float64 `json:"isk_destroyed"`
		ISKLost       float64 `json:"isk_lost"`
		Efficiency    int64   `json:"efficiency"`
		ISKEfficiency int64   `json:"isk_efficiency"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = entityStats{Kills: raw.Kills, Losses: raw.Losses, SoloKills: raw.SoloKills,
		NPCLosses: raw.NPCLosses, FinalBlows: raw.FinalBlows, Points: raw.Points,
		ISKDestroyed: raw.ISKDestroyed, ISKLost: raw.ISKLost,
		Efficiency: raw.Efficiency, ISKEfficiency: raw.ISKEfficiency}
	return nil
}

type entityRanking struct {
	Rating, Rank, Population int64
}

type entityPageView struct {
	Kind, Tab, Name, Accent, Image, Background, BioHTML string
	ID                                                  int64
	ShipGroup                                           int64
	KillPage, KillLimit                                 int
	Tabs                                                []entityTab
	Identity                                            entityIdentity
	Stats, Recent                                       entityStats
	Ranking                                             entityRanking
	HasOldPortrait                                      bool
	Killlist                                            api.KilllistPage
	Dashboard                                           api.EntityDashboard
	TopLeft, TopRight                                   api.EntityTopData
	History                                             []entityHistoryEntry
	HistoryQueued                                       bool
	Achievements                                        []api.EntityAchievement
	Members                                             api.EntityMembersData
	Corporations                                        api.EntityCorporationsData
	Battles                                             api.EntityBattlesData
	Wars                                                api.EntityWarsData
	Campaigns                                           api.EntityCampaignsData
}

type entityHistoryEntry struct {
	CorporationID     int64  `json:"corporation_id"`
	CorporationName   string `json:"corporation_name"`
	CorporationTicker string `json:"corporation_ticker"`
	AllianceID        int64  `json:"alliance_id"`
	AllianceName      string `json:"alliance_name"`
	AllianceTicker    string `json:"alliance_ticker"`
	StartDate         string `json:"start_date"`
	Kills             int64  `json:"kills"`
	Losses            int64  `json:"losses"`
}

type entityStatRow struct {
	Label, Value, Class string
}

func entityProgressValue(value string) string {
	n, _ := strconv.Atoi(strings.TrimSuffix(value, "%"))
	return strconv.Itoa(max(0, min(100, n)))
}

func entityDangerRow(stats entityStats) entityStatRow {
	ratio := entityDangerRatio(stats)
	class := "warning"
	if ratio >= 50 {
		class = "loss"
	}
	return entityStatRow{"Danger Ratio", strconv.FormatInt(ratio, 10) + "%", class}
}

func entityBalanceRow(balance float64) entityStatRow {
	if balance > 0 {
		return entityStatRow{"Balance", "+" + compactNumber(balance), "kill"}
	}
	return entityStatRow{"Balance", compactNumber(balance), "loss"}
}

type entityPeriod struct {
	Label string
	Days  int
}

type entityTopChoice struct {
	Label, Value string
}

var entityTopChoices = []entityTopChoice{
	{"1h", "0.041666666666666664"}, {"6h", "0.25"}, {"12h", "0.5"}, {"1d", "1"},
	{"7d", "7"}, {"14d", "14"}, {"30d", "30"}, {"90d", "90"},
	{"180d", "180"}, {"365d", "365"},
}

func entityTopPeriods(slice string) []entityTopChoice {
	if slice == "right" {
		//nolint:misspell // alltime is the established public query value.
		return append(append([]entityTopChoice(nil), entityTopChoices...), entityTopChoice{"All Time", "alltime"})
	}
	return entityTopChoices
}

func validEntityTopPeriod(value, slice string) bool {
	for _, choice := range entityTopPeriods(slice) {
		if choice.Value == value {
			return true
		}
	}
	return false
}

func parseEntityIdentity(payload map[string]any, key string, out any) error {
	encoded, err := json.Marshal(payload[key])
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, out)
}

func loadEntityPage(ctx context.Context, reader EntityReader, kind string, id int64, tab string, shipGroup int64, killPage, killLimit int) (entityPageView, error) {
	tabs := entityTabs(kind)
	if len(tabs) == 0 || id < 1 || id > 2_147_483_647 || kind == "faction" && (id < 500_000 || id > 599_999) {
		return entityPageView{}, ErrNotFound
	}
	if tab == "" {
		tab = "dashboard"
	}
	valid := false
	for _, candidate := range tabs {
		if candidate.ID == tab {
			valid = true
			break
		}
	}
	if !valid {
		return entityPageView{}, ErrNotFound
	}
	var payload map[string]any
	var killlist api.KilllistPage
	var dashboard api.EntityDashboard
	var topLeft, topRight api.EntityTopData
	var achievements []api.EntityAchievement
	var members api.EntityMembersData
	var corporations api.EntityCorporationsData
	var battles api.EntityBattlesData
	var wars api.EntityWarsData
	var campaigns api.EntityCampaignsData
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
		payload, err = reader.EntityDetail(groupCtx, kind, id)
		return err
	})
	if tab == "combined" || tab == "kills" || tab == "losses" {
		group.Go(func() error {
			var err error
			killlist, err = reader.EntityKilllist(groupCtx, kind, id, tab, killPage, killLimit, nil, shipGroup)
			return err
		})
	}
	if tab == "dashboard" && kind != "faction" {
		group.Go(func() error {
			var err error
			dashboard, err = reader.EntityDashboard(groupCtx, kind, id, 90)
			return err
		})
	}
	if tab == "top" {
		group.Go(func() error {
			var err error
			topRight, err = reader.EntityTop(groupCtx, kind, id, "right", "7")
			return err
		})
		if kind != "character" {
			group.Go(func() error {
				var err error
				topLeft, err = reader.EntityTop(groupCtx, kind, id, "left", "7")
				return err
			})
		}
	}
	if tab == "achievements" {
		group.Go(func() error {
			var err error
			achievements, err = reader.EntityAchievements(groupCtx, id)
			return err
		})
	}
	if tab == "members" {
		group.Go(func() error {
			var err error
			members, err = reader.EntityMembers(groupCtx, kind, id, 1, "name", 0, 0)
			return err
		})
	}
	if kind == "alliance" && (tab == "corporations" || tab == "members") {
		group.Go(func() error {
			var err error
			corporations, err = reader.EntityCorporations(groupCtx, id, "members", "desc")
			return err
		})
	}
	if tab == "battles" {
		group.Go(func() error {
			var err error
			battles, err = reader.EntityBattles(groupCtx, kind, id, 1)
			return err
		})
	}
	if tab == "wars" {
		group.Go(func() error {
			var err error
			wars, err = reader.EntityWars(groupCtx, kind, id, 1, "recent")
			return err
		})
	}
	if tab == "campaigns" {
		group.Go(func() error {
			var err error
			campaigns, err = reader.EntityCampaigns(groupCtx, kind, id, 1)
			return err
		})
	}
	err := group.Wait()
	if err != nil {
		return entityPageView{}, err
	}
	if payload == nil {
		return entityPageView{}, ErrNotFound
	}
	view := entityPageView{Kind: kind, ID: id, Tab: tab, Tabs: tabs, ShipGroup: shipGroup, KillPage: killPage, KillLimit: killLimit, Killlist: killlist, Dashboard: dashboard, TopLeft: topLeft, TopRight: topRight, Achievements: achievements, Members: members, Corporations: corporations, Battles: battles, Wars: wars, Campaigns: campaigns}
	if err := parseEntityIdentity(payload, kind, &view.Identity); err != nil {
		return entityPageView{}, fmt.Errorf("decode %s identity: %w", kind, err)
	}
	if view.Identity.Name == "" {
		return entityPageView{}, ErrNotFound
	}
	if err := parseEntityIdentity(payload, "stats", &view.Stats); err != nil {
		return entityPageView{}, fmt.Errorf("decode %s stats: %w", kind, err)
	}
	if err := parseEntityIdentity(payload, "recentStats", &view.Recent); err != nil {
		return entityPageView{}, fmt.Errorf("decode %s recent stats: %w", kind, err)
	}
	var rankings struct {
		AllTime struct {
			Rating     int64 `json:"eve_kill_rating"`
			Rank       int64 `json:"overall_rank"`
			Population int64 `json:"population"`
		} `json:"all_time"`
	}
	if err := parseEntityIdentity(payload, "rankings", &rankings); err != nil {
		return entityPageView{}, fmt.Errorf("decode %s rankings: %w", kind, err)
	}
	view.Ranking = entityRanking(rankings.AllTime)
	view.Name = view.Identity.Name
	if view.Identity.Ticker != "" {
		view.Name += " [" + view.Identity.Ticker + "]"
	}
	view.Accent = paletteAccent(view.Identity.Palette)
	view.BioHTML = entityBioHTML(view.Identity.CustomDescriptionHTML, view.Identity.Description)
	if tab == "achievements" {
		sort.SliceStable(view.Achievements, func(i, j int) bool {
			left, right := view.Achievements[i], view.Achievements[j]
			if left.IsCompleted != right.IsCompleted {
				return left.IsCompleted
			}
			if achievementRarityRank(left.Rarity) != achievementRarityRank(right.Rarity) {
				return achievementRarityRank(left.Rarity) < achievementRarityRank(right.Rarity)
			}
			return left.Points > right.Points
		})
	}
	if tab == "history" {
		key := "corporationHistory"
		if kind == "corporation" {
			key = "allianceHistory"
		}
		if err := parseEntityIdentity(payload, key, &view.History); err != nil {
			return entityPageView{}, fmt.Errorf("decode %s history: %w", kind, err)
		}
		view.HistoryQueued, _ = payload["corporationHistoryQueued"].(bool)
	}
	imageID := id
	imageKind := kind
	if kind == "faction" {
		imageKind = "corporation"
		if view.Identity.CorporationID != 0 {
			imageID = view.Identity.CorporationID
		}
	}
	view.Image = eveEntityImage(imageKind, imageID, 256)
	view.Background = eveEntityImage(imageKind, imageID, 1024)
	if kind == "character" {
		if born, err := time.Parse(time.RFC3339, view.Identity.Birthday); err == nil {
			view.HasOldPortrait = born.Before(time.Date(2011, time.January, 18, 0, 0, 0, 0, time.UTC))
		}
	}
	return view, nil
}

func entityPageMetadata(view entityPageView) Metadata {
	basePath := fmt.Sprintf("/%s/%d", view.Kind, view.ID)
	path := basePath
	title := view.Name
	if view.Tab != "dashboard" {
		for _, tab := range view.Tabs {
			if tab.ID == view.Tab {
				title += " (" + tab.Label + ")"
				break
			}
		}
		path += "/" + view.Tab
	}
	if view.ShipGroup > 0 && (view.Tab == "combined" || view.Tab == "kills" || view.Tab == "losses") {
		path += "?ship_group=" + strconv.FormatInt(view.ShipGroup, 10)
	}
	description := fmt.Sprintf("%s — %s EVE Online combat stats on EVE-KILL.", view.Name, strings.ToLower(view.Kind))
	if view.Stats.Kills+view.Stats.Losses > 0 {
		description = fmt.Sprintf("%s — %s EVE Online combat stats: %s kills, %s losses on EVE-KILL.",
			view.Name, strings.ToLower(view.Kind), fullNumber(float64(view.Stats.Kills)), fullNumber(float64(view.Stats.Losses)))
	}
	crumbs := []map[string]any{{"@type": "ListItem", "position": 1, "name": "Home", "item": canonicalOrigin + "/"},
		{"@type": "ListItem", "position": 2, "name": view.Name, "item": canonicalOrigin + fmt.Sprintf("/%s/%d", view.Kind, view.ID)}}
	structured := []any{map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": crumbs}}
	switch view.Kind {
	case "character":
		structured = append(structured, map[string]any{"@context": "https://schema.org", "@type": "ProfilePage", "name": view.Name, "url": canonicalOrigin + basePath})
	case "corporation", "alliance":
		structured = append(structured, map[string]any{"@context": "https://schema.org", "@type": "Organization", "name": view.Name, "url": canonicalOrigin + basePath, "logo": view.Image})
	}
	return Metadata{
		Title: title, Description: description, CanonicalPath: path, Image: view.Image,
		ImageAlt: view.Name, Kind: map[bool]string{true: "profile", false: "website"}[view.Kind == "character"],
		Indexable:      true,
		StructuredData: structured,
	}
}

func (s *site) registerEntityPages(mux *http.ServeMux) {
	mux.HandleFunc("GET /_fragments/entity/killlist/{kind}/{id}", s.entityKilllistFragment)
	mux.HandleFunc("GET /_fragments/entity/top-lists/{kind}/{id}", s.entityTopListsFragment)
	mux.HandleFunc("GET /_fragments/entity/dashboard/{kind}/{id}", s.entityDashboardFragment)
	mux.HandleFunc("GET /_fragments/entity/top/{kind}/{id}", s.entityTopFragment)
	mux.HandleFunc("GET /_fragments/entity/members/{kind}/{id}", s.entityMembersFragment)
	mux.HandleFunc("GET /_fragments/entity/corporations/{id}", s.entityCorporationsFragment)
	mux.HandleFunc("GET /_fragments/entity/battles/{kind}/{id}", s.entityBattlesFragment)
	mux.HandleFunc("GET /_fragments/entity/wars/{kind}/{id}", s.entityWarsFragment)
	mux.HandleFunc("GET /_fragments/entity/campaigns/{kind}/{id}", s.entityCampaignsFragment)
	mux.HandleFunc("GET /_fragments/entity/intel/{kind}/{id}", s.entityIntelFragment)
	mux.HandleFunc("GET /_fragments/entity/faction/{id}", s.entityFactionOverviewFragment)
	mux.HandleFunc("GET /_fragments/entity/faction/{id}/militia", s.entityFactionMilitiaFragment)
	mux.HandleFunc("GET /_fragments/entity/tab/{kind}/{id}/{tab}", s.entityTabFragment)
	for _, kind := range []string{"character", "corporation", "alliance", "faction"} {
		kind := kind
		load := func(r *http.Request) (Page, error) {
			id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
			if err != nil {
				return Page{}, ErrNotFound
			}
			var shipGroup int64
			if raw := r.URL.Query().Get("ship_group"); raw != "" {
				shipGroup, err = strconv.ParseInt(raw, 10, 64)
				if err != nil || shipGroup < 1 || shipGroup > 2_147_483_647 {
					return Page{}, ErrNotFound
				}
			}
			view, err := loadEntityPage(r.Context(), s.options.Entities, kind, id, r.PathValue("tab"), shipGroup, 1, 50)
			if err != nil {
				return Page{}, err
			}
			return Page{Metadata: entityPageMetadata(view), Content: entityPage(view), SharedCacheTTL: 2 * time.Minute}, nil
		}
		mux.Handle("GET /"+kind+"/{id}", s.pageHandler(load))
		mux.Handle("GET /"+kind+"/{id}/{tab}", s.pageHandler(load))
	}
}

func (s *site) entityTabFragment(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if len(entityTabs(kind)) == 0 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	tab := r.PathValue("tab")
	killPage, err := fragmentInt(r.URL.Query().Get("page"), 1, 1, 1_000_000)
	if err != nil {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	killLimit, err := fragmentInt(r.URL.Query().Get("limit"), 50, 10, 100)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}
	var shipGroup int64
	if raw := r.URL.Query().Get("ship_group"); raw != "" {
		shipGroup, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || shipGroup < 1 || shipGroup > 2_147_483_647 {
			http.Error(w, "Invalid ship group", http.StatusBadRequest)
			return
		}
	}
	view, err := loadEntityPage(r.Context(), s.options.Entities, kind, id, tab, shipGroup, killPage, killLimit)
	if err != nil {
		if err == ErrNotFound {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Could not load tab", http.StatusServiceUnavailable)
		}
		return
	}
	s.renderFragment(w, r, entityMain(view))
}

func (s *site) entityTopFragment(w http.ResponseWriter, r *http.Request) {
	kind, slice, days := r.PathValue("kind"), r.URL.Query().Get("slice"), r.URL.Query().Get("days")
	if (kind != "character" && kind != "corporation" && kind != "alliance") ||
		(slice != "left" && slice != "right") || !validEntityTopPeriod(days, slice) ||
		(kind == "character" && slice == "left") {
		http.Error(w, "Invalid entity top selection", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	data, err := s.options.Entities.EntityTop(r.Context(), kind, id, slice, days)
	if err != nil {
		http.Error(w, "Could not load top lists", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, entityTopSlice(kind, slice, days, data))
}

func (s *site) entityDashboardFragment(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if kind != "character" && kind != "corporation" && kind != "alliance" {
		http.Error(w, "Invalid entity type", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	days, err := strconv.Atoi(r.URL.Query().Get("days"))
	if err != nil || (days != 0 && days != 14 && days != 30 && days != 90 && days != 180 && days != 365) {
		http.Error(w, "Invalid period", http.StatusBadRequest)
		return
	}
	data, err := s.options.Entities.EntityDashboard(r.Context(), kind, id, days)
	if err != nil {
		http.Error(w, "Could not load dashboard", http.StatusServiceUnavailable)
		return
	}
	if kind == "corporation" || kind == "alliance" {
		s.renderFragment(w, r, organizationDashboardContent(data, days))
		return
	}
	s.renderFragment(w, r, entityDashboardContent(data, days))
}

func (s *site) entityTopListsFragment(w http.ResponseWriter, r *http.Request) {
	kind, tab := r.PathValue("kind"), r.URL.Query().Get("tab")
	if (kind != "character" && kind != "corporation" && kind != "alliance") ||
		(tab != "combined" && tab != "kills" && tab != "losses") {
		http.Error(w, "Invalid entity top list", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	lists, err := s.options.Entities.EntityTopLists(r.Context(), kind, id, 7)
	if err != nil {
		http.Error(w, "Could not load top lists", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, entityTopLists(entityPageView{Kind: kind, ID: id, Tab: tab}, lists))
}

func (s *site) entityKilllistFragment(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if len(entityTabs(kind)) == 0 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 || id > 2_147_483_647 {
		http.NotFound(w, r)
		return
	}
	role := r.URL.Query().Get("role")
	if role != "combined" && role != "kills" && role != "losses" {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}
	page, err := fragmentInt(r.URL.Query().Get("page"), 1, 1, 1_000_000)
	if err != nil {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	limit, err := fragmentInt(r.URL.Query().Get("limit"), 50, 10, 100)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}
	var after *int64
	if raw := r.URL.Query().Get("after"); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || value < 1 {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}
		after = &value
	}
	var shipGroup int64
	if raw := r.URL.Query().Get("ship_group"); raw != "" {
		shipGroup, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || shipGroup < 1 || shipGroup > 2_147_483_647 {
			http.Error(w, "Invalid ship group", http.StatusBadRequest)
			return
		}
	}
	items, err := s.options.Entities.EntityKilllist(r.Context(), kind, id, role, page, limit, after, shipGroup)
	if err != nil {
		http.Error(w, "Could not load killlist", http.StatusServiceUnavailable)
		return
	}
	view := entityPageView{Kind: kind, ID: id, Tab: role, ShipGroup: shipGroup}
	s.renderFragment(w, r, entityKillList(view, items, page, limit))
}

func entityHeaderStyle(view entityPageView) string {
	if view.Accent == "" {
		return ""
	}
	return "--entity-accent:" + view.Accent
}

func entityTaxRate(rate float64) string { return fmt.Sprintf("%.0f%%", rate*100) }

func entityLPTaxRate(rate float64) string {
	return strings.TrimSuffix(fmt.Sprintf("%.1f", rate*100), ".0") + "% LP"
}

func entityCorpsMembers(view entityPageView) string {
	if view.Kind == "corporation" {
		return fullNumber(float64(view.Identity.MemberCount))
	}
	return fullNumber(float64(view.Identity.CorporationCount)) + " / " + fullNumber(float64(view.Identity.MemberCount))
}

func entityMembershipLabel(kind string) string {
	if kind == "corporation" {
		return "Members"
	}
	return "Corps / Members"
}

func entityDate(raw string) string {
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return ""
	}
	return value.Format("Jan 2, 2006")
}

func entityAge(raw string) int {
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return 0
	}
	now := time.Now()
	years := now.Year() - value.Year()
	if now.Month() < value.Month() || now.Month() == value.Month() && now.Day() < value.Day() {
		years--
	}
	return max(years, 0)
}

func entityTimeAgo(raw string) string {
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return "Unknown"
	}
	minutes := int(time.Since(value).Minutes())
	switch {
	case minutes < 1:
		return "Just now"
	case minutes < 60:
		return strconv.Itoa(minutes) + "m ago"
	case minutes < 24*60:
		return strconv.Itoa(minutes/60) + "h ago"
	case minutes < 30*24*60:
		return strconv.Itoa(minutes/(24*60)) + "d ago"
	default:
		return entityDate(raw)
	}
}

func entityNextBirthday(raw string) string {
	born, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return ""
	}
	now := time.Now()
	next := time.Date(now.Year(), born.Month(), born.Day(), 0, 0, 0, 0, now.Location())
	if now.After(next) {
		next = next.AddDate(1, 0, 0)
	}
	days := int(math.Ceil(next.Sub(now).Hours() / 24))
	return fmt.Sprintf("%s (%d days)", next.Format("Jan 2, 2006"), days)
}

func entityActivity(recent entityStats) string {
	rate := float64(recent.Kills) / 90
	switch {
	case rate >= 2:
		return "Very High"
	case rate >= 0.5:
		return "High"
	case rate >= 0.1:
		return "Medium"
	case rate > 0:
		return "Low"
	default:
		return "Inactive"
	}
}

func entitySoloRatio(stats entityStats) int64 {
	if stats.Kills == 0 {
		return 0
	}
	return int64(math.Round(float64(stats.SoloKills) / float64(stats.Kills) * 100))
}

func entityAvgKills(stats entityStats, birthday string) string {
	born, err := time.Parse(time.RFC3339, birthday)
	if err != nil {
		return "0"
	}
	days := max(1, int(time.Since(born).Hours()/24))
	return fmt.Sprintf("%.2f", float64(stats.Kills)/float64(days))
}

func entityDangerRatio(stats entityStats) int64 {
	if stats.Kills+stats.Losses == 0 {
		return 0
	}
	return int64(math.Round((1 - float64(stats.NPCLosses)/float64(max(stats.Losses, 1))) * 100))
}

func entityNPCLossRatio(stats entityStats) string {
	if stats.Losses == 0 {
		return "0%"
	}
	return strconv.FormatInt(int64(math.Round(float64(stats.NPCLosses)/float64(stats.Losses)*100)), 10) + "%"
}

func entityTabClass(active, id string) string {
	if active == id {
		return "active"
	}
	return ""
}

func entityTabCurrent(active, id string) string {
	if active == id {
		return "page"
	}
	return "false"
}

func entityPeriodClass(value, active int) string {
	if value == active {
		return "active"
	}
	return ""
}

func entityShipEntries(ships []api.EntityShipEntry) []api.TopEntry {
	entries := make([]api.TopEntry, 0, len(ships))
	for _, ship := range ships {
		entries = append(entries, api.TopEntry{ID: ship.ShipTypeID, Name: ship.ShipName, Count: ship.Count})
	}
	return entries
}

func entityTopPeriodClass(value, active string) string {
	if value == active {
		return "active"
	}
	return ""
}

func entityTopPageClass(kind string) string {
	if kind == "character" {
		return "entity-top-page character"
	}
	return "entity-top-page organization"
}

func entityHistoryID(view entityPageView, entry entityHistoryEntry) int64 {
	if view.Kind == "character" {
		return entry.CorporationID
	}
	return entry.AllianceID
}

func entityHistoryKind(view entityPageView) string {
	if view.Kind == "character" {
		return "corporation"
	}
	return "alliance"
}

func entityHistoryName(view entityPageView, entry entityHistoryEntry) string {
	if view.Kind == "character" {
		return fallbackName(entry.CorporationName)
	}
	return fallbackName(entry.AllianceName, "No Alliance")
}

func entityHistoryTicker(view entityPageView, entry entityHistoryEntry) string {
	if view.Kind == "character" {
		return entry.CorporationTicker
	}
	return entry.AllianceTicker
}

func entityHistoryDays(entries []entityHistoryEntry, index int) int {
	start, err := time.Parse(time.RFC3339, entries[index].StartDate)
	if err != nil {
		return 0
	}
	end := time.Now()
	if index > 0 {
		if parsed, err := time.Parse(time.RFC3339, entries[index-1].StartDate); err == nil {
			end = parsed
		}
	}
	return max(0, int(end.Sub(start).Hours()/24))
}

func entityHistoryDuration(days int) string {
	years := days / 365
	months := (days % 365) / 30
	if years > 0 && months > 0 {
		return fmt.Sprintf("%dy %dm", years, months)
	}
	if years > 0 {
		return fmt.Sprintf("%dy", years)
	}
	if months > 0 {
		return fmt.Sprintf("%dm", months)
	}
	return fmt.Sprintf("%dd", days)
}

func entityHistoryEnd(entries []entityHistoryEntry, index int) string {
	if index == 0 {
		return "Present"
	}
	return entityDate(entries[index-1].StartDate)
}

func entityHistoryClass(index int) string {
	if index == 0 {
		return "entity-history-row current"
	}
	return "entity-history-row"
}

func achievementRarityRank(rarity string) int {
	switch rarity {
	case "legendary":
		return 0
	case "epic":
		return 1
	case "rare":
		return 2
	case "uncommon":
		return 3
	case "common":
		return 4
	default:
		return 5
	}
}

func achievementCategories(rows []api.EntityAchievement) []string {
	seen := map[string]bool{}
	for _, row := range rows {
		seen[row.Category] = true
	}
	out := make([]string, 0, len(seen))
	for category := range seen {
		out = append(out, category)
	}
	sort.Strings(out)
	return out
}

func achievementTotalPoints(rows []api.EntityAchievement) int64 {
	var points int64
	for _, row := range rows {
		points += row.Points
	}
	return points
}

func achievementCompletedCount(rows []api.EntityAchievement) int {
	var count int
	for _, row := range rows {
		if row.IsCompleted {
			count++
		}
	}
	return count
}

func achievementPercent(rows []api.EntityAchievement) int {
	if len(rows) == 0 {
		return 0
	}
	return int(math.Round(float64(achievementCompletedCount(rows)) / float64(len(rows)) * 100))
}

func signedNumber(value int64) string {
	if value > 0 {
		return "+" + fullNumber(float64(value))
	}
	return fullNumber(float64(value))
}

func achievementProgress(row api.EntityAchievement) int64 {
	if row.IsCompleted {
		return 100
	}
	var previous int64
	if row.Level > 0 && int(row.Level) <= len(row.LevelThresholds) {
		previous = row.LevelThresholds[row.Level-1]
	}
	next := row.NextThreshold
	if next == 0 {
		next = row.Threshold
	}
	if next <= previous {
		return 0
	}
	return min(100, max(0, int64(math.Round(float64(row.CurrentCount-previous)/float64(next-previous)*100))))
}

func achievementNextThreshold(row api.EntityAchievement) int64 {
	if row.NextThreshold > 0 {
		return row.NextThreshold
	}
	return row.Threshold
}

func achievementClass(row api.EntityAchievement) string {
	class := "achievement-card rarity-" + row.Rarity
	if !row.IsCompleted && row.CurrentCount == 0 {
		class += " untouched"
	}
	return class
}

func entityHeatMax(heat map[string]int64) int64 {
	var maximum int64
	for _, count := range heat {
		if count > maximum {
			maximum = count
		}
	}
	return maximum
}

func entityHeatClass(heat map[string]int64, hour int) string {
	count := heat[strconv.Itoa(hour)]
	if count == 0 {
		return "empty"
	}
	ratio := float64(count) / float64(entityHeatMax(heat))
	switch {
	case ratio > .75:
		return "very-high"
	case ratio > .5:
		return "high"
	case ratio > .25:
		return "medium"
	default:
		return "low"
	}
}

func entityTabURL(view entityPageView, tab entityTab) string {
	base := fmt.Sprintf("/%s/%d", view.Kind, view.ID)
	if tab.ID == "dashboard" {
		return base
	}
	return base + "/" + tab.ID
}
