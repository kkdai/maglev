// Package maglev implements Google's Maglev consistent hashing algorithm.
//
// A Maglev instance owns a lookup table of M slots (M must be prime) and
// assigns every slot to one of the N configured backends. Resolving a key is a
// single hash plus one table read, and adding or removing a backend only
// reshuffles roughly 1/N of the slots, so existing keys keep landing on the
// same backend.
//
// See the original paper for the details of the algorithm:
// https://research.google/pubs/pub44824/
package maglev

import (
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sync"

	"github.com/cespare/xxhash/v2"
)

// BigM is a convenient prime lookup table size. The paper recommends a table
// at least ~100x larger than the number of backends so that the slots spread
// evenly.
const BigM uint64 = 65537

// Salts for the two independent hashes each backend needs: one picks where its
// preference list starts, the other how far it steps between preferences.
const (
	offsetSalt byte = 0xba
	skipSalt   byte = 0xbe
)

// Errors reported by this package. They are wrapped with context, so test for
// them with errors.Is rather than comparing error strings.
var (
	ErrTableSizeNotPrime = errors.New("maglev: lookup table size is not a prime number")
	ErrTooManyBackends   = errors.New("maglev: more backends than lookup table slots")
	ErrBackendExists     = errors.New("maglev: backend already exists")
	ErrBackendNotFound   = errors.New("maglev: backend not found")
	ErrNoBackends        = errors.New("maglev: no backends configured")
)

// Maglev is a consistent hash ring. It is safe for concurrent use.
type Maglev struct {
	mu sync.RWMutex // guards every field below

	m           uint64     // number of lookup table slots, always prime
	nodeList    []string   // backend names, always kept sorted
	permutation [][]uint64 // permutation[i] is backend i's preference list of slots
	lookup      []int      // lookup[slot] is the index in nodeList owning that slot
}

// NewMaglev returns a Maglev populated with backends and a lookup table of m
// slots. m must be a prime number and no smaller than len(backends).
func NewMaglev(backends []string, m uint64) (*Maglev, error) {
	if !big.NewInt(0).SetUint64(m).ProbablyPrime(1) {
		return nil, fmt.Errorf("%w: %d", ErrTableSizeNotPrime, m)
	}

	mg := &Maglev{m: m}
	if err := mg.Set(backends); err != nil {
		return nil, err
	}
	return mg, nil
}

// Set replaces the whole backend list and rebuilds the lookup table.
func (mg *Maglev) Set(backends []string) error {
	mg.mu.Lock()
	defer mg.mu.Unlock()

	if uint64(len(backends)) > mg.m {
		return fmt.Errorf("%w: %d backends, %d slots", ErrTooManyBackends, len(backends), mg.m)
	}

	// Clone so that later edits by the caller cannot corrupt the table.
	mg.nodeList = slices.Clone(backends)
	mg.rebuild()
	return nil
}

// Add inserts a single backend. It reports ErrBackendExists if the backend is
// already present.
func (mg *Maglev) Add(backend string) error {
	mg.mu.Lock()
	defer mg.mu.Unlock()

	// nodeList is kept sorted, so a binary search finds both the duplicate and
	// the insertion point in one step.
	i, found := slices.BinarySearch(mg.nodeList, backend)
	if found {
		return fmt.Errorf("%w: %q", ErrBackendExists, backend)
	}
	if uint64(len(mg.nodeList)) >= mg.m {
		return fmt.Errorf("%w: %d slots", ErrTooManyBackends, mg.m)
	}

	mg.nodeList = slices.Insert(mg.nodeList, i, backend)
	mg.rebuild()
	return nil
}

// Remove deletes a single backend. It reports ErrBackendNotFound if the
// backend is unknown.
func (mg *Maglev) Remove(backend string) error {
	mg.mu.Lock()
	defer mg.mu.Unlock()

	i, found := slices.BinarySearch(mg.nodeList, backend)
	if !found {
		return fmt.Errorf("%w: %q", ErrBackendNotFound, backend)
	}

	mg.nodeList = slices.Delete(mg.nodeList, i, i+1)
	mg.rebuild()
	return nil
}

