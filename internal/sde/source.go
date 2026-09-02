// Package sde imports CCP's Static Data Export into Postgres.
//
// The archive is ~99 MB compressed and ~575 MB across 79 JSONL members once
// expanded (mapMoons alone is 224 MB), so nothing here materialises a whole
// member. Members are read line by line with one decoded row in flight, and the
// zip is streamed to disk rather than held in memory because archive/zip needs
// random access.
package sde

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	baseURL = "https://developers.eveonline.com/static-data/tranquility"

	// Some type descriptions run to several kilobytes, well past
	// bufio.Scanner's 64 KB default. 4 MB is far more than any observed line
	// and still bounded, so a corrupt file cannot exhaust memory.
	maxLineBytes = 4 << 20

	// The manifest is tiny; the archive is ~99 MB and gets its own generous
	// budget rather than sharing a single timeout with the metadata request.
	manifestTimeout = 30 * time.Second
	archiveTimeout  = 15 * time.Minute
)

// Manifest is the content of latest.jsonl.
type Manifest struct {
	Key         string `json:"_key"`
	BuildNumber int64  `json:"buildNumber"`
	ReleaseDate string `json:"releaseDate"`
}

// Source is an opened SDE archive.
type Source struct {
	Build    int64
	Release  string
	Path     string
	reader   *zip.ReadCloser
	byMember map[string]*zip.File
}

// FetchManifest reports the currently published build without downloading it.
func FetchManifest(ctx context.Context, userAgent string) (*Manifest, error) {
	ctx, cancel := context.WithTimeout(ctx, manifestTimeout)
	defer cancel()

	body, err := get(ctx, baseURL+"/latest.jsonl", userAgent)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	raw, err := io.ReadAll(io.LimitReader(body, 64<<10))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if m.BuildNumber == 0 {
		return nil, fmt.Errorf("manifest has no buildNumber: %s", string(raw))
	}
	return &m, nil
}

// Open returns the archive for a build, downloading it into cacheDir only when
// not already present. Caching by build number matters during development: a
// re-import otherwise re-fetches 99 MB every run, and the archive for a given
// build is immutable.
func Open(ctx context.Context, cacheDir string, m *Manifest, userAgent string, progress func(string)) (*Source, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	path := filepath.Join(cacheDir, fmt.Sprintf("sde-%d-jsonl.zip", m.BuildNumber))

	if _, err := os.Stat(path); err != nil {
		if progress != nil {
			progress(fmt.Sprintf("downloading build %d", m.BuildNumber))
		}
		if err := download(ctx, archiveURL(m.BuildNumber), path, userAgent); err != nil {
			return nil, err
		}
	} else if progress != nil {
		progress(fmt.Sprintf("using cached build %d", m.BuildNumber))
	}

	rc, err := zip.OpenReader(path)
	if err != nil {
		// A truncated cache file from an interrupted download would fail here
		// forever, so remove it and let the next run re-fetch.
		_ = os.Remove(path)
		return nil, fmt.Errorf("open archive (removed, retry): %w", err)
	}

	src := &Source{
		Build:    m.BuildNumber,
		Release:  m.ReleaseDate,
		Path:     path,
		reader:   rc,
		byMember: make(map[string]*zip.File, len(rc.File)),
	}
	for _, f := range rc.File {
		src.byMember[filepath.Base(f.Name)] = f
	}
	return src, nil
}

func (s *Source) Close() error {
	if s.reader != nil {
		return s.reader.Close()
	}
	return nil
}

// Has reports whether a member exists. CCP adds and removes members between
// builds, so a caller may legitimately skip a missing one.
func (s *Source) Has(member string) bool {
	_, ok := s.byMember[member+".jsonl"]
	return ok
}

// Members lists every JSONL member, for diagnostics.
func (s *Source) Members() []string {
	out := make([]string, 0, len(s.byMember))
	for name := range s.byMember {
		out = append(out, name)
	}
	return out
}

// Stream decodes each line of a member and hands it to fn. Iteration stops at
// the first error fn returns, and at ctx cancellation.
//
// fn must not retain the Row: it is reused between lines to avoid an allocation
// per record, which matters across the ~1.5 M rows of a full import.
func (s *Source) Stream(ctx context.Context, member string, fn func(Row) error) error {
	f, ok := s.byMember[member+".jsonl"]
	if !ok {
		return fmt.Errorf("member %s.jsonl not found in archive", member)
	}

	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", member, err)
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 0, 256<<10), maxLineBytes)

	line := 0
	for scanner.Scan() {
		line++
		// Checking every 1000 lines rather than every line: the context read is
		// cheap but not free, and 1000 lines is a sub-millisecond window.
		if line%1000 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}

		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}

		row := Row{}
		if err := json.Unmarshal(raw, &row); err != nil {
			return fmt.Errorf("%s line %d: %w", member, line, err)
		}
		if err := fn(row); err != nil {
			return fmt.Errorf("%s line %d: %w", member, line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", member, err)
	}
	return nil
}

// Collect reads a complete member into a map keyed by its SDE identifier.
// Most import paths must use Stream; this exists for bounded reference-data
// transformations such as the Dogma bundle, whose patch rules need cross-file
// name and relationship lookups before they can emit output.
func (s *Source) Collect(ctx context.Context, member string) (map[int32]map[string]any, error) {
	rows := make(map[int32]map[string]any)
	err := s.Stream(ctx, member, func(row Row) error {
		// Unlike the relational importer, the upstream Dogma bundle retains
		// CCP's ID-zero placeholder records: some unpublished visual types refer
		// to group 0 and EVEShipFit's converter preserves that relationship.
		keyValue := toInt64(row["_key"])
		if keyValue == nil || *keyValue > 2147483647 || *keyValue < -2147483648 {
			return nil
		}
		key := *keyValue
		copy := make(map[string]any, len(row)-1)
		for field, value := range row {
			if field != "_key" {
				copy[field] = value
			}
		}
		rows[int32(key)] = copy
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func archiveURL(build int64) string {
	return fmt.Sprintf("%s/eve-online-static-data-%d-jsonl.zip", baseURL, build)
}

func get(ctx context.Context, url, userAgent string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// CCP asks for an identifying User-Agent on their static-data endpoints and
	// may throttle requests without one.
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}

// download writes to a temp file and renames on success, so an interrupted run
// can never leave a partial archive that looks like a valid cache entry.
func download(ctx context.Context, url, dest, userAgent string) error {
	ctx, cancel := context.WithTimeout(ctx, archiveTimeout)
	defer cancel()

	body, err := get(ctx, url, userAgent)
	if err != nil {
		return err
	}
	defer body.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".sde-download-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		_ = os.Remove(tmpName) // no-op once renamed
	}()

	if _, err := io.Copy(tmp, body); err != nil {
		return fmt.Errorf("download: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dest)
}
