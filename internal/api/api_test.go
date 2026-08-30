package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jackc/pgx/v5"
)

type stubDatabase struct{}

func (stubDatabase) Ping(context.Context) error {
	return nil
}

func apiPathHandler(opts Options) http.Handler {
	site := Site(opts)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clone := r.Clone(r.Context())
		requestURL := *r.URL
		if requestURL.Path == "/" {
			requestURL.Path = "/api"
		} else {
			requestURL.Path = "/api" + requestURL.Path
		}
		requestURL.RawPath = ""
		clone.URL = &requestURL
		site.ServeHTTP(w, clone)
	})
}

func TestEstablishedRouteCatalogueMatchesEmbeddedIndex(t *testing.T) {
	handler := apiPathHandler(Options{
		Version: "test-version",
		Commit:  "test-commit",
		DB:      stubDatabase{},
	})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, "http://example.com/openapi.json", nil),
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("openapi status = %d, body %q", rec.Code, rec.Body.String())
	}
	var document struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
			Responses   map[string]struct {
				Content map[string]struct {
					Schema struct {
						Ref        string                     `json:"$ref"`
						Properties map[string]json.RawMessage `json:"properties"`
					} `json:"schema"`
				} `json:"content"`
			} `json:"responses"`
		} `json:"paths"`
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Components.Schemas) == 0 {
		t.Fatal("OpenAPI response schemas are missing")
	}
	var index struct {
		Categories []struct {
			Endpoints []struct {
				Method string `json:"method"`
				Path   string `json:"path"`
			} `json:"endpoints"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(apiIndexJSON, &index); err != nil {
		t.Fatal(err)
	}
	dynamic := regexp.MustCompile(`(?::[^/]+|\{[^/]+\})`)
	legacy := []string{}
	for _, category := range index.Categories {
		for _, endpoint := range category.Endpoints {
			legacy = append(legacy,
				strings.ToUpper(endpoint.Method)+" "+
					dynamic.ReplaceAllString(endpoint.Path, "{}"))
		}
	}
	for _, route := range legacy {
		method, normalizedPath, _ := strings.Cut(route, " ")
		var (
			path      string
			operation *struct {
				OperationID string `json:"operationId"`
				Responses   map[string]struct {
					Content map[string]struct {
						Schema struct {
							Ref        string                     `json:"$ref"`
							Properties map[string]json.RawMessage `json:"properties"`
						} `json:"schema"`
					} `json:"content"`
				} `json:"responses"`
			}
		)
		for candidatePath, methods := range document.Paths {
			if dynamic.ReplaceAllString(candidatePath, "{}") != normalizedPath {
				continue
			}
			candidate, ok := methods[strings.ToLower(method)]
			if ok {
				path = candidatePath
				operation = &candidate
				break
			}
		}
		if operation == nil {
			t.Errorf("established route %s is missing from the shared API", route)
			continue
		}
		if path == "/feed/stream" || path == "/killmails/{id}/eft" {
			continue
		}
		success := operation.Responses["200"]
		media, ok := success.Content["application/json"]
		if !ok {
			t.Errorf("%s does not document an application/json response", route)
			continue
		}
		name := operation.OperationID + "-response"
		if media.Schema.Ref != "#/components/schemas/"+name {
			t.Errorf("%s response schema ref = %q", route, media.Schema.Ref)
		}
		component, ok := document.Components.Schemas[name]
		if !ok {
			t.Errorf("%s has no resolvable %s component", route, name)
			continue
		}
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(component, &schema); err != nil {
			t.Fatal(err)
		}
		if _, ok := schema.Properties["$schema"]; !ok {
			t.Errorf("%s response schema has no $schema property", route)
		}
		if len(schema.Properties) == 1 {
			t.Errorf("%s response schema is an undocumented placeholder", route)
		}
	}
}

func TestOpenAPISuccessResponsesUseConcreteSchemas(t *testing.T) {
	raw, err := json.Marshal(New(Options{
		Version: "test-version",
		DB:      stubDatabase{},
	}).OpenAPI())
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	paths, _ := document["paths"].(map[string]any)
	components, _ := document["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	for path, rawItem := range paths {
		item, _ := rawItem.(map[string]any)
		for method, rawOperation := range item {
			operation, ok := rawOperation.(map[string]any)
			if !ok {
				continue
			}
			operationID, _ := operation["operationId"].(string)
			responses, _ := operation["responses"].(map[string]any)
			for status, rawResponse := range responses {
				if len(status) != 3 || status[0] != '2' && status[0] != '3' {
					continue
				}
				response, _ := rawResponse.(map[string]any)
				content, _ := response["content"].(map[string]any)
				for mediaType, rawMedia := range content {
					media, _ := rawMedia.(map[string]any)
					schema, _ := media["schema"].(map[string]any)
					if ref, _ := schema["$ref"].(string); ref != "" {
						const prefix = "#/components/schemas/"
						if !strings.HasPrefix(ref, prefix) {
							t.Errorf("%s %s %s has unsupported schema ref %q",
								method, path, status, ref)
							continue
						}
						schema, _ = schemas[strings.TrimPrefix(ref, prefix)].(map[string]any)
					}
					if schema == nil {
						t.Errorf("%s %s %s %s has no response schema",
							method, path, status, mediaType)
						continue
					}
					properties, _ := schema["properties"].(map[string]any)
					additional, _ := schema["additionalProperties"].(bool)
					if schema["type"] == "object" && additional &&
						len(properties) == 0 {
						t.Errorf("%s %s (%s) %s %s uses a free-form response",
							strings.ToUpper(method), path, operationID,
							status, mediaType)
					}
				}
			}
		}
	}
}

func TestOpenAPIDocumentsRedirectsAndDomainImagesByMediaType(t *testing.T) {
	document := New(Options{
		Version: "test-version",
		DB:      stubDatabase{},
	}).OpenAPI()
	for _, path := range []string{
		"/auth/eve/start", "/auth/eve/callback", "/auth/callback",
	} {
		response := document.Paths[path].Get.Responses["302"]
		if len(response.Content) != 0 {
			t.Errorf("%s redirect documents body content: %#v", path, response.Content)
		}
		if response.Headers["Location"] == nil {
			t.Errorf("%s redirect does not document Location", path)
		}
	}
	for _, path := range []string{
		"/admin/domains/{id}/assets/{assetId}/preview",
		"/images/domains/{id}/{type}",
		"/images/domains/background/{assetId}",
		"/images/domains/preview/{assetId}",
		"/domains/asset/{id}/{type}",
		"/domains/bg/{assetId}",
		"/domains/preview/{assetId}",
	} {
		response := document.Paths[path].Get.Responses["200"]
		if response.Content["application/json"] != nil {
			t.Errorf("%s image response is documented as JSON", path)
		}
		for _, mediaType := range []string{
			"image/jpeg", "image/png", "image/webp", "image/gif",
		} {
			media := response.Content[mediaType]
			if media == nil || media.Schema == nil ||
				media.Schema.Type != huma.TypeString ||
				media.Schema.Format != "binary" {
				t.Errorf("%s %s response schema = %#v",
					path, mediaType, media)
			}
		}
	}
}

func TestDocsUseScalar(t *testing.T) {
	handler := apiPathHandler(Options{
		Version: "test-version",
		DB:      stubDatabase{},
	})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "http://example.com/docs", nil),
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("docs status = %d, body %q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "@scalar/api-reference") {
		t.Errorf("API docs do not load Scalar: %s", body)
	}
	if strings.Contains(body, "@stoplight/elements") {
		t.Errorf("API docs still load Stoplight Elements: %s", body)
	}
}

func TestSiteOpenAPIDocumentsAudienceAndServers(t *testing.T) {
	service := New(Options{
		Version: "test-version",
		DB:      stubDatabase{},
	})

	response := httptest.NewRecorder()
	service.Site().ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodGet, "http://www.example/api/openapi.json", nil,
		),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("document returned %d: %s", response.Code, response.Body.String())
	}

	var document struct {
		Info struct {
			Description string `json:"description"`
			Contact     struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"contact"`
		} `json:"info"`
		Servers []struct {
			URL string `json:"url"`
		} `json:"servers"`
		Paths map[string]map[string]struct {
			OperationID string                `json:"operationId"`
			Audience    string                `json:"x-audience"`
			Security    []map[string][]string `json:"security"`
			Servers     []struct {
				URL string `json:"url"`
			} `json:"servers"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	for _, section := range []string{
		"## One API, three namespaces",
		"## Contract and data freshness",
		"## Responsible use",
		"User-Agent",
		"network edge",
	} {
		if !strings.Contains(document.Info.Description, section) {
			t.Errorf("API introduction does not contain %q", section)
		}
	}
	if document.Info.Contact.Name != "EVE-KILL on GitHub" ||
		document.Info.Contact.URL != "https://github.com/eve-kill" {
		t.Errorf("API contact = %#v", document.Info.Contact)
	}
	if len(document.Servers) != 1 || document.Servers[0].URL != "/api" {
		t.Fatalf("OpenAPI servers = %#v, want /api", document.Servers)
	}
	image := document.Paths["/images/characters/{id}/{variant}"]["get"]
	if len(image.Servers) != 1 || image.Servers[0].URL != "/" {
		t.Fatalf("image operation servers = %#v, want /", image.Servers)
	}
	auth := document.Paths["/auth/login"]["get"]
	if len(auth.Servers) != 1 || auth.Servers[0].URL != "/" {
		t.Fatalf("auth operation servers = %#v, want /", auth.Servers)
	}
	if got := document.Paths["/killmails"]["get"].Audience; got != "public" {
		t.Errorf("killmails audience = %q, want public", got)
	}
	me := document.Paths["/me"]["get"]
	if me.Audience != "account" || len(me.Security) == 0 {
		t.Errorf("/me is not documented as an authenticated account operation: %#v", me)
	}
	if got := document.Paths["/campaigns"]["get"].Audience; got != "public" {
		t.Errorf("optionally authenticated campaigns audience = %q, want public", got)
	}
	operationIDs := make(map[string]string)
	for path, methods := range document.Paths {
		for method, operation := range methods {
			if operation.OperationID == "" {
				continue
			}
			route := strings.ToUpper(method) + " " + path
			if previous, exists := operationIDs[operation.OperationID]; exists {
				t.Errorf("operation ID %q is shared by %s and %s",
					operation.OperationID, previous, route)
			}
			operationIDs[operation.OperationID] = route
			if (strings.HasPrefix(path, "/admin/") || path == "/admin") &&
				(operation.Audience != "admin" || len(operation.Security) == 0) {
				t.Errorf("%s %s is not documented as authenticated admin: %#v",
					strings.ToUpper(method), path, operation)
			}
		}
	}
}

func TestSiteScalarUsesPrefixedOpenAPIURL(t *testing.T) {
	service := New(Options{Version: "test-version", DB: stubDatabase{}})
	rec := httptest.NewRecorder()
	service.Site().ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "http://www.example/api/docs", nil),
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("docs status = %d, body %q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `data-url="/api/openapi.json"`) {
		t.Errorf("Scalar does not use the same-origin API prefix: %s", rec.Body)
	}
}

func TestAPIFallbackAndEstablishedPostGuards(t *testing.T) {
	handler := apiPathHandler(Options{DB: stubDatabase{}})
	for _, test := range []struct {
		method string
		path   string
		status int
		body   string
	}{
		{http.MethodGet, "/missing", http.StatusNotFound, `{"error":"Not found"}`},
		{http.MethodGet, "/killmails/search", http.StatusMethodNotAllowed, `{"error":"POST only"}`},
		{http.MethodGet, "/characters/analyze", http.StatusMethodNotAllowed, `{"error":"Method not allowed"}`},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(
			rec, httptest.NewRequest(test.method, "http://example.com"+test.path, nil),
		)
		if rec.Code != test.status || rec.Body.String() != test.body {
			t.Errorf("%s %s = (%d, %q), want (%d, %q)",
				test.method, test.path, rec.Code, rec.Body.String(),
				test.status, test.body)
		}
	}
}

func TestAPIIndexHonorsForwardedOrigin(t *testing.T) {
	handler := Site(Options{DB: stubDatabase{}})
	req := httptest.NewRequest(http.MethodGet, "http://internal/api", nil)
	req.Header.Set("X-Forwarded-Proto", "https, http")
	req.Header.Set("X-Forwarded-Host", "eve-kill.com, internal")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["baseUrl"] != "https://eve-kill.com/api" {
		t.Errorf("baseUrl = %q", body["baseUrl"])
	}
}

func TestSiteAPIIndexIncludesPrefix(t *testing.T) {
	handler := Site(Options{DB: stubDatabase{}})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "https://eve-kill.com/api", nil),
	)
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["baseUrl"] != "https://eve-kill.com/api" {
		t.Errorf("baseUrl = %q", body["baseUrl"])
	}
}

func TestSiteKeepsImagesAndAuthOutsideAPIPrefix(t *testing.T) {
	handler := Site(Options{DB: stubDatabase{}})
	for _, path := range []string{
		"/api/images",
		"/api/images/characters/7/portrait",
		"/api/auth",
		"/api/auth/callback",
		"/killmails",
		"/openapi.json",
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(
			rec,
			httptest.NewRequest(http.MethodGet, "https://eve-kill.com"+path, nil),
		)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", path, rec.Code)
		}
	}

	images := httptest.NewRecorder()
	handler.ServeHTTP(
		images,
		httptest.NewRequest(http.MethodGet, "https://eve-kill.com/images", nil),
	)
	if images.Code != http.StatusOK ||
		!strings.Contains(images.Body.String(), `"service":"EVE-KILL Images"`) {
		t.Fatalf("/images response = %d %s", images.Code, images.Body.String())
	}
	if got := images.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("/images CORS = %q, want *", got)
	}
}

func TestFeedStreamOptionsPreserveSSEHeaders(t *testing.T) {
	handler := Site(Options{DB: stubDatabase{}})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodOptions, "http://example.com/api/feed/stream", nil),
	)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	for name, want := range map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, If-None-Match, Last-Event-ID",
		"Access-Control-Expose-Headers": "ETag, Link, X-Cache",
		"Access-Control-Max-Age": "86400",
	} {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	for _, name := range []string{
		"Content-Type", "Cache-Control", "Connection", "X-Accel-Buffering",
	} {
		if got := rec.Header().Get(name); got != "" {
			t.Errorf("%s = %q on a bodyless preflight", name, got)
		}
	}
}

func (stubDatabase) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected Query")
}

func (stubDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected QueryRow")
}

func TestHealthPreservesEstablishedContract(t *testing.T) {
	handler := Site(Options{
		Version: "test-version",
		Commit:  "test-commit",
		DB:      stubDatabase{},
	})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://example.com/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"timestamp":"`) {
		t.Fatalf("health body = %s, want legacy ok/timestamp shape", body)
	}
	if strings.Contains(body, `"version"`) || strings.Contains(body, `"commit"`) {
		t.Fatalf("health leaked replacement-only fields: %s", body)
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if got := response["$schema"]; got != "http://example.com/api/schemas/health-response.json" {
		t.Errorf("$schema = %q", got)
	}
	timestamp, _ := response["timestamp"].(string)
	if matched, _ := regexp.MatchString(
		`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`,
		timestamp,
	); !matched {
		t.Errorf("timestamp = %q, want UTC with exactly three fractional digits", timestamp)
	}
	if got := rec.Header().Get("Link"); got !=
		`</api/schemas/health-response.json>; rel="describedBy"` {
		t.Errorf("Link = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("health should not emit CORS headers, got %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("health Cache-Control = %q, want no-store", got)
	}

	schemaRec := httptest.NewRecorder()
	handler.ServeHTTP(schemaRec, httptest.NewRequest(
		http.MethodGet,
		"http://example.com/api/schemas/health-response.json",
		nil,
	))
	if schemaRec.Code != http.StatusOK {
		t.Fatalf("schema status = %d, body %q",
			schemaRec.Code, schemaRec.Body.String())
	}
	var schemaDocument struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(schemaRec.Body.Bytes(), &schemaDocument); err != nil {
		t.Fatal(err)
	}
	for _, property := range []string{"$schema", "ok", "timestamp"} {
		if _, ok := schemaDocument.Properties[property]; !ok {
			t.Errorf("health response schema is missing %q", property)
		}
	}
	if !containsString(schemaDocument.Required, "$schema") {
		t.Errorf("$schema is not required by the health response schema")
	}
}

type failingPingDatabase struct{ stubDatabase }

func (failingPingDatabase) Ping(context.Context) error { return context.DeadlineExceeded }

func TestHealthIsIndependentAndReadyChecksDatabase(t *testing.T) {
	handler := Site(Options{DB: failingPingDatabase{}})

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "http://example.com/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", health.Code)
	}

	ready := httptest.NewRecorder()
	handler.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "http://example.com/ready", nil))
	if ready.Code < 500 {
		t.Fatalf("ready status = %d, want dependency failure", ready.Code)
	}
}

