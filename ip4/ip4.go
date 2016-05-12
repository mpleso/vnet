package ip4

import (
	"github.com/platinasystems/vnet"
	"github.com/platinasystems/vnet/ip"

	"bytes"
	"fmt"
	"net"
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

func (a *Address) Scan(ss fmt.ScanState, verb rune) (err error) {
	tok, err := ss.Token(false, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return true
		case r >= 'A' && r <= 'Z':
			return true
		case r >= '0' && r <= '9':
			return true
		case r == '.':
			return true
		case r == '-':
			return true
		}
		return false
	})

	as, err := net.LookupHost(string(tok))
	if err != nil {
		return
	}
	for i := range as {
		_, err = fmt.Sscanf(as[i], "%d.%d.%d.%d", &a[0], &a[1], &a[2], &a[3])
		if err == nil {
			return
		}
	}
	return
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

// 16 bit port; ^uint32(0) means no port given.
type Port uint32

const (
	NilPort Port = 0xffffffff
)

func (p *Port) Scan(ss fmt.ScanState, verb rune) (err error) {
	*p = NilPort
	tok, err := ss.Token(false, func(r rune) bool {
		switch {
		case r >= '0' && r <= '9':
			return true
		}
		return false
	})
	if len(tok) > 0 {
		var x uint32
		_, err = fmt.Sscanf(string(tok), "%d", &x)
		if err != nil {
			return
		}
		if x>>16 != 0 {
			err = fmt.Errorf("out or range: %s", string(tok))
			return
		}
		*p = Port(x)
	}
	return
}

type Socket struct {
	Address Address
	Port    Port
}

func (s *Socket) String() string {
	return s.Address.String() + fmt.Sprintf(":%d", s.Port)
}

func (s *Socket) Scan(ss fmt.ScanState, verb rune) (err error) {
	r, _, err := ss.ReadRune()
	if err != nil {
		return
	}
	if r == ':' {
		_, err = fmt.Fscanf(ss, "%d", &s.Port)
	} else {
		err = ss.UnreadRune()
		if err != nil {
			return
		}
		_, err = fmt.Fscanf(ss, "%s:%d", &s.Address, &s.Port)
	}
	return
}

func sum_1x8(sum, x uint64) uint64 {
	t := sum + x
	if t < x {
		t += 1
	}
	return t
}

func sum_2x1(sum uint64, x0, x1 uint8) uint64 {
	return sum_1x8(sum, uint64(x0)<<8+uint64(x1))
}

func sum_1x2(sum uint64, x uint16) uint64 {
	return sum_1x8(sum, uint64((x>>8)&0xff)<<8+uint64((x>>0)&0xff))
}

func sum_addr(sum uint64, x Address) uint64 {
	sum = sum_1x8(sum, uint64(x[0])<<8+uint64(x[1]))
	sum = sum_1x8(sum, uint64(x[2])<<8+uint64(x[3]))
	return sum
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
