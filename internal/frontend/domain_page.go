package frontend

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/api"
)

type domainPageKey struct{}
type domainPageContext struct {
	Config  *api.SiteDomainConfiguration
	Request *http.Request
}

func domainContext(ctx context.Context) *domainPageContext {
	value, _ := ctx.Value(domainPageKey{}).(*domainPageContext)
	return value
}

func pageDomain(ctx context.Context) *api.SiteDomainConfiguration {
	if value := domainContext(ctx); value != nil {
		return value.Config
	}
	return nil
}

func domainCandidate(host string) bool {
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host != "" && host != "localhost" && host != "eve-kill.com" && net.ParseIP(host) == nil
}

func domainSiteName(config *api.SiteDomainConfiguration) string {
	if config != nil && config.SiteName != nil && strings.TrimSpace(*config.SiteName) != "" {
		return *config.SiteName
	}
	return "EVE-KILL"
}

func domainString(value *string) string {
	if value != nil {
		return *value
	}
	return ""
}
func domainShow(value *bool) bool { return value == nil || *value }
func domainThemeJSON(config *api.SiteDomainConfiguration) string {
	if config == nil {
		return ""
	}
	return settingsJSON(config.Theme)
}
func domainBackground(config *api.SiteDomainConfiguration) string {
	if config == nil || len(config.Backgrounds) == 0 {
		return ""
	}
	return config.Backgrounds[0]
}
func domainBanner(config *api.SiteDomainConfiguration) bool {
	return config != nil && (domainString(config.Theme.BannerURL) != "" || config.Theme.TransparentBanner != nil && *config.Theme.TransparentBanner)
}

func domainContentStyle(config *api.SiteDomainConfiguration) string {
	if config == nil || config.Theme.ContentOpacity == nil {
		return ""
	}
	value := *config.Theme.ContentOpacity
	if value != value {
		return ""
	}
	return fmt.Sprintf("--content-alpha-scale:%g", min(100, max(20, value))/100)
}

func (s *site) domainHomePage(r *http.Request, kind string) (Page, error) {
	config := pageDomain(r.Context())
	if config == nil {
		return Page{}, ErrNotFound
	}
	title := ""
	if kind != "" {
		title = killsPageTitle(kind)
	}
	return Page{Metadata: Metadata{Title: title, Description: domainString(config.SiteDescription), CanonicalPath: r.URL.Path,
		Stylesheets: []string{"domain-page.css"}, ModuleScripts: []string{"domain-page.js"}}, Content: domainHome(config, kind)}, nil
}

func (s *site) domainWidgetFragment(w http.ResponseWriter, r *http.Request) {
	if pageDomain(r.Context()) == nil {
		http.NotFound(w, r)
		return
	}
	widget := r.PathValue("widget")
	killType := r.URL.Query().Get("type")
	if killType != "" && !validKillsPageType(killType) {
		http.Error(w, "Invalid type", 400)
		return
	}
	switch widget {
	case "valuable":
		kind := r.URL.Query().Get("kind")
		if kind == "" {
			kind = "kills"
		}
		if kind != "kills" && kind != "ships" && kind != "structures" {
			http.Error(w, "Invalid kind", 400)
			return
		}
		var data struct {
			Entries []api.ValuableKill `json:"entries"`
		}
		endpoint := "/api/custom/stats?" + url.Values{"dataType": {"most_valuable_" + kind}, "days": {"7"}, "limit": {"8"}}.Encode()
		if killType != "" {
			endpoint = "/api/custom/kills/most-valuable?" + url.Values{"type": {killType}, "days": {"7"}, "limit": {"7"}}.Encode()
		}
		if err := s.accountRead(r, endpoint, &data); err != nil {
			s.domainFragmentError(w, err)
			return
		}
		if killType != "" {
			s.renderFragment(w, r, filteredValuable(data.Entries))
		} else {
			s.renderFragment(w, r, mostValuable(data.Entries, kind, false))
		}
	case "campaigns":
		var data campaignDirectoryData
		if err := s.accountRead(r, "/api/campaigns", &data); err != nil {
			s.domainFragmentError(w, err)
			return
		}
		s.renderFragment(w, r, domainCampaigns(data.Campaigns))
	default:
		definition := trackerWidgetDefinitionFor(widget)
		if definition.DataType == "" {
			http.NotFound(w, r)
			return
		}
		var data struct {
			Entries []api.TopEntry `json:"entries"`
		}
		endpoint := "/api/custom/stats?" + url.Values{"dataType": {definition.DataType}, "days": {"7"}, "limit": {"10"}}.Encode()
		if killType != "" {
			endpoint = "/api/custom/kills/top?" + url.Values{"type": {killType}, "dataType": {definition.DataType}, "days": {"7"}, "limit": {"10"}}.Encode()
		}
		if err := s.accountRead(r, endpoint, &data); err != nil {
			s.domainFragmentError(w, err)
			return
		}
		s.renderFragment(w, r, domainTopBox(topBoxView{Title: domainTopTitle(widget, killType), DataType: definition.DataType, Entries: data.Entries, ShowFooter: true}))
	}
}

