package vnet

import (
	"github.com/platinasystems/elib/loop"
)

type Ref struct{ loop.Ref }

type Rewrite struct {
	// Software interface to mark re-written packets with.
	SwIf SwIfIndex

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
func (r *Rewrite) getData() []byte  { return r.data[:r.dataLen] }

func PerformRewrite(r0 *Ref, rw0 *Rewrite) {
	r0.Advance(-int(rw0.dataLen))
	copy(r0.DataSlice(), rw0.getData())
}

func PerformRewrites(r0, r1 *Ref, rw0, rw1 *Rewrite) {
	r0.Advance(-int(rw0.dataLen))
	r1.Advance(-int(rw1.dataLen))
	copy(r0.DataSlice(), rw0.getData())
	copy(r1.DataSlice(), rw1.getData())
}
