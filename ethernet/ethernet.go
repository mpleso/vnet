package ethernet

import (
	"bytes"
	"encoding/binary"
	"net"
	"vnet/layer"
)

// Header for ethernet packets as they appear on the network.
type Header struct {
	Dst  Address
	Src  Address
	Type Type
}

type Vlan uint16

// Tagged packets have VlanHeader after ethernet header.
type VlanHeader struct {
	/* 3 bit priority, 1 bit CFI and 12 bit vlan id. */
	Priority_cfi_and_id uint16

	/* Inner ethernet type. */
	Type Type
}

// Packet type from ethernet header.
type Type uint16

const (
	IP4  Type = 0x800
	IP6  Type = 0x86DD
	ARP  Type = 0x806
	VLAN Type = 0x8100
)

//go:generate stringer -type=Type

const (
	AddressBytes = 6
	HeaderBytes  = 14
)

type Address [AddressBytes]byte

const hexDigit = "0123456789abcdef"

func (a *Address) isBroadcast() bool {
	return a[0]&1 != 0
}
func (a *Address) isUnicast() bool {
	return !a.isBroadcast()
}

func (h *Header) isBroadcast() bool {
	return h.Dst.isBroadcast()
}
func (h *Header) isUnicast() bool {
	return !h.Dst.isBroadcast()
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

// Implement layer interface.
func (h *Header) Len() int              { return HeaderBytes }
func (h *Header) Fin(l []layer.Layer)   {}
func (h *Header) Write(b *bytes.Buffer) { binary.Write(b, binary.BigEndian, h) }
