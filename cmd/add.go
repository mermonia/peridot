package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mermonia/peridot/internal/appcontext"
	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/modmgr"
	"github.com/urfave/cli/v3"
)

type AddCommandConfig struct {
	ModuleName string
	Verbose    bool
	Quiet      bool
}

var addCommandDescription string = `
If not already existing, creates a directory and a module config file
for the specified <module>. By default, the module config will be
identical to the provided default-module.toml.

The module will be marked as managed by peridot, whether they were just
created or already existing.
`

var AddCommand cli.Command = cli.Command{
	Name:        "add",
	Aliases:     []string{"a"},
	Usage:       "add a module to the peridot dotfiles directory",
	ArgsUsage:   "<module>",
	Description: addCommandDescription,
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:  "moduleName",
			Value: "",
		},
	},
	MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
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
		appCtx := appcontext.New()
		cmdCfg := &AddCommandConfig{
			ModuleName: filepath.Clean(c.StringArg("moduleName")),
			Verbose:    c.Bool("verbose"),
			Quiet:      c.Bool("quiet"),
		}

		return ExecuteAdd(cmdCfg, appCtx)
	},
}

func ExecuteAdd(cmdCfg *AddCommandConfig, appCtx *appcontext.Context) error {
	if err := logger.InitFileLogging(appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not init file logging: %w", err)
	}
	defer logger.CloseDefaultLogFile()
	logger.SetQuietMode(cmdCfg.Quiet)
	logger.SetVerboseMode(cmdCfg.Verbose)

	if cmdCfg.ModuleName == "" {
		return fmt.Errorf("cannot create a module with an empty name. did you set the module argument?")
	}

	if err := modmgr.AddModule(cmdCfg.ModuleName, appCtx); err != nil {
		return err
	}

	logger.Info("Successfully executed command!", "command", "add")
	return nil
}
