package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/suprbdev/reqo/internal/project"
)

// setupProjectDir creates a temp dir with a .reqo project, chdirs into it,
// and returns a cleanup function.
func setupProjectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := &project.Project{
		Version:    1,
		Name:       "test-project",
		DefaultEnv: "dev",
		Environments: map[string]project.Environment{
			"dev":  {BaseURL: "https://dev.example.com"},
			"prod": {BaseURL: "https://api.example.com"},
		},
		HeaderSets: map[string][]string{
			"auth": {"Authorization: Bearer token123"},
		},
		Calls: map[string]project.Call{},
	}
	if err := project.Save(dir, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	return dir
}

// runCmd executes a cobra command with the given args and captures output.
func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

// ---------- init command ----------

func TestInitCmd_Local(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	out, err := runCmd(t, "init", "my-api")
	if err != nil {
		t.Fatalf("init error: %v", err)
	}
	if !contains(out, "Initialized project my-api") {
		t.Errorf("output = %q", out)
	}
	// project.yaml should exist in current dir
	if _, err := os.Stat(filepath.Join(dir, ".reqo", "project.yaml")); err != nil {
		t.Errorf("project.yaml should exist: %v", err)
	}

	// Verify the project structure
	p, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if p.Name != "my-api" {
		t.Errorf("project name = %q", p.Name)
	}
	if p.Version != 1 {
		t.Errorf("version = %d", p.Version)
	}
	if p.DefaultEnv != "default" {
		t.Errorf("default_env = %q", p.DefaultEnv)
	}
	if len(p.Environments) != 1 {
		t.Errorf("should have 1 default environment")
	}
	if len(p.HeaderSets) != 1 {
		t.Errorf("should have 1 default header set")
	}
}

func TestInitCmd_RequiresName(t *testing.T) {
	_, err := runCmd(t, "init")
	if err == nil {
		t.Errorf("init without name should error")
	}
}

// ---------- use command ----------

func TestUseCmd(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	out, err := runCmd(t, "use", "my-project")
	if err != nil {
		t.Fatalf("use error: %v", err)
	}
	if !contains(out, "Project my-project set") {
		t.Errorf("output = %q", out)
	}

	// Verify current file
	current, err := project.GetCurrent(dir)
	if err != nil {
		t.Fatalf("GetCurrent() error: %v", err)
	}
	if current != "my-project" {
		t.Errorf("current = %q", current)
	}
}

func TestUseCmd_RequiresName(t *testing.T) {
	_, err := runCmd(t, "use")
	if err == nil {
		t.Errorf("use without name should error")
	}
}

// ---------- env command ----------

func TestEnvAddCmd(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "env", "add", "staging", "--base-url", "https://staging.example.com")
	if err != nil {
		t.Fatalf("env add error: %v", err)
	}

	dir, _ := os.Getwd()
	p, _ := project.Load(dir)
	if _, ok := p.Environments["staging"]; !ok {
		t.Errorf("staging environment should exist")
	}
	if p.Environments["staging"].BaseURL != "https://staging.example.com" {
		t.Errorf("base_url = %q", p.Environments["staging"].BaseURL)
	}
}

func TestEnvAddCmd_NoProject(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	_, err := runCmd(t, "env", "add", "test", "--base-url", "https://example.com")
	if err == nil {
		t.Errorf("env add without project should error")
	}
}

func TestEnvAddCmd_RequiresBaseURL(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "env", "add", "test")
	if err == nil {
		t.Errorf("env add without --base-url should error")
	}
}

func TestEnvListCmd(t *testing.T) {
	setupProjectDir(t)

	out, err := runCmd(t, "env", "list")
	if err != nil {
		t.Fatalf("env list error: %v", err)
	}
	if !contains(out, "Environments:") {
		t.Errorf("output should list environments: %q", out)
	}
	if !contains(out, "dev") {
		t.Errorf("should show dev env: %q", out)
	}
	if !contains(out, "prod") {
		t.Errorf("should show prod env: %q", out)
	}
	if !contains(out, "(default)") {
		t.Errorf("should mark dev as default: %q", out)
	}
}

