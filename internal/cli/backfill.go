package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/achievements"
	"github.com/eve-kill/shrike/internal/campaign"
	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/stats"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/spf13/cobra"
)

// Backfills.
//
// Every derived subsystem — stats, achievements, fittings, the graph — is built
// forward from killmails as they arrive. A backfill replays history through the
// same path, which is what makes a new subsystem usable on day one instead of
// only on data that arrived after it shipped.
//
// They are all the same shape: walk a range of killmails, enqueue one job each.
// Deliberately enqueuing rather than computing inline — the workers already
// know how to do the work, they retry, they respect the rate limits, and the
// progress is visible in `queue:status` instead of trapped in a terminal.

var backfillCmd = &cobra.Command{
	Use:   "backfill",
	Short: "Replay history through the derived subsystems",
	Long: `Rebuilds what is computed from killmails rather than stored on them.

Each backfill enqueues jobs for the workers to run, so progress is visible in
queue:status and the work survives the command exiting. Nothing here recomputes
inline.

Ranges default to everything. Narrow with --from and --to when rebuilding after
a fix that only affects recent data.`,
}

var (
	flagBackfillFrom  string
	flagBackfillTo    string
	flagBackfillLimit int
	flagBackfillApply bool
)

// backfillRange resolves the requested window, defaulting to all of history.
func backfillRange() (time.Time, time.Time, error) {
	from := time.Date(2007, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Now().UTC().AddDate(0, 0, 1)

	if flagBackfillFrom != "" {
		p, err := time.Parse("2006-01-02", flagBackfillFrom)
		if err != nil {
			return from, to, fmt.Errorf("invalid --from %q, want YYYY-MM-DD", flagBackfillFrom)
		}
		from = p
	}
	if flagBackfillTo != "" {
		p, err := time.Parse("2006-01-02", flagBackfillTo)
		if err != nil {
			return from, to, fmt.Errorf("invalid --to %q, want YYYY-MM-DD", flagBackfillTo)
		}
		to = p
	}
	if !to.After(from) {
		return from, to, fmt.Errorf("--to (%s) is not after --from (%s)",
			to.Format("2006-01-02"), from.Format("2006-01-02"))
	}
	return from, to, nil
}

// enqueueByKillmail walks a range and enqueues one job per killmail.
//
// Batched in chunks rather than read into memory: a full backfill is hundreds
// of millions of rows, and both the id list and the job inserts have to be
// bounded. The cursor is the killmail id, so a run that is interrupted can be
// resumed from where it stopped rather than starting again.
func enqueueByKillmail(ctx context.Context, pool *pgxpool.Pool, c *queue.Client,
	from, to time.Time, limit int, build func(int64) river.JobArgs) (int64, error) {

	const chunk = 5000
	var cursor int64
	var total int64

	for {
		rows, err := pool.Query(ctx, `
            SELECT killmail_id FROM killmails
            WHERE killmail_time >= $1 AND killmail_time < $2 AND killmail_id > $3
            ORDER BY killmail_id
            LIMIT $4`, from, to, cursor, chunk)
		if err != nil {
			return total, err
		}

		var batch []river.JobArgs
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return total, err
			}
			batch = append(batch, build(id))
			cursor = id
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return total, err
		}
		if len(batch) == 0 {
			return total, nil
		}

		// Dormant backfill: this is work about the past and must never delay a
		// killmail arriving now.
		n, err := queue.DispatchMany(ctx, c, batch, queue.DormantBackfill)
		if err != nil {
			return total, err
		}
		total += int64(n)

		if limit > 0 && total >= int64(limit) {
			return total, nil
		}
	}
}

