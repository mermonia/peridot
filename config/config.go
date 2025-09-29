package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/mermonia/peridot/internal/logger"
)

//go:embed default.toml
var defaultConfig []byte

type Config struct {
	DotfilesDir    string   `toml:"dotfiles_dir"`
	BackupDir      string   `toml:"backup_dir"`
	DefaultRoot    string   `toml:"default_root"`
	ManagedModules []string `toml:"managed_modules"`

	// Modules are loaded based on the managed_modules field
	Modules map[string]*ModuleConfig `toml:"-"`
}

type ModuleConfig struct {
	Root               string            `toml:"root"`
	Ignore             []string          `toml:"ignore"`
	Dependencies       []string          `toml:"dependencies"`
	ModuleDependencies []string          `toml:"module_dependencies"`
	Conditions         Conditions        `toml:"conditions"`
	Hooks              Hooks             `toml:"hooks"`
	TemplateVariables  map[string]string `toml:"variables"`
}

type Conditions struct {
	OperatingSystem string `toml:"os"`
	Hostname        string `toml:"hostname"`
	EnvRequired     string `toml:"env_exists"`
}

type Hooks struct {
	PreDeploy  string `toml:"pre_deploy"`
	PostDeploy string `toml:"post_deploy"`
	PostRemove string `toml:"post_remove"`
}

type PathProvider interface {
	UserConfigPath() (string, error)
	UserConfigDir() (string, error)
}

type DefaultPathProvider struct{}

func (p DefaultPathProvider) UserConfigDir() (string, error) {
	dir, err := os.UserConfigDir()

	if err != nil {
		return "", fmt.Errorf("Could not find user config directory: %w", err)
	}

	return dir, nil
}

func (p DefaultPathProvider) UserConfigPath() (string, error) {
	base, err := p.UserConfigDir()

	if err != nil {
		return "", err
	}

	path := filepath.Join(base, "peridot", "peridot.toml")
	return path, nil
}

type Loader struct {
	pathProvider PathProvider
}

func NewLoader(pathProvider PathProvider) *Loader {
	return &Loader{pathProvider: pathProvider}
}

func (l *Loader) LoadConfig() (*Config, error) {
	logger.Info("Starting configuration loading...")

	cfg := &Config{}
	cfg.Modules = make(map[string]*ModuleConfig)

	// Try to load embedded config
	if _, err := toml.Decode(string(defaultConfig), cfg); err != nil {
		return nil, fmt.Errorf("Failed to decode embedded config: %w", err)
	}

	// Try to load user config (e.g. ~/.config/peridot/peridot.toml)
	if path, err := l.pathProvider.UserConfigPath(); err == nil {
		err := decodeToml(path, cfg)

		if os.IsNotExist(err) {
			logger.Warn("Could not find user configuration for peridot")
		} else if err != nil {
			logger.Warn("The user configuration for peridot has an invalid format, skipping...")
		}
	}

	// Resolve relative paths
	if err := l.resolveConfigPaths(cfg); err != nil {
		return nil, err
	}

	// Validate general project config
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("Invalid configuration: %w", err)
	}

	logger.Info("Succesfully loaded configuration!")
	return cfg, nil
}

func (cfg *Config) LoadModules() (*Config, error) {
	logger.Info("Starting modules configuration loading...")
	updatedCfg := cfg.DeepCopy()

	// Load per-module configuration files
	if err := updatedCfg.loadModuleConfigFiles(); err != nil {
		return cfg, err
	}

	// Validate each module's module dependencies existence and management
	if err := updatedCfg.validateModuleDependencies(); err != nil {
		return cfg, err
	}

	// Check for circular dependencies in the module configs
	if err := updatedCfg.checkCircularModuleDependencies(); err != nil {
		return cfg, err
	}

	logger.Info("Succesfully loaded all module configurations!")
	return updatedCfg, nil
}

func (cfg *Config) loadModuleConfigFiles() error {
	for _, module := range cfg.ManagedModules {
		modulePath := filepath.Join(cfg.DotfilesDir, module, "module.toml")

		mCfg := &ModuleConfig{}
		if err := decodeToml(modulePath, mCfg); err != nil {
			return fmt.Errorf("Failed to decode module config %s: %w", module, err)
		}

		if err := mCfg.validate(); err != nil {
			return fmt.Errorf("Invalid configuration for module '%s': %w", module, err)
		}

		cfg.Modules[module] = mCfg
	}

	return nil
}

func (cfg *Config) DeepCopy() *Config {
	if cfg == nil {
		return nil
	}

	newCfg := &Config{
		DotfilesDir:    cfg.DotfilesDir,
		DefaultRoot:    cfg.DefaultRoot,
		BackupDir:      cfg.BackupDir,
		ManagedModules: append([]string{}, cfg.ManagedModules...),
		Modules:        make(map[string]*ModuleConfig),
	}

	for moduleName, mCfg := range cfg.Modules {
		newCfg.Modules[moduleName] = mCfg.DeepCopy()
	}

	return newCfg
}

func (mCfg *ModuleConfig) DeepCopy() *ModuleConfig {
	if mCfg == nil {
		return nil
	}

	newMCfg := &ModuleConfig{
		Root:               mCfg.Root,
		Ignore:             append([]string{}, mCfg.Ignore...),
		Dependencies:       append([]string{}, mCfg.Dependencies...),
		ModuleDependencies: append([]string{}, mCfg.ModuleDependencies...),
		Conditions: Conditions{
			OperatingSystem: mCfg.Conditions.OperatingSystem,
			Hostname:        mCfg.Conditions.Hostname,
			EnvRequired:     mCfg.Conditions.EnvRequired,
		},
		Hooks: Hooks{
			PreDeploy:  mCfg.Hooks.PreDeploy,
			PostDeploy: mCfg.Hooks.PostDeploy,
			PostRemove: mCfg.Hooks.PostRemove,
		},
		TemplateVariables: make(map[string]string),
	}

	for k, v := range mCfg.TemplateVariables {
		newMCfg.TemplateVariables[k] = v
	}

	return newMCfg
}

