package frontend

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConfigurableLucideIconsUseSmallImmutableAssetsAndKeepSharedSprite(t *testing.T) {
	if !strings.Contains(iconURL("lucide:house"), "/icons.svg#house") {
		t.Fatal("common icon stopped using shared sprite")
	}
	href := iconURL("lucide:rainbow")
	if !strings.Contains(href, "/lucide/rainbow.svg#icon") {
		t.Fatal("configurable icon has no asset")
	}
	w := httptest.NewRecorder()
	serveAsset(w, httptest.NewRequest("GET", strings.TrimSuffix(href, "#icon"), nil))
	if w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") || !strings.Contains(w.Header().Get("Content-Type"), "image/svg+xml") || !strings.Contains(w.Body.String(), `<symbol id="icon" viewBox="0 0 24 24">`) {
		t.Fatal("individual icon transport failed")
	}
	if w.Body.Len() > 4096 {
		t.Fatal("individual icon downloaded a whole collection")
	}
	for _, path := range []string{"/assets/" + manifest.revision + "/lucide/unknown-fixture-icon.svg", "/assets/" + manifest.revision + "/lucide-icons.json"} {
		w = httptest.NewRecorder()
		serveAsset(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 404 {
			t.Fatal("unknown icon or source collection exposed")
		}
	}
}
