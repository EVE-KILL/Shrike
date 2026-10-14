// Package frontend renders the site's pages inside the Go process.
package frontend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/eve-kill/shrike/internal/api"
)

// Options contains deployment policy, not request-derived site identity.
type Options struct {
	Production      bool
	Homepage        HomepageReader
	Entities        EntityReader
	Universe        UniverseReader
	Wallet          WalletReader
	Legacy          LegacyReader
	Rankings        RankingsReader
	Statistics      StatisticsReader
	Scans           ScanReader
	Coalitions      CoalitionReader
	Wars            WarReader
	FactionWars     FactionWarReader
	BattleGenerator BattleGeneratorReader
	Blog            BlogReader
	Killmail        KillmailReader
	// BlogPreviewAPI is the existing authenticated API handler. The admin-only
	// preview route delegates its authorization and post lookup to it in-process.
	BlogPreviewAPI http.Handler
	CommentsAPI    http.Handler
	SitemapAPI     http.Handler
	CampaignsAPI   http.Handler
	Sessions       SessionReader
	AccountAPI     http.Handler
	ConflictsAPI   http.Handler
	FittingsAPI    http.Handler
	RoamReports    RoamReportReader
	AdvancedSearch AdvancedSearchReader
}

type AdvancedSearchReader interface {
	Labels(context.Context) ([]api.LabelEntry, error)
	AdvancedSearch(context.Context, url.Values) (api.AdvancedSearchResult, error)
}

type RoamReportReader interface {
	RoamReport(context.Context, string) (api.RoamReport, bool, error)
	RoamReportKilllist(context.Context, api.RoamReport, url.Values) (api.KilllistPage, error)
}

type BlogReader interface {
	PublicBlogPosts(context.Context, *api.BlogCursor, string, int) (api.BlogListPage, error)
	PublicBlogPost(context.Context, string) (api.BlogPost, bool, error)
}

type KillmailReader interface {
	KillmailPage(context.Context, int64) (api.KillmailPageData, bool, error)
}

type WalletReader interface {
	PublicWallet(context.Context, int, int) (api.PublicWalletData, error)
}

type LegacyReader interface {
	LegacyArchiveStats(context.Context) (api.LegacyArchiveStats, error)
	LegacyArchiveTop(context.Context, string, int) ([]api.TopEntry, error)
	LegacyArchiveKills(context.Context, url.Values) (api.KilllistPage, error)
	LegacyArchiveKill(context.Context, int64) (api.LegacyArchiveKillData, bool, error)
}

type RankingsReader interface {
	Rankings(context.Context, string, string) ([]api.RankingEntry, error)
}

type StatisticsReader interface {
	StatisticsSite(context.Context, string) (api.SiteConfigurationResponse, error)
	StatisticsBoardsCached(context.Context, api.SiteConfigurationResponse, []api.StatisticsQuery) map[int][]api.StatisticsEntry
	StatisticsBoard(context.Context, api.SiteConfigurationResponse, api.StatisticsQuery) ([]api.StatisticsEntry, error)
}

type ScanReader interface {
	SavedScan(context.Context, string, string) (json.RawMessage, bool, error)
}

type CoalitionReader interface {
	CoalitionDirectory(context.Context) (api.CoalitionDirectoryData, error)
	CoalitionDirectoryOverview(context.Context) (api.CoalitionDirectoryData, error)
	CoalitionDetail(context.Context, string) (api.CoalitionDetailData, bool, error)
}

type WarReader interface {
	WarDirectory(context.Context, url.Values, string) (api.WarDirectoryData, error)
	WarOverview(context.Context) (api.WarOverviewData, error)
	FactionWars(context.Context) (api.FactionWarsData, error)
	WarDetail(context.Context, int64) (api.WarDetailData, bool, error)
	WarKilllist(context.Context, int64, url.Values) (api.KilllistPage, error)
	WarStats(context.Context, int64) (api.WarStatsData, error)
	WarMembers(context.Context, int64, url.Values) (api.WarMembersData, error)
	WarIntel(context.Context, int64) (api.WarIntelData, error)
}

