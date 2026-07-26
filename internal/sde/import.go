package sde

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ImportOptions controls a complete published-build import.
type ImportOptions struct {
	CacheDir     string
	UserAgent    string
	Progress     func(string)
	SkipExternal bool
}

// ImportResult summarises a complete SDE import. The detailed per-source
// results are retained so both the CLI and scheduled worker can report useful
// progress without duplicating the orchestration.
type ImportResult struct {
	BuildNumber int64
	ReleaseDate string
	Read        int64
	Written     int64
	Pruned      int64
	Duration    time.Duration
	Elapsed     string

	Tables      []LoadResult
	Celestials  LoadResult
	SystemJumps LoadResult
	InvFlags    int64
	Dust514     SeedResult
	External    []ExternalResult
	Derived     []DeriveResult
}

// ImportBuild performs the complete import for a published SDE build.
//
// RecordBuild deliberately runs last. Every operation before it is idempotent,
// so an interrupted import can safely converge on its next run, while the build
// marker never claims a partial database is current.
func ImportBuild(
	ctx context.Context,
	pool *pgxpool.Pool,
	manifest *Manifest,
	opts ImportOptions,
) (ImportResult, error) {
	start := time.Now()
	res := ImportResult{
		BuildNumber: manifest.BuildNumber,
		ReleaseDate: manifest.ReleaseDate,
	}

	progress := func(message string) {
		if opts.Progress != nil {
			opts.Progress(message)
		}
	}

	src, err := Open(ctx, opts.CacheDir, manifest, opts.UserAgent, opts.Progress)
	if err != nil {
		return res, err
	}
	defer src.Close() //nolint:errcheck // the archive has already been consumed

	tables := append(append([]Table{}, Tables...), NestedTables...)
	res.Tables = make([]LoadResult, 0, len(tables))
	for _, table := range tables {
		progress("importing " + table.Name)
		loaded, err := Load(ctx, pool, table, src)
		if err != nil {
			return res, fmt.Errorf("import %s: %w", table.Name, err)
		}
		res.Tables = append(res.Tables, loaded)
		res.Read += loaded.Read
		res.Written += loaded.Written
		res.Pruned += loaded.Pruned
	}

	progress("importing celestials")
	res.Celestials, err = ImportCelestials(ctx, pool, src)
	if err != nil {
		return res, fmt.Errorf("import celestials: %w", err)
	}
	res.Pruned += res.Celestials.Pruned

	progress("importing solar system jumps")
	res.SystemJumps, err = ImportSystemJumps(ctx, pool, src)
	if err != nil {
		return res, fmt.Errorf("import system jumps: %w", err)
	}
	res.Pruned += res.SystemJumps.Pruned

	progress("seeding inventory flags")
	res.InvFlags, _, err = SeedInvFlags(ctx, pool)
	if err != nil {
		return res, fmt.Errorf("seed inv_flags: %w", err)
	}

	progress("seeding Dust 514 data")
	res.Dust514, err = SeedDust514(ctx, pool)
	if err != nil {
		return res, fmt.Errorf("seed dust514: %w", err)
	}

	if !opts.SkipExternal {
		progress("importing insurance prices")
		insurance, err := ImportInsurancePrices(ctx, pool, opts.UserAgent)
		if err != nil {
			return res, err
		}
		res.External = append(res.External, insurance)

		progress("importing public structures")
		structures, err := ImportStructures(ctx, pool, opts.UserAgent)
		if err != nil {
			return res, err
		}
		res.External = append(res.External, structures)

		// Generated values run first so the hand-maintained list wins for hulls
		// present in both sources. This is the TypeScript import order.
		progress("generating supercapital prices")
		supercaps, err := GenerateSupercapPrices(ctx, pool)
		if err != nil {
			return res, err
		}
		res.External = append(res.External, supercaps)

		progress("seeding manual custom prices")
		manual, err := SeedManualCustomPrices(ctx, pool)
		if err != nil {
			return res, err
		}
		res.External = append(res.External, manual)
	}

	progress("deriving denormalised SDE fields")
	res.Derived, err = Derive(ctx, pool)
	if err != nil {
		return res, fmt.Errorf("derive: %w", err)
	}

	stationNames, err := DeriveOne(ctx, pool, StationNameDerivation)
	if err != nil {
		return res, fmt.Errorf("derive station names: %w", err)
	}
	res.Derived = append(res.Derived, stationNames)

	if err := RecordBuild(ctx, pool, manifest.BuildNumber, manifest.ReleaseDate); err != nil {
		return res, err
	}

	res.Duration = time.Since(start)
	res.Elapsed = res.Duration.Round(time.Millisecond).String()
	return res, nil
}
