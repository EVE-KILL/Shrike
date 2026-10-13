package frontend

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type battleCorporation struct {
	ID        int64   `json:"corporation_id"`
	Name      string  `json:"corporation_name"`
	Kills     int64   `json:"kills"`
	Losses    int64   `json:"losses"`
	Destroyed float64 `json:"isk_destroyed"`
	Lost      float64 `json:"isk_lost"`
}
type battleAlliance struct {
	ID           *int64              `json:"alliance_id"`
	Name         string              `json:"alliance_name"`
	Kills        int64               `json:"kills"`
	Losses       int64               `json:"losses"`
	Destroyed    float64             `json:"isk_destroyed"`
	Lost         float64             `json:"isk_lost"`
	Corporations []battleCorporation `json:"corporations"`
}
type battleTeam struct {
	Name          string           `json:"name"`
	Index         int              `json:"team_index"`
	Kills         int64            `json:"total_kills"`
	Losses        int64            `json:"total_losses"`
	Destroyed     float64          `json:"total_isk_destroyed"`
	Lost          float64          `json:"total_isk_lost"`
	AllianceCount int              `json:"alliance_count"`
	CorpCount     int              `json:"corp_count"`
	Alliances     []battleAlliance `json:"alliances"`
	Palette       *api.TopPalette  `json:"dominant_corp_palette"`
}
type battleUnsided struct {
	Kills         int64            `json:"kills"`
	Losses        int64            `json:"losses"`
	Destroyed     float64          `json:"isk_destroyed"`
	Lost          float64          `json:"isk_lost"`
	AllianceCount int              `json:"alliance_count"`
	CorpCount     int              `json:"corp_count"`
	Alliances     []battleAlliance `json:"alliances"`
}
type battleSide struct {
	Corps     []int64 `json:"corps"`
	Alliances []int64 `json:"alliances"`
}
type battleDetailData struct {
	BattleID     int64          `json:"battle_id"`
	Redirect     bool           `json:"redirect"`
	SystemID     int64          `json:"solar_system_id"`
	SystemName   string         `json:"solar_system_name"`
	Security     *float64       `json:"solar_system_security"`
	RegionID     int64          `json:"region_id"`
	RegionName   string         `json:"region_name"`
	Start        string         `json:"start_time"`
	End          string         `json:"end_time"`
	Minutes      int            `json:"duration_minutes"`
	KillCount    int64          `json:"kill_count"`
	Destroyed    float64        `json:"total_isk_destroyed"`
	Damage       int64          `json:"total_damage"`
	Characters   int64          `json:"characters_involved"`
	Corporations int64          `json:"corporations_involved"`
	Alliances    int64          `json:"alliances_involved"`
	MultiParty   bool           `json:"is_multi_party"`
	Custom       bool           `json:"is_custom"`
	Teams        []battleTeam   `json:"teams"`
	Sides        []battleSide   `json:"team_entities"`
	Unsided      *battleUnsided `json:"unsided"`
}
type battleReplayKill struct {
	ID         int64    `json:"killmail_id"`
	Time       string   `json:"killmail_time"`
	SystemID   int64    `json:"solar_system_id"`
	SystemName string   `json:"solar_system_name"`
	X          *float64 `json:"position_x"`
	Y          *float64 `json:"position_y"`
	Z          *float64 `json:"position_z"`
	VictimName string   `json:"victim_character_name"`
	CorpID     int64    `json:"victim_corporation_id"`
	AllianceID int64    `json:"victim_alliance_id"`
	ShipName   string   `json:"ship_name"`
	ShipID     int64    `json:"ship_type_id"`
	GroupID    int64    `json:"ship_group_id"`
	GroupName  string   `json:"ship_group_name"`
	Value      float64  `json:"total_value"`
}
type battleLandmark struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	SystemID int64   `json:"solar_system_id"`
	GroupID  int64   `json:"group_id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Z        float64 `json:"z"`
}
type battleReplayData struct {
	Kills     []battleReplayKill `json:"kills"`
	Landmarks []battleLandmark   `json:"landmarks"`
}
type battleLossFilter struct {
	Side   *int64
	Group  *int64
	MinISK float64
	From   *int64
	To     *int64
}
type battleLossTotal struct {
	Count int
	ISK   float64
}
type battleLossBin struct {
	Start, End int64
	Count      int
	ISK        float64
	Big        int
	Sides      []battleLossTotal
}
type battleShipGroup struct {
	ID   int64
	Name string
}
type battleDetailView struct {
	ID                 int64
	KillmailID         int64
	Tab                string
	Query              url.Values
	Data               battleDetailData
	Failed             bool
	Error              string
	Filter             battleLossFilter
	Replay             battleReplayData
	Filtered           []battleReplayKill
	Biggest            []battleReplayKill
	Supers             []battleReplayKill
	Groups             []battleShipGroup
	Bins               []battleLossBin
	PeakCount, PeakISK float64
	Busiest            battleLossBin
	LossISK            float64
	Killlist           api.KilllistPage
	Composition        battleCompositionData
	CompositionRows    [][]battlePilot
	Intel              battleIntelData
	Page, Limit        int
}

var battleTabs = []struct{ ID, Label, Icon string }{
	{"summary", "Summary", "layout-list"}, {"kills", "Kills", "target"}, {"timeline", "Timeline", "clock"},
	{"replay", "Map replay", "play"}, {"composition", "Composition", "users"}, {"intel", "Intel", "scan-eye"}, {"comments", "Comments", "message-square"},
}

func parseBattleDetailRoute(r *http.Request) (battleDetailView, error) {
	v := battleDetailView{Query: r.URL.Query(), Tab: r.PathValue("tab"), Page: 1, Limit: 50}
	var err error
	v.ID, err = strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || v.ID < 1 {
		return v, ErrNotFound
	}
	if raw := v.Query.Get("killmail"); raw != "" {
		v.KillmailID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || v.KillmailID < 1 {
			return v, ErrNotFound
		}
	}
	if v.Tab == "" {
		v.Tab = "summary"
	}
	valid := false
	for _, tab := range battleTabs {
		if v.Tab == tab.ID {
			valid = true
		}
	}
	if !valid || v.KillmailID > 0 && v.Tab == "comments" {
		return v, ErrNotFound
	}
	// Nuxt ignores non-finite filter values; retain that behavior on direct URLs.
	number := func(key string) *int64 {
		raw := v.Query.Get(key)
		if raw == "" {
			return nil
		}
		f, e := strconv.ParseFloat(raw, 64)
		if e != nil || math.IsNaN(f) || math.IsInf(f, 0) || f > math.MaxInt64 || f < math.MinInt64 {
			return nil
		}
		n := int64(f)
		return &n
	}
	v.Filter = battleLossFilter{Side: number("side"), Group: number("group"), From: number("from"), To: number("to")}
	if f, e := strconv.ParseFloat(v.Query.Get("minIsk"), 64); e == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
		v.Filter.MinISK = max(0, f)
	}
	return v, nil
}
func (v battleDetailView) endpoint() string {
	if v.KillmailID > 0 {
		return fmt.Sprintf("/api/battle/killmail/%d", v.KillmailID)
	}
	return fmt.Sprintf("/api/battle/%d", v.ID)
}
func (v battleDetailView) path(tab string, overrides url.Values) string {
	q := make(url.Values, len(v.Query))
	for key, values := range v.Query {
		q[key] = append([]string{}, values...)
	}
	for key, values := range overrides {
		q.Del(key)
		for _, value := range values {
			q.Add(key, value)
		}
	}
	path := fmt.Sprintf("/battle/%d", v.ID)
	if tab != "summary" {
		path += "/" + tab
	}
	if query := q.Encode(); query != "" {
		path += "?" + query
	}
	return path
}
func (v battleDetailView) replayLink(k battleReplayKill) string {
	return v.path("replay", url.Values{"at": {strconv.FormatInt(battleTimeMS(k.Time), 10)}, "kill": {strconv.FormatInt(k.ID, 10)}})
}
func battleTimeMS(raw string) int64 {
	t, e := time.Parse(time.RFC3339Nano, raw)
	if e != nil {
		return 0
	}
	return t.UnixMilli()
}
func battleEVEDate(raw string) string {
	t, e := time.Parse(time.RFC3339Nano, raw)
	if e != nil {
		return ""
	}
	return t.UTC().Format("Jan 2, 2006, 15:04 EVE")
}
func battleEVETime(ms int64) string { return time.UnixMilli(ms).UTC().Format("01-02 15:04 EVE") }
func battleDetailDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%dh", minutes/60)
	}
	return fmt.Sprintf("%dh %dm", minutes/60, minutes%60)
}
func battleSideName(side int) string {
	if side < 0 {
		return "Unassigned"
	}
	return fmt.Sprintf("Team %c", 'A'+side)
}
func battleSideColor(side int) string {
	colors := []string{"#f87171", "#60a5fa", "#c084fc", "#fbbf24"}
	if side < 0 || side >= len(colors) {
		return "#94a3b8"
	}
	return colors[side]
}
func battleVictimSide(k battleReplayKill, sides []battleSide) int {
	for i, side := range sides {
		for _, id := range side.Corps {
			if id > 0 && id == k.CorpID {
				return i
			}
		}
	}
	for i, side := range sides {
		for _, id := range side.Alliances {
			if id > 0 && id == k.AllianceID {
				return i
			}
		}
	}
	return -1
}
func battleFilterActive(f battleLossFilter) bool {
	return f.Side != nil || f.Group != nil || f.MinISK > 0 || f.From != nil || f.To != nil
}

func battleClearFilters() url.Values {
	return url.Values{"side": nil, "group": nil, "minIsk": nil, "from": nil, "to": nil}
}
func buildBattleLosses(v *battleDetailView) {
	start, end := battleTimeMS(v.Data.Start), battleTimeMS(v.Data.End)
	step := int64(180 * 60000)
	for _, minutes := range []int64{1, 5, 15, 30, 60, 180} {
		if float64(end-start)/60000/float64(minutes) <= 60 {
			step = minutes * 60000
			break
		}
	}
	count := max(1, int(math.Ceil(float64(end-start)/float64(step))))
	v.Bins = make([]battleLossBin, count)
	for i := range v.Bins {
		binEnd := start + int64(i+1)*step
		if i == count-1 {
			binEnd = end + 1
		}
		v.Bins[i] = battleLossBin{Start: start + int64(i)*step, End: binEnd, Sides: make([]battleLossTotal, len(v.Data.Sides)+1)}
	}
	groups := map[int64]string{}
	for _, k := range v.Replay.Kills {
		if k.GroupID > 0 {
			name := k.GroupName
			if name == "" {
				name = fmt.Sprintf("Class %d", k.GroupID)
			}
			groups[k.GroupID] = name
		}
		t := battleTimeMS(k.Time)
		side := battleVictimSide(k, v.Data.Sides)
		f := v.Filter
		if t == 0 || f.Side != nil && int64(side) != *f.Side || f.Group != nil && k.GroupID != *f.Group || k.Value < f.MinISK || f.From != nil && t < *f.From || f.To != nil && t >= *f.To {
			continue
		}
		v.Filtered = append(v.Filtered, k)
		v.LossISK += k.Value
		if k.GroupID == 30 || k.GroupID == 659 {
			v.Supers = append(v.Supers, k)
		}
		if t < start || t > end {
			continue
		}
		index := min(count-1, int((t-start)/step))
		bin := &v.Bins[index]
		bin.Count++
		bin.ISK += k.Value
		if side < 0 {
			side = len(v.Data.Sides)
		}
		bin.Sides[side].Count++
		bin.Sides[side].ISK += k.Value
		if k.GroupID == 30 || k.GroupID == 659 {
			bin.Big++
		}
	}
	for id, name := range groups {
		v.Groups = append(v.Groups, battleShipGroup{id, name})
	}
	sort.Slice(v.Groups, func(i, j int) bool { return v.Groups[i].Name < v.Groups[j].Name })
	v.Biggest = append([]battleReplayKill{}, v.Filtered...)
	sort.SliceStable(v.Biggest, func(i, j int) bool { return v.Biggest[i].Value > v.Biggest[j].Value })
	v.Biggest = v.Biggest[:min(6, len(v.Biggest))]
	v.Busiest = v.Bins[0]
	v.PeakCount, v.PeakISK = 1, 1
	for _, bin := range v.Bins {
		v.PeakCount = max(v.PeakCount, float64(bin.Count))
		v.PeakISK = max(v.PeakISK, bin.ISK)
		if bin.Count > v.Busiest.Count {
			v.Busiest = bin
		}
	}
}
func (s *site) battleAPI(r *http.Request, path string, query url.Values, out any) (int, error) {
	request := r.Clone(r.Context())
	address := *r.URL
	address.Path = path
	address.RawPath = ""
	address.RawQuery = query.Encode()
	request.URL = &address
	request.Method = http.MethodGet
	request.RequestURI = ""
	w := httptest.NewRecorder()
	s.options.ConflictsAPI.ServeHTTP(w, request)
	if w.Code != http.StatusOK {
		return w.Code, fmt.Errorf("battle API returned %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		return w.Code, fmt.Errorf("decode battle: %w", err)
	}
	return w.Code, nil
}
func (s *site) readBattleDetail(r *http.Request) (battleDetailView, error) {
	v, err := parseBattleDetailRoute(r)
	if err != nil {
		return v, err
	}
	status, err := s.battleAPI(r, v.endpoint(), nil, &v.Data)
	if v.KillmailID > 0 && (err != nil || v.Data.Teams == nil && !v.Data.Redirect) {
		v.Failed = true
		return v, nil
	}
	if status == http.StatusNotFound {
		return v, ErrNotFound
	}
	return v, err
}
func (s *site) loadBattleTab(r *http.Request, v *battleDetailView) {
	if v.Failed || v.Tab == "comments" {
		return
	}
	if v.Tab == "composition" {
		if _, err := s.battleAPI(r, v.endpoint()+"/composition", nil, &v.Composition); err != nil {
			v.Error = "Unable to load composition."
		}
		for _, team := range v.Composition.Teams {
			v.CompositionRows = append(v.CompositionRows, battleCompositionInitialRows(team, v.Filter.Group))
		}
		return
	}
	if v.Tab == "intel" {
		if _, err := s.battleAPI(r, v.endpoint()+"/intel", nil, &v.Intel); err != nil {
			v.Error = "Unable to load intelligence."
		}
		return
	}
	if v.Tab == "summary" || v.Tab == "kills" || v.Tab == "timeline" || v.Tab == "replay" {
		_, err := s.battleAPI(r, v.endpoint()+"/replay", nil, &v.Replay)
		if err != nil {
			v.Error = "Unable to load battle losses."
			return
		}
		buildBattleLosses(v)
	}
	if v.Tab == "kills" {
		if err := s.readBattleKillfeed(r, v); err != nil {
			v.Error = "Unable to load battle losses."
		}
	}
}
func (s *site) readBattleKillfeed(r *http.Request, v *battleDetailView) error {
	q := url.Values{"limit": {strconv.Itoa(v.Limit)}}
	for key, value := range map[string]*int64{"side": v.Filter.Side, "group": v.Filter.Group, "from": v.Filter.From, "to": v.Filter.To} {
		if value != nil {
			q.Set(key, strconv.FormatInt(*value, 10))
		}
	}
	if v.Filter.MinISK > 0 {
		q.Set("minIsk", strconv.FormatFloat(v.Filter.MinISK, 'f', -1, 64))
	}
	if cursor := r.URL.Query().Get("after"); cursor != "" {
		id, e := strconv.ParseInt(cursor, 10, 64)
		if e != nil || id < 1 {
			return fmt.Errorf("invalid killmail cursor")
		}
		q.Set("after", cursor)
	}
	_, err := s.battleAPI(r, v.endpoint()+"/killlist", q, &v.Killlist)
	return err
}
func battleTeamTopName(team battleTeam) string {
	if len(team.Alliances) > 0 {
		return battleAllianceName(team.Alliances[0])
	}
	return fmt.Sprintf("Team %d", team.Index+1)
}
func battleAllianceName(a battleAlliance) string {
	if a.Name != "" {
		return a.Name
	}
	if len(a.Corporations) > 0 && a.Corporations[0].Name != "" {
		return a.Corporations[0].Name
	}
	return "Unknown"
}
func battleAllianceImage(a battleAlliance, size int) string {
	if a.ID != nil {
		return eveEntityImage("alliance", *a.ID, size)
	}
	if len(a.Corporations) > 0 {
		return eveEntityImage("corporation", a.Corporations[0].ID, size)
	}
	return ""
}
func battleSortedAlliances(rows []battleAlliance) []battleAlliance {
	result := append([]battleAlliance{}, rows...)
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID != nil && result[j].ID == nil })
	return result
}
func battlePaletteAccents(data battleDetailData) []string {
	if len(data.Teams) == 2 {
		entry := api.EntityCampaignEntry{}
		for _, team := range data.Teams {
			entry.Sides = append(entry.Sides, api.EntityCampaignSide{Palette: team.Palette})
		}
		return campaignSidePaletteAccents(entry)
	}
	return nil
}
func battleTeamColor(data battleDetailData, index int) string {
	if colors := battlePaletteAccents(data); len(colors) == 2 {
		return colors[index]
	}
	if index == 0 {
		return "var(--color-error)"
	}
	return "var(--color-brand-primary)"
}
func battleTeamStyle(data battleDetailData, index int) string {
	accent := battleTeamColor(data, index)
	summary := accent
	if battlePaletteAccents(data) == nil {
		summary = "color-mix(in srgb," + accent + " 80%,transparent)"
	}
	return "--battle-accent:" + accent + ";--battle-summary-accent:" + summary
}
func battleEfficiency(data battleDetailData) int {
	if len(data.Teams) < 2 {
		return 50
	}
	sum := data.Teams[0].Destroyed + data.Teams[1].Destroyed
	if sum == 0 {
		return 50
	}
	return int(math.Round(data.Teams[0].Destroyed / sum * 100))
}
func battleEditSides(v battleDetailView) string {
	q := url.Values{"system": {strconv.FormatInt(v.Data.SystemID, 10)}, "systemName": {v.Data.SystemName}, "start": {v.Data.Start}, "end": {v.Data.End}, "step": {"sides"}}
	if v.KillmailID == 0 {
		q.Set("battle", strconv.FormatInt(v.ID, 10))
	}
	return "/battlegenerator?" + q.Encode()
}
func battleSecurity(sec *float64) string {
	if sec == nil {
		return ""
	}
	return fmt.Sprintf("%.1f", *sec)
}
func battleSecurityClass(sec *float64) string {
	if sec == nil {
		return "text-gray-500"
	}
	if *sec >= 0.5 {
		return "text-green-400"
	}
	if *sec > 0 {
		return "text-amber-400"
	}
	return "text-red-400"
}
func battleUnknown(name string) string {
	if name == "" {
		return "Unknown"
	}
	return name
}
func battleFilterTime(value *int64, fallback int64) int64 {
	if value != nil {
		return *value
	}
	return fallback
}
func battleBinSide(index, count int) int {
	if index == count {
		return -1
	}
	return index
}
func battleBinHeight(v battleDetailView, side battleLossTotal, isk bool) float64 {
	if isk {
		return side.ISK / v.PeakISK * 100
	}
	return float64(side.Count) / v.PeakCount * 100
}

type battleTile struct{ Label, Icon, Value, Color string }

func battleDetailTiles(d battleDetailData) []battleTile {
	return []battleTile{{"Ships Lost", "crosshair", fullNumber(float64(d.KillCount)), "text-white"}, {"ISK Destroyed", "coins", compactNumber(d.Destroyed), "text-yellow-400"}, {"Damage", "zap", fullNumber(float64(d.Damage)), "text-gray-300"}, {"Characters", "user-round", fullNumber(float64(d.Characters)), "text-blue-400"}, {"Corporations", "building-2", fullNumber(float64(d.Corporations)), "text-purple-400"}, {"Alliances", "shield", fullNumber(float64(d.Alliances)), "text-indigo-400"}}
}

type battleTeamMetric struct{ Label, Value, Color, Unsided string }

func battleTeamMetrics(d battleDetailData, index int) []battleTeamMetric {
	t := d.Teams[index]
	m := []battleTeamMetric{{"kills", fullNumber(float64(t.Kills)), "text-green-400", ""}, {"losses", fullNumber(float64(t.Losses)), "text-red-400", ""}, {"destroyed", compactNumber(t.Destroyed), "text-green-400", ""}, {"lost", compactNumber(t.Lost), "text-red-400", ""}}
	if u := d.Unsided; u != nil {
		if u.Kills > 0 {
			m[0].Unsided = fullNumber(float64(u.Kills))
		}
		if u.Losses > 0 {
			m[1].Unsided = fullNumber(float64(u.Losses))
		}
		if u.Destroyed > 0 {
			m[2].Unsided = compactNumber(u.Destroyed)
		}
		if u.Lost > 0 {
			m[3].Unsided = compactNumber(u.Lost)
		}
	}
	return m
}
func battleMetadata(v battleDetailView) Metadata {
	assets := Metadata{Stylesheets: []string{"utilities.css", "utility-page.css", "battle-detail.css", "comment-thread.css"}, ModuleScripts: []string{"battle-detail.js"}, CommentMedia: true, CommentEditor: true}
	if v.Failed {
		assets.Title = "Battle"
		assets.Description = "View battle report on EVE-KILL."
		return assets
	}
	title := "Battle in " + v.Data.SystemName + " — " + battleEVEDate(v.Data.Start)
	description := fmt.Sprintf("%s ships lost and %s ISK destroyed in %s over %s.", fullNumber(float64(v.Data.KillCount)), compactNumber(v.Data.Destroyed), v.Data.SystemName, battleDetailDuration(v.Data.Minutes))
	if len(v.Data.Teams) >= 2 {
		description += " " + battleTeamTopName(v.Data.Teams[0]) + " vs " + battleTeamTopName(v.Data.Teams[1]) + "."
	}
	image := ""
	if len(v.Data.Teams) > 0 && len(v.Data.Teams[0].Alliances) > 0 {
		image = battleAllianceImage(v.Data.Teams[0].Alliances[0], 256)
	}
	base := fmt.Sprintf("/battle/%d", v.ID)
	baseTitle := title
	if v.Tab != "summary" {
		for _, tab := range battleTabs {
			if v.Tab == tab.ID {
				title += " (" + tab.Label + ")"
			}
		}
	}
	ogTitle := "Battle in " + v.Data.SystemName + " — EVE-KILL"
	if len(v.Data.Teams) >= 2 {
		ogTitle = battleTeamTopName(v.Data.Teams[0]) + " vs " + battleTeamTopName(v.Data.Teams[1]) + " — " + v.Data.SystemName
	}
	return Metadata{Title: title, OGTitle: ogTitle, TwitterTitle: baseTitle + " — EVE-KILL", Description: description, CanonicalPath: base, Indexable: v.KillmailID == 0, Image: image, TwitterCard: "summary", CommentMedia: true, CommentEditor: true,
		Stylesheets: []string{"utilities.css", "utility-page.css", "battle-detail.css", "comment-thread.css"}, ModuleScripts: []string{"battle-detail.js"},
		StructuredData: map[string]any{"@context": "https://schema.org", "@type": "Event", "name": title, "url": canonicalOrigin + base, "startDate": v.Data.Start, "endDate": v.Data.End, "eventStatus": "https://schema.org/EventCompleted", "eventAttendanceMode": "https://schema.org/OnlineEventAttendanceMode", "location": map[string]any{"@type": "Place", "name": v.Data.SystemName, "url": canonicalOrigin + fmt.Sprintf("/system/%d", v.Data.SystemID)}}}
}
func (s *site) battlePage(w http.ResponseWriter, r *http.Request) {
	s.pageHandler(func(r *http.Request) (Page, error) {
		v, err := s.readBattleDetail(r)
		if err != nil {
			return Page{}, err
		}
		if v.KillmailID > 0 && v.Data.Redirect && v.Data.BattleID > 0 {
			return Page{RedirectPath: fmt.Sprintf("/battle/%d", v.Data.BattleID)}, nil
		}
		s.loadBattleTab(r, &v)
		return Page{Metadata: battleMetadata(v), Content: battleDetailPage(v)}, nil
	}).ServeHTTP(w, r)
}
func (s *site) battleTabFragment(w http.ResponseWriter, r *http.Request) {
	v, err := s.readBattleDetail(r)
	if err == ErrNotFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Unable to load battle", http.StatusServiceUnavailable)
		return
	}
	s.loadBattleTab(r, &v)
	w.Header().Set("Cache-Control", "private, no-store")
	s.renderFragment(w, r, battleTabContent(v))
}
func (s *site) battleKillfeedFragment(w http.ResponseWriter, r *http.Request) {
	v, err := parseBattleDetailRoute(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		v.Limit, err = strconv.Atoi(raw)
		if err != nil || v.Limit != 10 && v.Limit != 25 && v.Limit != 50 && v.Limit != 100 {
			http.Error(w, "Invalid limit", 400)
			return
		}
	}
	if raw := r.URL.Query().Get("after"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err != nil || id < 1 {
			http.Error(w, "Invalid cursor", 400)
			return
		}
	}
	if raw := r.URL.Query().Get("page"); raw != "" {
		v.Page, err = strconv.Atoi(raw)
		if err != nil || v.Page < 1 || v.Page > 1_000_000 || v.Page > 1 && r.URL.Query().Get("after") == "" {
			http.Error(w, "Invalid page", 400)
			return
		}
	}
	if err := s.readBattleKillfeed(r, &v); err != nil {
		http.Error(w, "Unable to load battle losses", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	s.renderFragment(w, r, killList(v.Killlist, killListOptions{Kind: "battle", Page: v.Page, Limit: v.Limit}))
}
