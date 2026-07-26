package api

import (
	"reflect"
	"strings"
	"testing"
)

func TestEntityPageRouteInventoryIsConsolidatedAndRootRelative(t *testing.T) {
	want := map[string]struct {
		canonical string
		aliases   []string
		types     []string
	}{
		"detail": {
			"/entities/{type}/{id}",
			[]string{
				"/character/{id}", "/corporation/{id}",
				"/alliance/{id}", "/faction/{id}",
			},
			[]string{"alliance", "character", "corporation", "faction"},
		},
		"stats": {
			"/entities/{type}/{id}/stats",
			[]string{
				"/character/{id}/stats", "/corporation/{id}/stats",
				"/alliance/{id}/stats",
			},
			[]string{"alliance", "character", "corporation"},
		},
		"intel": {
			"/entities/{type}/{id}/intel",
			[]string{
				"/character/{id}/intel", "/corporation/{id}/intel",
				"/alliance/{id}/intel",
			},
			[]string{"alliance", "character", "corporation"},
		},
		"achievements": {
			"/entities/{type}/{id}/achievements",
			[]string{"/character/{id}/achievements"},
			[]string{"character"},
		},
		"members": {
			"/entities/{type}/{id}/members",
			[]string{"/corporation/{id}/members", "/alliance/{id}/members"},
			[]string{"alliance", "corporation"},
		},
		"corporations": {
			"/entities/{type}/{id}/corporations",
			[]string{"/alliance/{id}/corporations"},
			[]string{"alliance"},
		},
		"killlist": {
			"/entities/{type}/{id}/killlist",
			[]string{"/entity/{type}/{id}/killlist"},
			[]string{"alliance", "character", "corporation", "faction"},
		},
		"most-valuable": {
			"/entities/{type}/{id}/most-valuable",
			[]string{"/entity/{type}/{id}/most-valuable"},
			[]string{"alliance", "character", "corporation"},
		},
		"ship-classes": {
			"/entities/{type}/{id}/ship-classes",
			[]string{"/entity/{type}/{id}/ship-classes"},
			[]string{"alliance", "character", "corporation"},
		},
		"top-lists": {
			"/entities/{type}/{id}/top-lists",
			[]string{"/entity/{type}/{id}/top-lists"},
			[]string{"alliance", "character", "corporation"},
		},
	}
	if len(entityPageRoutes) != len(want) {
		t.Fatalf("entity page routes = %d, want %d", len(entityPageRoutes), len(want))
	}
	for _, route := range entityPageRoutes {
		expected, ok := want[route.Name]
		if !ok {
			t.Errorf("unexpected route %q", route.Name)
			continue
		}
		if route.Canonical != expected.canonical {
			t.Errorf("%s path = %q, want %q", route.Name, route.Canonical, expected.canonical)
		}
		if !reflect.DeepEqual(route.Aliases, expected.aliases) {
			t.Errorf("%s aliases = %v, want %v", route.Name, route.Aliases, expected.aliases)
		}
		var types []string
		for kind := range route.Types {
			types = append(types, kind)
		}
		sortStrings(types)
		if !reflect.DeepEqual(types, expected.types) {
			t.Errorf("%s types = %v, want %v", route.Name, types, expected.types)
		}
		if route.Load == nil || route.TTL <= 0 {
			t.Errorf("%s has incomplete loader/cache metadata", route.Name)
		}
		for _, path := range append([]string{route.Canonical}, route.Aliases...) {
			if strings.HasPrefix(path, "/api/") {
				t.Errorf("%s includes transport prefix: %s", route.Name, path)
			}
			if strings.Contains(path, "/top") && route.Name != "top-lists" {
				t.Errorf("dead /top wrapper was restored: %s", path)
			}
			if strings.Contains(path, "/fits") {
				t.Errorf("dead /fits wrapper was restored: %s", path)
			}
		}
	}
}

