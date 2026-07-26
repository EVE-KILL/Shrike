package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/configstore"
	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/everef"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

var everefCmd = &cobra.Command{
	Use:   "everef",
	Short: "Import the datasets published at data.everef.net",
	Long: `EVE Ref republishes what CCP only exposes as a live snapshot, with history.

That history is the only route to a killboard that goes back to 2007: insurance
payouts, daily Jita market history since 2007, the sovereignty map since 2017,
every war since 2003, and a daily archive of every killmail.`,
}

var (
	flagEverefFrom     string
	flagEverefTo       string
	flagEverefDays     int
	flagEverefDate     string
	flagEverefBackfill bool
	flagEverefLatest   bool
	flagEverefCurrent  bool
	flagEverefSkipKM   bool
	flagEverefSkipDone bool
	flagEverefReverse  bool
	flagEverefRedo     bool
)

// openPool is the preamble every importer shares.
func openPool(cmd *cobra.Command) (*pgxpool.Pool, error) {
	if err := requireConfig(); err != nil {
		return nil, err
	}
	return db.New(cmd.Context(), cfg)
}

var everefInsuranceCmd = &cobra.Command{
	Use:   "insurance",
	Short: "Replace insurance payouts with the published snapshot",
	Long: `Insurance has no history — only a current snapshot — so the table is
replaced rather than merged. A ship whose policy CCP withdrew has to disappear,
and an upsert would leave it behind forever.

The replacement runs in one transaction, so a failed import leaves the previous
snapshot intact.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		res, err := everef.ImportInsurance(cmd.Context(), pool, everef.NewClient(userAgent()))
		if err != nil {
			return err
		}
		return reportResults("Insurance", []everef.Result{res})
	},
}

var everefPricesCmd = &cobra.Command{
	Use:   "prices",
	Short: "Import daily Jita market history",
	Long: `Loads one bzip2'd CSV per day and merges it into the prices table.

Only The Forge is stored — it contains Jita, which is the price every valuation
in this codebase means. Keeping the other hundred-odd regions would multiply a
23-million-row table tenfold for data nothing reads.

A day EVE Ref has not published yet is reported as absent rather than failing;
the most recent day is routinely unavailable for several hours.

    shrike everef:prices                      # the last 7 days
    shrike everef:prices --days 30
    shrike everef:prices --date 2026-07-22
    shrike everef:prices --from 2024-01-01 --to 2024-12-31
    shrike everef:prices --backfill           # from the latest day held to today`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		dates, err := resolvePriceDates(cmd.Context(), pool)
		if err != nil {
			return err
		}
		if len(dates) == 0 {
			ui.Success("Prices are already up to date")
			ui.Newline()
			return nil
		}

		var each []everef.Result
		client := everef.NewClient(userAgent())
		total, err := everef.ImportPrices(cmd.Context(), pool, client, dates, func(r everef.Result) {
			each = append(each, r)
		})
		if err != nil {
			_ = reportResults("Market history", each)
			return err
		}
		_ = total
		return reportResults("Market history", each)
	},
}

// resolvePriceDates turns the flag combination into the list of days to fetch.
func resolvePriceDates(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	switch {
	case flagEverefDate != "":
		return []string{flagEverefDate}, nil

	case flagEverefBackfill:
		latest, err := everef.LatestPriceDate(ctx, pool)
		if err != nil {
			return nil, err
		}
		if latest == "" {
			return nil, fmt.Errorf("no prices held — use --from to choose a start date")
		}
		next, err := everef.DayAfter(latest)
		if err != nil {
			return nil, err
		}
		if next > everef.Today() {
			return nil, nil
		}
		return everef.DateRange(next, everef.Today())

	case flagEverefFrom != "":
		to := flagEverefTo
		if to == "" {
			to = everef.Today()
		}
		return everef.DateRange(flagEverefFrom, to)

	default:
		// TypeScript defines "last N days" as N days ago through today,
		// inclusive. Today is often unpublished, but attempting it means a late
		// run sees the same rows as the original command.
		start := time.Now().UTC().AddDate(0, 0, -flagEverefDays).Format("2006-01-02")
		return everef.DateRange(start, everef.Today())
	}
}

