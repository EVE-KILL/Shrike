package api

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	domainBodyLimit            = 256 << 10
	domainMaximumPerUser       = 3
	domainMaximumEntities      = 5
	domainMaximumNavbarLinks   = 10
	domainMaximumWidgets       = 20
	domainMaximumCampaigns     = 50
	domainMaximumSiteName      = 200
	domainMaximumDescription   = 2000
	domainCampaignAutomatic    = 0
	domainCampaignAllowlist    = 1
	domainCampaignSearchLength = 100
)

var (
	domainSubdomainPattern = regexp.MustCompile(
		`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`,
	)
	domainCampaignIDPattern  = regexp.MustCompile(`^[0-9A-Za-z]+$`)
	domainReservedSubdomains = map[string]struct{}{
		"www": {}, "api": {}, "mail": {}, "admin": {}, "dev": {},
		"staging": {}, "test": {}, "beta": {}, "alpha": {}, "app": {},
		"cdn": {}, "static": {}, "assets": {}, "img": {}, "images": {},
		"ws": {}, "wss": {}, "ftp": {}, "ssh": {}, "git": {}, "svn": {},
		"docs": {}, "wiki": {}, "blog": {}, "forum": {}, "support": {},
		"help": {}, "status": {}, "monitor": {}, "grafana": {},
		"prometheus": {}, "kibana": {}, "elastic": {}, "redis": {},
		"postgres": {}, "mysql": {}, "db": {}, "database": {},
		"meilisearch": {}, "relay": {}, "proxy": {}, "vpn": {}, "ns1": {},
		"ns2": {}, "mx": {}, "smtp": {}, "imap": {}, "pop": {}, "mcp": {},
	}
	domainWidgetTypes = map[string]struct{}{
		"mostValuable": {}, "killList": {}, "topCharacters": {},
		"topCorporations": {}, "topAlliances": {}, "topShips": {},
		"topSystems": {}, "topRegions": {}, "entityInfo": {},
		"textBlock": {}, "campaigns": {},
	}
	domainColumnRatios = map[string]struct{}{
		"250px_1fr": {}, "300px_1fr": {}, "1fr_1fr": {},
		"1fr_2fr": {}, "1fr_3fr": {},
	}
)

type domainService struct {
	auth       *authService
	db         Database
	mutations  MutationDatabase
	storeErr   error
	assets     DomainAssetStorage
	dispatcher domainAssetEventDispatcher
	now        func() time.Time
}

type domainEntity struct {
	Type string `json:"type"`
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type domainCreateInput struct {
	Subdomain       string
	Entities        []domainEntity
	Theme           map[string]any
	NavbarLinks     []any
	Widgets         map[string]any
	WidgetsPresent  bool
	SiteName        *string
	SiteDescription *string
}

type domainUpdateInput struct {
	EntitiesPresent        bool
	Entities               []domainEntity
	ThemePresent           bool
	Theme                  map[string]any
	NavbarPresent          bool
	NavbarLinks            []any
	WidgetsPresent         bool
	Widgets                map[string]any
	SiteNamePresent        bool
	SiteName               *string
	SiteDescriptionPresent bool
	SiteDescription        *string
	ActivePresent          bool
	Active                 bool
	CampaignPolicyPresent  bool
	CampaignPolicy         int16
	CampaignIDsPresent     bool
	CampaignIDs            []string
	PublicCampaignIDs      []string
}

func newDomainService(opts Options) *domainService {
	service := &domainService{
		auth:   newAuthService(opts),
		db:     opts.DB,
		assets: opts.DomainAssets,
		now:    time.Now,
	}
	service.mutations, service.storeErr = mutationDatabase(opts)
	if pool, ok := opts.DB.(*pgxpool.Pool); ok && pool != nil {
		if client, err := queue.New(queue.Options{Pool: pool}); err == nil {
			service.dispatcher = &riverDomainAssetEventDispatcher{
				client: client,
			}
		}
	}
	return service
}

func (s *domainService) requireStore() error {
	if s.storeErr != nil {
		return s.storeErr
	}
	if s.db == nil || s.mutations == nil {
		return apiError(
			http.StatusServiceUnavailable,
			"Domain storage is not configured",
		)
	}
	return nil
}

func (s *domainService) requireAccount(
	ctx context.Context,
	req *legacyRequest,
	mutation bool,
) (*Principal, error) {
	setAccountNoStore(req.Huma)
	if mutation {
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return nil, err
		}
	}
	if err := s.requireStore(); err != nil {
		return nil, err
	}
	return s.auth.requirePrincipal(ctx, req)
}

