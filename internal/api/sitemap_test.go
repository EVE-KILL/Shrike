package api

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSitemapKindsAreExplicitlyAllowlisted(t *testing.T) {
	if sitemapCanonicalPath != "/sitemap/{kind}" ||
		sitemapCompatibilityPrefix != "/__sitemap__/" {
		t.Fatalf(
			"sitemap paths = %q, %q; unified Huma paths must be root paths",
			sitemapCanonicalPath, sitemapCompatibilityPrefix,
		)
	}

	want := []string{
		"alliances", "battles", "characters", "corporations", "items",
		"kills", "regions", "ships", "systems", "wars",
	}
	got := make([]string, 0, len(sitemapSpecs))
	for _, spec := range sitemapSpecs {
		got = append(got, spec.Kind)
		if strings.TrimSpace(spec.Query) == "" {
			t.Errorf("%s has no query", spec.Kind)
		}
		if spec.LocationPrefix == "" || spec.ChangeFrequency == "" {
			t.Errorf("%s has an incomplete response specification", spec.Kind)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("kinds = %v, want %v", got, want)
	}

	for _, kind := range want {
		spec, err := resolveSitemapSpec(kind)
		if err != nil {
			t.Fatalf("resolve %q: %v", kind, err)
		}
		if spec.Kind != kind {
			t.Errorf("resolve %q returned %q", kind, spec.Kind)
		}
	}
}

func TestResolveSitemapSpecRejectsUnknownOrNormalizedKinds(t *testing.T) {
	for _, kind := range []string{"", "killmails", "KILLS", " kills "} {
		_, err := resolveSitemapSpec(kind)
		var apiErr *legacyAPIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("%q error = %v, want API error", kind, err)
		}
		if apiErr.Status != 400 {
			t.Errorf("%q status = %d, want 400", kind, apiErr.Status)
		}
	}
}

func TestBuildSitemapEntriesPreservesShapeAndOptionalLastmod(t *testing.T) {
	spec, err := resolveSitemapSpec("kills")
	if err != nil {
		t.Fatal(err)
	}
	modified := time.Date(
		2026, time.July, 26, 4, 5, 6, 123456789, time.FixedZone("test", 7200),
	)
	got := buildSitemapEntries(spec, []map[string]any{
		{"id": int32(123), "lastmod": modified},
		{"id": int64(456), "lastmod": nil},
		{"id": "invalid", "lastmod": modified},
	})
	want := []map[string]any{
		{
			"loc": "/kill/123", "lastmod": modified,
			"changefreq": "monthly", "priority": 0.6,
		},
		{
			"loc":        "/kill/456",
			"changefreq": "monthly", "priority": 0.6,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entries = %#v, want %#v", got, want)
	}

	normalized := normalizeJSON(got).([]map[string]any)
	if got, want := normalized[0]["lastmod"], "2026-07-26T02:05:06.123Z"; got != want {
		t.Errorf("lastmod = %v, want %s", got, want)
	}
	if _, exists := normalized[1]["lastmod"]; exists {
		t.Error("nil lastmod was emitted; the frontend omits it")
	}
}

func TestBuildSitemapEntriesReturnsStableEmptyArray(t *testing.T) {
	spec, err := resolveSitemapSpec("regions")
	if err != nil {
		t.Fatal(err)
	}
	got := buildSitemapEntries(spec, nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty entries = %#v, want non-nil empty slice", got)
	}
}
