package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/sso"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	authSessionCookie       = "evelogin"
	authHintCookie          = "ek_auth"
	oauthFlowCookiePrefix   = "ek_oauth_"
	oauthFlowTTL            = 5 * time.Minute
	oauthFutureClockSkew    = 30 * time.Second
	authSessionTTL          = 365 * 24 * time.Hour
	authSessionTouchEvery   = 5 * time.Minute
	maxOAuthStateLength     = 2048
	maxReturnToLength       = 2048
	oauthStateVersion       = 1
	redisOAuthFlowKeyPrefix = "shrike:oauth:flow:"
)

type oauthFlowPurpose string

const (
	oauthStatePurposeLogin  oauthFlowPurpose = "login"
	oauthStatePurposeWallet oauthFlowPurpose = "corporation_wallet"
)

var (
	errNoAuthSession  = errors.New("auth session not found")
	errOAuthFlow      = errors.New("invalid or expired OAuth flow")
	errOAuthCollision = errors.New("OAuth flow id already exists")
)

// Principal is the small request-time identity. It deliberately excludes
// user_config and account-detail fields: authorization should be one indexed
// lookup, not a settings-page query paid by every authenticated request.
type Principal struct {
	CharacterID            int32
	CharacterName          string
	IsAdmin                bool
	CorporationID          *int32
	CorporationName        *string
	AllianceID             *int32
	AllianceName           *string
	LastSeenNotificationID int64
}

type authSessionActivity struct {
	RowID       int64
	IPAddress   *string
	CountryCode *string
	UserAgent   *string
	ClientHint  *string
	LastSeenAt  time.Time
}

type authSessionMetadata struct {
	IPAddress   *string
	CountryCode *string
	UserAgent   *string
	ClientHint  *string
}

type authMeDetails struct {
	CharacterOwnerHash string
	Settings           map[string]any
	LastLogin          *time.Time
	CreatedAt          time.Time
}

type authTokenInfo struct {
	Scopes        []string
	RevokedScopes []string
	TokenExpiry   *time.Time
	LastFetched   *time.Time
	Disabled      bool
}

type authSessionSummary struct {
	ID          int64
	Current     bool
	IPAddress   *string
	CountryCode *string
	UserAgent   *string
	ClientHint  *string
	CreatedAt   time.Time
	LastSeenAt  time.Time
	ExpiresAt   time.Time
}

type authLoginCommit struct {
	SessionID string
	Now       time.Time
	ExpiresAt time.Time
	Delay     int
	Claims    sso.AccessClaims
	Tokens    sso.AuthorizationCodeTokens
	Metadata  authSessionMetadata
}

type authStore interface {
	ResolveSession(
		context.Context,
		string,
		time.Time,
	) (Principal, authSessionActivity, error)
	TouchSession(context.Context, int64, authSessionMetadata, time.Time) error
	LoadMeDetails(context.Context, int32) (authMeDetails, error)
	LoadTokenInfo(context.Context, int32) (authTokenInfo, bool, error)
	CompleteLogin(context.Context, authLoginCommit) error
	DeleteSession(context.Context, string) error
	ListSessions(
		context.Context,
		int32,
		string,
		time.Time,
	) ([]authSessionSummary, error)
	RevokeSession(
		context.Context,
		int32,
		int64,
		string,
	) (found bool, current bool, err error)
	RevokeOtherSessions(context.Context, int32, string) (int64, error)
}

type pendingOAuthFlow struct {
	Purpose  oauthFlowPurpose        `json:"purpose"`
	ReturnTo string                  `json:"return_to"`
	Delay    int                     `json:"delay"`
	CharKM   bool                    `json:"char_km"`
	CorpKM   bool                    `json:"corp_km"`
	IssuedAt time.Time               `json:"issued_at"`
	Wallet   *pendingWalletOAuthFlow `json:"wallet,omitempty"`
}

