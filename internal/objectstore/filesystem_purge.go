package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// LegacyEntityPurgeResult reports the obsolete dynamic image trees only.
type LegacyEntityPurgeResult struct {
	Applied bool   `json:"applied"`
	Files   uint64 `json:"files"`
	Bytes   uint64 `json:"bytes"`
}

// PurgeLegacyEntities removes the old character, corporation, and alliance
// cache trees. No other image category is eligible. A dry run is the default
// caller behavior; apply must be explicit at the CLI boundary.
func (s *FileStore) PurgeLegacyEntities(ctx context.Context, apply bool) (LegacyEntityPurgeResult, error) {
	result := LegacyEntityPurgeResult{Applied: apply}
	root, err := os.OpenRoot(s.root)
	if err != nil {
		return result, fmt.Errorf("open image root: %w", err)
	}
	defer root.Close() //nolint:errcheck

	for _, kind := range []string{"characters", "corporations", "alliances"} {
		relative := filepath.Join("entities", kind)
		path := filepath.Join(s.root, relative)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return result, fmt.Errorf("inspect %s: %w", relative, err)
		}
		if !info.IsDir() {
			return result, fmt.Errorf("%s is not a directory", relative)
		}
		err = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			result.Files++
			result.Bytes += uint64(info.Size())
			return nil
		})
		if err != nil {
			return result, fmt.Errorf("scan %s: %w", relative, err)
		}
		if apply {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			if err := root.RemoveAll(relative); err != nil {
				return result, fmt.Errorf("remove %s: %w", relative, err)
			}
		}
	}
	return result, nil
}
