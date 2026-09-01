package cli

import (
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/rankings"
	"github.com/eve-kill/shrike/internal/stats"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var (
	flagStatsPointFromMonth string
	flagStatsPointToMonth   string
	flagStatsPointWorkers   int
	flagStatsPointApply     bool
)

var backfillStatsPointsCmd = &cobra.Command{
	Use:   "stats-points",
	Short: "Replace only combat points in stats and rebuild ranking rollups",
	RunE: func(cmd *cobra.Command, _ []string) error {
		from, err := parseMonth(flagStatsPointFromMonth, "--from")
		if err != nil {
			return err
		}
		toValue := flagStatsPointToMonth
		if toValue == "" {
			toValue = time.Now().UTC().Format("2006-01")
		}
		to, err := parseMonth(toValue, "--to")
		if err != nil {
			return err
		}
		ui.Section("Backfill stats points")
		ui.KV("Months", fmt.Sprintf("%s–%s", from.Format("2006-01"), to.Format("2006-01")))
		ui.KV("Workers", fmt.Sprint(flagStatsPointWorkers))
		if !flagStatsPointApply {
			ui.Warn("Dry run. Re-run with --apply to replace stats points.")
			return nil
		}
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()
		rows, pipeline, err := stats.RebuildPoints(cmd.Context(), pool, stats.RebuildPointOptions{
			FromMonth: from, ToMonth: to, Workers: flagStatsPointWorkers,
		})
		if err != nil {
			return err
		}
		rankingRows, err := rankings.Refresh(cmd.Context(), pool)
		if err != nil {
			return err
		}
		ui.Success(
			"Replaced %s source rows, generated %s leaderboard rows, and generated %s entity ranking rows.",
			fmtCount(rows), fmtCount(pipeline.Leaderboards), fmtCount(rankingRows),
		)
		return nil
	},
}

func init() {
	backfillStatsPointsCmd.Flags().StringVarP(&flagStatsPointFromMonth, "from", "f", "2007-12", "Start month (YYYY-MM)")
	backfillStatsPointsCmd.Flags().StringVarP(&flagStatsPointToMonth, "to", "t", "", "End month, inclusive")
	backfillStatsPointsCmd.Flags().IntVarP(&flagStatsPointWorkers, "workers", "w", 4, "Concurrent month partitions")
	backfillStatsPointsCmd.Flags().BoolVar(&flagStatsPointApply, "apply", false, "Replace stats point values")
	backfillCmd.AddCommand(backfillStatsPointsCmd)
}
