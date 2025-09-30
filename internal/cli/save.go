package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/suprbdev/reqo/internal/project"
)

func newSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save",
		Short: "Manage saved calls (aliases) in the current project",
	}

	// Create subcommand
	createCmd := &cobra.Command{
		Use:   "create <alias> <method> <path>",
		Short: "Create a saved call (alias)",
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
	createCmd.Flags().String("use-headers", "", "header set to apply")
	createCmd.Flags().String("desc", "", "short description for the call")
	createCmd.Flags().String("json", "", "JSON body or @file to save with the call")
	createCmd.Flags().String("data", "", "raw body or @file to save with the call")
	createCmd.Flags().StringToString("form", nil, "multipart form fields to save with the call (k=v, use @file for uploads)")
	cmd.AddCommand(createCmd)

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all saved calls",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			if len(p.Project.Calls) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No saved calls found.")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Saved Calls:")
			for alias, call := range p.Project.Calls {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s %s", alias, call.Method, call.Path)
				if call.Description != "" {
					fmt.Fprintf(cmd.OutOrStdout(), " (%s)", call.Description)
				}
				if call.UseHeaderSet != "" {
					fmt.Fprintf(cmd.OutOrStdout(), " [uses: %s]", call.UseHeaderSet)
				}
				if call.Body != nil {
					if call.Body.JSON != nil {
						fmt.Fprintf(cmd.OutOrStdout(), " [JSON body]")
					} else if call.Body.Raw != nil {
						fmt.Fprintf(cmd.OutOrStdout(), " [raw body]")
					} else if len(call.Body.Form) > 0 {
						fmt.Fprintf(cmd.OutOrStdout(), " [form body]")
					}
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd)

	// Remove subcommand
	rmCmd := &cobra.Command{
		Use:   "rm <alias>",
		Args:  cobra.ExactArgs(1),
		Short: "Remove a saved call",
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			if _, exists := p.Project.Calls[alias]; !exists {
				return fmt.Errorf("saved call %q not found", alias)
			}
			delete(p.Project.Calls, alias)
			return project.Save(p.Dir, p.Project)
		},
	}
	cmd.AddCommand(rmCmd)

	return cmd
}
