// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"sync"
)

// Table is an IPv4 and IPv6 routing table with payload V.
// The zero value is ready to use.
type Table[V any] struct {
	rootV4 *node[V]
	rootV6 *node[V]

	// simple API, no constructor needed
	initOnce sync.Once
}

// init once, so no constructor is needed.
func (t *Table[V]) init() {
	t.initOnce.Do(func() {
		// BitSets have to be initialized.
		t.rootV4 = newNode[V]()
		t.rootV6 = newNode[V]()
	})
}

// rootNodeByVersion, select root node for ip version.
func (t *Table[V]) rootNodeByVersion(is4 bool) *node[V] {
	if is4 {
		return t.rootV4
	}
	return t.rootV6
}

// Insert adds pfx to the tree, with value val.
// If pfx is already present in the tree, its value is set to val.
func (t *Table[V]) Insert(pfx netip.Prefix, val V) {
	t.init()

	// always normalize the prefix
	pfx = pfx.Masked()

	// some needed values, see below
	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)

	// insert default route, easy peasy
	if bits == 0 {
		n.prefixes.insert(0, 0, val)
		return
	}

	// the ip is chunked in bytes, the multibit stride is 8
	bs := ip.AsSlice()

	// depth index for the child trie
	depth := 0
	for {
		addr := uint(bs[depth]) // stride = 8!

		// loop stop condition:
		// last non-masked addr chunk of prefix, insert the
		// byte and bits into the prefixHeap on this depth
		//
		// 8.0.0.0/5 ->       depth 0, addr byte  8,  bits 5
		// 10.0.0.0/8 ->      depth 0, addr byte  10, bits 8
		// 192.168.0.0/16  -> depth 1, addr byte 168, bits 8, (16-1*8 = 8)
		// 192.168.20.0/19 -> depth 2, addr byte  20, bits 3, (19-2*8 = 3)
		// 172.16.19.12/32 -> depth 3, addr byte  12, bits 8, (32-3*8 = 8)
		//
		if bits <= stride {
			n.prefixes.insert(addr, bits, val)
			return
		}

		// descend down to next child level
		child := n.children.get(addr)

		// create and insert missing intermediate child, no path compression!
		if child == nil {
			child = newNode[V]()
			n.children.insert(addr, child)
		}

		// go down
		depth++
		n = child
		bits -= stride
	}
}

// Delete removes pfx from the tree, pfx does not have to be present.
func (t *Table[V]) Delete(pfx netip.Prefix) {
	t.init()

	// always normalize the prefix
	pfx = pfx.Masked()

	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()

	n := t.rootNodeByVersion(is4)

	// delete default route, easy peasy
	if bits == 0 {
		n.prefixes.delete(0, 0)
		return
	}

	// stack of the traversed child path in order to
	// purge dangling paths after deletion
	pathStack := [maxTreeDepth]*node[V]{}

	bs := ip.AsSlice()
	depth := 0
	for {
		addr := uint(bs[depth]) // stride = 8!

		// push current node on stack for path recording
		pathStack[depth] = n

		// last non-masked byte
		if bits <= stride {
			// found a child on proper depth ...
			if !n.prefixes.delete(addr, bits) {
				// ... but prefix not in tree, nothing deleted
				return
			}

			// purge dangling path, if needed
			break
		}

		// descend down to next level, no path compression
		child := n.children.get(addr)
		if child == nil {
			// no child, nothing to delete
			return
		}

		// go down
		depth++
		bits -= stride
		n = child
	}

	// check for dangling path
	for {

		// loop stop condition
		if depth == 0 {
			break
		}

		// an empty node?
		if n.isEmpty() {
			// purge this node from parents children
			parent := pathStack[depth-1]
			parent.children.delete(uint(bs[depth-1]))
		}

		// go up
		depth--
		n = pathStack[depth]
	}
}

// Get does a route lookup for IP and returns the associated value and true, or false if
// no route matched.
func (t *Table[V]) Get(ip netip.Addr) (val V, ok bool) {
	t.init()
	_, _, val, ok = t.lpmByIP(ip)
	return
}

// Lookup does a route lookup for IP and returns the longest prefix,
// the associated value and true for success, or false otherwise if
// no route matched.
//
// Lookup is a bit slower than Get, so if you only need the payload V
// and not the matching longest-prefix back, you should use just Get.
func (t *Table[V]) Lookup(ip netip.Addr) (lpm netip.Prefix, val V, ok bool) {
	t.init()
	if depth, baseIdx, val, ok := t.lpmByIP(ip); ok {

		// add the bits from higher levels in child trie to pfxLen
		bits := depth*stride + baseIndexToPrefixLen(baseIdx)

		// mask prefix from lookup ip, masked with longest prefix bits.
		lpm = netip.PrefixFrom(ip, bits).Masked()

		return lpm, val, ok
	}
	return
}

