package vnet

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/cli"
	"github.com/platinasystems/elib/dep"
	"github.com/platinasystems/elib/loop"
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

type RefIn struct {
	loop.In
	pool *loop.BufferPool
	Refs [loop.V]Ref
}

type RefOut struct {
	loop.Out
	Outs []RefIn
}

func (r *RefIn) AllocPoolRefs(pool *loop.BufferPool) {
	r.pool = pool
	pool.AllocRefs(&r.Refs[0].RefHeader, uint(len(r.Refs)))
}
func (r *RefIn) AllocRefs()             { r.AllocPoolRefs(loop.DefaultBufferPool) }
func (i *RefIn) SetLen(v *Vnet, l uint) { i.In.SetLen(&v.loop, l) }

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
	InterfaceOutput(in *RefIn)
}
type InterfaceNode struct {
	Node
	i InterfaceNoder
}

func (n *InterfaceNode) GetInterfaceNode() *InterfaceNode         { return n }
func (n *InterfaceNode) MakeLoopIn() loop.LooperIn                { return &RefIn{} }
func (n *InterfaceNode) MakeLoopOut() loop.LooperOut              { return &RefOut{} }
func (n *InterfaceNode) LoopOutput(l *loop.Loop, i loop.LooperIn) { n.i.InterfaceOutput(i.(*RefIn)) }
func (n *InterfaceNode) LoopInput(l *loop.Loop, o loop.LooperOut) { n.i.InterfaceInput(o.(*RefOut)) }
func (v *Vnet) RegisterInterfaceNode(n InterfaceNoder, name string, args ...interface{}) {
	x := n.GetInterfaceNode()
	x.i = n
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