type FactionWarReader interface {
	FactionWarDashboard(context.Context, string) (api.FactionWarData, error)
	FactionWarOverview(context.Context, string, string) (api.FactionWarOverview, error)
	FactionWarSystems(context.Context, string) (api.FactionWarSystems, error)
	FactionWarMembers(context.Context, string, url.Values) (api.WarMembersData, error)
	FactionWarIntel(context.Context, string) (api.WarIntelData, error)
	FactionWarKilllist(context.Context, string, string, int, int, *int64) (api.KilllistPage, error)
}

type BattleGeneratorReader interface {
	BattleGeneratorPreview(context.Context, json.RawMessage) (json.RawMessage, error)
}

// HomepageReader keeps homepage and /kills data access outside templ components.
type HomepageReader interface {
	Top(context.Context, string, int, int) ([]api.TopEntry, error)
	MostValuable(context.Context, string, int, int) ([]api.ValuableKill, error)
	LatestKills(context.Context, int, int) (api.KilllistPage, error)
	LatestKillsFresh(context.Context, int, int) (api.KilllistPage, error)
	Killlist(context.Context, string, int, int) (api.KilllistPage, error)
	KilllistFresh(context.Context, string, int, int) (api.KilllistPage, error)
	TopKills(context.Context, string, string, int, int) ([]api.TopEntry, error)
	TopKillsCached(context.Context, string, string, int, int) ([]api.TopEntry, bool)
	MostValuableKills(context.Context, string, int, int) ([]api.ValuableKill, error)
	FilteredKills(context.Context, string, *int64, int) (api.KilllistPage, error)
	Labels(context.Context) ([]api.LabelEntry, error)
	SavedScan(context.Context, string, string) (json.RawMessage, bool, error)
}

// EntityReader shares the public entity-page loader with the API while
// keeping entity routes independent from the homepage's widget reader.
type EntityReader interface {
	EntityDetail(context.Context, string, int64) (map[string]any, error)
	EntityKilllist(context.Context, string, int64, string, int, int, *int64, int64) (api.KilllistPage, error)
	EntityTopLists(context.Context, string, int64, int) (api.EntityTopLists, error)
	EntityDashboard(context.Context, string, int64, int) (api.EntityDashboard, error)
	EntityTop(context.Context, string, int64, string, string) (api.EntityTopData, error)
	EntityAchievements(context.Context, int64) ([]api.EntityAchievement, error)
	EntityMembers(context.Context, string, int64, int, string, int, int64) (api.EntityMembersData, error)
	EntityCorporations(context.Context, int64, string, string) (api.EntityCorporationsData, error)
	EntityBattles(context.Context, string, int64, int) (api.EntityBattlesData, error)
	EntityWars(context.Context, string, int64, int, string) (api.EntityWarsData, error)
	EntityCampaigns(context.Context, string, int64, int) (api.EntityCampaignsData, error)
	EntityIntel(context.Context, string, int64, int) (api.EntityIntelData, error)
	EntityFactionOverview(context.Context, int64) (api.EntityFactionOverview, error)
	EntityFactionMilitiaCorps(context.Context, int64) ([]api.EntityFactionMilitiaCorp, error)
}

type UniverseReader interface {
	UniverseSystem(context.Context, int64) (api.UniverseSystemData, bool, error)
	UniverseRegion(context.Context, int64) (api.UniverseRegionData, bool, error)
	UniverseConstellation(context.Context, int64) (api.UniverseConstellationData, bool, error)
	UniverseLocationKills(context.Context, string, int64, int, *int64) (api.KilllistPage, error)
	UniverseLocationValuable(context.Context, string, int64, string) ([]api.ValuableKill, error)
	UniverseGroup(context.Context, int64) (api.UniverseGroupData, bool, error)
	UniverseGroupKills(context.Context, int64, int, *int64) (api.KilllistPage, error)
	UniverseItem(context.Context, int64) (api.UniverseItemData, bool, error)
	UniverseItemKills(context.Context, int64, bool, int, *int64) (api.KilllistPage, error)
	UniverseMarketOrders(context.Context, int64, int32, string) (api.UniverseMarketOrders, error)
	UniverseMarketHistory(context.Context, int64, int32, int) (api.UniverseMarketHistory, error)
	UniverseFitFamilies(context.Context, int64, url.Values) (api.UniverseFitFamilies, error)
	UniverseFitMeta(context.Context, int64, url.Values) (api.UniverseFitMeta, error)
	UniverseFitDistributions(context.Context, int64, url.Values) (api.UniverseFitDistributions, error)
	UniverseMarketTree(context.Context) ([]api.UniverseMarketTreeNode, error)
	UniverseMarketGroupItems(context.Context, int64) ([]api.UniverseMarketGroupItem, error)
}

