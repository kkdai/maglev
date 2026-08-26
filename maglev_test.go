package maglev

import (
	"errors"
	"fmt"
	"slices"
	"testing"
)

const (
	sizeN       = 5
	lookupSizeM = 13 // must be a prime number
)

// backends returns n deterministic backend names: backend-0 .. backend-(n-1).
func backends(n int) []string {
	names := make([]string, n)
	for i := range n {
		names[i] = fmt.Sprintf("backend-%d", i)
	}
	return names
}

// newTestMaglev builds a Maglev with the default test sizes.
func newTestMaglev(t *testing.T) (*Maglev, []string) {
	t.Helper()

	names := backends(sizeN)
	mg, err := NewMaglev(names, lookupSizeM)
	if err != nil {
		t.Fatal("NewMaglev failed:", err)
	}
	return mg, names
}

func TestSimple(t *testing.T) {
	mg, names := newTestMaglev(t)

	got, err := mg.Get("IP1")
	if err != nil {
		t.Fatal("Get failed:", err)
	}
	if !slices.Contains(names, got) {
		t.Errorf("Get returned %q, which is not one of %v", got, names)
	}
}

func TestNonPrimeTableSize(t *testing.T) {
	if _, err := NewMaglev(backends(sizeN), 12); !errors.Is(err, ErrTableSizeNotPrime) {
		t.Errorf("NewMaglev(_, 12) error = %v, want ErrTableSizeNotPrime", err)
	}
}

func TestTooManyBackends(t *testing.T) {
	if _, err := NewMaglev(backends(lookupSizeM+1), lookupSizeM); !errors.Is(err, ErrTooManyBackends) {
		t.Errorf("NewMaglev with too many backends: error = %v, want ErrTooManyBackends", err)
	}

	mg, _ := newTestMaglev(t)
	if err := mg.Set(backends(lookupSizeM + 1)); !errors.Is(err, ErrTooManyBackends) {
		t.Errorf("Set with too many backends: error = %v, want ErrTooManyBackends", err)
	}

	// Fill the table completely, then one more Add must be rejected.
	if err := mg.Set(backends(lookupSizeM)); err != nil {
		t.Fatal("Set failed:", err)
	}
	if err := mg.Add("backend-extra"); !errors.Is(err, ErrTooManyBackends) {
		t.Errorf("Add on a full table: error = %v, want ErrTooManyBackends", err)
	}
}

func TestAddRemoveErrors(t *testing.T) {
	mg, names := newTestMaglev(t)

	if err := mg.Add(names[0]); !errors.Is(err, ErrBackendExists) {
		t.Errorf("Add duplicate: error = %v, want ErrBackendExists", err)
	}
	if err := mg.Remove("no-such-backend"); !errors.Is(err, ErrBackendNotFound) {
		t.Errorf("Remove unknown: error = %v, want ErrBackendNotFound", err)
	}
}

func TestClear(t *testing.T) {
	mg, _ := newTestMaglev(t)

	mg.Clear()
	if _, err := mg.Get("IP1"); !errors.Is(err, ErrNoBackends) {
		t.Errorf("Get after Clear: error = %v, want ErrNoBackends", err)
	}
	if got := mg.Backends(); len(got) != 0 {
		t.Errorf("Backends after Clear = %v, want empty", got)
	}

	// The table size survives Clear, so the Maglev stays usable.
	if err := mg.Add("backend-0"); err != nil {
		t.Fatal("Add after Clear failed:", err)
	}
	got, err := mg.Get("IP1")
	if err != nil {
		t.Fatal("Get after refill failed:", err)
	}
	if got != "backend-0" {
		t.Errorf("Get = %q, want backend-0", got)
	}
}

func TestSetAddRemove(t *testing.T) {
	mg, names := newTestMaglev(t)

	if err := mg.Add("backend-test"); err != nil {
		t.Fatal("Add failed:", err)
	}
	assertConsistent(t, mg, len(names)+1)
	if !slices.Contains(mg.Backends(), "backend-test") {
		t.Error("backend-test was not added")
	}
	if len(names) != sizeN {
		t.Error("the caller's slice was modified")
	}

	if err := mg.Remove("backend-test"); err != nil {
		t.Fatal("Remove failed:", err)
	}
	if err := mg.Remove(names[0]); err != nil {
		t.Fatal("Remove failed:", err)
	}
	assertConsistent(t, mg, len(names)-1)
	if slices.Contains(mg.Backends(), "backend-test") {
		t.Error("backend-test was not removed")
	}
	if len(names) != sizeN {
		t.Error("the caller's slice was modified")
	}

	if err := mg.Set([]string{"backend-0", "backend-1"}); err != nil {
		t.Fatal("Set failed:", err)
	}
	assertConsistent(t, mg, 2)
}

// assertConsistent checks the internal invariants: the table is fully sized,
// there is one preference list per backend, the backend list stays sorted, and
// every slot points at a real backend.
func assertConsistent(t *testing.T, mg *Maglev, wantBackends int) {
	t.Helper()

	if len(mg.nodeList) != wantBackends {
		t.Fatalf("backend count = %d, want %d", len(mg.nodeList), wantBackends)
	}
	if !slices.IsSorted(mg.nodeList) {
		t.Errorf("backend list is not sorted: %v", mg.nodeList)
	}
	if uint64(len(mg.lookup)) != mg.m {
		t.Errorf("lookup size = %d, want %d", len(mg.lookup), mg.m)
	}
	if len(mg.permutation) != wantBackends {
		t.Errorf("permutation size = %d, want %d", len(mg.permutation), wantBackends)
	}
	for slot, owner := range mg.lookup {
		if owner < 0 || owner >= wantBackends {
			t.Fatalf("lookup[%d] = %d, out of range for %d backends", slot, owner, wantBackends)
		}
	}
}