func (s *domainService) requireAdmin(
	ctx context.Context,
	req *legacyRequest,
	mutation bool,
) (*Principal, error) {
	principal, err := s.requireAccount(ctx, req, mutation)
	if err != nil {
		return nil, err
	}
	if !principal.IsAdmin {
		return nil, apiError(
			http.StatusForbidden,
			"Administrator access required",
		)
	}
	return principal, nil
}

// registerDomainRoutes installs domain ownership, administration, campaign
// selection, and image delivery into the shared API catalogue. The /me paths
// are canonical; /user paths remain while the copied Nuxt frontend migrates.
// documentDomainBody attaches the right request schema for a domain route.
// The routes are registered from tables, so the operation ID is what
// identifies which body a route takes.
func documentDomainBody(
	a huma.API,
	id string,
	op huma.Operation,
) huma.Operation {
	switch id {
	case "domain-create", "domain-create-compat":
		return documentJSONBody[domainCreateDocument](a, op)
	case "domain-update", "domain-update-patch-compat",
		"domain-update-put-compat":
		return documentJSONBody[domainUpdateDocument](a, op)
	case "domain-assets-delete-type-compat":
		return documentJSONBody[domainAssetTypeBody](a, op)
	case "admin-domain-asset-review":
		return documentJSONBody[domainAssetReviewBody](a, op)
	}
	return op
}

func registerDomainRoutes(a huma.API, opts Options) {
	registerDomainServiceRoutes(a, newDomainService(opts))
}

