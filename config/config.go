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
	ProjectConfigPath() (string, error)
	CurrentWorkingDir() (string, error)
}

type DefaultPathProvider struct{}

func (p DefaultPathProvider) UserConfigPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(base, "peridot", "peridot.toml")
	return path, nil
}

func (p DefaultPathProvider) ProjectConfigPath() (string, error) {
	base, err := p.CurrentWorkingDir()

	if err != nil {
		return "", err
	}

	path := filepath.Join(base, "peridot.toml")
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
		return nil, fmt.Errorf("Failed to decode embedded config: %w", err)
	}

	// Try to load user config (e.g. ~/.config/peridot/peridot.toml)
	if path, err := l.pathProvider.UserConfigPath(); err == nil {
		if err := decodeToml(path, cfg); err != nil {
			return nil, fmt.Errorf("Failed to decode user config: %w", err)
		}
	}

	// Try to load project specific config (e.g. ./peridot.toml)
	if path, err := l.pathProvider.ProjectConfigPath(); err == nil {
		if err := decodeToml(path, cfg); err != nil {
			return nil, fmt.Errorf("Failed to decode project config: %w", err)
		}
	}

	// Validate general project config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("Invalid configuration: %w", err)
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
			return fmt.Errorf("Failed to load module %s: %w", module, err)
		}

		if err := mCfg.Validate(); err != nil {
			return fmt.Errorf("Invalid configuration for module '%s': %w", module, err)
		}

		cfg.Modules = append(cfg.Modules, mCfg)
	}

	return nil
}

func (cfg *Config) Validate() error {
	if cfg.DotfilesDir == "" {
		return fmt.Errorf("dotfiles_dir is a required field")
	}
	return nil
}

func (mCfg *ModuleConfig) Validate() error {
	if mCfg.Root == "" {
		return fmt.Errorf("root is a required field")
	}
	return nil
}

func decodeToml(path string, target any) error {
	if _, err := os.Stat(path); isFileNotFoundError(err) {
		return fmt.Errorf("Could not find file at: %s", path)
	} else if err != nil {
		return fmt.Errorf("Could not stat file at: %s", path)
	}

	if _, err := toml.DecodeFile(path, target); err != nil {
		return fmt.Errorf("failed to decode file at %s: %w", path, err)
	}

	return nil
}

func isFileNotFoundError(err error) bool {
	return os.IsNotExist(err)
}
