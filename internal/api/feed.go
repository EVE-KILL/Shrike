package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type feedNotification struct {
	Seq         int64    `json:"seq"`
	KillmailID  int64    `json:"killmail_id"`
	RoutingKeys []string `json:"routing_keys"`
}

type feedClient struct {
	Topics map[string]struct{}
	Events chan []byte
}

// FeedManager fans Redis notifications out to SSE clients. Poll/catch-up
// reads remain in Postgres, which is the durable sequence source.
type FeedManager struct {
	db    Database
	redis *redis.Client

	mu      sync.RWMutex
	clients map[int64]*feedClient
	nextID  int64
}

func NewFeedManager(db Database, client *redis.Client) *FeedManager {
	return &FeedManager{
		db: db, redis: client, clients: map[int64]*feedClient{},
	}
}

func (m *FeedManager) Start(ctx context.Context) {
	if m == nil || m.redis == nil {
		return
	}
	pubsub := m.redis.Subscribe(ctx, "ws:feed")
	go func() {
		defer pubsub.Close() //nolint:errcheck
		for {
			message, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				if ctx.Err() == nil {
					log.Error().Err(err).Msg("feed subscription stopped")
				}
				return
			}
			m.publish(ctx, message.Payload)
		}
	}()
}

func (m *FeedManager) publish(ctx context.Context, raw string) {
	var notification feedNotification
	if err := json.Unmarshal([]byte(raw), &notification); err != nil ||
		notification.Seq == 0 || notification.KillmailID == 0 {
		return
	}
	data, err := loadKillmailsESI(
		ctx, m.db, []int32{int32(notification.KillmailID)},
	)
	if err != nil || len(data) == 0 {
		return
	}
	body, err := json.Marshal(normalizeJSON(data[0]))
	if err != nil {
		return
	}
	event := []byte(fmt.Sprintf(
		"id: %d\nevent: killmail\ndata: %s\n\n",
		notification.Seq, body,
	))

	m.mu.Lock()
	defer m.mu.Unlock()
	for id, client := range m.clients {
		if !feedTopicMatch(client.Topics, notification.RoutingKeys) {
			continue
		}
		select {
		case client.Events <- event:
		default:
			delete(m.clients, id)
		}
	}
}

func (m *FeedManager) add(topics map[string]struct{}) (int64, <-chan []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	client := &feedClient{Topics: topics, Events: make(chan []byte, 64)}
	m.clients[m.nextID] = client
	return m.nextID, client.Events
}

func (m *FeedManager) remove(id int64) {
	m.mu.Lock()
	delete(m.clients, id)
	m.mu.Unlock()
}

func (m *FeedManager) clientCount() int {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

func registerFeedRoutes(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "feed-index",
		Method:      http.MethodGet,
		Path:        "/feed",
		Summary:     "Real-time feed documentation",
		Tags:        []string{"feed"},
	}, func(context.Context, *legacyRequest) (legacyPayload, error) {
		return jsonPayload(feedIndex()), nil
	})

	registerLegacy(a, huma.Operation{
		OperationID: "feed-poll",
		Method:      http.MethodGet,
		Path:        "/feed/poll",
		Summary:     "Poll the killmail feed",
		Tags:        []string{"feed"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		return pollFeed(ctx, opts, req)
	})

	registerLegacy(a, huma.Operation{
		OperationID: "feed-status",
		Method:      http.MethodGet,
		Path:        "/feed/status",
		Summary:     "Feed status",
		Tags:        []string{"feed"},
	}, func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		head, err := feedHead(ctx, opts.DB)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"status": "ok",
			"clients": func() int {
				if opts.Feed == nil {
					return 0
				}
				return opts.Feed.clientCount()
			}(),
			"latestSeq": head,
		}), nil
	})

	registerFeedStream(a, opts)
}

