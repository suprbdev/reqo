package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	httpx "github.com/suprbdev/reqo/internal/http"
	"github.com/suprbdev/reqo/internal/output"
	"github.com/suprbdev/reqo/internal/template"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run <alias>",
		Aliases: []string{"call"},
		Short:   "Execute a saved call (or alias) with optional overrides",
		Args:    cobra.ExactArgs(1),
		RunE:    runAlias,
	}
	// Flags that can override parts of the saved definition
	cmd.Flags().StringArray("header", nil, "extra header (Key: Value)")
	cmd.Flags().StringArray("query", nil, "extra query param (k=v)")
	cmd.Flags().String("env", "", "environment to use")
	cmd.Flags().Bool("as-curl", false, "print equivalent curl command and exit")
	cmd.Flags().BoolP("include", "i", false, "show response headers")
	cmd.Flags().Bool("raw", false, "output raw body")
	cmd.Flags().Int("timeout", 30, "request timeout in seconds")
	cmd.Flags().Int("retries", 0, "number of retries on failure")
	cmd.Flags().StringArray("var", nil, "variables for template expansion (key=value)")

	// data flags
	cmd.Flags().String("json", "", "JSON body or @file")
	cmd.Flags().String("data", "", "raw body or @file")
	cmd.Flags().StringToString("form", nil, "multipart form fields (k=v, use @file for uploads)")

	return cmd
}

func runAlias(cmd *cobra.Command, args []string) error {
	alias := args[0]
	pCtx, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	callDef, ok := pCtx.Project.Calls[alias]
	if !ok {
		return fmt.Errorf("call %q not defined in project %s", alias, pCtx.Project.Name)
	}

	// Gather overrides from flags
	vars := map[string]string{}
	for _, v := range getStringArray(cmd, "var") {
		kv := strings.SplitN(v, "=", 2)
		if len(kv) == 2 {
			vars[kv[0]] = kv[1]
		}
	}

	spec := httpx.RequestSpec{
		Method:       callDef.Method,
		Path:         callDef.Path,
		QueryParams:  getStringArray(cmd, "query"),
		Headers:      getStringArray(cmd, "header"),
		UseHeaderSet: callDef.UseHeaderSet,
		Vars:         vars,
		EnvName:      getString(cmd, "env"),
	}

	// Handle body from command line flags (override saved body)
	if jsonBody := getString(cmd, "json"); jsonBody != "" {
		spec.JSONBody = &jsonBody
	} else if callDef.Body != nil && callDef.Body.JSON != nil {
		// Use saved JSON body with variable expansion
		expandedJSON := template.Expand(*callDef.Body.JSON, vars)
		spec.JSONBody = &expandedJSON
	}

	if rawBody := getString(cmd, "data"); rawBody != "" {
		spec.RawBody = &rawBody
	} else if callDef.Body != nil && callDef.Body.Raw != nil {
		// Use saved raw body with variable expansion
		expandedRaw := template.Expand(*callDef.Body.Raw, vars)
		spec.RawBody = &expandedRaw
	}

	if formMap := getStringToString(cmd, "form"); len(formMap) > 0 {
		spec.FormFields = formMap
	} else if callDef.Body != nil && len(callDef.Body.Form) > 0 {
		// Use saved form fields with variable expansion
		expandedForm := make(map[string]string)
		for k, v := range callDef.Body.Form {
			expandedForm[k] = template.Expand(v, vars)
		}
		spec.FormFields = expandedForm
	}

	// Build request
	req, err := httpx.BuildRequest(pCtx.Project, spec)
	if err != nil {
		return err
	}

	// --as-curl handling
	if getBool(cmd, "as-curl") {
		curlCmd, _ := httpx.AsCurl(req)
		fmt.Fprintln(cmd.OutOrStdout(), curlCmd)
		return nil
	}

	// Execute with retry/timeout options
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
		JQExpr:      "", // not exposed here – could be added later
	}
	return output.Render(resp, cmd.OutOrStdout(), renderOpts)
}
