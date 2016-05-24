// autogenerated: do not edit!
// generated from gentemplate [gentemplate -d Package=ip -id nextHopHeap -d HeapType=nextHopHeap -d Data=elts -d Type=nextHop github.com/platinasystems/elib/heap.tmpl]

package ip

import (
	"github.com/platinasystems/elib"
)

type nextHopHeap struct {
	elib.Heap
	elts []nextHop
	ids  []elib.Index
}

func (p *nextHopHeap) GetAligned(size, log2Alignment uint) (offset uint) {
	l := uint(len(p.elts))
	id, offset := p.Heap.GetAligned(size, log2Alignment)
	if offset >= l {
		p.Validate(offset + size - 1)
	}
	for i := uint(0); i < size; i++ {
		p.ids[offset+i] = id
	}
	return
}

func (p *nextHopHeap) Get(size uint) uint { return p.GetAligned(size, 0) }

func (p *nextHopHeap) Put(offset uint) {
	p.Heap.Put(p.Id(offset))
}

func (p *nextHopHeap) Validate(i uint) {
	c := elib.Index(cap(p.elts))
	l := elib.Index(i) + 1
	if l > c {
		c = elib.NextResizeCap(l)
		q := make([]nextHop, l, c)
		r := make([]elib.Index, l, c)
		copy(q, p.elts)
		copy(r, p.ids)
		p.elts = q
		p.ids = r
	}
	if l > elib.Index(len(p.elts)) {
		p.elts = p.elts[:l]
		p.ids = p.ids[:l]
	}
}

func (p *nextHopHeap) Id(offset uint) elib.Index {
	return p.ids[offset]
}

func (p *nextHopHeap) Slice(offset uint) []nextHop {
	l := p.Len(p.Id(offset))
	return p.elts[offset : offset+l]
}
