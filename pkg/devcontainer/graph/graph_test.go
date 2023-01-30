package graph

import (
	"testing"
)

func TestGraph(t *testing.T) {
	var (
		root                   = NewNode("root", nil)
		rootChild1             = NewNode("rootChild1", nil)
		rootChild2             = NewNode("rootChild2", nil)
		rootChild3             = NewNode("rootChild3", nil)
		rootChild2Child1       = NewNode("rootChild2Child1", nil)
		rootChild2Child1Child1 = NewNode("rootChild2Child1Child1", nil)

		testGraph = NewGraph(root)
	)

	_, err := testGraph.InsertNodeAt("does not exits", rootChild1.ID, nil)
	if err == nil {
		t.Fatal("InsertNodeAt error expected")
	}

	_, _ = testGraph.InsertNodeAt(root.ID, rootChild1.ID, nil)
	_, _ = testGraph.InsertNodeAt(root.ID, rootChild2.ID, nil)
	_, _ = testGraph.InsertNodeAt(root.ID, rootChild3.ID, nil)

	_, _ = testGraph.InsertNodeAt(rootChild2.ID, rootChild2Child1.ID, nil)
	_, _ = testGraph.InsertNodeAt(rootChild2Child1.ID, rootChild2Child1Child1.ID, nil)
	_, _ = testGraph.InsertNodeAt(rootChild3.ID, rootChild2.ID, nil)

	// Cyclic graph error
	_, err = testGraph.InsertNodeAt(rootChild2Child1Child1.ID, rootChild3.ID, nil)
	if err == nil {
		t.Fatal("Cyclic error expected")
	} else {
		errMsg := `Cyclic dependency found: 
rootChild2Child1Child1
rootChild3
rootChild2
rootChild2Child1
rootChild2Child1Child1`

		if err.Error() != errMsg {
			t.Fatalf("Expected %s, got %s", errMsg, err.Error())
		}
	}

	// Find first path
	path := findFirstPath(rootChild1, rootChild2)
	if path != nil {
		t.Fatalf("Wrong path found: %#+v", path)
	}

	// Find first path
	path = findFirstPath(root, rootChild2Child1Child1)
	if len(path) != 4 || path[0].ID != root.ID || path[1].ID != rootChild2.ID || path[2].ID != rootChild2Child1.ID || path[3].ID != rootChild2Child1Child1.ID {
		t.Fatalf("Wrong path found: %#+v", path)
	}

	// Get leaf node
	leaf := testGraph.GetNextLeaf(root)
	if leaf.ID != rootChild1.ID {
		t.Fatalf("GetLeaf1: Got id %s, expected %s", leaf.ID, rootChild1.ID)
	}

	err = testGraph.AddEdge("NotThere", leaf.ID)
	if err == nil {
		t.Fatal("No error when adding an edge from a non-existing node")
	}

	err = testGraph.AddEdge(leaf.ID, "NotThere")
	if err == nil {
		t.Fatal("No error when adding an edge to a non-existing node")
	}

	// Remove node
	err = testGraph.RemoveNode(leaf.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Get leaf node
	leaf = testGraph.GetNextLeaf(root)
	if leaf.ID != rootChild2Child1Child1.ID {
		t.Fatalf("GetLeaf2: Got id %s, expected %s", leaf.ID, rootChild2Child1Child1.ID)
	}

	// Remove node
	err = testGraph.RemoveNode(root.ID)
	if err == nil {
		t.Fatal("Expected error")
	}
}
