// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=vnet -id CombinedCounter -d VecType=CombinedCounterVec -d Type=CombinedCounter github.com/platinasystems/elib/vec.tmpl]

package vnet

import (
	"github.com/platinasystems/elib"
)

type CombinedCounterVec []CombinedCounter

func (p *CombinedCounterVec) Resize(n uint) {
	c := elib.Index(cap(*p))
	l := elib.Index(len(*p)) + elib.Index(n)
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]CombinedCounter, l, c)
		copy(q, *p)
		*p = q
	}
	*p = (*p)[:l]
}

func (p *CombinedCounterVec) Validate(i uint) {
	c := elib.Index(cap(*p))
	l := elib.Index(i) + 1
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]CombinedCounter, l, c)
		copy(q, *p)
		*p = q
	}
	if l > elib.Index(len(*p)) {
		*p = (*p)[:l]
	}
}

func (p CombinedCounterVec) Len() uint { return uint(len(p)) }