package maglev

import (
	"fmt"

	"github.com/dchest/siphash"
)

const (
	bigM uint64 = 65537
)

//Maglev :
type Maglev struct {
	n           uint64 //size of VIP backeds
	m           uint64 //sie of the lookup table
	permutation [][]uint64
}

//NewMaglev :
func NewMaglev(backeds []string, m uint64) *Maglev {
	mag := new(Maglev)
	mag.n = uint64(len(backeds))
	mag.m = m
	return mag
}

func (m *Maglev) generatePopulation(backeds []string) {
	for i := 0; i < len(backeds); i++ {
		bData := []byte(backeds[i])
		offset := siphash.Hash(0xdeadbabe, 0, bData) % bigM
		skip := (siphash.Hash(0xdeadbeef, 0, bData) % (bigM - 1)) + 1

		fmt.Println(offset, skip)
	}
}
