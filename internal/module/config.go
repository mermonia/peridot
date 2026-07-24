package module

import (
	"maps"
	_ "embed"
)

//go:embed default-module.toml
var DefaultConfig []byte

type Config struct {
	Root               string            `toml:"root"`
	Ignore             []string          `toml:"ignore"`
	Dependencies       []string          `toml:"dependencies"`
	ModuleDependencies []string          `toml:"module_dependencies"`
	Conditions         Conditions        `toml:"conditions"`
	Hooks              Hooks             `toml:"hooks"`
	TemplateVariables  map[string]string `toml:"variables"`
}

type Conditions struct {
	OperatingSystem string   `toml:"os"`
	EnvRequired     []string `toml:"env_exists"`
}

type Hooks struct {
	PreDeploy  string `toml:"pre_deploy"`
	PostDeploy string `toml:"post_deploy"`
	PostRemove string `toml:"post_remove"`
}

type PathField struct {
	Name  string
	Value *string
}

func (c *Config) GetPathFields() []PathField {
	return []PathField{
		{Name: "root", Value: &c.Root},
	}
}

func (c *Config) DeepCopy() *Config {
	if c == nil {
		return nil
	}

	newMCfg := &Config{
		Root:               c.Root,
		Ignore:             append([]string{}, c.Ignore...),
		Dependencies:       append([]string{}, c.Dependencies...),
		ModuleDependencies: append([]string{}, c.ModuleDependencies...),
		Conditions: Conditions{
			OperatingSystem: c.Conditions.OperatingSystem,
			EnvRequired:     append([]string{}, c.Conditions.EnvRequired...),
		},
		Hooks: Hooks{
			PreDeploy:  c.Hooks.PreDeploy,
			PostDeploy: c.Hooks.PostDeploy,
			PostRemove: c.Hooks.PostRemove,
		},
		TemplateVariables: make(map[string]string),
	}

	maps.Copy(newMCfg.TemplateVariables, c.TemplateVariables)
	return newMCfg
}