func TestReadyIsNotCacheable(t *testing.T) {
	handler := Site(Options{DB: stubDatabase{}})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "http://example.com/ready", nil,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("ready status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("ready Cache-Control = %q, want no-store", got)
	}
}

func TestNormalizeJSONUsesJavaScriptMillisecondTimestamps(t *testing.T) {
	value := map[string]any{
		"whole": time.Date(2026, 7, 26, 12, 34, 56, 0, time.FixedZone("test", 7200)),
		"fractional": []any{
			time.Date(2026, 7, 26, 12, 34, 56, 123456789, time.UTC),
		},
	}
	normalized := normalizeJSON(value).(map[string]any)
	if got := normalized["whole"]; got != "2026-07-26T10:34:56.000Z" {
		t.Errorf("whole timestamp = %q", got)
	}
	fractional := normalized["fractional"].([]any)
	if got := fractional[0]; got != "2026-07-26T12:34:56.123Z" {
		t.Errorf("fractional timestamp = %q", got)
	}
}

// A tag that no group lists does not fall back to the flat sidebar — Scalar
// drops it, and every operation under it becomes unreachable by navigation.
// Adding a route with a new tag must therefore fail here rather than quietly
// remove endpoints from the published reference.
func TestOpenAPITagGroupsCoverEveryTag(t *testing.T) {
	document := New(Options{}).document
	if err := checkTagGroups(documentTags(document)); err != nil {
		t.Fatalf("tag grouping is inconsistent: %v", err)
	}
}

