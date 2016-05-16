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

// Main structure.
type Vnet struct {
	interfaceMain
}

var defaultVnet = &Vnet{}

func (v *Vnet) Register(n loop.Noder, format string, args ...interface{}) {
	nodeFormat := format

	if x, ok := n.(HwInterfacer); ok {
		v.RegisterHwInterface(x, format, args...)
		nodeFormat = nodeFormat + "-input"
	}

	loop.Register(n, nodeFormat, args...)
}
func Register(n loop.Noder, format string, args ...interface{}) {
	defaultVnet.Register(n, format, args...)
}
