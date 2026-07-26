package workers

import (
	"context"
	"fmt"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/sso"
	"github.com/riverqueue/river"
)

// Killmails read through a character's own token.
//
// The public feed carries almost everything, so the value here is the
// remainder: a character's own token returns kills nobody reported to
// zKillboard, which for a quiet corporation can be most of its losses.
//
// Both endpoints are paginated and both are cheap per page, so the loop runs to
// exhaustion rather than taking a bounded slice. The expensive part is what
// comes after — every id found still needs its killmail fetched — which is why
// only the ones not already stored are enqueued.

// killmailPageLimit bounds the walk. ESI pages these at 1,000 ids and a
// character's history is finite, but a bug in the paging condition would
// otherwise loop forever against a live endpoint.
const killmailPageLimit = 100

// CharacterKillmailWorker reads one character's killmails.
type CharacterKillmailWorker struct {
	river.WorkerDefaults[queue.CharacterKillmailArgs]
	Deps *Deps
}

func (w *CharacterKillmailWorker) Work(ctx context.Context, job *river.Job[queue.CharacterKillmailArgs]) error {
	id := job.Args.CharacterID

	token, err := w.Deps.usableToken(ctx, id, sso.ScopeCharacterKillmails)
	if err != nil || token == nil {
		return err
	}

	refs, err := w.Deps.walkKillmails(ctx, token.AccessToken,
		fmt.Sprintf("/latest/characters/%d/killmails/recent/", id), id, sso.ScopeCharacterKillmails)
	if err != nil {
		return err
	}
	return w.Deps.enqueueUnknownKillmails(ctx, refs, token.DelayHours)
}

// CorporationKillmailWorker reads one corporation's killmails.
type CorporationKillmailWorker struct {
	river.WorkerDefaults[queue.CorporationKillmailArgs]
	Deps *Deps
}

func (w *CorporationKillmailWorker) Work(ctx context.Context, job *river.Job[queue.CorporationKillmailArgs]) error {
	corpID, charID := job.Args.CorporationID, job.Args.CharacterID

	token, err := w.Deps.usableToken(ctx, charID, sso.ScopeCorporationKillmails)
	if err != nil || token == nil {
		return err
	}

	refs, err := w.Deps.walkKillmails(ctx, token.AccessToken,
		fmt.Sprintf("/latest/corporations/%d/killmails/recent/", corpID),
		charID, sso.ScopeCorporationKillmails)
	if err != nil {
		return err
	}
	return w.Deps.enqueueUnknownKillmails(ctx, refs, token.DelayHours)
}

// usableToken loads a token and checks it still carries the scope.
//
// Returns nil with no error when there is nothing to do: no token, disabled, or
// the scope has been locally revoked. All three are ordinary states rather than
// failures, and none is improved by retrying.
func (d *Deps) usableToken(ctx context.Context, characterID int32, scope string) (*sso.Token, error) {
	token, err := sso.LoadToken(ctx, d.Pool, characterID)
	if err != nil || token == nil {
		return nil, err
	}
	if token.Disabled || !token.Has(scope) || token.AccessToken == "" {
		return nil, nil
	}
	return token, nil
}

// walkKillmails pages an authenticated killmail list.
//
// A 403 is the interesting case and the reason this is not a plain paginated
// fetch. SSO grants the corporation killmail scope to anyone who consented to
// it, but the endpoint additionally requires an in-game role — so a character
// who lost that role keeps being granted a scope they cannot use, and every
// attempt is a 403 against the shared error budget. Recording the revocation
// locally is what stops the next refresh handing the scope straight back.
func (d *Deps) walkKillmails(ctx context.Context, accessToken, path string, characterID int32, scope string) ([]esi.WarKillmailRef, error) {
	var out []esi.WarKillmailRef

	for page := 1; page <= killmailPageLimit; page++ {
		url := fmt.Sprintf("%s?page=%d", path, page)
		res, err := esi.GetAuthenticated[[]esi.WarKillmailRef](ctx, d.ESI, url, accessToken)
		if err != nil {
			return out, err
		}

		if res.Status == 403 {
			var err error
			if scope == sso.ScopeCharacterKillmails {
				err = sso.RecordCharacterKillmailFailure(ctx, d.Pool, characterID)
			} else {
				err = sso.RevokeScope(ctx, d.Pool, characterID, scope)
			}
			if err != nil {
				return out, err
			}
			// Not an error to retry. Corporation access is revoked immediately;
			// character access is retired only after a five-request streak.
			return out, nil
		}
		// A character with no killmails answers 404 rather than an empty list.
		if res.Status == 404 {
			return out, nil
		}
		if !res.OK() || res.Data == nil {
			return out, fmt.Errorf("ESI returned %d for %s", res.Status, path)
		}
		if scope == sso.ScopeCharacterKillmails {
			if err := sso.ResetCharacterKillmailFailures(ctx, d.Pool, characterID); err != nil {
				return out, err
			}
		}

		refs := *res.Data
		if len(refs) == 0 {
			return out, nil
		}
		out = append(out, refs...)
	}
	return out, nil
}

// enqueueUnknownKillmails queues the killmails not already stored.
//
// The filter matters more here than elsewhere: these endpoints return a
// character's whole recent history on every run, so almost everything they
// report is already known, and enqueuing all of it would spend the killmail
// budget re-fetching what is on disk.
func (d *Deps) enqueueUnknownKillmails(
	ctx context.Context,
	refs []esi.WarKillmailRef,
	delayHours int,
) error {
	if len(refs) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(refs))
	for _, r := range refs {
		ids = append(ids, r.KillmailID)
	}

	rows, err := d.Pool.Query(ctx,
		`SELECT killmail_id FROM killmails WHERE killmail_id = ANY($1::bigint[])`, ids)
	if err != nil {
		return err
	}
	stored := make(map[int64]bool, len(ids))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		stored[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	var unknown []killmail.Ref
	for _, r := range refs {
		if !stored[r.KillmailID] {
			unknown = append(unknown, killmail.Ref{
				KillmailID:   r.KillmailID,
				KillmailHash: r.KillmailHash,
			})
		}
	}
	if len(unknown) == 0 {
		return nil
	}

	if delayHours > 0 {
		for _, ref := range unknown {
			if err := killmail.Delay(ctx, d.Pool, ref, delayHours); err != nil {
				return err
			}
		}
		return nil
	}

	ids = ids[:0]
	for _, ref := range unknown {
		ids = append(ids, ref.KillmailID)
	}
	if err := killmail.Undelay(ctx, d.Pool, ids); err != nil {
		return err
	}
	if d.Queue == nil {
		return nil
	}

	// Recent backfill rather than live: these are kills that already happened
	// and were missed, so they must never delay one arriving now.
	batch := make([]river.JobArgs, 0, len(unknown))
	for _, ref := range unknown {
		batch = append(batch, queue.KillmailArgs{
			KillmailID:   ref.KillmailID,
			KillmailHash: ref.KillmailHash,
		})
	}
	_, err = queue.DispatchMany(ctx, d.Queue, batch, queue.RecentBackfill)
	return err
}
