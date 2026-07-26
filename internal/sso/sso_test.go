package sso

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Two things here have teeth. Disabling a token that was only transiently
// unreachable costs a user their login. Failing to record a revoked scope costs
// the shared ESI error budget — 180 characters once produced 525,000 403s in a
// month exactly that way.

// jwtWith builds an access token carrying a scp claim.
//
// Unsigned, because the code deliberately does not verify the signature: the
// token arrives over TLS in the response to our own authenticated request, so
// there is no untrusted party in between. See ScopesFromAccessToken.
func jwtWith(scp any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	payload, _ := json.Marshal(map[string]any{"scp": scp})
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

// CCP sends a single scope as a bare string and several as an array. Handling
// only one of those silently loses every token in the other shape.
func TestScopesAreReadFromEitherClaimShape(t *testing.T) {
	cases := []struct {
		name string
		scp  any
		want []string
	}{
		{"array", []string{ScopeCharacterKillmails, ScopeCorporationKillmails},
			[]string{ScopeCharacterKillmails, ScopeCorporationKillmails}},
		{"space-separated string", ScopeCharacterKillmails + " " + ScopeCorporationKillmails,
			[]string{ScopeCharacterKillmails, ScopeCorporationKillmails}},
		{"single string", ScopeCharacterKillmails, []string{ScopeCharacterKillmails}},
		{"empty array", []string{}, nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ScopesFromAccessToken(jwtWith(c.scp))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Errorf("scope %d = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

// A token that is not a JWT must be an error rather than silently no scopes —
// no scopes means "disable this token", and disabling on a parse failure would
// lose a working login.
func TestMalformedAccessTokenIsAnError(t *testing.T) {
	for _, bad := range []string{"", "not-a-jwt", "only.two", "a.!!!.c"} {
		if _, err := ScopesFromAccessToken(bad); err == nil {
			t.Errorf("%q was accepted as a JWT", bad)
		}
	}
}

func TestCharacterIDIsReadFromAccessTokenSubject(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	payload := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"sub":"CHARACTER:EVE:90000001"}`),
	)
	got, err := CharacterIDFromAccessToken(header + "." + payload + ".signature")
	if err != nil {
		t.Fatal(err)
	}
	if got != 90000001 {
		t.Fatalf("character id = %d, want 90000001", got)
	}
}

func TestInvalidAccessTokenSubjectIsRejected(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	for _, subject := range []string{"", "CHARACTER:EVE:0", "USER:EVE:90000001", "CHARACTER:EVE:nope"} {
		payload, _ := json.Marshal(map[string]string{"sub": subject})
		token := header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
		if _, err := CharacterIDFromAccessToken(token); err == nil {
			t.Errorf("subject %q was accepted", subject)
		}
	}
}

// Usable subtracts locally revoked scopes from what SSO granted. This is the
// whole mechanism that stops the 403 loop.
func TestUsableSubtractsRevokedScopes(t *testing.T) {
	tok := Token{
		Scopes:        []string{ScopeCharacterKillmails, ScopeCorporationKillmails},
		RevokedScopes: []string{ScopeCorporationKillmails},
	}

	usable := tok.Usable()
	if len(usable) != 1 || usable[0] != ScopeCharacterKillmails {
		t.Fatalf("Usable() = %v, want only the character scope", usable)
	}
	if tok.Has(ScopeCorporationKillmails) {
		t.Error("a revoked scope is still reported as usable — the next refresh " +
			"would hand it back and the 403 loop resumes")
	}
	if !tok.Has(ScopeCharacterKillmails) {
		t.Error("an unrevoked scope was dropped")
	}
}

// ssoServer stands in for EVE SSO.
func ssoServer(t *testing.T, status int, body string) *Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The credentials go in a Basic header, not the form — a client that
		// puts them elsewhere is rejected by the real SSO.
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	return &Client{
		ClientID:     "client",
		ClientSecret: "secret",
		UserAgent:    "shrike-test/1.0",
		HTTP:         srv.Client(),
		TokenURL:     srv.URL,
	}
}

func TestRefreshReturnsTheNewTokens(t *testing.T) {
	c := ssoServer(t, http.StatusOK, `{
        "access_token": "new-access", "refresh_token": "new-refresh",
        "expires_in": 1199, "token_type": "Bearer"}`)

	got, err := c.Refresh(context.Background(), "old-refresh")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "new-access" {
		t.Errorf("access token = %q", got.AccessToken)
	}
	// The refresh token rotates, and storing the old one would break the next
	// refresh.
	if got.RefreshToken != "new-refresh" {
		t.Errorf("refresh token = %q, want the rotated one", got.RefreshToken)
	}
}

// A dead grant must be distinguishable, because it is the only case where the
// token is disabled rather than retried.
func TestPermanentlyDeadGrantsAreRecognised(t *testing.T) {
	for _, body := range []string{
		`{"error":"invalid_grant"}`,
		`{"error":"invalid_token","error_description":"Token missing"}`,
		`{"error_description":"refresh token expired"}`,
	} {
		c := ssoServer(t, http.StatusBadRequest, body)
		_, err := c.Refresh(context.Background(), "dead")
		if !errors.Is(err, ErrPermanentlyDead) {
			t.Errorf("body %s returned %v, want ErrPermanentlyDead", body, err)
		}
	}
}

// Everything else is transient. Disabling a token because SSO had a bad minute
// means the user has to log in again for no reason.
func TestTransientFailuresAreNotPermanent(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusTooManyRequests} {
		c := ssoServer(t, status, `{"error":"server_error"}`)
		_, err := c.Refresh(context.Background(), "fine")
		if err == nil {
			t.Errorf("status %d returned no error", status)
			continue
		}
		if errors.Is(err, ErrPermanentlyDead) {
			t.Errorf("status %d was treated as a dead grant — the token would be "+
				"disabled and the user forced to log in again", status)
		}
	}
}

// A 200 with no access token is not a success.
func TestEmptyTokenResponseIsAnError(t *testing.T) {
	c := ssoServer(t, http.StatusOK, `{"expires_in": 1199}`)
	if _, err := c.Refresh(context.Background(), "x"); err == nil {
		t.Error("a response with no access token was accepted")
	}
}

// Missing credentials must fail loudly rather than producing a confusing 401
// from SSO.
func TestMissingCredentialsFailClearly(t *testing.T) {
	c := &Client{UserAgent: "t"}
	_, err := c.Refresh(context.Background(), "x")
	if err == nil {
		t.Fatal("a client with no credentials attempted a refresh")
	}
	if !strings.Contains(err.Error(), "EVE_CLIENT_ID") {
		t.Errorf("the error does not name what is missing: %v", err)
	}
}
