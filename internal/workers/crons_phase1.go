package workers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/eve-kill/shrike/internal/entities"
	"github.com/eve-kill/shrike/internal/fw"
	"github.com/eve-kill/shrike/internal/intelrollup"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/eve-kill/shrike/internal/maintenance"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/sde"
	"github.com/eve-kill/shrike/internal/stats"
	"github.com/eve-kill/shrike/internal/universe"
	"github.com/eve-kill/shrike/internal/wars"
	"github.com/riverqueue/river"
)

// The scheduled jobs whose dependencies were already ported.

// cronWars discovers wars worth fetching and queues them.
func (d *Deps) cronWars(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("wars")
	}

	found, err := wars.Find(ctx, d.Pool, d.ESI)
	if err != nil {
		return "", err
	}

	live, repair := warDiscoveryJobs(found)
	if _, err := queue.DispatchMany(ctx, d.Queue, live, queue.Live); err != nil {
		return "", err
	}

	if _, err := queue.DispatchMany(ctx, d.Queue, repair, queue.DormantBackfill); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d new, %d active, %d missing metadata repaired",
		len(found.New), len(found.Active), len(found.Missing)), nil
}

func warDiscoveryJobs(found wars.Discover) (live, repair []river.JobArgs) {
	// New and active wars need their killmail lists walked.
	for _, id := range found.New {
		live = append(live, queue.WarArgs{WarID: id})
	}
	for _, id := range found.Active {
		live = append(live, queue.WarArgs{WarID: id})
	}

	// A stored killmail referencing a missing war proves only that we have one
	// of the war's kills, not all of them. The TypeScript hourly repair walks
	// ESI's authoritative list after restoring the metadata, and the Go repair
	// must do the same or a partially imported historical war can stay partial
	// forever. The explicit bulk backfill command remains metadata-only because
	// that operator-selected path intentionally trades completeness for cost.
	for _, id := range found.Missing {
		repair = append(repair, queue.WarArgs{WarID: id})
	}
	return live, repair
}

// cronCorporationUpdate queues corporations that have gone stale.
func (d *Deps) cronCorporationUpdate(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("corporation_update")
	}

	const staleDays = 14
	const batch = 1000

	// NPC corporations are excluded: every new character belongs to one, so
	// they would dominate the batch and no amount of fetching makes them
	// interesting.
	rows, err := d.Pool.Query(ctx, `
        SELECT corporation_id FROM corporations
        WHERE deleted IS NOT TRUE
          AND corporation_id >= $1
          AND (updated_at IS NULL OR updated_at < now() - make_interval(days => $2))
        LIMIT $3`, entities.PlayerCorporationIDMin, staleDays, batch)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var args []river.JobArgs
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		args = append(args, queue.CorporationArgs{CorporationID: id})
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	n, err := queue.DispatchMany(ctx, d.Queue, args, queue.Live)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d stale corporations queued", n), nil
}

// Per-tick history backfill limits.
//
// Deliberately small. Each history endpoint allows 30 requests a minute, and
// this is a backfill of hundreds of thousands of entities that must never
// compete with a live request — so it trickles rather than floods, and the
// dormant tier keeps it behind everything else regardless.
const (
	historyCharactersPerTick   = 20
	historyCorporationsPerTick = 10
)

// cronEntityHistoryBackfill feeds missing histories into the queues slowly.
func (d *Deps) cronEntityHistoryBackfill(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("entity_history_backfill")
	}

	characters, err := d.claimHistoryBatch(ctx, historyBatch{
		table: "characters", idColumn: "character_id",
		fetchedColumn: "corporation_history_fetched_at",
		queuedColumn:  "corporation_history_queued_at",
		order:         "c.last_active DESC NULLS LAST, c.character_id",
		limit:         historyCharactersPerTick,
	})
	if err != nil {
		return "", err
	}

	var args []river.JobArgs
	for _, id := range characters {
		args = append(args, queue.CharacterHistoryArgs{CharacterID: id})
	}
	if _, err := queue.DispatchMany(ctx, d.Queue, args, queue.DormantBackfill); err != nil {
		return "", err
	}
	if err := d.markHistoryBatchQueued(ctx, historyBatch{
		table: "characters", idColumn: "character_id",
		queuedColumn: "corporation_history_queued_at",
	}, characters); err != nil {
		return "", err
	}

	corporations, err := d.claimHistoryBatch(ctx, historyBatch{
		table: "corporations", idColumn: "corporation_id",
		fetchedColumn: "alliance_history_fetched_at",
		queuedColumn:  "alliance_history_queued_at",
		order:         "c.updated_at DESC NULLS LAST, c.corporation_id",
		limit:         historyCorporationsPerTick,
		playerOnly:    true,
	})
	if err != nil {
		return "", err
	}

	args = args[:0]
	for _, id := range corporations {
		args = append(args, queue.CorporationHistoryArgs{CorporationID: id})
	}
	if _, err := queue.DispatchMany(ctx, d.Queue, args, queue.DormantBackfill); err != nil {
		return "", err
	}
	if err := d.markHistoryBatchQueued(ctx, historyBatch{
		table: "corporations", idColumn: "corporation_id",
		queuedColumn: "alliance_history_queued_at",
	}, corporations); err != nil {
		return "", err
	}

	if len(characters) == 0 && len(corporations) == 0 {
		return "nothing left to backfill", nil
	}
	return fmt.Sprintf("%d characters, %d corporations", len(characters), len(corporations)), nil
}

