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

	"github.com/a-h/templ"
)

var settingsSections = []struct{ ID, Label, Icon string }{
	{"overview", "Overview", "user"}, {"wallet", "Wallet", "wallet-cards"}, {"sessions", "Sessions", "monitor-smartphone"},
	{"description", "Description", "user-pen"}, {"esi", "ESI", "activity"}, {"appearance", "Appearance", "palette"},
	{"preferences", "Preferences", "sliders-horizontal"}, {"domains", "Domains", "globe"}, {"comments", "Comments", "message-square"},
}

type settingsUser struct {
	ID              int64          `json:"characterId"`
	Name            string         `json:"characterName"`
	OwnerHash       string         `json:"characterOwnerHash"`
	Admin           bool           `json:"isAdmin"`
	CorporationID   int64          `json:"corporationId"`
	CorporationName string         `json:"corporationName"`
	AllianceID      int64          `json:"allianceId"`
	AllianceName    string         `json:"allianceName"`
	Settings        map[string]any `json:"settings"`
}

type settingsOverviewData struct {
	Account struct{ CreatedAt, LastLogin *time.Time } `json:"account"`
	Token   *struct {
		ScopeCount  int        `json:"scopeCount"`
		TokenExpiry *time.Time `json:"tokenExpiry"`
		LastFetched *time.Time `json:"lastFetched"`
	} `json:"esiToken"`
	RawStats map[string]any `json:"esiStats"`
}

type settingsView struct {
	Active, Title    string
	User             *settingsUser
	Overview         settingsOverviewData
	OverviewError    bool
	Now              time.Time
	Descriptions     settingsDescriptionsData
	DescriptionError bool
	Wallet           settingsWalletData
	WalletError      bool
	ESI              settingsESIData
	Comments         settingsCommentsData
	CommentsError    bool
	Domains          settingsDomainsData
	DomainsError     bool
}

type settingsBio struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Ticker          string  `json:"ticker"`
	CanEdit         bool    `json:"canEdit"`
	CEOName         *string `json:"ceo_name"`
	ExecutorCEOName *string `json:"executor_ceo_name"`
	ESIDescription  *string `json:"esi_description"`
	Description     *string `json:"custom_description"`
	Format          string  `json:"custom_description_format"`
	Pending         *struct {
		Body   string `json:"body"`
		Format string `json:"body_format"`
	} `json:"pending_submission"`
}
type settingsDescriptionsData struct {
	Character   *settingsBio `json:"character"`
	Corporation *settingsBio `json:"corporation"`
	Alliance    *settingsBio `json:"alliance"`
}

func (bio *settingsBio) text() string {
	if bio.Pending != nil {
		return bio.Pending.Body
	}
	if bio.Description != nil {
		return *bio.Description
	}
	if bio.ESIDescription != nil {
		return *bio.ESIDescription
	}
	return ""
}
func (bio *settingsBio) format() string {
	if bio.Pending != nil {
		return bio.Pending.Format
	}
	if bio.Format != "" {
		return bio.Format
	}
	return "markdown"
}

type settingsColor struct{ Key, Label string }
type settingsColorGroup struct {
	Label  string
	Colors []settingsColor
}
type settingsTabGroup struct {
	Key, Label string
	Options    []struct{ Value, Label string }
}
type settingsPreset struct {
	ID, Name, Description string
	Overrides             map[string]string
}
type settingsDefinitions struct {
	Defaults, Variables map[string]string
	Presets             []settingsPreset
	Colors              []settingsColorGroup
	Tabs                []settingsTabGroup
	CommentTargets      map[int]settingsCommentSpec
}

var settingsData = loadMenu[settingsDefinitions]("data/settings.json")

func settingsPath(id string) string {
	if id == "overview" {
		return "/settings"
	}
	return "/settings/" + id
}

func settingsSelection(id string) (string, string) {
	for _, section := range settingsSections {
		if section.ID == id {
			return section.ID, section.Label
		}
	}
	return "overview", "Overview"
}

// Reuse the account APIs' authoritative authorization, validation, and reads.
// No private account data enters a public/shared page cache.
func (s *site) accountRead(r *http.Request, path string, out any) error {
	_, err := s.accountReadHeaders(r, path, out)
	return err
}

// Fragment callers also need authentication lifecycle headers (expired-cookie
// clearing, for example) from the same authoritative account handler.
func (s *site) accountReadHeaders(r *http.Request, path string, out any) (http.Header, error) {
	if s.options.AccountAPI == nil {
		return nil, fmt.Errorf("account API unavailable")
	}
	request := r.Clone(r.Context())
	address := *r.URL
	endpoint, err := url.ParseRequestURI(path)
	if err != nil {
		return nil, err
	}
	address.Path, address.RawPath, address.RawQuery = endpoint.Path, "", endpoint.RawQuery
	request.URL, request.Method, request.RequestURI, request.Body, request.ContentLength = &address, http.MethodGet, "", nil, 0
	response := httptest.NewRecorder()
	s.options.AccountAPI.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		return response.Header(), accountPageError{response.Code}
	}
	return response.Header(), json.Unmarshal(response.Body.Bytes(), out)
}

type accountPageError struct{ Status int }

func (e accountPageError) Error() string { return fmt.Sprintf("account API returned %d", e.Status) }

func (s *site) settingsIdentity(r *http.Request) (*settingsUser, error) {
	if s.options.AccountAPI == nil {
		return nil, nil
	}
	var data struct{ User *settingsUser }
	if err := s.accountRead(r, "/auth/me", &data); err != nil {
		return nil, err
	}
	return data.User, nil
}

