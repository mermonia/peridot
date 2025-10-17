package cmd

import (
	"context"
	"fmt"

	"github.com/mermonia/peridot/internal/appcontext"
	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/modmgr"
	"github.com/urfave/cli/v3"
)

type RemoveCommandConfig struct {
	ModuleName string
}

var removeCommandDescription string = `
Removes a module from the dotfiles directory. 

The files that are currently managed by peridot (those that have been 
deployed at least once, run 'peridot status' to get a list) will be
treated specially to avoid leaving broken symlinks in the filesystem:

	- If there is a symlink managed by the module at their
	corresponding path (see 'peridot deploy --help' for more
	info), it will be removed.
	- Removed symlinks will be replaced by the rendered contents
	of the template files in the module dir.
	- Keep in mind that the new files will reflect the current state
	of the template files, not the state of their last deployment.

After taking care of deployed files, the entire module directory
will be removed from the dotfiles directory.
`

var RemoveCommand cli.Command = cli.Command{
	Name:        "remove",
	Aliases:     []string{"r"},
	Usage:       "remove a module from the dotfiles directory",
	ArgsUsage:   "<module>",
	Description: removeCommandDescription,
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:  "moduleName",
			Value: "",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		appCtx := appcontext.New()
		cmdCfg := &RemoveCommandConfig{
			ModuleName: c.StringArg("moduleName"),
		}

		return ExecuteRemove(cmdCfg, appCtx)
	},
}

func ExecuteRemove(cmdCfg *RemoveCommandConfig, appCtx *appcontext.Context) error {
	if cmdCfg.ModuleName == "" {
		return fmt.Errorf("cannot remove a directory with an empty name")
	}

	if err := modmgr.RemoveModule(cmdCfg.ModuleName, appCtx); err != nil {
		return err
	}

	logger.Info("Successfully executed command!", "command", "remove")
	return nil
}