type oauthFlowStore interface {
	Put(
		context.Context,
		string,
		string,
		pendingOAuthFlow,
		time.Duration,
	) error
	Take(context.Context, string, string) (pendingOAuthFlow, error)
}

type oauthCodeClient interface {
	BuildAuthorizationURL(string, []string) (string, error)
	ExchangeCode(context.Context, string) (sso.AuthorizationCodeTokens, error)
	VerifyAccessToken(context.Context, string) (sso.AccessClaims, error)
}

type authService struct {
	store       authStore
	storeErr    error
	flows       oauthFlowStore
	flowErr     error
	oauth       oauthCodeClient
	stateSecret []byte
	production  bool
	now         func() time.Time
	random      io.Reader
	wallet      *walletAuthorizationService
}

func newAuthService(opts Options) *authService {
	auth := opts.Auth
	service := &authService{
		store:      auth.store,
		flows:      auth.flowStore,
		oauth:      auth.oauth,
		production: auth.Production,
		now:        auth.now,
		random:     auth.random,
	}
	if service.now == nil {
		service.now = time.Now
	}
	if service.random == nil {
		service.random = rand.Reader
	}
	if service.store == nil {
		db, err := mutationDatabase(opts)
		if err != nil {
			service.storeErr = err
		} else {
			service.store = &postgresAuthStore{db: db}
		}
	}
	if service.flows == nil {
		if opts.Cache == nil {
			service.flowErr = apiError(
				http.StatusServiceUnavailable,
				"OAuth flow storage is not configured",
			)
		} else {
			service.flows = &redisOAuthFlowStore{client: opts.Cache}
		}
	}
	if service.oauth == nil {
		service.oauth = &sso.OAuthClient{
			ClientID: auth.ClientID, ClientSecret: auth.ClientSecret,
			CallbackURL: auth.CallbackURL, UserAgent: auth.UserAgent,
			HTTP: auth.HTTPClient,
		}
	}
	stateSecret := auth.StateSecret
	if stateSecret == "" {
		stateSecret = auth.ClientSecret
	}
	service.stateSecret = []byte(stateSecret)
	service.wallet = newWalletAuthorizationService(opts, service)
	return service
}

type signedOAuthState struct {
	Version  int              `json:"v"`
	Purpose  oauthFlowPurpose `json:"purpose"`
	ID       string           `json:"id"`
	IssuedAt int64            `json:"iat"`
}

func signOAuthState(state signedOAuthState, secret []byte) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("OAuth state secret is not configured")
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(encoded))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + signature, nil
}

