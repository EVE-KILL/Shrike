package cli

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/db"
	"github.com/eve-kill/shrike/internal/dogmadata"
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
	flagDogmaOutput     string
)

// cacheDir defaults to the application's writable runtime state.
func cacheDir() string {
	if flagSDECacheDir != "" {
		return flagSDECacheDir
	}
	dataDir := ""
	if cfg != nil {
		dataDir = cfg.DataDir
	}
	return sde.CacheDir(dataDir)
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
	Short: "Compare the loaded SDE release against the published one",
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

		loaded, err := sde.LoadedSnapshot(cmd.Context(), pool)
		if err != nil {
			return err
		}

		current := loaded.Current(manifest)

		if ui.JSONMode {
			return ui.JSON(map[string]any{
				"published":         manifest.BuildNumber,
				"published_date":    manifest.ReleaseDate.Format(time.RFC3339Nano),
				"published_release": manifest.Release,
				"loaded":            loaded.BuildNumber,
				"loaded_date":       loaded.ReleaseDate,
				"loaded_release":    loaded.Release,
				"up_to_date":        current,
				"never_imported":    loaded.BuildNumber == 0,
			})
		}

		ui.Section("SDE")
		ui.KV("Published", fmt.Sprintf("%s (build %d)", manifest.Release, manifest.BuildNumber))
		if loaded.BuildNumber == 0 {
			ui.KV("Loaded", ui.Dim("never imported"))
		} else {
			release := loaded.Release
			if release == "" {
				release = loaded.ReleaseDate
			}
			ui.KV("Loaded", fmt.Sprintf("%s (build %d)", release, loaded.BuildNumber))
		}
		ui.Newline()
		if current {
			ui.Success("Up to date.")
		} else {
			ui.Warn("The loaded SDE needs an import. Run %s", "shrike sde:import")
		}
		ui.Newline()
		return nil
	},
}

var sdeImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import the Static Data Export into Postgres",
	Long: `Downloads the published SDE tables over HTTP and loads them into Postgres.

Tables are cached by version under DATA_DIR/sde. Imports stage rows with COPY
and merge them with ON CONFLICT DO UPDATE, so retries converge.

Skips importing when the published table versions are already loaded, unless --force.

--only restricts the import to selected destination tables or SDE table names:

    shrike sde:import --only inv_types --only regions`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		var only []sde.Table
		for _, name := range flagSDEOnly {
			table := sde.TableByName(name)
			if table == nil {
				return fmt.Errorf("unknown table or member %q", name)
			}
			only = append(only, *table)
		}

		pool, err := db.New(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		start := time.Now()
		res, err := sde.ImportLatest(cmd.Context(), pool, sde.ImportOptions{
			CacheDir: cacheDir(), UserAgent: userAgent(), Force: flagSDEForce,
			SkipExternal: flagSDESkipExternal, Only: only,
			Progress: func(message string) {
				ui.Printf("  %s %s\n", ui.Dim("·"), message)
			},
		})
		if len(res.Tables) > 0 {
			renderSDEImport(res)
		}
		if err != nil {
			return err
		}
		if res.Skipped {
			ui.Success("SDE tables in release %s are already loaded. Use --force to re-import.", res.Release)
			ui.Newline()
			return nil
		}

		ui.Newline()
		ui.KV("Release", res.Release)
		ui.KV("Build", strconv.FormatInt(res.BuildNumber, 10))
		ui.KV("Rows read", fmtCount(res.Read))
		ui.KV("Rows written", fmtCount(res.Written))
		ui.KV("Total", time.Since(start).Round(time.Millisecond).String())
		ui.Newline()
		if len(only) > 0 {
			ui.Warn("Partial import (--only): the loaded release was not stamped.")
			ui.Newline()
		}
		return nil
	},
}

