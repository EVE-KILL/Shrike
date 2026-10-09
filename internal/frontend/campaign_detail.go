package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type campaignDetailLocation struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type campaignDetailMember struct {
	EntityType   int     `json:"entity_type"`
	EntityID     int64   `json:"entity_id"`
	Name         string  `json:"name"`
	Kills        int64   `json:"kills"`
	Losses       int64   `json:"losses"`
	ISKDestroyed float64 `json:"isk_destroyed"`
	ISKLost      float64 `json:"isk_lost"`
}

type campaignDetailSide struct {
	SideIndex    int64                  `json:"side_index"`
	Name         string                 `json:"name"`
	Kills        int64                  `json:"kills"`
	Losses       int64                  `json:"losses"`
	ISKDestroyed float64                `json:"isk_destroyed"`
	ISKLost      float64                `json:"isk_lost"`
	Palette      *api.TopPalette        `json:"palette"`
	Entities     []campaignDetailMember `json:"entities"`
}

type campaignDetailTotals struct {
	KillCount            int64   `json:"killCount"`
	ISKDestroyed         float64 `json:"iskDestroyed"`
	CharactersInvolved   int64   `json:"charactersInvolved"`
	CorporationsInvolved int64   `json:"corporationsInvolved"`
	AlliancesInvolved    int64   `json:"alliancesInvolved"`
}

type campaignTopPilot struct {
	CharacterID       int64   `json:"characterId"`
	Name              string  `json:"name"`
	CorporationTicker string  `json:"corporationTicker"`
	CorporationName   string  `json:"corporationName"`
	AllianceTicker    string  `json:"allianceTicker"`
	AllianceName      string  `json:"allianceName"`
	Kills             int64   `json:"kills"`
	Losses            int64   `json:"losses"`
	ISKDestroyed      float64 `json:"iskDestroyed"`
	ISKLost           float64 `json:"iskLost"`
}

type campaignShipClass struct {
	GroupID int64   `json:"groupId"`
	Name    string  `json:"name"`
	Losses  int64   `json:"losses"`
	ISKLost float64 `json:"iskLost"`
}

type campaignTopSystem struct {
	SystemID     int64   `json:"systemId"`
	Name         string  `json:"name"`
	RegionName   string  `json:"regionName"`
	Kills        int64   `json:"kills"`
	ISKDestroyed float64 `json:"iskDestroyed"`
}

type campaignIntelPilot struct {
	CharacterID int64   `json:"characterId"`
	Name        string  `json:"name"`
	ShipName    string  `json:"shipName"`
	Died        bool    `json:"died"`
	Damage      float64 `json:"damage"`
}

type campaignIntelSide struct {
	FCs            []campaignIntelPilot `json:"fcs"`
	Logistics      []campaignIntelPilot `json:"logistics"`
	Capitals       []campaignIntelPilot `json:"capitals"`
	LogisticsCount int64                `json:"logisticsCount"`
	CapitalsCount  int64                `json:"capitalsCount"`
}

type campaignPrizeResult struct {
	Rank             int     `json:"rank"`
	CharacterID      int64   `json:"character_id"`
	CharacterName    string  `json:"character_name"`
	MetricValue      string  `json:"metric_value"`
	PayoutPercentage float64 `json:"payout_percentage"`
	PayoutAmount     float64 `json:"payout_amount"`
	Claimed          bool    `json:"claimed"`
	Paid             bool    `json:"paid"`
	CanClaim         bool    `json:"can_claim"`
}

type campaignPrizeContribution struct {
	ID              string `json:"id"`
	Source          string `json:"source"`
	ContributorID   int64  `json:"contributor_id"`
	ContributorName string `json:"contributor_name"`
	ContributorType string `json:"contributor_type"`
	Amount          string `json:"amount"`
	ContributedAt   string `json:"contributed_at"`
}

