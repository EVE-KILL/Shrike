package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/campaign"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/sso"
	"github.com/eve-kill/shrike/internal/wallet"
	"github.com/riverqueue/river"
)

// CorporationWalletWorker syncs one corporation's balances and journal.
type CorporationWalletWorker struct {
	river.WorkerDefaults[queue.CorporationWalletArgs]
	Deps *Deps
}

const walletTokenRefreshMargin = 2 * time.Minute

type walletSyncBlockedError struct {
	message string
}

func (e *walletSyncBlockedError) Error() string { return e.message }

func (w *CorporationWalletWorker) Work(ctx context.Context, job *river.Job[queue.CorporationWalletArgs]) error {
	corpID := job.Args.CorporationID

	tokens, err := wallet.LoadTokens(ctx, w.Deps.Pool)
	if err != nil {
		return err
	}

	var token *wallet.Token
	for i := range tokens {
		if tokens[i].CorporationID == corpID {
			token = &tokens[i]
			break
		}
	}
	// No authorisation, or one already disabled. Neither is retryable.
	if token == nil {
		return nil
	}

	// The access token has to be current. Wallet tokens live in their own table
	// rather than user_esi_tokens, so the ordinary refresh job does not cover
	// them — the refresh happens here, inline, when the stored one has expired.
	if token.AccessToken == "" ||
		!token.Expiry.After(time.Now().UTC().Add(walletTokenRefreshMargin)) {
		if err := w.refreshWalletAccess(ctx, token); err != nil {
			return w.finishWalletError(ctx, corpID, err)
		}
	}

	now := time.Now().UTC()
	var balances, journal int64

	// Each half is gated separately: the balance is cheap and the journal is
	// seven paginated walks, so there is no reason to redo the expensive one
	// just because the cheap one came due.
	if job.Args.Force || wallet.Due(token.LastBalanceSync, now) {
		balances, err = w.withWalletRefreshRetry(ctx, token, func(access string) (int64, error) {
			return wallet.SyncBalances(ctx, w.Deps.Pool, w.Deps.ESI, corpID, access)
		})
		if err != nil {
			return w.finishWalletError(ctx, corpID, err)
		}
	}
	if job.Args.Force || wallet.Due(token.LastJournalSync, now) {
		journal, err = w.withWalletRefreshRetry(ctx, token, func(access string) (int64, error) {
			return wallet.SyncJournal(ctx, w.Deps.Pool, w.Deps.ESI, corpID, access)
		})
		if err != nil {
			return w.finishWalletError(ctx, corpID, err)
		}
	}

	// Donations to EVE-KILL that name a campaign fund its prize pool. Run after
	// the journal sync rather than inside it, so an entry that arrived in this
	// pass is classified in this pass — and unconditionally rather than only
	// when the sync ran, because this also catches up entries imported before
	// the funding feature existed.
	var funded int
	var deposits wallet.DepositResult
	if corpID == campaign.EveKillCorporationID {
		pending, err := campaign.PendingJournalEntries(ctx, w.Deps.Pool, corpID, campaign.PendingJournalLimit)
		if err != nil {
			return w.finishWalletError(ctx, corpID, err)
		}
		if funded, err = campaign.ProcessJournalEntries(ctx, w.Deps.Pool, pending); err != nil {
			return w.finishWalletError(ctx, corpID, err)
		}
		deposits, err = wallet.ProcessPendingDeposits(ctx, w.Deps.Pool, corpID)
		if err != nil {
			return w.finishWalletError(ctx, corpID, err)
		}
	}

	if _, err := w.Deps.Pool.Exec(ctx, `
        UPDATE corporation_wallet_tokens
        SET last_error = NULL, updated_at = now()
        WHERE corporation_id = $1`, corpID); err != nil {
		return err
	}

	w.Deps.Log.Debug().
		Int32("corporation", corpID).
		Int64("balances", balances).
		Int64("journal_entries", journal).
		Int("campaign_donations", funded).
		Int("wallet_deposits", deposits.Credited).
		Float64("wallet_deposit_isk", deposits.CreditedAmount).
		Msg("wallet synced")
	return nil
}