// lpmByIP does a route lookup for IP with longest prefix match.
// Returns also depth and baseIdx for Lookup to retrieve the
// lpm prefix out of the prefix tree.
func (t *Table[V]) lpmByIP(ip netip.Addr) (depth int, baseIdx uint, val V, ok bool) {
	t.init()

	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)

	// stack of the traversed nodes for fast backtracking, if needed
	pathStack := [maxTreeDepth]*node[V]{}

	// keep the lpm alloc free, don't use ip.AsSlice here
	a16 := ip.As16()
	bs := a16[:]
	if is4 {
		bs = bs[12:]
	}

	depth = 0
	addr := uint(bs[depth]) // bytewise, stride = 8
	// find leaf tree
	for {

		// push current node on stack for fast backtracking
		pathStack[depth] = n

		// go down in tight loop to leaf tree
		if child := n.children.get(addr); child != nil {
			depth++
			addr = uint(bs[depth])
			n = child
			continue
		}

		break
	}

	// start backtracking at leaf node in tight loop
	for {
		// lookup only in nodes with prefixes, skip over intermediate nodes
		if len(n.prefixes.values) != 0 {
			// longest prefix match?
			if baseIdx, val, ok := n.prefixes.lpmByIndex(addrToBaseIndex(addr)); ok {
				// return also baseIdx and the depth, needed to
				// calculate the lpm prefix by the Lookup method.
				return depth, baseIdx, val, true
			}
		}

		// end condition, stack is exhausted
		if depth == 0 {
			return
		}

		// go up, backtracking
		depth--
		addr = uint(bs[depth])
		n = pathStack[depth]
	}
}

// LookupShortest does a route lookup for IP and returns the
// shortest matching prefix, the associated value and true for success,
// or false otherwise if no route matched.
//
// It is, so to speak, the opposite of lookup and is only required for very
// special cases.
func (t *Table[V]) LookupShortest(ip netip.Addr) (spm netip.Prefix, val V, ok bool) {
	t.init()

	if depth, baseIdx, val, ok := t.spmByIP(ip); ok {

		// add the bits from higher levels in child trie to pfxLen
		bits := depth*stride + baseIndexToPrefixLen(baseIdx)

		// mask prefix from lookup ip, masked with longest prefix bits.
		spm = netip.PrefixFrom(ip, bits).Masked()

		return spm, val, ok
	}
	return
}

// spmByIP does a route lookup for IP with shortest prefix match.
// Returns also depth and baseIdx for Contains to retrieve the
// spm prefix out of the prefix tree.
func (t *Table[V]) spmByIP(ip netip.Addr) (depth int, baseIdx uint, val V, ok bool) {
	t.init()

	// some needed values, see below
	is4 := ip.Is4()

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)

	// keep the spm alloc free, don't use ip.AsSlice here
	a16 := ip.As16()
	bs := a16[:]
	if is4 {
		bs = bs[12:]
	}
	// depth index for the child trie
	depth = 0
	for {
		addr := uint(bs[depth]) // stride = 8!

		// skip intermediate nodes
		if len(n.prefixes.values) != 0 {
			// forward test, no level backtracking, take the first spm
			if baseIdx, val, ok = n.prefixes.spmByIndex(addrToBaseIndex(addr)); ok {
				return depth, baseIdx, val, ok
			}
		}

		// descend down to next child level
		child := n.children.get(addr)

		// stop condition
		if child == nil {
			return
		}

		// next round
		depth++
		n = child
	}
}

// OverlapsPrefix reports whether any IP in pfx matches a route in the table.
func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool {
	t.init()
	// always normalize the prefix
	pfx = pfx.Masked()

	// some needed values, see below
	bits := pfx.Bits()
	ip := pfx.Addr()
	is4 := ip.Is4()

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)

	// keep the overlaps alloc free, don't use ip.AsSlice here
	a16 := ip.As16()
	bs := a16[:]
	if is4 {
		bs = bs[12:]
	}

	// depth index for the child trie
	depth := 0
	addr := uint(bs[depth])

	for {

		// last prefix chunk reached
		if bits <= stride {
			return n.overlapsPrefix(addr, bits)
		}

		// still in the middle of prefix chunks
		// test if any route overlaps prefix´ addr chunk so far

		// but skip intermediate nodes, no routes to test?
		if len(n.prefixes.values) != 0 {
			if _, _, ok := n.prefixes.lpmByAddr(addr); ok {
				return true
			}
		}

		// no overlap so far, go down to next child
		child := n.children.get(addr)

		// no more children to explore, there can't be an overlap
		if child == nil {
			return false
		}

		// next round
		depth++
		addr = uint(bs[depth])
		bits -= stride
		n = child
	}
}

// Overlaps reports whether any IP in the table matches a route in the
// other table.
func (t *Table[V]) Overlaps(o *Table[V]) bool {
	t.init()
	o.init()
	return t.rootV4.overlapsRec(o.rootV4) || t.rootV6.overlapsRec(o.rootV6)
}

// Union combines two tables, changing the receiver table.
// If there are duplicate entries, the value is taken from the other table.
func (t *Table[V]) Union(o *Table[V]) {
	t.init()
	o.init()
	t.rootV4.unionRec(o.rootV4)
	t.rootV6.unionRec(o.rootV6)
}

// Clone returns a copy of the routing table.
// The payloads V are copied using assignment, so this is a shallow clone.
func (t *Table[V]) Clone() *Table[V] {
	t.init()

	c := new(Table[V])
	c.init()

	c.rootV4 = t.rootV4.cloneRec()
	c.rootV6 = t.rootV6.cloneRec()

	return c
}
