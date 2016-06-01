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

// See vnet.Arper interface.
// Dummy function to mark ethernet interfaces as supporting ARP.
func (i *Interface) SupportsArp() {}

func RegisterInterface(v *vnet.Vnet, hi HwInterfacer, config *InterfaceConfig, format string, args ...interface{}) {
	i := hi.GetInterface()
	i.InterfaceConfig = *config
	v.RegisterHwInterface(hi, format, args...)
}

var rewriteTypeMap = [...]Type{
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
	t := rewriteTypeMap[packetType].FromHost()
	size := uintptr(HeaderBytes)
	if sw != sup {
		h.Type = VLAN.FromHost()
		h.vlan[0].Priority_cfi_and_id = vnet.Uint16(sw.Id()).FromHost()
		h.vlan[0].Type = t
		size += VlanHeaderBytes
	} else {
		h.Type = t
	}
	if len(da) > 0 {
		copy(h.Dst[:], da)
	} else {
		h.Dst = BroadcastAddr
	}
	copy(h.Src[:], hi.Address[:])
	rw.AddData(unsafe.Pointer(&h), size)
}

// Block of ethernet addresses for allocation by a switch.
type AddressBlock struct {
	// Base ethernet address (stored in board's EEPROM).
	Base Address

	// Number of addresses starting at base.
	Count uint32

	nAlloc  uint32
	freeMap map[uint32]struct{}
}

func (a *Address) add(offset uint32) {
	for i, o := 0, offset; o != 0 && i < AddressBytes; i++ {
		j := AddressBytes - 1 - i
		x := uint8(o)
		y := a[j]
		y += x
		a[j] = y
		o >>= 8
		// Add in carry.
		if y < x {
			o++
		}
	}
}

func (b *AddressBlock) Alloc() (Address, bool) {
	a := b.Base
	ok := false
	var offset uint32
	for o, _ := range b.freeMap {
		delete(b.freeMap, o)
		offset = o
		ok = true
		break
	}
	if !ok {
		if ok = b.nAlloc < b.Count; ok {
			offset = b.nAlloc
			b.nAlloc++
		}
	}
	if ok {
		a.add(offset)
	}
	return a, ok
}

func (b *AddressBlock) Free(a *Address) {
	offset := uint64(0)
	for i := range a {
		j := AddressBytes - 1 - i
		offset += uint64(a[j]-b.Base[j]) << uint64(8*i)
	}

	if b.freeMap == nil {
		b.freeMap = make(map[uint32]struct{})
	}
	o := uint32(offset)
	if o >= b.Count {
		panic("bad free")
	}
	if _, ok := b.freeMap[o]; ok {
		panic("duplicate free")
	}
	b.freeMap[o] = struct{}{}
}