func (w *CorporationWalletWorker) withWalletRefreshRetry(
	ctx context.Context,
	token *wallet.Token,
	sync func(access string) (int64, error),
) (int64, error) {
	n, err := sync(token.AccessToken)
	if wallet.IsStatus(err, 401) {
		if refreshErr := w.refreshWalletAccess(ctx, token); refreshErr != nil {
			return n, refreshErr
		}
		n, err = sync(token.AccessToken)
	}
	if wallet.IsStatus(err, 401) || wallet.IsStatus(err, 403) {
		return n, &walletSyncBlockedError{message: err.Error()}
	}
	return n, err
}

func (w *CorporationWalletWorker) refreshWalletAccess(
	ctx context.Context,
	token *wallet.Token,
) error {
	refreshed, err := w.Deps.SSO.Refresh(ctx, token.RefreshToken)
	if errors.Is(err, sso.ErrPermanentlyDead) {
		const message = "Wallet authorization was revoked or expired"
		if disableErr := wallet.DisableToken(ctx, w.Deps.Pool, token.CorporationID, message); disableErr != nil {
			return errors.Join(err, disableErr)
		}
		return &walletSyncBlockedError{message: message}
	}
	if err != nil {
		return err
	}

	characterID, err := sso.CharacterIDFromAccessToken(refreshed.AccessToken)
	if err != nil {
		return err
	}
	if characterID != token.CharacterID {
		return &walletSyncBlockedError{
			message: "Refreshed wallet token belongs to a different character",
		}
	}

	scopes, err := sso.ScopesFromAccessToken(refreshed.AccessToken)
	if err != nil {
		return err
	}
	if !hasString(scopes, sso.ScopeCorporationWallet) {
		message := "Missing " + sso.ScopeCorporationWallet
		if _, updateErr := w.Deps.Pool.Exec(ctx, `
            UPDATE corporation_wallet_tokens
            SET disabled = true, scopes = $2, last_error = $3, updated_at = now()
            WHERE corporation_id = $1`,
			token.CorporationID, scopes, message); updateErr != nil {
			return updateErr
		}
		return &walletSyncBlockedError{
			message: "Wallet authorization no longer includes the corporation wallet scope",
		}
	}

	refreshToken := refreshed.RefreshToken
	if refreshToken == "" {
		refreshToken = token.RefreshToken
	}
	expiry := time.Now().UTC().Add(time.Duration(refreshed.ExpiresIn) * time.Second)
	if _, err := w.Deps.Pool.Exec(ctx, `
        UPDATE corporation_wallet_tokens SET
            access_token = $2,
            refresh_token = $3,
            token_expiry = $4,
            token_type = $5,
            scopes = $6,
            disabled = false,
            updated_at = now()
        WHERE corporation_id = $1`,
		token.CorporationID,
		refreshed.AccessToken,
		refreshToken,
		expiry,
		refreshed.TokenType,
		scopes,
	); err != nil {
		return err
	}

	token.AccessToken = refreshed.AccessToken
	token.RefreshToken = refreshToken
	token.Expiry = expiry
	return nil
}

func (w *CorporationWalletWorker) finishWalletError(
	ctx context.Context,
	corporationID int32,
	cause error,
) error {
	message := cause.Error()
	if len(message) > 1000 {
		message = message[:1000]
	}
	if _, err := w.Deps.Pool.Exec(ctx, `
        UPDATE corporation_wallet_tokens
        SET last_error = $2, updated_at = now()
        WHERE corporation_id = $1`, corporationID, message); err != nil {
		return errors.Join(cause, err)
	}

	var blocked *walletSyncBlockedError
	if errors.As(cause, &blocked) {
		return nil
	}
	return cause
}

func hasString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// cronCorporationWalletSync queues the wallets that have gone stale.
func (d *Deps) cronCorporationWalletSync(ctx context.Context) (string, error) {
	if d.Queue == nil {
		return "", errNeedsQueue("corporation_wallet_sync")
	}

	tokens, err := wallet.LoadTokens(ctx, d.Pool)
	if err != nil {
		return "", err
	}

	var args []river.JobArgs
	for _, t := range tokens {
		// The worker gates balances and journals independently. Dispatching
		// every enabled token also gives the reference processors their hourly
		// catch-up even when a sync timestamp was written slightly in the
		// future by clock skew.
		args = append(args, queue.CorporationWalletArgs{CorporationID: t.CorporationID})
	}
	if len(args) == 0 {
		return "", nil
	}

	n, err := queue.DispatchMany(ctx, d.Queue, args, queue.Live)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d of %d wallets queued", n, len(tokens)), nil
}