func registerDomainServiceRoutes(
	a huma.API,
	service *domainService,
) {
	if a.OpenAPI().Components.SecuritySchemes == nil {
		a.OpenAPI().Components.SecuritySchemes =
			make(map[string]*huma.SecurityScheme)
	}
	if a.OpenAPI().Components.SecuritySchemes["eveSession"] == nil {
		a.OpenAPI().Components.SecuritySchemes["eveSession"] =
			&huma.SecurityScheme{
				Type: "apiKey", In: "cookie", Name: authSessionCookie,
				Description: "EVE-KILL browser session for account and admin operations.",
			}
	}
	required := []map[string][]string{{"eveSession": {}}}

	accountRoutes := []struct {
		id, method, path, summary string
		handler                   legacyHandler
	}{
		{
			"domains-mine", http.MethodGet, "/me/domains",
			"Domains owned by the current account", service.listOwnedHandler(),
		},
		{
			"domain-create", http.MethodPost, "/me/domains",
			"Create a custom domain", service.createHandler(),
		},
		{
			"domain-subdomain-check", http.MethodGet,
			"/me/domains/check-subdomain", "Check subdomain availability",
			service.checkSubdomainHandler(),
		},
		{
			"domain-update", http.MethodPatch, "/me/domains/{id}",
			"Update a custom domain", service.updateHandler(),
		},
		{
			"domain-delete", http.MethodDelete, "/me/domains/{id}",
			"Delete a custom domain", service.deleteHandler(),
		},
		{
			"domain-campaign-search", http.MethodGet,
			"/me/domains/{id}/campaigns",
			"Find campaigns eligible for a custom domain",
			service.campaignSearchHandler(),
		},
		{
			"domain-asset-upload", http.MethodPost,
			"/me/domains/{id}/assets", "Upload a domain image",
			service.uploadHandler(),
		},
		{
			"domain-asset-delete", http.MethodDelete,
			"/me/domains/{id}/assets/{assetId}", "Delete a domain image",
			service.deleteAssetHandler(),
		},
		{
			"domain-assets-delete-type", http.MethodDelete,
			"/me/domains/{id}/assets", "Delete a banner or logo",
			service.deleteAssetTypeHandler(false),
		},
		{
			"domains-mine-compat", http.MethodGet, "/user/domains",
			"Domains owned by the current account", service.listOwnedHandler(),
		},
		{
			"domain-create-compat", http.MethodPost, "/user/domains",
			"Create a custom domain", service.createHandler(),
		},
		{
			"domain-subdomain-check-compat", http.MethodGet,
			"/user/domains/check-subdomain", "Check subdomain availability",
			service.checkSubdomainHandler(),
		},
		{
			"domain-update-patch-compat", http.MethodPatch,
			"/user/domains/{id}", "Update a custom domain",
			service.updateHandler(),
		},
		{
			"domain-update-put-compat", http.MethodPut, "/user/domains/{id}",
			"Update a custom domain", service.updateHandler(),
		},
		{
			"domain-delete-compat", http.MethodDelete, "/user/domains/{id}",
			"Delete a custom domain", service.deleteHandler(),
		},
		{
			"domain-campaign-search-compat", http.MethodGet,
			"/user/domains/{id}/campaigns/search",
			"Find campaigns eligible for a custom domain",
			service.campaignSearchHandler(),
		},
		{
			"domain-asset-upload-compat", http.MethodPost,
			"/user/domains/{id}/upload", "Upload a domain image",
			service.uploadHandler(),
		},
		{
			"domain-assets-delete-type-compat", http.MethodDelete,
			"/user/domains/{id}/upload", "Delete a banner or logo",
			service.deleteAssetTypeHandler(true),
		},
		{
			"domain-asset-delete-compat", http.MethodDelete,
			"/user/domains/{id}/assets/{assetId}", "Delete a domain image",
			service.deleteAssetHandler(),
		},
	}
	for _, route := range accountRoutes {
		registerLegacy(a, documentDomainBody(a, route.id, huma.Operation{
			OperationID: route.id,
			Method:      route.method,
			Path:        route.path,
			Summary:     route.summary,
			Tags:        []string{"account", "domains"},
			Security:    required,
		}), route.handler)
	}

	adminRoutes := []struct {
		id, method, path, summary string
		handler                   legacyHandler
	}{
		{
			"admin-domains", http.MethodGet, "/admin/domains",
			"List custom domains", service.adminListHandler(),
		},
		{
			"admin-domain", http.MethodGet, "/admin/domains/{id}",
			"Custom domain administration detail",
			service.adminDetailHandler(),
		},
		{
			"admin-domain-toggle", http.MethodPost,
			"/admin/domains/{id}/toggle-active",
			"Toggle a custom domain", service.adminToggleHandler(),
		},
		{
			"admin-domain-asset-preview", http.MethodGet,
			"/admin/domains/{id}/assets/{assetId}/preview",
			"Preview a domain image", service.adminPreviewHandler(),
		},
		{
			"admin-domain-asset-review", http.MethodPost,
			"/admin/domains/{id}/assets/{assetId}/review",
			"Review a domain image", service.reviewAssetHandler(),
		},
	}
	for _, route := range adminRoutes {
		registerLegacy(a, documentDomainBody(a, route.id, huma.Operation{
			OperationID: route.id,
			Method:      route.method,
			Path:        route.path,
			Summary:     route.summary,
			Tags:        []string{"admin", "domains"},
			Security:    required,
		}), route.handler)
	}

	for _, route := range []struct {
		id, path, summary string
		servers           []*huma.Server
		handler           legacyHandler
	}{
		{
			"image-domain-banner-or-logo",
			"/images/domains/{id}/{type}",
			"Approved custom-domain banner or logo",
			[]*huma.Server{{URL: "/", Description: "EVE-KILL images"}},
			service.publicSlotAssetHandler(),
		},
		{
			"image-domain-background",
			"/images/domains/background/{assetId}",
			"Approved custom-domain background",
			[]*huma.Server{{URL: "/", Description: "EVE-KILL images"}},
			service.publicBackgroundHandler(),
		},
		{
			"image-domain-asset-preview",
			"/images/domains/preview/{assetId}",
			"Domain image preview",
			[]*huma.Server{{URL: "/", Description: "EVE-KILL images"}},
			service.publicPreviewHandler(),
		},
		{
			"domain-banner-or-logo", "/domains/asset/{id}/{type}",
			"Approved custom-domain banner or logo",
			nil,
			service.publicSlotAssetHandler(),
		},
		{
			"domain-background", "/domains/bg/{assetId}",
			"Approved custom-domain background",
			nil,
			service.publicBackgroundHandler(),
		},
		{
			"domain-asset-preview", "/domains/preview/{assetId}",
			"Domain image preview", nil, service.publicPreviewHandler(),
		},
	} {
		registerLegacy(a, huma.Operation{
			OperationID: route.id,
			Method:      http.MethodGet,
			Path:        route.path,
			Summary:     route.summary,
			Tags:        []string{"domains", "images"},
			Servers:     route.servers,
		}, route.handler)
	}
}

