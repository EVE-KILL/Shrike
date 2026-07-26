package websocket

import "time"

// Stats is an internal operational snapshot.
//
// It is intentionally not mounted as a public HTTP endpoint. A later telemetry
// publisher can put this exact value in Redis alongside Caddy, database and API
// snapshots for the status page and admin tooling.
type Stats struct {
	Status      string                   `json:"status"`
	Uptime      int64                    `json:"uptime"`
	Timestamp   string                   `json:"timestamp"`
	Connections int                      `json:"connections"`
	Redis       bool                     `json:"redis"`
	Channels    []string                 `json:"channels"`
	Endpoints   map[string]EndpointStats `json:"endpoints"`
}

type EndpointStats struct {
	Connections   int            `json:"connections"`
	Subscriptions map[string]int `json:"subscriptions"`
}

// Snapshot reports current connection and subscription counts.
func (s *Server) Snapshot() Stats {
	out := Stats{
		Status:    "ok",
		Uptime:    int64(time.Since(s.startedAt).Seconds()),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Redis:     s.redisReady.Load(),
		Endpoints: make(map[string]EndpointStats, len(endpoints)),
	}
	for _, endpoint := range endpoints {
		out.Channels = append(out.Channels, endpoint.Channel)
		out.Endpoints[BasePath+endpoint.Path] = EndpointStats{
			Subscriptions: map[string]int{},
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	out.Connections = len(s.clients)
	for c := range s.clients {
		key := BasePath + c.endpoint.Path
		stats := out.Endpoints[key]
		stats.Connections++

		c.topicsMu.RLock()
		for topic := range c.topics {
			stats.Subscriptions[topic]++
		}
		c.topicsMu.RUnlock()
		out.Endpoints[key] = stats
	}
	return out
}
