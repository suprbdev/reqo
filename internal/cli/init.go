package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suprbdev/reqo/internal/project"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var global bool
	cmd := &cobra.Command{
		Use:   "init <project-name>",
		Short: "Create a new reqo project (local or global)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			var baseDir string
			if global {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				baseDir = filepath.Join(home, ".reqo", "projects", name)
			} else {
				cwd, _ := os.Getwd()
				baseDir = cwd
			}

			p := &project.Project{
				Version:    1,
				Name:       name,
				DefaultEnv: "default",
				Environments: map[string]project.Environment{
					"default": {BaseURL: "", Headers: []string{}},
				},
				HeaderSets: map[string][]string{
					"default": {"User-Agent: reqo/${version}"},
				},
				Calls: map[string]project.Call{},
			}
			if err := project.Save(baseDir, p); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Initialized project %s at %s\n", name, baseDir)
			return nil
		},
	}
	cmd.Flags().BoolVar(&global, "global", false, "store the project in ~/.reqo/projects")
	return cmd
}
