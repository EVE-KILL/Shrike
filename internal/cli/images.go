package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/eve-kill/shrike/internal/images"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Manage the Backblaze B2 image library",
	Long: `Imports durable source assets and synchronizes generated EVE images.

HTTP serving happens inside "shrike serve"; these commands intentionally do not
allocate its large in-memory response cache.`,
}

var (
	flagImagesSource         string
	flagImagesArchive        string
	flagImagesConcurrency    int
	flagOldImagesCacheDir    string
	flagOldImagesConcurrency int
	flagOldImagesForce       bool
	flagTypeSyncArchive      string
	flagTypeSyncConcurrency  int
)

var imagesImportStaticCmd = &cobra.Command{
	Use:   "import-static",
	Short: "Upload map, UI, Dust 514, and overlay assets to B2",
	RunE: func(cmd *cobra.Command, _ []string) error {
		store, err := openImageStore()
		if err != nil {
			return err
		}
		start := time.Now()
		result, err := images.ImportStaticTree(
			cmd.Context(),
			store,
			flagImagesSource,
			images.ImportOptions{
				Concurrency: flagImagesConcurrency,
				Progress:    reportImageProgress,
			},
		)
		if err != nil {
			return err
		}
		return reportImageImport("Static images", result, time.Since(start))
	},
}

var imagesImportOldCharactersCmd = &cobra.Command{
	Use:   "import-old-characters",
	Short: "Upload the static EVE Ref legacy portrait archive to B2",
	Long: `Downloads OldCharPortraits_256.zip from EVE Ref, or reads --archive,
and uploads each portrait under its sharded B2 key. Remote downloads resume
from the local cache, and completed B2 shards are skipped on restart.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		store, err := openImageStore()
		if err != nil {
			return err
		}
		start := time.Now()
		result, err := images.ImportOldCharacters(
			cmd.Context(),
			store,
			flagImagesArchive,
			&http.Client{},
			images.ImportOptions{
				Concurrency:      flagOldImagesConcurrency,
				CacheDirectory:   flagOldImagesCacheDir,
				Force:            flagOldImagesForce,
				Progress:         reportImageProgress,
				DownloadProgress: reportImageDownloadProgress,
			},
		)
		if err != nil {
			return err
		}
		return reportImageImport("Old character portraits", result, time.Since(start))
	},
}

var imagesSyncTypesCmd = &cobra.Command{
	Use:   "sync-types",
	Short: "Synchronize the latest TurtleTools Image Export Collection to B2",
	RunE: func(cmd *cobra.Command, _ []string) error {
		store, err := openImageStore()
		if err != nil {
			return err
		}
		start := time.Now()
		result, err := images.SyncTypeExport(
			cmd.Context(),
			store,
			images.TypeExportSyncOptions{
				Token: os.Getenv("GITHUB_TOKEN"), UserAgent: userAgent(),
				Archive:     flagTypeSyncArchive,
				Concurrency: flagTypeSyncConcurrency,
				Progress:    reportImageProgress,
			},
		)
		if err != nil {
			return err
		}
		if ui.JSONMode {
			return ui.JSON(result)
		}
		ui.Section("Type images")
		ui.KV("Release", result.Release)
		ui.KV("Digest", result.Digest)
		ui.KV("Changed", yesNo(result.Changed))
		ui.KV("Objects uploaded", fmtCount(result.Import.Uploaded))
		ui.KV("Objects skipped", fmtCount(result.Import.Skipped))
		ui.KV("Uploaded bytes", formatBytes(result.Import.Bytes))
		ui.KV("Time", time.Since(start).Round(time.Millisecond).String())
		ui.Newline()
		if result.Changed {
			ui.Success("Published TurtleTools release %s.", result.Release)
		} else {
			ui.Success("TurtleTools images are already current.")
		}
		ui.Newline()
		return nil
	},
}

func openImageStore() (images.ObjectStore, error) {
	if err := requireConfig(); err != nil {
		return nil, err
	}
	store, err := newImageStorage(cfg)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, fmt.Errorf(
			"image storage is disabled; configure B2_ENDPOINT, B2_IMAGES_BUCKET, B2_KEY_ID, and B2_APP_KEY",
		)
	}
	return store, nil
}

func reportImageProgress(result images.ImportResult) {
	if ui.JSONMode {
		return
	}
	ui.Printf(
		"  %s scanned, %s uploaded, %s skipped\r",
		fmtCount(result.Scanned),
		fmtCount(result.Uploaded),
		fmtCount(result.Skipped),
	)
}

func reportImageDownloadProgress(completed, total int64) {
	if ui.JSONMode {
		return
	}
	percentage := 0.0
	if total > 0 {
		percentage = float64(completed) / float64(total) * 100
	}
	ui.Printf(
		"  Archive %s / %s  %.1f%%\r",
		formatBytes(completed),
		formatBytes(total),
		percentage,
	)
}

func reportImageImport(
	name string,
	result images.ImportResult,
	elapsed time.Duration,
) error {
	if ui.JSONMode {
		return ui.JSON(result)
	}
	ui.Section(name)
	ui.KV("Objects scanned", fmtCount(result.Scanned))
	ui.KV("Objects uploaded", fmtCount(result.Uploaded))
	ui.KV("Objects skipped", fmtCount(result.Skipped))
	ui.KV("Uploaded bytes", formatBytes(result.Bytes))
	ui.KV("Time", elapsed.Round(time.Millisecond).String())
	ui.Newline()
	ui.Success("%s import complete.", name)
	ui.Newline()
	return nil
}

func formatBytes(value int64) string {
	const (
		kib = 1 << 10
		mib = 1 << 20
		gib = 1 << 30
	)
	switch {
	case value >= gib:
		return fmt.Sprintf("%.2f GiB", float64(value)/gib)
	case value >= mib:
		return fmt.Sprintf("%.2f MiB", float64(value)/mib)
	case value >= kib:
		return fmt.Sprintf("%.2f KiB", float64(value)/kib)
	default:
		return fmt.Sprintf("%d B", value)
	}
}

func init() {
	imagesImportStaticCmd.Flags().StringVar(
		&flagImagesSource,
		"source",
		"../imageserver",
		"Path to the existing image-server asset tree",
	)
	imagesImportStaticCmd.Flags().IntVar(
		&flagImagesConcurrency,
		"concurrency",
		8,
		"Concurrent B2 uploads",
	)
	imagesImportOldCharactersCmd.Flags().StringVar(
		&flagImagesArchive,
		"archive",
		"",
		"Local archive path or URL (default: EVE Ref)",
	)
	imagesImportOldCharactersCmd.Flags().IntVar(
		&flagOldImagesConcurrency,
		"concurrency",
		64,
		"Concurrent B2 uploads",
	)
	imagesImportOldCharactersCmd.Flags().StringVar(
		&flagOldImagesCacheDir,
		"cache-dir",
		filepath.Join(".data", "images"),
		"Directory for the resumable archive download",
	)
	imagesImportOldCharactersCmd.Flags().BoolVar(
		&flagOldImagesForce,
		"force",
		false,
		"Re-upload all portraits even when completion markers match",
	)
	imagesSyncTypesCmd.Flags().StringVar(
		&flagTypeSyncArchive,
		"archive",
		"",
		"Use an existing Image Export Collection archive instead of downloading",
	)
	imagesSyncTypesCmd.Flags().IntVar(
		&flagTypeSyncConcurrency,
		"concurrency",
		32,
		"Concurrent B2 uploads",
	)
	imagesCmd.AddCommand(
		imagesImportStaticCmd,
		imagesImportOldCharactersCmd,
		imagesSyncTypesCmd,
	)
}
