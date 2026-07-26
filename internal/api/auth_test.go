package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/eve-kill/shrike/internal/sso"
)

func TestNormalizeReturnToRejectsOpenRedirects(t *testing.T) {
	for _, value := range []string{
		"/",
		"/campaign/42?tab=overview#kills",
		"/settings/sessions",
	} {
		got, err := normalizeReturnTo(value)
		if err != nil || got != value {
			t.Errorf("normalizeReturnTo(%q) = %q, %v", value, got, err)
		}
	}

	for _, value := range []string{
		"https://evil.example/",
		"//evil.example/",
		"///evil.example/",
		"/%2f%2fevil.example/",
		"/\\evil.example/",
		"/%5cevil.example/",
		"relative/path",
		"/ok\nLocation: https://evil.example",
	} {
		if got, err := normalizeReturnTo(value); err == nil {
			t.Errorf("normalizeReturnTo(%q) = %q, want rejection", value, got)
		}
	}
}

func TestOAuthStateStrictSignatureAndLifetime(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	id := base64ID(bytes.Repeat([]byte{7}, 32))
	secret := []byte("a distinct state-signing secret")
	raw, err := signOAuthState(signedOAuthState{
		Version: oauthStateVersion, Purpose: oauthStatePurposeLogin,
		ID: id, IssuedAt: now.Unix(),
	}, secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifyOAuthState(raw, secret, now); err != nil {
		t.Fatalf("valid state rejected: %v", err)
	}

	payload, signature, _ := strings.Cut(raw, ".")
	tampered := payload + "." + signature[:len(signature)-1] + "A"
	if hmac.Equal([]byte(tampered), []byte(raw)) {
		t.Fatal("test did not alter the state")
	}
	if _, err := verifyOAuthState(tampered, secret, now); !errors.Is(err, errOAuthFlow) {
		t.Fatalf("tampered state error = %v", err)
	}
	if _, err := verifyOAuthState(raw, []byte("wrong secret"), now); !errors.Is(err, errOAuthFlow) {
		t.Fatalf("wrong-secret state error = %v", err)
	}
	if _, err := verifyOAuthState(raw, secret, now.Add(oauthFlowTTL+time.Second)); !errors.Is(err, errOAuthFlow) {
		t.Fatalf("expired state error = %v", err)
	}

	future, err := signOAuthState(signedOAuthState{
		Version: oauthStateVersion, Purpose: oauthStatePurposeLogin,
		ID: id, IssuedAt: now.Add(oauthFutureClockSkew + time.Second).Unix(),
	}, secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifyOAuthState(future, secret, now); !errors.Is(err, errOAuthFlow) {
		t.Fatalf("future state error = %v", err)
	}
}

func TestLoginFlowIsBrowserBoundOneTimeAndTransactionalAtStoreBoundary(t *testing.T) {
	rig := newAuthTestRig(t)
	handler := rig.handler(t)

	start := httptest.NewRequest(
		http.MethodGet,
		"/auth/login?redirect="+url.QueryEscape("/campaign/42?x=1#kills")+
			"&charKm=0&corpKm=1&delay=6",
		nil,
	)
	start.Header.Set("User-Agent", "Test Browser")
	startResponse := httptest.NewRecorder()
	handler.ServeHTTP(startResponse, start)
	if startResponse.Code != http.StatusOK {
		t.Fatalf("login start = %d: %s", startResponse.Code, startResponse.Body.String())
	}
	if got := startResponse.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("cache control = %q", got)
	}

	var loginBody struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(startResponse.Body.Bytes(), &loginBody); err != nil {
		t.Fatal(err)
	}
	authorizationURL, err := url.Parse(loginBody.URL)
	if err != nil {
		t.Fatal(err)
	}
	state := authorizationURL.Query().Get("state")
	if state == "" {
		t.Fatal("authorization URL has no state")
	}
	if got, want := rig.oauth.scopes, []string{
		"publicData", sso.ScopeCorporationKillmails,
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("scopes = %v, want %v", got, want)
	}
	flowCookie := cookieWithPrefix(startResponse.Result().Cookies(), oauthFlowCookiePrefix)
	if flowCookie == nil || !flowCookie.HttpOnly || flowCookie.Path != "/auth" {
		t.Fatalf("flow cookie = %#v", flowCookie)
	}

	stateInfo, err := verifyOAuthState(state, rig.secret, rig.now)
	if err != nil {
		t.Fatal(err)
	}
	wrongBrowser := httptest.NewRequest(
		http.MethodGet,
		"/auth/eve/callback?code=good&state="+url.QueryEscape(state),
		nil,
	)
	wrongBrowser.AddCookie(&http.Cookie{
		Name: oauthFlowCookieName(stateInfo.ID), Value: "wrong-browser",
	})
	wrongResponse := httptest.NewRecorder()
	handler.ServeHTTP(wrongResponse, wrongBrowser)
	if wrongResponse.Code != http.StatusBadRequest {
		t.Fatalf("wrong-browser callback = %d: %s", wrongResponse.Code, wrongResponse.Body.String())
	}

	callback := httptest.NewRequest(
		http.MethodGet,
		"/auth/callback?code=good&state="+url.QueryEscape(state),
		nil,
	)
	callback.AddCookie(flowCookie)
	callback.Header.Set("CF-Connecting-IP", "192.0.2.44")
	callbackResponse := httptest.NewRecorder()
	handler.ServeHTTP(callbackResponse, callback)
	if callbackResponse.Code != http.StatusFound {
		t.Fatalf("callback = %d: %s", callbackResponse.Code, callbackResponse.Body.String())
	}
	location, err := url.Parse(callbackResponse.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Path != "/campaign/42" ||
		location.Query().Get("x") != "1" ||
		location.Query().Get("ek_login") != "ok" ||
		location.Fragment != "kills" {
		t.Fatalf("callback location = %q", location.String())
	}
	if rig.store.completed == nil {
		t.Fatal("login was not committed")
	}
	if got := rig.store.completed.Delay; got != 6 {
		t.Errorf("stored delay = %d", got)
	}
	if got := valueOrEmpty(rig.store.completed.Metadata.IPAddress); got != "192.0.2.44" {
		t.Errorf("stored IP = %q", got)
	}
	if rig.store.completeCalls != 1 {
		t.Fatalf("CompleteLogin calls = %d, want exactly one", rig.store.completeCalls)
	}
	if cookieNamed(callbackResponse.Result().Cookies(), authSessionCookie) == nil {
		t.Error("callback did not set the bearer session cookie")
	}
	if cookieNamed(callbackResponse.Result().Cookies(), authHintCookie) == nil {
		t.Error("callback did not set the auth hint cookie")
	}

	replay := httptest.NewRequest(
		http.MethodGet,
		"/auth/eve/callback?code=good&state="+url.QueryEscape(state),
		nil,
	)
	replay.AddCookie(flowCookie)
	replayResponse := httptest.NewRecorder()
	handler.ServeHTTP(replayResponse, replay)
	if replayResponse.Code != http.StatusBadRequest {
		t.Fatalf("replayed callback = %d: %s", replayResponse.Code, replayResponse.Body.String())
	}
	if rig.store.completeCalls != 1 {
		t.Fatalf("replay committed login; calls = %d", rig.store.completeCalls)
	}
}

func TestLoginRejectsOpenRedirectBeforeCreatingFlow(t *testing.T) {
	rig := newAuthTestRig(t)
	handler := rig.handler(t)
	request := httptest.NewRequest(
		http.MethodGet,
		"/auth/eve/start?returnTo="+url.QueryEscape("https://evil.example/"),
		nil,
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	if len(rig.flows.items) != 0 {
		t.Fatalf("invalid redirect created %d OAuth flows", len(rig.flows.items))
	}
}

func TestCanonicalMeIsLightweightAndLegacyAliasLoadsSettings(t *testing.T) {
	rig := newAuthTestRig(t)
	rig.store.principal = &Principal{
		CharacterID: 9001, CharacterName: "Test Pilot", IsAdmin: true,
		LastSeenNotificationID: 77,
	}
	rig.store.details = authMeDetails{
		CharacterOwnerHash: "owner-hash",
		Settings:           map[string]any{"theme": map[string]any{"dark": true}},
		CreatedAt:          rig.now.Add(-24 * time.Hour),
	}
	handler := rig.handler(t)
	session := &http.Cookie{
		Name: authSessionCookie, Value: "b4529550-a8a5-43d8-8342-909719305ef0",
	}

	canonical := httptest.NewRequest(http.MethodGet, "/me", nil)
	canonical.AddCookie(session)
	canonicalResponse := httptest.NewRecorder()
	handler.ServeHTTP(canonicalResponse, canonical)
	if canonicalResponse.Code != http.StatusOK {
		t.Fatalf("canonical me = %d: %s", canonicalResponse.Code, canonicalResponse.Body.String())
	}
	canonicalUser := responseUser(t, canonicalResponse)
	if _, found := canonicalUser["settings"]; found {
		t.Error("canonical /me unexpectedly loaded settings")
	}
	if _, found := canonicalUser["characterOwnerHash"]; found {
		t.Error("canonical /me unexpectedly exposed owner hash")
	}
	if rig.store.detailsCalls != 0 {
		t.Fatalf("canonical /me made %d detail queries", rig.store.detailsCalls)
	}

	legacy := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	legacy.AddCookie(session)
	legacyResponse := httptest.NewRecorder()
	handler.ServeHTTP(legacyResponse, legacy)
	if legacyResponse.Code != http.StatusOK {
		t.Fatalf("legacy me = %d: %s", legacyResponse.Code, legacyResponse.Body.String())
	}
	legacyUser := responseUser(t, legacyResponse)
	if legacyUser["characterOwnerHash"] != "owner-hash" || legacyUser["settings"] == nil {
		t.Fatalf("legacy user = %#v", legacyUser)
	}
	if rig.store.detailsCalls != 1 {
		t.Fatalf("legacy details calls = %d", rig.store.detailsCalls)
	}
}

func TestAuthInfrastructureErrorsAreNotAnonymous(t *testing.T) {
	rig := newAuthTestRig(t)
	rig.store.resolveErr = errors.New("database unavailable")
	handler := rig.handler(t)
	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	request.AddCookie(&http.Cookie{
		Name: authSessionCookie, Value: "b4529550-a8a5-43d8-8342-909719305ef0",
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", response.Code, response.Body.String())
	}
}

func TestSessionRevokedDuringActivityTouchBecomesAnonymous(t *testing.T) {
	rig := newAuthTestRig(t)
	rig.store.principal = &Principal{
		CharacterID: 9001, CharacterName: "Test Pilot",
	}
	rig.store.touchErr = errNoAuthSession
	handler := rig.handler(t)

	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	request.AddCookie(&http.Cookie{
		Name: authSessionCookie, Value: "b4529550-a8a5-43d8-8342-909719305ef0",
	})
	request.Header.Set("User-Agent", "Changed browser metadata")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		User any `json:"user"`
	}
	decodeResponse(t, response, &body)
	if body.User != nil {
		t.Fatalf("user = %#v, want anonymous", body.User)
	}
	if cookieNamed(response.Result().Cookies(), authSessionCookie) == nil ||
		cookieNamed(response.Result().Cookies(), authHintCookie) == nil {
		t.Fatal("revoked session did not clear both browser cookies")
	}
}

func TestOAuthFlowCookieIsHostOnlyInProduction(t *testing.T) {
	flow := oauthFlowCookie("flow", "binding", 300, true)
	if !flow.HttpOnly || !flow.Secure || flow.Path != "/auth" {
		t.Fatalf("flow cookie security attributes = %#v", flow)
	}
	if flow.Domain != "" {
		t.Fatalf("flow cookie domain = %q, want host-only", flow.Domain)
	}

	session := authCookie(
		authSessionCookie, "session", 300, true, true, "/",
	)
	if session.Domain != ".eve-kill.com" {
		t.Fatalf("session cookie domain = %q", session.Domain)
	}
}

func TestMutationOriginCannotBeOverriddenByForwardedHost(t *testing.T) {
	rig := newAuthTestRig(t)
	handler := rig.handler(t)
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.Host = "eve-kill.com"
	request.Header.Set("Origin", "https://evil.example")
	request.Header.Set("X-Forwarded-Host", "evil.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", response.Code, response.Body.String())
	}
}

func TestSettingsTokenAndSessionCompatibilityRoutes(t *testing.T) {
	rig := newAuthTestRig(t)
	rig.store.principal = &Principal{
		CharacterID: 9001, CharacterName: "Test Pilot",
	}
	rig.store.details = authMeDetails{
		CharacterOwnerHash: "owner-hash",
		Settings:           map[string]any{"defaultTabs": map[string]any{}},
		CreatedAt:          rig.now.Add(-time.Hour),
	}
	expiry := rig.now.Add(time.Hour)
	rig.store.token = authTokenInfo{
		Scopes:        []string{"publicData", "scope.a"},
		RevokedScopes: []string{"scope.a"},
		TokenExpiry:   &expiry,
	}
	rig.store.tokenFound = true
	ua := "Mozilla/5.0 (iPhone) CriOS/120 Mobile"
	rig.store.sessions = []authSessionSummary{{
		ID: 8, Current: true, UserAgent: &ua,
		CreatedAt:  rig.now.Add(-time.Hour),
		LastSeenAt: rig.now, ExpiresAt: rig.now.Add(authSessionTTL),
	}}
	handler := rig.handler(t)
	session := &http.Cookie{
		Name: authSessionCookie, Value: "b4529550-a8a5-43d8-8342-909719305ef0",
	}

	settings := requestWithCookie(http.MethodGet, "/me/settings", session)
	settingsResponse := httptest.NewRecorder()
	handler.ServeHTTP(settingsResponse, settings)
	if settingsResponse.Code != http.StatusOK {
		t.Fatalf("settings = %d: %s", settingsResponse.Code, settingsResponse.Body.String())
	}
	var settingsBody map[string]any
	decodeResponse(t, settingsResponse, &settingsBody)
	token := settingsBody["esiToken"].(map[string]any)
	if token["scopeCount"] != float64(1) {
		t.Fatalf("settings token = %#v", token)
	}

	tokenInfo := requestWithCookie(http.MethodGet, "/auth/token-info", session)
	tokenResponse := httptest.NewRecorder()
	handler.ServeHTTP(tokenResponse, tokenInfo)
	if tokenResponse.Code != http.StatusOK {
		t.Fatalf("token info = %d: %s", tokenResponse.Code, tokenResponse.Body.String())
	}

	sessions := requestWithCookie(http.MethodGet, "/user/sessions", session)
	sessionsResponse := httptest.NewRecorder()
	handler.ServeHTTP(sessionsResponse, sessions)
	if sessionsResponse.Code != http.StatusOK {
		t.Fatalf("sessions = %d: %s", sessionsResponse.Code, sessionsResponse.Body.String())
	}
	var sessionsBody struct {
		Sessions []map[string]any `json:"sessions"`
	}
	decodeResponse(t, sessionsResponse, &sessionsBody)
	if got := sessionsBody.Sessions[0]["browser"]; got != "Chrome" {
		t.Fatalf("browser = %v", got)
	}
	if got := sessionsBody.Sessions[0]["device"]; got != "Mobile" {
		t.Fatalf("device = %v", got)
	}

	missingGuard := requestWithCookie(http.MethodDelete, "/me/sessions", session)
	missingGuardResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingGuardResponse, missingGuard)
	if missingGuardResponse.Code != http.StatusBadRequest {
		t.Fatalf("unguarded delete = %d", missingGuardResponse.Code)
	}

	revokeOthers := requestWithCookie(
		http.MethodDelete, "/me/sessions?except=current", session,
	)
	revokeOthersResponse := httptest.NewRecorder()
	handler.ServeHTTP(revokeOthersResponse, revokeOthers)
	if revokeOthersResponse.Code != http.StatusOK {
		t.Fatalf("revoke others = %d: %s", revokeOthersResponse.Code, revokeOthersResponse.Body.String())
	}
	if rig.store.revokeOthersCalls != 1 {
		t.Fatalf("revoke others calls = %d", rig.store.revokeOthersCalls)
	}
}

func TestLogoutClearsCookiesEvenWhenRevocationFails(t *testing.T) {
	rig := newAuthTestRig(t)
	rig.store.deleteErr = errors.New("database unavailable")
	handler := rig.handler(t)
	session := &http.Cookie{
		Name: authSessionCookie, Value: "b4529550-a8a5-43d8-8342-909719305ef0",
	}
	request := requestWithCookie(http.MethodPost, "/auth/logout", session)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("logout status = %d: %s", response.Code, response.Body.String())
	}
	if cookieNamed(response.Result().Cookies(), authSessionCookie) == nil ||
		cookieNamed(response.Result().Cookies(), authHintCookie) == nil {
		t.Fatal("logout failure did not clear both browser cookies")
	}
}

