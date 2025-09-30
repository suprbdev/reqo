package project

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load loads the project.yaml from the given directory.
func Load(dir string) (*Project, error) {
	f := filepath.Join(dir, ".reqo", "project.yaml")
	data, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	var p Project
	if err = yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	if p.Version == 0 {
		p.Version = 1
	}
	return &p, nil
}

// Save writes the project back to its .reqo folder.
func Save(dir string, p *Project) error {
	f := filepath.Join(dir, ".reqo", "project.yaml")
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	return os.WriteFile(f, data, 0o644)
}

// FindProject walks up from cwd looking for a .reqo folder.
// Returns the directory that contains it.
func FindProject(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".reqo", "project.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir { // reached root
			return "", errors.New("no .reqo/project.yaml found – run 'reqo init'")
		}
		dir = parent
	}
}