// killmailBackfill builds one of the per-killmail backfill commands.
//
// Five commands that differ only in which job they enqueue, so the shape is
// written once — the alternative is five near-identical files that drift.
func killmailBackfill(name, short, long string, build func(int64) river.JobArgs) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		Long:  long,
		RunE: func(cmd *cobra.Command, _ []string) error {
			from, to, err := backfillRange()
			if err != nil {
				return err
			}

			pool, err := openPool(cmd)
			if err != nil {
				return err
			}
			defer pool.Close()

			var pending int64
			if err := pool.QueryRow(cmd.Context(), `
                SELECT count(*) FROM killmails
                WHERE killmail_time >= $1 AND killmail_time < $2`, from, to).Scan(&pending); err != nil {
				return err
			}

			ui.Section("Backfill " + name)
			ui.KV("Range", from.Format("2006-01-02")+" .. "+to.Format("2006-01-02"))
			ui.KV("Killmails", fmtCount(pending))

			// A full backfill enqueues millions of jobs and takes days to
			// drain. Requiring --apply means nobody starts one by exploring.
			if !flagBackfillApply {
				ui.Newline()
				fmt.Printf("  %s\n", ui.Dim("dry run — pass --apply to enqueue"))
				ui.Newline()
				return nil
			}

			client, err := queue.New(queue.Options{Pool: pool})
			if err != nil {
				return err
			}

			n, err := enqueueByKillmail(cmd.Context(), pool, client, from, to, flagBackfillLimit, build)
			if err != nil {
				return err
			}

			ui.Newline()
			ui.Success("Enqueued %s jobs. Watch progress with queue:status.", fmtCount(n))
			ui.Newline()
			return nil
		},
	}
}

var (
	flagStatsFromMonth       string
	flagStatsToMonth         string
	flagStatsTable           string
	flagStatsEntity          string
	flagStatsReset           bool
	flagStatsReverse         bool
	flagStatsDailyCutoffDays int
	flagStatsSkipAggregation bool
	flagStatsSkipRollup      bool
)

var backfillStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Authoritatively rebuild aggregate counters from killmails",
	Long: `Populates stats and stats_breakdowns from killmail history.

Old months are written directly at monthly granularity, recent months at daily
granularity, and monthly/yearly rollups are then rebuilt. Existing rows are
refused unless --reset is supplied, because aggregation is additive.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		from, err := parseMonth(flagStatsFromMonth, "--from")
		if err != nil {
			return err
		}
		toValue := flagStatsToMonth
		if toValue == "" {
			toValue = time.Now().UTC().Format("2006-01")
		}
		to, err := parseMonth(toValue, "--to")
		if err != nil {
			return err
		}

		wantStats, wantBreakdowns := true, true
		switch flagStatsTable {
		case "":
		case "stats":
			wantBreakdowns = false
		case "breakdowns":
			wantStats = false
		default:
			return fmt.Errorf("invalid --table %q: want stats or breakdowns", flagStatsTable)
		}
		var entityTypes []stats.EntityType
		entityName := strings.ToLower(strings.TrimSpace(flagStatsEntity))
		if entityName != "" && entityName != "all" {
			entityType, ok := stats.ParseEntityType(entityName)
			if !ok {
				return fmt.Errorf(
					"invalid --entity %q: want character, corporation, alliance, "+
						"ship, system, constellation, region, faction, or all",
					flagStatsEntity,
				)
			}
			entityTypes = []stats.EntityType{entityType}
		}

		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		ui.Section("Backfill stats")
		if len(entityTypes) == 0 {
			ui.KV("Entities", "all")
		} else {
			ui.KV("Entities", entityName)
		}
		start := time.Now()
		res, err := stats.Backfill(cmd.Context(), pool, stats.BackfillOptions{
			FromMonth:       from,
			ToMonth:         to,
			DailyCutoff:     time.Duration(flagStatsDailyCutoffDays) * 24 * time.Hour,
			EntityTypes:     entityTypes,
			WantStats:       wantStats,
			WantBreakdowns:  wantBreakdowns,
			Reset:           flagStatsReset,
			Reverse:         flagStatsReverse,
			SkipAggregation: flagStatsSkipAggregation,
			SkipRollup:      flagStatsSkipRollup,
		})
		if err != nil {
			return err
		}
		if ui.JSONMode {
			return ui.JSON(res)
		}
		ui.KV("Months", fmt.Sprint(res.Months))
		ui.KV("Killmails", fmtCount(res.Killmails))
		ui.KV("Stats rows", fmtCount(res.Stats+res.MonthlyStats+res.YearlyStats))
		ui.KV("Breakdown rows", fmtCount(res.Breakdowns+res.MonthlyBreakdowns+res.YearlyBreakdowns))
		ui.Newline()
		ui.Success("Stats backfill finished in %s.", time.Since(start).Round(time.Millisecond))
		return nil
	},
}

var (
	flagFittingsDays   int
	flagFittingsFromID int64
	flagGraphDays      int
	flagGraphClear     bool
)

var backfillFittingsCmd = &cobra.Command{
	Use:   "fittings",
	Short: "Rebuild fit identities from recent killmails",
	Long: `Enqueues fit extraction for killmails in the requested recent window.

