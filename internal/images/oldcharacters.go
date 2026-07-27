package images

import (
	"archive/zip"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/objectstore"
	"golang.org/x/sync/errgroup"
)

const (
	OldCharacterArchiveURL = "https://data.everef.net/ccp/portraits/OldCharPortraits_256.zip"

	oldCharacterManifestVersion = 1
	oldCharacterManifestKey     = "oldcharacters/manifest.json"
	oldCharacterShardPrefix     = "oldcharacters/manifests/v1/"
	downloadProgressInterval    = 64 << 20
	oldCharacterUploadAttempts  = 4
)

var oldPortraitName = regexp.MustCompile(`^([0-9]+)_256\.jpg$`)

type oldCharacterManifest struct {
	Version       int       `json:"version"`
	ArchiveDigest string    `json:"archive_digest"`
	Images        int64     `json:"images"`
	Bytes         int64     `json:"bytes"`
	Shards        int       `json:"shards"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type oldCharacterShardManifest struct {
	Version       int       `json:"version"`
	ArchiveDigest string    `json:"archive_digest"`
	Shard         string    `json:"shard"`
	Images        int64     `json:"images"`
	Bytes         int64     `json:"bytes"`
	CompletedAt   time.Time `json:"completed_at"`
}

type oldCharacterArchive struct {
	source       string
	path         string
	digest       string
	expectedSHA1 string
	size         int64
	etag         string
	lastModified string
	remote       bool
}

type oldCharacterArchiveCache struct {
	Source       string `json:"source"`
	Digest       string `json:"digest,omitempty"`
	Size         int64  `json:"size"`
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
}

type oldCharacterProgress struct {
	mu       sync.Mutex
	result   ImportResult
	next     int64
	callback func(ImportResult)
}

// ImportOldCharacters downloads or opens the static EVE Ref archive and
// uploads individual portraits. Remote archives are kept in CacheDirectory and
// resumed with HTTP Range requests. B2 completion markers are published after
// each two-digit shard, so a restart neither probes every object nor repeats
// more than the interrupted shard.
func ImportOldCharacters(
	ctx context.Context,
	store ObjectStore,
	source string,
	client *http.Client,
	options ImportOptions,
) (ImportResult, error) {
	if store == nil {
		return ImportResult{}, unavailable()
	}
	if source == "" {
		source = OldCharacterArchiveURL
	}
	if client == nil {
		client = &http.Client{}
	}

	descriptor, cleanup, err := inspectOldCharacterArchive(
		ctx,
		source,
		client,
		options.CacheDirectory,
	)
	if err != nil {
		return ImportResult{}, err
	}
	defer cleanup()

	if descriptor.digest != "" && !options.Force {
		current, err := loadOldCharacterManifest(
			ctx,
			store,
			oldCharacterManifestKey,
		)
		if err != nil {
			return ImportResult{}, err
		}
		if current != nil &&
			current.Version == oldCharacterManifestVersion &&
			current.ArchiveDigest == descriptor.digest {
			result := ImportResult{
				Scanned: current.Images,
				Skipped: current.Images,
			}
			if options.Progress != nil {
				options.Progress(result)
			}
			return result, nil
		}
	}

	if descriptor.remote {
		if err := materializeOldCharacterArchive(
			ctx,
			client,
			descriptor,
			options.DownloadProgress,
		); err != nil {
			return ImportResult{}, err
		}
	}
	if descriptor.digest == "" {
		digest, err := hashFile(descriptor.path, "sha256")
		if err != nil {
			return ImportResult{}, err
		}
		descriptor.digest = "sha256:" + digest
	}

	if !options.Force {
		current, err := loadOldCharacterManifest(
			ctx,
			store,
			oldCharacterManifestKey,
		)
		if err != nil {
			return ImportResult{}, err
		}
		if current != nil &&
			current.Version == oldCharacterManifestVersion &&
			current.ArchiveDigest == descriptor.digest {
			result := ImportResult{
				Scanned: current.Images,
				Skipped: current.Images,
			}
			if options.Progress != nil {
				options.Progress(result)
			}
			return result, nil
		}
	}

	archive, err := zip.OpenReader(descriptor.path)
	if err != nil {
		return ImportResult{}, fmt.Errorf("open old portrait archive: %w", err)
	}
	defer archive.Close()

	shards, missing, totalImages, totalBytes := indexOldCharacterArchive(
		archive.File,
	)
	progress := &oldCharacterProgress{
		next:     1_000,
		callback: options.Progress,
	}
	concurrency := options.Concurrency
	if concurrency <= 0 {
		concurrency = 64
	}

	shardCount := 0
	if len(missing) > 0 {
		shardCount++
		if err := importOldCharacterShard(
			ctx,
			store,
			"missing",
			missing,
			descriptor.digest,
			concurrency,
			options.Force,
			progress,
		); err != nil {
			return progress.snapshot(), err
		}
	}
	for number := range 100 {
		name := fmt.Sprintf("%02d", number)
		files := shards[number]
		if len(files) == 0 {
			continue
		}
		shardCount++
		if err := importOldCharacterShard(
			ctx,
			store,
			name,
			files,
			descriptor.digest,
			concurrency,
			options.Force,
			progress,
		); err != nil {
			return progress.snapshot(), err
		}
	}

	manifest := oldCharacterManifest{
		Version:       oldCharacterManifestVersion,
		ArchiveDigest: descriptor.digest,
		Images:        totalImages,
		Bytes:         totalBytes,
		Shards:        shardCount,
		UpdatedAt:     time.Now().UTC(),
	}
	if err := putOldCharacterManifest(
		ctx,
		store,
		oldCharacterManifestKey,
		manifest,
	); err != nil {
		return progress.snapshot(), fmt.Errorf(
			"publish old character manifest: %w",
			err,
		)
	}
	result := progress.snapshot()
	if options.Progress != nil {
		options.Progress(result)
	}
	return result, nil
}

func indexOldCharacterArchive(
	files []*zip.File,
) ([100][]*zip.File, []*zip.File, int64, int64) {
	var shards [100][]*zip.File
	var missing []*zip.File
	var images int64
	var bytes int64
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(filepath.ToSlash(file.Name))
		if base == "missing_256.jpg" {
			missing = append(missing, file)
			images++
			bytes += int64(file.UncompressedSize64)
			continue
		}
		match := oldPortraitName.FindStringSubmatch(base)
		if match == nil {
			continue
		}
		id := match[1]
		if len(id) == 1 {
			id = "0" + id
		}
		shard, err := strconv.Atoi(id[len(id)-2:])
		if err != nil {
			continue
		}
		shards[shard] = append(shards[shard], file)
		images++
		bytes += int64(file.UncompressedSize64)
	}
	return shards, missing, images, bytes
}

func importOldCharacterShard(
	ctx context.Context,
	store ObjectStore,
	shard string,
	files []*zip.File,
	archiveDigest string,
	concurrency int,
	force bool,
	progress *oldCharacterProgress,
) error {
	var size int64
	for _, file := range files {
		size += int64(file.UncompressedSize64)
	}
	markerKey := oldCharacterShardPrefix + shard + ".json"
	if !force {
		current, err := loadOldCharacterShardManifest(ctx, store, markerKey)
		if err != nil {
			return err
		}
		if current != nil &&
			current.Version == oldCharacterManifestVersion &&
			current.ArchiveDigest == archiveDigest &&
			current.Shard == shard &&
			current.Images == int64(len(files)) &&
			current.Bytes == size {
			progress.skipped(int64(len(files)))
			return nil
		}
	}

	group, groupCtx := errgroup.WithContext(ctx)
	input := make(chan *zip.File, concurrency)
	for range concurrency {
		group.Go(func() error {
			for file := range input {
				body, err := readZipFile(file, 8<<20)
				if err != nil {
					return err
				}
				key, err := oldCharacterObjectKey(file.Name)
				if err != nil {
					return err
				}
				sum := sha256.Sum256(body)
				digest := hex.EncodeToString(sum[:])
				if err := putOldCharacterObject(
					groupCtx,
					store,
					key,
					body,
					objectstore.PutOptions{
						ContentType:  "image/jpeg",
						CacheControl: immutableCacheControl,
						Metadata:     map[string]string{"sha256": digest},
					},
					sleepOldCharacterRetry,
				); err != nil {
					return fmt.Errorf("upload %s: %w", key, err)
				}
				progress.uploaded(int64(len(body)))
			}
			return nil
		})
	}
	group.Go(func() error {
		defer close(input)
		for _, file := range files {
			select {
			case input <- file:
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
		return nil
	})
	if err := group.Wait(); err != nil {
		return err
	}

	marker := oldCharacterShardManifest{
		Version:       oldCharacterManifestVersion,
		ArchiveDigest: archiveDigest,
		Shard:         shard,
		Images:        int64(len(files)),
		Bytes:         size,
		CompletedAt:   time.Now().UTC(),
	}
	if err := putOldCharacterManifest(ctx, store, markerKey, marker); err != nil {
		return fmt.Errorf("publish old character shard %s: %w", shard, err)
	}
	return nil
}

func putOldCharacterObject(
	ctx context.Context,
	store ObjectStore,
	key string,
	body []byte,
	options objectstore.PutOptions,
	sleep func(context.Context, time.Duration) error,
) error {
	var err error
	for attempt := range oldCharacterUploadAttempts {
		err = store.PutWithOptions(ctx, key, body, options)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempt == oldCharacterUploadAttempts-1 {
			break
		}
		delay := min(2*time.Second<<attempt, 10*time.Second)
		if err := sleep(ctx, delay); err != nil {
			return err
		}
	}
	return err
}

func sleepOldCharacterRetry(
	ctx context.Context,
	delay time.Duration,
) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func oldCharacterObjectKey(name string) (string, error) {
	base := filepath.Base(filepath.ToSlash(name))
	if base == "missing_256.jpg" {
		return "oldcharacters/missing_256.jpg", nil
	}
	match := oldPortraitName.FindStringSubmatch(base)
	if match == nil {
		return "", fmt.Errorf("invalid old portrait archive entry %q", name)
	}
	id := match[1]
	if len(id) == 1 {
		id = "0" + id
	}
	return fmt.Sprintf(
		"oldcharacters/%s/%s/%s",
		id[len(id)-2:len(id)-1],
		id[len(id)-1:],
		base,
	), nil
}

func (p *oldCharacterProgress) uploaded(bytes int64) {
	p.mu.Lock()
	p.result.Scanned++
	p.result.Uploaded++
	p.result.Bytes += bytes
	p.reportLocked()
	p.mu.Unlock()
}

func (p *oldCharacterProgress) skipped(count int64) {
	p.mu.Lock()
	p.result.Scanned += count
	p.result.Skipped += count
	p.reportLocked()
	p.mu.Unlock()
}

func (p *oldCharacterProgress) reportLocked() {
	if p.callback == nil || p.result.Scanned < p.next {
		return
	}
	p.next = ((p.result.Scanned / 1_000) + 1) * 1_000
	p.callback(p.result)
}

func (p *oldCharacterProgress) snapshot() ImportResult {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.result
}

func loadOldCharacterManifest(
	ctx context.Context,
	store ObjectStore,
	key string,
) (*oldCharacterManifest, error) {
	object, err := store.GetObject(ctx, key)
	if err != nil || object == nil {
		return nil, err
	}
	var manifest oldCharacterManifest
	if err := json.Unmarshal(object.Body, &manifest); err != nil {
		return nil, fmt.Errorf("decode %s: %w", key, err)
	}
	return &manifest, nil
}

func loadOldCharacterShardManifest(
	ctx context.Context,
	store ObjectStore,
	key string,
) (*oldCharacterShardManifest, error) {
	object, err := store.GetObject(ctx, key)
	if err != nil || object == nil {
		return nil, err
	}
	var manifest oldCharacterShardManifest
	if err := json.Unmarshal(object.Body, &manifest); err != nil {
		return nil, fmt.Errorf("decode %s: %w", key, err)
	}
	return &manifest, nil
}

func putOldCharacterManifest(
	ctx context.Context,
	store ObjectStore,
	key string,
	manifest any,
) error {
	body, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	return store.PutWithOptions(
		ctx,
		key,
		body,
		objectstore.PutOptions{
			ContentType:  "application/json",
			CacheControl: "no-store",
		},
	)
}

func inspectOldCharacterArchive(
	ctx context.Context,
	source string,
	client *http.Client,
	cacheDirectory string,
) (*oldCharacterArchive, func(), error) {
	parsed, err := url.Parse(source)
	if err != nil {
		return nil, func() {}, fmt.Errorf("parse old portrait archive URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		archivePath, err := filepath.Abs(source)
		if err != nil {
			return nil, func() {}, err
		}
		info, err := os.Stat(archivePath)
		if err != nil {
			return nil, func() {}, fmt.Errorf(
				"stat old portrait archive: %w",
				err,
			)
		}
		digest, err := hashFile(archivePath, "sha256")
		if err != nil {
			return nil, func() {}, err
		}
		return &oldCharacterArchive{
			source: source,
			path:   archivePath,
			digest: "sha256:" + digest,
			size:   info.Size(),
		}, func() {}, nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodHead, source, nil)
	if err != nil {
		return nil, func() {}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, func() {}, fmt.Errorf("inspect old portrait archive: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, func() {}, fmt.Errorf(
			"inspect old portrait archive: HTTP %d",
			response.StatusCode,
		)
	}
	if response.ContentLength < 1 {
		return nil, func() {}, errors.New(
			"old portrait archive has no Content-Length",
		)
	}

	cleanup := func() {}
	if cacheDirectory == "" {
		cacheDirectory, err = os.MkdirTemp("", "shrike-oldcharacters-*")
		if err != nil {
			return nil, func() {}, err
		}
		cleanup = func() { _ = os.RemoveAll(cacheDirectory) }
	} else if err := os.MkdirAll(cacheDirectory, 0o755); err != nil {
		return nil, func() {}, fmt.Errorf(
			"create old portrait cache: %w",
			err,
		)
	}
	filename := path.Base(parsed.Path)
	if filename == "." || filename == "/" || filename == "" {
		filename = "OldCharPortraits_256.zip"
	}
	expectedSHA1 := strings.ToLower(strings.TrimSpace(
		response.Header.Get("X-Amz-Meta-Large-File-Sha1"),
	))
	if decoded, decodeErr := hex.DecodeString(expectedSHA1); decodeErr != nil ||
		len(decoded) != sha1.Size {
		expectedSHA1 = ""
	}
	digest := ""
	if expectedSHA1 != "" {
		digest = "sha1:" + expectedSHA1
	}
	return &oldCharacterArchive{
		source:       source,
		path:         filepath.Join(cacheDirectory, filename),
		digest:       digest,
		expectedSHA1: expectedSHA1,
		size:         response.ContentLength,
		etag:         strings.TrimSpace(response.Header.Get("ETag")),
		lastModified: strings.TrimSpace(response.Header.Get("Last-Modified")),
		remote:       true,
	}, cleanup, nil
}

func materializeOldCharacterArchive(
	ctx context.Context,
	client *http.Client,
	archive *oldCharacterArchive,
	progress func(completed, total int64),
) error {
	cache := oldCharacterArchiveCache{
		Source: archive.source, Digest: archive.digest, Size: archive.size,
		ETag: archive.etag, LastModified: archive.lastModified,
	}
	if cachedOldCharacterArchiveUsable(archive.path, cache) {
		if progress != nil {
			progress(archive.size, archive.size)
		}
		return nil
	}
	if info, err := os.Stat(archive.path); err == nil &&
		info.Size() == archive.size {
		valid, digest, hashErr := verifyOldCharacterArchive(
			archive.path,
			archive.expectedSHA1,
		)
		if hashErr != nil {
			return hashErr
		}
		if valid {
			if archive.digest == "" {
				archive.digest = digest
				cache.Digest = digest
			}
			if err := writeArchiveCache(archive.path+".json", cache); err != nil {
				return err
			}
			if progress != nil {
				progress(archive.size, archive.size)
			}
			return nil
		}
		if err := os.Remove(archive.path); err != nil {
			return fmt.Errorf("replace invalid old portrait archive: %w", err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat old portrait archive cache: %w", err)
	}

	partialPath := archive.path + ".part"
	partialCachePath := partialPath + ".json"
	if !cachedOldCharacterPartialUsable(partialPath, partialCachePath, cache) {
		if err := os.Remove(partialPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("replace partial old portrait archive: %w", err)
		}
	}
	if err := writeArchiveCache(partialCachePath, cache); err != nil {
		return err
	}

	offset := int64(0)
	if info, err := os.Stat(partialPath); err == nil {
		offset = info.Size()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat partial old portrait archive: %w", err)
	}
	if offset > archive.size {
		offset = 0
		if err := os.Truncate(partialPath, 0); err != nil {
			return fmt.Errorf("truncate partial old portrait archive: %w", err)
		}
	}
	if offset < archive.size {
		if err := downloadOldCharacterArchive(
			ctx,
			client,
			archive,
			partialPath,
			offset,
			progress,
		); err != nil {
			return err
		}
	}

	info, err := os.Stat(partialPath)
	if err != nil {
		return fmt.Errorf("stat downloaded old portrait archive: %w", err)
	}
	if info.Size() != archive.size {
		return fmt.Errorf(
			"download old portrait archive: got %d bytes, want %d",
			info.Size(),
			archive.size,
		)
	}
	valid, digest, err := verifyOldCharacterArchive(
		partialPath,
		archive.expectedSHA1,
	)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("download old portrait archive: SHA-1 mismatch")
	}
	if archive.digest == "" {
		archive.digest = digest
		cache.Digest = digest
	}
	if err := os.Rename(partialPath, archive.path); err != nil {
		return fmt.Errorf("publish old portrait archive cache: %w", err)
	}
	if err := writeArchiveCache(archive.path+".json", cache); err != nil {
		return err
	}
	_ = os.Remove(partialCachePath)
	if progress != nil {
		progress(archive.size, archive.size)
	}
	return nil
}

func downloadOldCharacterArchive(
	ctx context.Context,
	client *http.Client,
	archive *oldCharacterArchive,
	partialPath string,
	offset int64,
	progress func(completed, total int64),
) error {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		archive.source,
		nil,
	)
	if err != nil {
		return err
	}
	if offset > 0 {
		request.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		if archive.etag != "" {
			request.Header.Set("If-Range", archive.etag)
		}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download old portrait archive: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf(
			"download old portrait archive: HTTP %d",
			response.StatusCode,
		)
	}
	appendDownload := offset > 0 && response.StatusCode == http.StatusPartialContent
	if appendDownload {
		prefix := fmt.Sprintf("bytes %d-", offset)
		if !strings.HasPrefix(response.Header.Get("Content-Range"), prefix) {
			return fmt.Errorf(
				"download old portrait archive: invalid Content-Range %q",
				response.Header.Get("Content-Range"),
			)
		}
	} else {
		offset = 0
	}

	flags := os.O_CREATE | os.O_WRONLY
	if appendDownload {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(partialPath, flags, 0o600)
	if err != nil {
		return fmt.Errorf("open partial old portrait archive: %w", err)
	}
	writer := &archiveProgressWriter{
		writer: file, completed: offset, total: archive.size,
		next: offset + downloadProgressInterval, callback: progress,
	}
	if progress != nil {
		progress(offset, archive.size)
	}
	_, copyErr := io.Copy(writer, response.Body)
	syncErr := file.Sync()
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("download old portrait archive: %w", copyErr)
	}
	if syncErr != nil {
		return fmt.Errorf("sync old portrait archive: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close old portrait archive: %w", closeErr)
	}
	return nil
}

type archiveProgressWriter struct {
	writer    io.Writer
	completed int64
	total     int64
	next      int64
	callback  func(completed, total int64)
}

func (w *archiveProgressWriter) Write(body []byte) (int, error) {
	written, err := w.writer.Write(body)
	w.completed += int64(written)
	if w.callback != nil && w.completed >= w.next {
		w.next = w.completed + downloadProgressInterval
		w.callback(w.completed, w.total)
	}
	return written, err
}

func cachedOldCharacterArchiveUsable(
	archivePath string,
	expected oldCharacterArchiveCache,
) bool {
	info, err := os.Stat(archivePath)
	if err != nil || info.Size() != expected.Size {
		return false
	}
	var cached oldCharacterArchiveCache
	if err := readJSONFile(archivePath+".json", &cached); err != nil {
		return false
	}
	return archiveCacheMatches(cached, expected)
}

func cachedOldCharacterPartialUsable(
	partialPath string,
	cachePath string,
	expected oldCharacterArchiveCache,
) bool {
	info, err := os.Stat(partialPath)
	if err != nil || info.Size() > expected.Size {
		return false
	}
	var cached oldCharacterArchiveCache
	if err := readJSONFile(cachePath, &cached); err != nil {
		return false
	}
	return archiveCacheMatches(cached, expected)
}

func archiveCacheMatches(
	cached oldCharacterArchiveCache,
	expected oldCharacterArchiveCache,
) bool {
	return cached.Source == expected.Source &&
		cached.Size == expected.Size &&
		cached.ETag == expected.ETag &&
		cached.LastModified == expected.LastModified &&
		(expected.Digest == "" || cached.Digest == expected.Digest)
}

func verifyOldCharacterArchive(
	archivePath string,
	expectedSHA1 string,
) (bool, string, error) {
	algorithm := "sha256"
	if expectedSHA1 != "" {
		algorithm = "sha1"
	}
	digest, err := hashFile(archivePath, algorithm)
	if err != nil {
		return false, "", err
	}
	if expectedSHA1 != "" && digest != expectedSHA1 {
		return false, "sha1:" + digest, nil
	}
	return true, algorithm + ":" + digest, nil
}

func hashFile(filename string, algorithm string) (string, error) {
	var digest hash.Hash
	switch algorithm {
	case "sha1":
		digest = sha1.New()
	case "sha256":
		digest = sha256.New()
	default:
		return "", fmt.Errorf("unsupported archive digest %q", algorithm)
	}
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("open archive for hashing: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(digest, file); err != nil {
		return "", fmt.Errorf("hash archive: %w", err)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func readJSONFile(filename string, destination any) error {
	body, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, destination)
}

func writeArchiveCache(
	filename string,
	cache oldCharacterArchiveCache,
) error {
	body, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(filename), ".archive-*.json")
	if err != nil {
		return fmt.Errorf("create archive cache metadata: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()
	if _, err := temporary.Write(body); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write archive cache metadata: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync archive cache metadata: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close archive cache metadata: %w", err)
	}
	if err := os.Rename(temporaryName, filename); err != nil {
		return fmt.Errorf("publish archive cache metadata: %w", err)
	}
	return nil
}
