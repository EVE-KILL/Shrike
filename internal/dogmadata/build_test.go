package dogmadata

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/eve-kill/shrike/internal/dogmadata/upstream/fbs/eve"
	data "github.com/eve-kill/shrike/internal/dogmadata/upstream/sde"
)

func TestCancelledBuildPreservesExistingBundle(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "sde.dat")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := writeBundle(ctx, &data.Data{}, dir); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "existing" {
		t.Fatalf("old data lost: %q %v", content, err)
	}
}
func TestBundleIncludesVersionAndReplacesCompleteSet(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "old.pb2"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	raw := &data.Data{BuildNumber: 123, MajorVersion: DataMajorVersion, Types: map[int32]*data.Type{587: {Key: 587, Name: data.Localized{En: "Rifter"}, CategoryID: 6, Published: true}}}
	if err := writeBundle(t.Context(), raw, dir); err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(filepath.Join(dir, "sde.dat"))
	if err != nil {
		t.Fatal(err)
	}
	root := eve.GetRootAsSde(bytes, 0)
	if root.BuildNumber() != 123 || root.MajorVersion() != DataMajorVersion || root.TypesLength() != 1 {
		t.Fatal("bundle identity was not preserved")
	}
	for _, name := range []string{"names.dat", "texts.dat"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "old.pb2")); !os.IsNotExist(err) {
		t.Fatal("old format was retained in new bundle")
	}
}
