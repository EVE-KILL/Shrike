package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	campaignengine "github.com/eve-kill/shrike/internal/campaign"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/sso"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	walletESIBaseURL              = "https://esi.evetech.net/latest"
	walletESIBodyLimit            = 1 << 20
	walletAuthorizationErrorLimit = 300
)

type pendingWalletOAuthFlow struct {
	RequestedByCharacterID int32 `json:"requested_by_character_id"`
	CorporationID          int32 `json:"corporation_id"`
}

type walletAuthorizationService struct {
	auth       *authService
	db         MutationDatabase
	storeErr   error
	httpClient *http.Client
	userAgent  string
	queue      *queue.Client
	queueErr   error
	now        func() time.Time
}

type walletAffiliation struct {
	CharacterID   int32 `json:"character_id"`
	CorporationID int32 `json:"corporation_id"`
}

type walletESIBalance struct {
	Division int16   `json:"division"`
	Balance  float64 `json:"balance"`
}

func newWalletAuthorizationService(
	opts Options,
	auth *authService,
) *walletAuthorizationService {
	service := &walletAuthorizationService{
		auth:       auth,
		httpClient: opts.Auth.HTTPClient,
		userAgent:  opts.Auth.UserAgent,
		now:        time.Now,
	}
	if service.httpClient == nil {
		service.httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if service.userAgent == "" {
		service.userAgent = "EVE-Kill/2.0 (https://eve-kill.com)"
	}
	service.db, service.storeErr = mutationDatabase(opts)
	if pool, ok := opts.DB.(*pgxpool.Pool); ok && pool != nil {
		service.queue, service.queueErr = queue.New(queue.Options{Pool: pool})
	} else {
		service.queueErr = errors.New("wallet queue is not configured")
	}
	return service
}

func (s *authService) beginWalletAuthorization(
	ctx context.Context,
	requestedByCharacterID int32,
	corporationID int32,
) (string, *http.Cookie, error) {
	if s.flowErr != nil {
		return "", nil, s.flowErr
	}
	if len(s.stateSecret) == 0 {
		return "", nil, apiError(
			http.StatusServiceUnavailable,
			"OAuth state secret is not configured",
		)
	}
	if requestedByCharacterID <= 0 ||
		corporationID != campaignengine.EveKillCorporationID {
		return "", nil, apiError(
			http.StatusBadRequest,
			"Invalid corporation wallet authorization request",
		)
	}
	now := s.now().UTC()
	flow := pendingOAuthFlow{
		Purpose:  oauthStatePurposeWallet,
		IssuedAt: now,
		Wallet: &pendingWalletOAuthFlow{
			RequestedByCharacterID: requestedByCharacterID,
			CorporationID:          corporationID,
		},
	}
	var id, binding string
	var err error
	for range 3 {
		id, err = randomBase64(s.random, 32)
		if err != nil {
			return "", nil, err
		}
		binding, err = randomBase64(s.random, 32)
		if err != nil {
			return "", nil, err
		}
		err = s.flows.Put(
			ctx, id, bindingDigest(binding), flow, oauthFlowTTL,
		)
		if !errors.Is(err, errOAuthCollision) {
			break
		}
	}
	if err != nil {
		return "", nil, err
	}
	state, err := signOAuthState(signedOAuthState{
		Version:  oauthStateVersion,
		Purpose:  oauthStatePurposeWallet,
		ID:       id,
		IssuedAt: now.Unix(),
	}, s.stateSecret)
	if err != nil {
		return "", nil, err
	}
	authorizationURL, err := s.oauth.BuildAuthorizationURL(
		state, walletAuthorizationScopes,
	)
	if err != nil {
		return "", nil, err
	}
	return authorizationURL, oauthFlowCookie(
		oauthFlowCookieName(id),
		binding,
		int(oauthFlowTTL/time.Second),
		s.production,
	), nil
}

func (s *walletAuthorizationService) complete(
	ctx context.Context,
	req *legacyRequest,
	flow pendingOAuthFlow,
) (legacyPayload, error) {
	fail := func(message string) (legacyPayload, error) {
		return walletAuthorizationRedirect("error", message), nil
	}
	if flow.Wallet == nil ||
		flow.Wallet.RequestedByCharacterID <= 0 ||
		flow.Wallet.CorporationID != campaignengine.EveKillCorporationID {
		return fail("Invalid wallet authorization state")
	}
	if req.Query.Get("error") != "" {
		return fail("Wallet authorization was cancelled")
	}
	code := strings.TrimSpace(req.Query.Get("code"))
	if code == "" {
		return fail("CCP did not return an authorization code")
	}
	if s.storeErr != nil {
		return fail("Wallet storage is unavailable")
	}
	admin, err := s.auth.requirePrincipal(ctx, req)
	if err != nil || admin == nil || !admin.IsAdmin {
		return fail("Administrator session is required")
	}
	if admin.CharacterID != flow.Wallet.RequestedByCharacterID {
		return fail("The admin session changed during wallet authorization")
	}
	tokens, err := s.auth.oauth.ExchangeCode(ctx, code)
	if err != nil {
		return fail("EVE SSO token exchange failed")
	}
	claims, err := s.auth.oauth.VerifyAccessToken(ctx, tokens.AccessToken)
	if err != nil {
		return fail("Invalid EVE access token")
	}
	if !walletContainsScope(claims.Scopes, walletRequiredScope) {
		return fail(
			"CCP did not grant the required " + walletRequiredScope + " scope",
		)
	}
	affiliation, err := s.loadAffiliation(ctx, claims.CharacterID)
	if err != nil {
		return fail(err.Error())
	}
	if affiliation.CorporationID != flow.Wallet.CorporationID {
		return fail("The selected character is not a member of EVE-KILL.com")
	}
	balances, err := s.loadBalances(
		ctx, flow.Wallet.CorporationID, tokens.AccessToken,
	)
	if err != nil {
		return fail(err.Error())
	}
	if err := s.saveAuthorization(
		ctx,
		flow.Wallet.CorporationID,
		admin.CharacterID,
		claims,
		tokens,
		balances,
	); err != nil {
		return fail("Could not save the corporation wallet authorization")
	}

	message := ""
	if s.queue == nil || s.queueErr != nil {
		message = "Wallet authorized. The journal will be imported by the next hourly sync"
	} else if _, err := queue.Dispatch(
		context.WithoutCancel(ctx),
		s.queue,
		queue.CorporationWalletArgs{
			CorporationID: flow.Wallet.CorporationID,
			Force:         true,
		},
		queue.Immediate,
	); err != nil {
		message = "Wallet authorized. The journal will be imported by the next hourly sync"
	}
	return walletAuthorizationRedirect("ok", message), nil
}

func (s *walletAuthorizationService) loadAffiliation(
	ctx context.Context,
	characterID int32,
) (walletAffiliation, error) {
	body, _ := json.Marshal([]int32{characterID})
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		walletESIBaseURL+"/characters/affiliation/",
		bytes.NewReader(body),
	)
	if err != nil {
		return walletAffiliation{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", s.userAgent)
	response, err := s.httpClient.Do(req)
	if err != nil {
		return walletAffiliation{}, fmt.Errorf(
			"Could not verify the selected character",
		)
	}
	defer response.Body.Close() //nolint:errcheck
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return walletAffiliation{}, fmt.Errorf(
			"Could not verify the selected character: %s",
			readWalletESIError(response),
		)
	}
	var affiliations []walletAffiliation
	if err := json.NewDecoder(io.LimitReader(
		response.Body, walletESIBodyLimit,
	)).Decode(&affiliations); err != nil {
		return walletAffiliation{}, fmt.Errorf(
			"Could not verify the selected character",
		)
	}
	for _, affiliation := range affiliations {
		if affiliation.CharacterID == characterID {
			return affiliation, nil
		}
	}
	return walletAffiliation{}, fmt.Errorf(
		"ESI did not return an affiliation for the selected character",
	)
}

func (s *walletAuthorizationService) loadBalances(
	ctx context.Context,
	corporationID int32,
	accessToken string,
) ([]walletESIBalance, error) {
	endpoint := fmt.Sprintf(
		"%s/corporations/%d/wallets/",
		walletESIBaseURL,
		corporationID,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", s.userAgent)
	response, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Could not read the corporation wallet")
	}
	defer response.Body.Close() //nolint:errcheck
	if response.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf(
			"ESI denied corporation wallet access. Select a character with Accountant or Junior Accountant",
		)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"Could not read the corporation wallet: %s",
			readWalletESIError(response),
		)
	}
	var balances []walletESIBalance
	if err := json.NewDecoder(io.LimitReader(
		response.Body, walletESIBodyLimit,
	)).Decode(&balances); err != nil {
		return nil, fmt.Errorf("Could not read the corporation wallet")
	}
	return balances, nil
}

