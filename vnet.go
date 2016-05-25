package vnet

import (
	"github.com/platinasystems/elib"
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

type Node struct{ loop.Node }
type Noder interface {
	loop.Noder
}

// Main structure.
type Vnet struct {
	interfaceMain
}

var defaultVnet = &Vnet{}

func (v *Vnet) Register(n Noder, format string, args ...interface{}) {
	loop.Register(n, format, args...)
}
func Register(n Noder, format string, args ...interface{}) {
	defaultVnet.Register(n, format, args...)
}

func (v *Vnet) Logf(format string, args ...interface{})   { loop.Logf(format, args...) }
func (v *Vnet) Fatalf(format string, args ...interface{}) { loop.Fatalf(format, args...) }
