package everef

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/configstore"
	"github.com/eve-kill/shrike/internal/pgbulk"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Who holds which system, published three ways:
//
//	sovereignty-map-latest.json                current snapshot
//	history/<year>/<date>/*.json.bz2           one snapshot per day, 2022-12-16 on
//	history/sovereignty-map-<year>.tar.bz2     hourly snapshots, 2017-2022
//
// Two tables come out of it. `sovereignty` is current state, one row per
// system; `sovereignty_history` is an append-only log of every change. The log
// is the point — it is what makes "when did this alliance take this system"
// answerable — and it only stays meaningful if a row is appended when ownership
// actually changes and not otherwise.

const sovereigntyPath = "/sovereignty-map"

// The yearly archives EVE Ref publishes. 2020 is genuinely absent.
var sovereigntyYears = []int{2017, 2018, 2019, 2021, 2022}

// sovereigntyDailyStart is the first day published as an individual snapshot.
const sovereigntyDailyStart = "2022-12-16"

type sovEntry struct {
	SystemID      int32 `json:"system_id"`
	AllianceID    int32 `json:"alliance_id"`
	CorporationID int32 `json:"corporation_id"`
	FactionID     int32 `json:"faction_id"`
}

// owner is who holds a system. Zero means nobody, consistently with the rest of
// the codebase.
type owner struct {
	alliance    int32
	corporation int32
	faction     int32
}

func (o owner) held() bool { return o.alliance != 0 || o.corporation != 0 || o.faction != 0 }

// sovState is current ownership, held in memory across a run.
//
// A backfill replays years of snapshots in order, and each one is a diff
// against the last. Re-reading the table between snapshots would be thousands
// of round trips for data the importer just wrote.
type sovState struct {
	pool  *pgxpool.Pool
	owned map[int32]owner
}

func loadSovState(ctx context.Context, pool *pgxpool.Pool) (*sovState, error) {
	s := &sovState{pool: pool, owned: make(map[int32]owner, 9000)}

	rows, err := pool.Query(ctx, `
        SELECT system_id, coalesce(alliance_id, 0), coalesce(corporation_id, 0), coalesce(faction_id, 0)
        FROM sovereignty`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int32
		var o owner
		if err := rows.Scan(&id, &o.alliance, &o.corporation, &o.faction); err != nil {
			return nil, err
		}
		s.owned[id] = o
	}
	return s, rows.Err()
}

