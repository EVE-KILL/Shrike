package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// This file replaces Cobra's default help with the layout the Bun CLI already
// uses (backend/src/cli/Kernel.ts): a banner, a usage line, then commands
// grouped by namespace with colon-joined names. Keeping the two CLIs visually
// identical matters while both exist side by side during the port.

// installHelp swaps in the custom renderers for the whole command tree.
func installHelp(root *cobra.Command) {
	root.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		if cmd.Parent() == nil {
			renderRootHelp(cmd)
			return
		}
		renderCommandHelp(cmd)
	})
	// Cobra prints usage on error; route it through the same renderer so a
	// mistyped flag does not produce stock-Cobra output mid-session.
	root.SetUsageFunc(func(cmd *cobra.Command) error {
		if cmd.Parent() == nil {
			renderRootHelp(cmd)
		} else {
			renderCommandHelp(cmd)
		}
		return nil
	})
}

func renderRootHelp(root *cobra.Command) {
	fmt.Print(ui.Banner())

	ui.Section("Usage")
	fmt.Printf("  %s shrike %s %s\n",
		ui.Dim("$"), ui.Primary("<command>"), ui.Dim("[options] [arguments]"))

	// Top-level runnable commands have no namespace and come first, matching
	// Kernel.ts's "_ungrouped" handling.
	var ungrouped [][2]string
	groups := map[string][][2]string{}

	for _, c := range root.Commands() {
		if c.Hidden || c.Name() == "help" {
			continue
		}
		if isGroup(c) {
			for _, leaf := range leafCommands(c) {
				groups[c.Name()] = append(groups[c.Name()],
					[2]string{ui.Command(canonicalName(leaf)), leaf.Short})
			}
			// A group that is itself runnable (e.g. `serve` with a default
			// action) still deserves a line in its own section.
			if c.Runnable() {
				groups[c.Name()] = append(groups[c.Name()],
					[2]string{ui.Command(c.Name()), c.Short})
			}
			continue
		}
		if c.Runnable() {
			ungrouped = append(ungrouped, [2]string{ui.Command(c.Name()), c.Short})
		}
	}

	if len(ungrouped) > 0 {
		ui.Section("Available Commands")
		fmt.Println(ui.List(ungrouped, "  "))
	}

	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		rows := groups[name]
		sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
		ui.Section(name)
		fmt.Println(ui.List(rows, "  "))
	}

	if flags := visibleFlags(root.PersistentFlags()); len(flags) > 0 {
		ui.Section("Global Options")
		fmt.Println(ui.List(flags, "  "))
	}

	ui.Newline()
	fmt.Printf("  %s\n", ui.Dim("Run 'shrike <command> --help' for details on any command."))
	ui.Newline()
}

func renderCommandHelp(cmd *cobra.Command) {
	name := canonicalName(cmd)

	ui.Newline()
	fmt.Printf("  %s %s %s\n", ui.Bold(name), ui.Dim("—"), cmd.Short)

	ui.Section("Usage")
	usage := name
	if isGroup(cmd) {
		usage += " " + ui.Primary("<subcommand>")
	}
	if args := positionalHint(cmd); args != "" {
		usage += " " + args
	}
	fmt.Printf("  %s shrike %s %s\n", ui.Dim("$"), usage, ui.Dim("[options]"))

	// Subcommands, when this is a namespace.
	if cmd.HasSubCommands() {
		var rows [][2]string
		for _, leaf := range leafCommands(cmd) {
			rows = append(rows, [2]string{ui.Command(canonicalName(leaf)), leaf.Short})
		}
		if len(rows) > 0 {
			sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
			ui.Section("Subcommands")
			fmt.Println(ui.List(rows, "  "))
		}
	}

	if flags := visibleFlags(cmd.LocalFlags()); len(flags) > 0 {
		ui.Section("Options")
		fmt.Println(ui.List(flags, "  "))
	}

	if cmd.Long != "" {
		ui.Section("Help")
		for _, line := range strings.Split(strings.TrimSpace(cmd.Long), "\n") {
			fmt.Printf("  %s\n", line)
		}
	}

	if cmd.Example != "" {
		ui.Section("Examples")
		for _, line := range strings.Split(strings.TrimSpace(cmd.Example), "\n") {
			fmt.Printf("  %s\n", line)
		}
	}

	ui.Newline()
}

// visibleFlags formats a flag set into label/description pairs, skipping the
// help flag Cobra injects into every command.
func visibleFlags(fs *pflag.FlagSet) [][2]string {
	var rows [][2]string
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Name == "help" {
			return
		}
		label := "    --" + f.Name
		if f.Shorthand != "" {
			label = "-" + f.Shorthand + ", --" + f.Name
		}
		desc := f.Usage
		// Only surface defaults that carry information. Empty strings and false
		// are noise; a zero is worse than noise, because every numeric flag here
		// uses 0 as an "unset, fall back to config" sentinel, so printing it
		// would advertise a default the flag does not actually have.
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
			desc += ui.Dim(fmt.Sprintf(" (default: %s)", f.DefValue))
		}
		rows = append(rows, [2]string{ui.Command(label), desc})
	})
	return rows
}

// positionalHint renders declared positional arguments. Cobra has no argument
// metadata, so commands advertise theirs via the "args" annotation — e.g.
// Annotations: map[string]string{"args": "<queue> [count]"}.
func positionalHint(cmd *cobra.Command) string {
	if cmd.Annotations == nil {
		return ""
	}
	return cmd.Annotations["args"]
}
