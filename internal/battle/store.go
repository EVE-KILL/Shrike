package battle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Candidates and storage.

// Candidate is a system and window busy enough to be worth analysing.
type Candidate struct {
	SolarSystemID int32
	Kills         int
}

// HotspotWindow is one or more nearby UTC hours whose kill count passed the
// command's candidate threshold.
type HotspotWindow struct {
	SolarSystemID int32
	Start         time.Time
	End           time.Time
	Kills         int
}

// FindHotspotWindows reproduces the TypeScript backfill command's hourly
// candidate scan. Hot hours separated by at most one empty hour are merged.
func FindHotspotWindows(
	ctx context.Context,
	pool *pgxpool.Pool,
	from, to time.Time,
	minKills int,
	systemID int32,
) ([]HotspotWindow, error) {
	query := `
		SELECT solar_system_id,
		       date_trunc('hour', killmail_time) AS hour_bucket,
		       count(*)::int
		FROM killmails
		WHERE killmail_time >= $1 AND killmail_time < $2
		  AND is_npc = false
		  AND solar_system_id IS NOT NULL`
	args := []any{from, to, minKills}
	if systemID > 0 {
		query += " AND solar_system_id = $4"
		args = append(args, systemID)
	}
	rows, err := pool.Query(ctx, query+`
		GROUP BY solar_system_id, date_trunc('hour', killmail_time)
		HAVING count(*) >= $3
		ORDER BY solar_system_id, hour_bucket`,
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hours []HotspotWindow
	for rows.Next() {
		var h HotspotWindow
		if err := rows.Scan(&h.SolarSystemID, &h.Start, &h.Kills); err != nil {
			return nil, err
		}
		h.End = h.Start.Add(time.Hour)
		hours = append(hours, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mergeHotspotWindows(hours), nil
}

// ExtendWindow keeps an overlapping saved fight whole when a scan starts or
// ends in the middle of it.
func ExtendWindow(ctx context.Context, pool *pgxpool.Pool, systemID int32, from, to time.Time) (time.Time, time.Time, error) {
	err := pool.QueryRow(ctx, `
		SELECT LEAST($2, MIN(start_time)), GREATEST($3, MAX(end_time))
		FROM battles WHERE solar_system_id = $1 AND is_custom IS NOT TRUE
		  AND start_time < $3 AND end_time > $2`, systemID, from, to).Scan(&from, &to)
	return from, to, err
}

func mergeHotspotWindows(hours []HotspotWindow) []HotspotWindow {
	var out []HotspotWindow
	for _, hour := range hours {
		if len(out) == 0 {
			out = append(out, hour)
			continue
		}
		last := &out[len(out)-1]
		if last.SolarSystemID == hour.SolarSystemID &&
			hour.Start.Sub(last.End) <= time.Hour {
			last.End = hour.End
			last.Kills += hour.Kills
			continue
		}
		out = append(out, hour)
	}
	return out
}

// FindCandidates returns the systems with enough kills in a window to contain a
// battle.
//
// A cheap pre-filter: the detection itself loads every killmail and its
// attackers, which is far too expensive to run against every system in New Eden
// for every window. Requiring the minimum segment count over the whole window
// is a loose bound — a system can pass here and still have no active segment —
// but it turns thousands of systems into a handful.
func FindCandidates(ctx context.Context, pool *pgxpool.Pool, from, to any) ([]Candidate, error) {
	rows, err := pool.Query(ctx, `
        SELECT solar_system_id, count(*)
        FROM killmails
        WHERE killmail_time >= $1 AND killmail_time < $2
          AND is_npc IS NOT TRUE
          AND solar_system_id IS NOT NULL
        GROUP BY solar_system_id
        HAVING count(*) >= $3
        ORDER BY count(*) DESC`, from, to, MinKillsPerSegment)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Candidate
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.SolarSystemID, &c.Kills); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// LoadSystem reads one system's killmails and attackers for a window.
func LoadSystem(ctx context.Context, pool *pgxpool.Pool, systemID int32, from, to any) ([]Killmail, map[int64][]Attacker, error) {
	rows, err := pool.Query(ctx, `
        WITH latest_custom AS (
            SELECT DISTINCT ON (type_id) type_id, price
            FROM custom_prices
            ORDER BY type_id, date DESC
        ),
        latest_market AS (
            SELECT DISTINCT ON (type_id) type_id, average
            FROM prices
            WHERE region_id = 10000002
              AND type_id IN (SELECT type_id FROM latest_custom)
              AND date <= ($3::timestamptz AT TIME ZONE 'UTC')::date
            ORDER BY type_id, date DESC
        ),
        deltas AS (
            SELECT custom.type_id,
                   custom.price - coalesce(market.average, 0) AS delta
            FROM latest_custom custom
            LEFT JOIN latest_market market USING (type_id)
        )
        SELECT killmail.killmail_id,
               killmail.killmail_time,
               killmail.solar_system_id,
               coalesce(killmail.region_id, 0),
               coalesce(killmail.total_value, 0) + coalesce(delta.delta, 0),
               coalesce(killmail.victim_corporation_id, 0),
               coalesce(killmail.victim_alliance_id, 0),
               coalesce(killmail.victim_faction_id, 0),
               coalesce(killmail.victim_ship_type_id, 0)
        FROM killmails killmail
        LEFT JOIN deltas delta
          ON delta.type_id = killmail.victim_ship_type_id
        WHERE killmail.solar_system_id = $1
          AND killmail.killmail_time >= $2
          AND killmail.killmail_time < $3
          AND killmail.is_npc IS NOT TRUE
        ORDER BY killmail.killmail_time`,
		systemID,
		from,
		to,
	)
	if err != nil {
		return nil, nil, err
	}

	var kms []Killmail
	var ids []int64
	for rows.Next() {
		var k Killmail
		if err := rows.Scan(&k.KillmailID, &k.KillmailTime, &k.SolarSystemID, &k.RegionID,
			&k.TotalValue, &k.VictimCorporationID, &k.VictimAllianceID,
			&k.VictimFactionID, &k.VictimShipTypeID); err != nil {
			rows.Close()
			return nil, nil, err
		}
		kms = append(kms, k)
		ids = append(ids, k.KillmailID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(kms) == 0 {
		return nil, nil, nil
	}

	// One query for every attacker in the window rather than one per kill — a
	// large fight has thousands.
	arows, err := pool.Query(ctx, `
        SELECT killmail_id, coalesce(character_id, 0), coalesce(corporation_id, 0),
               coalesce(alliance_id, 0), coalesce(faction_id, 0),
               coalesce(damage_done, 0), coalesce(final_blow, false)
        FROM killmail_attackers WHERE killmail_id = ANY($1::bigint[])`, ids)
	if err != nil {
		return nil, nil, err
	}
	defer arows.Close()

	byKill := make(map[int64][]Attacker, len(ids))
	for arows.Next() {
		var a Attacker
		if err := arows.Scan(&a.KillmailID, &a.CharacterID, &a.CorporationID,
			&a.AllianceID, &a.FactionID, &a.DamageDone, &a.FinalBlow); err != nil {
			return nil, nil, err
		}
		byKill[a.KillmailID] = append(byKill[a.KillmailID], a)
	}
	return kms, byKill, arows.Err()
}

// ClearWindow removes auto-detected battles overlapping a window and
// reports their ids.
//
// Called before re-detecting the window, and it has to be the whole window
// rather than each battle as it is re-stored. A late killmail can move a
// fight's start earlier, and a delete keyed on the new start would match
// nothing and leave the old row behind — the same battle stored twice, minutes
// apart. Clearing first makes re-detection a replacement rather than an
// accumulation.
//
// The ids come back because each one may have an announcement in the ticker
// that now points at a battle that no longer exists.
//
// Custom battles are never touched: a person assembled those, and the detector
// does not get to overrule them.
func ClearWindow(ctx context.Context, pool *pgxpool.Pool, from, to any) ([]int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var ids []int64
	rows, err := tx.Query(ctx, `
        SELECT battle_id
        FROM battles
        WHERE start_time < $2
          AND end_time > $1
          AND is_custom IS NOT TRUE
        FOR UPDATE`,
		from,
		to,
	)
	if err != nil {
		return nil, fmt.Errorf("find battles in window: %w", err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := deleteBattles(ctx, tx, ids); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ids, nil
}

// HasOverlap reports whether a battle already occupies the same system and
// time span. The historical TS command skips those rather than replacing them.
func HasOverlap(
	ctx context.Context,
	pool *pgxpool.Pool,
	systemID int32,
	from, to time.Time,
) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM battles
			WHERE solar_system_id = $1
			  AND start_time < $3
			  AND end_time > $2
		)`, systemID, from, to).Scan(&exists)
	return exists, err
}

// Store updates a detected battle and its sides without changing its public ID.
// If new killmails join previously separate fights, their IDs become aliases.
func Store(ctx context.Context, pool *pgxpool.Pool, b *Detected, reports *queue.Client) (int64, error) {
	encoded, err := json.Marshal(b)
	if err != nil {
		return 0, err
	}
	fingerprint := sha256.Sum256(encoded)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	// Serialize readers preparing changes for the same system, including an
	// empty window where there is no existing row to lock.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1, $2)", 73117, b.SolarSystemID); err != nil {
		return 0, err
	}
	rows, err := tx.Query(ctx, `
        SELECT battle_id, detector_fingerprint
        FROM battles
        WHERE solar_system_id = $1
          AND start_time < $3
          AND end_time > $2
          AND is_custom IS NOT TRUE
        ORDER BY battle_id
        FOR UPDATE`,
		b.SolarSystemID,
		b.Start,
		b.End,
	)
	if err != nil {
		return 0, fmt.Errorf("find previous battle: %w", err)
	}
	var previous []int64
	var previousFingerprint []byte
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id, &previousFingerprint); err != nil {
			rows.Close()
			return 0, err
		}
		previous = append(previous, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(previous) == 1 && bytes.Equal(previousFingerprint, fingerprint[:]) {
		return previous[0], tx.Commit(ctx)
	}
	var battleID int64
	if len(previous) == 0 {
		err = tx.QueryRow(ctx, `
        INSERT INTO battles (
            solar_system_id, region_id, start_time, end_time, duration_minutes,
            kill_count, total_isk_destroyed, is_multi_party, is_custom, created_at, updated_at,
            detector_fingerprint
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,false,now(),now(),$9)
        RETURNING battle_id`,
			b.SolarSystemID, nullID(b.RegionID), b.Start, b.End, b.DurationMinutes,
			b.KillCount, b.IskDestroyed, b.MultiParty, fingerprint[:]).Scan(&battleID)
	} else {
		battleID = previous[0]
		err = updateDetectedBattle(ctx, tx, battleID, previous[1:], b, fingerprint[:])
	}
	if err != nil {
		return 0, fmt.Errorf("store battle: %w", err)
	}

	for index, team := range b.Teams {
		var teamID int64
		if err := tx.QueryRow(ctx, `
            INSERT INTO battle_teams (
                battle_id, team_index, total_kills, total_losses,
                total_isk_destroyed, total_isk_lost
            ) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
			battleID, index, team.Kills, team.Losses,
			team.IskDestroyed, team.IskLost).Scan(&teamID); err != nil {
			return 0, fmt.Errorf("store team %d: %w", index, err)
		}

		for _, e := range team.Entries {
			if _, err := tx.Exec(ctx, `
                INSERT INTO battle_team_members (
                    battle_team_id, corporation_id, alliance_id,
                    kills, losses, isk_destroyed, isk_lost
                ) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				teamID, e.CorporationID, nullID(e.AllianceID),
				e.Kills, e.Losses, e.IskDestroyed, e.IskLost); err != nil {
				return 0, fmt.Errorf("store team member: %w", err)
			}
		}
	}

	if err := queue.EnqueueBattleReportsTx(ctx, tx, reports, []int64{battleID}); err != nil {
		return 0, fmt.Errorf("queue battle report: %w", err)
	}
	return battleID, tx.Commit(ctx)
}

func updateDetectedBattle(ctx context.Context, tx pgx.Tx, id int64, merged []int64, b *Detected, fingerprint []byte) error {
	if _, err := tx.Exec(ctx, `
        UPDATE battles
        SET region_id = $2, start_time = $3, end_time = $4,
            duration_minutes = $5, kill_count = $6, total_isk_destroyed = $7,
            is_multi_party = $8, detector_fingerprint = $9,
            updated_at = clock_timestamp(), report_pending = true
        WHERE battle_id = $1`, id, nullID(b.RegionID), b.Start, b.End,
		b.DurationMinutes, b.KillCount, b.IskDestroyed, b.MultiParty, fingerprint); err != nil {
		return err
	}
	if err := deleteBattleTeams(ctx, tx, []int64{id}); err != nil {
		return err
	}
	if len(merged) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `
        UPDATE battle_aliases SET canonical_battle_id = $1
        WHERE canonical_battle_id = ANY($2::bigint[])`, id, merged); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
        INSERT INTO battle_aliases (battle_id, canonical_battle_id)
        SELECT alias, $1 FROM unnest($2::bigint[]) AS alias`, id, merged); err != nil {
		return err
	}
	return deleteBattles(ctx, tx, merged)
}

func deleteBattles(ctx context.Context, tx pgx.Tx, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if err := deleteBattleTeams(ctx, tx, ids); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "DELETE FROM battles WHERE battle_id = ANY($1::bigint[])", ids); err != nil {
		return fmt.Errorf("delete battles: %w", err)
	}
	return nil
}

func deleteBattleTeams(ctx context.Context, tx pgx.Tx, ids []int64) error {
	if _, err := tx.Exec(ctx, `
        DELETE FROM battle_team_members
        WHERE battle_team_id IN (
            SELECT id FROM battle_teams WHERE battle_id = ANY($1::bigint[])
        )`,
		ids,
	); err != nil {
		return fmt.Errorf("delete battle members: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM battle_teams WHERE battle_id = ANY($1::bigint[])`,
		ids,
	); err != nil {
		return fmt.Errorf("delete battle teams: %w", err)
	}
	return nil
}

func nullID(v int32) any {
	if v == 0 {
		return nil
	}
	return v
}
