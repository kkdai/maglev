package maglev

import (
	"errors"

	"github.com/dchest/siphash"
)

const (
	bigM uint64 = 65537
)

//Maglev :
type Maglev struct {
	n           uint64 //size of VIP backends
	m           uint64 //sie of the lookup table
	permutation [][]uint64
	lookup      []int64
	nodeList    []string
}

//NewMaglev :
func NewMaglev(backends []string, m uint64) *Maglev {
	mag := &Maglev{n: uint64(len(backends)), m: m}
	mag.nodeList = backends
	mag.generatePopulation()
	mag.populate()
	return mag
}

//Add : Return nil if add success, otherwise return error
func (m *Maglev) Add(backend string) error {
	for _, v := range m.nodeList {
		if v == backend {
			return errors.New("Exist already")
		}
	}

	m.nodeList = append(m.nodeList, backend)
	m.n = uint64(len(m.nodeList))
	m.generatePopulation()
	m.populate()
	return nil
}

//Remove :
func (m *Maglev) Remove(backend string) error {
	notFound := true
	for _, v := range m.nodeList {
		if v == backend {
			notFound = false
		}
	}
	if notFound {
		return errors.New("Not found")
	}

	for i, v := range m.nodeList {
		if v == backend {
			m.nodeList = append(m.nodeList[:i], m.nodeList[i+1:]...)
			break
		}
	}

	m.n = uint64(len(m.nodeList))
	m.generatePopulation()
	m.populate()
	return nil
}

//Get :Get node name by object string.
func (m *Maglev) Get(obj string) (string, error) {
	if len(m.nodeList) == 0 {
		return "", errors.New("Empty")
	}
	key := m.hashKey(obj)
	return m.nodeList[m.lookup[key%m.m]], nil
}

func (m *Maglev) hashKey(obj string) uint64 {
	return siphash.Hash(0xdeadbabe, 0, []byte(obj))
}

func (m *Maglev) generatePopulation() {
	if len(m.nodeList) == 0 {
		return
	}

	for i := 0; i < len(m.nodeList); i++ {
		bData := []byte(m.nodeList[i])

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

func (m *Maglev) populate() {
	if len(m.nodeList) == 0 {
		return
	}

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
				m.lookup = entry
				return
			}
		}

	}

}
