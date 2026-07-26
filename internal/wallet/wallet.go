// Package wallet syncs corporation wallet balances and journals from ESI.
//
// A corporation has seven wallet divisions and each keeps its own paginated
// journal, so one sync is up to seven paginated walks plus a balances call.
// That is expensive enough that it is gated on staleness rather than run
// unconditionally.
//
// The journal is append-only from our side: entries are inserted and never
// updated, because a journal entry is a historical fact that CCP does not
// revise. That makes the whole sync idempotent — re-reading a page inserts
// nothing new.
package wallet

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Divisions are the seven wallet divisions every corporation has.
var Divisions = []int{1, 2, 3, 4, 5, 6, 7}

// SyncInterval is how often a wallet is refreshed.
//
// The tolerance matters and is not padding for its own sake: the dispatching
// cron fires on wall-clock hour boundaries, so a sync that finished at HH:01
// would still look fresh at HH+1:00 and the wallet would refresh every second
// hour instead of every hour. Subtracting five minutes closes that gap.
const (
	SyncInterval  = time.Hour
	SyncTolerance = 5 * time.Minute
	StaleAfter    = SyncInterval - SyncTolerance
)

// JournalPageLimit bounds a division's walk. ESI reports the page count, so
// this is a guard against a paging bug rather than the normal exit.
const JournalPageLimit = 200

// Token is a stored corporation wallet authorisation.
type Token struct {
	CorporationID   int32
	CharacterID     int32
	AccessToken     string
	RefreshToken    string
	Expiry          time.Time
	Disabled        bool
	LastBalanceSync *time.Time
	LastJournalSync *time.Time
}

// Due reports whether a sync is worth doing.
func Due(last *time.Time, now time.Time) bool {
	return last == nil || now.Sub(*last) >= StaleAfter
}

// SyncResult reports one wallet sync.
type SyncResult struct {
	Balances int64 `json:"balances"`
	Journal  int64 `json:"journal"`
}

// Balance is one division's balance from ESI.
type Balance struct {
	Division int     `json:"division"`
	Balance  float64 `json:"balance"`
}

// JournalEntry is one wallet journal line.
type JournalEntry struct {
	ID            int64    `json:"id"`
	Date          string   `json:"date"`
	RefType       string   `json:"ref_type"`
	Description   string   `json:"description"`
	Amount        *float64 `json:"amount"`
	BalanceAfter  *float64 `json:"balance"`
	FirstPartyID  *int64   `json:"first_party_id"`
	SecondPartyID *int64   `json:"second_party_id"`
	ContextID     *int64   `json:"context_id"`
	ContextIDType *string  `json:"context_id_type"`
	Reason        *string  `json:"reason"`
	Tax           *float64 `json:"tax"`
	TaxReceiverID *int64   `json:"tax_receiver_id"`
}

