[![Go Reference](https://pkg.go.dev/badge/github.com/gaissmai/bart.svg)](https://pkg.go.dev/github.com/gaissmai/bart#section-documentation)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/gaissmai/bart)
[![CI](https://github.com/gaissmai/bart/actions/workflows/go.yml/badge.svg)](https://github.com/gaissmai/bart/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/gaissmai/bart/badge.svg)](https://coveralls.io/github/gaissmai/bart)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)

# package bart

`package bart` provides a Balanced-Routing-Table (BART).

BART is balanced in terms of memory consumption versus
lookup time.

The lookup time is by a factor of ~2 slower on average as the
routing algorithms ART, SMART, CPE, ... but reduces the memory
consumption by an order of magnitude in comparison.

BART is a multibit-trie, using the _baseIndex_ function from the
ART algorithm to build the complete binary prefix tree (CBT)
for each stride.

The second key factor is popcount level compression
and backtracking along the CBT prefix tree in O(k).

The CBT is implemented as a bitvector, backtracking is just
a matter of fast cache friendly bitmask operations.

Due to the cache locality of the popcount compressed CBT,
the backtracking algorithm is as fast as possible.

# API

 ... still under development, but the library is ready to use.

```golang
  import "github.com/gaissmai/bart"
  
  type Table[V any] struct {
  	// Has unexported fields.
  }
      Table is an IPv4 and IPv6 routing table with payload V. The zero value is
      ready to use.
  
  func (t *Table[V]) Insert(pfx netip.Prefix, val V)
  func (t *Table[V]) Delete(pfx netip.Prefix)
  func (t *Table[V]) Union(o *Table[V])
  func (t *Table[V]) Clone() *Table[V]
  
  func (t *Table[V]) Get(ip netip.Addr) (val V, ok bool)
  func (t *Table[V]) Lookup(ip netip.Addr) (lpm netip.Prefix, val V, ok bool)
  func (t *Table[V]) LookupShortest(ip netip.Addr) (spm netip.Prefix, val V, ok bool)

  func (t *Table[V]) Overlaps(o *Table[V]) bool
  func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool
  
  func (t *Table[V]) String() string
  func (t *Table[V]) Fprint(w io.Writer) error
  func (t *Table[V]) MarshalText() ([]byte, error)
  func (t *Table[V]) MarshalJSON() ([]byte, error)

  func (t *Table[V]) DumpList(is4 bool) []DumpListNode[V]
```

# benchmarks

Please see the extensive [benchmarks](https://github.com/gaissmai/iprbench) comparing `bart` with other IP routing table implementations.

# CONTRIBUTION

Please open an issue for discussion before sending a pull request.

# CREDIT

Credits for many inspirations go to the clever guys at tailscale,
to Daniel Lemire for the super-fast bitset package and
to Donald E. Knuth for the **ART** routing algorithm and
all the rest of his *Art* and for keeping important algorithms
in the public domain!

# LICENSE

MIT
