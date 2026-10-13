package frontend

import (
	"regexp"
	"strings"
	"testing"
)

func TestAssetVersioningPreservesAPIAndExternalURLs(t *testing.T) {
	source := "fetch(`/api/user/domains/${id}/assets/${asset}`);const sprite='/assets/icons.svg#user';const external='https://example.invalid/assets/image.png';const css=\"url(/assets/fonts/font.woff2)\";"
	result := string(versionAssetReferences([]byte(source), "fixture-revision"))
	for _, want := range []string{"/api/user/domains/${id}/assets/${asset}", "https://example.invalid/assets/image.png", "/assets/fixture-revision/icons.svg#user", "url(/assets/fixture-revision/fonts/font.woff2)"} {
		if !strings.Contains(result, want) {
			t.Errorf("versioned content lost %q", want)
		}
	}
	// Exercise the actual embedded controller, so a future cache rewrite cannot
	// silently redirect image removal to a nonexistent API endpoint again.
	var modules strings.Builder
	for name, asset := range manifest.files {
		if strings.HasSuffix(name, ".js") {
			modules.Write(asset.data)
		}
	}
	controller := modules.String()
	if !regexp.MustCompile(`/api/user/domains/\$\{[^}]+\}/assets/\$\{[^}]+\}`).MatchString(controller) {
		t.Fatal("domain image removal API was rewritten as a static asset")
	}
	if !strings.Contains(controller, "/assets/"+manifest.revision+"/icons.svg") {
		t.Fatal("domain editor sprite is missing its immutable asset revision")
	}
}
