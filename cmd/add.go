package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/mermonia/peridot/config"
	"github.com/mermonia/peridot/internal/logger"
)

func ExecuteAdd(moduleName string, manageModule bool) error {
	logger.Info("Executing command...", "command", "add")

	if moduleName == "" {
		return fmt.Errorf("Cannot create a module with an empty name! Did you set the module argument?")
	}

	l := config.NewLoader(config.DefaultPathProvider{})
	cfg, err := l.LoadConfig()
	if err != nil {
		return fmt.Errorf("Could not load config in add command: %w", err)
	}

	moduleDir := filepath.Join(cfg.DotfilesDir, moduleName)
	moduleConfigPath := filepath.Join(moduleDir, config.ModuleConfigFileName)

	if err := createModuleIfMissing(moduleDir, moduleConfigPath); err != nil {
		return err
	}

	if manageModule && !slices.Contains(cfg.ManagedModules, moduleName) {
		cfg.ManagedModules = append(cfg.ManagedModules, moduleName)
		l.OverwriteConfig(cfg)
	}

	logger.Info("Successfully executed command!", "command", "add")
	return nil
}

func createModuleIfMissing(moduleDir string, moduleConfigPath string) error {
	if _, err := os.Stat(moduleDir); os.IsNotExist(err) {
		if err := os.MkdirAll(moduleDir, 0766); err != nil {
			return fmt.Errorf("Could not create dir %s: %w", moduleDir, err)
		}
	} else if err != nil {
		return fmt.Errorf("Could not stat the dir %s. Are there any permission conflicts?: %w",
			moduleDir, err)
	}

	if _, err := os.Stat(moduleConfigPath); os.IsNotExist(err) {
		if err := os.WriteFile(moduleConfigPath, config.DefaultModuleConfig, 0766); err != nil {
			return fmt.Errorf("Could not create file %s: %w", moduleConfigPath, err)
		}
	} else if err != nil {
		return fmt.Errorf("Could not stat the file %s. Are there any permission conflicts?: %w",
			moduleConfigPath, err)
	}

	return nil
}
