package frontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMCPDocumentationUsesRegisteredCatalog(t *testing.T) {
	view := mcpCatalogView()
	if view.Error != "" || len(view.Tools) == 0 {
		t.Fatalf("MCP catalog unavailable: %s", view.Error)
	}
	handler := New(Options{Production: true})
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, canonicalOrigin+"/mcp", nil))
	if page.Code != http.StatusOK || strings.Count(page.Body.String(), `class="mcp-tool"`) != len(view.Tools) {
		t.Fatalf("MCP page status=%d, tools=%d, want %d", page.Code, strings.Count(page.Body.String(), `class="mcp-tool"`), len(view.Tools))
	}
	if strings.Contains(page.Body.String(), `"additionalProperties"`) {
		t.Fatal("collapsed MCP page eagerly rendered output schemas")
	}

	var guide string
	for _, path := range []string{"/mcp.txt", "/llms.txt"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, canonicalOrigin+path, nil))
		if w.Code != http.StatusOK || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain") {
			t.Fatalf("%s: status=%d content type=%q", path, w.Code, w.Header().Get("Content-Type"))
		}
		if guide != "" && guide != w.Body.String() {
			t.Fatal("MCP guides have drifted")
		}
		guide = w.Body.String()
	}
	for _, tool := range view.Tools {
		if !strings.Contains(guide, "### "+tool.Name+"\n") {
			t.Errorf("MCP guide omitted %s", tool.Name)
		}
	}

	fragment := httptest.NewRecorder()
	handler.ServeHTTP(fragment, httptest.NewRequest(http.MethodGet, canonicalOrigin+"/_fragments/mcp/output/search", nil))
	if fragment.Code != http.StatusOK || !strings.Contains(fragment.Body.String(), "Output schema") || !strings.Contains(fragment.Body.String(), "<pre>") {
		t.Fatalf("schema fragment: %d %s", fragment.Code, fragment.Body.String())
	}
	unknown := httptest.NewRecorder()
	handler.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, canonicalOrigin+"/_fragments/mcp/output/unknown", nil))
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown schema status=%d", unknown.Code)
	}
}
