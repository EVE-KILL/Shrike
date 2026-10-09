package frontend

import (
	"testing"

	"github.com/eve-kill/shrike/internal/api"
)

func TestPaletteAccentMatchesNuxtLeaderboardColor(t *testing.T) {
	if got := paletteAccent(&api.TopPalette{MainColor: "#960000"}); got != "#db5344" {
		t.Fatalf("accent for the first top character = %s, want #db5344", got)
	}
	if got := paletteAccent(&api.TopPalette{MainColor: "#999999"}); got != "" {
		t.Fatalf("neutral palette accent = %s, want none", got)
	}
}