func registerFeedStream(a huma.API, opts Options) {
	op := huma.Operation{
		OperationID: "feed-stream",
		Method:      http.MethodGet,
		Path:        "/feed/stream",
		Summary:     "Server-sent killmail stream",
		Tags:        []string{"feed"},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Server-sent events",
				Content: map[string]*huma.MediaType{
					"text/event-stream": {
						Schema: &huma.Schema{Type: huma.TypeString},
					},
				},
			},
		},
	}
	a.OpenAPI().AddOperation(&op)
	a.Adapter().Handle(&op, func(hctx huma.Context) {
		for name, value := range map[string]string{
			"Content-Type":      "text/event-stream",
			"Cache-Control":     "no-cache, no-store",
			"Connection":        "keep-alive",
			"X-Accel-Buffering": "no",
		} {
			hctx.SetHeader(name, value)
		}
		hctx.SetStatus(http.StatusOK)
		writer := hctx.BodyWriter()

		topicList := []string{"all"}
		if raw := hctx.Query("topics"); raw != "" {
			topicList = []string{}
			for _, topic := range strings.Split(raw, ",") {
				if topic != "" {
					topicList = append(topicList, topic)
				}
			}
		}
		_, _ = fmt.Fprintf(writer,
			": connected to EVE-Kill feed\n: topics: %s\n\n",
			strings.Join(topicList, ", "),
		)
		flushHTTP(writer)

		lastRaw := hctx.Header("Last-Event-ID")
		if lastRaw == "" {
			lastRaw = hctx.Query("lastEventId")
		}
		if last, err := strconv.ParseInt(lastRaw, 10, 64); err == nil && lastRaw != "" {
			_ = writeFeedCatchup(hctx.Context(), opts.DB, writer, last)
			flushHTTP(writer)
		}
		if opts.Feed == nil {
			<-hctx.Context().Done()
			return
		}
		topics := map[string]struct{}{}
		for _, topic := range topicList {
			topics[topic] = struct{}{}
		}
		clientID, events := opts.Feed.add(topics)
		defer opts.Feed.remove(clientID)
		for {
			select {
			case <-hctx.Context().Done():
				return
			case event := <-events:
				if _, err := writer.Write(event); err != nil {
					return
				}
				flushHTTP(writer)
			}
		}
	})
}

func flushHTTP(writer any) {
	if flusher, ok := writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func writeFeedCatchup(
	ctx context.Context,
	db Database,
	writer interface{ Write([]byte) (int, error) },
	after int64,
) error {
	rows, err := queryMaps(ctx, db, `
		SELECT seq, killmail_id FROM feed_queue
		WHERE seq > $1 ORDER BY seq LIMIT 1000`, after)
	if err != nil || len(rows) == 0 {
		return err
	}
	ids := make([]int32, 0, len(rows))
	for _, row := range rows {
		id, _ := int64Value(row["killmail_id"])
		ids = append(ids, int32(id))
	}
	esi, err := loadKillmailsESI(ctx, db, ids)
	if err != nil {
		return err
	}
	esiByID := map[int64]map[string]any{}
	for _, value := range esi {
		id, _ := int64Value(value["killmail_id"])
		esiByID[id] = value
	}
	for _, row := range rows {
		seq, _ := int64Value(row["seq"])
		id, _ := int64Value(row["killmail_id"])
		value := esiByID[id]
		if value == nil {
			continue
		}
		body, err := json.Marshal(normalizeJSON(value))
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer,
			"id: %d\nevent: killmail\ndata: %s\n\n", seq, body,
		); err != nil {
			return err
		}
	}
	return nil
}

