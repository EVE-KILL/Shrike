// Package sde imports CCP's Static Data Export into Postgres.
package sde

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/eve-kill/shrike/internal/telemetry"
)

const (
	latestURL       = "https://sde.evetech.net/tranquility/latest/"
	releaseIDLayout = "20060102T150405Z"
	documentTimeout = 30 * time.Second
	tableTimeout    = 15 * time.Minute
	maxDocumentSize = 1 << 20
	maxLineBytes    = 4 << 20
)

type Manifest struct {
	Release     string                `json:"release"`
	BuildNumber int64                 `json:"buildNumber"`
	ReleaseDate time.Time             `json:"releaseDate"`
	Tables      map[string]TableFiles `json:"tables"`
}

type TableFiles struct {
	Version string `json:"version"`
	JSONL   struct {
		URL       string `json:"url"`
		SchemaURL string `json:"recordSchemaUrl"`
	} `json:"jsonl"`
}

type cachedTable struct {
	path string
	key  string
}

// Source contains the locally cached inputs of one release.
type Source struct {
	Manifest *Manifest
	tables   map[string]cachedTable
}

func CacheDir(dataDir string) string {
	if dataDir == "" {
		dataDir = "data"
	}
	return filepath.Join(dataDir, "sde")
}

var sdeHTTP = &http.Client{Transport: telemetry.Transport("ccp.sde", http.DefaultTransport)}

func FetchManifest(ctx context.Context, userAgent string) (*Manifest, error) {
	return fetchManifest(ctx, sdeHTTP, latestURL, userAgent)
}

func fetchManifest(ctx context.Context, client *http.Client, url, userAgent string) (*Manifest, error) {
	ctx, cancel := context.WithTimeout(ctx, documentTimeout)
	defer cancel()
	resp, err := get(ctx, client, url, userAgent, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response

	var manifest Manifest
	if err := readDocument(resp.Body, &manifest); err != nil {
		return nil, fmt.Errorf("read SDE manifest: %w", err)
	}
	if _, err := time.Parse(releaseIDLayout, manifest.Release); err != nil {
		return nil, fmt.Errorf("invalid SDE release ID %q: %w", manifest.Release, err)
	}
	if manifest.BuildNumber <= 0 || manifest.ReleaseDate.IsZero() || len(manifest.Tables) == 0 {
		return nil, fmt.Errorf("incomplete SDE manifest for %s", manifest.Release)
	}
	return &manifest, nil
}

// Open stages every selected table before the caller can write anything. All
// URLs come from a single release manifest. If that release expires during
// staging, start over with the new latest release and reuse matching blobs.
func Open(ctx context.Context, cacheDir string, manifest *Manifest, members []string, userAgent string, progress func(string)) (*Source, error) {
	return open(ctx, sdeHTTP, latestURL, cacheDir, manifest, members, userAgent, progress)
}

func open(ctx context.Context, client *http.Client, latest, cacheDir string, manifest *Manifest, members []string, userAgent string, progress func(string)) (*Source, error) {
	src, err := stage(ctx, client, cacheDir, manifest, members, userAgent, progress)
	var status *httpStatusError
	if !errors.As(err, &status) || (status.code != http.StatusNotFound && status.code != http.StatusGone) {
		return src, err
	}
	newest, fetchErr := fetchManifest(ctx, client, latest, userAgent)
	if fetchErr != nil {
		return nil, errors.Join(err, fetchErr)
	}
	if newest.Release <= manifest.Release {
		return nil, err
	}
	return stage(ctx, client, cacheDir, newest, members, userAgent, progress)
}

func stage(ctx context.Context, client *http.Client, cacheDir string, manifest *Manifest, members []string, userAgent string, progress func(string)) (*Source, error) {
	for _, name := range members {
		table := manifest.Tables[name]
		if table.Version == "" || table.JSONL.URL == "" || table.JSONL.SchemaURL == "" {
			return nil, fmt.Errorf("SDE release %s is missing required table %s or its download links", manifest.Release, name)
		}
	}
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return nil, fmt.Errorf("create SDE cache: %w", err)
	}

	src := &Source{Manifest: manifest, tables: make(map[string]cachedTable, len(members))}
	for _, name := range members {
		table := manifest.Tables[name]
		// Hash the opaque version for a safe, stable file name.
		base := filepath.Join(cacheDir, fmt.Sprintf("%s-%x", name, sha256.Sum256([]byte(table.Version))))
		key, err := cacheKey(ctx, client, table.JSONL.SchemaURL, base+".schema.json", userAgent)
		if err != nil {
			return nil, fmt.Errorf("SDE %s schema: %w", name, err)
		}
		path := base + ".jsonl.gz"
		if err := cacheFile(path, func(f *os.File) error {
			if progress != nil {
				progress("downloading " + name)
			}
			return downloadTable(ctx, client, table.JSONL.URL, userAgent, f)
		}); err != nil {
			return nil, fmt.Errorf("SDE %s: %w", name, err)
		}
		src.tables[name] = cachedTable{path: path, key: key}
	}
	return src, nil
}

