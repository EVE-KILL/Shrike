package workers

import (
	"context"
	"errors"

	"github.com/eve-kill/shrike/internal/killmail"

	"github.com/eve-kill/shrike/internal/achievements"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
)

// AchievementsWorker awards the badges one killmail earned.
//
// The job carries the killmail's shape rather than its id, unlike stats_writer.
// That is the TypeScript's choice and it is kept: the fields needed are a small
// fixed set already in hand when the killmail is processed, and re-reading the
// mail and its attackers would be two queries to reconstruct what the caller
// just had.
type AchievementsWorker struct {
	river.WorkerDefaults[queue.AchievementsArgs]
	Deps *Deps
}

func (w *AchievementsWorker) Work(ctx context.Context, job *river.Job[queue.AchievementsArgs]) error {
	a := job.Args

	// A job with no attackers came from a backfill, which has only the id. The
	// embedded shape is an optimisation for the live path — where the caller
	// already holds it — not a requirement, so fall back to reading the
	// killmail. Without this a backfill enqueues millions of jobs that each
	// award nothing.
	if len(a.Attackers) == 0 {
		loaded, err := w.hydrate(ctx, a.KillmailID)
		if err != nil {
			return err
		}
		if loaded == nil {
			// Deleted between dispatch and now.
			return nil
		}
		a = *loaded
	}

	attackers := make([]achievements.Attacker, 0, len(a.Attackers))
	for _, at := range a.Attackers {
		attackers = append(attackers, achievements.Attacker{
			CharacterID: at.CharacterID,
			ShipGroupID: at.ShipGroupID,
			FinalBlow:   at.FinalBlow,
		})
	}

	_, err := achievements.Process(ctx, w.Deps.Pool, achievements.Killmail{
		TotalValue:        a.TotalValue,
		SystemSecurity:    a.SystemSecurity,
		HasSecurity:       a.HasSecurity,
		IsNPC:             a.IsNPC,
		IsSolo:            a.IsSolo,
		VictimShipGroupID: a.VictimShipGroupID,
		VictimCharacterID: a.VictimCharacterID,
		Attackers:         attackers,
	})
	return err
}

// hydrate rebuilds the args from the stored killmail.
func (w *AchievementsWorker) hydrate(ctx context.Context, killmailID int64) (*queue.AchievementsArgs, error) {
	p, err := killmail.Load(ctx, w.Deps.Pool, killmailID)
	if errors.Is(err, killmail.ErrNotStored) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	km := p.Killmail
	out := queue.AchievementsArgs{
		KillmailID:        km.KillmailID,
		TotalValue:        km.TotalValue,
		IsNPC:             km.IsNPC,
		IsSolo:            km.IsSolo,
		VictimShipGroupID: km.VictimShipGroupID,
		VictimCharacterID: km.VictimCharacterID,
	}
	if s, ok := w.Deps.Cache.System(km.SolarSystemID); ok {
		out.SystemSecurity, out.HasSecurity = s.Security, true
	}
	for _, at := range p.Attackers {
		out.Attackers = append(out.Attackers, queue.AchievementAttacker{
			CharacterID: at.CharacterID,
			ShipGroupID: at.ShipGroupID,
			FinalBlow:   at.FinalBlow,
		})
	}
	return &out, nil
}
