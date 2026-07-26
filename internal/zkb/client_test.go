package zkb

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixtures in testdata/ are real corpus killmails wrapped in a genuine R2Z2
// envelope, which is the part worth testing: the feed's `esi` block has to
// decode into the same type the parser consumes, or the cheap ingest path
// quietly produces different killmails from the expensive one.

// feedServer serves the fixture files at the paths the client constructs, so
// URL building is exercised rather than assumed.
func feedServer(t *testing.T) (*Client, *int) {
	t.Helper()

	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		// R2Z2 rejects an anonymous caller, so a client that forgets the header
		// must fail here rather than in production.
		if r.Header.Get("User-Agent") == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		name := strings.TrimPrefix(r.URL.Path, "/ephemeral/")
		body, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	return &Client{
		BaseURL:    srv.URL + "/ephemeral",
		HistoryURL: srv.URL + "/history",
		UserAgent:  "shrike-test/1.0",
		HTTP:       srv.Client(),
	}, &requests
}

func TestLatestSequence(t *testing.T) {
	c, _ := feedServer(t)

	got, err := c.LatestSequence(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != 900002 {
		t.Errorf("sequence = %d, want 900002", got)
	}
}

// A zero or negative head is not a usable starting point, and accepting one
// would have the listener walk up from sequence 1 through millions of 404s.
func TestLatestSequenceRejectsAnUnusableHead(t *testing.T) {
	for _, body := range []string{`{"sequence":0}`, `{"sequence":-5}`, `{}`} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		}))
		c := &Client{BaseURL: srv.URL, UserAgent: "t", HTTP: srv.Client()}
		if _, err := c.LatestSequence(context.Background()); err == nil {
			t.Errorf("body %s was accepted as a starting sequence", body)
		}
		srv.Close()
	}
}

// The envelope carries the full ESI document, which is the entire reason this
// ingest path is cheaper than the ESI one.
func TestKillmailDecodesTheEmbeddedESIDocument(t *testing.T) {
	c, _ := feedServer(t)

	got, err := c.Killmail(context.Background(), 900001)
	if err != nil {
		t.Fatal(err)
	}

	if got.KillmailID != 435341 {
		t.Errorf("killmail_id = %d, want 435341", got.KillmailID)
	}
	if got.ESI.KillmailID != got.KillmailID {
		t.Errorf("the embedded ESI document has id %d but the envelope says %d",
			got.ESI.KillmailID, got.KillmailID)
	}
	if got.ESI.SolarSystemID == 0 {
		t.Error("the embedded ESI document has no solar system — it did not decode")
	}
	if len(got.ESI.Attackers) != 4 {
		t.Errorf("decoded %d attackers, want 4", len(got.ESI.Attackers))
	}
	if len(got.ESI.Victim.Items) == 0 {
		t.Error("the victim's items did not decode, so nothing would be valued")
	}
	if got.ESI.KillmailTime.IsZero() {
		t.Error("killmail_time did not decode")
	}

	// The hash is the one thing ESI's own response never carries, so losing it
	// here means an unstorable killmail.
	if got.KillmailHash() == "" {
		t.Error("no killmail hash")
	}
	if got.ESI.KillmailHash != "" {
		t.Errorf("the embedded ESI document carries a hash (%q) — the fixture is not "+
			"shaped like a real ESI response", got.ESI.KillmailHash)
	}
}

// zKillboard's own valuation is decoded but must never be confused with ours;
// this pins that it is at least parsed, so a diagnostic can compare them.
func TestKillmailDecodesTheZKBBlock(t *testing.T) {
	c, _ := feedServer(t)

	got, err := c.Killmail(context.Background(), 900001)
	if err != nil {
		t.Fatal(err)
	}
	if got.ZKB.TotalValue == 0 {
		t.Error("zkb.totalValue did not decode")
	}
	if got.ZKB.AttackerCount != 4 {
		t.Errorf("zkb.attackerCount = %d, want 4", got.ZKB.AttackerCount)
	}
	if len(got.ZKB.Labels) != 2 {
		t.Errorf("zkb.labels = %v, want two entries", got.ZKB.Labels)
	}
}

// The hash appears twice in the wire format. Either alone must work.
func TestKillmailHashFallsBackToTheZKBBlock(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"top level only", `{"killmail_id":1,"hash":"aaa","zkb":{}}`, "aaa"},
		{"zkb only", `{"killmail_id":1,"zkb":{"hash":"bbb"}}`, "bbb"},
		{"both", `{"killmail_id":1,"hash":"aaa","zkb":{"hash":"aaa"}}`, "aaa"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tc.body)
			}))
			defer srv.Close()

			c := &Client{BaseURL: srv.URL, UserAgent: "t", HTTP: srv.Client()}
			got, err := c.Killmail(context.Background(), 1)
			if err != nil {
				t.Fatal(err)
			}
			if got.KillmailHash() != tc.want {
				t.Errorf("hash = %q, want %q", got.KillmailHash(), tc.want)
			}
		})
	}
}

