// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// package bart provides a Balanced-Routing-Table (BART).
//
// BART is balanced in terms of memory consumption versus
// lookup time.
//
// The lookup time is by a factor of ~2 slower on average as the
// routing algorithms ART, SMART, CPE, ... but reduces the memory
// consumption by an order of magnitude in comparison.
//
// BART is a multibit-trie with fixed stride length of 8 bits,
// using the _baseIndex_ function from the ART algorithm to
// build the complete-binary-tree (CBT) of prefixes for each stride.
//
// The second key factor is popcount array compression at each stride level
// of the CBT prefix tree and backtracking along the CBT in O(k).
//
// The CBT is implemented as a bitvector, backtracking is just
// a matter of fast cache friendly bitmask operations.
//
// The child array at each stride level is also popcount compressed.
package bart

import (
	"net/netip"
	"sync"
)

// Table is an IPv4 and IPv6 routing table with payload V.
// The zero value is ready to use.
//
// The Table is safe for concurrent readers but not for
// concurrent readers and/or writers.
type Table[V any] struct {
	rootV4 *node[V]
	rootV6 *node[V]

	size4 int
	size6 int

	// BitSets have to be initialized.
	initOnce sync.Once
}

// init BitSets once, so no constructor is needed
func (t *Table[V]) init() {
	// upfront nil test, faster than the atomic load in sync.Once.Do
	// this makes bulk inserts 5% faster and the table is not safe
	// for concurrent writers anyway
	if t.rootV6 != nil {
		return
	}

	t.initOnce.Do(func() {
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

	if !pfx.IsValid() {
		return
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)

	// Do not allocate!
	// As16() is inlined, the prefered AsSlice() is too complex for inlining.
	// starting with go1.23 we can use AsSlice(),
	// see https://github.com/golang/go/issues/56136

	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// 10.0.0.0/8    -> 0
	// 10.12.0.0/15  -> 1
	// 10.12.0.0/16  -> 1
	// 10.12.10.9/32 -> 3
	lastOctetIdx := (bits - 1) / strideLen

	// 10.0.0.0/8    -> 10
	// 10.12.0.0/15  -> 12
	// 10.12.0.0/16  -> 12
	// 10.12.10.9/32 -> 9
	lastOctet := octets[lastOctetIdx]

	// 10.0.0.0/8    -> 8
	// 10.12.0.0/15  -> 7
	// 10.12.0.0/16  -> 8
	// 10.12.10.9/32 -> 8
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// mask the prefix, this is faster than netip.Prefix.Masked()
	lastOctet = lastOctet & netMask(lastOctetBits)

	// find the proper trie node to insert prefix
	for _, octet := range octets[:lastOctetIdx] {
		// descend down to next trie level
		c := n.getChild(octet)
		if c == nil {
			// create and insert missing intermediate child
			c = newNode[V]()
			n.insertChild(octet, c)
		}

		// proceed with next level
		n = c
	}

	// insert prefix/val into node
	if n.insertPrefix(prefixToBaseIndex(lastOctet, lastOctetBits), val) {
		t.sizeUpdate(is4, 1)
	}
}

// Update or set the value at pfx with a callback function.
// The callback function is called with (value, ok) and returns a new value..
//
// If the pfx does not already exist, it is set with the new value.
func (t *Table[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) (newVal V) {
	t.init()
	if !pfx.IsValid() {
		var zero V
		return zero
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)

	// do not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// mask the prefix
	lastOctet = lastOctet & netMask(lastOctetBits)

	// find the proper trie node to update prefix
	for _, octet := range octets[:lastOctetIdx] {
		// descend down to next trie level
		c := n.getChild(octet)
		if c == nil {
			// create and insert missing intermediate child
			c = newNode[V]()
			n.insertChild(octet, c)
		}

		// proceed with next level
		n = c
	}

	// update/insert prefix into node
	var wasPresent bool
	newVal, wasPresent = n.updatePrefix(lastOctet, lastOctetBits, cb)
	if !wasPresent {
		t.sizeUpdate(is4, 1)
	}

	return newVal
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *Table[V]) Get(pfx netip.Prefix) (val V, ok bool) {
	if !pfx.IsValid() {
		return
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// do not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// mask the prefix
	lastOctet = lastOctet & netMask(lastOctetBits)

	// find the proper trie node
	for _, octet := range octets[:lastOctetIdx] {
		c := n.getChild(octet)
		if c == nil {
			// not found
			return
		}
		n = c
	}
	return n.getValue(prefixToBaseIndex(lastOctet, lastOctetBits))
}

// Delete removes pfx from the tree, pfx does not have to be present.
func (t *Table[V]) Delete(pfx netip.Prefix) {
	if !pfx.IsValid() {
		return
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// do not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// mask the prefix
	lastOctet = lastOctet & netMask(lastOctetBits)
	octets[lastOctetIdx] = lastOctet

	// record path to deleted node
	stack := [maxTreeDepth]*node[V]{}

	// run variable as stackPointer, see below
	var i int

	// find the trie node
	for i = range octets {
		// push current node on stack for path recording
		stack[i] = n

		if i == lastOctetIdx {
			break
		}

		// descend down to next level
		c := n.getChild(octets[i])
		if c == nil {
			return
		}
		n = c
	}

	// try to delete prefix in trie node
	if !n.deletePrefix(lastOctet, lastOctetBits) {
		// nothing deleted
		return
	}
	t.sizeUpdate(is4, -1)

	// purge dangling nodes after successful deletion
	for i > 0 {
		if n.isEmpty() {
			// purge empty node from parents children
			parent := stack[i-1]
			parent.deleteChild(octets[i-1])
		}

		// unwind the stack
		i--
		n = stack[i]
	}
}

// Lookup does a route lookup (longest prefix match) for IP and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	if !ip.IsValid() {
		return
	}

	is4 := ip.Is4()

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// do not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*node[V]{}

	// run variable, used after for loop
	var i int
	var octet byte

	// find leaf node
	for i, octet = range octets {
		// push current node on stack for fast backtracking
		stack[i] = n

		// go down in tight loop to leaf node
		c := n.getChild(octet)
		if c == nil {
			break
		}
		n = c
	}

	// start backtracking, unwind the stack
	for depth := i; depth >= 0; depth-- {
		n = stack[depth]
		octet = octets[depth]

		// longest prefix match
		// micro benchmarking: skip if node has no prefixes
		if len(n.prefixes) != 0 {
			if _, val, ok := n.lpm(octetToBaseIndex(octet)); ok {
				return val, true
			}
		}
	}
	return
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
	_, _, val, ok = t.lpmPrefix(pfx)
	return val, ok
}

// LookupPrefixLPM is similar to [Table.LookupPrefix],
// but it returns the lpm prefix in addition to value,ok.
//
// This method is about 20-30% slower than LookupPrefix and should only
// be used if the matching lpm entry is also required for other reasons.
//
// If LookupPrefixLPM is to be used for IP addresses,
// they must be converted to /32 or /128 prefixes.
func (t *Table[V]) LookupPrefixLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool) {
	depth, baseIdx, val, ok := t.lpmPrefix(pfx)

	if ok {
		// calculate the mask from baseIdx and depth
		mask := baseIndexToPrefixLen(baseIdx, depth)

		// calculate the lpm from ip and mask
		lpm, _ = pfx.Addr().Prefix(mask)
	}

	return lpm, val, ok
}

func (t *Table[V]) lpmPrefix(pfx netip.Prefix) (depth int, baseIdx uint, val V, ok bool) {
	if !pfx.IsValid() {
		return
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// do not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// mask the prefix
	lastOctet = lastOctet & netMask(lastOctetBits)
	octets[lastOctetIdx] = lastOctet

	var i int
	var octet byte

	// record path to leaf node
	stack := [maxTreeDepth]*node[V]{}

	// find the node
	for i, octet = range octets[:lastOctetIdx+1] {
		// push current node on stack
		stack[i] = n

		// go down in tight loop
		c := n.getChild(octet)
		if c == nil {
			break
		}
		n = c
	}

	// start backtracking, unwind the stack
	for depth = i; depth >= 0; depth-- {
		n = stack[depth]
		octet = octets[depth]

		// longest prefix match
		// micro benchmarking: skip if node has no prefixes
		if len(n.prefixes) != 0 {

			// only the lastOctet may have a different prefix len
			// all others are just host routes
			idx := uint(0)
			if depth == lastOctetIdx {
				idx = prefixToBaseIndex(octet, lastOctetBits)
			} else {
				idx = octetToBaseIndex(octet)
			}

			baseIdx, val, ok = n.lpm(idx)
			if ok {
				return depth, baseIdx, val, ok
			}
		}
	}
	return
}

// EachLookupPrefix calls yield() for each CIDR covering pfx
// in reverse CIDR sort order, from longest-prefix-match to
// shortest-prefix-match.
//
// If the yield function returns false, the iteration ends prematurely.
func (t *Table[V]) EachLookupPrefix(pfx netip.Prefix, yield func(pfx netip.Prefix, val V) bool) {
	if !pfx.IsValid() {
		return
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// do not allocate
	path := ip.As16()
	octets := path[:]
	if is4 {
		octets = octets[12:]
	}
	copy(path[:], octets[:])

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// mask the prefix
	lastOctet = lastOctet & netMask(lastOctetBits)
	octets[lastOctetIdx] = lastOctet

	// stack of the traversed nodes for reverse ordering of supernets
	stack := [maxTreeDepth]*node[V]{}

	// run variable, used after for loop
	var i int
	var octet byte

	// find last node
	for i, octet = range octets[:lastOctetIdx+1] {
		// push current node on stack
		stack[i] = n

		// go down in tight loop
		c := n.getChild(octet)
		if c == nil {
			break
		}
		n = c
	}

	// start backtracking, unwind the stack
	for depth := i; depth >= 0; depth-- {
		n = stack[depth]

		// microbenchmarking
		if len(n.prefixes) == 0 {
			continue
		}

		// only the lastOctet may have a different prefix len
		if depth == lastOctetIdx {
			if !n.eachLookupPrefix(path, depth, is4, lastOctet, lastOctetBits, yield) {
				// early exit
				return
			}
			continue
		}

		// all others are just host routes
		if !n.eachLookupPrefix(path, depth, is4, octets[depth], strideLen, yield) {
			// early exit
			return
		}
	}
}

// EachSubnet calls yield() for each CIDR covered by pfx.
// If the yield function returns false, the iteration ends prematurely.
//
// The sort order is undefined and you must not rely on it!
func (t *Table[V]) EachSubnet(pfx netip.Prefix, yield func(pfx netip.Prefix, val V) bool) {
	if !pfx.IsValid() {
		return
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)
	if n == nil {
		return
	}

	// do not allocate
	path := ip.As16()
	octets := path[:]
	if is4 {
		octets = octets[12:]
	}
	copy(path[:], octets[:])

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// mask the prefix
	lastOctet = lastOctet & netMask(lastOctetBits)
	octets[lastOctetIdx] = lastOctet

	// find the trie node
	for i, octet := range octets {
		if i == lastOctetIdx {
			_ = n.eachSubnet(path, i, is4, lastOctet, lastOctetBits, yield)
			return
		}

		c := n.getChild(octet)
		if c == nil {
			break
		}

		n = c
	}
}

// OverlapsPrefix reports whether any IP in pfx matches a route in the table.
func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool {
	if !pfx.IsValid() {
		return false
	}

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	// get the root node of the routing table
	n := t.rootNodeByVersion(is4)
	if n == nil {
		return false
	}

	// do not allocate
	a16 := ip.As16()
	octets := a16[:]
	if is4 {
		octets = octets[12:]
	}

	// see comment in Insert()
	lastOctetIdx := (bits - 1) / strideLen
	lastOctet := octets[lastOctetIdx]
	lastOctetBits := bits - (lastOctetIdx * strideLen)

	// mask the prefix
	lastOctet = lastOctet & netMask(lastOctetBits)

	for _, octet := range octets[:lastOctetIdx] {
		// test if any route overlaps prefix´ so far
		if n.lpmTest(octetToBaseIndex(octet)) {
			return true
		}

		// no overlap so far, go down to next c
		c := n.getChild(octet)
		if c == nil {
			return false
		}
		n = c
	}

	return n.overlapsPrefix(lastOctet, lastOctetBits)
}

// Overlaps reports whether any IP in the table matches a route in the
// other table.
func (t *Table[V]) Overlaps(o *Table[V]) bool {
	t.init()
	o.init()

	// t is empty
	if t.Size() == 0 {
		return false
	}

	// o is empty
	if o.Size() == 0 {
		return false
	}

	// at least one v4 is empty
	if t.size4 == 0 || o.size4 == 0 {
		return t.Overlaps6(o)
	}

	// at least one v6 is empty
	if t.size6 == 0 || o.size6 == 0 {
		return t.Overlaps4(o)
	}

	return t.Overlaps4(o) || t.Overlaps6(o)
}

// Overlaps4 reports whether any IPv4 in the table matches a route in the
// other table.
func (t *Table[V]) Overlaps4(o *Table[V]) bool {
	t.init()
	o.init()

	return t.rootV4.overlapsRec(o.rootV4)
}

// Overlaps6 reports whether any IPv6 in the table matches a route in the
// other table.
func (t *Table[V]) Overlaps6(o *Table[V]) bool {
	t.init()
	o.init()

	return t.rootV6.overlapsRec(o.rootV6)
}

// Union combines two tables, changing the receiver table.
// If there are duplicate entries, the value is taken from the other table.
func (t *Table[V]) Union(o *Table[V]) {
	t.init()
	o.init()

	dup4 := t.rootV4.unionRec(o.rootV4)
	dup6 := t.rootV6.unionRec(o.rootV6)

	t.size4 += o.size4 - dup4
	t.size6 += o.size6 - dup6
}

// Clone returns a copy of the routing table.
// The payloads V are copied using assignment, so this is a shallow clone.
func (t *Table[V]) Clone() *Table[V] {
	t.init()

	c := new(Table[V])
	c.init()

	c.rootV4 = t.rootV4.cloneRec()
	c.rootV6 = t.rootV6.cloneRec()

	c.size4 = t.size4
	c.size6 = t.size6

	return c
}

// All may be used in a for/range loop to iterate
// through all the prefixes.
// The sort order is undefined and you must not rely on it!
//
// Prefixes must not be inserted or deleted during iteration, otherwise
// the behavior is undefined. However, value updates are permitted.
//
// If the yield function returns false, the iteration ends prematurely.
func (t *Table[V]) All(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	// respect early exit
	_ = t.rootV4.allRec(zeroPath, 0, true, yield) &&
		t.rootV6.allRec(zeroPath, 0, false, yield)
}

// All4, like [Table.All] but only for the v4 routing table.
func (t *Table[V]) All4(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV4.allRec(zeroPath, 0, true, yield)
}

// All6, like [Table.All] but only for the v6 routing table.
func (t *Table[V]) All6(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV6.allRec(zeroPath, 0, false, yield)
}

// AllSorted may be used in a for/range loop to iterate
// through all the prefixes in natural CIDR sort order.
//
// Prefixes must not be inserted or deleted during iteration, otherwise
// the behavior is undefined. However, value updates are permitted.
//
// If the yield function returns false, the iteration ends prematurely.
func (t *Table[V]) AllSorted(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	// respect early exit
	_ = t.rootV4.allRecSorted(zeroPath, 0, true, yield) &&
		t.rootV6.allRecSorted(zeroPath, 0, false, yield)
}

// All4Sorted, like [Table.AllSorted] but only for the v4 routing table.
func (t *Table[V]) All4Sorted(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV4.allRecSorted(zeroPath, 0, true, yield)
}

// All6Sorted, like [Table.AllSorted] but only for the v6 routing table.
func (t *Table[V]) All6Sorted(yield func(pfx netip.Prefix, val V) bool) {
	t.init()
	t.rootV6.allRecSorted(zeroPath, 0, false, yield)
}

func (t *Table[V]) sizeUpdate(is4 bool, n int) {
	switch is4 {
	case true:
		t.size4 += n
	case false:
		t.size6 += n
	}
}

// Size returns the prefix count.
func (t *Table[V]) Size() int {
	return t.size4 + t.size6
}

// Size4 returns the IPv4 prefix count.
func (t *Table[V]) Size4() int {
	return t.size4
}

// Size6 returns the IPv6 prefix count.
func (t *Table[V]) Size6() int {
	return t.size6
}

// nodes, calculates the IPv4 and IPv6 nodes and returns the sum.
func (t *Table[V]) nodes() int {
	t.init()
	return t.rootV4.numNodesRec() + t.rootV6.numNodesRec()
}
