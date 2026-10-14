package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eve-kill/shrike/internal/config"
)

func TestSDECacheUsesRuntimeDirectoryAndExplicitOverride(t *testing.T) {
	previousConfig, previousOverride := cfg, flagSDECacheDir
	t.Cleanup(func() { cfg, flagSDECacheDir = previousConfig, previousOverride })
	state := filepath.Join(t.TempDir(), "runtime")
	cfg, flagSDECacheDir = &config.Config{DataDir: state}, ""
	if got := cacheDir(); got != filepath.Join(state, "sde") {
		t.Fatalf("configured runtime cache = %q", got)
	}
	if err := os.MkdirAll(cacheDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	flagSDECacheDir = filepath.Join(t.TempDir(), "explicit")
	if got := cacheDir(); got != flagSDECacheDir {
		t.Fatalf("explicit cache override = %q", got)
	}
	cfg, flagSDECacheDir = nil, ""
	if got := cacheDir(); got != filepath.Join("data", "sde") {
		t.Fatalf("default cache = %q", got)
	}
}