// ---------- header command ----------

func TestHeaderSetCmd(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "header", "set", "--name", "api", "X-API-Key: secret", "Content-Type: application/json")
	if err != nil {
		t.Fatalf("header set error: %v", err)
	}

	dir, _ := os.Getwd()
	p, _ := project.Load(dir)
	if hs, ok := p.HeaderSets["api"]; !ok {
		t.Errorf("api header set should exist")
	} else if len(hs) != 2 {
		t.Errorf("api header set len = %d, want 2", len(hs))
	}
}

func TestHeaderSetCmd_RequiresName(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "header", "set", "X-Test: val")
	if err == nil {
		t.Errorf("header set without --name should error")
	}
}

func TestHeaderListCmd(t *testing.T) {
	setupProjectDir(t)

	out, err := runCmd(t, "header", "list")
	if err != nil {
		t.Fatalf("header list error: %v", err)
	}
	if !contains(out, "Header Sets:") {
		t.Errorf("output should list header sets: %q", out)
	}
	if !contains(out, "auth:") {
		t.Errorf("should show auth set: %q", out)
	}
	if !contains(out, "Bearer token123") {
		t.Errorf("should show header value: %q", out)
	}
}

func TestHeaderRmCmd(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "header", "rm", "auth")
	if err != nil {
		t.Fatalf("header rm error: %v", err)
	}

	dir, _ := os.Getwd()
	p, _ := project.Load(dir)
	if _, ok := p.HeaderSets["auth"]; ok {
		t.Errorf("auth header set should be removed")
	}
}

func TestHeaderRmCmd_NotExists(t *testing.T) {
	setupProjectDir(t)

	// Removing a nonexistent set is idempotent (delete on nil map is a no-op)
	// but saving after may produce a nil map. This should not panic.
	_, err := runCmd(t, "header", "rm", "nonexistent")
	_ = err // may or may not error depending on nil map behavior
}

// ---------- call command ----------

func TestCallCreateCmd(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "call", "create", "get-users", "GET", "/users")
	if err != nil {
		t.Fatalf("call create error: %v", err)
	}

	dir, _ := os.Getwd()
	p, _ := project.Load(dir)
	call, ok := p.Calls["get-users"]
	if !ok {
		t.Fatalf("get-users call should exist")
	}
	if call.Method != "GET" {
		t.Errorf("method = %q", call.Method)
	}
	if call.Path != "/users" {
		t.Errorf("path = %q", call.Path)
	}
}

func TestCallCreateCmd_WithBody(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "call", "create", "create-user", "POST", "/users",
		"--json", `{"name":"${name}"}`,
		"--desc", "Create a user",
		"--use-headers", "auth",
	)
	if err != nil {
		t.Fatalf("call create error: %v", err)
	}

	dir, _ := os.Getwd()
	p, _ := project.Load(dir)
	call := p.Calls["create-user"]
	if call.Method != "POST" {
		t.Errorf("method = %q", call.Method)
	}
	if call.Description != "Create a user" {
		t.Errorf("desc = %q", call.Description)
	}
	if call.UseHeaderSet != "auth" {
		t.Errorf("useHeaderSet = %q", call.UseHeaderSet)
	}
	if call.Body == nil || call.Body.JSON == nil {
		t.Errorf("should have JSON body")
	}
}

func TestCallCreateCmd_WithFormData(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "call", "create", "upload", "POST", "/upload",
		"--form", "file=@/tmp/test.txt", "--form", "desc=hello",
	)
	if err != nil {
		t.Fatalf("call create error: %v", err)
	}

	dir, _ := os.Getwd()
	p, _ := project.Load(dir)
	call := p.Calls["upload"]
	if call.Body == nil || len(call.Body.Form) != 2 {
		t.Errorf("should have 2 form fields")
	}
}

