package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/paths"
)

//go:embed default.toml
var DefaultConfig []byte

//go:embed default-module.toml
var DefaultModuleConfig []byte

var GeneralConfigFileName string = "peridot.toml"
var ModuleConfigFileName string = "module.toml"

type ConfigSource int

const (
	Embedded ConfigSource = iota
	UserConfigPath
)

var sourceName = map[ConfigSource]string{
	Embedded:       "embedded",
	UserConfigPath: "user-config-path",
}

func (cs ConfigSource) String() string {
	return sourceName[cs]
}

type Config struct {
	DotfilesDir    string   `toml:"dotfiles_dir"`
	BackupDir      string   `toml:"backup_dir"`
	DefaultRoot    string   `toml:"default_root"`
	ManagedModules []string `toml:"managed_modules"`

	// modules are loaded based on the managed_modules field
	modules map[string]*ModuleConfig `toml:"-"`
	source  ConfigSource
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

	path := filepath.Join(base, "peridot", GeneralConfigFileName)
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
	cfg.modules = make(map[string]*ModuleConfig)

	// Try to load embedded config
	if _, err := toml.Decode(string(DefaultConfig), cfg); err != nil {
		return nil, fmt.Errorf("Failed to decode embedded config: %w", err)
	} else {
		cfg.source = Embedded
	}

	// Try to load user config (e.g. ~/.config/peridot/peridot.toml)
	if path, err := l.pathProvider.UserConfigPath(); err == nil {
		err := readConfigFromFile(path, cfg)

		if os.IsNotExist(err) {
			logger.Warn("Could not find user configuration for peridot", "path", path)
		} else if err != nil {
			logger.Warn("The user configuration for peridot has an invalid format, skipping...")
		} else {
			cfg.source = UserConfigPath
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

func (l *Loader) LoadModules(cfg *Config) (*Config, error) {
	logger.Info("Starting modules configuration loading...")
	updatedCfg := cfg.DeepCopy()

	// Load per-module configuration files
	if err := l.loadModuleConfigFiles(updatedCfg); err != nil {
		return cfg, err
	}

	// Resolve relative paths
	if err := l.resolveModuleConfigPaths(updatedCfg); err != nil {
		return cfg, err
	}

	// Validate general modules config
	if err := updatedCfg.validateModules(); err != nil {
		return cfg, err
	}

	// Validate each module's module dependencies existence and management
	if err := updatedCfg.checkModuleDependencies(); err != nil {
		return cfg, err
	}

	// Check for circular dependencies in the module configs
	if err := updatedCfg.checkCircularModuleDependencies(); err != nil {
		return cfg, err
	}

	logger.Info("Succesfully loaded all module configurations!")
	return updatedCfg, nil
}

func (l *Loader) loadModuleConfigFiles(cfg *Config) error {
	for _, module := range cfg.ManagedModules {
		modulePath := filepath.Join(cfg.DotfilesDir, module, ModuleConfigFileName)

		mCfg := &ModuleConfig{}
		if err := readConfigFromFile(modulePath, mCfg); err != nil {
			return fmt.Errorf("Failed to decode module config %s: %w", module, err)
		}

		cfg.modules[module] = mCfg
	}

	return nil
}

func (cfg *Config) validateModules() error {
	for moduleName, mCfg := range cfg.modules {
		if err := mCfg.validate(); err != nil {
			return fmt.Errorf("Invalid configuration for module '%s': %w", moduleName, err)
		}
	}
	return nil
}

func (l *Loader) resolveModuleConfigPaths(cfg *Config) error {
	for moduleName, mCfg := range cfg.modules {
		pathFields := mCfg.GetPathFields()
		base := filepath.Join(cfg.DotfilesDir, moduleName)

		for _, field := range pathFields {
			newPath, err := paths.ResolvePath(*field.Value, base)

			if err != nil {
				return fmt.Errorf("Could not resolve path fields for module %s: %w", moduleName, err)
			}

			*field.Value = newPath
		}
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
		modules:        make(map[string]*ModuleConfig),
		source:         cfg.source,
	}

	for moduleName, mCfg := range cfg.modules {
		newCfg.modules[moduleName] = mCfg.DeepCopy()
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
		resolved, err := paths.ResolvePath(*field.Value, base)

		if err != nil {
			return err
		}

		*field.Value = resolved
	}

	return nil
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
			return fmt.Errorf("The module config field %s references a non-existing path: %w", field.Name, err)
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

func (cfg *Config) checkModuleDependencies() error {
	managedSet := make(map[string]bool)

	for _, module := range cfg.ManagedModules {
		managedSet[module] = true
	}

	for moduleName, mCfg := range cfg.modules {
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

	modules := cfg.modules
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

	mCfg := cfg.modules[moduleName]

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

func readConfigFromFile(path string, target any) error {
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

func writeConfigToFile(source any, path string) error {
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(source); err != nil {
		return fmt.Errorf("Could not encode the given struct to toml: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0766); err != nil {
		return fmt.Errorf("Could not write encoded config to file: %w", err)
	}

	return nil
}

func (l *Loader) OverwriteConfig(cfg *Config) error {
	if cfg.source == Embedded {
		logger.Warn("Peridot is using the embbeded config, skipping config file override...")
		return nil
	}

	path, err := l.pathProvider.UserConfigPath()
	if err != nil {
		return fmt.Errorf("Could not overwrite configuration: %w", err)
	}

	return writeConfigToFile(cfg, path)
}
