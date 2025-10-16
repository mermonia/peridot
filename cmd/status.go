package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mermonia/peridot/internal/appcontext"
	"github.com/mermonia/peridot/internal/state"
	"github.com/mermonia/peridot/internal/tree"
	"github.com/urfave/cli/v3"
)

type StatusCommandConfig struct {
}

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
	Action: func(ctx context.Context, c *cli.Command) error {
		appCtx := appcontext.New()
		return ExecuteStatus(appCtx)
	},
}

func ExecuteStatus(appCtx *appcontext.Context) error {
	st, err := state.LoadState(appCtx.DotfilesDir)
	if err != nil {
		return fmt.Errorf("could not load state: %w", err)
	}

	if err := st.Refresh(appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not refresh state: %w", err)
	}

	tr, err := state.GetStateFileTree(st, appCtx.DotfilesDir)
	if err != nil {
		return fmt.Errorf("could not get state file tree: %w", err)
	}

	tree.PrintTree(tr, tree.DefaultTreeBranchSymbols, os.Stdout)

	if err := state.SaveState(st, appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not save state: %w", err)
	}

	return nil
}
