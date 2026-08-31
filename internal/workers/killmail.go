package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/entities"
	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/intelrollup"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/relay"
	"github.com/eve-kill/shrike/internal/wars"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// Processing one killmail.
//
// The shape is: get the ESI document, parse it, store it, then run everything
// derived from it. The first step has two paths that differ enormously in cost
// — the document may already be in hand, or may need fetching — and the rest is
// the same either way.
//
// The derived half is not a straight line. Each piece of it is a separate
// effect with its own bit in the ledger (see internal/killmail/effects.go), and
// they are attempted independently so that a Memgraph outage does not stop the
// statistics being written. A killmail already stored is therefore not
// necessarily finished: the common reason to see one again is that some of its
// effects have not run yet.

// KillmailWorker processes incoming killmails.
type KillmailWorker struct {
	river.WorkerDefaults[queue.KillmailArgs]
	Deps *Deps
}

// Shared with backend/src/queues/killmails/queue.ts. The old namespace ending
// in 722 belonged to a retired session-lock implementation and must not be
// reused while pooled server sessions may still hold it.
const killmailProcessingLockNamespace = 20_260_723

func (w *KillmailWorker) Work(ctx context.Context, job *river.Job[queue.KillmailArgs]) error {
	// Hold the same transaction-scoped advisory lock as the TypeScript worker
	// for the whole pipeline. The work itself may use other pool connections;
	// this otherwise-empty transaction exists solely to serialize every
	// producer of one killmail across both runtimes.
	lockTx, err := w.Deps.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer lockTx.Rollback(ctx) //nolint:errcheck // no-op after commit
	if _, err := lockTx.Exec(ctx,
		`SELECT pg_advisory_xact_lock($1, $2)`,
		killmailProcessingLockNamespace, job.Args.KillmailID); err != nil {
		return err
	}

	if err := w.process(ctx, job); err != nil {
		return err
	}
	return lockTx.Commit(ctx)
}

func (w *KillmailWorker) process(ctx context.Context, job *river.Job[queue.KillmailArgs]) error {
	args := job.Args

	// Reaching this queue from any source means the kill no longer needs to
	// wait, whatever put it in the delay table.
	if err := killmail.Undelay(ctx, w.Deps.Pool, []int64{args.KillmailID}); err != nil {
		return err
	}

	// Already stored is the common case, not an exception: R2Z2 re-publishes
	// during backfeed storms, the repair cron re-offers anything it cannot
	// prove we have, and the war queue re-enqueues kills it finds.
	existing, err := killmail.Prepare(ctx, w.Deps.Pool, args.KillmailID, args.WarID)
	if err != nil {
		if errors.Is(err, killmail.ErrWarConflict) {
			// Two wars cannot both own a killmail, and retrying will not change
			// which one does.
			return river.JobCancel(err)
		}
		return err
	}

	var parsed *killmail.Parsed

	switch {
	case existing.Exists:
		if !existing.Tracked || existing.Done() {
			return nil
		}
		// A historical kill the war queue has just associated has every other
		// effect already marked complete. Aggregating the war is then the only
		// outstanding work, and it needs nothing but the stored rows — no ESI
		// call, no reparse.
		if existing.Pending() == killmail.EffectWarInteractions {
			return w.aggregateWar(ctx, args.KillmailID)
		}
		if parsed, err = killmail.Load(ctx, w.Deps.Pool, args.KillmailID); err != nil {
			return fmt.Errorf("load stored killmail %d: %w", args.KillmailID, err)
		}

	default:
		doc := args.Killmail
		if doc == nil {
			// No embedded document, so it has to be fetched. This is the
			// expensive path: one ESI request against the killmail rate limit
			// and the shared error budget, for a document the live feed would
			// have handed over for free.
			fetched, err := w.fetch(ctx, args.KillmailID, args.KillmailHash)
			if err != nil {
				return err
			}
			doc = fetched
		}

		if parsed, err = killmail.Parse(ctx, w.Deps.Cache, w.Deps.Prices, doc, args.KillmailHash, args.WarID); err != nil {
			return fmt.Errorf("parse killmail %d: %w", args.KillmailID, err)
		}

		inserted, err := killmail.Insert(ctx, w.Deps.Pool, parsed)
		if err != nil {
			return fmt.Errorf("store killmail %d: %w", args.KillmailID, err)
		}
		if !inserted {
			// Another worker won the race between the check and the insert.
			// Whatever it is doing, it holds the ledger row, so the effects are
			// its responsibility rather than a race between the two of us.
			return nil
		}
	}

	return w.runEffects(ctx, parsed, cascadeTier(job.Priority))
}

