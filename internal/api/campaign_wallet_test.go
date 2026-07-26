package api

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func TestCampaignAndWalletRoutesRegisterCanonicalAndMigrationAliases(t *testing.T) {
	a := humago.New(
		http.NewServeMux(),
		huma.DefaultConfig("campaign-wallet-test", "test"),
	)
	registerCampaignRoutes(a, Options{})
	registerWalletRoutes(a, Options{})

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/campaigns"},
		{http.MethodPost, "/campaigns"},
		{http.MethodPost, "/campaign/create"},
		{http.MethodGet, "/campaigns/{id}"},
		{http.MethodGet, "/campaign/{id}"},
		{http.MethodPatch, "/campaigns/{id}"},
		{http.MethodPost, "/campaign/{id}/update"},
		{http.MethodDelete, "/campaigns/{id}"},
		{http.MethodGet, "/campaigns/{id}/killmails"},
		{http.MethodGet, "/campaign/{id}/killlist"},
		{http.MethodPost, "/campaigns/{id}/prizes/contributions"},
		{http.MethodPost, "/campaign/{id}/prize/contribute"},
		{http.MethodPost, "/campaigns/{id}/prizes/claim"},
		{http.MethodPost, "/campaign/{id}/prize/claim"},
		{http.MethodGet, "/admin/campaigns"},
		{http.MethodPost, "/admin/campaigns/{id}/actions"},
		{http.MethodPost, "/admin/campaigns/{id}/action"},
		{http.MethodPost, "/admin/campaigns/{id}/prizes/{rank}/payment"},
		{http.MethodPost, "/admin/campaign-prizes/{id}/{rank}/paid"},
		{http.MethodGet, "/wallet"},
		{http.MethodGet, "/me/wallet"},
		{http.MethodGet, "/user/wallet"},
		{http.MethodGet, "/me/wallet/balance"},
		{http.MethodGet, "/user/wallet/balance"},
		{http.MethodGet, "/admin/wallet"},
		{http.MethodPost, "/admin/wallet/sync"},
		{http.MethodGet, "/admin/wallet/authorize"},
	} {
		item := a.OpenAPI().Paths[route.path]
		if item == nil {
			t.Errorf("%s %s path is missing", route.method, route.path)
			continue
		}
		exists := false
		switch route.method {
		case http.MethodGet:
			exists = item.Get != nil
		case http.MethodPost:
			exists = item.Post != nil
		case http.MethodPatch:
			exists = item.Patch != nil
		case http.MethodDelete:
			exists = item.Delete != nil
		}
		if !exists {
			t.Errorf("%s %s operation is missing", route.method, route.path)
		}
	}

	account := a.OpenAPI().Paths["/me/wallet"].Get
	if got := account.Extensions["x-audience"]; got != "account" {
		t.Fatalf("account wallet audience = %#v", got)
	}
	admin := a.OpenAPI().Paths["/admin/wallet"].Get
	if got := admin.Extensions["x-audience"]; got != "admin" {
		t.Fatalf("admin wallet audience = %#v", got)
	}
	required := []map[string][]string{{"eveSession": {}}}
	if !reflect.DeepEqual(account.Security, required) ||
		!reflect.DeepEqual(admin.Security, required) {
		t.Fatalf(
			"wallet security account=%#v admin=%#v",
			account.Security,
			admin.Security,
		)
	}
	if got := a.OpenAPI().Paths["/wallet"].Get.Extensions["x-audience"]; got != "public" {
		t.Fatalf("public wallet audience = %#v", got)
	}
}

func TestCampaignWindowIsTheWorkloadBoundWithoutKillmailCap(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	within := now.AddDate(-1, 0, 1)
	for _, visibility := range []int16{
		campaignVisibilityPublic,
		campaignVisibilityPrivate,
		campaignVisibilityKillboard,
	} {
		if err := validateCampaignWindow(
			within, nil, visibility, false, now,
		); err != nil {
			t.Fatalf("365-day %d campaign rejected: %v", visibility, err)
		}
		if err := validateCampaignWindow(
			now.AddDate(-1, 0, -1), nil, visibility, false, now,
		); err == nil {
			t.Fatalf("over-one-year %d campaign was accepted", visibility)
		}
	}

	// Admins can repair/import older campaigns. There is deliberately no
	// killmail-count input or threshold in this guard: processing is bounded by
	// time, while River executes the complete matching set.
	if err := validateCampaignWindow(
		now.AddDate(-10, 0, 0), nil,
		campaignVisibilityPublic, true, now,
	); err != nil {
		t.Fatalf("admin repair window rejected: %v", err)
	}
}

