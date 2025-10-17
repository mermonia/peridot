package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/mermonia/peridot/internal/appcontext"
	"github.com/mermonia/peridot/internal/hash"
	"github.com/mermonia/peridot/internal/module"
	"github.com/mermonia/peridot/internal/paths"
	"github.com/mermonia/peridot/internal/state"
	"github.com/mermonia/peridot/internal/templating"
	"github.com/mermonia/peridot/internal/utils"
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
		appCtx := appcontext.New()
		cmdCfg := &DeployCommandConfig{
			Simulate:   c.Bool("simulate"),
			Overwrite:  c.Bool("overwrite"),
			Adopt:      c.Bool("adopt"),
			Dotreplace: c.Bool("dotreplace"),
			Root:       c.String("root"),
			ModuleName: c.StringArg("moduleName"),
		}

		return ExecuteDeploy(cmdCfg, appCtx)
	},
}

func ExecuteDeploy(cmdCfg *DeployCommandConfig, appCtx *appcontext.Context) error {
	dotfilesDir := appCtx.DotfilesDir
	moduleName := cmdCfg.ModuleName

	st, err := state.LoadState(dotfilesDir)
	if err != nil {
		return fmt.Errorf("could not load state: %w", err)
	}

	if err := st.Refresh(appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not refresh state: %w", err)
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
		if err := simulateDeployment(dotfilesDir, mod, filesToDeploy, cmdCfg); err != nil {
			return fmt.Errorf("could not simulate deployment of module %s, %w", moduleName, err)
		}
	} else {
		if err := deployFiles(dotfilesDir, mod, filesToDeploy, cmdCfg); err != nil {
			return fmt.Errorf("could not deploy module %s: %w", moduleName, err)
		}
	}

	// CRITICAL ERROR
	if err := state.SaveState(st, dotfilesDir); err != nil {
		return fmt.Errorf("could not save state: %w", err)
	}

	return nil
}

func getFilesToDeploy(dotfilesDir string, mod *module.Module) []string {
	moduleDir := paths.ModuleDir(dotfilesDir, mod.Name)
	files := []string{}

	// TODO: Handle WalkDir error
	filepath.WalkDir(moduleDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		if slices.Contains(mod.Config.Ignore, filepath.Base(path)) {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files
}

func deployFiles(dotfilesDir string, mod *module.Module, files []string, cmdCfg *DeployCommandConfig) error {
	if err := utils.ExecHook(mod.Config.Hooks.PreDeploy); err != nil {
		return fmt.Errorf("could not execute the pre-deploy hook: %w", err)
	}

	root := mod.Config.Root
	if cmdCfg.Root != "" {
		root = cmdCfg.Root
	}

	for _, path := range files {
		if cmdCfg.Dotreplace {
			path = paths.GetDotreplacedPath(path)
		}

		renderedFilePath, err := paths.RenderedFilePath(path, dotfilesDir)
		if err != nil {
			return fmt.Errorf("could not get potential rendered file path: %w", err)
		}

		symlinkPath, err := paths.SymlinkPath(path, dotfilesDir, mod.Name, root)
		if err != nil {
			return fmt.Errorf("could not get potential symlink path: %w", err)
		}

		if err := resolveSymlinkCollision(mod, path, symlinkPath, cmdCfg.Adopt, cmdCfg.Overwrite); err != nil {
			return err
		}

		if err := templating.CreateRenderedFile(path, renderedFilePath, mod.Config.TemplateVariables); err != nil {
			return fmt.Errorf("could not render template: %w", err)
		}

		if err := os.Remove(symlinkPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove file: %w", err)
		}

		if err := createSymlink(symlinkPath, renderedFilePath); err != nil {
			return err
		}

		fileHash, err := hash.HashFile(path)
		if err != nil {
			return err
		}

		mod.State.Files[path] = &state.Entry{
			Status:           state.Synced,
			SourceHash:       fileHash,
			IntermediatePath: renderedFilePath,
			SymlinkPath:      symlinkPath,
		}
	}

	mod.State.Status = state.Synced
	mod.State.DeployedAt = time.Now()

	if err := utils.ExecHook(mod.Config.Hooks.PostDeploy); err != nil {
		return fmt.Errorf("could not execute the post-deploy hook: %w", err)
	}

	return nil
}

func resolveSymlinkCollision(mod *module.Module, path, symlinkPath string, adopt, overwrite bool) error {
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("could not stat symlink: %w", err)
		}
	} else {
		if info.Mode()&os.ModeSymlink == 0 {
			if adopt {
				if err := utils.Copy(symlinkPath, path); err != nil {
					return fmt.Errorf("could not copy: %w", err)
				}
			} else if !overwrite {
				return fmt.Errorf("found non-symlink without adopt or overwrite option at: %s", symlinkPath)
			}
		} else if !mod.IsSymlinkManaged(symlinkPath) {
			return fmt.Errorf("found existing symlink not managed by module at: %s", symlinkPath)
		}
	}
	return nil
}

func createSymlink(symlinkPath, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(symlinkPath), 0755); err != nil {
		return fmt.Errorf("could not create parent dirs: %w", err)
	}

	if err := os.Symlink(targetPath, symlinkPath); err != nil {
		return fmt.Errorf("could not create symlink: %w", err)
	}

	return nil
}

// TODO: Implement simulation
func simulateDeployment(dotfilesDir string, mod *module.Module, files []string, cmdCfg *DeployCommandConfig) error {

	return nil
}
