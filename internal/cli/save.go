package cli

import (
	"strings"

	"github.com/suprbdev/reqo/internal/project"
	"github.com/spf13/cobra"
)

func newSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save <alias> <method> <path>",
		Short: "Create a saved call (alias) in the current project",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias, method, path := args[0], strings.ToUpper(args[1]), args[2]
			useHeaderSet, _ := cmd.Flags().GetString("use-headers")
			desc, _ := cmd.Flags().GetString("desc")

			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			if p.Project.Calls == nil {
				p.Project.Calls = map[string]project.Call{}
			}
			call := project.Call{
				Method:       method,
				Path:         path,
				UseHeaderSet: useHeaderSet,
				Description:  desc,
			}
			p.Project.Calls[alias] = call
			return project.Save(p.Dir, p.Project)
		},
	}
	cmd.Flags().String("use-headers", "", "header set to apply")
	cmd.Flags().String("desc", "", "short description for the call")
	return cmd
}
