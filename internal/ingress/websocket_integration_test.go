package ingress

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	shrikewebsocket "github.com/eve-kill/shrike/internal/websocket"
	"github.com/rs/zerolog"
)

// Upgrade a real connection to verify that the access logger preserves Hijacker.
func TestWebSocketUpgradeThroughIngress(t *testing.T) {
	cfg := testConfig()
	cfg.LegacyDomain = "example.test"

	wsServer := shrikewebsocket.New(nil, zerolog.Nop())
	surfaces := testSurfaces()
	surfaces[SurfaceWS] = wsServer
	surfaces[SurfaceFrontend] = http.NotFoundHandler()
	t.Cleanup(wsServer.Close)
	base := startTestServer(t, cfg, surfaces)
	for _, path := range []string{"/comments", "/status"} {
		res, err := http.Get(base + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s status = %d, want frontend 404", path, res.StatusCode)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := coderws.Dial(ctx,
		"ws"+strings.TrimPrefix(base, "http")+"/ws/status", nil)
	if err != nil {
		t.Fatalf("websocket dial through ingress: %v", err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })

	welcome := readWSObject(t, conn)
	if welcome["type"] != "welcome" || welcome["endpoint"] != "/ws/status" {
		t.Fatalf("welcome = %#v", welcome)
	}

	writeCtx, writeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := conn.Write(writeCtx, coderws.MessageText, []byte(`{"action":"ping"}`)); err != nil {
		writeCancel()
		t.Fatalf("write ping: %v", err)
	}
	writeCancel()
	if pong := readWSObject(t, conn); pong["type"] != "pong" {
		t.Fatalf("pong = %#v", pong)
	}

	legacy, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(base, "http")+"/status", &coderws.DialOptions{Host: "ws.example.test"})
	if err != nil {
		t.Fatalf("legacy websocket dial through ingress: %v", err)
	}
	t.Cleanup(func() { _ = legacy.CloseNow() })
	if welcome := readWSObject(t, legacy); welcome["endpoint"] != "/ws/status" {
		t.Fatalf("legacy welcome = %#v", welcome)
	}
	legacyCtx, legacyCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer legacyCancel()
	if err := legacy.Write(legacyCtx, coderws.MessageText, []byte(`{"action":"ping"}`)); err != nil {
		t.Fatal(err)
	}
	if pong := readWSObject(t, legacy); pong["type"] != "pong" {
		t.Fatalf("legacy pong = %#v", pong)
	}
}

func readWSObject(t *testing.T, conn *coderws.Conn) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, body, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("websocket read: %v", err)
	}
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return value
}
