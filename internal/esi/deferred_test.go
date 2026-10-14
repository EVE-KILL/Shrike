package esi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/retry"
)

type failingTransport func(*http.Request) (*http.Response, error)

func (f failingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTransportAndTruncatedBodyFailuresRetainCause(t *testing.T) {
	for _, truncated := range []bool{false, true} {
		boom := errors.New("upstream connection failed")
		if truncated {
			boom = io.ErrUnexpectedEOF
		}
		attempts := 0
		c := &Client{HTTP: &http.Client{Transport: failingTransport(func(*http.Request) (*http.Response, error) {
			attempts++
			if truncated {
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: &failedBody{}}, nil
			}
			return nil, boom
		})}, backoff: []time.Duration{0, 0, 0}}
		res, err := c.execute(t.Context(), "GET", "https://fixture.test/characters/1", nil, "", "")
		if !errors.Is(err, boom) || attempts != maxHTTPRetries+1 {
			t.Fatalf("truncated=%v attempts=%d err=%v", truncated, attempts, err)
		}
		_, err = typed[character](raw{Status: res.Status}, err)
		if _, deferred := errors.AsType[*retry.Deferred](err); deferred {
			t.Fatal("real upstream failure became admission deferral")
		}
	}
}

type failedBody struct{}

func (*failedBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (*failedBody) Close() error             { return nil }

func TestAuthenticatedAdmissionDefersAndCancellationPropagates(t *testing.T) {
	rdb := testRedis(t)
	fake := newFakeESI(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "[]") })
	c := testClient(t, rdb, fake.URL)
	c.maxWait = time.Millisecond
	group := ResolveGroup(c.fullURL("/latest/corporations/24497/killmails/recent/"))
	if err := rdb.Set(t.Context(), keyRemaining(group.Name), "0", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if err := rdb.Set(t.Context(), keyResetAt(group.Name), strconv.FormatInt(time.Now().Add(30*time.Second).UnixMilli(), 10), time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	res, err := GetAuthenticated[[]WarKillmailRef](t.Context(), c, "/latest/corporations/24497/killmails/recent/", "fixture-token")
	deferred, ok := errors.AsType[*retry.Deferred](err)
	if !ok || res.Status != 0 || deferred.After < 29*time.Second || fake.Hits() != 0 {
		t.Fatalf("response=%+v error=%v hits=%d", res, err, fake.Hits())
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = GetAuthenticated[[]WarKillmailRef](ctx, c, "/latest/corporations/24497/killmails/recent/", "fixture-token")
	if !errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "fixture-token") {
		t.Fatalf("cancellation lost or credentials leaked: %v", err)
	}
}
