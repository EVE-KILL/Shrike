package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultOldImagesCacheDirectory(t *testing.T) {
	temporaryRoot := t.TempDir()

	t.Run("uses existing data directory", func(t *testing.T) {
		dataRoot := filepath.Join(t.TempDir(), ".data")
		if err := os.Mkdir(dataRoot, 0o755); err != nil {
			t.Fatal(err)
		}

		got := defaultOldImagesCacheDirectory(dataRoot, temporaryRoot)
		want := filepath.Join(dataRoot, "images")
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("uses temporary directory when data directory is absent", func(t *testing.T) {
		dataRoot := filepath.Join(t.TempDir(), ".data")

		got := defaultOldImagesCacheDirectory(dataRoot, temporaryRoot)
		want := filepath.Join(temporaryRoot, "shrike", "images")
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}
