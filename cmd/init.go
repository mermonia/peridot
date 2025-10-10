package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mermonia/peridot/config"
	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/modmgr"
	"github.com/mermonia/peridot/internal/paths"
	"github.com/mermonia/peridot/internal/state"
	"github.com/urfave/cli/v3"
)

type InitCommandConfig struct {
	InitDir string
	Persist bool
}

var initCommandDescription string = `
If not already existing, creates the directory specified in the
"dotfiles_dir" field of the peridot config, along with directories
and config files for all modules specified in the "managed_modules"
field.
A '.cache/' directory will be created inside the dotfiles_dir, and
it will be populated by an empty state file 'state.json'
`

var InitCommand cli.Command = cli.Command{
	Name:        "init",
	Aliases:     []string{"i"},
	Usage:       "initialize dotfiles dir",
	Description: initCommandDescription,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "persist",
			Aliases: []string{"p"},
			Value:   false,
			Usage:   "overwrite the user config when setting a new dotfiles dir",
		},
	},
	MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
		{
			Required: false,
			Flags: [][]cli.Flag{
				{
					&cli.StringFlag{
						Name:    "dir",
						Aliases: []string{"d"},
						Value:   "",
						Usage:   "path of the dir to be initialized",
					},
				},
				{
					&cli.BoolFlag{
						Name:    "here",
						Aliases: []string{"H"},
						Value:   false,
						Usage:   "set the dir to be initialized to the current dir",
					},
				},
			},
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not initialize dotfiles_dir: %w", err)
		}

		var initDir string
		// Infer initDir from the flags. Even though some flags technically override
		// others, they are mutually exclusive. The code's priority order should be
		// ignored in practice.
		if c.Bool("here") {
			initDir = cwd
		}
		if c.String("dir") != "" {
			initDir, err = paths.ResolvePath(c.String("dir"), cwd)

			if err != nil {
				return fmt.Errorf("could not initialize dir scpecified by the --dir flag: %w", err)
			}
		}

		cmdCfg := &InitCommandConfig{
			InitDir: initDir,
			Persist: c.Bool("persist"),
		}

		return ExecuteInit(cmdCfg)
	},
}

func ExecuteInit(cmdCfg *InitCommandConfig) error {
	loader := config.NewConfigLoader(config.DefaultConfigPathProvider{})

	// Load general configuration
	cfg, err := loader.LoadConfig()
	if err != nil {
		return err
	}

	// Flags override config files
	if cmdCfg.InitDir != "" {
		cfg.DotfilesDir = cmdCfg.InitDir
	}

	// Missing dirs / files creation
	if err := createMissingDirs(cfg); err != nil {
		return err
	}

	if err := addMissingModules(cfg); err != nil {
		return err
	}

	if err := createStateFile(cfg); err != nil {
		return err
	}

	// Module loading (module config existence is needed)
	cfg, err = loader.LoadModules(cfg)
	if err != nil {
		return err
	}

	// Write to config file,
	if cmdCfg.Persist && cmdCfg.InitDir != "" {
		loader.OverwriteConfig(cfg)
	}

	logger.Info("Successfully executed command!", "command", "init")
	return nil
}

func addMissingModules(cfg *config.Config) error {
	modules := cfg.ManagedModules
	dotfilesDir := cfg.DotfilesDir

	for _, module := range modules {
		if err := modmgr.AddModule(module, dotfilesDir); err != nil {
			return err
		}
	}

	return nil
}

func createMissingDirs(cfg *config.Config) error {
	possibleDirs := cfg.GetPathFields()
	for _, dir := range possibleDirs {
		if err := os.MkdirAll(*dir.Value, 0755); err != nil {
			return fmt.Errorf("could not create missing dir %s: \n%w", *dir.Value, err)
		}
		logger.Info("Successfully created dir!", "dir", *dir.Value)
	}
	logger.Info("Successfully created all missing general config dirs!")
	return nil
}

func createStateFile(cfg *config.Config) error {
	if err := os.MkdirAll(filepath.Join(cfg.DotfilesDir, ".cache"), 0755); err != nil {
		return fmt.Errorf("could not create state file: %w", err)
	}

	if err := state.SaveState(&state.State{
		Modules: map[string]*state.ModuleState{},
	}); err != nil {
		return fmt.Errorf("could not save state: %w", err)
	}

	return nil
}
