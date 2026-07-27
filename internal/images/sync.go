package images

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eve-kill/shrike/internal/objectstore"
	"golang.org/x/sync/errgroup"
)

const (
	TurtleToolsRepository = "SentientTurtle/EVE-TurtleTools"
	TurtleTypeExportAsset = "Image.Export.Collection.zip"
	typeExportManifestKey = "types/manifest.json"
)

var typeExportAssetName = regexp.MustCompile(
	`^[1-9][0-9]*_(?:64(?:_bpc)?\.png|512\.jpg)$`,
)

type TypeExportSyncOptions struct {
	HTTPClient *http.Client
	Token      string
	UserAgent  string
	APIURL     string
	Progress   func(ImportResult)
	Now        func() time.Time
}

type TypeExportSyncResult struct {
	Release string       `json:"release"`
	Digest  string       `json:"digest"`
	Changed bool         `json:"changed"`
	Import  ImportResult `json:"import"`
}

type typeExportManifest struct {
	Version       int               `json:"version"`
	Release       string            `json:"release"`
	ArchiveDigest string            `json:"archive_digest"`
	UpdatedAt     time.Time         `json:"updated_at"`
	Images        map[string]string `json:"images"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Digest             string `json:"digest"`
	} `json:"assets"`
}

func SyncTypeExport(
	ctx context.Context,
	store ObjectStore,
	options TypeExportSyncOptions,
) (TypeExportSyncResult, error) {
	if store == nil {
		return TypeExportSyncResult{}, unavailable()
	}
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	userAgent := options.UserAgent
	if userAgent == "" {
		userAgent = "evekill-imageserver"
	}
	apiURL := options.APIURL
	if apiURL == "" {
		apiURL = "https://api.github.com/repos/" +
			TurtleToolsRepository + "/releases/latest"
	}
	release, assetURL, remoteDigest, err := latestTypeExport(
		ctx,
		client,
		apiURL,
		options.Token,
		userAgent,
	)
	if err != nil {
		return TypeExportSyncResult{}, err
	}
	current, err := loadTypeExportManifest(ctx, store)
	if err != nil {
		return TypeExportSyncResult{}, err
	}
	if current != nil &&
		((remoteDigest != "" &&
			current.ArchiveDigest == trimDigest(remoteDigest)) ||
			(remoteDigest == "" && current.Release == release)) {
		return TypeExportSyncResult{
			Release: release, Digest: current.ArchiveDigest, Changed: false,
		}, nil
	}

	archivePath, digest, cleanup, err := downloadTypeExport(
		ctx,
		client,
		assetURL,
		userAgent,
		remoteDigest,
	)
	if err != nil {
		return TypeExportSyncResult{}, err
	}
	defer cleanup()

	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return TypeExportSyncResult{}, fmt.Errorf(
			"open TurtleTools image export: %w",
			err,
		)
	}
	defer archive.Close()
	imageCount := 0
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(file.Name)
		if !safeRelativePath(name) {
			return TypeExportSyncResult{}, fmt.Errorf(
				"unsafe image export path %q",
				name,
			)
		}
		base := filepath.Base(name)
		if typeExportAssetName.MatchString(base) {
			imageCount++
		}
	}
	if imageCount == 0 {
		return TypeExportSyncResult{}, fmt.Errorf(
			"TurtleTools image export contains no type images",
		)
	}
	var previous map[string]string
	if current != nil {
		previous = current.Images
	}
	result, hashes, err := importTypeExportImages(
		ctx,
		store,
		archive.File,
		previous,
		options.Progress,
	)
	if err != nil {
		return TypeExportSyncResult{}, err
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	manifest := typeExportManifest{
		Version: 1, Release: release, ArchiveDigest: digest,
		UpdatedAt: now().UTC(), Images: hashes,
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		return TypeExportSyncResult{}, err
	}
	// Serving derives stable object keys directly from the type ID and never
	// reads this manifest. Publishing it last gives the next synchronization
	// one small object to compare instead of a HEAD request per image.
	if err := store.PutWithOptions(
		ctx,
		typeExportManifestKey,
		body,
		objectstore.PutOptions{
			ContentType: "application/json", CacheControl: "no-cache",
		},
	); err != nil {
		return TypeExportSyncResult{}, fmt.Errorf(
			"publish type export manifest: %w",
			err,
		)
	}
	return TypeExportSyncResult{
		Release: release, Digest: digest, Changed: true, Import: result,
	}, nil
}

func importTypeExportImages(
	ctx context.Context,
	store ObjectStore,
	files []*zip.File,
	previous map[string]string,
	progress func(ImportResult),
) (ImportResult, map[string]string, error) {
	unique := make(map[string]*zip.File)
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(filepath.ToSlash(file.Name))
		if !safeAssetName(base) {
			return ImportResult{}, nil, fmt.Errorf(
				"unsafe image export filename %q",
				base,
			)
		}
		if !typeExportAssetName.MatchString(base) {
			continue
		}
		if previous := unique[base]; previous != nil &&
			previous.Name != file.Name {
			return ImportResult{}, nil, fmt.Errorf(
				"TurtleTools image export contains duplicate filename %q",
				base,
			)
		}
		unique[base] = file
	}

	group, groupCtx := errgroup.WithContext(ctx)
	input := make(chan *zip.File, 8)
	var scanned, uploaded, skipped, uploadedBytes atomic.Int64
	hashes := make(map[string]string, len(unique))
	var hashesMu sync.Mutex
	for range 8 {
		group.Go(func() error {
			for file := range input {
				base := filepath.Base(filepath.ToSlash(file.Name))
				contentType, ok := typeExportContentType(base)
				if !ok {
					return fmt.Errorf(
						"unexpected TurtleTools image export filename %q",
						base,
					)
				}
				body, err := readZipFile(file, defaultMaximumObject)
				if err != nil {
					return err
				}
				sum := sha256.Sum256(body)
				digest := hex.EncodeToString(sum[:])
				hashesMu.Lock()
				hashes[base] = digest
				hashesMu.Unlock()

				changed := previous[base] != digest
				if changed {
					err := store.PutWithOptions(
						groupCtx,
						"types/"+base,
						body,
						objectstore.PutOptions{
							ContentType:  contentType,
							CacheControl: immutableCacheControl,
							Metadata: map[string]string{
								"sha256": digest,
							},
						},
					)
					if err != nil {
						return fmt.Errorf("upload types/%s: %w", base, err)
					}
				}
				scanned.Add(1)
				if changed {
					uploaded.Add(1)
					uploadedBytes.Add(int64(len(body)))
				} else {
					skipped.Add(1)
				}
				if progress != nil && scanned.Load()%1_000 == 0 {
					progress(ImportResult{
						Scanned: scanned.Load(), Uploaded: uploaded.Load(),
						Skipped: skipped.Load(), Bytes: uploadedBytes.Load(),
					})
				}
			}
			return nil
		})
	}
	group.Go(func() error {
		defer close(input)
		for _, file := range unique {
			select {
			case input <- file:
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
		return nil
	})
	err := group.Wait()
	result := ImportResult{
		Scanned: scanned.Load(), Uploaded: uploaded.Load(),
		Skipped: skipped.Load(), Bytes: uploadedBytes.Load(),
	}
	if progress != nil {
		progress(result)
	}
	return result, hashes, err
}

func typeExportContentType(name string) (string, bool) {
	if !typeExportAssetName.MatchString(name) {
		return "", false
	}
	if strings.HasSuffix(name, ".png") {
		return "image/png", true
	}
	if strings.HasSuffix(name, ".jpg") {
		return "image/jpeg", true
	}
	return "", false
}

func latestTypeExport(
	ctx context.Context,
	client *http.Client,
	apiURL, token, userAgent string,
) (release, assetURL, digest string, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", "", err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", "", "", fmt.Errorf("query TurtleTools release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", "", fmt.Errorf(
			"query TurtleTools release: HTTP %d",
			response.StatusCode,
		)
	}
	var value githubRelease
	if err := json.NewDecoder(
		io.LimitReader(response.Body, 4<<20),
	).Decode(&value); err != nil {
		return "", "", "", err
	}
	for _, asset := range value.Assets {
		if asset.Name == TurtleTypeExportAsset {
			return value.TagName, asset.BrowserDownloadURL, asset.Digest, nil
		}
	}
	return "", "", "", fmt.Errorf(
		"%s is missing from TurtleTools release %s",
		TurtleTypeExportAsset,
		value.TagName,
	)
}

func downloadTypeExport(
	ctx context.Context,
	client *http.Client,
	url, userAgent, expectedDigest string,
) (string, string, func(), error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", func() {}, err
	}
	request.Header.Set("User-Agent", userAgent)
	response, err := client.Do(request)
	if err != nil {
		return "", "", func() {}, fmt.Errorf(
			"download TurtleTools image export: %w",
			err,
		)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", func() {}, fmt.Errorf(
			"download TurtleTools image export: HTTP %d",
			response.StatusCode,
		)
	}
	file, err := os.CreateTemp("", "shrike-turtle-*.zip")
	if err != nil {
		return "", "", func() {}, err
	}
	name := file.Name()
	cleanup := func() { _ = os.Remove(name) }
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(file, hash), response.Body); err != nil {
		_ = file.Close()
		cleanup()
		return "", "", func() {}, err
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", "", func() {}, err
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	if expected := trimDigest(expectedDigest); expected != "" && expected != digest {
		cleanup()
		return "", "", func() {}, fmt.Errorf(
			"TurtleTools digest mismatch: got %s, want %s",
			digest,
			expected,
		)
	}
	return name, digest, cleanup, nil
}

func loadTypeExportManifest(
	ctx context.Context,
	store ObjectStore,
) (*typeExportManifest, error) {
	object, err := store.GetObject(ctx, typeExportManifestKey)
	if err != nil || object == nil {
		return nil, err
	}
	var manifest typeExportManifest
	if err := json.Unmarshal(object.Body, &manifest); err != nil {
		return nil, fmt.Errorf("decode current type export manifest: %w", err)
	}
	if manifest.Version != 1 {
		return nil, fmt.Errorf(
			"unsupported type export manifest version %d",
			manifest.Version,
		)
	}
	if manifest.Images == nil {
		manifest.Images = map[string]string{}
	}
	return &manifest, nil
}

func trimDigest(value string) string {
	value = strings.TrimSpace(value)
	if before, after, found := strings.Cut(value, ":"); found &&
		strings.EqualFold(before, "sha256") {
		value = after
	}
	return strings.ToLower(value)
}