func (s *Source) Has(member string) bool {
	_, ok := s.tables[member]
	return ok
}

// Scan reads a cached table one JSONL record at a time. The callback's bytes
// are only valid until it returns.
func (s *Source) Scan(ctx context.Context, member string, fn func([]byte) error) error {
	table, ok := s.tables[member]
	if !ok {
		return fmt.Errorf("SDE table %s was not staged", member)
	}
	// #nosec G304 -- the path was constructed from a configured cache directory and a version hash.
	file, err := os.Open(table.path)
	if err != nil {
		return err
	}
	defer file.Close() //nolint:errcheck // read-only file
	body, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("decompress SDE %s: %w", member, err)
	}
	defer body.Close() //nolint:errcheck // checksum is checked by reading to EOF
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 256<<10), maxLineBytes)
	for line := 1; scanner.Scan(); line++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(scanner.Bytes()) != 0 {
			if err := fn(scanner.Bytes()); err != nil {
				return fmt.Errorf("SDE %s line %d: %w", member, line, err)
			}
		}
	}
	return scanner.Err()
}

func (s *Source) Stream(ctx context.Context, member string, fn func(Row) error) error {
	key := s.tables[member].key
	return s.Scan(ctx, member, func(raw []byte) error {
		var row Row
		if err := json.Unmarshal(raw, &row); err != nil {
			return err
		}
		id, ok := row[key].(float64)
		if !ok || id != math.Trunc(id) || id < math.MinInt32 || id > math.MaxInt32 {
			return fmt.Errorf("missing or invalid record ID %q", key)
		}
		// The database mappings use a common key across all source tables.
		row["_key"] = id
		return fn(row)
	})
}

type recordSchema struct {
	Key      string   `json:"x-sde-key"`
	Required []string `json:"required"`
}

func schemaKey(r io.Reader) (string, error) {
	var schema recordSchema
	if err := readDocument(r, &schema); err != nil {
		return "", err
	}
	if schema.Key == "" || !slices.Contains(schema.Required, schema.Key) {
		return "", fmt.Errorf("invalid x-sde-key %q", schema.Key)
	}
	return schema.Key, nil
}

func cacheKey(ctx context.Context, client *http.Client, url, path, userAgent string) (string, error) {
	err := cacheFile(path, func(f *os.File) error {
		ctx, cancel := context.WithTimeout(ctx, documentTimeout)
		defer cancel()
		resp, err := get(ctx, client, url, userAgent, false)
		if err != nil {
			return err
		}
		defer resp.Body.Close() //nolint:errcheck // read-only response
		_, err = schemaKey(io.TeeReader(resp.Body, f))
		return err
	})
	if err != nil {
		return "", err
	}
	// #nosec G304 -- the path was constructed in stage.
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close() //nolint:errcheck // read-only file
	return schemaKey(file)
}

func readDocument(r io.Reader, dest any) error {
	raw, err := io.ReadAll(io.LimitReader(r, maxDocumentSize+1))
	if err != nil {
		return err
	}
	if len(raw) > maxDocumentSize {
		return fmt.Errorf("SDE JSON document exceeds %d bytes", maxDocumentSize)
	}
	return json.Unmarshal(raw, dest)
}

type httpStatusError struct {
	url  string
	code int
}

func (err *httpStatusError) Error() string {
	return fmt.Sprintf("GET %s: HTTP %d", err.url, err.code)
}

func get(ctx context.Context, client *http.Client, url, userAgent string, keepGzip bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	if keepGzip {
		// Setting the header explicitly disables Go's transparent decompression.
		req.Header.Set("Accept-Encoding", "gzip")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, &httpStatusError{url: url, code: resp.StatusCode}
	}
	return resp, nil
}

func downloadTable(ctx context.Context, client *http.Client, url, userAgent string, file *os.File) error {
	ctx, cancel := context.WithTimeout(ctx, tableTimeout)
	defer cancel()
	resp, err := get(ctx, client, url, userAgent, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response
	if resp.Header.Get("Content-Encoding") != "gzip" {
		return fmt.Errorf("GET %s: expected gzip, got %q", url, resp.Header.Get("Content-Encoding"))
	}

	// Validate the gzip stream and checksum as it is written, so a truncated
	// response never becomes a reusable cache entry.
	body, err := gzip.NewReader(io.TeeReader(resp.Body, file))
	if err != nil {
		return err
	}
	defer body.Close() //nolint:errcheck // checksum is checked by reading to EOF
	// #nosec G110 -- the official SDE stream is discarded without retaining decompressed bytes.
	_, err = io.Copy(io.Discard, body)
	return err
}

func cacheFile(path string, fetch func(*os.File) error) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".sde-download-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()
	if err := fetch(tmp); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
