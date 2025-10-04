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
		moduleNode, err := newTree.AddNode(name)

		if err != nil {
			return nil, fmt.Errorf("Could not add node to the tree: %w", err)
		}

		// Each dir below a module dir is a node.
		// A file inside one of those dirs is a leafless node.
		for path := range module.Files {
			dirPath, file := filepath.Split(path)

			dirList := strings.Split(strings.TrimSuffix(dirPath, string(filepath.Separator)),
				string(filepath.Separator))

			lastNode := moduleNode
			for _, dir := range dirList {
				// Check if the node is the root, or an immediate child
				node := lastNode.GetNodeByValueBFS(dir, 2)
				if node == nil {
					lastNode, err = lastNode.AddNode(dir)
					if err != nil {
						return nil, err
					}
				} else {
					lastNode = node
				}
			}

			// Since a map does not allow duplicate keys, we don't have to
			// check for that.
			if _, err := lastNode.AddNode(file); err != nil {
				return nil, err
			}
		}
	}

	return newTree, nil
}
