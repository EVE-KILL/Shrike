package frontend

import "strconv"

type settingsWalletData struct {
	Character struct {
		ID   int64  `json:"character_id"`
		Name string `json:"character_name"`
	} `json:"character"`
	Corporation struct {
		ID     int64  `json:"corporation_id"`
		Name   string `json:"name"`
		Ticker string `json:"ticker"`
	} `json:"corporation"`
	Total        string  `json:"totalBalance"`
	Reserved     string  `json:"reservedBalance"`
	Available    string  `json:"availableBalance"`
	LastSynced   *string `json:"lastSynced"`
	Transactions []struct {
		ID          int64   `json:"id"`
		Type        int     `json:"type"`
		Amount      string  `json:"amount"`
		Balance     string  `json:"balance_after"`
		Description string  `json:"description"`
		CampaignID  *string `json:"campaign_id"`
		CreatedAt   string  `json:"created_at"`
	} `json:"transactions"`
	Reservations []struct {
		Reference   string  `json:"external_reference"`
		Amount      string  `json:"amount"`
		Description string  `json:"description"`
		ExpiresAt   *string `json:"expires_at"`
	} `json:"reservations"`
	Page    int  `json:"page"`
	HasMore bool `json:"hasMore"`
}

func settingsIsk(value string) string { return walletIsk(&value) }
func settingsString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func settingsPositive(value string) bool {
	number, _ := strconv.ParseFloat(value, 64)
	return number > 0
}
func settingsTransactionLabel(kind int) string {
	labels := []string{"Deposit", "Purchase", "Refund", "Campaign contribution", "Campaign prize", "Adjustment"}
	if kind < 0 || kind >= len(labels) {
		return "Transaction"
	}
	return labels[kind]
}
