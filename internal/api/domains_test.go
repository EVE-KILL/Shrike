package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestDomainRoutesRegisterCanonicalAndFrontendContracts(t *testing.T) {
	a := humachi.New(
		chi.NewRouter(),
		huma.DefaultConfig("domain-test", "test"),
	)
	registerDomainRoutes(a, Options{})

	routes := []struct {
		method string
		path   string
		secure bool
	}{
		{http.MethodGet, "/me/domains", true},
		{http.MethodPost, "/me/domains", true},
		{http.MethodGet, "/me/domains/check-subdomain", true},
		{http.MethodPatch, "/me/domains/{id}", true},
		{http.MethodDelete, "/me/domains/{id}", true},
		{http.MethodGet, "/me/domains/{id}/campaigns", true},
		{http.MethodPost, "/me/domains/{id}/assets", true},
		{http.MethodDelete, "/me/domains/{id}/assets/{assetId}", true},
		{http.MethodDelete, "/me/domains/{id}/assets", true},
		{http.MethodGet, "/user/domains", true},
		{http.MethodPost, "/user/domains", true},
		{http.MethodGet, "/user/domains/check-subdomain", true},
		{http.MethodPut, "/user/domains/{id}", true},
		{http.MethodPatch, "/user/domains/{id}", true},
		{http.MethodDelete, "/user/domains/{id}", true},
		{
			http.MethodGet,
			"/user/domains/{id}/campaigns/search",
			true,
		},
		{http.MethodPost, "/user/domains/{id}/upload", true},
		{http.MethodDelete, "/user/domains/{id}/upload", true},
		{
			http.MethodDelete,
			"/user/domains/{id}/assets/{assetId}",
			true,
		},
		{http.MethodGet, "/admin/domains", true},
		{http.MethodGet, "/admin/domains/{id}", true},
		{
			http.MethodPost,
			"/admin/domains/{id}/toggle-active",
			true,
		},
		{
			http.MethodGet,
			"/admin/domains/{id}/assets/{assetId}/preview",
			true,
		},
		{
			http.MethodPost,
			"/admin/domains/{id}/assets/{assetId}/review",
			true,
		},
		{http.MethodGet, "/images/domains/{id}/{type}", false},
		{
			http.MethodGet,
			"/images/domains/background/{assetId}",
			false,
		},
		{
			http.MethodGet,
			"/images/domains/preview/{assetId}",
			false,
		},
		{http.MethodGet, "/domains/asset/{id}/{type}", false},
		{http.MethodGet, "/domains/bg/{assetId}", false},
		{http.MethodGet, "/domains/preview/{assetId}", false},
	}
	for _, route := range routes {
		operation := domainTestOperation(
			a.OpenAPI().Paths[route.path], route.method,
		)
		if operation == nil {
			t.Errorf("%s %s is not registered", route.method, route.path)
			continue
		}
		secured := false
		if len(operation.Security) > 0 {
			_, secured = operation.Security[0]["eveSession"]
		}
		if route.secure && (len(operation.Security) != 1 || !secured) {
			t.Errorf(
				"%s %s security = %#v",
				route.method, route.path, operation.Security,
			)
		}
		if !route.secure && len(operation.Security) != 0 {
			t.Errorf(
				"%s %s unexpectedly requires a session",
				route.method, route.path,
			)
		}
	}
}

