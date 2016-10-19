// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=ip -id AdjRemapVec -d VecType=AdjRemapVec -d Type=AdjRemap github.com/platinasystems/elib/vec.tmpl]

// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ip

import (
	"github.com/platinasystems/elib"
)

type AdjRemapVec []AdjRemap

func (p *AdjRemapVec) Resize(n uint) {
	c := elib.Index(cap(*p))
	l := elib.Index(len(*p)) + elib.Index(n)
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]AdjRemap, l, c)
		copy(q, *p)
		*p = q
	}
	*p = (*p)[:l]
}

func (p *AdjRemapVec) validate(new_len uint, zero *AdjRemap) *AdjRemap {
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

func (p *AdjRemapVec) validateSlowPath(zero *AdjRemap,
	c, l, lʹ elib.Index) *AdjRemap {
	if l > c {
		cNext := elib.NextResizeCap(l)
		q := make([]AdjRemap, cNext, cNext)
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

func (p *AdjRemapVec) Validate(i uint) *AdjRemap {
	return p.validate(i+1, (*AdjRemap)(nil))
}

func (p *AdjRemapVec) ValidateInit(i uint, zero AdjRemap) *AdjRemap {
	return p.validate(i+1, &zero)
}

func (p *AdjRemapVec) ValidateLen(l uint) (v *AdjRemap) {
	if l > 0 {
		v = p.validate(l, (*AdjRemap)(nil))
	}
	return
}

func (p *AdjRemapVec) ValidateLenInit(l uint, zero AdjRemap) (v *AdjRemap) {
	if l > 0 {
		v = p.validate(l, &zero)
	}
	return
}

func (p AdjRemapVec) Len() uint { return uint(len(p)) }