// LoadTokens returns the corporation wallet authorisations worth syncing.
func LoadTokens(ctx context.Context, pool *pgxpool.Pool) ([]Token, error) {
	rows, err := pool.Query(ctx, `
        SELECT corporation_id, coalesce(authorized_character_id, 0),
               coalesce(access_token, ''), coalesce(refresh_token, ''),
               coalesce(token_expiry, 'epoch'::timestamptz),
               coalesce(disabled, false), last_balance_sync, last_journal_sync
        FROM corporation_wallet_tokens
        WHERE disabled IS NOT TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Token
	for rows.Next() {
		var t Token
		if err := rows.Scan(&t.CorporationID, &t.CharacterID, &t.AccessToken,
			&t.RefreshToken, &t.Expiry, &t.Disabled,
			&t.LastBalanceSync, &t.LastJournalSync); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SyncBalances writes the current balance of each division.
//
// Balances are current state rather than history, so this overwrites. The
// journal below is the historical record.
func SyncBalances(ctx context.Context, pool *pgxpool.Pool, client *esi.Client, corporationID int32, accessToken string) (int64, error) {
	res, err := esi.GetAuthenticated[[]Balance](ctx, client,
		fmt.Sprintf("/latest/corporations/%d/wallets/", corporationID), accessToken)
	if err != nil {
		return 0, err
	}
	if err := checkAuth(res.Status, "balances"); err != nil {
		return 0, err
	}
	if !res.OK() || res.Data == nil {
		return 0, fmt.Errorf("ESI returned %d for corporation %d wallets", res.Status, corporationID)
	}

	now := time.Now().UTC()
	var written int64
	for _, b := range *res.Data {
		if b.Division < Divisions[0] || b.Division > Divisions[len(Divisions)-1] {
			continue
		}
		tag, err := pool.Exec(ctx, `
            INSERT INTO corporation_wallet_balances (corporation_id, division, balance, updated_at)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (corporation_id, division) DO UPDATE SET
                balance = EXCLUDED.balance,
                updated_at = EXCLUDED.updated_at`,
			corporationID, b.Division, b.Balance, now)
		if err != nil {
			return written, fmt.Errorf("store balance for division %d: %w", b.Division, err)
		}
		written += tag.RowsAffected()
	}

	if _, err := pool.Exec(ctx, `
        UPDATE corporation_wallet_tokens SET last_balance_sync = $2, updated_at = $2
        WHERE corporation_id = $1`, corporationID, now); err != nil {
		return written, err
	}
	return written, nil
}

// SyncJournal walks every division's journal and stores what is new.
//
// ON CONFLICT DO NOTHING rather than an update: a journal entry is a historical
// fact CCP does not revise, so re-reading a page must be a no-op. That is what
// makes an interrupted sync safe to simply run again.
func SyncJournal(ctx context.Context, pool *pgxpool.Pool, client *esi.Client, corporationID int32, accessToken string) (int64, error) {
	var written int64
	now := time.Now().UTC()

	for _, division := range Divisions {
		pages := 1
		for page := 1; page <= pages && page <= JournalPageLimit; page++ {
			path := fmt.Sprintf("/latest/corporations/%d/wallets/%d/journal/?page=%d",
				corporationID, division, page)

			res, err := esi.GetAuthenticated[[]JournalEntry](ctx, client, path, accessToken)
			if err != nil {
				return written, err
			}
			if err := checkAuth(res.Status, fmt.Sprintf("division %d journal", division)); err != nil {
				return written, err
			}
			// A division that has never been used answers 404.
			if res.Status == 404 {
				break
			}
			if !res.OK() || res.Data == nil {
				return written, fmt.Errorf("ESI returned %d for %s", res.Status, path)
			}
			if res.Pages > 0 {
				pages = res.Pages
			}

			for _, e := range *res.Data {
				tag, err := pool.Exec(ctx, `
                    INSERT INTO corporation_wallet_journal (
                        corporation_id, division, journal_id, date, ref_type, description,
                        amount, balance, first_party_id, second_party_id,
                        context_id, context_id_type, reason, tax, tax_receiver_id, created_at
                    ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,now())
                    ON CONFLICT DO NOTHING`,
					corporationID, division, e.ID, parseTime(e.Date), e.RefType,
					e.Description, e.Amount, e.BalanceAfter,
					e.FirstPartyID, e.SecondPartyID,
					e.ContextID, e.ContextIDType, e.Reason,
					e.Tax, e.TaxReceiverID)
				if err != nil {
					return written, fmt.Errorf("store journal entry %d: %w", e.ID, err)
				}
				written += tag.RowsAffected()
			}
		}
	}

	if _, err := pool.Exec(ctx, `
        UPDATE corporation_wallet_tokens SET last_journal_sync = $2, updated_at = $2
        WHERE corporation_id = $1`, corporationID, now); err != nil {
		return written, err
	}
	return written, nil
}

// ErrUnauthorized means the wallet authorisation no longer works.
//
// Separated because the response is different from an ordinary failure: the
// character lost the Accountant role or left the corporation, and no amount of
// retrying will restore it. The caller disables the token.
var ErrUnauthorized = fmt.Errorf("wallet authorization rejected")

// ResponseError preserves the ESI status so the worker can refresh-and-retry a
// 401 without treating a 403 role failure as a dead SSO grant.
type ResponseError struct {
	Status int
	What   string
}

func (e *ResponseError) Error() string {
	if e.Status == 403 {
		return fmt.Sprintf(
			"ESI denied %s; the character must still belong to the corporation and have Accountant or Junior Accountant",
			e.What,
		)
	}
	return fmt.Sprintf("ESI rejected the wallet authorization while fetching %s", e.What)
}

func (e *ResponseError) Unwrap() error { return ErrUnauthorized }

// IsStatus reports whether an error came from one ESI status.
func IsStatus(err error, status int) bool {
	var response *ResponseError
	return errors.As(err, &response) && response.Status == status
}

func checkAuth(status int, what string) error {
	if status == 401 || status == 403 {
		return &ResponseError{Status: status, What: what}
	}
	return nil
}

// DisableToken stops a corporation's wallet being synced and records why.
func DisableToken(ctx context.Context, pool *pgxpool.Pool, corporationID int32, reason string) error {
	_, err := pool.Exec(ctx, `
        UPDATE corporation_wallet_tokens
        SET disabled = true, last_error = $2, updated_at = now()
        WHERE corporation_id = $1`, corporationID, reason)
	return err
}

func parseTime(s string) any {
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return nil
}
