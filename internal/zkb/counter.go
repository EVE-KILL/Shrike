package zkb

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Ingest throughput, as hourly buckets in Redis.
//
// The status page shows the last day of "new" against "repost" activity, which
// answers a question worth answering: a feed that is mostly reposts is a feed
// that has fallen behind and is re-reading what we already have.
//
// Buckets are keyed by epoch hour with a TTL slightly over the window, so old
// ones expire by themselves and there is no cleanup job. The window matches the
// ESI-fetch stats window on the same page so the two are directly comparable.

// IngestWindowHours is how much history the counters keep.
const IngestWindowHours = 24

// ingestBucketTTL outlives the window by an hour, so the oldest bucket in a
// query is never one that expired between being written and being read.
const ingestBucketTTL = (IngestWindowHours + 1) * time.Hour

// IngestKind is what happened to a killmail from the feed.
type IngestKind string

const (
	// IngestNew is a killmail we did not already have.
	IngestNew IngestKind = "new"
	// IngestRepost is one we did — routine, since R2Z2 re-publishes during
	// backfeed storms and the same kill also arrives from the ESI backfill.
	IngestRepost IngestKind = "repost"
)

// RecordIngest increments the current hour's counter.
//
// Errors are returned but every caller ignores them: a counter is telemetry,
// and failing to record one must never interrupt the ingestion it measures.
func RecordIngest(ctx context.Context, rdb *redis.Client, kind IngestKind) error {
	if rdb == nil {
		return nil
	}

	key := ingestKey(kind, time.Now().UTC().Unix()/3600)
	pipe := rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ingestBucketTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// IngestStats is the throughput over the window.
type IngestStats struct {
	WindowHours int `json:"window_hours"`

	// Hourly counts, oldest first, always WindowHours long — an hour with no
	// traffic is a zero rather than a gap, so the series can be charted
	// directly.
	NewHourly    []int64 `json:"new_hourly"`
	RepostHourly []int64 `json:"repost_hourly"`

	NewTotal    int64 `json:"new_total"`
	RepostTotal int64 `json:"repost_total"`
}

// ReadIngestStats returns the throughput over the window.
func ReadIngestStats(ctx context.Context, rdb *redis.Client) (*IngestStats, error) {
	if rdb == nil {
		return nil, nil
	}

	nowHour := time.Now().UTC().Unix() / 3600
	out := &IngestStats{
		WindowHours:  IngestWindowHours,
		NewHourly:    make([]int64, IngestWindowHours),
		RepostHourly: make([]int64, IngestWindowHours),
	}

	// One pipeline for all 48 keys rather than 48 round trips.
	pipe := rdb.Pipeline()
	cmds := make([]*redis.StringCmd, 0, IngestWindowHours*2)
	for i := range IngestWindowHours {
		hour := nowHour - int64(IngestWindowHours-1-i)
		cmds = append(cmds, pipe.Get(ctx, ingestKey(IngestNew, hour)))
		cmds = append(cmds, pipe.Get(ctx, ingestKey(IngestRepost, hour)))
	}
	// redis.Nil is what a missing bucket returns, which is an hour with no
	// traffic rather than a failure.
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}

	for i := range IngestWindowHours {
		n := parseCount(cmds[i*2])
		r := parseCount(cmds[i*2+1])
		out.NewHourly[i], out.RepostHourly[i] = n, r
		out.NewTotal += n
		out.RepostTotal += r
	}
	return out, nil
}

func ingestKey(kind IngestKind, hour int64) string {
	return fmt.Sprintf("zkb:ingest:%s:%d", kind, hour)
}

func parseCount(cmd *redis.StringCmd) int64 {
	v, err := cmd.Result()
	if err != nil {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
