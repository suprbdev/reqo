package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	httpx "github.com/suprbdev/reqo/internal/http"
	"github.com/suprbdev/reqo/internal/output"
	"github.com/suprbdev/reqo/internal/project"
	"github.com/spf13/cobra"
)

func newReqCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "req <method|path> [path]",
		Short: "Perform an ad‑hoc HTTP request (no saved call)",
		Long: `If the first argument looks like an HTTP method it is used,
otherwise GET is assumed and the first argument is treated as the path.`,
		RunE: runReq,
	}
	cmd.Flags().StringArray("header", nil, "extra header")
	cmd.Flags().StringArray("query", nil, "extra query param (k=v)")
	cmd.Flags().String("env", "", "environment to use")
	cmd.Flags().BoolP("include", "i", false, "show response headers")
	cmd.Flags().Bool("raw", false, "output raw body")
	cmd.Flags().Int("timeout", 30, "seconds")
	cmd.Flags().Int("retries", 0, "retry count")
	cmd.Flags().StringToString("form", nil, "multipart form fields")
	cmd.Flags().String("json", "", "JSON body or @file")
	cmd.Flags().String("data", "", "raw body or @file")
	cmd.Flags().StringArray("var", nil, "template variables (key=value)")
	return cmd
}

func runReq(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("path or method required")
	}
	var method, path string
	if isHTTPMethod(strings.ToUpper(args[0])) && len(args) > 1 {
		method = strings.ToUpper(args[0])
		path = args[1]
	} else {
		method = "GET"
		path = args[0]
	}

	// Gather vars
	vars := map[string]string{}
	for _, v := range getStringArray(cmd, "var") {
		kv := strings.SplitN(v, "=", 2)
		if len(kv) == 2 {
			vars[kv[0]] = kv[1]
		}
	}

	pCtx, err := resolveProject(cmd)
	if err != nil {
		return err
	}

	spec := httpx.RequestSpec{
		Method:      method,
		Path:        path,
		QueryParams: getStringArray(cmd, "query"),
		Headers:     getStringArray(cmd, "header"),
		Vars:        vars,
		EnvName:     getString(cmd, "env"),
	}
	if jsonBody := getString(cmd, "json"); jsonBody != "" {
		spec.JSONBody = &jsonBody
	}
	if rawBody := getString(cmd, "data"); rawBody != "" {
		spec.RawBody = &rawBody
	}
	if fm := getStringToString(cmd, "form"); len(fm) > 0 {
		spec.FormFields = fm
	}

	req, err := httpx.BuildRequest(pCtx.Project, spec)
	if err != nil {
		return err
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
	}
	return output.Render(resp, cmd.OutOrStdout(), renderOpts)
}

// utility
func isHTTPMethod(s string) bool {
	switch s {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}

// helper to resolve the active project (current dir or overridden via flag)
type projContext struct {
	Dir          string
	Project      *project.Project
	Environments map[string]project.Environment
}

func resolveProject(cmd *cobra.Command) (*projContext, error) {
	// 1️⃣ if user forced a project name via --project flag:
	if projName, _ := cmd.Flags().GetString("project"); projName != "" {
		home, _ := os.UserHomeDir()
		dir := fmt.Sprintf("%s/.reqo/projects/%s", home, projName)
		p, err := project.Load(dir)
		return &projContext{Dir: dir, Project: p}, err
	}
	// 2️⃣ otherwise walk up from cwd:
	cwd, _ := os.Getwd()
	dir, err := project.FindProject(cwd)
	if err != nil {
		return nil, err
	}
	p, err := project.Load(dir)
	return &projContext{Dir: dir, Project: p}, err
}

// small flag getters (avoid repetition)
func getString(cmd *cobra.Command, name string) string {
	s, _ := cmd.Flags().GetString(name)
	return s
}
func getInt(cmd *cobra.Command, name string) int {
	i, _ := cmd.Flags().GetInt(name)
	return i
}
func getBool(cmd *cobra.Command, name string) bool {
	b, _ := cmd.Flags().GetBool(name)
	return b
}
func getStringArray(cmd *cobra.Command, name string) []string {
	sa, _ := cmd.Flags().GetStringArray(name)
	return sa
}
func getStringToString(cmd *cobra.Command, name string) map[string]string {
	m, _ := cmd.Flags().GetStringToString(name)
	return m
}
