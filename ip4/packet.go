package ip4

import (
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/ip"

	"bytes"
	"fmt"
	"unsafe"
)

const (
	AddressBytes              = 4
	HeaderBytes               = 20
	MoreFragments HeaderFlags = 1 << 13
	DontFragment  HeaderFlags = 1 << 14
	Congestion    HeaderFlags = 1 << 15
)

type Address [AddressBytes]byte

type HeaderFlags vnet.Uint16

func (h *Header) GetHeaderFlags() HeaderFlags {
	return HeaderFlags(h.Flags_and_fragment_offset.ToHost())
}
func (t HeaderFlags) FromHost() vnet.Uint16 { return vnet.Uint16(t).FromHost() }

type Header struct {
	// 4 bit header length (in 32bit units) and version VVVVLLLL.
	// e.g. for packets w/ no options ip_version_and_header_length == 0x45.
	Ip_version_and_header_length uint8

	// Type of service.
	Tos uint8

	// Total layer 3 packet length including this header.
	Length vnet.Uint16

	// 16-bit number such that Src, Dst, Protocol and Fragment ID together uniquely
	// identify packet for fragment re-assembly.
	Fragment_id vnet.Uint16

	// 3 bits of flags and 13 bits of fragment offset (in units of 8 bytes).
	Flags_and_fragment_offset vnet.Uint16

	// Time to live decremented by router at each hop.
	Ttl uint8

	// Next layer protocol.
	Protocol ip.Protocol

	Checksum vnet.Uint16

	// Source and destination address.
	Src, Dst Address
}

func (h *Header) String() (s string) {
	s = fmt.Sprintf("%s: %s -> %s", h.Protocol.String(), h.Src.String(), h.Dst.String())
	if h.Ip_version_and_header_length != 0x45 {
		s += fmt.Sprintf(", version: 0x%02x", h.Ip_version_and_header_length)
	}
	if got, want := h.Checksum, h.ComputeChecksum(); got != want {
		s += fmt.Sprintf(", checksum: 0x%04x (should be 0x%04x)", got.ToHost(), want.ToHost())
	}
	return
}

func (a *Address) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", a[0], a[1], a[2], a[3])
}

func (a *Address) Uint32() uint32 {
	return uint32(a[3]) | uint32(a[2])<<8 | uint32(a[1])<<16 | uint32(a[0])<<24
}

func (a *Address) FromUint32(x uint32) {
	a[0] = byte(x >> 24)
	a[1] = byte(x >> 16)
	a[2] = byte(x >> 8)
	a[3] = byte(x)
}

// 20 byte ip4 header wide access for efficient checksum.
type header64 struct {
	d64 [2]uint64
	d32 [1]uint32
}

func (h *Header) checksum() vnet.Uint16 {
	i := (*header64)(unsafe.Pointer(h))
	c := ip.Checksum(i.d64[0])
	c = c.AddWithCarry(ip.Checksum(i.d64[1]))
	c = c.AddWithCarry(ip.Checksum(i.d32[0]))
	return ^c.Fold()
}

func (h *Header) ComputeChecksum() vnet.Uint16 {
	var tmp Header = *h
	tmp.Checksum = 0
	return tmp.checksum()
}

func (h *Header) Len() uint { return HeaderBytes }
func (h *Header) Finalize(payload []vnet.PacketHeader) {
	var sum uint
	for _, l := range payload {
		sum += l.Len()
	}
	h.Length.Set(HeaderBytes + sum)
	h.Checksum = 0
	h.Checksum = h.checksum()
}

func (h *Header) Write(b *bytes.Buffer) {
	type t struct{ data [20]byte }
	i := (*t)(unsafe.Pointer(h))
	b.Write(i.data[:])
}
func (h *Header) Read(b []byte) vnet.PacketHeader { return (*Header)(vnet.Pointer(b)) }
