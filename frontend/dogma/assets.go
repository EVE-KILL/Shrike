// Package dogmaassets exposes the existing dogma engine data to the Go
// frontend without copying its 10 MB SDE into a second source tree.
package dogmaassets

import "embed"

//go:embed dist/upstream/data/dist/sde.dat dist/upstream/dogma-engine/*.wasm
var Data embed.FS
