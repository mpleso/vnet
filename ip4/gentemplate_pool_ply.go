// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=ip4 -id ply -d PoolType=plyPool -d Type=ply -d Data=plys github.com/platinasystems/elib/pool.tmpl]

// Copyright 2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ip4

import (
	"github.com/platinasystems/elib"
)

type plyPool struct {
	elib.Pool
	plys []ply
}

func (p *plyPool) GetIndex() (i uint) {
	l := uint(len(p.plys))
	i = p.Pool.GetIndex(l)
	if i >= l {
		p.Validate(i)
	}
	return i
}

func (p *plyPool) PutIndex(i uint) (ok bool) {
	return p.Pool.PutIndex(i)
}

func (p *plyPool) IsFree(i uint) (v bool) {
	v = i >= uint(len(p.plys))
	if !v {
		v = p.Pool.IsFree(i)
	}
	return
}

func (p *plyPool) Resize(n uint) {
	c := elib.Index(cap(p.plys))
	l := elib.Index(len(p.plys) + int(n))
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]ply, l, c)
		copy(q, p.plys)
		p.plys = q
	}
	p.plys = p.plys[:l]
}

func (p *plyPool) Validate(i uint) {
	c := elib.Index(cap(p.plys))
	l := elib.Index(i) + 1
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]ply, l, c)
		copy(q, p.plys)
		p.plys = q
	}
	if l > elib.Index(len(p.plys)) {
		p.plys = p.plys[:l]
	}
}

func (p *plyPool) Elts() uint {
	return uint(len(p.plys)) - p.FreeLen()
}

func (p *plyPool) Len() uint {
	return uint(len(p.plys))
}

func (p *plyPool) Foreach(f func(x ply)) {
	for i := range p.plys {
		if !p.Pool.IsFree(uint(i)) {
			f(p.plys[i])
		}
	}
}

func (p *plyPool) ForeachIndex(f func(i uint)) {
	for i := range p.plys {
		if !p.Pool.IsFree(uint(i)) {
			f(uint(i))
		}
	}
}
