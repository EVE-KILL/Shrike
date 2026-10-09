package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// PublicWalletData is the public wallet API contract rendered by the Go page.
// The reader delegates to the same handler as /api/wallet, so pagination,
// division filtering, and journal ordering stay consistent with API clients.
type PublicWalletData struct {
	Corporation struct {
		ID     int64  `json:"corporation_id"`
		Name   string `json:"name"`
		Ticker string `json:"ticker"`
	} `json:"corporation"`
	TotalBalance string              `json:"totalBalance"`
	LastSynced   *string             `json:"lastSynced"`
	Journal      []PublicWalletEntry `json:"journal"`
	Page         int                 `json:"page"`
	Division     *int                `json:"division"`
	HasMore      bool                `json:"hasMore"`
	PageSize     int                 `json:"pageSize"`
}

type PublicWalletEntry struct {
	Division      int     `json:"division"`
	JournalID     int64   `json:"journal_id"`
	Date          string  `json:"date"`
	RefType       string  `json:"ref_type"`
	Description   string  `json:"description"`
	Amount        *string `json:"amount"`
	Balance       *string `json:"balance"`
	FirstPartyID  *int64  `json:"first_party_id"`
	SecondPartyID *int64  `json:"second_party_id"`
	Reason        *string `json:"reason"`
}

func (r *HomepageReader) PublicWallet(ctx context.Context, page, division int) (PublicWalletData, error) {
	values := url.Values{"page": {strconv.Itoa(page)}}
	if division >= 1 && division <= 7 {
		values.Set("division", strconv.Itoa(division))
	}
	payload, err := (&walletService{db: r.db}).publicWalletHandler()(ctx, &legacyRequest{Query: values})
	if err != nil {
		return PublicWalletData{}, err
	}
	encoded, err := json.Marshal(payload.Body)
	if err != nil {
		return PublicWalletData{}, fmt.Errorf("encode public wallet: %w", err)
	}
	var result PublicWalletData
	if err := json.Unmarshal(encoded, &result); err != nil {
		return PublicWalletData{}, fmt.Errorf("decode public wallet: %w", err)
	}
	return result, nil
}
