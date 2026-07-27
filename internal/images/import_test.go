package images

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/objectstore"
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

	// Simulate interruption after every shard marker was published but before
	// the final manifest made the whole import a one-request no-op.
	delete(store.objects, oldCharacterManifestKey)
	store.puts = nil
	resumed, err := ImportOldCharacters(
		context.Background(),
		store,
		archivePath,
		nil,
		ImportOptions{Concurrency: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Scanned != 2 || resumed.Uploaded != 0 || resumed.Skipped != 2 {
		t.Fatalf("resumed result = %+v", resumed)
	}
	if len(store.puts) != 1 || store.puts[0] != oldCharacterManifestKey {
		t.Fatalf("resumed uploads = %v", store.puts)
	}

	store.puts = nil
	complete, err := ImportOldCharacters(
		context.Background(),
		store,
		archivePath,
		nil,
		ImportOptions{Concurrency: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	if complete.Uploaded != 0 || complete.Skipped != 2 {
		t.Fatalf("complete result = %+v", complete)
	}
	if len(store.puts) != 0 {
		t.Fatalf("complete import unexpectedly wrote %v", store.puts)
	}
}

func TestImportOldCharactersResumesRemoteArchiveDownload(t *testing.T) {
	archiveBody := makeZip(t, map[string][]byte{
		"1/2/900000012_256.jpg": []byte("portrait"),
	})
	sum := sha1.Sum(archiveBody)
	digest := hex.EncodeToString(sum[:])
	partialLength := len(archiveBody) / 3
	var gets atomic.Int64
	var ranges atomic.Int64
	lastModified := "Fri, 17 Jan 2014 14:39:36 GMT"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Length", strconv.Itoa(len(archiveBody)))
		w.Header().Set("ETag", `"archive-v1"`)
		w.Header().Set("Last-Modified", lastModified)
		// EVE Ref exposes the Backblaze large-file checksum with underscores.
		w.Header().Set("X-Amz-Meta-Large_File_Sha1", digest)
		if r.Method == http.MethodHead {
			return
		}
		gets.Add(1)
		wantRange := "bytes=" + strconv.Itoa(partialLength) + "-"
		if r.Header.Get("Range") != wantRange {
			t.Errorf("Range = %q, want %q", r.Header.Get("Range"), wantRange)
			http.Error(w, "bad range", http.StatusBadRequest)
			return
		}
		ranges.Add(1)
		w.Header().Set(
			"Content-Range",
			"bytes "+strconv.Itoa(partialLength)+"-"+
				strconv.Itoa(len(archiveBody)-1)+"/"+
				strconv.Itoa(len(archiveBody)),
		)
		w.Header().Set(
			"Content-Length",
			strconv.Itoa(len(archiveBody)-partialLength),
		)
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(archiveBody[partialLength:])
	}))
	defer server.Close()

	cacheDirectory := t.TempDir()
	source := server.URL + "/OldCharPortraits_256.zip"
	partialPath := filepath.Join(
		cacheDirectory,
		"OldCharPortraits_256.zip.part",
	)
	writeTestFile(t, partialPath, archiveBody[:partialLength])
	if err := writeArchiveCache(
		partialPath+".json",
		oldCharacterArchiveCache{
			Source: source, Digest: "sha1:" + digest,
			Size: int64(len(archiveBody)), ETag: `"archive-v1"`,
			LastModified: lastModified,
		},
	); err != nil {
		t.Fatal(err)
	}

	store := newMemoryStore()
	result, err := ImportOldCharacters(
		context.Background(),
		store,
		source,
		server.Client(),
		ImportOptions{
			Concurrency: 2, CacheDirectory: cacheDirectory,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Uploaded != 1 || gets.Load() != 1 || ranges.Load() != 1 {
		t.Fatalf(
			"result = %+v, GETs = %d, ranges = %d",
			result,
			gets.Load(),
			ranges.Load(),
		)
	}
	var manifest oldCharacterManifest
	if err := json.Unmarshal(
		store.objects[oldCharacterManifestKey].Body,
		&manifest,
	); err != nil {
		t.Fatal(err)
	}
	if manifest.ArchiveDigest != "sha1:"+digest {
		t.Fatalf("archive digest = %q", manifest.ArchiveDigest)
	}
	finalPath := filepath.Join(cacheDirectory, "OldCharPortraits_256.zip")
	finalBody, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(finalBody, archiveBody) {
		t.Fatal("resumed archive does not match its source")
	}

	// A matching final B2 manifest is checked from the remote SHA-1 advertised
	// by HEAD, before the local archive is downloaded or opened.
	if err := os.Remove(finalPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(finalPath + ".json"); err != nil {
		t.Fatal(err)
	}
	second, err := ImportOldCharacters(
		context.Background(),
		store,
		source,
		server.Client(),
		ImportOptions{
			Concurrency: 2, CacheDirectory: cacheDirectory,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if second.Uploaded != 0 || second.Skipped != 1 || gets.Load() != 1 {
		t.Fatalf("second = %+v, GETs = %d", second, gets.Load())
	}
}

func TestPutOldCharacterObjectRetriesLongStorageIncident(t *testing.T) {
	store := &flakyOldCharacterStore{memoryStore: newMemoryStore()}
	store.failures.Store(2)
	var delays []time.Duration
	err := putOldCharacterObject(
		context.Background(),
		store,
		"oldcharacters/0/1/900000001_256.jpg",
		[]byte("portrait"),
		objectstore.PutOptions{ContentType: "image/jpeg"},
		func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if store.calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", store.calls.Load())
	}
	if len(delays) != 2 ||
		delays[0] != 2*time.Second ||
		delays[1] != 4*time.Second {
		t.Fatalf("delays = %v", delays)
	}
	if store.objects["oldcharacters/0/1/900000001_256.jpg"] == nil {
		t.Fatal("portrait was not eventually uploaded")
	}
}

type flakyOldCharacterStore struct {
	*memoryStore
	failures atomic.Int64
	calls    atomic.Int64
}

func (s *flakyOldCharacterStore) PutWithOptions(
	ctx context.Context,
	key string,
	body []byte,
	options objectstore.PutOptions,
) error {
	s.calls.Add(1)
	if s.failures.Add(-1) >= 0 {
		return errors.New("sustained B2 incident")
	}
	return s.memoryStore.PutWithOptions(ctx, key, body, options)
}

func TestSyncTypeExportVerifiesDigestAndUsesDirectTypeIDKeys(t *testing.T) {
	icon := solidPNG(t, 8, 8, colorValue(20, 30, 40))
	archive := makeZip(t, map[string][]byte{
		"Image Export Collection/42_64.png":     icon,
		"Image Export Collection/42_bpc_64.png": icon,
		"README.txt":                            []byte("ignored"),
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
					"name":                 TurtleTypeExportAsset,
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
	options := TypeExportSyncOptions{
		HTTPClient: server.Client(), APIURL: server.URL + "/latest",
		Now: func() time.Time {
			return time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
		},
	}
	first, err := SyncTypeExport(context.Background(), store, options)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || first.Digest != digest ||
		store.objects["types/42_64.png"] == nil ||
		store.objects["types/42_bpc_64.png"] == nil {
		t.Fatalf("first sync = %+v, objects %v", first, store.objects)
	}
	manifestObject := store.objects[typeExportManifestKey]
	if manifestObject == nil {
		t.Fatal("current export manifest was not published")
	}
	var manifest typeExportManifest
	if err := json.Unmarshal(manifestObject.Body, &manifest); err != nil {
		t.Fatal(err)
	}
	iconSum := sha256.Sum256(icon)
	iconDigest := hex.EncodeToString(iconSum[:])
	if manifest.Version != typeExportManifestVersion ||
		manifest.ObjectPolicy != typeExportObjectPolicy ||
		manifest.Release != "icons-42" ||
		manifest.ArchiveDigest != digest ||
		manifest.Images["42_64.png"] != iconDigest ||
		manifest.Images["42_bpc_64.png"] != iconDigest {
		t.Fatalf("manifest = %+v", manifest)
	}

	second, err := SyncTypeExport(context.Background(), store, options)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed || downloads.Load() != 1 {
		t.Fatalf("second sync = %+v, downloads %d", second, downloads.Load())
	}
}

func TestSyncTypeExportRebuildsOlderManifestForNewFilenameRules(t *testing.T) {
	icon := solidPNG(t, 8, 8, colorValue(20, 30, 40))
	archive := makeZip(t, map[string][]byte{
		"42_64.png":     icon,
		"42_bpc_64.png": icon,
	})
	archiveSum := sha256.Sum256(archive)
	archiveDigest := hex.EncodeToString(archiveSum[:])
	iconSum := sha256.Sum256(icon)
	iconDigest := hex.EncodeToString(iconSum[:])

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.URL.Path == "/latest" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name": "icons-42",
				"assets": []map[string]any{{
					"name":                 TurtleTypeExportAsset,
					"browser_download_url": "http://" + r.Host + "/bundle",
					"digest":               "sha256:" + archiveDigest,
				}},
			})
			return
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	store := newMemoryStore()
	oldManifest, _ := json.Marshal(typeExportManifest{
		Version: 1, Release: "icons-42", ArchiveDigest: archiveDigest,
		Images: map[string]string{"42_64.png": iconDigest},
	})
	store.objects[typeExportManifestKey] = &objectstore.Object{Body: oldManifest}
	store.objects["types/42_64.png"] = &objectstore.Object{Body: icon}

	result, err := SyncTypeExport(
		context.Background(),
		store,
		TypeExportSyncOptions{
			HTTPClient: server.Client(), APIURL: server.URL + "/latest",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Import.Uploaded != 2 || result.Import.Skipped != 0 ||
		store.objects["types/42_bpc_64.png"] == nil {
		t.Fatalf("result = %+v, objects = %v", result, store.objects)
	}
	var manifest typeExportManifest
	if err := json.Unmarshal(
		store.objects[typeExportManifestKey].Body,
		&manifest,
	); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != typeExportManifestVersion ||
		manifest.ObjectPolicy != typeExportObjectPolicy ||
		len(manifest.Images) != 2 {
		t.Fatalf("manifest = %+v", manifest)
	}
}

func TestSyncTypeExportRejectsDigestMismatchBeforePublishing(t *testing.T) {
	archive := makeZip(t, map[string][]byte{
		"42_64.png": []byte("not-an-image"),
	})
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.URL.Path == "/latest" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name": "icons-bad",
				"assets": []map[string]any{{
					"name":                 TurtleTypeExportAsset,
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
	if _, err := SyncTypeExport(
		context.Background(),
		store,
		TypeExportSyncOptions{
			HTTPClient: server.Client(), APIURL: server.URL + "/latest",
		},
	); err == nil {
		t.Fatal("digest mismatch was accepted")
	}
	if store.objects[typeExportManifestKey] != nil {
		t.Fatal("failed image export was published")
	}
}

func TestImportTypeExportImagesUsesManifestHashes(t *testing.T) {
	unchanged := solidPNG(t, 8, 8, colorValue(10, 20, 30))
	changed := solidPNG(t, 8, 8, colorValue(40, 50, 60))
	added := solidPNG(t, 8, 8, colorValue(70, 80, 90))
	archiveBody := makeZip(t, map[string][]byte{
		"41_64.png": unchanged,
		"42_64.png": changed,
		"43_64.png": added,
	})
	archivePath := filepath.Join(t.TempDir(), "types.zip")
	writeTestFile(t, archivePath, archiveBody)
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()

	unchangedSum := sha256.Sum256(unchanged)
	store := newMemoryStore()
	result, hashes, err := importTypeExportImages(
		context.Background(),
		store,
		archive.File,
		map[string]string{
			"41_64.png": hex.EncodeToString(unchangedSum[:]),
			"42_64.png": strings.Repeat("0", 64),
		},
		false,
		0,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 3 || result.Uploaded != 2 || result.Skipped != 1 {
		t.Fatalf("result = %+v", result)
	}
	if len(hashes) != 3 {
		t.Fatalf("hashes = %v", hashes)
	}
	if store.objects["types/41_64.png"] != nil ||
		store.objects["types/42_64.png"] == nil ||
		store.objects["types/43_64.png"] == nil {
		t.Fatalf("uploaded objects = %v", store.objects)
	}
	if store.objects["types/42_64.png"].CacheControl != responseCacheControl {
		t.Fatalf(
			"cache control = %q",
			store.objects["types/42_64.png"].CacheControl,
		)
	}
}

func TestImportTypeExportImagesResumesWithoutManifest(t *testing.T) {
	icon := solidPNG(t, 8, 8, colorValue(10, 20, 30))
	sum := sha256.Sum256(icon)
	digest := hex.EncodeToString(sum[:])
	archiveBody := makeZip(t, map[string][]byte{"41_64.png": icon})
	archivePath := filepath.Join(t.TempDir(), "types.zip")
	writeTestFile(t, archivePath, archiveBody)
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()

	store := newMemoryStore()
	store.objects["types/41_64.png"] = &objectstore.Object{
		Body: icon,
		ObjectInfo: objectstore.ObjectInfo{
			Key: "types/41_64.png",
			Metadata: map[string]string{
				"sha256": digest,
			},
		},
	}
	result, hashes, err := importTypeExportImages(
		context.Background(),
		store,
		archive.File,
		nil,
		false,
		4,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Uploaded != 0 || result.Skipped != 1 ||
		hashes["41_64.png"] != digest {
		t.Fatalf("result = %+v, hashes = %v", result, hashes)
	}
	if len(store.puts) != 0 {
		t.Fatalf("unexpected uploads = %v", store.puts)
	}
}

func TestUseTypeExportArchiveVerifiesDigestWithoutRemovingFile(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "types.zip")
	body := []byte("archive")
	writeTestFile(t, archivePath, body)
	sum := sha256.Sum256(body)
	digest := hex.EncodeToString(sum[:])

	name, got, cleanup, err := useTypeExportArchive(
		archivePath,
		"sha256:"+digest,
	)
	if err != nil {
		t.Fatal(err)
	}
	cleanup()
	if name != archivePath || got != digest {
		t.Fatalf("archive = %q, digest = %q", name, got)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("local archive was removed: %v", err)
	}
	if _, _, _, err := useTypeExportArchive(
		archivePath,
		"sha256:"+strings.Repeat("0", 64),
	); err == nil {
		t.Fatal("digest mismatch was accepted")
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
