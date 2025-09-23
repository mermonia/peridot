package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type fieldToValidate struct {
	name   string
	value  any
	target any
}

type mockPathProvider struct {
	userConfigDir string
}

func newMockPathProvider(userConfigDir string) *mockPathProvider {
	return &mockPathProvider{
		userConfigDir: userConfigDir,
	}
}

func (p mockPathProvider) UserConfigDir() (string, error) {
	return p.userConfigDir, nil
}

func (p mockPathProvider) UserConfigPath() (string, error) {
	base, err := p.UserConfigDir()

	if err != nil {
		return "", err
	}

	return filepath.Join(base, "peridot.toml"), nil
}

func setupTestingEnvironment(t *testing.T) (*mockPathProvider, string) {
	t.Helper()
	tempDir := t.TempDir()

	directories := map[string]string{
		"dotfiles_dir": "dotfiles",
		"backup_dir":   "backups",
		"root":         "root",
		"user_cfg_dir": "user_cfg_dir",
	}

	moduleDirectories := map[string]string{
		"nvim":         "nvim",
		"hyprland":     "hyprland",
		"empty_module": "empty_module",
	}

	for _, dir := range directories {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)

		if err != nil {
			t.Fatalf("Could not create directory %s", dir)
		}
	}

	for _, moduleDir := range moduleDirectories {
		err := os.MkdirAll(filepath.Join(tempDir, directories["dotfiles_dir"], moduleDir), 0755)

		if err != nil {
			t.Fatalf("Could not create directory for module %s", moduleDir)
		}

	}

	pathProvider := newMockPathProvider(filepath.Join(tempDir, directories["user_cfg_dir"]))

	return pathProvider, tempDir
}

func TestLoad_normalConfig(t *testing.T) {
	pathProvider, tempDir := setupTestingEnvironment(t)

	cfg_file := `
	dotfiles_dir = "` + filepath.Join(tempDir, "dotfiles") + `"
	default_root = "` + filepath.Join(tempDir, "root") + `"
	backup_dir = "` + filepath.Join(tempDir, "backups") + `"
	managed_modules = ["nvim", "hyprland"]
	`

	moduleConfig_files := make(map[string]string)

	moduleConfig_files["nvim"] = `
	root = "` + filepath.Join(tempDir, "root") + `"
	ignore = [".ignore"]
	dependencies = ["bash", "lua"]
	module_dependencies = ["hyprland"]

	[conditions]
	os = "Linux"
	hostname = ""
	env_exists = ""

	[hooks]
	pre_deploy = "echo 'About to deploy nvim!'"
	post_deploy = "nvim"
	post_remove = "echo 'Just removed nvim'"

	[variables]
	red = "#FF0000"
	green = "#00FF00"
	blue = "#0000FF"
	`

	moduleConfig_files["hyprland"] = `
	root = "` + filepath.Join(tempDir, "root") + `"
	ignore = [".ignore"]
	dependencies = []
	module_dependencies = []

	[conditions]
	os = "Linux"
	hostname = ""
	env_exists = ""

	[hooks]
	pre_deploy = "echo 'About to deploy hyprland!'"
	post_deploy = ""
	post_remove = "echo 'Just removed hyprland'"

	[variables]
	`

	err := os.WriteFile(filepath.Join(tempDir, "user_cfg_dir", "peridot.toml"), []byte(cfg_file), 0755)

	if err != nil {
		t.Fatalf("Could not write the configuration file!")
	}

	for name, mCfg := range moduleConfig_files {
		err := os.WriteFile(filepath.Join(tempDir, "dotfiles", name, "module.toml"), []byte(mCfg), 0755)

		if err != nil {
			t.Fatalf(`
				Could not create the module config file for %s.
				Has the directory been properly created?
				`, name)
		}
	}

	loader := NewLoader(pathProvider)
	cfg, err := loader.Load()

	if err != nil {
		t.Fatalf("Failed to load the config: %s", err)
	}

	validateConfigFields(t, cfg, tempDir)
	validateModuleConfigFields(t, cfg, tempDir)
}

