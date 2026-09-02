package dogmadata

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ConverterCommit = "dc5c21440d6ac91e8753d520412ec4ccae266167"
	converterURL    = "https://github.com/EVEShipFit/data/archive/" + ConverterCommit + ".tar.gz"
)

// FetchPatches downloads the immutable EVEShipFit/data revision and returns
// its sorted YAML patch files. The archive is cached by commit.
func FetchPatches(ctx context.Context, cacheDir, userAgent string) ([][]byte, error) {
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return nil, err
	}
	path := filepath.Join(cacheDir, "eveshipfit-data-"+ConverterCommit+".tar.gz")
	if _, err := os.Stat(path); err != nil {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, converterURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch EVEShipFit/data: HTTP %d", resp.StatusCode)
		}
		tmp, err := os.CreateTemp(cacheDir, ".dogma-patches-*")
		if err != nil {
			return nil, err
		}
		tmpName := tmp.Name()
		defer os.Remove(tmpName) //nolint:errcheck
		if _, err = io.Copy(tmp, resp.Body); err != nil {
			_ = tmp.Close()
			return nil, err
		}
		if err = tmp.Close(); err != nil {
			return nil, err
		}
		if err = os.Rename(tmpName, path); err != nil {
			return nil, err
		}
	}

	// #nosec G304 -- path is constructed from the operator-selected cache directory and a pinned filename.
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer func() { _ = gz.Close() }()

	type namedPatch struct {
		name string
		data []byte
	}
	var found []namedPatch
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		name := filepath.ToSlash(header.Name)
		if !strings.Contains(name, "/patches/") || !strings.HasSuffix(name, ".yaml") {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(tr, 1<<20))
		if err != nil {
			return nil, err
		}
		found = append(found, namedPatch{name: filepath.Base(name), data: data})
	}
	sort.Slice(found, func(i, j int) bool { return found[i].name < found[j].name })
	if len(found) == 0 {
		return nil, fmt.Errorf("EVEShipFit/data archive contained no patches")
	}
	out := make([][]byte, len(found))
	for i := range found {
		out[i] = found[i].data
	}
	return out, nil
}
