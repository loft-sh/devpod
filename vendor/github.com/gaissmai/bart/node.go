// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"cmp"
	"net/netip"
	"slices"

	"github.com/bits-and-blooms/bitset"
)

const (
	strideLen       = 8                    // octet
	maxTreeDepth    = 128 / strideLen      // 16
	maxNodeChildren = 1 << strideLen       // 256
	maxNodePrefixes = 1 << (strideLen + 1) // 512
)

// zero value, used manifold
var zeroPath [16]byte

// node is a level node in the multibit-trie.
// A node has prefixes and children.
//
// The prefixes form a complete binary tree, see the artlookup.pdf
// paper in the doc folder to understand the data structure.
//
// In contrast to the ART algorithm, popcount-compressed slices are used
// instead of fixed-size arrays.
//
// The array slots are also not pre-allocated as in the ART algorithm,
// but backtracking is used for the longest-prefix-match.
//
// The lookup is then slower by a factor of about 2, but this is
// the intended trade-off to prevent memory consumption from exploding.
type node[V any] struct {
	prefixesBitset *bitset.BitSet
	childrenBitset *bitset.BitSet

	// popcount compressed slices
	prefixes []V
	children []*node[V]
}

// newNode, BitSets have to be initialized.
func newNode[V any]() *node[V] {
	return &node[V]{
		prefixesBitset: bitset.New(0), // init BitSet
		childrenBitset: bitset.New(0), // init BitSet
	}
}

// isEmpty returns true if node has neither prefixes nor children.
func (n *node[V]) isEmpty() bool {
	return len(n.prefixes) == 0 && len(n.children) == 0
}

// ################## prefixes ################################

// prefixRank, Rank() is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (n *node[V]) prefixRank(baseIdx uint) int {
	// adjust offset by one to slice index
	return int(n.prefixesBitset.Rank(baseIdx)) - 1
}

// insertPrefix adds the route for baseIdx, with value val.
// If the value already exists, overwrite it with val and return false.
func (n *node[V]) insertPrefix(baseIdx uint, val V) (ok bool) {
	// prefix exists, overwrite val
	if n.prefixesBitset.Test(baseIdx) {
		n.prefixes[n.prefixRank(baseIdx)] = val
		return false
	}

	// new, insert into bitset and slice
	n.prefixesBitset.Set(baseIdx)
	n.prefixes = slices.Insert(n.prefixes, n.prefixRank(baseIdx), val)
	return true
}

// deletePrefix removes the route octet/prefixLen.
// Returns false if there was no prefix to delete.
func (n *node[V]) deletePrefix(octet byte, prefixLen int) (ok bool) {
	baseIdx := prefixToBaseIndex(octet, prefixLen)

	// no route entry
	if !n.prefixesBitset.Test(baseIdx) {
		return false
	}

	rnk := n.prefixRank(baseIdx)

	// delete from slice
	n.prefixes = slices.Delete(n.prefixes, rnk, rnk+1)

	// delete from bitset, followed by Compact to reduce memory consumption
	n.prefixesBitset.Clear(baseIdx)
	n.prefixesBitset.Compact()

	return true
}

// updatePrefix, update or set the value at prefix via callback. The new value returned
// and a bool wether the prefix was already present in the node.
func (n *node[V]) updatePrefix(octet byte, prefixLen int, cb func(V, bool) V) (newVal V, wasPresent bool) {
	// calculate idx once
	baseIdx := prefixToBaseIndex(octet, prefixLen)

	var rnk int

	// if prefix is set, get current value
	var oldVal V
	if wasPresent = n.prefixesBitset.Test(baseIdx); wasPresent {
		rnk = n.prefixRank(baseIdx)
		oldVal = n.prefixes[rnk]
	}

	// callback function to get updated or new value
	newVal = cb(oldVal, wasPresent)

	// prefix is already set, update and return value
	if wasPresent {
		n.prefixes[rnk] = newVal
		return
	}

	// new prefix, insert into bitset ...
	n.prefixesBitset.Set(baseIdx)

	// bitset has changed, recalc rank
	rnk = n.prefixRank(baseIdx)

	// ... and insert value into slice
	n.prefixes = slices.Insert(n.prefixes, rnk, newVal)

	return
}