type campaignPrizePool struct {
	Metric               int                         `json:"metric"`
	MetricLabel          string                      `json:"metric_label"`
	WinnerCount          int                         `json:"winner_count"`
	PayoutPercentages    []float64                   `json:"payout_percentages"`
	Status               int                         `json:"status"`
	FundedTotal          string                      `json:"funded_total"`
	ContributionCount    int                         `json:"contribution_count"`
	RulesLockedAt        string                      `json:"rules_locked_at"`
	FinalizedAt          string                      `json:"finalized_at"`
	FundingReference     string                      `json:"funding_reference"`
	FundingClosesAt      string                      `json:"funding_closes_at"`
	SettlesAt            string                      `json:"settles_at"`
	LastWalletSync       string                      `json:"last_wallet_sync"`
	DiscordURL           string                      `json:"discord_url"`
	ProjectedLeadPercent *float64                    `json:"projected_lead_percent"`
	Results              []campaignPrizeResult       `json:"results"`
	Contributions        []campaignPrizeContribution `json:"contributions"`
}

type campaignDetailStats struct {
	Totals             campaignDetailTotals           `json:"totals"`
	TopKillersOverall  []campaignTopPilot             `json:"topKillersOverall"`
	TopVictimsOverall  []campaignTopPilot             `json:"topVictimsOverall"`
	ShipClassesOverall []campaignShipClass            `json:"shipClassesOverall"`
	TopKillersBySide   map[string][]campaignTopPilot  `json:"topKillersBySide"`
	ShipClassesBySide  map[string][]campaignShipClass `json:"shipClassesBySide"`
	TopSystems         []campaignTopSystem            `json:"topSystems"`
	IntelBySide        map[string]campaignIntelSide   `json:"intelBySide"`
	MostValuable       []struct {
		KillmailID            int64   `json:"killmailId"`
		ShipTypeID            int64   `json:"shipTypeId"`
		ShipName              string  `json:"shipName"`
		Value                 float64 `json:"value"`
		VictimCharacterName   string  `json:"victimCharacterName"`
		VictimCorporationName string  `json:"victimCorporationName"`
		VictimSide            *int    `json:"victimSide"`
	} `json:"mostValuable"`
}

