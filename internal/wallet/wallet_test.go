package wallet

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestOptionalJournalMoneyAndReferencesStayNull(t *testing.T) {
	var entry JournalEntry
	if err := json.Unmarshal([]byte(`{
		"id": 1,
		"date": "2026-07-26T12:00:00Z",
		"ref_type": "player_donation",
		"description": "donation"
	}`), &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Amount != nil || entry.BalanceAfter != nil || entry.Tax != nil {
		t.Fatalf("missing optional money decoded as amount=%v balance=%v tax=%v",
			entry.Amount, entry.BalanceAfter, entry.Tax)
	}
	if entry.FirstPartyID != nil || entry.SecondPartyID != nil ||
		entry.ContextID != nil || entry.TaxReceiverID != nil {
		t.Fatalf("missing optional ids decoded as non-null: %+v", entry)
	}
}

func TestWalletResponseErrorsPreserveStatus(t *testing.T) {
	err := checkAuth(401, "balances")
	if !errors.Is(err, ErrUnauthorized) || !IsStatus(err, 401) || IsStatus(err, 403) {
		t.Fatalf("401 error classification = %v", err)
	}
	err = checkAuth(403, "division 1 journal")
	if !errors.Is(err, ErrUnauthorized) || !IsStatus(err, 403) || IsStatus(err, 401) {
		t.Fatalf("403 error classification = %v", err)
	}
}
