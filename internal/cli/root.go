package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/logging"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

// cfg is populated by the root PersistentPreRunE and is safe to read from any
// command's Run. Commands must not read it at construction time — flags have
// not been parsed then.
var cfg *config.Config

var (
	flagJSON    bool
	flagNoColor bool
	flagVerbose bool
	flagConfig  string
)

const skipConfigAnnotation = "shrike.eve-kill.com/skip-config"

var rootCmd = &cobra.Command{
	Use:           "shrike",
	Short:         "Shrike — the EVE-KILL backend",
	SilenceUsage:  true, // usage on every error is noise; we print it deliberately
	SilenceErrors: true, // Execute() formats errors through ui.Error instead
	// PersistentPreRunE rather than PersistentPreRun: a bad --config path or an
	// unparseable .env must fail here with a clear message, not surface later as
	// a nil dereference inside a command.
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		ui.Init(flagJSON, flagNoColor)

		// Some commands only inspect compile-time registrations. Keeping them
		// independent of .env means code generation works in fresh checkouts
		// and CI even when no runtime services have been configured yet.
		if cmd.Annotations[skipConfigAnnotation] == "true" {
			return nil
		}

		var err error
		cfg, err = config.Load(flagConfig)
		if err != nil {
			return err
		}

		level := cfg.LogLevel
		if flagVerbose {
			level = "debug"
		}
		logging.Init(level, cfg.LogJSON || flagJSON)

		// Stop go-redis writing pool diagnostics directly to stderr; route them
		// through zerolog at debug level instead.
		redis.SetLogger(logging.ThirdParty{})
		return nil
	},
	Run: func(cmd *cobra.Command, _ []string) {
		renderRootHelp(cmd)
	},
}

// Execute parses argv and dispatches. It owns the process exit code so that
// main stays a one-liner.
func Execute(version, commit string) {
	ui.Version = version
	ui.Commit = commit

	// Cobra resolves unambiguous prefixes and suggests near-misses, matching
	// the Bun Kernel's behaviour so both CLIs feel the same.
	cobra.EnablePrefixMatching = true
	rootCmd.SuggestionsMinimumDistance = 3

	installHelp(rootCmd)

	args := os.Args[1:]

	// Cobra reports unknown commands and bad flags *before* PersistentPreRunE
	// runs, so without this pre-pass those errors would be formatted with
	// ColorEnabled still at its default and leak escape sequences into pipes
	// and logs. PersistentPreRunE calls ui.Init again with the properly parsed
	// values, which is authoritative for everything a command prints.
	ui.Init(hasFlag(args, "json"), hasFlag(args, "no-color"))

	// Translate `db:migrate` into `db migrate` before Cobra parses anything.
	rootCmd.SetArgs(expandColonArgs(args))

	if err := rootCmd.Execute(); err != nil {
		// Signal-initiated shutdown is a normal outcome, not a failure.
		if errors.Is(err, errInterrupted) {
			os.Exit(130) // 128 + SIGINT, the shell convention
		}
		ui.Newline()
		ui.Error("%v", err)
		ui.Newline()
		os.Exit(1)
	}
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.BoolVar(&flagJSON, "json", false, "Machine-readable output; suppresses all decoration")
	pf.BoolVar(&flagNoColor, "no-color", false, "Disable colored output")
	pf.BoolVarP(&flagVerbose, "verbose", "v", false, "Debug-level logging")
	pf.StringVar(&flagConfig, "config", "", "Path to a .env file (default: nearest .env, searching upward)")

	rootCmd.AddCommand(
		serveCmd,
		versionCmd,
		doctorCmd,
		configCmd,
		dbCmd,
		queueCmd,
		cronCmd,
		backfillCmd,
		campaignCmd,
		graphCmd,
		battleCmd,
		rebuildCmd,
		catchupCmd,
		scanCmd,
		importCmd,
		exportCmd,
		workCmd,
		sdeCmd,
		everefCmd,
		pricesCmd,
		esiCmd,
		killmailCmd,
		imagesCmd,
		openAPICmd,
		completionCmd,
	)
}

// hasFlag reports whether a boolean flag appears in raw argv, accepting both
// `--flag` and `--flag=value`. It deliberately does not interpret the value:
// this only feeds the pre-parse ui.Init, and erring toward less colour in an
// error path is the safe direction.
func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == "--"+name || strings.HasPrefix(a, "--"+name+"=") {
			return true
		}
	}
	return false
}

// requireConfig guards commands that cannot run without configuration. It
// exists so a nil cfg produces an explanatory error rather than a panic if a
// command is ever invoked outside the normal PersistentPreRunE path (as
// happens in unit tests that call RunE directly).
func requireConfig() error {
	if cfg == nil {
		return fmt.Errorf("configuration was not loaded")
	}
	return nil
}
