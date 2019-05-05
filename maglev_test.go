package maglev

import (
	"fmt"
	"log"
	"sort"
	"testing"
)

// func TestPopulate(t *testing.T) {

// 	var tests = []struct {
// 		dead []int
// 		want []int
// 	}{
// 		{nil, []int{1, 0, 1, 0, 2, 2, 0}},
// 		{[]int{1}, []int{0, 0, 0, 0, 2, 2, 2}},
// 	}

// 	permutations := [][]uint64{
// 		{3, 0, 4, 1, 5, 2, 6},
// 		{0, 2, 4, 6, 1, 3, 5},
// 		{3, 4, 5, 6, 0, 1, 2},
// 	}

// 	for _, tt := range tests {
// 		if got := populate(permutations, tt.dead); !reflect.DeepEqual(got, tt.want) {
// 			t.Errorf("populate(...,%v)=%v, want %v", tt.dead, got, tt.want)
// 		}
// 	}
// }

const sizeN = 5
const lookupSizeM = 13 //need prime and

func TestDistribution(t *testing.T) {
	var names []string
	for i := 0; i < sizeN; i++ {
		names = append(names, fmt.Sprintf("backend-%d", i))
	}
	mm, err := NewMaglev(names, lookupSizeM)
	if err != nil {
		t.Error("Creation failed", err)
	}
	v, err := mm.Get("IP1")
	if err != nil {
		t.Error("Get failed", err)
	}
	log.Println("node1:", v)
	v, err = mm.Get("IP2")
	if err != nil {
		t.Error("Get failed", err)
	}
	log.Println("node2:", v)
	v, err = mm.Get("IPasdasdwni2")
	if err != nil {
		t.Error("Get failed", err)
	}
	log.Println("node3:", v)
	log.Println("lookup:", mm.lookup)
	if err := mm.Remove("backend-0"); err != nil {
		t.Error("Remove failed", err)
	}
	log.Println("lookup:", mm.lookup)
	v, err = mm.Get("IPasdasdwni2")
	if err != nil {
		t.Error("Get failed", err)
	}
	log.Println("node3-D:", v)

	if err := mm.Remove("backend-1"); err != nil {
		t.Error("Remove failed", err)
	}
	v, err = mm.Get("IP2")
	if err != nil {
		t.Error("Get failed", err)
	}
	log.Println("node2-D:", v)

	mm.Remove("backend-2")
	mm.Remove("backend-3")
	mm.Remove("backend-4")

	if _, err = mm.Get("IP1"); err == nil {
		t.Error("Empty handle error")
	}
}

func TestSetAddRemove(t *testing.T) {
	var names []string
	for i := 0; i < sizeN; i++ {
		names = append(names, fmt.Sprintf("backend-%d", i))
	}

	mm, err := NewMaglev(names, lookupSizeM)
	if err != nil {
		t.Error("Creation failed", err)
	}

	err = mm.Add("backend-test")
	if err != nil {
		t.Error("Add failed", err)
	}
	if (uint64)(len(mm.lookup)) != mm.m {
		t.Error("lookup size not correct")
	}
	if len(mm.permutation) != len(mm.nodeList) {
		t.Error("permutation size not correct")
	}
	if len(names) != sizeN {
		t.Error("Original values has been modified")
	}

	err = mm.Remove("backend-test")
	if err != nil {
		t.Error("Remove failed", err)
	}
	err = mm.Remove(names[0])
	if err != nil {
		t.Error("Remove failed", err)
	}
	if (uint64)(len(mm.lookup)) != mm.m {
		t.Error("lookup size not correct")
	}
	if len(mm.permutation) != len(mm.nodeList) {
		t.Error("permutation size not correct")
	}
	if len(names) != sizeN {
		t.Error("Original values has been modified")
	}

	err = mm.Set([]string{"backend-0", "backend-1"})
	if err != nil {
		t.Error("Remove failed", err)
	}
	if (uint64)(len(mm.lookup)) != mm.m {
		t.Error("lookup size not correct")
	}
	if len(mm.permutation) != len(mm.nodeList) {
		t.Error("permutation size not correct")
	}

	for i := 0; i < lookupSizeM+1; i++ {
		names = append(names, fmt.Sprintf("backend-%d", i))
	}
	err = mm.Set(names)
	if err == nil {
		t.Error("No error thrown when trying to set too many backends")
	}
}

func TestRemovedBackend(t *testing.T) {
	var names []string
	for i := 0; i < sizeN; i++ {
		names = append(names, fmt.Sprintf("backend-%d", i))
	}
	mm, err := NewMaglev(names, lookupSizeM)
	if err != nil {
		t.Error("Creation failed", err)
	}

	var lookUpNames []string
	for i := 0; i < 1024; i++ {
		lookUpNames = append(lookUpNames, fmt.Sprintf("IP%d", i))
	}

	var backendFound []string
	for _, lookupName := range lookUpNames {
		name, err := mm.Get(lookupName)
		if err != nil {
			t.Error("Get failed", err)
		}
		if sort.SearchStrings(backendFound, name) == len(backendFound) {
			backendFound = append(backendFound, name)
		}
	}

	if len(backendFound) < 2 {
		t.Error("Distribution failed")
	}

	for i, name := range names {
		if i != 3 { // Remove a node in the middle
			if err := mm.Remove(name); err != nil {
				t.Error("Remove failed", err)
			}
		}
	}

	backendFound = nil
	for _, lookupName := range lookUpNames {
		name, err := mm.Get(lookupName)
		if err != nil {
			t.Error("Get failed", err)
		}
		if sort.SearchStrings(backendFound, name) == len(backendFound) {
			backendFound = append(backendFound, name)
		}
	}

	if len(backendFound) != 1 {
		t.Error("Distribution failed")
	}
}
