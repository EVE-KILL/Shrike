package cli

import (
	"fmt"
	"strconv"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Background queue inventory and state",
}

var queueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List declared queues and their settings",
	Long: `Prints the Go registry: every queue with its concurrency, retry policy,
and flags.

This is the transcription of backend/src/queues/*/queue.ts and is meant to be
diffed against it by eye.`,
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
	queueCmd.AddCommand(queueListCmd)
}
