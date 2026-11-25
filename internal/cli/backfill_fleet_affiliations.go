package cli

import (
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	var days int
	var apply bool
	command := &cobra.Command{
		Use:   "fleet-affiliations",
		Short: "Prepare recent killmails for character dashboard Flies With lists",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if days < 1 {
				return fmt.Errorf("days must be positive")
			}
			now := time.Now().UTC()
			firstDay := now.AddDate(0, 0, -days).Truncate(24 * time.Hour)
			ui.KV("Killmail window", firstDay.Format(time.DateOnly)+" through "+now.Format(time.RFC3339))
			if !apply {
				ui.Warn("Dry run. Use --apply to prepare this window.")
				return nil
			}
			pool, err := openPool(cmd)
			if err != nil {
				return err
			}
			defer pool.Close()

			var total int64
			for end := now; end.After(firstDay); {
				start := end.Truncate(24 * time.Hour)
				if end.Equal(start) {
					start = start.AddDate(0, 0, -1)
				}
				prepared, err := killmail.BackfillFleetAffiliations(cmd.Context(), pool, start, end)
				total += prepared
				if err != nil {
					return fmt.Errorf("backfill stopped after %d prepared killmails: %w", total, err)
				}
				if prepared > 0 {
					ui.KV(start.Format(time.DateOnly), fmtCount(prepared)+" prepared")
				}
				end = start
			}
			ui.Success("Prepared %s killmails.", fmtCount(total))
			return nil
		},
	}
	command.Flags().IntVar(&days, "days", 90, "Number of prior UTC days to prepare, plus today")
	command.Flags().BoolVar(&apply, "apply", false, "Write the projections")
	backfillCmd.AddCommand(command)
}