func base64ID(raw []byte) string {
	return base64.RawURLEncoding.EncodeToString(raw)
}

type authTestRig struct {
	now    time.Time
	secret []byte
	store  *fakeAuthStore
	flows  *fakeOAuthFlowStore
	oauth  *fakeOAuthCodeClient
}

func newAuthTestRig(t *testing.T) *authTestRig {
	t.Helper()
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	return &authTestRig{
		now:    now,
		secret: []byte("test state secret"),
		store:  &fakeAuthStore{now: now},
		flows:  &fakeOAuthFlowStore{items: map[string]fakeOAuthFlow{}},
		oauth: &fakeOAuthCodeClient{
			tokens: sso.AuthorizationCodeTokens{
				AccessToken: "access", RefreshToken: "refresh",
				ExpiresIn: 1200, TokenType: "Bearer",
			},
			claims: sso.AccessClaims{
				CharacterID: 9001, CharacterName: "Test Pilot",
				CharacterOwnerHash: "owner-hash",
				Scopes:             []string{"publicData"},
			},
		},
	}
}

func (r *authTestRig) handler(t *testing.T) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	cfg := huma.DefaultConfig("auth test", "test")
	cfg.OpenAPIPath = ""
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	api := humago.New(mux, cfg)
	registerAuthRoutes(api, Options{
		Auth: AuthOptions{
			ClientID: "client", ClientSecret: "client-secret",
			CallbackURL: "https://eve-kill.com/auth/eve/callback",
			StateSecret: string(r.secret),
			store:       r.store, flowStore: r.flows, oauth: r.oauth,
			now: func() time.Time { return r.now },
		},
	})
	return mux
}

