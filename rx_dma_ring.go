package vnet

import (
	"github.com/platinasystems/elib/hw"
)

type RxDmaRing struct {
	pool     *hw.BufferPool
	ring_len uint
	sequence uint
	refs     RefVec
	rxDmaRingState
}

func (r *RxDmaRing) Init(pool *hw.BufferPool, ring_len uint) {
	r.pool = pool
	r.ring_len = ring_len
	r.refs.Validate(2*ring_len - 1)
	r.pool.AllocRefs(&r.refs[0].RefHeader, r.refs.Len())
}

// Allocate new re-fill buffers when ring wraps.
func (r *RxDmaRing) WrapRefill() {
	ri0 := r.sequence & 1
	r.sequence++
	r.pool.AllocRefsStride(&r.refs[ri0].RefHeader, r.ring_len, 2)
}

type rxDmaRingIndex uint

func (r *RxDmaRing) Index(i uint) rxDmaRingIndex { return rxDmaRingIndex(2*i + (r.sequence & 1)) }
func (i rxDmaRingIndex) Next() rxDmaRingIndex    { return i + 2 }

// Even buffer is for packet receive; odd buffer is to refill ring.
func (r *RxDmaRing) Get(i rxDmaRingIndex) (rxRef, refillRef *Ref) {
	rxRef, refillRef = &r.refs[i^0], &r.refs[i^1]
	return
}

type rxDmaRingState struct {
	v                *Vnet
	out              *RefOut
	chain            RefChain
	last_miss_next   uint
	n_last_miss_next uint
}

// Shared code for rx slow path.
func (r *RxDmaRing) RxSlowPath(r0 *Ref, isEop0 bool, next0 uint, nextʹ uint, n_nextʹ uint) (next uint, n_next uint) {
	next, n_next = nextʹ, n_nextʹ

	s := &r.rxDmaRingState

	// Append buffer to current chain.
	s.chain.Append(r0)

	// If at end of packet, enqueue packet to next graph node.
	if !isEop0 {
		return
	}

	// Enqueue packet.
	ref := s.chain.Done()
	in := &s.out.Outs[next0]

	// Cache empty?
	if n_next == 0 {
		next = next0
	}

	// Cache hit?
	if next0 == next {
		in.Refs[n_next] = ref
		n_next++
		return
	}

	n_next0 := in.Len()
	in.SetPoolAndLen(s.v, r.pool, n_next0+1)
	in.Refs[n_next0] = ref
	n_next0++

	// Switch cached next after enough repeats of cache miss with same next.
	if next0 == s.last_miss_next {
		s.n_last_miss_next++
		if s.n_last_miss_next >= 4 {
			if n_next > 0 {
				s.out.Outs[next].SetPoolAndLen(s.v, r.pool, n_next)
			}
			next = next0
			n_next = n_next0
		}
	} else {
		s.last_miss_next = next0
		s.n_last_miss_next = 1
	}
	return
}
