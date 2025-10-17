package tree

import (
	"fmt"
	"io"
	"strings"
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
	printBranch([]string{}, root, syms, out)
}

func printBranch(prefix []string, root *Node, syms TreeBranchSymbols, out io.Writer) {
	// Print the root of the branch
	fmt.Fprintln(out, strings.Join(prefix, "")+root.Value)

	// Print the sub-branches
	for i := 0; i < len(root.Nodes); i++ {
		isLastBranchNode := i == len(root.Nodes)-1
		newPrefix := getNewPrefix(prefix, isLastBranchNode, syms)

		printBranch(newPrefix, root.Nodes[i], syms, out)
	}
}

func getNewPrefix(prevPrefix []string, isLastBranchNode bool, syms TreeBranchSymbols) []string {
	newPrefix := []string{}
	prevPrefixLen := len(prevPrefix)

	if prevPrefixLen != 0 {
		newPrefix = prevPrefix[:prevPrefixLen-1]

		prevPrefixLastSym := prevPrefix[prevPrefixLen-1]
		if prevPrefixLastSym == syms.Branch || prevPrefixLastSym == syms.Vertical {
			newPrefix = append(newPrefix, syms.Vertical)
		} else {
			newPrefix = append(newPrefix, syms.Space)
		}
	}

	if isLastBranchNode {
		newPrefix = append(newPrefix, syms.LastBranch)
	} else {
		newPrefix = append(newPrefix, syms.Branch)
	}

	return newPrefix
}

func NewTree(root string) *Node {
	if root == "" {
		root = "."
	}

	return &Node{
		Value: root,
		Nodes: make([]*Node, 0),
	}
}

func (r *Node) AddValue(value string) (*Node, error) {
	if value == "" {
		return nil, fmt.Errorf("cannot add node with an empty value")
	}

	if r == nil {
		return nil, fmt.Errorf("cannot add nodes to nil")
	}

	// append function. might sort it alphabetically later
	newNode := &Node{Value: value, Nodes: make([]*Node, 0)}
	r.Nodes = append(r.Nodes, newNode)
	return newNode, nil
}

func (r *Node) Add(node *Node) error {
	if node == nil {
		return fmt.Errorf("cannot add nil as a node")
	}

	if r == nil {
		return fmt.Errorf("cannot add nodes to nil")
	}

	// append function. might sort it alphabetically later
	r.Nodes = append(r.Nodes, node)
	return nil
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

	return nil
}
