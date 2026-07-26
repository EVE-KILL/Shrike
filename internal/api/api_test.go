package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type stubDatabase struct{}

func (stubDatabase) Ping(context.Context) error {
	return nil
}

func TestEstablishedRouteCatalogueMatchesEmbeddedIndex(t *testing.T) {
	handler := APIHost(Options{
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

func TestAPIHostDocsUseScalar(t *testing.T) {
	handler := APIHost(Options{
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

func TestTransportsShareOneOpenAPIDocument(t *testing.T) {
	service := New(Options{
		Version: "test-version",
		DB:      stubDatabase{},
	})

	apiHost := httptest.NewRecorder()
	service.APIHost().ServeHTTP(
		apiHost,
		httptest.NewRequest(http.MethodGet, "http://api.example/openapi.json", nil),
	)
	sameOrigin := httptest.NewRecorder()
	service.SameOrigin().ServeHTTP(
		sameOrigin,
		httptest.NewRequest(
			http.MethodGet, "http://www.example/api/openapi.json", nil,
		),
	)
	if apiHost.Code != http.StatusOK || sameOrigin.Code != http.StatusOK {
		t.Fatalf("documents returned %d and %d", apiHost.Code, sameOrigin.Code)
	}
	if !bytes.Equal(apiHost.Body.Bytes(), sameOrigin.Body.Bytes()) {
		t.Fatal("API-host and same-origin transports returned different OpenAPI documents")
	}

	var document struct {
		Paths map[string]map[string]struct {
			OperationID string                `json:"operationId"`
			Audience    string                `json:"x-audience"`
			Security    []map[string][]string `json:"security"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(apiHost.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
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

func TestSameOriginScalarUsesPrefixedOpenAPIURL(t *testing.T) {
	service := New(Options{Version: "test-version", DB: stubDatabase{}})
	rec := httptest.NewRecorder()
	service.SameOrigin().ServeHTTP(
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
	handler := APIHost(Options{DB: stubDatabase{}})
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
	handler := APIHost(Options{DB: stubDatabase{}})
	req := httptest.NewRequest(http.MethodGet, "http://internal/", nil)
	req.Header.Set("X-Forwarded-Proto", "https, http")
	req.Header.Set("X-Forwarded-Host", "api.eve-kill.com, internal")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["baseUrl"] != "https://api.eve-kill.com" {
		t.Errorf("baseUrl = %q", body["baseUrl"])
	}
}

func TestSameOriginAPIIndexIncludesPrefix(t *testing.T) {
	handler := SameOrigin(Options{DB: stubDatabase{}})
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

func TestFeedStreamOptionsPreserveSSEHeaders(t *testing.T) {
	handler := APIHost(Options{DB: stubDatabase{}})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodOptions, "http://example.com/feed/stream", nil),
	)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	for name, want := range map[string]string{
		"Access-Control-Allow-Origin":   "*",
		"Access-Control-Allow-Methods":  "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS",
		"Access-Control-Allow-Headers":  "Content-Type, If-None-Match, Last-Event-ID",
		"Access-Control-Expose-Headers": "ETag, Link, X-Cache",
		"Access-Control-Max-Age":        "86400",
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
	handler := APIHost(Options{
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
	if got := response["$schema"]; got != "http://example.com/schemas/health-response.json" {
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
		`</schemas/health-response.json>; rel="describedBy"` {
		t.Errorf("Link = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}

	schemaRec := httptest.NewRecorder()
	handler.ServeHTTP(schemaRec, httptest.NewRequest(
		http.MethodGet,
		"http://example.com/schemas/health-response.json",
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

func TestSameOriginHealthUsesSharedContractAndPrefixedSchema(t *testing.T) {
	handler := SameOrigin(Options{
		Version: "test-version",
		Commit:  "test-commit",
		DB:      stubDatabase{},
	})

	health := httptest.NewRecorder()
	handler.ServeHTTP(
		health,
		httptest.NewRequest(http.MethodGet, "http://example.com/health", nil),
	)
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200 (body %q)", health.Code, health.Body.String())
	}
	if got := health.Header().Get("Link"); got !=
		`</api/schemas/health-response.json>; rel="describedBy"` {
		t.Fatalf("health Link = %q, want same-origin schema path", got)
	}
	if body := health.Body.String(); !strings.Contains(body, `"ok":true`) ||
		!strings.Contains(body, `"timestamp":"`) {
		t.Errorf("health body does not use shared contract: %s", body)
	}

	schema := httptest.NewRecorder()
	handler.ServeHTTP(
		schema,
		httptest.NewRequest(
			http.MethodGet,
			"http://example.com/api/schemas/health-response.json",
			nil,
		),
	)
	if schema.Code != http.StatusOK {
		t.Fatalf("schema status = %d, want 200 (body %q)", schema.Code, schema.Body.String())
	}
}
