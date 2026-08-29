package objectstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
