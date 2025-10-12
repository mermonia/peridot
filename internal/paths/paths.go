package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DotfilesDirEnvName string = "PERIDOT_DOTFILES_DIR"

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

// Returns the directory specified by the PERIDOT_DOTFILES_DIR
// environment variable if set to a valid path, or the current
// directory otherwise.
// Falls back to an empty string if getting the wd fails.
func GetDotfilesDir() string {
	value, envExists := os.LookupEnv(DotfilesDirEnvName)

	if envExists {
		_, err := os.Stat(value)
		if err == nil {
			return value
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	return cwd
}
