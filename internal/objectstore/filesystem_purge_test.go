package objectstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPurgeLegacyEntitiesPreservesStaticAndOldPortraits(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		"entities/characters/1/current.json",
		"entities/corporations/2/logo.png",
		"entities/alliances/3/logo.png",
		"oldcharacters/0/1/1_256.jpg",
		"types/42_64.png",
		"static/systems/30000142_32.png",
		"social/killmails/v1/1.png",
	}
	for _, relative := range paths {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewFileStore(root, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	dry, err := store.PurgeLegacyEntities(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Applied || dry.Files != 3 || dry.Bytes != 12 {
		t.Fatalf("dry run = %+v", dry)
	}
	for _, relative := range paths[:3] {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			t.Fatalf("dry run removed %s: %v", relative, err)
		}
	}
	applied, err := store.PurgeLegacyEntities(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || applied.Files != 3 || applied.Bytes != 12 {
		t.Fatalf("applied = %+v", applied)
	}
	for _, relative := range paths[:3] {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Errorf("obsolete image %s remains: %v", relative, err)
		}
	}
	for _, relative := range paths[3:] {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			t.Errorf("retained image %s is gone: %v", relative, err)
		}
	}
}
