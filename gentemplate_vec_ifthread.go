// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=vnet -id ifThread -d VecType=ifThreadVec -d Type=*InterfaceThread github.com/platinasystems/elib/vec.tmpl]

// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vnet

import (
	"github.com/platinasystems/elib"
)

type ifThreadVec []*InterfaceThread

func (p *ifThreadVec) Resize(n uint) {
	c := elib.Index(cap(*p))
	l := elib.Index(len(*p)) + elib.Index(n)
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]*InterfaceThread, l, c)
		copy(q, *p)
		*p = q
	}
	*p = (*p)[:l]
}

func (p *ifThreadVec) validate(new_len uint, zero **InterfaceThread) **InterfaceThread {
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

func (p *ifThreadVec) validateSlowPath(zero **InterfaceThread,
	c, l, lʹ elib.Index) **InterfaceThread {
	if l > c {
		cNext := elib.NextResizeCap(l)
		q := make([]*InterfaceThread, cNext, cNext)
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

func (p *ifThreadVec) Validate(i uint) **InterfaceThread {
	return p.validate(i+1, (**InterfaceThread)(nil))
}

func (p *ifThreadVec) ValidateInit(i uint, zero *InterfaceThread) **InterfaceThread {
	return p.validate(i+1, &zero)
}

func (p *ifThreadVec) ValidateLen(l uint) (v **InterfaceThread) {
	if l > 0 {
		v = p.validate(l, (**InterfaceThread)(nil))
	}
	return
}

func (p *ifThreadVec) ValidateLenInit(l uint, zero *InterfaceThread) (v **InterfaceThread) {
	if l > 0 {
		v = p.validate(l, &zero)
	}
	return
}

func (p ifThreadVec) Len() uint { return uint(len(p)) }
