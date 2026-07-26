package cli

import "github.com/spf13/cobra"

// Compatibility entry points keep the TypeScript CLI spellings usable while
// the Go CLI also exposes the split, more discoverable command tree.

var cronjobsCmd = &cobra.Command{
	Use:   "cronjobs [job]",
	Short: "Start recurring jobs, or run one immediately",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return cronRunCmd.RunE(cmd, args)
		}
		return workCronCmd.RunE(cmd, nil)
	},
}

var queuesCompatCmd = &cobra.Command{
	Use:   "queues [queue]",
	Short: "List queues, or run one queue worker",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return queueListCmd.RunE(cmd, nil)
		}
		previous := flagWorkQueues
		flagWorkQueues = []string{args[0]}
		defer func() { flagWorkQueues = previous }()
		return workQueuesCmd.RunE(cmd, nil)
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset operational state",
}

var resetEntityHistoryCompatCmd = &cobra.Command{
	Use:   "entity-history-queues",
	Short: resetEntityHistoryQueuesCmd.Short,
	Long:  resetEntityHistoryQueuesCmd.Long,
	RunE: func(cmd *cobra.Command, args []string) error {
		return resetEntityHistoryQueuesCmd.RunE(cmd, args)
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update published datasets",
}

var updateSDECompatCmd = &cobra.Command{
	Use:   "sde",
	Short: sdeImportCmd.Short,
	Long:  sdeImportCmd.Long,
	RunE: func(cmd *cobra.Command, args []string) error {
		return sdeImportCmd.RunE(cmd, args)
	},
}

func init() {
	resetEntityHistoryCompatCmd.Flags().BoolVar(
		&flagResetHistoryApply, "apply", false, "Empty the history queues",
	)
	resetEntityHistoryCompatCmd.Flags().StringVar(
		&flagResetHistoryConfirm, "confirm", "", "Required with --apply",
	)
	resetCmd.AddCommand(resetEntityHistoryCompatCmd)

	updateSDECompatCmd.Flags().BoolVarP(
		&flagSDEForce, "force", "f", false, "Re-import even when the build is current",
	)
	updateCmd.AddCommand(updateSDECompatCmd)

	rootCmd.AddCommand(cronjobsCmd, queuesCompatCmd, resetCmd, updateCmd)
}
