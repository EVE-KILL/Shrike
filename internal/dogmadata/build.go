// Package dogmadata builds the FlatBuffer consumed by browser and native Rust.
package dogmadata

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/eve-kill/shrike/internal/dogmadata/upstream/fbs"
	"github.com/eve-kill/shrike/internal/dogmadata/upstream/patch"
	data "github.com/eve-kill/shrike/internal/dogmadata/upstream/sde"
	"github.com/eve-kill/shrike/internal/sde"
)

type Result struct {
	Types, Attributes, Effects, TypeDogma int
	SyntheticAttributes, SyntheticEffects int
}

func Build(ctx context.Context, source *sde.Source, patches map[string][]byte, destination string) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if source.Build < 1 || source.Build > math.MaxInt32 {
		return Result{}, fmt.Errorf("SDE build out of range: %d", source.Build)
	}
	raw, err := data.Load(source.Path, int32(source.Build))
	if err != nil {
		return Result{}, fmt.Errorf("load dogma SDE: %w", err)
	}
	raw.MajorVersion = DataMajorVersion
	patchDir, err := os.MkdirTemp("", "shrike-dogma-patches-")
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = os.RemoveAll(patchDir) }()
	for name, contents := range patches {
		if filepath.IsAbs(name) || filepath.Clean(name) != name || strings.HasPrefix(name, "..") {
			return Result{}, fmt.Errorf("invalid patch path %q", name)
		}
		path := filepath.Join(patchDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			return Result{}, err
		}
		if err := os.WriteFile(path, contents, 0600); err != nil {
			return Result{}, err
		}
	}
	spec, err := patch.Load(patchDir)
	if err != nil {
		return Result{}, fmt.Errorf("load dogma patches: %w", err)
	}
	if err := spec.Validate(); err != nil {
		return Result{}, fmt.Errorf("validate dogma patches: %w", err)
	}
	if _, err := patch.Apply(spec, raw); err != nil {
		return Result{}, fmt.Errorf("apply dogma patches: %w", err)
	}
	if err := writeBundle(ctx, raw, destination); err != nil {
		return Result{}, err
	}
	result := Result{Types: len(raw.Types), Attributes: len(raw.DogmaAttributes), Effects: len(raw.DogmaEffects), TypeDogma: len(raw.Types)}
	for id := range raw.DogmaAttributes {
		if id < 0 {
			result.SyntheticAttributes++
		}
	}
	for id := range raw.DogmaEffects {
		if id < 0 {
			result.SyntheticEffects++
		}
	}
	return result, nil
}
func writeBundle(ctx context.Context, raw *data.Data, destination string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0750); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".dogma-data-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if err := fbs.Write(raw, filepath.Join(staging, "sde.dat")); err != nil {
		return err
	}
	if err := fbs.WriteNames(raw, filepath.Join(staging, "names.dat")); err != nil {
		return err
	}
	if err := fbs.WriteTexts(raw, filepath.Join(staging, "texts.dat")); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Preserve the old complete bundle if the final rename fails.
	old := destination + ".previous"
	if _, err := os.Stat(old); err == nil {
		return fmt.Errorf("previous dogma bundle exists at %s", old)
	}
	hadOld := false
	if _, err := os.Stat(destination); err == nil {
		if err := os.Rename(destination, old); err != nil {
			return err
		}
		hadOld = true
	}
	if err := os.Rename(staging, destination); err != nil {
		if hadOld {
			_ = os.Rename(old, destination)
		}
		return err
	}
	if hadOld {
		return os.RemoveAll(old)
	}
	return nil
}
