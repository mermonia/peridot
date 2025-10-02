package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mermonia/peridot/config"
	"github.com/mermonia/peridot/internal/logger"
)

func ExecuteInit(initDir string, persist bool) error {
	l := config.NewLoader(config.DefaultPathProvider{})

	// Load general configuration
	cfg, err := l.LoadConfig()
	if err != nil {
		return err
	}

	// Flags override config files
	if initDir != "" {
		cfg.DotfilesDir = initDir
	}

	// Missing dirs / files creation
	if err := createMissingDirs(cfg); err != nil {
		return err
	}

	if err := createMissingModuleDirs(cfg); err != nil {
		return err
	}

	if err := createMissingModuleConfigs(cfg); err != nil {
		return err
	}

	// Module loading (module config existence is needed)
	cfg, err = l.LoadModules(cfg)
	if err != nil {
		return err
	}

	// Write to config file,
	if persist && initDir != "" {
		l.OverwriteConfig(cfg)
	}

	return nil
}

func createMissingDirs(cfg *config.Config) error {
	logger.Info("Creating missing general config dirs...")

	possibleDirs := cfg.GetPathFields()
	for _, dir := range possibleDirs {
		if _, err := os.Stat(*dir.Value); os.IsNotExist(err) {
			logger.Info("About to create dir...", "dir", *dir.Value)

			if err := os.MkdirAll(*dir.Value, 0766); err != nil {
				return fmt.Errorf("Could not create missing dir %s: \n%w", *dir.Value, err)
			}

			logger.Info("Successfully created dir!", "dir", *dir.Value)
		} else if err != nil {
			return err
		}
	}

	logger.Info("Successfully created all missing general config dirs!")
	return nil
}

func createMissingModuleDirs(cfg *config.Config) error {
	logger.Info("Creating missing module dirs...")

	modules := cfg.ManagedModules
	base := cfg.DotfilesDir

	for _, module := range modules {
		moduleDir := filepath.Join(base, module)

		if _, err := os.Stat(moduleDir); os.IsNotExist(err) {
			logger.Info("About to create dir...", "dir", moduleDir)

			if err := os.MkdirAll(moduleDir, 0766); err != nil {
				return fmt.Errorf("Could not create module dir %s: %w", moduleDir, err)
			}

			logger.Info("Successfully created dir!", "dir", moduleDir)
		} else if err != nil {
			return fmt.Errorf("Could not stat dir for module %s: %w", moduleDir, err)
		}
	}

	logger.Info("Successfully created all missing module dirs!")
	return nil
}

func createMissingModuleConfigs(cfg *config.Config) error {
	logger.Info("Creating missing module configs...")

	modules := cfg.ManagedModules
	base := cfg.DotfilesDir

	for _, module := range modules {
		moduleConfigPath := filepath.Join(base, module, config.ModuleConfigFileName)

		_, err := os.Stat(moduleConfigPath)

		if os.IsNotExist(err) {
			logger.Info("About to create file...", "file", moduleConfigPath)

			if err := os.WriteFile(moduleConfigPath, config.DefaultModuleConfig, 0766); err != nil {
				return fmt.Errorf("Could not write config file for module %s: %w", module, err)
			}

			logger.Info("Successfully created file!", "file", moduleConfigPath)
		} else if err != nil {
			return fmt.Errorf("Could not create config file for module %s: %w", module, err)
		}
	}

	logger.Info("Successfully created all missing module configs!")
	return nil
}