func TestEntityPageAliasTypeInferenceOnlyAcceptsSingularEntityPages(t *testing.T) {
	for path, want := range map[string]string{
		"/character/{id}":              "character",
		"/corporation/{id}/members":    "corporation",
		"/alliance/{id}/corporations":  "alliance",
		"/faction/{id}":                "faction",
		"/entity/{type}/{id}/killlist": "",
		"/entities/{type}/{id}":        "",
	} {
		if got := entityTypeFromAlias(path); got != want {
			t.Errorf("type for %q = %q, want %q", path, got, want)
		}
	}
}

func TestEntityPageDaysSnapToStatsRetentionWindows(t *testing.T) {
	for days, want := range map[int]string{
		-1: "alltime", 0: "alltime", 1: "1d", 2: "7d", 7: "7d",
		8: "14d", 14: "14d", 15: "30d", 30: "30d",
		31: "90d", 90: "90d", 91: "180d", 180: "180d",
		181: "365d", 9999: "365d",
	} {
		if got := entityPageDaysWindow(days); got != want {
			t.Errorf("%d days = %q, want %q", days, got, want)
		}
	}
}

func TestRenderEntityBioPreservesUsefulMarkupWithoutForwardingHTML(t *testing.T) {
	got, ok := renderEntityBio(
		"Hello **pilot**. [Board](https://example.com/?a=1&b=2)\n\n<script>alert(1)</script>",
		"markdown",
	).(string)
	if !ok {
		t.Fatalf("markdown bio = %#v, want string", got)
	}
	for _, fragment := range []string{
		"<strong>pilot</strong>",
		`href="https://example.com/?a=1&amp;b=2"`,
		`target="_blank"`,
		`rel="noopener noreferrer nofollow"`,
		"&lt;script&gt;alert(1)&lt;/script&gt;",
	} {
		if !strings.Contains(got, fragment) {
			t.Errorf("markdown bio missing %q: %s", fragment, got)
		}
	}
	if strings.Contains(got, "<script>") {
		t.Fatalf("raw script survived: %s", got)
	}

	eveBio := renderEntityBio(
		`<a href="showinfo:16159//99000001">Alliance</a>`+"\n"+
			`<url=javascript:alert(1)>bad</url>`,
		"eve_html",
	).(string)
	if !strings.Contains(eveBio, `href="/alliance/99000001"`) {
		t.Errorf("showinfo link was not translated: %s", eveBio)
	}
	if strings.Contains(eveBio, "href=\"javascript:") {
		t.Errorf("unsafe URL survived: %s", eveBio)
	}
	if got := renderEntityBio(" \n ", "markdown"); got != nil {
		t.Errorf("blank bio = %#v, want nil", got)
	}
}

func TestEntityDetailStatsMatchesFrontendDerivedValues(t *testing.T) {
	got := entityDetailStats(map[string]any{
		"kills": int64(2), "losses": int64(1),
		"solo_kills": int64(1), "npc_losses": int64(0),
		"isk_destroyed": float64(70), "isk_lost": float64(30),
		"points": int64(3), "final_blows": int64(2),
		"damage_dealt": int64(12), "damage_taken": int64(5),
	}, true)
	if got["efficiency"] != int64(67) || got["isk_efficiency"] != int64(70) {
		t.Errorf("derived efficiencies = %#v", got)
	}
	if got["damage_dealt"] != int64(12) || got["damage_taken"] != int64(5) {
		t.Errorf("damage fields = %#v", got)
	}
}

func TestEmptyOrganizationIntelIsStableAndAllianceOnlyHasCorpPlaceholder(t *testing.T) {
	corp := emptyOrganizationIntel("corporation")
	alliance := emptyOrganizationIntel("alliance")
	if corp["allies"] == nil || corp["recentJoins"] == nil {
		t.Fatalf("corporation empty shape has nil arrays: %#v", corp)
	}
	if _, ok := corp["census"].(map[string]any)["corps"]; ok {
		t.Error("corporation census unexpectedly has corps")
	}
	if _, ok := alliance["census"].(map[string]any)["corps"]; !ok {
		t.Error("alliance degraded census lost its compatibility corps array")
	}
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
