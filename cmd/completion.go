package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:                   "completion [bash|zsh|fish|powershell]",
	Short:                 "Generate a completion script for common shells",
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	DisableFlagsInUseLine: true,
	Hidden:                true,
	Args:                  cobra.ExactValidArgs(1),
	Long: `To load completions:

Bash:

$ source <(ssbak completion bash)

# To load completions for each session, execute once:
Linux:
  $ ssbak completion bash > /etc/bash_completion.d/ssbak
MacOS:
  $ ssbak completion bash > /usr/local/etc/bash_completion.d/ssbak

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ ssbak completion zsh > "${fpath[1]}/_ssbak"

# You will need to start a new shell for this setup to take effect.

Fish:

$ ssbak completion fish | source

# To load completions for each session, execute once:
$ ssbak completion fish > ~/.config/fish/completions/ssbak.fish`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		switch args[0] {
		case "bash":
			err = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			err = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			err = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			err = cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
		if err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