type campaignDetailData struct {
	ID             string             `json:"campaign_id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	Mode           string             `json:"mode"`
	Status         int                `json:"status"`
	Visibility     int                `json:"visibility"`
	StartTime      string             `json:"start_time"`
	EndTime        string             `json:"end_time"`
	StatsUpdatedAt string             `json:"stats_updated_at"`
	LastActivityAt string             `json:"last_activity_at"`
	PrizePool      *campaignPrizePool `json:"prize_pool"`
	Creator        struct {
		CharacterID int64  `json:"character_id"`
		Name        string `json:"name"`
	} `json:"creator"`
	Locations struct {
		Systems        []campaignDetailLocation `json:"systems"`
		Constellations []campaignDetailLocation `json:"constellations"`
		Regions        []campaignDetailLocation `json:"regions"`
	} `json:"location_details"`
	Processing struct {
		Paused             bool   `json:"paused"`
		Note               string `json:"note"`
		EstimatedKillmails *int64 `json:"estimated_killmails"`
	} `json:"processing"`
	Sides           []campaignDetailSide `json:"sides"`
	AllowedEntities []campaignEditEntity `json:"allowed_entities"`
	Stats           *campaignDetailStats `json:"stats"`
	Daily           struct {
		Granularity string `json:"granularity"`
		Rows        []struct {
			Period       string  `json:"period"`
			SideIndex    int64   `json:"side_index"`
			Kills        int64   `json:"kills"`
			Losses       int64   `json:"losses"`
			ISKDestroyed float64 `json:"isk_destroyed"`
			ISKLost      float64 `json:"isk_lost"`
		} `json:"rows"`
	} `json:"daily"`
}

type campaignDetailView struct {
	Data campaignDetailData
	Tab  string
}

func validCampaignID(id string) bool {
	if len(id) != 14 {
		return false
	}
	for _, character := range id {
		if character >= '0' && character <= '9' || character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z' {
			continue
		}
		return false
	}
	return true
}

func (s *site) readCampaignDetail(r *http.Request, id string) (campaignDetailData, bool, error) {
	request := r.Clone(r.Context())
	address := *r.URL
	address.Path = "/api/campaign/" + id
	address.RawPath = ""
	address.RawQuery = ""
	request.URL = &address
	request.Method = http.MethodGet
	request.RequestURI = ""
	w := httptest.NewRecorder()
	s.options.CampaignsAPI.ServeHTTP(w, request)
	if w.Code == http.StatusNotFound {
		return campaignDetailData{}, false, nil
	}
	if w.Code != http.StatusOK {
		return campaignDetailData{}, false, fmt.Errorf("campaign API returned %d", w.Code)
	}
	var data campaignDetailData
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		return campaignDetailData{}, false, fmt.Errorf("decode campaign detail: %w", err)
	}
	return data, true, nil
}

func campaignDetailTab(data campaignDetailData, raw string) (string, bool) {
	if raw == "" || raw == "overview" {
		return "overview", true
	}
	if raw == "comments" || raw == "intel" && data.Mode != "area" {
		return raw, true
	}
	return "", false
}

func campaignDetailTitle(data campaignDetailData) string {
	if len(data.Sides) >= 2 {
		names := make([]string, 0, len(data.Sides))
		for _, side := range data.Sides {
			if side.Name != "" {
				names = append(names, side.Name)
			}
		}
		if len(names) >= 2 {
			return strings.Join(names, " vs ") + " — EVE Online Campaign"
		}
	}
	return data.Name + " — EVE Online Campaign"
}

func campaignDetailDescription(data campaignDetailData) string {
	parts := make([]string, 0, 4)
	if data.Mode == "area" {
		places := make([]string, 0, 4)
		for _, collection := range [][]campaignDetailLocation{data.Locations.Systems, data.Locations.Constellations, data.Locations.Regions} {
			for _, location := range collection {
				if len(places) < 4 {
					places = append(places, location.Name)
				}
			}
		}
		if len(places) > 0 {
			parts = append(parts, fmt.Sprintf("%s tracks battlefield activity in %s.", data.Name, strings.Join(places, ", ")))
		} else {
			parts = append(parts, data.Name+" tracks EVE Online battlefield activity.")
		}
	} else {
		parts = append(parts, data.Name+".")
	}
	if data.Stats != nil {
		parts = append(parts, fmt.Sprintf("%s killmails and %s ISK destroyed.", fullNumber(float64(data.Stats.Totals.KillCount)), compactNumber(data.Stats.Totals.ISKDestroyed)))
	}
	if date := warDate(data.StartTime); date != "" {
		if end := warDate(data.EndTime); end != "" {
			parts = append(parts, date+"–"+end+".")
		} else {
			parts = append(parts, "Running since "+date+".")
		}
	}
	return strings.Join(parts, " ")
}

func campaignDetailDateTime(raw string) string {
	when, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return ""
	}
	return when.UTC().Format("Jan 2, 2006, 03:04 PM")
}

func (s *site) loadCampaignDetailPage(r *http.Request) (Page, error) {
	id := r.PathValue("id")
	if !validCampaignID(id) {
		return Page{}, ErrNotFound
	}
	data, found, err := s.readCampaignDetail(r, id)
	if err != nil {
		return Page{}, err
	}
	if !found {
		return Page{}, ErrNotFound
	}
	tab, ok := campaignDetailTab(data, r.PathValue("tab"))
	if !ok {
		return Page{}, ErrNotFound
	}
	image := "/og-default.png"
	if len(data.Sides) > 0 && len(data.Sides[0].Entities) > 0 {
		entity := data.Sides[0].Entities[0]
		kind := "character"
		switch entity.EntityType {
		case 1:
			kind = "corporation"
		case 2:
			kind = "alliance"
		}
		image = eveEntityImage(kind, entity.EntityID, 512)
	}
	structured := map[string]any{"@context": "https://schema.org", "@type": "Event", "name": campaignDetailTitle(data), "description": campaignDetailDescription(data), "url": canonicalOrigin + "/campaign/" + id, "startDate": data.StartTime}
	if data.EndTime != "" {
		structured["endDate"] = data.EndTime
	}
	return Page{Metadata: Metadata{
		Title: campaignDetailTitle(data), CanonicalPath: "/campaign/" + id, Indexable: data.Visibility == 0,
		Description: campaignDetailDescription(data), Image: image, ImageAlt: data.Name + " on EVE-KILL",
		StructuredData: structured, CommentMedia: true, CommentEditor: true,
		Stylesheets: []string{"campaign-detail.css", "comment-thread.css", "utilities.css", "utility-page.css"}, ModuleScripts: []string{"campaign-detail.js", "comment-thread.js"},
	}, Content: campaignDetailPage(campaignDetailView{Data: data, Tab: tab})}, nil
}

func campaignDetailPath(id, tab string) string {
	base := "/campaign/" + url.PathEscape(id)
	if tab != "overview" {
		return base + "/" + tab
	}
	return base
}

func campaignDetailColors(data campaignDetailData) []string {
	return campaignSideColors(campaignDetailSideEntry(data))
}

func campaignDetailSideEntry(data campaignDetailData) api.EntityCampaignEntry {
	sides := make([]api.EntityCampaignSide, 0, len(data.Sides))
	for _, side := range data.Sides {
		sides = append(sides, api.EntityCampaignSide{SideIndex: side.SideIndex, Palette: side.Palette})
	}
	return api.EntityCampaignEntry{Sides: sides}
}

func campaignKillSidePayload(data campaignDetailData) map[string]any {
	sides := make([]map[string][]int64, 0, len(data.Sides))
	for _, side := range data.Sides {
		members := map[string][]int64{"characterIds": {}, "corporationIds": {}, "allianceIds": {}}
		for _, entity := range side.Entities {
			key := "characterIds"
			switch entity.EntityType {
			case 1:
				key = "corporationIds"
			case 2:
				key = "allianceIds"
			}
			members[key] = append(members[key], entity.EntityID)
		}
		sides = append(sides, members)
	}
	return map[string]any{"sideEntities": sides, "sideColors": campaignSidePaletteAccents(campaignDetailSideEntry(data))}
}

func campaignDetailColor(data campaignDetailData, index int) string {
	colors := campaignDetailColors(data)
	if index >= 0 && index < len(colors) {
		return colors[index]
	}
	return "#6b7280"
}

func campaignDetailTextColor(data campaignDetailData, index int) string {
	if accents := campaignSidePaletteAccents(campaignDetailSideEntry(data)); index >= 0 && index < len(accents) {
		return accents[index]
	}
	colors := []string{"var(--color-error)", "var(--color-brand-primary)", "var(--color-brand-secondary)", "#fb923c"}
	if index >= 0 && index < len(data.Sides) {
		position := data.Sides[index].SideIndex
		if position >= 0 && position < int64(len(colors)) {
			return colors[position]
		}
	}
	return "#6b7280"
}

func campaignDetailBorderColor(data campaignDetailData, index int) string {
	if accents := campaignSidePaletteAccents(campaignDetailSideEntry(data)); index >= 0 && index < len(accents) {
		return accents[index] + "59"
	}
	return "color-mix(in srgb," + campaignDetailTextColor(data, index) + " 20%,transparent)"
}

func campaignMostValuableStyle(data campaignDetailData, side *int) string {
	if side == nil || *side < 0 {
		return ""
	}
	if accents := campaignSidePaletteAccents(campaignDetailSideEntry(data)); *side < len(accents) {
		return "border-color:" + accents[*side] + "59"
	}
	colors := []string{"rgba(239,68,68,.45)", "rgba(59,130,246,.45)", "rgba(16,185,129,.45)", "rgba(249,115,22,.45)"}
	if *side >= len(colors) {
		return ""
	}
	return "border-color:" + colors[*side]
}

func campaignHasStats(data campaignDetailData) bool {
	return data.Stats != nil && (data.Processing.Paused || data.Status != 0)
}

func campaignDetailEfficiencyColor(side campaignDetailSide) string {
	if accent := paletteAccent(side.Palette); accent != "" {
		return accent
	}
	return "#22c55e99"
}

func campaignDetailEfficiency(side campaignDetailSide) string {
	total := side.ISKDestroyed + side.ISKLost
	if total <= 0 {
		return "0.0"
	}
	return fmt.Sprintf("%.1f", 100*side.ISKDestroyed/total)
}

func campaignDetailShare(data campaignDetailData, index int) float64 {
	if index < 0 || index >= len(data.Sides) {
		return 0
	}
	if len(data.Sides) == 1 {
		total := data.Sides[0].ISKDestroyed + data.Sides[0].ISKLost
		if total > 0 {
			return 100 * data.Sides[0].ISKDestroyed / total
		}
		return 0
	}
	var total float64
	for _, side := range data.Sides {
		total += side.ISKDestroyed
	}
	if total <= 0 {
		return 0
	}
	return 100 * data.Sides[index].ISKDestroyed / total
}

func campaignDetailDamage(value float64) string {
	switch {
	case value >= 1e9:
		return fmt.Sprintf("%.1fb", value/1e9)
	case value >= 1e6:
		return fmt.Sprintf("%.1fm", value/1e6)
	case value >= 1e3:
		return fmt.Sprintf("%.0fk", value/1e3)
	default:
		return fmt.Sprintf("%.0f", value)
	}
}

func campaignPrizeISK(raw string) string {
	return compactNumber(campaignPrizeFundedValue(raw))
}

func campaignPrizeFundedValue(raw string) float64 {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func campaignPrizeFundingOpen(pool *campaignPrizePool) bool {
	if pool == nil || pool.Status != 0 {
		return false
	}
	closes, err := time.Parse(time.RFC3339Nano, pool.FundingClosesAt)
	return err == nil && closes.After(time.Now())
}

func campaignPrizeStatus(pool *campaignPrizePool) string {
	if pool == nil {
		return ""
	}
	switch pool.Status {
	case 2:
		return "Paid"
	case 1:
		return "Official results"
	default:
		if campaignPrizeFundingOpen(pool) {
			return "Funding open"
		}
		return "Finalizing"
	}
}

func campaignPrizeScore(pool *campaignPrizePool, result campaignPrizeResult) string {
	value, _ := strconv.ParseFloat(result.MetricValue, 64)
	if pool != nil && (pool.Metric == 0 || pool.Metric == 1) {
		return fullNumber(value)
	}
	return compactNumber(value)
}

func campaignPrizeContributorImage(contribution campaignPrizeContribution) string {
	if contribution.ContributorID < 1 {
		return ""
	}
	switch contribution.ContributorType {
	case "character", "corporation", "alliance":
		return eveEntityImage(contribution.ContributorType, contribution.ContributorID, 64)
	default:
		return ""
	}
}

func campaignPrizeContributorPath(contribution campaignPrizeContribution) string {
	if contribution.ContributorID < 1 {
		return "#"
	}
	switch contribution.ContributorType {
	case "character", "corporation", "alliance":
		return idPath(contribution.ContributorType, contribution.ContributorID)
	default:
		return "#"
	}
}

func campaignDetailMemberPath(member campaignDetailMember) string {
	switch member.EntityType {
	case 2:
		return idPath("alliance", member.EntityID)
	case 1:
		return idPath("corporation", member.EntityID)
	default:
		return idPath("character", member.EntityID)
	}
}

func campaignDetailMemberImage(member campaignDetailMember) string {
	return campaignEntityImage(member.EntityType, member.EntityID)
}

func campaignDetailMemberName(member campaignDetailMember) string {
	if member.Name != "" {
		return member.Name
	}
	return "#" + strconv.FormatInt(member.EntityID, 10)
}

func campaignActivityPayload(data campaignDetailData) map[string]any {
	sides := make([]map[string]any, 0, len(data.Sides))
	for index, side := range data.Sides {
		sides = append(sides, map[string]any{"index": side.SideIndex, "name": side.Name, "color": campaignDetailColor(data, index)})
	}
	return map[string]any{
		"rows": data.Daily.Rows, "granularity": data.Daily.Granularity,
		"area": data.Mode == "area", "sides": sides,
	}
}

func (s *site) readCampaignKillfeed(r *http.Request, id, side string, limit int, after *int64) (api.KilllistPage, error) {
	request := r.Clone(r.Context())
	address := *r.URL
	address.Path = "/api/campaign/" + id + "/killlist"
	address.RawPath = ""
	query := url.Values{"limit": {strconv.Itoa(limit)}}
	if side != "" {
		query.Set("side", side)
	}
	if after != nil {
		query.Set("after", strconv.FormatInt(*after, 10))
	}
	address.RawQuery = query.Encode()
	request.URL = &address
	request.Method = http.MethodGet
	request.RequestURI = ""
	w := httptest.NewRecorder()
	s.options.CampaignsAPI.ServeHTTP(w, request)
	if w.Code != http.StatusOK {
		return api.KilllistPage{}, fmt.Errorf("campaign killfeed API returned %d", w.Code)
	}
	var page api.KilllistPage
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		return api.KilllistPage{}, fmt.Errorf("decode campaign killfeed: %w", err)
	}
	return page, nil
}

func (s *site) campaignKillfeedFragment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validCampaignID(id) {
		http.NotFound(w, r)
		return
	}
	side := r.URL.Query().Get("side")
	if side != "" {
		if _, err := strconv.ParseUint(side, 10, 16); err != nil {
			http.Error(w, "Invalid campaign side", http.StatusBadRequest)
			return
		}
	}
	var after *int64
	if raw := r.URL.Query().Get("after"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 1 {
			http.Error(w, "Invalid killfeed cursor", http.StatusBadRequest)
			return
		}
		after = &value
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value != 10 && value != 25 && value != 50 && value != 100 {
			http.Error(w, "Invalid killfeed limit", http.StatusBadRequest)
			return
		}
		limit = value
	}
	pageNumber := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 1_000_000 || value > 1 && after == nil {
			http.Error(w, "Invalid killfeed page", http.StatusBadRequest)
			return
		}
		pageNumber = value
	}
	page, err := s.readCampaignKillfeed(r, id, side, limit, after)
	if err != nil {
		http.Error(w, "Could not load campaign killfeed", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	s.renderFragment(w, r, killList(page, killListOptions{Kind: "campaign", Page: pageNumber, Limit: limit}))
}

func (s *site) campaignTabFragment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validCampaignID(id) {
		http.NotFound(w, r)
		return
	}
	data, found, err := s.readCampaignDetail(r, id)
	if err != nil {
		http.Error(w, "Could not load campaign", http.StatusServiceUnavailable)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	tab, ok := campaignDetailTab(data, r.PathValue("tab"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	s.renderFragment(w, r, campaignDetailTabContent(campaignDetailView{Data: data, Tab: tab}))
}

func (s *site) campaignPrizeFragment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validCampaignID(id) {
		http.NotFound(w, r)
		return
	}
	data, found, err := s.readCampaignDetail(r, id)
	if err != nil {
		http.Error(w, "Could not load campaign prize pool", http.StatusServiceUnavailable)
		return
	}
	if !found || data.PrizePool == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	s.renderFragment(w, r, campaignPrizeDialog(*data.PrizePool))
}
