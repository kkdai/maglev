Maglev: The Google Maglev Hashing Algorithm implemented in Golang
==============

[![GoDoc](https://godoc.org/github.com/hardpointlabs/maglev?status.svg)](https://godoc.org/github.com/hardpointlabs/maglev)  [![](https://goreportcard.com/badge/github.com/hardpointlabs/maglev)](https://goreportcard.com/badge/github.com/hardpointlabs/maglev)

What is Maglev
=============

Maglev is [Google's network load balancer](https://research.google/pubs/maglev-a-fast-and-reliable-software-network-load-balancer/). One novelty in its implementation is a new form of consistent hashing with O(1) lookup time. This library implements the lookup table described in the paper for use in generic load balancing applications.

| Operation | Average case | Worst case |
| :------- | :------: | :-------: |
| Key Lookup     | O(1) | O(1)    |
| Lookup table rebuild   | O(M * N)   | O(M * N) |

(`M` is the size of lookup table, `N` is the number of nodes in the hash ring)

This repo forks the original work from `kkdai/maglev` and uses xxHash in place of SipHash-2-4 for speed and license-friendliness.

Installation and Usage
=============


Install
---------------
```
go get github.com/hardpointlabs/maglev
```

Usage
---------------



```go

func main() {
	sizeN := 5
	lookupSizeM := 13 //(must be prime number)

	var names []string
	for i := 0; i < sizeN; i++ {
		names = append(names, fmt.Sprintf("backend-%d", i))
	}
	//backend-0 ~ backend-4

	mm, err := NewMaglev(names, lookupSizeM)
	if err != nil {
		log.Fatal("NewMaglev failed:", err)
	}
	v, err := mm.Get("IP1")
	fmt.Println("node1:", v)
	//node1: backend-2
	v, _ = mm.Get("IP2")
	log.Println("node2:", v)
	//node2: backend-1
	v, _ = mm.Get("IPasdasdwni2")
	log.Println("node3:", v)
	//node3: backend-0

	if err := mm.Remove("backend-0"); err != nil {
		log.Fatal("Remove failed", err)
	}
	v, _ = mm.Get("IPasdasdwni2")
	log.Println("node3-D:", v)
	//node3-D: Change from "backend-0" to "backend-1"
}
```

License
---------------

This is under the Apache 2.0 [license](LICENSE). See the LICENSE file for details.
