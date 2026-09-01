package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/eve-kill/shrike/internal/workers"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
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

var (
	flagQueueJobsQueue  string
	flagQueueJobsState  string
	flagQueueJobsLimit  int
	flagQueueClearState []string
	flagQueueClearLimit int
)

func openRiverClient(cmd *cobra.Command) (*queue.Client, func(), error) {
	pool, err := openPool(cmd)
	if err != nil {
		return nil, nil, err
	}
	client, err := queue.New(queue.Options{Pool: pool})
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	return client, pool.Close, nil
}

var queueJobsCmd = &cobra.Command{
	Use: "jobs", Short: "List recent River jobs",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if flagQueueJobsLimit < 1 || flagQueueJobsLimit > 10000 {
			return fmt.Errorf("limit must be between 1 and 10000")
		}
		if flagQueueJobsState != "" && !validRiverState(flagQueueJobsState) {
			return fmt.Errorf("invalid River job state %q", flagQueueJobsState)
		}
		client, close, err := openRiverClient(cmd)
		if err != nil {
			return err
		}
		defer close()
		params := river.NewJobListParams().First(flagQueueJobsLimit).
			OrderBy(river.JobListOrderByID, river.SortOrderDesc)
		if flagQueueJobsQueue != "" {
			params = params.Queues(flagQueueJobsQueue)
		}
		if flagQueueJobsState != "" {
			params = params.States(rivertype.JobState(flagQueueJobsState))
		}
		result, err := client.JobList(cmd.Context(), params)
		if err != nil {
			return err
		}
		if ui.JSONMode {
			return ui.JSON(result.Jobs)
		}
		ui.Section("River jobs")
		table := ui.NewTable("ID", "QUEUE", "KIND", "STATE", "ATTEMPT", "SCHEDULED")
		for _, job := range result.Jobs {
			table.Row(strconv.FormatInt(job.ID, 10), job.Queue, job.Kind, string(job.State),
				fmt.Sprintf("%d/%d", job.Attempt, job.MaxAttempts), job.ScheduledAt.UTC().Format("2006-01-02 15:04:05Z"))
		}
		fmt.Println(table.Render())
		ui.Newline()
		return nil
	},
}

func queueJobID(args []string) (int64, error) {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid River job ID %q", args[0])
	}
	return id, nil
}

var queueJobCmd = &cobra.Command{
	Use: "job ID", Short: "Show one River job", Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := queueJobID(args)
		if err != nil {
			return err
		}
		client, close, err := openRiverClient(cmd)
		if err != nil {
			return err
		}
		defer close()
		job, err := client.JobGet(cmd.Context(), id)
		if err != nil {
			return err
		}
		if ui.JSONMode {
			return ui.JSON(job)
		}
		ui.Section(fmt.Sprintf("River job %d", job.ID))
		ui.KV("Queue", job.Queue)
		ui.KV("Kind", job.Kind)
		ui.KV("State", string(job.State))
		ui.KV("Attempt", fmt.Sprintf("%d/%d", job.Attempt, job.MaxAttempts))
		ui.KV("Created", job.CreatedAt.String())
		ui.KV("Scheduled", job.ScheduledAt.String())
		var pretty bytes.Buffer
		if json.Indent(&pretty, job.EncodedArgs, "", "  ") == nil {
			ui.KV("Arguments", pretty.String())
		} else {
			ui.KV("Arguments", string(job.EncodedArgs))
		}
		if output := job.Output(); len(output) > 0 {
			ui.KV("Output", string(output))
		}
		for i, e := range job.Errors {
			ui.KV(fmt.Sprintf("Error %d", i+1), e.Error)
		}
		ui.Newline()
		return nil
	},
}

func riverJobMutation(use, short string, mutate func(context.Context, *queue.Client, int64) (*rivertype.JobRow, error)) *cobra.Command {
	return &cobra.Command{Use: use + " ID", Short: short, Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := queueJobID(args)
		if err != nil {
			return err
		}
		client, close, err := openRiverClient(cmd)
		if err != nil {
			return err
		}
		defer close()
		job, err := mutate(cmd.Context(), client, id)
		if err != nil {
			return err
		}
		if ui.JSONMode {
			return ui.JSON(job)
		}
		ui.Success("River job %d is now %s.", job.ID, job.State)
		ui.Newline()
		return nil
	}}
}