func TestCampaignPrizeValidationAndPayoutRemainder(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	end := now.Add(24 * time.Hour)
	prize, amount, requestID, err := parseCampaignPrizeInput(
		map[string]any{
			"enabled":     true,
			"metric":      json.Number("2"),
			"winnerCount": json.Number("3"),
			"payoutPercentages": []any{
				json.Number("70"), json.Number("20"), json.Number("10"),
			},
			"initialContribution": "123.40",
			"fundingRequestId":    "8dd566e0-6963-44ae-bd78-c7b88c13d6d2",
		},
		&end,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if prize.Metric != 2 || prize.WinnerCount != 3 ||
		!reflect.DeepEqual(prize.Percentages, []int16{70, 20, 10}) {
		t.Fatalf("prize = %#v", prize)
	}
	if amount != "123.40" ||
		requestID != "8dd566e0-6963-44ae-bd78-c7b88c13d6d2" {
		t.Fatalf("funding = %q %q", amount, requestID)
	}
	if got := campaignPrizePayouts(101, []int16{70, 20, 10}); !reflect.DeepEqual(got, []float64{71, 20, 10}) {
		t.Fatalf("payouts = %#v", got)
	}

	_, _, _, err = parseCampaignPrizeInput(
		map[string]any{
			"enabled":             false,
			"initialContribution": "1.00",
		},
		&end,
		now,
	)
	if err == nil || !strings.Contains(err.Error(), "Enable campaign prizes") {
		t.Fatalf("disabled initial funding error = %v", err)
	}
}

func TestWalletAmountNormalizationUsesExactCents(t *testing.T) {
	for raw, expected := range map[string]string{
		"1":                        "1.00",
		"001.2":                    "1.20",
		"999999999999999999999.99": "999999999999999999999.99",
	} {
		got, err := normalizeWalletAmount(raw)
		if err != nil {
			t.Fatalf("%q: %v", raw, err)
		}
		if got != expected {
			t.Fatalf("%q = %q, want %q", raw, got, expected)
		}
		cents, err := parseWalletCentsBig(got)
		if err != nil || formatWalletCentsBig(cents) != expected {
			t.Fatalf("%q round trip = %v %q", raw, err, formatWalletCentsBig(cents))
		}
	}
	for _, raw := range []any{"0", "-1", "1.001", "ISK 5"} {
		if _, err := normalizeWalletAmount(raw); err == nil {
			t.Errorf("invalid amount %#v was accepted", raw)
		}
	}
}

func TestWalletOAuthStartUsesBoundOneTimePurpose(t *testing.T) {
	rig := newAuthTestRig(t)
	service := newAuthService(Options{
		Auth: AuthOptions{
			ClientID: "client", ClientSecret: "client-secret",
			CallbackURL: "https://eve-kill.com/auth/eve/callback",
			StateSecret: string(rig.secret),
			store:       rig.store, flowStore: rig.flows, oauth: rig.oauth,
			now:    func() time.Time { return rig.now },
			random: strings.NewReader(strings.Repeat("a", 256)),
		},
	})
	authorizationURL, cookie, err := service.beginWalletAuthorization(
		context.Background(),
		9001,
		98779905,
	)
	if err != nil {
		t.Fatal(err)
	}
	if authorizationURL == "" || cookie == nil ||
		cookie.Path != "/auth" || !cookie.HttpOnly {
		t.Fatalf("wallet OAuth start = %q %#v", authorizationURL, cookie)
	}
	if !reflect.DeepEqual(rig.oauth.scopes, walletAuthorizationScopes) {
		t.Fatalf("wallet scopes = %#v", rig.oauth.scopes)
	}
	if len(rig.flows.items) != 1 {
		t.Fatalf("stored flows = %d", len(rig.flows.items))
	}
	for _, item := range rig.flows.items {
		if item.flow.Purpose != oauthStatePurposeWallet ||
			item.flow.Wallet == nil ||
			item.flow.Wallet.RequestedByCharacterID != 9001 ||
			item.flow.Wallet.CorporationID != 98779905 {
			t.Fatalf("stored wallet flow = %#v", item.flow)
		}
	}
}
