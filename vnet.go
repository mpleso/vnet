package vnet

import (
	"github.com/platinasystems/elib"
	"github.com/platinasystems/elib/dep"
	"github.com/platinasystems/elib/hw"
	"github.com/platinasystems/elib/loop"
	"github.com/platinasystems/elib/parse"

	"unsafe"
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
	Dep       dep.Dep
	Errors    []string
	errorRefs []ErrorRef
}

func (n *Node) GetVnetNode() *Node { return n }

const MaxVectorLen = loop.MaxVectorLen

type Noder interface {
	loop.Noder
	GetVnetNode() *Node
}

func (v *Vnet) AddNamedNext(n Noder, name string) uint {
	if nextIndex, err := v.loop.AddNamedNext(n, name); err == nil {
		return nextIndex
	} else {
		panic(err)
	}
}

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

type RefVecIn struct {
	refInCommon
	Refs RefVec
}

type RefOut struct {
	loop.Out
	Outs []RefIn
}

func (r *RefIn) AllocPoolRefs(pool *hw.BufferPool) {
	r.BufferPool = pool
	pool.AllocRefs(&r.Refs[0].RefHeader, uint(len(r.Refs)))
}
func (r *RefIn) AllocRefs()             { r.AllocPoolRefs(hw.DefaultBufferPool) }
func (i *RefIn) SetLen(v *Vnet, l uint) { i.In.SetLen(&v.loop, l) }
func (i *RefIn) AddLen(v *Vnet) (l uint) {
	l = i.GetLen(&v.loop)
	i.SetLen(v, l+1)
	return
}

func (r *RefVecIn) FreePoolRefs(pool *hw.BufferPool) {
	pool.FreeRefs(&r.Refs[0].RefHeader, uint(len(r.Refs)))
}
func (r *RefVecIn) FreeRefs() { r.FreePoolRefs(r.BufferPool) }

type InputNode struct {
	Node
	o InputNoder
}

func (n *InputNode) GetInputNode() *InputNode                 { return n }
func (n *InputNode) MakeLoopOut() loop.LooperOut              { return &RefOut{} }
func (n *InputNode) LoopInput(l *loop.Loop, o loop.LooperOut) { n.o.NodeInput(o.(*RefOut)) }

type InputNoder interface {
	Noder
	GetInputNode() *InputNode
	NodeInput(o *RefOut)
}

func (v *Vnet) RegisterInputNode(n InputNoder, name string, args ...interface{}) {
	v.RegisterNode(n, name, args...)
	x := n.GetInputNode()
	x.o = n
}

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

// Main structure.
type Vnet struct {
	loop loop.Loop
	interfaceMain
	cliMain
	packageMain
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
}

//go:generate gentemplate -id initHook -d Package=vnet -d DepsType=initHookVec -d Type=initHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl
type initHook func(v *Vnet)

var initHooks initHookVec

func AddInit(f initHook, deps ...*dep.Dep) { initHooks.Add(f, deps...) }

func (v *Vnet) configure(in *parse.Input) (err error) {
	if err = v.ConfigurePackages(in); err != nil {
		return
	}
	if err = v.InitPackages(); err != nil {
		return
	}
	return
}

func (v *Vnet) Run(in *parse.Input) (err error) {
	loop.AddInit(func(l *loop.Loop) {
		v.interfaceMain.init()
		v.CliInit()
		for i := range initHooks.hooks {
			initHooks.Get(i)(v)
		}
		if err := v.configure(in); err != nil {
			panic(err)
		}
	})
	v.loop.Run()
	err = v.ExitPackages()
	return
}
