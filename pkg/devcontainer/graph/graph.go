package graph

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type Graph[T comparable] struct {
	Nodes map[string]*Node[T]
	Root  *Node[T]

	item string
}

func NewGraph[T comparable](root *Node[T]) *Graph[T] {
	graph := &Graph[T]{
		Nodes: make(map[string]*Node[T]),
		Root:  root,
	}

	graph.Nodes[root.ID] = root
	return graph
}

func NewGraphOf[T comparable](root *Node[T], item string) *Graph[T] {
	graph := &Graph[T]{
		Nodes: make(map[string]*Node[T]),
		Root:  root,
		item:  item,
	}

	graph.Nodes[root.ID] = root
	return graph
}

// Node is a node in a graph
type Node[T comparable] struct {
	ID   string
	Data T

	Parents []*Node[T]
	Childs  []*Node[T]

	Done bool
}

func NewNode[T comparable](id string, data T) *Node[T] {
	return &Node[T]{
		ID:   id,
		Data: data,

		Parents: []*Node[T]{},
		Childs:  []*Node[T]{},
	}
}

// Clone returns a cloned graph
func (g *Graph[T]) Clone() *Graph[T] {
	retGraph := &Graph[T]{
		Nodes: map[string]*Node[T]{},
		item:  g.item,
	}

	// copy nodes
	for k, v := range g.Nodes {
		retGraph.Nodes[k] = NewNode[T](v.ID, v.Data)
	}
	retGraph.Root = retGraph.Nodes[g.Root.ID]

	// copy edges
	for k, v := range g.Nodes {
		for _, child := range v.Childs {
			retGraph.Nodes[k].Childs = append(retGraph.Nodes[k].Childs, retGraph.Nodes[child.ID])
		}
		for _, parent := range v.Parents {
			retGraph.Nodes[k].Parents = append(retGraph.Nodes[k].Parents, retGraph.Nodes[parent.ID])
		}
	}

	return retGraph
}

func (g *Graph[T]) NextFromTop() *Node[T] {
	clonedGraph := g.Clone()
	orderedOptions := []string{}
	nextLeaf := clonedGraph.GetNextLeaf(clonedGraph.Root)
	for nextLeaf != clonedGraph.Root {
		orderedOptions = append(orderedOptions, nextLeaf.ID)
		err := clonedGraph.RemoveNode(nextLeaf.ID)
		if err != nil {
			return nil
		}

		nextLeaf = clonedGraph.GetNextLeaf(clonedGraph.Root)
	}

	for i := len(orderedOptions) - 1; i >= 0; i-- {
		nextNode := g.Nodes[orderedOptions[i]]
		if nextNode == nil || nextNode.Done {
			continue
		}

		nextNode.Done = true
		return nextNode
	}

	return nil
}

// InsertNodeAt inserts a new node at the given parent position
func (g *Graph[T]) InsertNodeAt(parentID string, id string, data T) (*Node[T], error) {
	parentNode, ok := g.Nodes[parentID]
	if !ok {
		return nil, errors.Errorf("Parent %s does not exist", parentID)
	}
	if existingNode, ok := g.Nodes[id]; ok {
		err := g.AddEdge(parentNode.ID, existingNode.ID)
		if err != nil {
			return nil, err
		}

		return existingNode, nil
	}

	node := NewNode[T](id, data)

	g.Nodes[node.ID] = node

	parentNode.Childs = append(parentNode.Childs, node)
	node.Parents = append(node.Parents, parentNode)

	return node, nil
}

func (g *Graph[T]) RemoveSubGraph(id string) error {
	if node, ok := g.Nodes[id]; ok {
		// remove all childs
		for _, child := range node.Childs {
			err := g.RemoveSubGraph(child.ID)
			if err != nil {
				return err
			}
		}

		// Remove child from parents
		return g.RemoveNode(id)
	}

	return nil
}

