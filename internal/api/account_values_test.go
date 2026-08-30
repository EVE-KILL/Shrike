package api

import (
	"strings"
	"testing"
)

func TestSanitizeDefaultTabsAcceptsRenderedFactionChoices(t *testing.T) {
	tabs, ok := sanitizeDefaultTabs(map[string]any{
		"faction": "losses",
		"region":  "bogus",
		"unknown": "dashboard",
	})
	if !ok {
		t.Fatal("valid object was rejected")
	}
	if len(tabs) != 1 || tabs["faction"] != "losses" {
		t.Fatalf("tabs = %#v", tabs)
	}
}

func TestSanitizeThemeRejectsUnknownKeysAndCSSInjection(t *testing.T) {
	theme, ok := sanitizeTheme(map[string]any{
		"brandPrimary": "#abc",
		"bgPrimary":    `url("javascript:alert(1)")`,
		"notAThemeKey": "red",
	})
	if !ok {
		t.Fatal("valid object was rejected")
	}
	if len(theme) != 1 || theme["brandPrimary"] != "#abc" {
		t.Fatalf("theme = %#v", theme)
	}
}

func TestCleanBoardListNormalizesDeduplicatesAndCaps(t *testing.T) {
	raw := []any{" Alpha ", "alpha", "board.example", "not valid!"}
	for i := range maxBoardEntries + 5 {
		raw = append(raw, "board-"+string(rune('a'+i)))
	}
	result := cleanBoardList(raw)
	if len(result) != maxBoardEntries {
		t.Fatalf("boards = %d, want cap %d: %#v",
			len(result), maxBoardEntries, result)
	}
	if result[0] != "alpha" || result[1] != "board.example" {
		t.Fatalf("boards = %#v", result)
	}
}

func TestRenderEVEHTMLRewritesInternalLinksAndDropsActiveContent(t *testing.T) {
	rendered := renderEVEHTML(
		`<url=showinfo:5//30000142>Jita</url>` +
			`<url=javascript:alert(1)>bad</url>` +
			`<script>alert(2)</script>`,
	)
	if !strings.Contains(rendered, `href="/system/30000142"`) {
		t.Fatalf("internal showinfo link was not rewritten: %s", rendered)
	}
	if strings.Contains(rendered, "javascript:") ||
		strings.Contains(rendered, "<script") ||
		strings.Contains(rendered, "alert(2)") {
		t.Fatalf("active content survived sanitization: %s", rendered)
	}
}

func TestAccountESILogFilterKeepsValuesParameterized(t *testing.T) {
	filter := newAccountESILogFilter(accountESILogQuery{
		CharacterID: 42,
		Source:      `x' OR TRUE --`,
		Status:      "error",
		Endpoint:    "character",
	})
	where := filter.where()
	if strings.Contains(where, `x' OR TRUE`) {
		t.Fatalf("source value was interpolated into SQL: %s", where)
	}
	if !strings.Contains(where, "source = $2") ||
		len(filter.args) != 2 ||
		filter.args[1] != `x' OR TRUE --` {
		t.Fatalf("filter = (%s, %#v)", where, filter.args)
	}
}
