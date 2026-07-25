package cli

import (
	"runtime"

	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and build information",
	RunE: func(_ *cobra.Command, _ []string) error {
		if ui.JSONMode {
			return ui.JSON(map[string]string{
				"version": ui.Version,
				"commit":  ui.Commit,
				"go":      runtime.Version(),
				"os":      runtime.GOOS,
				"arch":    runtime.GOARCH,
			})
		}

		ui.Printf("%s", ui.Banner())
		ui.Section("Build")
		ui.KV("Version", ui.Bold(ui.Version))
		ui.KV("Commit", ui.Commit)
		ui.KV("Go", runtime.Version())
		ui.KV("Platform", runtime.GOOS+"/"+runtime.GOARCH)
		ui.Newline()
		return nil
	},
}
