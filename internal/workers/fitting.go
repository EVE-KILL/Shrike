package workers

import (
	"context"
	"errors"

	"github.com/eve-kill/shrike/internal/fitting"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
)

// FitExtractWorker records the fit a victim was flying.
type FitExtractWorker struct {
	river.WorkerDefaults[queue.FitExtractArgs]
	Deps *Deps
}

func (w *FitExtractWorker) Work(ctx context.Context, job *river.Job[queue.FitExtractArgs]) error {
	p, err := killmail.Load(ctx, w.Deps.Pool, job.Args.KillmailID)
	if errors.Is(err, killmail.ErrNotStored) {
		// Deleted between dispatch and now. Nothing to extract.
		return nil
	}
	if err != nil {
		return err
	}

	// Only ships have fits worth recording. A pod, a structure or a deployable
	// has items but no fitting, and storing one would put meaningless rows in
	// the table the item pages query.
	if !fitting.IsShipType(w.Deps.Cache, p.Killmail.VictimShipTypeID) {
		return nil
	}

	items := make([]fitting.Item, 0, len(p.Items))
	for _, it := range p.Items {
		items = append(items, fitting.Item{
			TypeID:      it.TypeID,
			FlagID:      it.FlagID,
			ParentIndex: it.ParentIndex,
			// Dropped plus destroyed: a drone bay's contents are split across
			// the two and the total is what round-trips.
			Quantity: int32(it.QuantityDropped + it.QuantityDestroyed),
		})
	}

	f := fitting.Extract(w.Deps.Cache, p.Killmail.VictimShipTypeID, items)
	// An empty hull has no fit identity — see Extract.
	if f == nil {
		return nil
	}

	return fitting.Store(ctx, w.Deps.Pool, f, fitting.Link{
		KillmailID:          p.Killmail.KillmailID,
		ShipTypeID:          p.Killmail.VictimShipTypeID,
		KillTime:            p.Killmail.KillmailTime,
		VictimAllianceID:    p.Killmail.VictimAllianceID,
		VictimCorporationID: p.Killmail.VictimCorporationID,
	})
}
