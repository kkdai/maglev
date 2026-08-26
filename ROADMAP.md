Roadmap
==============

This file is written for **contributors — human or AI agent — who want to pick
up a task in this repository**. Each item below is self-contained: what it is,
why it matters, what "done" looks like, and which files to touch. Pick one
item, finish it completely, and open a PR.

If you are an AI agent: read the whole of this file before editing anything.
The [Invariants](#invariants-do-not-break-these) section describes properties
that are not obvious from the code, and breaking one of them silently
reshuffles every user's traffic.

Project snapshot
---------------

`github.com/kkdai/maglev` implements Google's Maglev consistent hashing
algorithm ([paper](http://static.googleusercontent.com/media/research.google.com/zh-TW//pubs/archive/44824.pdf)).
The whole implementation is one file.

| File | Contents |
|------|----------|
| `maglev.go` | The entire package: public API, permutation build, populate, lookup. |
| `maglev_test.go` | Unit tests, invariant checks, disruption test, `BenchmarkGet`. |
| `README.md` | User-facing docs, API table, usage example. |
| `ROADMAP.md` | This file. |

Requires Go 1.24+. The only dependency is `github.com/cespare/xxhash/v2`.

Before you start
---------------

```sh
go build ./...
go vet ./...
go test -race ./...
go test -bench=. -benchmem ./...
gofmt -l .                     # must print nothing
```

All five must pass before and after your change. CI runs the same commands.

Conventions:

- **Tests are mandatory.** Every behaviour change needs a test that fails
  before your change and passes after it.
- Errors are package-level sentinels wrapped with `fmt.Errorf("%w: ...")`.
  Add a new `Err...` var rather than a bare `errors.New` inside a function.
- Exported identifiers need doc comments. Comments explain *why*, not *what*.
- The receiver is `mg`, never `m` — `m` is the lookup table size.
- Methods named `build*` and `rebuild` assume the caller already holds
  `mg.mu` for writing. Do not lock inside them.

Invariants (do not break these)
---------------

1. **The lookup table is fully populated.** After any mutation, `len(lookup)`
   is exactly `m`, and every entry is a valid index into `nodeList`. No `-1`
   may survive into `mg.lookup`.
2. **`nodeList` is always sorted.** This is what makes the table depend only on
   the *set* of backends and not on insertion order. `TestDistributionCoherency`
   guards it.
3. **`m` is prime and `m >= len(nodeList)`.** Primality is what guarantees each
   preference row is a full permutation of the table, which is what guarantees
   `buildLookup`'s inner loop terminates. Remove primality and it can loop
   forever or index out of range.
4. **The key → backend mapping is a compatibility promise.** Changing the hash
   salts (`offsetSalt`, `skipSalt`), the hash function, or the sort order
   remaps every existing user's traffic. Any such change needs a new opt-in API
   and a major version bump, never a silent default change.
5. **`Get` does not allocate.** `BenchmarkGet` reports `0 allocs/op`; keep it
   that way. Do not add a defer, a map lookup, or a slice build to that path.
6. **The public API is concurrency safe.** Anything exported takes `mg.mu`.

Status
---------------

Completed:

| Item | Notes |
|------|-------|
| Maglev hashing (permutation + populate) | Follows the paper. |
| Thread-safe `Add` / `Remove` / `Set` / `Get` | Single `RWMutex`. |
| Order-independent, deterministic tables | Backends sorted before build. |
| Sentinel errors with `errors.Is` support | Replaced ad-hoc error strings. |
| Benchmarks and race-enabled CI | `BenchmarkGet`, `go test -race`. |

Open tasks, roughly in the order they are worth doing:

| ID | Task | Effort | Status |
|----|------|--------|--------|
| [R1](#r1-fuzz-test-for-table-invariants) | Fuzz test for table invariants | Small | Not Started |
| [R2](#r2-runnable-godoc-examples) | Runnable godoc examples | Small | Not Started |
| [R3](#r3-byte-lookup-api) | `[]byte` lookup API | Small | Not Started |
| [R4](#r4-pluggable-hash-function) | Pluggable hash function | Medium | Not Started |
| [R5](#r5-incremental-rebuild-on-add--remove) | Incremental rebuild on `Add`/`Remove` | Large | Not Started |
| [R6](#r6-weighted-backends) | Weighted backends | Large | Not Started |
| [R7](#r7-backend-health--draining) | Backend health / draining | Large | Not Started |
| [R8](#r8-disruption-metrics-helper) | Disruption metrics helper | Medium | Not Started |

---

### R1: Fuzz test for table invariants

**Why.** The invariants above are currently checked against hand-picked
backend sets. A fuzzer explores sets the tests never thought of — empty names,
duplicates, very long names, `n` exactly equal to `m`.

**Done when.** `FuzzTable` in `maglev_test.go` takes a seed and a backend list,
builds a `Maglev` over a small prime table, and asserts invariants 1–3 after a
random sequence of `Add`/`Remove`/`Set`/`Clear`. `go test -fuzz=FuzzTable
-fuzztime=60s` runs clean, and any crasher found is committed under
`testdata/fuzz/`.

**Files.** `maglev_test.go` only.

**Watch out.** `f.Add` seeds cannot be `[]string` — encode the backend list as
a single delimited string and split it inside the fuzz target.

---

### R2: Runnable godoc examples

**Why.** pkg.go.dev shows `Example` functions inline. Today a user has to read
the README to see how the package is used.

**Done when.** `example_test.go` exists with at least `ExampleNewMaglev` and
`ExampleMaglev_Remove`, each with an `// Output:` block, and `go test ./...`
verifies them. Keep the outputs stable — they are pinned by the hash seeds.

**Files.** New `example_test.go`, in package `maglev_test` so the examples read
like real user code.

---

### R3: `[]byte` lookup API

**Why.** Callers holding a `[]byte` key (a packet, a header value) have to
convert it to a `string` before they can call `Get`, which allocates on their
side even though `Get` itself does not.

**Done when.** `GetBytes(obj []byte) (string, error)` exists next to `Get`,
both funnel into one internal helper, `Get` still reports `0 allocs/op`, and a
test asserts `Get("x")` and `GetBytes([]byte("x"))` return the same backend.

**Files.** `maglev.go`, `maglev_test.go`, README API table.

---

### R4: Pluggable hash function

**Why.** xxhash with two hardcoded salts is the only option. Users embedding
this in a larger system may need to match an existing hash.

**Done when.** An optional constructor — e.g.
`NewMaglevWithHasher(backends []string, m uint64, h Hasher) (*Maglev, error)` —
accepts a salted hash, `NewMaglev` keeps using xxhash with the current salts
byte-for-byte (invariant 4), and a test proves the default constructor's
mapping is unchanged.

**Files.** `maglev.go`, `maglev_test.go`, README.

**Watch out.** Do not turn `Hasher` into a struct field that `Get` dereferences
through an interface on every call without re-checking the benchmark.

---

### R5: Incremental rebuild on `Add` / `Remove`

**Why.** Every mutation rebuilds everything: `buildPermutations` is O(M·N) time
*and* memory, and `buildLookup` re-runs the full populate. With `BigM` and 100
backends that is ~6.5M `uint64` per `Add`. This is the single biggest
performance problem in the package.

**Done when.** `Add` no longer recomputes the preference rows of backends that
did not change (only the new backend's row is built, and rows are keyed so
removal drops one row instead of rebuilding all), `BenchmarkAdd` and
`BenchmarkRemove` exist and show a clear improvement, and — critically — the
resulting table is **identical** to the one a full rebuild produces. Assert
that equality directly in a test over many random topologies.

**Files.** `maglev.go`, `maglev_test.go`.

**Watch out.** Populate is order-sensitive: backends claim slots in `nodeList`
order. A partial re-populate that does not reproduce the same claim order will
produce a different — and much more disruptive — table. Getting identical
output is the whole difficulty of this task; if you cannot, say so in the PR
rather than relaxing the test.

---

### R6: Weighted backends

**Why.** Real fleets are heterogeneous. Today every backend gets an equal share
of the table.

**Done when.** A backend can be given an integer weight, a weight of `w` earns
roughly `w/Σw` of the slots (assert the distribution within a tolerance over
`BigM`), and the unweighted API keeps producing the exact same table as today.

**Files.** `maglev.go`, `maglev_test.go`, README.

**Watch out.** The natural implementation — enter a backend `w` times in the
populate rotation — must not change the mapping when all weights are 1.
Consider a separate type or constructor rather than overloading `Add`.

---

### R7: Backend health / draining

**Why.** The paper pairs the lookup table with connection tracking so that a
briefly unhealthy backend does not trigger a full reshuffle. Today the only way
to take a backend out is `Remove`, which rebuilds the table.

**Done when.** A backend can be marked unhealthy and skipped by `Get` without
its slots being redistributed permanently, marking it healthy again restores
the original mapping exactly, and a test proves the round trip is lossless.

**Files.** `maglev.go`, `maglev_test.go`, README.

**Watch out.** This interacts with R5 and R6 — read both before designing the
API. Keep the fallback selection deterministic.

---

### R8: Disruption metrics helper

**Why.** The main reason to choose Maglev over modulo hashing is how few keys
move on a topology change. Users have no way to measure that for their own
backend set and table size.

**Done when.** An exported helper reports how many slots change owner between
two tables (or a documented example does the same), the README links to it from
the "How it works" section, and `TestMinimalDisruption` is rewritten to use it
instead of its own bookkeeping.

**Files.** `maglev.go` or a new `disruption.go`, `maglev_test.go`, README.

---

Proposing something else
---------------

Open an issue describing the problem before writing code, especially if it
touches invariant 4. A change that silently remaps traffic will not be merged,
however good the idea is.
