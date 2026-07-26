package zkb

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// The listener is a cursor walking a sequence, and the whole design follows
// from one property of the feed: sequence numbers are dense and assigned in
// order, so "what comes next" is always exactly cursor+1. There is no cursor
// token to carry, no overlap window to reconcile, and no possibility of two
// consumers disagreeing about ordering.
//
// What it does mean is that the cursor must never advance past an entry that
// was not handled. Every path below that fails leaves the cursor where it was,
// so the next attempt asks for the same sequence again.

// WaitOnCaughtUp is how long to pause after a 404. zKillboard's own guidance is
// six seconds; going faster earns a 429 and buys nothing, because the feed
// genuinely has nothing to hand over.
const WaitOnCaughtUp = 6 * time.Second

// WaitOnThrottled is the penalty pause after a 429 or 403.
const WaitOnThrottled = 30 * time.Second

// SaveInterval is how many sequences pass between cursor writes.
//
// The cursor is a performance optimisation, not a correctness mechanism: losing
// it costs a replay of up to fifty kills, all of which are rejected downstream
// as duplicates. Writing every sequence would be a database round trip per
// killmail for no benefit.
const SaveInterval = 50

// Store is what the listener needs from the rest of the system.
//
// Kept this narrow deliberately — the listener has no opinion about whether a
// killmail is enqueued for a worker or processed on the spot, and tests supply
// an implementation that does neither.
type Store interface {
	// Cursor reads the stored sequence, returning 0 when there is none.
	Cursor(ctx context.Context) (int64, error)

	// SaveCursor persists the sequence.
	SaveCursor(ctx context.Context, sequence int64) error

	// Has reports whether the killmail is already stored. R2Z2 re-publishes
	// mails during backfeed storms, and the same kill also arrives from the
	// ESI backfill path, so this is a routine hit rather than an anomaly.
	Has(ctx context.Context, killmailID int64) (bool, error)

	// Accept takes delivery of a killmail that is not yet stored.
	Accept(ctx context.Context, r *Response) error
}

// Event is one thing the listener did, reported for display.
type Event struct {
	Sequence   int64
	KillmailID int64
	Hash       string

	// Kind is "new", "repost", "caught-up", or "error".
	Kind string
	Err  error
}

// Stats is a running tally.
type Stats struct {
	Accepted int64 `json:"accepted"`
	Reposts  int64 `json:"reposts"`
	CaughtUp int64 `json:"caught_up"`
	Errors   int64 `json:"errors"`

	Sequence  int64 `json:"sequence"`
	StartedAt int64 `json:"started_at"`
}

// outcome is what one attempt at a sequence produced.
type outcome int

const (
	// handled means the entry was dealt with and the cursor may advance.
	handled outcome = iota
	// unpublished means the feed has nothing there yet. Not a failure: the
	// cursor stays put and no attempt is charged against the sequence.
	unpublished
	// failed means the attempt did not work and is worth retrying.
	failed
)

// Listener follows the feed.
type Listener struct {
	Client *Client
	Store  Store

	// OnEvent is called for every entry. Optional.
	OnEvent func(Event)

	// Counter records ingest throughput for the status page. Optional; a nil
	// client makes RecordIngest a no-op.
	Counter *redis.Client

	// Sleep is the pause between iterations, injectable so tests do not spend
	// six real seconds per catch-up.
	Sleep func(context.Context, time.Duration) error

	stats Stats
}

// Stats returns a copy of the running tally.
func (l *Listener) Stats() Stats { return l.stats }

