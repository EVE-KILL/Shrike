// Package campaign computes the aggregate view of a user-defined conflict.
//
// A campaign is a saved query: some entities, optionally a location, and a time
// window. Its statistics are recomputed from the killmails rather than
// incremented, which is what lets a campaign be edited after the fact — change
// a side and the numbers follow.
//
// Recomputation is expensive, so almost everything here exists to avoid doing
// it. The gate check answers "has anything happened since last time?" with a
// few index probes, and an idle campaign therefore costs nearly nothing however
// often the sweep runs. Getting that wrong does not produce wrong numbers; it
// produces an hourly job that recomputes hundreds of dormant campaigns.
package campaign

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Campaign lifecycle states, stored as smallints.
const (
	StatusPending  = 0
	StatusActive   = 1
	StatusArchived = 2
)

// Entity types on a side.
const (
	EntityCharacter   = 0
	EntityCorporation = 1
	EntityAlliance    = 2
)

// Timing policy.
const (
	// EndGraceHours is how long after a campaign's end it stays eligible for
	// one last catch-up compute. Killmails arrive late, so finalising the
	// instant the clock passes would freeze a campaign missing its final kills.
	EndGraceHours = 1

	// PrizeEndGraceHours is the longer window for a campaign with a prize
	// pool. Money is being settled from these numbers, so they get a full day
	// for stragglers before anything is paid.
	PrizeEndGraceHours = 24

	// InactivityArchiveDays retires an open-ended campaign that has seen no
	// matching kill. Area campaigns are exempt — a quiet region is still a
	// region, and its campaign is a standing watch rather than a dead one.
	InactivityArchiveDays = 30
)

// IDBoundSlack widens the killmail-id range derived from a time bound.
//
// Killmail ids correlate with time but are not strictly ordered by it — mails
// are backfilled and arrive late — so converting a timestamp to an id range
// needs slack on both sides or genuine kills fall outside it.
const IDBoundSlack = 2 * 24 * time.Hour

// Location narrows a campaign to part of space.
type Location struct {
	SystemIDs        []int32 `json:"systemIds,omitempty"`
	ConstellationIDs []int32 `json:"constellationIds,omitempty"`
	RegionIDs        []int32 `json:"regionIds,omitempty"`
}

// HasFilter reports whether the location narrows anything.
func (l Location) HasFilter() bool {
	return len(l.SystemIDs) > 0 || len(l.ConstellationIDs) > 0 || len(l.RegionIDs) > 0
}

// Entity is one participant on a side.
type Entity struct {
	ID         int64
	SideIndex  int16
	EntityType int16
	EntityID   int32
}

// Campaign is the stored definition.
type Campaign struct {
	ID               string
	StartTime        time.Time
	EndTime          *time.Time
	ProcessedThrough *time.Time
	LastActivityAt   *time.Time
	CreatedAt        time.Time
	Status           int16
	Location         Location
	HasPrizePool     bool
	ProcessingPaused bool
}

// EffectiveEnd is the campaign's end, or now for an open-ended one.
func (c Campaign) EffectiveEnd(now time.Time) time.Time {
	if c.EndTime == nil || c.EndTime.After(now) {
		return now
	}
	return *c.EndTime
}

// GracePeriod is how long after the end this campaign stays open for a final
// compute. Longer when money depends on the result.
func (c Campaign) GracePeriod() time.Duration {
	if c.HasPrizePool {
		return PrizeEndGraceHours * time.Hour
	}
	return EndGraceHours * time.Hour
}

// Finished reports whether the campaign's end plus its grace has passed.
func (c Campaign) Finished(now time.Time) bool {
	if c.EndTime == nil {
		return false
	}
	return now.After(c.EndTime.Add(c.GracePeriod()))
}

// entityIDs returns the ids of one type on any side.
func entityIDs(entities []Entity, entityType int16) []int32 {
	var out []int32
	for _, e := range entities {
		if e.EntityType == entityType && e.EntityID != 0 {
			out = append(out, e.EntityID)
		}
	}
	return out
}

// IsAreaCampaign reports a campaign defined by place rather than by
// participants.
func IsAreaCampaign(entities []Entity, loc Location) bool {
	return len(entities) == 0 && loc.HasFilter()
}

