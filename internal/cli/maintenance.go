package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/battle"
	"github.com/eve-kill/shrike/internal/configstore"
	"github.com/eve-kill/shrike/internal/graph"
	"github.com/eve-kill/shrike/internal/maintenance"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/spf13/cobra"
)

// Operational maintenance.
//
// Everything here repairs or refreshes something the running system normally
// keeps current on its own. They exist because "normally" is not "always": a
// worker outage leaves stale markers, a schema addition leaves a column
// unpopulated, and a bad night leaves entities queued that never got fetched.

var (
	flagLastActiveFromMonth string
	flagLastActiveToMonth   string
)

var backfillLastActiveCmd = &cobra.Command{
	Use:   "last-active",
	Short: "Recompute when each character was last seen",
	Long: `Rebuilds characters.last_active from the killmails.

The column is denormalised so "who is still playing" is an index scan rather
than an aggregate over every killmail a character appears on. It drifts when
killmails are backfilled out of order, because the live path only ever moves it
forward.

	Recomputed as a MAX over both sides — a character is active whether they killed
or died — so this is safe to re-run.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		from, to, err := inclusiveMonthRange(flagLastActiveFromMonth, flagLastActiveToMonth)
		if err != nil {
			return err
		}

		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		ui.Section("Backfill last-active")
		start := time.Now()

		// One statement rather than a per-month loop produces the same maximum
		// while preserving the TS command's monotonic update.
		tag, err := pool.Exec(cmd.Context(), `
            UPDATE characters c
            SET last_active = s.last_active
            FROM (
                SELECT character_id, MAX(last_seen) AS last_active FROM (
                    SELECT character_id, MAX(killmail_time) AS last_seen
                    FROM killmail_attackers
                    WHERE character_id IS NOT NULL
                      AND killmail_time >= $1 AND killmail_time < $2
                    GROUP BY character_id
                    UNION ALL
                    SELECT victim_character_id, MAX(killmail_time)
                    FROM killmails
                    WHERE victim_character_id IS NOT NULL
                      AND killmail_time >= $1 AND killmail_time < $2
                    GROUP BY victim_character_id
                ) seen
                GROUP BY character_id
            ) s
            WHERE c.character_id = s.character_id
              AND (c.last_active IS NULL OR c.last_active < s.last_active)`,
			from, to)
		if err != nil {
			return err
		}

		ui.Newline()
		ui.Success("Updated %s characters in %s.",
			fmtCount(tag.RowsAffected()), time.Since(start).Round(time.Millisecond))
		ui.Newline()
		return nil
	},
}

var backfillMissingWarsCmd = &cobra.Command{
	Use:   "missing-wars",
	Short: "Fetch war metadata referenced by killmails but never imported",
	Long: `Finds killmails whose war_id has no matching war and queues the fetch.

Historical killmail archives carry war ids whose metadata was never imported, so
a war page can be referenced by thousands of kills and not exist. The hourly
wars cron repairs a hundred at a time; this clears the whole backlog at once.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		rows, err := pool.Query(cmd.Context(), `
            SELECT DISTINCT k.war_id
            FROM killmails k
            LEFT JOIN wars w ON w.war_id = k.war_id
            WHERE k.war_id IS NOT NULL AND w.war_id IS NULL
            ORDER BY k.war_id`)
		if err != nil {
			return err
		}

		var args []river.JobArgs
		for rows.Next() {
			var id int32
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			// Metadata only: these wars' killmails are already stored, so
			// walking ESI's list for them would spend the request budget to
			// discover nothing.
			args = append(args, queue.WarArgs{WarID: id, MetadataOnly: true})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}

		ui.Section("Backfill missing wars")
		ui.KV("Missing", fmtCount(int64(len(args))))
		if len(args) == 0 || !flagBackfillApply {
			ui.Newline()
			if len(args) > 0 {
				fmt.Printf("  %s\n", ui.Dim("dry run — pass --apply to enqueue"))
			}
			ui.Newline()
			return nil
		}
		if flagMissingWarsConfirm != missingWarsConfirmation {
			return fmt.Errorf("refusing dispatch: pass --confirm=%s", missingWarsConfirmation)
		}

		client, err := queue.New(queue.Options{Pool: pool})
		if err != nil {
			return err
		}
		n, err := queue.DispatchMany(cmd.Context(), client, args, queue.DormantBackfill)
		if err != nil {
			return err
		}

		ui.Newline()
		ui.Success("Enqueued %s wars.", fmtCount(int64(n)))
		ui.Newline()
		return nil
	},
}

