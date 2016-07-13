// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=vnet -id interfaceNodeThreadVec -d VecType=interfaceNodeThreadVec -d Type=*interfaceNodeThread github.com/platinasystems/elib/vec.tmpl]

package vnet

import (
	"github.com/platinasystems/elib"
)

type interfaceNodeThreadVec []*interfaceNodeThread

func (p *interfaceNodeThreadVec) Resize(n uint) {
	c := elib.Index(cap(*p))
	l := elib.Index(len(*p)) + elib.Index(n)
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]*interfaceNodeThread, l, c)
		copy(q, *p)
		*p = q
	}
	*p = (*p)[:l]
}

func (p *interfaceNodeThreadVec) validate(i uint, zero **interfaceNodeThread) **interfaceNodeThread {
	c := elib.Index(cap(*p))
	l := elib.Index(i) + 1
	if l > c {
		cNext := elib.NextResizeCap(l)
		q := make([]*interfaceNodeThread, cNext, cNext)
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
func (p *interfaceNodeThreadVec) Validate(i uint) **interfaceNodeThread {
	return p.validate(i, (**interfaceNodeThread)(nil))
}
func (p *interfaceNodeThreadVec) ValidateInit(i uint, zero *interfaceNodeThread) **interfaceNodeThread {
	return p.validate(i, &zero)
}

func (p interfaceNodeThreadVec) Len() uint { return uint(len(p)) }