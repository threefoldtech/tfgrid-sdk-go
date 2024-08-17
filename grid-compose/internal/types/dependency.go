package types

import (
	"fmt"
	"log"
	"slices"
)

type DRGraph struct {
	Root  *DRNode
	Nodes map[string]*DRNode
}

func NewDRGraph(root *DRNode) *DRGraph {
	return &DRGraph{
		Nodes: make(map[string]*DRNode),
		Root:  root,
	}
}

type DRNode struct {
	Name         string
	Dependencies []*DRNode
	Parent       *DRNode
	Service      *Service
}

func NewDRNode(name string, service *Service) *DRNode {
	return &DRNode{
		Name:         name,
		Dependencies: []*DRNode{},
		Service:      service,
	}
}

func (n *DRNode) AddDependency(dependency *DRNode) {
	log.Printf("adding dependency %s -> %s", n.Name, dependency.Name)
	n.Dependencies = append(n.Dependencies, dependency)
}

func (g *DRGraph) AddNode(name string, node *DRNode) *DRNode {
	g.Nodes[name] = node

	return node
}

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
