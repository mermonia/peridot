package config

import (
	"os"
	"path/filepath"
	"testing"
)

type MockPathProvider struct {
	userConfigDir string
}

func NewMockPathProvider(userConfigDir string) *MockPathProvider {
	return &MockPathProvider{
		userConfigDir: userConfigDir,
	}
}

func (p MockPathProvider) UserConfigDir() (string, error) {
	return p.userConfigDir, nil
}

func (p MockPathProvider) UserConfigPath() (string, error) {
	base, err := p.UserConfigDir()

	if err != nil {
		return "", err
	}

	return filepath.Join(base, "peridot.toml"), nil
}

func setupTestingEnvironment(t *testing.T) (*MockPathProvider, string) {
	tempDir := t.TempDir()

	directories := map[string]string{
		"dotfiles_dir": "dotfiles",
		"backup_dir":   "backup",
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

	pathProvider := NewMockPathProvider(filepath.Join(tempDir, directories["user_cfg_dir"]))

	return pathProvider, tempDir
}

func TestLoad_normalConfig(t *testing.T) {
	pathProvider, tempDir := setupTestingEnvironment(t)

	cfg_file := `
	dotfiles_dir = "` + filepath.Join(tempDir, "dotfiles") + `"
	default_root = "` + filepath.Join(tempDir, "root") + `"
	backup_dir = "` + filepath.Join(tempDir, "backup") + `"
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
	_, err = loader.Load()

	if err != nil {
		t.Fatalf("Failed to load the config: %s", err)
	}
}
