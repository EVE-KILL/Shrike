package frontend

import (
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
)

var adminSections = []struct{ ID, Label, Icon string }{
	{"overview", "Overview", "layout-dashboard"}, {"users", "Users", "users"},
	{"domains", "Domains", "globe"}, {"campaigns", "Campaigns", "flag"},
	{"moderation", "Moderation", "shield-check"}, {"comments", "Comments", "message-square"},
	{"esi", "ESI Monitoring", "activity"}, {"river", "River Workers", "list-restart"},
	{"wallet", "Wallet", "wallet-cards"}, {"announcements", "Announcements", "megaphone"},
	{"blog", "Blog", "newspaper"},
}

type adminView struct {
	Active, Title string
	User          *settingsUser
	Overview      map[string]any
	OverviewError bool
	Users         adminUsersData
	UsersQuery    adminUsersQuery
	UsersError    bool
	ESI           adminESIData
	Announcements adminAnnouncementData
	Domains       adminDomainsData
	Campaigns     adminCampaignsData
	Moderation    adminModerationData
	Comments      adminCommentsData
	River         adminRiverData
	Wallet        adminWalletData
	WalletBanner  url.Values
	Blog          adminBlogData
	Now           time.Time
}

type adminUser struct {
	ID              int64      `json:"character_id"`
	Name            string     `json:"character_name"`
	OwnerHash       string     `json:"character_owner_hash"`
	Admin           bool       `json:"is_admin"`
	DiscordID       *string    `json:"discord_user_id"`
	LastLogin       *time.Time `json:"last_login"`
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
	CorporationID   int64      `json:"corporation_id"`
	CorporationName string     `json:"corporation_name"`
	AllianceID      int64      `json:"alliance_id"`
	AllianceName    string     `json:"alliance_name"`
}

type adminUsersData struct {
	Users              []adminUser `json:"users"`
	Total, Page, Pages int
}

type adminUsersQuery struct {
	Search, Sort, Direction string
	Page                    int
}

func adminUserQuery(query url.Values) adminUsersQuery {
	result := adminUsersQuery{Search: query.Get("search"), Sort: "last_login", Direction: "desc", Page: 1}
	if sort := query.Get("sort"); sort == "character_name" || sort == "created_at" {
		result.Sort = sort
	}
	if query.Get("dir") == "asc" {
		result.Direction = "asc"
	}
	if page, err := strconv.ParseInt(query.Get("page"), 10, 32); err == nil && page > 0 {
		result.Page = int(page)
	}
	return result
}

func (query adminUsersQuery) values() url.Values {
	return url.Values{"search": {query.Search}, "sort": {query.Sort}, "dir": {query.Direction}, "page": {strconv.Itoa(query.Page)}, "limit": {"50"}}
}

type adminUserDetailData struct {
	User  *adminUser `json:"user"`
	Token *struct {
		Scopes      []string   `json:"scopes"`
		Expiry      *time.Time `json:"token_expiry"`
		LastFetched *time.Time `json:"last_fetched"`
		CreatedAt   *time.Time `json:"created_at"`
	} `json:"esiToken"`
	Config []struct {
		Key       string
		Value     any
		UpdatedAt *time.Time `json:"updated_at"`
	} `json:"config"`
	Stats map[string]any `json:"esiStats"`
}

func adminPath(section string) string {
	if section == "overview" {
		return "/admin"
	}
	return "/admin/" + section
}
func adminSelection(id string) (string, string) {
	for _, section := range adminSections {
		if section.ID == id {
			return section.ID, section.Label
		}
	}
	return "overview", "Overview"
}

// Private redirects need the same cache protection as the HTML and fragments.
func (s *site) adminPageHandler(loader pageLoader) http.Handler {
	page := s.pageHandler(loader)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		page.ServeHTTP(w, r)
	})
}

func adminMetadata(title string) Metadata {
	if title != "Admin" {
		title += " — Admin"
	}
	return Metadata{Title: title, CommentMedia: true, Stylesheets: []string{"utilities.css", "utility-page.css", "settings.css", "admin.css"}, ModuleScripts: []string{"admin.js"}}
}

func (s *site) adminPage(r *http.Request) (Page, error) {
	active, title := adminSelection(r.PathValue("tab"))
	user, err := s.settingsIdentity(r)
	if err != nil {
		return Page{}, err
	}
	if user != nil && !user.Admin {
		return Page{RedirectPath: "/"}, nil
	}
	view := adminView{Active: active, Title: title, User: user, Now: time.Now(), UsersQuery: adminUserQuery(nil)}
	view.ESI.Entity = adminESIEntity(r.URL.Query())
	if user != nil {
		if adminComponent(view) == nil {
			return Page{}, ErrNotFound
		}
		view.OverviewError = s.accountRead(r, "/api/admin/overview", &view.Overview) != nil
		s.loadAdminSection(r, &view)
	}
	return Page{Metadata: adminMetadata(title), Content: adminPageContent(view)}, nil
}