const missingWarsConfirmation = "BACKFILL-MISSING-WARS"

var flagMissingWarsConfirm string

const battleBackfillCursorKey = "backfill:battles:last_date"

var (
	flagBattleBackfillFrom     string
	flagBattleBackfillTo       string
	flagBattleBackfillMinKills int
	flagBattleBackfillDryRun   bool
)

var backfillBattlesCmd = &cobra.Command{
	Use:   "battles",
	Short: "Detect battles over a historical range",
	Long: `Detects and stores battles one day at a time.

Without --from it resumes on the day after the saved cursor, or starts at the
earliest killmail. The end defaults to yesterday and is inclusive.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if flagBattleBackfillMinKills < 1 {
			return fmt.Errorf("--min-kills must be positive")
		}

		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		from, err := battleBackfillStart(cmd.Context(), pool, flagBattleBackfillFrom)
		if err != nil {
			return err
		}
		to := time.Now().UTC().AddDate(0, 0, -1)
		to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
		if flagBattleBackfillTo != "" {
			to, err = time.Parse("2006-01-02", flagBattleBackfillTo)
			if err != nil {
				return fmt.Errorf("invalid --to %q, want YYYY-MM-DD", flagBattleBackfillTo)
			}
		}

		ui.Section("Backfill battles")
		if from.After(to) {
			ui.Success("Already up to date.")
			return nil
		}
		ui.KV("Range", from.Format("2006-01-02")+" .. "+to.Format("2006-01-02"))
		ui.KV("Minimum kills/hour", fmt.Sprint(flagBattleBackfillMinKills))
		if flagBattleBackfillDryRun {
			ui.Warn("Dry run — battles and progress are not stored")
		}

		var detected, saved, skipped int
		for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
			next := day.AddDate(0, 0, 1)
			windows, err := battle.FindHotspotWindows(
				cmd.Context(), pool, day, next, flagBattleBackfillMinKills,
			)
			if err != nil {
				return err
			}

			for _, window := range windows {
				kms, attackers, err := battle.LoadSystem(
					cmd.Context(), pool, window.SolarSystemID,
					window.Start.Add(-30*time.Minute), window.End.Add(30*time.Minute),
				)
				if err != nil {
					return err
				}
				found := battle.Detect(kms, attackers, nil)
				if found == nil {
					continue
				}
				detected++
				if flagBattleBackfillDryRun {
					continue
				}
				overlap, err := battle.HasOverlap(
					cmd.Context(), pool, found.SolarSystemID, found.Start, found.End,
				)
				if err != nil {
					return err
				}
				if overlap {
					skipped++
					continue
				}
				if _, err := battle.Store(cmd.Context(), pool, found); err != nil {
					return err
				}
				saved++
			}

			if !flagBattleBackfillDryRun {
				if err := configstore.Set(
					cmd.Context(), pool, battleBackfillCursorKey, day.Format("2006-01-02"),
				); err != nil {
					return err
				}
			}
		}
		ui.Newline()
		ui.KV("Detected", fmt.Sprint(detected))
		if !flagBattleBackfillDryRun {
			ui.KV("Saved", fmt.Sprint(saved))
			ui.KV("Skipped overlaps", fmt.Sprint(skipped))
		}
		ui.Newline()
		return nil
	},
}

func battleBackfillStart(
	ctx context.Context,
	pool *pgxpool.Pool,
	explicit string,
) (time.Time, error) {
	if explicit != "" {
		from, err := time.Parse("2006-01-02", explicit)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid --from %q, want YYYY-MM-DD", explicit)
		}
		return from, nil
	}

	saved, err := configstore.Get(ctx, pool, battleBackfillCursorKey)
	if err != nil {
		return time.Time{}, err
	}
	if saved != "" {
		from, err := time.Parse("2006-01-02", saved)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid saved battle cursor %q: %w", saved, err)
		}
		return from.AddDate(0, 0, 1), nil
	}

	var earliest *time.Time
	if err := pool.QueryRow(ctx, `SELECT min(killmail_time)::date FROM killmails`).Scan(&earliest); err != nil {
		return time.Time{}, err
	}
	if earliest == nil {
		return time.Time{}, fmt.Errorf("no killmails in database")
	}
	return earliest.UTC(), nil
}

var (
	flagStaleLimit            int
	flagStaleBatch            int
	flagStaleCorporationDays  int
	flagStaleAllianceDays     int
	flagStaleSkipAlliances    bool
	flagStaleSkipCorporations bool
	flagStaleSkipCharacters   bool
	flagStaleDryRun           bool
)

var queueStaleEntitiesCmd = &cobra.Command{
	Use:   "stale-entities",
	Short: "Queue entities that have gone stale for a refresh",
	Long: `Queues stale alliances and player corporations, plus characters whose
birthday is missing or predates EVE.

Character dispatch requires --limit unless --dry-run or --skip-characters is
used. Alliances and corporations may be processed without a cap.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		var client *queue.Client
		if !flagStaleDryRun {
			client, err = queue.New(queue.Options{Pool: pool})
			if err != nil {
				return err
			}
		}

		ui.Section("Queue stale entities")
		if flagStaleDryRun {
			ui.Warn("Dry run — no jobs will be enqueued")
		}

		res, err := maintenance.QueueStaleEntities(
			cmd.Context(), pool, client, maintenance.StaleEntityOptions{
				SkipAlliances:    flagStaleSkipAlliances,
				SkipCorporations: flagStaleSkipCorporations,
				SkipCharacters:   flagStaleSkipCharacters,
				AllianceDays:     flagStaleAllianceDays,
				CorporationDays:  flagStaleCorporationDays,
				Batch:            flagStaleBatch,
				Limit:            flagStaleLimit,
				DryRun:           flagStaleDryRun,
			},
		)
		if err != nil {
			return err
		}
		if ui.JSONMode {
			return ui.JSON(res)
		}
		ui.KV("Alliance candidates", fmtCount(res.AllianceCandidates))
		ui.KV("Alliances queued", fmtCount(res.AllianceQueued))
		ui.KV("Corporation candidates", fmtCount(res.CorporationCandidates))
		ui.KV("Corporations queued", fmtCount(res.CorporationQueued))
		ui.KV("Character candidates", fmtCount(res.CharacterCandidates))
		ui.KV("Characters queued", fmtCount(res.CharacterQueued))
		ui.Newline()
		return nil
	},
}

