// Package intelrollup materializes reusable, per-day character intelligence.
package intelrollup

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	RetentionDays = 365
	RepairDays    = 3
)

// Result describes a bounded reconciliation run.
type Result struct {
	From       time.Time
	To         time.Time
	Days       int64
	Characters int64
	Ships      int64
	Targets    int64
}

// Reconcile refreshes recent days and bootstraps the full retention window on
// first use. Each day commits atomically so a timeout never exposes partial
// facts and the next run can resume from its coverage markers.
func Reconcile(ctx context.Context, pool *pgxpool.Pool) (Result, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	dates, err := reconcileDates(ctx, pool, today)
	if err != nil {
		return Result{}, err
	}
	result := Result{To: today}
	if len(dates) > 0 {
		result.From = dates[0]
	}
	for _, day := range dates {
		characters, ships, targets, err := reconcileDay(ctx, pool, day)
		if err != nil {
			return result, err
		}
		result.Days++
		result.Characters += characters
		result.Ships += ships
		result.Targets += targets
	}
	for _, table := range []string{
		"character_intel_daily", "character_intel_ship_daily",
		"character_intel_target_daily", "character_intel_rollup_days",
	} {
		if _, err := pool.Exec(ctx, "DELETE FROM "+table+
			" WHERE activity_date < CURRENT_DATE - $1::int", RetentionDays-1); err != nil {
			return result, fmt.Errorf("purge %s: %w", table, err)
		}
	}
	return result, nil
}

