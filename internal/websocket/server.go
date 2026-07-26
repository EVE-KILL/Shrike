// Package websocket serves EVE-KILL's live event streams.
//
// Workers publish small envelopes to Redis channels prefixed with ws:. This
// server subscribes once and fans each event out to browser connections on
// /ws/*. Killmail streams are topic-routed; comments, status and announcements
// are broadcast to every connection on their endpoint.
//
// Redis pub/sub is intentionally best-effort. The durable data is already in
// Postgres, and the feed API has its own sequence-backed recovery path. A relay
// outage may miss a live notification but must never retry killmail processing.
package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

const (
	clientQueueSize = 256
	maxMessageBytes = 16 << 10
	maxTopics       = 128
	maxTopicBytes   = 128
	pingInterval    = 30 * time.Second
	writeTimeout    = 10 * time.Second
	reconnectDelay  = time.Second
)

// Server owns the Redis subscription and every connected WebSocket.
//
// Start and Close are idempotent. A Server may be constructed before Caddy is
// started, but Start should be called before the handler becomes reachable so
// the Redis subscription is already attempting to connect.
type Server struct {
	openSubscription func(context.Context, ...string) subscription
	log              zerolog.Logger

	startedAt time.Time
	ctx       context.Context
	cancel    context.CancelFunc
	startOnce sync.Once
	closeOnce sync.Once
	wg        sync.WaitGroup

	mu      sync.RWMutex
	clients map[*client]struct{}

	pubsubMu sync.Mutex
	pubsub   subscription

	redisReady atomic.Bool
	closing    atomic.Bool
}

// New builds a relay over the coordination Redis instance.
func New(redisClient *redis.Client, log zerolog.Logger) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		log:       log,
		startedAt: time.Now().UTC(),
		ctx:       ctx,
		cancel:    cancel,
		clients:   make(map[*client]struct{}),
	}
	if redisClient != nil {
		server.openSubscription = func(ctx context.Context, channels ...string) subscription {
			return redisClient.Subscribe(ctx, channels...)
		}
	}
	return server
}

// Start begins the Redis subscription and binds its lifetime to ctx.
func (s *Server) Start(ctx context.Context) {
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			select {
			case <-ctx.Done():
				s.shutdown()
			case <-s.ctx.Done():
			}
		}()

		if s.openSubscription != nil {
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				s.subscribe()
			}()
		}
	})
}

// Close stops the subscription and closes active clients.
func (s *Server) Close() {
	s.shutdown()
	s.wg.Wait()
}

func (s *Server) shutdown() {
	s.closeOnce.Do(func() {
		s.closing.Store(true)
		s.redisReady.Store(false)

		s.mu.RLock()
		clients := make([]*client, 0, len(s.clients))
		for c := range s.clients {
			clients = append(clients, c)
		}
		s.mu.RUnlock()

		var closes sync.WaitGroup
		closes.Add(len(clients))
		for _, c := range clients {
			go func() {
				defer closes.Done()
				c.close(coderws.StatusGoingAway, "server shutting down")
			}()
		}
		closes.Wait()

		s.cancel()
		s.pubsubMu.Lock()
		if s.pubsub != nil {
			_ = s.pubsub.Close()
		}
		s.pubsubMu.Unlock()
	})
}

// ServeHTTP serves only the /ws contract.
//
// There is deliberately no /ws/health or /ws/stats. Process liveness already
// lives at /health, while operational telemetry is available through Snapshot
// for later publication into the shared Redis telemetry stream.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setCORS(w.Header())
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.URL.Path == BasePath || r.URL.Path == BasePath+"/" {
		s.serveDiscovery(w, r)
		return
	}

	endpoint := endpointByPath[r.URL.Path]
	if endpoint == nil {
		http.NotFound(w, r)
		return
	}
	s.serveSocket(w, r, endpoint)
}

func (s *Server) serveSocket(w http.ResponseWriter, r *http.Request, endpoint *Endpoint) {
	if s.closing.Load() {
		http.Error(w, "server shutting down", http.StatusServiceUnavailable)
		return
	}
	conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
		// The streams are public and unauthenticated, as the TypeScript relay
		// was. Origin checks would break non-browser consumers and third-party
		// sites without protecting any credential-bearing operation.
		InsecureSkipVerify: true,
		CompressionMode:    coderws.CompressionDisabled,
	})
	if err != nil {
		return
	}
	conn.SetReadLimit(maxMessageBytes)

	c := newClient(s.ctx, conn, endpoint)
	s.register(c)
	defer func() {
		s.unregister(c)
		c.closeNow()
	}()

	c.enqueue(mustJSON(welcomeFrame(endpoint)))
	go c.writeLoop()
	c.readLoop()
}

func (s *Server) register(c *client) {
	s.mu.Lock()
	s.clients[c] = struct{}{}
	s.mu.Unlock()
}

