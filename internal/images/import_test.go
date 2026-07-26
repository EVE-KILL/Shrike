package images

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestImportStaticTreeMapsAssetsAndSkipsMatchingHashes(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "regions", "10000002.png"), []byte("region"))
	writeTestFile(t, filepath.Join(root, "overlays", "ids.json"), []byte(`{"t2":[42]}`))

	store := newMemoryStore()
	first, err := ImportStaticTree(
		context.Background(),
		store,
		root,
		ImportOptions{Concurrency: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.Uploaded != 2 || first.Skipped != 0 {
		t.Fatalf("first import = %+v", first)
	}
	for _, key := range []string{
		"static/regions/10000002.png",
		"types/overlays/ids.json",
	} {
		if store.objects[key] == nil {
			t.Errorf("missing object %q", key)
		}
	}

	second, err := ImportStaticTree(
		context.Background(),
		store,
		root,
		ImportOptions{Concurrency: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	if second.Uploaded != 0 || second.Skipped != 2 {
		t.Fatalf("second import = %+v", second)
	}
}

func TestImportOldCharactersPreservesShardContract(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "old.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	addZipEntry(t, writer, "7/5/999802075_256.jpg", []byte("portrait"))
	addZipEntry(t, writer, "missing_256.jpg", []byte("missing"))
	addZipEntry(t, writer, "README.txt", []byte("ignored"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	store := newMemoryStore()
	result, err := ImportOldCharacters(
		context.Background(),
		store,
		archivePath,
		nil,
		ImportOptions{Concurrency: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 2 || result.Uploaded != 2 {
		t.Fatalf("result = %+v", result)
	}
	for _, key := range []string{
		"oldcharacters/7/5/999802075_256.jpg",
		"oldcharacters/missing_256.jpg",
	} {
		if store.objects[key] == nil {
			t.Errorf("missing object %q", key)
		}
	}
}

func TestSyncTypeBundleVerifiesDigestAndPublishesPointerLast(t *testing.T) {
	icon := solidPNG(t, 8, 8, colorValue(20, 30, 40))
	metadata := []byte(`{"42":{"icon":"icon-42.png"}}`)
	archive := makeZip(t, map[string][]byte{
		"service_metadata.json": metadata,
		"icon-42.png":           icon,
	})
	sum := sha256.Sum256(archive)
	digest := hex.EncodeToString(sum[:])
	var downloads atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		switch r.URL.Path {
		case "/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name": "icons-42",
				"assets": []map[string]any{{
					"name":                 TurtleBundleAsset,
					"browser_download_url": "http://" + r.Host + "/bundle",
					"digest":               "sha256:" + digest,
				}},
			})
		case "/bundle":
			downloads.Add(1)
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := newMemoryStore()
	options := BundleSyncOptions{
		HTTPClient: server.Client(), APIURL: server.URL + "/latest",
		Now: func() time.Time {
			return time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
		},
	}
	first, err := SyncTypeBundle(context.Background(), store, options)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || first.Digest != digest ||
		store.objects["types/blobs/icon-42.png"] == nil {
		t.Fatalf("first sync = %+v, objects %v", first, store.objects)
	}
	pointerObject := store.objects["types/current.json"]
	if pointerObject == nil {
		t.Fatal("current pointer was not published")
	}
	var pointer typeBundlePointer
	if err := json.Unmarshal(pointerObject.Body, &pointer); err != nil {
		t.Fatal(err)
	}
	if pointer.Release != "icons-42" || pointer.Digest != digest ||
		store.objects[pointer.MetadataKey] == nil {
		t.Fatalf("pointer = %+v", pointer)
	}

	second, err := SyncTypeBundle(context.Background(), store, options)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed || downloads.Load() != 1 {
		t.Fatalf("second sync = %+v, downloads %d", second, downloads.Load())
	}
}

func TestSyncTypeBundleRejectsDigestMismatchBeforePublishing(t *testing.T) {
	archive := makeZip(t, map[string][]byte{
		"service_metadata.json": []byte(`{"42":{"icon":"icon.png"}}`),
		"icon.png":              []byte("not-an-image"),
	})
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.URL.Path == "/latest" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name": "icons-bad",
				"assets": []map[string]any{{
					"name":                 TurtleBundleAsset,
					"browser_download_url": "http://" + r.Host + "/bundle",
					"digest":               "sha256:" + strings.Repeat("0", 64),
				}},
			})
			return
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	store := newMemoryStore()
	if _, err := SyncTypeBundle(
		context.Background(),
		store,
		BundleSyncOptions{
			HTTPClient: server.Client(), APIURL: server.URL + "/latest",
		},
	); err == nil {
		t.Fatal("digest mismatch was accepted")
	}
	if store.objects["types/current.json"] != nil {
		t.Fatal("failed bundle was published")
	}
}

func makeZip(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	for name, content := range entries {
		addZipEntry(t, writer, name, content)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func addZipEntry(t *testing.T, writer *zip.Writer, name string, body []byte) {
	t.Helper()
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(body); err != nil {
		t.Fatal(err)
	}
}

func writeTestFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}
