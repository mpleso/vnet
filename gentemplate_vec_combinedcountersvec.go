// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=vnet -id combinedCountersVec -d VecType=CombinedCountersVec -d Type=CombinedCounters github.com/platinasystems/elib/vec.tmpl]

package vnet

import (
	"github.com/platinasystems/elib"
)

type CombinedCountersVec []CombinedCounters

func (p *CombinedCountersVec) Resize(n uint) {
	c := elib.Index(cap(*p))
	l := elib.Index(len(*p)) + elib.Index(n)
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]CombinedCounters, l, c)
		copy(q, *p)
		*p = q
	}
	*p = (*p)[:l]
}

func (p *CombinedCountersVec) validate(i uint, zero *CombinedCounters) *CombinedCounters {
	c := elib.Index(cap(*p))
	l := elib.Index(i) + 1
	if l > c {
		cNext := elib.NextResizeCap(l)
		q := make([]CombinedCounters, cNext, cNext)
		copy(q, *p)
		if zero != nil {
			for i := c; i < cNext; i++ {
				q[i] = *zero
			}
		}
		*p = q[:l]
	}
	if l > elib.Index(len(*p)) {
		*p = (*p)[:l]
	}
	return &(*p)[i]
}
func (p *CombinedCountersVec) Validate(i uint) *CombinedCounters {
	return p.validate(i, (*CombinedCounters)(nil))
}
func (p *CombinedCountersVec) ValidateInit(i uint, zero CombinedCounters) *CombinedCounters {
	return p.validate(i, &zero)
}

func (p CombinedCountersVec) Len() uint { return uint(len(p)) }
