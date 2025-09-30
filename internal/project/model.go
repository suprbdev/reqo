package project

type Project struct {
	Version      int                    `yaml:"version"`
	Name         string                 `yaml:"name"`
	DefaultEnv   string                 `yaml:"default_env,omitempty"` // e.g. "default"
	Environments map[string]Environment `yaml:"environments,omitempty"`
	HeaderSets   map[string][]string    `yaml:"header_sets,omitempty"` // name → list of “Key: Value”
	Calls        map[string]Call        `yaml:"calls,omitempty"`       // alias → definition
}

type Environment struct {
	BaseURL string   `yaml:"base_url,omitempty"`
	Headers []string `yaml:"headers,omitempty"` // raw header lines
}

// Call describes a saved request.
type Call struct {
	Method       string            `yaml:"method,omitempty"` // GET, POST …
	Path         string            `yaml:"path,omitempty"`   // may contain ${var}
	Headers      []string          `yaml:"headers,omitempty"`
	Query        map[string]string `yaml:"query,omitempty"`
	Body         *BodySpec         `yaml:"body,omitempty"`
	UseHeaderSet string            `yaml:"uses_header_set,omitempty"` // name of a header set
	Description  string            `yaml:"description,omitempty"`
	LastUsed     string            `yaml:"last_used,omitempty"` // timestamp (optional)
}

type BodySpec struct {
	JSON *string           `yaml:"json,omitempty"` // raw JSON string or @file
	Raw  *string           `yaml:"raw,omitempty"`  // raw body string or @file
	Form map[string]string `yaml:"form,omitempty"` // key=value, file=@path supported
}
