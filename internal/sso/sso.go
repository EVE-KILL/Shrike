// Package sso refreshes the EVE SSO tokens that authenticated ESI calls need.
//
// A stored token is a refresh token plus a short-lived access token, and the
// refresh token is the valuable half: it survives until the user revokes
// consent, and losing it means asking them to log in again. So the write path
// here is careful — a token is only ever disabled on evidence that it is
// permanently dead, never on a transient failure.
//
// Scopes are the subtle part, and the reason this package is more than an HTTP
// call. See RevokedScopes below.
package sso

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TokenURL is where a refresh token is exchanged for an access token.
const TokenURL = "https://login.eveonline.com/v2/oauth/token"

// Scopes worth naming, because a job is dispatched for each.
const (
	ScopeCharacterKillmails   = "esi-killmails.read_killmails.v1"
	ScopeCorporationKillmails = "esi-killmails.read_corporation_killmails.v1"
	ScopeCorporationWallet    = "esi-wallet.read_corporation_wallets.v1"
)

// ErrPermanentlyDead means the refresh token will never work again — the user
// revoked consent, or CCP invalidated it. Distinct from a transient failure
// because the response is different: disable the token rather than retry it.
var ErrPermanentlyDead = errors.New("refresh token is permanently dead")

// Token is a stored SSO token.
type Token struct {
	CharacterID  int32
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	Scopes       []string
	DelayHours   int

	// RevokedScopes are scopes SSO keeps granting that this character cannot
	// actually exercise. See Refresh for why they are tracked separately.
	RevokedScopes []string

	CharacterFailureCount int
	Disabled              bool
}

// Usable reports the scopes worth acting on — granted and not locally revoked.
func (t Token) Usable() []string {
	revoked := make(map[string]bool, len(t.RevokedScopes))
	for _, s := range t.RevokedScopes {
		revoked[s] = true
	}

	var out []string
	for _, s := range t.Scopes {
		if !revoked[s] {
			out = append(out, s)
		}
	}
	return out
}

// Has reports whether a usable scope is present.
func (t Token) Has(scope string) bool {
	for _, s := range t.Usable() {
		if s == scope {
			return true
		}
	}
	return false
}

// Client refreshes tokens against EVE SSO.
type Client struct {
	ClientID     string
	ClientSecret string
	UserAgent    string
	HTTP         *http.Client

	// TokenURL is overridable for tests.
	TokenURL string
}

// refreshResponse is SSO's reply.
type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Refresh exchanges a refresh token for a new access token.
//
// Returns ErrPermanentlyDead when SSO says the grant is gone, which the caller
// turns into a disabled token. Any other failure is transient and worth
// retrying: SSO has bad minutes, and disabling a live token on one of them
// costs the user a re-login.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (*refreshResponse, error) {
	if c.ClientID == "" || c.ClientSecret == "" {
		return nil, errors.New("sso: EVE_CLIENT_ID and EVE_CLIENT_SECRET are not configured")
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	endpoint := c.TokenURL
	if endpoint == "" {
		endpoint = TokenURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(
		[]byte(c.ClientID+":"+c.ClientSecret)))

	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		// SSO signals a dead grant in the body rather than the status, and with
		// more than one wording. Matching on the text is unpleasant but it is
		// the only signal available.
		text := string(body)
		for _, marker := range []string{"invalid_grant", "Token missing", "expired"} {
			if strings.Contains(text, marker) {
				return nil, fmt.Errorf("%w: %s", ErrPermanentlyDead, marker)
			}
		}
		return nil, fmt.Errorf("sso: token refresh returned %d", resp.StatusCode)
	}

	var out refreshResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("sso: decode token response: %w", err)
	}
	if out.AccessToken == "" {
		return nil, errors.New("sso: token response carried no access token")
	}
	return &out, nil
}