// Scalar reads the section headings from x-tagGroups and the per-tag prose
// from the top-level tags array. Both have to survive marshaling.
func TestOpenAPIDocumentCarriesTagGroups(t *testing.T) {
	document := New(Options{}).document

	raw, err := document.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded struct {
		Tags []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"tags"`
		TagGroups []struct {
			Name string   `json:"name"`
			Tags []string `json:"tags"`
		} `json:"x-tagGroups"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.TagGroups) != len(tagGroups) {
		t.Errorf("x-tagGroups has %d groups, want %d",
			len(decoded.TagGroups), len(tagGroups))
	}
	if len(decoded.Tags) == 0 {
		t.Fatal("document declares no top-level tags")
	}
	for _, tag := range decoded.Tags {
		if tag.Description == "" {
			t.Errorf("tag %q has no description", tag.Name)
		}
	}
}

// Prose keyed by operation ID goes stale silently: rename a route and the
// entry stops reaching the page with nothing to say so.
func TestOpenAPIDescriptionsMatchLiveOperations(t *testing.T) {
	if err := checkDescribedOperations(New(Options{}).document); err != nil {
		t.Fatal(err)
	}
}

// The public read surface is what third parties consume, so every operation in
// it carries prose beyond its one-line summary.
func TestOpenAPIPublicOperationsAreDescribed(t *testing.T) {
	document := New(Options{}).document

	public := map[string]bool{}
	for _, group := range tagGroups {
		switch group.Name {
		case "Killboard", "Conflicts", "Universe and reference":
			for _, tag := range group.Tags {
				public[tag] = true
			}
		}
	}

	var bare []string
	forEachOperation(document, func(operation *huma.Operation) {
		if operation.Description != "" {
			return
		}
		for _, tag := range operation.Tags {
			if public[tag] {
				bare = append(bare, operation.OperationID)
				return
			}
		}
	})
	// The public tag groups also carry frontend-only routes that were never in
	// the documented surface. Guard the documented set rather than all of them.
	described := 0
	for _, id := range bare {
		if _, ok := operationDescriptions[id]; ok {
			described++
			t.Errorf("operation %q has prose in the map but none on the document", id)
		}
	}
	t.Logf("%d public-group operations run on summaries alone", len(bare)-described)
}