// RemoveNode removes a node with no children in the graph
func (g *Graph[T]) RemoveNode(id string) error {
	if node, ok := g.Nodes[id]; ok {
		if len(node.Childs) > 0 {
			return errors.Errorf("Cannot remove %s from graph because it has still children", getNameOrID(node))
		}

		// Remove child from parents
		for _, parent := range node.Parents {
			i := -1
			for idx, c := range parent.Childs {
				if c.ID == id {
					i = idx
				}
			}

			if i == -1 {
				return fmt.Errorf("couldn't find %s in parent", getNameOrID(node))
			}
			parent.Childs = append(parent.Childs[:i], parent.Childs[i+1:]...)
		}

		// Remove from graph nodes
		delete(g.Nodes, id)
	}

	return nil
}

// GetNextLeaf returns the next leaf in the graph from node start
func (g *Graph[T]) GetNextLeaf(start *Node[T]) *Node[T] {
	if len(start.Childs) == 0 {
		return start
	}

	return g.GetNextLeaf(start.Childs[0])
}

// CyclicError is the type that is returned if a cyclic edge would be inserted
type CyclicError[T comparable] struct {
	What string
	path []*Node[T]
}

// Error implements error interface
func (c *CyclicError[T]) Error() string {
	cycle := []string{getNameOrID(c.path[len(c.path)-1])}

	for _, node := range c.path {
		cycle = append(cycle, getNameOrID(node))
	}

	what := "dependency"
	if c.What != "" {
		what = c.What
	}

	return fmt.Sprintf("cyclic %s found: \n%s", what, strings.Join(cycle, "\n"))
}

func (g *Graph[T]) AddChild(parentID string, childID string) error {
	return g.AddEdge(parentID, childID)
}

// AddEdge adds a new edge from a node to a node and returns an error if it would result in a cyclic graph
func (g *Graph[T]) AddEdge(fromID string, toID string) error {
	from, ok := g.Nodes[fromID]
	if !ok {
		return errors.Errorf("fromID %s does not exist", fromID)
	}
	to, ok := g.Nodes[toID]
	if !ok {
		return errors.Errorf("toID %s does not exist", toID)
	}

	// Check if there is already an edge
	for _, child := range from.Childs {
		if child.ID == to.ID {
			return nil
		}
	}

	// Check if cyclic
	path := findFirstPath(to, from)
	if path != nil {
		return &CyclicError[T]{
			path: path,
			What: g.item,
		}
	}

	from.Childs = append(from.Childs, to)
	to.Parents = append(to.Parents, from)
	return nil
}

// find first path from node to node with DFS
func findFirstPath[T comparable](from *Node[T], to *Node[T]) []*Node[T] {
	isVisited := map[string]bool{}
	pathList := []*Node[T]{from}

	// Call recursive utility
	if findFirstPathRecursive(from, to, isVisited, &pathList) {
		return pathList
	}

	return nil
}

// A recursive function to print
// all paths from 'u' to 'd'.
// isVisited[] keeps track of
// vertices in current path.
// localPathList<> stores actual
// vertices in the current path
func findFirstPathRecursive[T comparable](u *Node[T], d *Node[T], isVisited map[string]bool, localPathList *[]*Node[T]) bool {
	// Mark the current node
	isVisited[u.ID] = true

	// Is destination?
	if u.ID == d.ID {
		return true
	}

	// Recur for all the vertices
	// adjacent to current vertex
	for _, child := range u.Childs {
		if _, ok := isVisited[child.ID]; !ok {
			// store current node
			// in path[]
			*localPathList = append(*localPathList, child)
			if findFirstPathRecursive(child, d, isVisited, localPathList) {
				return true
			}

			// remove current node
			// in path[]
			i := -1
			for idx, c := range *localPathList {
				if c.ID == child.ID {
					i = idx
				}
			}
			if i != -1 {
				*localPathList = append((*localPathList)[:i], (*localPathList)[i+1:]...)
			}
		}
	}

	// Mark the current node
	delete(isVisited, u.ID)
	return false
}

func getNameOrID[T comparable](n *Node[T]) string {
	return n.ID
}
