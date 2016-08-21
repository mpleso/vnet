package vnet

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/cli"
	"github.com/platinasystems/elib/hw"
	"github.com/platinasystems/elib/loop"

	"fmt"
	"sort"
	"unsafe"
)

type Ref struct {
	hw.RefHeader

	Err ErrorRef

	Si Si
}

func (r *Ref) Flags() BufferFlag         { return BufferFlag(r.RefHeader.Flags()) }
func (r *Ref) NextValidFlag() BufferFlag { return BufferFlag(r.RefHeader.NextValidFlag()) }

type BufferFlag hw.BufferFlag

const (
	NextValid = BufferFlag(hw.NextValid)
	Cloned    = BufferFlag(hw.Cloned)
)

func RefFlag1(f BufferFlag, r []Ref, i uint) bool {
	return hw.RefFlag1(hw.BufferFlag(f), &r[i+0].RefHeader)
}
func RefFlag2(f BufferFlag, r []Ref, i uint) bool {
	return hw.RefFlag2(hw.BufferFlag(f), &r[i+0].RefHeader, &r[i+1].RefHeader)
}
func RefFlag4(f BufferFlag, r []Ref, i uint) bool {
	return hw.RefFlag4(hw.BufferFlag(f), &r[i+0].RefHeader, &r[i+1].RefHeader, &r[i+2].RefHeader, &r[i+3].RefHeader)
}

type RefChain hw.RefChain

func (c *RefChain) Len() uint  { return (*hw.RefChain)(c).Len() }
func (c *RefChain) Reset()     { *c = RefChain{} }
func (c *RefChain) Head() *Ref { return (*Ref)(unsafe.Pointer((*hw.RefChain)(c).Head())) }
func (c *RefChain) Validate()  { (*hw.RefChain)(c).Validate() }

func (c *RefChain) Append(r *Ref) {
	if c.Len() == 0 {
		h := c.Head()
		*h = *r
	}
	(*hw.RefChain)(c).Append(&r.RefHeader)
	c.Validate()
}
func (c *RefChain) Done() (h Ref) {
	h = *c.Head()
	c.Validate()
	c.Reset()
	return
}

//go:generate gentemplate -d Package=vnet -id Ref -d VecType=RefVec -d Type=Ref github.com/platinasystems/elib/vec.tmpl

type refInCommon struct {
	loop.In
	BufferPool *hw.BufferPool
}

type RefIn struct {
	refInCommon
	Refs [MaxVectorLen]Ref
}

func (r *RefIn) Cap() uint { return uint(len(r.Refs)) }

type RefVecIn struct {
	refInCommon

	// Number of packets corresponding to vector of buffer refs.
	nPackets uint

	Refs RefVec
}

type RefOut struct {
	loop.Out
	Outs []RefIn
}

func (r *RefIn) AllocPoolRefs(p *hw.BufferPool, n uint) {
	r.BufferPool = p
	p.AllocRefs(&r.Refs[0].RefHeader, n)
}
func (r *RefIn) FreePoolRefs(p *hw.BufferPool, n uint) {
	p.FreeRefs(&r.Refs[0].RefHeader, n)
}
func (r *RefIn) AllocRefs(n uint)       { r.AllocPoolRefs(r.BufferPool, n) }
func (r *RefIn) FreeRefs(n uint)        { r.FreePoolRefs(r.BufferPool, n) }
func (i *RefIn) SetLen(v *Vnet, l uint) { i.In.SetLen(&v.loop, l) }
func (i *RefIn) AddLen(v *Vnet) (l uint) {
	l = i.GetLen(&v.loop)
	i.SetLen(v, l+1)
	return
}

func (r *RefVecIn) FreePoolRefs(p *hw.BufferPool) { p.FreeRefs(&r.Refs[0].RefHeader, r.Refs.Len()) }
func (r *RefVecIn) NPackets() uint                { return r.nPackets }
func (r *RefVecIn) FreeRefs()                     { r.FreePoolRefs(r.BufferPool) }

type showPool struct {
	Pool string `format:"%-30s" align:"left"`
	Size string `format:"%-12s" align:"right"`
	Free string `format:"%-12s" align:"right"`
	Used string `format:"%-12s" align:"right"`
}
type showPools []showPool

func (x showPools) Less(i, j int) bool { return x[i].Pool < x[j].Pool }
func (x showPools) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x showPools) Len() int           { return len(x) }

func (v *Vnet) showBufferUsage(c cli.Commander, w cli.Writer, in *cli.Input) (err error) {
	m := &v.BufferMain

	sps := []showPool{}
	fmt.Fprintf(w, "DMA heap: %s\n", hw.DmaHeapUsage())
	for _, p := range m.PoolByName {
		sps = append(sps, showPool{
			Pool: p.Name,
			Size: fmt.Sprintf("%12d", p.Size),
			Free: fmt.Sprintf("%12s", elib.MemorySize(p.SizeIncludingOverhead()*p.FreeLen())),
			Used: fmt.Sprintf("%12s", elib.MemorySize(p.DmaMemAllocBytes)),
		})
	}
	sort.Sort(showPools(sps))
	elib.Tabulate(sps).Write(w)
	return
}