The default is the same 90-day window as the TypeScript command. --from is a
killmail id cursor, not a date.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if flagFittingsDays < 1 {
			return fmt.Errorf("--days must be positive")
		}
		if flagFittingsFromID < 0 {
			return fmt.Errorf("--from cannot be negative")
		}

		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		client, err := queue.New(queue.Options{Pool: pool})
		if err != nil {
			return err
		}
		n, err := enqueueFittings(
			cmd.Context(), pool, client,
			time.Now().UTC().AddDate(0, 0, -flagFittingsDays),
			flagFittingsFromID,
		)
		if err != nil {
			return err
		}
		ui.Success("Enqueued %s fit-extraction jobs.", fmtCount(n))
		return nil
	},
}

func enqueueFittings(
	ctx context.Context,
	pool *pgxpool.Pool,
	client *queue.Client,
	cutoff time.Time,
	fromID int64,
) (int64, error) {
	const chunk = 5_000
	cursor := fromID
	var total int64
	for {
		rows, err := pool.Query(ctx, `
			SELECT killmail_id
			FROM killmails
			WHERE killmail_time >= $1
			  AND killmail_id >= $2
			  AND victim_ship_type_id IS NOT NULL
			ORDER BY killmail_id
			LIMIT $3`, cutoff, cursor, chunk)
		if err != nil {
			return total, err
		}

		var jobs []river.JobArgs
		var last int64
		for rows.Next() {
			if err := rows.Scan(&last); err != nil {
				rows.Close()
				return total, err
			}
			jobs = append(jobs, queue.FitExtractArgs{KillmailID: last})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return total, err
		}
		if len(jobs) == 0 {
			return total, nil
		}

		n, err := queue.DispatchMany(ctx, client, jobs, queue.DormantBackfill)
		if err != nil {
			return total, err
		}
		total += int64(n)
		cursor = last + 1
		if len(jobs) < chunk {
			return total, nil
		}
	}
}

var backfillGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Rebuild the relationship graph from recent killmails",
	Long: `Queues graph ingestion for the requested recent window.

Use --clear when upgrading a graph created before idempotent killmail markers
were introduced. Once markers exist, repeated backfills safely skip killmails
that are already represented in the graph.

Stop graph-ingest workers before using --clear. The command clears the derived
graph, installs its schema while it is empty, and only then queues the rebuild.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		if flagGraphDays < 1 {
			return fmt.Errorf("--days must be positive")
		}
		if flagGraphClear {
			d, err := deps(cmd.Context(), pool, false)
			if err != nil {
				return err
			}
			if d.Graph == nil {
				return fmt.Errorf("memgraph is unreachable")
			}
			if err := d.Graph.Clear(cmd.Context()); err != nil {
				return err
			}
			if err := d.Graph.EnsureSchema(cmd.Context()); err != nil {
				return err
			}
		}

		client, err := queue.New(queue.Options{Pool: pool})
		if err != nil {
			return err
		}
		from := time.Now().UTC().AddDate(0, 0, -flagGraphDays)
		to := time.Now().UTC().AddDate(0, 0, 1)
		n, err := enqueueByKillmail(
			cmd.Context(), pool, client, from, to, 0,
			func(id int64) river.JobArgs { return queue.GraphIngestArgs{KillmailID: id} },
		)
		if err != nil {
			return err
		}
		ui.Success("Enqueued %s graph-ingest jobs.", fmtCount(n))
		return nil
	},
}

var (
	flagAchievementID             string
	flagAchievementCategory       string
	flagAchievementSyncPointsOnly bool
)

var backfillAchievementsCmd = &cobra.Command{
	Use:   "achievements",
	Short: "Authoritatively rebuild character achievements",
	Long: `Computes achievement counters from stats and killmail source tables.

