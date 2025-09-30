package cli

import (
	"fmt"

	"github.com/suprbdev/reqo/internal/project"
	"github.com/spf13/cobra"
)

func newHeaderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "header",
		Short: "Manage reusable header sets in the current project",
	}
	setCmd := &cobra.Command{
		Use:   "set --name <set> \"Header: Value\"",
		Args:  cobra.MinimumNArgs(1),
		Short: "Replace a header set (or create new)",
		RunE: func(cmd *cobra.Command, args []string) error {
			setName, _ := cmd.Flags().GetString("name")
			if setName == "" {
				return fmt.Errorf("--name is required")
			}
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			if p.Project.HeaderSets == nil {
				p.Project.HeaderSets = map[string][]string{}
			}
			p.Project.HeaderSets[setName] = args
			return project.Save(p.Dir, p.Project)
		},
	}
	setCmd.Flags().String("name", "", "header set name")
	cmd.AddCommand(setCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List defined header sets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Header Sets:")
			for n, hdrs := range p.Project.HeaderSets {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s:\n", n)
				for _, h := range hdrs {
					fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", h)
				}
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd)

	rmCmd := &cobra.Command{
		Use:   "rm <set>",
		Args:  cobra.ExactArgs(1),
		Short: "Remove a header set",
		RunE: func(cmd *cobra.Command, args []string) error {
			setName := args[0]
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			delete(p.Project.HeaderSets, setName)
			return project.Save(p.Dir, p.Project)
		},
	}
	cmd.AddCommand(rmCmd)

	return cmd
}
