// Package zkb reads zKillboard's R2Z2 feed.
//
// R2Z2 is a sequential firehose: every killmail zKillboard receives is assigned
// a monotonically increasing sequence number and published at a URL derived
// from it. A consumer keeps a cursor, asks for cursor+1, and gets either the
// killmail or a 404 meaning "not yet". There is no long-poll and no websocket;
// the 404 is the backpressure.
//
// The important property, and the reason this path exists at all next to the
// ESI killmail fetcher, is that R2Z2 embeds the full ESI killmail document in
// its response. A kill arriving this way costs one request and nothing from the
// ESI budget — no hash lookup, no /killmails/{id}/{hash}/ round trip, no error
// limit exposure. The ESI fetcher remains for backfill, where only an id and a
// hash are known.
package zkb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/eve-kill/shrike/internal/killmail"
)

// DefaultBaseURL is the ephemeral feed. "Ephemeral" is literal: entries are
// dropped after a retention window measured in hours, so a consumer that falls
// far enough behind can no longer catch up by following the sequence and has to
// be repaired from the history endpoint instead.
const DefaultBaseURL = "https://r2z2.zkillboard.com/ephemeral"

// DefaultHistoryURL serves one JSON object of id → hash per day, going back far
// enough to repair a multi-day outage.
const DefaultHistoryURL = "https://r2z2.zkillboard.com/history"

// RateLimitPerSecond is what we allow ourselves. zKillboard permits 20; staying
// at half of that leaves room for the history cron and any ad-hoc repair
// running at the same time without the three of them together tripping a 429.
const RateLimitPerSecond = 10

// RequestTimeout bounds a single request. R2Z2 responds in milliseconds when
// healthy, so a request still open after ten seconds is not going to be
// answered usefully.
const RequestTimeout = 10 * time.Second

// ErrNotPublished means the sequence number has not been filled yet — the
// consumer has caught up with the feed. Routine, and the signal to wait.
var ErrNotPublished = errors.New("sequence not published")

// ErrThrottled means R2Z2 asked us to slow down (429) or rejected the request
// outright (403, which in practice means a missing or unacceptable User-Agent).
// Both call for a long pause rather than the short catch-up wait.
var ErrThrottled = errors.New("throttled by R2Z2")

// ZKB is zKillboard's own metadata block.
//
// Only Hash is load-bearing: it is the killmail hash, which the embedded ESI
// document does not carry and which the killmails table stores. Everything else
// is zKillboard's computed view of the kill — their ISK values, their points,
// their NPC and solo determination — and is deliberately not used. We compute
// all of it ourselves from the same ESI document, against our own price data,
// and storing theirs alongside ours would leave two disagreeing answers with no
// rule for which wins. It is decoded so the shape is documented and so a
// diagnostic can compare the two when a valuation looks wrong.
type ZKB struct {
	Hash string `json:"hash"`

	LocationID     int64    `json:"locationID"`
	FittedValue    float64  `json:"fittedValue"`
	DroppedValue   float64  `json:"droppedValue"`
	DestroyedValue float64  `json:"destroyedValue"`
	TotalValue     float64  `json:"totalValue"`
	Points         int32    `json:"points"`
	NPC            bool     `json:"npc"`
	Solo           bool     `json:"solo"`
	Awox           bool     `json:"awox"`
	Labels         []string `json:"labels"`
	AttackerCount  int32    `json:"attackerCount"`
	Href           string   `json:"href"`
}

// Response is one entry in the feed.
type Response struct {
	KillmailID int64 `json:"killmail_id"`

	// Hash appears both at the top level and inside ZKB. They have always
	// agreed; Hash reports whichever is present, preferring the top-level one.
	Hash string `json:"hash"`

	ESI        killmail.ESIKillmail `json:"esi"`
	ZKB        ZKB                  `json:"zkb"`
	UploadedAt int64                `json:"uploaded_at"`
	SequenceID int64                `json:"sequence_id"`
}

// KillmailHash returns the hash to store, tolerating either placement.
func (r *Response) KillmailHash() string {
	if r.Hash != "" {
		return r.Hash
	}
	return r.ZKB.Hash
}

// Limiter paces outbound requests.
//
// An interface so tests can run the listener at full speed without either
// sleeping for real or having the client silently skip its own rate limiting —
// the limiter is exercised in both cases, just with time supplied differently.
type Limiter interface {
	Wait(ctx context.Context) error
}

