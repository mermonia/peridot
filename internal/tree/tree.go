package tree

import (
	"fmt"
	"io"
	"strings"

	"github.com/mermonia/peridot/internal/logger"
)

type Node struct {
	Nodes []*Node
	Value string
}

type TreeBranchSymbols struct {
	Branch     string
	LastBranch string
	Vertical   string
	Space      string
}

var DefaultTreeBranchSymbols TreeBranchSymbols = TreeBranchSymbols{
	Branch:     "├── ",
	LastBranch: "└── ",
	Vertical:   "│   ",
	Space:      "    ",
}

func PrintTree(root *Node, syms TreeBranchSymbols, out io.Writer) {
	// Print the root
	printBranch("", root, 0, syms, out)
}

func printBranch(rootPrefix string, root *Node, indLevel int, syms TreeBranchSymbols, out io.Writer) {
	// Print the root
	fmt.Fprintln(out, rootPrefix+root.Value)

	// Print the sub-branches
	for i := 0; i < len(root.Nodes); i++ {
		// Prefix without vertical line
		prefix := strings.Repeat(syms.Space, indLevel) + syms.Branch

		// Change branch symbol if last node
		if i == len(root.Nodes)-1 {
			prefix = strings.Repeat(syms.Space, indLevel) + syms.LastBranch
		} else if indLevel != 0 {
			// Add vertical line if first indLevel
			prefix = syms.Vertical + strings.Repeat(syms.Space, indLevel-1) + syms.Branch
		}

		printBranch(prefix, root.Nodes[i], indLevel+1, syms, out)
	}
}

// let's start with creational patterns.
func NewTree(root string) *Node {
	if root == "" {
		root = "."
	}

	return &Node{
		Value: root,
		Nodes: make([]*Node, 0),
	}
}

func (r *Node) AddNode(value string) (*Node, error) {
	if value == "" {
		return nil, fmt.Errorf("Cannot add node with an empty value")
	}

	if r == nil {
		return nil, fmt.Errorf("Cannot add nodes to nil")
	}

	// append function. might sort it alphabetically later
	newNode := &Node{Value: value, Nodes: make([]*Node, 0)}
	r.Nodes = append(r.Nodes, newNode)
	return newNode, nil
}

// DFS implementation
func (r *Node) GetNodeByValueDFS(value string) *Node {
	for _, node := range r.Nodes {
		if node.Value == value {
			return node
		}
		rec := node.GetNodeByValueDFS(value)
		if rec != nil {
			return rec
		}
	}
	return nil
}

func (r *Node) GetNodeByValueBFS(value string, maxDepth int) *Node {
	if maxDepth == 0 {
		return nil
	}

	queue := []*Node{r}
	depth := 1

	for len(queue) > 0 {
		// This works, since the "range for loop" takes a snapshot of the queue
		for _, node := range queue {
			queue = queue[1:]

			if node.Value == value {
				return node
			}

			queue = append(queue, node.Nodes...)
		}

		depth++
		if depth > maxDepth {
			return nil
		}
	}

	logger.Debug("Did not find node!", "value", value)
	return nil
}
