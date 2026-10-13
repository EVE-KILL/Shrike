package frontend

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFitEditorRouteAndDogmaAssets(t *testing.T) {
	handler := New(Options{})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/fits/create?fit=v3%3Aexample", nil))
	if w.Code != 200 {
		t.Fatalf("editor status = %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"data-fit-editor", `data-controller="fit-editor"`, "click->fit-editor#bootAction", "pagehide@window->fit-editor#hide", "pageshow@window->fit-editor#show", "fit-editor.js", "fit-editor.css", "fit-wheel.css", "data-fit-save-dialog"} {
		if !strings.Contains(body, want) {
			t.Errorf("editor missing %q", want)
		}
	}
	if !strings.Contains(w.Header().Get("Content-Security-Policy"), "'wasm-unsafe-eval'") {
		t.Fatal("editor CSP blocks the dogma engine")
	}
	for _, name := range []string{"dogma.js", "fit-wheel.js", "fit-stats.js", "fit-resources.js", "fit-variants.js", "fit-ui/fitting-hull.webp", "dogma/sde.dat", "dogma/esf_dogma_engine_bg.wasm"} {
		asset := httptest.NewRecorder()
		handler.ServeHTTP(asset, httptest.NewRequest("GET", assetURL(name), nil))
		if asset.Code != 200 || asset.Body.Len() == 0 {
			t.Errorf("embedded %s: status %d, length %d", name, asset.Code, asset.Body.Len())
		}
	}
	home := httptest.NewRecorder()
	handler.ServeHTTP(home, httptest.NewRequest("GET", "/", nil))
	if strings.Contains(home.Header().Get("Content-Security-Policy"), "'wasm-unsafe-eval'") {
		t.Fatal("WASM permission leaked to unrelated pages")
	}
}

func TestFitVariantsFragmentUsesObservedFamilies(t *testing.T) {
	handler := New(Options{Universe: testUniverseReader{}})
	for _, tc := range []struct {
		path   string
		status int
		marker string
	}{
		{"/_fragments/fits/variants/11186", 200, `data-fit-variant="abc"`},
		{"/_fragments/fits/variants/34", 404, "404"},
		{"/_fragments/fits/variants/bad", 404, "404"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.marker) {
			t.Errorf("%s: status %d, body missing %q", tc.path, w.Code, tc.marker)
		}
		if tc.status == 200 {
			for _, marker := range []string{`id="fit-variants-data"`, `href="/advancedsearch?`, `data-fit-variant-index="1"`} {
				if !strings.Contains(w.Body.String(), marker) {
					t.Errorf("%s: missing %q", tc.path, marker)
				}
			}
		}
	}
}
