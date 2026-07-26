package ingress

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	shrikewebsocket "github.com/eve-kill/shrike/internal/websocket"
	"github.com/rs/zerolog"
)

// Exercise a real WebSocket upgrade through embedded Caddy. Marker handlers
// prove ordinary route ownership, but only an actual handshake proves Caddy's
// response-writer wrappers preserve Hijacker all the way to Shrike's handler.
func TestWebSocketUpgradeThroughCaddy(t *testing.T) {
	port := freePort(t)
	cfg := testConfig()
	cfg.Address = fmt.Sprintf("127.0.0.1:%d", port)

	wsServer := shrikewebsocket.New(nil, zerolog.Nop())
	surfaces := testSurfaces()
	surfaces[SurfaceWS] = wsServer
	manager := New(surfaces, zerolog.Nop())
	if err := manager.Start(context.Background(), cfg); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		_ = manager.Close()
		wsServer.Close()
	})

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	for _, path := range []string{"/comments", "/status"} {
		res, err := http.Get(base + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s status = %d, want Nuxt fallback 404", path, res.StatusCode)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := coderws.Dial(ctx,
		"ws"+strings.TrimPrefix(base, "http")+"/ws/status", nil)
	if err != nil {
		t.Fatalf("websocket dial through Caddy: %v", err)
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
