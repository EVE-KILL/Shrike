package fitting

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DistributionBuckets = 24

var DistributionMetrics = []struct {
	Name, Column string
}{
	{"ehp", "ehp"}, {"dps", "dps_with_reload"}, {"alpha", "alpha"},
	{"repair", "GREATEST(COALESCE(shield_effective_boost,0),COALESCE(armor_effective_repair,0),COALESCE(hull_effective_repair,0),COALESCE(passive_shield_effective,0))"},
	{"speed", "max_velocity"}, {"align", "align_time"},
	{"signature", "signature_radius"}, {"capacitor", "cap_capacity"},
}

type DistributionRefresh struct {
	WindowDays int
	Summaries  int64
	Buckets    int64
	Elapsed    time.Duration
}

// RefreshDistributions rebuilds one complete observation-weighted window.
// Each canonical fit contributes once to fit_count and once per recorded loss
// to observation_count. Bounds use P1/P99 so isolated pathological fits do not
// flatten the useful portion of the histogram; outliers clamp into edge bins.
func RefreshDistributions(ctx context.Context, pool *pgxpool.Pool, days int) (DistributionRefresh, error) {
	started := time.Now()
	result := DistributionRefresh{WindowDays: days}
	if days < 1 || days > 3650 {
		return result, fmt.Errorf("distribution window must be between 1 and 3650 days")
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `DELETE FROM fitting_stat_distribution_summaries WHERE window_days=$1`, days); err != nil {
		return result, err
	}

	for _, metric := range DistributionMetrics {
		summarySQL := fmt.Sprintf(`
			WITH samples AS (
				SELECT fs.ship_type_id, fs.fit_hash, %s::double precision AS value, count(*)::bigint AS observations
				FROM fitting_stats fs
				JOIN killmail_fittings kf USING (fit_hash)
				WHERE kf.kill_time >= now() - make_interval(days => $1)
				  AND fs.engine_version=$3 AND fs.sde_version=$4 AND %s > 0
				GROUP BY fs.ship_type_id, fs.fit_hash, %s
			), weighted AS (
				SELECT samples.*,
					sum(observations) OVER (PARTITION BY ship_type_id ORDER BY value,fit_hash ROWS UNBOUNDED PRECEDING) cumulative,
					sum(observations) OVER (PARTITION BY ship_type_id) total_observations
				FROM samples
			), aggregate AS (
				SELECT ship_type_id,count(*)::int fit_count,max(total_observations)::bigint observation_count,
					min(value) minimum,max(value) maximum,
					min(value) FILTER (WHERE cumulative>=total_observations*0.01) p01,
					min(value) FILTER (WHERE cumulative>=total_observations*0.10) p10,
					min(value) FILTER (WHERE cumulative>=total_observations*0.25) p25,
					min(value) FILTER (WHERE cumulative>=total_observations*0.50) median,
					min(value) FILTER (WHERE cumulative>=total_observations*0.75) p75,
					min(value) FILTER (WHERE cumulative>=total_observations*0.90) p90,
					min(value) FILTER (WHERE cumulative>=total_observations*0.99) p99
				FROM weighted GROUP BY ship_type_id
			)
			INSERT INTO fitting_stat_distribution_summaries
				(ship_type_id,window_days,metric,fit_count,observation_count,minimum,maximum,p10,p25,median,p75,p90,lower_bound,upper_bound)
			SELECT ship_type_id,$1,$2,fit_count,observation_count,minimum,maximum,
				p10,p25,median,p75,p90,p01,p99
			FROM aggregate`, metric.Column, metric.Column, metric.Column)
		command, err := tx.Exec(ctx, summarySQL, days, metric.Name, DogmaEngineVersion, DogmaSDEVersion)
		if err != nil {
			return result, fmt.Errorf("summarize %s: %w", metric.Name, err)
		}
		result.Summaries += command.RowsAffected()

		bucketSQL := fmt.Sprintf(`
			WITH samples AS (
				SELECT fs.ship_type_id, fs.fit_hash, %s::double precision AS value, count(*)::bigint AS observations
				FROM fitting_stats fs JOIN killmail_fittings kf USING (fit_hash)
				WHERE kf.kill_time >= now() - make_interval(days => $1)
				  AND fs.engine_version=$3 AND fs.sde_version=$4 AND %s > 0
				GROUP BY fs.ship_type_id, fs.fit_hash, %s
			), assigned AS (
				SELECT s.*, summary.lower_bound lb, summary.upper_bound ub,
					CASE WHEN summary.upper_bound <= summary.lower_bound THEN 1 ELSE
					LEAST($5, GREATEST(1, width_bucket(s.value, summary.lower_bound, summary.upper_bound, $5))) END bucket
				FROM samples s JOIN fitting_stat_distribution_summaries summary
				  ON summary.ship_type_id=s.ship_type_id AND summary.window_days=$1 AND summary.metric=$2
			), aggregated AS (
				SELECT ship_type_id,bucket,lb,ub,count(*)::int fit_count,sum(observations)::bigint observation_count
				FROM assigned GROUP BY ship_type_id,bucket,lb,ub
			)
			INSERT INTO fitting_stat_distribution_buckets
				(ship_type_id,window_days,metric,bucket,lower_bound,upper_bound,fit_count,observation_count)
			SELECT summary.ship_type_id,$1,$2,series.bucket,
				CASE WHEN summary.upper_bound<=summary.lower_bound THEN summary.lower_bound ELSE summary.lower_bound+(series.bucket-1)*(summary.upper_bound-summary.lower_bound)/$5 END,
				CASE WHEN summary.upper_bound<=summary.lower_bound THEN summary.upper_bound ELSE summary.lower_bound+series.bucket*(summary.upper_bound-summary.lower_bound)/$5 END,
				coalesce(aggregated.fit_count,0),coalesce(aggregated.observation_count,0)
			FROM fitting_stat_distribution_summaries summary
			CROSS JOIN LATERAL generate_series(1,CASE WHEN summary.upper_bound<=summary.lower_bound THEN 1 ELSE $5 END) AS series(bucket)
			LEFT JOIN aggregated ON aggregated.ship_type_id=summary.ship_type_id AND aggregated.bucket=series.bucket
			WHERE summary.window_days=$1 AND summary.metric=$2`, metric.Column, metric.Column, metric.Column)
		command, err = tx.Exec(ctx, bucketSQL, days, metric.Name, DogmaEngineVersion, DogmaSDEVersion, DistributionBuckets)
		if err != nil {
			return result, fmt.Errorf("bucket %s: %w", metric.Name, err)
		}
		result.Buckets += command.RowsAffected()
	}
	if err := tx.Commit(ctx); err != nil {
		return result, err
	}
	result.Elapsed = time.Since(started)
	return result, nil
}