var everefSovereigntyCmd = &cobra.Command{
	Use:   "sovereignty",
	Short: "Import the sovereignty map and its history",
	Long: `Records who holds which system, and appends to the history log when that
changes.

    2017-2022     yearly archives of hourly snapshots, one per day used
    2022-12-16 +  one published snapshot per day
    --latest      the current snapshot only

Note a divergence from the TypeScript cron, which is incorrect: it writes
"SET alliance_id = sovereignty.alliance_id", assigning each column to itself, so
the current-state table never advances. In production sovereignty.updated_at has
not moved since 2026-03-22 while sovereignty_history keeps growing, because
every run re-detects the same differences. This implementation updates the row.

    shrike everef:sovereignty --latest
    shrike everef:sovereignty --from 2017
    shrike everef:sovereignty --from 2023-01-01 --to 2023-12-31`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		client := everef.NewClient(userAgent())

		if flagEverefLatest || flagEverefFrom == "" {
			// With no range and no saved bookmark there is nothing to replay,
			// so the current snapshot is the only sensible default.
			saved, err := configstore.Get(cmd.Context(), pool, configstore.KeySovereigntyDate)
			if err != nil {
				return err
			}
			if flagEverefLatest || saved == "" {
				res, err := everef.ImportSovereigntyLatest(cmd.Context(), pool, client)
				if err != nil {
					return err
				}
				return reportResults("Sovereignty", []everef.Result{res})
			}
			flagEverefFrom = saved
			ui.KV("Resuming from", saved)
		}

		to := flagEverefTo
		if to == "" {
			to = everef.Today()
		}
		from := flagEverefFrom
		if len(from) == 4 {
			from += "-01-01"
		}

		var each []everef.Result
		total, err := everef.ImportSovereigntyRange(cmd.Context(), pool, client, from, to, func(r everef.Result) {
			if r.Rows > 0 {
				each = append(each, r)
			}
		})
		if err != nil {
			return err
		}
		if len(each) == 0 {
			// Nothing changed on any day; show the run rather than an empty table.
			each = append(each, total)
		}
		return reportResults("Sovereignty", each)
	},
}

var everefWarsCmd = &cobra.Command{
	Use:   "wars",
	Short: "Import wars and the killmails fought under them",
	Long: `War archives hold two kinds of document: the war record, and every killmail
fought under it.

The killmails matter beyond the war itself. A war kill is the only place war_id
is ever stated — the public killmail endpoint never carries one — so without
this import a war shows no activity at all. Kills already stored from the live
queue get their war_id filled in rather than reinserted.

    2003-2020  one archive per year
    2021 +     one archive per day
    --current  the active wars, metadata only

    shrike everef:wars --current
    shrike everef:wars --from 2003
    shrike everef:wars --from 2021 --to 2023 --skip-killmails`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		cache, prices, err := loadLookups(cmd.Context(), pool)
		if err != nil {
			return err
		}

		imp := &everef.WarImport{
			Pool:          pool,
			Client:        everef.NewClient(userAgent()),
			Cache:         cache,
			Prices:        prices,
			SkipKillmails: flagEverefSkipKM,
		}

		if flagEverefCurrent {
			res, err := imp.ImportCurrentWars(cmd.Context())
			if err != nil {
				return err
			}
			return reportResults("Wars", []everef.Result{res})
		}

		fromYear, err := resolveWarStartYear(cmd.Context(), pool)
		if err != nil {
			return err
		}
		toYear := time.Now().UTC().Year()
		if flagEverefTo != "" {
			toYear, err = strconv.Atoi(flagEverefTo[:4])
			if err != nil {
				return fmt.Errorf("invalid --to %q", flagEverefTo)
			}
		}

		var each []everef.Result
		total, err := imp.ImportWarYears(cmd.Context(), fromYear, toYear, flagEverefRedo, func(r everef.Result) {
			each = append(each, r)
		})
		if err != nil {
			_ = reportResults("Wars", each)
			return err
		}
		_ = total
		return reportResults("Wars", each)
	},
}

