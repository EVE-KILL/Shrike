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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	// AuthorizationURL is EVE SSO's authorization-code endpoint.
	AuthorizationURL = "https://login.eveonline.com/v2/oauth/authorize"
	// JWKSURL publishes the keys used to sign EVE access tokens.
	JWKSURL = "https://login.eveonline.com/oauth/jwks"
)

var acceptedIssuers = map[string]bool{
	"login.eveonline.com":         true,
	"https://login.eveonline.com": true,
}

// AuthorizationCodeTokens is the successful authorization-code response.
// Refresh tokens are deliberately never exposed by an HTTP API; this type is
// only passed to the transactional login store.
type AuthorizationCodeTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	TokenType    string
}

// AccessClaims is the identity and scope subset carried by EVE's access JWT.
type AccessClaims struct {
	CharacterID        int32
	CharacterName      string
	CharacterOwnerHash string
	Scopes             []string
}

// OAuthClient implements EVE's confidential authorization-code flow.
//
// The remote key set is process-local and long-lived. coreos/go-oidc refreshes
// it when a token references an unknown key, so ordinary requests do not make
// a JWKS round trip.
type OAuthClient struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
	UserAgent    string
	HTTP         *http.Client

	// Endpoint overrides exist for deterministic tests.
	AuthorizeURL string
	TokenURL     string
	JWKSURL      string

	verifierOnce sync.Once
	verifier     *oidc.IDTokenVerifier
	verifierErr  error
}

// BuildAuthorizationURL creates an EVE SSO authorization URL for a state that
// has already been browser-bound by the caller.
func (c *OAuthClient) BuildAuthorizationURL(state string, scopes []string) (string, error) {
	if c.ClientID == "" || c.ClientSecret == "" || c.CallbackURL == "" {
		return "", errors.New("sso: client id, client secret, and callback URL are required")
	}
	if state == "" {
		return "", errors.New("sso: OAuth state is required")
	}

	endpoint := c.AuthorizeURL
	if endpoint == "" {
		endpoint = AuthorizationURL
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("sso: parse authorization URL: %w", err)
	}
	query := parsed.Query()
	query.Set("response_type", "code")
	query.Set("redirect_uri", c.CallbackURL)
	query.Set("client_id", c.ClientID)
	query.Set("scope", strings.Join(nonEmpty(scopes), " "))
	query.Set("state", state)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

// ExchangeCode exchanges a one-time authorization code using HTTP Basic
// client authentication, matching EVE's confidential web-application flow.
func (c *OAuthClient) ExchangeCode(
	ctx context.Context,
	code string,
) (AuthorizationCodeTokens, error) {
	if c.ClientID == "" || c.ClientSecret == "" {
		return AuthorizationCodeTokens{},
			errors.New("sso: EVE_CLIENT_ID and EVE_CLIENT_SECRET are not configured")
	}
	if strings.TrimSpace(code) == "" {
		return AuthorizationCodeTokens{}, errors.New("sso: authorization code is required")
	}

	endpoint := c.TokenURL
	if endpoint == "" {
		endpoint = TokenURL
	}
	form := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return AuthorizationCodeTokens{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(
		[]byte(c.ClientID+":"+c.ClientSecret)))

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return AuthorizationCodeTokens{}, fmt.Errorf("sso: exchange authorization code: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return AuthorizationCodeTokens{}, fmt.Errorf("sso: read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return AuthorizationCodeTokens{},
			fmt.Errorf("sso: authorization-code exchange returned %d", resp.StatusCode)
	}

	var wire struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return AuthorizationCodeTokens{}, fmt.Errorf("sso: decode token response: %w", err)
	}
	if wire.AccessToken == "" || wire.RefreshToken == "" || wire.ExpiresIn <= 0 {
		return AuthorizationCodeTokens{}, errors.New("sso: token response is incomplete")
	}
	if wire.TokenType == "" {
		wire.TokenType = "Bearer"
	}
	return AuthorizationCodeTokens{
		AccessToken: wire.AccessToken, RefreshToken: wire.RefreshToken,
		ExpiresIn: wire.ExpiresIn, TokenType: wire.TokenType,
	}, nil
}

// VerifyAccessToken verifies signature, expiry, audience, issuer, and EVE's
// character subject before returning identity claims.
func (c *OAuthClient) VerifyAccessToken(
	ctx context.Context,
	accessToken string,
) (AccessClaims, error) {
	verifier, err := c.tokenVerifier()
	if err != nil {
		return AccessClaims{}, err
	}
	token, err := verifier.Verify(ctx, accessToken)
	if err != nil {
		return AccessClaims{}, fmt.Errorf("sso: verify access token: %w", err)
	}
	if !acceptedIssuers[token.Issuer] {
		return AccessClaims{}, fmt.Errorf("sso: unexpected issuer %q", token.Issuer)
	}

	var claims struct {
		Subject string          `json:"sub"`
		Name    string          `json:"name"`
		Owner   string          `json:"owner"`
		Scp     json.RawMessage `json:"scp"`
	}
	if err := token.Claims(&claims); err != nil {
		return AccessClaims{}, fmt.Errorf("sso: decode access-token claims: %w", err)
	}
	const subjectPrefix = "CHARACTER:EVE:"
	if !strings.HasPrefix(claims.Subject, subjectPrefix) {
		return AccessClaims{}, fmt.Errorf("sso: unexpected JWT subject %q", claims.Subject)
	}
	characterID, err := strconv.ParseInt(
		strings.TrimPrefix(claims.Subject, subjectPrefix), 10, 32,
	)
	if err != nil || characterID <= 0 {
		return AccessClaims{}, fmt.Errorf("sso: invalid JWT subject %q", claims.Subject)
	}
	if strings.TrimSpace(claims.Name) == "" || strings.TrimSpace(claims.Owner) == "" {
		return AccessClaims{}, errors.New("sso: access token is missing character identity claims")
	}

	scopes, err := scopesFromRawClaim(claims.Scp)
	if err != nil {
		return AccessClaims{}, err
	}
	return AccessClaims{
		CharacterID:        int32(characterID),
		CharacterName:      claims.Name,
		CharacterOwnerHash: claims.Owner,
		Scopes:             scopes,
	}, nil
}

func (c *OAuthClient) tokenVerifier() (*oidc.IDTokenVerifier, error) {
	c.verifierOnce.Do(func() {
		if c.ClientID == "" {
			c.verifierErr = errors.New("sso: EVE_CLIENT_ID is not configured")
			return
		}
		jwksURL := c.JWKSURL
		if jwksURL == "" {
			jwksURL = JWKSURL
		}
		ctx := context.WithValue(
			context.Background(), oauth2.HTTPClient, c.httpClient(),
		)
		keys := oidc.NewRemoteKeySet(ctx, jwksURL)
		c.verifier = oidc.NewVerifier("", keys, &oidc.Config{
			ClientID:             c.ClientID,
			SkipIssuerCheck:      true,
			SupportedSigningAlgs: []string{oidc.RS256},
		})
	})
	return c.verifier, c.verifierErr
}

func (c *OAuthClient) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func scopesFromRawClaim(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return nonEmpty(list), nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return nonEmpty(strings.Split(single, " ")), nil
	}
	return nil, errors.New("sso: scp claim is neither a string nor an array")
}
