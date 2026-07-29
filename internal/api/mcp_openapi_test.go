package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAPIDocumentsEveryTypedMCPTool(t *testing.T) {
	t.Parallel()
	document := New(Options{Version: "test"}).OpenAPI()
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal OpenAPI: %v", err)
	}
	var decoded struct {
		Paths map[string]map[string]struct {
			Tags        []string `json:"tags"`
			RequestBody *struct {
				Content map[string]struct {
					Schema any `json:"schema"`
				} `json:"content"`
			} `json:"requestBody"`
			Responses map[string]struct {
				Content map[string]struct {
					Schema any `json:"schema"`
				} `json:"content"`
			} `json:"responses"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode OpenAPI: %v", err)
	}
	count := 0
	for path, methods := range decoded.Paths {
		if !strings.HasPrefix(path, "/mcp/tools/") {
			continue
		}
		count++
		operation, ok := methods["post"]
		if !ok {
			t.Errorf("%s has no POST operation", path)
			continue
		}
		if len(operation.Tags) != 1 || operation.Tags[0] != "mcp" {
			t.Errorf("%s tags = %v, want [mcp]", path, operation.Tags)
		}
		if operation.RequestBody == nil ||
			operation.RequestBody.Content["application/json"].Schema == nil {
			t.Errorf("%s has no typed JSON request", path)
		}
		if operation.Responses["200"].Content["application/json"].Schema == nil {
			t.Errorf("%s has no typed JSON response", path)
		}
	}
	if count != 45 {
		t.Fatalf("typed MCP operation count = %d, want 45", count)
	}
}
