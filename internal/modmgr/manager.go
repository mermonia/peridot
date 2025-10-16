package modmgr

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mermonia/peridot/internal/appcontext"
	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/module"
	"github.com/mermonia/peridot/internal/state"
)

func AddModule(moduleName string, appCtx *appcontext.Context) error {
	dotfilesDir := appCtx.DotfilesDir

	if err := createModuleIfMissing(moduleName, dotfilesDir); err != nil {
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
