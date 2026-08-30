package objectstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

const metadataSuffix = ".shrike-meta.json"

// FileStore implements the image object-store contract on a local filesystem.
// Bodies and metadata are replaced atomically, so concurrent readers never see
// a partially written image.
type FileStore struct {
	root    string
	maximum int64
}

type fileMetadata struct {
	ContentType  string            `json:"content_type,omitempty"`
	CacheControl string            `json:"cache_control,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

func NewFileStore(root string, maximum int64) (*FileStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("filesystem object storage root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve filesystem object storage root: %w", err)
	}
	if maximum <= 0 {
		maximum = defaultMaximumBytes
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create filesystem object storage root: %w", err)
	}
	return &FileStore{root: abs, maximum: maximum}, nil
}

func (s *FileStore) objectPath(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(key))
	if key == "" || filepath.IsAbs(clean) || clean == "." || clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) ||
		strings.ContainsRune(key, '\\') {
		return "", fmt.Errorf("invalid object key %q", key)
	}
	return filepath.Join(s.root, clean), nil
}

func (s *FileStore) GetObject(_ context.Context, key string) (*Object, error) {
	path, err := s.objectPath(key)
	if err != nil {
		return nil, err
	}
	// path is rooted beneath s.root by objectPath's traversal validation.
	body, err := os.ReadFile(path) // #nosec G304
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read object %q: %w", key, err)
	}
	if int64(len(body)) > s.maximum {
		return nil, fmt.Errorf("object %q exceeds the configured size limit", key)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat object %q: %w", key, err)
	}
	meta, err := readFileMetadata(path + metadataSuffix)
	if err != nil {
		return nil, fmt.Errorf("read metadata for %q: %w", key, err)
	}
	return &Object{Body: body,
		Key: key, Size: info.Size(), ETag: digestETag(body),
		ContentType: meta.ContentType, CacheControl: meta.CacheControl,
		LastModified: info.ModTime(), Metadata: cloneMetadata(meta.Metadata)}, nil
}

func (s *FileStore) Stat(_ context.Context, key string) (*ObjectInfo, error) {
	path, err := s.objectPath(key)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat object %q: %w", key, err)
	}
	meta, err := readFileMetadata(path + metadataSuffix)
	if err != nil {
		return nil, fmt.Errorf("read metadata for %q: %w", key, err)
	}
	return &ObjectInfo{Key: key, Size: info.Size(), ContentType: meta.ContentType,
		CacheControl: meta.CacheControl, LastModified: info.ModTime(),
		Metadata: cloneMetadata(meta.Metadata)}, nil
}

func (s *FileStore) PutWithOptions(_ context.Context, key string, body []byte, options PutOptions) error {
	if int64(len(body)) > s.maximum {
		return fmt.Errorf("object %q exceeds the configured size limit", key)
	}
	path, err := s.objectPath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create object directory: %w", err)
	}
	meta, err := json.Marshal(fileMetadata{ContentType: options.ContentType,
		CacheControl: options.CacheControl, Metadata: cloneMetadata(options.Metadata)})
	if err != nil {
		return err
	}
	if err := atomicWrite(path+metadataSuffix, meta, 0o644); err != nil {
		return fmt.Errorf("write metadata for %q: %w", key, err)
	}
	if err := atomicWrite(path, body, 0o644); err != nil {
		return fmt.Errorf("write object %q: %w", key, err)
	}
	return nil
}

func (s *FileStore) Delete(_ context.Context, key string) error {
	path, err := s.objectPath(key)
	if err != nil {
		return err
	}
	for _, target := range []string{path, path + metadataSuffix} {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("delete object %q: %w", key, err)
		}
	}
	return nil
}

func atomicWrite(path string, body []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".shrike-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err = tmp.Chmod(mode); err == nil {
		_, err = tmp.Write(body)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func readFileMetadata(path string) (fileMetadata, error) {
	// Metadata paths are derived from an already validated object path.
	body, err := os.ReadFile(path) // #nosec G304
	if errors.Is(err, os.ErrNotExist) {
		return fileMetadata{}, nil
	}
	if err != nil {
		return fileMetadata{}, err
	}
	var meta fileMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return fileMetadata{}, err
	}
	return meta, nil
}

func digestETag(body []byte) string {
	sum := sha256.Sum256(body)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func cloneMetadata(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	copy := make(map[string]string, len(source))
	maps.Copy(copy, source)
	return copy
}
