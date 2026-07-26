package fw

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Faction statistics and leaderboards.
//
// Both are pure snapshots — ESI reports the current standing and there is no
// history to preserve — so unlike the occupancy importer these simply overwrite.
// That makes the TypeScript's self-assigning upsert especially costly here:
// with nothing else recording the numbers, a row that never updates is a number
// that is simply wrong. In production all six fw_faction_stats rows have been
// frozen since 2026-04-11, and fw_leaderboards holds 4,775 rows each stuck at
// whatever it was on the day it was first inserted, across 51 distinct days.

// StatsResult reports one stats import.
type StatsResult struct {
	Factions     int64 `json:"factions"`
	Leaderboards int64 `json:"leaderboards"`
}

// ImportStats refreshes faction standings and the three leaderboards.
func ImportStats(ctx context.Context, pool *pgxpool.Pool, client *esi.Client) (StatsResult, error) {
	var out StatsResult
	now := time.Now().UTC()

	statsRes, err := esi.FetchFwStats(ctx, client)
	if err != nil {
		return out, err
	}
	if !statsRes.OK() || statsRes.Data == nil {
		return out, fmt.Errorf("ESI returned %d for faction warfare stats", statsRes.Status)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	for _, f := range *statsRes.Data {
		if f.FactionID == 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
            INSERT INTO fw_faction_stats (
                faction_id, pilots, systems_controlled,
                kills_yesterday, kills_last_week, kills_total,
                vp_yesterday, vp_last_week, vp_total, updated_at
            ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
            ON CONFLICT (faction_id) DO UPDATE SET
                pilots = EXCLUDED.pilots,
                systems_controlled = EXCLUDED.systems_controlled,
                kills_yesterday = EXCLUDED.kills_yesterday,
                kills_last_week = EXCLUDED.kills_last_week,
                kills_total = EXCLUDED.kills_total,
                vp_yesterday = EXCLUDED.vp_yesterday,
                vp_last_week = EXCLUDED.vp_last_week,
                vp_total = EXCLUDED.vp_total,
                updated_at = EXCLUDED.updated_at`,
			f.FactionID, f.Pilots, f.SystemsControlled,
			f.Kills.Yesterday, f.Kills.LastWeek, f.Kills.Total,
			f.VictoryPoints.Yesterday, f.VictoryPoints.LastWeek, f.VictoryPoints.Total, now); err != nil {
			return out, fmt.Errorf("upsert faction stats %d: %w", f.FactionID, err)
		}
		out.Factions++
	}

	n, err := importLeaderboards(ctx, tx, client, now)
	if err != nil {
		return out, err
	}
	out.Leaderboards = n

	return out, tx.Commit(ctx)
}

// leaderboardSource pairs an entity type with the fetcher that returns it.
type leaderboardSource struct {
	entityType string
	fetch      func(context.Context, *esi.Client) (esi.Response[esi.FwLeaderboard], error)
}

// importLeaderboards writes all three leaderboards.
//
// Each is a triplet of periods for each of two metrics, so one ESI response
// becomes six ranked lists. They are flattened into one table keyed by
// (entity_type, entity_id, metric, period), which is why the row count is in
// the thousands rather than the dozens.
func importLeaderboards(ctx context.Context, tx pgx.Tx, client *esi.Client, now time.Time) (int64, error) {
	sources := []leaderboardSource{
		{"faction", esi.FetchFwLeaderboards},
		{"character", esi.FetchFwLeaderboardsCharacters},
		{"corporation", esi.FetchFwLeaderboardsCorporations},
	}

	var written int64
	for _, src := range sources {
		res, err := src.fetch(ctx, client)
		if err != nil {
			return written, err
		}
		// One leaderboard being unavailable must not lose the other two, nor
		// the faction stats already written in this transaction.
		if !res.OK() || res.Data == nil {
			continue
		}

		// Listed rather than built from a map: map iteration order is random in
		// Go, and a failing test that names a different row each run is much
		// harder to read than one that does not.
		lists := []struct {
			metric  string
			period  string
			entries []esi.FwLeaderboardEntry
		}{
			{"kills", "yesterday", res.Data.Kills.Yesterday},
			{"kills", "last_week", res.Data.Kills.LastWeek},
			{"kills", "active_total", res.Data.Kills.ActiveTotal},
			{"victory_points", "yesterday", res.Data.VictoryPoints.Yesterday},
			{"victory_points", "last_week", res.Data.VictoryPoints.LastWeek},
			{"victory_points", "active_total", res.Data.VictoryPoints.ActiveTotal},
		}

		for _, l := range lists {
			for _, e := range l.entries {
				id := leaderboardEntityID(src.entityType, e)
				if id == 0 {
					continue
				}
				if _, err := tx.Exec(ctx, `
                    INSERT INTO fw_leaderboards (entity_type, entity_id, metric, period, amount, updated_at)
                    VALUES ($1,$2,$3,$4,$5,$6)
                    ON CONFLICT (entity_type, entity_id, metric, period) DO UPDATE SET
                        amount = EXCLUDED.amount,
                        updated_at = EXCLUDED.updated_at`,
					src.entityType, id, l.metric, l.period, e.Amount, now); err != nil {
					return written, fmt.Errorf("upsert %s leaderboard: %w", src.entityType, err)
				}
				written++
			}
		}
	}
	return written, nil
}

// leaderboardEntityID picks whichever id field the entity type populates.
func leaderboardEntityID(entityType string, e esi.FwLeaderboardEntry) int32 {
	switch entityType {
	case "faction":
		return e.FactionID
	case "character":
		return e.CharacterID
	case "corporation":
		return e.CorporationID
	}
	return 0
}
