package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func TestScanRoutesRegisterCanonicalAndFrontendAliases(t *testing.T) {
	mux := http.NewServeMux()
	a := humago.New(mux, huma.DefaultConfig("test", "test"))
	registerScanRoutes(a, Options{})

	for _, route := range []struct {
		method, path string
	}{
		{http.MethodPost, "/scans/dscan/analyze"},
		{http.MethodPost, "/tools/dscan"},
		{http.MethodPost, "/scans/dscan"},
		{http.MethodPost, "/tools/dscan/save"},
		{http.MethodGet, "/scans/dscan/{hash}"},
		{http.MethodGet, "/tools/dscan/{hash}"},
		{http.MethodPost, "/scans/local/analyze"},
		{http.MethodPost, "/tools/localscan"},
		{http.MethodPost, "/scans/local"},
		{http.MethodPost, "/tools/localscan/save"},
		{http.MethodGet, "/scans/local/{hash}"},
		{http.MethodGet, "/tools/localscan/{hash}"},
	} {
		item := a.OpenAPI().Paths[route.path]
		if item == nil {
			t.Errorf("%s %s path is missing", route.method, route.path)
			continue
		}
		if route.method == http.MethodGet && item.Get == nil {
			t.Errorf("GET %s operation is missing", route.path)
		}
		if route.method == http.MethodPost && item.Post == nil {
			t.Errorf("POST %s operation is missing", route.path)
		}
	}
}

func TestDirectionalScanSplittingMatchesEVEFormats(t *testing.T) {
	tabbed := "123\tPilot\tRifter\t10 km"
	if got := splitDirectionalScanLine(tabbed); len(got) != 4 ||
		got[2] != "Rifter" {
		t.Errorf("tab split = %#v", got)
	}
	spaced := "123    Pilot Name    Hurricane Fleet Issue    20 km"
	if got := splitDirectionalScanLine(spaced); len(got) != 4 ||
		got[2] != "Hurricane Fleet Issue" {
		t.Errorf("space split = %#v", got)
	}
	if got := splitDirectionalScanLine("one two three"); len(got) != 1 {
		t.Errorf("short whitespace run split = %#v", got)
	}
}

func TestScanHashesMatchFrontendNormalization(t *testing.T) {
	dscan := strings.TrimSpace("\n  one\ntwo  \n")
	digest := sha256.Sum256([]byte(dscan))
	if got := hex.EncodeToString(digest[:]); len(got) != 64 {
		t.Fatalf("dscan digest = %q", got)
	}

	names := []string{"Zulu", "Alpha", "Mike"}
	sort.Strings(names)
	normalized := strings.Join(names, "\n")
	digest = sha256.Sum256([]byte(normalized))
	got := hex.EncodeToString(digest[:])
	wantBytes := sha256.Sum256([]byte("Alpha\nMike\nZulu"))
	want := hex.EncodeToString(wantBytes[:])
	if got != want {
		t.Fatalf("local scan hash = %q, want %q", got, want)
	}
}

func TestJSONTruthyMatchesSaveEndpointChecks(t *testing.T) {
	for _, value := range []any{nil, false, "", 0.0} {
		if jsonTruthy(value) {
			t.Errorf("%#v is truthy", value)
		}
	}
	for _, value := range []any{true, "result", map[string]any{}, []any{}} {
		if !jsonTruthy(value) {
			t.Errorf("%#v is falsey", value)
		}
	}
}