func (s *Server) unregister(c *client) {
	s.mu.Lock()
	delete(s.clients, c)
	s.mu.Unlock()
	c.cancel()
}

func (s *Server) subscribe() {
	channels := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		channels = append(channels, endpoint.Channel)
	}

	for s.ctx.Err() == nil {
		pubsub := s.openSubscription(s.ctx, channels...)
		s.pubsubMu.Lock()
		s.pubsub = pubsub
		s.pubsubMu.Unlock()

		if _, err := pubsub.Receive(s.ctx); err == nil {
			s.redisReady.Store(true)
			s.log.Info().Strs("channels", channels).Msg("websocket relay subscribed")
			for s.ctx.Err() == nil {
				message, err := pubsub.ReceiveMessage(s.ctx)
				if err != nil {
					if s.ctx.Err() == nil {
						s.log.Warn().Err(err).Msg("websocket Redis subscription interrupted")
					}
					break
				}
				s.redisReady.Store(true)
				s.handleRedisMessage(message.Channel, []byte(message.Payload))
			}
		} else if s.ctx.Err() == nil {
			s.log.Warn().Err(err).Msg("websocket Redis subscription failed")
		}

		s.redisReady.Store(false)
		_ = pubsub.Close()
		s.pubsubMu.Lock()
		if s.pubsub == pubsub {
			s.pubsub = nil
		}
		s.pubsubMu.Unlock()

		if !waitContext(s.ctx, reconnectDelay) {
			return
		}
	}
}

type subscription interface {
	Receive(context.Context) (any, error)
	ReceiveMessage(context.Context) (*redis.Message, error)
	Close() error
}

func (s *Server) handleRedisMessage(channel string, raw []byte) {
	endpoint := endpointByChannel[channel]
	if endpoint == nil || !json.Valid(raw) {
		if endpoint != nil {
			s.log.Warn().Str("channel", channel).Msg("discarding invalid websocket relay JSON")
		}
		return
	}

	var envelope struct {
		RoutingKeys []string        `json:"routing_keys"`
		Data        json.RawMessage `json:"data"`
	}
	// Non-object JSON is legal in the old relay: it has no data property, so
	// the entire value is forwarded. Treat a shape mismatch as that fallback,
	// while malformed JSON above is dropped.
	_ = json.Unmarshal(raw, &envelope)

	data := json.RawMessage(raw)
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		data = envelope.Data
	}
	payload, err := json.Marshal(struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}{
		Channel: strings.TrimPrefix(endpoint.Path, "/"),
		Data:    data,
	})
	if err != nil {
		s.log.Warn().Err(err).Str("channel", channel).Msg("encoding websocket frame failed")
		return
	}

	s.broadcast(endpoint, envelope.RoutingKeys, payload)
}

func (s *Server) broadcast(endpoint *Endpoint, routingKeys []string, payload []byte) {
	s.mu.RLock()
	clients := make([]*client, 0, len(s.clients))
	for c := range s.clients {
		if c.endpoint == endpoint && c.matches(routingKeys) {
			clients = append(clients, c)
		}
	}
	s.mu.RUnlock()

	for _, c := range clients {
		if !c.enqueue(payload) {
			go c.close(coderws.StatusTryAgainLater, "client is too slow")
		}
	}
}

func (s *Server) serveDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type endpointDocument struct {
		Name         string `json:"name"`
		URL          string `json:"url"`
		TopicRouting bool   `json:"topicRouting"`
	}
	documented := make(map[string]endpointDocument, len(endpoints))
	for _, endpoint := range endpoints {
		path := BasePath + endpoint.Path
		documented[path] = endpointDocument{
			Name:         endpoint.Name,
			URL:          websocketURL(r, path),
			TopicRouting: endpoint.TopicRouting,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name":      "EVE-KILL WebSocket Server",
		"endpoints": documented,
		"topics":    availableTopics,
		"protocol": map[string]any{
			"subscribe":   map[string]any{"action": "subscribe", "topics": []string{"solo", "10b"}},
			"unsubscribe": map[string]any{"action": "unsubscribe", "topics": []string{"all"}},
		},
		"website": "https://eve-kill.com",
	})
}

func websocketURL(r *http.Request, path string) string {
	scheme := "ws"
	if r.TLS != nil || firstHeaderValue(r.Header.Get("X-Forwarded-Proto")) == "https" {
		scheme = "wss"
	}
	host := firstHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host + path
}

func firstHeaderValue(value string) string {
	if first, _, ok := strings.Cut(value, ","); ok {
		value = first
	}
	return strings.TrimSpace(value)
}

func setCORS(header http.Header) {
	header.Set("Access-Control-Allow-Origin", "*")
	header.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	header.Set("Access-Control-Allow-Headers", "Content-Type")
}

