package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for cadre.

To load completions:

Bash:
  $ source <(cadre completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ cadre completion bash > /etc/bash_completion.d/cadre
  # macOS:
  $ cadre completion bash > $(brew --prefix)/etc/bash_completion.d/cadre

Zsh:
  $ source <(cadre completion zsh)
  # To load completions for each session, execute once:
  $ cadre completion zsh > "${fpath[1]}/_cadre"

Fish:
  $ cadre completion fish | source
  # To load completions for each session, execute once:
  $ cadre completion fish > ~/.config/fish/completions/cadre.fish

PowerShell:
  PS> cadre completion powershell | Out-String | Invoke-Expression
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}
