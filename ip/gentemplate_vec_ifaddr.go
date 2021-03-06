// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=ip -id IfAddr -d VecType=IfAddrVec -d Type=IfAddr github.com/platinasystems/elib/vec.tmpl]

// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ip

import (
	"github.com/platinasystems/elib"
)

type IfAddrVec []IfAddr

func (p *IfAddrVec) Resize(n uint) {
	c := elib.Index(cap(*p))
	l := elib.Index(len(*p)) + elib.Index(n)
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]IfAddr, l, c)
		copy(q, *p)
		*p = q
	}
	*p = (*p)[:l]
}

func (p *IfAddrVec) validate(new_len uint, zero *IfAddr) *IfAddr {
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

func (p *IfAddrVec) validateSlowPath(zero *IfAddr,
	c, l, lʹ elib.Index) *IfAddr {
	if l > c {
		cNext := elib.NextResizeCap(l)
		q := make([]IfAddr, cNext, cNext)
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

func (p *IfAddrVec) Validate(i uint) *IfAddr {
	return p.validate(i+1, (*IfAddr)(nil))
}

func (p *IfAddrVec) ValidateInit(i uint, zero IfAddr) *IfAddr {
	return p.validate(i+1, &zero)
}

func (p *IfAddrVec) ValidateLen(l uint) (v *IfAddr) {
	if l > 0 {
		v = p.validate(l, (*IfAddr)(nil))
	}
	return
}

func (p *IfAddrVec) ValidateLenInit(l uint, zero IfAddr) (v *IfAddr) {
	if l > 0 {
		v = p.validate(l, &zero)
	}
	return
}

func (p IfAddrVec) Len() uint { return uint(len(p)) }