func verifyOAuthState(
	raw string,
	secret []byte,
	now time.Time,
) (signedOAuthState, error) {
	if len(raw) == 0 || len(raw) > maxOAuthStateLength || len(secret) == 0 {
		return signedOAuthState{}, errOAuthFlow
	}
	payload, signature, found := strings.Cut(raw, ".")
	if !found || payload == "" || signature == "" || strings.Contains(signature, ".") {
		return signedOAuthState{}, errOAuthFlow
	}
	actualSignature, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil || len(actualSignature) != sha256.Size {
		return signedOAuthState{}, errOAuthFlow
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	if !hmac.Equal(actualSignature, mac.Sum(nil)) {
		return signedOAuthState{}, errOAuthFlow
	}
	decoded, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil || len(decoded) > 1024 {
		return signedOAuthState{}, errOAuthFlow
	}

	decoder := json.NewDecoder(strings.NewReader(string(decoded)))
	decoder.DisallowUnknownFields()
	var state signedOAuthState
	if err := decoder.Decode(&state); err != nil {
		return signedOAuthState{}, errOAuthFlow
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return signedOAuthState{}, errOAuthFlow
	}
	id, err := base64.RawURLEncoding.DecodeString(state.ID)
	if err != nil || len(id) != 32 ||
		state.Version != oauthStateVersion ||
		(state.Purpose != oauthStatePurposeLogin &&
			state.Purpose != oauthStatePurposeWallet) {
		return signedOAuthState{}, errOAuthFlow
	}
	issuedAt := time.Unix(state.IssuedAt, 0)
	if issuedAt.After(now.Add(oauthFutureClockSkew)) ||
		now.Sub(issuedAt) > oauthFlowTTL {
		return signedOAuthState{}, errOAuthFlow
	}
	return state, nil
}

type redisOAuthFlowStore struct {
	client *redis.Client
}

func (s *redisOAuthFlowStore) Put(
	ctx context.Context,
	id string,
	bindingHash string,
	flow pendingOAuthFlow,
	ttl time.Duration,
) error {
	payload, err := json.Marshal(flow)
	if err != nil {
		return err
	}
	value := bindingHash + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	stored, err := s.client.SetNX(
		ctx,
		redisOAuthFlowKeyPrefix+id,
		value,
		ttl,
	).Result()
	if err != nil {
		return err
	}
	if !stored {
		return errOAuthCollision
	}
	return nil
}

var takeOAuthFlowScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
  return nil
end
local dot = string.find(value, ".", 1, true)
if not dot or string.sub(value, 1, dot - 1) ~= ARGV[1] then
  return nil
end
redis.call("DEL", KEYS[1])
return string.sub(value, dot + 1)
`)

func (s *redisOAuthFlowStore) Take(
	ctx context.Context,
	id string,
	bindingHash string,
) (pendingOAuthFlow, error) {
	raw, err := takeOAuthFlowScript.Run(
		ctx,
		s.client,
		[]string{redisOAuthFlowKeyPrefix + id},
		bindingHash,
	).Text()
	if errors.Is(err, redis.Nil) {
		return pendingOAuthFlow{}, errOAuthFlow
	}
	if err != nil {
		return pendingOAuthFlow{}, err
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return pendingOAuthFlow{}, errOAuthFlow
	}
	var flow pendingOAuthFlow
	if err := json.Unmarshal(payload, &flow); err != nil {
		return pendingOAuthFlow{}, errOAuthFlow
	}
	return flow, nil
}

func normalizeReturnTo(raw string) (string, error) {
	if raw == "" {
		return "/", nil
	}
	if len(raw) > maxReturnToLength ||
		!strings.HasPrefix(raw, "/") ||
		strings.HasPrefix(raw, "//") ||
		strings.Contains(raw, "\\") {
		return "", apiError(http.StatusBadRequest, "Invalid redirect")
	}
	for _, r := range raw {
		if unicode.IsControl(r) {
			return "", apiError(http.StatusBadRequest, "Invalid redirect")
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Opaque != "" {
		return "", apiError(http.StatusBadRequest, "Invalid redirect")
	}
	if parsed.Path == "" ||
		!strings.HasPrefix(parsed.Path, "/") ||
		strings.HasPrefix(parsed.Path, "//") ||
		strings.Contains(parsed.Path, "\\") {
		return "", apiError(http.StatusBadRequest, "Invalid redirect")
	}
	for _, r := range parsed.Path {
		if unicode.IsControl(r) {
			return "", apiError(http.StatusBadRequest, "Invalid redirect")
		}
	}
	return parsed.String(), nil
}

func appendLoginMarker(returnTo string) string {
	parsed, err := url.Parse(returnTo)
	if err != nil {
		return "/?ek_login=ok"
	}
	query := parsed.Query()
	query.Set("ek_login", "ok")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func normalizeLoginDelay(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	switch value {
	case 0, 1, 3, 6, 12, 24, 72:
		return value
	default:
		return 0
	}
}

func randomBase64(reader io.Reader, size int) (string, error) {
	value := make([]byte, size)
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func bindingDigest(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func oauthFlowCookieName(id string) string {
	return oauthFlowCookiePrefix + id
}

func requestCookie(ctx huma.Context, name string) (string, bool) {
	cookies, err := http.ParseCookie(ctx.Header("Cookie"))
	if err != nil {
		return "", false
	}
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie.Value, true
		}
	}
	return "", false
}

func authCookie(
	name string,
	value string,
	maxAge int,
	httpOnly bool,
	production bool,
	path string,
) *http.Cookie {
	cookie := &http.Cookie{
		Name: name, Value: value, Path: path,
		MaxAge: maxAge, HttpOnly: httpOnly,
		Secure: production, SameSite: http.SameSiteLaxMode,
	}
	if production {
		cookie.Domain = ".eve-kill.com"
	}
	if maxAge < 0 {
		cookie.Expires = time.Unix(1, 0).UTC()
	}
	return cookie
}

func oauthFlowCookie(
	name string,
	value string,
	maxAge int,
	production bool,
) *http.Cookie {
	cookie := authCookie(name, value, maxAge, true, production, "/auth")
	// The callback and login start live on the same host. Keeping the binding
	// cookie host-only prevents a sibling subdomain from shadowing it; the
	// long-lived session cookies retain their legacy shared-domain behavior.
	cookie.Domain = ""
	return cookie
}

func appendCookie(ctx huma.Context, cookie *http.Cookie) {
	ctx.AppendHeader("Set-Cookie", cookie.String())
}

func clearAuthCookies(ctx huma.Context, production bool) {
	appendCookie(ctx, authCookie(
		authSessionCookie, "", -1, true, production, "/",
	))
	appendCookie(ctx, authCookie(
		authHintCookie, "", -1, false, production, "/",
	))
}

func setAccountNoStore(ctx huma.Context) {
	ctx.SetHeader("Cache-Control", "private, no-store")
	ctx.SetHeader("Pragma", "no-cache")
}

func requestSessionMetadata(ctx huma.Context) authSessionMetadata {
	ip := cleanRequestValue(ctx.Header("CF-Connecting-IP"), 128)
	if ip == nil {
		forwarded := strings.TrimSpace(strings.Split(ctx.Header("X-Forwarded-For"), ",")[0])
		ip = cleanOptionalValue(forwarded, 128)
	}
	if ip == nil {
		remoteAddr := ""
		if source, ok := any(ctx).(interface{ RemoteAddr() string }); ok {
			remoteAddr = source.RemoteAddr()
		}
		host, _, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			host = remoteAddr
		}
		ip = cleanOptionalValue(host, 128)
	}
	country := cleanRequestValue(ctx.Header("CF-IPCountry"), 2)
	if country != nil {
		upper := strings.ToUpper(*country)
		if len(upper) != 2 ||
			upper[0] < 'A' || upper[0] > 'Z' ||
			upper[1] < 'A' || upper[1] > 'Z' {
			country = nil
		} else {
			country = &upper
		}
	}
	return authSessionMetadata{
		IPAddress: ip, CountryCode: country,
		UserAgent:  cleanRequestValue(ctx.Header("User-Agent"), 1000),
		ClientHint: cleanRequestValue(ctx.Header("Sec-CH-UA"), 500),
	}
}

func cleanRequestValue(value string, maxLength int) *string {
	return cleanOptionalValue(strings.TrimSpace(value), maxLength)
}

func cleanOptionalValue(value string, maxLength int) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	runes := []rune(value)
	if len(runes) > maxLength {
		value = string(runes[:maxLength])
	}
	return &value
}

func metadataChanged(
	current authSessionActivity,
	next authSessionMetadata,
) bool {
	return changedNonEmpty(next.IPAddress, current.IPAddress) ||
		changedNonEmpty(next.CountryCode, current.CountryCode) ||
		changedNonEmpty(next.UserAgent, current.UserAgent) ||
		changedNonEmpty(next.ClientHint, current.ClientHint)
}

func changedNonEmpty(next, current *string) bool {
	return next != nil && (current == nil || *next != *current)
}

func pointerValue[T any](value *T) any {
	if value == nil {
		return nil
	}
	return *value
}

func principalJSON(principal Principal) map[string]any {
	return map[string]any{
		"characterId":            principal.CharacterID,
		"characterName":          principal.CharacterName,
		"isAdmin":                principal.IsAdmin,
		"corporationId":          pointerValue(principal.CorporationID),
		"corporationName":        pointerValue(principal.CorporationName),
		"allianceId":             pointerValue(principal.AllianceID),
		"allianceName":           pointerValue(principal.AllianceName),
		"lastSeenNotificationId": principal.LastSeenNotificationID,
	}
}

func (s *authService) resolvePrincipal(
	ctx context.Context,
	req *legacyRequest,
) (*Principal, error) {
	if s.storeErr != nil {
		return nil, s.storeErr
	}
	raw, ok := requestCookie(req.Huma, authSessionCookie)
	if !ok || raw == "" {
		return nil, nil
	}
	if _, err := uuid.Parse(raw); err != nil {
		clearAuthCookies(req.Huma, s.production)
		return nil, nil
	}

	now := s.now().UTC()
	principal, activity, err := s.store.ResolveSession(ctx, raw, now)
	if errors.Is(err, errNoAuthSession) {
		clearAuthCookies(req.Huma, s.production)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	metadata := requestSessionMetadata(req.Huma)
	if now.Sub(activity.LastSeenAt) >= authSessionTouchEvery ||
		metadataChanged(activity, metadata) {
		if err := s.store.TouchSession(
			ctx, activity.RowID, metadata, now,
		); err != nil {
			if errors.Is(err, errNoAuthSession) {
				clearAuthCookies(req.Huma, s.production)
				return nil, nil
			}
			return nil, err
		}
	}
	return &principal, nil
}

func (s *authService) requirePrincipal(
	ctx context.Context,
	req *legacyRequest,
) (*Principal, error) {
	principal, err := s.resolvePrincipal(ctx, req)
	if err != nil {
		return nil, err
	}
	if principal == nil {
		return nil, apiError(
			http.StatusUnauthorized,
			"Authentication required",
		)
	}
	return principal, nil
}

func (s *authService) beginLogin(
	ctx context.Context,
	req *legacyRequest,
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
	returnToRaw := req.Query.Get("returnTo")
	if returnToRaw == "" {
		returnToRaw = req.Query.Get("redirect")
	}
	returnTo, err := normalizeReturnTo(returnToRaw)
	if err != nil {
		return "", nil, err
	}
	flow := pendingOAuthFlow{
		Purpose:  oauthStatePurposeLogin,
		ReturnTo: returnTo,
		Delay:    normalizeLoginDelay(req.Query.Get("delay")),
		CharKM:   req.Query.Get("charKm") != "0",
		CorpKM:   req.Query.Get("corpKm") != "0",
		IssuedAt: s.now().UTC(),
	}

	var id, binding string
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
		Version: oauthStateVersion, Purpose: oauthStatePurposeLogin,
		ID: id, IssuedAt: flow.IssuedAt.Unix(),
	}, s.stateSecret)
	if err != nil {
		return "", nil, err
	}
	scopes := []string{"publicData"}
	if flow.CharKM {
		scopes = append(scopes, sso.ScopeCharacterKillmails)
	}
	if flow.CorpKM {
		scopes = append(scopes, sso.ScopeCorporationKillmails)
	}
	authorizationURL, err := s.oauth.BuildAuthorizationURL(state, scopes)
	if err != nil {
		return "", nil, err
	}
	cookie := oauthFlowCookie(
		oauthFlowCookieName(id),
		binding,
		int(oauthFlowTTL/time.Second),
		s.production,
	)
	return authorizationURL, cookie, nil
}

func (s *authService) completeCallback(
	ctx context.Context,
	req *legacyRequest,
) (legacyPayload, error) {
	setAccountNoStore(req.Huma)
	state, err := verifyOAuthState(
		req.Query.Get("state"), s.stateSecret, s.now().UTC(),
	)
	if err != nil {
		return legacyPayload{}, apiError(
			http.StatusBadRequest,
			"Invalid or expired OAuth state",
		)
	}
	binding, ok := requestCookie(req.Huma, oauthFlowCookieName(state.ID))
	if !ok {
		return legacyPayload{}, apiError(
			http.StatusBadRequest,
			"OAuth flow did not originate in this browser",
		)
	}
	if s.flowErr != nil {
		return legacyPayload{}, s.flowErr
	}
	if s.storeErr != nil {
		return legacyPayload{}, s.storeErr
	}
	flow, err := s.flows.Take(ctx, state.ID, bindingDigest(binding))
	if errors.Is(err, errOAuthFlow) {
		appendCookie(req.Huma, oauthFlowCookie(
			oauthFlowCookieName(state.ID), "", -1, s.production,
		))
		return legacyPayload{}, apiError(
			http.StatusBadRequest,
			"Invalid or expired OAuth flow",
		)
	}
	if err != nil {
		return legacyPayload{}, err
	}
	if flow.Purpose == "" {
		// Flows created during the transition predate the explicit discriminator
		// and are login flows by construction.
		flow.Purpose = oauthStatePurposeLogin
	}
	if flow.Purpose != state.Purpose {
		return legacyPayload{}, apiError(
			http.StatusBadRequest,
			"Invalid or expired OAuth flow",
		)
	}
	appendCookie(req.Huma, oauthFlowCookie(
		oauthFlowCookieName(state.ID), "", -1, s.production,
	))
	if flow.Purpose == oauthStatePurposeWallet {
		return s.wallet.complete(ctx, req, flow)
	}
	if req.Query.Get("error") != "" {
		return legacyPayload{}, apiError(
			http.StatusBadRequest,
			"EVE authorization was cancelled",
		)
	}
	code := strings.TrimSpace(req.Query.Get("code"))
	if code == "" {
		return legacyPayload{}, apiError(
			http.StatusBadRequest,
			"Missing code parameter",
		)
	}
	tokens, err := s.oauth.ExchangeCode(ctx, code)
	if err != nil {
		return legacyPayload{}, apiError(
			http.StatusBadGateway,
			"EVE SSO token exchange failed",
		)
	}
	claims, err := s.oauth.VerifyAccessToken(ctx, tokens.AccessToken)
	if err != nil {
		return legacyPayload{}, apiError(
			http.StatusUnauthorized,
			"Invalid EVE access token",
		)
	}
	sessionID, err := uuid.NewRandomFromReader(s.random)
	if err != nil {
		return legacyPayload{}, err
	}
	now := s.now().UTC()
	if err := s.store.CompleteLogin(ctx, authLoginCommit{
		SessionID: sessionID.String(), Now: now,
		ExpiresAt: now.Add(authSessionTTL), Delay: flow.Delay,
		Claims: claims, Tokens: tokens,
		Metadata: requestSessionMetadata(req.Huma),
	}); err != nil {
		return legacyPayload{}, err
	}

	appendCookie(req.Huma, authCookie(
		authSessionCookie,
		sessionID.String(),
		int(authSessionTTL/time.Second),
		true,
		s.production,
		"/",
	))
	hint := url.PathEscape(fmt.Sprintf(
		"%d:%s", claims.CharacterID, claims.CharacterName,
	))
	appendCookie(req.Huma, authCookie(
		authHintCookie,
		hint,
		int(authSessionTTL/time.Second),
		false,
		s.production,
		"/",
	))
	headers := make(http.Header)
	headers.Set("Location", appendLoginMarker(flow.ReturnTo))
	headers.Set("Cache-Control", "private, no-store")
	return legacyPayload{
		Status: http.StatusFound, Headers: headers, RawBody: []byte{},
	}, nil
}