// A killmail with no hash cannot be stored, so accepting one only defers the
// failure to the insert, by which point the cursor has moved past it.
func TestKillmailRejectsAnUnstorableEntry(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"no hash anywhere", `{"killmail_id":42,"zkb":{}}`},
		{"no killmail id", `{"hash":"aaa","zkb":{"hash":"aaa"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tc.body)
			}))
			defer srv.Close()

			c := &Client{BaseURL: srv.URL, UserAgent: "t", HTTP: srv.Client()}
			if _, err := c.Killmail(context.Background(), 1); err == nil {
				t.Error("an unstorable entry was accepted")
			}
		})
	}
}

// Being handed a different sequence than the one asked for would advance the
// cursor past a killmail nobody ever saw.
func TestKillmailRejectsAMismatchedSequence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"killmail_id":42,"hash":"aaa","sequence_id":999}`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, UserAgent: "t", HTTP: srv.Client()}
	_, err := c.Killmail(context.Background(), 100)
	if err == nil {
		t.Fatal("a response for a different sequence was accepted")
	}
	if !strings.Contains(err.Error(), "999") {
		t.Errorf("the error does not say what was received: %v", err)
	}
}

// The three status codes that mean different things, told apart. Conflating any
// two of them breaks the listener: a 404 read as an error burns retry attempts
// on an entry that simply is not written yet, and a 429 read as "caught up"
// keeps hammering at the rate that earned the 429.
func TestStatusCodesAreDistinguished(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/404.json":
			w.WriteHeader(http.StatusNotFound)
		case "/429.json":
			w.WriteHeader(http.StatusTooManyRequests)
		case "/403.json":
			w.WriteHeader(http.StatusForbidden)
		case "/500.json":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			_, _ = io.WriteString(w, `{"killmail_id":1,"hash":"a"}`)
		}
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, UserAgent: "t", HTTP: srv.Client()}
	ctx := context.Background()

	cases := []struct {
		seq     int64
		want    error
		wantNil bool
	}{
		{404, ErrNotPublished, false},
		{429, ErrThrottled, false},
		{403, ErrThrottled, false},
	}
	for _, tc := range cases {
		if _, err := c.Killmail(ctx, tc.seq); !errors.Is(err, tc.want) {
			t.Errorf("status %d returned %v, want %v", tc.seq, err, tc.want)
		}
	}

	// A 500 is zKillboard being unwell. It is neither "caught up" nor a
	// throttle, and must not be mistaken for either.
	_, err := c.Killmail(ctx, 500)
	if err == nil || errors.Is(err, ErrNotPublished) || errors.Is(err, ErrThrottled) {
		t.Errorf("status 500 returned %v, want a plain error", err)
	}
}

// R2Z2 answers an anonymous request with a 403, which is why the client sets a
// User-Agent on every request rather than only on some.
func TestEveryRequestIdentifiesItself(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("User-Agent"))
		_, _ = io.WriteString(w, `{"sequence":5,"killmail_id":1,"hash":"a"}`)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HistoryURL: srv.URL, UserAgent: "shrike/1.0 (test)", HTTP: srv.Client()}
	ctx := context.Background()
	_, _ = c.LatestSequence(ctx)
	_, _ = c.Killmail(ctx, 1)
	_, _ = c.History(ctx, "20260720")

	if len(seen) != 3 {
		t.Fatalf("made %d requests, want 3", len(seen))
	}
	for i, ua := range seen {
		if ua != "shrike/1.0 (test)" {
			t.Errorf("request %d sent User-Agent %q", i, ua)
		}
	}
}

// The history endpoint is the repair path, keyed by id as a string.
func TestHistoryParsesIDHashPairs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/20260720.json" {
			t.Errorf("history requested %s, want /20260720.json", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{
            "128000001": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "128000002": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
            "notanumber": "cccccccccccccccccccccccccccccccccccccccc",
            "128000003": ""
        }`)
	}))
	defer srv.Close()

	c := &Client{HistoryURL: srv.URL, UserAgent: "t", HTTP: srv.Client()}
	got, err := c.History(context.Background(), "20260720")
	if err != nil {
		t.Fatal(err)
	}

	// The two malformed entries are skipped rather than fatal: one bad key in a
	// day's index is not a reason to abandon repairing that day.
	if len(got) != 2 {
		t.Fatalf("parsed %d entries, want 2: %v", len(got), got)
	}
	if got[128000001] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf("hash for 128000001 = %q", got[128000001])
	}
	if _, ok := got[128000003]; ok {
		t.Error("an entry with an empty hash was kept — it cannot be fetched")
	}
}

// A missing day is normal (zKillboard does not publish every day forever) and
// must be distinguishable so the repair cron skips rather than reports failure.
func TestHistoryReportsAMissingDay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &Client{HistoryURL: srv.URL, UserAgent: "t", HTTP: srv.Client()}
	if _, err := c.History(context.Background(), "20200101"); !errors.Is(err, ErrNotPublished) {
		t.Errorf("a missing day returned %v, want ErrNotPublished", err)
	}
}
