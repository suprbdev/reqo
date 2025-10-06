package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewRootCmd builds the top‑level command and registers sub‑commands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "reqo",
		Short: "reqo – curl, but with projects & saved calls",
		Long: `reqo is a friendly HTTP client that groups requests into
projects, supports environments, reusable header sets and saved aliases.
Run 'reqo <command> --help' for details.`,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), cmd.UsageString())
		},
		SilenceErrors: false,
		SilenceUsage:  false,
	}

	// Global flags
	root.PersistentFlags().BoolP("no-color", "", false, "disable colour output")
	root.PersistentFlags().StringP("project", "p", "", "override active project")

	// Register sub‑commands
	root.AddCommand(
		newInitCmd(),
		newUseCmd(),
		newConfigCmd(),
		newEnvCmd(),
		newHeaderCmd(),
		newCallCmd(),
		newReqCmd(),
	)

	return root
}
