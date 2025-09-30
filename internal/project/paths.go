package project

import (
	"os"
	"path/filepath"
)

// CurrentFile returns the path of ".reqo/current" in dir (if exists).
func CurrentFile(dir string) string {
	return filepath.Join(dir, ".reqo", "current")
}

// SetCurrent marks a project as active for the directory.
// It simply writes the project's name to .reqo/current.
func SetCurrent(dir, projName string) error {
	f := CurrentFile(dir)
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	return os.WriteFile(f, []byte(projName), 0o644)
}

// GetCurrent reads the active project name from .reqo/current (if any).
func GetCurrent(dir string) (string, error) {
	f := CurrentFile(dir)
	b, err := os.ReadFile(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
