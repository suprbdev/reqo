package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	httpx "github.com/suprbdev/reqo/internal/http"
	"github.com/suprbdev/reqo/internal/output"
	"github.com/suprbdev/reqo/internal/project"
	"github.com/suprbdev/reqo/internal/template"
)

// newCallCmd provides a unified interface to manage and execute saved calls.
func newCallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call",
		Short: "Manage and run saved calls (aliases)",
        Args:  cobra.ArbitraryArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            // If a known subcommand matched, Cobra won't call this.
            // If we are here and an argument is provided, treat it as an alias and run it.
            if len(args) == 0 {
                return cmd.Help()
            }
            alias := args[0]
            return executeCall(cmd, alias)
        },
	}

	// call create <alias> <method> <path>
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

	// call list
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

	// call rm <alias>
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

    // call run <alias>
    runCmd := &cobra.Command{
		Use:     "run <alias>",
		Aliases: []string{"exec"},
		Short:   "Execute a saved call with optional overrides",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
            alias := args[0]
            return executeCall(cmd, alias)
		},
	}
	runCmd.Flags().StringArray("header", nil, "extra header (Key: Value)")
	runCmd.Flags().StringArray("query", nil, "extra query param (k=v)")
	runCmd.Flags().String("env", "", "environment to use")
	runCmd.Flags().Bool("as-curl", false, "print equivalent curl command and exit")
	runCmd.Flags().BoolP("include", "i", false, "show response headers")
	runCmd.Flags().Bool("raw", false, "output raw body")
	runCmd.Flags().Int("timeout", 30, "request timeout in seconds")
	runCmd.Flags().Int("retries", 0, "number of retries on failure")
	runCmd.Flags().StringArray("var", nil, "variables for template expansion (key=value)")
	// data flags
	runCmd.Flags().String("json", "", "JSON body or @file")
	runCmd.Flags().String("data", "", "raw body or @file")
	runCmd.Flags().StringToString("form", nil, "multipart form fields (k=v, use @file for uploads)")
	cmd.AddCommand(runCmd)

    // Also add run-related flags to the parent command to support shorthand: `reqo call <alias> [flags]`
    // These are duplicated so that Cobra can parse them at the parent level.
    cmd.Flags().StringArray("header", nil, "extra header (Key: Value)")
    cmd.Flags().StringArray("query", nil, "extra query param (k=v)")
    cmd.Flags().String("env", "", "environment to use")
    cmd.Flags().Bool("as-curl", false, "print equivalent curl command and exit")
    cmd.Flags().BoolP("include", "i", false, "show response headers")
    cmd.Flags().Bool("raw", false, "output raw body")
    cmd.Flags().Int("timeout", 30, "request timeout in seconds")
    cmd.Flags().Int("retries", 0, "number of retries on failure")
    cmd.Flags().StringArray("var", nil, "variables for template expansion (key=value)")
    cmd.Flags().String("json", "", "JSON body or @file")
    cmd.Flags().String("data", "", "raw body or @file")
    cmd.Flags().StringToString("form", nil, "multipart form fields (k=v, use @file for uploads)")

	return cmd
}

// executeCall contains the logic to execute a saved call. It is used by both
// the `call run` subcommand and the parent `call` command when invoked as
// `reqo call <alias>`.
func executeCall(cmd *cobra.Command, alias string) error {
    pCtx, err := resolveProject(cmd)
    if err != nil {
        return err
    }
    callDef, ok := pCtx.Project.Calls[alias]
    if !ok {
        return fmt.Errorf("call %q not defined in project %s", alias, pCtx.Project.Name)
    }

    vars := map[string]string{}
    for _, v := range getStringArray(cmd, "var") {
        kv := strings.SplitN(v, "=", 2)
        if len(kv) == 2 {
            vars[kv[0]] = kv[1]
        }
    }

    // derive environment name: flag > REQO_ENV > project default (handled in BuildRequest)
    envName := getString(cmd, "env")
    if envName == "" {
        envName = os.Getenv("REQO_ENV")
    }

    spec := httpx.RequestSpec{
        Method:       callDef.Method,
        Path:         callDef.Path,
        QueryParams:  getStringArray(cmd, "query"),
        Headers:      getStringArray(cmd, "header"),
        UseHeaderSet: callDef.UseHeaderSet,
        Vars:         vars,
        EnvName:      envName,
    }

    if jsonBody := getString(cmd, "json"); jsonBody != "" {
        spec.JSONBody = &jsonBody
    } else if callDef.Body != nil && callDef.Body.JSON != nil {
        expandedJSON := template.Expand(*callDef.Body.JSON, vars)
        spec.JSONBody = &expandedJSON
    }

    if rawBody := getString(cmd, "data"); rawBody != "" {
        spec.RawBody = &rawBody
    } else if callDef.Body != nil && callDef.Body.Raw != nil {
        expandedRaw := template.Expand(*callDef.Body.Raw, vars)
        spec.RawBody = &expandedRaw
    }

    if formMap := getStringToString(cmd, "form"); len(formMap) > 0 {
        spec.FormFields = formMap
    } else if callDef.Body != nil && len(callDef.Body.Form) > 0 {
        expandedForm := make(map[string]string)
        for k, v := range callDef.Body.Form {
            expandedForm[k] = template.Expand(v, vars)
        }
        spec.FormFields = expandedForm
    }

    req, err := httpx.BuildRequest(pCtx.Project, spec)
    if err != nil {
        return err
    }

    if getBool(cmd, "as-curl") {
        curlCmd, _ := httpx.AsCurl(req)
        fmt.Fprintln(cmd.OutOrStdout(), curlCmd)
        return nil
    }

    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getInt(cmd, "timeout"))*time.Second)
    defer cancel()

    execOpts := httpx.ExecOpts{
        Timeout:      time.Duration(getInt(cmd, "timeout")) * time.Second,
        Retries:      getInt(cmd, "retries"),
        Backoff:      200 * time.Millisecond,
        MaxRedirects: 10,
    }
    resp, err := httpx.Execute(ctx, nil, req, execOpts)
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    renderOpts := output.RenderOpts{
        ShowHeaders: getBool(cmd, "include"),
        RawOutput:   getBool(cmd, "raw"),
        JQExpr:      "",
    }
    return output.Render(resp, cmd.OutOrStdout(), renderOpts)
}