// Clear drops every backend. The lookup table size is kept, so the Maglev can
// be refilled with Set or Add.
func (mg *Maglev) Clear() {
	mg.mu.Lock()
	defer mg.mu.Unlock()

	mg.nodeList = nil
	mg.permutation = nil
	mg.lookup = nil
}

// Backends returns a copy of the current backend list, in sorted order.
func (mg *Maglev) Backends() []string {
	mg.mu.RLock()
	defer mg.mu.RUnlock()

	return slices.Clone(mg.nodeList)
}

// Get returns the backend that owns obj. It reports ErrNoBackends when no
// backend is configured.
func (mg *Maglev) Get(obj string) (string, error) {
	mg.mu.RLock()
	defer mg.mu.RUnlock()

	if len(mg.nodeList) == 0 {
		return "", ErrNoBackends
	}

	slot := hashKey(obj) % mg.m
	return mg.nodeList[mg.lookup[slot]], nil
}

// rebuild recomputes the preference lists and the lookup table from nodeList.
// The caller must hold mg.mu for writing.
func (mg *Maglev) rebuild() {
	// Sorting makes the result depend only on the set of backends, never on
	// the order they were added in.
	slices.Sort(mg.nodeList)
	mg.buildPermutations()
	mg.buildLookup()
}

// buildPermutations computes each backend's preference list: backend i prefers
// slot (offset_i + j*skip_i) mod m for j = 0..m-1. Because m is prime and
// 1 <= skip < m, every row is a full permutation of the table, which
// guarantees a backend can always walk down its list until it finds a free
// slot.
func (mg *Maglev) buildPermutations() {
	if len(mg.nodeList) == 0 {
		mg.permutation = nil
		return
	}

	mg.permutation = make([][]uint64, len(mg.nodeList))
	for i, node := range mg.nodeList {
		name := []byte(node)
		offset := saltedHash(offsetSalt, name) % mg.m
		skip := saltedHash(skipSalt, name)%(mg.m-1) + 1

		row := make([]uint64, mg.m)
		// Stepping instead of computing offset+j*skip keeps the arithmetic
		// bounded by m, so it cannot overflow for large tables.
		slot := offset
		for j := range mg.m {
			row[j] = slot
			slot = (slot + skip) % mg.m
		}
		mg.permutation[i] = row
	}
}

// buildLookup fills the lookup table by letting the backends take turns
// claiming the most preferred slot they have not lost yet, until every slot is
// owned. This is the "populate" procedure from the paper.
func (mg *Maglev) buildLookup() {
	n := uint64(len(mg.nodeList))
	if n == 0 {
		mg.lookup = nil
		return
	}

	lookup := make([]int, mg.m)
	for i := range lookup {
		lookup[i] = -1 // -1 marks a slot nobody owns yet
	}
	next := make([]uint64, n) // next[i]: how far backend i walked down its list

	for filled := uint64(0); ; {
		for i := range n {
			slot := mg.permutation[i][next[i]]
			for lookup[slot] >= 0 { // slot already taken, try the next preference
				next[i]++
				slot = mg.permutation[i][next[i]]
			}

			lookup[slot] = int(i)
			next[i]++
			filled++
			if filled == mg.m {
				mg.lookup = lookup
				return
			}
		}
	}
}

func hashKey(obj string) uint64 {
	return xxhash.Sum64String(obj)
}

// saltedHash mixes salt into the hash so that offset and skip are derived from
// independent hash values. Reusing a single unsalted hash for both would
// correlate them and weaken the permutation.
func saltedHash(salt byte, data []byte) uint64 {
	d := xxhash.New()
	d.Write([]byte{salt})
	d.Write(data)
	return d.Sum64()
}
