package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mermonia/peridot/config"
	"github.com/mermonia/peridot/internal/hash"
	"github.com/mermonia/peridot/internal/tree"
)

// State should only be created once, via peridot init.
// Modifications to state should only be made after loading it from
// a state file, and the state file should be updated right after.
type State struct {
	Modules map[string]*ModuleState `json:"modules"`
}

type ModuleState struct {
	Status     DeployStatus      `json:"status"`
	DeployedAt time.Time         `json:"deployedAt"`
	Files      map[string]*Entry `json:"files"`
}

type Entry struct {
	Status           DeployStatus `json:"status"`
	SourceHash       string       `json:"hash"`
	IntermediatePath string       `json:"intermediatePath"`
	Target           string       `json:"target"`
}

type DeployStatus int

const (
	NotDeployed DeployStatus = iota
	Unsynced
	Synced
)

var StateFilePath string = filepath.Join(".cache", "state.json")

func LoadState() (*State, error) {
	l := config.NewConfigLoader(config.DefaultConfigPathProvider{})
	cfg, err := l.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("Could not load config: %w", err)
	}

	state := &State{}
	stateFile, err := os.ReadFile(filepath.Join(cfg.DotfilesDir, StateFilePath))
	if err != nil {
		return nil, fmt.Errorf("Could not read state file: %w", err)
	}

	if err := json.Unmarshal(stateFile, state); err != nil {
		return nil, fmt.Errorf("Could not decode json state: %w", err)
	}

	return state, nil
}

func SaveState(state *State) error {
	l := config.NewConfigLoader(config.DefaultConfigPathProvider{})
	cfg, err := l.LoadConfig()
	if err != nil {
		return fmt.Errorf("Could not load config: %w", err)
	}

	stateFile, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("Could not encode json state: %w", err)
	}

	if err := os.WriteFile(filepath.Join(cfg.DotfilesDir, StateFilePath), stateFile, 0766); err != nil {
		return fmt.Errorf("Could not write state file: %w", err)
	}

	return nil
}

func GetStateFileTree(state *State) (*tree.Node, error) {
	newTree := tree.NewTree(".")

	// Systematically add nodes to the tree
	for name, module := range state.Modules {
		// Each module is a first-level node
		moduleNode, err := GetModuleFileTree(name, module)
		if err != nil {
			return nil, fmt.Errorf("Could not get moudule file tree: %w", err)
		}

		if err := newTree.Add(moduleNode); err != nil {
			return nil, fmt.Errorf("Could not add node to the tree: %w", err)
		}
	}

	return newTree, nil
}

func GetModuleFileTree(name string, module *ModuleState) (*tree.Node, error) {
	formattedStatus := getFormattedModuleStatus(name, module)
	moduleNode := tree.NewTree(formattedStatus)

	// Each dir below a module dir is a node.
	// A file inside one of those dirs is a leafless node.
	for path := range module.Files {
		dirPath, file := filepath.Split(path)
		dirPath = strings.TrimSuffix(dirPath, string(filepath.Separator))

		dirList := strings.Split(dirPath, string(filepath.Separator))

		var err error
		lastNode := moduleNode

		for _, dir := range dirList {
			// Check if the node is the root, or an immediate child
			node := lastNode.GetNodeByValueBFS(dir, 2)
			if node == nil {
				lastNode, err = lastNode.AddValue(dir)
				if err != nil {
					return nil, err
				}
			} else {
				lastNode = node
			}
		}

		// Since a map does not allow duplicate keys, we don't have to
		// check for that.
		if _, err := lastNode.AddValue(file); err != nil {
			return nil, err
		}
	}
	return moduleNode, nil
}

func getFormattedModuleStatus(name string, module *ModuleState) string {
	formattedStatus := ""

	switch module.Status {
	case NotDeployed:
		formattedStatus = "○ " + name + " - not deployed"
	case Unsynced:
		formattedStatus = "✗ " + name + " - deployed, pending sync"
	case Synced:
		formattedStatus = "✓ " + name + " - deployed and up to date"
	default:
		formattedStatus = "? " + name + " - status unknown"

	}

	return formattedStatus
}

func (s *State) UpdateDeploymentStatus() error {
	for _, module := range s.Modules {
		if module.Status != NotDeployed {
			for path, file := range module.Files {
				updatedHash, err := hash.HashFile(path)
				if err != nil {
					return fmt.Errorf("Could not hash file %s: %w", path, err)
				}

				if updatedHash != file.SourceHash {
					file.Status = Unsynced
					module.Status = Unsynced
				}
			}
		}
	}

	return nil
}
