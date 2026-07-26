package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/configstore"
	"github.com/eve-kill/shrike/internal/entities"
	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/stats"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/eve-kill/shrike/internal/wars"
	"github.com/eve-kill/shrike/internal/zkb"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/spf13/cobra"
)

// Repair commands.
//
// Each of these fixes something the running system got wrong or never had the
// chance to do. They are run by a person who has noticed a problem, which is
// why they preview by default and why the destructive ones want the scope named
// back to them before they touch anything.

var rebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "Recompute a derived table from its source",
	Long: `Replaces a derived table wholesale rather than incrementally.

These are the authorities the incremental paths can be checked against. Each
previews by default and needs the scope confirmed before it writes.`,
}

var catchupCmd = &cobra.Command{
	Use:   "catchup",
	Short: "Fill in what the live path missed",
	Long: `Recomputes recent aggregates the running system should have maintained.

For after an outage, a deploy gap, or a queue drained by hand. Each replaces
the range it covers rather than adding to it, so re-running is safe.`,
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Bring in data from outside sources",
}

// --- rebuild:war-interactions ---

var (
	flagRebuildWarID   int32
	flagRebuildAll     bool
	flagRebuildApply   bool
	flagRebuildConfirm string
)

// rebuildAllConfirmation is what --confirm must say for a full rebuild.
//
// Not the word "yes". A full rebuild truncates the table for every war there
// has ever been, and typing something specific is the difference between
// meaning it and having it in shell history.
const rebuildAllConfirmation = "REBUILD-ALL-WAR-INTERACTIONS"

var rebuildWarInteractionsCmd = &cobra.Command{
	Use:   "war-interactions",
	Short: "Preview or atomically rebuild war_interactions from killmails",
	Long: `Recomputes war_interactions from the killmails and attackers.

The incremental path adds one killmail at a time and is what normally keeps the
table current. This is the authority to check it against: it derives every row
from the killmails directly, so a table that has drifted can be replaced with
what the data actually says.

Read-only by default. With --apply it runs as one repeatable-read transaction
holding the exclusive war-interaction lock: it builds the replacement in a
temporary table, validates its shape, swaps the requested scope, and updates the
effect ledger. Live ingestion takes the same lock shared, so nothing can slip an
increment into a table being replaced.

Examples:
  shrike rebuild:war-interactions --war-id 755391
  shrike rebuild:war-interactions --war-id 755391 --apply --confirm 755391
  shrike rebuild:war-interactions --all --apply --confirm ` + rebuildAllConfirmation,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Exactly one scope. Defaulting either way would be wrong: silently
		// rebuilding one war when every war was meant leaves the problem, and
		// the reverse truncates the table.
		if flagRebuildAll == (flagRebuildWarID != 0) {
			return fmt.Errorf("choose exactly one scope: --war-id <id> or --all")
		}
		if flagRebuildWarID < 0 {
			return fmt.Errorf("invalid --war-id %d", flagRebuildWarID)
		}

		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		scope := fmt.Sprintf("war %d", flagRebuildWarID)
		if flagRebuildAll {
			scope = "all wars"
		}

		ui.Section("War interaction rebuild — " + scope)

		before, err := wars.Summarize(cmd.Context(), pool, flagRebuildWarID)
		if err != nil {
			return err
		}
		printWarSummary("Current", before)

		if !flagRebuildApply {
			ui.Newline()
			ui.Warn("Read-only preview. Pass --apply with the required --confirm to rebuild.")
			return nil
		}

		expected := fmt.Sprint(flagRebuildWarID)
		if flagRebuildAll {
			expected = rebuildAllConfirmation
		}
		if flagRebuildConfirm != expected {
			return fmt.Errorf("refusing to write: pass --confirm %s", expected)
		}

		ui.Newline()
		start := time.Now()
		res, err := wars.Rebuild(cmd.Context(), pool, flagRebuildWarID)
		if err != nil {
			return err
		}

		after, err := wars.Summarize(cmd.Context(), pool, flagRebuildWarID)
		if err != nil {
			return err
		}
		printWarSummary("Rebuilt", after)

		ui.Newline()
		ui.Success("Replaced %s rows across %s wars in %s",
			fmtCount(res.Rows), fmtCount(res.Wars), time.Since(start).Round(time.Millisecond))
		return nil
	},
}

