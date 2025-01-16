// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"encoding/json"
	"net/netip"
	"slices"
)

// DumpListNode contains CIDR, value and list of subnets (tree childrens).
type DumpListNode[V any] struct {
	CIDR    netip.Prefix      `json:"cidr"`
	Value   V                 `json:"value"`
	Subnets []DumpListNode[V] `json:"subnets,omitempty"`
}

// MarshalJSON dumps table into two sorted lists: for ipv4 and ipv6.
// Every root and subnet are array, not map, because the order matters.
func (t *Table[V]) MarshalJSON() ([]byte, error) {
	t.init()

	result := struct {
		Ipv4 []DumpListNode[V] `json:"ipv4,omitempty"`
		Ipv6 []DumpListNode[V] `json:"ipv6,omitempty"`
	}{
		Ipv4: t.DumpList(true),
		Ipv6: t.DumpList(false),
	}

	buf, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// DumpList dumps ipv4 or ipv6 tree into list of roots and their subnets.
// It can be used to analyze tree or build custom json representation.
func (t *Table[V]) DumpList(is4 bool) []DumpListNode[V] {
	t.init()
	rootNode := t.rootNodeByVersion(is4)
	if rootNode.isEmpty() {
		return nil
	}

	return rootNode.dumpListRec(0, nil, is4)
}

func (n *node[V]) dumpListRec(parentIdx uint, path []byte, is4 bool) []DumpListNode[V] {
	directKids := n.getKidsRec(parentIdx, path, is4)
	slices.SortFunc(directKids, sortPrefix[V])

	nodes := make([]DumpListNode[V], 0, len(directKids))
	for _, kid := range directKids {
		nodes = append(nodes, DumpListNode[V]{
			CIDR:    kid.cidr,
			Value:   kid.val,
			Subnets: kid.n.dumpListRec(kid.idx, kid.path, is4),
		})
	}

	return nodes
}
