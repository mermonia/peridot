package modmgr

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mermonia/peridot/config"
	"github.com/mermonia/peridot/internal/logger"
)

func AddModule(moduleName string, dotfilesDir string) error {
	if err := createModuleIfMissing(moduleName, dotfilesDir); err != nil {
		return fmt.Errorf("could not add module %s: %w", moduleName, err)
	}
	logger.Info("Successfully added module", "module", moduleName)
	return nil
}

func createModuleIfMissing(moduleName string, dotfilesDir string) error {
	moduleDir := filepath.Join(dotfilesDir, moduleName)
	moduleConfigPath := filepath.Join(moduleDir, config.ModuleConfigFileName)

	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", moduleDir, err)
	}

	if _, err := os.Stat(moduleConfigPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("could not stat config file %s: %w", moduleConfigPath, err)
	}

	if err := os.WriteFile(moduleConfigPath, config.DefaultModuleConfig, 0644); err != nil {
		return fmt.Errorf("could not create config file %s: %w", moduleConfigPath, err)
	}

	return nil
}
