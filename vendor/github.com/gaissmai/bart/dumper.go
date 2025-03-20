// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type nodeType byte

const (
	nullNode         nodeType = iota // empty node
	fullNode                         // prefixes and children
	leafNode                         // only prefixes
	intermediateNode                 // only children
)

// ##################################################
//  useful during development, debugging and testing
// ##################################################

// dumpString is just a wrapper for dump.
func (t *Table[V]) dumpString() string {
	w := new(strings.Builder)
	t.dump(w)
	return w.String()
}

// dump the table structure and all the nodes to w.
//
//	 Output:
//
//		[FULL] depth:  0 path: [] / 0
//		indexs(#6): 1 66 128 133 266 383
//		prefxs(#6): 0/0 8/6 0/7 10/7 10/8 127/8
//		childs(#3): 10 127 192
//
//		.[IMED] depth:  1 path: [10] / 8
//		.childs(#1): 0
//
//		..[LEAF] depth:  2 path: [10.0] / 16
//		..indexs(#2): 256 257
//		..prefxs(#2): 0/8 1/8
//
//		.[IMED] depth:  1 path: [127] / 8
//		.childs(#1): 0
//
//		..[IMED] depth:  2 path: [127.0] / 16
//		..childs(#1): 0
//
//		...[LEAF] depth:  3 path: [127.0.0] / 24
//		...indexs(#1): 257
//		...prefxs(#1): 1/8
//
// ...
func (t *Table[V]) dump(w io.Writer) {
	t.init()

	fmt.Fprint(w, "### IPv4:")
	t.rootV4.dumpRec(w, zeroPath, 0, true)

	fmt.Fprint(w, "### IPv6:")
	t.rootV6.dumpRec(w, zeroPath, 0, false)
}

// dumpRec, rec-descent the trie.
func (n *node[V]) dumpRec(w io.Writer, path [16]byte, depth int, is4 bool) {
	n.dump(w, path, depth, is4)

	// make backing arrays, no heap allocs
	addrBackingArray := [maxNodeChildren]uint{}

	// the node may have childs, the rec-descent monster starts
	for i, addr := range n.allChildAddrs(addrBackingArray[:]) {
		octet := byte(addr)
		child := n.children[i]
		path[depth] = octet

		child.dumpRec(w, path, depth+1, is4)
	}
}

// dump the node to w.
func (n *node[V]) dump(w io.Writer, path [16]byte, depth int, is4 bool) {
	bits := depth * strideLen
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	fmt.Fprintf(w, "\n%s[%s] depth:  %d path: [%s] / %d\n",
		indent, n.hasType(), depth, ipStridePath(path, depth, is4), bits)

	if nPfxLen := len(n.prefixes); nPfxLen != 0 {
		// make backing arrays, no heap allocs
		idxBackingArray := [maxNodePrefixes]uint{}
		allIndices := n.allStrideIndexes(idxBackingArray[:])

		// print the baseIndices for this node.
		fmt.Fprintf(w, "%sindexs(#%d): %v\n", indent, nPfxLen, allIndices)

		// print the prefixes for this node
		fmt.Fprintf(w, "%sprefxs(#%d):", indent, nPfxLen)

		for _, idx := range allIndices {
			octet, bits := baseIndexToPrefix(idx)
			fmt.Fprintf(w, " %s/%d", octetFmt(octet, is4), bits)
		}
		fmt.Fprintln(w)

		// print the values for this node
		fmt.Fprintf(w, "%svalues(#%d):", indent, nPfxLen)

		for _, val := range n.prefixes {
			fmt.Fprintf(w, " %v", val)
		}
		fmt.Fprintln(w)
	}

	if childs := len(n.children); childs != 0 {
		// print the childs for this node
		fmt.Fprintf(w, "%schilds(#%d):", indent, childs)

		addrBackingArray := [maxNodeChildren]uint{}
		for _, addr := range n.allChildAddrs(addrBackingArray[:]) {
			octet := byte(addr)
			fmt.Fprintf(w, " %s", octetFmt(octet, is4))
		}
		fmt.Fprintln(w)
	}
}

// octetFmt, different format strings for IPv4 and IPv6, decimal versus hex.
func octetFmt(octet byte, is4 bool) string {
	if is4 {
		return fmt.Sprintf("%d", octet)
	}
	return fmt.Sprintf("0x%02x", octet)
}

// ip stride path, different formats for IPv4 and IPv6, dotted decimal or hex.
//
//	127.0.0
//	2001:0d
func ipStridePath(path [16]byte, depth int, is4 bool) string {
	buf := new(strings.Builder)

	if is4 {
		for i, b := range path[:depth] {
			if i != 0 {
				buf.WriteString(".")
			}
			buf.WriteString(strconv.Itoa(int(b)))
		}
		return buf.String()
	}

	for i, b := range path[:depth] {
		if i != 0 && i%2 == 0 {
			buf.WriteString(":")
		}
		buf.WriteString(fmt.Sprintf("%02x", b))
	}
	return buf.String()
}

// String implements Stringer for nodeType.
func (nt nodeType) String() string {
	switch nt {
	case nullNode:
		return "NULL"
	case fullNode:
		return "FULL"
	case leafNode:
		return "LEAF"
	case intermediateNode:
		return "IMED"
	}
	panic("unreachable")
}

// hasType returns the nodeType.
func (n *node[V]) hasType() nodeType {
	lenPefixes := len(n.prefixes)
	lenChilds := len(n.children)

	if lenPefixes == 0 && lenChilds != 0 {
		return intermediateNode
	}

	if lenPefixes == 0 && lenChilds == 0 {
		return nullNode
	}

	if lenPefixes != 0 && lenChilds == 0 {
		return leafNode
	}

	if lenPefixes != 0 && lenChilds != 0 {
		return fullNode
	}
	panic("unreachable")
}