var queueCancelCmd = riverJobMutation("cancel", "Cancel a River job", func(ctx context.Context, c *queue.Client, id int64) (*rivertype.JobRow, error) {
	return c.JobCancel(ctx, id)
})
var queueRetryCmd = riverJobMutation("retry", "Retry or rerun a River job now", func(ctx context.Context, c *queue.Client, id int64) (*rivertype.JobRow, error) {
	return c.JobRetry(ctx, id)
})
var queueDeleteCmd = riverJobMutation("delete", "Permanently delete a River job", func(ctx context.Context, c *queue.Client, id int64) (*rivertype.JobRow, error) {
	return c.JobDelete(ctx, id)
})

func riverQueueControl(use, short string, control func(context.Context, *queue.Client, string, *river.QueuePauseOpts) error) *cobra.Command {
	return &cobra.Command{Use: use + " QUEUE", Short: short, Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, close, err := openRiverClient(cmd)
		if err != nil {
			return err
		}
		defer close()
		if err := control(cmd.Context(), client, args[0], nil); err != nil {
			return err
		}
		if ui.JSONMode {
			return ui.JSON(map[string]any{"queue": args[0], "action": use})
		}
		ui.Success("River queue %s %sd.", args[0], use)
		ui.Newline()
		return nil
	}}
}

var queuePauseCmd = riverQueueControl("pause", "Pause one River queue (or * for all)", func(ctx context.Context, client *queue.Client, name string, opts *river.QueuePauseOpts) error {
	return client.QueuePause(ctx, name, opts)
})
var queueResumeCmd = riverQueueControl("resume", "Resume one River queue (or * for all)", func(ctx context.Context, client *queue.Client, name string, opts *river.QueuePauseOpts) error {
	return client.QueueResume(ctx, name, opts)
})

var queueClearCmd = &cobra.Command{
	Use: "clear QUEUE", Short: "Permanently delete jobs in selected states", Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] == "*" {
			return fmt.Errorf("clear requires one explicit queue")
		}
		if flagQueueClearLimit < 1 || flagQueueClearLimit > 10000 {
			return fmt.Errorf("limit must be between 1 and 10000")
		}
		states := make([]rivertype.JobState, 0, len(flagQueueClearState))
		for _, state := range flagQueueClearState {
			if !validRiverState(state) || state == "running" {
				return fmt.Errorf("invalid or unsafe River job state %q", state)
			}
			states = append(states, rivertype.JobState(state))
		}
		if len(states) == 0 {
			return fmt.Errorf("at least one --state is required")
		}
		client, close, err := openRiverClient(cmd)
		if err != nil {
			return err
		}
		defer close()
		result, err := client.JobDeleteMany(cmd.Context(), river.NewJobDeleteManyParams().First(flagQueueClearLimit).Queues(args[0]).States(states...))
		if err != nil {
			return err
		}
		if ui.JSONMode {
			return ui.JSON(map[string]any{"queue": args[0], "deleted": len(result.Jobs)})
		}
		ui.Success("Deleted %d jobs from %s.", len(result.Jobs), args[0])
		ui.Newline()
		return nil
	},
}

func validRiverState(state string) bool {
	switch rivertype.JobState(state) {
	case rivertype.JobStateAvailable, rivertype.JobStateCancelled,
		rivertype.JobStateCompleted, rivertype.JobStateDiscarded,
		rivertype.JobStatePending, rivertype.JobStateRetryable,
		rivertype.JobStateRunning, rivertype.JobStateScheduled:
		return true
	default:
		return false
	}
}

func init() {
	queueJobsCmd.Flags().StringVar(&flagQueueJobsQueue, "queue", "", "filter by queue")
	queueJobsCmd.Flags().StringVar(&flagQueueJobsState, "state", "", "filter by state")
	queueJobsCmd.Flags().IntVar(&flagQueueJobsLimit, "limit", 50, "maximum jobs to show")
	queueClearCmd.Flags().StringSliceVar(&flagQueueClearState, "state", nil, "job state to delete (repeatable)")
	queueClearCmd.Flags().IntVar(&flagQueueClearLimit, "limit", 1000, "maximum jobs to delete")
	queueCmd.AddCommand(queueMigrateCmd, queueStatusCmd, queuePortedCmd, queueJobsCmd,
		queueJobCmd, queueCancelCmd, queueRetryCmd, queueDeleteCmd, queuePauseCmd,
		queueResumeCmd, queueClearCmd)
}