func pollFeed(
	ctx context.Context,
	opts Options,
	req *legacyRequest,
) (legacyPayload, error) {
	after := int64(numberOr(req.Query.Get("after"), 0))
	limit := boundedQueryInt(req, "limit", 100, 1, 1000)
	head, err := feedHead(ctx, opts.DB)
	if err != nil {
		return legacyPayload{}, err
	}
	headNumber := int64(0)
	if head != nil {
		headNumber = *head
	}
	lastPageAfter := max(int64(0), headNumber-int64(limit))
	pollURL := func(cursor int64) string {
		return fmt.Sprintf("/feed/poll?after=%d&limit=%d", cursor, limit)
	}
	headers := http.Header{"Cache-Control": []string{"no-cache"}}
	if after == 0 {
		return legacyPayload{Headers: headers, Body: map[string]any{
			"data": []any{}, "latest": head,
			"hasMore": head != nil && *head > 0,
			"next":    pollURL(0), "last": pollURL(lastPageAfter),
		}}, nil
	}
	rows, err := queryMaps(ctx, opts.DB, `
		SELECT f.seq, f.killmail_id, k.killmail_hash
		FROM feed_queue f
		INNER JOIN killmails k ON k.killmail_id = f.killmail_id
		WHERE f.seq > $1 ORDER BY f.seq LIMIT $2`, after, limit+1)
	if err != nil {
		return legacyPayload{}, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	if len(rows) == 0 {
		return legacyPayload{Headers: headers, Body: map[string]any{
			"data": []any{}, "latest": head, "hasMore": false,
			"next": pollURL(after), "last": pollURL(lastPageAfter),
		}}, nil
	}
	ids := make([]int32, 0, len(rows))
	for _, row := range rows {
		id, _ := int64Value(row["killmail_id"])
		ids = append(ids, int32(id))
	}
	esi, err := loadKillmailsESI(ctx, opts.DB, ids)
	if err != nil {
		return legacyPayload{}, err
	}
	esiByID := map[int64]map[string]any{}
	for _, value := range esi {
		id, _ := int64Value(value["killmail_id"])
		esiByID[id] = value
	}
	data := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id, _ := int64Value(row["killmail_id"])
		seq, _ := int64Value(row["seq"])
		data = append(data, map[string]any{
			"seq": seq, "killmail_id": id,
			"killmail_hash": row["killmail_hash"], "data": esiByID[id],
		})
	}
	last, _ := int64Value(rows[len(rows)-1]["seq"])
	return legacyPayload{Headers: headers, Body: map[string]any{
		"data": data, "latest": head, "hasMore": hasMore,
		"next": pollURL(last), "last": pollURL(lastPageAfter),
	}}, nil
}

func feedHead(ctx context.Context, db Database) (*int64, error) {
	row, err := queryMap(ctx, db,
		`SELECT seq FROM feed_queue ORDER BY seq DESC LIMIT 1`)
	if err != nil || row == nil {
		return nil, err
	}
	value, _ := int64Value(row["seq"])
	return &value, nil
}

func feedTopicMatch(topics map[string]struct{}, routing []string) bool {
	if _, all := topics["all"]; all {
		return true
	}
	for _, key := range routing {
		if _, exists := topics[key]; exists {
			return true
		}
	}
	return false
}

func feedIndex() map[string]any {
	return map[string]any{
		"name":        "EVE-Kill Feed API",
		"description": "Real-time killmail feed via SSE (push) or polling (pull). All killmails are delivered in ESI-compatible format. Events use a monotonic sequence ID that reflects processing order — use this as your cursor, not killmail_id.",
		"endpoints": map[string]any{
			"GET /feed/stream": map[string]any{
				"description": "Server-Sent Events stream of killmails in ESI format",
				"params":      map[string]any{"topics": "Comma-separated topic filters (default: all). Only matching kills are streamed."},
				"headers":     map[string]any{"Last-Event-ID": "Sequence ID to resume from after reconnect (catches up from database)"},
				"example":     "curl -N 'https://eve-kill.com/api/feed/stream?topics=solo,10b'",
			},
			"GET /feed/poll": map[string]any{
				"description": "Poll for killmails after a given sequence ID, returned in ESI format. Response includes `latest` (current head seq), `next` (URL to walk forward from the returned page), and `last` (URL to jump to the most recent page) so fetchers can both stream forward and skip ahead.",
				"params": map[string]any{
					"after": "Sequence ID cursor — returns events after this (use 1 to start from the beginning, 0 for an empty kickstart response that reports the head position)",
					"limit": "Max results 1–1000 (default: 100)",
				},
				"example": "curl 'https://eve-kill.com/api/feed/poll?after=1&limit=100'",
			},
			"GET /feed/status": map[string]any{
				"description": "Feed health: connected SSE clients, current sequence head",
			},
		},
		"topics": map[string]any{
			"universal": []string{"all"},
			"kill_type": []string{"solo", "npc"},
			"value":     []string{"10b", "5b"},
			"security":  []string{"highsec", "lowsec", "nullsec", "wspace", "abyssal"},
			"ship_class": []string{
				"big", "citadel", "capitals", "freighters", "supercarriers",
				"titans", "frigates", "destroyers", "cruisers",
				"battlecruisers", "battleships", "t1", "t2", "t3",
			},
			"location": []string{"system.{id}", "region.{id}", "constellation.{id}"},
			"entity":   []string{"victim.{id}", "attacker.{id}"},
		},
		"note": "Topic filtering is available on the SSE stream. The poll endpoint returns all kills — filter client-side if needed.",
	}
}