type fakeOAuthFlow struct {
	binding string
	flow    pendingOAuthFlow
}

type fakeOAuthFlowStore struct {
	items map[string]fakeOAuthFlow
}

func (s *fakeOAuthFlowStore) Put(
	_ context.Context,
	id string,
	binding string,
	flow pendingOAuthFlow,
	_ time.Duration,
) error {
	if _, exists := s.items[id]; exists {
		return errOAuthCollision
	}
	s.items[id] = fakeOAuthFlow{binding: binding, flow: flow}
	return nil
}

func (s *fakeOAuthFlowStore) Take(
	_ context.Context,
	id string,
	binding string,
) (pendingOAuthFlow, error) {
	item, exists := s.items[id]
	if !exists || !hmac.Equal([]byte(item.binding), []byte(binding)) {
		return pendingOAuthFlow{}, errOAuthFlow
	}
	delete(s.items, id)
	return item.flow, nil
}

type fakeOAuthCodeClient struct {
	scopes      []string
	tokens      sso.AuthorizationCodeTokens
	claims      sso.AccessClaims
	exchangeErr error
	verifyErr   error
}

func (c *fakeOAuthCodeClient) BuildAuthorizationURL(
	state string,
	scopes []string,
) (string, error) {
	c.scopes = append([]string(nil), scopes...)
	query := url.Values{"state": {state}}
	return "https://sso.test/authorize?" + query.Encode(), nil
}

