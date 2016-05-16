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
	AdminUp              // admin up/down
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

type Intf struct {
	vnet.HwIf

	duplex       IfDuplex
	phyInterface PhyIf

	autoNegotiation bool

	spanningTreeState IfSpanningTreeState
	loopback          vnet.IntfLoopbackType

	// Native VLAN for this interface.
	nativeVlan Vlan
}

func (i *Intf) Set(a Attr, x interface{}) (err error) {
	switch a {
	case Name:
		if v, ok := x.(string); ok {
			i.SetIfName(v)
			return
		}
	case Speed:
		if v, ok := x.(vnet.Bandwidth); ok {
			i.SetSpeed(v)
			return
		}
	case AdminUp:
		if v, ok := x.(bool); ok {
			i.SetAdminUp(v)
			return
		}
	case AutoNegotiation:
		if v, ok := x.(bool); ok {
			i.autoNegotiation = v
			return
		}
	case Duplex:
		if v, ok := x.(IfDuplex); ok {
			i.duplex = v
			return
		}
	case PhyInterface:
		if v, ok := x.(PhyIf); ok {
			i.phyInterface = v
			return
		}
	case Loopback:
		if v, ok := x.(vnet.IntfLoopbackType); ok {
			i.loopback = v
			return
		}
	case SpanningTreeState:
		if v, ok := x.(IfSpanningTreeState); ok {
			i.spanningTreeState = v
			return
		}
	case NativeVlan:
		switch v := x.(type) {
		case Vlan:
			i.nativeVlan = v
			return
		case int:
			i.nativeVlan = Vlan(v)
			return
		case uint:
			i.nativeVlan = Vlan(v)
			return
		}
	case MaxPacketSize:
		switch v := x.(type) {
		case int:
			i.SetMaxPacketSize(v * vnet.Bytes)
			return
		case uint:
			i.SetMaxPacketSize(int(v) * vnet.Bytes)
			return
		}
	default:
		return fmt.Errorf("unknown attribute %v", a)
	}
	return fmt.Errorf("wrong type for attribute (%v: %T(%v))", a, x, x)
}

func (i *Intf) SetAttrs(as Attrs) (err error) {
	for a, v := range as {
		err = i.Set(a, v)
		if err != nil {
			return
		}
	}
	return
}

func (i *Intf) SetMulti(x ...interface{}) (err error) {
	for len(x) >= 2 {
		var a Attr
		var ok bool
		if a, ok = x[0].(Attr); !ok {
			return fmt.Errorf("expecting attribute %v", x[0])
		}
		err = i.Set(a, x[1])
		if err != nil {
			return
		}
		x = x[2:]
	}
	if len(x) != 0 {
		return fmt.Errorf("odd number of arguments")
	}
	return
}
