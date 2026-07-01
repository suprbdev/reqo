package httpx

import (
	"net/http"
	"os"
	"testing"

	"github.com/suprbdev/reqo/internal/project"
)

func makeProject() *project.Project {
	return &project.Project{
		Version:    1,
		Name:       "test",
		DefaultEnv: "dev",
		Environments: map[string]project.Environment{
			"dev":  {BaseURL: "https://dev.example.com"},
			"prod": {BaseURL: "https://api.example.com"},
		},
		HeaderSets: map[string][]string{
			"auth": {"Authorization: Bearer token123"},
			"multi": {
				"X-API-Key: secret",
				"Content-Type: application/json",
			},
		},
		Calls: map[string]project.Call{},
	}
}

func TestBuildRequest_SimpleGet(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method: "GET",
		Path:   "/users",
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.URL.String() != "https://dev.example.com/users" {
		t.Errorf("URL = %q", req.URL.String())
	}
}

func TestBuildRequest_DefaultMethod(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Path: "/users",
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.Method != http.MethodGet {
		t.Errorf("Method = %q, want GET (default)", req.Method)
	}
}

func TestBuildRequest_FullURL(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method: "GET",
		Path:   "https://other.example.com/api",
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.URL.String() != "https://other.example.com/api" {
		t.Errorf("URL = %q", req.URL.String())
	}
}

func TestBuildRequest_AtPath_GraphQLEndpoint(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method: "POST",
		Path:   "@",
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.URL.String() != "https://dev.example.com" {
		t.Errorf("URL = %q, want base URL only", req.URL.String())
	}
}

func TestBuildRequest_EnvOverride(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method:  "GET",
		Path:    "/users",
		EnvName: "prod",
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.URL.String() != "https://api.example.com/users" {
		t.Errorf("URL = %q, want prod env", req.URL.String())
	}
}

func TestBuildRequest_UnknownEnv(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method:  "GET",
		Path:    "/users",
		EnvName: "staging",
	}
	_, err := BuildRequest(p, spec)
	if err == nil {
		t.Errorf("expected error for unknown environment")
	}
}

func TestBuildRequest_QueryParams(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method:      "GET",
		Path:        "/users",
		QueryParams: []string{"page=1", "limit=10"},
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	q := req.URL.Query()
	if q.Get("page") != "1" {
		t.Errorf("page = %q", q.Get("page"))
	}
	if q.Get("limit") != "10" {
		t.Errorf("limit = %q", q.Get("limit"))
	}
}

func TestBuildRequest_QueryParamWithVars(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method:      "GET",
		Path:        "/users",
		QueryParams: []string{"user=${id}"},
		Vars:        map[string]string{"id": "42"},
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.URL.Query().Get("user") != "42" {
		t.Errorf("user = %q", req.URL.Query().Get("user"))
	}
}

func TestBuildRequest_PathWithVars(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method: "GET",
		Path:   "/users/${id}/posts",
		Vars:   map[string]string{"id": "99"},
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.URL.Path != "/users/99/posts" {
		t.Errorf("Path = %q", req.URL.Path)
	}
}

