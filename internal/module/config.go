package module

import (
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

func (mCfg *Config) DeepCopy() *Config {
	if mCfg == nil {
		return nil
	}

	newMCfg := &Config{
		Root:               mCfg.Root,
		Ignore:             append([]string{}, mCfg.Ignore...),
		Dependencies:       append([]string{}, mCfg.Dependencies...),
		ModuleDependencies: append([]string{}, mCfg.ModuleDependencies...),
		Conditions: Conditions{
			OperatingSystem: mCfg.Conditions.OperatingSystem,
			EnvRequired:     append([]string{}, mCfg.Conditions.EnvRequired...),
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
