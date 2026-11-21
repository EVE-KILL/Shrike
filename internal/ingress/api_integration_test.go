package ingress

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/jackc/pgx/v5"
)

type healthyDB struct{}

func (healthyDB) Ping(context.Context) error {
	return nil
}

func (healthyDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected Query")
}

func (healthyDB) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected QueryRow")
}

// Exercise the shared Huma API through the HTTP listener. API schemas stay below /api
// while image operations are served directly below /images.
func TestSameOriginAPIAndImagesRouteThroughIngress(t *testing.T) {
	opts := api.Options{Version: "test-version", Commit: "test-commit", DB: healthyDB{}}
	apiService := api.New(opts)
	t.Cleanup(apiService.Close)
	base := startTestServer(t, testConfig(), map[string]http.Handler{
		SurfaceSameOrigin: apiService.Site(),
		SurfaceWS:         api.WS(opts),
		SurfaceFrontend:   http.NotFoundHandler(),
	})
	const host = "tenant.example.com"
	res := requestWithHost(t, base+"/health", host)
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want 200 (body %q)", res.StatusCode, body)
	}
	if !strings.Contains(string(body), `"ok":true`) ||
		!strings.Contains(string(body), `"timestamp":"`) {
		t.Errorf("health does not preserve shared contract: %s", body)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	schemaURL, err := url.Parse(payload["$schema"].(string))
	if err != nil {
		t.Fatal(err)
	}
	const schemaPath = "/api/schemas/health-response.json"
	if schemaURL.Path != schemaPath {
		t.Fatalf("$schema path = %q, want %q", schemaURL.Path, schemaPath)
	}
	schema := requestWithHost(t, base+schemaPath, host)
	schemaBody, _ := io.ReadAll(schema.Body)
	_ = schema.Body.Close()
	if schema.StatusCode != http.StatusOK ||
		!strings.Contains(string(schemaBody), `"$schema"`) {
		t.Fatalf("schema response = %d %s", schema.StatusCode, schemaBody)
	}

	images := requestWithHost(t, base+"/images", host)
	imageBody, _ := io.ReadAll(images.Body)
	_ = images.Body.Close()
	if images.StatusCode != http.StatusOK ||
		!strings.Contains(string(imageBody), `"service":"EVE-KILL Images"`) {
		t.Fatalf("images response = %d %s", images.StatusCode, imageBody)
	}
	var imagePayload map[string]any
	if err := json.Unmarshal(imageBody, &imagePayload); err != nil {
		t.Fatal(err)
	}
	imageSchemaURL, err := url.Parse(imagePayload["$schema"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(imageSchemaURL.Path, "/api/schemas/") ||
		!strings.HasSuffix(imageSchemaURL.Path, ".json") {
		t.Fatalf("image $schema path = %q, want /api/schemas/*.json", imageSchemaURL.Path)
	}
	imageSchema := requestWithHost(t, base+imageSchemaURL.Path, host)
	imageSchemaBody, _ := io.ReadAll(imageSchema.Body)
	_ = imageSchema.Body.Close()
	if imageSchema.StatusCode != http.StatusOK ||
		!strings.Contains(string(imageSchemaBody), `"$schema"`) {
		t.Fatalf(
			"image schema response = %d %s",
			imageSchema.StatusCode,
			imageSchemaBody,
		)
	}
	if got := images.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("image CORS header = %q, want *", got)
	}
}

func requestWithHost(t *testing.T, url, host string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = host
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s with Host %s: %v", url, host, err)
	}
	return res
}
