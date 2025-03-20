# package bart

[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/bart.svg)](https://pkg.go.dev/github.com/gaissmai/bart#section-documentation)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/bart)
[![CI](https://github.com/gaissmai/bart/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/bart/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/bart/badge.svg)](https://coveralls.io/github/gaissmai/bart)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

## Overview

`package bart` provides a Balanced-Routing-Table (BART).

BART is balanced in terms of memory consumption versus
lookup time.

The lookup time is by a factor of ~2 slower on average as the
routing algorithms ART, SMART, CPE, ... but reduces the memory
consumption by an order of magnitude in comparison.

BART is a multibit-trie with fixed stride length of 8 bits,
using the _baseIndex_ function from the ART algorithm to
build the complete-binary-tree (CBT) of prefixes for each stride.

The second key factor is popcount array compression at each stride level
of the CBT prefix tree and backtracking along the CBT in O(k).

The CBT is implemented as a bitvector, backtracking is just
a matter of fast cache friendly bitmask operations.

The child array at each stride level is also popcount compressed.

## API

The API changes in v0.4.2, 0.5.3, v0.6.3, v0.10.1, v0.11.0

```golang
  import "github.com/gaissmai/bart"
  
  type Table[V any] struct {
  	// Has unexported fields.
  }
    Table is an IPv4 and IPv6 routing table with payload V. The zero value is
    ready to use.

    The Table is safe for concurrent readers but not for concurrent readers
    and/or writers.

  func (t *Table[V]) Insert(pfx netip.Prefix, val V)
  func (t *Table[V]) Delete(pfx netip.Prefix)
  func (t *Table[V]) Get(pfx netip.Prefix) (val V, ok bool)
  func (t *Table[V]) Update(pfx netip.Prefix, cb func(val V, ok bool) V) (newVal V)

  func (t *Table[V]) Union(o *Table[V])
  func (t *Table[V]) Clone() *Table[V]

  func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool)

  func (t *Table[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool)
  func (t *Table[V]) LookupPrefixLPM(pfx netip.Prefix) (lpm netip.Prefix, val V, ok bool)
  func (t *Table[V]) EachLookupPrefix(pfx netip.Prefix, yield func(pfx netip.Prefix, val V) bool)
  func (t *Table[V]) EachSubnet(pfx netip.Prefix, yield func(pfx netip.Prefix, val V) bool)

  func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool

  func (t *Table[V]) Overlaps(o *Table[V]) bool
  func (t *Table[V]) Overlaps4(o *Table[V]) bool
  func (t *Table[V]) Overlaps6(o *Table[V]) bool

  func (t *Table[V]) Size() int
  func (t *Table[V]) Size4() int
  func (t *Table[V]) Size6() int

  func (t *Table[V]) All(yield func(pfx netip.Prefix, val V) bool)
  func (t *Table[V]) All4(yield func(pfx netip.Prefix, val V) bool)
  func (t *Table[V]) All6(yield func(pfx netip.Prefix, val V) bool)

  func (t *Table[V]) AllSorted(yield func(pfx netip.Prefix, val V) bool)
  func (t *Table[V]) All4Sorted(yield func(pfx netip.Prefix, val V) bool)
  func (t *Table[V]) All6Sorted(yield func(pfx netip.Prefix, val V) bool)

  func (t *Table[V]) String() string
  func (t *Table[V]) Fprint(w io.Writer) error
  func (t *Table[V]) MarshalText() ([]byte, error)
  func (t *Table[V]) MarshalJSON() ([]byte, error)

  func (t *Table[V]) DumpList4() []DumpListNode[V]
  func (t *Table[V]) DumpList6() []DumpListNode[V]
```

## benchmarks

Please see the extensive [benchmarks](https://github.com/gaissmai/iprbench) comparing `bart` with other IP routing table implementations.

Just a teaser, LPM lookups against the full Internet routing table with random probes:

```
goos: linux
goarch: amd64
pkg: github.com/gaissmai/bart
cpu: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz

BenchmarkFullMatchV4/Lookup                     24484828            49.03 ns/op
BenchmarkFullMatchV6/Lookup                     17098262            70.15 ns/op
BenchmarkFullMissV4/Lookup                      24480925            49.15 ns/op
BenchmarkFullMissV6/Lookup                      54955310            21.79 ns/op
```

## CONTRIBUTION

Please open an issue for discussion before sending a pull request.

## CREDIT

Credits for many inspirations go to the clever guys at tailscale,
to Daniel Lemire for the super-fast bitset package and
to Donald E. Knuth for the **ART** routing algorithm and
all the rest of his *Art* and for keeping important algorithms
in the public domain!

## LICENSE

MIT