func resolveWarStartYear(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	if flagEverefFrom != "" {
		y, err := strconv.Atoi(flagEverefFrom[:4])
		if err != nil {
			return 0, fmt.Errorf("invalid --from %q: expected a year or YYYY-MM-DD", flagEverefFrom)
		}
		return y, nil
	}
	saved, err := configstore.Get(ctx, pool, configstore.KeyWarsLastYear)
	if err != nil {
		return 0, err
	}
	if saved == "" {
		return 2003, nil
	}
	y, err := strconv.Atoi(saved)
	if err != nil {
		return 2003, nil
	}
	ui.KV("Resuming from", saved)
	return y, nil
}

var everefKillmailsCmd = &cobra.Command{
	Use:   "killmails",
	Short: "Import daily killmail archives",
	Long: `Each archive holds fifteen to twenty-five thousand killmails as raw ESI
documents, which go through the same parser a live mail does — an imported kill
and a streamed kill are indistinguishable once stored.

Every mail in an archive shares a kill date, so the day's prices are resolved in
one query rather than one per killmail.

    shrike everef:killmails --backfill                 # from the latest kill held
    shrike everef:killmails --from 2026-07-01
    shrike everef:killmails --from 2025-01-01 --to 2025-06-30 --skip-existing`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		cache, prices, err := loadLookups(cmd.Context(), pool)
		if err != nil {
			return err
		}

		client := everef.NewClient(userAgent())
		from, err := resolveKillmailStart(cmd.Context(), pool)
		if err != nil {
			return err
		}
		to := flagEverefTo
		if to == "" {
			to = everef.Yesterday()
		}
		if from > to {
			ui.Success("Already up to date")
			ui.Newline()
			return nil
		}

		dates, err := everef.DiscoverKillmailDays(cmd.Context(), client, from, to)
		if err != nil {
			return err
		}
		if len(dates) == 0 {
			ui.Warn("No archives published between %s and %s", from, to)
			ui.Newline()
			return nil
		}
		if flagEverefReverse {
			for i, j := 0, len(dates)-1; i < j; i, j = i+1, j-1 {
				dates[i], dates[j] = dates[j], dates[i]
			}
		}

		ui.KV("Archives", fmt.Sprintf("%d days, %s to %s", len(dates), dates[0], dates[len(dates)-1]))
		ui.Newline()

		imp := &everef.KillmailImport{
			Pool: pool, Client: client, Cache: cache, Prices: prices,
			SkipExisting: flagEverefSkipDone,
		}

		var each []everef.Result
		total, err := imp.ImportKillmails(cmd.Context(), dates, func(r everef.Result) {
			each = append(each, r)
		})
		if err != nil {
			_ = reportResults("Killmails", each)
			return err
		}
		_ = total
		return reportResults("Killmails", each)
	},
}

func resolveKillmailStart(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	if flagEverefFrom != "" {
		return flagEverefFrom, nil
	}
	if flagEverefBackfill {
		latest, err := everef.LatestKillmailDate(ctx, pool)
		if err != nil {
			return "", err
		}
		if latest == "" {
			return "", fmt.Errorf("no killmails held — use --from to choose a start date")
		}
		return everef.DayAfter(latest)
	}
	saved, err := configstore.Get(ctx, pool, configstore.KeyKillmailsLastDate)
	if err != nil {
		return "", err
	}
	if saved == "" {
		return "", fmt.Errorf("no saved position — use --from or --backfill")
	}
	// Resume at the saved day rather than past it: the bookmark advances only
	// when a day completes, so re-running it costs a skipped insert and never
	// loses a partial day.
	ui.KV("Resuming from", saved)
	return saved, nil
}

