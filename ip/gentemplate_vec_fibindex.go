// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=ip -id FibIndex -d VecType=FibIndexVec -d Type=FibIndex github.com/platinasystems/elib/vec.tmpl]

package ip

import (
	"github.com/platinasystems/elib"
)

type FibIndexVec []FibIndex

func (p *FibIndexVec) Resize(n uint) {
	c := elib.Index(cap(*p))
	l := elib.Index(len(*p)) + elib.Index(n)
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]FibIndex, l, c)
		copy(q, *p)
		*p = q
	}
	*p = (*p)[:l]
}

func (p *FibIndexVec) Validate(i uint) *FibIndex {
	c := elib.Index(cap(*p))
	l := elib.Index(i) + 1
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]FibIndex, l, c)
		copy(q, *p)
		*p = q
	}
	if l > elib.Index(len(*p)) {
		*p = (*p)[:l]
	}
	return &(*p)[i]
}

func (p FibIndexVec) Len() uint { return uint(len(p)) }
