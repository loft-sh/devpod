package graph

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type Graph struct {
	Nodes map[string]*Node

	Root *Node

	item string
}

func NewGraph(root *Node) *Graph {
	graph := &Graph{
		Nodes: make(map[string]*Node),
		Root:  root,
	}

	graph.Nodes[root.ID] = root
	return graph
}

func NewGraphOf(root *Node, item string) *Graph {
	graph := &Graph{
		Nodes: make(map[string]*Node),
		Root:  root,
		item:  item,
	}

	graph.Nodes[root.ID] = root
	return graph
}

// Node is a node in a graph
type Node struct {
	ID   string
	Data interface{}

	Parents []*Node
	Childs  []*Node
}

func NewNode(id string, data interface{}) *Node {
	return &Node{
		ID:   id,
		Data: data,

		Parents: make([]*Node, 0, 1),
		Childs:  make([]*Node, 0, 1),
	}
}

// InsertNodeAt inserts a new node at the given parent position
func (g *Graph) InsertNodeAt(parentID string, id string, data interface{}) (*Node, error) {
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

	node := NewNode(id, data)

	g.Nodes[node.ID] = node

	parentNode.Childs = append(parentNode.Childs, node)
	node.Parents = append(node.Parents, parentNode)

	return node, nil
}

// RemoveNode removes a node with no children in the graph
func (g *Graph) RemoveNode(id string) error {
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

			if i != -1 {
				parent.Childs = append(parent.Childs[:i], parent.Childs[i+1:]...)
			}
		}

		// Remove from graph nodes
		delete(g.Nodes, id)
	}

	return nil
}

// GetNextLeaf returns the next leaf in the graph from node start
func (g *Graph) GetNextLeaf(start *Node) *Node {
	if len(start.Childs) == 0 {
		return start
	}

	return g.GetNextLeaf(start.Childs[0])
}

// CyclicError is the type that is returned if a cyclic edge would be inserted
type CyclicError struct {
	What string
	path []*Node
}

// Error implements error interface
func (c *CyclicError) Error() string {
	cycle := []string{getNameOrID(c.path[len(c.path)-1])}

	for _, node := range c.path {
		cycle = append(cycle, getNameOrID(node))
	}

	what := "dependency"
	if c.What != "" {
		what = c.What
	}

	return fmt.Sprintf("Cyclic %s found: \n%s", what, strings.Join(cycle, "\n"))
}

// AddEdge adds a new edge from a node to a node and returns an error if it would result in a cyclic graph
func (g *Graph) AddEdge(fromID string, toID string) error {
	from, ok := g.Nodes[fromID]
	if !ok {
		return errors.Errorf("fromID %s does not exist", fromID)
	}
	to, ok := g.Nodes[toID]
	if !ok {
		return errors.Errorf("toID %s does not exist", toID)
	}

	// Check if cyclic
	path := findFirstPath(to, from)
	if path != nil {
		return &CyclicError{
			path: path,
			What: g.item,
		}
	}

	// Check if there is already an edge
	for _, child := range from.Childs {
		if child.ID == to.ID {
			return nil
		}
	}

	from.Childs = append(from.Childs, to)
	to.Parents = append(to.Parents, from)
	return nil
}

// find first path from node to node with DFS
func findFirstPath(from *Node, to *Node) []*Node {
	isVisited := map[string]bool{}
	pathList := []*Node{from}

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
func findFirstPathRecursive(u *Node, d *Node, isVisited map[string]bool, localPathList *[]*Node) bool {
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

func getNameOrID(n *Node) string {
	return n.ID
}
