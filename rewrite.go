package vnet

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/loop"

	"unsafe"
)

type Ref struct{ loop.Ref }

type Rewrite struct {
	// Software interface to mark re-written packets with.
	Si Si

	// Node where packet will be rewritten.
	NodeIndex uint32

	// Next node to feed packet after rewrite.
	NextIndex uint32

	// Max packet size layer 3 (MTU) for output interface.
	// Used for MTU check after packet rewrite.
	// Avoids having to lookup egress interface's MTU.
	MaxL3PacketSize uint16

	// Number of bytes in rewrite data.
	dataLen uint16

	data [loop.RewriteBytes]byte
}

func (r *Rewrite) SetData(d []byte) { r.dataLen = uint16(copy(r.data[:], d)) }
func (r *Rewrite) AddData(p unsafe.Pointer, size uintptr) (l uintptr) {
	l = uintptr(r.dataLen)
	r.dataLen += uint16(size)
	for i := uintptr(0); i < size; i++ {
		r.data[l+i] = *(*uint8)(elib.PointerAdd(p, i))
	}
	return l + size
}
func (r *Rewrite) getData() []byte { return r.data[:r.dataLen] }

func (v *Vnet) SetRewrite(rw *Rewrite, si Si, noder Noder, t PacketType, dstAddr []byte) {
	sw := v.SwIf(si)
	hw := v.SupHwIf(sw)
	h := v.HwIfer(hw.hi)
	n := noder.GetNode()
	rw.Si = si
	rw.NodeIndex = uint32(n.Index())
	rw.NextIndex = uint32(loop.AddNext(noder, h))
	rw.MaxL3PacketSize = uint16(hw.maxPacketSize)
	h.SetRewrite(v, rw, t, dstAddr)
}
func SetRewrite(rw *Rewrite, si Si, noder Noder, t PacketType, dstAddr []byte) {
	defaultVnet.SetRewrite(rw, si, noder, t, dstAddr)
}

func PerformRewrite(r0 *Ref, rw0 *Rewrite) {
	r0.Advance(-int(rw0.dataLen))
	copy(r0.DataSlice(), rw0.getData())
}

func Perform2Rewrites(r0, r1 *Ref, rw0, rw1 *Rewrite) {
	r0.Advance(-int(rw0.dataLen))
	r1.Advance(-int(rw1.dataLen))
	copy(r0.DataSlice(), rw0.getData())
	copy(r1.DataSlice(), rw1.getData())
}