// Client talks to R2Z2.
type Client struct {
	BaseURL    string
	HistoryURL string
	UserAgent  string
	HTTP       *http.Client
	Limiter    Limiter
}

// New builds a client with production defaults.
func New(userAgent string) *Client {
	return &Client{
		BaseURL:    DefaultBaseURL,
		HistoryURL: DefaultHistoryURL,
		UserAgent:  userAgent,
		HTTP:       &http.Client{Timeout: RequestTimeout},
		Limiter:    NewWindowLimiter(RateLimitPerSecond, time.Second),
	}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: RequestTimeout}
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

func (c *Client) historyURL() string {
	if c.HistoryURL != "" {
		return c.HistoryURL
	}
	return DefaultHistoryURL
}

// LatestSequence reports the newest sequence number R2Z2 has published. Used
// once, to bootstrap a consumer that has no stored cursor.
func (c *Client) LatestSequence(ctx context.Context) (int64, error) {
	var body struct {
		Sequence int64 `json:"sequence"`
	}
	if err := c.get(ctx, c.baseURL()+"/sequence.json", &body); err != nil {
		return 0, err
	}
	if body.Sequence <= 0 {
		return 0, fmt.Errorf("R2Z2 reported sequence %d, which cannot be a starting point", body.Sequence)
	}
	return body.Sequence, nil
}

// Killmail fetches one sequence entry.
//
// Returns ErrNotPublished when the entry does not exist yet, which is the
// normal steady state of a caught-up consumer rather than a failure.
func (c *Client) Killmail(ctx context.Context, sequence int64) (*Response, error) {
	var out Response
	url := fmt.Sprintf("%s/%d.json", c.baseURL(), sequence)
	if err := c.get(ctx, url, &out); err != nil {
		return nil, err
	}
	if out.KillmailID == 0 {
		return nil, fmt.Errorf("sequence %d decoded to a killmail with no id", sequence)
	}
	if out.KillmailHash() == "" {
		return nil, fmt.Errorf("sequence %d carries no killmail hash", sequence)
	}
	// The sequence id is echoed back; a mismatch means the feed handed us
	// something other than what was asked for, and silently storing it would
	// advance the cursor past a kill that was never seen.
	if out.SequenceID != 0 && out.SequenceID != sequence {
		return nil, fmt.Errorf("asked for sequence %d and got %d", sequence, out.SequenceID)
	}
	return &out, nil
}

// Totals returns the killmail count R2Z2 holds for each available day, keyed
// YYYYMMDD.
//
// This is the index of what history exists. A backfill reads it first to learn
// which days can be asked for at all, rather than walking the calendar and
// taking a 404 for every day before the archive begins — each of which would
// cost a request and tell it nothing.
func (c *Client) Totals(ctx context.Context) (map[string]int64, error) {
	var raw map[string]int64
	if err := c.get(ctx, c.historyURL()+"/totals.json", &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// History returns the id → hash map for one UTC day, keyed as R2Z2 spells it:
// YYYYMMDD with no separators.
//
// This is the repair path. The ephemeral feed only retains hours, so anything
// missed during a longer outage is recovered from here, by id and hash, and
// refetched through ESI.
func (c *Client) History(ctx context.Context, day string) (map[int64]string, error) {
	var raw map[string]string
	if err := c.get(ctx, fmt.Sprintf("%s/%s.json", c.historyURL(), day), &raw); err != nil {
		return nil, err
	}

	out := make(map[int64]string, len(raw))
	for k, hash := range raw {
		var id int64
		if _, err := fmt.Sscanf(k, "%d", &id); err != nil || id <= 0 {
			continue
		}
		if hash == "" {
			continue
		}
		out[id] = hash
	}
	return out, nil
}

// get performs one rate-limited request and decodes the body.
func (c *Client) get(ctx context.Context, url string, out any) error {
	if c.Limiter != nil {
		if err := c.Limiter.Wait(ctx); err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	// R2Z2 rejects requests without a User-Agent identifying the operator, so
	// this is not decoration — omitting it is a 403.
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotPublished
	case resp.StatusCode == http.StatusTooManyRequests:
		return fmt.Errorf("%w: 429", ErrThrottled)
	case resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: 403 — check the User-Agent header", ErrThrottled)
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("R2Z2 %s returned %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}