func (s *site) loadAdminSection(r *http.Request, view *adminView) {
	if view.Active == "blog" {
		view.Blog.Status = "all"
		if strings.HasPrefix(r.URL.Path, "/_fragments/") {
			view.Blog.Status = adminBlogStatus(r.URL.Query())
		}
		query := url.Values{"status": {view.Blog.Status}, "limit": {"100"}}
		view.Blog.Failed = s.accountRead(r, "/api/admin/blog?"+query.Encode(), &view.Blog) != nil
	}
	if view.Active == "wallet" {
		view.Wallet.Query = adminWalletSelection(nil)
		if strings.HasPrefix(r.URL.Path, "/_fragments/") {
			view.Wallet.Query = adminWalletSelection(r.URL.Query())
		}
		view.WalletBanner = r.URL.Query()
		view.Wallet.Failed = s.accountRead(r, "/api/admin/wallet?"+view.Wallet.Query.values().Encode(), &view.Wallet) != nil
	}
	if view.Active == "river" {
		if strings.HasPrefix(r.URL.Path, "/_fragments/") {
			view.River.Query = adminRiverSelection(r.URL.Query())
		}
		s.loadAdminRiver(r, &view.River)
	}
	if view.Active == "moderation" || view.Active == "comments" {
		query := adminReviewSelection(nil)
		if strings.HasPrefix(r.URL.Path, "/_fragments/") {
			query = adminReviewSelection(r.URL.Query())
		}
		if view.Active == "moderation" {
			view.Moderation.Query = query
			view.Moderation.Failed = s.accountRead(r, "/api/admin/moderation/queue?"+query.values(view.Active).Encode(), &view.Moderation) != nil
		} else {
			view.Comments.Query = query
			view.Comments.Failed = s.accountRead(r, "/api/admin/comments/queue?"+query.values(view.Active).Encode(), &view.Comments) != nil
		}
	}
	if view.Active == "campaigns" {
		view.Campaigns.Query = adminCampaignSelection(nil)
		if strings.HasPrefix(r.URL.Path, "/_fragments/") {
			view.Campaigns.Query = adminCampaignSelection(r.URL.Query())
		}
		view.Campaigns.Failed = s.accountRead(r, "/api/admin/campaigns?"+view.Campaigns.Query.values().Encode(), &view.Campaigns) != nil
	}
	if view.Active == "domains" {
		view.Domains.Failed = s.accountRead(r, "/api/admin/domains", &view.Domains) != nil
	}
	if view.Active == "users" {
		view.UsersError = s.accountRead(r, "/api/admin/users?"+view.UsersQuery.values().Encode(), &view.Users) != nil
	}
	if view.Active == "esi" {
		query := url.Values{"limit": {"50"}}
		if view.ESI.Entity != nil {
			query.Set("character_id", strconv.FormatInt(view.ESI.Entity.ID, 10))
		}
		view.ESI.LogError = s.accountRead(r, "/api/admin/esi-logs?"+query.Encode(), &view.ESI.Logs) != nil
	}
	if view.Active == "announcements" {
		view.Announcements.Status = adminAnnouncementStatus(r.URL.Query())
		query := url.Values{"status": {view.Announcements.Status}, "limit": {"100"}}
		view.Announcements.Failed = s.accountRead(r, "/api/admin/announcements?"+query.Encode(), &view.Announcements) != nil
	}
}

func (s *site) adminFragmentIdentity(w http.ResponseWriter, r *http.Request) (*settingsUser, bool) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	user, err := s.settingsIdentity(r)
	if err != nil {
		http.Error(w, "Could not load the admin panel", http.StatusServiceUnavailable)
		return nil, false
	}
	if user == nil {
		http.Error(w, "Your session expired. Log in again to continue.", http.StatusUnauthorized)
		return nil, false
	}
	if !user.Admin {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return nil, false
	}
	return user, true
}

func (s *site) adminFragment(w http.ResponseWriter, r *http.Request) {
	user, ok := s.adminFragmentIdentity(w, r)
	if !ok {
		return
	}
	active, title := adminSelection(r.PathValue("tab"))
	view := adminView{Active: active, Title: title, User: user, Now: time.Now(), UsersQuery: adminUserQuery(r.URL.Query())}
	view.ESI.Entity = adminESIEntity(r.URL.Query())
	if active != r.PathValue("tab") || adminComponent(view) == nil {
		http.NotFound(w, r)
		return
	}
	if active == "overview" || active == "esi" {
		view.OverviewError = s.accountRead(r, "/api/admin/overview", &view.Overview) != nil
	}
	s.loadAdminSection(r, &view)
	s.renderFragment(w, r, adminSectionContent(view))
}

