package workers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/relay"
)

// announcementActiveSet tracks which announcements have already been broadcast.
//
// A Redis set rather than a database column, because it is transient
// coordination rather than state worth keeping: if it is lost the next tick
// re-emits every active announcement, which the frontend deduplicates by id.
const announcementActiveSet = "announcements:emitted:active"

const relayTimeLayout = "2006-01-02T15:04:05.000Z"

// cronAnnouncementSchedule emits events as announcements start and expire.
//
// Announcements have a start and an end but nothing writes an event when those
// times pass — so this polls, and the set is what stops it re-emitting the same
// start every thirty seconds for the life of the announcement.
func (d *Deps) cronAnnouncementSchedule(ctx context.Context) (string, error) {
	if d.Redis == nil {
		return "", fmt.Errorf("announcement_schedule needs Redis to track what it has emitted")
	}

	rows, err := d.Pool.Query(ctx, `
        SELECT id, tier, title, body_md, body_html, color, icon,
               link_url, link_label, starts_at, expires_at
        FROM announcements
        WHERE archived_at IS NULL
          AND starts_at <= now()
          AND expires_at > now()`)
	if err != nil {
		return "", err
	}

	type announcement struct {
		ID        int64   `json:"id"`
		Tier      int32   `json:"tier"`
		Title     string  `json:"title"`
		BodyMD    string  `json:"body_md"`
		BodyHTML  string  `json:"body_html"`
		Color     string  `json:"color"`
		Icon      *string `json:"icon"`
		LinkURL   *string `json:"link_url"`
		LinkLabel *string `json:"link_label"`
		StartsAt  string  `json:"starts_at"`
		ExpiresAt string  `json:"expires_at"`
	}

	var active []announcement
	for rows.Next() {
		var a announcement
		var startsAt, expiresAt time.Time
		if err := rows.Scan(&a.ID, &a.Tier, &a.Title, &a.BodyMD, &a.BodyHTML,
			&a.Color, &a.Icon, &a.LinkURL, &a.LinkLabel, &startsAt, &expiresAt); err != nil {
			rows.Close()
			return "", err
		}
		a.StartsAt = startsAt.UTC().Format(relayTimeLayout)
		a.ExpiresAt = expiresAt.UTC().Format(relayTimeLayout)
		active = append(active, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", err
	}

	activeIDs := make(map[string]bool, len(active))
	var started int
	for _, a := range active {
		id := strconv.FormatInt(a.ID, 10)
		activeIDs[id] = true

		emitted, err := d.Redis.SIsMember(ctx, announcementActiveSet, id).Result()
		if err != nil {
			return "", err
		}
		if emitted {
			continue
		}

		d.Relay.Publish(ctx, relay.ChannelAnnouncements, []string{"all"}, map[string]any{
			"event_type":   "new",
			"announcement": a,
		})
		if err := d.Redis.SAdd(ctx, announcementActiveSet, id).Err(); err != nil {
			return "", err
		}
		started++
	}

	// Anything in the set that is no longer active has expired or been
	// archived since the last tick.
	members, err := d.Redis.SMembers(ctx, announcementActiveSet).Result()
	if err != nil {
		return "", err
	}

	var expired int
	for _, id := range members {
		if activeIDs[id] {
			continue
		}
		numeric, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			// Not an id we wrote; drop it rather than trying forever.
			_ = d.Redis.SRem(ctx, announcementActiveSet, id).Err()
			continue
		}

		d.Relay.Publish(ctx, relay.ChannelAnnouncements, []string{"all"}, map[string]any{
			"event_type":   "expired",
			"announcement": map[string]any{"id": numeric},
		})
		if err := d.Redis.SRem(ctx, announcementActiveSet, id).Err(); err != nil {
			return "", err
		}
		expired++
	}

	if started == 0 && expired == 0 {
		return "", nil
	}
	return fmt.Sprintf("%d started, %d expired", started, expired), nil
}

// cronStatusUpdate publishes the system status.
//
// Runs every second, which is why almost nothing here is gathered every second:
// see internal/status for how the tiers are split. The collector is built once
// per registry rather than per run, because it caches its expensive tiers
// between ticks and a fresh one would refetch everything every second.
func (d *Deps) cronStatusUpdate(ctx context.Context) (string, error) {
	s := d.statusCollector.Collect(ctx)
	d.Relay.Publish(ctx, relay.ChannelStatus, []string{"all", "status"}, s)

	// No report: this runs 86,400 times a day and a line per run would be the
	// only thing in the log.
	return "", nil
}
