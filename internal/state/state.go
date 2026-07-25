package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/mermonia/peridot/internal/hash"
	"github.com/mermonia/peridot/internal/paths"
	"github.com/mermonia/peridot/internal/templating"
	"github.com/mermonia/peridot/internal/tree"
)

// State should only be created once, via peridot init.
// Modifications to state should only be made after loading it from
// a state file, and the state file should be updated right after.
type State struct {
	Modules       map[string]*ModuleState       `json:"modules"`
	VariableFiles map[string]*VariableFileEntry `json:"variableFiles"`
}

type ModuleState struct {
	Status     DeployStatus                `json:"status"`
	DeployedAt time.Time                   `json:"deployedAt"`
	Files      map[string]*ModuleFileEntry `json:"files"`
}

type ModuleFileEntry struct {
	Status           DeployStatus `json:"status"`
	SourceHash       string       `json:"hash"`
	IntermediatePath string       `json:"intermediatePath"`
	SymlinkPath      string       `json:"symlinkPath"`
}

type VariableFileEntry struct {
	SourceHash string `json:"hash"`
}

type DeployStatus int

const (
	NotDeployed DeployStatus = iota
	Unsynced
	Synced
)

func LoadState(dotfilesDir string) (*State, error) {
	state := &State{}
	stateFile, err := os.ReadFile(paths.StateFilePath(dotfilesDir))
	if err != nil {
		return nil, fmt.Errorf("could not read state file: %w", err)
	}

	if err := json.Unmarshal(stateFile, state); err != nil {
		return nil, fmt.Errorf("could not decode json state: %w", err)
	}

	if state.Modules == nil {
		state.Modules = map[string]*ModuleState{}
	}

	if state.VariableFiles == nil {
		state.VariableFiles = map[string]*VariableFileEntry{}
	}

	return state, nil
}

// SaveState writes the state file atomically: the contents go to a
// temporary file in the same directory, which is then renamed over the
// real one. A crash mid-write therefore leaves either the old state or
// the new one, never a truncated file. This matters because state.json
// doubles as the marker that identifies a dotfiles dir.
func SaveState(state *State, dotfilesDir string) error {
	stateFile, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("could not encode json state: %w", err)
	}

	statePath := paths.StateFilePath(dotfilesDir)

	tmp, err := os.CreateTemp(filepath.Dir(statePath), "."+paths.StateFileName+".tmp")
	if err != nil {
		return fmt.Errorf("could not create temporary state file: %w", err)
	}
	tmpPath := tmp.Name()

	// Best effort cleanup: a successful rename makes this a no-op.
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(stateFile); err != nil {
		tmp.Close()
		return fmt.Errorf("could not write temporary state file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("could not sync temporary state file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not close temporary state file: %w", err)
	}

	// CreateTemp uses 0600; match the permissions of a plain write.
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return fmt.Errorf("could not set state file permissions: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		return fmt.Errorf("could not replace state file: %w", err)
	}

	return nil
}

func GetStateFileTree(state *State, dotfilesDir string, simple bool) (*tree.Node, error) {
	newTree := tree.NewTree(".")

	// Systematically add nodes to the tree
	for name, module := range state.Modules {
		// Each module is a first-level node
		moduleNode, err := GetModuleFileTree(name, module, dotfilesDir, simple)
		if err != nil {
			return nil, fmt.Errorf("could not get moudule file tree: %w", err)
		}

		if err := newTree.Add(moduleNode); err != nil {
			return nil, fmt.Errorf("could not add node to the tree: %w", err)
		}
	}

	return newTree, nil
}

func GetModuleFileTree(name string, module *ModuleState, dotfilesDir string, simple bool) (*tree.Node, error) {
	formattedStatus := getFormattedModuleStatus(name, module)
	moduleNode := tree.NewTree(formattedStatus)

	// Each dir below a module dir is a node.
	// A file inside one of those dirs is a leafless node.
	for path, entry := range module.Files {
		path, err := filepath.Rel(paths.ModuleDir(dotfilesDir, name), path)
		if err != nil {
			return nil, err
		}

		dirPath, fileName := filepath.Split(path)
		dirList := paths.SplitPath(dirPath)

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
		formattedFileStatus := getFormattedFileStatus(fileName, entry, simple)
		if _, err := lastNode.AddValue(formattedFileStatus); err != nil {
			return nil, err
		}
	}
	return moduleNode, nil
}

func (s *State) Refresh(dotfilesDir string) error {
	s.cleanModules(dotfilesDir)

	globalVarsChanged, err := s.syncVariableFiles(dotfilesDir)
	if err != nil {
		return err
	}

	return s.updateDeploymentStatus(globalVarsChanged)
}

func (s *State) syncVariableFiles(dotfilesDir string) (bool, error) {
	filePaths, err := templating.ListGlobalVarsFiles(dotfilesDir)
	if err != nil {
		return false, fmt.Errorf("could not list global variable files: %w", err)
	}

	globalVarsChanged := false

	for path := range s.VariableFiles {
		if !slices.Contains(filePaths, path) {
			delete(s.VariableFiles, path)
			globalVarsChanged = true
		}
	}

	for _, path := range filePaths {
		updatedHash, err := hash.HashFile(path)
		if err != nil {
			return false, fmt.Errorf("could not hash file %s: %w", path, err)
		}

		file, isTracked := s.VariableFiles[path]
		if !isTracked {
			s.VariableFiles[path] = &VariableFileEntry{SourceHash: updatedHash}
			globalVarsChanged = true
			continue
		}

		if updatedHash != file.SourceHash {
			globalVarsChanged = true
		}

		file.SourceHash = updatedHash
	}

	return globalVarsChanged, nil
}

func (s *State) updateDeploymentStatus(globalVarsChanged bool) error {
	for _, module := range s.Modules {
		if module.Status != NotDeployed {
			for path, file := range module.Files {
				updatedHash, err := hash.HashFile(path)
				if err != nil {
					return fmt.Errorf("could not hash file %s: %w", path, err)
				}

				if updatedHash != file.SourceHash || globalVarsChanged {
					file.Status = Unsynced
					module.Status = Unsynced
				}

				file.SourceHash = updatedHash
			}
		}
	}

	return nil
}

func (s *State) cleanModules(dotfilesDir string) {
	for name, module := range s.Modules {
		for path := range module.Files {
			if _, err := os.Stat(path); err != nil {
				delete(module.Files, path)
			}
		}

		if _, err := os.Stat(paths.ModuleDir(dotfilesDir, name)); err != nil {
			delete(s.Modules, name)
		}
	}
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

func getFormattedFileStatus(name string, entry *ModuleFileEntry, simple bool) string {
	formattedFileStatus := ""
	formattedSymlink := ""
	if !simple {
		formattedSymlink = " <- " + entry.SymlinkPath
	}

	switch entry.Status {
	case NotDeployed:
		formattedFileStatus = name
	case Unsynced:
		formattedFileStatus = "✗ " + name + formattedSymlink
	case Synced:
		formattedFileStatus = "✓ " + name + formattedSymlink
	default:
		formattedFileStatus = "? " + name
	}

	return formattedFileStatus
}