func TestDistribution(t *testing.T) {
	mg, names := newTestMaglev(t)

	// Every backend should own at least one slot of a 13-slot table.
	owners := make(map[string]int)
	for _, slot := range mg.lookup {
		owners[mg.nodeList[slot]]++
	}
	if len(owners) != len(names) {
		t.Errorf("%d backends own slots, want %d (%v)", len(owners), len(names), owners)
	}

	// Removing backends one by one keeps lookups working until none is left.
	for _, name := range names {
		if err := mg.Remove(name); err != nil {
			t.Fatal("Remove failed:", err)
		}
	}
	if _, err := mg.Get("IP1"); !errors.Is(err, ErrNoBackends) {
		t.Errorf("Get with no backends: error = %v, want ErrNoBackends", err)
	}
}

func TestDistributionCoherency(t *testing.T) {
	names1 := backends(sizeN)

	// Same set of backends, reversed insertion order: the tables must match.
	names2 := slices.Clone(names1)
	slices.Reverse(names2)

	mg1, err := NewMaglev(names1, lookupSizeM)
	if err != nil {
		t.Fatal("NewMaglev failed:", err)
	}
	mg2, err := NewMaglev(names2, lookupSizeM)
	if err != nil {
		t.Fatal("NewMaglev failed:", err)
	}

	for i := range 1024 {
		key := fmt.Sprintf("IP%d", i)
		got1, err := mg1.Get(key)
		if err != nil {
			t.Fatal("Get failed:", err)
		}
		got2, err := mg2.Get(key)
		if err != nil {
			t.Fatal("Get failed:", err)
		}
		if got1 != got2 {
			t.Fatalf("Get(%q) = %q and %q, want the same backend", key, got1, got2)
		}
	}
}

func TestRemovedBackend(t *testing.T) {
	mg, names := newTestMaglev(t)

	keys := make([]string, 1024)
	for i := range keys {
		keys[i] = fmt.Sprintf("IP%d", i)
	}

	found := distinctBackends(t, mg, keys)
	if len(found) < 2 {
		t.Errorf("keys resolved to %d backends, want the load spread over more", len(found))
	}

	// Leave exactly one backend standing; every key must resolve to it.
	survivor := names[3]
	for _, name := range names {
		if name == survivor {
			continue
		}
		if err := mg.Remove(name); err != nil {
			t.Fatal("Remove failed:", err)
		}
	}

	found = distinctBackends(t, mg, keys)
	if len(found) != 1 || found[0] != survivor {
		t.Errorf("keys resolved to %v, want only %q", found, survivor)
	}
}

// distinctBackends resolves every key and returns the distinct backends seen.
func distinctBackends(t *testing.T, mg *Maglev, keys []string) []string {
	t.Helper()

	var found []string
	for _, key := range keys {
		name, err := mg.Get(key)
		if err != nil {
			t.Fatal("Get failed:", err)
		}
		if !slices.Contains(found, name) {
			found = append(found, name)
		}
	}
	return found
}

func TestMinimalDisruption(t *testing.T) {
	// Removing one backend out of N should move roughly 1/N of the keys: the
	// ones it owned. The table has to be much larger than the backend count
	// for that property to hold, which is what BigM is for.
	const nodes = 10
	mg, err := NewMaglev(backends(nodes), BigM)
	if err != nil {
		t.Fatal("NewMaglev failed:", err)
	}

	keys := make([]string, 10000)
	for i := range keys {
		keys[i] = fmt.Sprintf("IP%d", i)
	}

	before := make(map[string]string, len(keys))
	for _, key := range keys {
		name, err := mg.Get(key)
		if err != nil {
			t.Fatal("Get failed:", err)
		}
		before[key] = name
	}

	const removed = "backend-2"
	if err := mg.Remove(removed); err != nil {
		t.Fatal("Remove failed:", err)
	}

	var moved, owned int
	for _, key := range keys {
		name, err := mg.Get(key)
		if err != nil {
			t.Fatal("Get failed:", err)
		}
		if before[key] == removed {
			owned++
			continue
		}
		if name != before[key] {
			moved++
		}
	}

	// Keys of the removed backend have to move; keys of the survivors should
	// almost all stay put.
	if collateral := float64(moved) / float64(len(keys)); collateral > 0.02 {
		t.Errorf("%.2f%% of untouched keys moved (%d keys), want under 2%%", collateral*100, moved)
	}
	if owned == 0 {
		t.Errorf("%q owned no key, the test proves nothing", removed)
	}
	t.Logf("removed %q: %d keys reassigned, %d survivor keys moved", removed, owned, moved)
}

func BenchmarkGet(b *testing.B) {
	mg, err := NewMaglev(backends(100), BigM)
	if err != nil {
		b.Fatal("NewMaglev failed:", err)
	}

	keys := make([]string, 1024)
	for i := range keys {
		keys[i] = fmt.Sprintf("IP%d", i)
	}

	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		if _, err := mg.Get(keys[i%len(keys)]); err != nil {
			b.Fatal("Get failed:", err)
		}
	}
}