func welcomeFrame(endpoint *Endpoint) map[string]any {
	frame := map[string]any{
		"type":     "welcome",
		"endpoint": BasePath + endpoint.Path,
		"name":     endpoint.Name,
	}
	if endpoint.TopicRouting {
		frame["message"] = `Send {"action":"subscribe","topics":["all"]} to start receiving data.`
		frame["availableTopics"] = availableTopics
	} else {
		frame["message"] = "Connected. You will receive all events on this endpoint."
	}
	return frame
}

func mustJSON(value any) []byte {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return body
}

func waitContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type client struct {
	conn     *coderws.Conn
	endpoint *Endpoint

	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
	out       chan []byte

	topicsMu sync.RWMutex
	topics   map[string]struct{}
	order    []string
}

func newClient(parent context.Context, conn *coderws.Conn, endpoint *Endpoint) *client {
	ctx, cancel := context.WithCancel(parent)
	c := &client{
		conn:     conn,
		endpoint: endpoint,
		ctx:      ctx,
		cancel:   cancel,
		out:      make(chan []byte, clientQueueSize),
		topics:   make(map[string]struct{}),
	}
	if !endpoint.TopicRouting {
		c.topics["all"] = struct{}{}
		c.order = append(c.order, "all")
	}
	return c
}

func (c *client) readLoop() {
	for {
		_, body, err := c.conn.Read(c.ctx)
		if err != nil {
			return
		}
		var command struct {
			Action string `json:"action"`
			Topics []any  `json:"topics"`
		}
		if json.Unmarshal(body, &command) != nil {
			continue
		}

		switch command.Action {
		case "subscribe":
			if command.Topics == nil {
				continue
			}
			topics, ok := c.subscribe(command.Topics)
			if !ok {
				c.close(coderws.StatusPolicyViolation, "too many or invalid topics")
				return
			}
			c.enqueue(mustJSON(map[string]any{"type": "subscribed", "topics": topics}))
		case "unsubscribe":
			if command.Topics == nil {
				continue
			}
			removed := c.unsubscribe(command.Topics)
			c.enqueue(mustJSON(map[string]any{"type": "unsubscribed", "topics": removed}))
		case "ping":
			c.enqueue(mustJSON(map[string]any{"type": "pong", "timestamp": time.Now().UnixMilli()}))
		}
	}
}

func (c *client) writeLoop() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case payload := <-c.out:
			ctx, cancel := context.WithTimeout(c.ctx, writeTimeout)
			err := c.conn.Write(ctx, coderws.MessageText, payload)
			cancel()
			if err != nil {
				c.cancel()
				return
			}
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(c.ctx, writeTimeout)
			err := c.conn.Ping(ctx)
			cancel()
			if err != nil {
				c.cancel()
				return
			}
		}
	}
}

func (c *client) enqueue(payload []byte) bool {
	select {
	case <-c.ctx.Done():
		return false
	default:
	}
	select {
	case c.out <- payload:
		return true
	default:
		return false
	}
}

func (c *client) subscribe(values []any) ([]string, bool) {
	c.topicsMu.Lock()
	defer c.topicsMu.Unlock()

	for _, raw := range values {
		topic, ok := raw.(string)
		if !ok {
			continue
		}
		if topic == "" || len(topic) > maxTopicBytes {
			return nil, false
		}
		if _, exists := c.topics[topic]; exists {
			continue
		}
		if len(c.topics) >= maxTopics {
			return nil, false
		}
		c.topics[topic] = struct{}{}
		c.order = append(c.order, topic)
	}
	return append([]string(nil), c.order...), true
}

func (c *client) unsubscribe(values []any) []string {
	c.topicsMu.Lock()
	defer c.topicsMu.Unlock()

	var removed []string
	remove := make(map[string]struct{}, len(values))
	for _, raw := range values {
		topic, ok := raw.(string)
		if !ok {
			continue
		}
		delete(c.topics, topic)
		remove[topic] = struct{}{}
		removed = append(removed, topic)
	}
	kept := c.order[:0]
	for _, topic := range c.order {
		if _, deleted := remove[topic]; !deleted {
			kept = append(kept, topic)
		}
	}
	c.order = kept
	return removed
}

func (c *client) matches(routingKeys []string) bool {
	if !c.endpoint.TopicRouting {
		return true
	}

	c.topicsMu.RLock()
	defer c.topicsMu.RUnlock()
	if len(c.topics) == 0 {
		return false
	}
	if _, all := c.topics["all"]; all {
		return true
	}
	for _, key := range routingKeys {
		if _, ok := c.topics[key]; ok {
			return true
		}
	}
	return false
}

func (c *client) close(code coderws.StatusCode, reason string) {
	c.closeOnce.Do(func() {
		_ = c.conn.Close(code, reason)
		c.cancel()
	})
}

func (c *client) closeNow() {
	c.closeOnce.Do(func() {
		c.cancel()
		_ = c.conn.CloseNow()
	})
}
