package cmd

import (
	"context"

	"github.com/urfave/cli/v3"
)

type DeployCommandConfig struct {
	Simulate   bool
	Overwrite  bool
	Dotreplace bool
	Dir        string
	Root       string
}

var deployCommandDescription string = `
If not already, deploys the files in the specified module directory.

In order to facilitate some of peridot's features (mainly templating),
the symlinks that are created in the filesystem are not links to the
module dir's files themselves. Instead, they point to an intermediate
file that contains the already preprocessed contents of the template
files. Even if templating was explicitly disables, the symlinks will
still point to these intermediate files, although their content will
be identical to those in the module dir.

All intermediate files are stored in the "DOTFILES_DIR/.cache" directory,
whose structure mimics that of the DOTFILES_DIR itself. For example,
deploying a file stored as "DOTFILES_DIR/kitty/.config/kitty/kitty.conf":
	- Creates an intermediate file: "DOTFILES_DIR/.cache/kitty/.config/kitty/kitty.conf"
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
			Name:    "overwrite",
			Aliases: []string{"O"},
			Value:   false,
			Usage: "if there are any files in the actual filesystem that share both name\n" +
				"and path with a file in the module dir, overwite them",
		},
		&cli.BoolFlag{
			Name:    "dotreplace",
			Aliases: []string{"D"},
			Value:   false,
			Usage: "rename both the intermediate file and the symlink to the deployed\n" +
				"files, from dot-* to .*",
		},
		&cli.StringFlag{
			Name:      "dir",
			Aliases:   []string{"d"},
			Value:     "",
			Usage:     "specify the path to the module dir to be deployed",
			TakesFile: true,
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
			Dotreplace: c.Bool("dotreplace"),
			Dir:        c.String("dir"),
			Root:       c.String("root"),
		}

		return ExecuteDeploy(cmdCfg)
	},
}

func ExecuteDeploy(cmdCfg *DeployCommandConfig) error {

	return nil
}