// lpm does a route lookup for idx in the 8-bit (stride) routing table
// at this depth and returns (baseIdx, value, true) if a matching
// longest prefix exists, or ok=false otherwise.
//
// backtracking is fast, it's just a bitset test and, if found, one popcount.
// max steps in backtracking is the stride length.
func (n *node[V]) lpm(idx uint) (baseIdx uint, val V, ok bool) {
	for baseIdx = idx; baseIdx > 0; baseIdx >>= 1 {
		if n.prefixesBitset.Test(baseIdx) {
			// longest prefix match
			return baseIdx, n.prefixes[n.prefixRank(baseIdx)], true
		}
	}

	// not found (on this level)
	return 0, val, false
}

// lpmTest for internal use in overlap tests, just return true or false, no value needed.
func (n *node[V]) lpmTest(baseIdx uint) bool {
	for idx := baseIdx; idx > 0; idx >>= 1 {
		if n.prefixesBitset.Test(idx) {
			return true
		}
	}

	return false
}

// getValue for baseIdx.
func (n *node[V]) getValue(baseIdx uint) (val V, ok bool) {
	if n.prefixesBitset.Test(baseIdx) {
		return n.prefixes[n.prefixRank(baseIdx)], true
	}
	return
}

// allStrideIndexes returns all baseIndexes set in this stride node in ascending order.
func (n *node[V]) allStrideIndexes(buffer []uint) []uint {
	if len(n.prefixes) > len(buffer) {
		panic("logic error, buffer is too small")
	}

	_, buffer = n.prefixesBitset.NextSetMany(0, buffer)
	return buffer
}

// ################## children ################################

// childRank, Rank() is the key of the popcount compression algorithm,
// mapping between bitset index and slice index.
func (n *node[V]) childRank(octet byte) int {
	// adjust offset by one to slice index
	return int(n.childrenBitset.Rank(uint(octet))) - 1
}

// insertChild, insert the child
func (n *node[V]) insertChild(octet byte, child *node[V]) {
	// child exists, overwrite it
	if n.childrenBitset.Test(uint(octet)) {
		n.children[n.childRank(octet)] = child
		return
	}

	// new insert into bitset and slice
	n.childrenBitset.Set(uint(octet))
	n.children = slices.Insert(n.children, n.childRank(octet), child)
}

// deleteChild, delete the child at octet. It is valid to delete a non-existent child.
func (n *node[V]) deleteChild(octet byte) {
	if !n.childrenBitset.Test(uint(octet)) {
		return
	}

	rnk := n.childRank(octet)

	// delete from slice
	n.children = slices.Delete(n.children, rnk, rnk+1)

	// delete from bitset, followed by Compact to reduce memory consumption
	n.childrenBitset.Clear(uint(octet))
	n.childrenBitset.Compact()
}

// getChild returns the child pointer for octet, or nil if none.
func (n *node[V]) getChild(octet byte) *node[V] {
	if !n.childrenBitset.Test(uint(octet)) {
		return nil
	}

	return n.children[n.childRank(octet)]
}

// allChildAddrs fills the buffer with the octets of all child nodes in ascending order,
// panics if the buffer isn't big enough.
func (n *node[V]) allChildAddrs(buffer []uint) []uint {
	if len(n.children) > len(buffer) {
		panic("logic error, buffer is too small")
	}

	_, buffer = n.childrenBitset.NextSetMany(0, buffer)
	return buffer
}

// #################### nodes #############################################

