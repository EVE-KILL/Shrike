package objectstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStoreRoundTrip(t *testing.T) {
	store, err := NewFileStore(t.TempDir(), 1024)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	options := PutOptions{ContentType: "image/png", CacheControl: "immutable",
		Metadata: map[string]string{"sha256": "known"}}
	if err := store.PutWithOptions(ctx, "static/systems/1.png", []byte("png"), options); err != nil {
		t.Fatal(err)
	}
	object, err := store.GetObject(ctx, "static/systems/1.png")
	if err != nil {
		t.Fatal(err)
	}
	if string(object.Body) != "png" || object.ContentType != "image/png" || object.Metadata["sha256"] != "known" {
		t.Fatalf("object = %#v", object)
	}
	info, err := store.Stat(ctx, "static/systems/1.png")
	if err != nil {
		t.Fatal(err)
	}
	if info == nil || info.Size != 3 {
		t.Fatalf("info = %#v", info)
	}
	if err := store.Delete(ctx, "static/systems/1.png"); err != nil {
		t.Fatal(err)
	}
	object, err = store.GetObject(ctx, "static/systems/1.png")
	if err != nil || object != nil {
		t.Fatalf("deleted object = %#v, %v", object, err)
	}
}

func TestFileStorePruneEvictsOldestObjectAtInodeWatermark(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStore(root, 1024)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, key := range []string{
		"entities/characters/1/old/image.webp",
		"entities/corporations/2/recent/image.webp",
		"types/static/image.webp",
		"oldcharacters/3_256.jpg",
	} {
		if err := store.PutWithOptions(ctx, key, []byte("image body"), PutOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().Round(time.Second)
	old := now.Add(-48 * time.Hour)
	recent := now.Add(-2 * time.Hour)
	for _, key := range []string{
		"entities/characters/1/old/image.webp",
		"types/static/image.webp",
		"oldcharacters/3_256.jpg",
	} {
		for _, suffix := range []string{"", metadataSuffix} {
			if err := os.Chtimes(filepath.Join(root, key)+suffix, old, old); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, suffix := range []string{"", metadataSuffix} {
		if err := os.Chtimes(filepath.Join(root, "entities/corporations/2/recent/image.webp")+suffix, recent, recent); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	store.usage = func() (filesystemUsage, error) {
		calls++
		if calls == 1 {
			return filesystemUsage{BytesTotal: 100, BytesFree: 50, InodesTotal: 10, InodesFree: 1}, nil
		}
		return filesystemUsage{BytesTotal: 100, BytesFree: 50, InodesTotal: 10, InodesFree: 3}, nil
	}
	result, err := store.Prune(ctx, FilePruneOptions{
		HighWatermarkPercent: 90, LowWatermarkPercent: 70,
		MinimumAge: 24 * time.Hour, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Triggered || result.ObjectsDeleted != 1 || result.InodesReclaimed != 2 {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "entities/characters/1/old/image.webp")); !os.IsNotExist(err) {
		t.Fatalf("old object still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "entities/corporations/2/recent/image.webp")); err != nil {
		t.Fatalf("recent object was removed: %v", err)
	}
	for _, key := range []string{"types/static/image.webp", "oldcharacters/3_256.jpg"} {
		if _, err := os.Stat(filepath.Join(root, key)); err != nil {
			t.Fatalf("static object %q was removed: %v", key, err)
		}
	}
}

func TestFileStorePruneDoesNothingBelowHighWatermark(t *testing.T) {
	store, err := NewFileStore(t.TempDir(), 1024)
	if err != nil {
		t.Fatal(err)
	}
	store.usage = func() (filesystemUsage, error) {
		return filesystemUsage{BytesTotal: 100, BytesFree: 11, InodesTotal: 100, InodesFree: 11}, nil
	}
	result, err := store.Prune(context.Background(), FilePruneOptions{
		HighWatermarkPercent: 90, LowWatermarkPercent: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Triggered {
		t.Fatalf("unexpected eviction: %#v", result)
	}
}

func TestFileStoreRejectsTraversal(t *testing.T) {
	store, err := NewFileStore(t.TempDir(), 1024)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../escape", "/absolute", `windows\\escape`} {
		if err := store.PutWithOptions(context.Background(), key, []byte("x"), PutOptions{}); err == nil {
			t.Errorf("accepted %q", key)
		}
	}
}

func TestFileStoreReplacesAtomically(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStore(root, 1024)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.PutWithOptions(ctx, "a/b", []byte("old"), PutOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutWithOptions(ctx, "a/b", []byte("new"), PutOptions{}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "a", "b"))
	if err != nil || string(body) != "new" {
		t.Fatalf("body = %q, %v", body, err)
	}
}