func TestSiteConfigurationUsesHostAndConcreteContract(t *testing.T) {
	handler := Site(Options{
		Version: "test-version",
		Commit:  "test-commit",
		DB:      stubDatabase{},
	})
	for _, test := range []struct {
		host         string
		isDomainHost bool
	}{
		{host: "eve-kill.com", isDomainHost: false},
		{host: "www.eve-kill.com", isDomainHost: false},
		{host: "127.0.0.1:4000", isDomainHost: false},
		{host: "nested.board.eve-kill.com", isDomainHost: true},
	} {
		request := httptest.NewRequest(
			http.MethodGet,
			"http://"+test.host+"/api/site",
			nil,
		)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf(
				"%s returned %d: %s",
				test.host, response.Code, response.Body.String(),
			)
		}
		var body SiteConfigurationResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Domain != nil || body.IsDomainHost != test.isDomainHost {
			t.Errorf("%s response = %#v", test.host, body)
		}
		if response.Header().Get("Vary") != "Host" ||
			response.Header().Get("X-Cache") != "BYPASS" ||
			response.Header().Get("Cache-Control") != siteConfigurationCache {
			t.Errorf("%s headers = %#v", test.host, response.Header())
		}
	}

	document := New(Options{Version: "test"}).OpenAPI()
	operation := document.Paths["/site"].Get
	if operation == nil {
		t.Fatal("GET /site is not registered")
	}
	response := operation.Responses["200"]
	media := response.Content["application/json"]
	if media == nil || media.Schema == nil ||
		media.Schema.Ref == "" {
		t.Fatalf("/site response schema = %#v", media)
	}
	schema := document.Components.Schemas.Map()[strings.TrimPrefix(media.Schema.Ref, "#/components/schemas/")]
	if schema == nil {
		t.Fatalf("missing schema for %s", media.Schema.Ref)
	}
	allowsAdditional, _ := schema.AdditionalProperties.(bool)
	domainSchema := schema.Properties["domain"]
	if schema.Properties["domain"] == nil ||
		schema.Properties["isDomainHost"] == nil ||
		allowsAdditional ||
		len(domainSchema.OneOf) != 2 ||
		domainSchema.OneOf[0].Ref !=
			"#/components/schemas/SiteDomainConfiguration" ||
		domainSchema.OneOf[1].Type != "null" {
		encoded, _ := json.Marshal(schema)
		t.Fatalf("/site response is not concrete: %s", encoded)
	}
}

func TestCustomDomainHostQuery(t *testing.T) {
	for _, test := range []struct {
		host, predicate, value string
		domainHost             bool
	}{
		{"eve-kill.com", "", "", false},
		{"localhost", "", "", false},
		{"10.0.0.1", "", "", false},
		{
			"board.eve-kill.com",
			"domain.subdomain = $1",
			"board",
			true,
		},
		{
			"board.localhost",
			"domain.subdomain = $1",
			"board",
			true,
		},
		{"nested.board.eve-kill.com", "", "", true},
		{
			"killboard.example.com",
			"LOWER(domain.custom_hostname) = $1",
			"killboard.example.com",
			true,
		},
	} {
		predicate, value, domainHost := customDomainHostQuery(test.host)
		if predicate != test.predicate ||
			value != test.value ||
			domainHost != test.domainHost {
			t.Errorf(
				"%q = %q, %q, %v; want %q, %q, %v",
				test.host, predicate, value, domainHost,
				test.predicate, test.value, test.domainHost,
			)
		}
	}
}

func TestSiteDomainConfigurationMapsStoredPresentation(t *testing.T) {
	config := siteDomainConfiguration(map[string]any{
		"id":              int32(7),
		"subdomain":       "my-board",
		"custom_hostname": "kills.example.com",
		"user_id":         int32(90000001),
		"entities": []any{
			map[string]any{
				"type": "character", "id": float64(90000001),
				"name": "Pilot",
			},
			map[string]any{
				"type": "corporation", "id": float64(98000001),
				"name": "Corporation",
			},
		},
		"theme": map[string]any{
			"bannerUrl":         "/api/domains/asset/7/banner",
			"transparentBanner": true,
			"contentOpacity":    float64(80),
			"defaultThemeOverrides": map[string]any{
				"brandPrimary": "#123456",
			},
		},
		"navbar_links": []any{map[string]any{
			"label": "Kills", "href": "/kills/latest",
			"children": []any{map[string]any{
				"label": "Activity",
				"items": []any{map[string]any{
					"label": "Latest", "href": "/kills/latest",
				}},
			}},
		}},
		"widgets":             nil,
		"site_name":           "My Board",
		"site_description":    "Board description",
		"campaign_policy":     int16(1),
		"campaign_ids":        []any{"one", "two"},
		"public_campaign_ids": []any{"two"},
		"background_ids":      []any{float64(11), float64(12)},
	})
	if config.Theme.BannerURL == nil ||
		*config.Theme.BannerURL != "/images/domains/7/banner" {
		t.Fatalf("banner URL = %#v", config.Theme.BannerURL)
	}
	if config.Theme.ContentOpacity == nil ||
		*config.Theme.ContentOpacity != 80 ||
		config.Theme.TransparentBanner == nil ||
		!*config.Theme.TransparentBanner {
		t.Fatalf("theme = %#v", config.Theme)
	}
	if config.Theme.DefaultThemeOverrides["brandPrimary"] != "#123456" {
		t.Fatalf("theme overrides = %#v", config.Theme.DefaultThemeOverrides)
	}
	if !reflect.DeepEqual(
		config.EntityIDs,
		SiteDomainEntityIDs{
			CharacterIDs:   []int32{90000001},
			CorporationIDs: []int32{98000001},
			AllianceIDs:    []int32{},
		},
	) {
		t.Fatalf("entity IDs = %#v", config.EntityIDs)
	}
	if !reflect.DeepEqual(config.Backgrounds, []string{
		"/images/domains/background/11",
		"/images/domains/background/12",
	}) {
		t.Fatalf("backgrounds = %#v", config.Backgrounds)
	}
	if len(config.Widgets.Top) != 1 ||
		len(config.Widgets.Left) != 6 ||
		len(config.Widgets.Right) != 1 {
		t.Fatalf("default widgets = %#v", config.Widgets)
	}
	if len(config.NavbarLinks) != 1 ||
		len(config.NavbarLinks[0].Children) != 1 ||
		len(config.NavbarLinks[0].Children[0].Items) != 1 {
		t.Fatalf("navbar = %#v", config.NavbarLinks)
	}
}

