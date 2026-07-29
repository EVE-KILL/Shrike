package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerListsEveryTypedTool(t *testing.T) {
	t.Parallel()
	server, err := NewServer(Dependencies{}, "test", nil)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name: "shrike-test", Version: "test",
	}, nil)
	clientSession, err := client.Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })
	result, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	want := []string{
		"battle_report", "capsuleer_dossier", "character_history",
		"coalition_graph", "compare", "doctrine_detect", "dogma_eval",
		"entity_kills", "entity_overview", "entity_timeline", "entity_top",
		"expensive_losses", "find_battles", "fit_compare", "flies_with",
		"global_pulse", "hunted_by", "hunts_in", "item_info", "killmail",
		"killmail_fitting", "killmail_forensics", "killmail_story", "kills_with",
		"me_dossier", "me_flies_with", "me_hunted_by", "me_hunts_in",
		"me_kills", "me_kills_with", "me_overview", "me_preys_on",
		"me_ships_used", "me_timeline", "meta_pulse", "pilot_efficiency",
		"preys_on", "route_danger", "search", "ship_compare", "ship_info",
		"ships_used", "system_info", "system_pulse", "war_report",
	}
	got := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		got = append(got, tool.Name)
		if tool.InputSchema == nil {
			t.Errorf("%s has no input schema", tool.Name)
		}
		if tool.OutputSchema == nil {
			t.Errorf("%s has no output schema", tool.Name)
		}
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("%s is not marked read-only", tool.Name)
		}
	}
	sort.Strings(got)
	if len(got) != len(want) {
		t.Fatalf("tool count = %d, want %d\n got: %v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("tool %d = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestDogmaBridgeEvaluatesFit(t *testing.T) {
	hull, err := evaluateDogma(t.Context(), EsfFit{
		ShipTypeID: 587,
		Modules:    []EsfModule{},
		Drones:     []EsfDrone{},
	}, false)
	if err != nil {
		t.Fatalf("evaluateDogma: %v", err)
	}
	if hull.EHP == nil || *hull.EHP <= 0 {
		t.Fatalf("EHP = %v, want a positive value", hull.EHP)
	}
	if hull.AlignTime == nil || *hull.AlignTime <= 0 {
		t.Fatalf("align time = %v, want a positive value", hull.AlignTime)
	}
}

func TestParseEFTSeparatesSubsystemsAndDrones(t *testing.T) {
	t.Parallel()
	parsed, err := parseEFT(`[Tengu, Test]
Damage Control II

10MN Afterburner II

Heavy Missile Launcher II, Scourge Heavy Missile

Medium Core Defense Field Extender I

Tengu Defensive - Covert Reconfiguration

Hornet II x5`)
	if err != nil {
		t.Fatalf("parseEFT: %v", err)
	}
	if len(parsed.blocks) != 5 {
		t.Fatalf("block count = %d, want 5", len(parsed.blocks))
	}
	if got := parsed.blocks[4]; len(got) != 1 ||
		got[0] != "Tengu Defensive - Covert Reconfiguration" {
		t.Fatalf("subsystems = %v", got)
	}
	if len(parsed.drones) != 1 ||
		parsed.drones[0].name != "Hornet II" ||
		parsed.drones[0].quantity != 5 {
		t.Fatalf("drones = %v", parsed.drones)
	}
}

func TestStreamableHTTPListsToolsWithoutSessionState(t *testing.T) {
	handler, err := Handler(Dependencies{}, "test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Mcp-Protocol-Version", "2025-06-18")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, body = %s", response.StatusCode, data)
	}
	var payload struct {
		Result struct {
			Tools []map[string]any `json:"tools"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error != nil {
		t.Fatalf("JSON-RPC error: %#v", payload.Error)
	}
	if len(payload.Result.Tools) != 45 {
		t.Fatalf("tool count = %d, want 45", len(payload.Result.Tools))
	}
	for _, tool := range payload.Result.Tools {
		if tool["inputSchema"] == nil || tool["outputSchema"] == nil {
			t.Errorf("%v is missing an input or output schema", tool["name"])
		}
	}
}