// runEffects performs everything derived from a stored killmail.
//
// Every effect is attempted even when an earlier one fails, and the failures
// are collected rather than returned at the first sign of trouble. They are
// independent: the graph being unreachable is no reason to skip the statistics,
// and returning early would leave the retry re-running whatever did succeed.
// Joining them means one retry picks up exactly the ones still outstanding.
func (w *KillmailWorker) runEffects(ctx context.Context, p *killmail.Parsed, tier queue.Priority) error {
	id := p.Killmail.KillmailID
	pool := w.Deps.Pool
	var errs []error

	// Database effects: the work and its ledger bit commit together.
	_, err := killmail.RunDBEffect(ctx, pool, id, killmail.EffectLastActive,
		func(ctx context.Context, tx pgx.Tx) (bool, error) {
			return true, killmail.UpdateLastActive(ctx, tx, *p)
		}, killmail.EffectOptions{})
	errs = append(errs, err)

	errs = append(errs, w.aggregateWar(ctx, id))

	types := killtype.Classify(killmail.Subject(w.Deps.Cache, p.Killmail))
	_, err = killmail.RunDBEffect(ctx, pool, id, killmail.EffectDailyKillRollup,
		func(ctx context.Context, tx pgx.Tx) (bool, error) {
			return true, killmail.BumpRollup(ctx, tx, types, killmail.UTCDateKey(*p))
		}, killmail.EffectOptions{})
	errs = append(errs, err)

	// External effects: enqueues and publishes, each keyed so that a repeat
	// collapses rather than duplicating.
	if w.Deps.Queue != nil {
		for _, e := range []struct {
			bit  killmail.Effect
			args river.JobArgs
		}{
			{killmail.EffectAchievementsDispatched, w.achievementArgs(p)},
			{killmail.EffectFitDispatched, queue.FitExtractArgs{KillmailID: id}},
			{killmail.EffectStatsDispatched, queue.StatsWriterArgs{KillmailID: id}},
			{killmail.EffectGraphDispatched, queue.GraphIngestArgs{KillmailID: id}},
		} {
			_, err := killmail.RunExternalEffect(ctx, pool, id, e.bit, func(ctx context.Context) error {
				_, err := queue.Dispatch(ctx, w.Deps.Queue, e.args, queue.Live)
				return err
			})
			errs = append(errs, err)
		}
		if p.Killmail.KillmailTime.After(time.Now().UTC().AddDate(0, 0, -intelrollup.RetentionDays)) {
			// Debounce a live day's burst. The durable dirty marker was committed
			// with the killmail, so collapsing arrivals for five minutes loses no
			// work and avoids rebuilding today once per killmail.
			_, err := queue.DispatchAt(ctx, w.Deps.Queue,
				queue.CharacterIntelRollupArgs{Day: p.Killmail.KillmailTime.UTC().Format("2006-01-02")},
				queue.RecentBackfill, 5*time.Minute)
			errs = append(errs, err)
		}
	}

	// Broadcast before the entity refresh: the live feed should show the kill
	// as soon as it is stored, not after several hundred ESI lookups have been
	// queued for the people on it.
	_, err = killmail.RunExternalEffect(ctx, pool, id, killmail.EffectEventPublished,
		func(ctx context.Context) error {
			return w.broadcast(ctx, p)
		})
	errs = append(errs, err)

	_, err = killmail.RunExternalEffect(ctx, pool, id, killmail.EffectTickerEvaluated,
		func(ctx context.Context) error {
			w.Deps.Ticker.EvaluateKillmail(ctx, *p)
			return nil
		})
	errs = append(errs, err)

	_, err = killmail.RunExternalEffect(ctx, pool, id, killmail.EffectEntitiesEnsured,
		func(ctx context.Context) error { return w.refreshEntities(ctx, p, tier) })
	errs = append(errs, err)

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("killmail %d has incomplete derived effects: %w", id, err)
	}
	return nil
}