func (s *domainService) listOwnedHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := s.requireAccount(ctx, req, false)
		if err != nil {
			return legacyPayload{}, err
		}
		domains, err := s.loadOwnedDomains(ctx, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"domains": domains}), nil
	}
}

func (s *domainService) checkSubdomainHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAccount(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		subdomain, reason := validateDomainSubdomain(req.Query.Get("subdomain"))
		if reason != "" {
			return jsonPayload(map[string]any{
				"available": false, "reason": reason,
			}), nil
		}
		exists, err := s.domainSubdomainExists(ctx, subdomain)
		if err != nil {
			return legacyPayload{}, err
		}
		if exists {
			return jsonPayload(map[string]any{
				"available": false,
				"reason":    "This subdomain is already taken",
			}), nil
		}
		return jsonPayload(map[string]any{"available": true}), nil
	}
}

// Runtime decode type for the custom-domain write routes.
//
// Every field is json.RawMessage: parseDomainCreate and parseDomainUpdate walk
// nested theme, widget, navbar and entity structures whose parsers coerce the
// way the TypeScript API did. The concrete request schemas live in
// openapi_body_types.go; presence still reads correctly for the patch path.
type domainWriteBody struct {
	Subdomain         json.RawMessage `json:"subdomain,omitempty" doc:"Subdomain the board answers on."`
	SiteName          json.RawMessage `json:"site_name,omitempty" doc:"Board name shown in the title and header."`
	SiteDescription   json.RawMessage `json:"site_description,omitempty" doc:"Short description for the board."`
	Active            json.RawMessage `json:"active,omitempty" doc:"Whether the board serves traffic."`
	Entities          json.RawMessage `json:"entities,omitempty" doc:"Characters, corporations and alliances the board covers."`
	CampaignIDs       json.RawMessage `json:"campaign_ids,omitempty" doc:"Campaigns featured on the board."`
	CampaignPublicIDs json.RawMessage `json:"campaign_public_ids,omitempty" doc:"Public identifiers of those campaigns."`
	CampaignPolicy    json.RawMessage `json:"campaign_policy,omitempty" doc:"How campaigns are selected for the board."`
	Theme             json.RawMessage `json:"theme,omitempty" doc:"Theme overrides: colors, fonts and background."`
	Widgets           json.RawMessage `json:"widgets,omitempty" doc:"Widgets shown on the board, in order."`
	NavbarLinks       json.RawMessage `json:"navbar_links,omitempty" doc:"Custom navigation entries."`
}

