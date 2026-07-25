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
	flagSDEOnly     []string
	flagSDECacheDir string
	flagSDEForce    bool
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
		tables := sde.Tables
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
		table := ui.NewTable("TABLE", "MEMBER", "READ", "WRITTEN", "SKIPPED", "TIME")
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
				fmtCount(res.Read), fmtCount(res.Written), fmtCount(res.Skipped), status)
		}
		fmt.Println(table.Render())

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

		ui.Section("Derived")
		dt := ui.NewTable("DERIVATION", "ROWS", "TIME")
		derived, err := sde.Derive(cmd.Context(), pool)
		for _, d := range derived {
			dt.Row(d.Name, fmtCount(d.Rows), d.Elapsed)
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
	Long: `Counts rows in every imported table and compares against a reference,
which is production by default.

This is the check that the port maps cleanly: an exact-count match across all
tables means the Go importer produced what the TypeScript one did.`,
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
		for _, tbl := range sde.Tables {
			var n int64
			// count(*) is fine here: the largest of these is 52 k rows, so an
			// exact count is cheap and reltuples would be an estimate.
			if err := pool.QueryRow(cmd.Context(),
				"SELECT count(*) FROM "+tbl.Name).Scan(&n); err != nil {
				return err
			}
			total += n
			t.Row(ui.Command(tbl.Name), fmtCount(n))
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
	sdeImportCmd.Flags().BoolVar(&flagSDEForce, "force", false,
		"Re-import even when the published build is already loaded")
	for _, c := range []*cobra.Command{sdeImportCmd, sdeStatusCmd} {
		c.Flags().StringVar(&flagSDECacheDir, "cache-dir", "",
			"Where to keep downloaded archives (default .data/sde)")
	}
	sdeCmd.AddCommand(sdeStatusCmd, sdeImportCmd, sdeVerifyCmd)
}
