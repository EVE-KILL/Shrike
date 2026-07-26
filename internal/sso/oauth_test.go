package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
)

func TestOAuthClientBuildAuthorizationURL(t *testing.T) {
	client := &OAuthClient{
		ClientID: "client", ClientSecret: "secret",
		CallbackURL:  "https://eve-kill.com/auth/eve/callback",
		AuthorizeURL: "https://sso.test/authorize",
	}
	raw, err := client.BuildAuthorizationURL(
		"signed-state",
		[]string{"publicData", ScopeCharacterKillmails},
	)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("response_type") != "code" ||
		parsed.Query().Get("client_id") != "client" ||
		parsed.Query().Get("redirect_uri") != client.CallbackURL ||
		parsed.Query().Get("state") != "signed-state" {
		t.Fatalf("authorization query = %v", parsed.Query())
	}
	if got := parsed.Query().Get("scope"); got !=
		"publicData "+ScopeCharacterKillmails {
		t.Fatalf("scope = %q", got)
	}
}

func TestOAuthClientDefaultHTTPClientIsBounded(t *testing.T) {
	client := (&OAuthClient{}).httpClient()
	if client.Timeout <= 0 {
		t.Fatal("default OAuth HTTP client has no timeout")
	}
}

func TestOAuthClientExchangeCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s", request.Method)
		}
		if got := request.Header.Get("Authorization"); got !=
			"Basic "+base64.StdEncoding.EncodeToString([]byte("client:secret")) {
			t.Errorf("authorization = %q", got)
		}
		if got := request.Header.Get("User-Agent"); got != "EVE-Kill/test" {
			t.Errorf("user agent = %q", got)
		}
		if err := request.ParseForm(); err != nil {
			t.Error(err)
		}
		if request.Form.Get("grant_type") != "authorization_code" ||
			request.Form.Get("code") != "one-time-code" {
			t.Errorf("form = %v", request.Form)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"access_token":  "access",
			"refresh_token": "refresh",
			"expires_in":    1200,
			"token_type":    "Bearer",
		})
	}))
	defer server.Close()

	client := &OAuthClient{
		ClientID: "client", ClientSecret: "secret",
		UserAgent: "EVE-Kill/test", TokenURL: server.URL,
	}
	tokens, err := client.ExchangeCode(context.Background(), "one-time-code")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "access" ||
		tokens.RefreshToken != "refresh" ||
		tokens.ExpiresIn != 1200 ||
		tokens.TokenType != "Bearer" {
		t.Fatalf("tokens = %#v", tokens)
	}
}

func TestOAuthClientVerifyAccessToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyServer := &oidctest.Server{PublicKeys: []oidctest.PublicKey{{
		PublicKey: privateKey.Public(),
		KeyID:     "test-key",
		Algorithm: oidc.RS256,
	}}}
	server := httptest.NewServer(keyServer)
	defer server.Close()
	keyServer.SetIssuer(server.URL)

	expires := time.Now().Add(time.Hour).Unix()
	claims := map[string]any{
		"iss":   "https://login.eveonline.com",
		"aud":   "client",
		"sub":   "CHARACTER:EVE:9001",
		"exp":   expires,
		"name":  "Test Pilot",
		"owner": "owner-hash",
		"scp":   []string{"publicData", ScopeCharacterKillmails},
	}
	token := signedOAuthTestToken(t, privateKey, claims)
	client := &OAuthClient{
		ClientID: "client",
		JWKSURL:  server.URL + "/keys",
	}
	got, err := client.VerifyAccessToken(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if got.CharacterID != 9001 ||
		got.CharacterName != "Test Pilot" ||
		got.CharacterOwnerHash != "owner-hash" ||
		!reflect.DeepEqual(got.Scopes, []string{
			"publicData", ScopeCharacterKillmails,
		}) {
		t.Fatalf("claims = %#v", got)
	}

	for name, mutate := range map[string]func(map[string]any){
		"issuer": func(c map[string]any) {
			c["iss"] = "https://attacker.invalid"
		},
		"audience": func(c map[string]any) {
			c["aud"] = "different-client"
		},
		"expired": func(c map[string]any) {
			c["exp"] = time.Now().Add(-time.Hour).Unix()
		},
		"subject": func(c map[string]any) {
			c["sub"] = "USER:9001"
		},
	} {
		t.Run(name, func(t *testing.T) {
			copy := make(map[string]any, len(claims))
			for key, value := range claims {
				copy[key] = value
			}
			mutate(copy)
			raw := signedOAuthTestToken(t, privateKey, copy)
			verifier := &OAuthClient{
				ClientID: "client",
				JWKSURL:  server.URL + "/keys",
			}
			if _, err := verifier.VerifyAccessToken(context.Background(), raw); err == nil {
				t.Fatal("invalid token was accepted")
			}
		})
	}

	parts := strings.Split(token, ".")
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	payload[0] ^= 1
	parts[1] = base64.RawURLEncoding.EncodeToString(payload)
	tampered := strings.Join(parts, ".")
	if _, err := (&OAuthClient{
		ClientID: "client", JWKSURL: server.URL + "/keys",
	}).VerifyAccessToken(context.Background(), tampered); err == nil {
		t.Fatal("tampered signature was accepted")
	}
}

func TestOAuthClientAcceptsStringScopeClaim(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyServer := &oidctest.Server{PublicKeys: []oidctest.PublicKey{{
		PublicKey: privateKey.Public(), KeyID: "test-key",
		Algorithm: oidc.RS256,
	}}}
	server := httptest.NewServer(keyServer)
	defer server.Close()
	keyServer.SetIssuer(server.URL)

	token := signedOAuthTestToken(t, privateKey, map[string]any{
		"iss":   "login.eveonline.com",
		"aud":   "client",
		"sub":   "CHARACTER:EVE:9001",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"name":  "Test Pilot",
		"owner": "owner-hash",
		"scp":   "publicData " + ScopeCorporationKillmails,
	})
	got, err := (&OAuthClient{
		ClientID: "client", JWKSURL: server.URL + "/keys",
	}).VerifyAccessToken(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"publicData", ScopeCorporationKillmails}; !reflect.DeepEqual(got.Scopes, want) {
		t.Fatalf("scopes = %v, want %v", got.Scopes, want)
	}
}

func signedOAuthTestToken(
	t *testing.T,
	key *rsa.PrivateKey,
	claims map[string]any,
) string {
	t.Helper()
	raw, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	return oidctest.SignIDToken(
		key, "test-key", oidc.RS256, string(raw),
	)
}
