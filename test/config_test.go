package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mermonia/peridot/config"
)

type TestPathProvider struct{}

func (p TestPathProvider) UserConfigPath() (string, error) {
	base, err := p.CurrentWorkingDir()

	if err != nil {
		return "", err
	}

	return filepath.Join(base, "data", "config", "example-config.toml"), nil
}

func (p TestPathProvider) ProjectConfigPath() (string, error) {
	base, err := p.CurrentWorkingDir()

	if err != nil {
		return "", err
	}

	return filepath.Join(base, "data", "config", "example-config.toml"), nil
}

func (p TestPathProvider) CurrentWorkingDir() (string, error) {
	return os.Getwd()
}

// TODO: Actual test validation
func TestLoad(t *testing.T) {
	l := config.NewLoader(TestPathProvider{})
	cfg, err := l.Load()

	fmt.Println(cfg, err)
}
