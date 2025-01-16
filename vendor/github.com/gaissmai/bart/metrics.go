// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

type tableStats struct {
	// "/ipv4/size:count"  sum of all IPv4 prefixes in the table
	// "/ipv6/size:count"  sum of all IPv6 prefixes in the table
	size4 int
	size6 int

	// "/ipv4/depth:histogram" node distribution over the depth
	// "/ipv6/depth:histogram" node distribution over the depth
	depth4 map[int]int
	depth6 map[int]int

	// "/ipv4/childs:histogram" child distribution, a.k.a fan out
	// "/ipv6/childs:histogram" child distribution, a.k.a fan out
	childs4 map[int]int
	childs6 map[int]int

	// "/ipv4/types:histogram" node-type distribution, is path compression useful?
	// "/ipv6/types:histogram" node-type distribution, is path compression useful?
	types4 map[string]int
	types6 map[string]int

	// "/ipv4/prefixlen:histogram" prefixlen distribution
	// "/ipv6/prefixlen:histogram" prefixlen distribution
	prefixlen4 map[int]int
	prefixlen6 map[int]int
}

// readTableStats returns some metrics of the routing table.
// TODO: contrived, not ready for public ...
// a little bit like runtime/metrics, but much simpler
func (t *Table[V]) readTableStats() map[string]any {
	ret := map[string]any{}

	// init the maps
	stats := tableStats{
		size4: 0,
		size6: 0,
		//
		depth4: map[int]int{},
		depth6: map[int]int{},
		//
		childs4: map[int]int{},
		childs6: map[int]int{},
		//
		types4: map[string]int{},
		types6: map[string]int{},
		//
		prefixlen4: map[int]int{},
		prefixlen6: map[int]int{},
	}

	// walk the routing table, gather stats
	t.walk(func(n *node[V], depth int, is4 bool) {
		switch is4 {
		case true:
			stats.size4 += len(n.prefixes.values)
			stats.childs4[len(n.children.nodes)]++
			stats.depth4[depth]++
			stats.types4[n.hasType().String()]++

			for _, idx := range n.prefixes.allIndexes() {
				_, pfxLen := baseIndexToPrefix(idx)
				stats.prefixlen4[stride*depth+pfxLen]++
			}
		case false:
			stats.size6 += len(n.prefixes.values)
			stats.childs6[len(n.children.nodes)]++
			stats.depth6[depth]++
			stats.types6[n.hasType().String()]++

			for _, idx := range n.prefixes.allIndexes() {
				_, pfxLen := baseIndexToPrefix(idx)
				stats.prefixlen6[stride*depth+pfxLen]++
			}
		}
	})

	ret["/ipv4/size:count"] = stats.size4
	ret["/ipv6/size:count"] = stats.size6
	//
	ret["/ipv4/depth:histogram"] = stats.depth4
	ret["/ipv6/depth:histogram"] = stats.depth6
	//
	ret["/ipv4/childs:histogram"] = stats.childs4
	ret["/ipv6/childs:histogram"] = stats.childs6
	//
	ret["/ipv4/types:histogram"] = stats.types4
	ret["/ipv6/types:histogram"] = stats.types6
	//
	ret["/ipv4/prefixlen:histogram"] = stats.prefixlen4
	ret["/ipv6/prefixlen:histogram"] = stats.prefixlen6

	return ret
}

// walkFunc is the type of the function called by walk to visit each node
// in the routing table. The depth argument is the depth in the tree,
// starting with 0.
// The is4 argument indicates whether the node is from the IPv4 routing
// table or from the IPv6 table.
type walkFunc[V any] func(n *node[V], depth int, is4 bool)

// walk the routing table, calling cb for each node.
func (t *Table[V]) walk(cb walkFunc[V]) {
	t.init()

	is4 := true
	root4 := t.rootNodeByVersion(is4)
	root4.walkRec(cb, 0, is4)

	is4 = false
	root6 := t.rootNodeByVersion(is4)
	root6.walkRec(cb, 0, is4)
}

func (n *node[V]) walkRec(cb walkFunc[V], depth int, is4 bool) {
	cb(n, depth, is4)

	for _, child := range n.children.nodes {
		child.walkRec(cb, depth+1, is4)
	}
}
