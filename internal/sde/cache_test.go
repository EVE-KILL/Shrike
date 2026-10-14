package sde

import (
	"path/filepath"
	"testing"
)

func TestCacheDirMatchesNonRootContainerState(t *testing.T) {
	if got := CacheDir("/app/data"); got != "/app/data/sde" {
		t.Fatalf("container cache = %q", got)
	}
	if got := CacheDir(""); got != filepath.Join("data", "sde") {
		t.Fatalf("default cache = %q", got)
	}
}