// asMap rebuilds the untyped view the domain parsers expect. Only keys the
// caller actually sent appear, so presence checks behave as they did.
func (b *domainWriteBody) asMap() map[string]any {
	out := map[string]any{}
	for key, raw := range map[string]json.RawMessage{
		"subdomain": b.Subdomain, "site_name": b.SiteName,
		"site_description": b.SiteDescription, "active": b.Active,
		"entities": b.Entities, "campaign_ids": b.CampaignIDs,
		"campaign_public_ids": b.CampaignPublicIDs,
		"campaign_policy":     b.CampaignPolicy, "theme": b.Theme,
		"widgets": b.Widgets, "navbar_links": b.NavbarLinks,
	} {
		if value, found := rawJSONField(raw); found {
			out[key] = value
		}
	}
	return out
}

func (s *domainService) createHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := s.requireAccount(ctx, req, true)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[domainWriteBody](req, domainBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		input, err := parseDomainCreate(body.asMap())
		if err != nil {
			return legacyPayload{}, err
		}
		row, err := s.createDomain(ctx, principal.CharacterID, input)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"domain": row}), nil
	}
}

func (s *domainService) updateHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := s.requireAccount(ctx, req, true)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := domainID(req.Param("id"), "Invalid domain ID")
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[domainWriteBody](req, domainBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		input, err := parseDomainUpdate(body.asMap())
		if err != nil {
			return legacyPayload{}, err
		}
		row, err := s.updateDomain(
			ctx, id, principal.CharacterID, *principal, input,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"domain": row}), nil
	}
}

func (s *domainService) deleteHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := s.requireAccount(ctx, req, true)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := domainID(req.Param("id"), "Invalid domain ID")
		if err != nil {
			return legacyPayload{}, err
		}
		keys, err := s.deleteDomain(ctx, id, principal.CharacterID)
		if err != nil {
			return legacyPayload{}, err
		}
		s.deleteStorageKeys(ctx, keys)
		return jsonPayload(map[string]any{"deleted": true}), nil
	}
}

func (s *domainService) campaignSearchHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := s.requireAccount(ctx, req, false)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := domainID(req.Param("id"), "Invalid domain ID")
		if err != nil {
			return legacyPayload{}, err
		}
		domain, err := s.loadOwnedDomain(
			ctx, s.db, id, principal.CharacterID, false,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if domain == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Domain not found",
			)
		}
		query := strings.TrimSpace(req.Query.Get("q"))
		options := domainCampaignOptions{Limit: 20, Search: query}
		if len([]rune(query)) < 2 {
			options = domainCampaignOptions{
				Limit: domainMaximumCampaigns, OwnOnly: true,
			}
		}
		campaigns, err := s.loadEligibleDomainCampaigns(
			ctx, s.db, domain, *principal, options,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"campaigns": campaigns}), nil
	}
}

func (s *domainService) adminListHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		rows, err := s.loadAdminDomains(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"domains": rows}), nil
	}
}

func (s *domainService) adminDetailHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		id, err := domainID(req.Param("id"), "Invalid domain ID")
		if err != nil {
			return legacyPayload{}, err
		}
		domain, assets, err := s.loadAdminDomain(ctx, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if domain == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Domain not found",
			)
		}
		return jsonPayload(map[string]any{
			"domain": domain, "assets": assets,
		}), nil
	}
}

func (s *domainService) adminToggleHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, true); err != nil {
			return legacyPayload{}, err
		}
		id, err := domainID(req.Param("id"), "Invalid domain ID")
		if err != nil {
			return legacyPayload{}, err
		}
		row, err := s.toggleDomain(ctx, id)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"domain": row}), nil
	}
}