func domainTestOperation(
	item *huma.PathItem,
	method string,
) *huma.Operation {
	if item == nil {
		return nil
	}
	switch method {
	case http.MethodGet:
		return item.Get
	case http.MethodPost:
		return item.Post
	case http.MethodPut:
		return item.Put
	case http.MethodPatch:
		return item.Patch
	case http.MethodDelete:
		return item.Delete
	default:
		return nil
	}
}

func TestDomainSubdomainValidationMatchesAvailabilityContract(t *testing.T) {
	for _, test := range []struct {
		raw, normalized, reason string
	}{
		{" My-Board ", "my-board", ""},
		{"ab", "ab", "Subdomain must be 3-32 characters"},
		{
			"-board", "-board",
			"Subdomain must be lowercase alphanumeric with optional hyphens",
		},
		{"API", "api", "This subdomain is reserved"},
		{
			"board_name", "board_name",
			"Subdomain must be lowercase alphanumeric with optional hyphens",
		},
	} {
		got, reason := validateDomainSubdomain(test.raw)
		if got != test.normalized || reason != test.reason {
			t.Errorf(
				"validate %q = %q, %q; want %q, %q",
				test.raw, got, reason, test.normalized, test.reason,
			)
		}
	}
}

func TestDomainCreateSanitizesManagedThemeAndValidatesNestedLinks(
	t *testing.T,
) {
	body := map[string]any{
		"subdomain": "my-board",
		"entities": []any{map[string]any{
			"type": "corporation", "id": json.Number("98000001"),
			"name": "Test Corporation",
		}},
		"theme": map[string]any{
			"brandPrimary": "#123456",
			"bannerUrl":    "https://evil.invalid/banner",
			"logoUrl":      "javascript:alert(1)",
		},
		"navbar_links": []any{map[string]any{
			"label": "Kills", "href": "/kills/latest",
			"children": []any{map[string]any{
				"items": []any{map[string]any{
					"label": "External",
					"href":  "https://example.com/path",
				}},
			}},
		}},
	}
	input, err := parseDomainCreate(body)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := input.Theme["bannerUrl"]; exists {
		t.Fatal("bannerUrl survived user theme sanitization")
	}
	if _, exists := input.Theme["logoUrl"]; exists {
		t.Fatal("logoUrl survived user theme sanitization")
	}
	if input.Theme["brandPrimary"] != "#123456" {
		t.Fatalf("theme = %#v", input.Theme)
	}

	body["navbar_links"] = []any{map[string]any{
		"label": "Unsafe", "href": "javascript:alert(1)",
	}}
	if _, err := parseDomainCreate(body); err == nil {
		t.Fatal("javascript navbar URL was accepted")
	}
}

