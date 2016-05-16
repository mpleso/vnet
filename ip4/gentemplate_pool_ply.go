// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=ip4 -id ply -d PoolType=plyPool -d Type=ply -d Data=plys github.com/platinasystems/elib/pool.tmpl]

package ip4

import (
	"github.com/platinasystems/elib"
)

type plyPool struct {
	elib.Pool
	plys []ply
}

func (p *plyPool) GetIndex() (i uint) {
	l := uint(len(p.plys))
	i = p.Pool.GetIndex(l)
	if i >= l {
		p.Validate(i)
	}
	return i
}

func (p *plyPool) PutIndex(i uint) (ok bool) {
	return p.Pool.PutIndex(i)
}

func (p *plyPool) IsFree(i uint) (ok bool) {
	return p.Pool.IsFree(i)
}

func (p *plyPool) Resize(n uint) {
	c := elib.Index(cap(p.plys))
	l := elib.Index(len(p.plys) + int(n))
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]ply, l, c)
		copy(q, p.plys)
		p.plys = q
	}
	p.plys = p.plys[:l]
}

func (p *plyPool) Validate(i uint) {
	c := elib.Index(cap(p.plys))
	l := elib.Index(i) + 1
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]ply, l, c)
		copy(q, p.plys)
		p.plys = q
	}
	if l > elib.Index(len(p.plys)) {
		p.plys = p.plys[:l]
	}
}

func (p *plyPool) Elts() uint {
	return uint(len(p.plys)) - p.FreeLen()
}

func (p *plyPool) Len() uint {
	return uint(len(p.plys))
}
