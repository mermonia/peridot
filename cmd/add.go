package cmd

import (
	"context"
	"fmt"
	"slices"

	"github.com/mermonia/peridot/config"
	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/modmgr"
	"github.com/urfave/cli/v3"
)

type AddCommandConfig struct {
	ManageModule bool
	ModuleName   string
}

var addCommandDescription string = `
If not already existing, creates a directory and a module config file
for the specified <module>. By default, the module config will be
identical to the provided default-module.toml
`

var AddCommand cli.Command = cli.Command{
	Name:        "add",
	Aliases:     []string{"a"},
	Usage:       "add a module to the peridot dotfiles directory",
	ArgsUsage:   "<module>",
	Description: addCommandDescription,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "manage",
			Aliases: []string{"m"},
			Value:   false,
			Usage:   "add the new module to the config's managed_modules field",
		},
	},
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:  "moduleName",
			Value: "",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		cmdCfg := &AddCommandConfig{
			ManageModule: c.Bool("manage"),
			ModuleName:   c.StringArg("moduleName"),
		}

		return ExecuteAdd(cmdCfg)
	},
}

func ExecuteAdd(cmdCfg *AddCommandConfig) error {
	if cmdCfg.ModuleName == "" {
		return fmt.Errorf("cannot create a module with an empty name. did you set the module argument?")
	}

	loader := config.NewConfigLoader(config.DefaultConfigPathProvider{})
	cfg, err := loader.LoadConfig()
	if err != nil {
		return fmt.Errorf("could not load config %w", err)
	}

	dotfilesDir := cfg.DotfilesDir

	if err := modmgr.AddModule(cmdCfg.ModuleName, dotfilesDir); err != nil {
		return err
	}

	if cmdCfg.ManageModule && !slices.Contains(cfg.ManagedModules, cmdCfg.ModuleName) {
		cfg.ManagedModules = append(cfg.ManagedModules, cmdCfg.ModuleName)
		loader.OverwriteConfig(cfg)
	}

	logger.Info("Successfully executed command!", "command", "add")
	return nil
}
