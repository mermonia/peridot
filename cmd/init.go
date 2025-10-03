package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mermonia/peridot/config"
	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/paths"
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
			return fmt.Errorf("Could not initialize dotfiles_dir: %w", err)
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
				return fmt.Errorf("Could not initialize dir scpecified by the --dir flag: %w", err)
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
	logger.Info("Executing command...", "command", "init")

	l := config.NewLoader(config.DefaultPathProvider{})

	// Load general configuration
	cfg, err := l.LoadConfig()
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
	if cmdCfg.Persist && cmdCfg.InitDir != "" {
		l.OverwriteConfig(cfg)
	}

	logger.Info("Successfully executed command!", "command", "init")
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
