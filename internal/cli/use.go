package cli

import (
	"fmt"
	"os"

	"github.com/suprbdev/reqo/internal/project"
	"github.com/spf13/cobra"
)

func newUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <project-name>",
		Short: "Mark a project as active for the current directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cwd, _ := os.Getwd()
			if err := project.SetCurrent(cwd, name); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Project %s set for %s\n", name, cwd)
			return nil
		},
	}
	return cmd
}
