package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolvePath(path string, base string) (string, error) {
	// Resolve leading tildes
	if s, found := strings.CutPrefix(path, "~"); found {
		homeDir, err := os.UserHomeDir()

		if err != nil {
			return "", fmt.Errorf("failed to find user home dir while resolving a tilde in the path %s: %w", path, err)
		}

		path = filepath.Join(homeDir, s)
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	resolved := filepath.Join(base, path)
	absPath, err := filepath.Abs(resolved)

	if err != nil {
		return "", fmt.Errorf("could not resolve relative path: %s, %w", path, err)
	}

	return absPath, nil
}
