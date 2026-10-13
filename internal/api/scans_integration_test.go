package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/eve-kill/shrike/internal/testfixtures"
)

func TestScanAnalyzeSaveShareAndRefreshAgainstPostgres(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_scans_test_")
	if err := testfixtures.SeedScans(ctx, pool); err != nil {
		t.Fatal(err)
	}
	const session = "00000000-0000-4000-8000-000000000002"
	if _, err := pool.Exec(ctx, `INSERT INTO users (character_id,character_name,character_owner_hash,session_id) VALUES (42,'Fixture owner','fixture',$1)`, session); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_sessions (session_id,character_id,expires_at) VALUES ($1,42,now()+interval '1 hour')`, session); err != nil {
		t.Fatal(err)
	}
	service := New(Options{DB: &campaignCreatorTestDB{pool}, ResponseCacheBytes: 1 << 20})
	call := func(method, path string, body any, authed bool, origin string) *httptest.ResponseRecorder {
		t.Helper()
		encoded, _ := json.Marshal(body)
		request := httptest.NewRequest(method, "http://localhost"+path, bytes.NewReader(encoded))
		request.Header.Set("Content-Type", "application/json")
		if origin != "" {
			request.Header.Set("Origin", origin)
		}
		if authed {
			request.AddCookie(&http.Cookie{Name: "evelogin", Value: session})
		}
		response := httptest.NewRecorder()
		service.Site().ServeHTTP(response, request)
		return response
	}
	for _, kind := range []string{"localscan", "dscan"} {
		t.Run(kind, func(t *testing.T) {
			var input, save any
			if kind == "localscan" {
				input = strings.Split(testfixtures.LocalScanNames, "\n")
			} else {
				input = map[string]any{"dscan": testfixtures.DirectionalScan}
			}
			analysis := call("POST", "/api/tools/"+kind, input, false, "http://localhost")
			if analysis.Code != 200 {
				t.Fatalf("analyze %d %s", analysis.Code, analysis.Body.String())
			}
			var result map[string]any
			if err := json.Unmarshal(analysis.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if kind == "localscan" {
				if result["totalCharacters"] != float64(4) || result["totalDangerous"] != float64(1) {
					t.Fatal("local analysis did not deduplicate input and count threats")
				}
				save = map[string]any{"names": input, "result": result}
			} else {
				if result["totalCount"] != float64(5) || result["uniqueTypes"] != float64(4) {
					t.Fatal("directional analysis lost known/unknown types")
				}
				save = map[string]any{"dscan": testfixtures.DirectionalScan, "result": result}
			}
			if denied := call("POST", "/api/tools/"+kind+"/save", save, false, "http://localhost"); denied.Code != 401 {
				t.Fatalf("anonymous save %d", denied.Code)
			}
			if denied := call("POST", "/api/tools/"+kind+"/save", save, true, "http://untrusted.example"); denied.Code != 403 {
				t.Fatalf("cross-origin save %d", denied.Code)
			}
			saved := call("POST", "/api/tools/"+kind+"/save", save, true, "http://localhost")
			if saved.Code != 200 {
				t.Fatalf("save %d %s", saved.Code, saved.Body.String())
			}
			var hash struct {
				Hash string `json:"hash"`
			}
			_ = json.Unmarshal(saved.Body.Bytes(), &hash)
			if len(hash.Hash) != sha256.Size*2 {
				t.Fatal("invalid share hash")
			}
			if _, err := hex.DecodeString(hash.Hash); err != nil {
				t.Fatal(err)
			}
			path := "/api/tools/" + kind + "/" + hash.Hash
			shared := call("GET", path, nil, false, "")
			if shared.Code != 200 {
				t.Fatalf("shared GET %d", shared.Code)
			}
			if shared.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal("mutable scan result must bypass browser and shared caches")
			}
			reader, found, err := service.Homepage().SavedScan(ctx, kind, hash.Hash)
			if err != nil || !found {
				t.Fatal("stored shared reader missing scan")
			}
			var direct, public map[string]any
			_ = json.Unmarshal(reader, &direct)
			_ = json.Unmarshal(shared.Body.Bytes(), &public)
			delete(public, "$schema")
			delete(direct, "$schema")
			if !reflect.DeepEqual(direct, public) {
				t.Fatal("shared page and API differ")
			}
			// The content hash is based on the input, while a later save refreshes
			// analysis. Warm every API alias before verifying read-after-write.
			alias := "/api/scans/local/" + hash.Hash
			if kind == "dscan" {
				alias = "/api/scans/dscan/" + hash.Hash
			}
			_ = call("GET", alias, nil, false, "")
			result["fixtureRevision"] = 2
			updated := call("POST", "/api/tools/"+kind+"/save", save, true, "http://localhost")
			if updated.Code != 200 {
				t.Fatal("refresh save failed")
			}
			for _, route := range []string{path, alias} {
				response := call("GET", route, nil, false, "")
				var current map[string]any
				_ = json.Unmarshal(response.Body.Bytes(), &current)
				if current["fixtureRevision"] != float64(2) {
					t.Fatalf("saved refresh remained stale at %s", route)
				}
			}
		})
	}
	if _, err := pool.Exec(ctx, `UPDATE user_sessions SET expires_at=now()-interval '1 hour' WHERE session_id=$1`, session); err != nil {
		t.Fatal(err)
	}
	response := call("POST", "/api/tools/dscan/save", map[string]any{"dscan": "test", "result": map[string]any{}}, true, "http://localhost")
	if response.Code != 401 {
		t.Fatalf("expired session save %d", response.Code)
	}
}
