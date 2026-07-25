package templating

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/mermonia/peridot/internal/paths"
)

type GlobalVars struct {
	Variables map[string]string `json:"variables"`
}

func LoadGlobalVars(dotfilesDir string) (map[string]string, error) {
	globalVars := make(map[string]string)

	filePaths, err := ListGlobalVarsFiles(dotfilesDir)
	if err != nil {
		return nil, err
	}

	for _, filePath := range filePaths {
		vars := &GlobalVars{}
		if _, err := toml.DecodeFile(filePath, vars); err == nil {
			maps.Copy(globalVars, vars.Variables)
		}
	}

	return globalVars, nil
}

func ListGlobalVarsFiles(dotfilesDir string) ([]string, error) {
	varsDir := paths.GlobalVarsDir(dotfilesDir)
	filePaths := []string{}

	entries, err := os.ReadDir(varsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return filePaths, nil
		}
		return nil, fmt.Errorf("could not read files from global vars dir: %w", err)
	}

	for _, e := range entries {
		if !isVarsFile(e) {
			continue
		}

		filePaths = append(filePaths, filepath.Join(varsDir, e.Name()))
	}

	return filePaths, nil
}

func isVarsFile(entry fs.DirEntry) bool {
	return !entry.IsDir() && filepath.Ext(entry.Name()) == ".toml"
}
