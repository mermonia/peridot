package cmd

import (
	"os"

	"github.com/mermonia/peridot/config"
	"github.com/mermonia/peridot/internal/logger"
)

func ExecuteInit() error {
	l := config.NewLoader(config.DefaultPathProvider{})
	cfg, err := l.Load()

	if err != nil {
		return err
	}

	createMissingDirs(cfg)

	return nil
}

func createMissingDirs(cfg *config.Config) error {
	possibleDirs := cfg.GetPathFields()

	for _, dir := range possibleDirs {
		if _, err := os.Stat(*dir.Value); os.IsNotExist(err) {
			logger.Info("About to create dir", "dir", *dir.Value)
		} else if err != nil {
			return err
		}
	}

	return nil
}
