package maglev

import "github.com/dchest/siphash"

const (
	bigM uint64 = 65537
)

//Maglev :
type Maglev struct {
	n           uint64 //size of VIP backeds
	m           uint64 //sie of the lookup table
	permutation [][]uint64
	lookup      []int64
	nodeList    []string
}

//NewMaglev :
func NewMaglev(backends []string, m uint64) *Maglev {
	mag := &Maglev{n: uint64(len(backends)), m: m}
	mag.generatePopulation(backends)
	mag.lookup = mag.populate()
	mag.nodeList = backends
	return mag
}

//Get :Get node name
func (m *Maglev) Get(obj string) string {
	key := m.hashKey(obj)
	return m.nodeList[m.lookup[key%m.m]]
}

func (m *Maglev) hashKey(obj string) uint64 {
	return siphash.Hash(0xdeadbabe, 0, []byte(obj))
}

func (m *Maglev) generatePopulation(backeds []string) {
	for i := 0; i < len(backeds); i++ {
		bData := []byte(backeds[i])

		offset := siphash.Hash(0xdeadbabe, 0, bData) % m.m
		skip := (siphash.Hash(0xdeadbeef, 0, bData) % (m.m - 1)) + 1

		iRow := make([]uint64, m.m)
		var j uint64
		for j = 0; j < m.m; j++ {
			iRow[j] = (offset + uint64(j)*skip) % m.m
		}

		m.permutation = append(m.permutation, iRow)
	}
}

//Populate :
func (m *Maglev) populate() []int64 {
	var i, j uint64
	next := make([]uint64, m.n)
	entry := make([]int64, m.m)
	for j = 0; j < m.m; j++ {
		entry[j] = -1
	}

	var n uint64

	for { //true
		for i = 0; i < m.n; i++ {
			c := m.permutation[i][next[i]]
			for entry[c] >= 0 {
				next[i] = next[i] + 1
				c = m.permutation[i][next[i]]
			}

			entry[c] = int64(i)
			next[i] = next[i] + 1
			n++

			if n == m.m {
				return entry
			}
		}

	}

}
