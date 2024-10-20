// Dependency package provides a directed graph of dependencies between services and resolves them
package dependency

import (
	"fmt"
	"log"
	"slices"
)

// DRGraph represents a directed graph of dependencies
type DRGraph struct {
	Root  *DRNode
	Nodes map[string]*DRNode
}

// NewDRGraph creates a new directed graph
func NewDRGraph(root *DRNode) *DRGraph {
	return &DRGraph{
		Nodes: make(map[string]*DRNode),
		Root:  root,
	}
}

// DRNode represents a node in the directed graph
type DRNode struct {
	Name         string
	Dependencies []*DRNode
	Parent       *DRNode
}

// NewDRNode creates a new node in the directed graph
func NewDRNode(name string) *DRNode {
	return &DRNode{
		Name:         name,
		Dependencies: []*DRNode{},
	}
}

// AddDependency adds a dependency to the node
func (n *DRNode) AddDependency(dependency *DRNode) {
	log.Printf("adding dependency %s -> %s", n.Name, dependency.Name)
	n.Dependencies = append(n.Dependencies, dependency)
}

// AddNode adds a node to the graph
func (g *DRGraph) AddNode(name string, node *DRNode) *DRNode {
	g.Nodes[name] = node

	return node
}

// ResolveDependencies resolves the dependencies of the node
func (g *DRGraph) ResolveDependencies(node *DRNode, resolved []*DRNode, unresolved []*DRNode) ([]*DRNode, error) {
	unresolved = append(unresolved, node)

	for _, dep := range node.Dependencies {
		if slices.Contains(resolved, dep) {
			continue
		}

		if slices.Contains(unresolved, dep) {
			return nil, fmt.Errorf("circular dependency detected %s -> %s", node.Name, dep.Name)
		}

		var err error
		resolved, err = g.ResolveDependencies(dep, resolved, unresolved)
		if err != nil {
			return nil, err
		}
	}

	resolved = append(resolved, node)

	for i, n := range unresolved {
		if n == node {
			unresolved = append(unresolved[:i], unresolved[i+1:]...)
			break
		}
	}

	return resolved, nil
}