// eachLookupPrefix does an all prefix match in the 8-bit (stride) routing table
// at this depth and calls yield() for any matching CIDR.
func (n *node[V]) eachLookupPrefix(path [16]byte, depth int, is4 bool, octet byte, bits int, yield func(pfx netip.Prefix, val V) bool) bool {
	for idx := prefixToBaseIndex(octet, bits); idx > 0; idx >>= 1 {
		if n.prefixesBitset.Test(idx) {
			cidr, _ := cidrFromPath(path, depth, is4, idx)
			val, _ := n.getValue(idx)

			if !yield(cidr, val) {
				// early exit
				return false
			}
		}
	}

	return true
}

// overlapsRec returns true if any IP in the nodes n or o overlaps.
func (n *node[V]) overlapsRec(o *node[V]) bool {
	// ##############################
	// 1. test if any routes overlaps
	// ##############################

	nPfxLen := len(n.prefixes)
	oPfxLen := len(o.prefixes)

	if oPfxLen > 0 && nPfxLen > 0 {

		// some prefixes are identical
		if n.prefixesBitset.IntersectionCardinality(o.prefixesBitset) > 0 {
			return true
		}

		var nIdx, oIdx uint

		nOK := nPfxLen > 0
		oOK := oPfxLen > 0

		// zip, range over n and o at the same time to help chance on its way
		for nOK || oOK {

			if nOK {
				// does any route in o overlap this prefix from n
				if nIdx, nOK = n.prefixesBitset.NextSet(nIdx); nOK {
					if o.lpmTest(nIdx) {
						return true
					}
				}
				nIdx++
			}

			if oOK {
				// does any route in n overlap this prefix from o
				if oIdx, oOK = o.prefixesBitset.NextSet(oIdx); oOK {
					if n.lpmTest(oIdx) {
						return true
					}
				}
				oIdx++
			}
		}
	}

	// ####################################
	// 2. test if routes overlaps any child
	// ####################################

	nChildLen := len(n.children)
	oChildLen := len(o.children)

	var nAddr, oAddr uint

	nOK := nChildLen > 0 && oPfxLen > 0 // test the childs in n against the routes in o
	oOK := oChildLen > 0 && nPfxLen > 0 // test the childs in o against the routes in n

	// zip, range over n and o at the same time to help chance on its way
	for nOK || oOK {

		if nOK {
			// does any route in o overlap this child from n
			if nAddr, nOK = n.childrenBitset.NextSet(nAddr); nOK {
				if o.lpmTest(octetToBaseIndex(byte(nAddr))) {
					return true
				}
			}
			nAddr++
		}

		if oOK {
			// does any route in n overlap this child from o
			if oAddr, oOK = o.childrenBitset.NextSet(oAddr); oOK {
				if n.lpmTest(octetToBaseIndex(byte(oAddr))) {
					return true
				}
			}
			oAddr++
		}
	}

	// ################################################################
	// 3. rec-descent call for childs with same octet in nodes n and o
	// ################################################################

	// stop condition, n or o have no childs
	if nChildLen == 0 || oChildLen == 0 {
		return false
	}

	// stop condition, no child with identical octet in n and o
	if n.childrenBitset.IntersectionCardinality(o.childrenBitset) == 0 {
		return false
	}

	// swap the nodes, range over shorter bitset
	if nChildLen > oChildLen {
		n, o = o, n
	}

	addrBackingArray := [maxNodeChildren]uint{}
	for i, addr := range n.allChildAddrs(addrBackingArray[:]) {
		oChild := o.getChild(byte(addr))
		if oChild == nil {
			// no child in o with this octet
			continue
		}

		// we have the slice index for n
		nChild := n.children[i]

		// rec-descent
		if nChild.overlapsRec(oChild) {
			return true
		}
	}

	return false
}

