package module

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/state"
)

type Module struct {
	Name   string
	Config *Config
	State  *state.ModuleState
}

func Load(dotfilesDir, moduleName string, moduleState *state.ModuleState) (*Module, error) {
	c, err := LoadConfig(dotfilesDir, moduleName)
	if err != nil {
		return nil, fmt.Errorf("could not load config: %w", err)
	}

	return &Module{
		Name:   moduleName,
		Config: c,
		State:  moduleState,
	}, nil
}

func LoadAll(dotfilesDir string, appState *state.State) ([]*Module, error) {
	modules := make([]*Module, 0, len(appState.Modules))

	for name, moduleState := range appState.Modules {
		mod, err := Load(dotfilesDir, name, moduleState)
		if err != nil {
			return nil, fmt.Errorf("could not load module %s: %w", name, err)
		}

		modules = append(modules, mod)
	}

	return modules, nil
}

func (m *Module) ShouldDeploy(appState *state.State) bool {
	if err := m.CheckBinaryDependencies(); err != nil {
		logger.Warn("Binary dependencies missing", "error", err.Error())
		return false
	}

	if err := m.CheckModuleDependencies(appState); err != nil {
		logger.Warn("Module dependencies missing", "error", err.Error())
		return false
	}

	if err := m.CheckConditions(); err != nil {
		logger.Warn("Module condition not fullfilled", "error", err.Error())
		return false
	}

	return true
}

func (m *Module) CheckBinaryDependencies() error {
	missing := []string{}

	for _, dep := range m.Config.Dependencies {
		if !binaryExists(dep) {
			missing = append(missing, dep)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("module %s has missing dependencies: [%s]", m.Name,
			strings.Join(missing, ", "))
	}

	return nil
}

func binaryExists(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func (m *Module) CheckModuleDependencies(appState *state.State) error {
	missing := []string{}

	for _, dep := range m.Config.ModuleDependencies {
		if appState.Modules[dep] == nil {
			missing = append(missing, dep)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("moudule %s requires modules [%s] to be deployed", m.Name,
			strings.Join(missing, ", "))
	}

	return nil
}

func (m *Module) CheckConditions() error {
	requiredOs := strings.ToLower(m.Config.Conditions.OperatingSystem)
	if requiredOs != runtime.GOOS {
		return fmt.Errorf("module %s requires os to be %s", m.Name, requiredOs)
	}

	for _, envvar := range m.Config.Conditions.EnvRequired {
		if _, exists := os.LookupEnv(envvar); !exists {
			return fmt.Errorf("module %s requires environment variable %s to be set", m.Name, envvar)
		}
	}

	return nil
}
