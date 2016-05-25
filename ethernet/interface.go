// Ethernet hardware interfaces.
package ethernet

import (
	"github.com/platinasystems/vnet"

	"unsafe"
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
type PhyInterface int

// Mac to PHY physical interface types.  Sorted alphabetically.
const (
	CAUI PhyInterface = iota + 1
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

//go:generate stringer -type=PhyInterface,IfSpanningTreeState,IfDuplex

type InterfaceConfig struct {
	Address      Address
	PhyInterface PhyInterface
	NativeVlan   Vlan
}

type Interface struct {
	vnet.HwIf

	InterfaceConfig

	duplex IfDuplex

	autoNegotiation bool

	spanningTreeState IfSpanningTreeState
	loopback          vnet.IfLoopbackType
}

func (i *Interface) GetInterface() *Interface { return i }

type HwInterfacer interface {
	GetInterface() *Interface
	vnet.HwInterfacer
}

func RegisterInterface(hi HwInterfacer, config *InterfaceConfig, format string, args ...interface{}) {
	i := hi.GetInterface()
	i.InterfaceConfig = *config
	vnet.RegisterHwInterface(hi, format, args...)
}

var typeMap = [...]Type{
	vnet.IP4:            IP4,
	vnet.IP6:            IP6,
	vnet.MPLS_UNICAST:   MPLS_UNICAST,
	vnet.MPLS_MULTICAST: MPLS_MULTICAST,
}

func (hi *Interface) SetRewrite(v *vnet.Vnet, rw *vnet.Rewrite, packetType vnet.PacketType, da []byte) {
	var h struct {
		Header
		vlan [2]VlanHeader
	}
	sw := v.SwIf(rw.Si)
	sup := v.SupSwIf(sw)
	t := typeMap[packetType].FromHost()
	size := uintptr(HeaderBytes)
	if sw != sup {
		h.Type = VLAN.FromHost()
		h.vlan[0].Priority_cfi_and_id = vnet.Uint16(sw.Id()).FromHost()
		h.vlan[0].Type = t
		size += VlanHeaderBytes
	} else {
		h.Type = t
	}
	copy(h.Dst[:], da)
	copy(h.Src[:], hi.Address[:])
	rw.AddData(unsafe.Pointer(&h), size)
}
