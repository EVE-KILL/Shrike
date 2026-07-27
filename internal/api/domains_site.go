package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	siteConfigurationCacheTTL = time.Minute
	siteConfigurationCache    = "public, max-age=30, s-maxage=60, stale-while-revalidate=60"
)

type siteRequestHostContextKey struct{}

// SiteConfigurationResponse is the host-aware bootstrap consumed by Nuxt.
// Keeping it as a real Huma response makes the generated TypeScript contract
// authoritative for both SSR and browser rendering.
type SiteConfigurationResponse struct {
	Domain       *SiteDomainConfiguration `json:"domain"`
	IsDomainHost bool                     `json:"isDomainHost"`
}

type SiteDomainConfiguration struct {
	ID                int32                  `json:"id"`
	Subdomain         string                 `json:"subdomain"`
	CustomHostname    *string                `json:"customHostname"`
	UserID            int32                  `json:"userId"`
	Entities          []SiteDomainEntity     `json:"entities" nullable:"false"`
	EntityIDs         SiteDomainEntityIDs    `json:"entityIds"`
	Theme             SiteDomainTheme        `json:"theme"`
	NavbarLinks       []SiteDomainNavbarLink `json:"navbarLinks" nullable:"false"`
	Widgets           SiteDomainWidgets      `json:"widgets"`
	SiteName          *string                `json:"siteName"`
	SiteDescription   *string                `json:"siteDescription"`
	CampaignPolicy    int16                  `json:"campaignPolicy" enum:"0,1"`
	CampaignIDs       []string               `json:"campaignIds" nullable:"false"`
	PublicCampaignIDs []string               `json:"publicCampaignIds" nullable:"false"`
	Backgrounds       []string               `json:"backgrounds" nullable:"false"`
}

