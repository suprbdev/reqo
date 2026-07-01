package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	p := &Project{
		Version:    1,
		Name:       "test-api",
		DefaultEnv: "dev",
		Environments: map[string]Environment{
			"dev":  {BaseURL: "https://dev.example.com", Headers: []string{"X-Debug: true"}},
			"prod": {BaseURL: "https://api.example.com"},
		},
		HeaderSets: map[string][]string{
			"auth": {"Authorization: Bearer token"},
		},
		Calls: map[string]Call{
			"get-users": {
				Method: "GET",
				Path:   "/users",
			},
		},
	}

	if err := Save(dir, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, ".reqo", "project.yaml")); err != nil {
		t.Fatalf("project.yaml should exist: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Name != p.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, p.Name)
	}
	if loaded.DefaultEnv != p.DefaultEnv {
		t.Errorf("DefaultEnv = %q, want %q", loaded.DefaultEnv, p.DefaultEnv)
	}
	if loaded.Version != p.Version {
		t.Errorf("Version = %d, want %d", loaded.Version, p.Version)
	}
	if len(loaded.Environments) != 2 {
		t.Errorf("Environments count = %d, want 2", len(loaded.Environments))
	}
	if loaded.Environments["dev"].BaseURL != "https://dev.example.com" {
		t.Errorf("dev BaseURL = %q", loaded.Environments["dev"].BaseURL)
	}
	if len(loaded.HeaderSets["auth"]) != 1 {
		t.Errorf("auth header set len = %d, want 1", len(loaded.HeaderSets["auth"]))
	}
	if loaded.Calls["get-users"].Method != "GET" {
		t.Errorf("get-users Method = %q", loaded.Calls["get-users"].Method)
	}
}

func TestLoad_DefaultVersion(t *testing.T) {
	dir := t.TempDir()
	// Write a YAML with no version field
	yamlContent := "name: test\n"
	target := filepath.Join(dir, ".reqo", "project.yaml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Version != 1 {
		t.Errorf("Version = %d, want 1 (default)", loaded.Version)
	}
}

func TestLoad_NotExists(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Errorf("expected error when project.yaml does not exist")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".reqo", "project.yaml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("name: [invalid yaml }}}}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Errorf("expected error for invalid YAML")
	}
}

func TestSave_NilMaps(t *testing.T) {
	dir := t.TempDir()
	p := &Project{
		Version:    1,
		Name:       "minimal",
		DefaultEnv: "default",
	}
	if err := Save(dir, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Name != "minimal" {
		t.Errorf("Name = %q", loaded.Name)
	}
}

func TestSave_BodySpec(t *testing.T) {
	dir := t.TempDir()
	jsonBody := `{"key":"value"}`
	rawBody := "raw text"
	p := &Project{
		Version:    1,
		Name:       "bodies",
		DefaultEnv: "default",
		Calls: map[string]Call{
			"json-call": {
				Method: "POST",
				Path:   "/api",
				Body: &BodySpec{
					JSON: &jsonBody,
				},
			},
			"raw-call": {
				Method: "POST",
				Path:   "/api",
				Body: &BodySpec{
					Raw: &rawBody,
				},
			},
			"form-call": {
				Method: "POST",
				Path:   "/upload",
				Body: &BodySpec{
					Form: map[string]string{"file": "@data.txt", "desc": "hello"},
				},
			},
		},
	}
	if err := Save(dir, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Calls["json-call"].Body.JSON == nil || *loaded.Calls["json-call"].Body.JSON != jsonBody {
		t.Errorf("json-call JSON body not preserved")
	}
	if loaded.Calls["raw-call"].Body.Raw == nil || *loaded.Calls["raw-call"].Body.Raw != rawBody {
		t.Errorf("raw-call Raw body not preserved")
	}
	if len(loaded.Calls["form-call"].Body.Form) != 2 {
		t.Errorf("form-call Form fields = %d, want 2", len(loaded.Calls["form-call"].Body.Form))
	}
}
