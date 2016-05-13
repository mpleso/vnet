package ethernet

import (
	"github.com/platinasystems/vnet"

	"bytes"
	"fmt"
	"net"
	"unsafe"
)

// Header for ethernet packets as they appear on the network.
type Header struct {
	Dst  Address
	Src  Address
	Type vnet.Uint16
}

type Vlan vnet.Uint16

// Tagged packets have VlanHeader after ethernet header.
type VlanHeader struct {
	/* 3 bit priority, 1 bit CFI and 12 bit vlan id. */
	Priority_cfi_and_id vnet.Uint16

	/* Inner ethernet type. */
	Type vnet.Uint16
}

// Packet type from ethernet header.
type Type uint16

const (
	IP4  Type = 0x800
	IP6  Type = 0x86DD
	ARP  Type = 0x806
	VLAN Type = 0x8100
)

func (h *Header) GetType() Type      { return Type(h.Type.ToHost()) }
func (t Type) FromHost() vnet.Uint16 { return vnet.Uint16(t).FromHost() }

//go:generate stringer -type=Type

const (
	AddressBytes = 6
	HeaderBytes  = 14
)

type Address [AddressBytes]byte

var BroadcastAddr = Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

const hexDigit = "0123456789abcdef"

func (a *Address) IsBroadcast() bool {
	return a[0]&1 != 0
}
func (a *Address) IsUnicast() bool {
	return !a.IsBroadcast()
}

func (h *Header) IsBroadcast() bool {
	return h.Dst.IsBroadcast()
}
func (h *Header) IsUnicast() bool {
	return !h.Dst.IsBroadcast()
}

func (a *Address) Add(x uint64) {
	var i int
	i = AddressBytes - 1
	for x != 0 && i > 0 {
		ai := uint64(a[i])
		y := ai + (x & 0xff)
		a[i] = byte(ai)
		x >>= 8
		if y < ai {
			x += 1
		}
		i--
	}
}

func (a *Address) FromUint64(x uint64) {
	for i := 0; i < AddressBytes; i++ {
		a[i] = byte((x >> uint(40-8*i)) & 0xff)
	}
}

func (a *Address) ToUint64() (x uint64) {
	for i := 0; i < AddressBytes; i++ {
		x |= uint64(a[i]) << uint(40-8*i)
	}
	return
}

func (a *Address) String() string {
	buf := make([]byte, 0, len(a)*3-1)
	for i, b := range a {
		if i > 0 {
			buf = append(buf, ':')
		}
		buf = append(buf, hexDigit[b>>4])
		buf = append(buf, hexDigit[b&0xF])
	}
	return string(buf)
}

func (a *Address) Equal(b Address) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (a *Address) Parse(s string) (err error) {
	ha, err := net.ParseMAC(s)
	if err != nil {
		return
	}
	if len(ha) != len(a) {
		err = &net.AddrError{Err: "expected 6 bytes", Addr: s}
		return
	}
	for i := 0; i < len(a); i++ {
		a[i] = ha[i]
	}
	return
}

func (h *Header) String() (s string) {
	return fmt.Sprintf("%s: %s -> %s", h.GetType().String(), h.Src.String(), h.Dst.String())
}

// Implement vnet.Header interface.
func (h *Header) Len() uint                      { return HeaderBytes }
func (h *Header) Finalize(l []vnet.PacketHeader) {}
func (h *Header) Write(b *bytes.Buffer) {
	type t struct{ data [unsafe.Sizeof(*h)]byte }
	i := (*t)(unsafe.Pointer(h))
	b.Write(i.data[:])
}
func (h *Header) Read(b []byte) vnet.PacketHeader { return (*Header)(vnet.Pointer(b)) }
