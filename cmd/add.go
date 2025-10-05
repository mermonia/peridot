package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/mermonia/peridot/config"
	"github.com/mermonia/peridot/internal/logger"
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
	logger.Info("Executing command...", "command", "add")

	if cmdCfg.ModuleName == "" {
		return fmt.Errorf("Cannot create a module with an empty name! Did you set the module argument?")
	}

	l := config.NewConfigLoader(config.DefaultConfigPathProvider{})
	cfg, err := l.LoadConfig()
	if err != nil {
		return fmt.Errorf("Could not load config in add command: %w", err)
	}

	moduleDir := filepath.Join(cfg.DotfilesDir, cmdCfg.ModuleName)
	moduleConfigPath := filepath.Join(moduleDir, config.ModuleConfigFileName)

	if err := createModuleIfMissing(moduleDir, moduleConfigPath); err != nil {
		return err
	}

	if cmdCfg.ManageModule && !slices.Contains(cfg.ManagedModules, cmdCfg.ModuleName) {
		cfg.ManagedModules = append(cfg.ManagedModules, cmdCfg.ModuleName)
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