// overlapsPrefix returns true if node overlaps with prefix.
func (n *node[V]) overlapsPrefix(octet byte, pfxLen int) bool {
	// ##################################################
	// 1. test if any route in this node overlaps prefix?
	// ##################################################

	pfxIdx := prefixToBaseIndex(octet, pfxLen)
	if n.lpmTest(pfxIdx) {
		return true
	}

	// #################################################
	// 2. test if prefix overlaps any route in this node
	// #################################################

	// lower/upper boundary for host routes
	pfxLower, pfxUpper := hostRoutesByIndex(pfxIdx)

	// increment to 'next' routeIdx for start in bitset search
	// since pfxIdx already testet by lpm in other direction
	routeIdx := pfxIdx * 2
	var ok bool
	for {
		if routeIdx, ok = n.prefixesBitset.NextSet(routeIdx); !ok {
			break
		}

		routeLower, routeUpper := hostRoutesByIndex(routeIdx)
		if routeLower >= pfxLower && routeUpper <= pfxUpper {
			return true
		}

		// next route
		routeIdx++
	}

	// #################################################
	// 3. test if prefix overlaps any child in this node
	// #################################################

	// set start octet in bitset search with prefix octet
	addr := uint(octet)
	for {
		if addr, ok = n.childrenBitset.NextSet(addr); !ok {
			break
		}

		idx := addr + firstHostIndex
		if idx >= pfxLower && idx <= pfxUpper {
			return true
		}

		// next round
		addr++
	}

	return false
}

// eachSubnet calls yield() for any covered CIDR by parent prefix.
func (n *node[V]) eachSubnet(path [16]byte, depth int, is4 bool, parentOctet byte, pfxLen int, yield func(pfx netip.Prefix, val V) bool) bool {
	// collect all routes covered by this pfx
	// see also algorithm in overlapsPrefix

	// can't use lpm, search prefix has no node
	parentIdx := prefixToBaseIndex(parentOctet, pfxLen)
	parentLower, parentUpper := hostRoutesByIndex(parentIdx)

	// start bitset search at parentIdx
	idx := parentIdx
	var ok bool
	for {
		if idx, ok = n.prefixesBitset.NextSet(idx); !ok {
			break
		}

		// can't use lpm, search prefix has no node
		lower, upper := hostRoutesByIndex(idx)

		// idx is covered by parentIdx?
		if lower >= parentLower && upper <= parentUpper {
			val, _ := n.getValue(idx)
			cidr, _ := cidrFromPath(path, depth, is4, idx)

			if !yield(cidr, val) {
				// early exit
				return false
			}

		}

		idx++
	}

	// collect all children covered
	var addr uint
	for {
		if addr, ok = n.childrenBitset.NextSet(addr); !ok {
			break
		}
		octet := byte(addr)

		// make host route for comparison with lower, upper
		idx := octetToBaseIndex(octet)

		// is child covered?
		if idx >= parentLower && idx <= parentUpper {
			c := n.getChild(octet)

			// add (set) this octet to path
			path[depth] = octet

			// all cidrs under this child are covered by pfx
			if !c.allRec(path, depth+1, is4, yield) {
				// early exit
				return false
			}
		}

		addr++
	}

	return true
}

// unionRec combines two nodes, changing the receiver node.
// If there are duplicate entries, the value is taken from the other node.
// Count duplicate entries to adjust the t.size struct members.
func (n *node[V]) unionRec(o *node[V]) (duplicates int) {
	// make backing arrays, no heap allocs
	idxBackingArray := [maxNodePrefixes]uint{}

	// for all prefixes in other node do ...
	for _, oIdx := range o.allStrideIndexes(idxBackingArray[:]) {
		// insert/overwrite prefix/value from oNode to nNode
		oVal, _ := o.getValue(oIdx)
		if !n.insertPrefix(oIdx, oVal) {
			duplicates++
		}
	}

	// make backing arrays, no heap allocs
	addrBackingArray := [maxNodeChildren]uint{}

	// for all children in other node do ...
	for i, oOctet := range o.allChildAddrs(addrBackingArray[:]) {
		octet := byte(oOctet)

		// we know the slice index, faster as o.getChild(octet)
		oc := o.children[i]

		// get n child with same octet,
		// we don't know the slice index in n.children
		nc := n.getChild(octet)

		if nc == nil {
			// insert cloned child from oNode into nNode
			n.insertChild(octet, oc.cloneRec())
		} else {
			// both nodes have child with octet, call union rec-descent
			duplicates += nc.unionRec(oc)
		}
	}
	return duplicates
}

