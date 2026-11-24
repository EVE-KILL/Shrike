package sde

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestOpenStagesOneReleaseAndReusesTableVersions(t *testing.T) {
	const old = "20261119T110708Z"
	const latest = "20261120T165826Z"
	var requested []string
	var mu sync.Mutex
	mux := http.NewServeMux()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requested = append(requested, r.URL.Path)
		mu.Unlock()
		if r.UserAgent() != "Shrike SDE test" {
			t.Errorf("User-Agent = %q", r.UserAgent())
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	makeManifest := func(release, groupVersion string) *Manifest {
		return &Manifest{
			Release: release, BuildNumber: 3720952,
			Tables: map[string]TableFiles{
				"types":  testTableFiles(server.URL+"/"+release+"/types", "type-version"),
				"groups": testTableFiles(server.URL+"/"+release+"/groups", groupVersion),
			},
		}
	}
	next := makeManifest(latest, "updated-groups")
	mux.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/"+latest+"/", http.StatusFound)
	})
	mux.HandleFunc("/"+latest+"/", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(struct {
			*Manifest
			ReleaseDate string `json:"releaseDate"`
			Padding     string `json:"padding"`
		}{next, "2026-11-20T16:58:26Z", strings.Repeat("x", 90_000)})
	})
	mux.HandleFunc("/"+old+"/types.schema.json", schemaHandler("typeID"))
	mux.HandleFunc("/"+old+"/types.jsonl", gzipHandler("{\"typeID\":0}\n{\"typeID\":587}\n"))
	mux.HandleFunc("/"+old+"/groups.schema.json", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGone)
	})
	mux.HandleFunc("/"+latest+"/groups.schema.json", schemaHandler("groupID"))
	mux.HandleFunc("/"+latest+"/groups.jsonl", gzipHandler("{\"groupID\":25}\n"))

	members := []string{"types", "groups"}
	previous := makeManifest(old, "previous-groups")
	cache := t.TempDir()
	src, err := open(t.Context(), server.Client(), server.URL+"/latest", cache, previous, members, "Shrike SDE test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if src.Manifest.Release != latest || src.Manifest.BuildNumber != previous.BuildNumber {
		t.Fatalf("staged unexpected release %s build %d", src.Manifest.Release, src.Manifest.BuildNumber)
	}
	for member, want := range map[string][]int64{"types": {587}, "groups": {25}} {
		var ids []int64
		err := src.Stream(t.Context(), member, func(row Row) error {
			if id, ok := row.Key(); ok {
				ids = append(ids, id)
			}
			return nil
		})
		if err != nil || !slices.Equal(ids, want) {
			t.Fatalf("%s: IDs %v, want %v, err %v", member, ids, want, err)
		}
	}
	if _, err := stage(t.Context(), server.Client(), cache, src.Manifest, members, "Shrike SDE test", nil); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	want := []string{
		"/" + old + "/types.schema.json", "/" + old + "/types.jsonl", "/" + old + "/groups.schema.json",
		"/latest", "/" + latest + "/", "/" + latest + "/groups.schema.json", "/" + latest + "/groups.jsonl",
	}
	if !slices.Equal(requested, want) {
		t.Fatalf("requests %v, want %v", requested, want)
	}

	loaded := snapshotFor(previous, members)
	current := snapshotFor(src.Manifest, members)
	if loaded.hasTables(current.TableVersions) {
		t.Fatal("a relevant table changed within the same build")
	}
	current.TableVersions["groups"] = loaded.TableVersions["groups"]
	if !loaded.hasTables(current.TableVersions) {
		t.Fatal("the same table versions should match across releases")
	}
	if (Snapshot{BuildNumber: loaded.BuildNumber}).hasTables(loaded.TableVersions) {
		t.Fatal("ZIP-era metadata should trigger a full import")
	}
}

func TestStageDoesNotCacheIncompleteDownloads(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/types.schema.json", schemaHandler("typeID"))
	mux.HandleFunc("/types.jsonl", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		body := gzip.NewWriter(w)
		fmt.Fprintln(body, `{"typeID":587}`)
		body.Flush() // Lose the footer, as on an interrupted download.
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	m := &Manifest{Tables: map[string]TableFiles{"types": testTableFiles(server.URL+"/types", "1")}}
	cache := t.TempDir()
	if _, err := stage(t.Context(), server.Client(), cache, m, []string{"types"}, "", nil); err == nil {
		t.Fatal("staged an incomplete table")
	}
	entries, err := filepath.Glob(filepath.Join(cache, "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0], ".schema.json") {
		t.Fatalf("cache contains %v; want only the schema", entries)
	}
}

func testTableFiles(url, version string) TableFiles {
	table := TableFiles{Version: version}
	table.JSONL.URL = url + ".jsonl"
	table.JSONL.SchemaURL = url + ".schema.json"
	return table
}

func schemaHandler(key string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(recordSchema{Key: key, Required: []string{key}})
	}
}

func gzipHandler(jsonl string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		body := gzip.NewWriter(w)
		fmt.Fprint(body, jsonl)
		body.Close()
	}
}
