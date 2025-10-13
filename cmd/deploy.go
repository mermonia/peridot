package cmd

import (
	"context"
	"fmt"

	"github.com/mermonia/peridot/internal/module"
	"github.com/mermonia/peridot/internal/paths"
	"github.com/mermonia/peridot/internal/state"
	"github.com/urfave/cli/v3"
)

type DeployCommandConfig struct {
	Simulate   bool
	Overwrite  bool
	Adopt      bool
	Dotreplace bool
	Root       string
	ModuleName string
}

var deployCommandDescription string = `
If not already, deploys the files in the specified module directory.

Before deploying a module, both their dependencies and module dependencies
(that is, modules that should be deployed before them) are checked.
If a dependency is not satisfied, the module will not be deployed and
the command will return an error.

In order to facilitate some of peridot's features (mainly templating),
the symlinks that are created in the filesystem are not links to the
module dir's files themselves. Instead, they point to an intermediate
file that contains the already preprocessed contents of the template
files. Even if templating was explicitly disables, the symlinks will
still point to these intermediate files, although their content will
be identical to those in the module dir.

All intermediate files are stored in the "DOTFILES_DIR/.peridot" directory,
whose structure mimics that of the DOTFILES_DIR itself. For example,
deploying a file stored as "DOTFILES_DIR/kitty/.config/kitty/kitty.conf":
	- Creates an intermediate file: "DOTFILES_DIR/.peridot/kitty/.config/kitty/kitty.conf"
	- Creates a symlink pointing to the intermediate file at ROOT/.config/kitty/kitty.conf
`

var DeployCommand cli.Command = cli.Command{
	Name:        "deploy",
	Aliases:     []string{"d"},
	Usage:       "create dir/file symlinks from filesystem to module dir",
	ArgsUsage:   "<module>",
	Description: deployCommandDescription,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "simulate",
			Aliases: []string{"s"},
			Value:   false,
			Usage:   "don't make any changes, merely show what would be done",
		},
		&cli.BoolFlag{
			Name:    "dotreplace",
			Aliases: []string{"D"},
			Value:   false,
			Usage: "rename both the intermediate file and the symlink to the deployed\n" +
				"files, from dot-* to .*",
		},
		&cli.StringFlag{
			Name:    "root",
			Aliases: []string{"r"},
			Value:   "",
			Usage: "specify the root path to which the module dir's structure should\n" +
				"be deployed",
			TakesFile: true,
		},
	},
	MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
		{
			Required: false,
			Flags: [][]cli.Flag{
				{
					&cli.BoolFlag{
						Name:    "overwrite",
						Aliases: []string{"O"},
						Value:   false,
						Usage: "forcefully replaces existing files in the filesystem by removing\n" +
							"them and creating the symlink",
					},
				},
				{
					&cli.BoolFlag{
						Name:    "adopt",
						Aliases: []string{"a"},
						Value:   false,
						Usage: "Imports existing files by copying their contents into the module,\n" +
							"then removes the originals and replaces them with symlinks",
					},
				},
			},
		},
	},
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:  "moduleName",
			Value: "",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		cmdCfg := &DeployCommandConfig{
			Simulate:   c.Bool("simulate"),
			Overwrite:  c.Bool("overwrite"),
			Adopt:      c.Bool("adopt"),
			Dotreplace: c.Bool("dotreplace"),
			Root:       c.String("root"),
			ModuleName: c.StringArg("moduleName"),
		}

		return ExecuteDeploy(cmdCfg)
	},
}

func ExecuteDeploy(cmdCfg *DeployCommandConfig) error {
	dotfilesDir := paths.DotfilesDir()
	moduleName := cmdCfg.ModuleName

	st, err := state.LoadState(paths.StateFilePath())
	if err != nil {
		return fmt.Errorf("could not load state: %w")
	}

	moduleState := st.Modules[moduleName]
	if moduleState == nil {
		return fmt.Errorf("the specified module is not managed by peridot")
	}

	mod, err := module.Load(dotfilesDir, moduleName, moduleState)
	if err != nil {
		return fmt.Errorf("could not load module %s: %w", moduleName, err)
	}

	if !mod.ShouldDeploy(st) {
		return fmt.Errorf("the module %s could not be deployed: %w", moduleName, err)
	}

	filesToDeploy := getFilesToDeploy(dotfilesDir, mod)
	if cmdCfg.Simulate {
		return simulateDeployment(dotfilesDir, mod, filesToDeploy, cmdCfg)
	} else {
		return deployFiles(dotfilesDir, mod, filesToDeploy, cmdCfg)
	}
}

func getFilesToDeploy(dotfilesDir string, mod *module.Module) []string {
	files := []string{}

	// getting files logic

	return files
}

func deployFiles(dotfilesDir string, mod *module.Module, files []string, cmdCfg *DeployCommandConfig) error {
	//deployment logic

	return nil
}

func simulateDeployment(dotfilesDir string, mod *module.Module, files []string, cmdCfg *DeployCommandConfig) error {
	//simulation logic

	return nil
}
