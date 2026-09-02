package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

var killmailCmd = &cobra.Command{
	Use:   "killmail",
	Short: "Fetch, parse and inspect killmails",
}

var (
	flagKMDryRun    bool
	flagKMForce     bool
	flagKMWarID     int32
	flagKMAgainst   string
	flagKMTolerance float64
	flagKMFromFile  string
)

var killmailProcessCmd = &cobra.Command{
	Use:   "process <killmail-id> <hash>",
	Short: "Fetch a killmail from ESI, parse it, and store it",
	Long: `Runs one killmail through the whole pipeline: ESI fetch, parse, insert.

The killmail endpoint is public — the hash is the credential — so this needs no
ESI token and works against a fresh local database.

    shrike killmail:process 137258027 1d9365aaed385213867e40390d29cd4c7596e0e3
    shrike killmail:process 137258027 1d93... --dry-run

--dry-run parses and prints without writing, which is the fast way to compare
valuation changes. --force replaces a killmail that is already stored.

--from-file replays a killmail from a saved ESI response instead of fetching it.
The queue accepts a raw killmail the same way, and it is the only way to compare
two parsers fairly: ESI does not guarantee a stable attacker order between
requests, so two fetches of the same mail can disagree on it.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		killmailID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid killmail id %q", args[0])
		}
		hash := args[1]

		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		cache, prices, err := loadLookups(cmd.Context(), pool)
		if err != nil {
			return err
		}

		var km killmail.ESIKillmail
		if flagKMFromFile != "" {
			// #nosec G304 -- flagKMFromFile is an explicit operator-provided input path.
			raw, err := os.ReadFile(flagKMFromFile)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(raw, &km); err != nil {
				return fmt.Errorf("parse %s: %w", flagKMFromFile, err)
			}
		} else {
			client := esi.New(cfg)
			defer client.Close()

			res, err := esi.Get[killmail.ESIKillmail](cmd.Context(), client, esi.KillmailPath(killmailID, hash))
			if err != nil {
				return err
			}
			if !res.OK() {
				// A 404 or 422 here means the id and hash do not go together,
				// which no retry will fix; anything else is worth another go.
				return fmt.Errorf("ESI returned %d for killmail %d", res.Status, killmailID)
			}
			km = *res.Data
		}

		parsed, err := killmail.Parse(cmd.Context(), cache, prices, &km, hash, flagKMWarID)
		if err != nil {
			return err
		}

		stored := false
		if !flagKMDryRun {
			if flagKMForce {
				if err := killmail.Delete(cmd.Context(), pool, killmailID); err != nil {
					return err
				}
			}
			stored, err = killmail.Insert(cmd.Context(), pool, parsed)
			if err != nil {
				return err
			}
		}

		if ui.JSONMode {
			return ui.JSON(map[string]any{
				"killmail_id": killmailID,
				"stored":      stored,
				"dry_run":     flagKMDryRun,
				"killmail":    parsed.Killmail,
				"attackers":   parsed.Attackers,
				"items":       parsed.Items,
			})
		}

		printKillmail(cache, parsed)
		switch {
		case flagKMDryRun:
			ui.KV("Stored", ui.Dim("no — dry run"))
		case stored:
			ui.Success("Stored killmail %d", killmailID)
		default:
			ui.KV("Stored", ui.Dim("no — already present (use --force to replace)"))
		}
		ui.Newline()
		return nil
	},
}

var killmailShowCmd = &cobra.Command{
	Use:   "show <killmail-id>",
	Short: "Print a stored killmail",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		killmailID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid killmail id %q", args[0])
		}

		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		parsed, err := killmail.Load(cmd.Context(), pool, killmailID)
		if err != nil {
			return err
		}
		cache, err := eve.Load(cmd.Context(), pool)
		if err != nil {
			return err
		}

		if ui.JSONMode {
			return ui.JSON(parsed)
		}
		printKillmail(cache, parsed)
		ui.Newline()
		return nil
	},
}

var killmailCompareCmd = &cobra.Command{
	Use:   "compare <killmail-id>",
	Short: "Diff a stored killmail against another database",
	Long: `Compares every column of a stored killmail, its attackers and its items
against the same killmail in another database.

This is how the port is verified: process a killmail production already holds,
then diff the result row by row. A mismatch is a porting bug, with the exception
of values on rarely traded hulls, whose custom prices move between imports.

    shrike killmail:compare 137258027
    shrike killmail:compare 137258027 --against .env.prod --tolerance 0.01`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		killmailID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid killmail id %q", args[0])
		}

		local, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer local.Close()

		otherCfg, err := config.Load(flagKMAgainst)
		if err != nil {
			return fmt.Errorf("load %s: %w", flagKMAgainst, err)
		}
		other, err := db.New(cmd.Context(), otherCfg)
		if err != nil {
			return fmt.Errorf("connect to %s: %w", flagKMAgainst, err)
		}
		defer other.Close()

		mine, err := killmail.Load(cmd.Context(), local, killmailID)
		if err != nil {
			return fmt.Errorf("local: %w", err)
		}
		theirs, err := killmail.Load(cmd.Context(), other, killmailID)
		if err != nil {
			return fmt.Errorf("%s: %w", flagKMAgainst, err)
		}

		diffs := killmail.Diff(mine, theirs, flagKMTolerance)

		if ui.JSONMode {
			return ui.JSON(map[string]any{
				"killmail_id": killmailID,
				"against":     flagKMAgainst,
				"matches":     len(diffs) == 0,
				"differences": diffs,
			})
		}

		ui.Section(fmt.Sprintf("Killmail %d vs %s", killmailID, flagKMAgainst))
		if len(diffs) == 0 {
			ui.Success("Identical across all %d columns, %d attackers and %d items",
				killmail.KillmailColumns, len(mine.Attackers), len(mine.Items))
			ui.Newline()
			return nil
		}

		t := ui.NewTable("FIELD", "LOCAL", strings.ToUpper(flagKMAgainst))
		for _, d := range diffs {
			t.Row(d.Field, d.Mine, d.Theirs)
		}
		fmt.Println(t.Render())
		ui.Newline()
		return fmt.Errorf("%d differences", len(diffs))
	},
}

var killmailCacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Report what the runtime lookup cache holds",
	Long: `Loads the static-data cache the parser runs on and reports its contents.

Useful as a pre-flight check: a killmail parsed against an empty cache produces
rows full of nulls rather than an error, because a missing type is
indistinguishable from a type with no group.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		cache, err := eve.Load(cmd.Context(), pool)
		if err != nil {
			return err
		}
		counts := cache.CountsByName()

		if ui.JSONMode {
			return ui.JSON(counts)
		}

		ui.Section("Runtime lookup cache")
		t := ui.NewTable("TABLE", "ROWS")
		for _, name := range []string{
			"inv_types", "inv_groups", "solar_systems", "regions",
			"constellations", "type_dogma_attributes", "custom_prices",
		} {
			rows := fmtCount(int64(counts[name]))
			if counts[name] == 0 {
				rows = ui.Warn2("empty")
			}
			t.Row(name, rows)
		}
		fmt.Println(t.Render())
		ui.Newline()
		return nil
	},
}

