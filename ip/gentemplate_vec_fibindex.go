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

func (p *FibIndexVec) validate(new_len uint, zero *FibIndex) *FibIndex {
	c := elib.Index(cap(*p))
	lʹ := elib.Index(len(*p))
	l := elib.Index(new_len)
	if l <= c {
		// Need to reslice to larger length?
		if l >= lʹ {
			*p = (*p)[:l]
		}
		return &(*p)[l-1]
	}
	return p.validateSlowPath(zero, c, l, lʹ)
}

func (p *FibIndexVec) validateSlowPath(zero *FibIndex,
	c, l, lʹ elib.Index) *FibIndex {
	if l > c {
		cNext := elib.NextResizeCap(l)
		q := make([]FibIndex, cNext, cNext)
		copy(q, *p)
		if zero != nil {
			for i := c; i < cNext; i++ {
				q[i] = *zero
			}
		}
		*p = q[:l]
	}
	if l > lʹ {
		*p = (*p)[:l]
	}
	return &(*p)[l-1]
}

func (p *FibIndexVec) Validate(i uint) *FibIndex {
	return p.validate(i+1, (*FibIndex)(nil))
}

func (p *FibIndexVec) ValidateInit(i uint, zero FibIndex) *FibIndex {
	return p.validate(i+1, &zero)
}

func (p *FibIndexVec) ValidateLen(l uint) (v *FibIndex) {
	if l > 0 {
		v = p.validate(l, (*FibIndex)(nil))
	}
	return
}

func (p *FibIndexVec) ValidateLenInit(l uint, zero FibIndex) (v *FibIndex) {
	if l > 0 {
		v = p.validate(l, &zero)
	}
	return
}

func (p FibIndexVec) Len() uint { return uint(len(p)) }
