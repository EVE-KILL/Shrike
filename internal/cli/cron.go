package cli

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/ui"
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

func init() {
	cronListCmd.Flags().BoolVar(&flagCronSortByFreq, "by-frequency", false,
		"Sort fastest-first instead of alphabetically")
	cronCmd.AddCommand(cronListCmd)
}