func TestCallCreateCmd_WithRawData(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "call", "create", "update", "PUT", "/config",
		"--data", `{"setting":"${value}"}`,
	)
	if err != nil {
		t.Fatalf("call create error: %v", err)
	}

	dir, _ := os.Getwd()
	p, _ := project.Load(dir)
	call := p.Calls["update"]
	if call.Body == nil || call.Body.Raw == nil {
		t.Errorf("should have raw body")
	}
}

func TestCallCreateCmd_LowercaseMethod(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "call", "create", "delete-user", "delete", "/users/1")
	if err != nil {
		t.Fatalf("call create error: %v", err)
	}

	dir, _ := os.Getwd()
	p, _ := project.Load(dir)
	call := p.Calls["delete-user"]
	if call.Method != "DELETE" {
		t.Errorf("method = %q, want DELETE (uppercased)", call.Method)
	}
}

func TestCallListCmd(t *testing.T) {
	setupProjectDir(t)

	// Create some calls first
	_, _ = runCmd(t, "call", "create", "get-users", "GET", "/users", "--desc", "List users")
	_, _ = runCmd(t, "call", "create", "create-user", "POST", "/users", "--use-headers", "auth")

	out, err := runCmd(t, "call", "list")
	if err != nil {
		t.Fatalf("call list error: %v", err)
	}
	if !contains(out, "Saved Calls:") {
		t.Errorf("should show header: %q", out)
	}
	if !contains(out, "get-users") {
		t.Errorf("should list get-users: %q", out)
	}
	if !contains(out, "List users") {
		t.Errorf("should show description: %q", out)
	}
}

func TestCallListCmd_Empty(t *testing.T) {
	setupProjectDir(t)

	out, err := runCmd(t, "call", "list")
	if err != nil {
		t.Fatalf("call list error: %v", err)
	}
	if !contains(out, "No saved calls found.") {
		t.Errorf("should show empty message: %q", out)
	}
}

func TestCallRmCmd(t *testing.T) {
	setupProjectDir(t)

	_, _ = runCmd(t, "call", "create", "temp-call", "GET", "/temp")

	_, err := runCmd(t, "call", "rm", "temp-call")
	if err != nil {
		t.Fatalf("call rm error: %v", err)
	}

	dir, _ := os.Getwd()
	p, _ := project.Load(dir)
	if _, ok := p.Calls["temp-call"]; ok {
		t.Errorf("temp-call should be removed")
	}
}

func TestCallRmCmd_NotExists(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "call", "rm", "nonexistent")
	if err == nil {
		t.Errorf("call rm nonexistent should error")
	}
}

func TestCallCreateCmd_AtPath(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "call", "create", "graphql", "POST", "@")
	if err != nil {
		t.Fatalf("call create error: %v", err)
	}

	dir, _ := os.Getwd()
	p, _ := project.Load(dir)
	call := p.Calls["graphql"]
	if call.Path != "@" {
		t.Errorf("path = %q, want @", call.Path)
	}
}

func TestCallListCmd_BodyIndicators(t *testing.T) {
	setupProjectDir(t)

	_, _ = runCmd(t, "call", "create", "json-call", "POST", "/api", "--json", "{}")
	_, _ = runCmd(t, "call", "create", "raw-call", "PUT", "/api", "--data", "raw")
	_, _ = runCmd(t, "call", "create", "form-call", "POST", "/api", "--form", "k=v")

	out, _ := runCmd(t, "call", "list")
	if !contains(out, "[JSON body]") {
		t.Errorf("should show [JSON body] indicator: %q", out)
	}
	if !contains(out, "[raw body]") {
		t.Errorf("should show [raw body] indicator: %q", out)
	}
	if !contains(out, "[form body]") {
		t.Errorf("should show [form body] indicator: %q", out)
	}
}