// aggregateWar folds the killmail into war_interactions.
//
// The shared advisory lock is what makes this safe to run against a table that
// rebuild:war-interactions may be replacing wholesale. The rebuild takes the
// same lock exclusively, so an increment and a swap cannot interleave and lose
// each other's work.
func (w *KillmailWorker) aggregateWar(ctx context.Context, killmailID int64) error {
	_, err := killmail.RunDBEffect(ctx, w.Deps.Pool, killmailID, killmail.EffectWarInteractions,
		func(ctx context.Context, tx pgx.Tx) (bool, error) {
			return wars.AggregateKillmail(ctx, tx, killmailID)
		},
		killmail.EffectOptions{
			BeforeLedgerLock: func(ctx context.Context, tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock_shared($1, $2)`,
					wars.LockNamespace, wars.LockKey)
				return err
			},
		})
	return err
}

// fetch retrieves a killmail from ESI.
func (w *KillmailWorker) fetch(ctx context.Context, id int64, hash string) (*killmail.ESIKillmail, error) {
	if hash == "" {
		// Without a hash the killmail cannot be requested at all, and no number
		// of retries will produce one. Cancelling says so once instead of
		// failing three times.
		return nil, river.JobCancel(fmt.Errorf("killmail %d has no hash", id))
	}

	res, err := esi.Get[killmail.ESIKillmail](ctx, w.Deps.ESI, esi.KillmailPath(id, hash))
	if err != nil {
		return nil, err
	}

	// A 404 or 422 here means the id and hash do not name a real killmail —
	// a bad hash, or a kill CCP has removed. Retrying asks the same question
	// and gets the same answer.
	if res.Status == 404 || res.Status == 422 {
		return nil, river.JobCancel(fmt.Errorf("killmail %d/%s does not exist (%d)", id, hash, res.Status))
	}
	if !res.OK() || res.Data == nil {
		return nil, fmt.Errorf("ESI returned %d for killmail %d", res.Status, id)
	}
	return res.Data, nil
}

// achievementArgs reduces a parsed killmail to what the badge processor needs.
func (w *KillmailWorker) achievementArgs(p *killmail.Parsed) queue.AchievementsArgs {
	km := p.Killmail

	args := queue.AchievementsArgs{
		KillmailID:        km.KillmailID,
		TotalValue:        km.TotalValue,
		IsNPC:             km.IsNPC,
		IsSolo:            km.IsSolo,
		VictimShipGroupID: km.VictimShipGroupID,
		VictimCharacterID: km.VictimCharacterID,
	}
	if s, ok := w.Deps.Cache.System(km.SolarSystemID); ok {
		args.SystemSecurity, args.HasSecurity = s.Security, true
	}
	for _, a := range p.Attackers {
		args.Attackers = append(args.Attackers, queue.AchievementAttacker{
			CharacterID: a.CharacterID,
			ShipGroupID: a.ShipGroupID,
			FinalBlow:   a.FinalBlow,
		})
	}
	return args
}

// broadcast publishes the kill to the live feed.
//
// Every failure here is swallowed rather than returned. The killmail is stored;
// failing the job would retry the whole parse and insert to re-send a
// notification, and the client recovers on its next poll from the feed sequence
// anyway.
func (w *KillmailWorker) broadcast(ctx context.Context, p *killmail.Parsed) error {
	if w.Deps.Relay == nil {
		return nil
	}

	var security float64
	var hasSecurity bool
	if s, ok := w.Deps.Cache.System(p.Killmail.SolarSystemID); ok {
		security, hasSecurity = s.Security, true
	}
	keys := relay.RoutingKeys(p, security, hasSecurity)

	names, err := w.killmailNames(ctx, p)
	if err != nil {
		return fmt.Errorf("resolve relay entity names: %w", err)
	}

	typeIDs := make([]int32, 0, len(p.Items))
	for _, item := range p.Items {
		typeIDs = append(typeIDs, item.TypeID)
	}
	day, err := w.Deps.Prices.On(
		ctx,
		p.Killmail.KillmailTime.UTC().Format("2006-01-02"),
		typeIDs,
	)
	if err != nil {
		return fmt.Errorf("resolve relay item prices: %w", err)
	}

	// Keep the TypeScript order: the full event is sent first, then the durable
	// feed row, then the compact killlist event.
	full := relay.BuildKillmailEvent(p, w.Deps.Cache, names, day.Of)
	w.Deps.Relay.Publish(ctx, relay.ChannelKillmail, keys, full)

	// Unlike pub/sub, feed_queue is durable. If this insert fails the effect
	// must stay pending so a River retry repairs it.
	if err := w.Deps.Relay.PublishToFeed(
		ctx, w.Deps.Pool, p.Killmail.KillmailID, keys,
	); err != nil {
		return fmt.Errorf("append killmail to feed: %w", err)
	}

	row := relay.BuildKilllistRow(p, w.Deps.Cache, w.Deps.MarketPaths, names)
	w.Deps.Relay.Publish(ctx, relay.ChannelKilllist, keys,
		relay.KilllistEvent{Event: "killlist", Killmail: row})
	return nil
}

// killmailNames resolves every entity named by the full killmail event. The
// same maps also cover the smaller killlist event.
func (w *KillmailWorker) killmailNames(ctx context.Context, p *killmail.Parsed) (relay.EntityNames, error) {
	km := p.Killmail
	characters := []int32{km.VictimCharacterID}
	corporations := []int32{km.VictimCorporationID}
	alliances := []int32{km.VictimAllianceID}

	for _, a := range p.Attackers {
		characters = append(characters, a.CharacterID)
		corporations = append(corporations, a.CorporationID)
		alliances = append(alliances, a.AllianceID)
	}
	return relay.LookupNames(ctx, w.Deps.Pool, characters, corporations, alliances)
}

// refreshEntities enqueues fetches for the entities this killmail named.
//
// A killmail is the main way the killboard learns that anyone exists at all, so
// this is where most entity discovery happens. The decision about which of the
// named entities is actually worth fetching belongs to internal/entities —
// there are up to several thousand on a large fight, and refetching all of them
// would spend the entire ESI budget on a single battle.
func (w *KillmailWorker) refreshEntities(ctx context.Context, p *killmail.Parsed, tier queue.Priority) error {
	if w.Deps.Queue == nil {
		return nil
	}

	ref := entities.Referenced{KillmailTime: p.Killmail.KillmailTime}

	if p.Killmail.VictimCharacterID != 0 {
		ref.Affiliations = append(ref.Affiliations, entities.Affiliation{
			CharacterID:   p.Killmail.VictimCharacterID,
			CorporationID: p.Killmail.VictimCorporationID,
			AllianceID:    p.Killmail.VictimAllianceID,
		})
	}
	ref.Corporations = append(ref.Corporations, p.Killmail.VictimCorporationID)
	ref.Alliances = append(ref.Alliances, p.Killmail.VictimAllianceID)

	for _, a := range p.Attackers {
		if a.CharacterID != 0 {
			ref.Affiliations = append(ref.Affiliations, entities.Affiliation{
				CharacterID:   a.CharacterID,
				CorporationID: a.CorporationID,
				AllianceID:    a.AllianceID,
			})
		}
		ref.Corporations = append(ref.Corporations, a.CorporationID)
		ref.Alliances = append(ref.Alliances, a.AllianceID)
	}

	cascade, err := entities.Stale(ctx, w.Deps.Pool, ref)
	if err != nil {
		return err
	}
	_, err = w.Deps.dispatchCascade(ctx, cascade, tier)
	return err
}

// killmailExists reports whether the killmail is already stored.
func killmailExists(ctx context.Context, d *Deps, id int64) (bool, error) {
	var found bool
	err := d.Pool.QueryRow(ctx, `SELECT true FROM killmails WHERE killmail_id = $1`, id).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return found, nil
}
