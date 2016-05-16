package vnet

import (
	"github.com/platinasystems/elib"
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

// Main structure.
type Vnet struct {
}

var defaultVnet = &Vnet{}
