package cli

import (
	"fmt"

	"github.com/suprbdev/reqo/internal/project"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environments inside the current project",
	}
	addCmd := &cobra.Command{
		Use:   "add <name> --base-url <url>",
		Args:  cobra.ExactArgs(1),
		Short: "Add a new environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			baseURL, _ := cmd.Flags().GetString("base-url")
			if baseURL == "" {
				return fmt.Errorf("--base-url is required")
			}
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			if p.Project.Environments == nil {
				p.Project.Environments = map[string]project.Environment{}
			}
			p.Project.Environments[name] = project.Environment{BaseURL: baseURL}
			return project.Save(p.Dir, p.Project) // helper defined later
		},
	}
	addCmd.Flags().String("base-url", "", "Base URL for this environment")
	cmd.AddCommand(addCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List environments of the current project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Environments:")
			for n, e := range p.Project.Environments {
				def := ""
				if n == p.Project.DefaultEnv {
					def = "(default)"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %s %s → %s\n", n, def, e.BaseURL)
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd)

	return cmd
}