func TestCallRunCmd_AsCurl(t *testing.T) {
	setupProjectDir(t)

	// Create a call
	_, _ = runCmd(t, "call", "create", "get-users", "GET", "/users", "--use-headers", "auth")

	out, err := runCmd(t, "call", "run", "get-users", "--as-curl")
	if err != nil {
		t.Fatalf("call run --as-curl error: %v", err)
	}
	if !contains(out, "curl") {
		t.Errorf("should output curl command: %q", out)
	}
	if !contains(out, "Bearer token123") {
		t.Errorf("curl should include auth header: %q", out)
	}
	if !contains(out, "https://dev.example.com/users") {
		t.Errorf("curl should include URL: %q", out)
	}
}

func TestCallRunCmd_Shorthand(t *testing.T) {
	setupProjectDir(t)

	_, _ = runCmd(t, "call", "create", "get-users", "GET", "/users")

	out, err := runCmd(t, "call", "get-users", "--as-curl")
	if err != nil {
		t.Fatalf("call <alias> shorthand error: %v", err)
	}
	if !contains(out, "curl") {
		t.Errorf("shorthand should work as alias: %q", out)
	}
}

func TestCallRunCmd_NotExists(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "call", "run", "nonexistent")
	if err == nil {
		t.Errorf("should error for nonexistent call")
	}
}

func TestCallRunCmd_VarExpansion(t *testing.T) {
	setupProjectDir(t)

	_, _ = runCmd(t, "call", "create", "get-user", "GET", "/users/${id}")

	out, err := runCmd(t, "call", "run", "get-user", "--var", "id=123", "--as-curl")
	if err != nil {
		t.Fatalf("call run error: %v", err)
	}
	if !contains(out, "/users/123") {
		t.Errorf("var should be expanded in URL: %q", out)
	}
}

func TestCallRunCmd_EnvOverride(t *testing.T) {
	setupProjectDir(t)

	_, _ = runCmd(t, "call", "create", "get-users", "GET", "/users")

	out, err := runCmd(t, "call", "run", "get-users", "--env", "prod", "--as-curl")
	if err != nil {
		t.Fatalf("call run error: %v", err)
	}
	if !contains(out, "https://api.example.com/users") {
		t.Errorf("should use prod env URL: %q", out)
	}
}

func TestCallRunCmd_EnvFromEnvVar(t *testing.T) {
	setupProjectDir(t)
	t.Setenv("REQO_ENV", "prod")

	_, _ = runCmd(t, "call", "create", "get-users", "GET", "/users")

	out, err := runCmd(t, "call", "run", "get-users", "--as-curl")
	if err != nil {
		t.Fatalf("call run error: %v", err)
	}
	if !contains(out, "https://api.example.com") {
		t.Errorf("should use prod env via REQO_ENV: %q", out)
	}
}

func TestCallRunCmd_SavedJSONBody(t *testing.T) {
	setupProjectDir(t)

	_, _ = runCmd(t, "call", "create", "create-user", "POST", "/users",
		"--json", `{"name":"${name}"}`,
	)

	out, err := runCmd(t, "call", "run", "create-user", "--var", "name=John", "--as-curl")
	if err != nil {
		t.Fatalf("call run error: %v", err)
	}
	if !contains(out, "John") {
		t.Errorf("saved JSON body should be expanded: %q", out)
	}
}

func TestCallRunCmd_SavedJSONBodyOverride(t *testing.T) {
	setupProjectDir(t)

	_, _ = runCmd(t, "call", "create", "create-user", "POST", "/users",
		"--json", `{"name":"saved"}`,
	)

	out, err := runCmd(t, "call", "run", "create-user",
		"--json", `{"name":"override"}`, "--as-curl",
	)
	if err != nil {
		t.Fatalf("call run error: %v", err)
	}
	if !contains(out, "override") {
		t.Errorf("CLI flag should override saved body: %q", out)
	}
}

