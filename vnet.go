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

type initHook func(v *Vnet)

//go:generate gentemplate -id initHook -d Package=vnet -d DepsType=initHookVec -d Type=initHook -d Data=hooks github.com/platinasystems/elib/dep/dep.tmpl

// Main structure.
type Vnet struct {
	loop loop.Loop
	interfaceMain
}

func (v *Vnet) Register(n Noder, format string, args ...interface{}) {
	v.loop.Register(n, format, args...)
	x := n.GetVnetNode()
	x.Vnet = v
}

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
