Maglev: A Google Maglev Hashing Algorithm implement in Golang
==============

[![GoDoc](https://godoc.org/github.com/kkdai/maglev?status.svg)](https://godoc.org/github.com/kkdai/maglev)  [![Build Status](https://travis-ci.org/kkdai/maglev.svg?branch=master)](https://travis-ci.org/kkdai/maglev) [![](https://goreportcard.com/badge/github.com/kkdai/maglev)](https://goreportcard.com/badge/github.com/kkdai/maglev) ![Go](https://github.com/kkdai/maglev/workflows/Go/badge.svg)


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

Installation and Usage
=============


Install
---------------
```
go get github.com/kkdai/maglev
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

	mm := NewMaglev(names, lookupSizeM)
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