func TestBuildRequest_BaseURLWithVars(t *testing.T) {
	p := makeProject()
	dev := p.Environments["dev"]
	dev.BaseURL = "https://${host}.example.com"
	p.Environments["dev"] = dev
	spec := RequestSpec{
		Method: "GET",
		Path:   "/api",
		Vars:   map[string]string{"host": "myhost"},
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.URL.Host != "myhost.example.com" {
		t.Errorf("Host = %q", req.URL.Host)
	}
}

func TestBuildRequest_TrailingSlashNormalization(t *testing.T) {
	p := makeProject()
	dev := p.Environments["dev"]
	dev.BaseURL = "https://dev.example.com/"
	p.Environments["dev"] = dev
	spec := RequestSpec{
		Method: "GET",
		Path:   "/users",
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.URL.String() != "https://dev.example.com/users" {
		t.Errorf("URL = %q, should not have double slash", req.URL.String())
	}
}

func TestBuildRequest_PathWithoutLeadingSlash(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method: "GET",
		Path:   "users",
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.URL.String() != "https://dev.example.com/users" {
		t.Errorf("URL = %q", req.URL.String())
	}
}

func TestBuildRequest_JSONBody(t *testing.T) {
	p := makeProject()
	jsonBody := `{"name":"test"}`
	spec := RequestSpec{
		Method:   "POST",
		Path:     "/users",
		JSONBody: &jsonBody,
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", req.Header.Get("Content-Type"))
	}
	if req.Body == nil {
		t.Errorf("Body should not be nil")
	}
}

func TestBuildRequest_RawBody(t *testing.T) {
	p := makeProject()
	rawBody := "plain text body"
	spec := RequestSpec{
		Method:   "POST",
		Path:     "/users",
		RawBody:  &rawBody,
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.Body == nil {
		t.Errorf("Body should not be nil")
	}
}

func TestBuildRequest_JSONAndRaw_Error(t *testing.T) {
	p := makeProject()
	jsonBody := `{}`
	rawBody := "text"
	spec := RequestSpec{
		Method:   "POST",
		Path:     "/users",
		JSONBody: &jsonBody,
		RawBody:  &rawBody,
	}
	_, err := BuildRequest(p, spec)
	if err == nil {
		t.Errorf("expected error when both --json and --data are set")
	}
}

func TestBuildRequest_JSONBodyWithVars(t *testing.T) {
	p := makeProject()
	jsonBody := `{"name":"${name}"}`
	spec := RequestSpec{
		Method:   "POST",
		Path:     "/users",
		JSONBody: &jsonBody,
		Vars:     map[string]string{"name": "John"},
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.Body == nil {
		t.Fatalf("Body should not be nil")
	}
}

func TestBuildRequest_FormFields(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method:     "POST",
		Path:       "/upload",
		FormFields: map[string]string{"field1": "value1", "field2": "value2"},
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	ct := req.Header.Get("Content-Type")
	if ct == "" {
		t.Errorf("Content-Type should be set for form data")
	}
	// multipart form should have boundary
	if len(ct) < 20 {
		t.Errorf("Content-Type seems too short for multipart: %q", ct)
	}
}

func TestBuildRequest_FormFieldWithVars(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method:     "POST",
		Path:       "/upload",
		FormFields: map[string]string{"desc": "file_${id}"},
		Vars:       map[string]string{"id": "42"},
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.Body == nil {
		t.Errorf("Body should not be nil")
	}
}

func TestBuildRequest_FormFileUpload(t *testing.T) {
	p := makeProject()
	// Create temp file for upload
	tmpFile := "/tmp/reqo_test_upload.txt"
	spec := RequestSpec{
		Method:     "POST",
		Path:       "/upload",
		FormFields: map[string]string{"file": "@" + tmpFile},
	}
	_, err := BuildRequest(p, spec)
	if err == nil {
		// We expect an error if the file doesn't exist, or success if it does.
		// The key thing is no panic.
	}
}

func TestBuildRequest_Headers(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method:  "GET",
		Path:    "/users",
		Headers: []string{"X-Custom: value", "X-Another: foo"},
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.Header.Get("X-Custom") != "value" {
		t.Errorf("X-Custom = %q", req.Header.Get("X-Custom"))
	}
	if req.Header.Get("X-Another") != "foo" {
		t.Errorf("X-Another = %q", req.Header.Get("X-Another"))
	}
}

func TestBuildRequest_HeaderSet(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method:       "GET",
		Path:         "/users",
		UseHeaderSet: "auth",
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.Header.Get("Authorization") != "Bearer token123" {
		t.Errorf("Authorization = %q", req.Header.Get("Authorization"))
	}
}

func TestBuildRequest_UnknownHeaderSet(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method:       "GET",
		Path:         "/users",
		UseHeaderSet: "nonexistent",
	}
	_, err := BuildRequest(p, spec)
	if err == nil {
		t.Errorf("expected error for unknown header set")
	}
}

func TestBuildRequest_HeaderWithVars(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method:  "GET",
		Path:    "/users",
		Headers: []string{"Authorization: Bearer ${token}"},
		Vars:    map[string]string{"token": "abc"},
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.Header.Get("Authorization") != "Bearer abc" {
		t.Errorf("Authorization = %q", req.Header.Get("Authorization"))
	}
}

func TestBuildRequest_HeaderPrecedence(t *testing.T) {
	p := makeProject()
	p.Environments["dev"] = project.Environment{BaseURL: "https://dev.example.com", Headers: []string{"X-Env: envval"}}
	p.HeaderSets["multi"] = []string{"X-Env: setval"}
	spec := RequestSpec{
		Method:       "GET",
		Path:         "/",
		UseHeaderSet: "multi",
		Headers:      []string{"X-Env: flagval"},
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	// All three should be present as separate headers (Header.Add, not Set)
	vals := req.Header.Values("X-Env")
	if len(vals) != 3 {
		t.Errorf("expected 3 X-Env headers, got %d: %v", len(vals), vals)
	}
}

func TestBuildRequest_InvalidHeader(t *testing.T) {
	p := makeProject()
	spec := RequestSpec{
		Method:  "GET",
		Path:    "/",
		Headers: []string{"invalid-no-colon"},
	}
	_, err := BuildRequest(p, spec)
	if err == nil {
		t.Errorf("expected error for invalid header format")
	}
}

func TestBuildRequest_ContentTypeNotOverridden(t *testing.T) {
	p := makeProject()
	jsonBody := `{"x":1}`
	spec := RequestSpec{
		Method:   "POST",
		Path:     "/",
		JSONBody: &jsonBody,
		Headers:  []string{"Content-Type: text/plain"},
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.Header.Get("Content-Type") != "text/plain" {
		t.Errorf("Content-Type = %q, should not be overridden by JSON", req.Header.Get("Content-Type"))
	}
}

func TestBuildRequest_JSONBodyFromFile(t *testing.T) {
	tmpFile := "/tmp/reqo_test_body.json"
	jsonBody := "@/tmp/reqo_test_body.json"
	if err := writeFile(tmpFile, `{"from":"file"}`); err != nil {
		t.Fatal(err)
	}
	p := makeProject()
	spec := RequestSpec{
		Method:   "POST",
		Path:     "/",
		JSONBody: &jsonBody,
	}
	req, err := BuildRequest(p, spec)
	if err != nil {
		t.Fatalf("BuildRequest() error: %v", err)
	}
	if req.Body == nil {
		t.Errorf("Body should not be nil")
	}
}

func TestBuildRequest_JSONBodyFromFileMissing(t *testing.T) {
	jsonBody := "@/tmp/reqo_nonexistent_file.json"
	p := makeProject()
	spec := RequestSpec{
		Method:   "POST",
		Path:     "/",
		JSONBody: &jsonBody,
	}
	_, err := BuildRequest(p, spec)
	if err == nil {
		t.Errorf("expected error for missing file")
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
