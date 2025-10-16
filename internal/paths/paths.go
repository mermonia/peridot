package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DotfilesDirEnvName   = "PERIDOT_DOTFILES_DIR"
	PeridotDirName       = ".peridot"
	StateFileName        = "state.json"
	ModuleConfigFileName = "module.toml"
	DotreplacePrefix     = "dot-"
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

// DotfilesDir returns the directory specified by the PERIDOT_DOTFILES_DIR
// environment variable. If not set, it searches upward for a .peridot directory
// containing state.json, starting from the current directory.
// Falls back to the current working directory if none is found.
func DotfilesDir() string {
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

	if dir, err := findDotfilesDir(cwd); err == nil {
		return dir
	}

	return cwd
}

func findDotfilesDir(start string) (string, error) {
	for dir := start; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, PeridotDirName, StateFileName)
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
	}

	return "", fmt.Errorf("no peridot directory found")
}

func PeridotDir(dotfilesDir string) string {
	return filepath.Join(dotfilesDir, PeridotDirName)
}

func StateFilePath(dotfilesDir string) string {
	return filepath.Join(PeridotDir(dotfilesDir), StateFileName)
}

func GetDotreplacedPath(path string) string {
	dir, file := filepath.Split(path)
	if cutFile, hasPrefix := strings.CutPrefix(file, DotreplacePrefix); hasPrefix {
		file = "." + cutFile
	}

	return filepath.Join(dir, file)
}

func RenderedFilePath(path string, dotfilesDir string) (string, error) {
	rel, err := filepath.Rel(dotfilesDir, path)
	if err != nil {
		return "", fmt.Errorf("could not relativize path: %w", err)
	}

	return filepath.Join(PeridotDir(dotfilesDir), rel), nil
}

func SymlinkPath(path, dotfilesDir, moduleName, root string) (string, error) {
	rel, err := filepath.Rel(ModuleDir(dotfilesDir, moduleName), path)
	if err != nil {
		return "", fmt.Errorf("could not relativize path: %w", err)
	}

	return filepath.Join(root, rel), nil
}

func ModuleDir(dotfilesDir, moduleName string) string {
	return filepath.Join(dotfilesDir, moduleName)
}

func SplitPath(path string) []string {
	path = filepath.Clean(path)
	var parts []string

	for {
		dir, file := filepath.Split(path)
		if file != "" {
			parts = append([]string{file}, parts...)
		}

		if dir == "" || dir == path {
			if dir != "" {
				parts = append([]string{dir}, parts...)
			}
			break
		}
		path = dir[:len(dir)-1]
	}

	return parts
}