// ScopesFromAccessToken reads the granted scopes out of the access token.
//
// The access token is a JWT and carries its scopes in the `scp` claim, so they
// can be read without a second request. CCP removed the /oauth/verify endpoint
// that used to answer this in March 2026, which is what forced the change.
//
// The signature is deliberately NOT verified here. That sounds alarming and is
// not: this token was received over TLS directly from SSO in the response to
// our own authenticated request moments ago, so its provenance is already
// established — there is no untrusted party in between to forge it. Verifying
// would mean fetching and caching CCP's JWKS and failing refreshes whenever
// that endpoint is unwell, for no gain. A token that is somehow malformed fails
// at the first ESI call instead, which is a cheaper and more honest place to
// find out.
func ScopesFromAccessToken(accessToken string) ([]string, error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return nil, errors.New("sso: access token is not a JWT")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("sso: decode JWT payload: %w", err)
	}

	// `scp` is a space-separated string for a single scope and an array for
	// several, so both have to be accepted.
	var claims struct {
		Scp json.RawMessage `json:"scp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("sso: decode JWT claims: %w", err)
	}
	if len(claims.Scp) == 0 {
		return nil, nil
	}

	var list []string
	if err := json.Unmarshal(claims.Scp, &list); err == nil {
		return nonEmpty(list), nil
	}

	var single string
	if err := json.Unmarshal(claims.Scp, &single); err == nil {
		return nonEmpty(strings.Split(single, " ")), nil
	}
	return nil, errors.New("sso: scp claim is neither a string nor an array")
}

func nonEmpty(in []string) []string {
	var out []string
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// LoadToken reads a stored token.
func LoadToken(ctx context.Context, pool *pgxpool.Pool, characterID int32) (*Token, error) {
	var t Token
	err := pool.QueryRow(ctx, `
        SELECT character_id, coalesce(access_token, ''), refresh_token,
               coalesce(token_expiry, 'epoch'::timestamptz),
               coalesce(scopes, '{}'), coalesce(revoked_scopes, '{}'),
               coalesce(delay, 0), coalesce(character_failure_count, 0),
               coalesce(disabled, false)
        FROM user_esi_tokens WHERE character_id = $1`, characterID).
		Scan(&t.CharacterID, &t.AccessToken, &t.RefreshToken, &t.Expiry,
			&t.Scopes, &t.RevokedScopes, &t.DelayHours,
			&t.CharacterFailureCount, &t.Disabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// StoreRefreshed writes a successful refresh.
func StoreRefreshed(ctx context.Context, pool *pgxpool.Pool, characterID int32, r *refreshResponse, usable []string) error {
	_, err := pool.Exec(ctx, `
        UPDATE user_esi_tokens SET
            access_token = $2,
            refresh_token = $3,
            token_expiry = $4,
            scopes = $5,
            last_fetched = now(),
            updated_at = now()
        WHERE character_id = $1`,
		characterID, r.AccessToken, r.RefreshToken,
		time.Now().UTC().Add(time.Duration(r.ExpiresIn)*time.Second), usable)
	return err
}

// Disable stops a token being refreshed again.
func Disable(ctx context.Context, pool *pgxpool.Pool, characterID int32, clearScopes bool) error {
	if clearScopes {
		_, err := pool.Exec(ctx, `
            UPDATE user_esi_tokens SET disabled = true, scopes = '{}', updated_at = now()
            WHERE character_id = $1`, characterID)
		return err
	}
	_, err := pool.Exec(ctx, `
        UPDATE user_esi_tokens SET disabled = true, updated_at = now()
        WHERE character_id = $1`, characterID)
	return err
}

// RevokeScope records that a scope SSO grants cannot actually be exercised.
//
// This exists because of a specific, expensive failure. SSO re-grants every
// scope the user ever consented to on every refresh, with no idea whether the
// character can still use it — the corporation killmail endpoint additionally
// requires an in-game role that the user may have lost. Before this existed the
// 403 handler stripped the scope, the next refresh wrote it straight back from
// the JWT, and the cycle repeated: 180 characters produced 525,000 403s in a
// month, all against the shared ESI error budget.
//
// The revocation is therefore stored separately from the granted list, so a
// refresh cannot undo it.
func RevokeScope(ctx context.Context, pool *pgxpool.Pool, characterID int32, scope string) error {
	_, err := pool.Exec(ctx, `
        UPDATE user_esi_tokens SET
            revoked_scopes = (
                SELECT array_agg(DISTINCT s)
                FROM unnest(coalesce(revoked_scopes, '{}') || $2::text) AS s
            ),
            scopes_revoked_at = now(),
            updated_at = now()
        WHERE character_id = $1`, characterID, scope)
	return err
}

// RecordCharacterKillmailFailure counts consecutive 403s from a character's
// own killmail endpoint.
//
// A single 403 is not enough evidence to retire a token: role and ESI state can
// briefly disagree. Once five consecutive requests fail, the token is disabled
// only if its corporation-killmail side is unusable too. This preserves tokens
// that can still contribute corporation kills.
func RecordCharacterKillmailFailure(
	ctx context.Context,
	pool *pgxpool.Pool,
	characterID int32,
) error {
	_, err := pool.Exec(ctx, `
        UPDATE user_esi_tokens
        SET character_failure_count = coalesce(character_failure_count, 0) + 1,
            disabled = coalesce(disabled, false) OR (
                coalesce(character_failure_count, 0) + 1 >= 5
                AND NOT (
                    $2 = ANY(coalesce(scopes, '{}'))
                    AND NOT ($2 = ANY(coalesce(revoked_scopes, '{}')))
                )
            ),
            updated_at = now()
        WHERE character_id = $1`,
		characterID,
		ScopeCorporationKillmails,
	)
	return err
}

// ResetCharacterKillmailFailures clears the consecutive-403 streak after a
// successful character request.
func ResetCharacterKillmailFailures(
	ctx context.Context,
	pool *pgxpool.Pool,
	characterID int32,
) error {
	_, err := pool.Exec(ctx, `
        UPDATE user_esi_tokens
        SET character_failure_count = 0, updated_at = now()
        WHERE character_id = $1
          AND coalesce(character_failure_count, 0) <> 0`,
		characterID,
	)
	return err
}

// StaleTokens returns tokens due for a refresh, soonest expiry first.
//
// A token is refreshed before it expires rather than after: an access token
// lasts twenty minutes and a job that picks one up at nineteen would spend its
// whole run racing the clock.
func StaleTokens(ctx context.Context, pool *pgxpool.Pool, within time.Duration, limit int) ([]int32, error) {
	rows, err := pool.Query(ctx, `
        SELECT character_id FROM user_esi_tokens
        WHERE disabled IS NOT TRUE
          AND (token_expiry IS NULL OR token_expiry <= now() + $1::interval)
        ORDER BY token_expiry ASC NULLS FIRST
        LIMIT $2`, within.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
