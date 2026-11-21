package dogmadata

import (
	"context"
	"embed"
	"io/fs"
)

const ConverterCommit = "d6b270f9b7121b25e0820ec9d4037f593beb7952"
const DataMajorVersion = 14

//go:embed upstream/patches
var patchFiles embed.FS

// FetchPatches retains the CLI contract. Pinned upstream patches are embedded,
// so building a new CCP dataset never depends on a mutable external patch set.
func FetchPatches(ctx context.Context, _ string, _ string) (map[string][]byte, error) {
	out := map[string][]byte{}
	err := fs.WalkDir(patchFiles, "upstream/patches", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := patchFiles.ReadFile(name)
		if err != nil {
			return err
		}
		out[name[len("upstream/patches/"):]] = data
		return nil
	})
	return out, err
}
