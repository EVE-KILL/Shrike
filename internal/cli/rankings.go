package cli

import (
	"github.com/eve-kill/shrike/internal/rankings"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var rebuildRankingsCmd = &cobra.Command{
	Use:   "rankings",
	Short: "Replace the weekly, 90-day, and all-time EVE-KILL rankings",
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()
		rows, err := rankings.Refresh(cmd.Context(), pool)
		if err != nil {
			return err
		}
		ui.Success("Generated %s entity ranking rows.", fmtCount(rows))
		return nil
	},
}

func init() {
	rebuildCmd.AddCommand(rebuildRankingsCmd)
}