This converges on the same rows when re-run. It does not replay the live award
path, because replaying additive counters would award everything twice.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()
		if flagAchievementSyncPointsOnly {
			ui.Section("Sync achievement points")
			characters, err := achievements.SyncPoints(cmd.Context(), pool)
			if err != nil {
				return err
			}
			if ui.JSONMode {
				return ui.JSON(map[string]any{"characters": characters})
			}
			ui.KV("Characters resynced", fmtCount(characters))
			ui.Newline()
			ui.Success("Achievement points synchronized.")
			return nil
		}

		definitions := achievements.Filter(flagAchievementID, flagAchievementCategory)

		ui.Section("Backfill achievements")
		ui.KV("Definitions", fmt.Sprint(len(definitions)))
		res, err := achievements.Rebuild(
			cmd.Context(), pool, definitions,
			flagAchievementID == "" && flagAchievementCategory == "",
		)
		if err != nil {
			return err
		}
		if ui.JSONMode {
			return ui.JSON(res)
		}
		ui.KV("Rows upserted", fmtCount(res.Rows))
		ui.KV("Stale rows removed", fmtCount(res.Removed))
		if flagAchievementID == "" && flagAchievementCategory == "" {
			ui.KV("Characters resynced", fmtCount(res.Characters))
		}
		ui.Newline()
		ui.Success("Achievement backfill complete.")
		return nil
	},
}

func parseMonth(value, flag string) (time.Time, error) {
	month, err := time.Parse("2006-01", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s %q, want YYYY-MM", flag, value)
	}
	return month, nil
}

var backfillKillsDailyCountCmd = &cobra.Command{
	Use:   "kills-daily-count",
	Short: "Rebuild the daily kill-count rollup",
	Long: `Recomputes kills_daily_count from the killmails themselves.

	Unlike the other backfills this runs inline rather than through the queue: it is
one INSERT … SELECT per kill type, not a job per killmail, and it overwrites
rather than accumulating — so it is safe to re-run.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		from, err := parseMonth(flagDailyCountFromMonth, "--from")
		if err != nil {
			return err
		}
		toValue := flagDailyCountToMonth
		if toValue == "" {
			toValue = time.Now().UTC().Format("2006-01")
		}
		to, err := parseMonth(toValue, "--to")
		if err != nil {
			return err
		}
		months := inclusiveMonths(from, to, flagDailyCountReverse)

		predicates := killtype.Predicates()
		if flagDailyCountType != "" {
			predicate, ok := predicates[flagDailyCountType]
			if !ok {
				return fmt.Errorf("unknown kill type %q", flagDailyCountType)
			}
			predicates = map[string]string{flagDailyCountType: predicate}
		}

		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		ui.Section("Backfill kills-daily-count")
		if flagDailyCountReset {
			if flagDailyCountType == "" {
				if _, err := pool.Exec(cmd.Context(), `TRUNCATE kills_daily_count`); err != nil {
					return err
				}
			} else if _, err := pool.Exec(cmd.Context(),
				`DELETE FROM kills_daily_count WHERE type = $1`, flagDailyCountType); err != nil {
				return err
			}
		}

		var total int64
		for _, month := range months {
			next := month.AddDate(0, 1, 0)
			for kind, predicate := range predicates {
				tx, err := pool.Begin(cmd.Context())
				if err != nil {
					return fmt.Errorf("begin rebuild %s for %s: %w", kind, month.Format("2006-01"), err)
				}

				if _, err := tx.Exec(cmd.Context(), `
					DELETE FROM kills_daily_count
					WHERE date >= $1::date
					  AND date < $2::date
					  AND type = $3`, month, next, kind); err != nil {
					_ = tx.Rollback(cmd.Context())
					return fmt.Errorf("clear %s for %s: %w", kind, month.Format("2006-01"), err)
				}

				tag, err := tx.Exec(cmd.Context(), fmt.Sprintf(`
					INSERT INTO kills_daily_count (date, type, count)
					SELECT (k.killmail_time AT TIME ZONE 'UTC')::date, $1, count(*)
					FROM killmails k
					WHERE k.killmail_time >= $2 AND k.killmail_time < $3
					  AND %s
					GROUP BY (k.killmail_time AT TIME ZONE 'UTC')::date
					ON CONFLICT (date, type) DO UPDATE SET count = EXCLUDED.count`,
					predicate), kind, month, next)
				if err != nil {
					_ = tx.Rollback(cmd.Context())
					return fmt.Errorf("rebuild %s for %s: %w", kind, month.Format("2006-01"), err)
				}
				if err := tx.Commit(cmd.Context()); err != nil {
					return fmt.Errorf("commit %s for %s: %w", kind, month.Format("2006-01"), err)
				}
				total += tag.RowsAffected()
			}
		}
		ui.Newline()
		ui.Success("Rebuilt %s (date, type) rows.", fmtCount(total))
		ui.Newline()
		return nil
	},
}

var (
	flagDailyCountFromMonth string
	flagDailyCountToMonth   string
	flagDailyCountType      string
	flagDailyCountReset     bool
	flagDailyCountReverse   bool
)

func inclusiveMonths(from, to time.Time, reverse bool) []time.Time {
	var months []time.Time
	for month := from; !month.After(to); month = month.AddDate(0, 1, 0) {
		months = append(months, month)
	}
	if reverse {
		for left, right := 0, len(months)-1; left < right; left, right = left+1, right-1 {
			months[left], months[right] = months[right], months[left]
		}
	}
	return months
}

var campaignProcessCmd = &cobra.Command{
	Use:   "process <campaign-id|pending|active|all>",
	Short: "Recompute campaign aggregates now",
	Long: `Recomputes one or more campaigns in the foreground.

Bypasses the hourly gate check, so it recomputes whether or not anything has
changed — which is what you want after editing campaign sides. The pending,
active and all selectors process every matching campaign without a row cap.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		var ids []string
		switch args[0] {
		case "pending":
			ids, err = campaignIDsByStatus(cmd.Context(), pool, &[]int16{campaign.StatusPending}[0])
		case "active":
			ids, err = campaignIDsByStatus(cmd.Context(), pool, &[]int16{campaign.StatusActive}[0])
		case "all":
			ids, err = campaignIDsByStatus(cmd.Context(), pool, nil)
		default:
			ids = []string{args[0]}
		}
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			ui.Section("Campaign processing")
			ui.KV("Campaigns", "0")
			return nil
		}

		results := make([]*campaign.Result, 0, len(ids))
		for _, id := range ids {
			res, err := campaign.Process(cmd.Context(), pool, id)
			if err != nil {
				return err
			}
			if res == nil {
				ui.Warn("Campaign %s does not exist.", id)
				continue
			}
			results = append(results, res)
			if !ui.JSONMode {
				ui.Section("Campaign " + res.CampaignID)
				ui.KV("Killmails", fmtCount(res.Killmails))
				ui.KV("Sides", fmt.Sprint(res.Sides))
				ui.KV("Through", res.Through.Format(time.RFC3339))
			}
		}

		if ui.JSONMode {
			return ui.JSON(results)
		}
		ui.Newline()
		ui.Success("Processed %s campaigns.", fmtCount(int64(len(results))))
		return nil
	},
}