func reconcileDates(ctx context.Context, pool *pgxpool.Pool, today time.Time) ([]time.Time, error) {
	rows, err := pool.Query(ctx, `
		SELECT candidate::date
		FROM generate_series($1::date, $2::date, interval '1 day') candidate
		LEFT JOIN character_intel_rollup_days covered ON covered.activity_date=candidate::date
		WHERE covered.activity_date IS NULL OR candidate::date >= $3::date
		ORDER BY candidate`, today.AddDate(0, 0, -(RetentionDays-1)), today,
		today.AddDate(0, 0, -(RepairDays-1)))
	if err != nil {
		return nil, fmt.Errorf("inspect intel rollup coverage: %w", err)
	}
	defer rows.Close()
	dates := []time.Time{}
	for rows.Next() {
		var day time.Time
		if err := rows.Scan(&day); err != nil {
			return nil, err
		}
		dates = append(dates, day)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return dates, nil
}

func reconcileDay(ctx context.Context, pool *pgxpool.Pool, day time.Time) (int64, int64, int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, table := range []string{"character_intel_daily", "character_intel_ship_daily", "character_intel_target_daily"} {
		if _, err := tx.Exec(ctx, "DELETE FROM "+table+" WHERE activity_date = $1", day); err != nil {
			return 0, 0, 0, fmt.Errorf("clear %s for %s: %w", table, day.Format("2006-01-02"), err)
		}
	}

	characterTag, err := tx.Exec(ctx, characterDailySQL, day)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("roll up character intel for %s: %w", day.Format("2006-01-02"), err)
	}
	shipTag, err := tx.Exec(ctx, shipDailySQL, day)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("roll up character ships for %s: %w", day.Format("2006-01-02"), err)
	}
	targetTag, err := tx.Exec(ctx, targetDailySQL, day)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("roll up character targets for %s: %w", day.Format("2006-01-02"), err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO character_intel_rollup_days (activity_date, refreshed_at)
		VALUES ($1, now()) ON CONFLICT (activity_date) DO UPDATE SET refreshed_at = EXCLUDED.refreshed_at`, day); err != nil {
		return 0, 0, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, 0, err
	}
	return characterTag.RowsAffected(), shipTag.RowsAffected(), targetTag.RowsAffected(), nil
}

const characterDailySQL = `
WITH bounds AS (SELECT $1::date AS d, $1::date + 1 AS e),
attacker AS MATERIALIZED (
 SELECT a.character_id,
  count(*)::int AS appearances,
  count(*) FILTER (WHERE k.attacker_count=1)::int AS solo,
  count(*) FILTER (WHERE k.attacker_count BETWEEN 2 AND 5)::int AS small_gang,
  count(*) FILTER (WHERE k.attacker_count BETWEEN 6 AND 15)::int AS mid_gang,
  count(*) FILTER (WHERE k.attacker_count BETWEEN 16 AND 50)::int AS fleet,
  count(*) FILTER (WHERE k.attacker_count>50)::int AS blob,
  sum(k.attacker_count)::bigint AS sum_attacker_count,
  count(*) FILTER (WHERE a.ship_type_id=45534)::int AS monitor_appearances,
  count(*) FILTER (WHERE a.damage_done=0 AND k.attacker_count>=10)::int AS zero_damage_fleet,
  count(*) FILTER (WHERE s.security>=0.5)::int AS highsec,
  count(*) FILTER (WHERE s.security>0 AND s.security<0.5)::int AS lowsec,
  count(*) FILTER (WHERE s.security<=0)::int AS nullsec,
  count(*) FILTER (WHERE s.security>=0.5 AND a.security_status < -5)::int AS gank_appearances,
  count(*) FILTER (WHERE a.alliance_id IS NOT NULL AND a.alliance_id=k.victim_alliance_id
                    AND a.corporation_id<>k.victim_corporation_id)::int AS awox_kills
 FROM killmail_attackers a JOIN killmails k USING (killmail_id)
 LEFT JOIN solar_systems s ON s.solar_system_id=k.solar_system_id, bounds b
 WHERE a.character_id IS NOT NULL AND a.killmail_time>=b.d AND a.killmail_time<b.e
 GROUP BY a.character_id
), deaths AS MATERIALIZED (
 SELECT k.victim_character_id AS character_id, count(*)::int AS losses,
  count(*) FILTER (WHERE k.total_value<50000000)::int AS cheap_deaths,
  count(*) FILTER (WHERE EXISTS (SELECT 1 FROM killmail_items i WHERE i.killmail_id=k.killmail_id
                    AND i.type_id IN (21096,28646,52694) AND i.flag_id BETWEEN 11 AND 34))::int AS cyno_losses,
  count(*) FILTER (WHERE k.total_value<50000000 AND EXISTS (
    SELECT 1 FROM killmails f WHERE f.solar_system_id=k.solar_system_id
      AND f.killmail_time BETWEEN k.killmail_time AND k.killmail_time+interval '5 minutes'
      AND f.killmail_id<>k.killmail_id AND f.attacker_count>=2))::int AS baited_deaths
 FROM killmails k, bounds b WHERE k.victim_character_id IS NOT NULL
  AND k.killmail_time>=b.d AND k.killmail_time<b.e GROUP BY k.victim_character_id
), ids AS (SELECT character_id FROM attacker UNION SELECT character_id FROM deaths)
INSERT INTO character_intel_daily
SELECT i.character_id,$1::date,coalesce(a.appearances,0),coalesce(a.solo,0),coalesce(a.small_gang,0),
 coalesce(a.mid_gang,0),coalesce(a.fleet,0),coalesce(a.blob,0),coalesce(a.sum_attacker_count,0),
 coalesce(a.monitor_appearances,0),coalesce(a.zero_damage_fleet,0),coalesce(a.highsec,0),
 coalesce(a.lowsec,0),coalesce(a.nullsec,0),coalesce(a.gank_appearances,0),coalesce(a.awox_kills,0),
 coalesce(d.losses,0),coalesce(d.cyno_losses,0),coalesce(d.cheap_deaths,0),coalesce(d.baited_deaths,0)
FROM ids i LEFT JOIN attacker a USING(character_id) LEFT JOIN deaths d USING(character_id)`

const shipDailySQL = `
WITH appearances AS (
 SELECT a.character_id,a.ship_type_id,count(*)::int AS appearances,max(a.killmail_time) AS last_appearance_at
 FROM killmail_attackers a WHERE a.character_id IS NOT NULL AND a.ship_type_id IS NOT NULL
  AND a.killmail_time >= $1::date AND a.killmail_time < $1::date+1 GROUP BY a.character_id,a.ship_type_id
), losses AS (
 SELECT DISTINCT ON (k.victim_character_id,k.victim_ship_type_id)
  k.victim_character_id AS character_id,k.victim_ship_type_id AS ship_type_id,
  count(*) OVER (PARTITION BY k.victim_character_id,k.victim_ship_type_id)::int AS losses,
  k.killmail_id AS last_loss_id,k.killmail_time AS last_loss_at
 FROM killmails k WHERE k.victim_character_id IS NOT NULL AND k.victim_ship_type_id IS NOT NULL
  AND k.killmail_time >= $1::date AND k.killmail_time < $1::date+1
 ORDER BY k.victim_character_id,k.victim_ship_type_id,k.killmail_time DESC,k.killmail_id DESC
), combined AS (
 SELECT coalesce(a.character_id,l.character_id) AS character_id,coalesce(a.ship_type_id,l.ship_type_id) AS ship_type_id,
  coalesce(a.appearances,0) AS appearances,coalesce(l.losses,0) AS losses,a.last_appearance_at,l.last_loss_id,l.last_loss_at
 FROM appearances a FULL JOIN losses l USING(character_id,ship_type_id)
)
INSERT INTO character_intel_ship_daily
SELECT character_id,$1::date,ship_type_id,appearances,losses,last_appearance_at,last_loss_id,last_loss_at FROM combined`

const targetDailySQL = `
INSERT INTO character_intel_target_daily
SELECT a.character_id,$1::date,k.victim_alliance_id,count(*)::int
FROM killmail_attackers a JOIN killmails k USING(killmail_id)
WHERE a.character_id IS NOT NULL AND k.victim_alliance_id IS NOT NULL
 AND a.killmail_time >= $1::date AND a.killmail_time < $1::date+1
GROUP BY a.character_id,k.victim_alliance_id`
