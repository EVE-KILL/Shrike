package sso

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/retry"
)

func TestRefreshRetriesBadGatewayAndPreservesRequest(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if err := r.ParseForm(); err != nil || r.Form.Get("refresh_token") != "original-refresh" || !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			t.Error("refresh request changed during retry")
		}
		if attempts < 3 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, "upstream session expired")
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"access","refresh_token":"rotated","expires_in":1199}`)
	}))
	defer server.Close()
	c := &Client{ClientID: "fixture", ClientSecret: "secret", TokenURL: server.URL, HTTP: server.Client(), backoff: []time.Duration{0, 0, 0}}
	res, err := c.Refresh(t.Context(), "original-refresh")
	if err != nil || res.RefreshToken != "rotated" || attempts != 3 {
		t.Fatalf("response=%+v err=%v attempts=%d", res, err, attempts)
	}
}

func TestRefreshDefersLongRetryAfterAndDoesNotRetryDeadGrant(t *testing.T) {
	for _, limited := range []bool{true, false} {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			attempts++
			if limited {
				w.Header().Set("Retry-After", "120")
				w.WriteHeader(http.StatusTooManyRequests)
			} else {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
			}
		}))
		c := &Client{ClientID: "fixture", ClientSecret: "secret", TokenURL: server.URL, HTTP: server.Client(), backoff: []time.Duration{0, 0, 0}}
		_, err := c.Refresh(t.Context(), "original-refresh")
		server.Close()
		if limited {
			if deferred, ok := errors.AsType[*retry.Deferred](err); !ok || deferred.After != 120*time.Second {
				t.Fatalf("Retry-After not preserved: %v", err)
			}
		} else if !errors.Is(err, ErrPermanentlyDead) {
			t.Fatalf("dead grant not recognized: %v", err)
		}
		if attempts != 1 {
			t.Fatalf("attempts=%d", attempts)
		}
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRefreshDoesNotReplayAmbiguousTransportFailure(t *testing.T) {
	boom := errors.New("connection interrupted after POST")
	attempts := 0
	c := &Client{ClientID: "fixture", ClientSecret: "secret", HTTP: &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return nil, boom
	})}}
	if _, err := c.Refresh(context.Background(), "original-refresh"); !errors.Is(err, boom) || attempts != 1 {
		t.Fatalf("ambiguous refresh replayed: attempts=%d err=%v", attempts, err)
	}
}

type truncatedTokenBody struct{}

func (truncatedTokenBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (truncatedTokenBody) Close() error             { return nil }

func TestRefreshDoesNotReplayTruncatedSuccess(t *testing.T) {
	attempts := 0
	c := &Client{ClientID: "fixture", ClientSecret: "secret", HTTP: &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: truncatedTokenBody{}}, nil
	})}}
	if _, err := c.Refresh(t.Context(), "original-refresh"); !errors.Is(err, io.ErrUnexpectedEOF) || attempts != 1 {
		t.Fatalf("truncated success replayed: attempts=%d err=%v", attempts, err)
	}
}
