// Ethernet hardware interfaces.
package ethernet

import (
	"github.com/platinasystems/vnet"

	"fmt"
)

// Spanning Tree State IEEE 802.1d
type IfSpanningTreeState int

// Possible spanning tree states.
const (
	Disable IfSpanningTreeState = iota + 1
	Block
	Listen
	Learn
	Forward
)

// Full or half duplex.
type IfDuplex int

const (
	Full IfDuplex = iota + 1
	Half
)

// Physical interface between ethernet MAC and PHY.
type PhyIf int

// Mac to PHY physical interface types.  Sorted alphabetically.
const (
	CAUI PhyIf = iota + 1
	CR
	CR2
	CR4
	GMII
	INTERLAKEN
	KR
	KR2
	KR4
	KX
	LR
	LR4
	MII
	QSGMII
	RGMII
	RXAUI
	SFI
	SGMII
	SPAUI
	SR
	SR10
	SR2
	SR4
	XAUI
	XFI
	XGMII
	XLAUI
	XLAUI2
	ZR
)

// Configurable interface attributes: name, speed, duplex, ...
type Attr int

const (
	Name            Attr = iota
	Speed                // float64 in units of bits per second
	AutoNegotiation      // bool
	Duplex
	PhyInterface
	Loopback
	NativeVlan
	SpanningTreeState
	MaxPacketSize
)

//go:generate stringer -type=Attr,PhyIf,IfSpanningTreeState,IfDuplex

type Attrs map[Attr]interface{}

func (as Attrs) String() string {
	s := "{"
	n := len(as)
	for ai, v := range as {
		s += fmt.Sprintf("%v: %v", ai, v)
		n--
		if n > 0 {
			s += ", "
		}
	}
	s += "}"
	return s
}

type HwInterface struct {
	duplex       IfDuplex
	phyInterface PhyIf

	autoNegotiation bool

	spanningTreeState IfSpanningTreeState
	loopback          vnet.IfLoopbackType

	// Native VLAN for this interface.
	nativeVlan Vlan
}

func (h *HwInterface) SetRewrite(rw *vnet.Rewrite) {
	panic("nyi")
}