// apply writes one snapshot, recording only what changed.
func (s *sovState) apply(ctx context.Context, entries []sovEntry, at time.Time) (Result, error) {
	res := Result{Name: at.UTC().Format(dateLayout), Seen: int64(len(entries))}

	type change struct {
		systemID int32
		o        owner
	}
	var changed []change
	var history []change

	for _, e := range entries {
		if e.SystemID == 0 {
			continue
		}
		o := owner{alliance: e.AllianceID, corporation: e.CorporationID, faction: e.FactionID}

		current, known := s.owned[e.SystemID]
		switch {
		case !known:
			changed = append(changed, change{e.SystemID, o})
			// A system first seen unowned is not an event worth logging — it
			// is the absence of one.
			if o.held() {
				history = append(history, change{e.SystemID, o})
			}
		case current != o:
			changed = append(changed, change{e.SystemID, o})
			history = append(history, change{e.SystemID, o})
		default:
			continue
		}
		s.owned[e.SystemID] = o
	}

	if len(changed) == 0 {
		return res, nil
	}

	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return res, err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	sovColumns := []string{"system_id", "alliance_id", "corporation_id", "faction_id", "date_added", "updated_at"}
	const sovStaging = "everef_staging_sovereignty"
	if err := pgbulk.StagingTx(ctx, tx, sovStaging, "sovereignty"); err != nil {
		return res, err
	}

	w := pgbulk.NewCopier(ctx, tx, sovStaging, sovColumns)
	for _, c := range changed {
		if err := w.Add([]any{
			c.systemID, nullID(c.o.alliance), nullID(c.o.corporation), nullID(c.o.faction), at, at,
		}); err != nil {
			return res, err
		}
	}
	if err := w.Flush(); err != nil {
		return res, err
	}

	// DoUpdate, and this is the whole reason the importer exists in this shape.
	// The TypeScript cron writes `SET alliance_id = sovereignty.alliance_id`,
	// which assigns each column to itself — the row is touched and nothing
	// changes. Production shows the consequence: sovereignty.updated_at has not
	// moved since 2026-03-22, while sovereignty_history keeps growing, because
	// every run re-detects the same differences against a current-state table
	// that never advances. Over one week that produced 19,132 history rows
	// describing 694 distinct ownership states.
	if _, err := tx.Exec(ctx, pgbulk.MergeSQL("sovereignty", sovStaging, sovColumns,
		[]string{"system_id"}, pgbulk.DoUpdate)); err != nil {
		return res, fmt.Errorf("merge sovereignty: %w", err)
	}

	if len(history) > 0 {
		histColumns := []string{"system_id", "alliance_id", "corporation_id", "faction_id", "date_added"}
		const histStaging = "everef_staging_sovereignty_history"
		if err := pgbulk.StagingTx(ctx, tx, histStaging, "sovereignty_history"); err != nil {
			return res, err
		}

		hw := pgbulk.NewCopier(ctx, tx, histStaging, histColumns)
		for _, c := range history {
			if err := hw.Add([]any{
				c.systemID, nullID(c.o.alliance), nullID(c.o.corporation), nullID(c.o.faction), at,
			}); err != nil {
				return res, err
			}
		}
		if err := hw.Flush(); err != nil {
			return res, err
		}
		// A plain append: the id column is a sequence and there is no key to
		// conflict on, so this is an INSERT ... SELECT with no ON CONFLICT.
		if _, err := tx.Exec(ctx, fmt.Sprintf(
			`INSERT INTO public.sovereignty_history (system_id, alliance_id, corporation_id, faction_id, date_added)
             SELECT system_id, alliance_id, corporation_id, faction_id, date_added FROM %s`,
			histStaging)); err != nil {
			return res, fmt.Errorf("append sovereignty_history: %w", err)
		}
		res.Related = hw.Written()
	}

	if err := tx.Commit(ctx); err != nil {
		return res, err
	}

	res.Rows = w.Written()
	return res, nil
}

// ImportSovereigntyLatest applies the current published snapshot.
func ImportSovereigntyLatest(ctx context.Context, pool *pgxpool.Pool, client *Client) (Result, error) {
	start := time.Now()

	var entries []sovEntry
	if err := client.JSON(ctx, client.url(sovereigntyPath+"/sovereignty-map-latest.json"), &entries); err != nil {
		return Result{Name: "sovereignty"}, err
	}

	state, err := loadSovState(ctx, pool)
	if err != nil {
		return Result{Name: "sovereignty"}, err
	}

	now := time.Now().UTC()
	res, err := state.apply(ctx, entries, now)
	if err != nil {
		return res, err
	}
	res.Name = "sovereignty (latest)"
	res.Seen = int64(len(entries))
	res.Elapsed = time.Since(start).Round(time.Millisecond).String()

	return res, configstore.Set(ctx, pool, configstore.KeySovereigntyDate, now.Format(dateLayout))
}

// snapshotDate pulls the date out of a filename such as
// sovereigntymap_2017-12-27T02:00:02.json.
var snapshotDate = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)

