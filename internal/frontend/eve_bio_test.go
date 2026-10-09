package frontend

import (
	"strings"
	"testing"
)

func TestEVEBioConvertsLegacyMarkupWithoutExecutableHTML(t *testing.T) {
	input := `u'<font size="24" color="#ff00ff00"><b>Queen</b></font><br>Sir, we\'re surrounded. \u2584 <a href="killReport:123169687:abc">Kill report</a><img src=x onerror="alert(1)"><script>alert(2)</script>'`
	got := renderEVEBio(input)
	for _, want := range []string{`class="eve-bio-magenta eve-bio-xl"`, "<strong>Queen</strong>", "Sir, we&#39;re surrounded", "▄", `href="/kill/123169687"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	for _, forbidden := range []string{"<script", "<img", "onerror", "alert(1)", "alert(2)", "killReport:"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("unsafe or unconverted %q in %q", forbidden, got)
		}
	}
}
