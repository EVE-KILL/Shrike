package frontend

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

type testWalletReader struct {
	page, division int
}

func (r *testWalletReader) PublicWallet(_ context.Context, page, division int) (api.PublicWalletData, error) {
	r.page, r.division = page, division
	data := api.PublicWalletData{Page: page, PageSize: 50, TotalBalance: "250000000.00"}
	data.Corporation.ID = 98779905
	data.Corporation.Name = "EVE-KILL.com"
	return data, nil
}

func TestPublicWalletPageAndDivisionFragment(t *testing.T) {
	reader := &testWalletReader{}
	handler := New(Options{Wallet: reader})
	for _, tt := range []struct {
		path           string
		page, division int
		marker         string
	}{
		{"/wallet", 1, 0, "250,000,000.00 ISK"},
		{"/wallet?division=2", 1, 2, "Division 2"},
		{"/_fragments/wallet?division=3&page=2", 2, 3, "No wallet journal entries"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", tt.path, nil))
		if w.Code != 200 || reader.page != tt.page || reader.division != tt.division ||
			!strings.Contains(w.Body.String(), tt.marker) {
			t.Errorf("%s: status=%d query=(%d,%d) marker=%v", tt.path, w.Code,
				reader.page, reader.division, strings.Contains(w.Body.String(), tt.marker))
		}
	}
}

func TestWalletNonFiniteDisplayDoesNotCrashRendering(t *testing.T) {
	for _, value := range []string{"NaN", "+Inf", "-Inf"} {
		if walletIsk(&value) != "—" {
			t.Fatal("non-finite wallet amount was rendered")
		}
	}
}
