package config

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveEnvVar resolves a $VAR_NAME reference to its environment variable
// value. If the value doesn't start with $, it is returned as-is.
// If the env var is empty or unset, returns "".
// See Spec Section 8.1 (item 4), Section 6.3.1.
func ResolveEnvVar(value string) string {
	if !strings.HasPrefix(value, "$") {
		return value
	}
	varName := value[1:]
	return os.Getenv(varName)
}

// ExpandHome replaces a leading ~ with the user's home directory.
// If the home directory can't be determined, returns the path unchanged.
func ExpandHome(path string) string {
	if path == "" {
		return path
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
