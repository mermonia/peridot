package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mermonia/peridot/internal/appcontext"
	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/state"
	"github.com/mermonia/peridot/internal/tree"
	"github.com/urfave/cli/v3"
)

type StatusCommandConfig struct {
	ModuleName string
	Depth      int
	Summary    bool
	Verbose    bool
	Quiet      bool
}

const MAX_DEPTH = 100

var statusCommandDescription string = `
Displays the current state of the peridot dotfiles directory.

If at least one module is managed by peridot, a file tree containing
all the managed files will be printed. The root of the tree will be
the dotfiles dir itself. 

A populated tree will show the status of each module and their files.
A module can be:
	- Not deployed
	- Up to date
	- Unsynced

Additionally, files that are part of a deployed module can be:
	- Up to date
	- Unsynced

An unsynced file / module can be updated via the 'peridot deploy'
command. Doing so will udpate its respective intermediate file
(run 'peridot deploy --help' for more information).

Example output:
.
├── ✓ module1 - deployed and up to date
│   ├── ✓ modulefile1.conf
│   └── ✓ modulefile2.conf
├── ✗ module2 - deployed, pending sync
│   ├── ✗ modulefile1.conf
│   └── ✓ modulefile2.conf
└── ○ module3 - not deployed
	└── modulefile.conf
`

var StatusCommand cli.Command = cli.Command{
	Name:        "status",
	Aliases:     []string{"s"},
	Usage:       "display the current state of the dotfiles dir",
	Description: statusCommandDescription,
	ArgsUsage:   "[module]",
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
					&cli.BoolFlag {
						Name: "summary",
						Aliases: []string{"s"},
						Value: false,
						Usage: "only show module status, skip files",
					},
				},
				{
					&cli.IntFlag {
						Name: "depth",
						Aliases: []string{"d"},
						Value: 0,
						Usage: "print status tree up until given depth",
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
		appCtx := appcontext.New()

		cmdCfg := &StatusCommandConfig{
			ModuleName: c.StringArg("moduleName"),
			Summary: c.Bool("summary"),
			Verbose: c.Bool("verbose"),
			Quiet:   c.Bool("quiet"),
			Depth: c.Int("depth"),
		}
		return ExecuteStatus(appCtx, cmdCfg)
	},
}

func ExecuteStatus(appCtx *appcontext.Context, cmdCfg *StatusCommandConfig) error {
	if err := logger.InitFileLogging(appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not init file logging: %w", err)
	}
	defer logger.CloseDefaultLogFile()
	logger.SetVerboseMode(cmdCfg.Verbose)
	logger.SetQuietMode(cmdCfg.Quiet)

	st, err := state.LoadState(appCtx.DotfilesDir)
	if err != nil {
		return fmt.Errorf("could not load state: %w", err)
	}

	if err := st.Refresh(appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not refresh state: %w", err)
	}

	if cmdCfg.ModuleName == "" {
		depth := MAX_DEPTH
		if cmdCfg.Depth > 0 {
			depth = cmdCfg.Depth
		}
		if cmdCfg.Summary {
			depth = 2
		}
		if err := printStateTree(st, appCtx.DotfilesDir, depth); err != nil {
			return err
		}
	} else {
		depth := MAX_DEPTH
		if cmdCfg.Depth > 0 {
			depth = cmdCfg.Depth
		}
		if cmdCfg.Summary {
			depth = 1
		}
		if err := printModuleTree(st, appCtx.DotfilesDir, cmdCfg.ModuleName, depth); err != nil {
			return err
		}
	}

	if err := state.SaveState(st, appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not save state: %w", err)
	}

	logger.Info("Successfully executed command!", "command", "status")
	return nil
}

func printStateTree(st *state.State, dotfilesDir string, depth int) error {
	tr, err := state.GetStateFileTree(st, dotfilesDir)
	if err != nil {
		return fmt.Errorf("could not get state file tree: %w", err)
	}

	tree.PrintTree(tr, tree.DefaultTreeBranchSymbols, os.Stdout, depth)
	return nil
}

func printModuleTree(st *state.State, dotfilesDir, moduleName string, depth int) error {
	moduleState := st.Modules[moduleName]
	if moduleState == nil {
		return fmt.Errorf("cannot print a non-existing module")
	}

	tr, err := state.GetModuleFileTree(moduleName, moduleState, dotfilesDir)
	if err != nil {
		return fmt.Errorf("could not get module file tree: %w", err)
	}

	tree.PrintTree(tr, tree.DefaultTreeBranchSymbols, os.Stdout, depth)
	return nil
}