type site struct {
	options Options
	pages   []staticPage
}

// Static routes supply the base sitemap. Validated kill category routes add
// their own entries; later entity pages can add paginated sitemap sources.
type staticPage struct {
	Path     string
	Metadata Metadata
	Content  templ.Component
}

// New builds the page handler. Ingress owns API, auth, images and WebSockets.
func New(options Options) http.Handler {
	homeMeta := Metadata{Title: "Home", CanonicalPath: "/"}
	if options.Homepage != nil {
		homeMeta.Indexable = true
		homeMeta.StructuredData = []any{
			map[string]any{
				"@context": "https://schema.org", "@type": "WebSite", "name": "EVE-KILL",
				"url": "https://eve-kill.com",
			},
			map[string]any{
				"@context": "https://schema.org", "@type": "CollectionPage",
				"name": "EVE-KILL — EVE Online Killboard",
			},
		}
	}
	s := &site{options: options, pages: []staticPage{
		{Path: "/", Metadata: homeMeta, Content: home()},
		{Path: "/about", Metadata: Metadata{
			Title: "About — Community Killboard Since 2007", CanonicalPath: "/about", Indexable: true,
			Description:    "Learn about EVE-KILL — a community-driven killboard for EVE Online founded in 2007. Our mission, history, and how to get in touch.",
			StructuredData: map[string]any{"@context": "https://schema.org", "@type": "AboutPage", "name": "About EVE-KILL", "description": "EVE-KILL is a community-driven killboard for EVE Online, founded in 2007."},
		}, Content: aboutPage()},
		{Path: "/donate", Metadata: Metadata{
			Title: "Support", CanonicalPath: "/donate", Indexable: true,
			Description: "Support EVE-KILL — help keep the community killboard running. Donations go directly to server costs and ongoing development.",
		}, Content: donatePage()},
		{Path: "/faq", Metadata: Metadata{
			Title: "Frequently Asked Questions", CanonicalPath: "/faq", Indexable: true,
			Description:    "Frequently asked questions about EVE-KILL — killmails, combat points, rankings, item pricing, accounts, and the API.",
			StructuredData: faqStructuredData(),
		}, Content: faqPage()},
		{Path: "/status", Metadata: Metadata{
			Title: "Status", Description: "EVE-KILL system status — service health, queue processing, and backend connectivity.",
			Stylesheets: []string{"status.css"}, ModuleScripts: []string{"status.js"},
		}, Content: statusPage()},
		{Path: "/tools", Metadata: Metadata{
			Title: "Tools", CanonicalPath: "/tools", Indexable: true,
			Description: "EVE Online utilities on EVE-KILL — Local Scan analyzer, D-Scan parser, Battle Generator, Campaign Creator, Advanced Search, Ship Fittings, Legacy Kill archive.",
		}, Content: toolsPage()},
		{Path: "/tools/dscan", Metadata: scanMetadata("dscan", false, ""), Content: scanInputPage("dscan")},
		{Path: "/tools/localscan", Metadata: scanMetadata("localscan", false, ""), Content: scanInputPage("localscan")},
		{Path: "/post", Metadata: Metadata{
			Title: "Post Killmail", CanonicalPath: "/post", Indexable: true,
			Description: "Manually post EVE Online killmails to EVE-KILL by pasting ESI killmail URLs.",
			Stylesheets: []string{"post.css"}, ModuleScripts: []string{"post.js"},
		}, Content: postKillmailPage()},
		{Path: "/docs", Metadata: Metadata{
			Title: "API Documentation", CanonicalPath: "/docs", Indexable: true,
			Description: "Browse and test the EVE-KILL API. Self-describing, JSON, CORS-open. Killmails, characters, corporations, alliances, ships, battles, wars, leaderboards, search, and the full SDE.",
		}, Content: docsPage()},
		{Path: "/mcp", Metadata: Metadata{
			Title: "MCP Server", CanonicalPath: "/mcp", Indexable: true,
			Description: "EVE-KILL MCP server — AI-friendly tools for searching, analyzing, and narrating EVE Online killmail data over the Model Context Protocol.",
		}, Content: mcpPage(mcpCatalogView())},
		{Path: "/fits/create", Metadata: Metadata{
			Title: "Create a Fit", CanonicalPath: "/fits/create",
			Description: "Build EVE Online ship fittings in your browser with dogma calculations and shareable links.",
			Stylesheets: []string{"fit-editor.css", "fit-wheel.css"}, ModuleScripts: []string{"fit-editor.js"}, NeedsWASM: true,
		}, Content: fitEditorPage()},
	}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /_fragments/account", s.accountFragment)
	mux.HandleFunc("GET /assets/", serveAsset)
	mux.HandleFunc("GET /robots.txt", s.robots)
	mux.HandleFunc("GET /sitemap.xml", s.sitemap)
	mux.HandleFunc("GET /sitemap_index.xml", s.sitemapIndex)
	if options.SitemapAPI != nil {
		mux.HandleFunc("GET /_sitemap-data/{kind}", s.sitemapData)
		mux.HandleFunc("GET /__sitemap__/{file}", s.sitemapCategory)
	}
	mux.HandleFunc("GET /feed.xml", s.feed)
	mux.HandleFunc("GET /icon.png", func(w http.ResponseWriter, r *http.Request) {
		assetRequest := r.Clone(r.Context())
		assetRequest.URL.Path = "/assets/icon.png"
		serveAsset(w, assetRequest)
	})
	mux.HandleFunc("GET /mcp.txt", s.mcpGuide)
	mux.HandleFunc("GET /llms.txt", s.mcpGuide)
	mux.Handle("GET /map", s.pageHandler(s.mapPage))
	mux.Handle("GET /graph", s.pageHandler(s.graphPage))
	mux.Handle("GET /settings", s.pageHandler(s.settingsPage))
	mux.Handle("GET /admin", s.adminPageHandler(s.adminPage))
	mux.Handle("GET /admin/{tab}", s.adminPageHandler(s.adminPage))
	mux.Handle("GET /admin/users/{id}", s.adminPageHandler(s.adminUserPage))
	mux.Handle("GET /admin/domains/{id}", s.adminPageHandler(s.adminDomainPage))
	mux.HandleFunc("GET /_fragments/admin/{tab}", s.adminFragment)
	mux.HandleFunc("GET /_fragments/admin/river/{part}", s.adminRiverFragment)
	mux.HandleFunc("GET /_fragments/admin/users/{id}", s.adminUserFragment)
	mux.HandleFunc("GET /_fragments/admin/domains/{id}", s.adminDomainFragment)
	mux.Handle("GET /trackers", s.pageHandler(s.trackersPage))
	mux.Handle("GET /tracker", s.pageHandler(s.trackerDashboardPage))
	mux.HandleFunc("GET /_fragments/tracker", s.trackerDashboardFragment)
	mux.HandleFunc("GET /_fragments/tracker/stats/{widget}", s.trackerDashboardStatsFragment)
	mux.HandleFunc("GET /_fragments/tracker/valuable", s.trackerDashboardValuableFragment)
	mux.HandleFunc("GET /_fragments/tracker/killmails", s.trackerDashboardKilllistFragment)
	mux.HandleFunc("GET /_fragments/trackers/{id}/killmails", s.trackerKilllistFragment)
	mux.Handle("GET /settings/{tab}", s.pageHandler(s.settingsPage))
	mux.HandleFunc("GET /_fragments/settings/{tab}", s.settingsFragment)
	mux.HandleFunc("GET /_fragments/settings-comments", s.settingsCommentsFragment)
	mux.Handle("GET /tools/roam-report", s.pageHandler(s.roamCreatePage))
	if options.AdvancedSearch != nil {
		mux.Handle("GET /advancedsearch", s.pageHandler(s.advancedSearchPage))
		mux.HandleFunc("GET /_fragments/advancedsearch", s.advancedSearchFragment)
	}
	if options.RoamReports != nil {
		mux.Handle("GET /tools/roam-report/{id}", s.pageHandler(s.roamDetailPage))
		mux.Handle("GET /tools/roam-report/{id}/{tab}", s.pageHandler(s.roamDetailPage))
		mux.HandleFunc("GET /_fragments/roam/{id}/{tab}", s.roamFragment)
	}
	mux.HandleFunc("GET /_fragments/mcp/output/{name}", s.mcpOutputFragment)
	mux.HandleFunc("GET /_fragments/domain/widgets/{widget}", s.domainWidgetFragment)
	mux.HandleFunc("GET /_fragments/domain/killlist", s.domainKilllistFragment)
	if options.Wallet != nil {
		mux.Handle("GET /wallet", s.pageHandler(s.walletPage))
		mux.HandleFunc("GET /_fragments/wallet", s.walletFragment)
	}
	if options.Legacy != nil {
		mux.Handle("GET /legacy", s.pageHandler(s.legacyPage))
		mux.Handle("GET /legacy/kill/{id}", s.pageHandler(s.loadLegacyKillPage))
		mux.HandleFunc("GET /_fragments/legacy/kills", s.legacyKillsFragment)
		mux.HandleFunc("GET /_fragments/legacy/top", s.legacyTopFragment)
	}
	if options.Killmail != nil {
		page := s.pageHandler(s.loadKillmailPage)
		mux.HandleFunc("GET /kill/{id}", func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
			if err == nil && id > 0 && id < 7_500_000 {
				http.Redirect(w, r, "/legacy/kill/"+strconv.FormatInt(id, 10), http.StatusMovedPermanently)
				return
			}
			page.ServeHTTP(w, r)
		})
	}
	if options.Rankings != nil {
		mux.Handle("GET /rankings", s.pageHandler(s.rankingsPage))
		mux.HandleFunc("GET /_fragments/rankings", s.rankingsFragment)
	}
	if options.Statistics != nil {
		mux.Handle("GET /stats", s.pageHandler(s.statisticsPage))
		mux.HandleFunc("GET /_fragments/stats/{card}", s.statisticsCardFragment)
	}
	if options.Coalitions != nil {
		mux.Handle("GET /coalitions", s.pageHandler(s.coalitionDirectoryPage))
		mux.Handle("GET /coalitions/{slug}", s.pageHandler(s.coalitionDetailPage))
		mux.HandleFunc("GET /_fragments/coalitions/activity", s.coalitionDirectoryFragment)
		mux.HandleFunc("GET /_fragments/coalitions/{slug}", s.coalitionDetailFragment)
	}
	if options.Wars != nil {
		mux.Handle("GET /wars", s.pageHandler(s.warsPage))
		mux.HandleFunc("GET /_fragments/wars/faction", s.warsFactionFragment)
		mux.Handle("GET /war/{id}", s.pageHandler(s.loadWarDetailPage))
		mux.HandleFunc("GET /_fragments/war/{id}/kills", s.warKillsFragment)
		mux.HandleFunc("GET /_fragments/war/{id}/stats", s.warStatsFragment)
		mux.HandleFunc("GET /_fragments/war/{id}/members", s.warMembersFragment)
		mux.HandleFunc("GET /_fragments/war/{id}/intel", s.warIntelFragment)
	}
	if options.FactionWars != nil {
		mux.Handle("GET /faction-war/{matchup}", s.pageHandler(s.factionWarPage))
		mux.HandleFunc("GET /_fragments/faction-war/{matchup}/{part}", s.factionWarFragment)
	}
	if options.ConflictsAPI != nil {
		mux.Handle("GET /battles", s.pageHandler(s.battlesPage))
		mux.HandleFunc("GET /_fragments/battles", s.battlesFragment)
		mux.HandleFunc("GET /battle/{id}", s.battlePage)
		mux.HandleFunc("GET /battle/{id}/{tab}", s.battlePage)
		mux.HandleFunc("GET /_fragments/battle/{id}/killfeed", s.battleKillfeedFragment)
		mux.HandleFunc("GET /_fragments/battle/{id}/{tab}", s.battleTabFragment)
		mux.Handle("GET /map/region/{id}", s.pageHandler(s.mapRegionPage))
	}
	if options.BattleGenerator != nil {
		mux.Handle("GET /battlegenerator", s.pageHandler(s.battleGeneratorPage))
		mux.HandleFunc("POST /_fragments/battlegenerator/preview", s.battleGeneratorPreviewFragment)
	}
	if options.CampaignsAPI != nil {
		mux.Handle("GET /campaigncreator", s.pageHandler(s.campaignCreatorPage))
		mux.Handle("GET /campaigns", s.pageHandler(s.campaignsPage))
		mux.HandleFunc("GET /_fragments/campaigns", s.campaignsFragment)
		mux.Handle("GET /campaign/{id}", s.pageHandler(s.loadCampaignDetailPage))
		mux.Handle("GET /campaign/{id}/{tab}", s.pageHandler(s.loadCampaignDetailPage))
		mux.HandleFunc("GET /_fragments/campaign/{id}/killfeed", s.campaignKillfeedFragment)
		mux.HandleFunc("GET /_fragments/campaign/{id}/{tab}", s.campaignTabFragment)
		mux.HandleFunc("GET /_fragments/campaign/{id}/prize", s.campaignPrizeFragment)
		mux.HandleFunc("GET /_fragments/campaign/{id}/edit", s.campaignEditFragment)
	}
	mux.Handle("GET /search", s.pageHandler(func(r *http.Request) (Page, error) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		queryRunes := []rune(query)
		if len(queryRunes) > 100 {
			query = string(queryRunes[:100])
		}
		title := "Search"
		if query != "" {
			title = "Search: " + query
		}
		return Page{Metadata: Metadata{Title: title, CanonicalPath: "/search", Indexable: true,
			Description:    "Search EVE-KILL — find pilots, corporations, alliances, ships, systems, and regions across New Eden.",
			StructuredData: map[string]any{"@context": "https://schema.org", "@type": "SearchResultsPage", "name": "Search — EVE-KILL"},
		}, Content: searchPage(query)}, nil
	}))
	s.registerLegacyRedirects(mux)
	if options.FittingsAPI != nil {
		mux.Handle("GET /fits", s.pageHandler(s.fitsCatalogPage))
		mux.Handle("GET /fits/search", s.pageHandler(s.fitsSearchPage))
		mux.HandleFunc("GET /_fragments/fits/search/{part}", s.fitsSearchFragment)
		mux.Handle("GET /fit/{id}", s.pageHandler(s.savedFitPage))
		mux.HandleFunc("GET /_fragments/fits/catalog/{part}", s.fitsCatalogFragment)
	}
	if options.Entities != nil {
		s.registerEntityPages(mux)
	}
	if options.Universe != nil {
		s.registerUniversePages(mux)
		mux.HandleFunc("GET /_fragments/fits/variants/{id}", s.fitVariantsFragment)
	}
	if options.Blog != nil {
		mux.Handle("GET /blog", s.pageHandler(s.blogListingPage))
		mux.Handle("GET /blog/{slug}", s.pageHandler(s.blogPostPage))
		mux.HandleFunc("GET /_fragments/blog", s.blogListFragment)
	}
	if options.BlogPreviewAPI != nil {
		mux.Handle("GET /blog/preview/{slug}", s.pageHandler(s.blogPreviewPage))
	}
	if options.CommentsAPI != nil {
		mux.Handle("GET /comments", s.pageHandler(s.commentsPage))
		mux.HandleFunc("GET /_fragments/comments", s.commentsFragment)
	}
	mux.HandleFunc("POST /_fragments/scans/{kind}", s.scanResultFragment)
	for _, kind := range []string{"dscan", "localscan"} {
		kind := kind
		mux.Handle("GET /tools/"+kind+"/{hash}", s.pageHandler(func(r *http.Request) (Page, error) {
			return s.savedScanPage(r.Context(), kind, r.PathValue("hash"))
		}))
	}
	if !options.Production {
		mux.HandleFunc("GET /__dev/reload", s.liveReload)
		mux.HandleFunc("GET /__dev/version", s.liveReloadVersion)
	}
	if options.Homepage != nil {
		mux.Handle("GET /categories", s.pageHandler(func(r *http.Request) (Page, error) {
			view, err := loadCategories(r.Context(), options.Homepage)
			if err != nil {
				return Page{}, err
			}
			return Page{Metadata: Metadata{
				Title: "Killmail Categories", CanonicalPath: "/categories", Indexable: true,
				Description: "Browse the categories EVE-KILL applies to EVE Online killmails, with historical counts and direct search links.",
			}, Content: categoriesPage(view), SharedCacheTTL: 300 * time.Second}, nil
		}))
		mux.HandleFunc("GET /_fragments/most-valuable", s.mostValuableFragment)
		mux.HandleFunc("GET /_fragments/killlist", s.killlistFragment)
		mux.HandleFunc("GET /_fragments/kills/top", s.killsTopFragment)
		mux.Handle("GET /kills/{type}", s.pageHandler(func(r *http.Request) (Page, error) {
			kind := r.PathValue("type")
			if !validKillsPageType(kind) {
				return Page{}, ErrNotFound
			}
			if pageDomain(r.Context()) != nil {
				return s.domainHomePage(r, kind)
			}
			view, err := loadKillsPage(r.Context(), options.Homepage, kind)
			if err != nil {
				return Page{}, err
			}
			return Page{Metadata: killsPageMetadata(kind), Content: killsPage(view), SharedCacheTTL: 30 * time.Second}, nil
		}))
	}
	for _, route := range s.pages {
		route := route
		pattern := "GET " + route.Path
		if route.Path == "/" {
			pattern += "{$}"
		}
		mux.Handle(pattern, s.pageHandler(func(r *http.Request) (Page, error) {
			if route.Path == "/" && options.Homepage != nil {
				if pageDomain(r.Context()) != nil {
					return s.domainHomePage(r, "")
				}
				view, err := loadHomepage(r.Context(), options.Homepage)
				if err != nil {
					return Page{}, err
				}
				return Page{Metadata: route.Metadata, Content: homepage(view), SharedCacheTTL: 30 * time.Second}, nil
			}
			return Page{Metadata: route.Metadata, Content: route.Content}, nil
		}))
	}
	mux.Handle("GET /", s.pageHandler(func(_ *http.Request) (Page, error) { return Page{}, ErrNotFound }))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		if options.Production && options.Homepage != nil && options.Statistics == nil && domainCandidate(r.Host) && (r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/kills/")) {
			s.pageHandler(func(*http.Request) (Page, error) {
				return Page{Status: 404, Metadata: Metadata{Title: "No Killboard Found"}, Content: unknownKillboard()}, nil
			}).ServeHTTP(w, r)
			return
		}
		if options.Statistics != nil && domainCandidate(r.Host) && !strings.HasPrefix(r.URL.Path, "/assets/") && !strings.HasPrefix(r.URL.Path, "/__dev/") {
			configuration, err := options.Statistics.StatisticsSite(r.Context(), r.Host)
			if err != nil {
				s.pageHandler(func(*http.Request) (Page, error) { return Page{}, err }).ServeHTTP(w, r)
				return
			}
			if configuration.IsDomainHost {
				if configuration.Domain == nil {
					s.pageHandler(func(*http.Request) (Page, error) {
						return Page{Status: 404, Metadata: Metadata{Title: "No Killboard Found"}, Content: unknownKillboard()}, nil
					}).ServeHTTP(w, r)
					return
				}
				r = r.WithContext(context.WithValue(r.Context(), domainPageKey{}, &domainPageContext{Config: configuration.Domain, Request: r}))
			}
		}
		mux.ServeHTTP(w, r)
	})
}