var resetEntityHistoryQueuesCmd = &cobra.Command{
	Use:   "reset-history-queues",
	Short: "Safely empty and resume the entity history queues",
	Long: `Reports both River history queues by default.

Apply mode pauses both queues, waits for running jobs to finish, removes every
retained history job, clears queued_at dispatch markers, and resumes the queues.
Stored history rows and fetched_at markers are untouched.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		ui.Section("Reset entity history queues")
		kinds := []string{
			(queue.CharacterHistoryArgs{}).Kind(),
			(queue.CorporationHistoryArgs{}).Kind(),
		}
		rows, err := pool.Query(cmd.Context(), `
			SELECT kind, state, count(*)::bigint
			FROM river_job
			WHERE kind = ANY($1::text[])
			GROUP BY kind, state
			ORDER BY kind, state`, kinds)
		if err != nil {
			return err
		}
		for rows.Next() {
			var kind, state string
			var count int64
			if err := rows.Scan(&kind, &state, &count); err != nil {
				rows.Close()
				return err
			}
			ui.KV(kind+" "+state, fmtCount(count))
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}

		if !flagResetHistoryApply {
			ui.Newline()
			ui.Warn("Read-only preview. Pass --apply --confirm=%s to reset.", resetHistoryConfirmation)
			return nil
		}
		if flagResetHistoryConfirm != resetHistoryConfirmation {
			return fmt.Errorf("refusing reset: pass --confirm=%s", resetHistoryConfirmation)
		}

		client, err := queue.New(queue.Options{Pool: pool})
		if err != nil {
			return err
		}
		paused := make([]string, 0, len(kinds))
		defer func() {
			for _, kind := range paused {
				_ = client.QueueResume(context.Background(), kind, nil)
			}
		}()
		for _, kind := range kinds {
			if err := client.QueuePause(cmd.Context(), kind, nil); err != nil {
				return err
			}
			paused = append(paused, kind)
		}
		if err := waitForRiverJobs(cmd.Context(), pool, kinds, 2*time.Minute); err != nil {
			return err
		}

		tag, err := pool.Exec(cmd.Context(), `DELETE FROM river_job WHERE kind = ANY($1::text[])`, kinds)
		if err != nil {
			return err
		}
		deleted := tag.RowsAffected()

		var markers int64
		for _, target := range []struct {
			table  string
			column string
		}{
			{"characters", "corporation_history_queued_at"},
			{"corporations", "alliance_history_queued_at"},
		} {
			tag, err := pool.Exec(cmd.Context(), fmt.Sprintf(
				`UPDATE %s SET %s = NULL WHERE %s IS NOT NULL`,
				target.table, target.column, target.column))
			if err != nil {
				return fmt.Errorf("reset %s: %w", target.table, err)
			}
			markers += tag.RowsAffected()
		}

		for _, kind := range paused {
			if err := client.QueueResume(cmd.Context(), kind, nil); err != nil {
				return err
			}
		}
		paused = nil
		ui.Newline()
		ui.Success("Removed %s jobs and cleared %s dispatch markers.",
			fmtCount(deleted), fmtCount(markers))
		ui.Newline()
		return nil
	},
}

const resetHistoryConfirmation = "RESET-ENTITY-HISTORY-QUEUES"

var (
	flagResetHistoryApply   bool
	flagResetHistoryConfirm string
)

func waitForRiverJobs(
	ctx context.Context,
	pool *pgxpool.Pool,
	kinds []string,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		var active int64
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM river_job
			WHERE kind = ANY($1::text[]) AND state = 'running'`, kinds).Scan(&active); err != nil {
			return err
		}
		if active == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %d active history jobs", active)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

