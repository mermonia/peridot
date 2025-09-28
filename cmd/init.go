package cmd

import (
	"fmt"
	"os"

	"github.com/mermonia/peridot/config"
)

func ExecuteInit() error {
	l := config.NewLoader(config.DefaultPathProvider{})
	cfg, err := l.Load()

	if err != nil {
		return err
	}

	createMissingDirs(cfg)

	fmt.Println("Executed init!")
	return nil
}

func createMissingDirs(cfg *config.Config) error {
	possibleDirs := cfg.GetPathFields()

	for _, dir := range possibleDirs {
		if _, err := os.Stat(*dir.Value); os.IsNotExist(err) {
			fmt.Printf("Will create %s: %s\n", dir.Name, *dir.Value)
		} else if err != nil {
			return err
		}
	}

	return nil
}
