package renderer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEntrypointFindsParentWebBuild(t *testing.T) {
	root := t.TempDir()
	entrypoint := filepath.Join(root, "web", ".output", "server", "index.mjs")
	if err := os.MkdirAll(filepath.Dir(entrypoint), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entrypoint, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "some", "nested", "directory")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)

	got, found, err := ResolveEntrypoint("")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("ResolveEntrypoint did not find the renderer")
	}
	if got != entrypoint {
		t.Fatalf("entrypoint = %q, want %q", got, entrypoint)
	}
}

func TestResolveEntrypointExplainsMissingBuild(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "web", "package.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	if _, _, err := ResolveEntrypoint(""); err == nil {
		t.Fatal("ResolveEntrypoint should explain that the frontend needs building")
	}
}

func TestResolveEntrypointAllowsAPIOnlyInstall(t *testing.T) {
	t.Chdir(t.TempDir())
	entrypoint, found, err := ResolveEntrypoint("")
	if err != nil {
		t.Fatal(err)
	}
	if found || entrypoint != "" {
		t.Fatalf("entrypoint = %q, found = %t; want no renderer", entrypoint, found)
	}
}

func TestRemoveStaleSocketNeverDeletesOrdinaryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "important")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleSocket(path); err == nil {
		t.Fatal("removeStaleSocket should reject an ordinary file")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("ordinary file was removed: %v", err)
	}
}

func TestReplaceEnv(t *testing.T) {
	got := replaceEnv(
		[]string{"PATH=/bin", "NODE_ENV=development", "NUXT_API_SOCKET=old"},
		map[string]string{
			"NODE_ENV":        "production",
			"NUXT_API_SOCKET": "/tmp/api.sock",
		},
	)
	values := make(map[string]string, len(got))
	for _, item := range got {
		key, value, _ := splitEnv(item)
		values[key] = value
	}
	if values["PATH"] != "/bin" {
		t.Fatalf("PATH = %q", values["PATH"])
	}
	if values["NODE_ENV"] != "production" {
		t.Fatalf("NODE_ENV = %q", values["NODE_ENV"])
	}
	if values["NUXT_API_SOCKET"] != "/tmp/api.sock" {
		t.Fatalf("NUXT_API_SOCKET = %q", values["NUXT_API_SOCKET"])
	}
}

func splitEnv(value string) (string, string, bool) {
	for i := range value {
		if value[i] == '=' {
			return value[:i], value[i+1:], true
		}
	}
	return value, "", false
}