func renderSDEImport(res sde.ImportResult) {
	ui.Section("Imported tables")
	t := ui.NewTable("TABLE", "SOURCE", "READ", "WRITTEN", "SKIPPED", "PRUNED", "TIME")
	for _, row := range res.Tables {
		t.Row(ui.Command(row.Table), row.Member, fmtCount(row.Read), fmtCount(row.Written),
			fmtCount(row.Skipped), fmtCount(row.Pruned), row.Elapsed)
	}
	fmt.Println(t.Render())

	if res.Celestials.Table != "" {
		ui.Section("Map and bundled data")
		t := ui.NewTable("TABLE", "WRITTEN", "PRUNED", "TIME")
		t.Row(ui.Command(res.Celestials.Table), fmtCount(res.Celestials.Written), fmtCount(res.Celestials.Pruned), res.Celestials.Elapsed)
		if res.SystemJumps.Table != "" {
			t.Row(ui.Command(res.SystemJumps.Table), fmtCount(res.SystemJumps.Written), fmtCount(res.SystemJumps.Pruned), res.SystemJumps.Elapsed)
		}
		if res.InvFlagsElapsed != "" {
			t.Row(ui.Command("inv_flags"), fmtCount(res.InvFlags), "0", res.InvFlagsElapsed)
		}
		fmt.Println(t.Render())
	}

	if res.Dust514.Elapsed != "" {
		ui.Section("Bundled Dust 514 data")
		t := ui.NewTable("CATEGORIES", "GROUPS", "MARKET GROUPS", "TYPES", "TIME")
		t.Row(fmtCount(res.Dust514.Categories), fmtCount(res.Dust514.Groups),
			fmtCount(res.Dust514.MarketGroups), fmtCount(res.Dust514.Types), res.Dust514.Elapsed)
		fmt.Println(t.Render())
	}

	if len(res.External) > 0 {
		ui.Section("External sources")
		t := ui.NewTable("SOURCE", "ROWS", "TIME")
		for _, row := range res.External {
			t.Row(ui.Command(row.Name), fmtCount(row.Rows), row.Elapsed)
		}
		fmt.Println(t.Render())
	}

	if len(res.Derived) > 0 {
		ui.Section("Derived")
		t := ui.NewTable("DERIVATION", "ROWS", "TIME")
		for _, row := range res.Derived {
			t.Row(row.Name, fmtCount(row.Rows), row.Elapsed)
		}
		fmt.Println(t.Render())
	}
}

var sdeVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Compare local SDE row counts against a reference database",
	Long: `Counts rows in every SDE-owned and derived table.

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

var sdeBuildDogmaCmd = &cobra.Command{
	Use:   "build-dogma",
	Short: "Build EVEShipFit-compatible Dogma data from the current CCP SDE",
	Long: `Generates the FlatBuffer bundle consumed by both the browser fitting editor