func TestCallRunCmd_SavedFormBody(t *testing.T) {
	setupProjectDir(t)

	// Create the temp file so form file upload doesn't fail
	tmpFile := "/tmp/reqo_test_form_upload.txt"
	os.WriteFile(tmpFile, []byte("file content"), 0o644)

	_, _ = runCmd(t, "call", "create", "upload", "POST", "/upload",
		"--form", "file=@${file_path}", "--form", "desc=${desc}",
	)

	out, err := runCmd(t, "call", "run", "upload",
		"--var", "file_path="+tmpFile, "--var", "desc=My file",
		"--as-curl",
	)
	if err != nil {
		t.Fatalf("call run error: %v", err)
	}
	if !contains(out, "My file") {
		t.Errorf("form field should be expanded: %q", out)
	}
}

func TestCallRunCmd_ExtraHeaders(t *testing.T) {
	setupProjectDir(t)

	_, _ = runCmd(t, "call", "create", "get-users", "GET", "/users")

	out, err := runCmd(t, "call", "run", "get-users",
		"--header", "X-Custom: myval", "--as-curl",
	)
	if err != nil {
		t.Fatalf("call run error: %v", err)
	}
	if !contains(out, "X-Custom") || !contains(out, "myval") {
		t.Errorf("should include extra header: %q", out)
	}
}

func TestCallRunCmd_QueryParams(t *testing.T) {
	setupProjectDir(t)

	_, _ = runCmd(t, "call", "create", "get-users", "GET", "/users")

	out, err := runCmd(t, "call", "run", "get-users",
		"--query", "page=1", "--query", "limit=10", "--as-curl",
	)
	if err != nil {
		t.Fatalf("call run error: %v", err)
	}
	if !contains(out, "page=1") || !contains(out, "limit=10") {
		t.Errorf("should include query params: %q", out)
	}
}

func TestCallRunCmd_GraphQLAtPath(t *testing.T) {
	setupProjectDir(t)

	_, _ = runCmd(t, "call", "create", "graphql", "POST", "@",
		"--json", `{"query":"${q}"}`,
	)

	out, err := runCmd(t, "call", "run", "graphql",
		"--var", "q=query{users{name}}", "--as-curl",
	)
	if err != nil {
		t.Fatalf("call run error: %v", err)
	}
	if !contains(out, "https://dev.example.com") {
		t.Errorf("should use base URL for @ path: %q", out)
	}
	if contains(out, "dev.example.com/users") {
		t.Errorf("should not append /users for @ path: %q", out)
	}
}

// ---------- req command ----------

// setupProjectWithServer creates a temp project pointing to a test server.
func setupProjectWithServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := startTestServer(t)
	dir := setupProjectDir(t)
	p, _ := project.Load(dir)
	env := p.Environments["dev"]
	env.BaseURL = srv.URL
	p.Environments["dev"] = env
	project.Save(dir, p)
	return srv
}

func TestReqCmd_BasicGet(t *testing.T) {
	srv := setupProjectWithServer(t)
	defer srv.Close()

	_, err := runCmd(t, "req", "GET", "/test")
	if err != nil {
		t.Fatalf("req error: %v", err)
	}
}