var graphPurgeNowCmd = &cobra.Command{
	Use:   "purge",
	Short: "Prune aged-out relationships now",
	Long: fmt.Sprintf(`Removes graph edges older than %d days and any character left with none.

Batched with a bounded number of transactions per edge type, so a large backlog
may need several runs. Repeat it until it reports nothing; the daily cron uses
the same bounded cleanup.`, 90),
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		d, err := deps(cmd.Context(), pool, false)
		if err != nil {
			return err
		}
		if d.Graph == nil {
			return fmt.Errorf("memgraph is unreachable, so there is nothing to prune")
		}

		res, err := d.Graph.Purge(cmd.Context(), 10_000)
		if err != nil {
			return err
		}

		if ui.JSONMode {
			return ui.JSON(res)
		}
		ui.Section("Graph purge")
		for edgeType, n := range res.ByType {
			ui.KV(edgeType, fmtCount(n))
		}
		ui.KV("Orphaned nodes", fmtCount(res.Orphans))
		ui.KV("Killmail markers", fmtCount(res.Killmails))
		ui.Newline()
		return nil
	},
}

var graphSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Install Memgraph indexes and uniqueness constraints",
	Long: `Installs the schema required by graph ingestion and reads.

The operation is idempotent, but uniqueness constraints fail when an existing
graph contains duplicate node ids. In that case, stop graph writers and run a
clean graph rebuild before retrying this command.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := graph.Connect(cmd.Context(), cfg.MemgraphURL)
		if err != nil {
			return err
		}
		defer client.Close(cmd.Context()) //nolint:errcheck
		if err := client.EnsureSchema(cmd.Context()); err != nil {
			return err
		}
		ui.Success("Memgraph schema is up to date.")
		return nil
	},
}

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Relationship graph maintenance",
}

// battleDetectNowCmd runs detection in the foreground for one window.
var battleDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect battles in a window, in the foreground",
	Long: `Runs detection now and prints what it found, rather than enqueuing.

