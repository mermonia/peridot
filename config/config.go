package config

import (
	_ "embed"
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
)

//go:embed default.toml
var defaultConfig []byte

type Config struct {
	DotfilesDir    string          `toml:"dotfiles_dir"`
	BackupDir      string          `toml:"backup_dir"`
	DefaultRoot    string          `toml:"default_root"`
	ManagedModules []string        `toml:"managed_modules"`
	Modules        []*ModuleConfig `toml:"-"`
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
	CurrentWorkingDir() (string, error)
}

type DefaultPathProvider struct{}

func (p DefaultPathProvider) UserConfigDir() (string, error) {
	dir, err := os.UserConfigDir()

	if err != nil {
		return "", fmt.Errorf("Could not find user config directory: \n%w", err)
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

func (p DefaultPathProvider) CurrentWorkingDir() (string, error) {
	return os.Getwd()
}

type Loader struct {
	pathProvider PathProvider
}

func NewLoader(pathProvider PathProvider) *Loader {
	return &Loader{pathProvider: pathProvider}
}

func (l *Loader) Load() (*Config, error) {
	cfg := &Config{}

	// Try to load embedded config
	if _, err := toml.Decode(string(defaultConfig), cfg); err != nil {
		return nil, fmt.Errorf("Failed to decode embedded config: \n%w", err)
	}

	// Try to load user config (e.g. ~/.config/peridot/peridot.toml)
	if path, err := l.pathProvider.UserConfigPath(); err == nil {
		if err := decodeToml(path, cfg); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("Failed to decode user config: \n%w", err)
		}
	}

	// Resolve relative paths
	if err := l.resolveConfigPaths(cfg); err != nil {
		return nil, err
	}

	// Validate general project config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("Invalid configuration: \n%w", err)
	}

	// Load per-module configuration files
	if err := l.loadModules(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (l *Loader) loadModules(cfg *Config) error {
	for _, module := range cfg.ManagedModules {
		modulePath := filepath.Join(cfg.DotfilesDir, module, "module.toml")

		mCfg := &ModuleConfig{}
		if err := decodeToml(modulePath, mCfg); err != nil {
			return fmt.Errorf("Failed to load module %s: \n%w", module, err)
		}

		if err := mCfg.Validate(); err != nil {
			return fmt.Errorf("Invalid configuration for module '%s': \n%w", module, err)
		}

		cfg.Modules = append(cfg.Modules, mCfg)
	}

	return nil
}

func (l *Loader) resolveConfigPaths(cfg *Config) error {
	pathFields := cfg.getPathFields()

	base, err := l.pathProvider.UserConfigDir()

	if err != nil {
		return fmt.Errorf("Could not start resolving config's relative paths: \n%w", err)
	}

	for _, field := range pathFields {
		resolved, err := l.resolvePath(*field.value, base)

		if err != nil {
			return err
		}

		*field.value = resolved
	}

	return nil
}

func (l *Loader) resolvePath(path string, base string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	resolved := filepath.Join(base, path)
	absPath, err := filepath.Abs(resolved)

	if err != nil {
		return "", fmt.Errorf("Could not resolve relative path: %s, \n%w", path, err)
	}

	return absPath, nil
}

func (cfg *Config) Validate() error {
	if cfg.DotfilesDir == "" {
		return fmt.Errorf("dotfiles_dir is a required field")
	}

	pathFields := cfg.getPathFields()

	for _, field := range pathFields {
		_, err := os.Stat(*field.value)

		// Separating non-existence to optionally handle it later (via dir creation, etc.)
		if os.IsNotExist(err) {
			return fmt.Errorf("The config field %s references a non-existing path", field.name)
		}

		if err != nil {
			return fmt.Errorf("The config field %s references an invalid path", field.name)
		}
	}

	return nil
}

func (mCfg *ModuleConfig) Validate() error {
	if mCfg.Root == "" {
		return fmt.Errorf("root is a required field")
	}

	pathFields := mCfg.getPathFields()

	for _, field := range pathFields {
		_, err := os.Stat(*field.value)

		// Separating non-existence to optionally handle it later (via dir creation, etc.)
		if os.IsNotExist(err) {
			return fmt.Errorf("The config field %s references a non-existing path", field.name)
		}

		if err != nil {
			return fmt.Errorf("The config field %s references an invalid path", field.name)
		}
	}

	return nil
}

func (cfg *Config) getPathFields() []struct {
	name  string
	value *string
} {
	return []struct {
		name  string
		value *string
	}{
		{"dotfiles_dir", &cfg.DotfilesDir},
		{"default_root", &cfg.DefaultRoot},
		{"backup_dir", &cfg.BackupDir},
	}
}

func (mCfg *ModuleConfig) getPathFields() []struct {
	name  string
	value *string
} {
	return []struct {
		name  string
		value *string
	}{
		{"root", &mCfg.Root},
	}
}

func decodeToml(path string, target any) error {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return err
	}

	if err != nil {
		return fmt.Errorf("Could not stat file at: %s", path)
	}

	if _, err := toml.DecodeFile(path, target); err != nil {
		return fmt.Errorf("failed to decode file at %s: \n%w", path, err)
	}

	return nil
}
