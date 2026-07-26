package workers

import (
	"context"
	"errors"
	"time"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/jackc/pgx/v5"
)

// Announcing wars starting and ending.
//
// Both are transitions, so both need to know what we believed before the
// refresh overwrote it. That state is read from our own row rather than
// tracked in memory: the war worker is one of many, any of them may pick up a
// given war, and in-process state would announce once per worker.

// WarTickerMaxAge bounds how stale a transition may be and still be announced.
//
// The repair sweep walks hundreds of thousands of finished wars. Without a
// bound, the first pass over that backlog would announce every war EVE has
// ever had, in the order it happened to read them.
const WarTickerMaxAge = 24 * time.Hour

// warPrior is what we believed about a war before refreshing it.
type warPrior struct {
	Known    bool
	Started  bool
	Finished bool
}

// priorState reads the stored war.
//
// A war we have never seen is not the same as one we have seen and know
// nothing about: the first is a discovery and may announce, the second has
// already been through here.
func (w *WarWorker) priorState(ctx context.Context, warID int32) (warPrior, error) {
	var p warPrior
	var started, finished, retracted *time.Time

	err := w.Deps.Pool.QueryRow(ctx,
		`SELECT started, finished, retracted FROM wars WHERE war_id = $1`, warID).
		Scan(&started, &finished, &retracted)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return p, err
	}

	p.Known = true
	p.Started = started != nil
	// Retracted and finished are the same fact for this purpose: the war is
	// over and there is nothing further to announce about it.
	p.Finished = finished != nil || retracted != nil
	return p, nil
}

// announceWar emits a start or end announcement when the refresh revealed one.
//
// Neither is emitted for a war that was already in that state. Wars are
// refetched on a schedule, and re-announcing on every refresh would put the
// same war in the ticker until it expired.
func (w *WarWorker) announceWar(ctx context.Context, warID int32, prior warPrior, war esi.War, now time.Time) {
	ended := war.Finished
	if ended == "" {
		ended = war.Retracted
	}

	justStarted := !prior.Started && war.Started != "" && ended == "" && isFresh(war.Started, now)
	justEnded := !prior.Finished && ended != "" && isFresh(ended, now)

	if !justStarted && !justEnded {
		return
	}

	aggressor := w.sideName(ctx, war.Aggressor)
	defender := w.sideName(ctx, war.Defender)

	if justStarted {
		w.Deps.Ticker.WarStarted(ctx, warID, aggressor, defender, war.Mutual, war.OpenForAllies)
		return
	}

	w.Deps.Ticker.WarEnded(ctx, warID, aggressor, defender,
		war.Aggressor.IskDestroyed, war.Defender.IskDestroyed,
		war.Finished == "" && war.Retracted != "")
}

// isFresh reports a timestamp recent enough to announce.
//
// Future timestamps are not fresh. A war declared with a start date tomorrow
// has not started, and announcing it as though it had would be wrong for as
// long as it took to arrive.
func isFresh(value string, now time.Time) bool {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return false
	}
	age := now.Sub(t)
	return age >= 0 && age <= WarTickerMaxAge
}

// sideName is how a belligerent is displayed: its alliance if it has one, its
// corporation otherwise.
func (w *WarWorker) sideName(ctx context.Context, side esi.WarSide) string {
	if side.AllianceID != 0 {
		var name string
		err := w.Deps.Pool.QueryRow(ctx,
			`SELECT name FROM alliances WHERE alliance_id = $1`, side.AllianceID).Scan(&name)
		if err == nil && name != "" {
			return name
		}
	}
	if side.CorporationID != 0 {
		var name string
		err := w.Deps.Pool.QueryRow(ctx,
			`SELECT name FROM corporations WHERE corporation_id = $1`, side.CorporationID).Scan(&name)
		if err == nil && name != "" {
			return name
		}
	}
	return "Unknown"
}
