package relay

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The feed is the durable half of the live stream.
//
// Pub/sub alone cannot serve a client that was disconnected for a minute:
// Redis delivers to whoever is listening and forgets. So every killmail is also
// appended to feed_queue, which assigns it a monotonically increasing sequence
// number, and the notification carries that number. A client tracks the last
// sequence it saw and asks for everything since — so a reconnect is a query
// rather than a gap.
//
// The table is bounded by the feed_purge cron at a year.

// FeedNotice is the payload published on the feed channel.
//
// Flat rather than wrapped in an Event, because this channel predates the
// routing-key envelope and its subscribers parse it at the top level.
type FeedNotice struct {
	Seq         int64    `json:"seq"`
	KillmailID  int64    `json:"killmail_id"`
	RoutingKeys []string `json:"routing_keys"`
}

// PublishToFeed appends a killmail to the feed and notifies subscribers.
//
// The insert is what matters and its failure is returned; the notification is
// best-effort, because a client that misses it recovers on its next poll from
// the sequence number it already holds. This intentionally avoids the
// TypeScript failure mode where a Redis outage retries a successful insert and
// appends duplicate feed rows.
func (p *Publisher) PublishToFeed(ctx context.Context, pool *pgxpool.Pool, killmailID int64, routingKeys []string) error {
	var seq int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO feed_queue (killmail_id) VALUES ($1) RETURNING seq`,
		killmailID).Scan(&seq); err != nil {
		return err
	}

	if routingKeys == nil {
		routingKeys = []string{}
	}
	p.PublishRaw(ctx, ChannelFeed, FeedNotice{
		Seq:         seq,
		KillmailID:  killmailID,
		RoutingKeys: routingKeys,
	})
	return nil
}
