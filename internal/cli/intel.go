package cli

import (
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/intelrollup"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/riverqueue/river"
	"github.com/spf13/cobra"
)

var (
	flagIntelFrom  string
	flagIntelTo    string
	flagIntelApply bool
)

var backfillIntelCmd = &cobra.Command{
	Use:   "intel",
	Short: "Queue a checkpointed character-intelligence day backfill",
	RunE: func(cmd *cobra.Command, _ []string) error {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		from := today.AddDate(0, 0, -(intelrollup.RetentionDays - 1))
		to := today
		var err error
		if flagIntelFrom != "" {
			from, err = time.Parse("2006-01-02", flagIntelFrom)
			if err != nil {
				return fmt.Errorf("invalid --from: %w", err)
			}
		}
		if flagIntelTo != "" {
			to, err = time.Parse("2006-01-02", flagIntelTo)
			if err != nil {
				return fmt.Errorf("invalid --to: %w", err)
			}
		}
		if to.Before(from) || to.Sub(from) >= intelrollup.RetentionDays*24*time.Hour {
			return fmt.Errorf("intel backfill range must be ordered and at most %d days", intelrollup.RetentionDays)
		}
		days := int(to.Sub(from)/(24*time.Hour)) + 1
		ui.Section("Backfill character intel")
		ui.KV("Range", from.Format("2006-01-02")+" .. "+to.Format("2006-01-02"))
		ui.KV("Days", fmt.Sprint(days))
		if !flagIntelApply {
			ui.KV("Mode", "dry run — pass --apply to enqueue")
			return nil
		}
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()
		if _, err := pool.Exec(cmd.Context(), `
			INSERT INTO character_intel_dirty_days (activity_date,dirtied_at)
			SELECT d::date,now() FROM generate_series($1::date,$2::date,interval '1 day') d
			ON CONFLICT (activity_date) DO UPDATE SET dirtied_at=EXCLUDED.dirtied_at`, from, to); err != nil {
			return err
		}
		client, err := queue.New(queue.Options{Pool: pool})
		if err != nil {
			return err
		}
		jobs := make([]river.JobArgs, 0, days)
		for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
			jobs = append(jobs, queue.CharacterIntelRollupArgs{Day: day.Format("2006-01-02")})
		}
		n, err := queue.DispatchMany(cmd.Context(), client, jobs, queue.DormantBackfill)
		if err != nil {
			return err
		}
		ui.Success("Queued %d day jobs.", n)
		return nil
	},
}

func init() {
	backfillIntelCmd.Flags().StringVar(&flagIntelFrom, "from", "", "First UTC day (YYYY-MM-DD)")
	backfillIntelCmd.Flags().StringVar(&flagIntelTo, "to", "", "Last UTC day, inclusive (YYYY-MM-DD)")
	backfillIntelCmd.Flags().BoolVar(&flagIntelApply, "apply", false, "Mark and enqueue the backfill")
	backfillCmd.AddCommand(backfillIntelCmd)
}