func parseDomainCreate(body map[string]any) (domainCreateInput, error) {
	subdomain, reason := validateDomainSubdomain(domainText(body["subdomain"]))
	if reason != "" {
		return domainCreateInput{}, apiError(http.StatusBadRequest, reason)
	}
	entities, err := parseDomainEntities(body["entities"])
	if err != nil {
		return domainCreateInput{}, err
	}
	navbar, err := parseDomainNavbar(body["navbar_links"])
	if err != nil {
		return domainCreateInput{}, err
	}
	var widgets map[string]any
	widgetsPresent := body["widgets"] != nil
	if widgetsPresent {
		widgets, _, err = parseOptionalDomainWidgets(body, "widgets")
		if err != nil {
			return domainCreateInput{}, err
		}
	}
	siteName, err := optionalDomainString(
		body["site_name"], domainMaximumSiteName, "site_name",
	)
	if err != nil {
		return domainCreateInput{}, err
	}
	description, err := optionalDomainString(
		body["site_description"], domainMaximumDescription,
		"site_description",
	)
	if err != nil {
		return domainCreateInput{}, err
	}
	return domainCreateInput{
		Subdomain:       subdomain,
		Entities:        entities,
		Theme:           sanitizeDomainTheme(body["theme"]),
		NavbarLinks:     navbar,
		Widgets:         widgets,
		WidgetsPresent:  widgetsPresent,
		SiteName:        siteName,
		SiteDescription: description,
	}, nil
}

func parseDomainUpdate(body map[string]any) (domainUpdateInput, error) {
	var input domainUpdateInput
	if value, exists := body["entities"]; exists {
		entities, err := parseDomainEntities(value)
		if err != nil {
			return input, err
		}
		input.EntitiesPresent, input.Entities = true, entities
	}
	if value, exists := body["theme"]; exists {
		input.ThemePresent = true
		input.Theme = sanitizeDomainTheme(value)
	}
	if value, exists := body["navbar_links"]; exists {
		navbar, err := parseDomainNavbar(value)
		if err != nil {
			return input, err
		}
		input.NavbarPresent, input.NavbarLinks = true, navbar
	}
	if _, exists := body["widgets"]; exists {
		widgets, _, err := parseOptionalDomainWidgets(body, "widgets")
		if err != nil {
			return input, err
		}
		input.WidgetsPresent, input.Widgets = true, widgets
	}
	if value, exists := body["site_name"]; exists {
		parsed, err := optionalDomainString(
			value, domainMaximumSiteName, "site_name",
		)
		if err != nil {
			return input, err
		}
		input.SiteNamePresent, input.SiteName = true, parsed
	}
	if value, exists := body["site_description"]; exists {
		parsed, err := optionalDomainString(
			value, domainMaximumDescription, "site_description",
		)
		if err != nil {
			return input, err
		}
		input.SiteDescriptionPresent = true
		input.SiteDescription = parsed
	}
	if value, exists := body["active"]; exists {
		active, ok := value.(bool)
		if !ok {
			return input, apiError(
				http.StatusBadRequest, "active must be a boolean",
			)
		}
		input.ActivePresent, input.Active = true, active
	}
	if value, exists := body["campaign_policy"]; exists {
		policy, ok := domainJSONInteger(value)
		if !ok || policy != domainCampaignAutomatic &&
			policy != domainCampaignAllowlist {
			return input, apiError(
				http.StatusBadRequest, "Invalid campaign policy",
			)
		}
		input.CampaignPolicyPresent = true
		input.CampaignPolicy = int16(policy)
	}
	if value, exists := body["campaign_ids"]; exists {
		ids, err := parseDomainCampaignIDs(value)
		if err != nil {
			return input, err
		}
		input.CampaignIDsPresent, input.CampaignIDs = true, ids
		publicIDs, err := parseDomainPublicCampaignIDs(
			body["campaign_public_ids"], ids,
		)
		if err != nil {
			return input, err
		}
		input.PublicCampaignIDs = publicIDs
	}
	return input, nil
}

func validateDomainSubdomain(raw string) (string, string) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if len(value) < 3 || len(value) > 32 {
		return value, "Subdomain must be 3-32 characters"
	}
	if !domainSubdomainPattern.MatchString(value) {
		return value,
			"Subdomain must be lowercase alphanumeric with optional hyphens"
	}
	if _, reserved := domainReservedSubdomains[value]; reserved {
		return value, "This subdomain is reserved"
	}
	return value, ""
}

