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
	DotfilesDir string `toml:"dotfiles_dir"`
	BackupDir   string `toml:"backup_dir"`
	DefaultRoot string `toml:"default_root"`
}

func warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Warning: "+format+"\n", args)
}

func Load() (*Config, error) {
	cfg := &Config{}

	// Try to load embedded config
	if _, err := toml.Decode(string(defaultConfig), cfg); err != nil {
		return nil, fmt.Errorf("failed to decode embedded config: %w", err)
	}

	// Try to load project specific config ./peridot.conf
	if path, err := findProjectConfig(); err == nil {
		if err := decodeIfExists(path, cfg); err != nil {
			return nil, fmt.Errorf("failed to decode project config: %w", err)
		}
	} else {
		warn("could not find project config at: %s", path)
	}

	// Try to load user config ~/.config/peridot/peridot.conf
	if path, err := findUserConfig(); err == nil {
		if err := decodeIfExists(path, cfg); err != nil {
			return nil, fmt.Errorf("failed to decode user config: %w", err)
		}
	} else {
		warn("could not find user config at: %s", path)
	}

	return cfg, nil
}

func decodeIfExists(path string, conf *Config) error {
	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, conf); err != nil {
			return fmt.Errorf("failed to decode file at %s: %w", path, err)
		}
	}
	return nil
}

func findProjectConfig() (string, error) {
	base, err := os.Getwd()

	if err != nil {
		return "", err
	}

	path := filepath.Join(base, "peridot.toml")
	if _, err := os.Stat(path); err != nil {
		return path, err
	}

	return path, nil
}

func findUserConfig() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(base, "peridot", "peridot.toml")
	if _, err := os.Stat(path); err != nil {
		return path, err
	}

	return path, nil
}