func TestDomainWidgetsAcceptCampaignListing(t *testing.T) {
	widgets := map[string]any{
		"top": []any{},
		"left": []any{map[string]any{
			"type": "campaigns", "enabled": true,
		}},
		"right": []any{},
	}

	got, exists, err := parseOptionalDomainWidgets(
		map[string]any{"widgets": widgets},
		"widgets",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !exists || !reflect.DeepEqual(got, widgets) {
		t.Fatalf("widgets = %#v, exists = %t", got, exists)
	}
}

func TestMergeDomainThemePreservesServerManagedKeys(t *testing.T) {
	got := mergeDomainTheme(
		map[string]any{
			"bannerUrl": "/api/domains/asset/1/banner",
			"color":     "old",
			"keep":      true,
		},
		sanitizeDomainTheme(map[string]any{
			"bannerUrl": "https://evil.invalid",
			"color":     "new",
			"keep":      nil,
		}),
	)
	want := map[string]any{
		"bannerUrl": "/api/domains/asset/1/banner",
		"color":     "new",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("merged theme = %#v, want %#v", got, want)
	}
}

func TestDomainImageMagicAndStorageKeysRejectSpoofing(t *testing.T) {
	png := make([]byte, 12)
	copy(png, []byte{0x89, 0x50, 0x4e, 0x47})
	if got := detectDomainImageMIME(png); got != "image/png" {
		t.Fatalf("PNG MIME = %q", got)
	}
	riff := []byte("RIFF0000NOPE")
	if got := detectDomainImageMIME(riff); got != "" {
		t.Fatalf("non-WebP RIFF MIME = %q", got)
	}
	webp := []byte("RIFF0000WEBP")
	if got := detectDomainImageMIME(webp); got != "image/webp" {
		t.Fatalf("WebP MIME = %q", got)
	}

	key, ok := domainAssetStorageKey(12, "background", 34)
	if !ok || key != "domains/12/background_34" {
		t.Fatalf("storage key = %q, %v", key, ok)
	}
	if validDomainStorageReference(domainStorageReference{
		AssetID: 34, DomainID: 12, Type: "background",
		Key: "domains/12/../private",
	}) {
		t.Fatal("traversal-shaped database key was accepted")
	}
}

func TestDomainPreviewTokensAreBoundToAssetAndHash(t *testing.T) {
	service := &domainService{
		auth: &authService{stateSecret: []byte("preview test secret")},
	}
	token := service.domainPreviewToken(42, strings.Repeat("a", 64))
	if token == "" ||
		!service.validDomainPreviewToken(
			42, strings.Repeat("a", 64), token,
		) {
		t.Fatal("valid preview token was rejected")
	}
	if service.validDomainPreviewToken(
		43, strings.Repeat("a", 64), token,
	) || service.validDomainPreviewToken(
		42, strings.Repeat("b", 64), token,
	) {
		t.Fatal("preview token was not bound to asset metadata")
	}
}

func TestDomainAssetPayloadUsesVerifiedBytesAndStoredType(t *testing.T) {
	png := make([]byte, 12)
	copy(png, []byte{0x89, 0x50, 0x4e, 0x47})
	hash := sha256.Sum256(png)
	service := &domainService{assets: &domainMemoryAssetStorage{
		objects: map[string][]byte{
			"domains/12/logo_34": png,
		},
	}}
	payload, err := service.domainAssetPayload(
		context.Background(),
		map[string]any{
			"id": int32(34), "domain_id": int32(12), "type": "logo",
			"storage_key":  "domains/12/logo_34",
			"content_type": "image/png", "file_size": int32(len(png)),
			"file_hash": hex.EncodeToString(hash[:]),
		},
		"private, no-store",
		"Asset not found",
	)
	if err != nil {
		t.Fatal(err)
	}
	if payload.ContentType != "image/png" ||
		!reflect.DeepEqual(payload.RawBody, png) ||
		payload.Headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("asset payload = %#v", payload)
	}
}

func TestDomainPaletteSeedMatchesFrontendAccentMath(t *testing.T) {
	theme := map[string]any{}
	seedDomainTheme(theme, map[string]any{
		"main_color":      "#001122",
		"secondary_color": "#ff0000",
		"tertiary_color":  "#00aa88",
	})
	overrides := theme["defaultThemeOverrides"].(map[string]any)
	if overrides["brandPrimary"] != "#ff0000" ||
		overrides["brandPrimaryHover"] != "#d50000" {
		t.Fatalf("palette overrides = %#v", overrides)
	}
}

type domainMemoryAssetStorage struct {
	objects map[string][]byte
}

func (s *domainMemoryAssetStorage) Put(
	_ context.Context,
	key string,
	body []byte,
	_ string,
) error {
	s.objects[key] = append([]byte(nil), body...)
	return nil
}

func (s *domainMemoryAssetStorage) Get(
	_ context.Context,
	key string,
) ([]byte, error) {
	body := s.objects[key]
	if body == nil {
		return nil, nil
	}
	return append([]byte(nil), body...), nil
}

func (s *domainMemoryAssetStorage) Delete(
	_ context.Context,
	key string,
) error {
	delete(s.objects, key)
	return nil
}
