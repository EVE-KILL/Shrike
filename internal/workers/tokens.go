package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/entities"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/sso"
	"github.com/riverqueue/river"
)

// TokenRefreshWorker refreshes one character's SSO token and dispatches the
// work its scopes permit.
//
// The dispatch is here rather than on a schedule because the scopes are only
// known after the refresh: SSO decides what the token can do, and that can
// change between one refresh and the next when a user re-consents or loses an
// in-game role.
type TokenRefreshWorker struct {
	river.WorkerDefaults[queue.TokenRefreshArgs]
	Deps *Deps
}

func (w *TokenRefreshWorker) Work(ctx context.Context, job *river.Job[queue.TokenRefreshArgs]) error {
	id := job.Args.CharacterID

	token, err := sso.LoadToken(ctx, w.Deps.Pool, id)
	if err != nil {
		return err
	}
	// No token, or one already given up on. Neither is retryable.
	if token == nil || token.Disabled {
		return nil
	}

	refreshed, err := w.Deps.SSO.Refresh(ctx, token.RefreshToken)
	if errors.Is(err, sso.ErrPermanentlyDead) {
		// The user revoked consent or CCP invalidated the grant. Clear the
		// scopes too: leaving them would make the token look capable of work it
		// can no longer do.
		return sso.Disable(ctx, w.Deps.Pool, id, true)
	}
	if err != nil {
		return err
	}

	granted, err := sso.ScopesFromAccessToken(refreshed.AccessToken)
	if err != nil {
		return fmt.Errorf("read scopes for character %d: %w", id, err)
	}
	if len(granted) == 0 {
		// A token with no scopes can do nothing, so refreshing it forever is
		// pure waste.
		return sso.Disable(ctx, w.Deps.Pool, id, true)
	}

	// Locally revoked scopes are subtracted here, and this is the whole reason
	// they are stored separately — see sso.RevokeScope.
	token.Scopes = granted
	usable := token.Usable()
	if len(usable) == 0 {
		return sso.Disable(ctx, w.Deps.Pool, id, false)
	}

	stored, err := sso.StoreRefreshed(
		ctx, w.Deps.Pool, id, token.RefreshToken, refreshed, usable,
	)
	if err != nil {
		return err
	}
	if !stored {
		// A login or another refresh replaced this token while the EVE request
		// was in flight. That writer owns both the new credentials and the jobs
		// their scopes should dispatch.
		return nil
	}
	if w.Deps.Queue == nil {
		return nil
	}

	token.Scopes = usable

	if token.Has(sso.ScopeCharacterKillmails) {
		if _, err := queue.Dispatch(ctx, w.Deps.Queue,
			queue.CharacterKillmailArgs{CharacterID: id}, queue.Live); err != nil {
			return err
		}
	}

	if token.Has(sso.ScopeCorporationKillmails) {
		corpID, err := w.corporationOf(ctx, id)
		if err != nil {
			return err
		}
		// NPC corporations are excluded: every character belongs to one at some
		// point, none of them have corporation killmails worth reading, and the
		// endpoint answers 403 for all of them.
		if entities.IsPlayerCorporation(corpID) {
			if _, err := queue.Dispatch(ctx, w.Deps.Queue,
				queue.CorporationKillmailArgs{CorporationID: corpID, CharacterID: id},
				queue.Live); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *TokenRefreshWorker) corporationOf(ctx context.Context, characterID int32) (int32, error) {
	var corpID int32
	err := w.Deps.Pool.QueryRow(ctx,
		`SELECT coalesce(corporation_id, 0) FROM characters WHERE character_id = $1`,
		characterID).Scan(&corpID)
	if err != nil {
		// A token for a character we have never fetched is not an error; the
		// corporation fan-out simply cannot happen yet.
		return 0, nil
	}
	return corpID, nil
}

// cronTokenSync refreshes tokens that are close to expiring.
//
// Runs every thirty seconds against a twenty-minute access token lifetime, so
// there is ample slack: a token is picked up well before it expires rather than
// after something has already failed using it.
func (d *Deps) cronTokenSync(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("esi_token_sync")
	}

	// Half an access token's twenty-minute life, so a token is always picked up
	// with time to spare rather than after something has already failed on it.
	const refreshWithin = 10 * time.Minute
	const batch = 200

	ids, err := sso.StaleTokens(ctx, d.Pool, refreshWithin, batch)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", nil
	}

	args := make([]river.JobArgs, 0, len(ids))
	for _, id := range ids {
		args = append(args, queue.TokenRefreshArgs{CharacterID: id})
	}

	n, err := queue.DispatchMany(ctx, d.Queue, args, queue.Live)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d tokens queued for refresh", n), nil
}
