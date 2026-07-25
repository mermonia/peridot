package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mermonia/peridot/internal/appcontext"
	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/module"
	"github.com/mermonia/peridot/internal/paths"
	"github.com/mermonia/peridot/internal/state"
	"github.com/mermonia/peridot/internal/templating"
	"github.com/mermonia/peridot/internal/watcher"
	"github.com/urfave/cli/v3"
)

type WatchCommandConfig struct {
	WatchDir   string
	Debounce   time.Duration
	Overwrite  bool
	Adopt      bool
	Dotreplace bool
	Root       string
	Verbose    bool
	Quiet      bool
}

// DefaultDebounce is long enough to coalesce the burst of events a single
// editor save produces, and short enough to feel immediate.
const DefaultDebounce = 500 * time.Millisecond

var watchCommandDescription string = `
Runs in the foreground and redeploys modules as their files change.

Because deployed symlinks point at intermediate files rather than at the
module dir's files themselves (run 'peridot deploy --help' for more
information), editing a module file has no effect until the module is
deployed again. This command closes that loop automatically.

A module is watched only if both of the following hold:
	- It has already been deployed at least once. Modules that were never
	  deployed are left alone; watch never creates symlinks on its own.
	- Its module.toml sets 'watch_changes = true'.

Changing a module.toml is itself picked up, so hot reloading can be
enabled or disabled without restarting the daemon.

Changes to the global variable files in the DOTFILES_DIR/.peridot/variables
directory affect the rendered output of every module, so any change there
redeploys all watched modules.

Events are debounced, so a burst of writes results in a single deployment.
Note that this means the module's pre_deploy and post_deploy hooks run once
per batch of changes, not once per changed file.

A module that fails to deploy is logged and skipped; the daemon keeps
running so that the problem can be fixed in place.

The dotfiles dir is resolved via the --dir flag, then the
PERIDOT_DOTFILES_DIR environment variable, then by searching upward from
the current directory. When running under a service manager such as
systemd, set it explicitly, since the working directory is not meaningful
in that context:

	[Service]
	Type=simple
	Environment=PERIDOT_DOTFILES_DIR=%h/.dotfiles
	ExecStart=%h/.local/bin/peridot watch
	Restart=on-failure
`

var WatchCommand cli.Command = cli.Command{
	Name:        "watch",
	Aliases:     []string{"w"},
	Usage:       "watch module dirs and redeploy them on change",
	Description: watchCommandDescription,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:      "dir",
			Aliases:   []string{"d"},
			Value:     "",
			Usage:     "path of the dotfiles dir to watch",
			TakesFile: true,
		},
		&cli.DurationFlag{
			Name:  "debounce",
			Value: DefaultDebounce,
			Usage: "how long to wait for changes to settle before deploying",
		},
		&cli.BoolFlag{
			Name:    "dotreplace",
			Aliases: []string{"D"},
			Value:   false,
			Usage:   "rename both the intermediate file and the symlink to the deployed\nfiles, from dot-* to .*",
		},
		&cli.StringFlag{
			Name:      "root",
			Aliases:   []string{"r"},
			Value:     "",
			Usage:     "specify the root path to which the module dir's structure should\nbe deployed",
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
						Usage:   "forcefully replaces existing files in the filesystem by removing\nthem and creating the symlink",
					},
				},
				{
					&cli.BoolFlag{
						Name:    "adopt",
						Aliases: []string{"a"},
						Value:   false,
						Usage:   "Imports existing files by copying their contents into the module,\nthen removes the originals and replaces them with symlinks",
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

		if dir := c.String("dir"); dir != "" {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("could not get the current dir: %w", err)
			}

			resolved, err := paths.ResolvePath(dir, cwd)
			if err != nil {
				return fmt.Errorf("could not resolve path specified by the --dir flag: %w", err)
			}
			appCtx.DotfilesDir = resolved
		}

		cmdCfg := &WatchCommandConfig{
			WatchDir:   appCtx.DotfilesDir,
			Debounce:   c.Duration("debounce"),
			Overwrite:  c.Bool("overwrite"),
			Adopt:      c.Bool("adopt"),
			Dotreplace: c.Bool("dotreplace"),
			Root:       c.String("root"),
			Verbose:    c.Bool("verbose"),
			Quiet:      c.Bool("quiet"),
		}

		return ExecuteWatch(ctx, cmdCfg, appCtx)
	},
}

