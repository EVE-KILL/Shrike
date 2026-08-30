package websocket

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func TestDiscoveryContainsProtocolButNoOperationalStats(t *testing.T) {
	server := New(nil, zerolog.Nop())
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	t.Cleanup(server.Close)

	res, err := http.Get(httpServer.URL + BasePath)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("discovery status = %d, want 200 (body %q)", res.StatusCode, body)
	}
	for _, want := range []string{`"/ws/killlist"`, `"subscribe"`, `"topicRouting"`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("discovery does not contain %s: %s", want, body)
		}
	}
	for _, unwanted := range []string{`"connections"`, `"redis"`, `"uptime"`} {
		if strings.Contains(string(body), unwanted) {
			t.Errorf("discovery exposes operational field %s: %s", unwanted, body)
		}
	}

	for _, path := range []string{"/ws/health", "/ws/stats", "/killlist"} {
		res, err := http.Get(httpServer.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", path, res.StatusCode)
		}
	}
}

func TestKilllistTopicRoutingAndProtocol(t *testing.T) {
	server := New(nil, zerolog.Nop())
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	t.Cleanup(server.Close)

	conn := dial(t, httpServer.URL, "/ws/killlist")
	welcome := readObject(t, conn)
	if welcome["type"] != "welcome" || welcome["endpoint"] != "/ws/killlist" {
		t.Fatalf("welcome = %#v", welcome)
	}

	writeObject(t, conn, map[string]any{
		"action": "subscribe",
		"topics": []any{"solo", 42, "10b"},
	})
	subscribed := readObject(t, conn)
	if subscribed["type"] != "subscribed" {
		t.Fatalf("subscribe response = %#v", subscribed)
	}
	assertStrings(t, subscribed["topics"], []string{"solo", "10b"})

	// The first event has no matching key and must not occupy the connection's
	// outgoing queue. The matching event should therefore be the next frame.
	server.handleRedisMessage("ws:killlist", []byte(
		`{"routing_keys":["npc"],"data":{"killmail":{"killmail_id":1}}}`,
	))
	server.handleRedisMessage("ws:killlist", []byte(
		`{"routing_keys":["solo"],"data":{"killmail":{"killmail_id":2}}}`,
	))
	event := readObject(t, conn)
	if event["channel"] != "killlist" {
		t.Fatalf("event channel = %#v, want killlist", event["channel"])
	}
	data := event["data"].(map[string]any)
	killmail := data["killmail"].(map[string]any)
	if killmail["killmail_id"] != float64(2) {
		t.Fatalf("killmail = %#v, want id 2", killmail)
	}

	writeObject(t, conn, map[string]any{
		"action": "unsubscribe",
		"topics": []string{"solo"},
	})
	unsubscribed := readObject(t, conn)
	if unsubscribed["type"] != "unsubscribed" {
		t.Fatalf("unsubscribe response = %#v", unsubscribed)
	}

	server.handleRedisMessage("ws:killlist", []byte(
		`{"routing_keys":["solo"],"data":{"killmail":{"killmail_id":3}}}`,
	))
	writeObject(t, conn, map[string]any{"action": "ping"})
	pong := readObject(t, conn)
	if pong["type"] != "pong" {
		t.Fatalf("frame after unsubscribed event = %#v, want pong", pong)
	}
	if _, ok := pong["timestamp"].(float64); !ok {
		t.Fatalf("pong timestamp = %#v, want milliseconds", pong["timestamp"])
	}
}

func TestNonTopicEndpointReceivesEveryEvent(t *testing.T) {
	server := New(nil, zerolog.Nop())
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	t.Cleanup(server.Close)

	conn := dial(t, httpServer.URL, "/ws/comments")
	_ = readObject(t, conn) // welcome

	server.handleRedisMessage("ws:comments", []byte(
		`{"routing_keys":[],"data":{"event_type":"new","comment_id":7}}`,
	))
	event := readObject(t, conn)
	if event["channel"] != "comments" {
		t.Fatalf("event = %#v", event)
	}
	data := event["data"].(map[string]any)
	if data["comment_id"] != float64(7) {
		t.Fatalf("comment event = %#v", data)
	}

	stats := server.Snapshot()
	if stats.Connections != 1 {
		t.Fatalf("connections = %d, want 1", stats.Connections)
	}
	if stats.Endpoints["/ws/comments"].Subscriptions["all"] != 1 {
		t.Fatalf("comment stats = %#v", stats.Endpoints["/ws/comments"])
	}
}

