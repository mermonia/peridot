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
	varsDir := paths.GlobalVarsDir(dotfilesDir)

	entries, err := os.ReadDir(varsDir)
	if err != nil {
		return nil, fmt.Errorf("could not read files from global vars dir: %w", err)
	}

	for _, e := range entries {
		if !isVarsFile(e) {
			continue
		}

		vars := &GlobalVars{}
		filePath := filepath.Join(varsDir, e.Name())
		if _, err := toml.DecodeFile(filePath, vars); err == nil {
			maps.Copy(globalVars, vars.Variables)
		}
	}

	return globalVars, nil
}

func isVarsFile(entry fs.DirEntry) bool {
	return !entry.IsDir() && filepath.Ext(entry.Name()) == ".toml"
}
