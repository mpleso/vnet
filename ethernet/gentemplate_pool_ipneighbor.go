// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=ethernet -id ipNeighbor -d PoolType=ipNeighborPool -d Data=neighbors -d Type=ipNeighbor github.com/platinasystems/elib/pool.tmpl]

package ethernet

import (
	"github.com/platinasystems/elib"
)

type ipNeighborPool struct {
	elib.Pool
	neighbors []ipNeighbor
}

func (p *ipNeighborPool) GetIndex() (i uint) {
	l := uint(len(p.neighbors))
	i = p.Pool.GetIndex(l)
	if i >= l {
		p.Validate(i)
	}
	return i
}

func (p *ipNeighborPool) PutIndex(i uint) (ok bool) {
	return p.Pool.PutIndex(i)
}

func (p *ipNeighborPool) IsFree(i uint) (v bool) {
	v = i >= uint(len(p.neighbors))
	if !v {
		v = p.Pool.IsFree(i)
	}
	return
}

func (p *ipNeighborPool) Resize(n uint) {
	c := elib.Index(cap(p.neighbors))
	l := elib.Index(len(p.neighbors) + int(n))
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]ipNeighbor, l, c)
		copy(q, p.neighbors)
		p.neighbors = q
	}
	p.neighbors = p.neighbors[:l]
}

func (p *ipNeighborPool) Validate(i uint) {
	c := elib.Index(cap(p.neighbors))
	l := elib.Index(i) + 1
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]ipNeighbor, l, c)
		copy(q, p.neighbors)
		p.neighbors = q
	}
	if l > elib.Index(len(p.neighbors)) {
		p.neighbors = p.neighbors[:l]
	}
}

func (p *ipNeighborPool) Elts() uint {
	return uint(len(p.neighbors)) - p.FreeLen()
}

func (p *ipNeighborPool) Len() uint {
	return uint(len(p.neighbors))
}

func (p *ipNeighborPool) Foreach(f func(x ipNeighbor)) {
	for i := range p.neighbors {
		if !p.Pool.IsFree(uint(i)) {
			f(p.neighbors[i])
		}
	}
}