func (l *Listener) pause(ctx context.Context, d time.Duration) error {
	if l.Sleep != nil {
		return l.Sleep(ctx, d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (l *Listener) emit(e Event) {
	if l.OnEvent != nil {
		l.OnEvent(e)
	}
}

// Start bootstraps the cursor and follows the feed until the context is
// cancelled, which is the only way it stops — every error is a pause, not an
// exit. A killmail feed that dies on the first transient failure is a feed that
// needs a babysitter.
//
// Returns the final stats and ctx.Err().
func (l *Listener) Start(ctx context.Context) (Stats, error) {
	l.stats.StartedAt = time.Now().Unix()

	sequence, err := l.Store.Cursor(ctx)
	if err != nil {
		return l.stats, err
	}

	if sequence == 0 {
		// No stored position. Start at the head rather than at the beginning:
		// the ephemeral feed's retention is hours, so there is no beginning to
		// start from, and a backfill is the history endpoint's job.
		sequence, err = l.Client.LatestSequence(ctx)
		if err != nil {
			return l.stats, err
		}
		if err := l.Store.SaveCursor(ctx, sequence); err != nil {
			return l.stats, err
		}
	}
	l.stats.Sequence = sequence

	// Every exit saves the cursor, through this one deferred write rather than
	// at each return. Doing it per return is what went wrong first: cancellation
	// almost always lands inside a pause, so step() returned the context error
	// and the loop bailed past the save at the top — leaving the cursor at the
	// last periodic write and replaying up to fifty killmails on restart.
	//
	// The context is detached because the reason we are saving is that the
	// original one has just been cancelled.
	defer func() {
		_ = l.Store.SaveCursor(context.WithoutCancel(ctx), l.stats.Sequence)
	}()

	for {
		if err := ctx.Err(); err != nil {
			return l.stats, err
		}

		next := sequence + 1
		got, err := l.step(ctx, next)
		if err != nil {
			return l.stats, err
		}

		switch got {
		case unpublished:
			// Caught up. The cursor stays, and this does not count as an
			// attempt — a consumer sitting at the head of the feed would
			// otherwise exhaust its attempts on an entry that has simply not
			// been written yet.
			continue

		case failed:
			// Never advance past an entry that was not accepted. R2Z2 sequence
			// numbers are dense, so doing so permanently loses that killmail
			// from this ingest path. This also matches the TypeScript listener:
			// a bad entry is retried until it succeeds or the process is
			// stopped, making the failure visible instead of turning the repair
			// cron into part of the normal delivery guarantee.
			continue
		}

		sequence = next
		l.stats.Sequence = sequence
		if sequence%SaveInterval == 0 {
			if err := l.Store.SaveCursor(ctx, sequence); err != nil {
				return l.stats, err
			}
		}
	}
}

// step makes one attempt at one sequence number.
//
// It returns an error only when the context is done; every other failure is
// absorbed into a pause and reported as `failed`, because the alternative is a
// listener that exits on the first bad minute R2Z2 has.
func (l *Listener) step(ctx context.Context, sequence int64) (outcome, error) {
	res, err := l.Client.Killmail(ctx, sequence)

	switch {
	case errors.Is(err, ErrNotPublished):
		l.stats.CaughtUp++
		l.emit(Event{Sequence: sequence, Kind: "caught-up"})
		return unpublished, l.pause(ctx, WaitOnCaughtUp)

	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return failed, err

	case errors.Is(err, ErrThrottled):
		// A throttle says nothing about this entry, only about our pace, so it
		// is charged as an attempt but waits far longer before the next one.
		l.stats.Errors++
		l.emit(Event{Sequence: sequence, Kind: "error", Err: err})
		return failed, l.pause(ctx, WaitOnThrottled)

	case err != nil:
		l.stats.Errors++
		l.emit(Event{Sequence: sequence, Kind: "error", Err: err})
		return failed, l.pause(ctx, WaitOnCaughtUp)
	}

	stored, err := l.Store.Has(ctx, res.KillmailID)
	if err != nil {
		l.stats.Errors++
		l.emit(Event{Sequence: sequence, KillmailID: res.KillmailID, Kind: "error", Err: err})
		return failed, l.pause(ctx, WaitOnCaughtUp)
	}

	if stored {
		l.stats.Reposts++
		// Telemetry only, so a Redis failure here must not interrupt the feed.
		_ = RecordIngest(ctx, l.Counter, IngestRepost)
		l.emit(Event{
			Sequence: sequence, KillmailID: res.KillmailID,
			Hash: res.KillmailHash(), Kind: "repost",
		})
		return handled, nil
	}

	if err := l.Store.Accept(ctx, res); err != nil {
		l.stats.Errors++
		l.emit(Event{
			Sequence: sequence, KillmailID: res.KillmailID,
			Hash: res.KillmailHash(), Kind: "error", Err: err,
		})
		return failed, l.pause(ctx, WaitOnCaughtUp)
	}

	l.stats.Accepted++
	_ = RecordIngest(ctx, l.Counter, IngestNew)
	l.emit(Event{
		Sequence: sequence, KillmailID: res.KillmailID,
		Hash: res.KillmailHash(), Kind: "new",
	})
	return handled, nil
}
