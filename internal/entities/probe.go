package entities

import (
	"context"
	"errors"
	"fmt"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Discovering characters by probing ids.
//
// A killboard learns about characters from killmails, which means it only ever
// sees people who have shot something or been shot. Everyone else is invisible.
// Probing the id space directly is how the rest are found — CCP allocates
// character ids roughly sequentially, so both the space above the highest id we
// know and the gaps between the ones we do know contain real people.
//
// The constraint throughout is the shared ESI error budget. A 404 counts
// against it, and the id space is mostly empty, so an unbounded scan would
// exhaust the budget for every other consumer within minutes. Every loop here
// is bounded by a miss count for that reason rather than for tidiness.
//
// Note what this deliberately does not use: Refresher.Character, which records
// a 404 as a deleted character row. That is right when a killmail named an id
// and it turned out to be biomassed, and completely wrong here — it would write
// a row for every empty id probed, filling in the very gaps the scan walks and
// leaving a re-run with nothing to find.

// ProbeOutcome is what one probe found.
type ProbeOutcome int

const (
	// ProbeHit means a character exists at that id.
	ProbeHit ProbeOutcome = iota
	// ProbeMiss means the id is unallocated or biomassed.
	ProbeMiss
	// ProbeError means ESI could not answer, which is not evidence either way.
	ProbeError
)

// Probe reports one id's result to a caller that wants to log it.
type Probe struct {
	ID      int32
	Outcome ProbeOutcome
	Name    string
}

// Prober probes character ids and ingests what it finds.
type Prober struct {
	Pool *pgxpool.Pool
	ESI  *esi.Client

	// DryRun probes without storing anything. The ESI calls still happen — a
	// dry run that skipped them would report nothing useful — so it still
	// spends the error budget.
	DryRun bool

	// OnProbe, when set, is called for every id attempted.
	OnProbe func(Probe)

	// OnCascade dispatches the history and corporation work implied by a hit.
	// The entity package deliberately does not know which queue implementation
	// the caller uses; scan commands provide the River adapter here.
	OnCascade func(context.Context, Cascade) error
}

// probe tries one id.
//
// Three statuses mean "nobody here", and all three are misses rather than
// errors: 404 for an id that was allocated and is now gone, 400 for a
// malformed one, and 422 for an id outside the range CCP has ever issued.
//
// The 422 is the one that matters and the one that is easy to get wrong.
// Treating it as an error aborts the scan on the very first probe past the
// allocated range — which is precisely where a trailing scan spends most of
// its time, so the whole command silently does nothing.
func (p *Prober) probe(ctx context.Context, id int32) (Probe, *esi.Character, error) {
	out := Probe{ID: id}

	res, err := esi.FetchCharacter(ctx, p.ESI, id)
	if err != nil {
		out.Outcome = ProbeError
		return out, nil, err
	}
	switch {
	case res.Status == 404 || res.Status == 400 || res.Status == 422:
		out.Outcome = ProbeMiss
		return out, nil, nil
	case res.OK() && res.Data != nil:
		out.Outcome = ProbeHit
		out.Name = res.Data.Name
		return out, res.Data, nil
	default:
		out.Outcome = ProbeError
		return out, nil, fmt.Errorf("%w: character %d returned %d", ErrTransient, id, res.Status)
	}
}

// ingest stores a discovered character.
func (p *Prober) ingest(ctx context.Context, id int32) (Cascade, error) {
	if p.DryRun {
		return Cascade{}, nil
	}
	r := &Refresher{Pool: p.Pool, ESI: p.ESI}
	// Refetched rather than passed through: Character does the full write, and
	// the second call is served from the ESI cache the probe just populated.
	res, err := r.Character(ctx, id)
	if err != nil {
		return res.Cascade, err
	}
	if p.OnCascade != nil && !res.Cascade.Empty() {
		if err := p.OnCascade(ctx, res.Cascade); err != nil {
			return res.Cascade, err
		}
	}
	return res.Cascade, nil
}

// TrailingResult reports a scan above the highest known id.
type TrailingResult struct {
	Probed  int          `json:"probed"`
	Hits    int          `json:"hits"`
	Misses  int          `json:"misses"`
	NewIDs  []int32      `json:"new_ids,omitempty"`
	LastID  int32        `json:"last_id"`
	Stopped StopReason   `json:"stopped"`
	Cascade CascadeTotal `json:"cascade,omitzero"`
}

// CascadeTotal counts the follow-up work discovery implied.
type CascadeTotal struct {
	Characters int `json:"characters,omitempty"`
	Histories  int `json:"histories,omitempty"`
}

// StopReason says why a scan ended.
type StopReason string

const (
	StopMaxMisses StopReason = "max-misses"
	StopReached   StopReason = "range-covered"
	StopESIError  StopReason = "esi-error"
)

// ScanTrailing walks forward from an id looking for newly allocated characters.
//
// Stops on cumulative misses rather than consecutive ones. CCP does not
// allocate ids densely, so a run of gaps inside live territory is normal and
// stopping at the first would give up immediately; a high cumulative count is
// the signal that the allocated range has genuinely been left behind.
func (p *Prober) ScanTrailing(ctx context.Context, start int32, ahead, maxMisses int, keepGoing bool) (TrailingResult, error) {
	out := TrailingResult{LastID: start}

	id := start + 1
	for {
		if out.Misses > maxMisses {
			out.Stopped = StopMaxMisses
			break
		}
		if !keepGoing && out.Probed >= ahead {
			out.Stopped = StopReached
			break
		}

		pr, _, _ := p.probe(ctx, id)
		out.Probed++
		out.LastID = id
		if p.OnProbe != nil {
			p.OnProbe(pr)
		}

		switch pr.Outcome {
		case ProbeError:
			out.Stopped = StopESIError
			// The scan is abandoned, not failed: everything found before the
			// error is real and has been stored.
			return out, nil
		case ProbeMiss:
			out.Misses++
		case ProbeHit:
			out.Hits++
			out.NewIDs = append(out.NewIDs, id)
			cascade, err := p.ingest(ctx, id)
			if err != nil && !errors.Is(err, ErrTransient) {
				return out, err
			}
			out.Cascade.Characters += len(cascade.Characters)
			out.Cascade.Histories += len(cascade.CharacterHistories)
		}

		id++
	}

	return out, nil
}

// HoleResult reports a scan of the gaps between known ids.
type HoleResult struct {
	GapsScanned int          `json:"gaps_scanned"`
	GapsSkipped int          `json:"gaps_skipped"`
	Probed      int          `json:"probed"`
	Hits        int          `json:"hits"`
	Misses      int          `json:"misses"`
	LastID      int32        `json:"last_id"`
	Stopped     StopReason   `json:"stopped,omitempty"`
	Cascade     CascadeTotal `json:"cascade,omitzero"`
}

// HoleOptions tunes a gap scan.
type HoleOptions struct {
	// From and To bound which known ids are walked. To of zero means no bound.
	From, To int32

	// MaxGap skips gaps wider than this.
	//
	// The very large gaps are CCP's unallocated allocation blocks — tens of
	// millions of ids with nobody in them. Walking one exhaustively would spend
	// weeks of error budget to find nothing, so they are left alone.
	MaxGap int

	// ProbeBlock is how many consecutive ids are tried at each step, and how far
	// a step advances when any of them hit.
	ProbeBlock int

	// Skip is how far to jump when a whole block missed. Larger than
	// ProbeBlock: an empty block is evidence the neighbourhood is empty, and
	// stepping one block at a time through a million-id void is what makes an
	// exhaustive scan infeasible.
	Skip int

	// MaxMisses ends the run.
	MaxMisses int
}

// ScanHoles walks the gaps between known character ids.
//
// Sampling rather than exhaustive: each step probes a small block and then
// either advances by that block, if anything was there, or jumps ahead. The
// assumption is that allocated ids cluster, so a hit means keep looking here
// and a miss means look elsewhere. It will not find every character in a gap,
// and it is not meant to — it is meant to find the populated regions of a space
// too large to enumerate.
func (p *Prober) ScanHoles(ctx context.Context, opts HoleOptions) (HoleResult, error) {
	var out HoleResult

	known, err := p.knownIDs(ctx, opts.From, opts.To)
	if err != nil {
		return out, err
	}

	var prev int32
	havePrev := false

	for _, cur := range known {
		if havePrev && cur > prev+1 {
			gapStart, gapEnd := prev+1, cur-1
			if int(gapEnd-gapStart)+1 > opts.MaxGap {
				out.GapsSkipped++
			} else {
				out.GapsScanned++
				stop, err := p.walkGap(ctx, gapStart, gapEnd, opts, &out)
				if err != nil {
					return out, err
				}
				if stop != "" {
					out.Stopped = stop
					out.LastID = prev
					return out, nil
				}
			}
		}
		prev = cur
		havePrev = true
		out.LastID = cur
	}

	return out, nil
}

// walkGap probes one gap, returning a stop reason when the run must end.
func (p *Prober) walkGap(ctx context.Context, gapStart, gapEnd int32, opts HoleOptions, out *HoleResult) (StopReason, error) {
	for id := gapStart; id <= gapEnd; {
		if out.Misses >= opts.MaxMisses {
			return StopMaxMisses, nil
		}

		blockEnd := id + int32(opts.ProbeBlock) - 1
		if blockEnd > gapEnd {
			blockEnd = gapEnd
		}

		blockHits := 0
		for probeID := id; probeID <= blockEnd; probeID++ {
			// Hole-scan dry runs are geometry-only in the TS command: count
			// which ids would be probed without spending ESI's shared error
			// budget and without treating simulated probes as misses.
			if p.DryRun {
				out.Probed++
				continue
			}

			pr, _, _ := p.probe(ctx, probeID)
			out.Probed++
			if p.OnProbe != nil {
				p.OnProbe(pr)
			}

			switch pr.Outcome {
			case ProbeError:
				return StopESIError, nil
			case ProbeMiss:
				out.Misses++
			case ProbeHit:
				out.Hits++
				blockHits++
				cascade, err := p.ingest(ctx, probeID)
				if err != nil && !errors.Is(err, ErrTransient) {
					return "", err
				}
				out.Cascade.Characters += len(cascade.Characters)
				out.Cascade.Histories += len(cascade.CharacterHistories)
			}
		}

		if blockHits > 0 {
			id += int32(opts.ProbeBlock)
		} else {
			id += int32(opts.Skip)
		}
	}
	return "", nil
}

// knownIDs reads the character ids bounding the gaps.
func (p *Prober) knownIDs(ctx context.Context, from, to int32) ([]int32, error) {
	query := `SELECT character_id FROM characters WHERE character_id >= $1`
	args := []any{from}
	if to != 0 {
		query += ` AND character_id <= $2`
		args = append(args, to)
	}
	query += ` ORDER BY character_id`

	rows, err := p.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// HighestCharacterID is the top of the known id space, which is where a
// trailing scan starts.
func HighestCharacterID(ctx context.Context, pool *pgxpool.Pool) (int32, error) {
	var id *int32
	if err := pool.QueryRow(ctx,
		`SELECT MAX(character_id) FROM characters`).Scan(&id); err != nil {
		return 0, err
	}
	if id == nil {
		return 0, nil
	}
	return *id, nil
}
