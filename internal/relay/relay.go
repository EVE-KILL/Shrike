// Package relay publishes events for the WebSocket server.
//
// Events go out over Redis pub/sub on channels prefixed `ws:`, which is what
// the relay process subscribes to. That indirection is what lets the workers
// stay stateless: no worker holds a socket, and any number of them can publish
// to a relay that may be restarted, scaled or redeployed independently.
//
// Publishing is deliberately fire-and-forget. A killmail that was parsed,
// valued and stored has done the work that matters; failing the job because a
// notification did not go out would retry the whole thing and store nothing new.
// The event is lost, the data is not, and the site catches up on next load.
package relay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ChannelPrefix is what the relay process subscribes to.
const ChannelPrefix = "ws:"

// Channels currently in use. Named here rather than at each call site so the
// set is greppable, and because these strings are a contract with a separate
// process — changing one silently stops delivering to it.
const (
	ChannelKillmail = "killmails"
	ChannelKilllist = "killlist"
	ChannelFeed     = "feed"
	ChannelStatus   = "status"

	// Plural, matching the relay's endpoints (/comments, /announcements). The
	// mismatch with the singular queue names above is the existing convention,
	// not a typo — the queue is named for the event, the channel for the page.
	ChannelComments      = "comments"
	ChannelAnnouncements = "announcements"
)

// Event is one published message.
//
// RoutingKeys let a subscriber filter without the relay understanding the
// payload: a client subscribed to `victim.98000001` receives only the kills
// carrying that key. The relay matches keys and forwards; it never inspects
// Data.
type Event struct {
	RoutingKeys []string `json:"routing_keys"`
	Data        any      `json:"data"`
}

// Publisher sends events to the relay.
type Publisher struct {
	Redis *redis.Client

	// OnError is called when a publish fails, for logging. Optional — the
	// failure is never returned, because no caller should act on it.
	OnError func(channel string, err error)
}

// Publish sends one event.
//
// Never returns an error. See the package comment: the caller has already done
// the work that matters, and there is no useful action to take when a
// notification does not go out.
func (p *Publisher) Publish(ctx context.Context, channel string, routingKeys []string, data any) {
	if p == nil || p.Redis == nil {
		return
	}

	// An empty slice rather than nil, so the JSON carries `[]` and a subscriber
	// decoding into a typed array does not have to handle null.
	if routingKeys == nil {
		routingKeys = []string{}
	}

	body, err := json.Marshal(Event{RoutingKeys: routingKeys, Data: data})
	if err != nil {
		p.fail(channel, fmt.Errorf("encode event: %w", err))
		return
	}

	if err := p.Redis.Publish(ctx, ChannelPrefix+channel, body).Err(); err != nil {
		p.fail(channel, err)
	}
}

// PublishRaw sends a pre-shaped payload that is not wrapped in an Event.
//
// The feed channel predates the routing-key envelope and carries its own shape
// — `{seq, killmail_id, routing_keys}` at the top level rather than nested
// under `data`. Subscribers parse it that way, so it cannot be normalised
// without changing them.
func (p *Publisher) PublishRaw(ctx context.Context, channel string, payload any) {
	if p == nil || p.Redis == nil {
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		p.fail(channel, fmt.Errorf("encode payload: %w", err))
		return
	}
	if err := p.Redis.Publish(ctx, ChannelPrefix+channel, body).Err(); err != nil {
		p.fail(channel, err)
	}
}

func (p *Publisher) fail(channel string, err error) {
	if p.OnError != nil {
		p.OnError(channel, err)
	}
}
