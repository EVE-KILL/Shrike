package images

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/eve-kill/shrike/internal/objectstore"
	"golang.org/x/sync/errgroup"
)

const OldCharacterArchiveURL = "https://data.everef.net/ccp/portraits/OldCharPortraits_256.zip"

type ImportResult struct {
	Scanned  int64 `json:"scanned"`
	Uploaded int64 `json:"uploaded"`
	Skipped  int64 `json:"skipped"`
	Bytes    int64 `json:"bytes"`
}

type ImportOptions struct {
	Concurrency int
	Progress    func(ImportResult)
}

type importObject struct {
	Key         string
	Body        []byte
	ContentType string
}

type localImportObject struct {
	Path        string
	Key         string
	ContentType string
}

var oldPortraitName = regexp.MustCompile(`^([0-9]+)_256\.jpg$`)

// ImportStaticTree uploads the checked-in image-server assets that are not
// supplied by the TurtleTools release: maps, UI icons, Dust 514 images, and
// overlay definitions.
func ImportStaticTree(
	ctx context.Context,
	store ObjectStore,
	root string,
	options ImportOptions,
) (ImportResult, error) {
	if store == nil {
		return ImportResult{}, unavailable()
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return ImportResult{}, err
	}
	mappings := []struct {
		local  string
		remote string
	}{
		{"regions", "static/regions"},
		{"systems", "static/systems"},
		{"constellations", "static/constellations"},
		{"ui", "static/ui"},
		{"dust514", "types/dust514"},
		{"overlays", "types/overlays"},
	}
	var objects []localImportObject
	for _, mapping := range mappings {
		localRoot := filepath.Join(root, mapping.local)
		info, statErr := os.Stat(localRoot)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return ImportResult{}, statErr
		}
		if !info.IsDir() {
			return ImportResult{}, fmt.Errorf("%s is not a directory", localRoot)
		}
		walkErr := filepath.WalkDir(
			localRoot,
			func(localPath string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() {
					return nil
				}
				relative, err := filepath.Rel(localRoot, localPath)
				if err != nil {
					return err
				}
				relative = filepath.ToSlash(relative)
				if !safeRelativePath(relative) {
					return fmt.Errorf("unsafe static asset path %q", relative)
				}
				contentType := mime.TypeByExtension(
					strings.ToLower(filepath.Ext(relative)),
				)
				if contentType == "" {
					contentType = "application/octet-stream"
				}
				objects = append(objects, localImportObject{
					Path:        localPath,
					Key:         mapping.remote + "/" + relative,
					ContentType: contentType,
				})
				return nil
			},
		)
		if walkErr != nil {
			return ImportResult{}, walkErr
		}
	}
	return importLocalObjects(ctx, store, objects, options)
}