// cloneRec, clones the node recursive.
func (n *node[V]) cloneRec() *node[V] {
	c := newNode[V]()
	if n.isEmpty() {
		return c
	}

	c.prefixesBitset = n.prefixesBitset.Clone() // deep
	c.prefixes = slices.Clone(n.prefixes)       // shallow values

	c.childrenBitset = n.childrenBitset.Clone() // deep
	c.children = slices.Clone(n.children)       // shallow

	// now clone the children deep
	for i, child := range c.children {
		c.children[i] = child.cloneRec()
	}

	return c
}

// allRec runs recursive the trie, starting at this node and
// the yield function is called for each route entry with prefix and value.
// If the yield function returns false the recursion ends prematurely and the
// false value is propagated.
//
// The iteration order is not defined, just the simplest and fastest recursive implementation.
func (n *node[V]) allRec(path [16]byte, depth int, is4 bool, yield func(netip.Prefix, V) bool) bool {
	// for all prefixes in this node do ...
	idxBackingArray := [maxNodePrefixes]uint{}
	for _, idx := range n.allStrideIndexes(idxBackingArray[:]) {
		val, _ := n.getValue(idx)
		cidr, _ := cidrFromPath(path, depth, is4, idx)

		// make the callback for this prefix
		if !yield(cidr, val) {
			// early exit
			return false
		}
	}

	// for all children in this node do ...
	addrBackingArray := [maxNodeChildren]uint{}
	for i, addr := range n.allChildAddrs(addrBackingArray[:]) {
		child := n.children[i]
		path[depth] = byte(addr)

		if !child.allRec(path, depth+1, is4, yield) {
			// early exit
			return false
		}
	}

	return true
}

