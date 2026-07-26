package workers

import (
	"context"
	"fmt"

	"github.com/eve-kill/shrike/internal/entities"
	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/fw"
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

	// New and active wars are worth a full refresh including their killmails.
	var live []river.JobArgs
	for _, id := range found.New {
		live = append(live, queue.WarArgs{WarID: id})
	}
	for _, id := range found.Active {
		live = append(live, queue.WarArgs{WarID: id})
	}
	if _, err := queue.DispatchMany(ctx, d.Queue, live, queue.Live); err != nil {
		return "", err
	}

	// Repairs are metadata-only: their killmails are already stored.
	var repair []river.JobArgs
	for _, id := range found.Missing {
		repair = append(repair, queue.WarArgs{WarID: id, MetadataOnly: true})
	}
	if _, err := queue.DispatchMany(ctx, d.Queue, repair, queue.DormantBackfill); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d new, %d active, %d missing metadata repaired",
		len(found.New), len(found.Active), len(found.Missing)), nil
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
          AND (updated_at IS NULL OR updated_at < now() - ($2 || ' days')::interval)
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
		historyTable:  "character_corporation_history",
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

	corporations, err := d.claimHistoryBatch(ctx, historyBatch{
		table: "corporations", idColumn: "corporation_id",
		fetchedColumn: "alliance_history_fetched_at",
		queuedColumn:  "alliance_history_queued_at",
		historyTable:  "corporation_alliance_history",
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
	historyTable  string
	order         string
	limit         int
	playerOnly    bool
}

// claimHistoryBatch selects entities with no history and marks them queued.
//
// The queued marker is what stops the same twenty entities being re-dispatched
// every minute while their fetches are still in flight; it expires after six
// hours so a dispatch that was lost does not park an entity forever.
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
          AND NOT EXISTS (SELECT 1 FROM %[6]s h WHERE h.%[1]s = c.%[1]s)
        ORDER BY %[7]s
        LIMIT %[8]d`,
		b.idColumn, b.table, b.fetchedColumn, b.queuedColumn, playerFilter,
		b.historyTable, b.order, b.limit))
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
	if len(ids) == 0 {
		return nil, nil
	}

	if _, err := d.Pool.Exec(ctx, fmt.Sprintf(
		`UPDATE %s SET %s = now() WHERE %s = ANY($1::int[])`,
		b.table, b.queuedColumn, b.idColumn), ids); err != nil {
		return nil, err
	}
	return ids, nil
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
//
// Character ids are allocated sequentially, so the few above our maximum are
// the ones created since the last probe. Walking a short way past the end and
// stopping after consecutive misses finds them without scanning anything.
func (d *Deps) cronFindNewCharacters(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("find_new_characters")
	}

	const ahead = 10
	const maxMisses = 2

	var highest int32
	if err := d.Pool.QueryRow(ctx,
		`SELECT max(character_id) FROM characters`).Scan(&highest); err != nil {
		return "", err
	}
	if highest == 0 {
		return "no characters stored yet", nil
	}

	var found []int32
	misses := 0
	for i := int32(1); i <= ahead && misses < maxMisses; i++ {
		id := highest + i

		res, err := esi.FetchCharacter(ctx, d.ESI, id)
		if err != nil {
			return "", err
		}
		if res.Status == 404 || res.Status == 400 {
			misses++
			continue
		}
		if !res.OK() {
			// ESI being unwell is not evidence about the id; stop rather than
			// counting it as a miss and giving up early.
			break
		}
		misses = 0
		found = append(found, id)
	}

	if len(found) == 0 {
		return "", nil
	}

	args := make([]river.JobArgs, 0, len(found))
	for _, id := range found {
		args = append(args, queue.CharacterArgs{CharacterID: id})
	}
	if _, err := queue.DispatchMany(ctx, d.Queue, args, queue.RecentBackfill); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d new characters above %d", len(found), highest), nil
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

// cronSDEUpdate checks whether CCP has published a newer static data export.
//
// It reports rather than imports. The import is tens of minutes of work, tens
// of tables and about a hundred megabytes, and running it from inside a cron
// tick — which is what the TypeScript does by shelling out to the CLI — means a
// scheduled job that can occupy a worker for an hour and cannot be observed
// while it does. Surfacing "a new build exists" and leaving `sde:import` to be
// run deliberately keeps the long operation something a human starts and
// watches.
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
	if loaded == 0 {
		return fmt.Sprintf("no SDE imported; build %d is available — run sde:import",
			manifest.BuildNumber), nil
	}
	return fmt.Sprintf("SDE build %d is available, %d is loaded — run sde:import",
		manifest.BuildNumber, loaded), nil
}