// ImportOldCharacters downloads or opens the static EVE Ref archive and
// uploads individual portraits. The archive is never held in memory and the
// operation is safe to restart: object SHA-256 metadata suppresses unchanged
// writes.
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
	archivePath, cleanup, err := materializeArchive(ctx, source, client)
	if err != nil {
		return ImportResult{}, err
	}
	defer cleanup()

	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return ImportResult{}, fmt.Errorf("open old portrait archive: %w", err)
	}
	defer archive.Close()

	concurrency := options.Concurrency
	if concurrency <= 0 {
		concurrency = 8
	}
	group, groupCtx := errgroup.WithContext(ctx)
	files := make(chan *zip.File, concurrency)
	var result ImportResult
	var resultMu sync.Mutex
	for range concurrency {
		group.Go(func() error {
			for file := range files {
				body, err := readZipFile(file, 8<<20)
				if err != nil {
					return err
				}
				base := filepath.Base(filepath.ToSlash(file.Name))
				key := ""
				if base == "missing_256.jpg" {
					key = "oldcharacters/missing_256.jpg"
				} else if match := oldPortraitName.FindStringSubmatch(base); match != nil {
					id := match[1]
					padded := strings.Repeat("0", max(0, 2-len(id))) + id
					key = fmt.Sprintf(
						"oldcharacters/%s/%s/%s",
						padded[len(padded)-2:len(padded)-1],
						padded[len(padded)-1:],
						base,
					)
				}
				if key == "" {
					continue
				}
				uploaded, err := putIfChanged(
					groupCtx,
					store,
					importObject{
						Key: key, Body: body, ContentType: "image/jpeg",
					},
				)
				if err != nil {
					return err
				}
				resultMu.Lock()
				result.Scanned++
				if uploaded {
					result.Uploaded++
					result.Bytes += int64(len(body))
				} else {
					result.Skipped++
				}
				snapshot := result
				resultMu.Unlock()
				if options.Progress != nil && snapshot.Scanned%10_000 == 0 {
					options.Progress(snapshot)
				}
			}
			return nil
		})
	}
	group.Go(func() error {
		defer close(files)
		for _, file := range archive.File {
			if file.FileInfo().IsDir() {
				continue
			}
			select {
			case files <- file:
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
		return nil
	})
	if err := group.Wait(); err != nil {
		return result, err
	}
	if options.Progress != nil {
		options.Progress(result)
	}
	return result, nil
}

func importLocalObjects(
	ctx context.Context,
	store ObjectStore,
	objects []localImportObject,
	options ImportOptions,
) (ImportResult, error) {
	concurrency := options.Concurrency
	if concurrency <= 0 {
		concurrency = 8
	}
	group, groupCtx := errgroup.WithContext(ctx)
	input := make(chan localImportObject, concurrency)
	var scanned, uploaded, skipped, uploadedBytes atomic.Int64
	for range concurrency {
		group.Go(func() error {
			for object := range input {
				body, err := os.ReadFile(object.Path)
				if err != nil {
					return fmt.Errorf("read %s: %w", object.Path, err)
				}
				changed, err := putIfChanged(groupCtx, store, importObject{
					Key: object.Key, Body: body, ContentType: object.ContentType,
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
				if options.Progress != nil && scanned.Load()%1_000 == 0 {
					options.Progress(ImportResult{
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
		for _, object := range objects {
			select {
			case input <- object:
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
	if options.Progress != nil {
		options.Progress(result)
	}
	return result, err
}

func putIfChanged(
	ctx context.Context,
	store ObjectStore,
	object importObject,
) (bool, error) {
	sum := sha256.Sum256(object.Body)
	digest := hex.EncodeToString(sum[:])
	info, err := store.Stat(ctx, object.Key)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", object.Key, err)
	}
	if info != nil && info.Metadata["sha256"] == digest {
		return false, nil
	}
	if err := store.PutWithOptions(
		ctx,
		object.Key,
		object.Body,
		objectstore.PutOptions{
			ContentType: object.ContentType, CacheControl: immutableCacheControl,
			Metadata: map[string]string{"sha256": digest},
		},
	); err != nil {
		return false, fmt.Errorf("upload %s: %w", object.Key, err)
	}
	return true, nil
}

func materializeArchive(
	ctx context.Context,
	source string,
	client *http.Client,
) (string, func(), error) {
	if !strings.HasPrefix(source, "http://") &&
		!strings.HasPrefix(source, "https://") {
		path, err := filepath.Abs(source)
		return path, func() {}, err
	}
	if client == nil {
		client = &http.Client{}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return "", func() {}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", func() {}, fmt.Errorf("download archive: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", func() {}, fmt.Errorf(
			"download archive: HTTP %d",
			response.StatusCode,
		)
	}
	file, err := os.CreateTemp("", "shrike-images-*.zip")
	if err != nil {
		return "", func() {}, err
	}
	name := file.Name()
	cleanup := func() { _ = os.Remove(name) }
	if _, err := io.Copy(file, response.Body); err != nil {
		_ = file.Close()
		cleanup()
		return "", func() {}, fmt.Errorf("download archive: %w", err)
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return name, cleanup, nil
}

func readZipFile(file *zip.File, maximum int64) ([]byte, error) {
	if file.UncompressedSize64 > uint64(maximum) {
		return nil, fmt.Errorf("archive entry %q exceeds %d bytes", file.Name, maximum)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	body, err := readBounded(reader, maximum)
	if err != nil {
		return nil, fmt.Errorf("read archive entry %q: %w", file.Name, err)
	}
	return body, nil
}

func safeRelativePath(name string) bool {
	clean := filepath.ToSlash(filepath.Clean(name))
	return name != "" && clean == name && clean != "." &&
		!strings.HasPrefix(clean, "../") &&
		!strings.HasPrefix(clean, "/") &&
		!strings.ContainsRune(clean, '\\')
}
