package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/paths"
	"github.com/mermonia/peridot/internal/state"
	"github.com/urfave/cli/v3"
)

type InitCommandConfig struct {
	InitDir string
	Verbose bool
	Quiet   bool
}

const initCommandDescription string = `
If not already, initializes a directory as a peridot dotfiles dir.
The directory to be initialized can be specified via flags (such as
--dir, --here) or the PERIDOT_DOTFILES_DIR environment variable.

If none of those flags are set and the environment variable is not set
or set to an invalid path, peridot will issue a warning and initialize
the current directory.

Initializing a directory essentially means ensuring the .peridot/state.json
file exists.
`

var InitCommand cli.Command = cli.Command{
	Name:        "init",
	Aliases:     []string{"i"},
	Usage:       "initialize dotfiles dir",
	Description: initCommandDescription,
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
		{
			Required: false,
			Flags: [][]cli.Flag{
				{
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Value:   false,
						Usage:   "show verbose debug info",
					},
				},
				{
					&cli.BoolFlag{
						Name:    "quiet",
						Aliases: []string{"q"},
						Value:   false,
						Usage:   "supress most logging output",
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
		if c.Bool("here") {
			initDir = cwd
		} else if c.String("dir") != "" {
			initDir, err = paths.ResolvePath(c.String("dir"), cwd)

			if err != nil {
				return fmt.Errorf("could not resolve path specified by the --dir flag: %w", err)
			}
		} else if val, exists := os.LookupEnv(paths.DotfilesDirEnvName); exists {
			initDir = val
		} else {
			initDir = cwd
		}

		cmdCfg := &InitCommandConfig{
			InitDir: initDir,
			Verbose: c.Bool("verbose"),
			Quiet:   c.Bool("quiet"),
		}

		return ExecuteInit(cmdCfg)
	},
}

func ExecuteInit(cmdCfg *InitCommandConfig) error {
	dotfilesDir := cmdCfg.InitDir

	if err := logger.InitFileLogging(dotfilesDir); err != nil {
		return fmt.Errorf("could not init file logging: %w", err)
	}
	defer logger.CloseDefaultLogFile()
	logger.SetVerboseMode(cmdCfg.Verbose)
	logger.SetQuietMode(cmdCfg.Quiet)

	if err := createStateFile(dotfilesDir); err != nil {
		return fmt.Errorf("could not create state file: %w", err)
	}

	logger.Info("Successfully executed command!", "command", "init")
	return nil
}

func createStateFile(dotfilesDir string) error {
	if err := os.MkdirAll(filepath.Join(dotfilesDir, paths.PeridotDirName), 0755); err != nil {
		return fmt.Errorf("could not create state file: %w", err)
	}

	if err := state.SaveState(&state.State{
		Modules: map[string]*state.ModuleState{},
	}, dotfilesDir); err != nil {
		return fmt.Errorf("could not save state: %w", err)
	}
	return nil
}