// allRecSorted runs recursive the trie, starting at node and
// the yield function is called for each route entry with prefix and value.
// If the yield function returns false the recursion ends prematurely and the
// false value is propagated.
//
// The iteration is in prefix sort order, it's a very complex implemenation compared with allRec.
func (n *node[V]) allRecSorted(path [16]byte, depth int, is4 bool, yield func(netip.Prefix, V) bool) bool {
	// make backing arrays, no heap allocs
	addrBackingArray := [maxNodeChildren]uint{}
	idxBackingArray := [maxNodePrefixes]uint{}

	// get slice of all child octets, sorted by addr
	childAddrs := n.allChildAddrs(addrBackingArray[:])
	childCursor := 0

	// get slice of all indexes, sorted by idx
	allIndices := n.allStrideIndexes(idxBackingArray[:])

	// re-sort indexes by prefix in place
	slices.SortFunc(allIndices, cmpIndexRank)

	// example for entry with root node:
	//
	//  ▼
	//  ├─ 0.0.0.1/32        <-- FOOTNOTE A: child  0     in first node
	//  ├─ 10.0.0.0/7        <-- FOOTNOTE B: prefix 10/7  in first node
	//  │  └─ 10.0.0.0/8     <-- FOOTNOTE C: prefix 10/8  in first node
	//  │     └─ 10.0.0.1/32 <-- FOOTNOTE D: child  10    in first node
	//  ├─ 127.0.0.0/8       <-- FOOTNOTE E: prefix 127/8 in first node
	//  └─ 192.168.0.0/16    <-- FOOTNOTE F: child  192   in first node

	// range over all indexes in this node, now in prefix sort order
	// FOOTNOTE: B, C, E
	for i, idx := range allIndices {
		// get the host routes for this index
		lower, upper := hostRoutesByIndex(idx)

		// adjust host routes for this idx in case the host routes
		// of the following idx overlaps
		// FOOTNOTE: B and C have overlaps in host routes
		// FOOTNOTE: C, E don't overlap in host routes
		// FOOTNOTE: E has no following prefix in this node
		if i+1 < len(allIndices) {
			lower, upper = adjustHostRoutes(idx, allIndices[i+1])
		}

		// handle childs before the host routes of idx
		// FOOTNOTE: A
		for j := childCursor; j < len(childAddrs); j++ {
			addr := childAddrs[j]
			octet := byte(addr)

			if octetToBaseIndex(octet) >= lower {
				// lower border of host routes
				break
			}

			// we know the slice index, faster as n.getChild(octet)
			c := n.children[j]

			// add (set) this octet to path
			path[depth] = octet

			if !c.allRecSorted(path, depth+1, is4, yield) {
				// early exit
				return false
			}

			childCursor++
		}

		// FOOTNOTE: B, C, F
		// now handle prefix for idx
		val, _ := n.getValue(idx)
		cidr, _ := cidrFromPath(path, depth, is4, idx)

		if !yield(cidr, val) {
			// early exit
			return false
		}

		// handle the children in host routes for this prefix
		// FOOTNOTE: D
		for j := childCursor; j < len(childAddrs); j++ {
			addr := childAddrs[j]
			octet := byte(addr)
			if octetToBaseIndex(octet) > upper {
				// out of host routes
				break
			}

			// we know the slice index, faster as n.getChild(octet)
			c := n.children[j]

			// add (set) this octet to path
			path[depth] = octet

			if !c.allRecSorted(path, depth+1, is4, yield) {
				// early exit
				return false
			}

			childCursor++
		}
	}

	// FOOTNOTE: F
	// handle all the rest of the children
	for j := childCursor; j < len(childAddrs); j++ {
		addr := childAddrs[j]
		octet := byte(addr)

		// we know the slice index, faster as n.getChild(octet)
		c := n.children[j]

		// add (set) this octet to path
		path[depth] = octet

		if !c.allRecSorted(path, depth+1, is4, yield) {
			// early exit
			return false
		}
	}

	return true
}

// adjustHostRoutes, helper function to adjust the lower, upper bounds of the
// host routes in case the host routes of the next idx overlaps
func adjustHostRoutes(idx, next uint) (lower, upper uint) {
	lower, upper = hostRoutesByIndex(idx)

	// get the lower host route border of the next idx
	nextLower, _ := hostRoutesByIndex(next)

	// is there an overlap?
	switch {
	case nextLower == lower:
		upper = 0

		// [------------] idx
		// [-----]        next
		// make host routes for this idx invalid
		//
		// ][             idx
		// [-----]^^^^^^] next
		//
		//  these ^^^^^^ children are handled before next prefix
		//
		// sorry, I know, it's completely confusing

	case nextLower <= upper:
		upper = nextLower - 1

		// [------------] idx
		//       [------] next
		//
		// shrink host routes for this idx
		// [----][------] idx, next
		//      ^
	}

	return lower, upper
}

// numPrefixesRec, calculate the number of prefixes under n.
func (n *node[V]) numPrefixesRec() int {
	size := len(n.prefixes) // this node
	for _, c := range n.children {
		size += c.numPrefixesRec()
	}
	return size
}

// numNodesRec, calculate the number of nodes under n.
func (n *node[V]) numNodesRec() int {
	size := 1 // this node
	for _, c := range n.children {
		size += c.numNodesRec()
	}
	return size
}

// cmpPrefix, compare func for prefix sort,
// all cidrs are already normalized
func cmpPrefix(a, b netip.Prefix) int {
	if cmp := a.Addr().Compare(b.Addr()); cmp != 0 {
		return cmp
	}
	return cmp.Compare(a.Bits(), b.Bits())
}
