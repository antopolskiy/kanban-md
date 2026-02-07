package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate a shell completion script for the specified shell.

To load completions:

  bash:
    source <(kanban-md completion bash)

  zsh:
    echo "autoload -U compinit; compinit" >> ~/.zshrc
    kanban-md completion zsh > "${fpath[1]}/_kanban-md"

  fish:
    kanban-md completion fish | source
    kanban-md completion fish > ~/.config/fish/completions/kanban-md.fish

  PowerShell:
    kanban-md completion powershell | Out-String | Invoke-Expression`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE:                  runCompletion,
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

func runCompletion(cmd *cobra.Command, args []string) error {
	// Use the actual binary name so "kbmd completion zsh" generates
	// completions registered for "kbmd", not "kanban-md".
	cmd.Root().Use = filepath.Base(os.Args[0])

	switch args[0] {
	case "bash":
		return cmd.Root().GenBashCompletionV2(cmd.OutOrStdout(), true)
	case "zsh":
		return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
	case "fish":
		return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
	default:
		return nil
	}
}