func parseDomainEntities(value any) ([]domainEntity, error) {
	items, ok := value.([]any)
	if !ok || len(items) < 1 || len(items) > domainMaximumEntities {
		return nil, apiError(
			http.StatusBadRequest,
			fmt.Sprintf(
				"Entities must be 1-%d items",
				domainMaximumEntities,
			),
		)
	}
	entities := make([]domainEntity, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, apiError(
				http.StatusBadRequest,
				"Invalid entity: each must have type, id, and name",
			)
		}
		entityType, _ := object["type"].(string)
		id, idOK := domainJSONInteger(object["id"])
		name, nameOK := object["name"].(string)
		name = strings.TrimSpace(name)
		if (entityType != "character" &&
			entityType != "corporation" &&
			entityType != "alliance") ||
			!idOK || id <= 0 || id > math.MaxInt32 ||
			!nameOK || name == "" || len([]rune(name)) > 200 {
			return nil, apiError(
				http.StatusBadRequest,
				"Invalid entity: each must have type, id, and name",
			)
		}
		entities = append(entities, domainEntity{
			Type: entityType, ID: int32(id), Name: name,
		})
	}
	return entities, nil
}

func parseDomainNavbar(value any) ([]any, error) {
	if value == nil {
		return []any{}, nil
	}
	items, ok := value.([]any)
	if !ok || len(items) > domainMaximumNavbarLinks {
		return nil, apiError(
			http.StatusBadRequest,
			fmt.Sprintf(
				"Maximum %d navbar links",
				domainMaximumNavbarLinks,
			),
		)
	}
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok || !validDomainNavbarLink(object, true) {
			return nil, apiError(
				http.StatusBadRequest,
				"Each navbar link must have label and href",
			)
		}
	}
	return items, nil
}

func validDomainNavbarLink(
	link map[string]any,
	allowChildren bool,
) bool {
	label, labelOK := link["label"].(string)
	href, hrefOK := link["href"].(string)
	if !labelOK || strings.TrimSpace(label) == "" ||
		len([]rune(label)) > 100 || !hrefOK ||
		!validDomainNavbarURL(href) {
		return false
	}
	children, exists := link["children"]
	if !exists || children == nil {
		return true
	}
	if !allowChildren {
		return false
	}
	groups, ok := children.([]any)
	if !ok || len(groups) > 10 {
		return false
	}
	for _, rawGroup := range groups {
		group, ok := rawGroup.(map[string]any)
		if !ok {
			return false
		}
		rawItems, ok := group["items"].([]any)
		if !ok || len(rawItems) > 20 {
			return false
		}
		for _, rawItem := range rawItems {
			item, ok := rawItem.(map[string]any)
			if !ok || !validDomainNavbarLink(item, false) {
				return false
			}
		}
	}
	return true
}

func validDomainNavbarURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || len([]rune(raw)) > 2048 {
		return false
	}
	if strings.HasPrefix(raw, "/") {
		return !strings.HasPrefix(raw, "//")
	}
	if strings.HasPrefix(raw, "#") {
		return true
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Host != "" &&
		(parsed.Scheme == "https" || parsed.Scheme == "http")
}