func (s *walletAuthorizationService) saveAuthorization(
	ctx context.Context,
	corporationID int32,
	adminCharacterID int32,
	claims sso.AccessClaims,
	tokens sso.AuthorizationCodeTokens,
	balances []walletESIBalance,
) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	now := s.now().UTC()
	scopes := claims.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO corporation_wallet_tokens (
		  corporation_id, authorized_character_id,
		  authorized_character_name, authorized_character_owner_hash,
		  authorized_by_admin_character_id, access_token, refresh_token,
		  token_expiry, token_type, scopes, disabled, last_balance_sync,
		  last_error, created_at, updated_at
		) VALUES (
		  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		  false, $11, NULL, $11, $11
		)
		ON CONFLICT (corporation_id) DO UPDATE SET
		  authorized_character_id = excluded.authorized_character_id,
		  authorized_character_name = excluded.authorized_character_name,
		  authorized_character_owner_hash =
		    excluded.authorized_character_owner_hash,
		  authorized_by_admin_character_id =
		    excluded.authorized_by_admin_character_id,
		  access_token = excluded.access_token,
		  refresh_token = excluded.refresh_token,
		  token_expiry = excluded.token_expiry,
		  token_type = excluded.token_type,
		  scopes = excluded.scopes,
		  disabled = false,
		  last_balance_sync = excluded.last_balance_sync,
		  last_error = NULL,
		  updated_at = excluded.updated_at`,
		corporationID,
		claims.CharacterID,
		claims.CharacterName,
		claims.CharacterOwnerHash,
		adminCharacterID,
		tokens.AccessToken,
		tokens.RefreshToken,
		now.Add(time.Duration(tokens.ExpiresIn)*time.Second),
		tokens.TokenType,
		scopes,
		now,
	); err != nil {
		return err
	}
	for _, balance := range balances {
		if balance.Division < 1 || balance.Division > 7 {
			continue
		}
		value := "0.00"
		if !isInvalidFloat(balance.Balance) {
			value = strconv.FormatFloat(balance.Balance, 'f', 2, 64)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO corporation_wallet_balances (
			  corporation_id, division, balance, updated_at
			) VALUES ($1, $2, $3::numeric, $4)
			ON CONFLICT (corporation_id, division) DO UPDATE SET
			  balance = excluded.balance,
			  updated_at = excluded.updated_at`,
			corporationID, balance.Division, value, now,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func walletAuthorizationRedirect(status, message string) legacyPayload {
	query := url.Values{"wallet_auth": {status}}
	message = truncateRunes(strings.TrimSpace(message), walletAuthorizationErrorLimit)
	if message != "" {
		query.Set("message", message)
	}
	headers := make(http.Header)
	headers.Set("Location", "/admin/wallet?"+query.Encode())
	headers.Set("Cache-Control", "private, no-store")
	headers.Set("Pragma", "no-cache")
	return legacyPayload{
		Status: http.StatusFound, Headers: headers, RawBody: []byte{},
	}
}

func readWalletESIError(response *http.Response) string {
	var body struct {
		Error string `json:"error"`
	}
	if json.NewDecoder(io.LimitReader(
		response.Body, walletESIBodyLimit,
	)).Decode(&body) == nil && strings.TrimSpace(body.Error) != "" {
		return truncateRunes(body.Error, walletAuthorizationErrorLimit)
	}
	return fmt.Sprintf("HTTP %d", response.StatusCode)
}

func walletContainsScope(values []string, expected string) bool {
	return slices.Contains(values, expected)
}

func isInvalidFloat(value float64) bool {
	return value != value || value > 1.7976931348623157e308 ||
		value < -1.7976931348623157e308
}
