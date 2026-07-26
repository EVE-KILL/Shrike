package cli

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/sde"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var sdeCmd = &cobra.Command{
	Use:   "sde",
	Short: "EVE Static Data Export",
}

var (
	flagSDEOnly         []string
	flagSDECacheDir     string
	flagSDEForce        bool
	flagSDESkipExternal bool
)

// cacheDir resolves where archives are kept. Defaults under .data/ so it shares
// the gitignored directory the docker stack uses and survives between runs — the
// archive is ~99 MB and immutable per build, so re-downloading it every run is
// pure waste.
func cacheDir() string {
	if flagSDECacheDir != "" {
		return flagSDECacheDir
	}
	return filepath.Join(".data", "sde")
}

func userAgent() string {
	if cfg != nil && cfg.ESIUserAgent != "" {
		return cfg.ESIUserAgent
	}
	// CCP asks for an identifying agent on the static-data endpoints.
	return "shrike/dev (+https://github.com/EVE-KILL/Shrike)"
}

var sdeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Compare the loaded SDE build against the published one",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		manifest, err := sde.FetchManifest(cmd.Context(), userAgent())
		if err != nil {
			return err
		}

		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		loaded, loadedRelease, err := sde.LoadedBuild(cmd.Context(), pool)
		if err != nil {
			return err
		}

		current := loaded == manifest.BuildNumber

		if ui.JSONMode {
			return ui.JSON(map[string]any{
				"published":      manifest.BuildNumber,
				"published_date": manifest.ReleaseDate,
				"loaded":         loaded,
				"loaded_date":    loadedRelease,
				"up_to_date":     current,
				"never_imported": loaded == 0,
			})
		}

		ui.Section("SDE")
		ui.KV("Published", fmt.Sprintf("%d  %s", manifest.BuildNumber, ui.Dim(manifest.ReleaseDate)))
		if loaded == 0 {
			ui.KV("Loaded", ui.Dim("never imported"))
		} else {
			ui.KV("Loaded", fmt.Sprintf("%d  %s", loaded, ui.Dim(loadedRelease)))
		}
		ui.Newline()
		if current {
			ui.Success("Up to date.")
		} else {
			ui.Warn("A newer build is available — run %s", "shrike sde:import")
		}
		ui.Newline()
		return nil
	},
}

var sdeImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import the Static Data Export into Postgres",
	Long: `Downloads the published SDE archive and loads it into Postgres.

The archive is cached per build under .data/sde, so re-running is cheap. Import
is idempotent: each table is staged with COPY and merged with ON CONFLICT DO
UPDATE, so a partial or repeated run converges rather than duplicating.

Skips work entirely when the published build is already loaded, unless --force.

--only restricts to named tables or archive members, which is what you want when
iterating on one mapping:

    shrike sde:import --only inv_types --only regions`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		start := time.Now()

		manifest, err := sde.FetchManifest(cmd.Context(), userAgent())
		if err != nil {
			return err
		}

		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		loaded, _, err := sde.LoadedBuild(cmd.Context(), pool)
		if err != nil {
			return err
		}
		if loaded == manifest.BuildNumber && !flagSDEForce && len(flagSDEOnly) == 0 {
			ui.Success("Build %d is already loaded. Use --force to re-import.", loaded)
			ui.Newline()
			return nil
		}

		// Resolve which tables to run before downloading, so a typo in --only
		// fails immediately rather than after a 99 MB transfer.
		// Simple mappings first, then the nested ones — blueprint activities
		// reference types, so types must already be present.
		tables := append(append([]sde.Table{}, sde.Tables...), sde.NestedTables...)
		if len(flagSDEOnly) > 0 {
			tables = nil
			for _, name := range flagSDEOnly {
				t := sde.TableByName(name)
				if t == nil {
					return fmt.Errorf("unknown table or member %q", name)
				}
				tables = append(tables, *t)
			}
		}

		src, err := sde.Open(cmd.Context(), cacheDir(), manifest, userAgent(), func(msg string) {
			ui.Printf("  %s %s\n", ui.Dim("·"), msg)
		})
		if err != nil {
			return err
		}
		defer src.Close()

		ui.Section("Importing")
		table := ui.NewTable("TABLE", "MEMBER", "READ", "WRITTEN", "SKIPPED", "PRUNED", "TIME")
		var results []sde.LoadResult
		for _, t := range tables {
			res, err := sde.Load(cmd.Context(), pool, t, src)
			if err != nil {
				// Report what already succeeded before failing — a mid-import
				// error is much easier to place with the preceding rows visible.
				fmt.Println(table.Render())
				ui.Newline()
				return fmt.Errorf("import %s: %w", t.Name, err)
			}
			results = append(results, res)
			status := res.Elapsed
			if res.Missing {
				status = ui.Dim("absent")
			}
			table.Row(ui.Command(res.Table), res.Member,
				fmtCount(res.Read), fmtCount(res.Written), fmtCount(res.Skipped),
				fmtCount(res.Pruned), status)
		}
		fmt.Println(table.Render())

		// Celestials draw on regions, constellations, systems, types,
		// corporations and operations, so they run once those are all in place.
		if len(flagSDEOnly) == 0 {
			cel, err := sde.ImportCelestials(cmd.Context(), pool, src)
			if err != nil {
				return fmt.Errorf("import celestials: %w", err)
			}
			jumps, err := sde.ImportSystemJumps(cmd.Context(), pool, src)
			if err != nil {
				return fmt.Errorf("import system jumps: %w", err)
			}
			flagCount, flagTime, err := sde.SeedInvFlags(cmd.Context(), pool)
			if err != nil {
				return fmt.Errorf("seed inv_flags: %w", err)
			}

			ui.Section("Derived from the map")
			ct := ui.NewTable("TABLE", "SOURCE", "ROWS", "PRUNED", "TIME")
			ct.Row(ui.Command("celestials"), "9 members",
				fmtCount(cel.Written), fmtCount(cel.Pruned), cel.Elapsed)
			ct.Row(ui.Command("solar_system_jumps"), "mapStargates",
				fmtCount(jumps.Written), fmtCount(jumps.Pruned), jumps.Elapsed)
			ct.Row(ui.Command("inv_flags"), ui.Dim("bundled"),
				fmtCount(flagCount), "0", flagTime)
			fmt.Println(ct.Render())
		}

		// Seeded before the derivations, so Dust groups get their category_id
		// denormalised along with everything else.
		if len(flagSDEOnly) == 0 {
			seed, err := sde.SeedDust514(cmd.Context(), pool)
			if err != nil {
				return fmt.Errorf("seed dust514: %w", err)
			}
			ui.Section("Dust 514 (bundled, removed from the SDE in 2017)")
			st := ui.NewTable("CATEGORIES", "GROUPS", "MARKET GROUPS", "TYPES", "TIME")
			st.Row(fmtCount(seed.Categories), fmtCount(seed.Groups),
				fmtCount(seed.MarketGroups), fmtCount(seed.Types), seed.Elapsed)
			fmt.Println(st.Render())
		}

		// EVE Ref feeds and the bundled price lists. Network-dependent, so
		// --skip-external exists for working offline or iterating on the archive.
		if len(flagSDEOnly) == 0 && !flagSDESkipExternal {
			ui.Section("External sources")
			et := ui.NewTable("SOURCE", "ROWS", "TIME")
			ins, err := sde.ImportInsurancePrices(cmd.Context(), pool, userAgent())
			if err != nil {
				return err
			}
			et.Row(ui.Command(ins.Name), fmtCount(ins.Rows), ins.Elapsed)

			str, err := sde.ImportStructures(cmd.Context(), pool, userAgent())
			if err != nil {
				return err
			}
			et.Row(ui.Command(str.Name), fmtCount(str.Rows), str.Elapsed)

			// Generated first so the hand-set list below can override it: a few
			// hulls are both supercapitals and manually priced, and the manual
			// value is the deliberate one. Matches the TypeScript ordering.
			//
			// Needs the prices table, which a different importer fills; writes
			// nothing rather than failing when it is empty.
			sup, err := sde.GenerateSupercapPrices(cmd.Context(), pool)
			if err != nil {
				return err
			}
			rows := fmtCount(sup.Rows)
			if sup.Rows == 0 {
				rows = ui.Dim("0 (no market prices loaded)")
			}
			et.Row(ui.Command(sup.Name), rows, sup.Elapsed)

			man, err := sde.SeedManualCustomPrices(cmd.Context(), pool)
			if err != nil {
				return err
			}
			et.Row(ui.Command(man.Name), fmtCount(man.Rows), man.Elapsed)
			fmt.Println(et.Render())
		}

		ui.Section("Derived")
		dt := ui.NewTable("DERIVATION", "ROWS", "TIME")
		derived, err := sde.Derive(cmd.Context(), pool)
		for _, d := range derived {
			dt.Row(d.Name, fmtCount(d.Rows), d.Elapsed)
		}
		// Depends on celestials, so it cannot live in the ordered list above.
		if err == nil && len(flagSDEOnly) == 0 {
			var sn sde.DeriveResult
			sn, err = sde.DeriveOne(cmd.Context(), pool, sde.StationNameDerivation)
			dt.Row(sn.Name, fmtCount(sn.Rows), sn.Elapsed)
		}
		fmt.Println(dt.Render())
		if err != nil {
			ui.Newline()
			return fmt.Errorf("derive: %w", err)
		}

		// Only stamp the build once everything has succeeded, so an interrupted
		// import does not look complete to the next run.
		if len(flagSDEOnly) == 0 {
			if err := sde.RecordBuild(cmd.Context(), pool, manifest.BuildNumber, manifest.ReleaseDate); err != nil {
				return err
			}
		}

		var read, written int64
		for _, r := range results {
			read += r.Read
			written += r.Written
		}
		ui.Newline()
		ui.KV("Build", strconv.FormatInt(manifest.BuildNumber, 10))
		ui.KV("Rows read", fmtCount(read))
		ui.KV("Rows written", fmtCount(written))
		ui.KV("Total", time.Since(start).Round(time.Millisecond).String())
		ui.Newline()
		if len(flagSDEOnly) > 0 {
			ui.Warn("Partial import (--only): the loaded build was not stamped.")
			ui.Newline()
		}
		return nil
	},
}

var sdeVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Compare local SDE row counts against a reference database",
	Long: `Counts rows in every archive-owned and derived SDE table.

Run it once locally and once with --config .env.prod. Matching counts are the
first parity check; primary-key and value comparisons remain the stronger
check when a count differs.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		ui.Section("Row counts")
		t := ui.NewTable("TABLE", "ROWS")
		var total int64
		tables := sde.AllTables()
		tableNames := make([]string, 0, len(tables)+3)
		for _, tbl := range tables {
			tableNames = append(tableNames, tbl.Name)
		}
		tableNames = append(tableNames,
			"celestials",
			"solar_system_jumps",
			"inv_flags",
		)
		for _, tableName := range tableNames {
			var n int64
			// Exact counts are intentional. type_dogma_attributes is the
			// largest table at roughly 646k rows, still small enough that an
			// index-only count is preferable to hiding stale rows behind a
			// reltuples estimate.
			if err := pool.QueryRow(cmd.Context(),
				"SELECT count(*) FROM "+tableName).Scan(&n); err != nil {
				return err
			}
			total += n
			t.Row(ui.Command(tableName), fmtCount(n))
		}
		fmt.Println(t.Render())
		ui.Newline()
		ui.KV("Total", fmtCount(total))
		ui.Newline()
		fmt.Printf("  %s\n", ui.Dim("Compare against production with:"))
		fmt.Printf("  %s\n", ui.Dim("  shrike sde:verify --config .env.prod"))
		ui.Newline()
		return nil
	},
}

// fmtCount groups thousands so 645752 reads as 645,752 in a narrow column.
func fmtCount(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}

func init() {
	sdeImportCmd.Flags().StringArrayVar(&flagSDEOnly, "only", nil,
		"Restrict to named tables or archive members (repeatable)")
	sdeImportCmd.Flags().BoolVar(&flagSDESkipExternal, "skip-external", false,
		"Skip the EVE Ref feeds (insurance prices, structures)")
	sdeImportCmd.Flags().BoolVar(&flagSDEForce, "force", false,
		"Re-import even when the published build is already loaded")
	for _, c := range []*cobra.Command{sdeImportCmd, sdeStatusCmd} {
		c.Flags().StringVar(&flagSDECacheDir, "cache-dir", "",
			"Where to keep downloaded archives (default .data/sde)")
	}
	sdeCmd.AddCommand(sdeStatusCmd, sdeImportCmd, sdeVerifyCmd)
}
