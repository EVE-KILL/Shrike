package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var (
	flagOpenAPIOutput  string
	flagOpenAPIFormat  string
	flagOpenAPIVersion string
)

var openAPICmd = &cobra.Command{
	Use:         "openapi-spec",
	Short:       "Write the generated Huma OpenAPI document",
	Annotations: map[string]string{skipConfigAnnotation: "true"},
	Long: `Builds the complete API catalogue without starting a listener or
reading runtime configuration or touching Postgres, Redis, Memgraph, ESI, or B2.

The committed JSON output is the source for the generated TypeScript contract
used by the Nuxt renderer.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if flagOpenAPIVersion != "3.0" && flagOpenAPIVersion != "3.1" {
			return fmt.Errorf(
				"unknown --version %q (want 3.0 or 3.1)",
				flagOpenAPIVersion,
			)
		}

		document := api.New(api.Options{
			Version: ui.Version,
			Commit:  ui.Commit,
		}).OpenAPI()

		var (
			output []byte
			err    error
		)
		switch flagOpenAPIFormat {
		case "json":
			if flagOpenAPIVersion == "3.0" {
				output, err = document.Downgrade()
			} else {
				output, err = json.MarshalIndent(document, "", "  ")
			}
		case "yaml":
			if flagOpenAPIVersion == "3.0" {
				output, err = document.DowngradeYAML()
			} else {
				output, err = document.YAML()
			}
		default:
			return fmt.Errorf(
				"unknown --format %q (want json or yaml)",
				flagOpenAPIFormat,
			)
		}
		if err != nil {
			return fmt.Errorf("marshal OpenAPI document: %w", err)
		}

		if flagOpenAPIOutput == "" || flagOpenAPIOutput == "-" {
			_, err = os.Stdout.Write(output)
			if err == nil && (len(output) == 0 || output[len(output)-1] != '\n') {
				_, err = os.Stdout.Write([]byte("\n"))
			}
			return err
		}
		if err := os.WriteFile(flagOpenAPIOutput, output, 0o600); err != nil {
			return fmt.Errorf(
				"write OpenAPI document %s: %w",
				flagOpenAPIOutput,
				err,
			)
		}
		return nil
	},
}

func init() {
	openAPICmd.Flags().StringVarP(
		&flagOpenAPIOutput,
		"output",
		"o",
		"",
		"Write to this file instead of stdout",
	)
	openAPICmd.Flags().StringVar(
		&flagOpenAPIFormat,
		"format",
		"json",
		"Output format: json or yaml",
	)
	openAPICmd.Flags().StringVar(
		&flagOpenAPIVersion,
		"version",
		"3.1",
		"OpenAPI version: 3.0 or 3.1",
	)
}
