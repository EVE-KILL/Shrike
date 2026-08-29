package ingress

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/rs/zerolog"
)

func TestNginxAccessLineUsesForwardedIPAndRedactsSecrets(t *testing.T) {
	requestURL, err := url.Parse("/auth/eve/callback?code=super-secret&foo=bar&state=csrf-secret")
	if err != nil {
		t.Fatal(err)
	}
	request := &http.Request{
		Method:     http.MethodGet,
		URL:        requestURL,
		Proto:      "HTTP/2.0",
		RemoteAddr: "10.0.0.7:54321",
		Header:     make(http.Header),
	}
	request.Header.Set("CF-Connecting-IP", "203.0.113.42")
	request.Header.Set("Referer", "https://eve-kill.com/login")
	request.Header.Set("User-Agent", "Mozilla/5.0")

	line := nginxAccessLine(
		request,
		http.StatusFound,
		123,
		time.Date(2026, time.July, 27, 18, 30, 0, 0, time.FixedZone("CEST", 2*60*60)),
		1250*time.Millisecond,
	)

	for _, want := range []string{
		`203.0.113.42 - - [27/Jul/2026:18:30:00 +0200]`,
		`"GET /auth/eve/callback?code=REDACTED&foo=bar&state=REDACTED HTTP/2.0"`,
		`302 123`,
		`"https://eve-kill.com/login" "Mozilla/5.0" 1.250`,
	} {
		if !strings.Contains(line, want) {
			t.Errorf("access line %q does not contain %q", line, want)
		}
	}
	if strings.Contains(line, "super-secret") || strings.Contains(line, "csrf-secret") {
		t.Errorf("access line leaked a secret: %s", line)
	}
}

func TestAccessLogHandlerEmitsInfoAndErrorLevels(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		handlerErr error
		wantLevel  string
		wantCode   string
	}{
		{
			name:      "successful request",
			status:    http.StatusCreated,
			wantLevel: "info",
			wantCode:  " 201 ",
		},
		{
			name:      "server error response",
			status:    http.StatusServiceUnavailable,
			wantLevel: "error",
			wantCode:  " 503 ",
		},
		{
			name:       "caddy handler error",
			handlerErr: caddyhttp.Error(http.StatusBadGateway, errors.New("renderer unavailable")),
			wantLevel:  "error",
			wantCode:   " 502 ",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			handler := &accessLogHandler{log: zerolog.New(&output)}
			next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) error {
				if tc.status != 0 {
					w.WriteHeader(tc.status)
				}
				return tc.handlerErr
			})
			request := httptest.NewRequest(http.MethodGet, "http://eve-kill.test/api/test", nil)
			response := httptest.NewRecorder()

			err := handler.ServeHTTP(response, request, next)
			if tc.handlerErr == nil && err != nil {
				t.Fatalf("ServeHTTP: %v", err)
			}
			if tc.handlerErr != nil && err == nil {
				t.Fatal("ServeHTTP swallowed the handler error")
			}

			var record map[string]any
			if decodeErr := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record); decodeErr != nil {
				t.Fatalf("decode log %q: %v", output.String(), decodeErr)
			}
			if got := record["level"]; got != tc.wantLevel {
				t.Errorf("level = %v, want %q", got, tc.wantLevel)
			}
			message, _ := record["message"].(string)
			if !strings.Contains(message, tc.wantCode) {
				t.Errorf("message %q does not contain status %q", message, tc.wantCode)
			}
		})
	}
}

func TestAccessLogHandlerSuppressesOnlyHealthyProbe(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		wantOutput bool
	}{
		{name: "healthy", status: http.StatusOK, wantOutput: false},
		{name: "unhealthy", status: http.StatusServiceUnavailable, wantOutput: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			handler := &accessLogHandler{log: zerolog.New(&output)}
			request := httptest.NewRequest(http.MethodGet, "http://eve-kill.test/health", nil)
			response := httptest.NewRecorder()
			err := handler.ServeHTTP(response, request, caddyhttp.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) error {
					w.WriteHeader(tc.status)
					return nil
				},
			))
			if err != nil {
				t.Fatal(err)
			}
			if got := output.Len() > 0; got != tc.wantOutput {
				t.Errorf("logged = %v, want %v: %s", got, tc.wantOutput, output.String())
			}
		})
	}
}

func TestAccessLogHandlerDoesNotWrapWebSocketUpgrade(t *testing.T) {
	var output bytes.Buffer
	handler := &accessLogHandler{log: zerolog.New(&output)}
	request := httptest.NewRequest(http.MethodGet, "http://eve-kill.test/ws/status", nil)
	request.Header.Set("Connection", "keep-alive, Upgrade")
	request.Header.Set("Upgrade", "websocket")
	response := httptest.NewRecorder()

	err := handler.ServeHTTP(response, request, caddyhttp.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) error {
			if w != response {
				t.Fatalf("websocket response writer was wrapped as %T", w)
			}
			return nil
		},
	))
	if err != nil {
		t.Fatal(err)
	}

	var record map[string]any
	if decodeErr := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record); decodeErr != nil {
		t.Fatalf("decode log %q: %v", output.String(), decodeErr)
	}
	message, _ := record["message"].(string)
	if !strings.Contains(message, " 101 0 ") {
		t.Errorf("message %q does not contain websocket handshake status", message)
	}
}