func printWarSummary(label string, s wars.Summary) {
	ui.Newline()
	ui.KV(label+" killmails", fmtCount(s.Killmails))
	ui.KV(label+" killmail ISK", fmtISK(s.KillmailISK))
	ui.KV(label+" interaction rows", fmtCount(s.Rows))
	ui.KV(label+" wars", fmtCount(s.Wars))
	// The comparable pair: one combined corporation row per victim corporation
	// per war, so these should track the killmail figures above.
	ui.KV(label+" combined corp kills", fmtCount(s.CombinedCorpKills))
	ui.KV(label+" combined corp ISK", fmtISK(s.CombinedCorpISK))
}

// --- catchup:stats ---

var (
	flagCatchupFrom  string
	flagCatchupTo    string
	flagCatchupDays  int
	flagCatchupTable string
)

var catchupStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Re-aggregate daily stats for a recent date range",
	Long: `Deletes and recomputes the daily stats rows for a date range.

For when the live stats path has not been running and a few days are short.
Replaces rather than adds, so it is safe to re-run — a catchup that merely
added would double every day it had already covered.

Daily granularity only. The monthly and yearly rollups are derived from these
by the nightly pipeline, so they are left alone rather than half-rebuilt here.

The aggregation is the same accumulator the live path uses, so a caught-up day
holds exactly what it would have held had the workers never stopped.

Examples:
  shrike catchup:stats --days 1
  shrike catchup:stats --days 7
  shrike catchup:stats --from 2026-07-18 --to 2026-07-20`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		from, to, err := catchupRange()
		if err != nil {
			return err
		}

		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		wantStats, wantBreakdowns := true, true
		switch flagCatchupTable {
		case "":
		case "stats":
			wantBreakdowns = false
		case "breakdowns":
			wantStats = false
		default:
			return fmt.Errorf("invalid --table %q, want stats or breakdowns", flagCatchupTable)
		}

		days := int(to.Sub(from).Hours() / 24)
		ui.Section(fmt.Sprintf("Stats catchup — %d day(s), %s to %s",
			days, from.Format("2006-01-02"), to.AddDate(0, 0, -1).Format("2006-01-02")))

		start := time.Now()
		res, err := stats.CatchupTargets(cmd.Context(), pool, from, to, wantStats, wantBreakdowns)
		if err != nil {
			return err
		}

		ui.Newline()
		ui.KV("Days", fmt.Sprint(res.Days))
		ui.KV("Killmails", fmtCount(res.Killmails))
		ui.KV("Rows replaced", fmtCount(res.Deleted))
		ui.KV("Stats written", fmtCount(res.Stats))
		ui.KV("Breakdowns written", fmtCount(res.Breakdowns))
		ui.Newline()
		ui.Success("Catchup finished in %s", time.Since(start).Round(time.Millisecond))
		ui.Warn("Monthly and yearly rollups untouched — the nightly pipeline rebuilds those.")
		return nil
	},
}

// catchupRange resolves --from/--to/--days into a half-open UTC day range.
func catchupRange() (time.Time, time.Time, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	switch {
	case flagCatchupFrom != "":
		from, err := time.Parse("2006-01-02", flagCatchupFrom)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --from %q, want YYYY-MM-DD", flagCatchupFrom)
		}
		// Exclusive upper bound of tomorrow, so today is included by default.
		to := today.AddDate(0, 0, 1)
		if flagCatchupTo != "" {
			if to, err = time.Parse("2006-01-02", flagCatchupTo); err != nil {
				return time.Time{}, time.Time{}, fmt.Errorf("invalid --to %q, want YYYY-MM-DD", flagCatchupTo)
			}
		}
		if !to.After(from) {
			return time.Time{}, time.Time{}, fmt.Errorf("--to (%s) is not after --from (%s)",
				to.Format("2006-01-02"), from.Format("2006-01-02"))
		}
		return from, to, nil

	case flagCatchupDays > 0:
		return today.AddDate(0, 0, -(flagCatchupDays - 1)), today.AddDate(0, 0, 1), nil

	default:
		return time.Time{}, time.Time{}, fmt.Errorf("pass --from or --days")
	}
}