type SiteDomainEntity struct {
	Type string `json:"type" enum:"character,corporation,alliance"`
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type SiteDomainEntityIDs struct {
	CharacterIDs   []int32 `json:"characterIds" nullable:"false"`
	CorporationIDs []int32 `json:"corporationIds" nullable:"false"`
	AllianceIDs    []int32 `json:"allianceIds" nullable:"false"`
}

type SiteDomainTheme struct {
	PrimaryColor            *string           `json:"primaryColor,omitempty"`
	AccentColor             *string           `json:"accentColor,omitempty"`
	BackgroundColor         *string           `json:"bgColor,omitempty"`
	TextColor               *string           `json:"textColor,omitempty"`
	BannerURL               *string           `json:"bannerUrl,omitempty"`
	LogoURL                 *string           `json:"logoUrl,omitempty"`
	ShowLogoInBanner        *bool             `json:"showLogoInBanner,omitempty"`
	ShowNameInBanner        *bool             `json:"showNameInBanner,omitempty"`
	ShowDescriptionInBanner *bool             `json:"showDescriptionInBanner,omitempty"`
	TransparentBanner       *bool             `json:"transparentBanner,omitempty"`
	ContentOpacity          *float64          `json:"contentOpacity,omitempty"`
	DefaultThemePreset      *string           `json:"defaultThemePreset,omitempty"`
	DefaultThemeOverrides   map[string]string `json:"defaultThemeOverrides,omitempty"`
}

type SiteDomainNavbarLink struct {
	Label    string                  `json:"label"`
	Href     string                  `json:"href"`
	External *bool                   `json:"external,omitempty"`
	Icon     *string                 `json:"icon,omitempty"`
	Children []SiteDomainNavbarGroup `json:"children,omitempty" nullable:"false"`
}

type SiteDomainNavbarGroup struct {
	Label *string                `json:"label,omitempty"`
	Items []SiteDomainNavbarItem `json:"items" nullable:"false"`
}

type SiteDomainNavbarItem struct {
	Label    string  `json:"label"`
	Href     string  `json:"href"`
	External *bool   `json:"external,omitempty"`
	Icon     *string `json:"icon,omitempty"`
}

type SiteDomainWidgets struct {
	Top         []SiteDomainWidget `json:"top" nullable:"false"`
	Left        []SiteDomainWidget `json:"left" nullable:"false"`
	Right       []SiteDomainWidget `json:"right" nullable:"false"`
	ColumnRatio *string            `json:"columnRatio,omitempty"`
}

type SiteDomainWidget struct {
	Type         string  `json:"type" enum:"mostValuable,killList,topCharacters,topCorporations,topAlliances,topShips,topSystems,topRegions,entityInfo,textBlock"`
	Enabled      bool    `json:"enabled"`
	Content      *string `json:"content,omitempty"`
	KilllistType *string `json:"killlistType,omitempty"`
}

type siteConfigurationOutput struct {
	CacheControl string `header:"Cache-Control"`
	Vary         string `header:"Vary"`
	CacheStatus  string `header:"X-Cache"`
	Body         SiteConfigurationResponse
}

type siteConfigurationService struct {
	db     Database
	cache  *redis.Client
	commit string
}

func registerSiteConfigurationRoute(a huma.API, opts Options) {
	service := &siteConfigurationService{
		db: opts.DB, cache: opts.Cache, commit: opts.Commit,
	}
	huma.Register(a, huma.Operation{
		OperationID: "site-configuration",
		Method:      http.MethodGet,
		Path:        "/site",
		Summary:     "Configuration for the current website host",
		Description: "Returns the active custom-domain presentation and entity scope selected by the request Host. Apex, development, and IP hosts return domain: null with isDomainHost: false; an unknown custom host returns domain: null with isDomainHost: true.",
		Tags:        []string{"site", "domains"},
		Errors:      []int{http.StatusServiceUnavailable},
		Extensions:  map[string]any{"x-audience": "public"},
		Middlewares: huma.Middlewares{
			func(ctx huma.Context, next func(huma.Context)) {
				host := ctx.Host()
				if host == "" {
					host = firstForwarded(ctx.Header("X-Forwarded-Host"))
				}
				next(huma.WithValue(
					ctx,
					siteRequestHostContextKey{},
					normalizeRequestHost(host),
				))
			},
		},
	}, func(
		ctx context.Context,
		_ *struct{},
	) (*siteConfigurationOutput, error) {
		host, _ := ctx.Value(siteRequestHostContextKey{}).(string)
		body, cacheStatus, err := service.resolve(ctx, host)
		if err != nil {
			var unavailable *legacyAPIError
			if errors.As(err, &unavailable) &&
				unavailable.Status == http.StatusServiceUnavailable {
				return nil, huma.Error503ServiceUnavailable(unavailable.Message)
			}
			log.Error().Err(err).Str("host", host).
				Msg("load site configuration")
			return nil, err
		}
		return &siteConfigurationOutput{
			CacheControl: siteConfigurationCache,
			Vary:         "Host",
			CacheStatus:  cacheStatus,
			Body:         body,
		}, nil
	})

	// Huma deliberately does not infer nullable object references from Go
	// pointers. The wire contract does return JSON null for apex and unknown
	// hosts, so describe that union explicitly for generated clients.
	if schema := a.OpenAPI().Components.Schemas.Map()["SiteConfigurationResponse"]; schema != nil {
		schema.Properties["domain"] = &huma.Schema{
			OneOf: []*huma.Schema{
				{Ref: "#/components/schemas/SiteDomainConfiguration"},
				{Type: "null"},
			},
		}
	}
}

func (s *siteConfigurationService) resolve(
	ctx context.Context,
	host string,
) (SiteConfigurationResponse, string, error) {
	predicate, value, domainHost := customDomainHostQuery(host)
	response := SiteConfigurationResponse{IsDomainHost: domainHost}
	if !domainHost || predicate == "" {
		return response, "BYPASS", nil
	}

	key := s.cacheKey(host)
	if entry, ok := cacheLoad(ctx, s.cache, key); ok {
		if json.Unmarshal(entry.Body, &response) == nil {
			return response, "HIT", nil
		}
	}

	row, err := domainQueryMap(ctx, s.db, fmt.Sprintf(`
		SELECT domain.id, domain.subdomain, domain.custom_hostname,
		       domain.user_id, domain.entities, domain.theme,
		       domain.navbar_links, domain.widgets, domain.site_name,
		       domain.site_description, domain.campaign_policy,
		       (
		         SELECT COALESCE(
		           jsonb_agg(asset.id ORDER BY asset.id),
		           '[]'::jsonb
		         )
		         FROM domain_assets asset
		         WHERE asset.domain_id = domain.id
		           AND asset.type = 'background'
		           AND asset.status = 'approved'
		       ) AS background_ids,
		       (
		         SELECT COALESCE(
		           jsonb_agg(selection.campaign_id ORDER BY selection.created_at),
		           '[]'::jsonb
		         )
		         FROM custom_domain_campaigns selection
		         WHERE selection.domain_id = domain.id
		       ) AS campaign_ids,
		       (
		         SELECT COALESCE(
		           jsonb_agg(selection.campaign_id ORDER BY selection.created_at)
		             FILTER (WHERE selection.public_on_domain),
		           '[]'::jsonb
		         )
		         FROM custom_domain_campaigns selection
		         WHERE selection.domain_id = domain.id
		       ) AS public_campaign_ids
		FROM custom_domains domain
		WHERE domain.active IS TRUE AND %s
		LIMIT 1`, predicate), value)
	if err != nil {
		return response, "MISS", err
	}
	if row != nil {
		domain := siteDomainConfiguration(row)
		response.Domain = &domain
	}

	if body, marshalErr := json.Marshal(response); marshalErr == nil {
		cacheStore(context.WithoutCancel(ctx), s.cache, key, cachedResponse{
			ContentType: "application/json",
			Body:        body,
		}, siteConfigurationCacheTTL)
	}
	return response, "MISS", nil
}

func (s *siteConfigurationService) cacheKey(host string) string {
	build := s.commit
	if build == "" {
		build = "dev"
	}
	return "shrike:site:" + build + ":" + host
}

func siteDomainConfiguration(row map[string]any) SiteDomainConfiguration {
	entities := siteDomainEntities(row["entities"])
	config := SiteDomainConfiguration{
		ID:                int32From(row["id"]),
		Subdomain:         siteStringOrEmpty(row["subdomain"]),
		CustomHostname:    stringPointer(row["custom_hostname"]),
		UserID:            int32From(row["user_id"]),
		Entities:          entities,
		EntityIDs:         siteDomainEntityIDs(entities),
		Theme:             siteDomainTheme(jsonObject(row["theme"])),
		NavbarLinks:       siteDomainNavbarLinks(row["navbar_links"]),
		Widgets:           siteDomainWidgets(row["widgets"]),
		SiteName:          stringPointer(row["site_name"]),
		SiteDescription:   stringPointer(row["site_description"]),
		CampaignPolicy:    int16From(row["campaign_policy"]),
		CampaignIDs:       siteStringArray(row["campaign_ids"]),
		PublicCampaignIDs: siteStringArray(row["public_campaign_ids"]),
		Backgrounds:       siteDomainBackgrounds(row["background_ids"]),
	}
	return config
}

func siteDomainEntities(value any) []SiteDomainEntity {
	result := []SiteDomainEntity{}
	for _, raw := range jsonArray(value) {
		entity := jsonObject(raw)
		kind, _ := stringValue(entity["type"])
		id := int32From(entity["id"])
		name, _ := stringValue(entity["name"])
		if id <= 0 || name == "" ||
			kind != "character" &&
				kind != "corporation" &&
				kind != "alliance" {
			continue
		}
		result = append(result, SiteDomainEntity{
			Type: kind, ID: id, Name: name,
		})
	}
	return result
}

func siteDomainEntityIDs(entities []SiteDomainEntity) SiteDomainEntityIDs {
	ids := SiteDomainEntityIDs{
		CharacterIDs:   []int32{},
		CorporationIDs: []int32{},
		AllianceIDs:    []int32{},
	}
	for _, entity := range entities {
		switch entity.Type {
		case "character":
			ids.CharacterIDs = append(ids.CharacterIDs, entity.ID)
		case "corporation":
			ids.CorporationIDs = append(ids.CorporationIDs, entity.ID)
		case "alliance":
			ids.AllianceIDs = append(ids.AllianceIDs, entity.ID)
		}
	}
	return ids
}

func siteDomainTheme(value map[string]any) SiteDomainTheme {
	overrides := map[string]string{}
	for key, raw := range jsonObject(value["defaultThemeOverrides"]) {
		if text, ok := stringValue(raw); ok {
			overrides[key] = text
		}
	}
	if len(overrides) == 0 {
		overrides = nil
	}
	return SiteDomainTheme{
		PrimaryColor:            stringPointer(value["primaryColor"]),
		AccentColor:             stringPointer(value["accentColor"]),
		BackgroundColor:         stringPointer(value["bgColor"]),
		TextColor:               stringPointer(value["textColor"]),
		BannerURL:               canonicalDomainImagePointer(value["bannerUrl"]),
		LogoURL:                 canonicalDomainImagePointer(value["logoUrl"]),
		ShowLogoInBanner:        boolPointer(value["showLogoInBanner"]),
		ShowNameInBanner:        boolPointer(value["showNameInBanner"]),
		ShowDescriptionInBanner: boolPointer(value["showDescriptionInBanner"]),
		TransparentBanner:       boolPointer(value["transparentBanner"]),
		ContentOpacity:          floatPointer(value["contentOpacity"]),
		DefaultThemePreset:      stringPointer(value["defaultThemePreset"]),
		DefaultThemeOverrides:   overrides,
	}
}

func siteDomainNavbarLinks(value any) []SiteDomainNavbarLink {
	result := []SiteDomainNavbarLink{}
	for _, raw := range jsonArray(value) {
		link := jsonObject(raw)
		label, labelOK := stringValue(link["label"])
		href, hrefOK := stringValue(link["href"])
		if !labelOK || !hrefOK {
			continue
		}
		item := SiteDomainNavbarLink{
			Label: label, Href: href,
			External: boolPointer(link["external"]),
			Icon:     stringPointer(link["icon"]),
		}
		for _, rawGroup := range jsonArray(link["children"]) {
			group := jsonObject(rawGroup)
			outputGroup := SiteDomainNavbarGroup{
				Label: stringPointer(group["label"]),
				Items: []SiteDomainNavbarItem{},
			}
			for _, rawChild := range jsonArray(group["items"]) {
				child := jsonObject(rawChild)
				childLabel, childLabelOK := stringValue(child["label"])
				childHref, childHrefOK := stringValue(child["href"])
				if !childLabelOK || !childHrefOK {
					continue
				}
				outputGroup.Items = append(
					outputGroup.Items,
					SiteDomainNavbarItem{
						Label: childLabel, Href: childHref,
						External: boolPointer(child["external"]),
						Icon:     stringPointer(child["icon"]),
					},
				)
			}
			item.Children = append(item.Children, outputGroup)
		}
		result = append(result, item)
	}
	return result
}

func siteDomainWidgets(value any) SiteDomainWidgets {
	if value == nil {
		value = defaultDomainWidgets()
	}
	object := jsonObject(value)
	return SiteDomainWidgets{
		Top:         siteDomainWidgetList(object["top"]),
		Left:        siteDomainWidgetList(object["left"]),
		Right:       siteDomainWidgetList(object["right"]),
		ColumnRatio: stringPointer(object["columnRatio"]),
	}
}

func siteDomainWidgetList(value any) []SiteDomainWidget {
	result := []SiteDomainWidget{}
	for _, raw := range jsonArray(value) {
		widget := jsonObject(raw)
		kind, ok := stringValue(widget["type"])
		if !ok || kind == "" {
			continue
		}
		result = append(result, SiteDomainWidget{
			Type:         kind,
			Enabled:      boolFrom(widget["enabled"]),
			Content:      stringPointer(widget["content"]),
			KilllistType: stringPointer(widget["killlistType"]),
		})
	}
	return result
}

func siteDomainBackgrounds(value any) []string {
	result := []string{}
	for _, raw := range jsonArray(value) {
		id := int64From(raw)
		if id > 0 {
			result = append(
				result,
				fmt.Sprintf("/images/domains/background/%d", id),
			)
		}
	}
	return result
}

func siteStringArray(value any) []string {
	result := []string{}
	for _, raw := range jsonArray(value) {
		if text, ok := stringValue(raw); ok {
			result = append(result, text)
		}
	}
	return result
}

func siteStringOrEmpty(value any) string {
	text, _ := stringValue(value)
	return text
}

func stringPointer(value any) *string {
	text, ok := stringValue(value)
	if !ok {
		return nil
	}
	return &text
}

func boolPointer(value any) *bool {
	boolean, ok := boolValue(value)
	if !ok {
		return nil
	}
	return &boolean
}

func floatPointer(value any) *float64 {
	number, ok := float64Value(value)
	if !ok {
		return nil
	}
	return &number
}

func canonicalDomainImagePointer(value any) *string {
	raw := stringPointer(value)
	if raw == nil {
		return nil
	}
	canonical := strings.Replace(
		*raw,
		"/api/domains/asset/",
		"/images/domains/",
		1,
	)
	return &canonical
}