func parseOptionalDomainWidgets(
	body map[string]any,
	key string,
) (map[string]any, bool, error) {
	value, exists := body[key]
	if !exists {
		return nil, false, nil
	}
	widgets, ok := value.(map[string]any)
	if !ok {
		return nil, true, apiError(
			http.StatusBadRequest, "Invalid widgets config",
		)
	}
	for _, section := range []string{"top", "left", "right"} {
		items, ok := widgets[section].([]any)
		if !ok {
			return nil, true, apiError(
				http.StatusBadRequest,
				fmt.Sprintf("widgets.%s must be an array", section),
			)
		}
		if len(items) > domainMaximumWidgets {
			return nil, true, apiError(
				http.StatusBadRequest,
				fmt.Sprintf(
					"Maximum %d widgets per section",
					domainMaximumWidgets,
				),
			)
		}
		for _, rawWidget := range items {
			widget, ok := rawWidget.(map[string]any)
			if !ok {
				return nil, true, apiError(
					http.StatusBadRequest, "Invalid widget config",
				)
			}
			kind, _ := widget["type"].(string)
			if _, valid := domainWidgetTypes[kind]; !valid {
				return nil, true, apiError(
					http.StatusBadRequest,
					fmt.Sprintf("Invalid widget type: %s", kind),
				)
			}
			if _, ok := widget["enabled"].(bool); !ok {
				return nil, true, apiError(
					http.StatusBadRequest,
					"Widget enabled must be boolean",
				)
			}
			if kind == "textBlock" {
				if content, ok := widget["content"].(string); ok &&
					len([]rune(content)) > 2000 {
					return nil, true, apiError(
						http.StatusBadRequest,
						"Text block content max 2000 characters",
					)
				}
			}
		}
	}
	if value, exists := widgets["columnRatio"]; exists && value != nil {
		raw, ok := value.(string)
		if !ok {
			return nil, true, apiError(
				http.StatusBadRequest, "Invalid column ratio",
			)
		}
		if raw != "" {
			if _, valid := domainColumnRatios[raw]; valid {
				return widgets, true, nil
			}
			return nil, true, apiError(
				http.StatusBadRequest, "Invalid column ratio",
			)
		}
	}
	return widgets, true, nil
}

func sanitizeDomainTheme(value any) map[string]any {
	input, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	result := make(map[string]any, len(input))
	for key, item := range input {
		if key == "bannerUrl" || key == "logoUrl" {
			continue
		}
		result[key] = item
	}
	return result
}

func mergeDomainTheme(
	existing map[string]any,
	incoming map[string]any,
) map[string]any {
	result := make(map[string]any, len(existing)+len(incoming))
	maps.Copy(result, existing)
	for key, value := range incoming {
		if value == nil {
			delete(result, key)
		} else {
			result[key] = value
		}
	}
	return result
}

func parseDomainCampaignIDs(value any) ([]string, error) {
	items, ok := value.([]any)
	if !ok || len(items) > domainMaximumCampaigns {
		return nil, apiError(
			http.StatusBadRequest,
			fmt.Sprintf(
				"Maximum %d selected campaigns",
				domainMaximumCampaigns,
			),
		)
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		id, ok := item.(string)
		if !ok {
			if number, numberOK := item.(json.Number); numberOK {
				id = number.String()
				ok = true
			}
		}
		if !ok || !domainCampaignIDPattern.MatchString(id) {
			return nil, apiError(
				http.StatusBadRequest, "Invalid campaign ID",
			)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

func parseDomainPublicCampaignIDs(
	value any,
	selected []string,
) ([]string, error) {
	if value == nil {
		return []string{}, nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil, apiError(
			http.StatusBadRequest,
			"campaign_public_ids must be an array",
		)
	}
	selectedSet := make(map[string]struct{}, len(selected))
	for _, id := range selected {
		selectedSet[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		id, ok := item.(string)
		if !ok || !domainCampaignIDPattern.MatchString(id) {
			return nil, apiError(
				http.StatusBadRequest, "Invalid campaign ID",
			)
		}
		if _, selected := selectedSet[id]; !selected {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

func optionalDomainString(
	value any,
	maximum int,
	field string,
) (*string, error) {
	if value == nil {
		return nil, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, apiError(
			http.StatusBadRequest, field+" must be a string or null",
		)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	if len([]rune(text)) > maximum {
		return nil, apiError(
			http.StatusBadRequest,
			fmt.Sprintf("%s must be at most %d characters", field, maximum),
		)
	}
	return &text, nil
}

func domainID(raw, message string) (int32, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil || value <= 0 {
		return 0, apiError(http.StatusBadRequest, message)
	}
	return int32(value), nil
}

func domainJSONInteger(value any) (int64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := strconv.ParseInt(typed.String(), 10, 64)
		return parsed, err == nil
	default:
		number, ok := int64Value(value)
		return number, ok
	}
}

func domainText(value any) string {
	text, _ := value.(string)
	return text
}
