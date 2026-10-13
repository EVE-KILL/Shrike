package frontend

import (
	"net/url"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type adminWalletAuthorization struct {
	CharacterID   int64   `json:"authorized_character_id"`
	CharacterName string  `json:"authorized_character_name"`
	Disabled      bool    `json:"disabled"`
	BalanceSync   *string `json:"last_balance_sync"`
	JournalSync   *string `json:"last_journal_sync"`
	Error         *string `json:"last_error"`
}
type adminWalletBalance struct {
	Division int    `json:"division"`
	Balance  string `json:"balance"`
}
type adminWalletSettlement struct {
	CampaignID    string  `json:"campaign_id"`
	Name          string  `json:"campaign_name"`
	Status        int     `json:"pool_status"`
	Funded        string  `json:"funded_total"`
	Finalized     *string `json:"finalized_at"`
	Rank          *int    `json:"rank"`
	CharacterID   *int64  `json:"character_id"`
	CharacterName *string `json:"character_name"`
	Percentage    *int    `json:"payout_percentage"`
	Amount        *string `json:"payout_amount"`
	Claimed       *string `json:"claimed_at"`
	Paid          *string `json:"paid_at"`
}
type adminWalletReference struct {
	Division  int     `json:"division"`
	JournalID int64   `json:"journal_id"`
	Type      string  `json:"reference_type"`
	ID        string  `json:"reference_id"`
	Status    int     `json:"status"`
	Amount    *string `json:"amount"`
	Note      *string `json:"note"`
	Date      string  `json:"date"`
}
type adminWalletQuery struct{ Page, Division int }
type adminWalletData struct {
	api.PublicWalletData
	Authorization *adminWalletAuthorization `json:"authorization"`
	Balances      []adminWalletBalance      `json:"balances"`
	Settlements   []adminWalletSettlement   `json:"prizeSettlements"`
	References    []adminWalletReference    `json:"walletReferences"`
	Query         adminWalletQuery          `json:"-"`
	Failed        bool                      `json:"-"`
}
type adminWalletGroup struct {
	Pool    adminWalletSettlement
	Results []adminWalletSettlement
}

func adminWalletSelection(query url.Values) adminWalletQuery {
	result := adminWalletQuery{Page: 1}
	if page, err := strconv.Atoi(query.Get("page")); err == nil && page > 0 {
		result.Page = min(page, 1000)
	}
	if division, err := strconv.Atoi(query.Get("division")); err == nil && division >= 1 && division <= 7 {
		result.Division = division
	}
	return result
}
func (query adminWalletQuery) values() url.Values {
	values := url.Values{"page": {strconv.Itoa(query.Page)}}
	if query.Division > 0 {
		values.Set("division", strconv.Itoa(query.Division))
	}
	return values
}
func (data adminWalletData) divisions() []adminWalletBalance {
	result := make([]adminWalletBalance, 7)
	for i := range result {
		result[i] = adminWalletBalance{Division: i + 1, Balance: "0.00"}
	}
	for _, wallet := range data.Balances {
		if wallet.Division >= 1 && wallet.Division <= 7 {
			result[wallet.Division-1] = wallet
		}
	}
	return result
}
func (data adminWalletData) groups() []adminWalletGroup {
	var result []adminWalletGroup
	indices := map[string]int{}
	for _, row := range data.Settlements {
		index, exists := indices[row.CampaignID]
		if !exists {
			index = len(result)
			indices[row.CampaignID] = index
			result = append(result, adminWalletGroup{Pool: row})
		}
		if row.Rank != nil {
			result[index].Results = append(result[index].Results, row)
		}
	}
	return result
}
func adminWalletDate(raw *string) string {
	if raw == nil || *raw == "" {
		return "Never"
	}
	value, err := time.Parse(time.RFC3339Nano, *raw)
	if err != nil {
		return "Never"
	}
	return value.Local().Format("2 Jan 2006, 15:04")
}
func adminWalletRaw(raw *string) string {
	if raw == nil {
		return ""
	}
	return *raw
}
func adminWalletInt(raw *int) string {
	if raw == nil {
		return ""
	}
	return strconv.Itoa(*raw)
}
func adminWalletReferenceStatus(status int) string {
	if status >= 0 && status < 4 {
		return []string{"Matched", "Unmatched", "Late", "Invalid"}[status]
	}
	return "Unknown"
}
func adminWalletReferenceTone(status int) string {
	if status == 0 {
		return "text-green-400"
	}
	if status == 2 {
		return "text-yellow-400"
	}
	return "text-red-400"
}
func walletCorporationID(id int64) string {
	if id == 0 {
		return "98779905"
	}
	return warID(id)
}
func adminWalletAmountTone(raw *string) string {
	switch walletAmountTone(raw) {
	case "positive":
		return "text-green-400"
	case "negative":
		return "text-red-400"
	}
	return "text-gray-500"
}
func adminWalletBanner(query url.Values) (string, string) {
	status := ""
	if len(query["wallet_auth"]) == 1 {
		status = query.Get("wallet_auth")
	}
	message := ""
	tone := ""
	if status == "ok" {
		tone = "border-green-500/20 bg-green-500/10 text-green-300"
		message = "Wallet authorization saved. The first journal import has been queued."
	}
	if status == "error" {
		tone = "border-red-500/20 bg-red-500/10 text-red-300"
		message = "Wallet authorization failed."
	}
	if tone != "" && len(query["message"]) == 1 {
		message = query.Get("message")
	}
	return message, tone
}