Useful for checking whether a specific fight is being detected the way you
expect — the queue path gives no visibility into why a battle was or was not
found.`,
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

		candidates, err := battle.FindCandidates(cmd.Context(), pool, from, to)
		if err != nil {
			return err
		}

		ui.Section("Battle detection")
		ui.KV("Range", from.Format("2006-01-02")+" .. "+to.Format("2006-01-02"))
		ui.KV("Candidate systems", fmtCount(int64(len(candidates))))

		table := ui.NewTable("SYSTEM", "KILLS", "ISK", "TEAMS", "MULTI")
		found := 0
		for _, c := range candidates {
			kms, atts, err := battle.LoadSystem(cmd.Context(), pool, c.SolarSystemID, from, to)
			if err != nil {
				return err
			}
			b := battle.Detect(kms, atts, nil)
			if b == nil {
				continue
			}
			found++
			multi := ""
			if b.MultiParty {
				multi = ui.Warn2("yes")
			}
			table.Row(
				fmt.Sprint(b.SolarSystemID),
				fmt.Sprint(b.KillCount),
				fmtISK(b.IskDestroyed),
				fmt.Sprintf("%d v %d", len(b.Teams[0].Entries), len(b.Teams[1].Entries)),
				multi,
			)
		}

		if found == 0 {
			ui.Newline()
			fmt.Printf("  %s\n", ui.Dim("no battles detected in this range"))
			ui.Newline()
			return nil
		}
		fmt.Println(table.Render())
		ui.Newline()
		ui.KV("Battles", fmt.Sprint(found))
		ui.Newline()
		return nil
	},
}

var battleCmd = &cobra.Command{
	Use:   "battle",
	Short: "Battle detection",
}

func init() {
	backfillLastActiveCmd.Flags().StringVarP(&flagLastActiveFromMonth, "from", "f", "2007-12", "Start month (YYYY-MM)")
	backfillLastActiveCmd.Flags().StringVarP(&flagLastActiveToMonth, "to", "t", "", "End month, inclusive (YYYY-MM; default current)")

	backfillMissingWarsCmd.Flags().BoolVar(&flagBackfillApply, "apply", false, "Dispatch missing-war repair jobs")
	backfillMissingWarsCmd.Flags().StringVar(&flagMissingWarsConfirm, "confirm", "", "Required with --apply")
	backfillBattlesCmd.Flags().StringVarP(&flagBattleBackfillFrom, "from", "f", "", "Start date (YYYY-MM-DD)")
	backfillBattlesCmd.Flags().StringVarP(&flagBattleBackfillTo, "to", "t", "", "End date, inclusive (YYYY-MM-DD; default yesterday)")
	backfillBattlesCmd.Flags().IntVar(&flagBattleBackfillMinKills, "min-kills", 10, "Minimum kills per hotspot hour")
	backfillBattlesCmd.Flags().BoolVar(&flagBattleBackfillDryRun, "dry-run", false, "Detect without saving")

	battleDetectCmd.Flags().StringVar(&flagBackfillFrom, "from", "", "Start date (YYYY-MM-DD)")
	battleDetectCmd.Flags().StringVar(&flagBackfillTo, "to", "", "End date, exclusive (YYYY-MM-DD)")

	queueStaleEntitiesCmd.Flags().BoolVar(&flagStaleSkipAlliances, "skip-alliances", false, "Skip alliance refresh")
	queueStaleEntitiesCmd.Flags().BoolVar(&flagStaleSkipCorporations, "skip-corporations", false, "Skip corporation refresh")
	queueStaleEntitiesCmd.Flags().BoolVar(&flagStaleSkipCharacters, "skip-characters", false, "Skip character refresh")
	queueStaleEntitiesCmd.Flags().IntVar(&flagStaleCorporationDays, "corp-stale-days", 30, "Corporation staleness threshold")
	queueStaleEntitiesCmd.Flags().IntVar(&flagStaleAllianceDays, "alli-stale-days", 30, "Alliance staleness threshold")
	queueStaleEntitiesCmd.Flags().IntVarP(&flagStaleBatch, "batch", "b", 500, "Bulk enqueue chunk size")
	queueStaleEntitiesCmd.Flags().IntVarP(&flagStaleLimit, "limit", "l", 0, "Maximum candidates per entity type")
	queueStaleEntitiesCmd.Flags().BoolVar(&flagStaleDryRun, "dry-run", false, "Count candidates without enqueueing")
	resetEntityHistoryQueuesCmd.Flags().BoolVar(&flagResetHistoryApply, "apply", false, "Empty the history queues")
	resetEntityHistoryQueuesCmd.Flags().StringVar(&flagResetHistoryConfirm, "confirm", "", "Required with --apply")

	backfillCmd.AddCommand(backfillLastActiveCmd, backfillMissingWarsCmd, backfillBattlesCmd)
	queueCmd.AddCommand(queueStaleEntitiesCmd, resetEntityHistoryQueuesCmd)
	graphCmd.AddCommand(graphPurgeNowCmd, graphSchemaCmd)
	battleCmd.AddCommand(battleDetectCmd)
}
