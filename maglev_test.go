package maglev

import (
	"fmt"
	"log"
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

	mm := NewMaglev(names, lookupSizeM)
	v, err := mm.Get("IP1")
	log.Println("node1:", v)
	v, _ = mm.Get("IP2")
	log.Println("node2:", v)
	v, _ = mm.Get("IPasdasdwni2")
	log.Println("node3:", v)
	log.Println("lookup:", mm.lookup)
	if err := mm.Remove("backend-0"); err != nil {
		t.Error("Remove failed", err)
	}
	log.Println("lookup:", mm.lookup)
	v, _ = mm.Get("IPasdasdwni2")
	log.Println("node3-D:", v)

	if err := mm.Remove("backend-1"); err != nil {
		t.Error("Remove failed", err)
	}
	v, _ = mm.Get("IP2")
	log.Println("node2-D:", v)

	mm.Remove("backend-2")
	mm.Remove("backend-3")
	mm.Remove("backend-4")

	if _, err = mm.Get("IP1"); err == nil {
		t.Error("Empty handle error")
	}
}