// historyBatch describes one of the two near-identical history queries.
type historyBatch struct {
	table         string
	idColumn      string
	fetchedColumn string
	queuedColumn  string
	order         string
	limit         int
	playerOnly    bool
}

// claimHistoryBatch selects entities whose history fetch has not completed.
//
// The marker is written only after the River dispatch succeeds. Marking first
// would park a batch for six hours if insertion failed.
func (d *Deps) claimHistoryBatch(ctx context.Context, b historyBatch) ([]int32, error) {
	playerFilter := ""
	if b.playerOnly {
		playerFilter = fmt.Sprintf("AND c.%s >= %d", b.idColumn, entities.PlayerCorporationIDMin)
	}

	// Every interpolated fragment here comes from the two call sites above,
	// never from input.
	rows, err := d.Pool.Query(ctx, fmt.Sprintf(`
        SELECT c.%[1]s
        FROM %[2]s c
        WHERE c.deleted IS NOT TRUE
          AND c.%[3]s IS NULL
          AND (c.%[4]s IS NULL OR c.%[4]s < now() - interval '6 hours')
          %[5]s
        ORDER BY %[6]s
        LIMIT %[7]d`,
		b.idColumn, b.table, b.fetchedColumn, b.queuedColumn, playerFilter,
		b.order, b.limit))
	if err != nil {
		return nil, err
	}

	var ids []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func (d *Deps) markHistoryBatchQueued(ctx context.Context, b historyBatch, ids []int32) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := d.Pool.Exec(ctx, fmt.Sprintf(
		`UPDATE %s SET %s = now() WHERE %s = ANY($1::int[])`,
		b.table, b.queuedColumn, b.idColumn), ids)
	return err
}

// cronKillmailDelayed dispatches killmails whose delay has expired.
func (d *Deps) cronKillmailDelayed(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("killmail_delayed")
	}

	const batch = 1000
	expired, err := killmail.ClaimExpired(ctx, d.Pool, batch)
	if err != nil {
		return "", err
	}
	if len(expired) == 0 {
		return "", nil
	}

	args := make([]river.JobArgs, 0, len(expired))
	for _, km := range expired {
		args = append(args, queue.KillmailArgs{
			KillmailID:   km.KillmailID,
			KillmailHash: km.KillmailHash,
		})
	}

	n, err := queue.DispatchMany(ctx, d.Queue, args, queue.Live)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d expired killmails dispatched", n), nil
}

// cronFwUpdate refreshes faction warfare occupancy.
func (d *Deps) cronFwUpdate(ctx context.Context) (string, error) {
	res, err := fw.ImportSystems(ctx, d.Pool, d.ESI)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d systems seen, %d changed, %d flips", res.Seen, res.Rows, res.Flips), nil
}

// cronFwStats refreshes faction standings and leaderboards.
func (d *Deps) cronFwStats(ctx context.Context) (string, error) {
	res, err := fw.ImportStats(ctx, d.Pool, d.ESI)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d factions, %d leaderboard entries", res.Factions, res.Leaderboards), nil
}

// cronSystemActivity records the hourly kills and jumps per system.
func (d *Deps) cronSystemActivity(ctx context.Context) (string, error) {
	res, err := universe.ImportActivity(ctx, d.Pool, d.ESI)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d systems, %d old rows purged", res.Systems, res.Purged), nil
}

