package wallet

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// DepositsEnabledConfigKey is the launch cutoff. Historical donations must
	// not suddenly become wallet credit when this processor is deployed.
	DepositsEnabledConfigKey = "ek-wallet:deposits-enabled-at"

	eveKillCorporationID = 98779905
	depositBatchLimit    = 5_000
	transactionDeposit   = 0

	referenceMatched   = 0
	referenceUnmatched = 1
	referenceInvalid   = 3
)

// DepositResult reports one catch-up pass.
type DepositResult struct {
	Credited       int
	CreditedAmount float64
	Unmatched      int
	Invalid        int
}

type depositCandidate struct {
	CorporationID  int32
	Division       int16
	JournalID      int64
	Date           time.Time
	RefType        string
	Amount         string
	FirstPartyID   sql.NullInt64
	Reason         sql.NullString
	ReferenceState sql.NullInt16
	CharacterID    sql.NullInt64
	CharacterName  sql.NullString
}

// ProcessPendingDeposits credits eligible EVE-KILL player donations to the
// authoritative sender's per-character wallet.
func ProcessPendingDeposits(
	ctx context.Context,
	pool *pgxpool.Pool,
	corporationID int32,
) (DepositResult, error) {
	var out DepositResult
	if corporationID != eveKillCorporationID {
		return out, nil
	}

	var rawCutoff string
	err := pool.QueryRow(ctx,
		`SELECT value FROM config WHERE key = $1 LIMIT 1`,
		DepositsEnabledConfigKey,
	).Scan(&rawCutoff)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	cutoff, err := time.Parse(time.RFC3339Nano, rawCutoff)
	if err != nil {
		return out, nil
	}

	rows, err := pool.Query(ctx, `
        SELECT journal.corporation_id,
               journal.division,
               journal.journal_id,
               journal.date,
               journal.ref_type,
               journal.amount::text,
               journal.first_party_id,
               journal.reason,
               reference.status,
               "user".character_id,
               "user".character_name
        FROM corporation_wallet_journal journal
        LEFT JOIN wallet_journal_references reference
          ON reference.corporation_id = journal.corporation_id
         AND reference.division = journal.division
         AND reference.journal_id = journal.journal_id
        LEFT JOIN users "user"
          ON "user".character_id = journal.first_party_id
        WHERE journal.corporation_id = $1
          AND journal.date >= $2
          AND journal.ref_type = 'player_donation'
          AND journal.amount > 0
          AND lower(trim(coalesce(journal.reason, ''))) NOT LIKE 'campaign:%'
          AND (
              reference.journal_id IS NULL
              OR (
                  reference.reference_type = 'ek_wallet'
                  AND reference.status = $3
              )
          )
        ORDER BY journal.date, journal.journal_id
        LIMIT $4`,
		corporationID,
		cutoff,
		referenceUnmatched,
		depositBatchLimit,
	)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	var candidates []depositCandidate
	for rows.Next() {
		var candidate depositCandidate
		if err := rows.Scan(
			&candidate.CorporationID,
			&candidate.Division,
			&candidate.JournalID,
			&candidate.Date,
			&candidate.RefType,
			&candidate.Amount,
			&candidate.FirstPartyID,
			&candidate.Reason,
			&candidate.ReferenceState,
			&candidate.CharacterID,
			&candidate.CharacterName,
		); err != nil {
			return out, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	for _, candidate := range candidates {
		if reservedDepositReason(candidate.Reason.String) {
			continue
		}

		switch {
		case !candidate.FirstPartyID.Valid || candidate.FirstPartyID.Int64 <= 0:
			inserted, err := classifyDeposit(
				ctx,
				pool,
				candidate,
				referenceInvalid,
				"ESI did not identify a valid sending character",
			)
			if err != nil {
				return out, err
			}
			if inserted {
				out.Invalid++
			}

		case !candidate.CharacterID.Valid:
			inserted, err := classifyDeposit(
				ctx,
				pool,
				candidate,
				referenceUnmatched,
				"The sending character has not signed in to EVE-KILL yet",
			)
			if err != nil {
				return out, err
			}
			if inserted {
				out.Unmatched++
			}

		default:
			credited, err := creditDeposit(ctx, pool, candidate)
			if err != nil {
				return out, err
			}
			if credited {
				out.Credited++
				amount, _ := strconv.ParseFloat(candidate.Amount, 64)
				out.CreditedAmount += amount
			}
		}
	}

	return out, nil
}

func reservedDepositReason(reason string) bool {
	reason = strings.ToLower(strings.TrimSpace(reason))
	return strings.HasPrefix(reason, "campaign:") ||
		strings.HasPrefix(reason, "sponsor:")
}

func classifyDeposit(
	ctx context.Context,
	pool *pgxpool.Pool,
	candidate depositCandidate,
	status int16,
	note string,
) (bool, error) {
	if candidate.ReferenceState.Valid {
		return false, nil
	}

	referenceID := "unknown"
	if candidate.FirstPartyID.Valid {
		referenceID = strconv.FormatInt(candidate.FirstPartyID.Int64, 10)
	}
	var inserted int64
	err := pool.QueryRow(ctx, `
        INSERT INTO wallet_journal_references (
            corporation_id, division, journal_id,
            reference_type, reference_id, status, amount, note)
        VALUES ($1,$2,$3,'ek_wallet',$4,$5,$6::numeric,$7)
        ON CONFLICT DO NOTHING
        RETURNING journal_id`,
		candidate.CorporationID,
		candidate.Division,
		candidate.JournalID,
		referenceID,
		status,
		candidate.Amount,
		note,
	).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func creditDeposit(
	ctx context.Context,
	pool *pgxpool.Pool,
	candidate depositCandidate,
) (bool, error) {
	if !candidate.CharacterID.Valid ||
		!candidate.CharacterName.Valid ||
		candidate.CharacterName.String == "" ||
		candidate.Amount == "" {
		return false, nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var referenced int64
	if candidate.ReferenceState.Valid &&
		candidate.ReferenceState.Int16 == referenceUnmatched {
		err = tx.QueryRow(ctx, `
            UPDATE wallet_journal_references
            SET status = $4, note = NULL, updated_at = now()
            WHERE corporation_id = $1
              AND division = $2
              AND journal_id = $3
              AND reference_type = 'ek_wallet'
              AND status = $5
            RETURNING journal_id`,
			candidate.CorporationID,
			candidate.Division,
			candidate.JournalID,
			referenceMatched,
			referenceUnmatched,
		).Scan(&referenced)
	} else {
		err = tx.QueryRow(ctx, `
            INSERT INTO wallet_journal_references (
                corporation_id, division, journal_id,
                reference_type, reference_id, status, amount, note)
            VALUES ($1,$2,$3,'ek_wallet',$4,$5,$6::numeric,NULL)
            ON CONFLICT DO NOTHING
            RETURNING journal_id`,
			candidate.CorporationID,
			candidate.Division,
			candidate.JournalID,
			strconv.FormatInt(candidate.CharacterID.Int64, 10),
			referenceMatched,
			candidate.Amount,
		).Scan(&referenced)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	characterID := int32(candidate.CharacterID.Int64)
	if _, err := tx.Exec(ctx, `
        INSERT INTO ek_wallet_accounts (character_id)
        VALUES ($1) ON CONFLICT DO NOTHING`, characterID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
        SELECT character_id FROM ek_wallet_accounts
        WHERE character_id = $1 FOR UPDATE`, characterID); err != nil {
		return false, err
	}

	var balance string
	if err := tx.QueryRow(ctx, `
        UPDATE ek_wallet_accounts
        SET balance = balance + $2::numeric, updated_at = now()
        WHERE character_id = $1
        RETURNING balance::text`,
		characterID,
		candidate.Amount,
	).Scan(&balance); err != nil {
		return false, fmt.Errorf("update EK Wallet balance: %w", err)
	}

	metadata, err := json.Marshal(map[string]any{
		"senderCharacterId": characterID,
		"reason":            nullableReason(candidate.Reason),
	})
	if err != nil {
		return false, err
	}
	externalReference := fmt.Sprintf(
		"corporation-wallet:%d:%d:%d",
		candidate.CorporationID,
		candidate.Division,
		candidate.JournalID,
	)
	if _, err := tx.Exec(ctx, `
        INSERT INTO ek_wallet_transactions (
            character_id, type, amount, balance_after,
            description, external_reference,
            corporation_id, division, journal_id, metadata)
        VALUES ($1,$2,$3::numeric,$4::numeric,$5,$6,$7,$8,$9,$10::jsonb)`,
		characterID,
		transactionDeposit,
		candidate.Amount,
		balance,
		"Deposit from "+candidate.CharacterName.String,
		externalReference,
		candidate.CorporationID,
		candidate.Division,
		candidate.JournalID,
		metadata,
	); err != nil {
		return false, err
	}

	return true, tx.Commit(ctx)
}

func nullableReason(reason sql.NullString) any {
	if !reason.Valid {
		return nil
	}
	return reason.String
}
