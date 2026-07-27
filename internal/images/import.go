package images

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/eve-kill/shrike/internal/objectstore"
	"golang.org/x/sync/errgroup"
)

type ImportResult struct {
	Scanned  int64 `json:"scanned"`
	Uploaded int64 `json:"uploaded"`
	Skipped  int64 `json:"skipped"`
	Bytes    int64 `json:"bytes"`
}

type ImportOptions struct {
	Concurrency      int
	CacheDirectory   string
	Force            bool
	Progress         func(ImportResult)
	DownloadProgress func(completed, total int64)
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
