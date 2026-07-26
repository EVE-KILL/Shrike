package images

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/eve-kill/shrike/internal/objectstore"
	"golang.org/x/sync/errgroup"
)

const (
	TurtleToolsRepository = "SentientTurtle/EVE-TurtleTools"
	TurtleBundleAsset     = "Service.Bundle.zip"
)

type BundleSyncOptions struct {
	HTTPClient *http.Client
	Token      string
	UserAgent  string
	APIURL     string
	Progress   func(ImportResult)
	Now        func() time.Time
}

type BundleSyncResult struct {
	Release string       `json:"release"`
	Digest  string       `json:"digest"`
	Changed bool         `json:"changed"`
	Import  ImportResult `json:"import"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Digest             string `json:"digest"`
	} `json:"assets"`
}

func SyncTypeBundle(
	ctx context.Context,
	store ObjectStore,
	options BundleSyncOptions,
) (BundleSyncResult, error) {
	if store == nil {
		return BundleSyncResult{}, unavailable()
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
	release, assetURL, remoteDigest, err := latestBundle(
		ctx,
		client,
		apiURL,
		options.Token,
		userAgent,
	)
	if err != nil {
		return BundleSyncResult{}, err
	}
	current, err := loadTypeBundlePointer(ctx, store)
	if err != nil {
		return BundleSyncResult{}, err
	}
	if current != nil &&
		((remoteDigest != "" && current.Digest == trimDigest(remoteDigest)) ||
			(remoteDigest == "" && current.Release == release)) {
		return BundleSyncResult{
			Release: release, Digest: current.Digest, Changed: false,
		}, nil
	}

	archivePath, digest, cleanup, err := downloadBundle(
		ctx,
		client,
		assetURL,
		userAgent,
		remoteDigest,
	)
	if err != nil {
		return BundleSyncResult{}, err
	}
	defer cleanup()

	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return BundleSyncResult{}, fmt.Errorf("open TurtleTools bundle: %w", err)
	}
	defer archive.Close()
	var metadata []byte
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(file.Name)
		if !safeRelativePath(name) {
			return BundleSyncResult{}, fmt.Errorf("unsafe bundle path %q", name)
		}
		base := filepath.Base(name)
		if base == "service_metadata.json" {
			body, err := readZipFile(file, defaultMaximumObject)
			if err != nil {
				return BundleSyncResult{}, err
			}
			metadata = body
		}
	}
	if len(metadata) == 0 {
		return BundleSyncResult{}, fmt.Errorf("TurtleTools bundle has no service_metadata.json")
	}
	var validate typeMetadata
	if err := json.Unmarshal(metadata, &validate); err != nil {
		return BundleSyncResult{}, fmt.Errorf("invalid TurtleTools metadata: %w", err)
	}
	result, err := importBundleImages(
		ctx,
		store,
		archive.File,
		options.Progress,
	)
	if err != nil {
		return BundleSyncResult{}, err
	}
	metadataKey := fmt.Sprintf(
		"types/releases/%s/service_metadata.json",
		digest,
	)
	if _, err := putIfChanged(ctx, store, importObject{
		Key: metadataKey, Body: metadata, ContentType: "application/json",
	}); err != nil {
		return BundleSyncResult{}, err
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	pointer := typeBundlePointer{
		Version: 1, Release: release, Digest: digest,
		MetadataKey: metadataKey, UpdatedAt: now().UTC(),
	}
	body, err := json.Marshal(pointer)
	if err != nil {
		return BundleSyncResult{}, err
	}
	// The pointer is the commit marker. A failed blob or metadata upload can
	// leave harmless unreferenced objects, but can never publish half a release.
	if err := store.PutWithOptions(
		ctx,
		"types/current.json",
		body,
		objectstore.PutOptions{
			ContentType: "application/json", CacheControl: "no-cache",
		},
	); err != nil {
		return BundleSyncResult{}, fmt.Errorf("publish type bundle: %w", err)
	}
	return BundleSyncResult{
		Release: release, Digest: digest, Changed: true, Import: result,
	}, nil
}

func importBundleImages(
	ctx context.Context,
	store ObjectStore,
	files []*zip.File,
	progress func(ImportResult),
) (ImportResult, error) {
	unique := make(map[string]*zip.File)
	for _, file := range files {
		if file.FileInfo().IsDir() ||
			filepath.Base(filepath.ToSlash(file.Name)) == "service_metadata.json" {
			continue
		}
		base := filepath.Base(filepath.ToSlash(file.Name))
		if !safeAssetName(base) {
			return ImportResult{}, fmt.Errorf("unsafe bundle filename %q", base)
		}
		if previous := unique[base]; previous != nil &&
			previous.Name != file.Name {
			return ImportResult{}, fmt.Errorf(
				"TurtleTools bundle contains duplicate filename %q",
				base,
			)
		}
		unique[base] = file
	}

	group, groupCtx := errgroup.WithContext(ctx)
	input := make(chan *zip.File, 8)
	var scanned, uploaded, skipped, uploadedBytes atomic.Int64
	for range 8 {
		group.Go(func() error {
			for file := range input {
				base := filepath.Base(filepath.ToSlash(file.Name))
				if !safeAssetName(base) {
					return fmt.Errorf("unsafe bundle filename %q", base)
				}
				contentType := mime.TypeByExtension(
					strings.ToLower(filepath.Ext(base)),
				)
				if !strings.HasPrefix(contentType, "image/") {
					continue
				}
				body, err := readZipFile(file, defaultMaximumObject)
				if err != nil {
					return err
				}
				changed, err := putIfChanged(groupCtx, store, importObject{
					Key:  "types/blobs/" + base,
					Body: body, ContentType: contentType,
				})
				if err != nil {
					return err
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
	return result, err
}

func latestBundle(
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
		if asset.Name == TurtleBundleAsset {
			return value.TagName, asset.BrowserDownloadURL, asset.Digest, nil
		}
	}
	return "", "", "", fmt.Errorf(
		"%s is missing from TurtleTools release %s",
		TurtleBundleAsset,
		value.TagName,
	)
}

func downloadBundle(
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
		return "", "", func() {}, fmt.Errorf("download TurtleTools bundle: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", func() {}, fmt.Errorf(
			"download TurtleTools bundle: HTTP %d",
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

func loadTypeBundlePointer(
	ctx context.Context,
	store ObjectStore,
) (*typeBundlePointer, error) {
	object, err := store.GetObject(ctx, "types/current.json")
	if err != nil || object == nil {
		return nil, err
	}
	var pointer typeBundlePointer
	if err := json.Unmarshal(object.Body, &pointer); err != nil {
		return nil, fmt.Errorf("decode current type bundle: %w", err)
	}
	return &pointer, nil
}

func trimDigest(value string) string {
	value = strings.TrimSpace(value)
	if before, after, found := strings.Cut(value, ":"); found &&
		strings.EqualFold(before, "sha256") {
		value = after
	}
	return strings.ToLower(value)
}