func TestNullDataFallsBackToOriginalEnvelope(t *testing.T) {
	server := New(nil, zerolog.Nop())
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	t.Cleanup(server.Close)

	conn := dial(t, httpServer.URL, "/ws/status")
	_ = readObject(t, conn)

	server.handleRedisMessage("ws:status", []byte(
		`{"routing_keys":["all"],"data":null,"event":"fallback"}`,
	))
	event := readObject(t, conn)
	data := event["data"].(map[string]any)
	if data["event"] != "fallback" {
		t.Fatalf("fallback data = %#v", data)
	}
}

func TestRedisSubscriptionFeedsConnectionsAndStops(t *testing.T) {
	fake := &fakeSubscription{
		messages: make(chan *redis.Message, 1),
		closed:   make(chan struct{}),
	}
	server := New(nil, zerolog.Nop())
	var subscribed []string
	server.openSubscription = func(_ context.Context, channels ...string) subscription {
		subscribed = append([]string(nil), channels...)
		return fake
	}
	ctx, cancel := context.WithCancel(context.Background())
	server.Start(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for !server.Snapshot().Redis && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !server.Snapshot().Redis {
		t.Fatal("subscription did not become ready")
	}
	if len(subscribed) != len(endpoints) {
		t.Fatalf("subscribed channels = %#v, want %d", subscribed, len(endpoints))
	}

	httpServer := httptest.NewServer(server)
	conn := dial(t, httpServer.URL, "/ws/status")
	_ = readObject(t, conn)
	fake.messages <- &redis.Message{
		Channel: "ws:status",
		Payload: `{"routing_keys":["all"],"data":{"event":"status"}}`,
	}
	event := readObject(t, conn)
	if event["channel"] != "status" {
		t.Fatalf("event = %#v", event)
	}
	_ = conn.Close(coderws.StatusNormalClosure, "")
	httpServer.Close()

	cancel()
	server.Close()
	select {
	case <-fake.closed:
	default:
		t.Fatal("subscription was not closed")
	}
}

func TestClientQueueIsBounded(t *testing.T) {
	ctx := t.Context()
	c := newClient(ctx, nil, endpoints[0])
	for range clientQueueSize {
		if !c.enqueue([]byte("event")) {
			t.Fatal("queue rejected an event before reaching its bound")
		}
	}
	if c.enqueue([]byte("overflow")) {
		t.Fatal("queue accepted an event beyond its bound")
	}
}

func TestCloseDisconnectsClientsWithGoingAway(t *testing.T) {
	server := New(nil, zerolog.Nop())
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)

	conn := dial(t, httpServer.URL, "/ws/status")
	_ = readObject(t, conn)

	done := make(chan struct{})
	go func() {
		server.Close()
		close(done)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := conn.Read(ctx)
	if got := coderws.CloseStatus(err); got != coderws.StatusGoingAway {
		t.Fatalf("close status = %d, want %d (err %v)", got, coderws.StatusGoingAway, err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server Close did not finish after the client acknowledged shutdown")
	}
}

func dial(t *testing.T, httpURL, path string) *coderws.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(httpURL, "http") + path
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := coderws.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", path, err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	return conn
}

func writeObject(t *testing.T, conn *coderws.Conn, value any) {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := conn.Write(ctx, coderws.MessageText, body); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readObject(t *testing.T, conn *coderws.Conn) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, body, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return value
}

func assertStrings(t *testing.T, raw any, want []string) {
	t.Helper()
	values, ok := raw.([]any)
	if !ok {
		t.Fatalf("value = %#v, want string array", raw)
	}
	if len(values) != len(want) {
		t.Fatalf("value = %#v, want %#v", raw, want)
	}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("value = %#v, want %#v", raw, want)
		}
	}
}

type fakeSubscription struct {
	messages chan *redis.Message
	closed   chan struct{}
	once     sync.Once
}

func (f *fakeSubscription) Receive(context.Context) (any, error) {
	return &redis.Subscription{Kind: "subscribe"}, nil
}

func (f *fakeSubscription) ReceiveMessage(ctx context.Context) (*redis.Message, error) {
	select {
	case message := <-f.messages:
		return message, nil
	case <-f.closed:
		return nil, context.Canceled
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (f *fakeSubscription) Close() error {
	f.once.Do(func() { close(f.closed) })
	return nil
}
