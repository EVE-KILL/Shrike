package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
)

const pruneBucketWidth = time.Hour

type filesystemUsage struct {
	BytesTotal  uint64
	BytesFree   uint64
	InodesTotal uint64
	InodesFree  uint64
}

// FilePruneOptions controls the high/low-watermark image eviction pass.
// Access recency comes from filesystem atime; production mounts must not use
// noatime. Relatime is sufficient and deliberately coarsens write traffic.
type FilePruneOptions struct {
	HighWatermarkPercent int
	LowWatermarkPercent  int
	MinimumAge           time.Duration
	DryRun               bool
	Now                  time.Time
}

type FilePruneResult struct {
	Triggered        bool   `json:"triggered"`
	DryRun           bool   `json:"dry_run"`
	ObjectsScanned   uint64 `json:"objects_scanned"`
	ObjectsDeleted   uint64 `json:"objects_deleted"`
	BytesReclaimed   uint64 `json:"bytes_reclaimed"`
	InodesReclaimed  uint64 `json:"inodes_reclaimed"`
	BytesUsedBefore  uint64 `json:"bytes_used_before"`
	BytesUsedAfter   uint64 `json:"bytes_used_after"`
	InodesUsedBefore uint64 `json:"inodes_used_before"`
	InodesUsedAfter  uint64 `json:"inodes_used_after"`
}

type pruneBucket struct {
	objects uint64
	bytes   uint64
	inodes  uint64
}

func readFilesystemUsage(root string) (filesystemUsage, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(root, &stat); err != nil {
		return filesystemUsage{}, err
	}
	return filesystemUsage{
		BytesTotal:  stat.Blocks * uint64(stat.Bsize),
		BytesFree:   stat.Bavail * uint64(stat.Bsize),
		InodesTotal: stat.Files, InodesFree: stat.Ffree,
	}, nil
}

