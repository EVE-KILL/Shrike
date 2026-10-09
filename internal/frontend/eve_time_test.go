package frontend

import "testing"

func TestEveDateMatchesUTCDateRatherThanInputOffset(t *testing.T) {
	for raw, want := range map[string]string{
		"2026-10-05T23:30:00-02:00": "6 Oct 2026",
		"2026-10-05":                "5 Oct 2026",
		"":                          "",
		"invalid":                   "",
	} {
		if got := eveDate(raw); got != want {
			t.Fatalf("eveDate(%q) = %q, want %q", raw, got, want)
		}
	}
}