// cronFindNewCharacters probes ids above the highest one known.
func (d *Deps) cronFindNewCharacters(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("find_new_characters")
	}

	const ahead = 10
	const maxMisses = 2

	highest, err := entities.HighestCharacterID(ctx, d.Pool)
	if err != nil {
		return "", err
	}
	if highest == 0 {
		return "no characters stored yet", nil
	}

	prober := &entities.Prober{
		Pool: d.Pool,
		ESI:  d.ESI,
		OnCascade: func(cascadeCtx context.Context, cascade entities.Cascade) error {
			_, err := d.dispatchCascade(cascadeCtx, cascade, queue.RecentBackfill)
			return err
		},
	}
	result, err := prober.ScanTrailing(ctx, highest, ahead, maxMisses, false)
	if err != nil {
		return "", err
	}
	if result.Hits == 0 {
		return "", nil
	}
	return fmt.Sprintf("%d new characters above %d", result.Hits, highest), nil
}

// --- Maintenance ---

func (d *Deps) cronFeedPurge(ctx context.Context) (string, error) {
	n, err := maintenance.PurgeFeed(ctx, d.Pool)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d feed rows purged", n), nil
}

func (d *Deps) cronFittingsPurge(ctx context.Context) (string, error) {
	res, err := maintenance.PurgeFittings(ctx, d.Pool)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d linkages, %d fits, %d items purged", res.Linkages, res.Fits, res.Items), nil
}

func (d *Deps) cronPriceCompaction(ctx context.Context) (string, error) {
	n, err := maintenance.CompactPrices(ctx, d.Pool)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "not enough history to compact yet", nil
	}
	return fmt.Sprintf("%d daily price rows compacted into weeks", n), nil
}

func (d *Deps) cronEntitySnapshot(ctx context.Context) (string, error) {
	n, err := maintenance.SnapshotEntities(ctx, d.Pool)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d entity snapshots written", n), nil
}

func (d *Deps) cronKillsDailyCountReconcile(ctx context.Context) (string, error) {
	n, err := maintenance.ReconcileDailyCounts(ctx, d.Pool, killtype.Predicates())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d (date, type) rows reconciled over %d days", n, maintenance.ReconcileDays), nil
}

// cronCharacterIntelRollup rebuilds reusable daily facts from canonical
// killmails. It is independent of Memgraph and safe to replay.
func (d *Deps) cronCharacterIntelRollup(ctx context.Context) (string, error) {
	res, err := intelrollup.Reconcile(ctx, d.Pool)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d days: %d characters, %d ships, %d targets",
		res.Days, res.Characters, res.Ships, res.Targets), nil
}

// cronStatsPipeline rebuilds the rolled-up stats periods and the leaderboards.
func (d *Deps) cronStatsPipeline(ctx context.Context) (string, error) {
	res, err := stats.RunPipeline(ctx, d.Pool)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"monthly %d/%d, yearly %d/%d, leaderboards %d, purged %d daily and %d monthly",
		res.MonthlyStats, res.MonthlyBreakdowns,
		res.YearlyStats, res.YearlyBreakdowns,
		res.Leaderboards, res.PurgedDaily, res.PurgedMonthly), nil
}

func errNeedsQueue(name string) error {
	return fmt.Errorf("%s needs a queue to dispatch into", name)
}

// cronSDEUpdate checks whether CCP has published a newer static data export and
// imports it. The River cron worker has a two-hour timeout for long maintenance
// jobs; the build marker is only written after the full import succeeds.
func (d *Deps) cronSDEUpdate(ctx context.Context) (string, error) {
	manifest, err := sde.FetchManifest(ctx, d.UserAgent)
	if err != nil {
		return "", err
	}

	loaded, _, err := sde.LoadedBuild(ctx, d.Pool)
	if err != nil {
		return "", err
	}

	if loaded == manifest.BuildNumber {
		return "", nil
	}

	result, err := sde.ImportBuild(ctx, d.Pool, manifest, sde.ImportOptions{
		CacheDir:  filepath.Join(".data", "sde"),
		UserAgent: d.UserAgent,
		Progress: func(message string) {
			d.Log.Info().
				Str("cron", "sde_update").
				Msg(message)
		},
	})
	if err != nil {
		return "", fmt.Errorf("import SDE build %d: %w", manifest.BuildNumber, err)
	}

	return fmt.Sprintf(
		"SDE build %d imported (%d rows read, %d written, %d stale rows pruned in %s; previous build %d)",
		result.BuildNumber, result.Read, result.Written, result.Pruned, result.Elapsed, loaded,
	), nil
}
