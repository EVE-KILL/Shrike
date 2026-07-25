package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// The CLI presents Symfony-style colon names (`db:migrate`) over a Cobra
// subcommand tree (`db migrate`). Both spellings work: the tree gives us
// grouped help, per-group `--help`, and generated shell completion, while the
// colon form keeps every existing runbook, Dockerfile CMD, and K8s manifest
// that already invokes `cli db:migrate` valid.
//
// The bridge is a single argv rewrite before Cobra ever sees the arguments.

// expandColonArgs rewrites a leading colon-joined command into separate path
// segments. Only the first argument is considered — a colon appearing in a
// flag value (`--filter=type:ship`) or a positional argument must survive
// untouched, which is why this does not scan the whole slice.
func expandColonArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	first := args[0]
	if !strings.Contains(first, ":") || strings.HasPrefix(first, "-") {
		return args
	}
	segments := strings.Split(first, ":")
	out := make([]string, 0, len(segments)+len(args)-1)
	for _, s := range segments {
		if s == "" {
			// A malformed name like `db::migrate` — leave the original alone so
			// Cobra reports an unknown command rather than us silently
			// resolving something the user did not type.
			return args
		}
		out = append(out, s)
	}
	return append(out, args[1:]...)
}

// canonicalName renders a command's colon-joined name, which is how every
// command is displayed and documented.
func canonicalName(cmd *cobra.Command) string {
	var parts []string
	for c := cmd; c != nil && c.Parent() != nil; c = c.Parent() {
		parts = append([]string{c.Name()}, parts...)
	}
	return strings.Join(parts, ":")
}

// isGroup reports whether a command exists only to hold subcommands. Groups are
// rendered as help sections rather than as runnable commands.
func isGroup(cmd *cobra.Command) bool {
	return cmd.HasSubCommands() && !cmd.Runnable()
}

// leafCommands returns every runnable command beneath root, depth-first, with
// hidden and Cobra's generated helper commands excluded.
func leafCommands(root *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	for _, c := range root.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		if c.Runnable() {
			out = append(out, c)
		}
		if c.HasSubCommands() {
			out = append(out, leafCommands(c)...)
		}
	}
	return out
}
