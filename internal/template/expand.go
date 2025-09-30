package template

import (
	"os"
	"regexp"
)

var varRE = regexp.MustCompile(`\$\{([^}]+)\}`)

// Expand replaces ${key} with the first value found in:
//  1. vars map (provided via --var)
//  2. environment variables (os.Getenv)
//  3. If still missing, leaves the placeholder untouched.
func Expand(input string, vars map[string]string) string {
	return varRE.ReplaceAllStringFunc(input, func(m string) string {
		key := varRE.FindStringSubmatch(m)[1]
		if v, ok := vars[key]; ok {
			return v
		}
		if ev := os.Getenv(key); ev != "" {
			return ev
		}
		// TODO: future project‑level variables could be added here.
		return m // leave as is – caller may decide to error later.
	})
}

// ExpandMap expands every value in a map[string]string.
func ExpandMap(in map[string]string, vars map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = Expand(v, vars)
	}
	return out
}