func (c *fakeOAuthCodeClient) ExchangeCode(
	context.Context,
	string,
) (sso.AuthorizationCodeTokens, error) {
	return c.tokens, c.exchangeErr
}

func (c *fakeOAuthCodeClient) VerifyAccessToken(
	context.Context,
	string,
) (sso.AccessClaims, error) {
	return c.claims, c.verifyErr
}

type fakeAuthStore struct {
	now               time.Time
	principal         *Principal
	resolveErr        error
	touchErr          error
	details           authMeDetails
	detailsCalls      int
	token             authTokenInfo
	tokenFound        bool
	completed         *authLoginCommit
	completeCalls     int
	completeErr       error
	deleteErr         error
	sessions          []authSessionSummary
	revokeFound       bool
	revokeCurrent     bool
	revokeErr         error
	revokeOthers      int64
	revokeOthersCalls int
	revokeOthersErr   error
}

func (s *fakeAuthStore) ResolveSession(
	context.Context,
	string,
	time.Time,
) (Principal, authSessionActivity, error) {
	if s.resolveErr != nil {
		return Principal{}, authSessionActivity{}, s.resolveErr
	}
	if s.principal == nil {
		return Principal{}, authSessionActivity{}, errNoAuthSession
	}
	return *s.principal, authSessionActivity{
		RowID: 1, LastSeenAt: s.now,
	}, nil
}