// cronEkWalletReservationExpiry releases reservations that were never captured.
//
// A reservation holds funds: the amount sits in ek_wallet_accounts.reserved_balance
// and is unavailable to spend until the reservation is captured or released.
// Expiring one therefore has to do two things together — return the held amount
// to the account and mark the reservation closed. Doing only the second, which
// is the obvious reading of "expire the reservation", leaves the balance
// reserved against something that will never happen, and the user's spendable
// funds shrink permanently with every expiry.
//
// Both halves run in one transaction per reservation, with the account row
// locked, because the invariant being maintained is that reserved_balance
// equals the sum of the active reservations. A concurrent capture on the same
// account would otherwise interleave and break it.
func (d *Deps) cronEkWalletReservationExpiry(ctx context.Context) (string, error) {
	const batch = 1000

	rows, err := d.Pool.Query(ctx, `
        SELECT id, character_id, amount FROM ek_wallet_reservations
        WHERE status = $1 AND expires_at IS NOT NULL AND expires_at <= now()
        ORDER BY expires_at, id
        LIMIT $2`, reservationActive, batch)
	if err != nil {
		return "", err
	}

	type expiring struct {
		id          int64
		characterID int32
		amount      string
	}
	var due []expiring
	for rows.Next() {
		var e expiring
		if err := rows.Scan(&e.id, &e.characterID, &e.amount); err != nil {
			rows.Close()
			return "", err
		}
		due = append(due, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(due) == 0 {
		return "", nil
	}

	var released int
	for _, e := range due {
		ok, err := d.expireReservation(ctx, e.id, e.characterID, e.amount)
		if err != nil {
			return "", err
		}
		if ok {
			released++
		}
	}

	if released == 0 {
		return "", nil
	}
	return fmt.Sprintf("%d holds released", released), nil
}

// Reservation states, mirroring EkWalletReservationStatus in the schema. Stored
// as smallints and never renumbered.
const (
	reservationActive  = 0
	reservationExpired = 3
)

// expireReservation returns one hold to its account and closes it.
//
// Reports false when the reservation was no longer expirable — captured or
// released by something else between the scan above and the lock here, which is
// an ordinary race rather than a failure.
func (d *Deps) expireReservation(ctx context.Context, id int64, characterID int32, amount string) (bool, error) {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	// Re-read under lock. The status may have changed since the scan, and
	// acting on the stale value would double-release the amount.
	var status int16
	err = tx.QueryRow(ctx, `
        SELECT status FROM ek_wallet_reservations
        WHERE id = $1 AND expires_at IS NOT NULL AND expires_at <= now()
        FOR UPDATE`, id).Scan(&status)
	if err != nil {
		// Gone, or no longer expired. Either way there is nothing to do.
		return false, nil
	}
	if status != reservationActive {
		return false, nil
	}

	// The account is locked in the same transaction so a concurrent capture
	// cannot interleave between the read and the decrement.
	var reserved string
	if err := tx.QueryRow(ctx,
		`SELECT reserved_balance FROM ek_wallet_accounts WHERE character_id = $1 FOR UPDATE`,
		characterID).Scan(&reserved); err != nil {
		return false, fmt.Errorf("lock wallet account %d: %w", characterID, err)
	}

	// Releasing more than is held would make the balance negative, which means
	// the two records already disagree. Refusing is right: silently clamping
	// would hide a real inconsistency in somebody's money.
	tag, err := tx.Exec(ctx, `
        UPDATE ek_wallet_accounts
        SET reserved_balance = reserved_balance - $2::numeric, updated_at = now()
        WHERE character_id = $1 AND reserved_balance >= $2::numeric`, characterID, amount)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, fmt.Errorf(
			"wallet account %d holds %s reserved but reservation %d claims %s — "+
				"refusing to release more than is held",
			characterID, reserved, id, amount)
	}

	if _, err := tx.Exec(ctx, `
        UPDATE ek_wallet_reservations
        SET status = $2, close_reason = 'Reservation expired', closed_at = now(), updated_at = now()
        WHERE id = $1`, id, reservationExpired); err != nil {
		return false, err
	}

	return true, tx.Commit(ctx)
}