// ImportSovereigntyRange replays every snapshot between two dates.
//
// The yearly archives carry hourly snapshots; only the first of each day is
// used, which is what the TypeScript does and is enough resolution for a
// history that is read by the day.
func ImportSovereigntyRange(ctx context.Context, pool *pgxpool.Pool, client *Client, from, to string, progress func(Result)) (Result, error) {
	start := time.Now()
	total := Result{Name: "sovereignty"}

	state, err := loadSovState(ctx, pool)
	if err != nil {
		return total, err
	}

	fromYear, err := strconv.Atoi(from[:4])
	if err != nil {
		return total, fmt.Errorf("invalid start date %q", from)
	}
	toYear, err := strconv.Atoi(to[:4])
	if err != nil {
		return total, fmt.Errorf("invalid end date %q", to)
	}

	// Phase one: the yearly archives.
	for _, year := range sovereigntyYears {
		if year < fromYear || year > toYear {
			continue
		}
		url := client.url(fmt.Sprintf("%s/history/sovereignty-map-%d.tar.bz2", sovereigntyPath, year))

		// One snapshot per day, the first seen. Held in memory because a tar
		// cannot be read out of order and the days have to be applied in
		// sequence for the change detection to mean anything.
		daily := map[string][]sovEntry{}
		err := client.WalkArchive(ctx, url, func(name string, data []byte) error {
			m := snapshotDate.FindStringSubmatch(name)
			if m == nil {
				return nil
			}
			date := m[1]
			if date < from || date > to {
				return nil
			}
			if _, seen := daily[date]; seen {
				return nil
			}
			var entries []sovEntry
			if !decodeMember(data, &entries) || len(entries) == 0 {
				return nil
			}
			daily[date] = entries
			return nil
		})
		if errors.Is(err, ErrNotPublished) {
			continue
		}
		if err != nil {
			return total, err
		}

		for _, date := range sortedKeys(daily) {
			at, _ := time.Parse(dateLayout, date)
			// Noon UTC, matching the TypeScript: the archives hold hourly
			// snapshots and the stored timestamp stands for the day, not the
			// moment the snapshot was taken.
			r, err := state.apply(ctx, daily[date], at.Add(12*time.Hour))
			if err != nil {
				return total, err
			}
			total.Rows += r.Rows
			total.Related += r.Related
			if progress != nil {
				progress(r)
			}
			if err := configstore.Set(ctx, pool, configstore.KeySovereigntyDate, date); err != nil {
				return total, err
			}
		}
	}

	// Phase two: the daily snapshots.
	dailyFrom := from
	if dailyFrom < sovereigntyDailyStart {
		dailyFrom = sovereigntyDailyStart
	}
	if dailyFrom > to {
		total.Elapsed = time.Since(start).Round(time.Millisecond).String()
		return total, nil
	}

	dates, err := discoverSovereigntyDays(ctx, client, dailyFrom, to)
	if err != nil {
		return total, err
	}

	for _, date := range dates {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		entries, err := fetchSovereigntyDay(ctx, client, date)
		if err != nil {
			if errors.Is(err, ErrNotPublished) {
				total.Failed++
				continue
			}
			return total, err
		}
		at, _ := time.Parse(dateLayout, date)
		r, err := state.apply(ctx, entries, at.Add(12*time.Hour))
		if err != nil {
			return total, err
		}
		total.Rows += r.Rows
		total.Related += r.Related
		if progress != nil {
			progress(r)
		}
		if err := configstore.Set(ctx, pool, configstore.KeySovereigntyDate, date); err != nil {
			return total, err
		}
	}

	total.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return total, nil
}

var sovDayDir = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})/$`)

func discoverSovereigntyDays(ctx context.Context, client *Client, from, to string) ([]string, error) {
	fromYear, _ := strconv.Atoi(from[:4])
	toYear, _ := strconv.Atoi(to[:4])

	var out []string
	for year := fromYear; year <= toYear; year++ {
		dates, err := client.List(ctx, client.url(fmt.Sprintf("%s/history/%d/", sovereigntyPath, year)), sovDayDir)
		if err != nil {
			if errors.Is(err, ErrNotPublished) {
				continue
			}
			return nil, err
		}
		for _, d := range dates {
			if d >= from && d <= to {
				out = append(out, d)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

var sovDayFile = regexp.MustCompile(`(sovereignty-map-[^"/]+\.json\.bz2)$`)

func fetchSovereigntyDay(ctx context.Context, client *Client, date string) ([]sovEntry, error) {
	dir := client.url(fmt.Sprintf("%s/history/%s/%s/", sovereigntyPath, date[:4], date))
	files, err := client.List(ctx, dir, sovDayFile)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s: %w", dir, ErrNotPublished)
	}

	var entries []sovEntry
	if err := client.JSON(ctx, dir+files[0], &entries); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%s: %w", dir, ErrNotPublished)
	}
	return entries, nil
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// nullID turns the zero-means-absent convention into a real NULL.
func nullID(v int32) any {
	if v == 0 {
		return nil
	}
	return v
}
