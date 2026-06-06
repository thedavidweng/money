package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newCompletionCommand(stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for money.

To load completions:

Bash:

  $ source <(money completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ money completion bash > /etc/bash_completion.d/money
  # macOS:
  $ money completion bash > $(brew --prefix)/etc/bash_completion.d/money

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ money completion zsh > "${fpath[1]}/_money"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ money completion fish | source

  # To load completions for each session, execute once:
  $ money completion fish > ~/.config/fish/completions/money.fish

PowerShell:

  PS> money completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> money completion powershell > money.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(stdout, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(stdout)
			default:
				return fmt.Errorf("unsupported shell: %q", args[0])
			}
		},
	}
	return cmd
}
