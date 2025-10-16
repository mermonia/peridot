package module

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/mermonia/peridot/internal/paths"
)

const ConfigFileName string = "module.toml"

func LoadConfig(dotfilesDir, moduleName string) (*Config, error) {
	moduleDir := filepath.Join(dotfilesDir, moduleName)
	path := filepath.Join(moduleDir, paths.ModuleConfigFileName)

	c := &Config{}
	if _, err := toml.DecodeFile(path, c); err != nil {
		return nil, fmt.Errorf("could not decode module config file: %w", err)
	}

	if err := c.resolvePaths(moduleDir); err != nil {
		return nil, fmt.Errorf("could not resolve paths for module %s: %w", moduleName, err)
	}

	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("module %s has an invalid configuration: %w", moduleName, err)
	}

	return c, nil
}

func (c *Config) resolvePaths(base string) error {
	pathFields := c.GetPathFields()

	for _, field := range pathFields {
		resolved, err := paths.ResolvePath(*field.Value, base)
		if err != nil {
			return fmt.Errorf("could not resolve path field %s: %w", field.Name, err)
		}
		*field.Value = resolved
	}

	return nil
}

func (c *Config) validate() error {
	if err := c.validateRequiredFields(); err != nil {
		return err
	}

	if err := c.validatePaths(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateRequiredFields() error {
	requiredFields := []string{"root"}

	for _, field := range requiredFields {
		if field == "" {
			return fmt.Errorf("%s is a required field", field)
		}
	}

	return nil
}

func (c *Config) validatePaths() error {
	pathFields := c.GetPathFields()

	for _, field := range pathFields {
		_, err := os.Stat(*field.Value)

		if os.IsNotExist(err) {
			return fmt.Errorf("the config field %s references a non-existing path: %w", field.Name, err)
		}

		if err != nil {
			return fmt.Errorf("could not stat file: %w", err)
		}
	}

	return nil
}