func validateConfigFields(t *testing.T, cfg *Config, tempDir string) {
	t.Helper()
	configTestTable := []fieldToValidate{
		{
			name:   "dotfiles_dir",
			value:  cfg.DotfilesDir,
			target: filepath.Join(tempDir, "dotfiles"),
		},
		{
			name:   "backup_dir",
			value:  cfg.BackupDir,
			target: filepath.Join(tempDir, "backups"),
		},
		{
			name:   "root",
			value:  cfg.DefaultRoot,
			target: filepath.Join(tempDir, "root"),
		},
		{
			name:   "number of modules",
			value:  len(cfg.Modules),
			target: 2,
		},
		{
			name:   "nvim module existence",
			value:  cfg.Modules["nvim"] != nil,
			target: true,
		},
		{
			name:   "hyprland module existence",
			value:  cfg.Modules["hyprland"] != nil,
			target: true,
		},
	}

	for _, field := range configTestTable {
		validateField(field, t)
	}
}

func validateModuleConfigFields(t *testing.T, cfg *Config, tempDir string) {
	nvimMcfg := cfg.Modules["nvim"]
	nvimConfigTestTable := []fieldToValidate{
		{
			name:   "nvim root",
			value:  nvimMcfg.Root,
			target: filepath.Join(tempDir, "root"),
		},
		{
			name:   "nvim ignore",
			value:  nvimMcfg.Ignore,
			target: []string{".ignore"},
		},
		{
			name:   "nvim dependencies",
			value:  nvimMcfg.Dependencies,
			target: []string{"bash", "lua"},
		},
		{
			name:   "nvim module dependencies",
			value:  nvimMcfg.ModuleDependencies,
			target: []string{"hyprland"},
		},
		{
			name:  "nvim conditions",
			value: nvimMcfg.Conditions,
			target: Conditions{
				OperatingSystem: "Linux",
				Hostname:        "",
				EnvRequired:     "",
			},
		},
		{
			name:  "nvim hooks",
			value: nvimMcfg.Hooks,
			target: Hooks{
				PreDeploy:  "echo 'About to deploy nvim!'",
				PostDeploy: "nvim",
				PostRemove: "echo 'Just removed nvim'",
			},
		},
	}

	for _, field := range nvimConfigTestTable {
		validateField(field, t)
	}

	hyprlandMcfg := cfg.Modules["hyprland"]
	hyprlandConfigTestTable := []fieldToValidate{
		{
			name:   "hyprland root",
			value:  hyprlandMcfg.Root,
			target: filepath.Join(tempDir, "root"),
		},
		{
			name:   "hyprland ignore",
			value:  hyprlandMcfg.Ignore,
			target: []string{".ignore"},
		},
		{
			name:   "hyprland dependencies",
			value:  hyprlandMcfg.Dependencies,
			target: []string{},
		},
		{
			name:   "hyprland module dependencies",
			value:  hyprlandMcfg.ModuleDependencies,
			target: []string{},
		},
		{
			name:  "hyprland conditions",
			value: hyprlandMcfg.Conditions,
			target: Conditions{
				OperatingSystem: "Linux",
				Hostname:        "",
				EnvRequired:     "",
			},
		},
		{
			name:  "hyprland hooks",
			value: hyprlandMcfg.Hooks,
			target: Hooks{
				PreDeploy:  "echo 'About to deploy hyprland!'",
				PostDeploy: "",
				PostRemove: "echo 'Just removed hyprland'",
			},
		},
	}

	for _, field := range hyprlandConfigTestTable {
		validateField(field, t)
	}
}

func validateField(f fieldToValidate, t *testing.T) {
	t.Helper()
	if !reflect.DeepEqual(f.value, f.target) {
		t.Errorf("The field %q should be \"%v\", got \"%v\" instead.", f.name, f.target, f.value)
	}
}
