package vnet

import (
	"github.com/platinasystems/elib/loop"
)

type interfaceInputer interface {
	InterfaceInput(out *RefOut)
}

type outputInterfaceNoder interface {
	Noder
	GetInterfaceNode() *interfaceNode
	InterfaceOutput(in *TxRefVecIn)
}

type inputOutputInterfaceNoder interface {
	outputInterfaceNoder
	interfaceInputer
}

type interfaceNode struct {
	Node

	threads interfaceNodeThreadVec

	hi Hi

	outChan chan *TxRefVecIn
	tx      outputInterfaceNoder
	rx      interfaceInputer
}

type OutputInterfaceNode struct{ interfaceNode }
type InterfaceNode struct{ interfaceNode }

func (n *interfaceNode) SetHi(hi Hi)                              { n.hi = hi }
func (n *interfaceNode) MakeLoopIn() loop.LooperIn                { return &RefIn{} }
func (n *interfaceNode) MakeLoopOut() loop.LooperOut              { return &RefOut{} }
func (n *interfaceNode) LoopOutput(l *loop.Loop, i loop.LooperIn) { n.ifOutput(i.(*RefIn)) }
func (n *interfaceNode) GetInterfaceNode() *interfaceNode         { return n }

func (n *InterfaceNode) LoopInput(l *loop.Loop, o loop.LooperOut) {
	n.rx.InterfaceInput(o.(*RefOut))
}

func (v *Vnet) RegisterInterfaceNode(n inputOutputInterfaceNoder, hi Hi, name string, args ...interface{}) {
	x := n.GetInterfaceNode()
	x.hi = hi
	x.rx = n
	x.setupTx(n)
	v.RegisterNode(n, name, args...)
}

func (v *Vnet) RegisterOutputInterfaceNode(n outputInterfaceNoder, hi Hi, name string, args ...interface{}) {
	x := n.GetInterfaceNode()
	x.hi = hi
	x.setupTx(n)
	v.RegisterNode(n, name, args...)
}

func (n *interfaceNode) setupTx(tx outputInterfaceNoder) {
	n.tx = tx
	n.outChan = make(chan *TxRefVecIn, 64)
	go n.ifOutputThread()
}

func (n *interfaceNode) slowPath(rvʹ RefVec, rs []Ref, is, ivʹ, nBytesʹ uint) (rv RefVec, iv, nBytes uint) {
	rv, iv, nBytes = rvʹ, ivʹ, nBytesʹ
	s := rs[is]
	for {
		// Copy buffer reference.
		rv.Validate(iv)
		rv[iv] = s
		iv++

		if h := s.NextRef(); h == nil {
			break
		} else {
			s.RefHeader = *h
		}
		nBytes += s.DataLen()
	}
	return
}

type interfaceNodeThread struct {
	freeChan chan *TxRefVecIn
	n_alloc  uint
}

//go:generate gentemplate -d Package=vnet -id interfaceNodeThreadVec -d VecType=interfaceNodeThreadVec -d Type=*interfaceNodeThread github.com/platinasystems/elib/vec.tmpl

func (t *interfaceNodeThread) allocTxRefVecIn(n *interfaceNode, in *RefIn) (i *TxRefVecIn) {
	l := n.Vnet.loop
	for {
		select {
		case i = <-t.freeChan:
			i.FreeRefs(false)
			return
		default:
		}
		if t.n_alloc < 2 {
			t.n_alloc++
			i = &TxRefVecIn{t: t}
			return
		}
		l.Suspend(&in.In)
	}
	return
}

type TxRefVecIn struct {
	RefVecIn
	t *interfaceNodeThread
}

func (v *Vnet) FreeTxRefIn(i *TxRefVecIn) {
	i.t.freeChan <- i
	v.loop.Resume(&i.In)
}
func (i *TxRefVecIn) Free(v *Vnet) { v.FreeTxRefIn(i) }

func (n *interfaceNode) ifOutputThread() {
	for x := range n.outChan {
		n.tx.InterfaceOutput(x)
	}
}

func (n *interfaceNode) ifOutput(ri *RefIn) {
	id := ri.ThreadId()
	n.threads.Validate(id)
	if n.threads[id] == nil {
		n.threads[id] = &interfaceNodeThread{}
		n.threads[id].freeChan = make(chan *TxRefVecIn, 64)
	}
	nt := n.threads[id]
	rvi := nt.allocTxRefVecIn(n, ri)
	nPackets := ri.Len()
	rvi.nPackets = nPackets

	// Copy common fields.
	rvi.refInCommon = ri.refInCommon

	rvi.Refs.Validate(nPackets - 1)
	rvi.Refs = rvi.Refs[:nPackets]

	n_left := nPackets
	rs := ri.Refs[:]
	rv := rvi.Refs
	is, iv := uint(0), uint(0)
	nBytes := uint(0)
	for n_left >= 4 {
		rv[iv+0] = rs[is+0]
		rv[iv+1] = rs[is+1]
		rv[iv+2] = rs[is+2]
		rv[iv+3] = rs[is+3]
		nBytes += rs[is+0].DataLen() + rs[is+1].DataLen() + rs[is+2].DataLen() + rs[is+3].DataLen()
		iv += 4
		is += 4
		n_left -= 4
		if RefFlag4(NextValid, rs, is-4) {
			iv -= 4
			rv, iv, nBytes = n.slowPath(rv, rs, is-4, iv, nBytes)
			rv, iv, nBytes = n.slowPath(rv, rs, is-3, iv, nBytes)
			rv, iv, nBytes = n.slowPath(rv, rs, is-2, iv, nBytes)
			rv, iv, nBytes = n.slowPath(rv, rs, is-1, iv, nBytes)
			rv.Validate(iv + n_left - 1)
		}
	}
	rv.Validate(iv + n_left - 1)
	for n_left > 0 {
		rv[iv+0] = rs[is+0]
		nBytes += rs[is+0].DataLen()
		is += 1
		iv += 1
		n_left -= 1
		if RefFlag1(NextValid, rs, is-1) {
			iv -= 1
			rv, iv, nBytes = n.slowPath(rv, rs, is-1, iv, nBytes)
			rv.Validate(iv + n_left - 1)
		}
	}

	rvi.Refs = rv[:iv]

	// Bump interface packet and byte counters.
	t := n.Vnet.GetIfThread(ri.ThreadId())
	hw := n.Vnet.HwIf(n.hi)
	IfTxCounter.Add(t, hw.si, nPackets, nBytes)

	// Send to output thread, which then calls n.tx.InterfaceOutput.
	n.outChan <- rvi
}

// Transmit ring common code.
type TxDmaRing struct {
	v           *Vnet
	ToInterrupt chan *TxRefVecIn
	o           *TxRefVecIn
	n           uint
}

func (r *TxDmaRing) Init(v *Vnet) {
	r.v = v
	r.ToInterrupt = make(chan *TxRefVecIn, 64)
}

func (r *TxDmaRing) InterruptAdvance(n uint) {
	for n > 0 {
		// Nothing in current output vector: refill from channel.
		if r.n == 0 {
			r.o = <-r.ToInterrupt
			r.n = r.o.Len()
		}

		// Advanced past end of current output vector?
		if n < r.n {
			r.n -= n
			break
		}

		// If so, free it.
		n -= r.n
		r.n = 0
		r.o.Free(r.v)
	}
}
