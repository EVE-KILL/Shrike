package cli

import (
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/sde"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var pricesCmd = &cobra.Command{
	Use:   "prices",
	Short: "EVE Ref market history",
}

var (
	flagPriceDays int
	flagPriceDate string
)

var pricesImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import daily market history from EVE Ref",
	Long: `Loads one bzip2'd CSV per day and merges it into the prices table.

Defaults to the last 7 days, which is what the scheduled job uses. A day that
EVE Ref has not published yet is reported as absent rather than failing — the
most recent day is routinely unavailable for several hours.

Supercapital valuation depends on this data: those hulls never trade, so their
custom price is computed from blueprint materials priced at market.

    shrike prices:import --days 30
    shrike prices:import --date 2026-07-22`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		ui.Section("Market history")
		t := ui.NewTable("DATE", "ROWS", "TIME")
		var total int64

		record := func(r sde.PriceDayResult) {
			rows := fmtCount(r.Rows)
			if r.Missing {
				rows = ui.Dim("not published")
			}
			total += r.Rows
			t.Row(r.Date, rows, r.Elapsed)
		}

		if flagPriceDate != "" {
			day, perr := time.Parse("2006-01-02", flagPriceDate)
			if perr != nil {
				return fmt.Errorf("invalid --date %q: expected YYYY-MM-DD", flagPriceDate)
			}
			r, ierr := sde.ImportPriceDay(cmd.Context(), pool, day, userAgent())
			if ierr != nil {
				return ierr
			}
			record(r)
		} else {
			if _, err := sde.ImportPriceRange(cmd.Context(), pool, flagPriceDays, userAgent(), record); err != nil {
				fmt.Println(t.Render())
				return err
			}
		}

		fmt.Println(t.Render())
		ui.Newline()
		ui.KV("Rows merged", fmtCount(total))

		latest, err := sde.LatestPriceDate(cmd.Context(), pool)
		if err != nil {
			return err
		}
		if latest != "" {
			ui.KV("Latest held", latest)
			if err := sde.RecordPriceProgress(cmd.Context(), pool, latest); err != nil {
				return err
			}
		}
		ui.Newline()
		return nil
	},
}

var pricesStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show what market history is loaded",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		var rows, days, types int64
		if err := pool.QueryRow(cmd.Context(), `
            SELECT count(*), count(DISTINCT date), count(DISTINCT type_id) FROM prices
        `).Scan(&rows, &days, &types); err != nil {
			return err
		}
		latest, err := sde.LatestPriceDate(cmd.Context(), pool)
		if err != nil {
			return err
		}

		if ui.JSONMode {
			return ui.JSON(map[string]any{
				"rows": rows, "days": days, "types": types, "latest": latest,
			})
		}

		ui.Section("Market history")
		ui.KV("Rows", fmtCount(rows))
		ui.KV("Days", fmtCount(days))
		ui.KV("Types", fmtCount(types))
		if latest == "" {
			ui.KV("Latest", ui.Dim("none loaded"))
		} else {
			ui.KV("Latest", latest)
		}
		ui.Newline()
		return nil
	},
}

func init() {
	pricesImportCmd.Flags().IntVar(&flagPriceDays, "days", 7, "Number of days to import, ending yesterday")
	pricesImportCmd.Flags().StringVar(&flagPriceDate, "date", "", "Import a single day (YYYY-MM-DD)")
	pricesCmd.AddCommand(pricesImportCmd, pricesStatusCmd)
}
