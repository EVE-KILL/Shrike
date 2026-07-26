// Package universe imports the hourly state of New Eden that ESI publishes as
// aggregates rather than as events.
package universe

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/pgbulk"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ActivityRetention is how long hourly system activity is kept.
//
// Three days. The data is used for "is anything happening here right now"
// displays, and a row from last week answers no question anybody asks — while
// keeping them all would add ~8,500 rows an hour forever.
const ActivityRetention = 72 * time.Hour

// SystemKills is /universe/system_kills/.
type SystemKills struct {
	SystemID  int32 `json:"system_id"`
	ShipKills int32 `json:"ship_kills"`
	NPCKills  int32 `json:"npc_kills"`
	PodKills  int32 `json:"pod_kills"`
}

// SystemJumps is /universe/system_jumps/.
type SystemJumps struct {
	SystemID  int32 `json:"system_id"`
	ShipJumps int32 `json:"ship_jumps"`
}

// ActivityResult reports one import.
type ActivityResult struct {
	Systems int64 `json:"systems"`
	Purged  int64 `json:"purged"`
}

// ImportActivity fetches system kills and jumps and stores them against the
// current hour.
//
// The two endpoints report overlapping but different sets of systems — a system
// with jumps and no kills appears only in one — so the union is taken and the
// missing half recorded as zero. Storing only the intersection would silently
// drop every quiet system.
func ImportActivity(ctx context.Context, pool *pgxpool.Pool, client *esi.Client) (ActivityResult, error) {
	var out ActivityResult

	killsRes, err := esi.Get[[]SystemKills](ctx, client, "/latest/universe/system_kills/")
	if err != nil {
		return out, err
	}
	jumpsRes, err := esi.Get[[]SystemJumps](ctx, client, "/latest/universe/system_jumps/")
	if err != nil {
		return out, err
	}
	if !killsRes.OK() || killsRes.Data == nil || !jumpsRes.OK() || jumpsRes.Data == nil {
		return out, fmt.Errorf("ESI returned kills=%d jumps=%d for system activity",
			killsRes.Status, jumpsRes.Status)
	}

	// Truncated to the hour, so a run that starts at 14:59 and one that starts
	// at 14:01 write the same row rather than two.
	hour := time.Now().UTC().Truncate(time.Hour)

	type activity struct {
		shipKills, npcKills, podKills, shipJumps int32
	}
	merged := make(map[int32]*activity, len(*killsRes.Data)+len(*jumpsRes.Data))
	at := func(id int32) *activity {
		if a, ok := merged[id]; ok {
			return a
		}
		a := &activity{}
		merged[id] = a
		return a
	}

	for _, k := range *killsRes.Data {
		if k.SystemID == 0 {
			continue
		}
		a := at(k.SystemID)
		a.shipKills, a.npcKills, a.podKills = k.ShipKills, k.NPCKills, k.PodKills
	}
	for _, j := range *jumpsRes.Data {
		if j.SystemID == 0 {
			continue
		}
		at(j.SystemID).shipJumps = j.ShipJumps
	}

	// COPY into a staging table and merge, rather than eight and a half
	// thousand individual upserts every hour.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	columns := []string{"system_id", "timestamp", "ship_kills", "npc_kills", "pod_kills", "ship_jumps"}
	const staging = "universe_staging_activity"
	if err := pgbulk.StagingTx(ctx, tx, staging, "system_activity"); err != nil {
		return out, err
	}

	w := pgbulk.NewCopier(ctx, tx, staging, columns)
	for id, a := range merged {
		if err := w.Add([]any{id, hour, a.shipKills, a.npcKills, a.podKills, a.shipJumps}); err != nil {
			return out, err
		}
	}
	if err := w.Flush(); err != nil {
		return out, err
	}

	// DoUpdate because a re-run within the same hour must correct the row it
	// already wrote rather than keep the earlier reading.
	if _, err := tx.Exec(ctx, pgbulk.MergeSQL("system_activity", staging, columns,
		[]string{"system_id", "timestamp"}, pgbulk.DoUpdate)); err != nil {
		return out, fmt.Errorf("merge system_activity: %w", err)
	}

	tag, err := tx.Exec(ctx,
		`DELETE FROM system_activity WHERE timestamp < $1`, time.Now().UTC().Add(-ActivityRetention))
	if err != nil {
		return out, err
	}
	out.Purged = tag.RowsAffected()

	if err := tx.Commit(ctx); err != nil {
		return out, err
	}
	out.Systems = w.Written()
	return out, nil
}
