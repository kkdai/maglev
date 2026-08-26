Maglev: A Google Maglev Hashing Algorithm implement in Golang
==============

[![GoDoc](https://pkg.go.dev/badge/github.com/kkdai/maglev.svg)](https://pkg.go.dev/github.com/kkdai/maglev) [![](https://goreportcard.com/badge/github.com/kkdai/maglev)](https://goreportcard.com/badge/github.com/kkdai/maglev) ![Go](https://github.com/kkdai/maglev/workflows/Go/badge.svg)


![](http://www.evanlin.com/images/2016/maglev1.png)


What is Maglev
=============

Maglev is Google’s network load balancer. It is a
large distributed software system that runs on commodity
Linux servers. Unlike traditional hardware network load
balancers, it does not require a specialized physical rack
deployment, and its capacity can be easily adjusted by
adding or removing servers. 
(cite from [paper](http://static.googleusercontent.com/media/research.google.com/zh-TW//pubs/archive/44824.pdf))


### Here is a Chinese reading note about Maglev: [[論文中文導讀] Maglev : A Fast and Reliable Software Network Load Balancer (using Consistent Hashing)](http://www.evanlin.com/maglev/)

How it works
=============

The package keeps a lookup table of `M` slots (`M` must be prime) and assigns
every slot to one of the `N` backends:

1. **Preference lists** — each backend gets two independent hashes of its name,
   an `offset` and a `skip` (xxhash, salted two different ways so the two
   values stay independent). Its preference list is
   `(offset + j*skip) mod M` for `j = 0..M-1`. Because `M` is prime and
   `1 <= skip < M`, every list is a full permutation of the table.
2. **Populate** — backends take turns claiming the most preferred slot they
   have not lost yet, until every slot is owned. Turn-taking is what keeps the
   table balanced.
3. **Lookup** — `Get(key)` is one hash plus one table read, so it is O(1) and
   allocation free.

Because the backend list is sorted before the table is built, the result
depends only on the *set* of backends, never on the order they were added in.
Removing one backend out of `N` moves the ~`1/N` of keys it owned and leaves
almost every other key in place (see `TestMinimalDisruption`).

Pick `M` much larger than `N` — the paper suggests at least 100x — otherwise
the slots do not spread evenly. `BigM` (65537) is provided as a sane default.

Installation and Usage
=============

Requires Go 1.24 or newer.

Install
---------------
```
go get github.com/kkdai/maglev
```

Usage
---------------

```go
package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/kkdai/maglev"
)

func main() {
	names := make([]string, 5)
	for i := range names {
		names[i] = fmt.Sprintf("backend-%d", i)
	}
	// backend-0 ~ backend-4

	// The lookup table size must be a prime number and >= len(names).
	mm, err := maglev.NewMaglev(names, 13)
	if err != nil {
		log.Fatal("NewMaglev failed: ", err)
	}

	a, err := mm.Get("10.0.0.1")
	if err != nil {
		log.Fatal("Get failed: ", err)
	}
	b, _ := mm.Get("10.0.0.3")
	fmt.Println("10.0.0.1 ->", a) // 10.0.0.1 -> backend-4
	fmt.Println("10.0.0.3 ->", b) // 10.0.0.3 -> backend-0

	// Take backend-0 out. Only the keys it owned are reassigned.
	if err := mm.Remove("backend-0"); err != nil {
		log.Fatal("Remove failed: ", err)
	}

	a, _ = mm.Get("10.0.0.1")
	b, _ = mm.Get("10.0.0.3")
	fmt.Println("10.0.0.1 ->", a) // 10.0.0.1 -> backend-4 (unchanged)
	fmt.Println("10.0.0.3 ->", b) // 10.0.0.3 -> backend-4 (moved off backend-0)

	// Errors are sentinel values, match them with errors.Is.
	if err := mm.Remove("backend-0"); errors.Is(err, maglev.ErrBackendNotFound) {
		fmt.Println("already gone")
	}
}
```

API
---------------

| Function | Description |
|----------|-------------|
| `NewMaglev(backends []string, m uint64) (*Maglev, error)` | Build a ring with an `m`-slot lookup table. `m` must be prime. |
| `(*Maglev) Get(obj string) (string, error)` | Resolve a key to a backend. |
| `(*Maglev) Set(backends []string) error` | Replace the whole backend list. |
| `(*Maglev) Add(backend string) error` | Add one backend. |
| `(*Maglev) Remove(backend string) error` | Remove one backend. |
| `(*Maglev) Backends() []string` | Copy of the current backend list, sorted. |
| `(*Maglev) Clear()` | Drop every backend, keep the table size. |

Errors: `ErrTableSizeNotPrime`, `ErrTooManyBackends`, `ErrBackendExists`,
`ErrBackendNotFound`, `ErrNoBackends`. They are wrapped with context, so test
them with `errors.Is` instead of comparing strings.

All methods are safe for concurrent use.

Roadmap
=============

The core algorithm is complete and stable. What is done, what is open, and how
to pick up a task is documented in **[ROADMAP.md](ROADMAP.md)**.

| Done | Open |
|------|------|
| Maglev hashing (permutation + populate) | Fuzz test for table invariants |
| Thread-safe `Add` / `Remove` / `Set` / `Get` | Runnable godoc examples |
| Order-independent, deterministic tables | `[]byte` lookup API |
| Sentinel errors with `errors.Is` support | Pluggable hash function |
| Benchmarks and race-enabled CI | Incremental rebuild on `Add` / `Remove` |
| | Weighted backends |
| | Backend health / draining |
| | Disruption metrics helper |

Contributing
=============

Contributions are welcome, from humans and from AI agents alike.
[ROADMAP.md](ROADMAP.md) is written to be picked up directly: every open task
lists why it matters, what "done" looks like, which files to touch, and the
traps to avoid. It also documents the invariants of this package — most
notably that the key-to-backend mapping is a compatibility promise, so a change
that silently remaps traffic will not be merged.

Before opening a PR:

```sh
go build ./...
go vet ./...
go test -race ./...
gofmt -l .                     # must print nothing
```

Inspired By
---------------

- [Wiki Consistent_hashing](https://en.wikipedia.org/wiki/Consistent_hashing)
- [Go implementation of maglev hashing](https://github.com/dgryski/go-maglev)
- [每天进步一点点——五分钟理解一致性哈希算法(consistent hashing)](http://blog.csdn.net/cywosp/article/details/23397179)
- [Distributed Systems Part-1: A peek into consistent hashing!](https://loveforprogramming.quora.com/Distributed-Systems-Part-1-A-peek-into-consistent-hashing)

Project52
---------------

It is one of my [project 52](https://github.com/kkdai/project52).


License
---------------

This is under the Apache 2.0 [license](LICENSE). See the LICENSE file for details.
