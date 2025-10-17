package modmgr

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mermonia/peridot/internal/appcontext"
	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/module"
	"github.com/mermonia/peridot/internal/paths"
	"github.com/mermonia/peridot/internal/state"
	"github.com/mermonia/peridot/internal/templating"
)

func AddModule(moduleName string, appCtx *appcontext.Context) error {
	if err := createModuleIfMissing(moduleName, appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not add module %s: %w", moduleName, err)
	}

	st, err := state.LoadState(appCtx.DotfilesDir)
	if err != nil {
		return fmt.Errorf("could not load state: %w", err)
	}

	if err := st.Refresh(appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not refresh state: %w", err)
	}

	if st.Modules[moduleName] == nil {
		st.Modules[moduleName] = &state.ModuleState{
			Status: state.NotDeployed,
			Files:  make(map[string]*state.Entry),
		}
	}

	if err := state.SaveState(st, appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not save state: %w", err)
	}

	logger.Info("Successfully added module", "module", moduleName)
	return nil
}

func RemoveModule(moduleName string, appCtx *appcontext.Context) error {
	st, err := state.LoadState(appCtx.DotfilesDir)
	if err != nil {
		return fmt.Errorf("could not load state: %w", err)
	}

	if err := st.Refresh(appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not refresh state: %w", err)
	}

	moduleState := st.Modules[moduleName]
	if moduleState == nil {
		return nil
	}

	mod, err := module.Load(appCtx.DotfilesDir, moduleName, moduleState)
	if err != nil {
		return fmt.Errorf("could not load module: %w", err)
	}

	for path, entry := range moduleState.Files {
		if err := removeIfSymlink(entry.SymlinkPath); err != nil {
			return err
		}

		if err := templating.CreateRenderedFile(path, entry.SymlinkPath, mod.Config.TemplateVariables); err != nil {
			return fmt.Errorf("could not create rendered file: %w", err)
		}
	}

	if err := os.RemoveAll(paths.ModuleDir(appCtx.DotfilesDir, moduleName)); err != nil {
		return fmt.Errorf("could not remove module dir: %w", err)
	}

	if err := st.Refresh(appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not refresh state: %w", err)
	}

	if err := state.SaveState(st, appCtx.DotfilesDir); err != nil {
		return fmt.Errorf("could not save state: %w", err)
	}

	return nil
}

func removeIfSymlink(path string) error {
	if path == "" {
		return nil
	}

	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("could not stat: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(path)
	}

	return nil
}

func createModuleIfMissing(moduleName string, dotfilesDir string) error {
	moduleDir := filepath.Join(dotfilesDir, moduleName)
	moduleConfigPath := filepath.Join(moduleDir, module.ConfigFileName)

	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", moduleDir, err)
	}

	if _, err := os.Stat(moduleConfigPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("could not stat config file %s: %w", moduleConfigPath, err)
	}

	if err := os.WriteFile(moduleConfigPath, module.DefaultConfig, 0644); err != nil {
		return fmt.Errorf("could not create config file %s: %w", moduleConfigPath, err)
	}

	return nil
}
