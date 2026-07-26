package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

const walletCampaignContributionType = 3

var walletAmountPattern = regexp.MustCompile(`^\s*(\d+)(?:\.(\d{1,2}))?\s*$`)

type walletOperationError struct {
	Code    string
	Message string
}

func (e *walletOperationError) Error() string { return e.Message }

type walletContributionResult struct {
	Replayed bool
	Balance  map[string]any
}

func normalizeWalletAmount(value any) (string, error) {
	var raw string
	switch typed := value.(type) {
	case json.Number:
		raw = typed.String()
	case string:
		raw = typed
	case float64:
		raw = fmt.Sprintf("%.15g", typed)
	case float32:
		raw = fmt.Sprintf("%.8g", typed)
	default:
		return "", &walletOperationError{
			Code:    "invalid_amount",
			Message: "EK Wallet amount must be a positive decimal with at most two decimal places",
		}
	}
	match := walletAmountPattern.FindStringSubmatch(raw)
	if match == nil {
		return "", &walletOperationError{
			Code:    "invalid_amount",
			Message: "EK Wallet amount must be a positive decimal with at most two decimal places",
		}
	}
	whole := strings.TrimLeft(match[1], "0")
	if whole == "" {
		whole = "0"
	}
	if len(whole) > 28 {
		return "", &walletOperationError{
			Code:    "invalid_amount",
			Message: "EK Wallet amount exceeds the supported balance size",
		}
	}
	fraction := match[2]
	if len(fraction) == 0 {
		fraction = "00"
	} else if len(fraction) == 1 {
		fraction += "0"
	}
	amount := whole + "." + fraction
	cents, err := parseWalletCentsBig(amount)
	if err != nil || cents.Sign() <= 0 {
		return "", &walletOperationError{
			Code:    "invalid_amount",
			Message: "EK Wallet amount must be greater than zero",
		}
	}
	return formatWalletCentsBig(cents), nil
}

func contributeCampaignWallet(
	ctx context.Context,
	tx pgx.Tx,
	characterID int32,
	campaignID, amount, description, externalReference string,
	metadata map[string]any,
) (walletContributionResult, error) {
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		externalReference,
	); err != nil {
		return walletContributionResult{}, err
	}

	var existingCharacter int32
	var existingType int16
	var existingAmount string
	var existingCampaign *string
	err := tx.QueryRow(ctx, `
		SELECT character_id, type, amount::text, campaign_id
		FROM ek_wallet_transactions
		WHERE external_reference = $1
		LIMIT 1`,
		externalReference,
	).Scan(
		&existingCharacter,
		&existingType,
		&existingAmount,
		&existingCampaign,
	)
	if err == nil {
		if existingCharacter != characterID ||
			existingType != walletCampaignContributionType ||
			existingAmount != "-"+amount ||
			existingCampaign == nil ||
			*existingCampaign != campaignID {
			return walletContributionResult{}, &walletOperationError{
				Code:    "idempotency_conflict",
				Message: "EK Wallet external reference was already used for a different operation",
			}
		}
		balance, balanceErr := readWalletBalanceTx(ctx, tx, characterID)
		return walletContributionResult{
			Replayed: true, Balance: balance,
		}, balanceErr
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return walletContributionResult{}, err
	}

	var reservedReference bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM ek_wallet_reservations
		  WHERE external_reference = $1
		)`, externalReference).Scan(&reservedReference); err != nil {
		return walletContributionResult{}, err
	}
	if reservedReference {
		return walletContributionResult{}, &walletOperationError{
			Code:    "idempotency_conflict",
			Message: "EK Wallet reference was already used for a reservation",
		}
	}

	var poolStatus int16
	var endTimeValid bool
	err = tx.QueryRow(ctx, `
		SELECT pool.status,
		       campaign.end_time IS NOT NULL
		         AND campaign.end_time > now() AS funding_open
		FROM campaign_prize_pools pool
		JOIN campaigns campaign
		  ON campaign.campaign_id = pool.campaign_id
		WHERE pool.campaign_id = $1
		LIMIT 1
		FOR UPDATE OF pool`, campaignID).Scan(&poolStatus, &endTimeValid)
	if errors.Is(err, pgx.ErrNoRows) ||
		err == nil &&
			(poolStatus != campaignPrizeFunding || !endTimeValid) {
		return walletContributionResult{}, &walletOperationError{
			Code:    "campaign_not_fundable",
			Message: "Campaign prize funding is not open",
		}
	}
	if err != nil {
		return walletContributionResult{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO ek_wallet_accounts (character_id)
		VALUES ($1)
		ON CONFLICT (character_id) DO NOTHING`, characterID); err != nil {
		return walletContributionResult{}, err
	}
	var total, reserved, available string
	if err := tx.QueryRow(ctx, `
		SELECT balance::text, reserved_balance::text,
		       (balance - reserved_balance)::text
		FROM ek_wallet_accounts
		WHERE character_id = $1
		FOR UPDATE`, characterID).Scan(&total, &reserved, &available); err != nil {
		return walletContributionResult{}, err
	}
	availableCents, err := parseWalletCentsBig(available)
	if err != nil {
		return walletContributionResult{}, err
	}
	amountCents, err := parseWalletCentsBig(amount)
	if err != nil {
		return walletContributionResult{}, err
	}
	if availableCents.Cmp(amountCents) < 0 {
		return walletContributionResult{}, &walletOperationError{
			Code:    "insufficient_funds",
			Message: "EK Wallet does not have enough available funds",
		}
	}
	totalCents, err := parseWalletCentsBig(total)
	if err != nil {
		return walletContributionResult{}, err
	}
	balanceAfter := new(big.Int).Sub(totalCents, amountCents)
	metadata["campaignId"] = campaignID
	metadataJSON, err := json.Marshal(metadata)
	if err != nil || len(metadataJSON) > 32768 {
		return walletContributionResult{}, &walletOperationError{
			Code:    "invalid_metadata",
			Message: "EK Wallet metadata must be JSON serializable and no larger than 32 KiB",
		}
	}
	command, err := tx.Exec(ctx, `
		WITH updated AS (
		  UPDATE ek_wallet_accounts
		  SET balance = $2::numeric, updated_at = now()
		  WHERE character_id = $1
		  RETURNING balance
		)
		INSERT INTO ek_wallet_transactions (
		  character_id, type, amount, balance_after, description,
		  external_reference, campaign_id, metadata
		)
		SELECT $1, $3, $4::numeric, balance, $5, $6, $7, $8::jsonb
		FROM updated`,
		characterID,
		formatWalletCentsBig(balanceAfter),
		walletCampaignContributionType,
		"-"+amount,
		description,
		externalReference,
		campaignID,
		metadataJSON,
	)
	if err != nil {
		return walletContributionResult{}, err
	}
	if command.RowsAffected() != 1 {
		return walletContributionResult{}, fmt.Errorf(
			"could not charge EK Wallet",
		)
	}
	command, err = tx.Exec(ctx, `
		UPDATE campaign_prize_pools
		SET funded_total = funded_total + $2::numeric,
		    rules_locked_at = COALESCE(rules_locked_at, now()),
		    updated_at = now()
		WHERE campaign_id = $1 AND status = $3`,
		campaignID, amount, campaignPrizeFunding,
	)
	if err != nil {
		return walletContributionResult{}, err
	}
	if command.RowsAffected() != 1 {
		return walletContributionResult{}, &walletOperationError{
			Code:    "campaign_not_fundable",
			Message: "Campaign prize funding closed before the contribution completed",
		}
	}
	balance, err := readWalletBalanceTx(ctx, tx, characterID)
	return walletContributionResult{Balance: balance}, err
}