// --- backfill:points ---

var (
	flagPointsFromMonth string
	flagPointsToMonth   string
)

var backfillPointsCmd = &cobra.Command{
	Use:   "points",
	Short: "Score killmails that have no points",
	Long: `Computes points for killmails stored with points = 0.

Points are the difficulty score, computed at ingest. Killmails imported before
the scorer existed, or through a path that skipped it, carry zero — which reads
as "worthless kill" rather than "not scored".

	Only touches rows at zero, so a re-run is cheap and cannot rescore anything.
The score comes from the same function the live path uses, so a backfilled kill
gets exactly what ingest would have given it.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		from, to, err := inclusiveMonthRange(flagPointsFromMonth, flagPointsToMonth)
		if err != nil {
			return err
		}

		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		ui.Section("Backfill points")

		cache, err := eve.Load(cmd.Context(), pool)
		if err != nil {
			return fmt.Errorf("load SDE cache: %w", err)
		}

		start := time.Now()
		scored, err := scorePoints(cmd.Context(), pool, cache, from, to, 0)
		if err != nil {
			return err
		}

		ui.Newline()
		ui.Success("Scored %s killmails in %s",
			fmtCount(scored), time.Since(start).Round(time.Millisecond))
		return nil
	},
}

func inclusiveMonthRange(fromValue, toValue string) (time.Time, time.Time, error) {
	if fromValue == "" {
		fromValue = "2007-12"
	}
	if toValue == "" {
		toValue = time.Now().UTC().Format("2006-01")
	}
	from, err := time.Parse("2006-01", fromValue)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --from %q, want YYYY-MM", fromValue)
	}
	toMonth, err := time.Parse("2006-01", toValue)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --to %q, want YYYY-MM", toValue)
	}
	if toMonth.Before(from) {
		return time.Time{}, time.Time{}, fmt.Errorf("--to (%s) is before --from (%s)", toValue, fromValue)
	}
	return from, toMonth.AddDate(0, 1, 0), nil
}

// scorePoints walks the unscored killmails and writes their score.
//
// Paged by id rather than by OFFSET, and the cursor only moves forward, so an
// interrupted run resumes by simply being run again — the rows it already
// scored are no longer at zero and no longer selected.
func scorePoints(ctx context.Context, pool *pgxpool.Pool, cache *eve.Cache,
	from, to time.Time, limit int) (int64, error) {

	const chunk = 2000
	var cursor, total int64

	for {
		if limit > 0 && total >= int64(limit) {
			return total, nil
		}

		rows, err := pool.Query(ctx, `
            SELECT killmail_id, coalesce(victim_ship_type_id, 0)
            FROM killmails
            WHERE coalesce(points, 0) = 0
              AND killmail_time >= $1 AND killmail_time < $2
              AND killmail_id > $3
            ORDER BY killmail_id
            LIMIT $4`, from, to, cursor, chunk)
		if err != nil {
			return total, err
		}

		type target struct {
			id     int64
			shipID int32
		}
		var batch []target
		for rows.Next() {
			var t target
			if err := rows.Scan(&t.id, &t.shipID); err != nil {
				rows.Close()
				return total, err
			}
			batch = append(batch, t)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return total, err
		}
		if len(batch) == 0 {
			return total, nil
		}

		ids := make([]int64, len(batch))
		for i, t := range batch {
			ids[i] = t.id
		}

		items, err := loadPointsItems(ctx, pool, ids)
		if err != nil {
			return total, err
		}
		attackers, err := loadPointsAttackers(ctx, pool, ids)
		if err != nil {
			return total, err
		}

		scores := make([]int32, len(batch))
		for i, t := range batch {
			scores[i] = killmail.Points(cache, killmail.PointsInput{
				VictimShipTypeID: t.shipID,
				Items:            items[t.id],
				Attackers:        attackers[t.id],
			})
		}

		// One statement for the batch rather than one per killmail.
		if _, err := pool.Exec(ctx, `
            UPDATE killmails k SET points = v.points
            FROM unnest($1::bigint[], $2::int[]) AS v(killmail_id, points)
            WHERE k.killmail_id = v.killmail_id`, ids, scores); err != nil {
			return total, fmt.Errorf("write points: %w", err)
		}

		total += int64(len(batch))
		cursor = batch[len(batch)-1].id
	}
}

// loadPointsItems reads the same item set as the TypeScript command. The scorer
// itself filters to module-slot flags and module category.
func loadPointsItems(ctx context.Context, pool *pgxpool.Pool, ids []int64) (map[int64][]killmail.PointsItem, error) {
	rows, err := pool.Query(ctx, `
        SELECT killmail_id, type_id, flag_id,
               coalesce(quantity_dropped, 0), coalesce(quantity_destroyed, 0)
        FROM killmail_items
        WHERE killmail_id = ANY($1::bigint[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64][]killmail.PointsItem, len(ids))
	for rows.Next() {
		var id int64
		var it killmail.PointsItem
		if err := rows.Scan(&id, &it.TypeID, &it.Flag,
			&it.QuantityDropped, &it.QuantityDestroyed); err != nil {
			return nil, err
		}
		out[id] = append(out[id], it)
	}
	return out, rows.Err()
}

