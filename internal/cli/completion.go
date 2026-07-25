package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Completion is generated from the Cobra tree, so it completes the
// space-separated spelling (`shrike db migrate`). The colon aliases are an
// argv rewrite that happens after the shell has already done its work, so they
// cannot be completed — one more reason the tree is the real structure and
// colons are the compatibility layer.
var completionCmd = &cobra.Command{
	Use:       "completion [bash|zsh|fish]",
	Short:     "Generate a shell completion script",
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	ValidArgs: []string{"bash", "zsh", "fish"},
	Long: `Generate a completion script for your shell.

  zsh:   shrike completion zsh  > "${fpath[1]}/_evekill"
  bash:  shrike completion bash > /etc/bash_completion.d/evekill
  fish:  shrike completion fish > ~/.config/fish/completions/evekill.fish`,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		}
		return fmt.Errorf("unsupported shell %q", args[0])
	},
}