func readWalletBalanceTx(
	ctx context.Context,
	tx pgx.Tx,
	characterID int32,
) (map[string]any, error) {
	var total, reserved, available string
	err := tx.QueryRow(ctx, `
		SELECT balance::text, reserved_balance::text,
		       (balance - reserved_balance)::text
		FROM ek_wallet_accounts
		WHERE character_id = $1
		LIMIT 1`, characterID).Scan(&total, &reserved, &available)
	if errors.Is(err, pgx.ErrNoRows) {
		return walletBalance(nil), nil
	}
	if err != nil {
		return nil, err
	}
	return walletBalance(map[string]any{
		"total_balance":     total,
		"reserved_balance":  reserved,
		"available_balance": available,
	}), nil
}

func parseWalletCentsBig(value string) (*big.Int, error) {
	value = strings.TrimSpace(value)
	negative := strings.HasPrefix(value, "-")
	if negative {
		value = strings.TrimPrefix(value, "-")
	}
	whole, fraction, _ := strings.Cut(value, ".")
	if whole == "" {
		whole = "0"
	}
	if len(fraction) > 2 {
		fraction = fraction[:2]
	}
	fraction += strings.Repeat("0", 2-len(fraction))
	combined := strings.TrimLeft(whole+fraction, "0")
	if combined == "" {
		combined = "0"
	}
	cents := new(big.Int)
	if _, ok := cents.SetString(combined, 10); !ok {
		return nil, fmt.Errorf("invalid wallet amount %q", value)
	}
	if negative {
		cents.Neg(cents)
	}
	return cents, nil
}

func formatWalletCentsBig(cents *big.Int) string {
	if cents == nil {
		return "0.00"
	}
	negative := cents.Sign() < 0
	absolute := new(big.Int).Abs(new(big.Int).Set(cents))
	whole := new(big.Int).Quo(absolute, big.NewInt(100))
	fraction := new(big.Int).Mod(absolute, big.NewInt(100))
	return fmt.Sprintf(
		"%s%s.%02d",
		map[bool]string{true: "-", false: ""}[negative],
		whole.String(),
		fraction.Int64(),
	)
}

func parseWalletCents(value string) (int64, error) {
	cents, err := parseWalletCentsBig(value)
	if err != nil {
		return 0, err
	}
	if !cents.IsInt64() {
		return 0, fmt.Errorf("wallet amount is too large")
	}
	return cents.Int64(), nil
}

func formatWalletCents(cents int64) string {
	return formatWalletCentsBig(big.NewInt(cents))
}