func (s *site) settingsPage(r *http.Request) (Page, error) {
	active, title := settingsSelection(r.PathValue("tab"))
	user, err := s.settingsIdentity(r)
	if err != nil {
		return Page{}, err
	}
	view := settingsView{Active: active, Title: title, User: user, Now: time.Now()}
	if user != nil && settingsComponent(view) == nil {
		return Page{}, ErrNotFound
	}
	s.loadSettingsSection(r, &view)
	return Page{Metadata: Metadata{Title: title + " — Settings", CanonicalPath: settingsPath(active), Stylesheets: []string{"utilities.css", "utility-page.css", "settings.css", "comment-thread.css"}, ModuleScripts: []string{"settings.js"}}, Content: settingsPageContent(view)}, nil
}

func (s *site) settingsFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	active, title := settingsSelection(r.PathValue("tab"))
	if active != r.PathValue("tab") {
		http.NotFound(w, r)
		return
	}
	user, err := s.settingsIdentity(r)
	if err != nil {
		http.Error(w, "Unable to load settings", http.StatusServiceUnavailable)
		return
	}
	if user == nil {
		http.Error(w, "Your session expired. Log in again to continue.", http.StatusUnauthorized)
		return
	}
	view := settingsView{Active: active, Title: title, User: user, Now: time.Now()}
	if settingsComponent(view) == nil {
		http.NotFound(w, r)
		return
	}
	s.loadSettingsSection(r, &view)
	s.renderFragment(w, r, settingsSectionContent(view))
}

func (s *site) loadSettingsSection(r *http.Request, view *settingsView) {
	if view.User == nil {
		return
	}
	switch view.Active {
	case "overview":
		view.OverviewError = s.accountRead(r, "/api/user/overview", &view.Overview) != nil
	case "description":
		view.DescriptionError = s.accountRead(r, "/api/user/manageable-entities", &view.Descriptions) != nil
	case "wallet":
		page, _ := walletQuery(r.URL.Query())
		view.WalletError = s.accountRead(r, "/api/user/wallet?page="+strconv.Itoa(page), &view.Wallet) != nil
	case "esi":
		view.ESI.TokenError = s.accountRead(r, "/auth/token-info", &view.ESI.Token) != nil
		view.ESI.MetricsError = s.accountRead(r, "/api/user/esi", &view.ESI.Metrics) != nil
		view.ESI.LogError = s.accountRead(r, "/api/user/esi-logs", &view.ESI.Logs) != nil
	case "comments":
		view.CommentsError = s.accountRead(r, "/api/user/comments?limit=25", &view.Comments) != nil
	case "domains":
		view.DomainsError = s.accountRead(r, "/api/user/domains", &view.Domains) != nil
	}
}

func settingsJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func settingsUserConfig(user *settingsUser) map[string]any {
	if user == nil || user.Settings == nil {
		return map[string]any{}
	}
	return user.Settings
}

func settingsScopeCount(data settingsOverviewData) int {
	if data.Token != nil {
		return data.Token.ScopeCount
	}
	return 0
}

func settingsHasErrors(data settingsOverviewData) bool {
	value, _ := data.RawStats["errors_24h"].(float64)
	return value > 0
}

func settingsLastFetch(data settingsOverviewData, now time.Time) string {
	if data.Token == nil {
		return "—"
	}
	return settingsSince(data.Token.LastFetched, now)
}

func settingsOptionLabel(group settingsTabGroup, value string) string {
	for _, option := range group.Options {
		if option.Value == value {
			return option.Label
		}
	}
	return value
}

func settingsValue(user *settingsUser, key, subkey string) string {
	if values, ok := user.Settings[key].(map[string]any); ok {
		if value, ok := values[subkey].(string); ok {
			return value
		}
	}
	return "dashboard"
}

func settingsNumber(raw any) string {
	value, ok := raw.(float64)
	if !ok {
		return "—"
	}
	return fullNumber(value)
}
func settingsDate(value *time.Time) string {
	if value == nil {
		return "—"
	}
	return value.UTC().Format("2006-01-02 15:04")
}
func settingsSince(value *time.Time, now time.Time) string {
	if value == nil {
		return "—"
	}
	minutes := int(now.Sub(*value).Minutes())
	if minutes < 1 {
		return "just now"
	}
	if minutes < 60 {
		return fmt.Sprintf("%dm ago", minutes)
	}
	if minutes < 1440 {
		return fmt.Sprintf("%dh ago", minutes/60)
	}
	return fmt.Sprintf("%dd ago", minutes/1440)
}

func settingsTokenStatus(data settingsOverviewData, now time.Time) (string, string) {
	if data.Token == nil {
		return "None", "text-red-400"
	}
	if data.Token.TokenExpiry != nil && data.Token.TokenExpiry.After(now) {
		return "Active", "text-green-400"
	}
	return "Refreshing", "text-yellow-400"
}

func settingsPresetColor(preset settingsPreset, key string) string {
	if value := preset.Overrides[key]; value != "" {
		return value
	}
	return settingsData.Defaults[key]
}

func settingsHex(value string) string {
	if strings.HasPrefix(value, "#") && len(value) > 7 {
		return value[:7]
	}
	return value
}

// Keep all section dispatch in one place; later sections share this shell,
// retained fragment navigation, session handling, and source utility CSS.
func settingsComponent(view settingsView) templ.Component {
	switch view.Active {
	case "overview":
		return settingsOverview(view)
	case "preferences":
		return settingsPreferences(view.User)
	case "appearance":
		return settingsAppearance()
	case "sessions":
		return settingsSessions()
	case "description":
		return settingsDescription(view)
	case "wallet":
		return settingsWallet(view)
	case "esi":
		return settingsESI(view.ESI, view.Now)
	case "comments":
		return settingsMyComments(view)
	case "domains":
		return settingsDomains(view)
	default:
		return nil
	}
}