func (l *Loader) resolveConfigPaths(cfg *Config) error {
	pathFields := cfg.GetPathFields()

	base, err := l.pathProvider.UserConfigDir()

	if err != nil {
		return fmt.Errorf("Could not start resolving config's relative paths: %w", err)
	}

	for _, field := range pathFields {
		resolved, err := l.resolvePath(*field.Value, base)

		if err != nil {
			return err
		}

		*field.Value = resolved
	}

	return nil
}

func (l *Loader) resolvePath(path string, base string) (string, error) {
	// Resolve leading tildes
	if s, found := strings.CutPrefix(path, "~"); found {
		homeDir, err := os.UserHomeDir()

		if err != nil {
			return "", fmt.Errorf("Failed to find user home dir while resolving a tilde in the path %s: %w", path, err)
		}

		path = filepath.Join(homeDir, s)
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	resolved := filepath.Join(base, path)
	absPath, err := filepath.Abs(resolved)

	if err != nil {
		return "", fmt.Errorf("Could not resolve relative path: %s, %w", path, err)
	}

	return absPath, nil
}

func (cfg *Config) validate() error {
	if err := cfg.validateRequiredFields(); err != nil {
		return err
	}

	if err := cfg.validatePaths(); err != nil {
		return err
	}

	return nil
}

func (mCfg *ModuleConfig) validate() error {
	if err := mCfg.validateRequiredFields(); err != nil {
		return err
	}

	if err := mCfg.validatePaths(); err != nil {
		return err
	}

	return nil
}

func (cfg *Config) validateRequiredFields() error {
	if cfg.DotfilesDir == "" {
		return fmt.Errorf("dotfiles_dir is a required field")
	}
	return nil
}

func (mCfg *ModuleConfig) validateRequiredFields() error {
	if mCfg.Root == "" {
		return fmt.Errorf("root is a required field")
	}
	return nil
}

func (cfg *Config) validatePaths() error {
	pathFields := cfg.GetPathFields()

	for _, field := range pathFields {
		_, err := os.Stat(*field.Value)

		// Separating non-existence to optionally handle it later (via dir creation, etc.)
		if os.IsNotExist(err) {
			// return fmt.Errorf("The config field %s references a non-existing path", field.Name)
			logger.Warn("The config field references a non-existing path",
				"field", field.Name,
				"path", *field.Value)
		}

		if err != nil {
			return fmt.Errorf("The config field %s references an invalid path: %w", field.Name, err)
		}
	}

	return nil
}

func (mCfg *ModuleConfig) validatePaths() error {
	pathFields := mCfg.GetPathFields()

	for _, field := range pathFields {
		_, err := os.Stat(*field.Value)

		// Separating non-existence to optionally handle it later (via dir creation, etc.)
		if os.IsNotExist(err) {
			return fmt.Errorf("The module config field %s references a non-existing path", field.Name)
		}

		if err != nil {
			return fmt.Errorf("The module config field %s references an invalid path: %w", field.Name, err)
		}
	}

	return nil
}

func (cfg *Config) GetPathFields() []struct {
	Name  string
	Value *string
} {
	return []struct {
		Name  string
		Value *string
	}{
		{"dotfiles_dir", &cfg.DotfilesDir},
		{"default_root", &cfg.DefaultRoot},
		{"backup_dir", &cfg.BackupDir},
	}
}

func (mCfg *ModuleConfig) GetPathFields() []struct {
	Name  string
	Value *string
} {
	return []struct {
		Name  string
		Value *string
	}{
		{"root", &mCfg.Root},
	}
}

func (cfg *Config) validateModuleDependencies() error {
	managedSet := make(map[string]bool)

	for _, module := range cfg.ManagedModules {
		managedSet[module] = true
	}

	for moduleName, mCfg := range cfg.Modules {
		for _, dep := range mCfg.ModuleDependencies {
			if !managedSet[dep] {
				return fmt.Errorf(
					"The dependency '%s' from the module '%s' is not managed by peridot or does not exist",
					dep,
					moduleName)
			}
		}
	}

	return nil
}

func (cfg *Config) checkCircularModuleDependencies() error {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	modules := cfg.Modules
	for moduleName := range modules {
		if !visited[moduleName] && cfg.hasCycle(moduleName, visited, recursionStack) {
			return fmt.Errorf("Detected cyclical module dependency regarding the module %s", moduleName)
		}
	}

	return nil
}

func (cfg *Config) hasCycle(moduleName string, visited, recursionStack map[string]bool) bool {
	visited[moduleName] = true
	recursionStack[moduleName] = true

	mCfg := cfg.Modules[moduleName]

	deps := mCfg.ModuleDependencies
	for _, dep := range deps {
		if !visited[dep] {
			if cfg.hasCycle(dep, visited, recursionStack) {
				return true
			}
		} else if recursionStack[dep] {
			return true
		}
	}

	recursionStack[moduleName] = false
	return false
}

func decodeToml(path string, target any) error {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return err
	}

	if err != nil {
		return fmt.Errorf("Could not stat file at %s: %w", path, err)
	}

	if _, err := toml.DecodeFile(path, target); err != nil {
		return fmt.Errorf("Failed to decode file at %s: %w", path, err)
	}

	return nil
}