// reportResults renders what an import did, in the same shape for all five.
func reportResults(title string, results []everef.Result) error {
	if ui.JSONMode {
		return ui.JSON(results)
	}

	ui.Section(title)
	if len(results) == 0 {
		ui.KV("Result", ui.Dim("nothing to do"))
		ui.Newline()
		return nil
	}

	t := ui.NewTable("SOURCE", "SEEN", "WRITTEN", "RELATED", "SKIPPED", "TIME")
	var rows, related, adjusted, seen, skipped, failed int64
	for _, r := range results {
		written := fmtCount(r.Rows)
		if r.Missing {
			written = ui.Dim("not published")
			failed++
		}
		t.Row(r.Name, fmtCount(r.Seen), written, fmtCount(r.Related), fmtCount(r.Skipped), r.Elapsed)
		rows += r.Rows
		related += r.Related
		adjusted += r.Adjusted
		seen += r.Seen
		skipped += r.Skipped
		failed += r.Failed
	}
	fmt.Println(t.Render())

	ui.Newline()
	ui.KV("Rows written", fmtCount(rows))
	if related > 0 {
		ui.KV("Related rows", fmtCount(related))
	}
	if adjusted > 0 {
		ui.KV("Existing rows corrected", fmtCount(adjusted))
	}
	if skipped > 0 {
		ui.KV("Skipped", fmtCount(skipped))
	}
	if failed > 0 {
		ui.KV("Unavailable", fmtCount(failed))
	}
	ui.Newline()
	return nil
}

func init() {
	everefPricesCmd.Flags().IntVar(&flagEverefDays, "days", 7, "Days to look back from today")
	everefPricesCmd.Flags().StringVar(&flagEverefDate, "date", "", "Import a single day (YYYY-MM-DD)")
	everefPricesCmd.Flags().StringVar(&flagEverefFrom, "from", "", "Start date (YYYY-MM-DD)")
	everefPricesCmd.Flags().StringVar(&flagEverefTo, "to", "", "End date (YYYY-MM-DD), defaults to today")
	everefPricesCmd.Flags().BoolVar(&flagEverefBackfill, "backfill", false, "Fill the gap from the latest day held to today")

	everefSovereigntyCmd.Flags().BoolVar(&flagEverefLatest, "latest", false, "Apply the current snapshot only")
	everefSovereigntyCmd.Flags().StringVar(&flagEverefFrom, "from", "", "Start date (YYYY-MM-DD) or year (YYYY)")
	everefSovereigntyCmd.Flags().StringVar(&flagEverefTo, "to", "", "End date (YYYY-MM-DD), defaults to today")

	everefWarsCmd.Flags().BoolVar(&flagEverefCurrent, "current", false, "Import the active wars only, without killmails")
	everefWarsCmd.Flags().StringVar(&flagEverefFrom, "from", "", "Start year (YYYY) or date")
	everefWarsCmd.Flags().StringVar(&flagEverefTo, "to", "", "End year or date, defaults to this year")
	everefWarsCmd.Flags().BoolVar(&flagEverefSkipKM, "skip-killmails", false, "Import war metadata only")
	everefWarsCmd.Flags().BoolVar(&flagEverefRedo, "reprocess", false, "Ignore the saved position and redo every archive")

	everefKillmailsCmd.Flags().StringVar(&flagEverefFrom, "from", "", "Start date (YYYY-MM-DD)")
	everefKillmailsCmd.Flags().StringVar(&flagEverefTo, "to", "", "End date (YYYY-MM-DD), defaults to yesterday")
	everefKillmailsCmd.Flags().BoolVar(&flagEverefBackfill, "backfill", false, "Start from the day after the latest kill held")
	everefKillmailsCmd.Flags().BoolVar(&flagEverefReverse, "reverse", false, "Process the newest day first")
	everefKillmailsCmd.Flags().BoolVar(&flagEverefSkipDone, "skip-existing", false, "Skip killmails already stored, avoiding the parse")

	everefCmd.AddCommand(
		everefInsuranceCmd,
		everefPricesCmd,
		everefSovereigntyCmd,
		everefWarsCmd,
		everefKillmailsCmd,
	)
}