// loadLookups builds the two things every parse needs.
func loadLookups(ctx context.Context, pool *pgxpool.Pool) (*eve.Cache, *eve.Prices, error) {
	cache, err := eve.Load(ctx, pool)
	if err != nil {
		return nil, nil, err
	}
	return cache, eve.NewPrices(pool, cache), nil
}

func printKillmail(cache *eve.Cache, p *killmail.Parsed) {
	km := p.Killmail
	system, _ := cache.System(km.SolarSystemID)
	region, _ := cache.Region(km.RegionID)
	shipType, _ := cache.Type(km.VictimShipTypeID)

	ui.Section(fmt.Sprintf("Killmail %d", km.KillmailID))
	ui.KV("Time", km.KillmailTime.UTC().Format("2006-01-02 15:04:05 UTC"))
	ui.KV("System", fmt.Sprintf("%s (%.2f) — %s", orDim(system.Name), system.Security, orDim(region.Name)))
	ui.KV("Victim ship", orDim(shipType.Name))
	ui.Newline()

	t := ui.NewTable("VALUE", "ISK")
	t.Row("Total", fmtISK(km.TotalValue))
	t.Row("Fitted", fmtISK(km.FittedValue))
	t.Row("Dropped", fmtISK(km.DroppedValue))
	t.Row("Destroyed", fmtISK(km.DestroyedValue))
	fmt.Println(t.Render())
	ui.Newline()

	ui.KV("Points", strconv.Itoa(int(km.Points)))
	ui.KV("Attackers", fmt.Sprintf("%d", len(p.Attackers)))
	ui.KV("Items", fmt.Sprintf("%d", len(p.Items)))
	ui.KV("Solo", yesNo(km.IsSolo))
	ui.KV("NPC", yesNo(km.IsNPC))
}

func orDim(s string) string {
	if s == "" {
		return ui.Dim("unknown")
	}
	return s
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return ui.Dim("no")
}

// fmtISK renders a value the way a killboard does: thousands separated, and
// abbreviated once the exact figure stops being the interesting part.
func fmtISK(v float64) string {
	switch {
	case v >= 1e12:
		return fmt.Sprintf("%.2f T", v/1e12)
	case v >= 1e9:
		return fmt.Sprintf("%.2f B", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.2f M", v/1e6)
	default:
		return fmt.Sprintf("%.2f", v)
	}
}

func init() {
	killmailProcessCmd.Flags().BoolVar(&flagKMDryRun, "dry-run", false, "Parse and print without writing")
	killmailProcessCmd.Flags().BoolVar(&flagKMForce, "force", false, "Replace the killmail if it is already stored")
	killmailProcessCmd.Flags().Int32Var(&flagKMWarID, "war", 0, "Associate the killmail with a war")
	killmailProcessCmd.Flags().StringVar(&flagKMFromFile, "from-file", "", "Read the ESI response from a file instead of fetching it")

	killmailCompareCmd.Flags().StringVar(&flagKMAgainst, "against", ".env.prod", "Env file naming the database to compare against")
	killmailCompareCmd.Flags().Float64Var(&flagKMTolerance, "tolerance", 0,
		"Relative tolerance for float columns (0.01 = 1%)")

	killmailCmd.AddCommand(killmailProcessCmd, killmailShowCmd, killmailCompareCmd, killmailCacheCmd)
}