func ExecuteWatch(ctx context.Context, cmdCfg *WatchCommandConfig, appCtx *appcontext.Context) error {
	dotfilesDir := appCtx.DotfilesDir

	// paths.DotfilesDir falls back to the current directory when it finds no
	// marker, which under a service manager would silently watch nothing.
	if _, err := os.Stat(paths.StateFilePath(dotfilesDir)); err != nil {
		return fmt.Errorf("%s is not a peridot dotfiles dir (no %s); "+
			"pass --dir or set %s: %w", dotfilesDir,
			filepath.Join(paths.PeridotDirName, paths.StateFileName),
			paths.DotfilesDirEnvName, err)
	}

	if err := logger.InitFileLogging(dotfilesDir); err != nil {
		return fmt.Errorf("could not init file logging: %w", err)
	}
	defer logger.CloseDefaultLogFile()
	logger.SetVerboseMode(cmdCfg.Verbose)
	logger.SetQuietMode(cmdCfg.Quiet)

	if cmdCfg.Debounce <= 0 {
		return fmt.Errorf("the debounce interval must be positive, got %s", cmdCfg.Debounce)
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	w, err := watcher.New(dotfilesDir, cmdCfg.Debounce)
	if err != nil {
		return fmt.Errorf("could not create watcher: %w", err)
	}
	defer w.Close()

	deployCfg := cmdCfg.deployConfig()

	if err := reconcileWatchSet(dotfilesDir, w); err != nil {
		return fmt.Errorf("could not build the initial watch set: %w", err)
	}

	logger.Info("Watching for changes", "dir", dotfilesDir,
		"modules", fmt.Sprint(w.WatchedModules()), "debounce", cmdCfg.Debounce.String())

	go func() {
		if err := w.Run(ctx); err != nil {
			logger.Error(fmt.Sprintf("watcher stopped: %s", err))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopped watching for changes")
			return nil

		case ev, ok := <-w.Events():
			if !ok {
				return nil
			}

			switch ev.Kind {
			case watcher.GlobalVarsChanged:
				logger.Info("Global variables changed, redeploying all watched modules")
				redeployModules(dotfilesDir, w.WatchedModules(), deployCfg)

			case watcher.ModuleChanged:
				redeployModules(dotfilesDir, ev.Modules, deployCfg)

			case watcher.StateChanged:
				logger.Debug("State file changed, reconciling the watch set")
			}

			if err := reconcileWatchSet(dotfilesDir, w); err != nil {
				logger.Error(fmt.Sprintf("could not reconcile the watch set: %s", err))
			}
		}
	}
}

func (c *WatchCommandConfig) deployConfig() *DeployCommandConfig {
	return &DeployCommandConfig{
		Simulate:   false,
		Overwrite:  c.Overwrite,
		Adopt:      c.Adopt,
		Dotreplace: c.Dotreplace,
		Root:       c.Root,
		Verbose:    c.Verbose,
		Quiet:      c.Quiet,
	}
}

func redeployModules(dotfilesDir string, names []string, deployCfg *DeployCommandConfig) {
	if len(names) == 0 {
		return
	}

	release, err := state.Acquire(dotfilesDir)
	if err != nil {
		logger.Error(fmt.Sprintf("could not acquire state lock: %s", err))
		return
	}
	defer release()

	st, err := state.LoadState(dotfilesDir)
	if err != nil {
		logger.Error(fmt.Sprintf("could not load state: %s", err))
		return
	}

	vars, err := templating.LoadGlobalVars(dotfilesDir)
	if err != nil {
		logger.Error(fmt.Sprintf("could not load global variables: %s", err))
		return
	}

	if err := st.Refresh(dotfilesDir); err != nil {
		logger.Error(fmt.Sprintf("could not refresh state: %s", err))
		return
	}

	for _, name := range names {
		if err := deployModule(dotfilesDir, st, name, vars, deployCfg); err != nil {
			logger.Error(fmt.Sprintf("could not deploy module %s: %s", name, err))
			continue
		}

		logger.Info("Redeployed module", "module", name)
	}

	if err := state.SaveState(st, dotfilesDir); err != nil {
		logger.Error(fmt.Sprintf("could not save state: %s", err))
	}
}

func reconcileWatchSet(dotfilesDir string, w *watcher.Watcher) error {
	release, err := state.Acquire(dotfilesDir)
	if err != nil {
		return fmt.Errorf("could not acquire state lock: %w", err)
	}

	st, err := state.LoadState(dotfilesDir)
	if err != nil {
		release()
		return fmt.Errorf("could not load state: %w", err)
	}

	if err := st.Refresh(dotfilesDir); err != nil {
		release()
		return fmt.Errorf("could not refresh state: %w", err)
	}

	release()

	known := []string{}
	deployed := []string{}
	for name, moduleState := range st.Modules {
		known = append(known, name)

		if moduleState.Status == state.NotDeployed {
			continue
		}

		mod, err := module.Load(dotfilesDir, name, moduleState)
		if err != nil {
			// A module with a broken config should not stop the others from
			// being watched.
			logger.Warn("Could not load module config, not watching it",
				"module", name, "error", err.Error())
			continue
		}

		if mod.Config.WatchChanges {
			deployed = append(deployed, name)
		}
	}

	if err := w.SetModules(deployed, known); err != nil {
		return fmt.Errorf("could not update the watch set: %w", err)
	}

	return nil
}
