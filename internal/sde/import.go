package sde

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ImportOptions controls a complete or selected-table import.
type ImportOptions struct {
	CacheDir     string
	UserAgent    string
	Progress     func(string)
	SkipExternal bool
	Force        bool
	Only         []Table
}

// ImportResult summarises a complete SDE import. The detailed per-source
// results are retained so both the CLI and scheduled worker can report useful
// progress without duplicating the orchestration.
type ImportResult struct {
	Snapshot
	PreviousRelease string
	Skipped         bool
	Read            int64
	Written         int64
	Pruned          int64
	Duration        time.Duration
	Elapsed         string

	Tables          []LoadResult
	Celestials      LoadResult
	SystemJumps     LoadResult
	InvFlags        int64
	InvFlagsElapsed string
	Dust514         SeedResult
	External        []ExternalResult
	Derived         []DeriveResult
}

// ImportLatest coordinates the CLI and scheduled imports. The release and
// versions are recorded only after a full import has completed.
func ImportLatest(ctx context.Context, pool *pgxpool.Pool, opts ImportOptions) (ImportResult, error) {
	// Hold the lock on a dedicated session. The import itself needs the pool's
	// connections for staging tables and the lookups that populate them.
	lock, err := pgx.ConnectConfig(ctx, pool.Config().ConnConfig)
	if err != nil {
		return ImportResult{}, fmt.Errorf("connect for SDE import lock: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = lock.Close(closeCtx)
	}()
	var acquired bool
	if err := lock.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended('shrike.sde.import', 0))`).Scan(&acquired); err != nil {
		return ImportResult{}, fmt.Errorf("acquire SDE import lock: %w", err)
	}
	if !acquired {
		return ImportResult{}, fmt.Errorf("another SDE import is in progress")
	}

	manifest, err := FetchManifest(ctx, opts.UserAgent)
	if err != nil {
		return ImportResult{}, err
	}
	loaded, err := LoadedSnapshot(ctx, pool)
	if err != nil {
		return ImportResult{}, err
	}
	full := len(opts.Only) == 0
	tables := opts.Only
	if full {
		tables = AllTables()
	}
	members := importMembers(tables, full)
	next := snapshotFor(manifest, members)
	if loaded.Release > next.Release {
		return ImportResult{}, fmt.Errorf("CCP SDE release %s is older than the loaded release %s", next.Release, loaded.Release)
	}
	if full && !opts.Force && loaded.hasTables(next.TableVersions) {
		result := ImportResult{Snapshot: next, PreviousRelease: loaded.Release, Skipped: true}
		if loaded.Release != next.Release {
			return result, recordSnapshot(ctx, pool, next)
		}
		return result, nil
	}

	src, err := Open(ctx, opts.CacheDir, manifest, members, opts.UserAgent, opts.Progress)
	if err != nil {
		return ImportResult{}, err
	}
	next = snapshotFor(src.Manifest, members)

	// Each table commits independently. Invalidate the version set before
	// changing it so a failed attempt cannot make a later run skip the import.
	if len(loaded.TableVersions) != 0 && (full || tablesChanged(loaded, next)) {
		if err := clearTableVersions(ctx, pool); err != nil {
			return ImportResult{}, fmt.Errorf("clear SDE table versions: %w", err)
		}
	}
	result, err := importSource(ctx, pool, src, next, tables, full, opts)
	result.PreviousRelease = loaded.Release
	return result, err
}

func tablesChanged(loaded, next Snapshot) bool {
	for name, version := range next.TableVersions {
		if loaded.TableVersions[name] != version {
			return true
		}
	}
	return false
}

func importMembers(tables []Table, full bool) []string {
	var members []string
	seen := make(map[string]bool)
	add := func(member string) {
		if !seen[member] {
			seen[member] = true
			members = append(members, member)
		}
	}
	for _, table := range tables {
		add(table.Member)
	}
	if full {
		for _, table := range celestialTables {
			add(table.member)
		}
	}
	return members
}

func importSource(ctx context.Context, pool *pgxpool.Pool, src *Source, snapshot Snapshot, tables []Table, full bool, opts ImportOptions) (ImportResult, error) {
	start := time.Now()
	res := ImportResult{Snapshot: snapshot}
	progress := func(message string) {
		if opts.Progress != nil {
			opts.Progress(message)
		}
	}

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

	var err error
	if full {
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
		res.InvFlags, res.InvFlagsElapsed, err = SeedInvFlags(ctx, pool)
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
	}

	progress("deriving denormalised SDE fields")
	res.Derived, err = Derive(ctx, pool)
	if err != nil {
		return res, fmt.Errorf("derive: %w", err)
	}

	if full {
		stationNames, err := DeriveOne(ctx, pool, StationNameDerivation)
		if err != nil {
			return res, fmt.Errorf("derive station names: %w", err)
		}
		res.Derived = append(res.Derived, stationNames)
		if err := recordSnapshot(ctx, pool, snapshot); err != nil {
			return res, err
		}
	}

	res.Duration = time.Since(start)
	res.Elapsed = res.Duration.Round(time.Millisecond).String()
	return res, nil
}