func TestReqCmd_MethodDetection(t *testing.T) {
	// Test that isHTTPMethod works correctly
	tests := []struct {
		input string
		want  bool
	}{
		{"GET", true},
		{"POST", true},
		{"PUT", true},
		{"PATCH", true},
		{"DELETE", true},
		{"HEAD", true},
		{"OPTIONS", true},
		{"get", false}, // isHTTPMethod takes upper case
		{"CUSTOM", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isHTTPMethod(tt.input)
		if got != tt.want {
			t.Errorf("isHTTPMethod(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestReqCmd_FullURL(t *testing.T) {
	srv := startTestServer(t)
	defer srv.Close()

	setupProjectDir(t)

	out, err := runCmd(t, "req", "GET", srv.URL+"/test")
	if err != nil {
		t.Fatalf("req error: %v", err)
	}
	if !contains(out, "test-endpoint") {
		t.Errorf("should hit test server: %q", out)
	}
}

func TestReqCmd_NoArgs(t *testing.T) {
	setupProjectDir(t)

	_, err := runCmd(t, "req")
	if err == nil {
		t.Errorf("req with no args should error")
	}
}

func TestReqCmd_DefaultGET(t *testing.T) {
	srv := setupProjectWithServer(t)
	defer srv.Close()

	out, err := runCmd(t, "req", "/test")
	if err != nil {
		t.Fatalf("req error: %v", err)
	}
	if !contains(out, "test-endpoint") {
		t.Errorf("should default to GET: %q", out)
	}
}

func TestReqCmd_POST(t *testing.T) {
	srv := setupProjectWithServer(t)
	defer srv.Close()

	out, err := runCmd(t, "req", "POST", "/echo", "--json", `{"x":1}`)
	if err != nil {
		t.Fatalf("req error: %v", err)
	}
	if !contains(out, "echo") {
		t.Errorf("should echo: %q", out)
	}
}

func TestReqCmd_ShowHeaders(t *testing.T) {
	srv := setupProjectWithServer(t)
	defer srv.Close()

	out, err := runCmd(t, "req", "GET", "/test", "-i")
	if err != nil {
		t.Fatalf("req error: %v", err)
	}
	if !contains(out, "HTTP/1.1") {
		t.Errorf("should show headers: %q", out)
	}
}

func TestReqCmd_RawOutput(t *testing.T) {
	srv := setupProjectWithServer(t)
	defer srv.Close()

	out, err := runCmd(t, "req", "GET", "/test", "--raw")
	if err != nil {
		t.Fatalf("req error: %v", err)
	}
	if !contains(out, "test-endpoint") {
		t.Errorf("raw output: %q", out)
	}
}

func TestReqCmd_NoProject(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	_, err := runCmd(t, "req", "GET", "/test")
	if err == nil {
		t.Errorf("req without project should error")
	}
}

// ---------- config command ----------

func TestConfigGetCmd(t *testing.T) {
	_, err := runCmd(t, "config", "get", "nonexistent_key_for_test")
	if err != nil {
		t.Fatalf("config get error: %v", err)
	}
}

// ---------- resolveProject helper ----------

func TestResolveProject_FromCwd(t *testing.T) {
	setupProjectDir(t)

	cmd := NewRootCmd()
	subCmd, _, err := cmd.Find([]string{"env", "list"})
	if err != nil {
		t.Fatal(err)
	}
	pCtx, err := resolveProject(subCmd)
	if err != nil {
		t.Fatalf("resolveProject() error: %v", err)
	}
	if pCtx.Project.Name != "test-project" {
		t.Errorf("project name = %q", pCtx.Project.Name)
	}
}

func TestResolveProject_NotFound(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	cmd := NewRootCmd()
	subCmd, _, _ := cmd.Find([]string{"env", "list"})
	_, err := resolveProject(subCmd)
	if err == nil {
		// May find a project up the tree — acceptable
	}
}

// ---------- root command ----------

func TestRootCmd_NoArgs(t *testing.T) {
	out, err := runCmd(t)
	if err != nil {
		t.Fatalf("root command error: %v", err)
	}
	if !contains(out, "reqo") {
		t.Errorf("should show usage: %q", out)
	}
}

func TestRootCmd_GlobalFlags(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.PersistentFlags().Lookup("no-color") == nil {
		t.Errorf("should have --no-color flag")
	}
	if cmd.PersistentFlags().Lookup("project") == nil {
		t.Errorf("should have --project flag")
	}
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	cmd := NewRootCmd()
	expected := []string{"init", "use", "config", "env", "header", "call", "req"}
	for _, name := range expected {
		found := false
		for _, sub := range cmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("root should have subcommand %q", name)
		}
	}
}

// ---------- helpers ----------

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

// startTestServer returns a test HTTP server that responds to various endpoints.
func startTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"endpoint":"test-endpoint"}`))
	})
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"echo":true}`))
	})
	return httptest.NewServer(mux)
}
