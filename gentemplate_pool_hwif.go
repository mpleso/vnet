// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=vnet -id HwIf -d PoolType=hwIferPool -d Type=HwInterfacer -d Data=elts github.com/platinasystems/elib/pool.tmpl]

// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vnet

import (
	"github.com/platinasystems/elib"
)

type hwIferPool struct {
	elib.Pool
	elts []HwInterfacer
}

func (p *hwIferPool) GetIndex() (i uint) {
	l := uint(len(p.elts))
	i = p.Pool.GetIndex(l)
	if i >= l {
		p.Validate(i)
	}
	return i
}

func (p *hwIferPool) PutIndex(i uint) (ok bool) {
	return p.Pool.PutIndex(i)
}

func (p *hwIferPool) IsFree(i uint) (v bool) {
	v = i >= uint(len(p.elts))
	if !v {
		v = p.Pool.IsFree(i)
	}
	return
}

func (p *hwIferPool) Resize(n uint) {
	c := elib.Index(cap(p.elts))
	l := elib.Index(len(p.elts) + int(n))
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]HwInterfacer, l, c)
		copy(q, p.elts)
		p.elts = q
	}
	p.elts = p.elts[:l]
}

func (p *hwIferPool) Validate(i uint) {
	c := elib.Index(cap(p.elts))
	l := elib.Index(i) + 1
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]HwInterfacer, l, c)
		copy(q, p.elts)
		p.elts = q
	}
	if l > elib.Index(len(p.elts)) {
		p.elts = p.elts[:l]
	}
}

func (p *hwIferPool) Elts() uint {
	return uint(len(p.elts)) - p.FreeLen()
}

func (p *hwIferPool) Len() uint {
	return uint(len(p.elts))
}

func (p *hwIferPool) Foreach(f func(x HwInterfacer)) {
	for i := range p.elts {
		if !p.Pool.IsFree(uint(i)) {
			f(p.elts[i])
		}
	}
}

func (p *hwIferPool) ForeachIndex(f func(i uint)) {
	for i := range p.elts {
		if !p.Pool.IsFree(uint(i)) {
			f(uint(i))
		}
	}
}