func adminComponent(view adminView) templ.Component {
	switch view.Active {
	case "overview":
		return adminOverview(view)
	case "users":
		return adminUsers(view)
	case "esi":
		return adminESI(view)
	case "announcements":
		return adminAnnouncements(view)
	case "domains":
		return adminDomains(view)
	case "campaigns":
		return adminCampaigns(view.Campaigns)
	case "moderation":
		return adminModeration(view.Moderation)
	case "comments":
		return adminComments(view.Comments)
	case "river":
		return adminRiver(view.River)
	case "wallet":
		return adminWallet(view.Wallet, view.WalletBanner)
	case "blog":
		return adminBlog(view.Blog)
	default:
		return nil
	}
}

func adminValue(data map[string]any, path ...string) any {
	var value any = data
	for _, key := range path {
		values, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value = values[key]
	}
	return value
}

func adminNumber(value any) string {
	if number, ok := value.(float64); ok {
		return fullNumber(number)
	}
	return "—"
}
func adminFloat(value any) float64 { number, _ := value.(float64); return number }
func adminRate(value any) string {
	if number, ok := value.(float64); ok {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	return ""
}
func adminDailyCount(data map[string]any) string {
	return fullNumber(math.Round(adminFloat(adminValue(data, "killmails", "last7d")) / 7))
}
func adminBadge(view adminView, id string) int {
	key := map[string]string{"moderation": "moderation", "domains": "domainAssets", "campaigns": "campaignsPaused"}[id]
	count := adminFloat(adminValue(view.Overview, "attention", key))
	if id == "campaigns" {
		count += adminFloat(adminValue(view.Overview, "attention", "campaignsFailed"))
	}
	return int(count)
}

func (s *site) readAdminUser(r *http.Request) (adminUserDetailData, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id <= 0 {
		return adminUserDetailData{}, accountPageError{http.StatusBadRequest}
	}
	var data adminUserDetailData
	err = s.accountRead(r, "/api/admin/users/"+strconv.FormatInt(id, 10), &data)
	if err == nil && data.User == nil {
		err = ErrNotFound
	}
	return data, err
}

func (s *site) adminUserPage(r *http.Request) (Page, error) {
	user, err := s.settingsIdentity(r)
	if err != nil {
		return Page{}, err
	}
	if user != nil && !user.Admin {
		return Page{RedirectPath: "/"}, nil
	}
	if _, err := strconv.ParseInt(r.PathValue("id"), 10, 32); err != nil || r.PathValue("id") == "0" {
		return Page{Status: http.StatusBadRequest, Metadata: adminMetadata("Admin"), Content: errorPage("Invalid user ID", "This user ID is not valid.")}, nil
	}
	if user == nil {
		return Page{Metadata: adminMetadata("Admin"), Content: adminPageContent(adminView{})}, nil
	}
	data, err := s.readAdminUser(r)
	if err != nil {
		var failure accountPageError
		if errors.As(err, &failure) && (failure.Status == 404 || failure.Status == 400) {
			return Page{Status: failure.Status, Metadata: adminMetadata("Admin"), Content: errorPage("User not found", "This user is not available.")}, nil
		}
		return Page{}, err
	}
	return Page{Metadata: adminMetadata(data.User.Name), Content: adminUserDetail(data, time.Now())}, nil
}

func (s *site) adminUserFragment(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminFragmentIdentity(w, r); !ok {
		return
	}
	data, err := s.readAdminUser(r)
	if err != nil {
		status := http.StatusServiceUnavailable
		var failure accountPageError
		if errors.As(err, &failure) {
			status = failure.Status
		}
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, "Could not load this user", status)
		return
	}
	s.renderFragment(w, r, adminUserDetail(data, time.Now()))
}

func adminDate(value *time.Time) string {
	if value == nil {
		return "—"
	}
	return value.UTC().Format("2006-01-02 15:04")
}
func adminTimeValue(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
func adminDateShort(value *time.Time) string {
	if value == nil {
		return "—"
	}
	return value.UTC().Format("Jan 2")
}
func adminTokenActive(data adminUserDetailData, now time.Time) bool {
	return data.Token != nil && data.Token.Expiry != nil && data.Token.Expiry.After(now)
}
func adminOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func adminStatDate(value any) string {
	raw, _ := value.(string)
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999Z07"} {
		if at, err := time.Parse(layout, raw); err == nil {
			return adminDate(&at)
		}
	}
	return "—"
}

func adminUserESIURL(user *adminUser) string {
	return "/admin/esi?" + url.Values{"character_id": {warID(user.ID)}, "character_name": {user.Name}}.Encode()
}
