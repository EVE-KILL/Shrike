package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPTransportInitializesWithUserAgent(t *testing.T) {
	service := New(Options{
		Version:      "test",
		RequestGuard: NewRequestGuard(),
	})
	body := []byte(`{
		"jsonrpc":"2.0",
		"id":1,
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-06-18",
			"capabilities":{},
			"clientInfo":{"name":"api-test","version":"1"}
		}
	}`)
	request := httptest.NewRequest(
		http.MethodPost,
		"http://eve-kill.test/api/mcp",
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("MCP-Protocol-Version", "2025-06-18")
	request.Header.Set("User-Agent", "evekill-mcp-test/1.0")
	response := httptest.NewRecorder()

	service.Site().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Result struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error != nil {
		t.Fatalf("MCP error: %#v", payload.Error)
	}
	if payload.Result.ServerInfo.Name != "evekill-mcp" {
		t.Fatalf("server name = %q", payload.Result.ServerInfo.Name)
	}
}
