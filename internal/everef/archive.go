package everef

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Several datasets ship as one tar.bz2 holding thousands of small JSON files —
// a year of sovereignty snapshots, a day of killmails, a year of wars. They are
// walked as a stream: an uncompressed year of wars is gigabytes, and the point
// of each walk is to keep only a fraction of it.

// maxMemberBytes caps a single file inside an archive. The largest legitimate
// member is a killmail with a few thousand items, well under a megabyte; the
// cap is what stops a corrupt header from being read as a huge length.
const maxMemberBytes = 32 << 20

// WalkArchive calls fn for every .json member of a tar.bz2, in archive order.
//
// fn receives the member's name and its already-read contents, because every
// caller needs the whole document to parse it and streaming a 2 KB file buys
// nothing. Returning an error from fn aborts the walk.
func (c *Client) WalkArchive(ctx context.Context, url string, fn func(name string, data []byte) error) error {
	return c.Stream(ctx, url, func(r io.Reader) error {
		tr := tar.NewReader(r)
		for {
			// A cancelled import must stop here rather than after the archive
			// finishes, which for a yearly file can be many minutes later.
			if err := ctx.Err(); err != nil {
				return err
			}

			hdr, err := tr.Next()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("read %s: %w", url, err)
			}
			if hdr.Typeflag != tar.TypeReg || !strings.HasSuffix(hdr.Name, ".json") {
				continue
			}

			data, err := io.ReadAll(io.LimitReader(tr, maxMemberBytes))
			if err != nil {
				return fmt.Errorf("read %s from %s: %w", hdr.Name, url, err)
			}
			if err := fn(hdr.Name, data); err != nil {
				return err
			}
		}
	})
}

// decodeMember parses one archive member, reporting whether it was usable.
//
// Malformed members are skipped rather than fatal. These archives are
// machine-generated and occasionally truncated, and one bad file out of twenty
// thousand is not a reason to abandon a day's import.
func decodeMember(data []byte, out any) bool {
	return json.Unmarshal(data, out) == nil
}
