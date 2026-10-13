package dogmadata

import (
	"context"
	"embed"
	"io/fs"
)

const ConverterCommit = "2ee8ea5334ed5f90da800fddc90a6fab0d6c2178"
const DataMajorVersion = 12

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
