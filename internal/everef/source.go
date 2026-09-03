// Package everef imports the datasets EVE Ref publishes at data.everef.net:
// insurance payouts, market history, the sovereignty map, wars, and the daily
// killmail archives.
//
// These are the datasets CCP either does not expose or exposes only as a live
// snapshot. EVE Ref keeps the history, which is the only way to backfill a
// killboard that intends to go back to 2007.
//
// Every dataset is published the same handful of ways — a latest JSON, a daily
// file, a yearly tar.bz2, and an Apache-style directory index to discover what
// exists — so the transport lives here and each importer only describes its own
// data.
package everef

import (
	"compress/bzip2"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// DefaultBaseURL is the root of every dataset.
const DefaultBaseURL = "https://data.everef.net"

// Timeouts are per request, not per import. A yearly war archive is a few
// hundred megabytes over a slow link, so the ceiling has to be generous; the
// context the caller passes is what actually bounds an import.
const (
	requestTimeout = 30 * time.Minute
	listTimeout    = 2 * time.Minute
)

// ErrNotPublished marks a file EVE Ref does not have.
//
// It is a routine outcome, not a failure: the most recent day is regularly
// absent for several hours, and whole years are missing from some datasets. An
// importer reports it and moves on.
var ErrNotPublished = fmt.Errorf("not published")

// Client fetches from data.everef.net.
type Client struct {
	// BaseURL is where the datasets live. Overridable so the importers can be
	// pointed at a fixture server; empty means DefaultBaseURL.
	BaseURL   string
	UserAgent string
	HTTP      *http.Client
}

// FileMetadata identifies one published object. EVE Ref may replace historical
// files in place, so the URL alone is not a version; ETag is the primary
// identity and size/last-modified make the comparison observable.
type FileMetadata struct {
	URL          string
	ETag         string
	Size         int64
	LastModified *time.Time
}

// url joins a dataset path onto the client's root.
func (c *Client) url(path string) string {
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	return strings.TrimSuffix(base, "/") + path
}

// NewClient builds a client. EVE Ref asks for an identifying user agent the
// same way CCP does.
func NewClient(userAgent string) *Client {
	return &Client{
		BaseURL:   DefaultBaseURL,
		UserAgent: userAgent,
		HTTP:      &http.Client{Timeout: requestTimeout},
	}
}

// Metadata reads an object's current identity without downloading its body.
func (c *Client) Metadata(ctx context.Context, url string) (FileMetadata, error) {
	meta := FileMetadata{URL: url}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return meta, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return meta, fmt.Errorf("HEAD %s: %w", url, err)
	}
	if err := resp.Body.Close(); err != nil {
		return meta, fmt.Errorf("close HEAD %s response: %w", url, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return meta, fmt.Errorf("%s: %w", url, ErrNotPublished)
	}
	if resp.StatusCode != http.StatusOK {
		return meta, fmt.Errorf("HEAD %s: HTTP %d", url, resp.StatusCode)
	}

	meta.ETag = strings.TrimSpace(resp.Header.Get("ETag"))
	meta.Size = resp.ContentLength
	if value := resp.Header.Get("Last-Modified"); value != "" {
		if parsed, parseErr := http.ParseTime(value); parseErr == nil {
			parsed = parsed.UTC()
			meta.LastModified = &parsed
		}
	}
	return meta, nil
}

// get performs a GET and hands the caller the open body.
func (c *Client) get(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	// The payloads are already bzip2; asking for identity stops a
	// transport-level re-compression that would only be undone again.
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("%s: %w", url, ErrNotPublished)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}

// JSON fetches and decodes a JSON document, transparently decompressing a
// `.bz2` URL.
func (c *Client) JSON(ctx context.Context, url string, out any) error {
	body, err := c.get(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()

	var r io.Reader = body
	if strings.HasSuffix(url, ".bz2") {
		r = bzip2.NewReader(body)
	}
	if err := json.NewDecoder(r).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}

// Stream hands the caller the decompressed body of a `.bz2` file, which is how
// anything large is read: a year of market history or a day of killmails is
// parsed as it arrives rather than buffered whole.
func (c *Client) Stream(ctx context.Context, url string, fn func(io.Reader) error) error {
	body, err := c.get(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()

	var r io.Reader = body
	if strings.HasSuffix(url, ".bz2") {
		r = bzip2.NewReader(body)
	}
	return fn(r)
}

// listing matches the hrefs in an Apache-style directory index. EVE Ref
// publishes no manifest, so discovering what exists means reading the index —
// which is also why a dataset gap is invisible until the listing is parsed.
var listing = regexp.MustCompile(`href="([^"]+)"`)

// List returns the hrefs in a directory index whose text matches pattern, in
// sorted order.
//
// The capture group of pattern is what gets returned, so a caller asks for the
// date or the filename rather than the raw href.
func (c *Client) List(ctx context.Context, url string, pattern *regexp.Regexp) ([]string, error) {
	listCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	body, err := c.get(listCtx, url)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	// A directory index is a page, not a dataset — a few hundred kilobytes at
	// most. The cap stops a redirect to something enormous from being read
	// into memory.
	html, err := io.ReadAll(io.LimitReader(body, 8<<20))
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var out []string
	for _, m := range listing.FindAllStringSubmatch(string(html), -1) {
		sub := pattern.FindStringSubmatch(m[1])
		if sub == nil {
			continue
		}
		v := sub[1]
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out, nil
}

// DateRange lists every day from..to inclusive, as YYYY-MM-DD.
func DateRange(from, to string) ([]string, error) {
	start, err := time.Parse(dateLayout, from)
	if err != nil {
		return nil, fmt.Errorf("invalid start date %q: expected YYYY-MM-DD", from)
	}
	end, err := time.Parse(dateLayout, to)
	if err != nil {
		return nil, fmt.Errorf("invalid end date %q: expected YYYY-MM-DD", to)
	}

	var out []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format(dateLayout))
	}
	return out, nil
}

const dateLayout = "2006-01-02"

// Today and Yesterday are UTC, which is the only timezone EVE has.
func Today() string     { return time.Now().UTC().Format(dateLayout) }
func Yesterday() string { return time.Now().UTC().AddDate(0, 0, -1).Format(dateLayout) }

// DayAfter returns the day following date, for resuming an import one past
// where it stopped.
func DayAfter(date string) (string, error) {
	d, err := time.Parse(dateLayout, date)
	if err != nil {
		return "", fmt.Errorf("invalid date %q: expected YYYY-MM-DD", date)
	}
	return d.AddDate(0, 0, 1).Format(dateLayout), nil
}

// Result reports what one import did. Every importer returns one so the CLI can
// render them all the same way.
type Result struct {
	Name string `json:"name"`
	// Seen is how many records the source offered, against which Rows shows how
	// many were new or changed. The two together are what distinguish "nothing
	// happened" from "nothing was fetched".
	Seen int64 `json:"seen,omitempty"`
	// Rows written to the importer's primary table.
	Rows int64 `json:"rows"`
	// Related rows written to a companion table: sovereignty's history log, or
	// the killmails carried inside a war archive.
	Related int64 `json:"related,omitempty"`
	// Adjusted counts rows that already existed and were corrected in place —
	// war ids filled in on killmails the live queue got to first.
	Adjusted int64  `json:"adjusted,omitempty"`
	Skipped  int64  `json:"skipped,omitempty"`
	Failed   int64  `json:"failed,omitempty"`
	Missing  bool   `json:"missing,omitempty"`
	Elapsed  string `json:"elapsed"`
}