// Prune evicts least-recently-accessed objects when either byte or inode use
// reaches the high watermark, and continues until both are below the low
// watermark. Metadata sidecars are always removed with their object.
func (s *FileStore) Prune(ctx context.Context, options FilePruneOptions) (FilePruneResult, error) {
	if options.HighWatermarkPercent == 0 {
		options.HighWatermarkPercent = 90
	}
	if options.LowWatermarkPercent == 0 {
		options.LowWatermarkPercent = 80
	}
	if options.MinimumAge == 0 {
		options.MinimumAge = 24 * time.Hour
	}
	if options.Now.IsZero() {
		options.Now = time.Now()
	}
	if options.LowWatermarkPercent < 1 || options.HighWatermarkPercent > 99 ||
		options.LowWatermarkPercent >= options.HighWatermarkPercent {
		return FilePruneResult{}, errors.New("watermarks must satisfy 1 <= low < high <= 99")
	}
	if options.MinimumAge < 0 {
		return FilePruneResult{}, errors.New("minimum age cannot be negative")
	}

	before, err := s.usage()
	if err != nil {
		return FilePruneResult{}, fmt.Errorf("stat image filesystem: %w", err)
	}
	result := FilePruneResult{DryRun: options.DryRun,
		BytesUsedBefore:  before.BytesTotal - before.BytesFree,
		InodesUsedBefore: before.InodesTotal - before.InodesFree}
	result.BytesUsedAfter, result.InodesUsedAfter = result.BytesUsedBefore, result.InodesUsedBefore
	if !atOrAbove(result.BytesUsedBefore, before.BytesTotal, options.HighWatermarkPercent) &&
		!atOrAbove(result.InodesUsedBefore, before.InodesTotal, options.HighWatermarkPercent) {
		return result, nil
	}
	result.Triggered = true
	bytesNeeded := reclaimNeeded(result.BytesUsedBefore, before.BytesTotal, options.LowWatermarkPercent)
	inodesNeeded := reclaimNeeded(result.InodesUsedBefore, before.InodesTotal, options.LowWatermarkPercent)
	oldestAllowed := options.Now.Add(-options.MinimumAge)
	buckets := make(map[int64]pruneBucket)
	err = filepath.WalkDir(s.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if !dynamicEntityDirectory(s.root, path) {
				return fs.SkipDir
			}
			return nil
		}
		if !dynamicEntityObjectPath(s.root, path) || strings.HasSuffix(entry.Name(), metadataSuffix) || strings.HasPrefix(entry.Name(), ".shrike-write-") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		accessed := fileAccessTime(info)
		if accessed.After(oldestAllowed) {
			return nil
		}
		bytes, inodes := objectFootprint(path, info)
		bucketID := accessed.Unix() / int64(pruneBucketWidth/time.Second)
		bucket := buckets[bucketID]
		bucket.objects++
		bucket.bytes += bytes
		bucket.inodes += inodes
		buckets[bucketID] = bucket
		result.ObjectsScanned++
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("scan image filesystem: %w", err)
	}

	keys := make([]int64, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var cutoff int64
	var availableBytes, availableInodes uint64
	for _, key := range keys {
		cutoff = key
		availableBytes += buckets[key].bytes
		availableInodes += buckets[key].inodes
		if availableBytes >= bytesNeeded && availableInodes >= inodesNeeded {
			break
		}
	}
	if len(keys) == 0 {
		return result, errors.New("watermark exceeded but no objects are old enough to evict")
	}
	root, err := os.OpenRoot(s.root)
	if err != nil {
		return result, fmt.Errorf("open image filesystem root: %w", err)
	}
	defer func() { _ = root.Close() }()

	err = filepath.WalkDir(s.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if result.BytesReclaimed >= bytesNeeded && result.InodesReclaimed >= inodesNeeded {
			return fs.SkipAll
		}
		if entry.IsDir() {
			if !dynamicEntityDirectory(s.root, path) {
				return fs.SkipDir
			}
			return nil
		}
		if !dynamicEntityObjectPath(s.root, path) || strings.HasSuffix(entry.Name(), metadataSuffix) || strings.HasPrefix(entry.Name(), ".shrike-write-") {
			return nil
		}
		info, err := entry.Info()
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		accessed := fileAccessTime(info)
		if accessed.After(oldestAllowed) || accessed.Unix()/int64(pruneBucketWidth/time.Second) > cutoff {
			return nil
		}
		bytes, inodes := objectFootprint(path, info)
		result.ObjectsDeleted++
		result.BytesReclaimed += bytes
		result.InodesReclaimed += inodes
		if options.DryRun {
			return nil
		}
		relative, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		if err := root.Remove(relative); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := root.Remove(relative + metadataSuffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		removeEmptyParents(root, filepath.Dir(relative))
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("evict image objects: %w", err)
	}
	if !options.DryRun {
		after, statErr := s.usage()
		if statErr != nil {
			return result, fmt.Errorf("stat image filesystem after eviction: %w", statErr)
		}
		result.BytesUsedAfter = after.BytesTotal - after.BytesFree
		result.InodesUsedAfter = after.InodesTotal - after.InodesFree
	}
	return result, nil
}

func objectFootprint(path string, info fs.FileInfo) (uint64, uint64) {
	bytes, inodes := uint64(info.Size()), uint64(1)
	if meta, err := os.Stat(path + metadataSuffix); err == nil {
		bytes += uint64(meta.Size())
		inodes++
	}
	return bytes, inodes
}

var dynamicEntityKinds = map[string]struct{}{
	"characters":   {},
	"corporations": {},
	"alliances":    {},
}

func dynamicEntityDirectory(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || relative == "entities" {
		return err == nil
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) < 2 || parts[0] != "entities" {
		return false
	}
	_, allowed := dynamicEntityKinds[parts[1]]
	return allowed
}

func dynamicEntityObjectPath(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) < 3 || parts[0] != "entities" {
		return false
	}
	_, allowed := dynamicEntityKinds[parts[1]]
	return allowed
}

func removeEmptyParents(root *os.Root, directory string) {
	for directory != "." && directory != "" {
		if err := root.Remove(directory); err != nil {
			return
		}
		directory = filepath.Dir(directory)
	}
}

func atOrAbove(used, total uint64, percent int) bool {
	return total > 0 && used*100 >= total*uint64(percent)
}

func reclaimNeeded(used, total uint64, percent int) uint64 {
	target := total * uint64(percent) / 100
	if used <= target {
		return 0
	}
	return used - target
}
