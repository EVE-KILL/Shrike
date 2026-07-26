package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/cron"
	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/eve-kill/shrike/internal/workers"
	"github.com/spf13/cobra"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Scheduled job inventory",
}

var flagCronSortByFreq bool

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled jobs and their intervals",
	Long: `Prints the Go registry: every cron with its interval and flags.

Schedules are fixed intervals, not cron expressions — parseSchedule() in the Bun
implementation understands only s/m/h/d. Jobs that want a specific wall-clock
time check the clock inside their own run().

--by-frequency sorts fastest-first, which is the useful order when judging load.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		list := make([]jobs.Cron, len(jobs.Crons))
		copy(list, jobs.Crons)

		// Sorting by parsed duration rather than by the string, so "30s" orders
		// before "1m" instead of lexically after it.
		if flagCronSortByFreq {
			sort.SliceStable(list, func(i, j int) bool {
				di, erri := jobs.ParseSchedule(list[i].Schedule)
				dj, errj := jobs.ParseSchedule(list[j].Schedule)
				if erri != nil || errj != nil {
					return list[i].Name < list[j].Name
				}
				return di < dj
			})
		}

		if ui.JSONMode {
			return ui.JSON(list)
		}

		ui.Section("Scheduled jobs")
		table := ui.NewTable("CRON", "EVERY", "PER DAY", "FLAGS", "DESCRIPTION")
		var perDayTotal float64
		for _, c := range list {
			every, err := jobs.ParseSchedule(c.Schedule)
			if err != nil {
				// A schedule the parser rejects would never run; surface it here
				// rather than letting it fail silently at boot.
				table.Row(ui.Command(c.Name), ui.StatusBadge("fail"), "?", "", c.Description)
				continue
			}
			perDay := (24 * time.Hour).Seconds() / every.Seconds()
			perDayTotal += perDay

			var flags []string
			if c.RunOnStart {
				flags = append(flags, ui.Primary("boot"))
			}
			if c.RequiresTQ {
				flags = append(flags, ui.Warn2("tq"))
			}
			if c.LegacyName != "" {
				flags = append(flags, ui.Accent("renamed"))
			}

			table.Row(
				ui.Command(c.Name),
				c.Schedule,
				formatPerDay(perDay),
				joinFlags(flags),
				c.Description,
			)
		}
		fmt.Println(table.Render())
		ui.Newline()
		ui.KV("Jobs", strconv.Itoa(len(list)))
		ui.KV("Runs/day", formatPerDay(perDayTotal))
		ui.Newline()
		fmt.Printf("  %s\n", ui.Dim("boot = also runs immediately at startup"))
		fmt.Printf("  %s\n", ui.Dim("tq = skipped while Tranquility is offline"))

		// Call out the rename explicitly: these strings become River job kinds,
		// so the inconsistency is worth fixing before anything depends on it.
		for _, c := range list {
			if c.LegacyName != "" {
				fmt.Printf("  %s\n", ui.Dim(fmt.Sprintf(
					"renamed = %q in the Bun implementation, normalised to %q here",
					c.LegacyName, c.Name)))
			}
		}
		ui.Newline()
		return nil
	},
}

// formatPerDay keeps the column readable across five orders of magnitude —
// status_update runs 86,400 times a day, entity_snapshot once.
func formatPerDay(n float64) string {
	switch {
	case n >= 1000:
		return fmt.Sprintf("%.0fk", n/1000)
	case n >= 10:
		return fmt.Sprintf("%.0f", n)
	default:
		return fmt.Sprintf("%.1f", n)
	}
}

var cronRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run one scheduled job immediately",
	Long: `Runs a cron once, in the foreground, and prints what it did.

This deliberately bypasses the queue. An operator running a job by hand wants it
to run now and to see the result on their terminal — not to insert a row and
hope a worker somewhere picks it up.

Jobs run this way are not gated on Tranquility being online. If you are asking
for it explicitly, you get it.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		// A queue client, because several crons exist only to enqueue work and
		// would otherwise fail with "needs a queue to dispatch into".
		d, err := deps(cmd.Context(), pool, true)
		if err != nil {
			return err
		}

		registry, err := workers.RegisterCrons(d)
		if err != nil {
			return err
		}

		run, err := cron.RunOnce(cmd.Context(), registry, name)
		if err != nil {
			// A cron that is declared but unported is a fact about the port, not
			// a failure of this invocation — say so plainly and list what does
			// work rather than printing a bare error.
			if len(registry.Implemented()) > 0 {
				ui.Newline()
				ui.KV("Implemented", strings.Join(registry.Implemented(), ", "))
				ui.Newline()
			}
			return err
		}

		if ui.JSONMode {
			return ui.JSON(map[string]any{
				"cron":       run.Name,
				"report":     run.Report,
				"elapsed_ms": run.Elapsed.Milliseconds(),
			})
		}

		ui.Newline()
		ui.KV("Cron", run.Name)
		if run.Report != "" {
			ui.KV("Result", run.Report)
		}
		ui.KV("Elapsed", run.Elapsed.Round(time.Millisecond).String())
		ui.Newline()
		return nil
	},
}

var cronStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show which crons have a Go implementation",
	Long: `Compares the declared crons against the ones actually implemented.

During the port this gap is most of them, and it is worth looking at directly: a
scheduler running eight of thirty-two jobs is indistinguishable from one running
all thirty-two and finding nothing to do.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// No pool: this reports on wiring, not on data, and must work against a
		// machine with no database reachable.
		registry, err := workers.RegisterCrons(&workers.Deps{})
		if err != nil {
			return err
		}

		implemented := registry.Implemented()
		missing := registry.Unimplemented()

		if ui.JSONMode {
			return ui.JSON(map[string]any{
				"implemented":   implemented,
				"unimplemented": missing,
				"declared":      len(jobs.Crons),
			})
		}

		ui.Section("Cron implementation status")
		table := ui.NewTable("CRON", "STATUS", "EVERY", "DESCRIPTION")
		for _, c := range jobs.Crons {
			// "unported" rather than "fail": these jobs are not broken, they
			// have not been written yet, and a red badge for expected work in
			// progress trains people to ignore red badges.
			status := ui.StatusBadge("unported")
			if _, ok := registry.Lookup(c.Name); ok {
				status = ui.StatusBadge("ok")
			}
			table.Row(ui.Command(c.Name), status, c.Schedule, c.Description)
		}
		fmt.Println(table.Render())
		ui.Newline()
		ui.KV("Implemented", fmt.Sprintf("%d/%d", len(implemented), len(jobs.Crons)))
		ui.Newline()
		return nil
	},
}

func init() {
	cronListCmd.Flags().BoolVar(&flagCronSortByFreq, "by-frequency", false,
		"Sort fastest-first instead of alphabetically")
	cronCmd.AddCommand(cronListCmd, cronRunCmd, cronStatusCmd)
}