func campaignIDsByStatus(
	ctx context.Context,
	pool *pgxpool.Pool,
	status *int16,
) ([]string, error) {
	query := `SELECT campaign_id FROM campaigns`
	var args []any
	if status != nil {
		query += ` WHERE status = $1`
		args = append(args, *status)
	}
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

var campaignCmd = &cobra.Command{
	Use:   "campaign",
	Short: "Campaign maintenance",
}

var (
	flagDBVacuumFull  bool
	flagDBVacuumTable string
)

var dbVacuumCmd = &cobra.Command{
	Use:   "vacuum",
	Short: "Reclaim space and refresh planner statistics",
	Long: `Runs VACUUM ANALYZE over the large tables.

Autovacuum's thresholds are proportional, so on a table with hundreds of
millions of rows it waits for tens of millions of dead tuples before acting.
This is the manual nudge for after a large delete — a purge, a backfill that was
re-run, a stats truncate.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		var tables []string
		if flagDBVacuumTable != "" {
			tables = []string{flagDBVacuumTable}
		} else {
			rows, err := pool.Query(cmd.Context(), `
				SELECT tablename
				FROM pg_tables
				WHERE schemaname = 'public'
				ORDER BY tablename`)
			if err != nil {
				return err
			}
			for rows.Next() {
				var table string
				if err := rows.Scan(&table); err != nil {
					rows.Close()
					return err
				}
				tables = append(tables, table)
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				return err
			}
		}

		ui.Section("Vacuum")
		if flagDBVacuumFull {
			ui.Warn("VACUUM FULL locks each table until it completes.")
		}
		vacuum := "VACUUM ANALYZE "
		if flagDBVacuumFull {
			vacuum = "VACUUM FULL ANALYZE "
		}
		for _, t := range tables {
			start := time.Now()
			identifier := pgx.Identifier{t}.Sanitize()
			if _, err := pool.Exec(cmd.Context(), vacuum+identifier); err != nil {
				fmt.Printf("  %s %s %s — %v\n", ui.Warn2("✗"), ui.Command(t),
					ui.Dim(time.Since(start).Round(time.Millisecond).String()), err)
				continue
			}
			fmt.Printf("  %s %s %s\n", ui.Primary("✓"), ui.Command(t),
				ui.Dim(time.Since(start).Round(time.Millisecond).String()))
		}

		var database string
		if err := pool.QueryRow(cmd.Context(), `SELECT current_database()`).Scan(&database); err != nil {
			return err
		}
		if _, err := pool.Exec(cmd.Context(),
			"REINDEX DATABASE CONCURRENTLY "+pgx.Identifier{database}.Sanitize()); err != nil {
			return fmt.Errorf("reindex database: %w", err)
		}
		ui.Newline()
		ui.Success("Vacuum pass finished for %d tables and the database was reindexed.", len(tables))
		ui.Newline()
		return nil
	},
}

func init() {
	backfillStatsCmd.Flags().StringVarP(&flagStatsFromMonth, "from", "f", "2007-12", "Start month (YYYY-MM)")
	backfillStatsCmd.Flags().StringVarP(&flagStatsToMonth, "to", "t", "", "End month, inclusive (YYYY-MM; default current)")
	backfillStatsCmd.Flags().StringVar(&flagStatsTable, "table", "", "Target only stats or breakdowns")
	backfillStatsCmd.Flags().StringVar(&flagStatsEntity, "entity", "all", "Target one entity type or all")
	backfillStatsCmd.Flags().BoolVar(&flagStatsReset, "reset", false, "Clear selected entity rows before rebuilding")
	backfillStatsCmd.Flags().BoolVar(&flagStatsReverse, "reverse", false, "Process newest months first")
	backfillStatsCmd.Flags().IntVar(&flagStatsDailyCutoffDays, "daily-cutoff-days", 365, "Recent daily-retention window")
	backfillStatsCmd.Flags().BoolVar(&flagStatsSkipAggregation, "skip-aggregation", false, "Only rebuild rollups")
	backfillStatsCmd.Flags().BoolVar(&flagStatsSkipRollup, "skip-rollup", false, "Skip monthly and yearly rollups")

	backfillAchievementsCmd.Flags().StringVarP(&flagAchievementID, "achievement", "a", "", "Only process one achievement ID")
	backfillAchievementsCmd.Flags().StringVar(&flagAchievementCategory, "category", "", "Only process one category")
	backfillAchievementsCmd.Flags().BoolVar(&flagAchievementSyncPointsOnly, "sync-points-only", false, "Only synchronize denormalized character achievement points")

	backfillFittingsCmd.Flags().IntVarP(&flagFittingsDays, "days", "d", 90, "How many days back to process")
	backfillFittingsCmd.Flags().Int64Var(&flagFittingsFromID, "from", 0, "Start killmail ID cursor")
	backfillGraphCmd.Flags().IntVarP(&flagGraphDays, "days", "d", 90, "Days to backfill")
	backfillGraphCmd.Flags().BoolVar(&flagGraphClear, "clear", false, "Clear the graph before backfill")
	backfillKillsDailyCountCmd.Flags().StringVarP(&flagDailyCountFromMonth, "from", "f", "2007-12", "Start month (YYYY-MM)")
	backfillKillsDailyCountCmd.Flags().StringVarP(&flagDailyCountToMonth, "to", "t", "", "End month, inclusive (YYYY-MM; default current)")
	backfillKillsDailyCountCmd.Flags().StringVar(&flagDailyCountType, "type", "", "Only rebuild one kill type")
	backfillKillsDailyCountCmd.Flags().BoolVar(&flagDailyCountReset, "reset", false, "Clear selected rows before rebuilding")
	backfillKillsDailyCountCmd.Flags().BoolVarP(&flagDailyCountReverse, "reverse", "r", false, "Process newest months first")
	dbVacuumCmd.Flags().BoolVar(&flagDBVacuumFull, "full", false, "Run VACUUM FULL (locks tables)")
	dbVacuumCmd.Flags().StringVarP(&flagDBVacuumTable, "table", "t", "", "Only vacuum one table")

	backfillCmd.AddCommand(backfillStatsCmd, backfillFittingsCmd, backfillGraphCmd,
		backfillAchievementsCmd, backfillKillsDailyCountCmd)
	campaignCmd.AddCommand(campaignProcessCmd)
	dbCmd.AddCommand(dbVacuumCmd)
}
