package cli

import (
	"fmt"

	"github.com/eve-kill/shrike/internal/configstore"
	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

// Importing prices lives under `everef:prices`, with the rest of the datasets
// from the same publisher. What stays here is the question that has nothing to
// do with importing: what does the table currently hold.
var pricesCmd = &cobra.Command{
	Use:   "prices",
	Short: "Inspect stored market history",
}

var pricesStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show what market history is loaded",
	Long: `Reports the extent of the prices table.

A gap here is worth noticing: every killmail is valued from the most recent
average at or before its kill date, so missing days do not fail — they quietly
value items at an older price, or at the 0.01 ISK floor.

Prices are imported with ` + "`shrike everef:prices`" + `.`,
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
		var earliest, latest *string
		if err := pool.QueryRow(cmd.Context(), `
            SELECT count(*), count(DISTINCT date), count(DISTINCT type_id),
                   min(date)::text, max(date)::text
            FROM prices
        `).Scan(&rows, &days, &types, &earliest, &latest); err != nil {
			return err
		}

		bookmark, err := configstore.Get(cmd.Context(), pool, configstore.KeyPricesLastDate)
		if err != nil {
			return err
		}

		if ui.JSONMode {
			return ui.JSON(map[string]any{
				"rows": rows, "days": days, "types": types,
				"earliest": deref(earliest), "latest": deref(latest),
				"bookmark": bookmark,
			})
		}

		ui.Section("Market history")
		ui.KV("Rows", fmtCount(rows))
		ui.KV("Days", fmtCount(days))
		ui.KV("Types", fmtCount(types))
		if latest == nil {
			ui.KV("Range", ui.Dim("none loaded"))
		} else {
			ui.KV("Range", fmt.Sprintf("%s to %s", deref(earliest), deref(latest)))
		}
		if bookmark == "" {
			ui.KV("Import bookmark", ui.Dim("unset"))
		} else {
			ui.KV("Import bookmark", bookmark)
		}
		ui.Newline()
		return nil
	},
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func init() {
	pricesCmd.AddCommand(pricesStatusCmd)
}