// Load reads a campaign and its sides.
func Load(ctx context.Context, pool *pgxpool.Pool, id string) (*Campaign, []Entity, error) {
	var c Campaign
	var rawLocation []byte

	err := pool.QueryRow(ctx, `
        SELECT campaign_id, start_time, end_time, processed_through, last_activity_at,
               created_at, coalesce(status, 0), location,
               coalesce(processing_paused, false),
               EXISTS (SELECT 1 FROM campaign_prize_pools p WHERE p.campaign_id = c.campaign_id)
        FROM campaigns c WHERE campaign_id = $1`, id).
		Scan(&c.ID, &c.StartTime, &c.EndTime, &c.ProcessedThrough, &c.LastActivityAt,
			&c.CreatedAt, &c.Status, &rawLocation, &c.ProcessingPaused, &c.HasPrizePool)
	if err != nil {
		return nil, nil, err
	}
	if len(rawLocation) > 0 {
		// A malformed location is treated as no location rather than failing
		// the whole compute — the campaign is still valid, just unfiltered.
		_ = json.Unmarshal(rawLocation, &c.Location)
	}

	rows, err := pool.Query(ctx, `
        SELECT id, side_index, entity_type, entity_id
        FROM campaign_side_entities WHERE campaign_id = $1
        ORDER BY side_index, id`, id)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.SideIndex, &e.EntityType, &e.EntityID); err != nil {
			return nil, nil, err
		}
		entities = append(entities, e)
	}
	return &c, entities, rows.Err()
}

// HasNewKills reports whether anything matching the campaign has happened since
// it was last computed.
//
// This is the gate that makes an hourly sweep over hundreds of campaigns
// affordable: a handful of index probes that stop at the first match, rather
// than the full recompute. It is deliberately allowed to be optimistic — a
// false positive costs one wasted recompute, while a false negative would leave
// a campaign silently stale.
func HasNewKills(ctx context.Context, pool *pgxpool.Pool, c *Campaign, entities []Entity) (bool, error) {
	now := time.Now().UTC()

	since := c.StartTime
	if c.ProcessedThrough != nil {
		since = *c.ProcessedThrough
	}
	until := c.EffectiveEnd(now)
	if !since.Before(until) {
		return false, nil
	}

	chars := entityIDs(entities, EntityCharacter)
	corps := entityIDs(entities, EntityCorporation)
	allies := entityIDs(entities, EntityAlliance)

	// A campaign with neither participants nor a location matches nothing.
	if len(chars)+len(corps)+len(allies) == 0 && !c.Location.HasFilter() {
		return false, nil
	}

	// Attacker legs first: killmail_attackers is indexed by (entity, time), so
	// these probe directly and are the cheapest question available.
	for _, leg := range []struct {
		column string
		ids    []int32
	}{
		{"alliance_id", allies},
		{"corporation_id", corps},
		{"character_id", chars},
	} {
		if len(leg.ids) == 0 {
			continue
		}
		var found bool
		err := pool.QueryRow(ctx, fmt.Sprintf(`
            SELECT true FROM killmail_attackers
            WHERE %s = ANY($1::int[])
              AND killmail_time > $2::timestamptz AND killmail_time <= $3::timestamptz
            LIMIT 1`, leg.column), leg.ids, since, until).Scan(&found)
		if err == nil && found {
			return true, nil
		}
		if err != nil && !isNoRows(err) {
			return false, err
		}
	}

	// Victim and location legs are keyed by killmail_id rather than by time, so
	// the time bound has to become an id bound first.
	idLo, ok, err := idAtOrAfter(ctx, pool, since.Add(-IDBoundSlack))
	if err != nil || !ok {
		return false, err
	}

	for _, leg := range []struct {
		column string
		ids    []int32
	}{
		{"victim_alliance_id", allies},
		{"victim_corporation_id", corps},
		{"victim_character_id", chars},
		{"solar_system_id", c.Location.SystemIDs},
		{"constellation_id", c.Location.ConstellationIDs},
		{"region_id", c.Location.RegionIDs},
	} {
		if len(leg.ids) == 0 {
			continue
		}
		var found bool
		err := pool.QueryRow(ctx, fmt.Sprintf(`
            SELECT true FROM killmails
            WHERE %s = ANY($1::int[]) AND killmail_id >= $2
              AND killmail_time > $3::timestamptz AND killmail_time <= $4::timestamptz
            LIMIT 1`, leg.column), leg.ids, idLo, since, until).Scan(&found)
		if err == nil && found {
			return true, nil
		}
		if err != nil && !isNoRows(err) {
			return false, err
		}
	}

	return false, nil
}

// idAtOrAfter converts a time bound into a killmail id bound.
func idAtOrAfter(ctx context.Context, pool *pgxpool.Pool, at time.Time) (int64, bool, error) {
	var id int64
	err := pool.QueryRow(ctx, `
        SELECT killmail_id FROM killmails
        WHERE killmail_time >= $1::timestamptz
        ORDER BY killmail_time ASC LIMIT 1`, at).Scan(&id)
	if isNoRows(err) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}
