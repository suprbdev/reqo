package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentFile(t *testing.T) {
	got := CurrentFile("/some/dir")
	want := filepath.Join("/some/dir", ".reqo", "current")
	if got != want {
		t.Errorf("CurrentFile() = %q, want %q", got, want)
	}
}

func TestSetCurrent(t *testing.T) {
	dir := t.TempDir()
	name := "my-project"
	if err := SetCurrent(dir, name); err != nil {
		t.Fatalf("SetCurrent() error: %v", err)
	}
	got, err := GetCurrent(dir)
	if err != nil {
		t.Fatalf("GetCurrent() error: %v", err)
	}
	if got != name {
		t.Errorf("GetCurrent() = %q, want %q", got, name)
	}
}

func TestSetCurrent_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	if _, err := os.Stat(filepath.Join(dir, ".reqo")); !os.IsNotExist(err) {
		t.Fatalf(".reqo should not exist yet")
	}
	if err := SetCurrent(dir, "proj"); err != nil {
		t.Fatalf("SetCurrent() error: %v", err)
	}
	if _, err := os.Stat(CurrentFile(dir)); err != nil {
		t.Fatalf("current file should exist: %v", err)
	}
}

func TestGetCurrent_NotExists(t *testing.T) {
	dir := t.TempDir()
	_, err := GetCurrent(dir)
	if err == nil {
		t.Errorf("expected error when current file does not exist")
	}
}

func TestGetCurrent_EmptyName(t *testing.T) {
	dir := t.TempDir()
	if err := SetCurrent(dir, ""); err != nil {
		t.Fatalf("SetCurrent() error: %v", err)
	}
	got, err := GetCurrent(dir)
	if err != nil {
		t.Fatalf("GetCurrent() error: %v", err)
	}
	if got != "" {
		t.Errorf("GetCurrent() = %q, want empty", got)
	}
}
