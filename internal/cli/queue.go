package cli

import (
	"fmt"
	"slices"
	"sort"
	"strconv"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Background queue inventory and state",
}

// queueRedis opens the queue Valkey instance. Cache lives on a different one,
// so this is deliberately not the cache client.
func queueRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
}

var queueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List declared queues and their settings",
	Long: `Prints the Go registry: every queue with its concurrency, retry policy,
and flags.

This is the transcription of backend/src/queues/*/queue.ts and is meant to be
diffed against it by eye. queue:verify checks it against live Redis instead.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if ui.JSONMode {
			return ui.JSON(jobs.Queues)
		}

		ui.Section("Queues")
		table := ui.NewTable("QUEUE", "CONC", "RETRIES", "BACKOFF", "FLAGS", "DESCRIPTION")
		for _, q := range jobs.Queues {
			var flags []string
			if q.RequiresTQ {
				flags = append(flags, ui.Warn2("tq"))
			}
			if q.ConsumerElsewhere {
				flags = append(flags, ui.Accent("external"))
			}
			table.Row(
				ui.Command(q.Name),
				strconv.Itoa(q.Concurrency),
				strconv.Itoa(q.Retries),
				fmt.Sprintf("%dms", q.BackoffDelay),
				joinFlags(flags),
				q.Description,
			)
		}
		fmt.Println(table.Render())
		ui.Newline()
		fmt.Printf("  %s\n", ui.Dim("tq = pauses while Tranquility is offline"))
		fmt.Printf("  %s\n", ui.Dim("external = enqueue only; another pod consumes it"))
		ui.Newline()
		return nil
	},
}

var queueStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show live queue depths",
	Long: `Reads live job counts from Redis for every declared queue.

Counts come from BullMQ's keys, since the Bun workers still own these queues.
Pending is waiting + prioritized + delayed; completed and failed are history.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		rdb := queueRedis()
		defer rdb.Close()

		depths, err := jobs.ReadDepths(cmd.Context(), rdb, jobs.QueueNames())
		if err != nil {
			return fmt.Errorf("read queue depths: %w", err)
		}

		if ui.JSONMode {
			return ui.JSON(map[string]any{"source": "bullmq", "queues": depths})
		}

		var totalPending, totalFailed int64
		ui.Section("Queue depths")
		table := ui.NewTable("QUEUE", "WAIT", "PRIO", "ACTIVE", "DELAYED", "FAILED", "STATE")
		for _, d := range depths {
			totalPending += d.Pending()
			totalFailed += d.Failed

			state := ui.Dim("idle")
			switch {
			case !d.Known:
				// No metadata key: nothing has ever constructed this queue.
				state = ui.Dim("unseen")
			case d.Paused:
				state = ui.StatusBadge("warn")
			case d.Active > 0:
				state = ui.StatusBadge("ok")
			case d.Pending() > 0:
				state = ui.Warn2("backlog")
			}

			table.Row(
				ui.Command(d.Name),
				count(d.Waiting), count(d.Prioritized), count(d.Active),
				count(d.Delayed), count(d.Failed), state,
			)
		}
		fmt.Println(table.Render())
		ui.Newline()
		ui.KV("Total pending", strconv.FormatInt(totalPending, 10))
		ui.KV("Total failed", strconv.FormatInt(totalFailed, 10))
		ui.KV("Source", ui.Dim("bullmq keys (bull:<queue>:*)"))
		ui.Newline()
		return nil
	},
}

var queueVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Diff the Go registry against what is live in Redis",
	Long: `Cross-checks the registry against reality, reporting three kinds of drift:

  missing   declared here but never seen in Redis
  unknown   live in Redis but absent from the registry
  ok        present in both

An "unknown" queue is the one that matters — it means the Bun side has a queue
this port would silently never process.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		rdb := queueRedis()
		defer rdb.Close()

		live, err := jobs.DiscoverQueues(cmd.Context(), rdb)
		if err != nil {
			return fmt.Errorf("scan redis: %w", err)
		}
		sort.Strings(live)

		declared := jobs.QueueNames()
		var missing, unknown, matched []string

		for _, d := range declared {
			if slices.Contains(live, d) {
				matched = append(matched, d)
			} else {
				missing = append(missing, d)
			}
		}
		for _, l := range live {
			if !slices.Contains(declared, l) {
				unknown = append(unknown, l)
			}
		}

		if ui.JSONMode {
			return ui.JSON(map[string]any{
				"declared": len(declared),
				"live":     len(live),
				"matched":  matched,
				"missing":  missing,
				"unknown":  unknown,
				"clean":    len(unknown) == 0,
			})
		}

		ui.Section("Registry vs Redis")
		ui.KV("Declared", strconv.Itoa(len(declared)))
		ui.KV("Live in Redis", strconv.Itoa(len(live)))
		ui.KV("Matched", strconv.Itoa(len(matched)))

		if len(missing) > 0 {
			ui.Section("Declared but never seen in Redis")
			for _, m := range missing {
				fmt.Printf("  %s %s\n", ui.Dim("·"), m)
			}
			ui.Newline()
			fmt.Printf("  %s\n", ui.Dim("Expected for queues with no traffic yet — not necessarily wrong."))
		}

		if len(unknown) > 0 {
			ui.Section("Live in Redis but NOT declared")
			for _, u := range unknown {
				fmt.Printf("  %s %s\n", ui.Accent("!"), ui.Bold(u))
			}
			ui.Newline()
			ui.Error("%d queue(s) exist in Redis with no Go declaration.", len(unknown))
			ui.Newline()
			return fmt.Errorf("registry is missing %d live queue(s)", len(unknown))
		}

		ui.Newline()
		ui.Success("No unknown queues — the registry covers everything live in Redis.")
		ui.Newline()
		return nil
	},
}

// count renders zero dimmed so non-zero values stand out in a mostly-idle table.
func count(n int64) string {
	if n == 0 {
		return ui.Dim("0")
	}
	return ui.Bold(strconv.FormatInt(n, 10))
}

func joinFlags(flags []string) string {
	if len(flags) == 0 {
		return ui.Dim("—")
	}
	out := flags[0]
	for _, f := range flags[1:] {
		out += " " + f
	}
	return out
}

func init() {
	queueCmd.AddCommand(queueListCmd, queueStatusCmd, queueVerifyCmd)
}
