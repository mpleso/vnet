package vnet

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/cli"
	"github.com/platinasystems/elib/dep"
	"github.com/platinasystems/elib/loop"

	"fmt"
)

type RxTx int

const (
	Rx RxTx = iota
	Tx
	NRxTx
)

var rxTxStrings = [...]string{
	Rx: "rx",
	Tx: "tx",
}

func (x RxTx) String() (s string) {
	return elib.Stringer(rxTxStrings[:], int(x))
}

type Node struct {
	Vnet *Vnet
	loop.Node
	Errors    []string
	errorRefs []ErrorRef
	Next      []string
}

func (n *Node) GetVnetNode() *Node { return n }

type Noder interface {
	loop.Noder
	GetVnetNode() *Node
}

type Ref struct {
	loop.RefHeader

	Err ErrorRef

	unused [loop.RefOpaqueBytes - 4]byte
}

func (r *Ref) Flags() BufferFlag { return BufferFlag(r.Flags()) }

type BufferFlag loop.BufferFlag

const (
	NextValid = BufferFlag(loop.NextValid)
	Cloned    = BufferFlag(loop.Cloned)
)

func RefFlag1(f BufferFlag, r []Ref, i uint) bool {
	return loop.RefFlag1(loop.BufferFlag(f), &r[i+0].RefHeader)
}
func RefFlag2(f BufferFlag, r []Ref, i uint) bool {
	return loop.RefFlag2(loop.BufferFlag(f), &r[i+0].RefHeader, &r[i+1].RefHeader)
}
func RefFlag4(f BufferFlag, r []Ref, i uint) bool {
	return loop.RefFlag4(loop.BufferFlag(f), &r[i+0].RefHeader, &r[i+1].RefHeader, &r[i+2].RefHeader, &r[i+3].RefHeader)
}

//go:generate gentemplate -d Package=vnet -id Ref -d VecType=RefVec -d Type=Ref github.com/platinasystems/elib/vec.tmpl

type refInCommon struct {
	loop.In
	BufferPool *loop.BufferPool
}

type RefIn struct {
	refInCommon
	Refs [loop.V]Ref
}

type RefVecIn struct {
	refInCommon
	Refs RefVec
}

//go:generate gentemplate -d Package=vnet -id RefVecIn -d VecType=RefVecInVec -d Type=*RefVecIn github.com/platinasystems/elib/vec.tmpl

type RefOut struct {
	loop.Out
	Outs []RefIn
}

func (r *RefIn) AllocPoolRefs(pool *loop.BufferPool) {
	r.BufferPool = pool
	pool.AllocRefs(&r.Refs[0].RefHeader, uint(len(r.Refs)))
}
func (r *RefIn) AllocRefs()             { r.AllocPoolRefs(loop.DefaultBufferPool) }
func (i *RefIn) SetLen(v *Vnet, l uint) { i.In.SetLen(&v.loop, l) }

func (r *RefVecIn) FreePoolRefs(pool *loop.BufferPool) {
	pool.FreeRefs(&r.Refs[0].RefHeader, uint(len(r.Refs)))
}
func (r *RefVecIn) FreeRefs() { r.FreePoolRefs(r.BufferPool) }

type OutputNode struct {
	Node
	o OutputNoder
}

func (n *OutputNode) GetOutputNode() *OutputNode               { return n }
func (n *OutputNode) MakeLoopIn() loop.LooperIn                { return &RefIn{} }
func (n *OutputNode) LoopOutput(l *loop.Loop, i loop.LooperIn) { n.o.NodeOutput(i.(*RefIn)) }

type OutputNoder interface {
	Noder
	GetOutputNode() *OutputNode
	NodeOutput(i *RefIn)
}

func (v *Vnet) RegisterOutputNode(n OutputNoder, name string, args ...interface{}) {
	v.RegisterNode(n, name, args...)
	x := n.GetOutputNode()
	x.o = n
}

type InOutNode struct {
	Node
	t InOutNoder
}

func (n *InOutNode) GetInOutNode() *InOutNode    { return n }
func (n *InOutNode) MakeLoopIn() loop.LooperIn   { return &RefIn{} }
func (n *InOutNode) MakeLoopOut() loop.LooperOut { return &RefOut{} }
func (n *InOutNode) LoopInputOutput(l *loop.Loop, i loop.LooperIn, o loop.LooperOut) {
	n.t.NodeInput(i.(*RefIn), o.(*RefOut))
}

type InOutNoder interface {
	Noder
	GetInOutNode() *InOutNode
	NodeInput(i *RefIn, o *RefOut)
}

func (v *Vnet) RegisterInOutNode(n InOutNoder, name string, args ...interface{}) {
	v.RegisterNode(n, name, args...)
	x := n.GetInOutNode()
	x.t = n
}

type InterfaceNoder interface {
	Noder
	GetInterfaceNode() *InterfaceNode
	InterfaceInput(out *RefOut)
	InterfaceOutput(in *RefVecIn)
}

type InterfaceNode struct {
	Node

	refVecIns RefVecInVec

	Hi Hi

	i InterfaceNoder
}

func (n *InterfaceNode) GetInterfaceNode() *InterfaceNode         { return n }
func (n *InterfaceNode) MakeLoopIn() loop.LooperIn                { return &RefIn{} }
func (n *InterfaceNode) MakeLoopOut() loop.LooperOut              { return &RefOut{} }
func (n *InterfaceNode) LoopOutput(l *loop.Loop, i loop.LooperIn) { n.InterfaceOutput(i.(*RefIn)) }

func (n *InterfaceNode) LoopInput(l *loop.Loop, o loop.LooperOut) { n.i.InterfaceInput(o.(*RefOut)) }
func (v *Vnet) RegisterInterfaceNode(n InterfaceNoder, hi Hi, name string, args ...interface{}) {
	x := n.GetInterfaceNode()
	x.i = n
	x.Hi = hi
	v.RegisterNode(n, name, args...)
}

// Main structure.
type Vnet struct {
	loop loop.Loop
	interfaceMain
}

func (v *Vnet) RegisterNode(n Noder, format string, args ...interface{}) {
	v.loop.RegisterNode(n, format, args...)
	x := n.GetVnetNode()
	x.Vnet = v

	x.errorRefs = make([]ErrorRef, len(x.Errors))
	for i := range x.Errors {
		er := ^ErrorRef(0)
		if len(x.Errors[i]) > 0 {
			er = x.NewError(x.Errors[i])
		}
		x.errorRefs[i] = er
	}

	for i := range x.Next {
		if _, ok := v.loop.AddNamedNext(n, x.Next[i]); !ok {
			panic(fmt.Errorf("unknown next named %s", x.Next[i]))
		}
	}
}

//go:generate gentemplate -id initHook -d Package=vnet -d DepsType=initHookVec -d Type=initHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl
type initHook func(v *Vnet)

var initHooks initHookVec

func AddInit(f initHook, deps ...*dep.Dep) { initHooks.Add(f, deps...) }

func (v *Vnet) Run() {
	loop.AddInit(func(l *loop.Loop) {
		for i := range initHooks.hooks {
			initHooks.Get(i)(v)
		}
	})
	v.loop.Run()
}

func Run() {
	v := &Vnet{}
	v.Run()
}

func (v *Vnet) CliAdd(c *cli.Command)                     { v.loop.CliAdd(c) }
func (v *Vnet) Logf(format string, args ...interface{})   { v.loop.Logf(format, args...) }
func (v *Vnet) Fatalf(format string, args ...interface{}) { v.loop.Fatalf(format, args...) }