// loadPointsAttackers reads the attackers for a batch.
//
// A known and unfixable imprecision lives here. At ingest the scorer sees the
// raw ESI ship_type_id and skips an attacker who has none, but the stored
// column holds the *resolved* hull: when ESI omits the ship and the weapon is
// itself a ship type, the parser infers the hull from the weapon and stores
// that. Nothing records which of the two happened, so a rescore from stored
// rows counts a hull the original scoring skipped, inflating the average
// attacker size and lowering the score slightly.
//
// Measured at 14 killmails in 15,669 on the local corpus — every one of them
// with an attacker whose stored ship_type_id equals its weapon_type_id, which
// is the signature of the inference. Reproducing it faithfully is impossible
// without a column that says so, and guessing from ship == weapon would be
// wrong for the 4,221 mails where ESI genuinely reported both the same.
//
// It cannot affect a real run: this command only scores rows at zero, and a
// row at zero was never scored by ingest, so there is nothing to disagree
// with. It matters only if someone deliberately rescores scored killmails.
// The TypeScript backfill reads the same column and has the same behaviour.
func loadPointsAttackers(ctx context.Context, pool *pgxpool.Pool, ids []int64) (map[int64][]killmail.PointsAttacker, error) {
	rows, err := pool.Query(ctx, `
        SELECT killmail_id, coalesce(character_id, 0), coalesce(ship_type_id, 0)
        FROM killmail_attackers
        WHERE killmail_id = ANY($1::bigint[])
        ORDER BY killmail_id, attacker_index`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64][]killmail.PointsAttacker, len(ids))
	for rows.Next() {
		var id int64
		var a killmail.PointsAttacker
		if err := rows.Scan(&id, &a.CharacterID, &a.ShipTypeID); err != nil {
			return nil, err
		}
		out[id] = append(out[id], a)
	}
	return out, rows.Err()
}

// --- scan:character-trailing and scan:character-holes ---

var (
	flagScanFrom       int32
	flagScanTo         int32
	flagScanAhead      int
	flagScanMaxMisses  int
	flagScanKeepGoing  bool
	flagScanDryRun     bool
	flagScanMaxGap     int
	flagScanProbeBlock int
	flagScanSkip       int
	flagScanResume     bool
)

// scanHolesResumeKey remembers how far a gap scan got.
const scanHolesResumeKey = "scan:character-holes:last_id"

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Discover entities by probing the id space",
	Long: `Finds characters the killmails have never mentioned.

A killboard only learns about people who have shot something or been shot.
These walk the character id space directly to find the rest.

Both spend the shared ESI error budget on 404s, so both are bounded by a miss
count and neither is safe to leave running unattended.`,
}

