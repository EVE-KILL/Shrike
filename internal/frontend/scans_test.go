package frontend

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testScanHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

type scanReaderStub struct{ homepageReaderStub }

func (scanReaderStub) SavedScan(_ context.Context, kind, hash string) (json.RawMessage, bool, error) {
	if hash != testScanHash {
		return nil, false, nil
	}
	if kind == "dscan" {
		return json.RawMessage(`{"grouped":{"Ship":{"categoryId":6,"groups":{"Frigate":{"groupId":25,"types":[{"typeName":"Astero","typeId":33468,"count":2}]}}}},"totalCount":2,"uniqueTypes":1}`), true, nil
	}
	return json.RawMessage(`{"alliances":{},"corporations":{"98598862":{"name":"Example Corp","ticker":"EX","characters":[{"characterId":2121270074,"name":"Pilot","kills":7}]}},"unresolved":[],"totalCharacters":1,"totalDangerous":1}`), true, nil
}

func TestScanPagesAndSharedResults(t *testing.T) {
	handler := New(Options{Homepage: scanReaderStub{}})
	for _, kind := range []string{"dscan", "localscan"} {
		page := httptest.NewRecorder()
		handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/tools/"+kind, nil))
		if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Paste from Clipboard") || !strings.Contains(page.Body.String(), "scans.js") {
			t.Fatalf("%s input page = %d", kind, page.Code)
		}
		saved := httptest.NewRecorder()
		handler.ServeHTTP(saved, httptest.NewRequest(http.MethodGet, "/tools/"+kind+"/"+testScanHash, nil))
		if saved.Code != http.StatusOK || !strings.Contains(saved.Body.String(), "New Scan") {
			t.Fatalf("%s saved page = %d: %s", kind, saved.Code, saved.Body.String())
		}
		if saved.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("a repeated save must not leave a shared page cached")
		}
	}
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/tools/dscan/not-a-hash", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("invalid scan hash = %d", missing.Code)
	}

	fragment := httptest.NewRecorder()
	input := `{"grouped":{"Ship":{"groups":{"Frigate":{"types":[{"typeName":"<script>alert(1)</script>","typeId":null,"count":1}]}}}},"totalCount":1,"uniqueTypes":1}`
	handler.ServeHTTP(fragment, httptest.NewRequest(http.MethodPost, "/_fragments/scans/dscan", bytes.NewBufferString(input)))
	if fragment.Code != http.StatusOK || !strings.Contains(fragment.Body.String(), "&lt;script&gt;") || strings.Contains(fragment.Body.String(), "<script>alert(1)</script>") || strings.Contains(fragment.Body.String(), "<!DOCTYPE html>") {
		t.Fatalf("unsafe or invalid scan fragment = %d: %s", fragment.Code, fragment.Body.String())
	}
}

func TestLocalScanPreservesSourceOrderingForTies(t *testing.T) {
	people := []localCharacter{{CharacterID: 2, Name: "Zulu", Kills: 7}, {CharacterID: 1, Name: "Alpha", Kills: 7}, {CharacterID: 3, Name: "Top", Kills: 10}}
	result := localScanResult{Corporations: map[string]localCorporation{
		"20": {Name: "Alpha", Characters: people},
		"10": {Name: "Zulu", Characters: people},
	}}
	view := makeLocalScanView(result)
	if view.Corporations[0].ID != 10 || view.Corporations[1].ID != 20 {
		t.Fatal("equal-size corporations must retain JavaScript numeric-key order")
	}
	for _, corporation := range view.Corporations {
		if corporation.People[0].Name != "Top" || corporation.People[1].Name != "Zulu" || corporation.People[2].Name != "Alpha" {
			t.Fatal("equal-kill pilots must retain their incoming order")
		}
	}
	if people[0].Name != "Zulu" {
		t.Fatal("rendering modified the source analysis")
	}
	if view.Threats[0].Corporation != "Zulu" || view.Threats[2].Name != "Zulu" {
		t.Fatal("threat ties must preserve the source traversal order")
	}
}