func (s *site) domainFragmentError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	if failure, ok := err.(accountPageError); ok {
		status = failure.Status
	}
	http.Error(w, "Could not load killboard data", status)
}

func (s *site) domainKilllistFragment(w http.ResponseWriter, r *http.Request) {
	if pageDomain(r.Context()) == nil {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	limit, err := fragmentInt(q.Get("limit"), 50, 10, 100)
	if err != nil {
		http.Error(w, "Invalid limit", 400)
		return
	}
	page, err := fragmentInt(q.Get("page"), 1, 1, 100000)
	if err != nil {
		http.Error(w, "Invalid page", 400)
		return
	}
	kind := q.Get("type")
	if kind == "" {
		kind = "latest"
	}
	if !validKillsPageType(kind) {
		http.Error(w, "Invalid type", 400)
		return
	}
	params := url.Values{"type": {kind}, "limit": {strconv.Itoa(limit)}}
	if after := q.Get("after"); after != "" {
		params.Set("after", after)
	}
	var data api.KilllistPage
	if err := s.accountRead(r, "/api/custom/killlist?"+params.Encode(), &data); err != nil {
		s.domainFragmentError(w, err)
		return
	}
	s.renderFragment(w, r, killList(data, killListOptions{Kind: kind, Page: page, Limit: limit, Live: q.Get("live") == "1"}))
}

func domainNavigation(config *api.SiteDomainConfiguration) []navLink {
	if config == nil || len(config.NavbarLinks) == 0 {
		return navigation
	}
	links := make([]navLink, 0, len(config.NavbarLinks))
	for _, source := range config.NavbarLinks {
		link := navLink{Label: source.Label, Href: source.Href, Icon: domainString(source.Icon), External: source.External != nil && *source.External}
		for _, group := range source.Children {
			output := navGroup{Label: domainString(group.Label)}
			for _, item := range group.Items {
				output.Items = append(output.Items, navLink{Label: item.Label, Href: item.Href, Icon: domainString(item.Icon), External: item.External != nil && *item.External})
			}
			link.Children = append(link.Children, output)
		}
		links = append(links, link)
	}
	return links
}

func domainNavbarHome(config *api.SiteDomainConfiguration) string {
	if config == nil {
		return "Home"
	}
	return domainSiteName(config)
}
func navTarget(link navLink) string {
	if link.External {
		return "_blank"
	}
	return ""
}
func navRel(link navLink) string {
	if link.External {
		return "noopener noreferrer"
	}
	return ""
}

func (s *site) locationKilllist(ctx context.Context, kind string, id int64, limit int, after *int64) (api.KilllistPage, error) {
	domain := domainContext(ctx)
	if domain == nil {
		return s.options.Universe.UniverseLocationKills(ctx, kind, id, limit, after)
	}
	params := url.Values{"limit": {strconv.Itoa(limit)}}
	if after != nil {
		params.Set("after", strconv.FormatInt(*after, 10))
	}
	var result api.KilllistPage
	err := s.accountRead(domain.Request.Clone(ctx), fmt.Sprintf("/api/custom/%s/%d/killlist?%s", kind, id, params.Encode()), &result)
	return result, err
}

func domainTopTitle(widget, kind string) string {
	definition := trackerWidgetDefinitionFor(widget)
	if kind != "" {
		return topKillsTitle(kind, definition.DataType)
	}
	return definition.Title
}
func domainCampaignSides(sides []api.EntityCampaignSide) []string {
	names := make([]string, 0, len(sides))
	for _, side := range sides {
		names = append(names, side.Name)
	}
	return names
}

func (s *site) statisticsSite(r *http.Request) (api.SiteConfigurationResponse, error) {
	if config := pageDomain(r.Context()); config != nil {
		return api.SiteConfigurationResponse{Domain: config, IsDomainHost: true}, nil
	}
	return s.options.Statistics.StatisticsSite(r.Context(), r.Host)
}