var scanCharacterTrailingCmd = &cobra.Command{
	Use:   "character-trailing",
	Short: "Probe ids above the highest known character",
	Long: `Walks forward from the highest character id we know, looking for new ones.

CCP allocates character ids roughly in order, so the space just above the
highest we have seen is where new players appear. This finds them before they
show up on a killmail.

Stops after --max-misses cumulative misses. Cumulative rather than consecutive:
allocation is not dense, so a run of gaps inside live territory is normal, and
stopping at the first would give up immediately.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		client := esi.New(cfg)
		defer client.Close()

		start := flagScanFrom
		if start == 0 {
			if start, err = entities.HighestCharacterID(cmd.Context(), pool); err != nil {
				return err
			}
			if start == 0 {
				return fmt.Errorf("no characters are stored, so there is no id to scan from — pass --from")
			}
		}

		ui.Section("Trailing character scan")
		ui.KV("Starting after", fmt.Sprint(start))
		ui.KV("Probe budget", fmt.Sprintf("%d ids, %d misses", flagScanAhead, flagScanMaxMisses))
		if flagScanDryRun {
			ui.Warn("Dry run — ESI is still called, nothing is stored")
		}
		ui.Newline()

		prober := &entities.Prober{
			Pool: pool, ESI: client, DryRun: flagScanDryRun,
			OnProbe: func(p entities.Probe) {
				if p.Outcome == entities.ProbeHit {
					ui.Printf("  %s  %d → %s\n", ui.Primary("hit"), p.ID, p.Name)
				}
			},
		}
		if !flagScanDryRun {
			qc, err := queue.New(queue.Options{Pool: pool})
			if err != nil {
				return err
			}
			prober.OnCascade = scanCascadeDispatcher(qc)
		}

		res, err := prober.ScanTrailing(cmd.Context(), start,
			flagScanAhead, flagScanMaxMisses, flagScanKeepGoing)
		if err != nil {
			return err
		}

		ui.Newline()
		ui.KV("Probed", fmt.Sprint(res.Probed))
		ui.KV("Found", fmt.Sprint(res.Hits))
		ui.KV("Missed", fmt.Sprint(res.Misses))
		ui.KV("Stopped at", fmt.Sprintf("%d (%s)", res.LastID, res.Stopped))
		return nil
	},
}

var scanCharacterHolesCmd = &cobra.Command{
	Use:   "character-holes",
	Short: "Probe the gaps between known character ids",
	Long: `Walks the gaps between character ids we know and samples each one.

At each step it probes a small block: if anything is there it advances by that
block, and if the whole block missed it jumps ahead by --skip. The assumption is
that allocated ids cluster, so a hit means keep looking here and a miss means
look elsewhere.

This is sampling, not enumeration. It will not find every character in a gap and
is not meant to — the id space is far too large to walk exhaustively against a
shared error budget.

Gaps wider than --max-gap are skipped entirely. Those are CCP's unallocated
blocks, and scanning one would spend weeks of budget to find nobody.

Examples:
  shrike scan:character-holes --resume
  shrike scan:character-holes --from 90000000 --to 200000000
  shrike scan:character-holes --max-gap 50000 --skip 100 --dry-run`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		client := esi.New(cfg)
		defer client.Close()

		from := flagScanFrom
		switch {
		case from != 0:
			// An explicit --from wins over a saved position.
		case flagScanResume:
			saved, err := configstore.Get(cmd.Context(), pool, scanHolesResumeKey)
			if err != nil {
				return err
			}
			if saved != "" {
				var parsed int32
				if _, err := fmt.Sscanf(saved, "%d", &parsed); err == nil {
					from = parsed
				}
			}
		}

		ui.Section("Character hole scan")
		bound := "no upper bound"
		if flagScanTo != 0 {
			bound = fmt.Sprint(flagScanTo)
		}
		ui.KV("Range", fmt.Sprintf("%d … %s", from, bound))
		ui.KV("Probe block", fmt.Sprintf("%d, skipping %d on an empty block", flagScanProbeBlock, flagScanSkip))
		ui.KV("Max gap", fmtCount(int64(flagScanMaxGap)))
		ui.KV("Max misses", fmtCount(int64(flagScanMaxMisses)))
		if flagScanDryRun {
			ui.Warn("Dry run — no ESI calls or writes")
		}
		ui.Newline()

		prober := &entities.Prober{
			Pool: pool, ESI: client, DryRun: flagScanDryRun,
			OnProbe: func(p entities.Probe) {
				if p.Outcome == entities.ProbeHit {
					ui.Printf("  %s  %d → %s\n", ui.Primary("hit"), p.ID, p.Name)
				}
			},
		}
		if !flagScanDryRun {
			qc, err := queue.New(queue.Options{Pool: pool})
			if err != nil {
				return err
			}
			prober.OnCascade = scanCascadeDispatcher(qc)
		}

		res, err := prober.ScanHoles(cmd.Context(), entities.HoleOptions{
			From: from, To: flagScanTo,
			MaxGap:     flagScanMaxGap,
			ProbeBlock: flagScanProbeBlock,
			Skip:       flagScanSkip,
			MaxMisses:  flagScanMaxMisses,
		})
		if err != nil {
			return err
		}

		// Saved even on a partial run — the point of the cursor is that the next
		// run continues rather than re-probing what this one already covered.
		if res.LastID != 0 {
			if err := configstore.Set(cmd.Context(), pool,
				scanHolesResumeKey, fmt.Sprint(res.LastID)); err != nil {
				return err
			}
		}

		ui.Newline()
		ui.KV("Gaps scanned", fmtCount(int64(res.GapsScanned)))
		ui.KV("Gaps skipped", fmt.Sprintf("%s (wider than %s)",
			fmtCount(int64(res.GapsSkipped)), fmtCount(int64(flagScanMaxGap))))
		ui.KV("Probed", fmtCount(int64(res.Probed)))
		ui.KV("Found", fmtCount(int64(res.Hits)))
		ui.KV("Missed", fmtCount(int64(res.Misses)))
		ui.KV("Resume from", fmt.Sprint(res.LastID))
		if res.Stopped != "" {
			ui.Warn("Stopped early: %s", res.Stopped)
		}
		return nil
	},
}

func scanCascadeDispatcher(qc *queue.Client) func(context.Context, entities.Cascade) error {
	return func(ctx context.Context, cascade entities.Cascade) error {
		_, err := queue.DispatchCascade(ctx, qc, queue.Cascade{
			Characters:           cascade.Characters,
			Corporations:         cascade.Corporations,
			Alliances:            cascade.Alliances,
			CharacterHistories:   cascade.CharacterHistories,
			CorporationHistories: cascade.CorporationHistories,
		}, queue.RecentBackfill)
		return err
	}
}

// --- import:zkb_history ---

var (
	flagImportFrom string
	flagImportTo   string
)

// zkbHistoryCursorKey remembers the last day imported.
const zkbHistoryCursorKey = "import:zkb_history:last_date"

var importZkbHistoryCmd = &cobra.Command{
	Use:     "zkb_history",
	Aliases: []string{"zkb-history"},
	Short:   "Backfill killmails from the zKillboard history archive",
	Long: `Reads R2Z2's daily id+hash lists and queues whatever is missing.

Walks newest to oldest. The archive index says which days exist, so days before
it begins are never requested. Each day's ids are checked against what is
already stored and only the difference is queued, which makes a re-run over
covered ground nearly free.

The killmails themselves are fetched from ESI by the workers — the archive
carries only ids and hashes. A large backfill is therefore bounded by the ESI
budget rather than by this command.

Examples:
  shrike import:zkb_history
  shrike import:zkb_history --from 2024-06-30 --to 2024-01-01`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		pool, err := openPool(cmd)
		if err != nil {
			return err
		}
		defer pool.Close()

		client := zkb.New(userAgent())

		ui.Section("zKillboard history import")

		totals, err := client.Totals(cmd.Context())
		if err != nil {
			return fmt.Errorf("fetch the history index: %w", err)
		}
		if len(totals) == 0 {
			return fmt.Errorf("the history index is empty")
		}

		from := flagImportFrom
		if from == "" {
			from = time.Now().UTC().AddDate(0, 0, -1).Format("20060102")
		}
		days := selectHistoryDays(totals, from, flagImportTo)
		if len(days) == 0 {
			ui.Warn("No archived days fall in the requested range.")
			return nil
		}

		ui.KV("Archive", fmt.Sprintf("%d days available", len(totals)))
		ui.KV("Selected", fmt.Sprintf("%d days, %s back to %s",
			len(days), days[0], days[len(days)-1]))

		qc, err := queue.New(queue.Options{Pool: pool})
		if err != nil {
			return err
		}

		ui.Newline()
		var queued, skipped int64
		for i, day := range days {
			n, already, err := importHistoryDay(cmd.Context(), pool, qc, client, day)
			if err != nil {
				// One bad day does not end the run: the archive occasionally
				// serves a malformed file, and the days behind it are fine.
				ui.Warn("[%d/%d] %s — %v", i+1, len(days), day, err)
				if err := waitHistoryRequest(cmd.Context()); err != nil {
					return err
				}
				continue
			}
			queued += n
			skipped += already

			ui.Printf("  [%d/%d] %s — queued %s, already stored %s\n",
				i+1, len(days), day, fmtCount(n), fmtCount(already))

			// The cursor is written per day, so an interrupted import resumes
			// from the day it reached rather than the start.
			if err := configstore.Set(cmd.Context(), pool, zkbHistoryCursorKey, day); err != nil {
				return err
			}
			if err := waitHistoryRequest(cmd.Context()); err != nil {
				return err
			}
		}

		ui.Newline()
		ui.Success("Queued %s killmails, %s already stored", fmtCount(queued), fmtCount(skipped))
		return nil
	},
}

func waitHistoryRequest(ctx context.Context) error {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// selectHistoryDays picks and orders the days to import.
//
// Newest first: the recent past is what anyone is actually missing, and an
// import that runs out of time or budget should have covered that rather than
// 2011.
func selectHistoryDays(totals map[string]int64, from, to string) []string {
	from, to = compactDate(from), compactDate(to)

	var days []string
	for day := range totals {
		if from != "" && day > from {
			continue
		}
		if to != "" && day < to {
			continue
		}
		days = append(days, day)
	}

	// Descending.
	for i := 0; i < len(days); i++ {
		for j := i + 1; j < len(days); j++ {
			if days[j] > days[i] {
				days[i], days[j] = days[j], days[i]
			}
		}
	}
	return days
}

// compactDate accepts YYYY-MM-DD or YYYYMMDD and returns the archive's form.
func compactDate(s string) string { return strings.ReplaceAll(s, "-", "") }

// importHistoryDay queues one day's missing killmails.
func importHistoryDay(ctx context.Context, pool *pgxpool.Pool, qc *queue.Client,
	client *zkb.Client, day string) (int64, int64, error) {

	entries, err := client.History(ctx, day)
	if err != nil {
		return 0, 0, err
	}
	if len(entries) == 0 {
		return 0, 0, nil
	}

	ids := make([]int64, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}

	// Bounded by the day as well as by id. The id column is indexed with time,
	// so naming the day turns this from a scan of every id into one partition.
	stored := map[int64]bool{}
	rows, err := pool.Query(ctx, `
        SELECT killmail_id FROM killmails
        WHERE killmail_id = ANY($1::bigint[])
          AND killmail_time >= $2::date
          AND killmail_time < ($2::date + interval '1 day')`,
		ids, fmt.Sprintf("%s-%s-%s", day[0:4], day[4:6], day[6:8]))
	if err != nil {
		return 0, 0, err
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, 0, err
		}
		stored[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	var batch []river.JobArgs
	for id, hash := range entries {
		if stored[id] {
			continue
		}
		batch = append(batch, queue.KillmailArgs{KillmailID: id, KillmailHash: hash})
	}

	already := int64(len(stored))
	if len(batch) == 0 {
		return 0, already, nil
	}

	// The lowest priority there is. A backfill of old killmails must never
	// delay the live feed, however many days are being imported.
	n, err := queue.DispatchMany(ctx, qc, batch, queue.DormantBackfill)
	return int64(n), already, err
}

func init() {
	rebuildWarInteractionsCmd.Flags().Int32Var(&flagRebuildWarID, "war-id", 0, "Rebuild one war")
	rebuildWarInteractionsCmd.Flags().BoolVar(&flagRebuildAll, "all", false, "Rebuild every war")
	rebuildWarInteractionsCmd.Flags().BoolVar(&flagRebuildApply, "apply", false, "Apply the rebuild")
	rebuildWarInteractionsCmd.Flags().StringVar(&flagRebuildConfirm, "confirm", "", "Required with --apply")
	rebuildCmd.AddCommand(rebuildWarInteractionsCmd)

	catchupStatsCmd.Flags().StringVarP(&flagCatchupFrom, "from", "f", "", "Start date (YYYY-MM-DD, inclusive)")
	catchupStatsCmd.Flags().StringVarP(&flagCatchupTo, "to", "t", "", "End date (YYYY-MM-DD, exclusive)")
	catchupStatsCmd.Flags().IntVarP(&flagCatchupDays, "days", "d", 0, "Catch up the last N days including today")
	catchupStatsCmd.Flags().StringVar(&flagCatchupTable, "table", "", "Repair only one table: stats or breakdowns")
	catchupCmd.AddCommand(catchupStatsCmd)

	backfillPointsCmd.Flags().StringVarP(&flagPointsFromMonth, "from", "f", "2007-12", "Start month (YYYY-MM)")
	backfillPointsCmd.Flags().StringVarP(&flagPointsToMonth, "to", "t", "", "End month, inclusive (YYYY-MM; default current)")
	backfillCmd.AddCommand(backfillPointsCmd)

	scanCharacterTrailingCmd.Flags().Int32Var(&flagScanFrom, "from", 0, "Start after this id (default: highest known)")
	scanCharacterTrailingCmd.Flags().IntVarP(&flagScanAhead, "ahead", "a", 10, "Probe at most this many ids")
	scanCharacterTrailingCmd.Flags().IntVar(&flagScanMaxMisses, "max-misses", 2, "Stop after this many cumulative misses")
	scanCharacterTrailingCmd.Flags().BoolVarP(&flagScanKeepGoing, "continue", "c", false, "Ignore --ahead and run until the miss cap")
	scanCharacterTrailingCmd.Flags().BoolVar(&flagScanDryRun, "dry-run", false, "Probe without storing")
	scanCmd.AddCommand(scanCharacterTrailingCmd)

	scanCharacterHolesCmd.Flags().Int32Var(&flagScanFrom, "from", 0, "Start at this id")
	scanCharacterHolesCmd.Flags().Int32Var(&flagScanTo, "to", 0, "Stop at this id")
	scanCharacterHolesCmd.Flags().IntVar(&flagScanMaxGap, "max-gap", 10_000, "Skip gaps wider than this")
	scanCharacterHolesCmd.Flags().IntVar(&flagScanProbeBlock, "probe-block", 3, "Ids to probe at each step")
	scanCharacterHolesCmd.Flags().IntVar(&flagScanSkip, "skip", 50, "Advance this far after an empty block")
	scanCharacterHolesCmd.Flags().IntVar(&flagScanMaxMisses, "max-misses", 5_000, "Stop after this many cumulative misses")
	scanCharacterHolesCmd.Flags().BoolVarP(&flagScanResume, "resume", "r", false, "Resume from the saved position")
	scanCharacterHolesCmd.Flags().BoolVar(&flagScanDryRun, "dry-run", false, "Probe without storing")
	scanCmd.AddCommand(scanCharacterHolesCmd)

	importZkbHistoryCmd.Flags().StringVarP(&flagImportFrom, "from", "f", "", "Newest day to import (YYYY-MM-DD)")
	importZkbHistoryCmd.Flags().StringVarP(&flagImportTo, "to", "t", "", "Oldest day to import (YYYY-MM-DD)")
	importCmd.AddCommand(importZkbHistoryCmd)
}
