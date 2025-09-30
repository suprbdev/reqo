package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/suprbdev/reqo/internal/project"
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
			jsonBody, _ := cmd.Flags().GetString("json")
			rawBody, _ := cmd.Flags().GetString("data")
			formFields, _ := cmd.Flags().GetStringToString("form")

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

			// Add body specification if provided
			if jsonBody != "" || rawBody != "" || len(formFields) > 0 {
				bodySpec := &project.BodySpec{}
				if jsonBody != "" {
					bodySpec.JSON = &jsonBody
				}
				if rawBody != "" {
					bodySpec.Raw = &rawBody
				}
				if len(formFields) > 0 {
					bodySpec.Form = formFields
				}
				call.Body = bodySpec
			}

			p.Project.Calls[alias] = call
			return project.Save(p.Dir, p.Project)
		},
	}
	cmd.Flags().String("use-headers", "", "header set to apply")
	cmd.Flags().String("desc", "", "short description for the call")
	cmd.Flags().String("json", "", "JSON body or @file to save with the call")
	cmd.Flags().String("data", "", "raw body or @file to save with the call")
	cmd.Flags().StringToString("form", nil, "multipart form fields to save with the call (k=v, use @file for uploads)")
	return cmd
}
