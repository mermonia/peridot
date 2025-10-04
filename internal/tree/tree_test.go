package tree

import (
	"os"
	"testing"

	"github.com/mermonia/peridot/internal/logger"
)

func TestTree(t *testing.T) {
	tree := Node{
		Value: "root",
		Nodes: []*Node{
			{
				Value: "child1",
				Nodes: []*Node{
					{
						Value: "grandchild1",
						Nodes: nil,
					},
					{
						Value: "grandchild2",
						Nodes: nil,
					},
				},
			},
			{
				Value: "child2",
				Nodes: []*Node{
					{
						Value: "grandchild1",
						Nodes: nil,
					},
				},
			},
		},
	}

	PrintTree(&tree, DefaultTreeBranchSymbols, os.Stdout)
}

func TestGetNodeByValueDFS(t *testing.T) {
	nodeToFind := &Node{
		Value: "grandchild1",
		Nodes: nil,
	}

	tree := Node{
		Value: "root",
		Nodes: []*Node{
			{
				Value: "child1",
				Nodes: []*Node{
					nodeToFind,
					{
						Value: "grandchild2",
						Nodes: nil,
					},
				},
			},
			{
				Value: "child2",
				Nodes: []*Node{
					{
						Value: "grandchild1",
						Nodes: nil,
					},
				},
			},
		},
	}

	if tree.GetNodeByValueDFS("grandchild1") != nodeToFind {
		t.Fatalf("DFS did not find the expected node")
	}
}

func TestGetNodeByValueBFS(t *testing.T) {
	nodeToFind := &Node{
		Value: "grandchild1",
		Nodes: nil,
	}

	tree := Node{
		Value: "root",
		Nodes: []*Node{
			{
				Value: "child1",
				Nodes: []*Node{
					{
						Value: "grandchild1",
						Nodes: nil,
					},
					{
						Value: "grandchild2",
						Nodes: nil,
					},
				},
			},
			{
				Value: "child2",
				Nodes: []*Node{
					{
						Value: "grandchild1",
						Nodes: nil,
					},
				},
			},
			nodeToFind,
		},
	}

	foundNode := tree.GetNodeByValueBFS("grandchild1", 2)
	if foundNode != nodeToFind {
		logger.Error("Nodes not identical!", "node1", foundNode, "node2", nodeToFind)
		t.Fatalf("BFS did not find the expected node")
	}
}