func (s *fakeAuthStore) TouchSession(
	context.Context,
	int64,
	authSessionMetadata,
	time.Time,
) error {
	return s.touchErr
}

func (s *fakeAuthStore) LoadMeDetails(
	context.Context,
	int32,
) (authMeDetails, error) {
	s.detailsCalls++
	return s.details, nil
}

func (s *fakeAuthStore) LoadTokenInfo(
	context.Context,
	int32,
) (authTokenInfo, bool, error) {
	return s.token, s.tokenFound, nil
}

func (s *fakeAuthStore) CompleteLogin(
	_ context.Context,
	login authLoginCommit,
) error {
	s.completeCalls++
	copy := login
	s.completed = &copy
	return s.completeErr
}

func (s *fakeAuthStore) DeleteSession(context.Context, string) error {
	return s.deleteErr
}

func (s *fakeAuthStore) ListSessions(
	context.Context,
	int32,
	string,
	time.Time,
) ([]authSessionSummary, error) {
	return append([]authSessionSummary(nil), s.sessions...), nil
}

func (s *fakeAuthStore) RevokeSession(
	context.Context,
	int32,
	int64,
	string,
) (bool, bool, error) {
	return s.revokeFound, s.revokeCurrent, s.revokeErr
}

func (s *fakeAuthStore) RevokeOtherSessions(
	context.Context,
	int32,
	string,
) (int64, error) {
	s.revokeOthersCalls++
	return s.revokeOthers, s.revokeOthersErr
}

func requestWithCookie(method, target string, cookie *http.Cookie) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	request.AddCookie(cookie)
	return request
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), out); err != nil {
		t.Fatalf("decode %q: %v", response.Body.String(), err)
	}
}

func responseUser(
	t *testing.T,
	response *httptest.ResponseRecorder,
) map[string]any {
	t.Helper()
	var body struct {
		User map[string]any `json:"user"`
	}
	decodeResponse(t, response, &body)
	return body.User
}

func cookieNamed(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func cookieWithPrefix(cookies []*http.Cookie, prefix string) *http.Cookie {
	for _, cookie := range cookies {
		if strings.HasPrefix(cookie.Name, prefix) {
			return cookie
		}
	}
	return nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
