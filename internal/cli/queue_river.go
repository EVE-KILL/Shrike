package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/eve-kill/shrike/internal/workers"
	"github.com/spf13/cobra"
)

// The River-backed half of the queue commands.

var queueMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply River's schema migrations",
	Long: `Creates or upgrades the tables River needs.

This is separate from db:migrate on purpose. River owns its own schema, versioned
against the library rather than against our data model, and it keeps its own
ledger in river_migration. Copying its DDL into a goose migration would freeze it
at whatever the library shipped that week and make every upgrade a hand
transcription of someone else's schema change.

Like db:migrate, this is never run automatically at startup. A process that
migrates its own database on boot will eventually migrate one you did not mean.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		before, err := queue.MigrationState(cmd.Context(), pool)
		if err != nil {
			return err
		}
		if before.UpToDate() {
			ui.Newline()
			ui.Success("River schema is up to date (%d migrations applied).", len(before.Applied))
			ui.Newline()
			return nil
		}

		ui.Section("River migrations")
		ui.KV("Pending", fmt.Sprintf("%d", len(before.Pending)))

		applied, err := queue.Migrate(cmd.Context(), pool)
		if err != nil {
			return err
		}

		for _, name := range applied {
			fmt.Printf("  %s %s\n", ui.Primary("+"), name)
		}
		ui.Newline()
		ui.Success("Applied %d migration(s).", len(applied))
		ui.Newline()
		return nil
	},
}

var queueStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show River queue depths",
	Long: `Reads live job counts from Postgres, grouped by queue and state.

River is the only queue backend. Valkey is not consulted.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		state, err := queue.MigrationState(cmd.Context(), pool)
		if err != nil {
			return err
		}
		if !state.UpToDate() {
			ui.Newline()
			ui.Error("River schema is not migrated — run queue:migrate first.")
			ui.Newline()
			return fmt.Errorf("%d River migration(s) pending", len(state.Pending))
		}

		depths, err := queue.Depths(cmd.Context(), pool)
		if err != nil {
			return err
		}

		if ui.JSONMode {
			return ui.JSON(depths)
		}

		ui.Section("River queues")
		table := ui.NewTable("QUEUE", "AVAILABLE", "RUNNING", "SCHEDULED", "RETRYABLE", "DISCARDED", "COMPLETED")
		var any bool
		for _, d := range depths {
			if d.Total() > 0 {
				any = true
			}
			table.Row(
				ui.Command(d.Queue),
				count(d.Available), count(d.Running), count(d.Scheduled),
				count(d.Retryable), count(d.Discarded), count(d.Completed),
			)
		}
		fmt.Println(table.Render())

		if !any {
			ui.Newline()
			fmt.Printf("  %s\n", ui.Dim("no jobs — nothing has been enqueued yet"))
		}
		ui.Newline()
		return nil
	},
}

var queuePortedCmd = &cobra.Command{
	Use:   "ported",
	Short: "Show which queues have a Go worker",
	Long: `Compares the declared queues against the ones actually implemented.

A worker process consuming six of twenty queues looks, from the outside, exactly
like one consuming all twenty and finding nothing to do. This is the difference,
written down.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		implemented := workers.ImplementedQueues()
		missing := workers.UnimplementedQueues()

		if ui.JSONMode {
			return ui.JSON(map[string]any{
				"implemented":   implemented,
				"unimplemented": missing,
			})
		}

		ui.Section("Queue implementation status")
		have := map[string]bool{}
		for _, name := range implemented {
			have[name] = true
		}

		table := ui.NewTable("QUEUE", "STATUS", "CONC", "DESCRIPTION")
		for _, q := range jobs.Queues {
			status := ui.StatusBadge("unported")
			switch {
			case have[q.Name]:
				status = ui.StatusBadge("ok")
			case q.ConsumerElsewhere:
				status = ui.Accent("external")
			}
			table.Row(ui.Command(q.Name), status, strconv.Itoa(q.Concurrency), q.Description)
		}
		fmt.Println(table.Render())
		ui.Newline()
		ui.KV("Implemented", fmt.Sprintf("%d/%d", len(implemented), len(jobs.Queues)))
		if len(missing) > 0 {
			ui.KV("Not consumed", strings.Join(missing, ", "))
		}
		ui.Newline()
		return nil
	},
}

func init() {
	queueCmd.AddCommand(queueMigrateCmd, queueStatusCmd, queuePortedCmd)
}
