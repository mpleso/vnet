// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=vnet -id miniCombinedCounter -d VecType=miniCombinedCounterVec -d Type=miniCombinedCounter github.com/platinasystems/elib/vec.tmpl]

package vnet

import (
	"github.com/platinasystems/elib"
)

type miniCombinedCounterVec []miniCombinedCounter

func (p *miniCombinedCounterVec) Resize(n uint) {
	c := elib.Index(cap(*p))
	l := elib.Index(len(*p)) + elib.Index(n)
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]miniCombinedCounter, l, c)
		copy(q, *p)
		*p = q
	}
	*p = (*p)[:l]
}

func (p *miniCombinedCounterVec) Validate(i uint) {
	c := elib.Index(cap(*p))
	l := elib.Index(i) + 1
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]miniCombinedCounter, l, c)
		copy(q, *p)
		*p = q
	}
	if l > elib.Index(len(*p)) {
		*p = (*p)[:l]
	}
}