and the native server fitting worker. The converter is implemented in Go, while
the calculation semantics remain pinned to EVEShipFit/sde-patched's upstream YAML
patch set. Output is validated before the existing bundle is replaced.`,
	Annotations: map[string]string{skipConfigAnnotation: "true"},
	RunE: func(cmd *cobra.Command, _ []string) error {
		start := time.Now()
		manifest, err := sde.FetchManifest(cmd.Context(), userAgent())
		if err != nil {
			return err
		}
		source, err := sde.Open(cmd.Context(), cacheDir(), manifest, dogmadata.InputTables(), userAgent(), func(message string) {
			ui.Printf("  %s %s\n", ui.Dim("·"), message)
		})
		if err != nil {
			return err
		}
		manifest = source.Manifest

		patches, err := dogmadata.FetchPatches(cmd.Context(), filepath.Join(cacheDir(), "dogma"), userAgent())
		if err != nil {
			return err
		}
		result, err := dogmadata.Build(cmd.Context(), source, patches, flagDogmaOutput)
		if err != nil {
			return err
		}
		outputPath, err := filepath.Abs(flagDogmaOutput)
		if err != nil {
			return err
		}
		defaultPath, err := filepath.Abs(filepath.Join("frontend", "dogma", "dist", "upstream", "data", "dist"))
		if err != nil {
			return err
		}
		if outputPath == defaultPath {
			if err := updateDogmaMetadata(manifest.BuildNumber, manifest.Release); err != nil {
				return err
			}
		}

		ui.Section("Dogma data")
		ui.KV("SDE release", manifest.Release)
		ui.KV("SDE build", strconv.FormatInt(manifest.BuildNumber, 10))
		ui.KV("Converter", dogmadata.ConverterCommit)
		ui.KV("Types", fmtCount(int64(result.Types)))
		ui.KV("Attributes", fmt.Sprintf("%s (%d synthetic)", fmtCount(int64(result.Attributes)), result.SyntheticAttributes))
		ui.KV("Effects", fmt.Sprintf("%s (%d synthetic)", fmtCount(int64(result.Effects)), result.SyntheticEffects))
		ui.KV("Dogma types", fmtCount(int64(result.TypeDogma)))
		ui.KV("Output", flagDogmaOutput)
		ui.KV("Total", time.Since(start).Round(time.Millisecond).String())
		ui.Newline()
		return nil
	},
}

func updateDogmaMetadata(build int64, release string) error {
	path := filepath.Join("frontend", "dogma", "dist", "upstream", "VERSION.json")
	// #nosec G304 -- this is the fixed repository metadata path for an operator-only generator.
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Dogma version: %w", err)
	}
	var version map[string]any
	if err := json.Unmarshal(raw, &version); err != nil {
		return fmt.Errorf("parse Dogma version: %w", err)
	}
	version["sde_build"] = build
	version["sde_release"] = release
	version["data_major"] = dogmadata.DataMajorVersion
	version["data"] = fmt.Sprintf("%d.%d.0", dogmadata.DataMajorVersion, build)
	version["data_converter_commit"] = dogmadata.ConverterCommit
	version["generated_at"] = time.Now().UTC().Format(time.RFC3339)
	encoded, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	temporary := path + ".new"
	if err := os.WriteFile(temporary, encoded, 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}

	goPath := filepath.Join("internal", "fitting", "dogma_version_generated.go")
	source := fmt.Sprintf("// Code generated by `shrike sde:build-dogma`; DO NOT EDIT.\n\npackage fitting\n\nconst DogmaSDEVersion = %q\n", fmt.Sprintf("%d@%s", build, release))
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return err
	}
	goTemporary := goPath + ".new"
	if err := os.WriteFile(goTemporary, formatted, 0o600); err != nil {
		return err
	}
	return os.Rename(goTemporary, goPath)
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
		"Restrict to named destination tables or SDE tables (repeatable)")
	sdeImportCmd.Flags().BoolVar(&flagSDESkipExternal, "skip-external", false,
		"Skip the EVE Ref feeds (insurance prices, structures)")
	sdeImportCmd.Flags().BoolVar(&flagSDEForce, "force", false,
		"Re-import even when the published table versions are already loaded")
	for _, c := range []*cobra.Command{sdeImportCmd, sdeStatusCmd} {
		c.Flags().StringVar(&flagSDECacheDir, "cache-dir", "",
			"Where to keep downloaded tables (default DATA_DIR/sde)")
	}
	sdeBuildDogmaCmd.Flags().StringVar(&flagDogmaOutput, "output",
		filepath.Join("frontend", "dogma", "dist", "upstream", "data", "dist"),
		"Destination directory for generated FlatBuffer files")
	sdeBuildDogmaCmd.Flags().StringVar(&flagSDECacheDir, "cache-dir", "",
		"Where to keep downloaded tables (default DATA_DIR/sde)")
	sdeCmd.AddCommand(sdeStatusCmd, sdeImportCmd, sdeVerifyCmd, sdeBuildDogmaCmd)
}
