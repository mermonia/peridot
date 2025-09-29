package cmd

import (
	"fmt"
	"os"

	"github.com/mermonia/peridot/config"
	"github.com/mermonia/peridot/internal/logger"
)

func ExecuteInit() error {
	l := config.NewLoader(config.DefaultPathProvider{})

	cfg, err := l.LoadConfig()
	if err != nil {
		return err
	}

	cfg, err = cfg.LoadModules()
	if err != nil {
		return err
	}

	if err := createMissingDirs(cfg); err != nil {
		return err
	}

	if err := createMissingModuleDirs(cfg); err != nil {
		return err
	}

	if err := createMissingModuleConfigs(cfg); err != nil {
		return err
	}

	return nil
}

func createMissingDirs(cfg *config.Config) error {
	possibleDirs := cfg.GetPathFields()

	for _, dir := range possibleDirs {
		if _, err := os.Stat(*dir.Value); os.IsNotExist(err) {
			logger.Info("About to create dir...", "dir", *dir.Value)

			if err := os.MkdirAll(*dir.Value, 0766); err != nil {
				return fmt.Errorf("Could not create dir %s: \n%w", *dir.Value, err)
			}

			logger.Info("Successfuly created dir!", "dir", *dir.Value)
		} else if err != nil {
			return err
		}
	}

	return nil
}

func createMissingModuleDirs(cfg *config.Config) error {

	return nil
}

func createMissingModuleConfigs(cfg *config.Config) error {

	return nil
}
