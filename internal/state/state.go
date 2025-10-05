package state

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mermonia/peridot/internal/tree"
)

type State struct {
	// Modules[moduleName] = &ModuleState
	Modules map[string]*ModuleState
}

type ModuleState struct {
	DeployedAt time.Time
	// Entries[".config/kitty/kitty.conf"] =
	Files map[string]*Entry
}

type Entry struct {
	SourceHash       string
	IntermediatePath string
	Target           string
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
	moduleNode := tree.NewTree(name)

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
